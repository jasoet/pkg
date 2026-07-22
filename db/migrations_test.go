package db

import (
	"context"
	"database/sql"
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

//go:embed testdata/empty_migrations/*
var emptyMigrationsFS embed.FS

// TestRunPostgresMigrations_EmitsSpan verifies that RunPostgresMigrations emits
// an operations-layer span (with a correlated logger via LayerContext) even on
// the failure path, using an unreachable database to avoid needing a live DB.
func TestRunPostgresMigrations_EmitsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithTracerProvider(tp))
	ctx := pkgotel.ContextWithConfig(context.Background(), cfg)

	// Create a sql.DB with invalid connection that will fail quickly
	db, err := sql.Open("postgres", "host=invalid-host-that-does-not-exist.local port=5432 connect_timeout=1")
	require.NoError(t, err)
	defer db.Close()

	err = RunPostgresMigrations(ctx, db, emptyMigrationsFS, "testdata/empty_migrations")
	require.Error(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one ended span")
	assert.Equal(t, "db.RunPostgresMigrations", spans[0].Name)
	assert.Equal(t, "operations.db", spans[0].InstrumentationScope.Name)
}

// TestRunPostgresMigrationsDown_EmitsSpan verifies the same span instrumentation
// for the DOWN variant on the failure path.
func TestRunPostgresMigrationsDown_EmitsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := pkgotel.NewConfig("test-service", pkgotel.WithTracerProvider(tp))
	ctx := pkgotel.ContextWithConfig(context.Background(), cfg)

	db, err := sql.Open("postgres", "host=invalid-host-that-does-not-exist.local port=5432 connect_timeout=1")
	require.NoError(t, err)
	defer db.Close()

	err = RunPostgresMigrationsDown(ctx, db, emptyMigrationsFS, "testdata/empty_migrations")
	require.Error(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one ended span")
	assert.Equal(t, "db.RunPostgresMigrationsDown", spans[0].Name)
	assert.Equal(t, "operations.db", spans[0].InstrumentationScope.Name)
}

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
	// Create a sql.DB with invalid connection
	db, err := sql.Open("postgres", "host=invalid-host.local port=5432 connect_timeout=1")
	require.NoError(t, err)
	defer db.Close()

	// Should fail when trying to create database driver
	_, err = setupMigration(db, emptyMigrationsFS, "testdata/empty_migrations")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create database driver")
}
