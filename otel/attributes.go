package otel

import "go.opentelemetry.io/otel/attribute"

// Semantic conventions specific to github.com/jasoet/pkg/v2 library.
// These complement standard OpenTelemetry semantic conventions.
const (
	// Package identification
	AttrPackageName    = attribute.Key("pkg.name")
	AttrPackageVersion = attribute.Key("pkg.version")

	// Server package attributes
	AttrServerPort = attribute.Key("pkg.server.port")

	// gRPC package attributes
	AttrGRPCMode           = attribute.Key("pkg.grpc.mode")
	AttrGRPCPort           = attribute.Key("pkg.grpc.port")
	AttrGRPCHTTPPort       = attribute.Key("pkg.grpc.http_port")
	AttrGRPCReflection     = attribute.Key("pkg.grpc.reflection_enabled")
	AttrGRPCGatewayEnabled = attribute.Key("pkg.grpc.gateway_enabled")

	// REST client package attributes
	AttrRESTClientName   = attribute.Key("pkg.rest.client.name")
	AttrRESTRetryCount   = attribute.Key("pkg.rest.retry.max_count")
	AttrRESTRetryAttempt = attribute.Key("pkg.rest.retry.attempt")
	AttrRESTTimeout      = attribute.Key("pkg.rest.timeout_ms")

	// Database package attributes
	AttrDBConnectionPool = attribute.Key("pkg.db.pool.name")
	AttrDBType           = attribute.Key("pkg.db.type")
	AttrDBMaxIdleConns   = attribute.Key("pkg.db.pool.max_idle")
	AttrDBMaxOpenConns   = attribute.Key("pkg.db.pool.max_open")

	// Concurrent package attributes
	AttrConcurrentTaskCount   = attribute.Key("pkg.concurrent.task.count")
	AttrConcurrentTaskSuccess = attribute.Key("pkg.concurrent.task.success")
	AttrConcurrentTaskFailed  = attribute.Key("pkg.concurrent.task.failed")
	AttrConcurrentMaxWorkers  = attribute.Key("pkg.concurrent.max_workers")
)

// Common attribute values
const (
	// gRPC modes
	GRPCModeSeparate = "separate"
	GRPCModeH2C      = "h2c"

	// Database types
	DBTypePostgreSQL = "postgresql"
	DBTypeMySQL      = "mysql"
	DBTypeMSSQL      = "mssql"
)
