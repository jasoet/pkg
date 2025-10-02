package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jasoet/pkg/v2/docker"
	_ "github.com/lib/pq" // PostgreSQL driver
)

func main() {
	ctx := context.Background()

	// Example: PostgreSQL database container
	postgresExample(ctx)
}

func postgresExample(ctx context.Context) {
	fmt.Println("=== PostgreSQL Database Container ===\n")

	// Create PostgreSQL container
	req := docker.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		Name:       "example-postgres",
		AutoRemove: true,
		WaitingFor: docker.WaitForLog("database system is ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	exec, err := docker.NewFromRequest(req)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Start the container
	fmt.Println("Starting PostgreSQL container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer func() {
		fmt.Println("\nCleaning up...")
		exec.Terminate(ctx)
	}()

	// Get connection details
	endpoint, _ := exec.Endpoint(ctx, "5432/tcp")
	fmt.Printf("PostgreSQL is running at: %s\n", endpoint)

	// Build connection string
	connStr, _ := exec.ConnectionString(ctx, "5432/tcp",
		"postgres://testuser:testpass@%s/testdb?sslmode=disable")
	fmt.Printf("Connection String: %s\n\n", connStr)

	// Connect to database
	fmt.Println("Connecting to database...")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	fmt.Println("Successfully connected to PostgreSQL! ✓\n")

	// Create a test table
	fmt.Println("Creating test table...")
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			email VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}
	fmt.Println("Table created successfully! ✓\n")

	// Insert test data
	fmt.Println("Inserting test data...")
	_, err = db.ExecContext(ctx, `
		INSERT INTO users (name, email) VALUES
		('Alice', 'alice@example.com'),
		('Bob', 'bob@example.com'),
		('Charlie', 'charlie@example.com')
	`)
	if err != nil {
		log.Fatalf("Failed to insert data: %v", err)
	}
	fmt.Println("Data inserted successfully! ✓\n")

	// Query data
	fmt.Println("Querying data...")
	rows, err := db.QueryContext(ctx, "SELECT id, name, email FROM users ORDER BY id")
	if err != nil {
		log.Fatalf("Failed to query data: %v", err)
	}
	defer rows.Close()

	fmt.Println("Users:")
	fmt.Println("ID | Name    | Email")
	fmt.Println("---|---------|------------------")
	for rows.Next() {
		var id int
		var name, email string
		if err := rows.Scan(&id, &name, &email); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		fmt.Printf("%-2d | %-7s | %s\n", id, name, email)
	}
	fmt.Println("\nQuery successful! ✓\n")

	// Get container status
	status, _ := exec.Status(ctx)
	fmt.Printf("Container Status:\n")
	fmt.Printf("  State: %s\n", status.State)
	fmt.Printf("  Running: %v\n", status.Running)
	fmt.Printf("  Started At: %s\n", status.StartedAt.Format(time.RFC3339))

	// Get logs
	fmt.Println("\nLast 10 lines of PostgreSQL logs:")
	logs, _ := exec.GetLastNLines(ctx, 10)
	fmt.Println(logs)

	fmt.Println("\nExample completed successfully! ✓")
}
