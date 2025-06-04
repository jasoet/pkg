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
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

func RunPostgresMigrationsWithGorm(ctx context.Context, db *gorm.DB, migrationFs embed.FS, migrationsPath string) error {
	logger := logging.ContextLogger(ctx, "db.migrations")
	logger.Debug().Msg("Starting PostgreSQL migrations UP with GORM")

	sqlDb, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB from GORM: %w", err)
	}
	return RunPostgresMigrations(ctx, sqlDb, migrationFs, migrationsPath)
}

func RunPostgresMigrationsDownWithGorm(ctx context.Context, db *gorm.DB, migrationFs embed.FS, migrationsPath string) error {
	logger := logging.ContextLogger(ctx, "db.migrations")
	logger.Debug().Msg("Starting PostgreSQL migrations DOWN with GORM")

	sqlDb, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB from GORM: %w", err)
	}
	return RunPostgresMigrationsDown(ctx, sqlDb, migrationFs, migrationsPath)
}

func RunPostgresMigrationsDropWithGorm(ctx context.Context, db *gorm.DB, migrationFs embed.FS, migrationsPath string) error {
	logger := logging.ContextLogger(ctx, "db.migrations")
	logger.Debug().Msg("Starting PostgreSQL migrations DROP with GORM")

	sqlDb, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB from GORM: %w", err)
	}
	return RunPostgresMigrationsDrop(ctx, sqlDb, migrationFs, migrationsPath)
}

func setupMigration(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) (*migrate.Migrate, zerolog.Logger, error) {
	logger := logging.ContextLogger(ctx, "db.migrations")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, logger, fmt.Errorf("failed to create database driver: %w", err)
	}
	logger.Debug().Msg("Database driver created successfully")

	d, err := iofs.New(migrationFs, migrationsPath)
	if err != nil {
		return nil, logger, fmt.Errorf("failed to create migration source: %w", err)
	}
	logger.Debug().Msg("Migration source created successfully")

	m, err := migrate.NewWithInstance("iofs", d, "", driver)
	if err != nil {
		return nil, logger, fmt.Errorf("failed to create migrate instance: %w", err)
	}
	logger.Debug().Msg("Migrate instance created successfully")

	return m, logger, nil
}

func RunPostgresMigrations(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) error {
	m, logger, err := setupMigration(ctx, db, migrationFs, migrationsPath)
	if err != nil {
		return err
	}

	logger.Debug().Msg("Starting PostgreSQL migrations UP")
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}
	logger.Debug().Msg("Migrations applied successfully")

	return nil
}

func RunPostgresMigrationsDown(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) error {
	m, logger, err := setupMigration(ctx, db, migrationFs, migrationsPath)
	if err != nil {
		return err
	}

	logger.Debug().Msg("Starting PostgreSQL migrations DOWN")
	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to roll back migrations: %w", err)
	}
	logger.Debug().Msg("Migrations rolled back successfully")

	return nil
}

func RunPostgresMigrationsDrop(ctx context.Context, db *sql.DB, migrationFs embed.FS, migrationsPath string) error {
	m, logger, err := setupMigration(ctx, db, migrationFs, migrationsPath)
	if err != nil {
		return err
	}

	logger.Debug().Msg("Starting PostgreSQL migrations DROP")
	if err := m.Drop(); err != nil {
		return fmt.Errorf("failed to drop migrations: %w", err)
	}
	logger.Debug().Msg("Migrations dropped successfully")

	return nil
}
