// Package db provides database connection pooling with GORM, supporting PostgreSQL,
// MySQL, and MSSQL, with optional OpenTelemetry instrumentation.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"weak"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pkgotel "github.com/jasoet/pkg/v3/otel"
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

	// defaultMaxIdleConns is applied when MaxIdleConns is unset (<= 0) so a
	// zero-value config still keeps idle connections instead of dialing a fresh
	// TCP connection for every query.
	defaultMaxIdleConns = 10

	// defaultMaxOpenConns is applied when MaxOpenConns is unset (<= 0) to cap the
	// pool at a sane upper bound rather than leaving it effectively unbounded.
	defaultMaxOpenConns = 100
)

// validMSSQLSSL maps the accepted SSLMode values for MSSQL to the corresponding
// go-mssqldb "encrypt" DSN value. The Postgres-style "require" (and the default,
// empty SSLMode) map to "true"; go-mssqldb does not accept "require" itself.
var validMSSQLSSL = map[string]string{
	"disable": "disable",
	"false":   "false",
	"true":    "true",
	"require": "true",
	"strict":  "strict",
}

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
	// PostgreSQL: "disable", "require", "verify-ca", "verify-full", "prefer", "allow" (default: "require")
	// MSSQL: "disable", "false", "true", "require", "strict" (default: "require").
	//   "require" and "true" both request mandatory encryption; go-mssqldb's own
	//   "encrypt" values are used in the DSN ("require" is mapped to "true").
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

// mssqlEncrypt maps the configured SSLMode to a valid go-mssqldb "encrypt" value.
// Unknown modes fall back to "true" (mandatory encryption); Validate() rejects
// unknown modes before this is reached on the NewPool path.
func (c *ConnectionConfig) mssqlEncrypt() string {
	if v, ok := validMSSQLSSL[c.effectiveSSLMode()]; ok {
		return v
	}
	return "true"
}

// connectTimeoutSeconds returns the effective timeout rounded up to whole seconds,
// with a floor of 1. Rounding up avoids a sub-second timeout truncating to 0, which
// PostgreSQL treats as "no timeout" (indefinite wait).
func (c *ConnectionConfig) connectTimeoutSeconds() int {
	d := c.effectiveTimeout()
	secs := int((d + time.Second - 1) / time.Second)
	if secs < 1 {
		secs = 1
	}
	return secs
}

// effectiveMaxIdleConns returns MaxIdleConns or the default when unset (<= 0).
func (c *ConnectionConfig) effectiveMaxIdleConns() int {
	if c.MaxIdleConns <= 0 {
		return defaultMaxIdleConns
	}
	return c.MaxIdleConns
}

// effectiveMaxOpenConns returns MaxOpenConns or the default when unset (<= 0).
func (c *ConnectionConfig) effectiveMaxOpenConns() int {
	if c.MaxOpenConns <= 0 {
		return defaultMaxOpenConns
	}
	return c.MaxOpenConns
}

// effectiveGormLogLevel returns the configured GORM log level or Silent if unset.
func (c *ConnectionConfig) effectiveGormLogLevel() logger.LogLevel {
	if c.GormLogLevel >= int(logger.Silent) && c.GormLogLevel <= int(logger.Info) {
		return logger.LogLevel(c.GormLogLevel)
	}
	return logger.Silent
}

// Validate checks that the ConnectionConfig has all required fields set and
// values are within acceptable ranges. It is called automatically by NewPool().
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
	if c.DBType == MSSQL && c.SSLMode != "" {
		if _, ok := validMSSQLSSL[c.SSLMode]; !ok {
			return fmt.Errorf("invalid SSLMode %q for MSSQL (valid: disable, false, true, require, strict)", c.SSLMode)
		}
	}
	// MaxOpenConns <= 0 means "unset" (a default is applied); only reject when an
	// explicit open limit is smaller than the requested idle count.
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("MaxIdleConns (%d) cannot exceed MaxOpenConns (%d)", c.MaxIdleConns, c.MaxOpenConns)
	}
	return nil
}

// dsn builds the data source name string for the configured database type.
// It is unexported to prevent accidental logging of credentials.
// Use RedactedDsn() for safe logging.
func (c *ConnectionConfig) dsn() string {
	return c.dsnWithPassword(c.Password)
}

// quotePostgresValue escapes a value for the PostgreSQL keyword/value DSN format.
// It backslash-escapes backslashes and single quotes, then wraps the value in
// single quotes. This is the libpq/pgx quoting convention and prevents a value
// (e.g. a password) from being interpreted as additional connection parameters.
func quotePostgresValue(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

// dsnWithPassword builds the DSN using pw in the password position, so callers
// can substitute a mask without corrupting other fields that happen to contain
// the real password as a substring.
//
// All user-controlled values are escaped for their driver's DSN grammar so that
// credentials containing special characters cannot inject or override connection
// parameters (e.g. a password cannot smuggle sslmode=disable or a different host).
func (c *ConnectionConfig) dsnWithPassword(pw string) string {
	switch c.DBType {
	case Mysql:
		// go-sql-driver accepts Go duration strings, so the effective timeout is
		// formatted directly (preserving sub-second values instead of truncating).
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%s",
			c.Username, pw, c.Host, c.Port, c.DBName, c.effectiveTimeout().String())
	case Postgresql:
		return fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s connect_timeout=%d",
			quotePostgresValue(c.Username), quotePostgresValue(pw), quotePostgresValue(c.Host),
			c.Port, quotePostgresValue(c.DBName), c.effectiveSSLMode(), c.connectTimeoutSeconds())
	case MSSQL:
		// Build via net/url so the userinfo and query values are percent-encoded.
		query := url.Values{}
		query.Set("database", c.DBName)
		query.Set("connection timeout", strconv.Itoa(c.connectTimeoutSeconds()))
		query.Set("encrypt", c.mssqlEncrypt())
		u := url.URL{
			Scheme:   "sqlserver",
			User:     url.UserPassword(c.Username, pw),
			Host:     net.JoinHostPort(c.Host, strconv.Itoa(c.Port)),
			RawQuery: query.Encode(),
		}
		return u.String()
	default:
		return ""
	}
}

// RedactedDsn returns the DSN with the password replaced by "***",
// safe for use in logs and error messages.
func (c *ConnectionConfig) RedactedDsn() string {
	if c.Password == "" {
		return c.dsn()
	}
	return c.dsnWithPassword("***")
}

// Option configures a ConnectionConfig during NewPool.
type Option func(*ConnectionConfig)

// WithConnectionConfig seeds the pool configuration from cfg.
// Apply it first — it replaces the whole config.
func WithConnectionConfig(cfg ConnectionConfig) Option {
	return func(c *ConnectionConfig) {
		*c = cfg
	}
}

// WithOTelConfig overrides the ConnectionConfig's OTelConfig when cfg is non-nil.
func WithOTelConfig(cfg *pkgotel.Config) Option {
	return func(c *ConnectionConfig) {
		if cfg != nil {
			c.OTelConfig = cfg
		}
	}
}

// NewPool creates a new GORM database connection pool from the given options.
//
// It starts from an empty ConnectionConfig, applies opts in order, then runs the
// pool pipeline: validate, open, configure pool parameters, ping, and optionally
// install OTel instrumentation.
func NewPool(opts ...Option) (*gorm.DB, error) {
	cfg := ConnectionConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg.openPool()
}

// openPool validates the config, opens the connection, configures pool
// parameters, pings to verify connectivity, and optionally installs OTel
// instrumentation.
func (c *ConnectionConfig) openPool() (*gorm.DB, error) {
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

	// DisableAutomaticPing: gorm.Open would otherwise open the pool and immediately
	// ping it, so a failure there leaks the underlying sql.DB (we never receive a
	// handle to close). We disable that ping and run our own below, on a handle we
	// can close on failure.
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:               logger.Default.LogMode(c.effectiveGormLogLevel()),
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection to %s:%d/%s: %w", c.Host, c.Port, c.DBName, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool. Zero-value config fields fall back to sane
	// defaults so idle pooling is not silently disabled.
	sqlDB.SetMaxIdleConns(c.effectiveMaxIdleConns())
	sqlDB.SetMaxOpenConns(c.effectiveMaxOpenConns())
	if c.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)
	}
	if c.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(c.ConnMaxIdleTime)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), c.effectiveTimeout())
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database at %s:%d/%s: %w", c.Host, c.Port, c.DBName, err)
	}

	// Install OpenTelemetry instrumentation if configured.
	// Tracing and metrics are gated independently: the otelgorm plugin requires
	// tracing, while pool metrics only require a MeterProvider.
	if c.OTelConfig != nil && c.OTelConfig.IsTracingEnabled() {
		// Configure otelgorm plugin options.
		//
		// WithoutMetrics is always passed: otelgorm reports DBStats via otelsql on
		// the GLOBAL meter provider, whereas collectPoolMetrics below reports pool
		// metrics on the configured provider. Enabling both would emit duplicate
		// connection-pool metrics under different names/providers.
		opts := []otelgorm.Option{
			otelgorm.WithDBName(c.DBName),
			otelgorm.WithAttributes(
				semconv.DBSystemKey.String(string(c.DBType)),
				semconv.ServerAddressKey.String(c.Host),
				semconv.ServerPortKey.Int(c.Port),
			),
			otelgorm.WithoutMetrics(),
		}

		// Use the TracerProvider from OTelConfig
		if c.OTelConfig.TracerProvider != nil {
			opts = append(opts, otelgorm.WithTracerProvider(c.OTelConfig.TracerProvider))
		}

		// Install the uptrace otelgorm plugin
		if err := db.Use(otelgorm.NewPlugin(opts...)); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("failed to install otelgorm plugin: %w", err)
		}
	}

	// Register connection pool metrics if metrics enabled, independently of tracing.
	// Note: collectPoolMetrics only registers an observable callback and returns
	// immediately, so it does not need a goroutine.
	if c.OTelConfig != nil && c.OTelConfig.IsMetricsEnabled() {
		c.collectPoolMetrics(sqlDB)
	}

	return db, nil
}

// SQLDB creates a new connection pool internally. The caller is responsible for closing
// the returned *sql.DB. Prefer NewPool() when you need the GORM wrapper.
//
// Each call to SQLDB() opens a new connection pool; close the returned *sql.DB when done
// to avoid leaking connections.
func (c *ConnectionConfig) SQLDB() (*sql.DB, error) {
	gormDB, err := NewPool(WithConnectionConfig(*c))
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

	attrs := []attribute.KeyValue{
		attribute.String("db.system", string(c.DBType)),
		attribute.String("db.name", c.DBName),
		attribute.String("server.address", c.Host),
		attribute.Int("server.port", c.Port),
	}

	// Hold sqlDB weakly inside the callback so that the callback (retained by the
	// meter provider via its Registration) does not keep the pool reachable after
	// the caller drops it. Once the pool is garbage-collected the runtime cleanup
	// below unregisters the callback, so repeated NewPool calls do not accumulate
	// callbacks emitting stale gauges from closed pools.
	weakDB := weak.Make(sqlDB)

	reg, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			db := weakDB.Value()
			if db == nil {
				return nil
			}
			stats := db.Stats()

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
			"github.com/jasoet/pkg/v3/db", "db.collectPoolMetrics")
		logger.Error(err, "Failed to register pool metrics callback")
		return
	}

	// Retain the Registration so the callback is unregistered when the pool is no
	// longer reachable. arg (reg) must not be reachable from sqlDB for the cleanup
	// to run; it is only referenced by the meter provider, so this holds.
	runtime.AddCleanup(sqlDB, func(r metric.Registration) { _ = r.Unregister() }, reg)
}
