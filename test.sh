#!/bin/bash

# Test runner script for the drawing board project

set -e

echo "🧪 Running Drawing Board Tests"
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to run tests for a specific package
run_package_tests() {
    local package=$1
    local description=$2
    
    echo -e "\n${YELLOW}Testing $description...${NC}"
    echo "Package: $package"
    echo "----------------------------------------"
    
    if go test -v "$package"; then
        echo -e "${GREEN}✅ $description tests passed${NC}"
        return 0
    else
        echo -e "${RED}❌ $description tests failed${NC}"
        return 1
    fi
}

# Function to run tests with coverage
run_coverage_tests() {
    local package=$1
    local description=$2
    
    echo -e "\n${YELLOW}Testing $description with coverage...${NC}"
    echo "Package: $package"
    echo "----------------------------------------"
    
    if go test -v -cover "$package"; then
        echo -e "${GREEN}✅ $description coverage tests passed${NC}"
        return 0
    else
        echo -e "${RED}❌ $description coverage tests failed${NC}"
        return 1
    fi
}

# Function to run all tests
run_all_tests() {
    echo -e "\n${YELLOW}Running all tests...${NC}"
    echo "----------------------------------------"
    
    if go test -v ./...; then
        echo -e "${GREEN}✅ All tests passed${NC}"
        return 0
    else
        echo -e "${RED}❌ Some tests failed${NC}"
        return 1
    fi
}

# Function to run tests with coverage for all packages
run_all_coverage() {
    echo -e "\n${YELLOW}Running all tests with coverage...${NC}"
    echo "----------------------------------------"
    
    if go test -v -cover ./...; then
        echo -e "${GREEN}✅ All coverage tests passed${NC}"
        return 0
    else
        echo -e "${RED}❌ Some coverage tests failed${NC}"
        return 1
    fi
}

# Function to generate coverage report
generate_coverage_report() {
    echo -e "\n${YELLOW}Generating coverage report...${NC}"
    echo "----------------------------------------"
    
    # Create coverage directory
    mkdir -p coverage
    
    # Run tests with coverage
    go test -coverprofile=coverage/coverage.out ./...
    
    # Generate HTML report
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html
    
    # Generate text report
    go tool cover -func=coverage/coverage.out > coverage/coverage.txt
    
    echo -e "${GREEN}✅ Coverage report generated in coverage/ directory${NC}"
    echo "HTML report: coverage/coverage.html"
    echo "Text report: coverage/coverage.txt"
}

# Function to run benchmarks
run_benchmarks() {
    echo -e "\n${YELLOW}Running benchmarks...${NC}"
    echo "----------------------------------------"
    
    if go test -bench=. ./...; then
        echo -e "${GREEN}✅ Benchmarks completed${NC}"
        return 0
    else
        echo -e "${RED}❌ Benchmarks failed${NC}"
        return 1
    fi
}

# Function to run race detection
run_race_tests() {
    echo -e "\n${YELLOW}Running race detection tests...${NC}"
    echo "----------------------------------------"
    
    if go test -race ./...; then
        echo -e "${GREEN}✅ No race conditions detected${NC}"
        return 0
    else
        echo -e "${RED}❌ Race conditions detected${NC}"
        return 1
    fi
}

# Main execution
main() {
    local command=${1:-"all"}
    
    case $command in
        "db")
            run_package_tests "./internal/db" "Database Layer"
            ;;
        "auth")
            run_package_tests "./internal/auth" "Authentication"
            ;;
        "recognize")
            run_package_tests "./internal/recognize" "Recognition System"
            ;;
        "httpapi")
            run_package_tests "./internal/httpapi" "HTTP API"
            ;;
        "ws")
            run_package_tests "./internal/ws" "WebSocket Handler"
            ;;
        "coverage")
            run_all_coverage
            ;;
        "report")
            generate_coverage_report
            ;;
        "bench")
            run_benchmarks
            ;;
        "race")
            run_race_tests
            ;;
        "all")
            run_all_tests
            ;;
        "help"|"-h"|"--help")
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  all       Run all tests (default)"
            echo "  db        Test database layer"
            echo "  auth      Test authentication"
            echo "  recognize Test recognition system"
            echo "  httpapi   Test HTTP API"
            echo "  ws        Test WebSocket handler"
            echo "  coverage  Run all tests with coverage"
            echo "  report    Generate coverage report"
            echo "  bench     Run benchmarks"
            echo "  race      Run race detection tests"
            echo "  help      Show this help message"
            ;;
        *)
            echo -e "${RED}Unknown command: $command${NC}"
            echo "Use '$0 help' for available commands"
            exit 1
            ;;
    esac
}

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}Error: go.mod not found. Please run this script from the project root.${NC}"
    exit 1
fi

# Run main function
main "$@"
