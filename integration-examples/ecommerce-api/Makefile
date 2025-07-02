# E-commerce API Makefile

.PHONY: help build run test clean docker-up docker-down docker-logs deps

# Default target
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Development commands
deps: ## Install Go dependencies
	go mod download
	go mod tidy

build: ## Build the application
	go build -o bin/ecommerce-api .

run: ## Run the application locally
	go run main.go

test: ## Run tests
	go test -v ./...

test-integration: ## Run integration tests
	docker-compose up -d postgres redis
	sleep 5
	ECOMMERCE_DATABASE_HOST=localhost go test -v -tags=integration ./...
	docker-compose down

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf uploads/*
	go clean

# Docker commands
docker-build: ## Build Docker image
	docker build -t ecommerce-api:latest .

docker-up: ## Start all services with Docker Compose
	docker-compose up -d

docker-down: ## Stop all services
	docker-compose down -v

docker-logs: ## Show logs from all services
	docker-compose logs -f

docker-restart: ## Restart all services
	docker-compose restart

# Database commands
db-migrate: ## Run database migrations (manual)
	@echo "Database migrations are handled automatically on startup"

db-seed: ## Seed database with sample data
	curl -X POST http://localhost:8080/api/v1/admin/seed \
		-H "Authorization: Bearer demo-admin-token"

# Development helpers
dev-setup: ## Set up development environment
	docker-compose up -d postgres redis
	sleep 10
	make run

dev-reset: ## Reset development environment
	docker-compose down -v
	docker-compose up -d postgres redis
	sleep 10

# API testing
api-test: ## Test API endpoints
	@echo "Testing health endpoint..."
	curl -s http://localhost:8080/api/v1/health | jq
	@echo "\nTesting categories endpoint..."
	curl -s http://localhost:8080/api/v1/categories | jq
	@echo "\nTesting products endpoint..."
	curl -s http://localhost:8080/api/v1/products | jq

api-docs: ## Generate API documentation
	@echo "API documentation is available at:"
	@echo "Health: http://localhost:8080/api/v1/health"
	@echo "Categories: http://localhost:8080/api/v1/categories"
	@echo "Products: http://localhost:8080/api/v1/products"
	@echo "Admin: http://localhost:8080/api/v1/admin/* (requires auth)"

# Monitoring
logs: ## Show application logs
	docker-compose logs -f app

metrics: ## Show application metrics
	curl -s http://localhost:8080/api/v1/metrics | jq

# Quality checks
lint: ## Run linter
	golangci-lint run ./...

security: ## Run security checks
	gosec ./...

format: ## Format code
	gofmt -w .
	goimports -w .

check-all: deps lint security test ## Run all quality checks

# Production commands
prod-build: ## Build for production
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o bin/ecommerce-api .

prod-deploy: prod-build docker-build ## Build and prepare for production deployment
	@echo "Production deployment ready. Push image to registry:"
	@echo "docker tag ecommerce-api:latest your-registry/ecommerce-api:latest"
	@echo "docker push your-registry/ecommerce-api:latest"