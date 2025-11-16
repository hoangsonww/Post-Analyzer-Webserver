.PHONY: help build run test clean docker-build docker-up docker-down lint format install-tools dev migrate

# Variables
APP_NAME := post-analyzer
MAIN_FILE := main_new.go
BINARY := $(APP_NAME)
DOCKER_IMAGE := $(APP_NAME):latest
GO := go
GOFLAGS := -v
LDFLAGS := -w -s

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "$(BLUE)Post Analyzer Webserver - Makefile Commands$(NC)"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'

## install: Install dependencies
install:
	@echo "$(GREEN)Installing dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod verify

## install-tools: Install development tools
install-tools:
	@echo "$(GREEN)Installing development tools...$(NC)"
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install github.com/swaggo/swag/cmd/swag@latest

## build: Build the application
build:
	@echo "$(GREEN)Building $(APP_NAME)...$(NC)"
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BINARY) $(MAIN_FILE)
	@echo "$(GREEN)Build complete: $(BINARY)$(NC)"

## run: Run the application
run: build
	@echo "$(GREEN)Running $(APP_NAME)...$(NC)"
	./$(BINARY)

## dev: Run the application in development mode with file watching
dev:
	@echo "$(GREEN)Running in development mode...$(NC)"
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "$(YELLOW)Air not installed. Running normally...$(NC)"; \
		$(GO) run $(MAIN_FILE); \
	fi

## test: Run all tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	$(GO) test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

## test-coverage: Run tests with coverage report
test-coverage: test
	@echo "$(GREEN)Generating coverage report...$(NC)"
	$(GO) tool cover -html=coverage.txt -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## lint: Run linter
lint:
	@echo "$(GREEN)Running linter...$(NC)"
	golangci-lint run --timeout=5m

## format: Format Go code
format:
	@echo "$(GREEN)Formatting code...$(NC)"
	$(GO) fmt ./...
	gofmt -s -w .

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -f $(BINARY)
	rm -f coverage.txt coverage.html
	rm -rf dist/
	$(GO) clean

## docker-build: Build Docker image
docker-build:
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE) .

## docker-up: Start all services with Docker Compose
docker-up:
	@echo "$(GREEN)Starting services...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Services started. Application available at http://localhost:8080$(NC)"
	@echo "$(GREEN)Prometheus available at http://localhost:9090$(NC)"
	@echo "$(GREEN)Grafana available at http://localhost:3000 (admin/admin)$(NC)"

## docker-down: Stop all services
docker-down:
	@echo "$(YELLOW)Stopping services...$(NC)"
	docker-compose down

## docker-logs: View logs from all services
docker-logs:
	docker-compose logs -f

## docker-restart: Restart all services
docker-restart: docker-down docker-up

## migrate: Run database migrations
migrate:
	@echo "$(GREEN)Running database migrations...$(NC)"
	@echo "$(YELLOW)Migrations are automatically handled by the application$(NC)"

## db-shell: Connect to PostgreSQL database
db-shell:
	@echo "$(GREEN)Connecting to database...$(NC)"
	docker-compose exec postgres psql -U postgres -d postanalyzer

## benchmark: Run benchmarks
benchmark:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	$(GO) test -bench=. -benchmem ./...

## security: Run security checks
security:
	@echo "$(GREEN)Running security checks...$(NC)"
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest$(NC)"; \
	fi

## deps-update: Update dependencies
deps-update:
	@echo "$(GREEN)Updating dependencies...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy

## check: Run all checks (lint, test, security)
check: lint test security
	@echo "$(GREEN)All checks passed!$(NC)"

## prod-build: Build for production
prod-build:
	@echo "$(GREEN)Building for production...$(NC)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY) $(MAIN_FILE)
	@echo "$(GREEN)Production build complete$(NC)"

## init: Initialize development environment
init: install install-tools
	@echo "$(GREEN)Creating .env file from example...$(NC)"
	@if [ ! -f .env ]; then cp .env.example .env 2>/dev/null || echo "$(YELLOW)No .env.example found$(NC)"; fi
	@echo "$(GREEN)Development environment ready!$(NC)"

## version: Display version information
version:
	@echo "$(BLUE)Post Analyzer Webserver$(NC)"
	@echo "Go version: $$($(GO) version)"
	@echo "Git commit: $$(git rev-parse --short HEAD 2>/dev/null || echo 'N/A')"

.DEFAULT_GOAL := help
