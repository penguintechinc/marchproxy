#!/bin/bash
# gRPC integration tests for WaddleAI service
# Requires: WaddleAI gRPC server on localhost:50051, grpcurl installed

set -e

WADDLEAI_GRPC="${WADDLEAI_GRPC:-localhost:50051}"
PASS=0
FAIL=0

if ! command -v grpcurl &>/dev/null; then
    echo "grpcurl not found — skipping gRPC tests"
    echo "Install: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest"
    exit 0
fi

test_rpc() {
    local name="$1"
    local service="$2"
    local method="$3"
    local data="$4"

    if grpcurl -plaintext -d "$data" "$WADDLEAI_GRPC" "$service/$method" &>/dev/null; then
        echo "  ✓ $name"
        PASS=$((PASS + 1))
    else
        echo "  ✗ $name"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== WaddleAI gRPC Tests ==="
echo ""

# EvaluateRoute
test_rpc "EvaluateRoute (bash, low)" \
    "marchproxy.WaddleAIService" "EvaluateRoute" \
    '{"prompt":"ls -la","tool_type":"bash","region":"NA"}'

# EvaluateSecurity (safe command)
test_rpc "EvaluateSecurity (safe)" \
    "marchproxy.WaddleAIService" "EvaluateSecurity" \
    '{"raw_command":"echo hello","tool_type":"bash"}'

# EvaluateSecurity (dangerous command)
test_rpc "EvaluateSecurity (dangerous)" \
    "marchproxy.WaddleAIService" "EvaluateSecurity" \
    '{"raw_command":"rm -rf /","tool_type":"bash"}'

# ReportUsage
test_rpc "ReportUsage" \
    "marchproxy.WaddleAIService" "ReportUsage" \
    '{"user_id":"test-user","model":"llama3.1:8b","provider":"ollama","input_tokens":100,"output_tokens":50,"total_tokens":150}'

# StoreTurn
test_rpc "StoreTurn" \
    "marchproxy.WaddleAIService" "StoreTurn" \
    '{"session_id":"test-session","user_id":"test-user","user_message":"hello","assistant_response":"hi there","model":"llama3.1:8b","provider":"ollama"}'

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
