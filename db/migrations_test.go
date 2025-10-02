package db

import (
	"context"
	"database/sql"
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/empty_migrations/*
var emptyMigrationsFS embed.FS

// TestRunPostgresMigrations_ConnectionError tests that RunPostgresMigrations
// returns an error when the database connection fails
func TestRunPostgresMigrations_ConnectionError(t *testing.T) {
	ctx := context.Background()

	// Create a sql.DB with invalid connection that will fail quickly
	db, err := sql.Open("postgres", "host=invalid-host-that-does-not-exist.local port=5432 connect_timeout=1")
	require.NoError(t, err)
	defer db.Close()

	// Should fail when trying to create database driver
	err = RunPostgresMigrations(ctx, db, emptyMigrationsFS, "testdata/empty_migrations")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create database driver")
}

// TestRunPostgresMigrationsDown_ConnectionError tests that RunPostgresMigrationsDown
// returns an error when the database connection fails
func TestRunPostgresMigrationsDown_ConnectionError(t *testing.T) {
	ctx := context.Background()

	// Create a sql.DB with invalid connection that will fail quickly
	db, err := sql.Open("postgres", "host=invalid-host-that-does-not-exist.local port=5432 connect_timeout=1")
	require.NoError(t, err)
	defer db.Close()

	// Should fail when trying to create database driver
	err = RunPostgresMigrationsDown(ctx, db, emptyMigrationsFS, "testdata/empty_migrations")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create database driver")
}

// TestSetupMigration_ConnectionError tests that setupMigration returns an error
// when the database connection fails
func TestSetupMigration_ConnectionError(t *testing.T) {
	ctx := context.Background()

	// Create a sql.DB with invalid connection
	db, err := sql.Open("postgres", "host=invalid-host.local port=5432 connect_timeout=1")
	require.NoError(t, err)
	defer db.Close()

	// Should fail when trying to create database driver
	_, _, err = setupMigration(ctx, db, emptyMigrationsFS, "testdata/empty_migrations")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create database driver")
}
