package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/db"
)

func TestNewPool_InvalidConfig(t *testing.T) {
	_, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{}))
	require.Error(t, err)
}

func TestRedactedDsn_SubstringCollision(t *testing.T) {
	cfg := db.ConnectionConfig{
		DBType: db.Postgresql, Host: "localhost", Port: 54321,
		Username: "user", Password: "4321", DBName: "mydb",
	}
	redacted := cfg.RedactedDsn()
	assert.Contains(t, redacted, "password=***")
	assert.Contains(t, redacted, "port=54321") // naive ReplaceAll would corrupt this
	assert.NotContains(t, redacted, "password=4321")
}

func TestRedactedDsn_EmptyPassword(t *testing.T) {
	cfg := db.ConnectionConfig{
		DBType: db.Postgresql, Host: "localhost", Port: 5432,
		Username: "user", DBName: "mydb",
	}
	redacted := cfg.RedactedDsn()
	assert.NotContains(t, redacted, "***")
	assert.Contains(t, redacted, "user=user")
}
