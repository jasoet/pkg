package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	noopm "go.opentelemetry.io/otel/metric/noop"
	noopt "go.opentelemetry.io/otel/trace/noop"
	"gorm.io/gorm/logger"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

func TestDatabaseConfigValidation(t *testing.T) {
	validConfig := &ConnectionConfig{
		DBType:       Mysql,
		Host:         "localhost",
		Port:         3306,
		Username:     "root",
		Password:     "",
		DBName:       "mydb",
		Timeout:      3 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	invalidConfig := &ConnectionConfig{
		DBType:       "invalid_db_type",
		Host:         "",
		Port:         -1,
		Username:     "",
		Password:     "",
		DBName:       "",
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

func TestConnectionConfig_dsn(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnectionConfig
		wantDsn string
	}{
		{
			name: "MySQL connection",
			config: ConnectionConfig{
				DBType:   Mysql,
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				DBName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: "root:password@tcp(localhost:3306)/test?parseTime=true&timeout=3s",
		},
		{
			name: "Postgres connection",
			config: ConnectionConfig{
				DBType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				DBName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: "user='postgres' password='password' host='localhost' port=5432 dbname='test' sslmode=require connect_timeout=3",
		},
		{
			name: "Different port",
			config: ConnectionConfig{
				DBType:   Mysql,
				Host:     "localhost",
				Port:     8080,
				Username: "root",
				Password: "password",
				DBName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "root:password@tcp(localhost:8080)/test?parseTime=true&timeout=5s",
		},
		{
			name: "All configurations are empty",
			config: ConnectionConfig{
				DBType:   "",
				Host:     "",
				Port:     0,
				Username: "",
				Password: "",
				DBName:   "",
				Timeout:  0 * time.Second,
			},
			wantDsn: "", // Unknown DbType returns empty DSN
		},
		{
			name: "MSSQL connection",
			config: ConnectionConfig{
				DBType:   MSSQL,
				Host:     "localhost",
				Port:     1433,
				Username: "sa",
				Password: "password",
				DBName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "sqlserver://sa:password@localhost:1433?connection+timeout=5&database=test&encrypt=true",
		},
		{
			name: "Postgres with custom SSLMode",
			config: ConnectionConfig{
				DBType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "password",
				DBName:   "test",
				Timeout:  3 * time.Second,
				SSLMode:  "require",
			},
			wantDsn: "user='postgres' password='password' host='localhost' port=5432 dbname='test' sslmode=require connect_timeout=3",
		},
		{
			name: "Zero timeout uses default 30s",
			config: ConnectionConfig{
				DBType:   Mysql,
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: "password",
				DBName:   "test",
				Timeout:  0,
			},
			wantDsn: "root:password@tcp(localhost:3306)/test?parseTime=true&timeout=30s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDsn := tt.config.dsn()
			assert.Equal(t, tt.wantDsn, gotDsn)
		})
	}
}

// TestConnectionConfig_dsn_SpecialCharacters verifies that credentials containing
// special characters (spaces, @, #, %, quotes, backslashes) and DSN-injection
// payloads are safely escaped so they cannot alter connection parameters.
//
// The golden strings below were verified to round-trip correctly through the real
// driver parsers (jackc/pgx pgconn.ParseConfig and microsoft/go-mssqldb msdsn.Parse):
// the payloads are parsed back verbatim as the password, and the host/sslmode/encrypt
// remain untouched — i.e. no TLS downgrade or host redirection is possible.
func TestConnectionConfig_dsn_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name    string
		config  ConnectionConfig
		wantDsn string
	}{
		{
			name: "Postgres password with spaces and injection attempt",
			config: ConnectionConfig{
				DBType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: "x sslmode=disable host=evil",
				DBName:   "test",
				Timeout:  3 * time.Second,
			},
			// The injection payload is contained inside the single-quoted password
			// field; sslmode=require and host='localhost' are unaffected.
			wantDsn: "user='postgres' password='x sslmode=disable host=evil' host='localhost' port=5432 dbname='test' sslmode=require connect_timeout=3",
		},
		{
			name: "Postgres password with quote and backslash",
			config: ConnectionConfig{
				DBType:   Postgresql,
				Host:     "localhost",
				Port:     5432,
				Username: "postgres",
				Password: `he'llo\world`,
				DBName:   "test",
				Timeout:  3 * time.Second,
			},
			wantDsn: `user='postgres' password='he\'llo\\world' host='localhost' port=5432 dbname='test' sslmode=require connect_timeout=3`,
		},
		{
			name: "MSSQL password with @ # % and space",
			config: ConnectionConfig{
				DBType:   MSSQL,
				Host:     "localhost",
				Port:     1433,
				Username: "sa",
				Password: "p@ss w#rd%50",
				DBName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "sqlserver://sa:p%40ss%20w%23rd%2550@localhost:1433?connection+timeout=5&database=test&encrypt=true",
		},
		{
			name: "MSSQL password with injection attempt",
			config: ConnectionConfig{
				DBType:   MSSQL,
				Host:     "localhost",
				Port:     1433,
				Username: "sa",
				Password: "x sslmode=disable host=evil",
				DBName:   "test",
				Timeout:  5 * time.Second,
			},
			wantDsn: "sqlserver://sa:x%20sslmode=disable%20host=evil@localhost:1433?connection+timeout=5&database=test&encrypt=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantDsn, tt.config.dsn())
		})
	}
}

// TestConnectionConfig_dsn_SubSecondTimeout verifies that sub-second timeouts do
// not truncate to zero in the DSN (which would mean "no dial timeout" for MySQL or
// an indefinite connect_timeout for PostgreSQL).
func TestConnectionConfig_dsn_SubSecondTimeout(t *testing.T) {
	mysqlCfg := ConnectionConfig{
		DBType: Mysql, Host: "localhost", Port: 3306,
		Username: "root", Password: "password", DBName: "test",
		Timeout: 500 * time.Millisecond,
	}
	// MySQL accepts Go duration strings, so the sub-second value is preserved.
	assert.Equal(t, "root:password@tcp(localhost:3306)/test?parseTime=true&timeout=500ms", mysqlCfg.dsn())

	pgCfg := ConnectionConfig{
		DBType: Postgresql, Host: "localhost", Port: 5432,
		Username: "postgres", Password: "password", DBName: "test",
		Timeout: 500 * time.Millisecond,
	}
	// connect_timeout is integer seconds; a sub-second timeout rounds up to 1
	// rather than truncating to 0 (which Postgres treats as indefinite).
	assert.Equal(t, "user='postgres' password='password' host='localhost' port=5432 dbname='test' sslmode=require connect_timeout=1", pgCfg.dsn())
}

// TestConnectionConfig_Validate_MSSQLSSLMode verifies MSSQL SSLMode validation and
// that valid modes map to encrypt values go-mssqldb accepts.
func TestConnectionConfig_Validate_MSSQLSSLMode(t *testing.T) {
	base := ConnectionConfig{
		DBType: MSSQL, Host: "localhost", Port: 1433,
		Username: "sa", Password: "pw", DBName: "test",
		MaxIdleConns: 5, MaxOpenConns: 10,
	}

	for _, mode := range []string{"disable", "false", "true", "require", "strict"} {
		cfg := base
		cfg.SSLMode = mode
		assert.NoError(t, cfg.Validate(), "SSLMode %q should be valid for MSSQL", mode)
	}

	for _, mode := range []string{"verify-full", "prefer", "yes", "1", "enable"} {
		cfg := base
		cfg.SSLMode = mode
		assert.Error(t, cfg.Validate(), "SSLMode %q should be rejected for MSSQL", mode)
	}
}

// TestConnectionConfig_mssqlEncrypt verifies the SSLMode -> encrypt mapping used in
// the MSSQL DSN. In particular the default ("require") maps to "true", not the
// invalid go-mssqldb value "require".
func TestConnectionConfig_mssqlEncrypt(t *testing.T) {
	cases := map[string]string{
		"":        "true", // default require -> true
		"require": "true",
		"true":    "true",
		"disable": "disable",
		"false":   "false",
		"strict":  "strict",
	}
	for mode, want := range cases {
		cfg := ConnectionConfig{DBType: MSSQL, SSLMode: mode}
		assert.Equal(t, want, cfg.mssqlEncrypt(), "SSLMode %q", mode)
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
	assert.Equal(t, "require", c.effectiveSSLMode())

	c.SSLMode = "disable"
	assert.Equal(t, "disable", c.effectiveSSLMode())
}

func TestEffectiveGormLogLevel(t *testing.T) {
	c := &ConnectionConfig{}
	assert.Equal(t, logger.Silent, c.effectiveGormLogLevel())

	c.GormLogLevel = int(logger.Info)
	assert.Equal(t, logger.Info, c.effectiveGormLogLevel())

	c.GormLogLevel = 99 // Invalid value
	assert.Equal(t, logger.Silent, c.effectiveGormLogLevel())
}

func TestConnectionConfig_effectiveMaxConns(t *testing.T) {
	// Zero-value config: sane defaults are applied so idle pooling still works
	// and every query does not dial a fresh TCP connection.
	zero := &ConnectionConfig{}
	assert.Equal(t, defaultMaxIdleConns, zero.effectiveMaxIdleConns())
	assert.Equal(t, defaultMaxOpenConns, zero.effectiveMaxOpenConns())

	// Explicit values are honored.
	c := &ConnectionConfig{MaxIdleConns: 3, MaxOpenConns: 7}
	assert.Equal(t, 3, c.effectiveMaxIdleConns())
	assert.Equal(t, 7, c.effectiveMaxOpenConns())
}

func TestConnectionConfig_Validate_MaxOpenConnsZeroAllowed(t *testing.T) {
	// MaxOpenConns == 0 must not be rejected by the MaxIdle > MaxOpen check;
	// a zero value means "unset" (defaulted), not "smaller than idle".
	cfg := &ConnectionConfig{
		DBType: Postgresql, Host: "localhost", Port: 5432,
		Username: "u", Password: "p", DBName: "db",
		MaxIdleConns: 5, MaxOpenConns: 0,
	}
	assert.NoError(t, cfg.Validate())

	// Explicit idle > explicit open is still rejected.
	bad := &ConnectionConfig{
		DBType: Postgresql, Host: "localhost", Port: 5432,
		Username: "u", Password: "p", DBName: "db",
		MaxIdleConns: 20, MaxOpenConns: 10,
	}
	assert.Error(t, bad.Validate())
}

func TestConnectionConfig_collectPoolMetrics_NilOTelConfig(t *testing.T) {
	config := &ConnectionConfig{
		DBType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DBName:     "test",
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
	otelConfig := pkgotel.NewConfig("test",
		pkgotel.WithTracerProvider(noopt.NewTracerProvider()))

	config := &ConnectionConfig{
		DBType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DBName:     "test",
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
	otelConfig := pkgotel.NewConfig("test-metrics",
		pkgotel.WithMeterProvider(noopm.NewMeterProvider()))

	config := &ConnectionConfig{
		DBType:     Postgresql,
		Host:       "localhost",
		Port:       5432,
		Username:   "test",
		Password:   "test",
		DBName:     "test",
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

func TestConnectionConfig_NewPool_InvalidDbType(t *testing.T) {
	config := &ConnectionConfig{
		DBType:       "invalid-db-type",
		Host:         "localhost",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DBName:       "test",
		Timeout:      5 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	_, err := NewPool(WithConnectionConfig(*config))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database type")
}

func TestConnectionConfig_NewPool_ConnectionFailure(t *testing.T) {
	config := &ConnectionConfig{
		DBType:       Postgresql,
		Host:         "invalid-host-that-does-not-exist.local",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DBName:       "test",
		Timeout:      1 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	_, err := NewPool(WithConnectionConfig(*config))
	assert.Error(t, err)
	// The error should be from the connection attempt
}

func TestConnectionConfig_NewPool_EmptyDSN(t *testing.T) {
	config := &ConnectionConfig{
		DBType:       "",
		Host:         "",
		Port:         0,
		Username:     "",
		Password:     "",
		DBName:       "",
		Timeout:      5 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	_, err := NewPool(WithConnectionConfig(*config))
	assert.Error(t, err)
}

func TestRedactedDsn_MasksPassword(t *testing.T) {
	cfg := ConnectionConfig{
		Host: "localhost", Port: 5432, Username: "admin",
		Password: "supersecret", DBName: "testdb", DBType: Postgresql,
	}
	redacted := cfg.RedactedDsn()
	assert.Contains(t, redacted, "***")
	assert.NotContains(t, redacted, "supersecret")
}

func TestEffectiveSSLMode_DefaultsToRequire(t *testing.T) {
	cfg := ConnectionConfig{}
	assert.Equal(t, "require", cfg.effectiveSSLMode())
}

func TestConnectionConfig_SQLDB_ConnectionFailure(t *testing.T) {
	config := &ConnectionConfig{
		DBType:       Postgresql,
		Host:         "invalid-host.local",
		Port:         5432,
		Username:     "test",
		Password:     "test",
		DBName:       "test",
		Timeout:      1 * time.Second,
		MaxIdleConns: 5,
		MaxOpenConns: 10,
	}

	// SQLDB() calls NewPool() internally, which will fail to connect
	db, err := config.SQLDB()
	assert.Error(t, err)
	assert.Nil(t, db)
	// Should get a connection error
}
