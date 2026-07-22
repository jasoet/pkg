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

func setupMigration(db *sql.DB, migrationFs embed.FS, migrationsPath string) (*migrate.Migrate, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create database driver: %w", err)
	}

	d, err := iofs.New(migrationFs, migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", d, "", driver)
	if err != nil {
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

	m, err := setupMigration(db, migrationFs, migrationsPath)
	if err != nil {
		return lc.Error(err, "failed to set up migration")
	}

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

	m, err := setupMigration(db, migrationFs, migrationsPath)
	if err != nil {
		return lc.Error(err, "failed to set up migration")
	}

	lc.Logger.Debug("Starting PostgreSQL migrations DOWN")
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return lc.Error(fmt.Errorf("failed to roll back migrations: %w", err), "failed to roll back migrations")
	}

	lc.Success("Migrations rolled back successfully")
	return nil
}
