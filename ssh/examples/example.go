//go:build example

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/ssh"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("SSH Package Examples")
	fmt.Println("===================")

	// Example 1: Basic SSH Tunnel
	fmt.Println("\n1. Basic SSH Tunnel Example")
	basicTunnelExample()

	// Example 2: Database Connection Through SSH Tunnel
	fmt.Println("\n2. Database Connection Through SSH Tunnel")
	databaseTunnelExample()

	// Example 3: Multiple Tunnels Management
	fmt.Println("\n3. Multiple Tunnels Management")
	multipleTunnelsExample()

	// Example 4: Configuration from YAML
	fmt.Println("\n4. Configuration from YAML")
	yamlConfigExample()

	// Example 5: Error Handling and Connection Management
	fmt.Println("\n5. Error Handling and Connection Management")
	errorHandlingExample()
}

func basicTunnelExample() {
	fmt.Println("Creating basic SSH tunnel...")

	config := ssh.Config{
		Host:       getEnvOrDefault("SSH_HOST", "example.com"),
		Port:       22,
		User:       getEnvOrDefault("SSH_USER", "username"),
		Password:   getEnvOrDefault("SSH_PASSWORD", "password"),
		RemoteHost: getEnvOrDefault("REMOTE_HOST", "database.internal.com"),
		RemotePort: 5432,
		LocalPort:  5433,
		Timeout:    10 * time.Second,
	}

	tunnel := ssh.New(config)

	fmt.Printf("SSH Tunnel Config: %s@%s:%d -> localhost:%d -> %s:%d\n",
		config.User, config.Host, config.Port,
		config.LocalPort, config.RemoteHost, config.RemotePort)

	// Note: Actual connection would require valid SSH server
	fmt.Println("✓ SSH tunnel configuration created (connection not attempted in example)")
}

func databaseTunnelExample() {
	fmt.Println("Setting up database connection through SSH tunnel...")

	// SSH tunnel configuration for PostgreSQL
	tunnelConfig := ssh.Config{
		Host:       getEnvOrDefault("SSH_HOST", "bastion.example.com"),
		Port:       22,
		User:       getEnvOrDefault("SSH_USER", "deploy"),
		Password:   getEnvOrDefault("SSH_PASSWORD", "secure-password"),
		RemoteHost: getEnvOrDefault("DB_HOST", "postgres.internal.example.com"),
		RemotePort: 5432,
		LocalPort:  5433,
		Timeout:    10 * time.Second,
	}

	fmt.Printf("Database tunnel: %s@%s -> %s:%d\n",
		tunnelConfig.User, tunnelConfig.Host,
		tunnelConfig.RemoteHost, tunnelConfig.RemotePort)

	// Create tunnel
	tunnel := ssh.New(tunnelConfig)

	// In a real scenario, you would:
	// 1. Start the tunnel: err := tunnel.Start()
	// 2. Connect to database through tunnel
	// 3. Perform database operations
	// 4. Close tunnel: defer tunnel.Close()

	fmt.Println("Database connection string would be:")
	fmt.Println("host=localhost port=5433 user=dbuser password=dbpass dbname=mydb sslmode=disable")
	fmt.Println("✓ Database tunnel configuration prepared")

	// Example of database connection (commented out as it requires actual tunnel)
	/*
		err := tunnel.Start()
		if err != nil {
			log.Fatal("Failed to start tunnel:", err)
		}
		defer tunnel.Close()

		db, err := sql.Open("postgres", "host=localhost port=5433 user=dbuser password=dbpass dbname=mydb sslmode=disable")
		if err != nil {
			log.Fatal("Failed to connect to database:", err)
		}
		defer db.Close()

		err = db.Ping()
		if err != nil {
			log.Fatal("Failed to ping database:", err)
		}
		fmt.Println("✓ Successfully connected to database through SSH tunnel")
	*/
}

func multipleTunnelsExample() {
	fmt.Println("Managing multiple SSH tunnels...")

	// Database tunnel
	dbTunnel := ssh.New(ssh.Config{
		Host:       getEnvOrDefault("SSH_HOST", "bastion.example.com"),
		Port:       22,
		User:       getEnvOrDefault("SSH_USER", "deploy"),
		Password:   getEnvOrDefault("SSH_PASSWORD", "password"),
		RemoteHost: "postgres.internal.com",
		RemotePort: 5432,
		LocalPort:  5433,
		Timeout:    10 * time.Second,
	})

	// Redis tunnel
	redisTunnel := ssh.New(ssh.Config{
		Host:       getEnvOrDefault("SSH_HOST", "bastion.example.com"),
		Port:       22,
		User:       getEnvOrDefault("SSH_USER", "deploy"),
		Password:   getEnvOrDefault("SSH_PASSWORD", "password"),
		RemoteHost: "redis.internal.com",
		RemotePort: 6379,
		LocalPort:  6380,
		Timeout:    10 * time.Second,
	})

	// Web service tunnel
	webTunnel := ssh.New(ssh.Config{
		Host:       getEnvOrDefault("SSH_HOST", "bastion.example.com"),
		Port:       22,
		User:       getEnvOrDefault("SSH_USER", "deploy"),
		Password:   getEnvOrDefault("SSH_PASSWORD", "password"),
		RemoteHost: "api.internal.com",
		RemotePort: 8080,
		LocalPort:  8081,
		Timeout:    10 * time.Second,
	})

	tunnels := []*ssh.Tunnel{dbTunnel, redisTunnel, webTunnel}
	services := []string{"PostgreSQL", "Redis", "Web API"}

	fmt.Println("Tunnel configurations:")
	for i, service := range services {
		fmt.Printf("- %s: localhost:%d\n", service, []int{5433, 6380, 8081}[i])
	}

	// In a real scenario, you would start all tunnels:
	/*
		for i, tunnel := range tunnels {
			err := tunnel.Start()
			if err != nil {
				log.Printf("Failed to start %s tunnel: %v", services[i], err)
				continue
			}
			defer tunnel.Close()
			fmt.Printf("✓ %s tunnel started\n", services[i])
		}
	*/

	fmt.Println("✓ Multiple tunnel configurations prepared")
}

func yamlConfigExample() {
	fmt.Println("Loading SSH configuration from YAML...")

	yamlConfig := `
host: bastion.example.com
port: 22
user: deploy
password: secure-password
remoteHost: service.internal.com
remotePort: 8080
localPort: 8081
timeout: 30s
`

	var config ssh.Config
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	if err != nil {
		log.Printf("Failed to parse YAML config: %v", err)
		return
	}

	fmt.Printf("Loaded config: %s@%s:%d -> localhost:%d -> %s:%d (timeout: %v)\n",
		config.User, config.Host, config.Port,
		config.LocalPort, config.RemoteHost, config.RemotePort, config.Timeout)

	tunnel := ssh.New(config)
	fmt.Println("✓ SSH tunnel created from YAML configuration")

	// Example of loading from file (commented out)
	/*
		configFile := "tunnel-config.yaml"
		yamlData, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Printf("Failed to read config file: %v", err)
			return
		}

		var config ssh.Config
		err = yaml.Unmarshal(yamlData, &config)
		if err != nil {
			log.Printf("Failed to parse YAML: %v", err)
			return
		}
	*/
}

func errorHandlingExample() {
	fmt.Println("Demonstrating error handling patterns...")

	config := ssh.Config{
		Host:       getEnvOrDefault("SSH_HOST", "nonexistent.example.com"),
		Port:       22,
		User:       getEnvOrDefault("SSH_USER", "testuser"),
		Password:   getEnvOrDefault("SSH_PASSWORD", "wrongpassword"),
		RemoteHost: "localhost",
		RemotePort: 5432,
		LocalPort:  5433,
		Timeout:    5 * time.Second,
	}

	tunnel := ssh.New(config)

	fmt.Println("Attempting to start tunnel (this will demonstrate error handling)...")

	// This would normally fail with connection errors
	/*
		err := tunnel.Start()
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				log.Println("Error: SSH server is not accessible")
				log.Println("Solution: Check SSH server address and port")
			} else if strings.Contains(err.Error(), "authentication failed") {
				log.Println("Error: Invalid SSH credentials")
				log.Println("Solution: Verify username and password")
			} else if strings.Contains(err.Error(), "timeout") {
				log.Println("Error: Connection timeout")
				log.Println("Solution: Check network connectivity and increase timeout")
			} else {
				log.Printf("Error: SSH tunnel failed: %v", err)
			}
			return
		}
	*/

	// Graceful shutdown example
	fmt.Println("Example of graceful shutdown handling:")
	fmt.Println(`
c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
go func() {
    <-c
    log.Println("Shutting down SSH tunnel...")
    tunnel.Close()
    os.Exit(0)
}()`)

	fmt.Println("✓ Error handling patterns demonstrated")
}

// runGracefulShutdownExample demonstrates proper tunnel lifecycle management
func runGracefulShutdownExample(tunnel *ssh.Tunnel) {
	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Received shutdown signal, closing SSH tunnel...")
		if err := tunnel.Close(); err != nil {
			log.Printf("Error closing tunnel: %v", err)
		}
		log.Println("SSH tunnel closed successfully")
		os.Exit(0)
	}()

	// Keep the program running
	fmt.Println("SSH tunnel is running. Press Ctrl+C to stop.")
	select {} // Wait forever
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Additional helper functions for testing and validation

// validateConfig validates SSH tunnel configuration
func validateConfig(config ssh.Config) error {
	if config.Host == "" {
		return fmt.Errorf("SSH host is required")
	}
	if config.User == "" {
		return fmt.Errorf("SSH user is required")
	}
	if config.Password == "" {
		return fmt.Errorf("SSH password is required")
	}
	if config.RemoteHost == "" {
		return fmt.Errorf("remote host is required")
	}
	if config.LocalPort <= 0 || config.LocalPort > 65535 {
		return fmt.Errorf("invalid local port: %d", config.LocalPort)
	}
	if config.RemotePort <= 0 || config.RemotePort > 65535 {
		return fmt.Errorf("invalid remote port: %d", config.RemotePort)
	}
	return nil
}

// testConnection tests if a local port is available
func testConnection(port int) error {
	conn, err := sql.Open("postgres", fmt.Sprintf("host=localhost port=%d user=test dbname=test sslmode=disable", port))
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Ping()
}
