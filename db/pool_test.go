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
	"gorm.io/gorm/logger"
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

	err := validate.Struct(validConfig)
	assert.NoError(t, err, "valid database config should pass validation")

	err = validate.Struct(invalidConfig)
	assert.Error(t, err, "invalid database config should fail validation")
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

	err := validate.Struct(validStruct)
	assert.NoError(t, err, "valid custom struct should pass validation")

	err = validate.Struct(invalidStruct)
	assert.Error(t, err, "invalid custom struct should fail validation")
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
			wantDsn: "", // Unknown DbType returns empty DSN
		},
		{
			name: "MSSQL connection",
			config: ConnectionConfig{
				DbType:   MSSQL,
				Host:     "localhost",
				Port:     1433,
				Username: "sa",
				Password: "password",
				DbName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "sqlserver://sa:password@localhost:1433?database=test&connectTimeout=5s&encrypt=disable",
		},
		{
			name: "Postgres with custom SSLMode",
			config: ConnectionConfig{
				DbType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				DbName:   "test",
				Timeout:  3 * time.Second,
				SSLMode:  "require",
			},
			wantDsn: "user=postgres password=password host=localhost port=5432 dbname=test sslmode=require connect_timeout=3",
		},
		{
			name: "Zero timeout uses default 30s",
			config: ConnectionConfig{
				DbType:   Mysql,
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				DbName:   "test",
				Timeout:  0,
			},
			wantDsn: "root:password@tcp(localhost:3306)/test?parseTime=true&timeout=30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDsn := tt.config.Dsn()
			assert.Equal(t, tt.wantDsn, gotDsn)
		})
	}
}

// TestExtractOperationType removed - extractOperationType is no longer used
// The uptrace otelgorm library handles operation type extraction internally

func TestEffectiveTimeout(t *testing.T) {
	c := &ConnectionConfig{Timeout: 0}
	assert.Equal(t, 30*time.Second, c.effectiveTimeout())

	c.Timeout = 5 * time.Second
	assert.Equal(t, 5*time.Second, c.effectiveTimeout())
}

func TestEffectiveSSLMode(t *testing.T) {
	c := &ConnectionConfig{}
	assert.Equal(t, "disable", c.effectiveSSLMode())

	c.SSLMode = "require"
	assert.Equal(t, "require", c.effectiveSSLMode())
}

func TestEffectiveGormLogLevel(t *testing.T) {
	c := &ConnectionConfig{}
	assert.Equal(t, logger.Silent, c.effectiveGormLogLevel())

	c.GormLogLevel = int(logger.Info)
	assert.Equal(t, logger.Info, c.effectiveGormLogLevel())

	c.GormLogLevel = 99 // Invalid value
	assert.Equal(t, logger.Silent, c.effectiveGormLogLevel())
}

func TestConnectionConfig_collectPoolMetrics_NilOTelConfig(t *testing.T) {
	config := &ConnectionConfig{
		DbType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DbName:     "test",
		OTelConfig: nil, // No OTel config
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

// TestConnectionConfig_installOTelCallbacks tests removed
// The uptrace otelgorm plugin is now used instead of custom callbacks

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
	assert.Contains(t, err.Error(), "unsupported database type")
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
