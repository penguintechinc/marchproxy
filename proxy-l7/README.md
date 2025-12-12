# MarchProxy L7 Proxy (Envoy)

High-performance Layer 7 proxy using Envoy with custom WASM filters and XDP acceleration.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MarchProxy L7 Proxy                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────┐     ┌──────────────────────────────┐      │
│  │ XDP Program │────▶│   Envoy Proxy Core           │      │
│  │             │     │                               │      │
│  │ - Protocol  │     │  ┌────────────────────────┐  │      │
│  │   Detection │     │  │ WASM Filters           │  │      │
│  │ - Rate      │     │  ├────────────────────────┤  │      │
│  │   Limiting  │     │  │ 1. Auth Filter         │  │      │
│  │ - DDoS      │     │  │    - JWT validation    │  │      │
│  │   Protection│     │  │    - Base64 tokens     │  │      │
│  └─────────────┘     │  ├────────────────────────┤  │      │
│         │            │  │ 2. License Filter      │  │      │
│         │            │  │    - Feature gating    │  │      │
│         ▼            │  │    - Proxy limits      │  │      │
│  Wire-speed          │  ├────────────────────────┤  │      │
│  Filtering           │  │ 3. Metrics Filter      │  │      │
│                      │  │    - Custom metrics    │  │      │
│                      │  │    - Prometheus        │  │      │
│                      │  └────────────────────────┘  │      │
│                      │                               │      │
│                      │  xDS Control Plane Client    │      │
│                      │  (Dynamic Configuration)     │      │
│                      └──────────────────────────────┘      │
│                                 │                            │
│                                 ▼                            │
│                      api-server:18000 (xDS)                 │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. XDP Program (`xdp/envoy_xdp.c`)
- **Early packet classification** before Envoy processing
- **Protocol detection**: HTTP/HTTPS/HTTP2/gRPC/WebSocket
- **Wire-speed rate limiting** using BPF maps
- **DDoS protection** at driver level
- **Statistics collection** for monitoring

**Performance**: 40+ Gbps throughput, sub-microsecond latency

### 2. WASM Filters (Rust)

#### Auth Filter (`filters/auth_filter/`)
- JWT token validation (HS256/HS384/HS512)
- Base64 token authentication
- Path-based exemptions (/healthz, /metrics)
- Automatic token rotation support

#### License Filter (`filters/license_filter/`)
- Enterprise feature gating
- Proxy count enforcement
- License validation
- Feature availability checks

#### Metrics Filter (`filters/metrics_filter/`)
- Custom MarchProxy metrics
- Request/response tracking
- Latency histograms
- Prometheus format

### 3. Envoy Configuration
- **Dynamic configuration** via xDS protocol
- **Static bootstrap** pointing to api-server:18000
- **Admin interface** on port 9901
- **HTTP/HTTPS listeners** with protocol auto-detection

## Building

### Prerequisites
- Docker (for containerized build)
- OR for local build:
  - Rust 1.75+ with wasm32-unknown-unknown target
  - LLVM/Clang for XDP
  - Linux kernel headers
  - libbpf-dev

### Build All Components
```bash
# Using Docker (recommended)
docker build -f envoy/Dockerfile -t marchproxy/proxy-l7:latest .

# Or build individually
./scripts/build_xdp.sh      # Build XDP program
./scripts/build_filters.sh  # Build WASM filters
```

### Build Output
```
build/
├── envoy_xdp.o           # XDP program
├── auth_filter.wasm      # Authentication filter
├── license_filter.wasm   # License filter
└── metrics_filter.wasm   # Metrics filter
```

## Running

### Docker Compose
```yaml
services:
  proxy-l7:
    image: marchproxy/proxy-l7:latest
    container_name: marchproxy-proxy-l7
    environment:
      - XDS_SERVER=api-server:18000
      - CLUSTER_API_KEY=${CLUSTER_API_KEY}
      - XDP_INTERFACE=eth0  # Optional
      - XDP_MODE=native     # native, skb
      - LOGLEVEL=info
    ports:
      - "80:10000"    # HTTP/HTTPS
      - "9901:9901"   # Admin
    cap_add:
      - NET_ADMIN     # Required for XDP
    networks:
      - marchproxy-network
    depends_on:
      - api-server
```

### Standalone Docker
```bash
docker run -d \
  --name proxy-l7 \
  -p 80:10000 \
  -p 9901:9901 \
  --cap-add=NET_ADMIN \
  -e XDS_SERVER=api-server:18000 \
  -e CLUSTER_API_KEY=your-api-key \
  marchproxy/proxy-l7:latest
```

### Load XDP Manually
```bash
# Inside container or host with NET_ADMIN
sudo ./scripts/load_xdp.sh eth0 native

# Modes:
# - native: Driver-level XDP (best performance)
# - skb: Generic XDP (compatible)
# - hw: Hardware offload (if supported)
```

## Configuration

### XDP Rate Limiting
Configure via BPF maps (done through api-server):
```json
{
  "window_ns": 1000000000,  // 1 second
  "max_packets": 10000,     // 10k packets/sec per IP
  "enabled": true
}
```

### WASM Filter Configuration

#### Auth Filter
```json
{
  "jwt_secret": "your-secret-key",
  "jwt_algorithm": "HS256",
  "require_auth": true,
  "base64_tokens": ["token1", "token2"],
  "exempt_paths": ["/healthz", "/metrics"]
}
```

#### License Filter
```json
{
  "license_key": "PENG-XXXX-XXXX-XXXX-XXXX-ABCD",
  "is_enterprise": true,
  "features": {
    "rate_limiting": true,
    "multi_cloud": true,
    "distributed_tracing": true
  },
  "max_proxies": 100
}
```

## Monitoring

### Admin Interface
```bash
# Health check
curl http://localhost:9901/ready
curl http://localhost:9901/server_info

# Statistics
curl http://localhost:9901/stats
curl http://localhost:9901/stats/prometheus

# Configuration dump
curl http://localhost:9901/config_dump
```

### XDP Statistics
```bash
# View XDP stats via BPF maps
bpftool map dump name stats_map

# View rate limiting stats
bpftool map dump name rate_limit_map
```

### Metrics
Prometheus metrics available at:
- Envoy built-in: `http://localhost:9901/stats/prometheus`
- Custom WASM: Integrated into Envoy stats

## Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Throughput | 40+ Gbps | HTTP/HTTPS/HTTP2 |
| Requests/sec | 1M+ | gRPC/HTTP2 |
| Latency (p99) | <10ms | End-to-end |
| XDP processing | <1μs | Per packet |

## Troubleshooting

### XDP Not Loading
```bash
# Check capabilities
docker inspect proxy-l7 | grep -i cap_add

# Check interface
ip link show

# Try SKB mode
docker run ... -e XDP_MODE=skb ...
```

### WASM Filters Not Loading
```bash
# Check Envoy logs
docker logs proxy-l7

# Verify WASM files
docker exec proxy-l7 ls -lh /var/lib/envoy/wasm/

# Check Envoy config
curl http://localhost:9901/config_dump
```

### xDS Connection Issues
```bash
# Test xDS server
nc -zv api-server 18000

# Check Envoy logs
docker logs proxy-l7 | grep xDS

# Verify bootstrap config
docker exec proxy-l7 cat /etc/envoy/envoy.yaml
```

## Development

### Testing WASM Filters Locally
```bash
# Build filters
cd filters/auth_filter
cargo build --target wasm32-unknown-unknown --release

# Test with Envoy locally
envoy -c test-config.yaml --log-level debug
```

### XDP Development
```bash
# Compile XDP program
cd xdp
make

# Verify BPF object
llvm-objdump -S envoy_xdp.o

# Load and test
sudo ./scripts/load_xdp.sh eth0 skb
```

## License

Limited AGPL-3.0 - See LICENSE file for details

## Contributing

See CONTRIBUTING.md for guidelines
