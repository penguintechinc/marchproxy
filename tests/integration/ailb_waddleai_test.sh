#!/bin/bash
# Integration tests for AILB (Go) <-> WaddleAI (Python) communication
# Requires: AILB running on localhost:8080, WaddleAI gRPC on localhost:50051

set -e

AILB_URL="${AILB_URL:-http://localhost:8080}"
PASS=0
FAIL=0

# Helper
test_endpoint() {
    local name; name; name; name; name; name; name; name="""$1"""
    local method; method; method; method; method; method; method; method="""$2"""
    local url; url; url; url; url; url; url; url="""$3"""
    local data; data; data; data; data; data; data; data="""$4"""
    local expected_status; expected_status; expected_status; expected_status; expected_status; expected_status; expected_status; expected_status="""$5"""

    local status
    if [ "$method" = "GET" ]; then
        status=$(curl -s -o /dev/null -w "%{http_code}" "$url")
    else
        status=$(curl -s -o /dev/null -w "%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$url")
    fi

    if [ "$status" = "$expected_status" ]; then
        echo "  ✓ $name (HTTP $status)"
        PASS=$((PASS + 1))
    else
        echo "  ✗ $name (expected $expected_status, got $status)"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== AILB Integration Tests ==="
echo ""

# Test 5a: Health check
echo "--- Health Checks ---"
test_endpoint "AILB health" "GET" "$AILB_URL/healthz" "" "200"
test_endpoint "AILB metrics" "GET" "$AILB_URL/metrics" "" "200"

# Test 5a: OpenAI-compatible endpoints
echo ""
echo "--- OpenAI-Compatible API ---"
test_endpoint "List models" "GET" "$AILB_URL/v1/models" "" "200"

# Chat completions (will fail without upstream, but should return proper error format)
CHAT_BODY='{"model":"test","messages":[{"role":"user","content":"hello"}]}'
# Test with auth header
test_endpoint "Chat completions (no auth)" "POST" "$AILB_URL/v1/chat/completions" "$CHAT_BODY" "401"

# Test 5b: Ollama-compatible endpoints
echo ""
echo "--- Ollama-Compatible API ---"
test_endpoint "Ollama tags" "GET" "$AILB_URL/api/tags" "" "200"

# Test 5c: X-Model-Selector bypass
echo ""
echo "--- X-Model-Selector ---"
# With selector header (should bypass routing)
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST \
    -H "Content-Type: application/json" \
    -H "X-Model-Selector: Ollama:llama3.1:8b" \
    -H "Authorization: Bearer test-token" \
    -d "$CHAT_BODY" \
    "$AILB_URL/v1/chat/completions")
echo "  X-Model-Selector bypass: HTTP $STATUS"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
