package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pkgotel "github.com/jasoet/pkg/v2/otel"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
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
)

type ConnectionConfig struct {
	DbType       DatabaseType  `yaml:"dbType" validate:"required,oneof=MYSQL POSTGRES MSSQL" mapstructure:"dbType"`
	Host         string        `yaml:"host" validate:"required,min=1" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	Username     string        `yaml:"username" validate:"required,min=1" mapstructure:"username"`
	Password     string        `yaml:"password" mapstructure:"password"`
	DbName       string        `yaml:"dbName" validate:"required,min=1" mapstructure:"dbName"`
	Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout"`
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
	if c.OTelConfig != nil && c.OTelConfig.IsTracingEnabled() {
		// Configure otelgorm plugin options
		opts := []otelgorm.Option{
			otelgorm.WithDBName(c.DbName),
			otelgorm.WithAttributes(
				semconv.DBSystemKey.String(string(c.DbType)),
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
			return nil, fmt.Errorf("failed to install otelgorm plugin: %w", err)
		}

		// Start custom connection pool metrics collection in background if metrics enabled
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
		logger := pkgotel.NewLogHelper(context.Background(), c.OTelConfig,
			"github.com/jasoet/pkg/v2/db", "db.collectPoolMetrics")
		logger.Error(err, "Failed to register pool metrics callback")
	}
}
