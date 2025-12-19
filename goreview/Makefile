.PHONY: all build test lint clean run

# Variables
BINARY_NAME=goreview
BUILD_DIR=build
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: lint test build

# Build binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/goreview

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Run locally
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Generate coverage report
coverage: test
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Install locally
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Development mode with hot reload (requires air)
dev:
	@which air > /dev/null || go install github.com/air-verse/air@latest
	air
