//go:build integration

package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// MariaDB connection constants
const (
	mariaDBHost     = "localhost"
	mariaDBPort     = 3309
	mariaDBUser     = "jasoet"
	mariaDBPassword = "localhost"
	mariaDBName     = "pkg_db"
)

// SQL Server connection constants
const (
	sqlServerHost     = "localhost"
	sqlServerPort     = 1439
	sqlServerUser     = "sa"
	sqlServerPassword = "Localhost12$"
	sqlServerName     = "msdb"
)

// Using constants defined in migration_integration_test.go

// Test models are defined in test_types.go

func TestPostgresPoolConnection(t *testing.T) {
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

	// Test the DSN generation
	dsn := config.Dsn()
	assert.Contains(t, dsn, "user=jasoet")
	assert.Contains(t, dsn, "password=localhost")
	assert.Contains(t, dsn, "host=localhost")
	assert.Contains(t, dsn, "port=5439")
	assert.Contains(t, dsn, "dbname=pkg_db")
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

	// Test connection to the database using SqlDB()
	sqlDB, err := config.SqlDB()
	require.NoError(t, err, "Failed to connect to database using SqlDB()")
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
	// We can't directly check MaxIdleConns and MaxOpenConns as they're not exposed in Stats
	// But we can verify the pool is working by checking other stats
	assert.GreaterOrEqual(t, stats.MaxOpenConnections, 1, "Should allow connections")

	// We can also verify the connection is still working
	err = sqlDB2.Ping()
	require.NoError(t, err, "Connection pool should be working")
}

func TestMariaDBPoolConnection(t *testing.T) {
	// Create a connection config using values from docker-compose.yml
	config := &ConnectionConfig{
		DbType:       Mysql,
		Host:         mariaDBHost,
		Port:         mariaDBPort,
		Username:     mariaDBUser,
		Password:     mariaDBPassword,
		DbName:       mariaDBName,
		Timeout:      dbTimeout,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	// Test the DSN generation
	dsn := config.Dsn()
	assert.Contains(t, dsn, "jasoet:localhost@tcp(localhost:3309)/pkg_db")
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

	// Test connection to the database using SqlDB()
	sqlDB, err := config.SqlDB()
	require.NoError(t, err, "Failed to connect to database using SqlDB()")
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

	// We can also verify the connection is still working
	err = sqlDB2.Ping()
	require.NoError(t, err, "Connection pool should be working")
}

func TestSQLServerPoolConnection(t *testing.T) {
	// Create a connection config using values from docker-compose.yml
	config := &ConnectionConfig{
		DbType:       MSSQL,
		Host:         sqlServerHost,
		Port:         sqlServerPort,
		Username:     sqlServerUser,
		Password:     sqlServerPassword,
		DbName:       sqlServerName,
		Timeout:      dbTimeout,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	// Test the DSN generation
	dsn := config.Dsn()
	assert.Contains(t, dsn, "sqlserver://sa:Localhost12$@localhost:1439")
	assert.Contains(t, dsn, "database=msdb")
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

	// Test connection to the database using SqlDB()
	sqlDB, err := config.SqlDB()
	require.NoError(t, err, "Failed to connect to database using SqlDB()")
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

	// We can also verify the connection is still working
	err = sqlDB2.Ping()
	require.NoError(t, err, "Connection pool should be working")
}

func TestPostgresPoolTransactions(t *testing.T) {
	// Create a connection config
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
	db, err := config.Pool()
	require.NoError(t, err, "Failed to connect to database")

	// Test transaction with commit
	err = db.Transaction(func(tx *gorm.DB) error {
		// Create a new test product with valid UUID format
		testProduct := Product{
			ID:            "11111111-abcd-1234-abcd-111111111111", // Valid UUID format
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
			ID:            "22222222-abcd-1234-abcd-222222222222", // Valid UUID format
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
