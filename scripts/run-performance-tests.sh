#!/bin/bash
# Run performance tests for MarchProxy

set -e

echo "========================================="
echo "MarchProxy - Performance Tests"
echo "========================================="

cd "$(dirname "$0")/.."

# Run Locust load tests
echo "Starting Locust load tests..."
echo "Open http://localhost:8089 to view dashboard"

cd tests/performance
locust -f locustfile.py --host=http://localhost:8000 --html=../../reports/locust-report.html

echo "Performance tests completed!"
echo "Report: reports/locust-report.html"
