# MarchProxy L7 Proxy Testing Guide

## Overview

This guide covers unit testing, integration testing, and functional testing for the MarchProxy L7 proxy components including XDP programs, WASM filters, and Envoy configuration.

## Prerequisites

```bash
# Docker (for containerized testing)
docker --version

# For local testing (optional)
# Rust
rustup --version
cargo --version

# Go
go version

# Testing tools
curl --version
nc (netcat)
```

## Running Tests with Docker

### Build Test Image
```bash
cd /home/penguin/code/MarchProxy/proxy-l7
docker build -f envoy/Dockerfile -t marchproxy/proxy-l7:test .
```

### Basic Container Test
```bash
# Start proxy
docker run -d --name test-proxy \
  -p 10000:10000 \
  -p 9901:9901 \
  --cap-add=NET_ADMIN \
  -e XDS_SERVER=api-server:18000 \
  -e CLUSTER_API_KEY=test-key \
  marchproxy/proxy-l7:test

# Health check
curl -i http://localhost:9901/ready

# Stop container
docker stop test-proxy && docker rm test-proxy
```

## Unit Testing

### WASM Filter Tests

#### Auth Filter Tests
```bash
cd /home/penguin/code/MarchProxy/proxy-l7/filters/auth_filter

# Build for testing
cargo build --target wasm32-unknown-unknown --release

# Run unit tests
cargo test --lib

# Expected output:
# running 5 tests
# test auth::tests::test_jwt_validation ... ok
# test auth::tests::test_base64_validation ... ok
# test auth::tests::test_exempt_paths ... ok
# test auth::tests::test_missing_token ... ok
# test auth::tests::test_token_rotation ... ok
#
# test result: ok. 5 passed; 0 failed; 0 ignored
```

#### License Filter Tests
```bash
cd /home/penguin/code/MarchProxy/proxy-l7/filters/license_filter

# Run unit tests
cargo test --lib

# Expected output:
# running 4 tests
# test license::tests::test_feature_gating ... ok
# test license::tests::test_proxy_count_limit ... ok
# test license::tests::test_license_validation ... ok
# test license::tests::test_enterprise_features ... ok
#
# test result: ok. 4 passed; 0 failed; 0 ignored
```

#### Metrics Filter Tests
```bash
cd /home/penguin/code/MarchProxy/proxy-l7/filters/metrics_filter

# Run unit tests
cargo test --lib

# Expected output:
# running 3 tests
# test metrics::tests::test_request_counting ... ok
# test metrics::tests::test_latency_tracking ... ok
# test metrics::tests::test_histogram_buckets ... ok
#
# test result: ok. 3 passed; 0 failed; 0 ignored
```

### XDP Program Tests

```bash
cd /home/penguin/code/MarchProxy/proxy-l7/xdp

# Compile with debug symbols
make

# Verify compilation
llvm-objdump -d envoy_xdp.o | head -20

# Expected: Assembly output showing BPF instructions
```

## Integration Testing

### Test with Docker Compose

Create `docker-compose.test.yml`:
```yaml
version: '3.8'

services:
  api-server:
    image: marchproxy/api-server:test
    environment:
      - DATABASE_URL=postgresql://test:test@postgres:5432/marchproxy_test
      - XDS_GRPC_PORT=18000
      - XDS_HTTP_PORT=19000
    ports:
      - "8000:8000"
      - "18000:18000"
      - "19000:19000"
    depends_on:
      - postgres

  postgres:
    image: postgres:15-bookworm
    environment:
      - POSTGRES_USER=test
      - POSTGRES_PASSWORD=test
      - POSTGRES_DB=marchproxy_test
    ports:
      - "5432:5432"

  backend:
    image: hashicorp/http-echo
    command: -listen=:8080 -text='Hello from backend'
    ports:
      - "8080:8080"

  proxy-l7:
    image: marchproxy/proxy-l7:test
    environment:
      - XDS_SERVER=api-server:18000
      - CLUSTER_API_KEY=test-key
      - XDP_MODE=skb  # Use SKB mode for testing
      - LOGLEVEL=debug
    ports:
      - "10000:10000"
      - "9901:9901"
    cap_add:
      - NET_ADMIN
    depends_on:
      - api-server
```

### Run Integration Tests
```bash
# Start services
docker-compose -f docker-compose.test.yml up -d

# Wait for services to be ready
sleep 10

# Test connectivity to all services
curl -i http://localhost:8000/healthz     # API Server
curl -i http://localhost:9901/ready       # Proxy L7 Admin
curl -i http://localhost:8080/            # Backend

# Check xDS configuration
curl -s http://localhost:19000/v1/version

# Stop services
docker-compose -f docker-compose.test.yml down
```

## Functional Testing

### Test Case 1: Basic HTTP Routing

```bash
# Setup
docker-compose -f docker-compose.test.yml up -d
sleep 10

# Create backend service via API
curl -X POST http://localhost:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1",
    "services": [{
      "name": "test-backend",
      "hosts": ["backend"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "test-route",
      "prefix": "/",
      "cluster_name": "test-backend",
      "hosts": ["*"],
      "timeout": 30
    }]
  }'

# Wait for xDS update
sleep 2

# Test request
curl -i http://localhost:10000/

# Expected: 200 OK with "Hello from backend"

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Test Case 2: Authentication Filter

```bash
# Setup
docker-compose -f docker-compose.test.yml up -d
sleep 10

# Create authenticated route
curl -X POST http://localhost:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "2",
    "services": [{
      "name": "test-backend",
      "hosts": ["backend"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "authenticated-route",
      "prefix": "/api",
      "cluster_name": "test-backend",
      "hosts": ["*"],
      "require_auth": true,
      "timeout": 30
    }]
  }'

# Wait for update
sleep 2

# Test without token (should fail)
RESULT=$(curl -i http://localhost:10000/api/test 2>&1)
if echo "$RESULT" | grep -q "401\|403"; then
  echo "✓ Auth filter correctly rejected unauthenticated request"
else
  echo "✗ Auth filter failed to reject request"
fi

# Test with valid JWT token
JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.4Adcj0P0EYzIVunMBEy8wAXHH0TXYzS0DtJv4cJrAH4"
RESULT=$(curl -i -H "Authorization: Bearer $JWT_TOKEN" http://localhost:10000/api/test 2>&1)
if echo "$RESULT" | grep -q "200"; then
  echo "✓ Auth filter correctly accepted authenticated request"
else
  echo "✗ Auth filter failed to accept valid token"
fi

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Test Case 3: Rate Limiting via XDP

```bash
# Setup
docker-compose -f docker-compose.test.yml up -d
sleep 10

# Create configuration
curl -X POST http://localhost:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "3",
    "services": [{
      "name": "test-backend",
      "hosts": ["backend"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "test-route",
      "prefix": "/",
      "cluster_name": "test-backend",
      "hosts": ["*"],
      "timeout": 30
    }]
  }'

# Wait for update
sleep 2

# Generate burst of requests (ab = ApacheBench)
ab -n 1000 -c 100 http://localhost:10000/ > /tmp/ab.log

# Check for rate limit responses
if grep -q "502\|503" /tmp/ab.log; then
  echo "✓ Rate limiting active"
else
  echo "⚠ No rate limit detected (expected in test mode)"
fi

# View XDP statistics
docker exec $(docker-compose -f docker-compose.test.yml ps -q proxy-l7) \
  bpftool map dump name stats_map 2>/dev/null || echo "XDP stats not available in SKB mode"

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Test Case 4: Graceful Drain

```bash
# Setup
docker-compose -f docker-compose.test.yml up -d
sleep 10

# Start background traffic
while true; do
  curl -s http://localhost:10000/ >/dev/null 2>&1
done &
BG_PID=$!

# Initiate graceful drain
curl -X POST http://localhost:9901/drain_listeners?inboundonly

# Wait for connections to drain
sleep 5

# Stop background traffic
kill $BG_PID 2>/dev/null || true

# Verify no new connections accepted
START=$(date +%s)
curl -i http://localhost:10000/ &
sleep 2
if ! curl -i http://localhost:10000/ 2>&1 | grep -q "200"; then
  echo "✓ Proxy correctly rejected new connections during drain"
fi

# Cleanup
docker-compose -f docker-compose.test.yml down
```

## Performance Testing

### Load Test with ApacheBench
```bash
# Start proxy
docker-compose -f docker-compose.test.yml up -d
sleep 10

# Configure backend
curl -X POST http://localhost:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1",
    "services": [{
      "name": "test-backend",
      "hosts": ["backend"],
      "port": 8080,
      "protocol": "http"
    }],
    "routes": [{
      "name": "test-route",
      "prefix": "/",
      "cluster_name": "test-backend",
      "hosts": ["*"],
      "timeout": 30
    }]
  }'

sleep 2

# Run load test
ab -n 10000 -c 100 http://localhost:10000/

# Expected output shows:
# - Requests per second
# - Mean response time
# - Transfer rate
# - Concurrency level

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Load Test with wrk
```bash
# If wrk is installed
wrk -t12 -c400 -d30s http://localhost:10000/
```

## Debugging Tests

### View Proxy Logs
```bash
docker logs proxy-l7 --follow
```

### Monitor Admin Interface
```bash
# In one terminal
watch -n 1 'curl -s http://localhost:9901/stats/prometheus | grep envoy_http'

# In another terminal
curl -n 1000 http://localhost:10000/
```

### Check XDS Configuration
```bash
curl -s http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Listener"))'
```

### Verify WASM Filters Loaded
```bash
curl -s http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("HttpConnectionManager"))' | grep -i wasm
```

## Test Coverage

Required test coverage for components:
- WASM Filters: 80%+ coverage
- XDP Program: Logic verification via llvm-objdump
- Integration: All major code paths tested

## Continuous Integration

Tests are automatically run in CI/CD pipeline:
```bash
# Local CI simulation
./scripts/run-tests.sh  # Runs all test suites
```

## Troubleshooting Tests

### Port Already in Use
```bash
# Kill existing containers
docker-compose -f docker-compose.test.yml down -v
docker ps -a | grep test | awk '{print $1}' | xargs -r docker rm -f
```

### Slow Tests
```bash
# Increase timeouts in docker-compose.test.yml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:9901/ready"]
  interval: 5s
  timeout: 10s
  retries: 30
```

### XDP Not Available
```bash
# Use SKB mode instead of native
# In docker-compose.test.yml:
environment:
  - XDP_MODE=skb
```

## Related Documentation

- [API.md](./API.md) - Admin API reference
- [CONFIGURATION.md](./CONFIGURATION.md) - Configuration guide
- [USAGE.md](./USAGE.md) - Operational guide
