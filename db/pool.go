package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pkgotel "github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseType string

const (
	Mysql      DatabaseType = "MYSQL"
	Postgresql DatabaseType = "POSTGRES"
	MSSQL      DatabaseType = "MSSQL"

	// Default database operation name for tracing
	defaultDBOperation = "db.query"
)

type ConnectionConfig struct {
	DbType       DatabaseType  `yaml:"dbType" validate:"required,oneof=MYSQL POSTGRES MSSQL" mapstructure:"dbType"`
	Host         string        `yaml:"host" validate:"required,min=1" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	Username     string        `yaml:"username" validate:"required,min=1" mapstructure:"username"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DbName       string        `yaml:"dbName" validate:"required,min=1" mapstructure:"dbName"`
	Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout" validate:"min=3s"`
	MaxIdleConns int           `yaml:"maxIdleConns" mapstructure:"maxIdleConns" validate:"min=1"`
	MaxOpenConns int           `yaml:"maxOpenConns" mapstructure:"maxOpenConns" validate:"min=2"`

	// OpenTelemetry Configuration (optional - nil disables telemetry)
	OTelConfig *pkgotel.Config `yaml:"-" mapstructure:"-"` // Not serializable from config files
}

func (c *ConnectionConfig) Dsn() string {
	timeoutString := fmt.Sprintf("%ds", c.Timeout/time.Second)

	var dsn string
	switch c.DbType {
	case Mysql:
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=%s", c.Username, c.Password, c.Host, c.Port, c.DbName, timeoutString)
	case Postgresql:
		dsn = fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=disable connect_timeout=%d", c.Username, c.Password, c.Host, c.Port, c.DbName, int(c.Timeout.Seconds()))
	case MSSQL:
		dsn = fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&connectTimeout=%s&encrypt=disable", c.Username, c.Password, c.Host, c.Port, c.DbName, timeoutString)
	}

	return dsn
}

func (c *ConnectionConfig) Pool() (*gorm.DB, error) {
	if c.Dsn() == "" {
		return nil, fmt.Errorf("dsn is empty")
	}

	var dialector gorm.Dialector
	switch c.DbType {
	case Mysql:
		dialector = mysql.Open(c.Dsn())
	case Postgresql:
		dialector = postgres.Open(c.Dsn())
	case MSSQL:
		dialector = sqlserver.Open(c.Dsn())
	default:
		return nil, fmt.Errorf("unsupported database type: %s", c.DbType)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	// Install OpenTelemetry instrumentation if configured
	if c.OTelConfig != nil {
		if err := c.installOTelCallbacks(db); err != nil {
			return nil, fmt.Errorf("failed to install OTel callbacks: %w", err)
		}

		// Start connection pool metrics collection in background
		if c.OTelConfig.IsMetricsEnabled() {
			go c.collectPoolMetrics(sqlDB)
		}
	}

	return db, nil
}

func (c *ConnectionConfig) SQLDB() (*sql.DB, error) {
	gormDB, err := c.Pool()
	if err != nil {
		return nil, err
	}

	return gormDB.DB()
}

// installOTelCallbacks installs GORM callbacks for OpenTelemetry tracing
// nolint:gocyclo // Complexity is due to registering callbacks for 6 GORM operations with proper error handling
func (c *ConnectionConfig) installOTelCallbacks(db *gorm.DB) error {
	if c.OTelConfig == nil || !c.OTelConfig.IsTracingEnabled() {
		return nil
	}

	tracer := c.OTelConfig.GetTracer("db.gorm")

	// Callback for before query
	beforeCallback := func(db *gorm.DB) {
		ctx := db.Statement.Context
		if ctx == nil {
			return
		}

		// Start a new span
		operation := defaultDBOperation
		if db.Statement.SQL.String() != "" {
			operation = extractOperationType(db.Statement.SQL.String())
		}

		ctx, span := tracer.Start(ctx, operation,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.DBSystemKey.String(string(c.DbType)),
				attribute.String("db.name", c.DbName),
				semconv.ServerAddressKey.String(c.Host),
				semconv.ServerPortKey.Int(c.Port),
			),
		)

		// Store span in statement context
		db.Statement.Context = ctx
		db.InstanceSet("otel:span", span)
		db.InstanceSet("otel:start_time", time.Now())
	}

	// Callback for after query
	afterCallback := func(db *gorm.DB) {
		spanVal, ok := db.InstanceGet("otel:span")
		if !ok {
			return
		}

		span, ok := spanVal.(trace.Span)
		if !ok {
			return
		}
		defer span.End()

		// Add query details
		if db.Statement.SQL.String() != "" {
			span.SetAttributes(
				attribute.String("db.statement", db.Statement.SQL.String()),
			)
		}

		if db.Statement.Table != "" {
			span.SetAttributes(
				semconv.DBCollectionNameKey.String(db.Statement.Table),
			)
		}

		// Record rows affected
		span.SetAttributes(
			attribute.Int64("db.rows_affected", db.Statement.RowsAffected),
		)

		// Record error if any
		if db.Error != nil && db.Error != gorm.ErrRecordNotFound {
			span.RecordError(db.Error)
			span.SetAttributes(
				attribute.String("db.error", db.Error.Error()),
			)
		}

		// Calculate duration
		if startTime, ok := db.InstanceGet("otel:start_time"); ok {
			if t, ok := startTime.(time.Time); ok {
				duration := time.Since(t)
				span.SetAttributes(
					attribute.Int64("db.duration_ms", duration.Milliseconds()),
				)
			}
		}
	}

	// Register callbacks
	if err := db.Callback().Create().Before("gorm:create").Register("otel:before_create", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Create().After("gorm:create").Register("otel:after_create", afterCallback); err != nil {
		return err
	}

	if err := db.Callback().Query().Before("gorm:query").Register("otel:before_query", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Query().After("gorm:query").Register("otel:after_query", afterCallback); err != nil {
		return err
	}

	if err := db.Callback().Update().Before("gorm:update").Register("otel:before_update", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Update().After("gorm:update").Register("otel:after_update", afterCallback); err != nil {
		return err
	}

	if err := db.Callback().Delete().Before("gorm:delete").Register("otel:before_delete", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("gorm:delete").Register("otel:after_delete", afterCallback); err != nil {
		return err
	}

	if err := db.Callback().Row().Before("gorm:row").Register("otel:before_row", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Row().After("gorm:row").Register("otel:after_row", afterCallback); err != nil {
		return err
	}

	if err := db.Callback().Raw().Before("gorm:raw").Register("otel:before_raw", beforeCallback); err != nil {
		return err
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("otel:after_raw", afterCallback); err != nil {
		return err
	}

	return nil
}

// extractOperationType extracts the operation type from SQL query
func extractOperationType(sql string) string {
	if len(sql) == 0 {
		return defaultDBOperation
	}

	// Simple extraction of first word (operation type)
	for i, c := range sql {
		if c == ' ' || c == '\n' || c == '\t' {
			operation := sql[:i]
			return "db." + operation
		}
	}

	return defaultDBOperation
}

// collectPoolMetrics periodically collects connection pool metrics
func (c *ConnectionConfig) collectPoolMetrics(sqlDB *sql.DB) {
	if c.OTelConfig == nil || !c.OTelConfig.IsMetricsEnabled() {
		return
	}

	meter := c.OTelConfig.GetMeter("db.pool")

	// Create metrics instruments
	// Note: errors are intentionally ignored as they only occur with nil meter (checked by GetMeter)
	idleConns, _ := meter.Int64ObservableGauge( //nolint:errcheck
		"db.client.connections.idle",
		metric.WithDescription("Number of idle database connections"),
		metric.WithUnit("{connection}"),
	)

	activeConns, _ := meter.Int64ObservableGauge( //nolint:errcheck
		"db.client.connections.active",
		metric.WithDescription("Number of active database connections"),
		metric.WithUnit("{connection}"),
	)

	totalConns, _ := meter.Int64ObservableGauge( //nolint:errcheck
		"db.client.connections.max",
		metric.WithDescription("Maximum number of open database connections"),
		metric.WithUnit("{connection}"),
	)

	// Register callback to collect metrics
	_, err := meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			stats := sqlDB.Stats()

			attrs := []attribute.KeyValue{
				attribute.String("db.system", string(c.DbType)),
				attribute.String("db.name", c.DbName),
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
		fmt.Printf("Failed to register pool metrics callback: %v\n", err)
	}
}
