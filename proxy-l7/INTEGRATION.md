# MarchProxy L7 Proxy Integration Guide

## Overview

This guide explains how Phase 4 (Envoy L7 Proxy with WASM + XDP) integrates with Phase 3 (xDS Control Plane) and the overall MarchProxy architecture.

## Architecture Integration

```
┌──────────────────────────────────────────────────────────────────┐
│                     MarchProxy System                             │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌─────────────┐         ┌──────────────┐         ┌───────────┐ │
│  │   WebUI     │◀───────▶│ API Server   │◀───────▶│ Postgres  │ │
│  │ (React)     │  HTTP   │  (FastAPI)   │   SQL   │           │ │
│  │ Port 3000   │         │  Port 8000   │         │ Port 5432 │ │
│  └─────────────┘         └──────────────┘         └───────────┘ │
│                                  │                                │
│                                  │ gRPC                           │
│                                  ▼                                │
│                          ┌──────────────┐                         │
│                          │ xDS Server   │                         │
│                          │   (Go)       │                         │
│                          │ Port 18000   │                         │
│                          │ HTTP: 19000  │                         │
│                          └──────────────┘                         │
│                                  │                                │
│                        ┌─────────┴─────────┐                     │
│                        │ xDS Protocol      │                     │
│                        │ (ADS/LDS/RDS/     │                     │
│                        │  CDS/EDS)         │                     │
│                        └─────────┬─────────┘                     │
│                                  │                                │
│                                  ▼                                │
│                          ┌──────────────┐                         │
│                          │ Proxy L7     │                         │
│                          │  (Envoy)     │                         │
│                          │              │                         │
│                          │ ┌──────────┐ │                         │
│                          │ │   XDP    │ │                         │
│                          │ │ Program  │ │                         │
│                          │ └──────────┘ │                         │
│                          │      ▼       │                         │
│                          │ ┌──────────┐ │                         │
│                          │ │  WASM    │ │                         │
│                          │ │ Filters  │ │                         │
│                          │ └──────────┘ │                         │
│                          │      ▼       │                         │
│                          │ ┌──────────┐ │                         │
│                          │ │  Envoy   │ │                         │
│                          │ │   Core   │ │                         │
│                          │ └──────────┘ │                         │
│                          │              │                         │
│                          │ Port 10000   │                         │
│                          │ Admin: 9901  │                         │
│                          └──────────────┘                         │
│                                                                    │
└──────────────────────────────────────────────────────────────────┘
```

## Component Dependencies

### Startup Order
1. **Postgres** (database)
2. **API Server** (FastAPI + xDS server)
3. **WebUI** (optional, for management)
4. **Proxy L7** (Envoy)

### Communication Flow

#### 1. Configuration Updates
```
User → WebUI → API Server → xDS Server → Envoy (via xDS protocol)
```

**Example Flow:**
1. User creates a new route in WebUI
2. WebUI sends POST request to API Server
3. API Server updates database
4. API Server triggers xDS configuration update
5. xDS Server pushes new snapshot to Envoy
6. Envoy hot-reloads configuration

#### 2. Traffic Processing
```
Client → XDP → WASM Filters → Envoy Core → Backend Service
```

**Processing Steps:**
1. **XDP Layer**:
   - Protocol detection
   - Rate limiting
   - DDoS protection
   - Statistics collection

2. **WASM Filters**:
   - License validation (blocks if license expired)
   - Authentication (JWT/Base64 tokens)
   - Custom metrics collection

3. **Envoy Core**:
   - Routing (based on xDS config)
   - Load balancing
   - Connection pooling
   - Observability

## Phase 3: xDS Control Plane

### Components Created

#### 1. xDS Server (`api-server/xds/`)
```
xds/
├── server.go      # gRPC server implementation
├── snapshot.go    # Configuration snapshot generator
├── api.go         # HTTP API for config updates
├── Dockerfile     # Container build
├── Makefile       # Build automation
└── go.mod         # Go dependencies
```

**Key Features:**
- **gRPC Server**: Implements Envoy xDS protocol (ADS, LDS, RDS, CDS, EDS)
- **Snapshot Cache**: Stores versioned configurations
- **HTTP API**: Accepts config updates from FastAPI
- **Auto-reload**: Pushes updates to connected Envoy instances

**Ports:**
- `18000`: gRPC (xDS protocol)
- `19000`: HTTP (configuration API)

#### 2. Configuration Model

**Input Format** (from API Server):
```json
{
  "version": "1",
  "services": [
    {
      "name": "backend-service",
      "hosts": ["10.0.1.10", "10.0.1.11"],
      "port": 8080,
      "protocol": "http"
    }
  ],
  "routes": [
    {
      "name": "api-route",
      "prefix": "/api",
      "cluster_name": "backend-service",
      "hosts": ["api.example.com"],
      "timeout": 30
    }
  ]
}
```

**Output**: Envoy xDS snapshot (LDS, RDS, CDS, EDS)

### API Endpoints

#### xDS Server HTTP API
```bash
# Update configuration
POST http://localhost:19000/v1/config
Content-Type: application/json

{
  "version": "1",
  "services": [...],
  "routes": [...]
}

# Get current version
GET http://localhost:19000/v1/version

# Health check
GET http://localhost:19000/healthz
```

## Phase 4: Envoy L7 Proxy

### Components Created

#### 1. XDP Program (`proxy-l7/xdp/`)
```c
// Wire-speed packet processing
- Protocol detection (HTTP/HTTPS/HTTP2/gRPC/WebSocket)
- Per-IP rate limiting (configurable via BPF maps)
- DDoS protection
- Statistics collection
```

**Performance**: 40+ Gbps, <1μs per packet

#### 2. WASM Filters (`proxy-l7/filters/`)

**Auth Filter** (`auth_filter/`):
```rust
- JWT validation (HS256/HS384/HS512)
- Base64 token authentication
- Path exemptions
- 401/403 error responses
```

**License Filter** (`license_filter/`):
```rust
- Enterprise feature gating
- Proxy count limits
- License key validation
- 402/429 error responses
```

**Metrics Filter** (`metrics_filter/`):
```rust
- Request/response tracking
- Latency histograms
- Size metrics
- Prometheus format
```

#### 3. Envoy Configuration (`proxy-l7/envoy/`)

**Bootstrap** (`bootstrap.yaml`):
```yaml
# Static configuration
- Admin interface: 9901
- xDS cluster: api-server:18000
- Node ID: marchproxy-node

# Dynamic resources (from xDS)
- Listeners (LDS)
- Routes (RDS)
- Clusters (CDS)
- Endpoints (EDS)
```

## Integration Steps

### 1. API Server Setup

**Environment Variables:**
```bash
# API Server
DATABASE_URL=postgresql://user:pass@postgres:5432/marchproxy
REDIS_URL=redis://:pass@redis:6379/0
SECRET_KEY=your-secret-key
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD

# xDS Server (embedded)
XDS_GRPC_PORT=18000
XDS_HTTP_PORT=19000
```

**Start Command:**
```bash
# Start API server (includes xDS server)
docker run -d \
  --name api-server \
  -p 8000:8000 \
  -p 18000:18000 \
  -p 19000:19000 \
  -e DATABASE_URL=... \
  -e XDS_GRPC_PORT=18000 \
  marchproxy/api-server:latest
```

### 2. Proxy L7 Setup

**Environment Variables:**
```bash
# Required
XDS_SERVER=api-server:18000      # xDS server address
CLUSTER_API_KEY=your-api-key     # Authentication

# Optional
XDP_INTERFACE=eth0               # Network interface for XDP
XDP_MODE=native                  # native, skb, or hw
LOGLEVEL=info                    # debug, info, warn, error
```

**Start Command:**
```bash
docker run -d \
  --name proxy-l7 \
  -p 10000:10000 \
  -p 9901:9901 \
  --cap-add=NET_ADMIN \
  -e XDS_SERVER=api-server:18000 \
  -e CLUSTER_API_KEY=your-api-key \
  -e XDP_INTERFACE=eth0 \
  marchproxy/proxy-l7:latest
```

### 3. Configuration Updates

**Via API Server:**
```python
import requests

# Update routing configuration
config = {
    "version": "2",
    "services": [
        {
            "name": "my-backend",
            "hosts": ["backend1.local", "backend2.local"],
            "port": 8080,
            "protocol": "http"
        }
    ],
    "routes": [
        {
            "name": "api-route",
            "prefix": "/api",
            "cluster_name": "my-backend",
            "hosts": ["*.example.com"],
            "timeout": 30
        }
    ]
}

# Send to xDS server
response = requests.post(
    "http://api-server:19000/v1/config",
    json=config
)

print(response.json())
# Output: {"status": "success", "version": "2", ...}
```

**Verification:**
```bash
# Check Envoy received update
curl http://localhost:9901/config_dump | jq '.configs'

# Check active routes
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Route"))'
```

## Testing

### 1. xDS Connectivity Test

```bash
# From proxy-l7 container
nc -zv api-server 18000

# Check xDS logs
docker logs api-server | grep xDS

# Check Envoy logs
docker logs proxy-l7 | grep xDS
```

### 2. WASM Filter Test

**Auth Filter:**
```bash
# Without token (should return 401)
curl -i http://localhost:10000/api/test

# With valid JWT
TOKEN="eyJ..."
curl -i -H "Authorization: Bearer $TOKEN" http://localhost:10000/api/test

# With Base64 token
TOKEN="base64encodedtoken"
curl -i -H "Authorization: Bearer $TOKEN" http://localhost:10000/api/test
```

**License Filter:**
```bash
# Community feature (should work)
curl http://localhost:10000/api/basic

# Enterprise feature (should return 402 without license)
curl http://localhost:10000/api/v1/multi-cloud
```

### 3. XDP Test

```bash
# Check XDP is loaded
docker exec proxy-l7 ip link show eth0 | grep xdp

# View XDP statistics
docker exec proxy-l7 bpftool map dump name stats_map

# Rate limiting test (exceed 10k req/s)
ab -n 100000 -c 100 http://localhost:10000/
```

### 4. End-to-End Test

```bash
# 1. Create backend service
docker run -d --name backend --network marchproxy-network \
  -e PORT=8080 hashicorp/http-echo -text="Hello from backend"

# 2. Update configuration via API
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

# 3. Test traffic through proxy
curl http://localhost:10000/
# Output: Hello from backend

# 4. Check metrics
curl http://localhost:9901/stats/prometheus | grep marchproxy
```

## Performance Tuning

### XDP Optimization
```bash
# Use native mode for best performance
-e XDP_MODE=native

# Hardware offload if supported
-e XDP_MODE=hw

# Multiple RX queues
ethtool -L eth0 combined 8
```

### Envoy Optimization
```yaml
# In bootstrap.yaml
layered_runtime:
  layers:
    - name: static_layer
      static_layer:
        overload:
          global_downstream_max_connections: 100000
        re2:
          max_program_size: 1000
```

### WASM Optimization
```toml
# In Cargo.toml
[profile.release]
opt-level = "z"        # Optimize for size
lto = true             # Link-time optimization
codegen-units = 1      # Single codegen unit
strip = true           # Strip symbols
```

## Troubleshooting

### Common Issues

#### 1. xDS Connection Failure
```bash
# Symptoms
docker logs proxy-l7 | grep "xDS.*error"

# Solution
# Check API server is running
docker ps | grep api-server

# Verify xDS port is exposed
docker port api-server 18000

# Test connectivity
docker exec proxy-l7 nc -zv api-server 18000
```

#### 2. WASM Filter Not Loading
```bash
# Symptoms
docker logs proxy-l7 | grep -i wasm

# Solution
# Verify WASM files exist
docker exec proxy-l7 ls -lh /var/lib/envoy/wasm/

# Check Envoy config
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("HttpConnectionManager"))'
```

#### 3. XDP Not Working
```bash
# Symptoms
# No XDP performance improvement

# Solution
# Check capability
docker inspect proxy-l7 | grep -i cap_add

# Verify XDP loaded
docker exec proxy-l7 ip link show eth0

# Try SKB mode
-e XDP_MODE=skb
```

## Monitoring

### Key Metrics

**xDS Server:**
```bash
# Configuration updates
curl http://localhost:19000/v1/version

# gRPC connections
netstat -an | grep 18000
```

**Envoy Proxy:**
```bash
# Request metrics
curl http://localhost:9901/stats/prometheus | grep envoy_http

# xDS sync status
curl http://localhost:9901/stats | grep xds

# WASM filter metrics
curl http://localhost:9901/stats | grep wasm
```

**XDP Statistics:**
```bash
# Packet counts
docker exec proxy-l7 bpftool map dump name stats_map

# Rate limiting
docker exec proxy-l7 bpftool map dump name rate_limit_map
```

## Next Steps

1. **Phase 5**: WebUI integration for visual configuration
2. **Phase 6**: Multi-cloud routing implementation
3. **Phase 7**: Enhanced observability (tracing, dashboards)
4. **Phase 8**: Production deployment and testing

## References

- [Envoy xDS Protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol)
- [Envoy WASM Filters](https://github.com/proxy-wasm/spec)
- [XDP Tutorial](https://www.kernel.org/doc/html/latest/networking/af_xdp.html)
- [MarchProxy Documentation](../docs/)
