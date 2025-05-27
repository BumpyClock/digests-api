# Makefile for Digests API

.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: run
run: ## Run the API server
	@go run cmd/api/main.go

.PHONY: build
build: ## Build the API binary
	@echo "Building API binary..."
	@go build -o bin/api cmd/api/main.go
	@echo "Binary built at bin/api"

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	@go test -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

.PHONY: test-core
test-core: ## Run core package tests
	@echo "Running core tests..."
	@go test -v ./core/...

.PHONY: test-api
test-api: ## Run API tests
	@echo "Running API tests..."
	@go test -v ./api/...

.PHONY: test-infra
test-infra: ## Run infrastructure tests
	@echo "Running infrastructure tests..."
	@go test -v ./infrastructure/...

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ coverage.out coverage.html

.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

.PHONY: tidy
tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	@go mod tidy

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t digests-api:latest .

.PHONY: docker-run
docker-run: ## Run API in Docker
	@docker run -p 8080:8080 --env-file .env digests-api:latest

.PHONY: redis-start
redis-start: ## Start Redis for development
	@echo "Starting Redis..."
	@docker run -d --name digests-redis -p 6379:6379 redis:alpine

.PHONY: redis-stop
redis-stop: ## Stop Redis
	@echo "Stopping Redis..."
	@docker stop digests-redis
	@docker rm digests-redis