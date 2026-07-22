package db_test

import (
	"fmt"

	"github.com/jasoet/pkg/v3/db"
)

// RedactedDsn masks the password in the DSN, making it safe for logs.
func ExampleConnectionConfig_RedactedDsn() {
	cfg := db.ConnectionConfig{
		DBType:   db.Postgresql,
		Host:     "localhost",
		Port:     5432,
		Username: "admin",
		Password: "s3cret-password",
		DBName:   "myapp",
	}
	fmt.Println(cfg.RedactedDsn())

	// Output: user=admin password=*** host=localhost port=5432 dbname=myapp sslmode=require connect_timeout=30
}

// Validate rejects configs with missing required fields.
func ExampleConnectionConfig_Validate() {
	cfg := db.ConnectionConfig{
		DBType:   db.Postgresql,
		Host:     "localhost",
		Port:     5432,
		Username: "admin",
		// DBName is missing.
	}
	fmt.Println(cfg.Validate())

	// Output: dbName is required
}

// NewPool validates the config before dialing, so an invalid config fails fast
// without any network access. On a valid config it dials a real database;
// that path is non-deterministic and therefore not shown here.
func ExampleNewPool() {
	_, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
		DBType: db.Postgresql,
		// Host is missing.
	}))
	fmt.Println(err)

	// Output: invalid config: host is required
}
