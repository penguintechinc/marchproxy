#!/bin/bash
# Test script to verify proxy-alb structure and dependencies

set -e

echo "=========================================="
echo "MarchProxy ALB Structure Verification"
echo "=========================================="
echo ""

# Check directory structure
echo "✓ Checking directory structure..."
test -d internal/config && echo "  - internal/config exists"
test -d internal/envoy && echo "  - internal/envoy exists"
test -d internal/grpc && echo "  - internal/grpc exists"
test -d internal/metrics && echo "  - internal/metrics exists"
test -d envoy && echo "  - envoy exists"

echo ""

# Check key files
echo "✓ Checking key files..."
test -f main.go && echo "  - main.go exists"
test -f go.mod && echo "  - go.mod exists"
test -f Dockerfile && echo "  - Dockerfile exists"
test -f Makefile && echo "  - Makefile exists"
test -f README.md && echo "  - README.md exists"
test -f INTEGRATION.md && echo "  - INTEGRATION.md exists"

echo ""

# Check Go files
echo "✓ Checking Go source files..."
test -f internal/config/config.go && echo "  - config.go exists"
test -f internal/envoy/manager.go && echo "  - manager.go exists"
test -f internal/envoy/xds.go && echo "  - xds.go exists"
test -f internal/grpc/server.go && echo "  - server.go exists"
test -f internal/metrics/collector.go && echo "  - collector.go exists"

echo ""

# Check Envoy config
echo "✓ Checking Envoy configuration..."
test -f envoy/envoy.yaml && echo "  - envoy.yaml exists"

echo ""

# Check proto file
echo "✓ Checking proto definition..."
test -f ../proto/marchproxy/module_service.proto && echo "  - module_service.proto exists"

echo ""

# Syntax check Go files
echo "✓ Checking Go syntax..."
for file in main.go internal/*/*.go; do
    if [ -f "$file" ]; then
        gofmt -l "$file" > /dev/null 2>&1 && echo "  - $file syntax OK" || echo "  - $file has syntax errors"
    fi
done

echo ""

# Check Dockerfile syntax
echo "✓ Checking Dockerfile syntax..."
if command -v hadolint &> /dev/null; then
    hadolint Dockerfile && echo "  - Dockerfile OK"
else
    echo "  - hadolint not installed, skipping"
fi

echo ""

# Check YAML syntax
echo "✓ Checking YAML syntax..."
if command -v yamllint &> /dev/null; then
    yamllint envoy/envoy.yaml && echo "  - envoy.yaml OK"
    yamllint docker-compose.example.yml && echo "  - docker-compose.example.yml OK"
    yamllint prometheus.yml && echo "  - prometheus.yml OK"
else
    echo "  - yamllint not installed, skipping"
fi

echo ""
echo "=========================================="
echo "Structure verification complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo "  1. cd /home/penguin/code/MarchProxy/proxy-alb"
echo "  2. make proto    # Generate protobuf code"
echo "  3. make build    # Build Go binary"
echo "  4. make build-docker  # Build Docker image"
echo ""
