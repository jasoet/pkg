//go:build template

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jasoet/pkg/config"
	"github.com/jasoet/pkg/db"
	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/rest"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

// AppConfig defines the CLI application configuration structure
type AppConfig struct {
	Environment string              `yaml:"environment" mapstructure:"environment" validate:"required,oneof=development staging production"`
	Debug       bool                `yaml:"debug" mapstructure:"debug"`
	Database    db.ConnectionConfig `yaml:"database" mapstructure:"database" validate:"required"`
	ExternalAPI ExternalAPIConfig   `yaml:"externalApi" mapstructure:"externalApi"`
}

type ExternalAPIConfig struct {
	BaseURL    string        `yaml:"baseUrl" mapstructure:"baseUrl" validate:"url"`
	Timeout    time.Duration `yaml:"timeout" mapstructure:"timeout"`
	RetryCount int           `yaml:"retryCount" mapstructure:"retryCount"`
	APIKey     string        `yaml:"apiKey" mapstructure:"apiKey"`
}

// Services contains all CLI dependencies
type Services struct {
	DB        *gorm.DB
	APIClient *rest.Client
	Config    *AppConfig
	Logger    zerolog.Logger
}

// User model for demonstration
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

func main() {
	var (
		configFile = flag.String("config", "config.yaml", "Configuration file path")
		command    = flag.String("cmd", "", "Command to execute (migrate|seed|backup|users|help)")
		userID     = flag.Int("id", 0, "User ID for user operations")
		userName   = flag.String("name", "", "User name for user operations")
		userEmail  = flag.String("email", "", "User email for user operations")
		verbose    = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	// Initialize logging
	debug := *verbose || os.Getenv("DEBUG") == "true"
	logging.Initialize("cli-tool", debug)

	ctx := context.Background()
	logger := logging.ContextLogger(ctx, "main")

	// Load configuration
	appConfig, err := loadConfiguration(*configFile)
	if err != nil {
		logger.Fatal().Err(err).Str("config_file", *configFile).Msg("Failed to load configuration")
	}

	// Setup database
	database, err := appConfig.Database.Pool()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}

	// Setup HTTP client
	var apiClient *rest.Client
	if appConfig.ExternalAPI.BaseURL != "" {
		restConfig := &rest.Config{
			Timeout:    appConfig.ExternalAPI.Timeout,
			RetryCount: appConfig.ExternalAPI.RetryCount,
		}
		apiClient = rest.NewClient(rest.WithRestConfig(*restConfig))
	}

	// Create services container
	services := &Services{
		DB:        database,
		APIClient: apiClient,
		Config:    appConfig,
		Logger:    logger,
	}

	// Execute command
	if err := executeCommand(ctx, services, *command, *userID, *userName, *userEmail); err != nil {
		logger.Error().Err(err).Str("command", *command).Msg("Command execution failed")
		os.Exit(1)
	}

	logger.Info().Str("command", *command).Msg("Command completed successfully")
}

func loadConfiguration(configFile string) (*AppConfig, error) {
	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Use default configuration if file doesn't exist
		return loadDefaultConfiguration()
	}

	// Load configuration from file
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	// Load with environment variable overrides
	appConfig, err := config.LoadString[AppConfig](string(configData), "CLI")
	if err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	return appConfig, nil
}

func loadDefaultConfiguration() (*AppConfig, error) {
	defaultConfig := `
environment: development
debug: true
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: cli_app
  timeout: 30s
  maxIdleConns: 5
  maxOpenConns: 25
externalApi:
  baseUrl: https://api.example.com
  timeout: 30s
  retryCount: 3
  apiKey: ""
`

	appConfig, err := config.LoadString[AppConfig](defaultConfig, "CLI")
	if err != nil {
		return nil, fmt.Errorf("failed to load default configuration: %w", err)
	}

	return appConfig, nil
}

func executeCommand(ctx context.Context, services *Services, command string, userID int, userName, userEmail string) error {
	switch strings.ToLower(command) {
	case "migrate":
		return runMigrations(ctx, services)
	case "seed":
		return seedData(ctx, services)
	case "backup":
		return backupData(ctx, services)
	case "users":
		return userCommands(ctx, services, userID, userName, userEmail)
	case "help", "":
		return showHelp()
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runMigrations(ctx context.Context, services *Services) error {
	logger := logging.ContextLogger(ctx, "migrations")
	logger.Info().Msg("Running database migrations")

	// Auto-migrate models
	if err := services.DB.AutoMigrate(&User{}); err != nil {
		return fmt.Errorf("failed to migrate User model: %w", err)
	}

	logger.Info().Msg("Database migrations completed successfully")
	return nil
}

func seedData(ctx context.Context, services *Services) error {
	logger := logging.ContextLogger(ctx, "seed")
	logger.Info().Msg("Seeding database with sample data")

	// Check if data already exists
	var count int64
	services.DB.Model(&User{}).Count(&count)
	if count > 0 {
		logger.Info().Int64("existing_users", count).Msg("Database already contains data, skipping seed")
		return nil
	}

	// Create sample users
	sampleUsers := []User{
		{Name: "John Doe", Email: "john.doe@example.com"},
		{Name: "Jane Smith", Email: "jane.smith@example.com"},
		{Name: "Bob Johnson", Email: "bob.johnson@example.com"},
		{Name: "Alice Williams", Email: "alice.williams@example.com"},
		{Name: "Charlie Brown", Email: "charlie.brown@example.com"},
	}

	for _, user := range sampleUsers {
		if err := services.DB.Create(&user).Error; err != nil {
			logger.Error().Err(err).Str("email", user.Email).Msg("Failed to create user")
			return fmt.Errorf("failed to create user %s: %w", user.Email, err)
		}
		logger.Info().Uint("id", user.ID).Str("email", user.Email).Msg("Created user")
	}

	logger.Info().Int("count", len(sampleUsers)).Msg("Database seeding completed successfully")
	return nil
}

func backupData(ctx context.Context, services *Services) error {
	logger := logging.ContextLogger(ctx, "backup")
	logger.Info().Msg("Starting database backup")

	// Get all users
	var users []User
	if err := services.DB.Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users for backup: %w", err)
	}

	// Create backup file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	backupFile := fmt.Sprintf("backup_users_%s.json", timestamp)

	// TODO: Implement actual backup logic
	// This is a placeholder implementation
	logger.Info().
		Str("backup_file", backupFile).
		Int("user_count", len(users)).
		Msg("Backup would be created here")

	// If external API is configured, could upload backup
	if services.APIClient != nil && services.Config.ExternalAPI.APIKey != "" {
		logger.Info().Msg("Could upload backup to external storage")
	}

	logger.Info().Msg("Database backup completed successfully")
	return nil
}

func userCommands(ctx context.Context, services *Services, userID int, userName, userEmail string) error {
	// If no specific user parameters provided, list all users
	if userID == 0 && userName == "" && userEmail == "" {
		return listUsers(ctx, services)
	}

	// If userID provided, show specific user
	if userID > 0 {
		return showUser(ctx, services, userID)
	}

	// If name and email provided, create user
	if userName != "" && userEmail != "" {
		return createUser(ctx, services, userName, userEmail)
	}

	// If only email provided, search by email
	if userEmail != "" {
		return searchUserByEmail(ctx, services, userEmail)
	}

	return fmt.Errorf("invalid user command parameters")
}

func listUsers(ctx context.Context, services *Services) error {
	logger := logging.ContextLogger(ctx, "list-users")

	var users []User
	if err := services.DB.Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	fmt.Printf("Found %d users:\n", len(users))
	fmt.Println("ID\tName\t\tEmail\t\t\tCreated")
	fmt.Println("--\t----\t\t-----\t\t\t-------")

	for _, user := range users {
		fmt.Printf("%d\t%s\t\t%s\t\t%s\n",
			user.ID,
			truncateString(user.Name, 15),
			truncateString(user.Email, 20),
			user.CreatedAt.Format("2006-01-02"))
	}

	logger.Info().Int("count", len(users)).Msg("Listed users")
	return nil
}

func showUser(ctx context.Context, services *Services, userID int) error {
	logger := logging.ContextLogger(ctx, "show-user")

	var user User
	if err := services.DB.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user with ID %d not found", userID)
		}
		return fmt.Errorf("failed to fetch user: %w", err)
	}

	fmt.Printf("User Details:\n")
	fmt.Printf("ID:         %d\n", user.ID)
	fmt.Printf("Name:       %s\n", user.Name)
	fmt.Printf("Email:      %s\n", user.Email)
	fmt.Printf("Created:    %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:    %s\n", user.UpdatedAt.Format("2006-01-02 15:04:05"))

	logger.Info().Uint("user_id", user.ID).Msg("Showed user details")
	return nil
}

func createUser(ctx context.Context, services *Services, name, email string) error {
	logger := logging.ContextLogger(ctx, "create-user")

	user := User{
		Name:  name,
		Email: email,
	}

	if err := services.DB.Create(&user).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return fmt.Errorf("user with email %s already exists", email)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("Created user: ID=%d, Name=%s, Email=%s\n", user.ID, user.Name, user.Email)

	logger.Info().
		Uint("user_id", user.ID).
		Str("name", user.Name).
		Str("email", user.Email).
		Msg("Created user")

	return nil
}

func searchUserByEmail(ctx context.Context, services *Services, email string) error {
	logger := logging.ContextLogger(ctx, "search-user")

	var user User
	if err := services.DB.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("user with email %s not found", email)
		}
		return fmt.Errorf("failed to search user: %w", err)
	}

	fmt.Printf("Found User:\n")
	fmt.Printf("ID:         %d\n", user.ID)
	fmt.Printf("Name:       %s\n", user.Name)
	fmt.Printf("Email:      %s\n", user.Email)
	fmt.Printf("Created:    %s\n", user.CreatedAt.Format("2006-01-02 15:04:05"))

	logger.Info().Uint("user_id", user.ID).Str("email", email).Msg("Found user by email")
	return nil
}

func showHelp() error {
	fmt.Println("CLI Tool - Usage:")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  migrate              Run database migrations")
	fmt.Println("  seed                 Seed database with sample data")
	fmt.Println("  backup               Create database backup")
	fmt.Println("  users                List all users")
	fmt.Println("  help                 Show this help message")
	fmt.Println()
	fmt.Println("User Operations:")
	fmt.Println("  -cmd users                           List all users")
	fmt.Println("  -cmd users -id <ID>                  Show user by ID")
	fmt.Println("  -cmd users -email <EMAIL>            Search user by email")
	fmt.Println("  -cmd users -name <NAME> -email <EMAIL>   Create new user")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -config <FILE>       Configuration file (default: config.yaml)")
	fmt.Println("  -v                   Verbose output")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  DEBUG=true           Enable debug logging")
	fmt.Println("  CLI_DATABASE_HOST    Override database host")
	fmt.Println("  CLI_DATABASE_PASSWORD  Override database password")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ./cli-tool -cmd migrate")
	fmt.Println("  ./cli-tool -cmd seed")
	fmt.Println("  ./cli-tool -cmd users")
	fmt.Println("  ./cli-tool -cmd users -id 1")
	fmt.Println("  ./cli-tool -cmd users -name \"John Doe\" -email \"john@example.com\"")
	fmt.Println("  ./cli-tool -cmd backup -v")

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
