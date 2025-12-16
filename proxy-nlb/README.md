# MarchProxy Network Load Balancer (NLB)

The MarchProxy NLB is a unified network load balancer that serves as the single entry point for all traffic in the MarchProxy Unified Architecture. It provides intelligent protocol detection, traffic routing, rate limiting, autoscaling orchestration, and blue/green deployment capabilities.

## Architecture Overview

The NLB is designed as the central traffic controller that:

1. **Receives All Traffic** - Single entry point on configurable ports
2. **Detects Protocol** - Inspects initial packets to identify protocol type
3. **Routes to Modules** - Forwards traffic to appropriate specialized module containers
4. **Manages Rate Limiting** - Token bucket algorithm for traffic control
5. **Orchestrates Scaling** - Auto-scales module containers based on load
6. **Handles Deployments** - Blue/green and canary deployment support

## Features

### Protocol Detection

Supports automatic detection of:
- **HTTP/HTTPS** - Web traffic (GET, POST, PUT, DELETE, etc.)
- **MySQL** - MySQL database protocol (greeting packet 0x0a)
- **PostgreSQL** - PostgreSQL database protocol (startup messages)
- **MongoDB** - MongoDB wire protocol (OP_MSG, OP_QUERY)
- **Redis** - Redis RESP protocol (*n\r\n)
- **RTMP** - Real-Time Messaging Protocol (0x03 handshake)

### Traffic Routing

- **Least Connections** - Routes to module with fewest active connections
- **Health-Aware** - Only routes to healthy module instances
- **Protocol-Specific** - Dedicated routing per protocol type
- **Connection Tracking** - Monitors active connections per module

### Rate Limiting

- **Token Bucket Algorithm** - Industry-standard rate limiting
- **Per-Protocol Buckets** - Separate limits for each protocol
- **Per-Service Buckets** - Fine-grained control per service
- **Configurable Refill** - Customizable capacity and refill rates
- **Real-time Metrics** - Prometheus metrics for monitoring

### Autoscaling

- **Metric-Based** - Scales on CPU, memory, connection count
- **Policy-Driven** - Configurable policies per protocol
- **Cooldown Periods** - Prevents flapping with separate up/down cooldowns
- **Min/Max Replicas** - Bounded scaling with safety limits
- **Evaluation Periods** - Multi-period average for stability

### Blue/Green Deployments

- **Instant Switch** - Immediate traffic cutover
- **Canary Rollout** - Gradual traffic shifting
- **Version Tracking** - Separate blue and green versions
- **Rollback Support** - Quick rollback on issues
- **Traffic Splitting** - Weighted traffic distribution

### gRPC Communication

- **Client Pool** - Connection pooling to module containers
- **Health Checks** - Automatic connection health monitoring
- **Auto-Reconnect** - Automatic reconnection on failures
- **Keepalive** - Connection keepalive for stability
- **Server API** - gRPC server for module registration

## Directory Structure

```
proxy-nlb/
├── cmd/
│   ├── main.go               # Simplified entry point
│   └── nlb/
│       └── main.go           # Alternative entry point (CLI)
├── internal/
│   ├── nlb/
│   │   ├── inspector.go      # Protocol detection
│   │   ├── router.go         # Traffic routing
│   │   ├── ratelimit.go      # Rate limiting
│   │   ├── autoscaler.go     # Autoscaling controller
│   │   └── bluegreen.go      # Blue/green deployments
│   ├── grpc/
│   │   ├── client.go         # gRPC client pool
│   │   └── server.go         # gRPC server
│   └── config/
│       └── config.go         # Configuration management
├── Dockerfile                # Multi-stage Docker build
├── go.mod                    # Go module dependencies
└── README.md                 # This file
```

## Configuration

### Environment Variables

```bash
# Manager connection
MARCHPROXY_NLB_MANAGER_URL=http://api-server:8000
CLUSTER_API_KEY=your-cluster-api-key

# Server settings
MARCHPROXY_NLB_BIND_ADDR=:8080
MARCHPROXY_NLB_GRPC_PORT=50051
MARCHPROXY_NLB_METRICS_ADDR=:8082

# Rate limiting
MARCHPROXY_NLB_ENABLE_RATE_LIMITING=true
MARCHPROXY_NLB_DEFAULT_RATE_LIMIT=10000.0
MARCHPROXY_NLB_DEFAULT_BURST_SIZE=20000.0

# Autoscaling
MARCHPROXY_NLB_ENABLE_AUTOSCALING=true
MARCHPROXY_NLB_AUTOSCALE_INTERVAL=30s
MARCHPROXY_NLB_SCALE_UP_COOLDOWN=3m
MARCHPROXY_NLB_SCALE_DOWN_COOLDOWN=5m

# Blue/Green
MARCHPROXY_NLB_ENABLE_BLUEGREEN=true
MARCHPROXY_NLB_CANARY_STEP_SIZE=10
MARCHPROXY_NLB_CANARY_STEP_DURATION=2m

# Module management
MARCHPROXY_NLB_MAX_MODULES_PER_PROTOCOL=50
MARCHPROXY_NLB_MAX_CONNECTIONS_PER_MODULE=10000

# Licensing (Enterprise)
MARCHPROXY_NLB_LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
MARCHPROXY_NLB_LICENSE_SERVER=https://license.penguintech.io
MARCHPROXY_NLB_RELEASE_MODE=false
```

### Configuration File (YAML)

```yaml
# Server settings
bind_addr: ":8080"
grpc_port: 50051
metrics_addr: ":8082"

# Manager connection
manager_url: "http://api-server:8000"
cluster_api_key: "your-cluster-api-key"

# Rate limiting
enable_rate_limiting: true
default_rate_limit: 10000.0
default_burst_size: 20000.0
rate_limit_buckets:
  - name: "http_global"
    protocol: "http"
    capacity: 50000.0
    refill_rate: 10000.0
  - name: "mysql_global"
    protocol: "mysql"
    capacity: 10000.0
    refill_rate: 2000.0

# Autoscaling
enable_autoscaling: true
autoscale_interval: 30s
scale_up_cooldown: 3m
scale_down_cooldown: 5m

# Blue/Green deployments
enable_bluegreen: true
canary_step_size: 10
canary_step_duration: 2m

# Module management
max_modules_per_protocol: 50
module_health_check_interval: 10s
max_connections_per_module: 10000
```

## Building

### Docker Build

```bash
# Production build
docker build --target production -t marchproxy-nlb:latest .

# Development build
docker build --target development -t marchproxy-nlb:dev .

# Testing build
docker build --target testing -t marchproxy-nlb:test .

# Debug build
docker build --target debug -t marchproxy-nlb:debug .
```

### Local Build

```bash
# Download dependencies
go mod download

# Build binary (simplified entry point)
go build ./cmd/main.go

# Or build from cmd/nlb/main.go (with CLI flags)
go build -o proxy-nlb ./cmd/nlb/main.go

# Run
./main  # Uses config.example.yaml by default
# Or with custom config
CONFIG_PATH=/path/to/config.yaml ./main
```

## Running

### Docker

```bash
docker run -d \
  --name marchproxy-nlb \
  -p 8080:8080 \
  -p 8082:8082 \
  -p 50051:50051 \
  -e CLUSTER_API_KEY=your-api-key \
  marchproxy-nlb:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  nlb:
    image: marchproxy-nlb:latest
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
```

## Monitoring

### Health Check

```bash
curl http://localhost:8082/healthz
```

### Metrics (Prometheus)

```bash
curl http://localhost:8082/metrics
```

Key metrics:
- `nlb_routed_connections_total` - Connections routed by protocol/module
- `nlb_routing_errors_total` - Routing errors by protocol/error type
- `nlb_active_connections` - Active connections per module
- `nlb_ratelimit_allowed_total` - Requests allowed by rate limiter
- `nlb_ratelimit_denied_total` - Requests denied by rate limiter
- `nlb_scale_operations_total` - Scaling operations by direction
- `nlb_current_replicas` - Current replica count per protocol
- `nlb_bluegreen_traffic_split` - Traffic split percentage

### Status Endpoint

```bash
curl http://localhost:8082/status | jq
```

## gRPC API

The NLB provides a gRPC API for module registration and management:

### Module Registration

Modules register themselves with the NLB on startup:

```protobuf
service NLBService {
  rpc RegisterModule(RegisterModuleRequest) returns (RegisterModuleResponse);
  rpc UnregisterModule(UnregisterModuleRequest) returns (UnregisterModuleResponse);
  rpc UpdateHealth(HealthUpdateRequest) returns (HealthUpdateResponse);
  rpc GetStats(StatsRequest) returns (StatsResponse);
}
```

## License

Licensed under the Limited AGPL3 with preamble for fair use.

Copyright (c) 2025 PenguinTech, LLC.
https://www.penguintech.io

## Version

Version: 1.0.0
Build: Development
