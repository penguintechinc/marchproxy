#!/bin/bash
# Setup script for MarchProxy testing environment
# Makes test scripts executable and prepares the testing environment

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_status "Setting up MarchProxy testing environment..."

# Make scripts executable
chmod +x scripts/generate-certs.sh
chmod +x scripts/test-proxies.sh
chmod +x scripts/test-mtls.sh
chmod +x scripts/setup-tests.sh
chmod +x scripts/setup-website-submodule.sh

print_success "Test scripts are now executable"

# Create certs directory if it doesn't exist
if [ ! -d "certs" ]; then
    mkdir -p certs
    print_status "Created certs directory"
fi

# Check Docker and Docker Compose
if command -v docker >/dev/null 2>&1; then
    print_success "Docker is available"
else
    echo "❌ Docker is not installed or not in PATH"
    exit 1
fi

if command -v docker-compose >/dev/null 2>&1; then
    print_success "Docker Compose is available"
else
    echo "❌ Docker Compose is not installed or not in PATH"
    exit 1
fi

# Check OpenSSL
if command -v openssl >/dev/null 2>&1; then
    print_success "OpenSSL is available"
else
    echo "❌ OpenSSL is not installed or not in PATH"
    exit 1
fi

print_status "Testing environment setup complete!"
print_status ""
print_status "Next steps:"
print_status "1. Generate certificates: docker-compose --profile tools run --rm cert-generator"
print_status "2. Start services: docker-compose up -d"
print_status "3. Run tests: ./scripts/test-proxies.sh"
print_status ""
print_status "For more information, see TESTING.md"