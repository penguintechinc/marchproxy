# MarchProxy ALB (Application Load Balancer)

The ALB container wraps Envoy L7 proxy with a Go supervisor that implements the ModuleService gRPC interface for NLB communication.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MarchProxy ALB                            │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────────┐         ┌──────────────────┐          │
│  │  Go Supervisor   │◄───────►│   Envoy Proxy    │          │
│  │                  │         │                   │          │
│  │ - gRPC Server    │         │ - L7 Routing     │          │
│  │ - Lifecycle Mgmt │         │ - Load Balancing │          │
│  │ - Metrics        │         │ - Rate Limiting  │          │
│  │ - Health Checks  │         │ - TLS Termination│          │
│  └──────────────────┘         └──────────────────┘          │
│         │                              │                     │
│         │ ModuleService gRPC           │ xDS                 │
│         ▼                              ▼                     │
│       NLB                      API Server (xDS)              │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

## Features

### ModuleService gRPC Interface

The ALB implements the following gRPC methods for NLB communication:

- **GetStatus**: Returns health and operational status
- **GetRoutes**: Returns current route configuration
- **ApplyRateLimit**: Applies rate limiting to specific routes
- **GetMetrics**: Returns performance and traffic metrics
- **SetTrafficWeight**: Controls traffic distribution for blue/green deployments
- **Reload**: Triggers graceful configuration reload

### Envoy Integration

- **Lifecycle Management**: Start, stop, reload Envoy gracefully
- **xDS Integration**: Dynamic configuration from control plane
- **Health Monitoring**: Continuous health checks via admin API
- **Metrics Collection**: Aggregates Envoy stats for gRPC responses

### Blue/Green Deployments

- **Traffic Weighting**: Dynamically adjust traffic distribution
- **Version Control**: Support for multiple backend versions
- **Gradual Rollout**: Safely migrate traffic between versions

## Building

### Prerequisites

- Go 1.24+
- Docker (for containerized builds)
- protoc (Protocol Buffers compiler)

### Local Build

```bash
# Generate protobuf code
make proto

# Build Go supervisor
make build

# Run locally (requires Envoy binary)
make run
```

### Docker Build

```bash
# Build Docker image
make build-docker

# Build multi-architecture image
make build-multi
```

## Running

### Docker Compose

```yaml
services:
  proxy-alb:
    image: marchproxy/proxy-alb:latest
    container_name: marchproxy-alb
    environment:
      - MODULE_ID=alb-1
      - XDS_SERVER=api-server:18000
      - CLUSTER_API_KEY=${CLUSTER_API_KEY}
      - LOG_LEVEL=info
    ports:
      - "10000:10000"  # HTTP/HTTPS
      - "9901:9901"    # Envoy admin
      - "50051:50051"  # gRPC ModuleService
      - "9090:9090"    # Prometheus metrics
      - "8080:8080"    # Health checks
    depends_on:
      - api-server
```

### Standalone Docker

```bash
docker run -d \
  --name proxy-alb \
  -p 10000:10000 \
  -p 9901:9901 \
  -p 50051:50051 \
  -p 9090:9090 \
  -p 8080:8080 \
  -e XDS_SERVER=api-server:18000 \
  -e MODULE_ID=alb-1 \
  marchproxy/proxy-alb:latest
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MODULE_ID` | `alb-1` | Unique identifier for this ALB instance |
| `VERSION` | `v1.0.0` | ALB version |
| `ENVOY_BINARY` | `/usr/local/bin/envoy` | Path to Envoy binary |
| `ENVOY_CONFIG_PATH` | `/etc/envoy/envoy.yaml` | Path to Envoy config |
| `ENVOY_ADMIN_PORT` | `9901` | Envoy admin port |
| `ENVOY_LISTEN_PORT` | `10000` | HTTP/HTTPS listen port |
| `ENVOY_LOG_LEVEL` | `info` | Envoy log level |
| `XDS_SERVER` | `api-server:18000` | xDS control plane address |
| `XDS_NODE_ID` | `alb-node` | Node ID for xDS |
| `XDS_CLUSTER` | `marchproxy-cluster` | Cluster name for xDS |
| `GRPC_PORT` | `50051` | ModuleService gRPC port |
| `METRICS_PORT` | `9090` | Prometheus metrics port |
| `HEALTH_PORT` | `8080` | Health check port |
| `LOG_LEVEL` | `info` | Supervisor log level |

## Monitoring

### Health Checks

```bash
# Liveness check
curl http://localhost:8080/healthz

# Readiness check
curl http://localhost:8080/ready

# Envoy admin
curl http://localhost:9901/ready
```

### Metrics

```bash
# Prometheus metrics
curl http://localhost:9090/metrics

# Envoy stats
curl http://localhost:9901/stats/prometheus
```

### gRPC API

```bash
# Get status (using grpcurl)
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus

# Get routes
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetRoutes

# Get metrics
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetMetrics
```

## Development

### Project Structure

```
proxy-alb/
├── main.go                      # Entry point
├── go.mod                       # Go module definition
├── Makefile                     # Build automation
├── Dockerfile                   # Multi-stage Docker build
├── README.md                    # This file
├── envoy/
│   └── envoy.yaml              # Envoy bootstrap config
└── internal/
    ├── config/
    │   └── config.go           # Configuration management
    ├── envoy/
    │   ├── manager.go          # Envoy lifecycle manager
    │   └── xds.go              # xDS client
    ├── grpc/
    │   └── server.go           # ModuleService implementation
    └── metrics/
        └── collector.go        # Metrics collection
```

### Testing

```bash
# Run tests
make test

# Run linters
make lint

# Format code
make fmt
```

## Integration with NLB

The ALB communicates with the NLB (Network Load Balancer) via gRPC:

```
NLB → gRPC (port 50051) → ALB ModuleService
```

### Example: Set Traffic Weights

```go
import pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"

// Connect to ALB
conn, _ := grpc.Dial("alb:50051", grpc.WithInsecure())
client := pb.NewModuleServiceClient(conn)

// Set traffic weights for blue/green deployment
req := &pb.TrafficWeightRequest{
    RouteName: "api-route",
    Weights: []*pb.BackendWeight{
        {BackendName: "api-blue", Weight: 90, Version: "blue"},
        {BackendName: "api-green", Weight: 10, Version: "green"},
    },
}

resp, _ := client.SetTrafficWeight(context.Background(), req)
```

## Troubleshooting

### Envoy Won't Start

```bash
# Check Envoy config
docker exec proxy-alb cat /etc/envoy/envoy.yaml

# Check Envoy logs
docker logs proxy-alb

# Verify xDS connectivity
docker exec proxy-alb wget -O- http://api-server:18000/healthz
```

### gRPC Connection Issues

```bash
# Test gRPC connectivity
grpcurl -plaintext localhost:50051 list

# Check gRPC server logs
docker logs proxy-alb | grep grpc
```

### Metrics Not Updating

```bash
# Check Envoy admin API
curl http://localhost:9901/stats

# Verify metrics collector
docker logs proxy-alb | grep metrics
```

## Performance

### Benchmarks

| Metric | Value | Notes |
|--------|-------|-------|
| Throughput | 40+ Gbps | HTTP/HTTPS traffic |
| Requests/sec | 1M+ | gRPC/HTTP2 |
| Latency (p99) | <10ms | End-to-end |
| gRPC overhead | <1ms | ModuleService calls |

## License

Limited AGPL-3.0 - See LICENSE file for details

## Contributing

See CONTRIBUTING.md for guidelines
