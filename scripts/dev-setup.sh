#!/bin/bash

# MarchProxy Development Setup Script
# This script sets up the development environment and initializes the project

set -e

echo "🚀 MarchProxy Development Setup"
echo "================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running from project root
if [ ! -f ".PLAN" ] || [ ! -f ".TODO" ]; then
    log_error "This script must be run from the MarchProxy project root directory"
    exit 1
fi

# Check prerequisites
log_info "Checking prerequisites..."

# Check Docker
if ! command -v docker &> /dev/null; then
    log_error "Docker is required but not installed"
    exit 1
fi
log_success "Docker found"

# Check Docker Compose
if ! command -v docker-compose &> /dev/null; then
    log_error "Docker Compose is required but not installed"
    exit 1
fi
log_success "Docker Compose found"

# Check if Docker is running
if ! docker info &> /dev/null; then
    log_error "Docker is not running. Please start Docker first."
    exit 1
fi
log_success "Docker is running"

# Create necessary directories
log_info "Creating development directories..."
mkdir -p docker/dev/certs
mkdir -p manager/logs
mkdir -p proxy/logs

# Generate development certificates if they don't exist
if [ ! -f "docker/dev/certs/cert.pem" ]; then
    log_info "Generating development TLS certificates..."
    openssl req -x509 -newkey rsa:4096 -keyout docker/dev/certs/key.pem -out docker/dev/certs/cert.pem \
        -days 365 -nodes -subj "/C=US/ST=Dev/L=Development/O=MarchProxy/CN=localhost"
    log_success "Development certificates generated"
else
    log_info "Development certificates already exist"
fi

# Create development password file for py4web
if [ ! -f "manager/password.txt" ]; then
    echo "marchproxy_admin_dev" > manager/password.txt
    log_success "py4web password file created"
fi

# Create development environment file
if [ ! -f ".env.dev" ]; then
    log_info "Creating development environment file..."
    cat > .env.dev << EOF
# MarchProxy Development Environment
# Source this file: source .env.dev

# Database
export DB_URI="postgresql://marchproxy:marchproxy_dev_password@localhost:5432/marchproxy"

# py4web Manager
export PY4WEB_APPS_FOLDER="/app"
export PY4WEB_PASSWORD="marchproxy_admin_dev"
export JWT_SECRET="marchproxy_jwt_secret_dev_change_in_production"

# Proxy Configuration
export MANAGER_URL="http://localhost:8000"
export CLUSTER_API_KEY="dev_cluster_api_key_change_in_production"
export LOG_LEVEL="DEBUG"
export SYSLOG_ENDPOINT="127.0.0.1:514"

# License (optional, for Enterprise features)
export LICENSE_KEY=""
export LICENSE_SERVER_URL="https://license.penguintech.io"

# Development flags
export FLASK_ENV="development"
export PYTHONPATH="/app"
EOF
    log_success "Development environment file created (.env.dev)"
    log_warning "Remember to source .env.dev for local development: source .env.dev"
else
    log_info "Development environment file already exists"
fi

# Initialize Go module if needed
if [ ! -f "proxy/go.sum" ]; then
    log_info "Initializing Go dependencies..."
    cd proxy
    go mod tidy
    cd ..
    log_success "Go dependencies initialized"
fi

# Create development Docker Compose override
if [ ! -f "docker-compose.override.yml" ]; then
    log_info "Creating Docker Compose override for development..."
    cat > docker-compose.override.yml << EOF
# MarchProxy Development Override
# This file provides development-specific Docker Compose configuration

version: '3.8'

services:
  manager:
    build:
      target: development
    volumes:
      - ./manager:/app:cached
      - ./docker/dev/certs:/app/certs:ro
    environment:
      - FLASK_DEBUG=1
      - PYTHONDONTWRITEBYTECODE=1
    command: ["python", "-m", "py4web", "run", "apps", "--host=0.0.0.0", "--port=8000", "--debug"]
    
  proxy:
    build:
      target: development
    volumes:
      - ./proxy:/app:cached
    environment:
      - CGO_ENABLED=1
      - GOOS=linux
    # Enable for debugging
    # ports:
    #   - "40000:40000"  # Delve debugger port

  # Development tools
  postgres:
    ports:
      - "5432:5432"  # Expose postgres for local development
    
  adminer:
    environment:
      - ADMINER_DESIGN=nette
      - ADMINER_DEFAULT_SERVER=postgres
EOF
    log_success "Docker Compose override created"
else
    log_info "Docker Compose override already exists"
fi

# Build images
log_info "Building development Docker images..."
docker-compose build

log_success "Development setup complete!"
echo
echo "📋 Next Steps:"
echo "==============="
echo "1. Start the development environment:"
echo "   docker-compose up -d"
echo
echo "2. Check service status:"
echo "   docker-compose ps"
echo
echo "3. View logs:"
echo "   docker-compose logs -f manager  # Manager logs"
echo "   docker-compose logs -f proxy    # Proxy logs"
echo
echo "4. Access services:"
echo "   - Manager Web UI: http://localhost:8000"
echo "   - Proxy Admin: http://localhost:8081"
echo "   - Database Admin (Adminer): http://localhost:8082"
echo "   - Health checks:"
echo "     - Manager: http://localhost:8000/healthz"
echo "     - Proxy: http://localhost:8081/healthz"
echo
echo "5. For local development (outside Docker):"
echo "   source .env.dev"
echo
echo "6. Stop the environment:"
echo "   docker-compose down"
echo
echo "🔧 Development Resources:"
echo "========================="
echo "- Project documentation: ./docs/"
echo "- Architecture plan: ./.PLAN"
echo "- Development tasks: ./.TODO"
echo "- API documentation: ./docs/api/"
echo
echo "Happy coding! 🎉"