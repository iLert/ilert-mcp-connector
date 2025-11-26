.PHONY: help build run test clean fmt vet lint docker-build docker-run docker-push deps tidy version

# Variables
BINARY_NAME=ilert-mcp-connector
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DOCKER_IMAGE?=ghcr.io/ilert/ilert-mcp-connector
DOCKER_TAG?=latest
PORT?=8383

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: $(BINARY_NAME)"

run: ## Run the application locally
	@echo "Starting $(BINARY_NAME)..."
	@go run $(LDFLAGS) .

start: build ## Build and start the server
	@echo "Starting $(BINARY_NAME)..."
	@./$(BINARY_NAME)

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race ./...

test-short: ## Run tests without race detector
	@echo "Running tests (short)..."
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Code formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

lint: ## Run golangci-lint (if installed)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running golangci-lint..."; \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install it from https://golangci-lint.run/"; \
	fi

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@go mod tidy

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):$(VERSION) \
		.
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -p $(PORT):8383 \
		-e PORT=8383 \
		$(DOCKER_IMAGE):$(DOCKER_TAG)

docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@docker push $(DOCKER_IMAGE):$(VERSION)

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Binary: $(BINARY_NAME)"

install: build ## Build and install binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .
	@echo "Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)"

dev: ## Run in development mode with hot reload (requires air)
	@if command -v air >/dev/null 2>&1; then \
		echo "Running with air (hot reload)..."; \
		air; \
	else \
		echo "air not installed. Install it: go install github.com/cosmtrek/air@latest"; \
		echo "Falling back to regular run..."; \
		make run; \
	fi

check: fmt vet test ## Run all checks (fmt, vet, test)

ci: deps tidy fmt vet test build ## Run CI pipeline locally

