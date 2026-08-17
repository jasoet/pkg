package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/jasoet/pkg/v3/otel"
)

// setupMigration builds a *migrate.Migrate bound to a single connection checked
// out from the caller's pool (via db.Conn). The driver is created with
// postgres.WithConnection rather than postgres.WithInstance, so the returned
// instance's Close() releases only that pinned connection back to the pool — it
// does NOT close the caller's *sql.DB. Callers MUST call m.Close() when done, or
// the connection stays pinned for the lifetime of the pool.
func setupMigration(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) (*migrate.Migrate, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire database connection: %w", err)
	}

	driver, err := postgres.WithConnection(ctx, conn, &postgres.Config{})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	d, err := iofs.New(migrationFs, migrationsPath)
	if err != nil {
		// driver owns conn; closing the driver releases it back to the pool.
		_ = driver.Close()
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "", driver)
	if err != nil {
		_ = driver.Close()
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}

// RunPostgresMigrations applies pending UP migrations using a raw *sql.DB connection.
// GORM users can obtain a *sql.DB via gormDB.DB().
//
// Note: only PostgreSQL is supported. For MySQL or MSSQL, use a different migration tool.
func RunPostgresMigrations(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) error {
	lc := otel.Layers.StartOperations(ctx, "db", "RunPostgresMigrations")
	defer lc.End()

	m, err := setupMigration(ctx, db, migrationFs, migrationsPath)
	if err != nil {
		return lc.Error(err, "failed to set up migration")
	}
	// Release the pinned connection back to the caller's pool. This closes only
	// the dedicated connection, not the caller's *sql.DB.
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			lc.Logger.Debug("error closing migrate instance",
				otel.F("sourceErr", srcErr), otel.F("dbErr", dbErr))
		}
	}()

	lc.Logger.Debug("Starting PostgreSQL migrations UP")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return lc.Error(fmt.Errorf("failed to apply migrations: %w", err), "failed to apply migrations")
	}

	lc.Success("Migrations applied successfully")
	return nil
}

// RunPostgresMigrationsDown rolls back all migrations using a raw *sql.DB connection.
// GORM users can obtain a *sql.DB via gormDB.DB().
//
// Note: only PostgreSQL is supported. For MySQL or MSSQL, use a different migration tool.
func RunPostgresMigrationsDown(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) error {
	lc := otel.Layers.StartOperations(ctx, "db", "RunPostgresMigrationsDown")
	defer lc.End()

	m, err := setupMigration(ctx, db, migrationFs, migrationsPath)
	if err != nil {
		return lc.Error(err, "failed to set up migration")
	}
	// Release the pinned connection back to the caller's pool. This closes only
	// the dedicated connection, not the caller's *sql.DB.
	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			lc.Logger.Debug("error closing migrate instance",
				otel.F("sourceErr", srcErr), otel.F("dbErr", dbErr))
		}
	}()

	lc.Logger.Debug("Starting PostgreSQL migrations DOWN")
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return lc.Error(fmt.Errorf("failed to roll back migrations: %w", err), "failed to roll back migrations")
	}

	lc.Success("Migrations rolled back successfully")
	return nil
}
