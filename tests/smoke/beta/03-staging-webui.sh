#!/bin/bash
# Beta Smoke Test 3: Staging WebUI check
# Verifies WebUI is accessible at https://marchproxy.penguintech.io

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=========================================="
echo "Beta Smoke Test 3: Staging WebUI"
echo "=========================================="
echo ""

BASE_URL="https://marchproxy.penguintech.io"
FAILED=0

echo "Checking WebUI at $BASE_URL..."
if curl -f -L -s "$BASE_URL" > /tmp/webui-response.html 2>&1; then
    # Check for common HTML elements
    if grep -q "<html\|<head\|<body" /tmp/webui-response.html; then
        echo "✅ WebUI HTML content found"

        # Check for React app
        if grep -q "react\|React\|root" /tmp/webui-response.html; then
            echo "✅ React app detected"
        else
            echo "⚠️  React app not detected in HTML"
        fi

        # Check for MarchProxy branding
        if grep -qi "marchproxy\|March Proxy" /tmp/webui-response.html; then
            echo "✅ MarchProxy branding found"
        else
            echo "⚠️  MarchProxy branding not found"
        fi
    else
        echo "❌ No valid HTML content found"
        echo "Response preview:"
        head -20 /tmp/webui-response.html
        FAILED=1
    fi
else
    echo "❌ WebUI not responding"
    FAILED=1
fi

# Check if Cloudflare is intercepting
if grep -q "cloudflare\|challenge" /tmp/webui-response.html 2>/dev/null; then
    echo "⚠️  Cloudflare challenge detected - unable to fully test WebUI"
    echo "   Manual verification required"
fi

rm -f /tmp/webui-response.html

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ WebUI checks passed"
    exit 0
else
    echo "❌ WebUI checks failed"
    exit 1
fi
