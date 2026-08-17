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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

//go:embed migrations_test
var testMigrationFs embed.FS

func startPostgresForMigrations(t *testing.T) *postgres.PostgresContainer {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgresReady(),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")
	return container
}

func migrationTestConfig(t *testing.T, container *postgres.PostgresContainer) *ConnectionConfig {
	t.Helper()
	ctx := context.Background()

	host, err := container.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	return &ConnectionConfig{
		DBType:       Postgresql,
		Host:         host,
		Port:         port.Int(),
		Username:     "testuser",
		Password:     "testpass",
		DBName:       "testdb",
		SSLMode:      "disable", // testcontainer has no TLS
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}
}

func TestPostgresMigrationsWithTestcontainers(t *testing.T) {
	ctx := context.Background()

	container := startPostgresForMigrations(t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	config := migrationTestConfig(t, container)

	db, err := config.SQLDB()
	require.NoError(t, err, "Failed to connect to database")
	defer db.Close()

	// Run migrations UP
	require.NoError(t, RunPostgresMigrations(ctx, db, testMigrationFs, "migrations_test"), "Failed to run migrations UP")
	require.NoError(t, verifyTestMigrations(db), "Migration verification failed after UP")

	// Run migrations DOWN
	require.NoError(t, RunPostgresMigrationsDown(ctx, db, testMigrationFs, "migrations_test"), "Failed to run migrations DOWN")
	require.NoError(t, verifyTestTablesDropped(db), "Migration DOWN verification failed")
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

// TestPostgresMigrationsFromGormPool tests the GORM call-site pattern:
// obtain the underlying *sql.DB via gormDB.DB() and run the sql.DB migration variants.
func TestPostgresMigrationsFromGormPool(t *testing.T) {
	ctx := context.Background()

	container := startPostgresForMigrations(t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	config := migrationTestConfig(t, container)

	// Connect to the database using NewPool (GORM)
	gormDB, err := NewPool(WithConnectionConfig(*config))
	require.NoError(t, err, "Failed to connect to database")

	// Get underlying sql.DB — the call-site pattern for GORM users
	sqlDB, err := gormDB.DB()
	require.NoError(t, err, "Failed to get sql.DB")
	defer sqlDB.Close()

	// Run migrations UP via the sql.DB variant
	require.NoError(t, RunPostgresMigrations(ctx, sqlDB, testMigrationFs, "migrations_test"), "Failed to run migrations UP")
	require.NoError(t, verifyTestMigrations(sqlDB), "Migration verification failed after UP")

	// Run migrations DOWN via the sql.DB variant
	require.NoError(t, RunPostgresMigrationsDown(ctx, sqlDB, testMigrationFs, "migrations_test"), "Failed to run migrations DOWN")
	require.NoError(t, verifyTestTablesDropped(sqlDB), "Migration DOWN verification failed")

	// The pool must still be usable after migrations: setupMigration checks out a
	// dedicated connection and releases it, so it never pins a pool slot.
	require.NoError(t, sqlDB.PingContext(ctx), "pool should still be usable after migrations")
}

// TestPostgresMigrationsInvalidPath tests error handling with an invalid migration path
func TestPostgresMigrationsInvalidPath(t *testing.T) {
	ctx := context.Background()

	container := startPostgresForMigrations(t)
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	config := migrationTestConfig(t, container)

	sqlDB, err := config.SQLDB()
	require.NoError(t, err, "Failed to connect to database")
	defer sqlDB.Close()

	// Try to run migrations with non-existent path
	assert.Error(t, RunPostgresMigrations(ctx, sqlDB, testMigrationFs, "non_existent_path"),
		"Expected error with invalid migration path")

	// Try to run migrations down with non-existent path
	assert.Error(t, RunPostgresMigrationsDown(ctx, sqlDB, testMigrationFs, "non_existent_path"),
		"Expected error with invalid migration path")
}
