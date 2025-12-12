#!/bin/bash
# MarchProxy API Server - Phase 2 Test Script
#
# This script tests the Phase 2 API endpoints
# Run after starting the API server: uvicorn app.main:app --reload

set -e

API_URL="http://localhost:8000/api/v1"
TOKEN=""

echo "=== MarchProxy API Server Phase 2 Test ==="
echo ""

# Test 1: Health Check
echo "[1] Testing health check..."
curl -s "$API_URL/../healthz" | jq '.'
echo ""

# Test 2: Register first user (becomes admin)
echo "[2] Registering first user (admin)..."
REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@marchproxy.local",
    "password": "SuperSecret123!"
  }')
echo "$REGISTER_RESPONSE" | jq '.'
TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.access_token')
echo "Access Token: ${TOKEN:0:20}..."
echo ""

# Test 3: Get current user info
echo "[3] Getting current user info..."
curl -s -X GET "$API_URL/auth/me" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Test 4: Create a cluster
echo "[4] Creating a cluster..."
CLUSTER_RESPONSE=$(curl -s -X POST "$API_URL/clusters" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production",
    "description": "Production cluster",
    "syslog_endpoint": "syslog.example.com:514",
    "log_auth": true,
    "log_netflow": true,
    "log_debug": false,
    "max_proxies": 10
  }')
echo "$CLUSTER_RESPONSE" | jq '.'
CLUSTER_ID=$(echo "$CLUSTER_RESPONSE" | jq -r '.id')
CLUSTER_API_KEY=$(echo "$CLUSTER_RESPONSE" | jq -r '.api_key // empty')
echo "Cluster ID: $CLUSTER_ID"
if [ -n "$CLUSTER_API_KEY" ]; then
  echo "Cluster API Key: ${CLUSTER_API_KEY:0:20}... (SAVE THIS!)"
fi
echo ""

# Test 5: List clusters
echo "[5] Listing clusters..."
curl -s -X GET "$API_URL/clusters" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Test 6: Create a service
echo "[6] Creating a service with JWT auth..."
SERVICE_RESPONSE=$(curl -s -X POST "$API_URL/services" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-api",
    "ip_fqdn": "api.example.com",
    "port": 443,
    "protocol": "https",
    "collection": "web-services",
    "cluster_id": '$CLUSTER_ID',
    "auth_type": "jwt",
    "tls_enabled": true,
    "health_check_enabled": true,
    "health_check_path": "/health"
  }')
echo "$SERVICE_RESPONSE" | jq '.'
SERVICE_ID=$(echo "$SERVICE_RESPONSE" | jq -r '.id')
echo "Service ID: $SERVICE_ID"
echo ""

# Test 7: List services
echo "[7] Listing services..."
curl -s -X GET "$API_URL/services?cluster_id=$CLUSTER_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

# Test 8: Register a proxy (if cluster API key available)
if [ -n "$CLUSTER_API_KEY" ]; then
  echo "[8] Registering a proxy..."
  PROXY_RESPONSE=$(curl -s -X POST "$API_URL/proxies/register" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "proxy-01",
      "hostname": "proxy-01.example.com",
      "ip_address": "10.0.1.100",
      "port": 8080,
      "version": "1.0.0",
      "capabilities": {"xdp": true, "af_xdp": true},
      "cluster_api_key": "'"$CLUSTER_API_KEY"'"
    }')
  echo "$PROXY_RESPONSE" | jq '.'
  PROXY_ID=$(echo "$PROXY_RESPONSE" | jq -r '.id')
  echo "Proxy ID: $PROXY_ID"
  echo ""

  # Test 9: Proxy heartbeat
  echo "[9] Sending proxy heartbeat..."
  curl -s -X POST "$API_URL/proxies/heartbeat" \
    -H "Content-Type: application/json" \
    -d '{
      "proxy_id": '$PROXY_ID',
      "cluster_api_key": "'"$CLUSTER_API_KEY"'",
      "status": "active",
      "config_version": "abc123",
      "metrics": {
        "cpu_usage": 25.5,
        "memory_usage": 45.2,
        "connections_active": 150
      }
    }' | jq '.'
  echo ""

  # Test 10: List proxies
  echo "[10] Listing proxies..."
  curl -s -X GET "$API_URL/proxies?cluster_id=$CLUSTER_ID" \
    -H "Authorization: Bearer $TOKEN" | jq '.'
  echo ""
else
  echo "[8-10] Skipped (no cluster API key returned)"
  echo ""
fi

# Test 11: Create a regular user
echo "[11] Creating a regular user..."
curl -s -X POST "$API_URL/users" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "service_owner",
    "first_name": "Service",
    "last_name": "Owner",
    "password": "Password123!",
    "is_admin": false,
    "is_active": true
  }' | jq '.'
echo ""

# Test 12: List users
echo "[12] Listing users..."
curl -s -X GET "$API_URL/users" \
  -H "Authorization: Bearer $TOKEN" | jq '.'
echo ""

echo "=== All Tests Complete ==="
echo ""
echo "Documentation: http://localhost:8000/api/docs"
echo "Next steps: Build docker container and test with database"
