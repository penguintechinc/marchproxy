#!/bin/bash

set -euo pipefail

# MarchProxy Restart Script
# Gracefully restarts all containers while preserving data

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
ENV_FILE="$ROOT_DIR/.env"

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

# Parse optional service argument
TARGET_SERVICE="${1:-all}"

log_info "Restarting MarchProxy services..."
log_info "Target: $TARGET_SERVICE"

if [ "$TARGET_SERVICE" = "all" ]; then
    log_info "Stopping all services..."
    docker-compose -f "$COMPOSE_FILE" down --remove-orphans

    log_info "Starting all services..."
    docker-compose -f "$COMPOSE_FILE" up -d
else
    log_info "Restarting specific service: $TARGET_SERVICE..."
    docker-compose -f "$COMPOSE_FILE" restart "$TARGET_SERVICE"
fi

log_success "Service restart initiated!"
log_info ""
log_info "Waiting for services to stabilize..."
sleep 5

log_info "Current status:"
docker-compose -f "$COMPOSE_FILE" ps

log_info ""
log_info "To check health status, run:"
log_info "  ./scripts/health-check.sh"
log_info ""
log_info "To view logs, run:"
log_info "  ./scripts/logs.sh [service-name]"
log_info ""
