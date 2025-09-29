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
)

//go:embed migrations_test
var migrationFs embed.FS

const (
	dbHost     = "localhost" // Host machine address
	dbPort     = 5439        // Host port mapped to container
	dbUser     = "jasoet"    // From POSTGRES_USER
	dbPassword = "localhost" // From POSTGRES_PASSWORD
	dbName     = "pkg_db"    // From POSTGRES_DB
	dbTimeout  = 10 * time.Second
)

func TestPostgresMigrationsDownAndDrop(t *testing.T) {
	// Create a connection config using values from docker-compose.yml
	config := &ConnectionConfig{
		DbType:       Postgresql,
		Host:         dbHost,
		Port:         dbPort,
		Username:     dbUser,
		Password:     dbPassword,
		DbName:       dbName,
		Timeout:      dbTimeout,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	// Connect to the database
	db, err := config.SqlDB()
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create a test context
	ctx := context.Background()

	// Run migrations UP
	err = RunPostgresMigrations(ctx, db, migrationFs, "migrations_test")
	if err != nil {
		t.Fatalf("Failed to run migrations UP: %v", err)
	}

	// Verify migrations were applied
	if err := verifyMigrations(db); err != nil {
		t.Fatalf("Migration verification failed after UP: %v", err)
	}

	// Run migrations DOWN
	err = RunPostgresMigrationsDown(ctx, db, migrationFs, "migrations_test")
	if err != nil {
		t.Fatalf("Failed to run migrations DOWN: %v", err)
	}

	// Verify tables were dropped
	if err := verifyTablesDropped(db); err != nil {
		t.Fatalf("Migration DOWN verification failed: %v", err)
	}
}

func verifyMigrations(db *sql.DB) error {
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

func verifyTablesDropped(db *sql.DB) error {
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
