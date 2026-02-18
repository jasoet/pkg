//go:build example

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/jasoet/pkg/v2/otel"
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

	// Example 6: OTLP Log Export (commented out - requires OTLP collector)
	fmt.Println("\n6. OTLP Log Export (see code for details)")
	fmt.Println("  Uncomment otlpLogExportExample() to test with OTLP collector")

	fmt.Println("\nAll examples completed!")
}

func basicLoggerProviderExample() {
	// Create a LoggerProvider with debug level
	provider, err := otel.NewLoggerProviderWithOptions("basic-example", otel.WithLogLevel(logging.LogLevelDebug))
	if err != nil {
		fmt.Printf("Failed to create logger provider: %v\n", err)
		return
	}

	// Get a logger from the provider
	logger := provider.Logger("main")

	// Create a log record
	var record log.Record
	record.SetBody(log.StringValue("Application started"))
	record.SetSeverity(log.SeverityInfo)
	record.SetTimestamp(time.Now())

	// Emit the log
	logger.Emit(context.Background(), record)

	fmt.Println("Basic LoggerProvider created and used")
	fmt.Println("  Check above for log output with service name, scope, and timestamp")
}

func otelConfigExample() {
	// Create LoggerProvider (default info level)
	provider, err := otel.NewLoggerProviderWithOptions("otel-example")
	if err != nil {
		fmt.Printf("Failed to create logger provider: %v\n", err)
		return
	}

	// Create OTel config with LoggerProvider
	cfg := otel.NewConfig("otel-example").
		WithServiceVersion("1.0.0").
		WithLoggerProvider(provider)

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

	fmt.Println("OTel Config with LoggerProvider demonstrated")
	fmt.Println("  Using cfg.GetLogger() to get scoped loggers")
}

func traceCorrelationExample() {
	// Create LoggerProvider with debug level
	provider, err := otel.NewLoggerProviderWithOptions("trace-example", otel.WithLogLevel(logging.LogLevelDebug))
	if err != nil {
		fmt.Printf("Failed to create logger provider: %v\n", err)
		return
	}
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

	fmt.Println("Trace correlation demonstrated")
	fmt.Println("  Notice the trace_id, span_id, and trace_flags in the log output")
	fmt.Println("  These fields enable log-span correlation in Grafana!")
}

func multipleScopesExample() {
	// Create LoggerProvider (default info level)
	provider, err := otel.NewLoggerProviderWithOptions("multi-scope")
	if err != nil {
		fmt.Printf("Failed to create logger provider: %v\n", err)
		return
	}

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

	fmt.Println("Multiple scopes demonstrated")
	fmt.Println("  Each logger has its own scope field (auth, database, api)")
}

func severityLevelsExample() {
	// Create LoggerProvider with debug level to see all severities
	provider, err := otel.NewLoggerProviderWithOptions("severity-example", otel.WithLogLevel(logging.LogLevelDebug))
	if err != nil {
		fmt.Printf("Failed to create logger provider: %v\n", err)
		return
	}
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

	fmt.Println("Different severity levels demonstrated")
	fmt.Println("  Debug, Info, Warn, and Error logs shown")
	fmt.Println("  Notice the different colors and log levels in the output")
}

// otlpLogExportExample demonstrates using OTLP log export.
// Uncomment this function call in main() to test with a running OTLP collector.
// You can use Grafana, Jaeger, or any OTLP-compatible backend.
//
// To test locally:
// 1. Start Grafana with OTLP receiver (docker-compose or local setup)
// 2. Uncomment this function in main()
// 3. Run the example
// 4. Check Grafana Loki or your OTLP backend for the exported logs
func otlpLogExportExample() {
	// Create LoggerProvider with OTLP export
	// This will send logs to localhost:4318 (standard OTLP HTTP port)
	provider, err := otel.NewLoggerProviderWithOptions("otlp-example",
		otel.WithOTLPEndpoint("localhost:4318", true), // insecure=true for local testing
		otel.WithConsoleOutput(true),                  // also log to console
	)
	if err != nil {
		fmt.Printf("Failed to create OTLP logger provider: %v\n", err)
		return
	}

	logger := provider.Logger("otlp-scope")

	// Create and emit log records
	for i := 0; i < 3; i++ {
		var record log.Record
		record.SetBody(log.StringValue(fmt.Sprintf("Log entry %d exported to OTLP", i+1)))
		record.SetSeverity(log.SeverityInfo)
		record.SetTimestamp(time.Now())
		record.AddAttributes(
			log.Int64("iteration", int64(i+1)),
			log.String("destination", "otlp-collector"),
		)

		logger.Emit(context.Background(), record)
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("OTLP log export demonstrated")
	fmt.Println("  Logs sent to OTLP collector at localhost:4318")
	fmt.Println("  Check your OTLP backend (Grafana/Jaeger) to see the exported logs")
}
