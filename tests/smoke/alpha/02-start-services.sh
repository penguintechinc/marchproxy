#!/bin/bash
# Alpha Smoke Test 2: Start all services via docker-compose
# Verifies containers start and remain healthy

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "=========================================="
echo "Alpha Smoke Test 2: Start All Services"
echo "=========================================="
echo ""

cd "$PROJECT_ROOT"

# Cleanup any existing test containers
echo "Cleaning up existing containers..."
docker-compose -f docker-compose.yml down -v 2>/dev/null || true

# Start services
echo "Starting services with docker-compose..."
if docker-compose -f docker-compose.yml up -d; then
    echo "✅ Docker-compose up successful"
else
    echo "❌ Docker-compose up failed"
    exit 1
fi

# Wait for services to start
echo ""
echo "Waiting for services to stabilize (30 seconds)..."
sleep 30

# Check container status
echo ""
echo "Checking container status..."
FAILED=0

CONTAINERS=$(docker-compose -f docker-compose.yml ps --services)

for SERVICE in $CONTAINERS; do
    STATUS=$(docker-compose -f docker-compose.yml ps $SERVICE | tail -n +2 | awk '{print $NF}')
    if [[ "$STATUS" == *"Up"* ]] || [[ "$STATUS" == *"healthy"* ]]; then
        echo "✅ $SERVICE is running"
    else
        echo "❌ $SERVICE is not running (status: $STATUS)"
        docker-compose -f docker-compose.yml logs --tail=50 $SERVICE
        FAILED=1
    fi
done

echo ""
if [ $FAILED -eq 0 ]; then
    echo "✅ All services started successfully"
    exit 0
else
    echo "❌ Some services failed to start"
    exit 1
fi
