#!/bin/bash
# Beta Smoke Tests - Master Runner
# Tests against staging K8s cluster at https://marchproxy.penguintech.io
# These tests verify the deployed application in the staging environment

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Setup logging
TIMESTAMP=$(date +%s)
LOG_DIR="/tmp/marchproxy-smoke-beta-$TIMESTAMP"
mkdir -p "$LOG_DIR"
SUMMARY_LOG="$LOG_DIR/summary.log"

echo "=========================================="
echo "MarchProxy Beta Smoke Tests"
echo "=========================================="
echo ""
echo "Testing staging cluster at https://marchproxy.penguintech.io"
echo "Results will be logged to: $LOG_DIR"
echo ""

# Track results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run a test
run_test() {
    local test_script=$1
    local test_name=$2

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    echo "==========================================  " | tee -a "$SUMMARY_LOG"
    echo "Running: $test_name" | tee -a "$SUMMARY_LOG"
    echo "==========================================  " | tee -a "$SUMMARY_LOG"
    echo ""

    local test_log="$LOG_DIR/$(basename $test_script .sh).log"

    if bash "$test_script" 2>&1 | tee "$test_log"; then
        echo "" | tee -a "$SUMMARY_LOG"
        echo "✅ PASSED: $test_name" | tee -a "$SUMMARY_LOG"
        echo "" | tee -a "$SUMMARY_LOG"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo "" | tee -a "$SUMMARY_LOG"
        echo "❌ FAILED: $test_name" | tee -a "$SUMMARY_LOG"
        echo "   See: $test_log" | tee -a "$SUMMARY_LOG"
        echo "" | tee -a "$SUMMARY_LOG"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Run all tests
run_test "$SCRIPT_DIR/01-k8s-status.sh" "Kubernetes Cluster Status"
run_test "$SCRIPT_DIR/02-staging-api-health.sh" "Staging API Health Checks"
run_test "$SCRIPT_DIR/03-staging-webui.sh" "Staging WebUI Check"

# Summary
echo "" | tee -a "$SUMMARY_LOG"
echo "==========================================" | tee -a "$SUMMARY_LOG"
echo "Beta Smoke Test Summary" | tee -a "$SUMMARY_LOG"
echo "==========================================" | tee -a "$SUMMARY_LOG"
echo "Total tests:  $TOTAL_TESTS" | tee -a "$SUMMARY_LOG"
echo "Passed:       $PASSED_TESTS" | tee -a "$SUMMARY_LOG"
echo "Failed:       $FAILED_TESTS" | tee -a "$SUMMARY_LOG"
echo "" | tee -a "$SUMMARY_LOG"
echo "Logs saved to: $LOG_DIR" | tee -a "$SUMMARY_LOG"
echo "Summary: $SUMMARY_LOG" | tee -a "$SUMMARY_LOG"
echo "" | tee -a "$SUMMARY_LOG"

if [ $FAILED_TESTS -eq 0 ]; then
    echo "✅ ALL BETA SMOKE TESTS PASSED" | tee -a "$SUMMARY_LOG"
    echo "   Staging environment is healthy" | tee -a "$SUMMARY_LOG"
    exit 0
else
    echo "❌ SOME TESTS FAILED" | tee -a "$SUMMARY_LOG"
    echo "   Staging environment has issues" | tee -a "$SUMMARY_LOG"
    exit 1
fi
