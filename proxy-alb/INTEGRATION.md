# ALB Integration Guide

This guide explains how to integrate the MarchProxy ALB (Application Load Balancer) with the NLB and other components in the Unified NLB Architecture.

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                  Unified NLB Architecture                     │
├──────────────────────────────────────────────────────────────┤
│                                                                │
│  ┌─────────┐         ┌─────────┐         ┌─────────┐        │
│  │   NLB   │◄───────►│   ALB   │◄───────►│   xDS   │        │
│  │         │  gRPC   │         │  gRPC   │  Server │        │
│  │ Layer 4 │         │ Layer 7 │         │         │        │
│  └─────────┘         └─────────┘         └─────────┘        │
│       │                   │                                   │
│       │                   │                                   │
│       ▼                   ▼                                   │
│  Backend Services    HTTP/HTTPS Traffic                      │
│                                                                │
└──────────────────────────────────────────────────────────────┘
```

## Phase 3 Implementation: ALB Container

The ALB container serves as a critical component in the unified architecture:

1. **L7 Proxy**: Handles HTTP/HTTPS/WebSocket/gRPC traffic via Envoy
2. **ModuleService Interface**: Exposes gRPC API for NLB communication
3. **Dynamic Configuration**: Receives configuration from xDS server
4. **Metrics & Monitoring**: Provides comprehensive observability

## ModuleService gRPC Interface

### Protocol Definition

The ModuleService is defined in `/home/penguin/code/MarchProxy/proto/marchproxy/module_service.proto`:

```protobuf
service ModuleService {
  rpc GetStatus(StatusRequest) returns (StatusResponse);
  rpc GetRoutes(RoutesRequest) returns (RoutesResponse);
  rpc ApplyRateLimit(RateLimitRequest) returns (RateLimitResponse);
  rpc GetMetrics(MetricsRequest) returns (MetricsResponse);
  rpc SetTrafficWeight(TrafficWeightRequest) returns (TrafficWeightResponse);
  rpc Reload(ReloadRequest) returns (ReloadResponse);
}
```

### Go Client Example

```go
package main

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
)

func main() {
    // Connect to ALB
    conn, err := grpc.Dial(
        "localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    client := pb.NewModuleServiceClient(conn)

    // Get ALB status
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    statusResp, err := client.GetStatus(ctx, &pb.StatusRequest{})
    if err != nil {
        log.Fatalf("GetStatus failed: %v", err)
    }

    log.Printf("ALB Status: %v", statusResp)
    log.Printf("  Module ID: %s", statusResp.ModuleId)
    log.Printf("  Health: %v", statusResp.Health)
    log.Printf("  Uptime: %d seconds", statusResp.UptimeSeconds)
}
```

### Python Client Example

```python
import grpc
from proto.marchproxy import module_service_pb2
from proto.marchproxy import module_service_pb2_grpc

def get_alb_status():
    # Connect to ALB
    channel = grpc.insecure_channel('localhost:50051')
    stub = module_service_pb2_grpc.ModuleServiceStub(channel)

    # Get status
    request = module_service_pb2.StatusRequest()
    response = stub.GetStatus(request)

    print(f"ALB Status:")
    print(f"  Module ID: {response.module_id}")
    print(f"  Health: {response.health}")
    print(f"  Uptime: {response.uptime_seconds}s")

if __name__ == '__main__':
    get_alb_status()
```

## NLB Integration Points

### 1. Health Monitoring

The NLB should regularly query ALB health status:

```go
func (nlb *NLB) monitorALBHealth() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        status, err := nlb.albClient.GetStatus(context.Background(), &pb.StatusRequest{})
        if err != nil {
            nlb.logger.WithError(err).Error("Failed to get ALB status")
            nlb.markALBUnhealthy()
            continue
        }

        if status.Health != pb.HealthStatus_HEALTHY {
            nlb.logger.Warn("ALB is unhealthy")
            nlb.markALBUnhealthy()
        } else {
            nlb.markALBHealthy()
        }
    }
}
```

### 2. Route Discovery

NLB queries ALB for available routes:

```go
func (nlb *NLB) syncRoutesFromALB() error {
    resp, err := nlb.albClient.GetRoutes(context.Background(), &pb.RoutesRequest{})
    if err != nil {
        return fmt.Errorf("failed to get routes: %w", err)
    }

    nlb.logger.WithField("count", len(resp.Routes)).Info("Synced routes from ALB")

    for _, route := range resp.Routes {
        nlb.updateRoute(route)
    }

    return nil
}
```

### 3. Dynamic Rate Limiting

NLB can adjust rate limits dynamically:

```go
func (nlb *NLB) applyRateLimitToRoute(routeName string, rps int32, burst int32) error {
    req := &pb.RateLimitRequest{
        RouteName: routeName,
        Config: &pb.RateLimitConfig{
            RequestsPerSecond: rps,
            BurstSize:         burst,
            Enabled:           true,
        },
    }

    resp, err := nlb.albClient.ApplyRateLimit(context.Background(), req)
    if err != nil {
        return fmt.Errorf("failed to apply rate limit: %w", err)
    }

    if !resp.Success {
        return fmt.Errorf("rate limit application failed: %s", resp.Message)
    }

    nlb.logger.WithFields(logrus.Fields{
        "route": routeName,
        "rps":   rps,
    }).Info("Rate limit applied")

    return nil
}
```

### 4. Blue/Green Deployments

NLB controls traffic weights for deployments:

```go
func (nlb *NLB) executeBlueGreenDeployment(routeName string, blueWeight, greenWeight int32) error {
    req := &pb.TrafficWeightRequest{
        RouteName: routeName,
        Weights: []*pb.BackendWeight{
            {
                BackendName: "service-blue",
                Weight:      blueWeight,
                Version:     "blue",
            },
            {
                BackendName: "service-green",
                Weight:      greenWeight,
                Version:     "green",
            },
        },
    }

    resp, err := nlb.albClient.SetTrafficWeight(context.Background(), req)
    if err != nil {
        return fmt.Errorf("failed to set traffic weight: %w", err)
    }

    if !resp.Success {
        return fmt.Errorf("traffic weight update failed: %s", resp.Message)
    }

    nlb.logger.WithFields(logrus.Fields{
        "route":       routeName,
        "blue_weight": blueWeight,
        "green_weight": greenWeight,
    }).Info("Traffic weights updated")

    return nil
}
```

### 5. Metrics Collection

NLB aggregates metrics from ALB:

```go
func (nlb *NLB) collectALBMetrics() (*pb.MetricsResponse, error) {
    req := &pb.MetricsRequest{
        StartTimestamp: time.Now().Add(-5 * time.Minute).Unix(),
        EndTimestamp:   time.Now().Unix(),
    }

    metrics, err := nlb.albClient.GetMetrics(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("failed to get metrics: %w", err)
    }

    nlb.logger.WithFields(logrus.Fields{
        "total_requests": metrics.TotalRequests,
        "active_conns":   metrics.ActiveConnections,
        "rps":            metrics.RequestsPerSecond,
        "p99_latency":    metrics.Latency.P99Ms,
    }).Debug("Collected ALB metrics")

    return metrics, nil
}
```

## xDS Integration

The ALB connects to the xDS server for dynamic configuration:

### xDS Flow

```
API Server → xDS Server (gRPC:18000) → ALB Envoy
                                         ↓
                                   Hot Reload
```

### Configuration Update

1. **API Server** receives configuration change via REST API
2. **xDS Server** generates new snapshot with incremented version
3. **ALB Envoy** receives xDS update via ADS (Aggregated Discovery Service)
4. **Envoy** validates and hot-reloads configuration
5. **ALB Supervisor** detects reload and updates internal state

## Deployment Patterns

### Pattern 1: Single ALB

```yaml
services:
  nlb:
    image: marchproxy/nlb:latest
    environment:
      - ALB_ENDPOINTS=alb:50051

  alb:
    image: marchproxy/proxy-alb:latest
    environment:
      - MODULE_ID=alb-1
      - XDS_SERVER=api-server:18000
```

### Pattern 2: Multiple ALBs (High Availability)

```yaml
services:
  nlb:
    image: marchproxy/nlb:latest
    environment:
      - ALB_ENDPOINTS=alb-1:50051,alb-2:50051,alb-3:50051

  alb-1:
    image: marchproxy/proxy-alb:latest
    environment:
      - MODULE_ID=alb-1

  alb-2:
    image: marchproxy/proxy-alb:latest
    environment:
      - MODULE_ID=alb-2

  alb-3:
    image: marchproxy/proxy-alb:latest
    environment:
      - MODULE_ID=alb-3
```

### Pattern 3: Regional Deployment

```yaml
services:
  nlb-us-east:
    environment:
      - ALB_ENDPOINTS=alb-us-east-1:50051,alb-us-east-2:50051

  nlb-eu-west:
    environment:
      - ALB_ENDPOINTS=alb-eu-west-1:50051,alb-eu-west-2:50051

  # ALB instances per region
  alb-us-east-1:
    environment:
      - MODULE_ID=alb-us-east-1
      - XDS_SERVER=api-server-us-east:18000

  alb-eu-west-1:
    environment:
      - MODULE_ID=alb-eu-west-1
      - XDS_SERVER=api-server-eu-west:18000
```

## Error Handling

### Connection Failures

```go
func (nlb *NLB) connectToALB(addr string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    conn, err := grpc.DialContext(ctx, addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                30 * time.Second,
            Timeout:             5 * time.Second,
            PermitWithoutStream: true,
        }),
    )
    if err != nil {
        return fmt.Errorf("failed to connect to ALB: %w", err)
    }

    nlb.albConn = conn
    nlb.albClient = pb.NewModuleServiceClient(conn)

    return nil
}
```

### Retry Logic

```go
func (nlb *NLB) getALBStatusWithRetry(maxRetries int) (*pb.StatusResponse, error) {
    var lastErr error

    for i := 0; i < maxRetries; i++ {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        resp, err := nlb.albClient.GetStatus(ctx, &pb.StatusRequest{})
        cancel()

        if err == nil {
            return resp, nil
        }

        lastErr = err
        nlb.logger.WithError(err).Warnf("GetStatus attempt %d/%d failed", i+1, maxRetries)

        // Exponential backoff
        time.Sleep(time.Duration(1<<uint(i)) * time.Second)
    }

    return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}
```

## Performance Considerations

### gRPC Connection Pooling

```go
type ALBClientPool struct {
    mu      sync.RWMutex
    clients map[string]pb.ModuleServiceClient
    conns   map[string]*grpc.ClientConn
}

func (p *ALBClientPool) GetClient(addr string) (pb.ModuleServiceClient, error) {
    p.mu.RLock()
    if client, ok := p.clients[addr]; ok {
        p.mu.RUnlock()
        return client, nil
    }
    p.mu.RUnlock()

    p.mu.Lock()
    defer p.mu.Unlock()

    // Double-check
    if client, ok := p.clients[addr]; ok {
        return client, nil
    }

    conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        return nil, err
    }

    client := pb.NewModuleServiceClient(conn)
    p.clients[addr] = client
    p.conns[addr] = conn

    return client, nil
}
```

### Metrics Caching

The ALB caches metrics for 5 seconds to reduce load. Configure this via:

```go
metricsCollector := metrics.NewCollector(adminAddr, logger)
metricsCollector.SetCacheTimeout(10 * time.Second) // Custom cache timeout
```

## Testing

### Unit Tests

```bash
cd proxy-alb
go test ./...
```

### Integration Tests

```bash
# Start test environment
docker-compose -f docker-compose.example.yml up -d

# Run integration tests
go test -tags=integration ./tests/integration/...
```

### Load Testing

```bash
# Test gRPC endpoint
ghz --insecure \
    --proto proto/marchproxy/module_service.proto \
    --call marchproxy.ModuleService/GetMetrics \
    --duration 60s \
    --connections 50 \
    --concurrency 100 \
    localhost:50051

# Test HTTP traffic through ALB
wrk -t12 -c400 -d60s http://localhost:10000/api/test
```

## Troubleshooting

### gRPC Connection Refused

**Problem**: NLB cannot connect to ALB gRPC server

**Solution**:
```bash
# Check if ALB is running
docker ps | grep alb

# Check gRPC port
docker port marchproxy-alb 50051

# Test with grpcurl
grpcurl -plaintext localhost:50051 list
```

### xDS Configuration Not Updating

**Problem**: ALB not receiving configuration updates

**Solution**:
```bash
# Check xDS connectivity
docker exec marchproxy-alb wget -O- http://api-server:18000/healthz

# View Envoy config dump
curl http://localhost:9901/config_dump

# Check Envoy logs
docker logs marchproxy-alb | grep xDS
```

### High Latency

**Problem**: Slow gRPC responses

**Solution**:
```bash
# Check metrics collection interval
# Increase cache timeout to reduce load

# Monitor gRPC server
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetMetrics
```

## Best Practices

1. **Health Checks**: Query GetStatus every 10-30 seconds
2. **Connection Reuse**: Use connection pooling for multiple ALB instances
3. **Timeout Configuration**: Set appropriate timeouts (5-10s recommended)
4. **Error Handling**: Implement exponential backoff for retries
5. **Metrics Collection**: Cache metrics to avoid overloading Envoy admin API
6. **Graceful Shutdown**: Always close gRPC connections properly
7. **TLS**: Use TLS for production deployments
8. **Load Balancing**: Distribute gRPC calls across multiple ALB instances

## Next Steps

After implementing ALB integration:

1. **Phase 4**: Implement NLB container with L4 load balancing
2. **Phase 5**: Add service mesh capabilities
3. **Phase 6**: Implement advanced traffic management (canary, A/B testing)
4. **Phase 7**: Add observability and distributed tracing

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Envoy Proxy](https://www.envoyproxy.io/)
- [xDS Protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol)
- [Protocol Buffers](https://developers.google.com/protocol-buffers)
