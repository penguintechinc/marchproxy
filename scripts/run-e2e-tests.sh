#!/bin/bash
# Run end-to-end tests for MarchProxy

set -e

echo "========================================="
echo "MarchProxy - End-to-End Tests"
echo "========================================="

cd "$(dirname "$0")/.."

# Start Docker services
echo "Starting Docker services..."
docker-compose -f docker-compose.test.yml up -d

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 30

# Run E2E tests
echo "Running E2E tests..."
pytest tests/e2e/ -v -m e2e --html=reports/e2e-report.html

# Cleanup
echo "Stopping Docker services..."
docker-compose -f docker-compose.test.yml down -v

echo "E2E tests completed!"
