//go:build example

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/otel"
	"go.opentelemetry.io/otel/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func main() {
	fmt.Println("OpenTelemetry LoggerProvider Examples")
	fmt.Println("======================================")

	// Example 1: Basic LoggerProvider Setup
	fmt.Println("\n1. Basic LoggerProvider Setup")
	basicLoggerProviderExample()

	// Example 2: LoggerProvider with OTel Config
	fmt.Println("\n2. LoggerProvider with OTel Config")
	otelConfigExample()

	// Example 3: Automatic Trace Correlation
	fmt.Println("\n3. Automatic Trace Correlation")
	traceCorrelationExample()

	// Example 4: Multiple Scopes
	fmt.Println("\n4. Multiple Logger Scopes")
	multipleScopesExample()

	// Example 5: Different Severity Levels
	fmt.Println("\n5. Different Severity Levels")
	severityLevelsExample()

	fmt.Println("\n✓ All examples completed!")
}

func basicLoggerProviderExample() {
	// Create a LoggerProvider with zerolog backend
	provider := logging.NewLoggerProvider("basic-example", true)

	// Get a logger from the provider
	logger := provider.Logger("main")

	// Create a log record
	var record log.Record
	record.SetBody(log.StringValue("Application started"))
	record.SetSeverity(log.SeverityInfo)
	record.SetTimestamp(time.Now())

	// Emit the log
	logger.Emit(context.Background(), record)

	fmt.Println("✓ Basic LoggerProvider created and used")
	fmt.Println("  Check above for log output with service name, scope, and timestamp")
}

func otelConfigExample() {
	// Create OTel config with LoggerProvider
	cfg := otel.NewConfig("otel-example").
		WithServiceVersion("1.0.0").
		WithLoggerProvider(logging.NewLoggerProvider("otel-example", false))

	// Get a logger from the config
	logger := cfg.GetLogger("business-logic")

	// Create and emit a log record
	var record log.Record
	record.SetBody(log.StringValue("Processing business logic"))
	record.SetSeverity(log.SeverityInfo)
	record.AddAttributes(
		log.String("operation", "calculate"),
		log.Int64("input", 42),
		log.Bool("cached", false),
	)

	logger.Emit(context.Background(), record)

	fmt.Println("✓ OTel Config with LoggerProvider demonstrated")
	fmt.Println("  Using cfg.GetLogger() to get scoped loggers")
}

func traceCorrelationExample() {
	// Create LoggerProvider
	provider := logging.NewLoggerProvider("trace-example", true)
	logger := provider.Logger("trace-scope")

	// Create a TracerProvider for testing
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("trace-example")

	// Start a span
	ctx, span := tracer.Start(context.Background(), "ProcessOrder")
	defer span.End()

	// Create log record
	var record log.Record
	record.SetBody(log.StringValue("Processing order with trace context"))
	record.SetSeverity(log.SeverityInfo)
	record.SetTimestamp(time.Now())
	record.AddAttributes(
		log.String("order_id", "ORDER-12345"),
		log.Float64("amount", 99.99),
	)

	// Emit log - this will automatically include trace_id and span_id
	logger.Emit(ctx, record)

	fmt.Println("✓ Trace correlation demonstrated")
	fmt.Println("  Notice the trace_id, span_id, and trace_flags in the log output")
	fmt.Println("  These fields enable log-span correlation in Grafana!")
}

func multipleScopesExample() {
	provider := logging.NewLoggerProvider("multi-scope", false)

	// Create loggers for different components
	authLogger := provider.Logger("auth")
	dbLogger := provider.Logger("database")
	apiLogger := provider.Logger("api")

	// Log from auth scope
	var authRecord log.Record
	authRecord.SetBody(log.StringValue("User authentication successful"))
	authRecord.SetSeverity(log.SeverityInfo)
	authRecord.AddAttributes(log.String("user_id", "12345"))
	authLogger.Emit(context.Background(), authRecord)

	// Log from database scope
	var dbRecord log.Record
	dbRecord.SetBody(log.StringValue("Query executed"))
	dbRecord.SetSeverity(log.SeverityDebug)
	dbRecord.AddAttributes(
		log.String("query", "SELECT * FROM users"),
		log.Int64("duration_ms", 45),
	)
	dbLogger.Emit(context.Background(), dbRecord)

	// Log from API scope
	var apiRecord log.Record
	apiRecord.SetBody(log.StringValue("Request completed"))
	apiRecord.SetSeverity(log.SeverityInfo)
	apiRecord.AddAttributes(
		log.String("method", "GET"),
		log.String("path", "/api/users"),
		log.Int64("status", 200),
	)
	apiLogger.Emit(context.Background(), apiRecord)

	fmt.Println("✓ Multiple scopes demonstrated")
	fmt.Println("  Each logger has its own scope field (auth, database, api)")
}

func severityLevelsExample() {
	provider := logging.NewLoggerProvider("severity-example", true)
	logger := provider.Logger("severity-test")
	ctx := context.Background()

	severities := []struct {
		severity log.Severity
		message  string
	}{
		{log.SeverityDebug, "Debug message - detailed information"},
		{log.SeverityInfo, "Info message - general information"},
		{log.SeverityWarn, "Warning message - something needs attention"},
		{log.SeverityError, "Error message - operation failed"},
	}

	for _, s := range severities {
		var record log.Record
		record.SetBody(log.StringValue(s.message))
		record.SetSeverity(s.severity)
		record.SetTimestamp(time.Now())

		// Check if logger is enabled for this severity
		if logger.Enabled(ctx, log.EnabledParameters{Severity: s.severity}) {
			logger.Emit(ctx, record)
		}
	}

	fmt.Println("✓ Different severity levels demonstrated")
	fmt.Println("  Debug, Info, Warn, and Error logs shown")
	fmt.Println("  Notice the different colors and log levels in the output")
}
