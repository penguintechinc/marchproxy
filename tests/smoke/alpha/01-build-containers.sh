#!/bin/bash
# Alpha Smoke Test 1: Build all containers
# This verifies that all Dockerfiles are valid and containers build successfully

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "=========================================="
echo "Alpha Smoke Test 1: Build All Containers"
echo "=========================================="
echo ""

cd "$PROJECT_ROOT"

FAILED=0

# Build manager
echo "Building manager..."
if docker build --target production -t marchproxy-manager:smoke-test -f manager/Dockerfile . > /tmp/build-manager.log 2>&1; then
    echo "✅ Manager build successful"
else
    echo "❌ Manager build failed"
    echo "See: /tmp/build-manager.log"
    FAILED=1
fi

# Build proxy-egress
echo "Building proxy-egress..."
if docker build -t marchproxy-egress:smoke-test -f proxy-egress/Dockerfile proxy-egress > /tmp/build-egress.log 2>&1; then
    echo "✅ Proxy-egress build successful"
else
    echo "❌ Proxy-egress build failed"
    echo "See: /tmp/build-egress.log"
    FAILED=1
fi

# Build proxy-ailb
echo "Building proxy-ailb..."
if docker build -t marchproxy-ailb:smoke-test -f proxy-ailb/Dockerfile proxy-ailb > /tmp/build-ailb.log 2>&1; then
    echo "✅ Proxy-ailb build successful"
else
    echo "❌ Proxy-ailb build failed"
    echo "See: /tmp/build-ailb.log"
    FAILED=1
fi

# Build proxy-nlb
echo "Building proxy-nlb..."
if docker build -t marchproxy-nlb:smoke-test -f proxy-nlb/Dockerfile proxy-nlb > /tmp/build-nlb.log 2>&1; then
    echo "✅ Proxy-nlb build successful"
else
    echo "❌ Proxy-nlb build failed"
    echo "See: /tmp/build-nlb.log"
    FAILED=1
fi

# Build webui
echo "Building webui..."
if docker build -t marchproxy-webui:smoke-test -f webui/Dockerfile webui > /tmp/build-webui.log 2>&1; then
    echo "✅ WebUI build successful"
else
    echo "❌ WebUI build failed"
    echo "See: /tmp/build-webui.log"
    FAILED=1
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All containers built successfully"
    exit 0
else
    echo "❌ Some containers failed to build"
    exit 1
fi
