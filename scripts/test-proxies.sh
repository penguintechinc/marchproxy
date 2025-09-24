#!/bin/bash
# Comprehensive test script for MarchProxy dual proxy architecture
# Tests both proxy-egress and proxy-ingress with mTLS authentication

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CERT_DIR="${CERT_DIR:-./certs}"
MANAGER_URL="${MANAGER_URL:-http://localhost:8000}"
EGRESS_URL="${EGRESS_URL:-http://localhost:8081}"
INGRESS_HTTP_URL="${INGRESS_HTTP_URL:-http://localhost:80}"
INGRESS_HTTPS_URL="${INGRESS_HTTPS_URL:-https://localhost:443}"
INGRESS_ADMIN_URL="${INGRESS_ADMIN_URL:-http://localhost:8082}"

# Test configuration
TEST_TIMEOUT=30
TEST_RESULTS=()
FAILED_TESTS=0
TOTAL_TESTS=0

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

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_result="$3"

    ((TOTAL_TESTS++))
    print_status "Running test: $test_name"

    if eval "$test_command"; then
        if [[ -n "$expected_result" ]]; then
            local result=$(eval "$test_command" 2>/dev/null)
            if echo "$result" | grep -q "$expected_result"; then
                print_success "✓ $test_name"
                TEST_RESULTS+=("PASS: $test_name")
            else
                print_error "✗ $test_name (unexpected result)"
                TEST_RESULTS+=("FAIL: $test_name - unexpected result")
                ((FAILED_TESTS++))
            fi
        else
            print_success "✓ $test_name"
            TEST_RESULTS+=("PASS: $test_name")
        fi
    else
        print_error "✗ $test_name"
        TEST_RESULTS+=("FAIL: $test_name")
        ((FAILED_TESTS++))
    fi
}

wait_for_service() {
    local service_url="$1"
    local service_name="$2"
    local timeout="${3:-60}"

    print_status "Waiting for $service_name to be ready ($timeout seconds timeout)..."

    for i in $(seq 1 $timeout); do
        if curl -sf "$service_url" > /dev/null 2>&1; then
            print_success "$service_name is ready"
            return 0
        fi
        sleep 1
        if [ $((i % 10)) -eq 0 ]; then
            print_status "Still waiting for $service_name... ($i/$timeout)"
        fi
    done

    print_error "$service_name did not become ready within $timeout seconds"
    return 1
}

test_certificate_generation() {
    print_status "Testing certificate generation..."

    if [ ! -d "$CERT_DIR" ]; then
        mkdir -p "$CERT_DIR"
    fi

    # Generate certificates if they don't exist
    if [ ! -f "$CERT_DIR/ca.pem" ]; then
        print_status "Generating test certificates..."
        if docker-compose --profile tools run --rm cert-generator; then
            print_success "Certificates generated successfully"
        else
            print_error "Certificate generation failed"
            return 1
        fi
    fi

    # Verify certificate files exist
    local cert_files=("ca.pem" "server-cert.pem" "server-key.pem" "client-cert.pem" "client-key.pem")
    for cert_file in "${cert_files[@]}"; do
        run_test "Certificate file exists: $cert_file" \
                 "test -f '$CERT_DIR/$cert_file'" ""
    done

    # Verify certificate validity
    run_test "CA certificate is valid" \
             "openssl x509 -in '$CERT_DIR/ca.pem' -text -noout" ""

    run_test "Server certificate is valid" \
             "openssl verify -CAfile '$CERT_DIR/ca.pem' '$CERT_DIR/server-cert.pem'" ""

    run_test "Client certificate is valid" \
             "openssl verify -CAfile '$CERT_DIR/ca.pem' '$CERT_DIR/client-cert.pem'" ""
}

test_docker_services() {
    print_status "Testing Docker services startup..."

    # Check if docker-compose is available
    run_test "Docker Compose is available" \
             "docker-compose --version" ""

    # Start core services
    print_status "Starting core services..."
    if docker-compose up -d postgres redis manager; then
        print_success "Core services started"
    else
        print_error "Failed to start core services"
        return 1
    fi

    # Wait for services to be ready
    wait_for_service "$MANAGER_URL/healthz" "Manager" 60

    # Start proxy services
    print_status "Starting proxy services..."
    if docker-compose up -d proxy-egress proxy-ingress; then
        print_success "Proxy services started"
    else
        print_error "Failed to start proxy services"
        return 1
    fi

    # Wait for proxy services
    wait_for_service "$EGRESS_URL/healthz" "Proxy Egress" 60
    wait_for_service "$INGRESS_ADMIN_URL/healthz" "Proxy Ingress" 60
}

test_manager_api() {
    print_status "Testing Manager API..."

    run_test "Manager health endpoint" \
             "curl -sf '$MANAGER_URL/healthz'" "healthy"

    run_test "Manager metrics endpoint" \
             "curl -sf '$MANAGER_URL/metrics'" "marchproxy_users_total"

    run_test "Manager license status endpoint" \
             "curl -sf '$MANAGER_URL/license-status'" "tier"

    run_test "Manager root API endpoint" \
             "curl -sf '$MANAGER_URL/'" "MarchProxy Manager"
}

test_proxy_egress() {
    print_status "Testing Proxy Egress..."

    run_test "Egress health endpoint" \
             "curl -sf '$EGRESS_URL/healthz'" "healthy"

    run_test "Egress metrics endpoint" \
             "curl -sf '$EGRESS_URL/metrics'" "marchproxy_tcp_connections_total"

    run_test "Egress stats endpoint" \
             "curl -sf '$EGRESS_URL/stats'" "tcp_connections"

    # Test mTLS configuration if enabled
    if [ -f "$CERT_DIR/client-cert.pem" ]; then
        run_test "Egress mTLS metrics" \
                 "curl -sf '$EGRESS_URL/metrics'" "marchproxy_mtls_enabled"
    fi
}

test_proxy_ingress() {
    print_status "Testing Proxy Ingress..."

    run_test "Ingress admin health endpoint" \
             "curl -sf '$INGRESS_ADMIN_URL/healthz'" "healthy"

    run_test "Ingress admin metrics endpoint" \
             "curl -sf '$INGRESS_ADMIN_URL/metrics'" "marchproxy_ingress_http_requests_total"

    # Test HTTP endpoint (should be available)
    run_test "Ingress HTTP endpoint accessible" \
             "curl -sf '$INGRESS_HTTP_URL/' -I" "HTTP"

    # Test HTTPS endpoint with self-signed certificate
    if [ -f "$CERT_DIR/server-cert.pem" ]; then
        run_test "Ingress HTTPS endpoint accessible (insecure)" \
                 "curl -k -sf '$INGRESS_HTTPS_URL/' -I" "HTTP"
    fi
}

test_mtls_communication() {
    print_status "Testing mTLS communication..."

    if [ ! -f "$CERT_DIR/client-cert.pem" ] || [ ! -f "$CERT_DIR/client-key.pem" ]; then
        print_warning "Client certificates not found, skipping mTLS tests"
        return 0
    fi

    # Test mTLS connection to egress proxy
    run_test "Egress mTLS health check" \
             "curl --cert '$CERT_DIR/client-cert.pem' --key '$CERT_DIR/client-key.pem' --cacert '$CERT_DIR/ca.pem' -sf 'https://localhost:8081/healthz'" "healthy"

    # Test mTLS connection to ingress proxy
    run_test "Ingress mTLS health check" \
             "curl --cert '$CERT_DIR/client-cert.pem' --key '$CERT_DIR/client-key.pem' --cacert '$CERT_DIR/ca.pem' -k -sf '$INGRESS_HTTPS_URL/'" ""

    # Test certificate validation
    run_test "mTLS certificate validation" \
             "openssl s_client -connect localhost:443 -cert '$CERT_DIR/client-cert.pem' -key '$CERT_DIR/client-key.pem' -CAfile '$CERT_DIR/ca.pem' -verify_return_error -quiet < /dev/null" ""
}

test_integration() {
    print_status "Testing integration scenarios..."

    # Test manager to proxy communication
    run_test "Manager can reach egress proxy" \
             "docker exec marchproxy-manager curl -sf http://marchproxy-proxy-egress:8081/healthz" "healthy"

    run_test "Manager can reach ingress proxy" \
             "docker exec marchproxy-manager curl -sf http://marchproxy-proxy-ingress:8082/healthz" "healthy"

    # Test inter-proxy communication
    run_test "Egress can reach ingress admin" \
             "docker exec marchproxy-proxy-egress curl -sf http://marchproxy-proxy-ingress:8082/healthz" "healthy"

    run_test "Ingress can reach egress admin" \
             "docker exec marchproxy-proxy-ingress curl -sf http://marchproxy-proxy-egress:8081/healthz" "healthy"
}

test_monitoring_stack() {
    print_status "Testing monitoring stack..."

    # Start monitoring services
    if docker-compose up -d prometheus grafana; then
        print_success "Monitoring services started"

        # Wait for Prometheus
        wait_for_service "http://localhost:9090" "Prometheus" 30

        # Wait for Grafana
        wait_for_service "http://localhost:3000" "Grafana" 30

        run_test "Prometheus is collecting metrics" \
                 "curl -sf 'http://localhost:9090/api/v1/targets'" "up"

        run_test "Grafana is accessible" \
                 "curl -sf 'http://localhost:3000/api/health'" "ok"
    else
        print_warning "Failed to start monitoring services"
    fi
}

test_performance() {
    print_status "Testing basic performance..."

    # Basic load test for ingress
    if command -v ab >/dev/null 2>&1; then
        run_test "Ingress basic load test (100 requests)" \
                 "ab -n 100 -c 10 -t 10 '$INGRESS_HTTP_URL/' | grep 'Requests per second'" ""
    else
        print_warning "Apache Bench (ab) not available, skipping load tests"
    fi

    # Basic connectivity test for egress
    run_test "Egress connection handling" \
             "for i in {1..10}; do curl -sf '$EGRESS_URL/healthz' >/dev/null; done" ""
}

cleanup() {
    print_status "Cleaning up test environment..."

    # Stop services gracefully
    docker-compose down --timeout 30

    # Remove test volumes if requested
    if [[ "${CLEANUP_VOLUMES:-false}" == "true" ]]; then
        docker-compose down -v
        print_status "Test volumes removed"
    fi
}

print_test_summary() {
    echo
    echo "=============================================="
    echo "           TEST SUMMARY"
    echo "=============================================="
    echo "Total tests: $TOTAL_TESTS"
    echo "Passed: $((TOTAL_TESTS - FAILED_TESTS))"
    echo "Failed: $FAILED_TESTS"
    echo

    if [ $FAILED_TESTS -eq 0 ]; then
        print_success "All tests passed! 🎉"
        echo
        print_status "MarchProxy dual proxy architecture is working correctly!"
        print_status "Both proxy-egress and proxy-ingress are operational with mTLS support."
    else
        print_error "Some tests failed. Please check the logs above."
        echo
        print_status "Failed tests:"
        for result in "${TEST_RESULTS[@]}"; do
            if [[ $result == FAIL* ]]; then
                echo "  - $result"
            fi
        done
    fi

    echo "=============================================="
    return $FAILED_TESTS
}

main() {
    echo "=============================================="
    echo "     MarchProxy Dual Proxy Test Suite"
    echo "=============================================="
    echo "Testing proxy-egress and proxy-ingress with mTLS"
    echo "Cert directory: $CERT_DIR"
    echo "Manager URL: $MANAGER_URL"
    echo "Egress URL: $EGRESS_URL"
    echo "Ingress HTTP URL: $INGRESS_HTTP_URL"
    echo "Ingress HTTPS URL: $INGRESS_HTTPS_URL"
    echo "Ingress Admin URL: $INGRESS_ADMIN_URL"
    echo "=============================================="
    echo

    # Set up signal handlers
    trap cleanup EXIT
    trap "print_error 'Test interrupted'; exit 1" INT TERM

    # Run test suites
    test_certificate_generation
    test_docker_services
    test_manager_api
    test_proxy_egress
    test_proxy_ingress
    test_mtls_communication
    test_integration
    test_monitoring_stack
    test_performance

    # Print final summary
    print_test_summary
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --cert-dir)
            CERT_DIR="$2"
            shift 2
            ;;
        --cleanup-volumes)
            CLEANUP_VOLUMES=true
            shift
            ;;
        --quick)
            # Skip performance and monitoring tests for quick validation
            test_monitoring_stack() { print_status "Skipping monitoring tests (quick mode)"; }
            test_performance() { print_status "Skipping performance tests (quick mode)"; }
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  --cert-dir DIR        Certificate directory (default: ./certs)"
            echo "  --cleanup-volumes     Remove Docker volumes after tests"
            echo "  --quick               Skip performance and monitoring tests"
            echo "  --help                Show this help message"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Run main function
main