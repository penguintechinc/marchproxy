# MarchProxy L7 Proxy - Usage Guide

## Overview

The MarchProxy L7 Proxy is an Envoy-based application load balancer with advanced features including authentication, licensing, rate limiting, and xDP acceleration. This guide covers operational usage and common scenarios.

## Quick Start

### Prerequisites
- Docker installed (version 20.10+)
- Access to MarchProxy API Server
- Network access to backend services

### Basic Setup

1. **Start the Proxy**
```bash
docker run -d \
  --name proxy-l7 \
  -p 10000:10000 \
  -p 9901:9901 \
  --cap-add=NET_ADMIN \
  -e XDS_SERVER=api-server:18000 \
  -e CLUSTER_API_KEY=your-cluster-key \
  marchproxy/proxy-l7:latest
```

2. **Verify Health**
```bash
curl http://localhost:9901/ready
# Expected: HTTP 200 OK
```

3. **Configure Routes**
```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1",
    "services": [{
      "name": "my-backend",
      "hosts": ["backend.internal"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "my-route",
      "prefix": "/",
      "cluster_name": "my-backend",
      "hosts": ["*"],
      "timeout": 30
    }]
  }'
```

4. **Test Traffic**
```bash
curl http://localhost:10000/
```

## Common Usage Scenarios

### Scenario 1: HTTP Service Routing

Route HTTP traffic to a backend service.

```bash
# Create configuration
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1",
    "services": [{
      "name": "api-backend",
      "hosts": ["api1.internal", "api2.internal"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "api-route",
      "prefix": "/api",
      "cluster_name": "api-backend",
      "hosts": ["api.example.com"],
      "timeout": 30
    }]
  }'

# Test from client
curl -H "Host: api.example.com" http://localhost:10000/api/users
```

### Scenario 2: HTTPS/TLS Termination

Route HTTPS traffic with TLS termination at the proxy.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "2",
    "services": [{
      "name": "secure-backend",
      "hosts": ["backend.internal"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "https-route",
      "prefix": "/",
      "cluster_name": "secure-backend",
      "hosts": ["secure.example.com"],
      "require_tls": true,
      "timeout": 30
    }]
  }'

# Test
curl -k https://localhost:10443/
```

### Scenario 3: Multiple Backends with Load Balancing

Route traffic to multiple backend instances with automatic load balancing.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "3",
    "services": [
      {
        "name": "web-backend",
        "hosts": ["web1.internal", "web2.internal", "web3.internal"],
        "port": 80,
        "protocol": "http"
      }
    ],
    "routes": [{
      "name": "web-route",
      "prefix": "/",
      "cluster_name": "web-backend",
      "hosts": ["www.example.com"],
      "load_balancing_policy": "round_robin",
      "timeout": 30
    }]
  }'

# Monitor load distribution
curl http://localhost:9901/stats | grep -E 'upstream_rq|upstream_cx'
```

### Scenario 4: Authentication with JWT

Require JWT token authentication for specific routes.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "4",
    "auth_config": {
      "jwt_secret": "your-secret-key",
      "jwt_algorithm": "HS256",
      "require_auth": true,
      "exempt_paths": ["/public/*", "/healthz", "/metrics"]
    },
    "services": [{
      "name": "api-backend",
      "hosts": ["api.internal"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "protected-route",
      "prefix": "/api",
      "cluster_name": "api-backend",
      "hosts": ["api.example.com"],
      "require_auth": true,
      "timeout": 30
    }]
  }'

# Test without token (returns 401)
curl -i http://localhost:10000/api/users
# Expected: HTTP 401 Unauthorized

# Test with valid JWT token
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
curl -i -H "Authorization: Bearer $TOKEN" http://localhost:10000/api/users
# Expected: HTTP 200 OK
```

### Scenario 5: Rate Limiting

Configure rate limiting per source IP or globally.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "5",
    "rate_limiting": {
      "enabled": true,
      "max_requests_per_second": 100,
      "window_seconds": 1
    },
    "services": [{
      "name": "public-api",
      "hosts": ["api.internal"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "public-route",
      "prefix": "/",
      "cluster_name": "public-api",
      "hosts": ["*"],
      "rate_limit": true,
      "timeout": 30
    }]
  }'

# Test rate limiting
ab -n 1000 -c 100 http://localhost:10000/
```

### Scenario 6: WebSocket Proxying

Enable WebSocket upgrade and connection proxying.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "6",
    "services": [{
      "name": "websocket-backend",
      "hosts": ["ws.internal"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "ws-route",
      "prefix": "/ws",
      "cluster_name": "websocket-backend",
      "hosts": ["ws.example.com"],
      "websocket_enabled": true,
      "timeout": 300
    }]
  }'

# Test WebSocket connection
wscat -c ws://localhost:10000/ws
```

### Scenario 7: gRPC Routing

Route gRPC traffic with protocol-aware handling.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "7",
    "services": [{
      "name": "grpc-backend",
      "hosts": ["grpc.internal"],
      "port": 50051,
      "protocol": "http2"
    }],
    "routes": [{
      "name": "grpc-route",
      "prefix": "/",
      "cluster_name": "grpc-backend",
      "hosts": ["*"],
      "protocol": "grpc",
      "timeout": 60
    }]
  }'

# Test gRPC
grpcurl -plaintext localhost:10000 service.Service/Method
```

### Scenario 8: Header-Based Routing

Route requests based on HTTP headers.

```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "8",
    "services": [
      {
        "name": "mobile-api",
        "hosts": ["api-mobile.internal"],
        "port": 8080,
        "protocol": "http"
      },
      {
        "name": "web-api",
        "hosts": ["api-web.internal"],
        "port": 8080,
        "protocol": "http"
      }
    ],
    "routes": [
      {
        "name": "mobile-route",
        "prefix": "/api",
        "cluster_name": "mobile-api",
        "hosts": ["api.example.com"],
        "match_headers": [
          {"name": "X-Client-Type", "value": "mobile"}
        ],
        "timeout": 30
      },
      {
        "name": "web-route",
        "prefix": "/api",
        "cluster_name": "web-api",
        "hosts": ["api.example.com"],
        "timeout": 30
      }
    ]
  }'

# Test header-based routing
curl -H "X-Client-Type: mobile" http://localhost:10000/api/users
# Routed to mobile-api

curl http://localhost:10000/api/users
# Routed to web-api
```

## Monitoring and Troubleshooting

### Check Proxy Status
```bash
# Basic health check
curl http://localhost:9901/ready

# Detailed server info
curl http://localhost:9901/server_info | jq .

# Configuration status
curl http://localhost:9901/config_dump | jq '.configs[0].timestamp'
```

### View Metrics

```bash
# All metrics
curl http://localhost:9901/stats/prometheus

# HTTP metrics only
curl http://localhost:9901/stats/prometheus | grep envoy_http

# Upstream metrics
curl http://localhost:9901/stats/prometheus | grep upstream

# Custom filter metrics
curl http://localhost:9901/stats/prometheus | grep wasm
```

### Monitor Specific Route
```bash
# View route stats
curl http://localhost:9901/stats | grep route

# View cluster stats
curl http://localhost:9901/stats | grep "cluster.my-backend"
```

### Troubleshooting: Route Not Working

```bash
# 1. Check if route configuration is loaded
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Route"))'

# 2. Check upstream cluster status
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Cluster"))'

# 3. Check endpoint health
curl http://localhost:9901/stats | grep "cluster.my-cluster.membership"

# 4. View request logs
docker logs proxy-l7 --follow

# 5. Enable debug logging
docker exec proxy-l7 curl -X POST http://localhost:9901/logging?config=debug
```

### Troubleshooting: High Latency

```bash
# 1. Check response times
curl http://localhost:9901/stats/prometheus | grep 'envoy_http_ingress_http_request_duration'

# 2. Check upstream latency
curl http://localhost:9901/stats | grep upstream_rq_time

# 3. Check connection pooling
curl http://localhost:9901/stats | grep "cluster.*.cx_pool"

# 4. Monitor CPU usage
docker stats proxy-l7

# 5. Check for errors
curl http://localhost:9901/stats | grep -E 'upstream_rq.*error|upstream_rq.*5xx'
```

### Troubleshooting: Authentication Issues

```bash
# 1. Check auth filter is loaded
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("HttpConnectionManager"))' | grep -i auth

# 2. View auth filter stats
curl http://localhost:9901/stats/prometheus | grep wasm_auth

# 3. Check token validation
# Generate valid token for debugging
python3 -c "
import jwt
token = jwt.encode({'sub': 'test'}, 'your-secret-key', algorithm='HS256')
print(f'Authorization: Bearer {token}')
"

# 4. Test with token
curl -H 'Authorization: Bearer <token>' http://localhost:10000/api
```

## Performance Tuning

### Enable XDP for Best Performance

```bash
# Check if NET_ADMIN capability is available
docker inspect proxy-l7 | grep -A 20 '"CapAdd"'

# Use native XDP mode
-e XDP_MODE=native

# Use hardware offload if supported (requires compatible NIC)
-e XDP_MODE=hw
```

### Optimize Connection Limits

```bash
# Increase max connections in docker-compose.yml
environment:
  - MAX_CONNECTIONS=100000
```

### Monitor Performance

```bash
# Watch in real-time
watch -n 1 'curl -s http://localhost:9901/stats/prometheus | grep envoy_http_ingress_http_requests_total'

# Detailed statistics
curl http://localhost:9901/stats | grep -E 'envoy_http|upstream' | head -20
```

## Configuration Updates

### Live Configuration Update

Configuration updates are applied without restarting the proxy:

```bash
# Update configuration
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "9",
    ...
  }'

# Verify update (may take a few seconds)
sleep 2
curl http://localhost:9901/config_dump | jq '.configs[0].timestamp'
```

### Configuration Rollback

If issues occur after configuration update:

```bash
# Get previous configuration version number
curl http://api-server:19000/v1/version | jq '.previous_version'

# Restore previous configuration
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "8",
    ...
  }'
```

## Container Management

### View Logs
```bash
docker logs proxy-l7 --follow
docker logs proxy-l7 --tail 100  # Last 100 lines
```

### Graceful Shutdown
```bash
# Initiate graceful drain (stop accepting new connections)
curl -X POST http://localhost:9901/drain_listeners?inboundonly

# Monitor draining
watch -n 1 'curl -s http://localhost:9901/stats | grep listener'

# Stop container (waits for drain to complete)
docker stop proxy-l7
```

### Force Restart
```bash
docker restart proxy-l7

# Verify health after restart
sleep 5
curl http://localhost:9901/ready
```

## Advanced Usage

### Path-Based Routing
```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "10",
    "routes": [
      {
        "name": "api-route",
        "prefix": "/api/v1",
        "cluster_name": "api-v1"
      },
      {
        "name": "api-v2-route",
        "prefix": "/api/v2",
        "cluster_name": "api-v2"
      }
    ]
  }'
```

### Canary Deployments
```bash
# Route 90% to stable, 10% to canary
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "11",
    "routes": [{
      "name": "canary-route",
      "prefix": "/",
      "weighted_clusters": [
        {"cluster_name": "stable", "weight": 90},
        {"cluster_name": "canary", "weight": 10}
      ]
    }]
  }'
```

### Request/Response Modification

Headers can be added/removed:
```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "12",
    "routes": [{
      "name": "modified-route",
      "prefix": "/",
      "cluster_name": "backend",
      "request_headers": {
        "add": {"X-Proxy": "marchproxy"},
        "remove": ["X-Internal"]
      }
    }]
  }'
```

## Related Documentation

- [API.md](./API.md) - Complete API reference
- [CONFIGURATION.md](./CONFIGURATION.md) - Configuration details
- [TESTING.md](./TESTING.md) - Testing procedures
- [README.md](../README.md) - Project overview
