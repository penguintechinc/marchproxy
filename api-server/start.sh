#!/bin/bash
# MarchProxy API Server Startup Script
# Starts both the FastAPI application and the Go xDS server

set -e

echo "Starting MarchProxy API Server..."

# Start xDS server in background
echo "Starting xDS control plane on port 18000..."
/usr/local/bin/xds-server -port 18000 -metrics 19000 &
XDS_PID=$!

# Wait a moment for xDS server to initialize
sleep 2

# Start FastAPI application
echo "Starting FastAPI application on port 8000..."
uvicorn app.main:app --host 0.0.0.0 --port 8000 &
API_PID=$!

# Function to handle shutdown
shutdown() {
    echo "Shutting down MarchProxy API Server..."
    kill -TERM $API_PID 2>/dev/null || true
    kill -TERM $XDS_PID 2>/dev/null || true
    wait $API_PID 2>/dev/null || true
    wait $XDS_PID 2>/dev/null || true
    exit 0
}

# Trap signals
trap shutdown SIGTERM SIGINT

# Wait for both processes
wait -n

# If either process exits, shut down both
shutdown
