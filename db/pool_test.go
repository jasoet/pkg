package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	pkgotel "github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	noopm "go.opentelemetry.io/otel/metric/noop"
	noopt "go.opentelemetry.io/otel/trace/noop"
)

func TestDatabaseConfigValidation(t *testing.T) {
	validConfig := &ConnectionConfig{
		DbType:       Mysql,
		Host:         "localhost",
		Port:         3306,
		Username:     "root",
		Password:     "",
		DbName:       "mydb",
		Timeout:      3 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	invalidConfig := &ConnectionConfig{
		DbType:       "invalid_db_type",
		Host:         "",
		Port:         -1,
		Username:     "",
		Password:     "",
		DbName:       "",
		MaxIdleConns: 0,
		MaxOpenConns: 0,
	}

	validate := validator.New()

	if err := validate.Struct(validConfig); err != nil {
		t.Errorf("validation of valid database config failed: %v", err)
	}

	if err := validate.Struct(invalidConfig); err == nil {
		t.Error("validation of invalid database config passed unexpectedly")
	}
}

func TestCustomValidationTags(t *testing.T) {
	type CustomStruct struct {
		CustomField string `validate:"custom"`
	}

	validate := validator.New()
	_ = validate.RegisterValidation("custom", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return value == "foo" || value == "bar"
	})

	validStruct := &CustomStruct{CustomField: "foo"}
	invalidStruct := &CustomStruct{CustomField: "baz"}

	if err := validate.Struct(validStruct); err != nil {
		t.Errorf("validation of valid custom struct failed: %v", err)
	}

	if err := validate.Struct(invalidStruct); err == nil {
		t.Error("validation of invalid custom struct passed unexpectedly")
	}
}

func TestConnectionConfig_Dsn(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnectionConfig
		wantDsn string
	}{
		{
			name: "MySQL connection",
			config: ConnectionConfig{
				DbType:   Mysql,
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				DbName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: "root:password@tcp(localhost:3306)/test?parseTime=true&timeout=3s",
		},
		{
			name: "Postgres connection",
			config: ConnectionConfig{
				DbType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				DbName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: "user=postgres password=password host=localhost port=5432 dbname=test sslmode=disable connect_timeout=3",
		},
		{
			name: "Different port",
			config: ConnectionConfig{
				DbType:   Mysql,
				Host:     "localhost",
				Port:     8080,
				Username: "root",
				Password: "password",
				DbName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "root:password@tcp(localhost:8080)/test?parseTime=true&timeout=5s",
		},
		{
			name: "All configurations are empty",
			config: ConnectionConfig{
				DbType:   "",
				Host:     "",
				Port:     0,
				Username: "",
				Password: "",
				DbName:   "",
				Timeout:  0 * time.Second,
			},
			wantDsn: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDsn := tt.config.Dsn()
			if gotDsn != tt.wantDsn {
				t.Errorf("ConnectionConfig.Dsn() = %v, want %v", gotDsn, tt.wantDsn)
			}
		})
	}
}

func TestExtractOperationType(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "SELECT query",
			sql:      "SELECT * FROM users",
			expected: "db.SELECT",
		},
		{
			name:     "INSERT query",
			sql:      "INSERT INTO users (name) VALUES ('John')",
			expected: "db.INSERT",
		},
		{
			name:     "UPDATE query",
			sql:      "UPDATE users SET name='Jane' WHERE id=1",
			expected: "db.UPDATE",
		},
		{
			name:     "DELETE query",
			sql:      "DELETE FROM users WHERE id=1",
			expected: "db.DELETE",
		},
		{
			name:     "CREATE query",
			sql:      "CREATE TABLE users (id INT)",
			expected: "db.CREATE",
		},
		{
			name:     "DROP query",
			sql:      "DROP TABLE users",
			expected: "db.DROP",
		},
		{
			name:     "query with newline",
			sql:      "SELECT\n* FROM users",
			expected: "db.SELECT",
		},
		{
			name:     "query with tab",
			sql:      "SELECT\t* FROM users",
			expected: "db.SELECT",
		},
		{
			name:     "empty SQL",
			sql:      "",
			expected: "db.query",
		},
		{
			name:     "single word SQL",
			sql:      "COMMIT",
			expected: "db.query",
		},
		{
			name:     "lowercase query",
			sql:      "select * from users",
			expected: "db.select",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractOperationType(tt.sql)
			if result != tt.expected {
				t.Errorf("extractOperationType() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestConnectionConfig_collectPoolMetrics_NilOTelConfig(t *testing.T) {
	config := &ConnectionConfig{
		DbType:       Postgresql,
		Host:         "localhost",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DbName:       "test",
		OTelConfig:   nil, // No OTel config
	}

	// Create a mock sql.DB - this won't actually connect
	db, err := sql.Open("postgres", "host=invalid")
	require.NoError(t, err)
	defer db.Close()

	// Should not panic and should return early
	config.collectPoolMetrics(db)
	// If we get here without panic, the nil check worked
}

func TestConnectionConfig_collectPoolMetrics_MetricsDisabled(t *testing.T) {
	// OTel config with only tracing enabled (no MeterProvider = metrics disabled)
	otelConfig := pkgotel.NewConfig("test").
		WithTracerProvider(noopt.NewTracerProvider())

	config := &ConnectionConfig{
		DbType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DbName:     "test",
		OTelConfig: otelConfig,
	}

	// Create a mock sql.DB
	db, err := sql.Open("postgres", "host=invalid")
	require.NoError(t, err)
	defer db.Close()

	// Should return early when metrics are disabled
	config.collectPoolMetrics(db)
	// If we get here without panic, the metrics disabled check worked
}

func TestConnectionConfig_collectPoolMetrics_WithValidConfig(t *testing.T) {
	// OTel config with metrics enabled (using noop MeterProvider for testing)
	otelConfig := pkgotel.NewConfig("test-metrics").
		WithMeterProvider(noopm.NewMeterProvider())

	config := &ConnectionConfig{
		DbType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DbName:     "test",
		OTelConfig: otelConfig,
	}

	// Create a mock sql.DB
	db, err := sql.Open("postgres", "host=invalid")
	require.NoError(t, err)
	defer db.Close()

	// Should successfully register metrics callbacks
	config.collectPoolMetrics(db)
	// If we get here, metrics were collected successfully
}

func TestConnectionConfig_installOTelCallbacks_NilOTelConfig(t *testing.T) {
	config := &ConnectionConfig{
		DbType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DbName:     "test",
		OTelConfig: nil, // No OTel config
	}

	// Create a nil GORM DB - the function should return early before using it
	config.installOTelCallbacks(nil)
	// If we get here without panic, the nil check worked
}

func TestConnectionConfig_installOTelCallbacks_TracingDisabled(t *testing.T) {
	// OTel config with only metrics enabled (no TracerProvider = tracing disabled)
	otelConfig := pkgotel.NewConfig("test").
		WithMeterProvider(noopm.NewMeterProvider())

	config := &ConnectionConfig{
		DbType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DbName:     "test",
		OTelConfig: otelConfig,
	}

	// Should return early when tracing is disabled
	config.installOTelCallbacks(nil)
	// If we get here without panic, the tracing disabled check worked
}

func TestConnectionConfig_Pool_InvalidDbType(t *testing.T) {
	config := &ConnectionConfig{
		DbType:       "invalid-db-type",
		Host:         "localhost",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DbName:       "test",
		Timeout:      5 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	_, err := config.Pool()
	assert.Error(t, err)
	// Invalid DB type results in empty DSN, which causes GORM to fail
	assert.Contains(t, err.Error(), "dsn is empty")
}

func TestConnectionConfig_Pool_ConnectionFailure(t *testing.T) {
	config := &ConnectionConfig{
		DbType:       Postgresql,
		Host:         "invalid-host-that-does-not-exist.local",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DbName:       "test",
		Timeout:      1 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	_, err := config.Pool()
	assert.Error(t, err)
	// The error should be from the connection attempt
}

func TestConnectionConfig_Pool_EmptyDSN(t *testing.T) {
	config := &ConnectionConfig{
		DbType:       "",
		Host:         "",
		Port:         0,
		Username:     "",
		Password:     "",
		DbName:       "",
		Timeout:      5 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	_, err := config.Pool()
	assert.Error(t, err)
}

func TestConnectionConfig_SQLDB_ConnectionFailure(t *testing.T) {
	config := &ConnectionConfig{
		DbType:       Postgresql,
		Host:         "invalid-host.local",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DbName:       "test",
		Timeout:      1 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	// SQLDB() calls Pool() internally, which will fail to connect
	db, err := config.SQLDB()
	assert.Error(t, err)
	assert.Nil(t, db)
	// Should get a connection error
}
