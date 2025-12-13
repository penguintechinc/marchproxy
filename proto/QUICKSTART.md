# MarchProxy gRPC Quickstart

Quick reference for working with MarchProxy gRPC services.

## Generate Code

```bash
# From project root
./scripts/gen-proto.sh
```

This generates:
- Go: `pkg/proto/marchproxy/*.pb.go`
- Python: `manager/proto/marchproxy/*_pb2.py`

## Module Implementation Checklist

Every module (ALB, DBLB, AILB, RTMP) must implement `ModuleService`:

### Required RPCs

- [ ] `GetStatus` - Return module health and status
- [ ] `Reload` - Reload configuration without restart
- [ ] `CanHandle` - Check if route can be handled
- [ ] `GetRoutes` - Return configured routes
- [ ] `UpdateRoutes` - Update routing configuration
- [ ] `GetRateLimits` - Return rate limit configuration
- [ ] `SetRateLimit` - Configure rate limits
- [ ] `GetMetrics` - Return current metrics
- [ ] `Scale` - Handle scaling requests
- [ ] `GetInstances` - Return instance information
- [ ] `SetTrafficWeight` - Set traffic weight for blue/green
- [ ] `GetActiveVersion` - Return active version info
- [ ] `HealthCheck` - Perform health check
- [ ] `GetStats` - Return detailed statistics

### Optional RPCs

- [ ] `Rollback` - Rollback deployment
- [ ] `PromoteVersion` - Promote canary to production
- [ ] `StreamMetrics` - Stream real-time metrics
- [ ] `GetConfig` - Return configuration
- [ ] `UpdateConfig` - Update configuration

## NLB Implementation Checklist

The NLB must implement `NLBService`:

### Core RPCs

- [ ] `RegisterModule` - Accept module registrations
- [ ] `UnregisterModule` - Handle module deregistration
- [ ] `Heartbeat` - Accept module heartbeats
- [ ] `ListModules` - Return all registered modules
- [ ] `UpdateRouting` - Update routing table
- [ ] `GetRoutingTable` - Return routing table
- [ ] `RouteRequest` - Route requests to modules
- [ ] `ReportMetrics` - Accept metrics from modules
- [ ] `CheckHealth` - Perform health checks
- [ ] `GetNLBStatus` - Return NLB status

### Advanced RPCs

- [ ] `RebalanceLoad` - Trigger load rebalancing
- [ ] `GetLoadDistribution` - Return load distribution
- [ ] `TriggerScaling` - Trigger module scaling
- [ ] `StreamNLBMetrics` - Stream aggregated metrics

## Common Message Types

### ModuleInstance
```protobuf
message ModuleInstance {
  string instance_id = 1;
  ModuleType module_type = 2;
  string address = 3;
  HealthStatus health_status = 4;
  int32 current_connections = 7;
  string version = 9;
  int32 traffic_weight = 10;
}
```

### Route
```protobuf
message Route {
  string route_id = 1;
  Protocol protocol = 2;
  string source_pattern = 3;
  string destination_pattern = 4;
  int32 source_port = 5;
  int32 destination_port = 6;
  string path_pattern = 8;
  int32 priority = 9;
}
```

### RateLimit
```protobuf
message RateLimit {
  string limit_id = 1;
  string target = 2;
  int64 requests_per_second = 3;
  int64 burst_size = 6;
  bool enabled = 7;
}
```

## Module Types

```go
MODULE_TYPE_ALB   = 1  // Application Load Balancer
MODULE_TYPE_DBLB  = 2  // Database Load Balancer
MODULE_TYPE_AILB  = 3  // AI Load Balancer
MODULE_TYPE_RTMP  = 4  // RTMP Streaming
MODULE_TYPE_NLB   = 5  // Network Load Balancer
```

## Protocols

```go
PROTOCOL_HTTP       = 3
PROTOCOL_HTTPS      = 4
PROTOCOL_HTTP2      = 5
PROTOCOL_GRPC       = 7
PROTOCOL_WEBSOCKET  = 8
PROTOCOL_MYSQL      = 9
PROTOCOL_POSTGRES   = 10
PROTOCOL_RTMP       = 13
```

## Health Status

```go
HEALTH_STATUS_HEALTHY   = 1
HEALTH_STATUS_DEGRADED  = 2
HEALTH_STATUS_UNHEALTHY = 3
HEALTH_STATUS_STARTING  = 4
HEALTH_STATUS_STOPPING  = 5
```

## Quick Examples

### Go: Start Module Server

```go
import (
    pb "github.com/penguintech/marchproxy/pkg/proto/marchproxy"
    "google.golang.org/grpc"
)

type MyModule struct {
    pb.UnimplementedModuleServiceServer
}

func main() {
    lis, _ := net.Listen("tcp", ":50051")
    server := grpc.NewServer()
    pb.RegisterModuleServiceServer(server, &MyModule{})
    server.Serve(lis)
}
```

### Go: Register with NLB

```go
conn, _ := grpc.Dial("nlb:50050", grpc.WithInsecure())
client := pb.NewNLBServiceClient(conn)

resp, _ := client.RegisterModule(ctx, &pb.RegisterModuleRequest{
    Instance: &pb.ModuleInstance{
        InstanceId:  "alb-001",
        ModuleType:  pb.ModuleType_MODULE_TYPE_ALB,
        Address:     "alb-001:50051",
    },
})
```

### Python: Start Module Server

```python
from proto.marchproxy import module_pb2_grpc
import grpc

class MyModule(module_pb2_grpc.ModuleServiceServicer):
    pass

server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
module_pb2_grpc.add_ModuleServiceServicer_to_server(MyModule(), server)
server.add_insecure_port('[::]:50051')
server.start()
```

### Python: Register with NLB

```python
from proto.marchproxy import nlb_pb2, nlb_pb2_grpc, types_pb2

channel = grpc.insecure_channel('nlb:50050')
client = nlb_pb2_grpc.NLBServiceStub(channel)

response = client.RegisterModule(nlb_pb2.RegisterModuleRequest(
    instance=types_pb2.ModuleInstance(
        instance_id="alb-001",
        module_type=types_pb2.MODULE_TYPE_ALB,
        address="alb-001:50051"
    )
))
```

## Testing

### Unit Tests

Test proto generation:
```bash
./scripts/validate-proto.sh
```

### Integration Tests

Test module registration:
```bash
# Start NLB
docker-compose up -d nlb

# Start module and verify registration
docker-compose up -d alb
docker-compose logs alb | grep "Registered with NLB"
```

## Debugging

### Check Generated Files

```bash
# Go
ls -la pkg/proto/marchproxy/

# Python
ls -la manager/proto/marchproxy/
```

### Test Imports

```bash
# Go
cd pkg/proto/marchproxy && go build

# Python
python3 -c "from manager.proto.marchproxy import types_pb2"
```

### gRPC Server Reflection

Enable reflection for debugging with grpcurl:

```go
import "google.golang.org/grpc/reflection"

reflection.Register(server)
```

Then query:
```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 describe marchproxy.ModuleService
```

## Common Patterns

### Heartbeat Loop

```go
ticker := time.NewTicker(10 * time.Second)
for range ticker.C {
    client.Heartbeat(ctx, &pb.HeartbeatRequest{
        InstanceId: instanceID,
        Status: pb.HealthStatus_HEALTH_STATUS_HEALTHY,
    })
}
```

### Metric Collection

```go
func collectMetrics() *pb.ModuleMetrics {
    return &pb.ModuleMetrics{
        InstanceId: instanceID,
        ModuleType: pb.ModuleType_MODULE_TYPE_ALB,
        Metrics: []*pb.MetricDataPoint{
            {
                Name: "requests_total",
                Value: float64(requestCount),
            },
        },
    }
}
```

### Error Handling

```go
resp, err := client.RegisterModule(ctx, req)
if err != nil {
    log.Fatalf("Registration failed: %v", err)
}
if !resp.Success {
    log.Printf("Registration rejected: %s", resp.Message)
}
```

## Performance Tips

1. **Connection Pooling**: Reuse gRPC connections
2. **Streaming**: Use streaming RPCs for real-time data
3. **Batching**: Batch metric reports instead of per-request
4. **Compression**: Enable gRPC compression for large payloads
5. **Keepalives**: Configure keepalives for long-lived connections

## Troubleshooting

### "unimplemented" errors
- Ensure all required RPCs are implemented
- Return `status.Errorf(codes.Unimplemented, "not implemented")`

### Import errors
- Regenerate code: `./scripts/gen-proto.sh`
- Check go.mod for correct module path
- Verify PYTHONPATH includes manager/

### Connection refused
- Check module is listening on correct port
- Verify Docker network connectivity
- Check firewall rules

## References

- Full docs: `proto/README.md`
- Proto files: `proto/marchproxy/*.proto`
- Examples: `examples/grpc/`
