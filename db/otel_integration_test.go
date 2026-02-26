//go:build integration

package db

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	pkgotel "github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	noopl "go.opentelemetry.io/otel/log/noop"
	noopm "go.opentelemetry.io/otel/metric/noop"
	noopt "go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm"
)

// TestPostgresPoolWithOTelTracing tests OTel tracing callbacks
func TestPostgresPoolWithOTelTracing(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(filepath.Join("..", "scripts", "compose", "pg", "backup", "default.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	// Create OTel config with noop providers
	otelConfig := pkgotel.NewConfig("db-test").
		WithTracerProvider(noopt.NewTracerProvider()).
		WithMeterProvider(noopm.NewMeterProvider()).
		WithLoggerProvider(noopl.NewLoggerProvider())

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
		OTelConfig:   otelConfig,
	}

	// Test Pool() with OTel config
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database with OTel config")
	require.NotNil(t, db, "Database connection should not be nil")

	// Test CREATE operation with OTel callbacks
	t.Run("Create with OTel callbacks", func(t *testing.T) {
		testProduct := Product{
			ID:            uuid.New().String(),
			Name:          "OTel Test Product",
			Description:   "Testing OTel callbacks",
			Category:      "Test",
			Price:         199.99,
			StockQuantity: 50,
			IsAvailable:   true,
			CreatedAt:     time.Now(),
		}

		err := db.WithContext(ctx).Create(&testProduct).Error
		require.NoError(t, err, "Failed to create product with OTel callbacks")

		// Verify product was created
		var count int64
		err = db.Model(&Product{}).Where("id = ?", testProduct.ID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count, "Product should be created")

		// Cleanup
		db.Delete(&Product{}, "id = ?", testProduct.ID)
	})

	// Test QUERY operation with OTel callbacks
	t.Run("Query with OTel callbacks", func(t *testing.T) {
		var products []Product
		err := db.WithContext(ctx).Where("is_available = ?", true).Limit(5).Find(&products).Error
		require.NoError(t, err, "Failed to query products with OTel callbacks")
		assert.Greater(t, len(products), 0, "Should find at least one product")
	})

	// Test UPDATE operation with OTel callbacks
	t.Run("Update with OTel callbacks", func(t *testing.T) {
		// Create a test product first
		testProduct := Product{
			ID:            uuid.New().String(),
			Name:          "OTel Update Test",
			Description:   "Testing OTel update callbacks",
			Category:      "Test",
			Price:         99.99,
			StockQuantity: 10,
			IsAvailable:   true,
			CreatedAt:     time.Now(),
		}

		err := db.Create(&testProduct).Error
		require.NoError(t, err)

		// Update the product
		err = db.WithContext(ctx).Model(&Product{}).Where("id = ?", testProduct.ID).Update("price", 149.99).Error
		require.NoError(t, err, "Failed to update product with OTel callbacks")

		// Verify update
		var updated Product
		err = db.First(&updated, "id = ?", testProduct.ID).Error
		require.NoError(t, err)
		assert.Equal(t, 149.99, updated.Price, "Price should be updated")

		// Cleanup
		db.Delete(&Product{}, "id = ?", testProduct.ID)
	})

	// Test DELETE operation with OTel callbacks
	t.Run("Delete with OTel callbacks", func(t *testing.T) {
		// Create a test product first
		testProduct := Product{
			ID:            uuid.New().String(),
			Name:          "OTel Delete Test",
			Description:   "Testing OTel delete callbacks",
			Category:      "Test",
			Price:         79.99,
			StockQuantity: 5,
			IsAvailable:   true,
			CreatedAt:     time.Now(),
		}

		err := db.Create(&testProduct).Error
		require.NoError(t, err)

		// Delete the product
		err = db.WithContext(ctx).Delete(&Product{}, "id = ?", testProduct.ID).Error
		require.NoError(t, err, "Failed to delete product with OTel callbacks")

		// Verify deletion
		var count int64
		err = db.Model(&Product{}).Where("id = ?", testProduct.ID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), count, "Product should be deleted")
	})

	// Test ROW operation with OTel callbacks
	t.Run("Row with OTel callbacks", func(t *testing.T) {
		var productName string
		err := db.WithContext(ctx).Model(&Product{}).Where("is_available = ?", true).Select("name").Row().Scan(&productName)
		require.NoError(t, err, "Failed to execute row query with OTel callbacks")
		assert.NotEmpty(t, productName, "Should get a product name")
	})

	// Test RAW operation with OTel callbacks
	t.Run("Raw with OTel callbacks", func(t *testing.T) {
		var count int64
		err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM products WHERE is_available = ?", true).Scan(&count).Error
		require.NoError(t, err, "Failed to execute raw query with OTel callbacks")
		assert.Greater(t, count, int64(0), "Should have available products")
	})

	// Test error handling in OTel callbacks
	t.Run("Error in query with OTel callbacks", func(t *testing.T) {
		var product Product
		// Use a valid UUID format that doesn't exist
		nonExistentID := uuid.New().String()
		err := db.WithContext(ctx).First(&product, "id = ?", nonExistentID).Error
		assert.Error(t, err, "Should get error for non-existent product")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound, "Should be record not found error")
	})
}

// TestPostgresPoolWithOTelMetrics tests OTel metrics collection
func TestPostgresPoolWithOTelMetrics(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(filepath.Join("..", "scripts", "compose", "pg", "backup", "default.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	// Create OTel config with noop providers and metrics enabled
	otelConfig := pkgotel.NewConfig("db-metrics-test").
		WithTracerProvider(noopt.NewTracerProvider()).
		WithMeterProvider(noopm.NewMeterProvider()).
		WithLoggerProvider(noopl.NewLoggerProvider())

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
		OTelConfig:   otelConfig,
	}

	// Test Pool() with OTel metrics
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database with OTel metrics")
	require.NotNil(t, db, "Database connection should not be nil")

	// Get underlying sql.DB to check stats
	sqlDB, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB")

	// Wait a bit for metrics collection to start
	time.Sleep(100 * time.Millisecond)

	// Verify connection pool stats are available
	stats := sqlDB.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1, "Should have max open connections configured")
	assert.GreaterOrEqual(t, stats.Idle, 0, "Should have idle connections tracked")

	// Perform some operations to generate metrics
	var productCount int64
	err = db.Model(&Product{}).Count(&productCount).Error
	require.NoError(t, err, "Failed to count products")

	// Check stats again
	stats = sqlDB.Stats()
	assert.GreaterOrEqual(t, stats.InUse, 0, "Should track in-use connections")

	t.Logf("Connection pool stats - Idle: %d, InUse: %d, Max: %d", stats.Idle, stats.InUse, stats.MaxOpenConnections)
}

// TestPostgresPoolWithOTelDisabled tests when OTel is disabled
func TestPostgresPoolWithOTelDisabled(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(filepath.Join("..", "scripts", "compose", "pg", "backup", "default.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	// Test with nil OTel config (disabled)
	t.Run("Nil OTel config", func(t *testing.T) {
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
			OTelConfig:   nil,
		}

		db, err := config.Pool()
		require.NoError(t, err, "Failed to connect to database without OTel")
		require.NotNil(t, db, "Database connection should not be nil")

		// Test basic operation still works
		var count int64
		err = db.Model(&Product{}).Count(&count).Error
		require.NoError(t, err, "Failed to count products")
		assert.Greater(t, count, int64(0), "Should have products")
	})

	// Test with OTel config but tracing disabled
	t.Run("OTel config without tracer", func(t *testing.T) {
		otelConfig := pkgotel.NewConfig("db-no-trace-test").
			WithMeterProvider(noopm.NewMeterProvider()).
			WithLoggerProvider(noopl.NewLoggerProvider())
		// TracerProvider is nil

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
			OTelConfig:   otelConfig,
		}

		db, err := config.Pool()
		require.NoError(t, err, "Failed to connect to database with OTel but no tracer")
		require.NotNil(t, db, "Database connection should not be nil")

		// Test basic operation still works
		var count int64
		err = db.Model(&Product{}).Count(&count).Error
		require.NoError(t, err, "Failed to count products")
		assert.Greater(t, count, int64(0), "Should have products")
	})
}

// TestMySQLPoolWithOTel tests OTel with MySQL
func TestMySQLPoolWithOTel(t *testing.T) {
	ctx := context.Background()

	// Start MySQL container
	container, config := setupMySQLContainer(t)
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Add OTel config
	otelConfig := pkgotel.NewConfig("db-mysql-test").
		WithTracerProvider(noopt.NewTracerProvider()).
		WithMeterProvider(noopm.NewMeterProvider()).
		WithLoggerProvider(noopl.NewLoggerProvider())

	config.OTelConfig = otelConfig

	// Test Pool() with OTel
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to MySQL with OTel")
	require.NotNil(t, db, "Database connection should not be nil")

	// Test operations with OTel callbacks
	var productCount int64
	err = db.WithContext(ctx).Model(&Product{}).Count(&productCount).Error
	require.NoError(t, err, "Failed to count products with OTel")
	assert.Greater(t, productCount, int64(0), "Should have products")

	// Test create with OTel
	testProduct := Product{
		ID:            uuid.New().String(),
		Name:          "MySQL OTel Test",
		Description:   "Testing MySQL with OTel",
		Category:      "Test",
		Price:         99.99,
		StockQuantity: 10,
		IsAvailable:   true,
		CreatedAt:     time.Now(),
	}

	err = db.WithContext(ctx).Create(&testProduct).Error
	require.NoError(t, err, "Failed to create product in MySQL with OTel")

	// Cleanup
	db.Delete(&Product{}, "id = ?", testProduct.ID)
}

// TestMSSQLPoolWithOTel tests OTel with MSSQL
func TestMSSQLPoolWithOTel(t *testing.T) {
	ctx := context.Background()

	// Start MSSQL container
	container, config := setupMSSQLContainer(t)
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Add OTel config
	otelConfig := pkgotel.NewConfig("db-mssql-test").
		WithTracerProvider(noopt.NewTracerProvider()).
		WithMeterProvider(noopm.NewMeterProvider()).
		WithLoggerProvider(noopl.NewLoggerProvider())

	config.OTelConfig = otelConfig

	// Test Pool() with OTel
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to MSSQL with OTel")
	require.NotNil(t, db, "Database connection should not be nil")

	// Test operations with OTel callbacks
	var result int
	err = db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
	require.NoError(t, err, "Failed to execute query with OTel")
	assert.Equal(t, 1, result, "Query result should be 1")
}

// TestOTelCallbacksWithoutContext tests callbacks when context is nil
func TestOTelCallbacksWithoutContext(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(filepath.Join("..", "scripts", "compose", "pg", "backup", "default.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

	// Create OTel config
	otelConfig := pkgotel.NewConfig("db-no-ctx-test").
		WithTracerProvider(noopt.NewTracerProvider()).
		WithMeterProvider(noopm.NewMeterProvider()).
		WithLoggerProvider(noopl.NewLoggerProvider())

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
		OTelConfig:   otelConfig,
	}

	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database")
	require.NotNil(t, db, "Database connection should not be nil")

	// Execute query without context (Statement.Context will be nil in callback)
	var count int64
	err = db.Model(&Product{}).Count(&count).Error
	require.NoError(t, err, "Query without context should still work")
	assert.Greater(t, count, int64(0), "Should have products")
}

// TestPoolInvalidConfig tests error handling in Pool()
func TestPoolInvalidConfig(t *testing.T) {
	t.Run("Empty DSN", func(t *testing.T) {
		config := &ConnectionConfig{
			DbType:       "",
			Host:         "",
			Port:         0,
			Username:     "",
			Password:     "",
			DbName:       "",
			Timeout:      10 * time.Second,
			MaxIdleConns: 5,
			MaxOpenConns: 10,
		}

		db, err := config.Pool()
		assert.Error(t, err, "Should fail with invalid config")
		assert.Nil(t, db, "DB should be nil on error")
		assert.Contains(t, err.Error(), "unsupported database type", "Error should mention unsupported type")
	})

	t.Run("Unsupported database type", func(t *testing.T) {
		config := &ConnectionConfig{
			DbType:       DatabaseType("UNSUPPORTED"),
			Host:         "localhost",
			Port:         5432,
			Username:     "test",
			Password:     "test",
			DbName:       "testdb",
			Timeout:      10 * time.Second,
			MaxIdleConns: 5,
			MaxOpenConns: 10,
		}

		// DSN will be empty for unsupported type, so test DSN generation separately
		dsn := config.Dsn()
		assert.Equal(t, "", dsn, "DSN should be empty for unsupported database type")

		db, err := config.Pool()
		assert.Error(t, err, "Should fail with unsupported database type")
		assert.Nil(t, db, "DB should be nil on error")
		assert.Contains(t, err.Error(), "unsupported database type", "Error should mention unsupported type")
	})

	t.Run("Invalid connection parameters", func(t *testing.T) {
		config := &ConnectionConfig{
			DbType:       Postgresql,
			Host:         "invalid-host-that-does-not-exist-12345",
			Port:         9999,
			Username:     "test",
			Password:     "test",
			DbName:       "testdb",
			Timeout:      1 * time.Second,
			MaxIdleConns: 5,
			MaxOpenConns: 10,
		}

		db, err := config.Pool()
		assert.Error(t, err, "Should fail with invalid connection parameters")
		assert.Nil(t, db, "DB should be nil on error")
	})
}

// TestSQLDBErrorHandling tests SQLDB error handling
func TestSQLDBErrorHandling(t *testing.T) {
	t.Run("SQLDB with invalid connection", func(t *testing.T) {
		config := &ConnectionConfig{
			DbType:       Postgresql,
			Host:         "invalid-host-12345",
			Port:         9999,
			Username:     "test",
			Password:     "test",
			DbName:       "testdb",
			Timeout:      1 * time.Second,
			MaxIdleConns: 5,
			MaxOpenConns: 10,
		}

		sqlDB, err := config.SQLDB()
		assert.Error(t, err, "Should fail with invalid connection")
		assert.Nil(t, sqlDB, "SQL DB should be nil on error")
	})
}

// TestOTelCallbacksTableAndRowsAffected tests OTel callbacks with table names and rows affected
func TestOTelCallbacksTableAndRowsAffected(t *testing.T) {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		postgres.WithInitScripts(filepath.Join("..", "scripts", "compose", "pg", "backup", "default.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start PostgreSQL container")
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Create OTel config
	otelConfig := pkgotel.NewConfig("db-table-test").
		WithTracerProvider(noopt.NewTracerProvider()).
		WithMeterProvider(noopm.NewMeterProvider()).
		WithLoggerProvider(noopl.NewLoggerProvider())

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
		OTelConfig:   otelConfig,
	}

	db, err := config.Pool()
	require.NoError(t, err)

	// Test with table name in statement
	t.Run("Query with table name", func(t *testing.T) {
		var products []Product
		result := db.WithContext(ctx).Table("products").Where("is_available = ?", true).Limit(5).Find(&products)
		require.NoError(t, result.Error)
		assert.Greater(t, result.RowsAffected, int64(0), "Should have rows affected")
	})

	// Test batch update with multiple rows affected
	t.Run("Update with multiple rows affected", func(t *testing.T) {
		// Create multiple test products
		for i := 0; i < 3; i++ {
			testProduct := Product{
				ID:            uuid.New().String(),
				Name:          fmt.Sprintf("Batch Test Product %d", i),
				Description:   "Testing batch operations",
				Category:      "BatchTest",
				Price:         99.99,
				StockQuantity: 10,
				IsAvailable:   true,
				CreatedAt:     time.Now(),
			}
			err := db.Create(&testProduct).Error
			require.NoError(t, err)
		}

		// Update all batch test products
		result := db.WithContext(ctx).Model(&Product{}).Where("category = ?", "BatchTest").Update("price", 149.99)
		require.NoError(t, result.Error)
		assert.GreaterOrEqual(t, result.RowsAffected, int64(3), "Should affect multiple rows")

		// Cleanup
		db.Delete(&Product{}, "category = ?", "BatchTest")
	})
}
