#!/bin/bash
# Alpha Smoke Tests - Master Runner
# Runs all alpha smoke tests in sequence
# These tests MUST pass before committing code

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Setup logging
TIMESTAMP=$(date +%s)
LOG_DIR="/tmp/marchproxy-smoke-alpha-$TIMESTAMP"
mkdir -p "$LOG_DIR"
SUMMARY_LOG="$LOG_DIR/summary.log"

echo "=========================================="
echo "MarchProxy Alpha Smoke Tests"
echo "=========================================="
echo ""
echo "Full end-to-end testing before commit"
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
run_test "$SCRIPT_DIR/01-build-containers.sh" "Build All Containers"
run_test "$SCRIPT_DIR/02-start-services.sh" "Start All Services"
run_test "$SCRIPT_DIR/03-api-health-checks.sh" "API Health Checks"
run_test "$SCRIPT_DIR/04-webui-page-loads.sh" "WebUI Page Loads"
run_test "$SCRIPT_DIR/05-security-checks.sh" "Security & Vulnerability Checks"
run_test "$SCRIPT_DIR/06-linters.sh" "Code Linters"

# Cleanup - stop services
echo "Cleaning up services..."
cd "$PROJECT_ROOT"
docker-compose -f docker-compose.yml down -v 2>/dev/null || true

# Summary
echo "" | tee -a "$SUMMARY_LOG"
echo "==========================================" | tee -a "$SUMMARY_LOG"
echo "Alpha Smoke Test Summary" | tee -a "$SUMMARY_LOG"
echo "==========================================" | tee -a "$SUMMARY_LOG"
echo "Total tests:  $TOTAL_TESTS" | tee -a "$SUMMARY_LOG"
echo "Passed:       $PASSED_TESTS" | tee -a "$SUMMARY_LOG"
echo "Failed:       $FAILED_TESTS" | tee -a "$SUMMARY_LOG"
echo "" | tee -a "$SUMMARY_LOG"
echo "Logs saved to: $LOG_DIR" | tee -a "$SUMMARY_LOG"
echo "Summary: $SUMMARY_LOG" | tee -a "$SUMMARY_LOG"
echo "" | tee -a "$SUMMARY_LOG"

if [ $FAILED_TESTS -eq 0 ]; then
    echo "✅ ALL ALPHA SMOKE TESTS PASSED" | tee -a "$SUMMARY_LOG"
    echo "   Safe to commit code" | tee -a "$SUMMARY_LOG"
    exit 0
else
    echo "❌ SOME TESTS FAILED" | tee -a "$SUMMARY_LOG"
    echo "   DO NOT COMMIT until all tests pass" | tee -a "$SUMMARY_LOG"
    exit 1
fi
