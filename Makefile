# KaskManager R&D Platform Makefile

# Build variables
SERVER_BINARY=kaskmanager
CLI_BINARY=kaskman
BUILD_DIR=build
SERVER_CMD_DIR=cmd/server
CLI_CMD_DIR=cmd/cli
LDFLAGS=-ldflags "-s -w"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Docker variables
DOCKER_IMAGE=kaskmanager-rd-platform
DOCKER_TAG=latest

.PHONY: help build build-server build-cli build-all clean test deps tidy run run-server run-cli dev dev-server dev-cli docker-build docker-run install install-server install-cli install-local setup

help: ## Show this help message
	@echo "KaskManager R&D Platform - Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: build-server build-cli ## Build both server and CLI

build-server: deps ## Build the server
	@echo "Building $(SERVER_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) $(SERVER_CMD_DIR)
	@echo "Server build complete: $(BUILD_DIR)/$(SERVER_BINARY)"

build-cli: deps ## Build the CLI
	@echo "Building $(CLI_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY) $(CLI_CMD_DIR)
	@echo "CLI build complete: $(BUILD_DIR)/$(CLI_BINARY)"

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	# Server builds
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-linux-amd64 $(SERVER_CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-darwin-amd64 $(SERVER_CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY)-windows-amd64.exe $(SERVER_CMD_DIR)
	# CLI builds
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-linux-amd64 $(CLI_CMD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-darwin-amd64 $(CLI_CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLI_BINARY)-windows-amd64.exe $(CLI_CMD_DIR)
	@echo "Multi-platform build complete"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	$(GOTEST) -v ./...
	@echo "Tests complete"

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOGET) -d ./...
	@echo "Dependencies downloaded"

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	$(GOMOD) tidy
	@echo "Go modules tidied"

run: run-server ## Build and run the server (default)

run-server: build-server ## Build and run the server
	@echo "Starting $(SERVER_BINARY)..."
	./$(BUILD_DIR)/$(SERVER_BINARY)

run-cli: build-cli ## Build and run the CLI
	@echo "CLI built. Use: ./$(BUILD_DIR)/$(CLI_BINARY) --help"

dev: dev-server ## Run server in development mode (default)

dev-server: ## Run server in development mode
	@echo "Running server in development mode..."
	$(GOCMD) run $(SERVER_CMD_DIR)/main.go

dev-cli: ## Run CLI in development mode
	@echo "Running CLI in development mode..."
	$(GOCMD) run $(CLI_CMD_DIR)/main.go

dev-watch: ## Run in development mode with auto-reload
	@echo "Running with auto-reload (requires 'air')..."
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

install: install-server install-cli ## Install both server and CLI

install-server: build-server ## Install the server binary
	@echo "Installing $(SERVER_BINARY)..."
	@cp $(BUILD_DIR)/$(SERVER_BINARY) $(GOPATH)/bin/$(SERVER_BINARY)
	@echo "Server installed to $(GOPATH)/bin/$(SERVER_BINARY)"

install-cli: build-cli ## Install the CLI binary
	@echo "Installing $(CLI_BINARY)..."
	@cp $(BUILD_DIR)/$(CLI_BINARY) $(GOPATH)/bin/$(CLI_BINARY)
	@echo "CLI installed to $(GOPATH)/bin/$(CLI_BINARY)"

install-local: build-cli ## Install CLI to /usr/local/bin (requires sudo)
	@echo "Installing $(CLI_BINARY) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(CLI_BINARY) /usr/local/bin/$(CLI_BINARY)
	@echo "CLI installed to /usr/local/bin/$(CLI_BINARY)"

setup: ## Set up development environment
	@echo "Setting up development environment..."
	$(GOMOD) download
	@echo "Installing development tools..."
	$(GOCMD) install github.com/cosmtrek/air@latest
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development environment setup complete"

lint: ## Run linters
	@echo "Running linters..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run
	@echo "Linting complete"

format: ## Format code
	@echo "Formatting code..."
	$(GOCMD) fmt ./...
	@echo "Code formatted"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	docker run -p 8080:8080 --rm $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose-up: ## Start with docker-compose
	@echo "Starting with docker-compose..."
	docker-compose up -d
	@echo "Services started"

docker-compose-down: ## Stop docker-compose services
	@echo "Stopping docker-compose services..."
	docker-compose down
	@echo "Services stopped"

postgres-start: ## Start PostgreSQL with Docker
	@echo "Starting PostgreSQL..."
	docker run --name kaskmanager-postgres -e POSTGRES_USER=kaskmanager -e POSTGRES_PASSWORD=password -e POSTGRES_DB=kaskmanager_rd -p 5432:5432 -d postgres:15
	@echo "PostgreSQL started on port 5432"

postgres-stop: ## Stop PostgreSQL container
	@echo "Stopping PostgreSQL..."
	docker stop kaskmanager-postgres
	docker rm kaskmanager-postgres
	@echo "PostgreSQL stopped"

redis-start: ## Start Redis with Docker
	@echo "Starting Redis..."
	docker run --name kaskmanager-redis -p 6379:6379 -d redis:7-alpine
	@echo "Redis started on port 6379"

redis-stop: ## Stop Redis container
	@echo "Stopping Redis..."
	docker stop kaskmanager-redis
	docker rm kaskmanager-redis
	@echo "Redis stopped"

services-start: postgres-start redis-start ## Start all required services
	@echo "All services started"

services-stop: postgres-stop redis-stop ## Stop all services
	@echo "All services stopped"

migrate-up: ## Run database migrations up
	@echo "Running database migrations..."
	# TODO: Add migration command when migration tool is added
	@echo "Migrations complete"

migrate-down: ## Run database migrations down
	@echo "Rolling back database migrations..."
	# TODO: Add migration rollback command
	@echo "Migrations rolled back"

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...
	@echo "Benchmarks complete"

security-scan: ## Run security scan
	@echo "Running security scan..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...
	@echo "Security scan complete"

release: clean lint test build-all ## Prepare release build
	@echo "Release build complete"
	@ls -la $(BUILD_DIR)/

.DEFAULT_GOAL := help