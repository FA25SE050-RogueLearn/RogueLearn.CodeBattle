#!/bin/bash

# Simple Test Runner for RogueLearn CodeBattle Working Packages
# This script runs tests only for packages that are currently working

set -e

echo "🚀 Starting RogueLearn CodeBattle Simple Test Suite"
echo "=================================================="

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

# Working packages (no complex mocking required)
working_packages=(
    "./pkg/jwt"
    "./pkg/request"
    "./pkg/response"
    "./pkg/env"
    "./internal/events"
)

echo ""
echo "📦 Running tests for working packages..."

total_passed=0
total_failed=0

for package in "${working_packages[@]}"; do
    if [ -d "$package" ]; then
        print_status "Testing package: $package"
        echo ""

        if go test -v "$package" 2>&1 | tee "test_results/$(basename $package)_test.log"; then
            print_success "✅ $package tests passed"
            ((total_passed++))
        else
            print_error "❌ $package tests failed"
            ((total_failed++))
        fi
        echo ""
    else
        print_warning "Package $package not found, skipping..."
    fi
done

# Run coverage for working packages
echo ""
echo "📊 Running coverage for working packages..."
if go test -coverprofile=test_results/coverage.out "${working_packages[@]}" 2>&1 | tee test_results/coverage_output.log; then
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

# Run benchmarks for working packages
echo ""
echo "⚡ Running benchmarks for working packages..."
if go test -bench=. -benchmem "${working_packages[@]}" 2>&1 | tee test_results/benchmark_output.log; then
    print_success "Benchmarks completed!"
else
    print_warning "Some benchmarks failed. Check test_results/benchmark_output.log for details."
fi

# Generate test summary
echo ""
echo "📋 Test Summary"
echo "==============="
echo "Working packages tested: ${#working_packages[@]}"
echo "Packages passed: $total_passed"
echo "Packages failed: $total_failed"

# Note about complex packages
echo ""
echo "📝 Note: Complex packages requiring database mocks are currently excluded:"
echo "  - ./internal/service (requires database mocking)"
echo "  - ./internal/handler (requires SSE and hub mocking)"
echo "  - ./internal/hub (requires database and worker pool mocking)"
echo ""
echo "These packages contain business logic that would benefit from integration tests"
echo "with actual database connections or more sophisticated mocking frameworks."

if [ "$total_failed" -eq 0 ]; then
    print_success "🎉 All working package tests completed successfully!"
    echo ""
    echo "📁 Test artifacts generated in test_results/:"
    echo "  - coverage.out: Coverage data"
    echo "  - coverage.html: HTML coverage report"
    echo "  - benchmark_output.log: Benchmark results"
    echo "  - *_test.log: Individual package test outputs"

    # Open coverage report if running on desktop
    if command -v xdg-open &> /dev/null; then
        print_status "Opening coverage report in browser..."
        xdg-open test_results/coverage.html 2>/dev/null &
    elif command -v open &> /dev/null; then
        print_status "Opening coverage report in browser..."
        open test_results/coverage.html 2>/dev/null &
    fi

    exit 0
else
    print_error "⚠️  Some package tests failed. Please review the test output."
    exit 1
fi
