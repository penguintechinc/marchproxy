#!/bin/bash

set -euo pipefail

# MarchProxy Health Check Script
# Verifies all services are healthy

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
FAILED_CHECKS=0
PASSED_CHECKS=0

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
    ((PASSED_CHECKS++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
    ((FAILED_CHECKS++))
}

check_service() {
    local service_name=$1
    local port=$2
    local path=${3:-"/"}
    local protocol=${4:-"http"}

    # Check if container is running
    if ! docker-compose -f "$COMPOSE_FILE" ps "$service_name" 2>/dev/null | grep -q "running"; then
        log_error "$service_name is not running"
        return 1
    fi

    # Try to connect
    local url="${protocol}://localhost:${port}${path}"
    if curl -sf "$url" > /dev/null 2>&1; then
        log_success "$service_name is healthy ($url)"
        return 0
    else
        log_error "$service_name is not responding ($url)"
        return 1
    fi
}

check_container_running() {
    local service_name=$1

    if docker-compose -f "$COMPOSE_FILE" ps "$service_name" 2>/dev/null | grep -q "running"; then
        log_success "$service_name is running"
        return 0
    else
        log_error "$service_name is not running"
        return 1
    fi
}

# Header
log_info "Running MarchProxy Health Checks"
echo ""

# Infrastructure Services
log_info "Infrastructure Services:"
echo ""

check_container_running "postgres" || true
check_container_running "redis" || true
check_container_running "elasticsearch" || true

echo ""

# Observability Services
log_info "Observability Services:"
echo ""

check_container_running "logstash" || true
check_container_running "prometheus" || true
check_service "jaeger" "16686" "/api/traces" "http" || true
check_service "grafana" "3000" "/" "http" || true
check_service "kibana" "5601" "/api/status" "http" || true
check_container_running "alertmanager" || true
check_container_running "loki" || true
check_container_running "promtail" || true

echo ""

# Core Services
log_info "Core Services:"
echo ""

check_service "api-server" "8000" "/healthz" "http" || true
check_service "manager" "8000" "/health" "http" || true
check_service "webui" "3000" "/" "http" || true

echo ""

# Proxy Services
log_info "Proxy Services:"
echo ""

check_service "proxy-l7" "9901" "/stats" "http" || true
check_service "proxy-l3l4" "8082" "/healthz" "http" || true
check_service "proxy-egress" "8081" "/healthz" "http" || true
check_service "proxy-ingress" "8082" "/healthz" "http" || true

echo ""

# Other Services
log_info "Additional Services:"
echo ""

check_container_running "config-sync" || true

echo ""
echo ""

# Summary
log_info "=============================================="
log_info "Health Check Summary"
log_info "=============================================="
log_success "Passed: $PASSED_CHECKS"
log_error "Failed: $FAILED_CHECKS"

if [ $FAILED_CHECKS -eq 0 ]; then
    log_success "All services are healthy!"
    exit 0
else
    log_warn "Some services are not healthy. Check logs with:"
    echo "  docker-compose logs -f <service_name>"
    exit 1
fi
