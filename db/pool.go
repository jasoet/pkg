// Package db provides database connection pooling with GORM, supporting PostgreSQL,
// MySQL, and MSSQL, with optional OpenTelemetry instrumentation.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pkgotel "github.com/jasoet/pkg/v2/otel"
)

// DatabaseType identifies the database backend.
type DatabaseType string

const (
	// Mysql identifies a MySQL/MariaDB backend. The string value is "MYSQL".
	Mysql DatabaseType = "MYSQL"

	// Postgresql identifies a PostgreSQL backend. The string value is "POSTGRES"
	// (not "POSTGRESQL") for compatibility with existing configurations and OTel attributes.
	Postgresql DatabaseType = "POSTGRES"

	// MSSQL identifies a Microsoft SQL Server backend. The string value is "MSSQL".
	MSSQL DatabaseType = "MSSQL"

	// defaultTimeout is applied when Timeout is zero to avoid immediate connection failure.
	defaultTimeout = 30 * time.Second
)

// ConnectionConfig holds the connection parameters for a database pool.
type ConnectionConfig struct {
	DBType       DatabaseType  `yaml:"dbType" validate:"required,oneof=MYSQL POSTGRES MSSQL" mapstructure:"dbType"`
	Host         string        `yaml:"host" validate:"required,min=1" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port" validate:"required,min=1,max=65535"`
	Username     string        `yaml:"username" validate:"required,min=1" mapstructure:"username"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DBName       string        `yaml:"dbName" validate:"required,min=1" mapstructure:"dbName"`
	Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout"`
	MaxIdleConns int           `yaml:"maxIdleConns" mapstructure:"maxIdleConns" validate:"min=1"`
	MaxOpenConns int           `yaml:"maxOpenConns" mapstructure:"maxOpenConns" validate:"min=2"`

	// ConnMaxLifetime sets the maximum duration a connection may be reused.
	// Zero means connections are not closed due to age.
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" mapstructure:"connMaxLifetime"`

	// ConnMaxIdleTime sets the maximum duration a connection may sit idle.
	// Zero means connections are not closed due to idle time.
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" mapstructure:"connMaxIdleTime"`

	// SSLMode configures TLS for the connection.
	// PostgreSQL: "disable", "require", "verify-ca", "verify-full" (default: "disable")
	// MSSQL: "disable", "true", "false" (default: "disable")
	// MySQL: handled via DSN parameters (this field is ignored for MySQL)
	SSLMode string `yaml:"sslMode" mapstructure:"sslMode"`

	// GormLogLevel sets the GORM logger verbosity (1=Silent, 2=Error, 3=Warn, 4=Info).
	// Default: 1 (Silent)
	GormLogLevel int `yaml:"gormLogLevel" mapstructure:"gormLogLevel"`

	// OpenTelemetry Configuration (optional - nil disables telemetry)
	OTelConfig *pkgotel.Config `yaml:"-" mapstructure:"-"` // Not serializable from config files
}

// effectiveTimeout returns the configured timeout or the default if zero.
func (c *ConnectionConfig) effectiveTimeout() time.Duration {
	if c.Timeout <= 0 {
		return defaultTimeout
	}
	return c.Timeout
}

// effectiveSSLMode returns the configured SSL mode or "require" if empty.
func (c *ConnectionConfig) effectiveSSLMode() string {
	if c.SSLMode == "" {
		return "require"
	}
	return c.SSLMode
}

// effectiveGormLogLevel returns the configured GORM log level or Silent if unset.
func (c *ConnectionConfig) effectiveGormLogLevel() logger.LogLevel {
	if c.GormLogLevel >= int(logger.Silent) && c.GormLogLevel <= int(logger.Info) {
		return logger.LogLevel(c.GormLogLevel)
	}
	return logger.Silent
}

// Validate checks that the ConnectionConfig has all required fields set and
// values are within acceptable ranges. It is called automatically by Pool().
func (c *ConnectionConfig) Validate() error {
	if c.DBType != Mysql && c.DBType != Postgresql && c.DBType != MSSQL {
		return fmt.Errorf("unsupported database type: %q", c.DBType)
	}
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	if c.DBName == "" {
		return fmt.Errorf("dbName is required")
	}
	validPostgresSSL := map[string]bool{
		"disable": true, "require": true, "verify-ca": true,
		"verify-full": true, "prefer": true, "allow": true,
	}
	if c.DBType == Postgresql && c.SSLMode != "" && !validPostgresSSL[c.SSLMode] {
		return fmt.Errorf("invalid SSLMode %q for PostgreSQL", c.SSLMode)
	}
	if c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("MaxIdleConns (%d) cannot exceed MaxOpenConns (%d)", c.MaxIdleConns, c.MaxOpenConns)
	}
	return nil
}

// dsn builds the data source name string for the configured database type.
// It is unexported to prevent accidental logging of credentials.
// Use RedactedDsn() for safe logging.
func (c *ConnectionConfig) dsn() string {
	timeout := c.effectiveTimeout()
	sslMode := c.effectiveSSLMode()

	switch c.DBType {
	case Mysql:
		timeoutStr := fmt.Sprintf("%ds", timeout/time.Second)
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%s",
			c.Username, c.Password, c.Host, c.Port, c.DBName, timeoutStr)
	case Postgresql:
		return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s connect_timeout=%d",
			c.Username, c.Password, c.Host, c.Port, c.DBName, sslMode, int(timeout.Seconds()))
	case MSSQL:
		timeoutStr := fmt.Sprintf("%ds", timeout/time.Second)
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&connectTimeout=%s&encrypt=%s",
			c.Username, c.Password, c.Host, c.Port, c.DBName, timeoutStr, sslMode)
	default:
		return ""
	}
}

// RedactedDsn returns the DSN with the password replaced by "***",
// safe for use in logs and error messages.
func (c *ConnectionConfig) RedactedDsn() string {
	original := c.dsn()
	if c.Password != "" {
		return strings.ReplaceAll(original, c.Password, "***")
	}
	return original
}

// Pool creates a new GORM database connection pool.
//
// It validates the DSN, opens the connection, configures pool parameters,
// pings to verify connectivity, and optionally installs OTel instrumentation.
func (c *ConnectionConfig) Pool() (*gorm.DB, error) {
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	dsn := c.dsn()

	var dialector gorm.Dialector
	switch c.DBType {
	case Mysql:
		dialector = mysql.Open(dsn)
	case Postgresql:
		dialector = postgres.Open(dsn)
	case MSSQL:
		dialector = sqlserver.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.DBType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(c.effectiveGormLogLevel()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection to %s:%d/%s: %w", c.Host, c.Port, c.DBName, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	if c.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	}
	if c.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(c.ConnMaxIdleTime)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), c.effectiveTimeout())
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("failed to ping database at %s:%d/%s: %w", c.Host, c.Port, c.DBName, err)
	}

	// Install OpenTelemetry instrumentation if configured
	if c.OTelConfig != nil && c.OTelConfig.IsTracingEnabled() {
		// Configure otelgorm plugin options
		opts := []otelgorm.Option{
			otelgorm.WithDBName(c.DBName),
			otelgorm.WithAttributes(
				semconv.DBSystemKey.String(string(c.DBType)),
				semconv.ServerAddressKey.String(c.Host),
				semconv.ServerPortKey.Int(c.Port),
			),
		}

		// Use the TracerProvider from OTelConfig
		if c.OTelConfig.TracerProvider != nil {
			opts = append(opts, otelgorm.WithTracerProvider(c.OTelConfig.TracerProvider))
		}

		// Disable metrics if not enabled in config
		if !c.OTelConfig.IsMetricsEnabled() {
			opts = append(opts, otelgorm.WithoutMetrics())
		}

		// Install the uptrace otelgorm plugin
		if err := db.Use(otelgorm.NewPlugin(opts...)); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("failed to install otelgorm plugin: %w", err)
		}

		// Register connection pool metrics if metrics enabled.
		// Note: collectPoolMetrics only registers an observable callback and returns
		// immediately, so it does not need a goroutine.
		if c.OTelConfig.IsMetricsEnabled() {
			c.collectPoolMetrics(sqlDB)
		}
	}

	return db, nil
}

// SQLDB creates a new connection pool internally. The caller is responsible for closing
// the returned *sql.DB. Prefer Pool() when you need the GORM wrapper.
//
// Each call to SQLDB() opens a new connection pool; close the returned *sql.DB when done
// to avoid leaking connections.
func (c *ConnectionConfig) SQLDB() (*sql.DB, error) {
	gormDB, err := c.Pool()
	if err != nil {
		return nil, err
	}

	return gormDB.DB()
}

// collectPoolMetrics registers observable gauge callbacks for connection pool stats.
func (c *ConnectionConfig) collectPoolMetrics(sqlDB *sql.DB) {
	if c.OTelConfig == nil || !c.OTelConfig.IsMetricsEnabled() {
		return
	}

	meter := c.OTelConfig.GetMeter("db.pool")

	// Create metrics instruments
	idleConns, err := meter.Int64ObservableGauge(
		"db.client.connections.idle",
		metric.WithDescription("Number of idle database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db.collectPoolMetrics: failed to create idle gauge: %v\n", err)
		return
	}

	activeConns, err := meter.Int64ObservableGauge(
		"db.client.connections.active",
		metric.WithDescription("Number of active database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db.collectPoolMetrics: failed to create active gauge: %v\n", err)
		return
	}

	totalConns, err := meter.Int64ObservableGauge(
		"db.client.connections.max",
		metric.WithDescription("Maximum number of open database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db.collectPoolMetrics: failed to create max gauge: %v\n", err)
		return
	}

	// Register callback to collect metrics
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			stats := sqlDB.Stats()

			attrs := []attribute.KeyValue{
				attribute.String("db.system", string(c.DBType)),
				attribute.String("db.name", c.DBName),
				attribute.String("server.address", c.Host),
				attribute.Int("server.port", c.Port),
			}

			observer.ObserveInt64(idleConns, int64(stats.Idle), metric.WithAttributes(attrs...))
			observer.ObserveInt64(activeConns, int64(stats.InUse), metric.WithAttributes(attrs...))
			observer.ObserveInt64(totalConns, int64(stats.MaxOpenConnections), metric.WithAttributes(attrs...))

			return nil
		},
		idleConns,
		activeConns,
		totalConns,
	)
	if err != nil {
		// Log error but don't fail
		logger := pkgotel.NewLogHelper(context.Background(), c.OTelConfig,
			"github.com/jasoet/pkg/v2/db", "db.collectPoolMetrics")
		logger.Error(err, "Failed to register pool metrics callback")
	}
}
