# LogpushEstimator Makefile

.PHONY: test test-verbose test-coverage test-unit test-integration clean build run

# Default target
all: test

# Run all tests
test:
	go test ./...

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run tests with coverage report
test-coverage:
	go test -v -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run only unit tests (exclude integration tests)
test-unit:
	go test -v ./src/database/
	go test -v ./src/gui/handlers/
	go test -v ./ -run "Test.*" -skip "TestFull.*|TestConcurrent.*|TestAPI.*Integration|TestErrorHandling"

# Run only integration tests
test-integration:
	go test -v ./ -run "TestFull.*|TestConcurrent.*|TestAPI.*Integration|TestErrorHandling"

# Run tests for a specific package
test-database:
	go test -v ./src/database/

test-handlers:
	go test -v ./src/gui/handlers/

test-main:
	go test -v ./ -run "Test.*" -skip "TestFull.*|TestConcurrent.*|TestAPI.*Integration|TestErrorHandling"

# Clean up test artifacts
clean:
	rm -f test_*.db
	rm -f coverage.out coverage.html
	rm -f logpush.db

# Build the application
build:
	go build -o LogpushEstimator .

# Run the application
run: build
	./LogpushEstimator

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Run benchmarks
bench:
	go test -bench=. ./...

# Check for race conditions
test-race:
	go test -race ./...

# Quick test (fast subset)
test-quick:
	go test ./src/database/ -short
	go test ./src/gui/handlers/ -short

# Generate test coverage badge/report
coverage-report: test-coverage
	@echo "Test coverage report available at: coverage.html"
	@echo "Open it in your browser to view detailed coverage information"

# Help target
help:
	@echo "Available targets:"
	@echo "  test           - Run all tests"
	@echo "  test-verbose   - Run tests with verbose output"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  test-unit      - Run only unit tests"
	@echo "  test-integration - Run only integration tests"
	@echo "  test-database  - Run database tests only"
	@echo "  test-handlers  - Run handler tests only"
	@echo "  test-main      - Run main package tests only"
	@echo "  test-race      - Run tests with race detection"
	@echo "  test-quick     - Run fast subset of tests"
	@echo "  build          - Build the application"
	@echo "  run            - Build and run the application"
	@echo "  clean          - Clean up test artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  fmt            - Format code"
	@echo "  lint           - Lint code (requires golangci-lint)"
	@echo "  bench          - Run benchmarks"
	@echo "  coverage-report - Generate detailed coverage report"
	@echo "  help           - Show this help message"