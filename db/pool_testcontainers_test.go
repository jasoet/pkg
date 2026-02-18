//go:build integration

package db

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mssql"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

// Test models are defined in test_types.go

func setupPostgresContainer(t *testing.T) (*postgres.PostgresContainer, *ConnectionConfig) {
	ctx := context.Background()

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

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err, "Failed to get port")

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

	return postgresContainer, config
}

func setupMySQLContainer(t *testing.T) (*mysql.MySQLContainer, *ConnectionConfig) {
	ctx := context.Background()

	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("testdb"),
		mysql.WithUsername("testuser"),
		mysql.WithPassword("testpass"),
		mysql.WithScripts(filepath.Join("..", "scripts", "compose", "mariadb", "backup", "default.sql")),
		testcontainers.WithWaitStrategy(
			wait.ForLog("port: 3306  MySQL Community Server").
				WithStartupTimeout(90*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start MySQL container")

	// Wait a bit more for MySQL to be fully ready
	time.Sleep(3 * time.Second)

	host, err := mysqlContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := mysqlContainer.MappedPort(ctx, "3306")
	require.NoError(t, err, "Failed to get port")

	config := &ConnectionConfig{
		DbType:       Mysql,
		Host:         host,
		Port:         port.Int(),
		Username:     "testuser",
		Password:     "testpass",
		DbName:       "testdb",
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	return mysqlContainer, config
}

func setupMSSQLContainer(t *testing.T) (*mssql.MSSQLServerContainer, *ConnectionConfig) {
	ctx := context.Background()

	mssqlContainer, err := mssql.Run(ctx,
		"mcr.microsoft.com/mssql/server:2022-latest",
		mssql.WithAcceptEULA(),
		mssql.WithPassword("StrongPass123!"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("SQL Server is now ready for client connections").
				WithStartupTimeout(90*time.Second),
		),
	)
	require.NoError(t, err, "Failed to start MSSQL container")

	// Wait a bit more for SQL Server to be fully ready
	time.Sleep(5 * time.Second)

	host, err := mssqlContainer.Host(ctx)
	require.NoError(t, err, "Failed to get host")

	port, err := mssqlContainer.MappedPort(ctx, "1433")
	require.NoError(t, err, "Failed to get port")

	config := &ConnectionConfig{
		DbType:       MSSQL,
		Host:         host,
		Port:         port.Int(),
		Username:     "sa",
		Password:     "StrongPass123!",
		DbName:       "master",
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	return mssqlContainer, config
}

func TestPostgresPoolWithTestcontainers(t *testing.T) {
	container, config := setupPostgresContainer(t)
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Test the DSN generation
	dsn := config.Dsn()
	assert.Contains(t, dsn, "user=testuser")
	assert.Contains(t, dsn, "password=testpass")
	assert.Contains(t, dsn, "dbname=testdb")
	assert.Contains(t, dsn, "sslmode=disable")

	// Test connection to the database using Pool()
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database using Pool()")
	require.NotNil(t, db, "Database connection should not be nil")

	// Test basic query using GORM
	var productCount int64
	err = db.Model(&Product{}).Count(&productCount).Error
	require.NoError(t, err, "Failed to count products")
	assert.Greater(t, productCount, int64(0), "Should have at least one product")

	// Test retrieving a specific product
	var product Product
	err = db.First(&product, "id = ?", "11111111-1111-1111-1111-111111111111").Error
	require.NoError(t, err, "Failed to retrieve product")
	assert.Equal(t, "Wireless Headphones", product.Name, "Product name should match")

	// Test connection to the database using SQLDB()
	sqlDB, err := config.SQLDB()
	require.NoError(t, err, "Failed to connect to database using SQLDB()")
	require.NotNil(t, sqlDB, "SQL database connection should not be nil")
	defer sqlDB.Close()

	// Test basic query using sql.DB
	var customerCount int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM customers").Scan(&customerCount)
	require.NoError(t, err, "Failed to count customers")
	assert.Greater(t, customerCount, 0, "Should have at least one customer")

	// Test retrieving a specific customer
	var firstName, lastName string
	err = sqlDB.QueryRow("SELECT first_name, last_name FROM customers WHERE id = $1", "aaaaaaaa-1111-1111-1111-111111111111").Scan(&firstName, &lastName)
	require.NoError(t, err, "Failed to retrieve customer")
	assert.Equal(t, "Alice", firstName, "First name should match")
	assert.Equal(t, "Johnson", lastName, "Last name should match")

	// Test connection pool settings
	sqlDB2, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB from gorm.DB")

	// Verify the connection pool is working by checking stats
	stats := sqlDB2.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1, "Should allow connections")

	// Verify the connection is still working
	err = sqlDB2.Ping()
	require.NoError(t, err, "Connection pool should be working")
}

func TestMySQLPoolWithTestcontainers(t *testing.T) {
	container, config := setupMySQLContainer(t)
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Test the DSN generation
	dsn := config.Dsn()
	assert.Contains(t, dsn, fmt.Sprintf("testuser:testpass@tcp(%s:%d)/testdb", config.Host, config.Port))
	assert.Contains(t, dsn, "parseTime=true")

	// Test connection to the database using Pool()
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database using Pool()")
	require.NotNil(t, db, "Database connection should not be nil")

	// Test basic query using GORM
	var productCount int64
	err = db.Model(&Product{}).Count(&productCount).Error
	require.NoError(t, err, "Failed to count products")
	assert.Greater(t, productCount, int64(0), "Should have at least one product")

	// Test retrieving a specific product
	var product Product
	err = db.First(&product, "id = ?", "11111111-1111-1111-1111-111111111111").Error
	require.NoError(t, err, "Failed to retrieve product")
	assert.Equal(t, "Wireless Headphones", product.Name, "Product name should match")

	// Test connection to the database using SQLDB()
	sqlDB, err := config.SQLDB()
	require.NoError(t, err, "Failed to connect to database using SQLDB()")
	require.NotNil(t, sqlDB, "SQL database connection should not be nil")
	defer sqlDB.Close()

	// Test basic query using sql.DB
	var customerCount int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM customers").Scan(&customerCount)
	require.NoError(t, err, "Failed to count customers")
	assert.Greater(t, customerCount, 0, "Should have at least one customer")

	// Test retrieving a specific customer
	var firstName, lastName string
	err = sqlDB.QueryRow("SELECT first_name, last_name FROM customers WHERE id = ?", "aaaaaaaa-1111-1111-1111-111111111111").Scan(&firstName, &lastName)
	require.NoError(t, err, "Failed to retrieve customer")
	assert.Equal(t, "Alice", firstName, "First name should match")
	assert.Equal(t, "Johnson", lastName, "Last name should match")

	// Test connection pool settings
	sqlDB2, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB from gorm.DB")

	// Verify the connection pool is working by checking stats
	stats := sqlDB2.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1, "Should allow connections")

	// Verify the connection is still working
	err = sqlDB2.Ping()
	require.NoError(t, err, "Connection pool should be working")
}

func TestMSSQLPoolWithTestcontainers(t *testing.T) {
	container, config := setupMSSQLContainer(t)
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Test the DSN generation
	dsn := config.Dsn()
	assert.Contains(t, dsn, fmt.Sprintf("sqlserver://sa:StrongPass123!@%s:%d", config.Host, config.Port))
	assert.Contains(t, dsn, "database=master")
	assert.Contains(t, dsn, "encrypt=disable")

	// Test connection to the database using Pool()
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database using Pool()")
	require.NotNil(t, db, "Database connection should not be nil")

	// Test basic connectivity with a simple query
	var result int
	err = db.Raw("SELECT 1").Scan(&result).Error
	require.NoError(t, err, "Failed to execute simple query")
	assert.Equal(t, 1, result, "Query result should be 1")

	// Test connection to the database using SQLDB()
	sqlDB, err := config.SQLDB()
	require.NoError(t, err, "Failed to connect to database using SQLDB()")
	require.NotNil(t, sqlDB, "SQL database connection should not be nil")
	defer sqlDB.Close()

	// Test basic connectivity with a simple query using sql.DB
	err = sqlDB.QueryRow("SELECT 1").Scan(&result)
	require.NoError(t, err, "Failed to execute simple query with sql.DB")
	assert.Equal(t, 1, result, "Query result should be 1")

	// Test connection pool settings
	sqlDB2, err := db.DB()
	require.NoError(t, err, "Failed to get sql.DB from gorm.DB")

	// Verify the connection pool is working by checking stats
	stats := sqlDB2.Stats()
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1, "Should allow connections")

	// Verify the connection is still working
	err = sqlDB2.Ping()
	require.NoError(t, err, "Connection pool should be working")
}

func TestPostgresPoolTransactionsWithTestcontainers(t *testing.T) {
	container, config := setupPostgresContainer(t)
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Connect to the database
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database")

	// Test transaction with commit
	err = db.Transaction(func(tx *gorm.DB) error {
		// Create a new test product with valid UUID format
		testProduct := Product{
			ID:            "11111111-abcd-1234-abcd-111111111111",
			Name:          "Test Product",
			Description:   "Product for testing transactions",
			Category:      "Test",
			Price:         99.99,
			StockQuantity: 100,
			IsAvailable:   true,
			CreatedAt:     time.Now(),
		}

		// Insert the product
		if err := tx.Create(&testProduct).Error; err != nil {
			return err
		}

		// Verify the product was inserted
		var count int64
		if err := tx.Model(&Product{}).Where("id = ?", "11111111-abcd-1234-abcd-111111111111").Count(&count).Error; err != nil {
			return err
		}

		if count != 1 {
			t.Errorf("Expected 1 product, got %d", count)
		}

		return nil
	})
	require.NoError(t, err, "Transaction should commit successfully")

	// Verify the product exists after commit
	var product Product
	err = db.First(&product, "id = ?", "11111111-abcd-1234-abcd-111111111111").Error
	require.NoError(t, err, "Product should exist after transaction commit")
	assert.Equal(t, "Test Product", product.Name, "Product name should match")

	// Test transaction with rollback
	err = db.Transaction(func(tx *gorm.DB) error {
		// Create another test product with valid UUID format
		testProduct2 := Product{
			ID:            "22222222-abcd-1234-abcd-222222222222",
			Name:          "Test Product Rollback",
			Description:   "Product for testing transaction rollback",
			Category:      "Test",
			Price:         199.99,
			StockQuantity: 50,
			IsAvailable:   true,
			CreatedAt:     time.Now(),
		}

		// Insert the product
		if err := tx.Create(&testProduct2).Error; err != nil {
			return err
		}

		// Force a rollback
		return sql.ErrConnDone
	})
	require.Error(t, err, "Transaction should fail and rollback")

	// Verify the product does not exist after rollback
	var count int64
	err = db.Model(&Product{}).Where("id = ?", "22222222-abcd-1234-abcd-222222222222").Count(&count).Error
	require.NoError(t, err, "Count query should succeed")
	assert.Equal(t, int64(0), count, "Product should not exist after transaction rollback")

	// Clean up the test product
	db.Delete(&Product{}, "id = ?", "11111111-abcd-1234-abcd-111111111111")
}
