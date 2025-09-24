#!/bin/bash
# Comprehensive test runner for MarchProxy dual proxy architecture
# This script runs unit tests, integration tests, and builds all components

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
VERBOSE=${VERBOSE:-false}
INTEGRATION=${INTEGRATION:-false}
COVERAGE=${COVERAGE:-false}
BENCHMARK=${BENCHMARK:-false}
BUILD_CHECK=${BUILD_CHECK:-true}
LOAD_TEST=${LOAD_TEST:-false}
SECURITY_TEST=${SECURITY_TEST:-false}

print_header() {
    echo -e "${BLUE}================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Check if we're in the right directory
check_project_root() {
    if [[ ! -f "docker-compose.yml" ]] || [[ ! -d "manager" ]] || [[ ! -d "proxy-egress" ]]; then
        print_error "Please run this script from the MarchProxy root directory"
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    print_header "Checking Dependencies"

    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi
    print_success "Go $(go version | cut -d' ' -f3) found"

    # Check Python
    if ! command -v python3 &> /dev/null; then
        print_error "Python 3 is not installed or not in PATH"
        exit 1
    fi
    print_success "Python $(python3 --version | cut -d' ' -f2) found"

    # Check Docker (for integration tests)
    if [[ "$INTEGRATION" == "true" ]]; then
        if ! command -v docker &> /dev/null; then
            print_warning "Docker not found - integration tests will be skipped"
            INTEGRATION=false
        else
            print_success "Docker $(docker --version | cut -d' ' -f3 | tr -d ',') found"
        fi

        if ! command -v docker-compose &> /dev/null; then
            print_warning "Docker Compose not found - integration tests will be skipped"
            INTEGRATION=false
        else
            print_success "Docker Compose $(docker-compose --version | cut -d' ' -f3 | tr -d ',') found"
        fi
    fi
}

# Run manager tests
run_manager_tests() {
    print_header "Running Manager Tests (Python)"

    cd manager

    # Check if virtual environment exists
    if [[ ! -d "venv" ]]; then
        print_info "Creating Python virtual environment..."
        python3 -m venv venv
    fi

    # Activate virtual environment
    source venv/bin/activate

    # Install dependencies
    if [[ -f "requirements.txt" ]]; then
        print_info "Installing Python dependencies..."
        pip install -q -r requirements.txt
    fi

    # Install test dependencies
    pip install -q pytest pytest-cov pytest-mock

    # Run tests
    if [[ "$COVERAGE" == "true" ]]; then
        print_info "Running manager tests with coverage..."
        if [[ "$VERBOSE" == "true" ]]; then
            pytest tests/ -v --cov=. --cov-report=term-missing --cov-report=html:../coverage/manager
        else
            pytest tests/ --cov=. --cov-report=term-missing --cov-report=html:../coverage/manager
        fi
    else
        print_info "Running manager tests..."
        if [[ "$VERBOSE" == "true" ]]; then
            pytest tests/ -v
        else
            pytest tests/
        fi
    fi

    print_success "Manager tests completed"

    deactivate
    cd ..
}

# Run proxy-egress tests
run_proxy_egress_tests() {
    print_header "Running Proxy-Egress Tests (Go)"

    cd proxy-egress

    # Download dependencies
    print_info "Downloading Go dependencies..."
    go mod download
    go mod tidy

    # Run tests
    if [[ "$COVERAGE" == "true" ]]; then
        print_info "Running proxy-egress tests with coverage..."
        mkdir -p ../coverage/proxy-egress
        if [[ "$VERBOSE" == "true" ]]; then
            go test -v -race -coverprofile=../coverage/proxy-egress/coverage.out ./...
            go tool cover -html=../coverage/proxy-egress/coverage.out -o ../coverage/proxy-egress/coverage.html
        else
            go test -race -coverprofile=../coverage/proxy-egress/coverage.out ./...
            go tool cover -html=../coverage/proxy-egress/coverage.out -o ../coverage/proxy-egress/coverage.html
        fi
    else
        print_info "Running proxy-egress tests..."
        if [[ "$VERBOSE" == "true" ]]; then
            go test -v -race ./...
        else
            go test -race ./...
        fi
    fi

    # Run benchmarks if requested
    if [[ "$BENCHMARK" == "true" ]]; then
        print_info "Running proxy-egress benchmarks..."
        go test -bench=. -benchmem ./...
    fi

    print_success "Proxy-egress tests completed"

    cd ..
}

# Run proxy-ingress tests
run_proxy_ingress_tests() {
    print_header "Running Proxy-Ingress Tests (Go)"

    cd proxy-ingress

    # Download dependencies
    print_info "Downloading Go dependencies..."
    go mod download
    go mod tidy

    # Run tests
    if [[ "$COVERAGE" == "true" ]]; then
        print_info "Running proxy-ingress tests with coverage..."
        mkdir -p ../coverage/proxy-ingress
        if [[ "$VERBOSE" == "true" ]]; then
            go test -v -race -coverprofile=../coverage/proxy-ingress/coverage.out ./...
            go tool cover -html=../coverage/proxy-ingress/coverage.out -o ../coverage/proxy-ingress/coverage.html
        else
            go test -race -coverprofile=../coverage/proxy-ingress/coverage.out ./...
            go tool cover -html=../coverage/proxy-ingress/coverage.out -o ../coverage/proxy-ingress/coverage.html
        fi
    else
        print_info "Running proxy-ingress tests..."
        if [[ "$VERBOSE" == "true" ]]; then
            go test -v -race ./...
        else
            go test -race ./...
        fi
    fi

    # Run benchmarks if requested
    if [[ "$BENCHMARK" == "true" ]]; then
        print_info "Running proxy-ingress benchmarks..."
        go test -bench=. -benchmem ./...
    fi

    print_success "Proxy-ingress tests completed"

    cd ..
}

# Test builds
test_builds() {
    if [[ "$BUILD_CHECK" != "true" ]]; then
        return
    fi

    print_header "Testing Component Builds"

    # Test manager build
    print_info "Testing manager startup..."
    cd manager
    if [[ -d "venv" ]]; then
        source venv/bin/activate
        # Test that the manager can import without errors
        python3 -c "
import sys
sys.path.append('.')
try:
    from apps.manager import main
    print('Manager imports successfully')
except Exception as e:
    print(f'Manager import failed: {e}')
    sys.exit(1)
" || {
        print_error "Manager build test failed"
        exit 1
    }
        deactivate
    fi
    cd ..
    print_success "Manager build test passed"

    # Test proxy-egress build
    print_info "Testing proxy-egress build..."
    cd proxy-egress
    if ! go build -o ../test-artifacts/proxy-egress ./cmd/proxy; then
        print_error "Proxy-egress build failed"
        exit 1
    fi
    cd ..
    print_success "Proxy-egress build test passed"

    # Test proxy-ingress build
    print_info "Testing proxy-ingress build..."
    cd proxy-ingress
    if ! go build -o ../test-artifacts/proxy-ingress ./cmd/proxy; then
        print_error "Proxy-ingress build failed"
        exit 1
    fi
    cd ..
    print_success "Proxy-ingress build test passed"

    # Test version outputs
    print_info "Testing version outputs..."
    if [[ -f "test-artifacts/proxy-egress" ]]; then
        ./test-artifacts/proxy-egress --version || print_warning "Proxy-egress version check failed"
    fi
    if [[ -f "test-artifacts/proxy-ingress" ]]; then
        ./test-artifacts/proxy-ingress --version || print_warning "Proxy-ingress version check failed"
    fi
}

# Test Docker builds
test_docker_builds() {
    if [[ "$INTEGRATION" != "true" ]]; then
        return
    fi

    print_header "Testing Docker Builds"

    # Test manager Docker build
    print_info "Testing manager Docker build..."
    if ! docker build -f manager/Dockerfile -t marchproxy-manager:test manager/; then
        print_error "Manager Docker build failed"
        exit 1
    fi
    print_success "Manager Docker build passed"

    # Test proxy-egress Docker build
    print_info "Testing proxy-egress Docker build..."
    if ! docker build -f proxy-egress/Dockerfile -t marchproxy-proxy-egress:test proxy-egress/; then
        print_error "Proxy-egress Docker build failed"
        exit 1
    fi
    print_success "Proxy-egress Docker build passed"

    # Test proxy-ingress Docker build
    print_info "Testing proxy-ingress Docker build..."
    if ! docker build -f proxy-ingress/Dockerfile -t marchproxy-proxy-ingress:test proxy-ingress/; then
        print_error "Proxy-ingress Docker build failed"
        exit 1
    fi
    print_success "Proxy-ingress Docker build passed"
}

# Run integration tests
run_integration_tests() {
    if [[ "$INTEGRATION" != "true" ]]; then
        return
    fi

    print_header "Running Integration Tests"

    # Start services with docker-compose
    print_info "Starting test environment with docker-compose..."
    docker-compose -f docker-compose.test.yml up -d

    # Wait for services to be ready
    print_info "Waiting for services to be ready..."
    sleep 30

    # Run integration tests
    print_info "Running integration tests..."
    cd test
    go mod init test 2>/dev/null || true
    go mod tidy

    # Set integration test environment variable
    export INTEGRATION_TEST=true

    if [[ "$VERBOSE" == "true" ]]; then
        go test -v -timeout=5m ./...
    else
        go test -timeout=5m ./...
    fi

    cd ..

    # Clean up
    print_info "Cleaning up test environment..."
    docker-compose -f docker-compose.test.yml down

    print_success "Integration tests completed"
}

# Run load tests
run_load_tests() {
    if [[ "$LOAD_TEST" != "true" ]]; then
        return
    fi

    print_header "Running Load Tests"

    # Start services with docker-compose
    print_info "Starting test environment for load testing..."
    docker-compose -f docker-compose.test.yml up -d

    # Wait for services to be ready
    print_info "Waiting for services to be ready..."
    sleep 30

    # Run load tests
    print_info "Running load tests..."
    cd test
    go mod init test 2>/dev/null || true
    go mod tidy

    # Set load test environment variable
    export LOAD_TEST=true

    if [[ "$VERBOSE" == "true" ]]; then
        go test -v -timeout=10m -run="TestManager.*Load|TestProxy.*Load|TestDualProxy.*|TestConcurrent.*" ./...
    else
        go test -timeout=10m -run="TestManager.*Load|TestProxy.*Load|TestDualProxy.*|TestConcurrent.*" ./...
    fi

    cd ..

    # Clean up
    print_info "Cleaning up test environment..."
    docker-compose -f docker-compose.test.yml down

    print_success "Load tests completed"
}

# Run security tests
run_security_tests() {
    if [[ "$SECURITY_TEST" != "true" ]]; then
        return
    fi

    print_header "Running Security Tests"

    # Start services with docker-compose
    print_info "Starting test environment for security testing..."
    docker-compose -f docker-compose.test.yml up -d

    # Wait for services to be ready
    print_info "Waiting for services to be ready..."
    sleep 30

    # Run security tests
    print_info "Running security tests..."
    cd test
    go mod init test 2>/dev/null || true
    go mod tidy

    # Set security test environment variable
    export SECURITY_TEST=true

    if [[ "$VERBOSE" == "true" ]]; then
        go test -v -timeout=10m -run="Test.*Security|TestAuthentication.*|TestInput.*|TestTLS.*|TestCORS.*|TestDOS.*" ./...
    else
        go test -timeout=10m -run="Test.*Security|TestAuthentication.*|TestInput.*|TestTLS.*|TestCORS.*|TestDOS.*" ./...
    fi

    cd ..

    # Clean up
    print_info "Cleaning up test environment..."
    docker-compose -f docker-compose.test.yml down

    print_success "Security tests completed"
}

# Generate test report
generate_test_report() {
    print_header "Generating Test Report"

    mkdir -p test-results

    cat > test-results/test-summary.md << EOF
# MarchProxy Test Summary

**Test Run Date:** $(date)

## Test Configuration
- Verbose: $VERBOSE
- Coverage: $COVERAGE
- Integration: $INTEGRATION
- Benchmark: $BENCHMARK
- Build Check: $BUILD_CHECK

## Components Tested
- ✓ Manager (Python)
- ✓ Proxy-Egress (Go)
- ✓ Proxy-Ingress (Go)

EOF

    if [[ "$COVERAGE" == "true" ]]; then
        echo "## Coverage Reports" >> test-results/test-summary.md
        echo "- Manager: coverage/manager/index.html" >> test-results/test-summary.md
        echo "- Proxy-Egress: coverage/proxy-egress/coverage.html" >> test-results/test-summary.md
        echo "- Proxy-Ingress: coverage/proxy-ingress/coverage.html" >> test-results/test-summary.md
        echo "" >> test-results/test-summary.md
    fi

    if [[ "$INTEGRATION" == "true" ]]; then
        echo "- ✓ Integration Tests" >> test-results/test-summary.md
    else
        echo "- ⚠ Integration Tests (skipped)" >> test-results/test-summary.md
    fi

    echo "" >> test-results/test-summary.md
    echo "**Test completed successfully!**" >> test-results/test-summary.md

    print_success "Test report generated: test-results/test-summary.md"
}

# Main execution
main() {
    print_header "MarchProxy Test Suite"

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -i|--integration)
                INTEGRATION=true
                shift
                ;;
            -c|--coverage)
                COVERAGE=true
                shift
                ;;
            -b|--benchmark)
                BENCHMARK=true
                shift
                ;;
            -l|--load)
                LOAD_TEST=true
                shift
                ;;
            -s|--security)
                SECURITY_TEST=true
                shift
                ;;
            --no-build)
                BUILD_CHECK=false
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  -v, --verbose     Enable verbose output"
                echo "  -i, --integration Run integration tests"
                echo "  -c, --coverage    Generate coverage reports"
                echo "  -b, --benchmark   Run benchmarks"
                echo "  -l, --load        Run load tests"
                echo "  -s, --security    Run security tests"
                echo "  --no-build        Skip build tests"
                echo "  -h, --help        Show this help message"
                echo ""
                echo "Environment variables:"
                echo "  VERBOSE=true      Same as --verbose"
                echo "  INTEGRATION=true  Same as --integration"
                echo "  COVERAGE=true     Same as --coverage"
                echo "  BENCHMARK=true    Same as --benchmark"
                echo "  LOAD_TEST=true    Same as --load"
                echo "  SECURITY_TEST=true Same as --security"
                echo "  BUILD_CHECK=false Same as --no-build"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Create directories
    mkdir -p test-artifacts
    mkdir -p test-results
    if [[ "$COVERAGE" == "true" ]]; then
        mkdir -p coverage
    fi

    # Run test suite
    check_project_root
    check_dependencies

    # Run unit tests
    run_manager_tests
    run_proxy_egress_tests
    run_proxy_ingress_tests

    # Run build tests
    test_builds
    test_docker_builds

    # Run integration tests
    run_integration_tests

    # Run load tests
    run_load_tests

    # Run security tests
    run_security_tests

    # Generate report
    generate_test_report

    print_header "All Tests Completed Successfully!"
    print_info "Run with --help for more options"
}

# Run main function
main "$@"