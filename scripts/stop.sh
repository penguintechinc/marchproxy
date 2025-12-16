#!/bin/bash

set -euo pipefail

# MarchProxy Stop Script
# Gracefully stops all containers

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
    log_error "docker-compose is not installed"
    exit 1
fi

# Check if .env exists for docker-compose context
if [ ! -f "$ENV_FILE" ]; then
    log_warn "No .env file found, using defaults"
fi

log_info "Stopping MarchProxy services..."

# Stop all services gracefully
docker-compose -f "$COMPOSE_FILE" down --remove-orphans

log_success "All MarchProxy services stopped"

# Show summary
log_info "=============================================="
log_info "Stop Summary:"
log_info "=============================================="
log_info "All containers have been stopped"
log_info "Volumes and networks are preserved"
log_info ""
log_info "To remove volumes as well (warning: data loss):"
log_info "  docker-compose down -v"
log_info ""
log_info "To remove everything including networks:"
log_info "  docker-compose down --remove-orphans"
log_info ""
