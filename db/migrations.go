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
	"github.com/jasoet/pkg/logging"
	"gorm.io/gorm"
)

func RunPostgresMigrationsWithGorm(ctx context.Context, db *gorm.DB, migrationFs embed.FS) error {
	logger := logging.ContextLogger(ctx, "db.migrations")
	logger.Debug().Msg("Starting PostgreSQL migrations with GORM")

	sqlDb, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB from GORM: %w", err)
	}
	return RunPostgresMigrations(ctx, sqlDb, migrationFs)
}

func RunPostgresMigrations(ctx context.Context, db *sql.DB, migrationFs embed.FS) error {
	logger := logging.ContextLogger(ctx, "db.migrations")
	logger.Debug().Msg("Starting PostgreSQL migrations")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}
	logger.Debug().Msg("Database driver created successfully")

	d, err := iofs.New(migrationFs, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}
	logger.Debug().Msg("Migration source created successfully")

	m, err := migrate.NewWithInstance("iofs", d, "", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	logger.Debug().Msg("Migrate instance created successfully")

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	logger.Debug().Msg("Migrations applied successfully")

	return nil
}
