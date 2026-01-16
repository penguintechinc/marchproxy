#!/bin/bash
# Alpha Smoke Test 3: API health checks
# Verifies all API endpoints respond correctly

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "=========================================="
echo "Alpha Smoke Test 3: API Health Checks"
echo "=========================================="
echo ""

FAILED=0

# Manager API health check
echo "Checking Manager API (/healthz)..."
if curl -f -s http://localhost:8000/healthz > /dev/null 2>&1; then
    RESPONSE=$(curl -s http://localhost:8000/healthz)
    if echo "$RESPONSE" | grep -q "healthy"; then
        echo "✅ Manager /healthz responding correctly"
    else
        echo "❌ Manager /healthz returned unexpected response"
        echo "Response: $RESPONSE"
        FAILED=1
    fi
else
    echo "❌ Manager /healthz not responding"
    FAILED=1
fi

# Manager readiness check
echo "Checking Manager API (/healthz/ready)..."
if curl -f -s http://localhost:8000/healthz/ready > /dev/null 2>&1; then
    echo "✅ Manager /healthz/ready responding correctly"
else
    echo "❌ Manager /healthz/ready not responding"
    FAILED=1
fi

# Manager metrics check
echo "Checking Manager API (/metrics)..."
if curl -f -s http://localhost:8000/metrics > /dev/null 2>&1; then
    echo "✅ Manager /metrics responding correctly"
else
    echo "❌ Manager /metrics not responding"
    FAILED=1
fi

# Manager root endpoint
echo "Checking Manager API (/)..."
if curl -f -s http://localhost:8000/ > /dev/null 2>&1; then
    RESPONSE=$(curl -s http://localhost:8000/)
    if echo "$RESPONSE" | grep -q "MarchProxy"; then
        echo "✅ Manager / responding correctly"
    else
        echo "❌ Manager / returned unexpected response"
        FAILED=1
    fi
else
    echo "❌ Manager / not responding"
    FAILED=1
fi

# RBAC permissions endpoint (added in this PR)
echo "Checking RBAC API (/api/v1/roles/permissions)..."
if curl -f -s http://localhost:8000/api/v1/roles/permissions > /dev/null 2>&1; then
    RESPONSE=$(curl -s http://localhost:8000/api/v1/roles/permissions)
    if echo "$RESPONSE" | grep -q "permissions"; then
        echo "✅ RBAC /api/v1/roles/permissions responding correctly"
    else
        echo "❌ RBAC API returned unexpected response"
        FAILED=1
    fi
else
    echo "❌ RBAC /api/v1/roles/permissions not responding"
    FAILED=1
fi

# WebUI health check (if available)
echo "Checking WebUI (http://localhost:3000)..."
if curl -f -s http://localhost:3000 > /dev/null 2>&1; then
    echo "✅ WebUI responding"
else
    echo "⚠️  WebUI not responding (may not be in docker-compose)"
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All API health checks passed"
    exit 0
else
    echo "❌ Some API health checks failed"
    exit 1
fi
