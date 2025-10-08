#!/bin/bash

# Test runner script for RogueLearn CodeBattle project
# This script runs all unit tests with coverage and generates reports

set -e

echo "🚀 Starting RogueLearn CodeBattle Test Suite"
echo "============================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Change to project directory
cd "$(dirname "$0")"

# Clean test cache
print_status "Cleaning test cache..."
go clean -testcache

# Create test results directory
mkdir -p test_results

# Run tests with various options
print_status "Running unit tests..."

# Basic test run
echo ""
echo "📋 Running all tests..."
if go test -v ./... 2>&1 | tee test_results/test_output.log; then
    print_success "All tests passed!"
else
    print_error "Some tests failed. Check test_results/test_output.log for details."
fi

echo ""
echo "📊 Running tests with coverage..."
if go test -v -coverprofile=test_results/coverage.out ./... 2>&1 | tee test_results/coverage_output.log; then
    print_success "Coverage tests completed!"

    # Generate coverage report
    print_status "Generating coverage report..."
    go tool cover -html=test_results/coverage.out -o test_results/coverage.html

    # Show coverage summary
    echo ""
    echo "📈 Coverage Summary:"
    go tool cover -func=test_results/coverage.out | grep total:

    print_success "Coverage report generated at test_results/coverage.html"
else
    print_error "Coverage tests failed. Check test_results/coverage_output.log for details."
fi

# Run race condition tests
echo ""
echo "🏁 Running race condition tests..."
if go test -race ./... 2>&1 | tee test_results/race_output.log; then
    print_success "Race condition tests passed!"
else
    print_warning "Race condition tests failed. Check test_results/race_output.log for details."
fi

# Run benchmarks
echo ""
echo "⚡ Running benchmarks..."
if go test -bench=. -benchmem ./... 2>&1 | tee test_results/benchmark_output.log; then
    print_success "Benchmarks completed!"
else
    print_warning "Some benchmarks failed. Check test_results/benchmark_output.log for details."
fi

# Run specific package tests
echo ""
echo "📦 Running package-specific tests..."

packages=(
    "./pkg/jwt"
    "./pkg/request"
    "./pkg/response"
    "./pkg/env"
    "./internal/events"
    "./internal/service"
    "./internal/handler"
    "./internal/hub"
)

for package in "${packages[@]}"; do
    if [ -d "$package" ]; then
        print_status "Testing package: $package"
        if go test -v "$package" 2>&1 | tee "test_results/$(basename $package)_test.log"; then
            print_success "✅ $package tests passed"
        else
            print_error "❌ $package tests failed"
        fi
    else
        print_warning "Package $package not found, skipping..."
    fi
done

# Generate test summary
echo ""
echo "📋 Test Summary"
echo "==============="

total_tests=$(grep -c "=== RUN" test_results/test_output.log || echo "0")
passed_tests=$(grep -c "--- PASS:" test_results/test_output.log || echo "0")
failed_tests=$(grep -c "--- FAIL:" test_results/test_output.log || echo "0")

echo "Total tests run: $total_tests"
echo "Tests passed: $passed_tests"
echo "Tests failed: $failed_tests"

if [ "$failed_tests" -eq 0 ]; then
    print_success "🎉 All tests completed successfully!"
    echo ""
    echo "📁 Test artifacts generated in test_results/:"
    echo "  - test_output.log: Complete test output"
    echo "  - coverage.out: Coverage data"
    echo "  - coverage.html: HTML coverage report"
    echo "  - benchmark_output.log: Benchmark results"
    echo "  - race_output.log: Race condition test results"
    exit 0
else
    print_error "⚠️  Some tests failed. Please review the test output."
    exit 1
fi
