.PHONY: all build test test-coverage test-verbose clean lint fmt vet help install run dev

# Variables
BINARY_NAME=favicon-server
BUILD_DIR=bin
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

# Default target
all: fmt vet lint test build

# Help target
help:
	@echo "Favicon Fetcher - Makefile commands:"
	@echo ""
	@echo "  make build          - Build the binary (full format support)"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make bench          - Run benchmarks"
	@echo "  make lint           - Run linter (golangci-lint)"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install binary to GOPATH/bin"
	@echo "  make run            - Build and run server"
	@echo "  make dev            - Run in development mode"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"
	@echo ""

# Build target (full format support: PNG + WebP + AVIF)
build:
	@echo "Building $(BINARY_NAME) with full format support (PNG, WebP, AVIF)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) ./cmd/server

# Test targets
test:
	@echo "Running tests..."
	$(GO) test -race -short ./...

test-integration:
	@echo "Running integration tests..."
	$(GO) test -v -tags=integration ./tests

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-verbose:
	@echo "Running tests (verbose)..."
	$(GO) test -v -race ./...

test-all: test test-integration
	@echo "All tests passed!"

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...

# Code quality targets
lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install from https://golangci-lint.run/"; \
		exit 1; \
	fi

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if command -v gofumpt > /dev/null; then \
		gofumpt -l -w .; \
	fi

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

# Run targets
run: build
	@echo "Starting server..."
	$(BUILD_DIR)/$(BINARY_NAME) -log-level debug

dev:
	@echo "Running in development mode..."
	$(GO) run ./cmd/server -log-level debug -cache-dir ./cache-dev

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t favicon-fetcher:latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 9090:9090 -v $(PWD)/cache:/cache favicon-fetcher:latest

docker-compose-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d

docker-compose-down:
	@echo "Stopping services..."
	docker-compose down

docker-compose-logs:
	@echo "Showing logs..."
	docker-compose logs -f

# Clean target
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	rm -rf cache cache-dev
	$(GO) clean
	@echo "Clean complete"

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

# Generate go.sum if missing
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy

# Check for updates
check-updates:
	@echo "Checking for dependency updates..."
	$(GO) list -u -m all

# Security check
security:
	@echo "Running security checks..."
	@if command -v gosec > /dev/null; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# CI target (for CI/CD pipelines)
ci: deps fmt vet lint test

# Release build (optimized)
release:
	@echo "Building release version..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/server
	@echo "Release builds complete in $(BUILD_DIR)/"
