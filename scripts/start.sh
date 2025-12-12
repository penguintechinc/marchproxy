#!/bin/bash

set -euo pipefail

# MarchProxy Start Script
# Starts all containers with proper dependency ordering

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
ENV_FILE="$ROOT_DIR/.env"
STARTUP_TIMEOUT=120

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if docker-compose is installed
if ! command -v docker-compose &> /dev/null; then
    log_error "docker-compose is not installed. Please install it first."
    exit 1
fi

# Check if .env file exists
if [ ! -f "$ENV_FILE" ]; then
    log_warn ".env file not found. Creating from .env.example..."
    if [ -f "$ROOT_DIR/.env.example" ]; then
        cp "$ROOT_DIR/.env.example" "$ENV_FILE"
        log_info "Please review and update $ENV_FILE with your settings"
    else
        log_error ".env.example not found"
        exit 1
    fi
fi

# Parse environment to get passwords
export $(grep -v '^#' "$ENV_FILE" | grep -v '^$' | xargs)

log_info "Starting MarchProxy services..."
log_info "Docker Compose file: $COMPOSE_FILE"

# Start the Docker Compose services
# Start infrastructure first (postgres, redis, elasticsearch)
log_info "Starting infrastructure services (postgres, redis, elasticsearch)..."
docker-compose -f "$COMPOSE_FILE" up -d postgres redis elasticsearch

# Wait for infrastructure to be healthy
log_info "Waiting for infrastructure services to be healthy..."
sleep 5

# Start observability services (logstash, jaeger, prometheus)
log_info "Starting observability services..."
docker-compose -f "$COMPOSE_FILE" up -d logstash jaeger prometheus alertmanager loki promtail

# Wait for observability
sleep 5

# Start core services (api-server, manager, webui)
log_info "Starting core services..."
docker-compose -f "$COMPOSE_FILE" up -d kibana grafana config-sync api-server manager webui

# Wait for core services
sleep 10

# Start proxy services (proxy-l7, proxy-l3l4, proxy-egress, proxy-ingress)
log_info "Starting proxy services..."
docker-compose -f "$COMPOSE_FILE" up -d proxy-l7 proxy-l3l4 proxy-egress proxy-ingress

log_info "All services started. Waiting for health checks..."

# Wait for services to become healthy with timeout
start_time=$(date +%s)
unhealthy=true

while [ $unhealthy = true ]; do
    current_time=$(date +%s)
    elapsed=$((current_time - start_time))

    if [ $elapsed -gt $STARTUP_TIMEOUT ]; then
        log_warn "Startup timeout reached. Some services may still be initializing..."
        break
    fi

    # Check critical services
    postgres_status=$(docker-compose -f "$COMPOSE_FILE" ps postgres --format json 2>/dev/null | grep -q '"State":"running"' && echo "up" || echo "down")
    redis_status=$(docker-compose -f "$COMPOSE_FILE" ps redis --format json 2>/dev/null | grep -q '"State":"running"' && echo "up" || echo "down")
    api_status=$(docker-compose -f "$COMPOSE_FILE" ps api-server --format json 2>/dev/null | grep -q '"State":"running"' && echo "up" || echo "down")

    if [ "$postgres_status" = "up" ] && [ "$redis_status" = "up" ] && [ "$api_status" = "up" ]; then
        unhealthy=false
    else
        sleep 2
    fi
done

# Show status
log_info "Docker Compose services status:"
docker-compose -f "$COMPOSE_FILE" ps

# Print access information
log_success "MarchProxy services are starting up!"
log_info "=============================================="
log_info "Access Information:"
log_info "=============================================="
log_info "Web UI:              http://localhost:3000"
log_info "Manager API:         http://localhost:8000"
log_info "Envoy Admin:         http://localhost:9901"
log_info "Prometheus:          http://localhost:9090"
log_info "Grafana:             http://localhost:3000 (WebUI)"
log_info "Jaeger Tracing:      http://localhost:16686"
log_info "Kibana:              http://localhost:5601"
log_info "AlertManager:        http://localhost:9093"
log_info "=============================================="
log_info ""
log_info "Services may still be initializing. Check health with:"
log_info "  docker-compose ps"
log_info "  docker-compose logs -f api-server"
log_info "  docker-compose logs -f webui"
log_info ""
log_info "To stop all services:"
log_info "  ./scripts/stop.sh"
log_info ""
log_info "For health checks, run:"
log_info "  ./scripts/health-check.sh"
log_info ""
