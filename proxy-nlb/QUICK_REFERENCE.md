# MarchProxy NLB Quick Reference

## Quick Start

```bash
# Build
docker build -t marchproxy-nlb:latest .

# Run
docker run -d \
  -p 8080:8080 -p 8082:8082 -p 50051:50051 \
  -e CLUSTER_API_KEY=your-key \
  marchproxy-nlb:latest
```

## Ports

| Port | Purpose |
|------|---------|
| 8080 | Main traffic ingress |
| 8082 | Metrics and health |
| 50051 | gRPC module API |

## Endpoints

```bash
# Health check
curl http://localhost:8082/healthz

# Metrics
curl http://localhost:8082/metrics

# Status
curl http://localhost:8082/status | jq
```

## Environment Variables

```bash
# Required
CLUSTER_API_KEY=your-api-key

# Optional
MARCHPROXY_NLB_BIND_ADDR=:8080
MARCHPROXY_NLB_GRPC_PORT=50051
MARCHPROXY_NLB_ENABLE_RATE_LIMITING=true
MARCHPROXY_NLB_ENABLE_AUTOSCALING=true
MARCHPROXY_NLB_ENABLE_BLUEGREEN=true
```

## Supported Protocols

| Protocol | Signature | Detection |
|----------|-----------|-----------|
| HTTP | `GET `, `POST `, `HTTP/` | HTTP methods/version |
| MySQL | `0x0a` | Protocol version byte |
| PostgreSQL | `0x00030000` | Protocol version |
| MongoDB | `OP_MSG`, `OP_QUERY` | Wire protocol opcodes |
| Redis | `*`, `$`, `+`, `-`, `:` | RESP protocol markers |
| RTMP | `0x03` | Handshake version |

## Key Metrics

```prometheus
# Routing
nlb_routed_connections_total{protocol="http",module="http-1"}
nlb_active_connections{protocol="http",module="http-1"}
nlb_routing_errors_total{protocol="http",error_type="no_module"}

# Rate Limiting
nlb_ratelimit_allowed_total{protocol="http",bucket="http_global"}
nlb_ratelimit_denied_total{protocol="http",bucket="http_global"}
nlb_ratelimit_tokens_available{protocol="http",bucket="http_global"}

# Autoscaling
nlb_scale_operations_total{protocol="http",direction="up"}
nlb_current_replicas{protocol="http"}

# Blue/Green
nlb_bluegreen_traffic_split{protocol="http",version="v1.0",color="blue"}
nlb_bluegreen_deployments_total{protocol="http",status="completed"}
```

## Configuration Snippets

### Minimal config.yaml
```yaml
bind_addr: ":8080"
grpc_port: 50051
metrics_addr: ":8082"
manager_url: "http://api-server:8000"
cluster_api_key: "your-key"
```

### Rate Limiting
```yaml
enable_rate_limiting: true
rate_limit_buckets:
  - name: "http_global"
    protocol: "http"
    capacity: 50000.0
    refill_rate: 10000.0
```

### Autoscaling
```yaml
enable_autoscaling: true
autoscale_interval: 30s
scale_up_cooldown: 3m
scale_down_cooldown: 5m
```

### Blue/Green
```yaml
enable_bluegreen: true
canary_step_size: 10
canary_step_duration: 2m
```

## Common Operations

### Check NLB Status
```bash
curl -s http://localhost:8082/status | jq '.status.router_stats'
```

### Monitor Rate Limiting
```bash
curl -s http://localhost:8082/status | jq '.status.ratelimit_stats'
```

### View Autoscaling Status
```bash
curl -s http://localhost:8082/status | jq '.status.autoscaler_stats'
```

### Check Blue/Green Deployments
```bash
curl -s http://localhost:8082/status | jq '.status.bluegreen_stats'
```

## Troubleshooting

### No modules available
```bash
# Check registered modules
curl -s http://localhost:8082/status | jq '.status.router_stats.protocols'

# Verify gRPC server is running
grpcurl -plaintext localhost:50051 list
```

### Rate limiting issues
```bash
# Check token availability
curl -s http://localhost:8082/metrics | grep nlb_ratelimit_tokens

# View denied requests
curl -s http://localhost:8082/metrics | grep nlb_ratelimit_denied
```

### Autoscaling not working
```bash
# Check autoscaler enabled
curl -s http://localhost:8082/status | jq '.status.autoscaler_stats.enabled'

# View recent scaling decisions
curl -s http://localhost:8082/status | jq '.status.autoscaler_stats.recent_scaling_history'
```

### Connection failures
```bash
# Check gRPC client pool
curl -s http://localhost:8082/status | jq '.status.client_pool_stats'

# View routing errors
curl -s http://localhost:8082/metrics | grep nlb_routing_errors
```

## Development

### Build from source
```bash
go mod download
go build -o proxy-nlb ./cmd/nlb/main.go
./proxy-nlb --config config.yaml
```

### Run tests
```bash
go test -v -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting
```bash
golangci-lint run
go fmt ./...
go vet ./...
```

## Docker Compose Example

```yaml
version: '3.8'
services:
  nlb:
    build: ./proxy-nlb
    ports:
      - "8080:8080"
      - "8082:8082"
      - "50051:50051"
    environment:
      - CLUSTER_API_KEY=${CLUSTER_API_KEY}
      - MARCHPROXY_NLB_MANAGER_URL=http://api-server:8000
    volumes:
      - ./config.yaml:/app/config.yaml
    command: ["--config", "/app/config.yaml"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Architecture

```
                    ┌─────────────────┐
                    │   Client        │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   NLB :8080     │
                    │  ┌──────────┐   │
                    │  │Inspector │   │
                    │  └─────┬────┘   │
                    │  ┌─────▼────┐   │
                    │  │ Router   │   │
                    │  └─────┬────┘   │
                    └────────┼────────┘
                             │ gRPC
        ┌────────────────────┼────────────────────┐
        │                    │                    │
┌───────▼───────┐   ┌────────▼────────┐   ┌──────▼──────┐
│  HTTP Module  │   │  MySQL Module   │   │Redis Module │
│    :50052     │   │    :50053       │   │   :50054    │
└───────────────┘   └─────────────────┘   └─────────────┘
```

## Support

- Documentation: `/proxy-nlb/README.md`
- Examples: `/proxy-nlb/config.example.yaml`
- Issues: GitHub Issues
- Website: https://www.penguintech.io
