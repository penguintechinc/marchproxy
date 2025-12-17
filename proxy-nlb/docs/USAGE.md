# MarchProxy NLB Usage Guide

## Quick Start

### Local Development

```bash
# Build the NLB
go build -o proxy-nlb ./cmd/nlb/main.go

# Run with default configuration
./proxy-nlb

# Run with custom configuration
./proxy-nlb --config /path/to/config.yaml
```

### Docker Container

```bash
# Build Docker image
docker build -t marchproxy-nlb:latest .

# Run container
docker run -d \
  --name marchproxy-nlb \
  -p 8080:8080 -p 8082:8082 -p 50051:50051 \
  -e CLUSTER_API_KEY=your-key \
  marchproxy-nlb:latest

# View logs
docker logs -f marchproxy-nlb

# Stop container
docker stop marchproxy-nlb
```

### Docker Compose

```bash
# Start all services
docker compose up -d

# View logs
docker compose logs -f nlb

# Stop services
docker compose down
```

---

## Port Usage

The NLB uses three main ports:

| Port | Purpose | Protocol |
|------|---------|----------|
| 8080 | Main traffic ingress | TCP (any protocol) |
| 8082 | Metrics & health | HTTP |
| 50051 | gRPC module API | gRPC |

Customize in configuration:
```yaml
bind_addr: ":8080"
metrics_addr: ":8082"
grpc_port: 50051
```

---

## Monitoring NLB Health

### Health Check Endpoint

```bash
# Simple liveness check
curl http://localhost:8082/healthz
# Response: OK
```

### Status Endpoint

```bash
# Comprehensive status
curl http://localhost:8082/status | jq '.'

# Router statistics
curl http://localhost:8082/status | jq '.status.router_stats'

# Rate limit status
curl http://localhost:8082/status | jq '.status.ratelimit_stats'

# Autoscaler status
curl http://localhost:8082/status | jq '.status.autoscaler_stats'

# Blue/green status
curl http://localhost:8082/status | jq '.status.bluegreen_stats'
```

### Prometheus Metrics

```bash
# Get all metrics
curl http://localhost:8082/metrics

# Filter for specific metric
curl http://localhost:8082/metrics | grep nlb_routed

# Monitor in real-time
watch -n 1 'curl -s http://localhost:8082/metrics | grep nlb_'
```

---

## Working with Modules

Modules (HTTP, MySQL, PostgreSQL, etc.) register with the NLB via gRPC.

### Module Registration Flow

1. Module container starts
2. Connects to NLB gRPC server on port 50051
3. Calls `RegisterModule` with:
   - Module ID (unique identifier)
   - Protocol type
   - Listen address and port
   - Version information
4. NLB adds module to routing table
5. Module health status sent periodically

### Example Module Registration (Go)

```go
import (
    "context"
    "time"
    "google.golang.org/grpc"
    nlb "marchproxy-nlb/api/v1"
)

func registerWithNLB(nlbAddr string) error {
    conn, err := grpc.Dial(nlbAddr, grpc.WithInsecure())
    if err != nil {
        return err
    }
    defer conn.Close()

    client := nlb.NewNLBServiceClient(conn)

    resp, err := client.RegisterModule(context.Background(), &nlb.RegisterModuleRequest{
        ModuleId: "http-1",
        Protocol: "http",
        Address:  "127.0.0.1",
        Port:     50052,
        Version:  "1.0.0",
        Labels: map[string]string{
            "service": "web",
            "tier":    "frontend",
        },
    })

    if err != nil {
        return err
    }

    log.Printf("Registered: %s", resp.ModuleId)
    return nil
}
```

### Monitoring Registered Modules

```bash
# Check registered modules
curl -s http://localhost:8082/status | jq '.status.router_stats.protocols'

# Example response
{
  "http": {
    "active_modules": 3,
    "active_connections": 250,
    "total_routed": 50000
  },
  "mysql": {
    "active_modules": 2,
    "active_connections": 150,
    "total_routed": 25000
  }
}
```

### Module Health Status

Modules update health status via gRPC:

```bash
# Healthy module
curl -s http://localhost:8082/status | jq '.status.router_stats.protocols.http.health'

# Returns: "healthy" | "degraded" | "unhealthy"
```

---

## Traffic Routing

### Protocol Detection

The NLB automatically detects incoming traffic protocol by inspecting the first bytes:

**Supported Protocols**:
- HTTP (GET, POST, HTTP/)
- MySQL (0x0a byte signature)
- PostgreSQL (0x00030000 version)
- MongoDB (OP_MSG, OP_QUERY opcodes)
- Redis (RESP markers: *, $, +, -, :)
- RTMP (0x03 handshake)

### Routing Algorithm

The NLB uses **least connections** algorithm:
1. Detect protocol from incoming traffic
2. Find all healthy modules for that protocol
3. Route to module with fewest active connections
4. Increment connection counter
5. Forward all traffic to selected module

### Health-Aware Routing

Only healthy modules receive traffic:

```bash
# Check module health
curl -s http://localhost:8082/status | \
  jq '.status.router_stats.protocols.http.modules'

# Example: unhealthy module excluded from routing
```

---

## Rate Limiting

Rate limiting uses token bucket algorithm to control traffic flow.

### How Rate Limiting Works

1. **Token Bucket**: Holds N tokens (capacity)
2. **Refill Rate**: Adds M tokens per second
3. **Request Cost**: Each request costs 1 token
4. **Decision**:
   - Has token? Allow request, consume token
   - No token? Reject request (return 429 Too Many Requests)

### Configuring Rate Limits

Global defaults in configuration:
```yaml
default_rate_limit: 10000.0      # 10k requests/sec
default_burst_size: 20000.0      # Allow 20k burst
```

Per-protocol buckets:
```yaml
rate_limit_buckets:
  - name: "http_global"
    protocol: "http"
    capacity: 50000.0
    refill_rate: 10000.0
```

Environment variable override:
```bash
MARCHPROXY_NLB_DEFAULT_RATE_LIMIT=50000.0
MARCHPROXY_NLB_DEFAULT_BURST_SIZE=100000.0
```

### Monitoring Rate Limiting

```bash
# Check token availability
curl -s http://localhost:8082/status | \
  jq '.status.ratelimit_stats.buckets'

# Monitor denied requests
curl http://localhost:8082/metrics | grep nlb_ratelimit_denied_total

# Monitor allowed requests
curl http://localhost:8082/metrics | grep nlb_ratelimit_allowed_total

# Real-time rate limit monitoring
watch -n 1 'curl -s http://localhost:8082/metrics | grep nlb_ratelimit'
```

### Rate Limit Behavior

When rate limit exceeded:
```
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1703064645
Retry-After: 1

{"error": "rate_limit_exceeded", "retry_after": 1}
```

---

## Autoscaling

The NLB automatically scales module replicas based on load.

### How Autoscaling Works

1. **Evaluation**: Every 30 seconds (configurable)
2. **Metric Collection**: Gather CPU, memory, connection stats
3. **Policy Check**: Apply scaling policies
4. **Decision**: Scale up, scale down, or hold steady
5. **Cooldown**: Wait before next scaling operation

### Scaling Policies

**Scale Up** (add replicas):
- Trigger: CPU > 80% OR active connections > threshold
- Increment: 1 replica per scale up
- Cooldown: 3 minutes before next scale up

**Scale Down** (remove replicas):
- Trigger: CPU < 20% AND active connections < threshold
- Decrement: 1 replica per scale down
- Cooldown: 5 minutes before next scale down

**Bounds**:
- Minimum: 1 replica
- Maximum: 50 replicas (configurable)

### Configuring Autoscaling

```yaml
enable_autoscaling: true
autoscale_interval: 30s           # Evaluation frequency
scale_up_cooldown: 3m             # Wait 3 min before next scale up
scale_down_cooldown: 5m           # Wait 5 min before next scale down
scale_up_threshold: 80            # CPU % to trigger scale up
scale_down_threshold: 20          # CPU % to trigger scale down
```

### Monitoring Autoscaling

```bash
# View autoscaling status
curl -s http://localhost:8082/status | \
  jq '.status.autoscaler_stats'

# View recent scaling decisions
curl -s http://localhost:8082/status | \
  jq '.status.autoscaler_stats.recent_scaling_history'

# Monitor current replica count
curl http://localhost:8082/metrics | grep nlb_current_replicas

# Monitor scaling operations
curl http://localhost:8082/metrics | grep nlb_scale_operations_total
```

### Example Scaling Event

```json
{
  "timestamp": "2025-12-16T10:25:00Z",
  "protocol": "http",
  "direction": "up",
  "previous_replicas": 2,
  "new_replicas": 3,
  "reason": "high_cpu",
  "metrics": {
    "cpu_usage": 85,
    "active_connections": 5000
  }
}
```

---

## Blue/Green Deployments

Blue/Green deployments enable zero-downtime updates of module versions.

### How Blue/Green Works

1. **Blue Version**: Current production version running
2. **Green Version**: New version being deployed
3. **Traffic Splitting**: Gradually shift traffic from blue to green
4. **Canary Rollout**: 10% increments, 2 minutes per step (configurable)
5. **Rollback**: Instantly switch back to blue if issues detected

### Deployment Flow

```
Initial State (100% Blue):
┌─────────────────┐
│  Blue v1.0.0    │ 100%
└─────────────────┘

After 2 minutes (90% Blue, 10% Green):
┌─────────────────┐
│  Blue v1.0.0    │ 90%
└─────────────────┘
┌─────────────────┐
│  Green v1.1.0   │ 10%
└─────────────────┘

After 20 minutes (100% Green):
┌─────────────────┐
│  Green v1.1.0   │ 100%
└─────────────────┘
```

### Configuring Blue/Green

```yaml
enable_bluegreen: true
canary_step_size: 10              # 10% traffic per step
canary_step_duration: 2m          # Wait 2 min between steps
```

### Initiating Deployment

```bash
# Trigger blue/green deployment
curl -X POST http://localhost:8082/api/v1/deployments \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "green_version": "v1.1.0",
    "rollout_strategy": "canary",
    "step_size": 10,
    "step_duration": "2m"
  }'
```

### Monitoring Deployment

```bash
# View active deployments
curl -s http://localhost:8082/status | \
  jq '.status.bluegreen_stats.active_deployments'

# Monitor traffic split percentage
curl http://localhost:8082/metrics | grep nlb_bluegreen_traffic_split

# Check deployment completion
curl http://localhost:8082/metrics | grep nlb_bluegreen_deployments_total
```

### Rollback

```bash
# Immediate rollback to blue
curl -X POST http://localhost:8082/api/v1/deployments/rollback \
  -H "Content-Type: application/json" \
  -d '{"protocol": "http"}'
```

---

## Performance Optimization

### Connection Pooling

Enable gRPC connection pooling for efficient module communication:

```yaml
enable_connection_pooling: true
max_connections_per_module: 10000
```

### Rate Limit Tuning

For different workloads:

```yaml
# High-throughput APIs
rate_limit_buckets:
  - name: "api_global"
    protocol: "http"
    capacity: 500000.0
    refill_rate: 100000.0

# Database connections (lower rate)
rate_limit_buckets:
  - name: "db_global"
    protocol: "mysql"
    capacity: 10000.0
    refill_rate: 2000.0
```

### Autoscaling Tuning

For different response patterns:

```yaml
# Aggressive scaling (quick response)
autoscale_interval: 5s
scale_up_cooldown: 1m
scale_down_cooldown: 2m

# Conservative scaling (stability)
autoscale_interval: 60s
scale_up_cooldown: 10m
scale_down_cooldown: 15m
```

---

## Troubleshooting

### No Traffic Routed

```bash
# Check registered modules
curl -s http://localhost:8082/status | \
  jq '.status.router_stats.protocols'

# Ensure modules are registered via gRPC
grpcurl -plaintext localhost:50051 list

# Check if protocol is detected correctly
# Verify incoming traffic matches supported protocols
```

### High Latency

```bash
# Check active connections per module
curl -s http://localhost:8082/status | \
  jq '.status.router_stats.protocols[].active_connections'

# Trigger autoscaling if needed
# Increase rate limits if hitting limits
```

### Rate Limit Issues

```bash
# Check available tokens
curl -s http://localhost:8082/status | \
  jq '.status.ratelimit_stats.buckets'

# Increase capacity if consistently hitting limits
# Monitor denied requests
curl http://localhost:8082/metrics | grep nlb_ratelimit_denied_total
```

### Module Failures

```bash
# Check module health status
curl -s http://localhost:8082/status | \
  jq '.status.router_stats.protocols[].modules'

# Verify module is running
docker ps | grep module-name

# Check module logs
docker logs module-name

# Manually update health status via gRPC
grpcurl -d '{"module_id":"http-1","status":"UNHEALTHY"}' \
  -plaintext localhost:50051 nlb.NLBService/UpdateHealth
```

---

## Common Operations

### Restart NLB

```bash
# Docker
docker restart marchproxy-nlb

# Docker Compose
docker compose restart nlb

# Local (manual restart)
pkill -f "proxy-nlb"
./proxy-nlb --config config.yaml
```

### Check Configuration

```bash
# Validate configuration file
./proxy-nlb --config config.yaml --dry-run

# View active configuration
curl -s http://localhost:8082/status | jq '.status'
```

### View Detailed Logs

```bash
# Docker logs
docker logs --follow --tail=100 marchproxy-nlb

# With specific log level
LOGLEVEL=DEBUG docker-compose up nlb

# Filter logs
docker logs marchproxy-nlb | grep ERROR
```

### Update Rate Limits (without restart)

Rate limits require restart to change. Workflow:
1. Update configuration file
2. Restart NLB: `docker restart marchproxy-nlb`
3. Verify new limits: `curl http://localhost:8082/status`

---

## Production Best Practices

1. **Always enable health checks** - Detect and recover from failures
2. **Use rate limiting** - Prevent overload and ensure fair usage
3. **Monitor metrics** - Use Prometheus for continuous monitoring
4. **Configure autoscaling** - Respond to load changes automatically
5. **Use blue/green deployments** - Zero-downtime updates
6. **Log everything** - Enable JSON logging for analysis
7. **Test configuration changes** - Validate in staging first
8. **Keep backups** - Save working configurations
9. **Review metrics regularly** - Identify optimization opportunities
10. **Set up alerts** - Monitor for anomalies and failures
