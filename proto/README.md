# MarchProxy gRPC Protocol Definitions

This directory contains the Protocol Buffer (protobuf) definitions for the MarchProxy Unified NLB Architecture.

## Overview

MarchProxy uses gRPC for all inter-container communication in the Unified NLB Architecture:

- **NLB Container**: Routes L4 traffic to specialized module containers
- **Module Containers**: ALB, DBLB, AILB, RTMP - handle specialized L7 traffic
- **Manager Container**: Configuration and management (existing API, not proto-based)

## Proto Files

### `marchproxy/types.proto`

Common message types and enumerations used across all services:

- **Enums**: `StatusCode`, `ModuleType`, `Protocol`, `HealthStatus`
- **Core Types**: `ModuleInstance`, `Route`, `RateLimit`, `MetricDataPoint`
- **Config Types**: `ScalingConfig`, `BlueGreenConfig`, `HealthCheckConfig`
- **Stats Types**: `ModuleMetrics`, `ModuleStats`
- **Utility Types**: `Error`, `Response`, `Pagination`, `Filter`, `Query`

### `marchproxy/module.proto`

Defines `ModuleService` - the interface that all module containers (ALB, DBLB, AILB, RTMP) must implement.

**Service Categories**:

1. **Lifecycle Management**
   - `GetStatus` - Get module instance status
   - `Reload` - Reload configuration without restart
   - `Shutdown` - Graceful shutdown

2. **Traffic Routing**
   - `CanHandle` - Check if module can handle a route
   - `GetRoutes` - List configured routes
   - `UpdateRoutes` - Update routing configuration
   - `DeleteRoute` - Remove a route

3. **Rate Limiting**
   - `GetRateLimits` - List rate limits
   - `SetRateLimit` - Configure rate limit
   - `RemoveRateLimit` - Remove rate limit

4. **Scaling and Instance Management**
   - `GetMetrics` - Retrieve module metrics
   - `Scale` - Adjust scaling configuration
   - `GetInstances` - List module instances

5. **Blue/Green Deployment**
   - `SetTrafficWeight` - Set traffic weight for version
   - `GetActiveVersion` - Get active version info
   - `Rollback` - Rollback to previous version
   - `PromoteVersion` - Promote canary to production

6. **Health and Monitoring**
   - `HealthCheck` - Perform health check
   - `GetStats` - Get detailed statistics
   - `StreamMetrics` - Stream real-time metrics (server streaming)

7. **Configuration Management**
   - `GetConfig` - Retrieve configuration
   - `UpdateConfig` - Update configuration

### `marchproxy/nlb.proto`

Defines `NLBService` - the service provided by the NLB container for module registration and coordination.

**Service Categories**:

1. **Module Registration and Discovery**
   - `RegisterModule` - Register new module instance
   - `UnregisterModule` - Unregister module instance
   - `Heartbeat` - Periodic heartbeat from modules
   - `ListModules` - List all registered modules
   - `GetModuleInfo` - Get detailed module information

2. **Routing Management**
   - `UpdateRouting` - Update NLB routing table
   - `GetRoutingTable` - Get current routing table
   - `RouteRequest` - Route a single request
   - `ValidateRoute` - Validate route configuration

3. **Metrics and Monitoring**
   - `ReportMetrics` - Modules report metrics to NLB
   - `GetNLBMetrics` - Get aggregated NLB metrics
   - `GetModuleMetrics` - Get specific module metrics
   - `StreamNLBMetrics` - Stream real-time NLB metrics

4. **Health and Status**
   - `CheckHealth` - Perform NLB health check
   - `GetNLBStatus` - Get NLB status and info

5. **Configuration Management**
   - `UpdateNLBConfig` - Update NLB configuration
   - `GetNLBConfig` - Get NLB configuration

6. **Load Balancing and Scaling**
   - `RebalanceLoad` - Trigger load rebalancing
   - `GetLoadDistribution` - Get load distribution
   - `TriggerScaling` - Trigger module scaling

## Code Generation

### Prerequisites

**Go**:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

**Python**:
```bash
pip3 install grpcio-tools
```

**protoc compiler**:
- Debian/Ubuntu: `sudo apt-get install -y protobuf-compiler`
- macOS: `brew install protobuf`
- Or download from: https://github.com/protocolbuffers/protobuf/releases

### Generate Code

Run the generation script from the project root:

```bash
./scripts/gen-proto.sh
```

This will generate:

- **Go code**: `pkg/proto/marchproxy/*.pb.go` and `*_grpc.pb.go`
- **Python code**: `manager/proto/marchproxy/*_pb2.py` and `*_pb2_grpc.py`

## Usage Examples

### Go - Implementing ModuleService

```go
package main

import (
    "context"
    pb "github.com/penguintech/marchproxy/pkg/proto/marchproxy"
    "google.golang.org/grpc"
)

type ALBModule struct {
    pb.UnimplementedModuleServiceServer
}

func (m *ALBModule) GetStatus(ctx context.Context, req *pb.GetStatusRequest) (*pb.GetStatusResponse, error) {
    return &pb.GetStatusResponse{
        Instance: &pb.ModuleInstance{
            InstanceId:  "alb-001",
            ModuleType:  pb.ModuleType_MODULE_TYPE_ALB,
            Address:     "alb-001:50051",
            HealthStatus: pb.HealthStatus_HEALTH_STATUS_HEALTHY,
        },
        Status: pb.HealthStatus_HEALTH_STATUS_HEALTHY,
        Message: "ALB module is healthy",
    }, nil
}

// Implement other ModuleService methods...
```

### Go - Client Calling NLB

```go
package main

import (
    "context"
    pb "github.com/penguintech/marchproxy/pkg/proto/marchproxy"
    "google.golang.org/grpc"
)

func registerWithNLB(nlbAddress string) error {
    conn, err := grpc.Dial(nlbAddress, grpc.WithInsecure())
    if err != nil {
        return err
    }
    defer conn.Close()

    client := pb.NewNLBServiceClient(conn)

    req := &pb.RegisterModuleRequest{
        Instance: &pb.ModuleInstance{
            InstanceId:  "alb-001",
            ModuleType:  pb.ModuleType_MODULE_TYPE_ALB,
            Address:     "alb-001:50051",
        },
        InitialRoutes: []*pb.Route{
            {
                RouteId:  "http-default",
                Protocol: pb.Protocol_PROTOCOL_HTTPS,
                DestinationPort: 443,
            },
        },
    }

    resp, err := client.RegisterModule(context.Background(), req)
    if err != nil {
        return err
    }

    log.Printf("Registered with ID: %s", resp.RegistrationId)
    return nil
}
```

### Python - Implementing ModuleService

```python
from concurrent import futures
import grpc
from proto.marchproxy import module_pb2, module_pb2_grpc, types_pb2

class ALBModule(module_pb2_grpc.ModuleServiceServicer):
    def GetStatus(self, request, context):
        return module_pb2.GetStatusResponse(
            instance=types_pb2.ModuleInstance(
                instance_id="alb-001",
                module_type=types_pb2.MODULE_TYPE_ALB,
                address="alb-001:50051",
                health_status=types_pb2.HEALTH_STATUS_HEALTHY
            ),
            status=types_pb2.HEALTH_STATUS_HEALTHY,
            message="ALB module is healthy"
        )

    # Implement other methods...

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    module_pb2_grpc.add_ModuleServiceServicer_to_server(ALBModule(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()
```

### Python - Client Calling NLB

```python
import grpc
from proto.marchproxy import nlb_pb2, nlb_pb2_grpc, types_pb2
from google.protobuf.timestamp_pb2 import Timestamp

def register_with_nlb(nlb_address: str):
    channel = grpc.insecure_channel(nlb_address)
    client = nlb_pb2_grpc.NLBServiceStub(channel)

    request = nlb_pb2.RegisterModuleRequest(
        instance=types_pb2.ModuleInstance(
            instance_id="alb-001",
            module_type=types_pb2.MODULE_TYPE_ALB,
            address="alb-001:50051"
        ),
        initial_routes=[
            types_pb2.Route(
                route_id="http-default",
                protocol=types_pb2.PROTOCOL_HTTPS,
                destination_port=443
            )
        ]
    )

    response = client.RegisterModule(request)
    print(f"Registered with ID: {response.registration_id}")
```

## Architecture Flow

### Module Registration Flow

1. Module container starts and initializes
2. Module calls `NLBService.RegisterModule()` with instance info and initial routes
3. NLB acknowledges registration and returns registration ID
4. Module starts sending periodic heartbeats via `NLBService.Heartbeat()`
5. NLB monitors module health and updates routing table

### Request Routing Flow

1. NLB receives L4 connection
2. NLB inspects protocol and determines routing based on routing table
3. NLB calls `ModuleService.CanHandle()` on potential modules
4. Module responds with priority and capability
5. NLB selects best module and forwards connection
6. Module processes request and returns response

### Scaling Flow

1. NLB monitors module metrics via heartbeats
2. NLB detects resource thresholds exceeded
3. NLB calls `ModuleService.Scale()` with new configuration
4. Module spawns/terminates instances as needed
5. New instances register with NLB
6. NLB rebalances load across instances

### Blue/Green Deployment Flow

1. New version of module deployed alongside current version
2. Module calls `ModuleService.SetTrafficWeight()` to set canary traffic (e.g., 5%)
3. NLB routes small percentage of traffic to new version
4. Metrics monitored for errors/latency
5. If successful, call `ModuleService.PromoteVersion()` to increase traffic
6. Gradually shift 100% traffic to new version
7. Old version can be decommissioned or kept for rollback

## Message Design Principles

1. **Consistency**: All messages follow consistent naming and structure
2. **Extensibility**: Use `map<string, string> metadata` for future extensibility
3. **Timestamps**: Use `google.protobuf.Timestamp` for all time fields
4. **Errors**: Consistent error handling with `Error` message type
5. **Pagination**: Support for large result sets via `Pagination` and `Query`
6. **Versioning**: Include version fields for backward compatibility
7. **Metrics**: Comprehensive metrics collection at all levels

## Proto Best Practices

1. **Never remove fields**: Mark as deprecated instead
2. **Don't reuse field numbers**: Even for deleted fields
3. **Use reserved for removed fields**: Prevent accidental reuse
4. **Add new fields to end**: Maintains backward compatibility
5. **Use enums for fixed sets**: Better than strings for known values
6. **Always set default values**: Ensure consistent behavior
7. **Document everything**: Clear comments for all messages and fields

## Testing

Test proto generation:

```bash
# Generate code
./scripts/gen-proto.sh

# Verify Go compilation
cd pkg/proto/marchproxy && go build

# Verify Python imports
python3 -c "from manager.proto.marchproxy import types_pb2, module_pb2, nlb_pb2"
```

## Version History

- **v1.0.0** - Initial proto definitions for Unified NLB Architecture
  - Module service with lifecycle, routing, scaling, blue/green
  - NLB service with registration, routing, metrics
  - Comprehensive type system

## Contributing

When modifying proto files:

1. Update the proto files
2. Run `./scripts/gen-proto.sh` to regenerate code
3. Update this README if adding new services or messages
4. Test both Go and Python implementations
5. Update version history

## References

- [Protocol Buffers Documentation](https://protobuf.dev/)
- [gRPC Documentation](https://grpc.io/docs/)
- [gRPC Go Tutorial](https://grpc.io/docs/languages/go/quickstart/)
- [gRPC Python Tutorial](https://grpc.io/docs/languages/python/quickstart/)
