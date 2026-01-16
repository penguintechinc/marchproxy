#!/bin/bash
# Beta Smoke Test 2: Staging API health checks
# Tests against https://marchproxy.penguintech.io

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "Beta Smoke Test 2: Staging API Health"
echo "=========================================="
echo ""

BASE_URL="https://marchproxy.penguintech.io"
FAILED=0

# Health check
echo "Checking $BASE_URL/healthz..."
if curl -f -s "$BASE_URL/healthz" > /dev/null 2>&1; then
    RESPONSE=$(curl -s "$BASE_URL/healthz")
    if echo "$RESPONSE" | grep -q "healthy\|status"; then
        echo "✅ /healthz responding correctly"
    else
        echo "❌ /healthz returned unexpected response"
        echo "Response: $RESPONSE"
        FAILED=1
    fi
else
    echo "❌ /healthz not responding"
    FAILED=1
fi

# Readiness check
echo "Checking $BASE_URL/healthz/ready..."
if curl -f -s "$BASE_URL/healthz/ready" > /dev/null 2>&1; then
    echo "✅ /healthz/ready responding correctly"
else
    echo "⚠️  /healthz/ready not responding (may be behind auth)"
fi

# Metrics check
echo "Checking $BASE_URL/metrics..."
if curl -f -s "$BASE_URL/metrics" > /dev/null 2>&1; then
    echo "✅ /metrics responding correctly"
else
    echo "⚠️  /metrics not responding (may be behind auth)"
fi

# Root endpoint
echo "Checking $BASE_URL/..."
if curl -f -s "$BASE_URL/" > /dev/null 2>&1; then
    RESPONSE=$(curl -s "$BASE_URL/")
    if echo "$RESPONSE" | grep -q "MarchProxy\|API"; then
        echo "✅ / responding correctly"
    else
        echo "⚠️  / returned unexpected response"
    fi
else
    echo "⚠️  / not responding (may be behind Cloudflare)"
fi

# RBAC API check
echo "Checking $BASE_URL/api/v1/roles/permissions..."
if curl -f -s "$BASE_URL/api/v1/roles/permissions" > /dev/null 2>&1; then
    RESPONSE=$(curl -s "$BASE_URL/api/v1/roles/permissions")
    if echo "$RESPONSE" | grep -q "permissions"; then
        PERM_COUNT=$(echo "$RESPONSE" | grep -o "global:" | wc -l)
        echo "✅ RBAC API responding correctly ($PERM_COUNT permissions found)"
    else
        echo "❌ RBAC API returned unexpected response"
        FAILED=1
    fi
else
    echo "⚠️  RBAC API not responding (may be behind auth)"
fi

# Check SSL certificate
echo ""
echo "Checking SSL certificate..."
if curl -vI "$BASE_URL" 2>&1 | grep -q "SSL certificate verify ok"; then
    echo "✅ SSL certificate valid"
else
    echo "⚠️  SSL certificate check inconclusive"
fi

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All critical API checks passed"
    exit 0
else
    echo "❌ Some critical API checks failed"
    exit 1
fi
