//go:build integration

package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

//go:embed migrations_test
var testMigrationFs embed.FS

func TestPostgresMigrationsWithTestcontainers(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get host: %v", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get port: %v", err)
	}

	// Create connection config
	config := &ConnectionConfig{
		DbType:       Postgresql,
		Host:         host,
		Port:         port.Int(),
		Username:     "testuser",
		Password:     "testpass",
		DbName:       "testdb",
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	// Connect to the database
	db, err := config.SqlDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations UP
	err = RunPostgresMigrations(ctx, db, testMigrationFs, "migrations_test")
	if err != nil {
		t.Fatalf("Failed to run migrations UP: %v", err)
	}

	// Verify migrations were applied
	if err := verifyTestMigrations(db); err != nil {
		t.Fatalf("Migration verification failed after UP: %v", err)
	}

	// Run migrations DOWN
	err = RunPostgresMigrationsDown(ctx, db, testMigrationFs, "migrations_test")
	if err != nil {
		t.Fatalf("Failed to run migrations DOWN: %v", err)
	}

	// Verify tables were dropped
	if err := verifyTestTablesDropped(db); err != nil {
		t.Fatalf("Migration DOWN verification failed: %v", err)
	}
}

func verifyTestMigrations(db *sql.DB) error {
	// Check if schema_migrations table exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM pg_tables 
			WHERE schemaname = 'public' AND 
			tablename = 'schema_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if schema_migrations table exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("schema_migrations table does not exist, migrations may not have been applied")
	}

	// Check if there are any migration versions in the table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to count migrations: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("no migrations found in schema_migrations table")
	}

	// Verify specific tables from our migrations
	tables := []string{"users", "posts"}
	for _, table := range tables {
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM pg_tables 
				WHERE schemaname = 'public' AND 
				tablename = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if %s table exists: %w", table, err)
		}

		if !exists {
			return fmt.Errorf("%s table does not exist, migration may not have been applied correctly", table)
		}
	}

	// Verify the index on posts.user_id
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM pg_indexes
			WHERE schemaname = 'public' AND
			tablename = 'posts' AND
			indexname = 'idx_posts_user_id'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if idx_posts_user_id index exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("idx_posts_user_id index does not exist, migration may not have been applied correctly")
	}

	return nil
}

func verifyTestTablesDropped(db *sql.DB) error {
	// Check if schema_migrations table still exists
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM pg_tables 
			WHERE schemaname = 'public' AND 
			tablename = 'schema_migrations'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if schema_migrations table exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("schema_migrations table does not exist, which is unexpected after DOWN migration")
	}

	// Verify specific tables from our migrations are dropped
	tables := []string{"users", "posts"}
	for _, table := range tables {
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT FROM pg_tables 
				WHERE schemaname = 'public' AND 
				tablename = $1
			)
		`, table).Scan(&exists)

		if err != nil {
			return fmt.Errorf("failed to check if %s table exists: %w", table, err)
		}

		if exists {
			return fmt.Errorf("%s table still exists, migration DOWN may not have been applied correctly", table)
		}
	}

	// Verify the index on posts.user_id is dropped
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM pg_indexes
			WHERE schemaname = 'public' AND
			tablename = 'posts' AND
			indexname = 'idx_posts_user_id'
		)
	`).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check if idx_posts_user_id index exists: %w", err)
	}

	if exists {
		return fmt.Errorf("idx_posts_user_id index still exists, migration DOWN may not have been applied correctly")
	}

	return nil
}
