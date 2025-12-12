#!/bin/bash

set -euo pipefail

# MarchProxy Database Migration Script
# Runs database migrations using Alembic in the API server

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

# Check if .env exists
if [ ! -f "$ENV_FILE" ]; then
    log_error ".env file not found. Please create it first: cp .env.example .env"
    exit 1
fi

log_info "Running MarchProxy database migrations..."

# Check if api-server is running
if ! docker-compose -f "$COMPOSE_FILE" ps api-server 2>/dev/null | grep -q "running"; then
    log_warn "api-server is not running. Starting it..."
    docker-compose -f "$COMPOSE_FILE" up -d api-server
    log_info "Waiting for api-server to be ready..."
    sleep 10
fi

# Check if postgres is healthy
log_info "Checking PostgreSQL connection..."
if ! docker-compose -f "$COMPOSE_FILE" exec -T postgres pg_isready -U marchproxy &>/dev/null; then
    log_error "PostgreSQL is not responding"
    exit 1
fi

log_success "PostgreSQL is ready"

# Run Alembic migrations
log_info "Running Alembic migrations..."
if docker-compose -f "$COMPOSE_FILE" exec -T api-server alembic upgrade head; then
    log_success "Database migrations completed successfully"
else
    log_error "Database migration failed"
    exit 1
fi

# Verify migration
log_info "Verifying migration..."
if docker-compose -f "$COMPOSE_FILE" exec -T api-server python -c "from app.models import Base; print('Models loaded successfully')" 2>/dev/null; then
    log_success "Database verification passed"
else
    log_warn "Could not verify database models"
fi

log_success "Migration complete!"

# Print next steps
log_info "=============================================="
log_info "Migration Complete"
log_info "=============================================="
log_info "Next steps:"
log_info "1. Verify data integrity:"
log_info "   docker-compose exec postgres psql -U marchproxy -d marchproxy -c '\\dt'"
log_info ""
log_info "2. Create initial admin user (if needed):"
log_info "   docker-compose exec api-server python -c 'from app.utils import create_admin; create_admin()'"
log_info ""
log_info "3. Check logs:"
log_info "   docker-compose logs -f api-server"
log_info ""
