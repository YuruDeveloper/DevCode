# Makefile for ollamacode Go project

.PHONY: build clean test run format lint install help

# Default target
All: Build

# Build the project
build:
	@echo "Building..."
	go build -o build/UniCode ./src

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf build/
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

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
	@echo "  Build   - Build the project"
	@echo "  Clean   - Clean build artifacts"
	@echo "  Test    - Run tests"
	@echo "  Run     - Run the application"
	@echo "  Format  - Format source code"
	@echo "  Lint    - Lint source code"
	@echo "  Install - Install dependencies"
	@echo "  Help    - Show this help message"