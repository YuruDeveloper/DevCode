# Makefile for ollamacode Go project

.PHONY: build clean test test-unit test-service test-integration test-all run format lint install help

# Default target
All: build

# Build the project
build:
	@echo "Building..."
	go build -o build/UniCode ./src

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf build/
	go clean

# Run all tests (unit tests only, excluding integration)
test:
	@echo "Running unit tests from src directory..."
	go test -v ./...
	@echo "Running unit tests from tests directory..."
	cd tests && go test -v -run "^Test.*[^Integration]$$" ./...

# Run only unit tests (src directory)
test-unit:
	@echo "Running unit tests from src directory..."
	go test -v ./...

# Run only service unit tests (tests directory, excluding integration)
test-service:
	@echo "Running service unit tests..."
	cd tests && go test -v -run "^Test.*[^Integration]$$" ./...

# Run only integration tests (tests directory)  
test-integration:
	@echo "Running integration tests..."
	cd tests && go test -v -run ".*Integration" ./...

# Run all tests including integration tests
test-all:
	@echo "Running all unit tests..."
	go test -v ./...
	@echo "Running all tests in tests directory..."
	cd tests && go test -v ./...

# Run the application
run:
	@echo "Running application..."
	go run ./...

# Format code
format:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	go vet ./...

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Show help
help:
	@echo "Available targets:"
	@echo "  build            - Build the project"
	@echo "  clean            - Clean build artifacts"
	@echo "  test             - Run unit tests only (excluding integration)"
	@echo "  test-unit        - Run only unit tests (src directory)"
	@echo "  test-service     - Run only service unit tests (tests directory, no integration)"
	@echo "  test-integration - Run only integration tests (tests directory)"
	@echo "  test-all         - Run all tests including integration tests"
	@echo "  run              - Run the application"
	@echo "  format           - Format source code"
	@echo "  lint             - Lint source code"
	@echo "  install          - Install dependencies"
	@echo "  help             - Show this help message"