.PHONY: help test test-cover test-race lint lint-fix build clean fmt vet install-tools

# Default target
help:
	@echo "Available targets:"
	@echo "  make test          - Run tests"
	@echo "  make test-cover    - Run tests with coverage"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make lint          - Run linter"
	@echo "  make lint-fix      - Run linter and fix issues"
	@echo "  make build         - Build all packages"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make fmt           - Format code"
	@echo "  make vet           - Run go vet"
	@echo "  make install-tools - Install required tools"

# Run tests
test:
	@echo "Running tests..."
	@go test -v $(shell go list ./... | grep -v /examples)

# Run tests with coverage
test-cover:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v /examples)
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@go test -v -race $(shell go list ./... | grep -v /examples)

# Run linter
lint:
	@echo "Running linter..."
	@GOPATH=$$(go env GOPATH); \
	if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	elif [ -f "$$GOPATH/bin/golangci-lint" ]; then \
		$$GOPATH/bin/golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install with: make install-tools"; \
		exit 1; \
	fi

# Run linter and fix issues
lint-fix:
	@echo "Running linter and fixing issues..."
	@GOPATH=$$(go env GOPATH); \
	if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix --timeout=5m; \
	elif [ -f "$$GOPATH/bin/golangci-lint" ]; then \
		$$GOPATH/bin/golangci-lint run --fix --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install with: make install-tools"; \
		exit 1; \
	fi

# Build all packages
build:
	@echo "Building packages..."
	@go build ./...

# Build examples
build-examples:
	@echo "Building examples..."
	@go build ./examples/simple
	@go build ./examples/stdio-server

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -f coverage.out coverage.html
	@go clean ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix; \
	else \
		echo "golangci-lint not installed, skipping auto-fix (install with: make install-tools)"; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	@go vet ./...

# Install required tools
install-tools:
	@echo "Installing required tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed successfully"

# CI target (runs all checks)
ci: fmt vet lint test-race
	@echo "All CI checks passed!"

