.PHONY: proto run clean docker-up docker-down help

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/user.proto
	@echo "✓ Proto files generated"

# Run the example
run: proto
	@echo "Starting fullstack OTel example..."
	@go run -tags example main.go

# Start dependencies (Jaeger, PostgreSQL)
docker-up:
	@echo "Starting dependencies with Docker Compose..."
	@docker-compose up -d
	@echo "✓ Jaeger UI: http://localhost:16686"
	@echo "✓ OTLP endpoint: localhost:4318"
	@echo "✓ PostgreSQL: localhost:5432"

# Stop dependencies
docker-down:
	@echo "Stopping dependencies..."
	@docker-compose down

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@find proto -name "*.pb.go" -delete
	@echo "✓ Generated files removed"

# Show help
help:
	@echo "Full-Stack OpenTelemetry Example"
	@echo ""
	@echo "Available commands:"
	@echo "  make proto       - Generate protobuf files"
	@echo "  make run         - Run the example (generates proto first)"
	@echo "  make docker-up   - Start Jaeger and PostgreSQL"
	@echo "  make docker-down - Stop all Docker containers"
	@echo "  make clean       - Remove generated protobuf files"
	@echo ""
	@echo "Quick start:"
	@echo "  1. make docker-up    # Start Jaeger"
	@echo "  2. make run          # Run the example"
	@echo "  3. Open http://localhost:16686 to view traces"
