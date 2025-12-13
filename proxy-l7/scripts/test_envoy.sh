#!/bin/bash
# Test script for Envoy configuration validation

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BOOTSTRAP_FILE="$PROJECT_ROOT/envoy/bootstrap.yaml"
DOCKER_IMAGE="marchproxy/proxy-l7:latest"

echo "═══════════════════════════════════════════════════════════"
echo " MarchProxy Envoy Configuration Validation"
echo "═══════════════════════════════════════════════════════════"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Test 1: Check bootstrap configuration exists
echo "Test 1: Bootstrap configuration file"
if [ -f "$BOOTSTRAP_FILE" ]; then
    success "Bootstrap file exists: $BOOTSTRAP_FILE"
else
    error "Bootstrap file not found: $BOOTSTRAP_FILE"
    exit 1
fi

# Test 2: Validate YAML syntax
echo ""
echo "Test 2: YAML syntax validation"
if command -v python3 &> /dev/null; then
    if python3 -c "import yaml; yaml.safe_load(open('$BOOTSTRAP_FILE'))" 2>/dev/null; then
        success "YAML syntax is valid"
    else
        error "YAML syntax is invalid"
        python3 -c "import yaml; yaml.safe_load(open('$BOOTSTRAP_FILE'))"
        exit 1
    fi
else
    warning "Python3 not available, skipping YAML validation"
fi

# Test 3: Check required Envoy sections
echo ""
echo "Test 3: Required Envoy configuration sections"

check_config_section() {
    local section=$1
    if grep -q "^$section:" "$BOOTSTRAP_FILE"; then
        success "Section present: $section"
        return 0
    else
        error "Section missing: $section"
        return 1
    fi
}

SECTIONS_OK=0
REQUIRED_SECTIONS=("admin" "node" "dynamic_resources" "static_resources")

for section in "${REQUIRED_SECTIONS[@]}"; do
    if check_config_section "$section"; then
        SECTIONS_OK=$((SECTIONS_OK + 1))
    fi
done

if [ $SECTIONS_OK -ne ${#REQUIRED_SECTIONS[@]} ]; then
    error "Missing required configuration sections"
    exit 1
fi

# Test 4: Check xDS configuration
echo ""
echo "Test 4: xDS configuration"

if grep -q "xds_cluster" "$BOOTSTRAP_FILE"; then
    success "xDS cluster defined"
else
    error "xDS cluster not defined"
    exit 1
fi

if grep -q "api-server" "$BOOTSTRAP_FILE"; then
    success "API server reference found"
else
    error "API server reference not found"
    exit 1
fi

if grep -q "18000" "$BOOTSTRAP_FILE"; then
    success "xDS port configured (18000)"
else
    warning "xDS port 18000 not found in config"
fi

# Test 5: Check admin interface
echo ""
echo "Test 5: Admin interface configuration"

if grep -q "9901" "$BOOTSTRAP_FILE"; then
    success "Admin port configured (9901)"
else
    error "Admin port not configured"
    exit 1
fi

# Test 6: Check ADS configuration
echo ""
echo "Test 6: Aggregated Discovery Service (ADS)"

if grep -q "ads_config" "$BOOTSTRAP_FILE"; then
    success "ADS configuration present"
else
    warning "ADS configuration not found"
fi

if grep -q "resource_api_version: V3" "$BOOTSTRAP_FILE"; then
    success "Using xDS API v3"
else
    warning "xDS API version not V3"
fi

# Test 7: Validate with Envoy (if Docker image exists)
echo ""
echo "Test 7: Envoy configuration validation"

if command -v docker &> /dev/null && docker info &> /dev/null; then
    # Check if Docker image exists
    if docker image inspect "$DOCKER_IMAGE" &> /dev/null; then
        success "Docker image found: $DOCKER_IMAGE"

        echo "  Validating configuration with Envoy..."

        # Run Envoy in validation mode
        VALIDATION_OUTPUT=$(docker run --rm \
            -v "$BOOTSTRAP_FILE:/tmp/envoy.yaml:ro" \
            "$DOCKER_IMAGE" \
            envoy --mode validate -c /tmp/envoy.yaml 2>&1)

        if echo "$VALIDATION_OUTPUT" | grep -q "OK"; then
            success "Envoy configuration is valid"
        else
            error "Envoy configuration validation failed"
            echo "$VALIDATION_OUTPUT"
            exit 1
        fi
    else
        warning "Docker image not found: $DOCKER_IMAGE"
        echo "  Build image with: make build-docker"
        echo "  Skipping Envoy validation..."
    fi
else
    warning "Docker not available, skipping Envoy validation"
fi

# Test 8: Check Dockerfile
echo ""
echo "Test 8: Dockerfile validation"

DOCKERFILE="$PROJECT_ROOT/envoy/Dockerfile"
if [ -f "$DOCKERFILE" ]; then
    success "Dockerfile exists"

    # Check for multi-stage build
    if grep -q "AS xdp-builder" "$DOCKERFILE" && \
       grep -q "AS wasm-builder" "$DOCKERFILE"; then
        success "Multi-stage build configured"
    else
        warning "Multi-stage build not detected"
    fi

    # Check for Envoy base image
    if grep -q "envoyproxy/envoy" "$DOCKERFILE"; then
        success "Using official Envoy base image"
    else
        error "Envoy base image not found"
    fi

    # Check Envoy version
    if grep -q "envoyproxy/envoy:v1.28" "$DOCKERFILE"; then
        success "Using Envoy v1.28+"
    else
        warning "Envoy version might not be v1.28+"
    fi
else
    error "Dockerfile not found: $DOCKERFILE"
    exit 1
fi

# Test 9: Check for security best practices
echo ""
echo "Test 9: Security configuration"

# Check for TLS configuration potential
if grep -qi "tls" "$BOOTSTRAP_FILE" || grep -qi "ssl" "$BOOTSTRAP_FILE"; then
    success "TLS/SSL configuration present"
else
    warning "No TLS/SSL configuration found"
fi

# Check for access logging
if grep -q "access_log" "$BOOTSTRAP_FILE"; then
    success "Access logging configured"
else
    warning "Access logging not configured"
fi

# Test 10: Check runtime limits
echo ""
echo "Test 10: Runtime limits and safety"

if grep -q "global_downstream_max_connections" "$BOOTSTRAP_FILE"; then
    MAX_CONN=$(grep "global_downstream_max_connections" "$BOOTSTRAP_FILE" | grep -oE '[0-9]+')
    success "Max connections limit set: $MAX_CONN"
else
    warning "Max connections limit not set"
fi

# Summary
echo ""
echo "═══════════════════════════════════════════════════════════"
echo " Validation Summary"
echo "═══════════════════════════════════════════════════════════"
echo ""
success "All critical validations passed!"
echo ""
echo "Configuration Details:"
echo "  - Bootstrap: $BOOTSTRAP_FILE"
echo "  - xDS Server: api-server:18000"
echo "  - Admin Port: 9901"
echo "  - API Version: V3"
echo ""
echo "Next Steps:"
echo "  1. Build Docker image: make build-docker"
echo "  2. Start container: docker run -d -p 10000:10000 -p 9901:9901 $DOCKER_IMAGE"
echo "  3. Check health: curl http://localhost:9901/ready"
echo "  4. View stats: curl http://localhost:9901/stats/prometheus"
echo ""

exit 0
