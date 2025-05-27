#!/bin/bash
# ABOUTME: Test runner script for the Digests API
# ABOUTME: Runs all tests with coverage and generates reports

set -e

echo "Running all tests for Digests API..."
echo "=================================="

# Run tests with coverage
echo "Running unit tests with coverage..."
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
echo -e "\n\nGenerating coverage report..."
go tool cover -html=coverage.out -o coverage.html

# Show coverage summary
echo -e "\n\nCoverage Summary:"
go tool cover -func=coverage.out | tail -n 1

# Run specific test suites
echo -e "\n\nRunning test suites by package:"
echo "------------------------------"

# Core tests
echo -e "\n[Core Domain Tests]"
go test -v ./core/domain/...

echo -e "\n[Core Services Tests]"
go test -v ./core/feed/... ./core/search/... ./core/share/...

echo -e "\n[Core Errors Tests]"
go test -v ./core/errors/...

# Infrastructure tests
echo -e "\n[Infrastructure Tests]"
go test -v ./infrastructure/...

# API tests
echo -e "\n[API Tests]"
go test -v ./api/...

# Integration tests (if Redis is available)
echo -e "\n[Integration Tests]"
if command -v redis-cli &> /dev/null && redis-cli ping &> /dev/null; then
    echo "Redis available, running integration tests..."
    go test -v -tags=integration ./infrastructure/cache/redis/...
else
    echo "Redis not available, skipping Redis integration tests"
fi

echo -e "\n\nAll tests completed!"
echo "===================="
echo "Coverage report available at: coverage.html"