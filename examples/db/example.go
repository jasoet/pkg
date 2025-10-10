//go:build example

package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jasoet/pkg/v2/db"
	"github.com/jasoet/pkg/v2/logging"
	"gorm.io/gorm"
)

// Embed migration files for examples
//
//go:embed migrations/*.sql
var migrationFS embed.FS

// Example model structures
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"not null" json:"name"`
	Email     string    `gorm:"uniqueIndex" json:"email"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `json:"description"`
	Price       float64   `gorm:"not null" json:"price"`
	Stock       int       `gorm:"default:0" json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Order struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	Total     float64   `gorm:"not null" json:"total"`
	Status    string    `gorm:"default:'pending'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func main() {
	// Initialize logging
	logging.Initialize("db-examples", true)
	ctx := context.Background()

	fmt.Println("DB Package Examples")
	fmt.Println("===================")

	// Example 1: Basic Database Connection
	fmt.Println("\n1. Basic Database Connection")
	basicConnectionExample(ctx)

	// Example 2: Connection Pool Configuration
	fmt.Println("\n2. Connection Pool Configuration")
	connectionPoolExample(ctx)

	// Example 3: Database Migrations
	fmt.Println("\n3. Database Migrations")
	migrationExample(ctx)

	// Example 4: Multiple Database Connections
	fmt.Println("\n4. Multiple Database Connections")
	multipleConnectionsExample(ctx)

	// Example 5: GORM Model Operations
	fmt.Println("\n5. GORM Model Operations")
	gormOperationsExample(ctx)

	// Example 6: Raw SQL with Connection Pool
	fmt.Println("\n6. Raw SQL with Connection Pool")
	rawSQLExample(ctx)

	// Example 7: Transaction Management
	fmt.Println("\n7. Transaction Management")
	transactionExample(ctx)

	// Example 8: Health Checks and Monitoring
	fmt.Println("\n8. Health Checks and Monitoring")
	healthCheckExample(ctx)
}

func basicConnectionExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "basic-connection")

	// PostgreSQL connection configuration
	config := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getIntEnvOrDefault("DB_PORT", 5439),
		Username:     getEnvOrDefault("DB_USER", "jasoet"),
		Password:     getEnvOrDefault("DB_PASSWORD", "localhost"),
		DbName:       getEnvOrDefault("DB_NAME", "pkg_db"),
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 25,
	}

	logger.Info().
		Str("host", config.Host).
		Int("port", config.Port).
		Str("database", config.DbName).
		Msg("Connecting to PostgreSQL database")

	fmt.Printf("Database Configuration:\n")
	fmt.Printf("- Type: %s\n", config.DbType)
	fmt.Printf("- Host: %s:%d\n", config.Host, config.Port)
	fmt.Printf("- Database: %s\n", config.DbName)
	fmt.Printf("- DSN: %s\n", maskPassword(config.Dsn()))

	// Connect to database
	database, err := config.Pool()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to database")
		fmt.Printf("✗ Connection failed: %v\n", err)
		return
	}

	// Test connection
	sqlDB, err := database.DB()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get SQL DB")
		fmt.Printf("✗ Failed to get SQL DB: %v\n", err)
		return
	}

	if err := sqlDB.Ping(); err != nil {
		logger.Error().Err(err).Msg("Database ping failed")
		fmt.Printf("✗ Database ping failed: %v\n", err)
		return
	}

	logger.Info().Msg("Database connection successful")
	fmt.Println("✓ Database connection successful")

	// Display connection stats
	stats := sqlDB.Stats()
	fmt.Printf("Connection Pool Stats:\n")
	fmt.Printf("- Open Connections: %d\n", stats.OpenConnections)
	fmt.Printf("- Idle Connections: %d\n", stats.Idle)
	fmt.Printf("- Max Open Connections: %d\n", stats.MaxOpenConnections)
}

func connectionPoolExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "connection-pool")

	// Different configuration for different environments
	configs := map[string]*db.ConnectionConfig{
		"development": {
			DbType:       db.Postgresql,
			Host:         "localhost",
			Port:         5439,
			Username:     "jasoet",
			Password:     "localhost",
			DbName:       "pkg_db",
			Timeout:      10 * time.Second,
			MaxIdleConns: 5,  // Small for development
			MaxOpenConns: 25, // Moderate for development
		},
		"production": {
			DbType:       db.Postgresql,
			Host:         "prod-db.example.com",
			Port:         5432,
			Username:     "app_user",
			Password:     "secure_password",
			DbName:       "production_db",
			Timeout:      30 * time.Second,
			MaxIdleConns: 10,  // More idle connections
			MaxOpenConns: 100, // Higher concurrency
		},
		"analytics": {
			DbType:       db.Mysql,
			Host:         "analytics.example.com",
			Port:         3306,
			Username:     "analytics",
			Password:     "analytics_pass",
			DbName:       "analytics_db",
			Timeout:      60 * time.Second,
			MaxIdleConns: 3,  // Fewer idle connections
			MaxOpenConns: 50, // Medium concurrency
		},
	}

	for env, config := range configs {
		fmt.Printf("\n%s Configuration:\n", env)
		fmt.Printf("- Database Type: %s\n", config.DbType)
		fmt.Printf("- Max Idle Connections: %d\n", config.MaxIdleConns)
		fmt.Printf("- Max Open Connections: %d\n", config.MaxOpenConns)
		fmt.Printf("- Timeout: %v\n", config.Timeout)

		// Only attempt to connect to development database
		if env == "development" {
			if database, err := config.Pool(); err != nil {
				logger.Warn().Err(err).Str("env", env).Msg("Failed to connect")
				fmt.Printf("  ✗ Connection failed (expected for demo)\n")
			} else {
				// Demonstrate connection pool usage
				demonstrateConnectionPool(ctx, database)
			}
		} else {
			fmt.Printf("  ℹ Configuration shown for reference (connection not attempted)\n")
		}
	}
}

func demonstrateConnectionPool(ctx context.Context, database *gorm.DB) {
	sqlDB, err := database.DB()
	if err != nil {
		return
	}

	// Show initial stats
	stats := sqlDB.Stats()
	fmt.Printf("  - Initial open connections: %d\n", stats.OpenConnections)

	// Simulate concurrent connections
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			// Each goroutine gets a connection from the pool
			var result int
			err := sqlDB.QueryRowContext(ctx, "SELECT 1").Scan(&result)
			if err != nil {
				fmt.Printf("  - Query %d failed: %v\n", id, err)
			} else {
				fmt.Printf("  - Query %d completed successfully\n", id)
			}
			done <- true
		}(i)
	}

	// Wait for all queries to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Show final stats
	stats = sqlDB.Stats()
	fmt.Printf("  - Final open connections: %d\n", stats.OpenConnections)
	fmt.Printf("  - Total connections opened: %d\n", stats.MaxOpenConnections)
}

func migrationExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "migrations")

	// Note: This example shows the migration pattern but doesn't run actual migrations
	// since we don't have migration files in the example
	fmt.Println("Migration Example (conceptual - requires actual migration files)")

	config := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getIntEnvOrDefault("DB_PORT", 5439),
		Username:     getEnvOrDefault("DB_USER", "jasoet"),
		Password:     getEnvOrDefault("DB_PASSWORD", "localhost"),
		DbName:       getEnvOrDefault("DB_NAME", "pkg_db"),
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 25,
	}

	database, err := config.Pool()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to database")
		fmt.Printf("✗ Database connection failed: %v\n", err)
		return
	}

	fmt.Println("Migration process would include:")
	fmt.Println("1. Embed migration files:")
	fmt.Println("   //go:embed migrations/*.sql")
	fmt.Println("   var migrationFS embed.FS")
	fmt.Println()
	fmt.Println("2. Run migrations up:")
	fmt.Println("   err := db.RunPostgresMigrationsWithGorm(ctx, database, migrationFS, \"migrations\")")
	fmt.Println()
	fmt.Println("3. Migration file structure:")
	fmt.Println("   migrations/")
	fmt.Println("   ├── 001_initial_schema.up.sql")
	fmt.Println("   ├── 001_initial_schema.down.sql")
	fmt.Println("   ├── 002_add_users_table.up.sql")
	fmt.Println("   └── 002_add_users_table.down.sql")

	// Demonstrate the migration function call (would fail without actual files)
	logger.Info().Msg("Running database migrations")
	err = db.RunPostgresMigrationsWithGorm(ctx, database, migrationFS, "migrations")
	if err != nil {
		logger.Error().Err(err).Msg("Migration failed")
		fmt.Printf("✗ Migration failed: %v\n", err)
		return
	}
	logger.Info().Msg("Migrations completed successfully")
	fmt.Println("✓ Migrations completed successfully")
	fmt.Println("✓ Migrations completed successfully")

	fmt.Println("✓ Migration example completed (conceptual)")
}

func multipleConnectionsExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "multiple-connections")

	// Define multiple database configurations
	databases := map[string]*db.ConnectionConfig{
		"primary": {
			DbType:       db.Postgresql,
			Host:         getEnvOrDefault("PRIMARY_DB_HOST", "localhost"),
			Port:         getIntEnvOrDefault("PRIMARY_DB_PORT", 5439),
			Username:     getEnvOrDefault("PRIMARY_DB_USER", "jasoet"),
			Password:     getEnvOrDefault("PRIMARY_DB_PASSWORD", "localhost"),
			DbName:       getEnvOrDefault("PRIMARY_DB_NAME", "pkg_db"),
			MaxIdleConns: 5,
			MaxOpenConns: 25,
		},
		"analytics": {
			DbType:       db.Mysql,
			Host:         getEnvOrDefault("ANALYTICS_DB_HOST", "analytics.example.com"),
			Port:         getIntEnvOrDefault("ANALYTICS_DB_PORT", 3306),
			Username:     getEnvOrDefault("ANALYTICS_DB_USER", "analytics"),
			Password:     getEnvOrDefault("ANALYTICS_DB_PASSWORD", "password"),
			DbName:       getEnvOrDefault("ANALYTICS_DB_NAME", "analytics"),
			MaxIdleConns: 3,
			MaxOpenConns: 15,
		},
		"cache": {
			DbType:       db.MSSQL,
			Host:         getEnvOrDefault("CACHE_DB_HOST", "cache.example.com"),
			Port:         getIntEnvOrDefault("CACHE_DB_PORT", 1433),
			Username:     getEnvOrDefault("CACHE_DB_USER", "cache_user"),
			Password:     getEnvOrDefault("CACHE_DB_PASSWORD", "password"),
			DbName:       getEnvOrDefault("CACHE_DB_NAME", "cache_db"),
			MaxIdleConns: 2,
			MaxOpenConns: 10,
		},
	}

	connections := make(map[string]*gorm.DB)

	for name, config := range databases {
		// Only connect to primary database for demo
		if name == "primary" {
			database, err := config.Pool()
			if err != nil {
				logger.Error().Err(err).Str("database", name).Msg("Connection failed")
				fmt.Printf("✗ %s connection failed: %v\n", name, err)
				continue
			}
			connections[name] = database
			logger.Info().Str("database", name).Msg("Connection successful")
			fmt.Printf("✓ %s connection successful\n", name)
		} else {
			fmt.Printf("ℹ %s connection skipped for demo (would connect to %s:%d)\n",
				name, config.Host, config.Port)
		}
	}

	if primaryDB, exists := connections["primary"]; exists {
		demonstrateMultiDBOperations(ctx, primaryDB)
	}

	fmt.Printf("\nMultiple database pattern allows:\n")
	fmt.Printf("- Primary database for core application data\n")
	fmt.Printf("- Analytics database for reporting and metrics\n")
	fmt.Printf("- Cache database for session and temporary data\n")
	fmt.Printf("- Different database types for optimal use cases\n")
}

func demonstrateMultiDBOperations(ctx context.Context, primaryDB *gorm.DB) {
	logger := logging.ContextLogger(ctx, "multi-db-operations")

	// Auto-migrate tables
	err := primaryDB.AutoMigrate(&User{}, &Product{}, &Order{})
	if err != nil {
		logger.Error().Err(err).Msg("Auto-migration failed")
		return
	}

	// Create sample data
	user := User{Name: "John Doe", Email: "john@example.com"}
	result := primaryDB.Create(&user)
	if result.Error != nil {
		logger.Error().Err(result.Error).Msg("Failed to create user")
	} else {
		fmt.Printf("✓ Created user: %s (ID: %d)\n", user.Name, user.ID)
	}

	// Query data
	var users []User
	primaryDB.Find(&users)
	fmt.Printf("✓ Found %d users in primary database\n", len(users))
}

func gormOperationsExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "gorm-operations")

	config := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getIntEnvOrDefault("DB_PORT", 5439),
		Username:     getEnvOrDefault("DB_USER", "jasoet"),
		Password:     getEnvOrDefault("DB_PASSWORD", "localhost"),
		DbName:       getEnvOrDefault("DB_NAME", "pkg_db"),
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 25,
	}

	database, err := config.Pool()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to database")
		fmt.Printf("✗ Database connection failed: %v\n", err)
		return
	}

	// Auto-migrate the schema
	err = database.AutoMigrate(&User{}, &Product{}, &Order{})
	if err != nil {
		logger.Error().Err(err).Msg("Auto-migration failed")
		fmt.Printf("✗ Auto-migration failed: %v\n", err)
		return
	}
	fmt.Println("✓ Database schema migrated")

	// Create operations

	// Create users
	users := []User{
		{Name: "Alice Johnson", Email: "alice@example.com"},
		{Name: "Bob Smith", Email: "bob@example.com"},
		{Name: "Charlie Brown", Email: "charlie@example.com"},
	}

	for _, user := range users {
		result := database.Create(&user)
		if result.Error != nil {
			logger.Error().Err(result.Error).Str("name", user.Name).Msg("Failed to create user")
		} else {
			fmt.Printf("✓ Created user: %s (ID: %d)\n", user.Name, user.ID)
		}
	}

	// Create products
	products := []Product{
		{Name: "Laptop", Description: "High-performance laptop", Price: 999.99, Stock: 10},
		{Name: "Mouse", Description: "Wireless mouse", Price: 29.99, Stock: 50},
		{Name: "Keyboard", Description: "Mechanical keyboard", Price: 79.99, Stock: 25},
	}

	for _, product := range products {
		result := database.Create(&product)
		if result.Error != nil {
			logger.Error().Err(result.Error).Str("name", product.Name).Msg("Failed to create product")
		} else {
			fmt.Printf("✓ Created product: %s (ID: %d, Price: $%.2f)\n",
				product.Name, product.ID, product.Price)
		}
	}

	// Read operations

	var allUsers []User
	database.Find(&allUsers)
	fmt.Printf("✓ Found %d users\n", len(allUsers))

	var activeUsers []User
	database.Where("active = ?", true).Find(&activeUsers)
	fmt.Printf("✓ Found %d active users\n", len(activeUsers))

	var expensiveProducts []Product
	database.Where("price > ?", 50.0).Find(&expensiveProducts)
	fmt.Printf("✓ Found %d products over $50\n", len(expensiveProducts))

	// Update operations

	if len(allUsers) > 0 {
		user := allUsers[0]
		database.Model(&user).Update("Email", "newemail@example.com")
		fmt.Printf("✓ Updated user %s email\n", user.Name)
	}

	// Update multiple records
	database.Model(&Product{}).Where("stock < ?", 30).Update("stock", gorm.Expr("stock + ?", 10))
	fmt.Println("✓ Restocked products with low inventory")

	// Delete operations (soft delete since we have gorm.Model)

	if len(allUsers) > 2 {
		user := allUsers[2]
		database.Delete(&user)
		fmt.Printf("✓ Deleted user: %s\n", user.Name)
	}

	// Cleanup - actually delete records for demo
	database.Unscoped().Delete(&User{})
	database.Unscoped().Delete(&Product{})
	database.Unscoped().Delete(&Order{})
	fmt.Println("✓ Cleaned up demo data")
}

func rawSQLExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "raw-sql")

	config := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getIntEnvOrDefault("DB_PORT", 5439),
		Username:     getEnvOrDefault("DB_USER", "jasoet"),
		Password:     getEnvOrDefault("DB_PASSWORD", "localhost"),
		DbName:       getEnvOrDefault("DB_NAME", "pkg_db"),
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 25,
	}

	// Get SQL DB directly
	sqlDB, err := config.SQLDB()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get SQL DB")
		fmt.Printf("✗ Failed to get SQL DB: %v\n", err)
		return
	}
	defer sqlDB.Close()

	fmt.Println("Raw SQL Examples:")

	// Create a simple table for demonstration
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS demo_users (
		id SERIAL PRIMARY KEY,
		NAME VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = sqlDB.ExecContext(ctx, createTableSQL)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create table")
		fmt.Printf("✗ Failed to create table: %v\n", err)
		return
	}
	fmt.Println("✓ Created demo_users table")

	// Insert data
	insertSQL := `INSERT INTO demo_users (name, email) VALUES ($1, $2) RETURNING id`
	var userID int
	err = sqlDB.QueryRowContext(ctx, insertSQL, "John Doe", "john@example.com").Scan(&userID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to insert user")
		fmt.Printf("✗ Failed to insert user: %v\n", err)
	} else {
		fmt.Printf("✓ Inserted user with ID: %d\n", userID)
	}

	// Insert multiple users
	insertMultipleSQL := `INSERT INTO demo_users (name, email) VALUES ($1, $2), ($3, $4), ($5, $6)`
	result, err := sqlDB.ExecContext(ctx, insertMultipleSQL,
		"Alice Smith", "alice@example.com",
		"Bob Johnson", "bob@example.com",
		"Carol Williams", "carol@example.com")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to insert multiple users")
	} else {
		rowsAffected, _ := result.RowsAffected()
		fmt.Printf("✓ Inserted %d users\n", rowsAffected)
	}

	// Query data
	querySQL := `SELECT id, name, email, created_at FROM demo_users ORDER BY created_at DESC`
	rows, err := sqlDB.QueryContext(ctx, querySQL)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to query users")
		fmt.Printf("✗ Failed to query users: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("\nQueried Users:")
	for rows.Next() {
		var id int
		var name, email string
		var createdAt time.Time
		err := rows.Scan(&id, &name, &email, &createdAt)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to scan row")
			continue
		}
		fmt.Printf("- ID: %d, Name: %s, Email: %s, Created: %s\n",
			id, name, email, createdAt.Format("2006-01-02 15:04:05"))
	}

	// Parameterized query
	searchSQL := `SELECT name, email FROM demo_users WHERE name ILIKE $1`
	rows, err = sqlDB.QueryContext(ctx, searchSQL, "%john%")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to search users")
	} else {
		defer rows.Close()
		fmt.Println("\nUsers matching 'john':")
		for rows.Next() {
			var name, email string
			rows.Scan(&name, &email)
			fmt.Printf("- %s (%s)\n", name, email)
		}
	}

	// Aggregate query
	countSQL := `SELECT COUNT(*) AS total_users FROM demo_users`
	var totalUsers int
	err = sqlDB.QueryRowContext(ctx, countSQL).Scan(&totalUsers)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to count users")
	} else {
		fmt.Printf("✓ Total users in database: %d\n", totalUsers)
	}

	// Cleanup
	_, err = sqlDB.ExecContext(ctx, "DROP TABLE IF EXISTS demo_users")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cleanup table")
	} else {
		fmt.Println("✓ Cleaned up demo table")
	}
}

func transactionExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "transactions")

	config := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getIntEnvOrDefault("DB_PORT", 5439),
		Username:     getEnvOrDefault("DB_USER", "jasoet"),
		Password:     getEnvOrDefault("DB_PASSWORD", "localhost"),
		DbName:       getEnvOrDefault("DB_NAME", "pkg_db"),
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 25,
	}

	database, err := config.Pool()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to database")
		fmt.Printf("✗ Database connection failed: %v\n", err)
		return
	}

	// Auto-migrate for transaction example
	err = database.AutoMigrate(&User{}, &Order{})
	if err != nil {
		logger.Error().Err(err).Msg("Auto-migration failed")
		return
	}

	fmt.Println("Transaction Examples:")

	// Example 1: Successful transaction
	fmt.Println("\n1. Successful Transaction:")
	err = database.Transaction(func(tx *gorm.DB) error {
		// Create user
		user := User{Name: "Transaction User", Email: "transaction@example.com"}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		fmt.Printf("   ✓ Created user: %s (ID: %d)\n", user.Name, user.ID)

		// Create order for the user
		order := Order{UserID: user.ID, Total: 99.99, Status: "pending"}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		fmt.Printf("   ✓ Created order: ID %d, Total: $%.2f\n", order.ID, order.Total)

		return nil // Commit transaction
	})

	if err != nil {
		logger.Error().Err(err).Msg("Transaction failed")
		fmt.Printf("   ✗ Transaction failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Transaction completed successfully\n")
	}

	// Example 2: Failed transaction (rollback)
	fmt.Println("\n2. Failed Transaction (Rollback):")
	err = database.Transaction(func(tx *gorm.DB) error {
		// Create user
		user := User{Name: "Rollback User", Email: "rollback@example.com"}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		fmt.Printf("   ✓ Created user: %s (ID: %d)\n", user.Name, user.ID)

		// Simulate error condition
		return errors.New("simulated error - transaction will rollback")
	})
	if err != nil {
		fmt.Printf("   ✓ Transaction rolled back as expected: %v\n", err)
	}

	// Verify rollback - user should not exist
	var rollbackUser User
	result := database.Where("email = ?", "rollback@example.com").First(&rollbackUser)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		fmt.Printf("   ✓ Confirmed: rollback user was not saved\n")
	}

	// Example 3: Manual transaction control
	fmt.Println("\n3. Manual Transaction Control:")
	tx := database.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Error().Interface("panic", r).Msg("Transaction panic, rolling back")
		}
	}()

	if err := tx.Error; err != nil {
		fmt.Printf("   ✗ Failed to begin transaction: %v\n", err)
		return
	}

	// Create user
	user := User{Name: "Manual Transaction User", Email: "manual@example.com"}
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		fmt.Printf("   ✗ Failed to create user, rolling back: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Created user: %s (ID: %d)\n", user.Name, user.ID)

	// Update user
	if err := tx.Model(&user).Update("active", false).Error; err != nil {
		tx.Rollback()
		fmt.Printf("   ✗ Failed to update user, rolling back: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Updated user status\n")

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		fmt.Printf("   ✗ Failed to commit transaction: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Transaction committed successfully\n")

	// Example 4: Transaction with context timeout

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = database.WithContext(timeoutCtx).Transaction(func(tx *gorm.DB) error {
		user := User{Name: "Timeout User", Email: "timeout@example.com"}
		return tx.Create(&user).Error
	})

	if err != nil {
		fmt.Printf("   ✗ Transaction with timeout failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Transaction with timeout completed\n")
	}

	// Cleanup
	database.Unscoped().Delete(&User{})
	database.Unscoped().Delete(&Order{})
	fmt.Println("\n✓ Cleaned up transaction demo data")
}

func healthCheckExample(ctx context.Context) {
	logger := logging.ContextLogger(ctx, "health-check")

	config := &db.ConnectionConfig{
		DbType:       db.Postgresql,
		Host:         getEnvOrDefault("DB_HOST", "localhost"),
		Port:         getIntEnvOrDefault("DB_PORT", 5439),
		Username:     getEnvOrDefault("DB_USER", "jasoet"),
		Password:     getEnvOrDefault("DB_PASSWORD", "localhost"),
		DbName:       getEnvOrDefault("DB_NAME", "pkg_db"),
		Timeout:      10 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 25,
	}

	database, err := config.Pool()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to database")
		fmt.Printf("✗ Database connection failed: %v\n", err)
		return
	}

	fmt.Println("Database Health Check Examples:")

	// Basic health check
	fmt.Println("\n1. Basic Health Check:")
	if err := checkDatabaseHealth(ctx, database); err != nil {
		fmt.Printf("   ✗ Health check failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Database is healthy\n")
	}

	// Detailed health check with metrics
	fmt.Println("\n2. Detailed Health Check with Metrics:")
	healthInfo := getDatabaseHealthInfo(ctx, database)
	fmt.Printf("   Database Status: %s\n", healthInfo.Status)
	fmt.Printf("   Response Time: %v\n", healthInfo.ResponseTime)
	fmt.Printf("   Open Connections: %d\n", healthInfo.OpenConnections)
	fmt.Printf("   Idle Connections: %d\n", healthInfo.IdleConnections)
	fmt.Printf("   Max Open Connections: %d\n", healthInfo.MaxOpenConnections)

	// Connection pool monitoring
	fmt.Println("\n3. Connection Pool Monitoring:")
	monitorConnectionPool(ctx, database)

	// Database query performance test
	fmt.Println("\n4. Query Performance Test:")
	testQueryPerformance(ctx, database)
}

type DatabaseHealthInfo struct {
	Status             string        `json:"status"`
	ResponseTime       time.Duration `json:"response_time"`
	OpenConnections    int           `json:"open_connections"`
	IdleConnections    int           `json:"idle_connections"`
	MaxOpenConnections int           `json:"max_open_connections"`
	Error              string        `json:"error,omitempty"`
}

func checkDatabaseHealth(ctx context.Context, database *gorm.DB) error {
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Ping with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

func getDatabaseHealthInfo(ctx context.Context, database *gorm.DB) DatabaseHealthInfo {
	info := DatabaseHealthInfo{
		Status: "unknown",
	}

	start := time.Now()

	sqlDB, err := database.DB()
	if err != nil {
		info.Status = "error"
		info.Error = err.Error()
		return info
	}

	// Test connection with ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		info.Status = "unhealthy"
		info.Error = err.Error()
		info.ResponseTime = time.Since(start)
		return info
	}

	info.ResponseTime = time.Since(start)
	info.Status = "healthy"

	// Get connection pool stats
	stats := sqlDB.Stats()
	info.OpenConnections = stats.OpenConnections
	info.IdleConnections = stats.Idle
	info.MaxOpenConnections = stats.MaxOpenConnections

	return info
}

func monitorConnectionPool(ctx context.Context, database *gorm.DB) {
	sqlDB, err := database.DB()
	if err != nil {
		fmt.Printf("   ✗ Failed to get SQL DB: %v\n", err)
		return
	}

	stats := sqlDB.Stats()
	fmt.Printf("   Connection Pool Statistics:\n")
	fmt.Printf("   - Open Connections: %d\n", stats.OpenConnections)
	fmt.Printf("   - Connections In Use: %d\n", stats.InUse)
	fmt.Printf("   - Idle Connections: %d\n", stats.Idle)
	fmt.Printf("   - Wait Count: %d\n", stats.WaitCount)
	fmt.Printf("   - Wait Duration: %v\n", stats.WaitDuration)
	fmt.Printf("   - Max Idle Closed: %d\n", stats.MaxIdleClosed)
	fmt.Printf("   - Max Idle Time Closed: %d\n", stats.MaxIdleTimeClosed)
	fmt.Printf("   - Max Lifetime Closed: %d\n", stats.MaxLifetimeClosed)

	// Check if connection pool is healthy
	if stats.WaitCount > 1000 {
		fmt.Printf("   ⚠ Warning: High wait count indicates connection pool pressure\n")
	}
	if stats.OpenConnections >= stats.MaxOpenConnections {
		fmt.Printf("   ⚠ Warning: Connection pool at maximum capacity\n")
	}
}

func testQueryPerformance(ctx context.Context, database *gorm.DB) {
	// Simple query performance test
	start := time.Now()
	var result int
	err := database.Raw("SELECT 1").Scan(&result).Error
	queryTime := time.Since(start)

	if err != nil {
		fmt.Printf("   ✗ Query failed: %v\n", err)
	} else {
		fmt.Printf("   ✓ Query executed successfully in %v\n", queryTime)
		if queryTime > 100*time.Millisecond {
			fmt.Printf("   ⚠ Warning: Query took longer than expected\n")
		}
	}

	// Test multiple concurrent queries
	start = time.Now()
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			var result int
			database.Raw("SELECT 1").Scan(&result)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
	concurrentTime := time.Since(start)
	fmt.Printf("   ✓ 10 concurrent queries executed in %v\n", concurrentTime)
}

// Utility functions
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && intValue == 1 {
			return defaultValue
		}
	}
	return defaultValue
}

func maskPassword(dsn string) string {
	// Simple password masking for display purposes
	if len(dsn) > 50 {
		return dsn[:20] + "***masked***" + dsn[len(dsn)-10:]
	}
	return "***masked***"
}
