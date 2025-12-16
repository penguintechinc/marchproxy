# MarchProxy Unified NLB Architecture - gRPC Communication Diagram

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         MarchProxy Unified NLB                          │
│                                                                         │
│  ┌───────────┐                                                         │
│  │  Manager  │  Python/py4web - Configuration & Web UI                │
│  │ Container │  (Existing - Not using gRPC protos)                     │
│  └─────┬─────┘                                                         │
│        │                                                               │
│        │ HTTP API                                                      │
│        ↓                                                               │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │                    NLB Container                              │   │
│  │                    (Layer 4 Router)                           │   │
│  │                                                               │   │
│  │  Implements: NLBService (21 RPCs)                            │   │
│  │  - Module registration & discovery                           │   │
│  │  - Routing table management                                  │   │
│  │  - Metrics aggregation                                       │   │
│  │  - Load balancing & scaling                                  │   │
│  │                                                               │   │
│  │  gRPC Server: :50050                                         │   │
│  └─────┬────────────┬────────────┬────────────┬────────────────┘   │
│        │            │            │            │                     │
│        │ gRPC       │ gRPC       │ gRPC       │ gRPC                │
│        ↓            ↓            ↓            ↓                     │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐               │
│  │   ALB   │  │  DBLB   │  │  AILB   │  │  RTMP   │               │
│  │ Module  │  │ Module  │  │ Module  │  │ Module  │               │
│  ├─────────┤  ├─────────┤  ├─────────┤  ├─────────┤               │
│  │ HTTP(S) │  │Database │  │ AI/ML   │  │Streaming│               │
│  │   L7    │  │   L7    │  │   L7    │  │   L7    │               │
│  ├─────────┤  ├─────────┤  ├─────────┤  ├─────────┤               │
│  │Implements│  │Implements│  │Implements│  │Implements│             │
│  │ Module  │  │ Module  │  │ Module  │  │ Module  │               │
│  │ Service │  │ Service │  │ Service │  │ Service │               │
│  │(25 RPCs)│  │(25 RPCs)│  │(25 RPCs)│  │(25 RPCs)│               │
│  │         │  │         │  │         │  │         │               │
│  │:50051   │  │:50052   │  │:50053   │  │:50054   │               │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Communication Patterns

### 1. Module Registration (Startup)

```
Module Container                        NLB Container
     │                                       │
     │  RegisterModule(instance, routes)    │
     ├──────────────────────────────────────>│
     │                                       │
     │  RegisterModuleResponse(reg_id)      │
     │<──────────────────────────────────────┤
     │                                       │
     │  Start Heartbeat Loop                │
     │  (every 10 seconds)                   │
     │                                       │
     │  Heartbeat(instance_id, stats)       │
     ├──────────────────────────────────────>│
     │                                       │
     │  HeartbeatResponse(ack, instructions)│
     │<──────────────────────────────────────┤
     │                                       │
```

### 2. Traffic Routing (Request Flow)

```
Client → [NLB Container] → [Module Container] → Backend
          │                 │
          │                 │
    1. Receive L4          4. Process L7
       connection             request
          │                 │
    2. Check routing       5. Route to
       table                  backend
          │                 │
    3. Call CanHandle()    6. Return
       on modules             response
          │                 │
       Select module        │
       based on priority    │
          │                 │
       Forward connection ──┘
```

### 3. Metrics Flow (Monitoring)

```
Module Container                        NLB Container
     │                                       │
     │  Heartbeat(stats)                    │
     ├──────────────────────────────────────>│
     │                                       │ Aggregate
     │                                       │ Metrics
     │  ReportMetrics(detailed_metrics)     │
     ├──────────────────────────────────────>│
     │                                       │
     │  StreamMetrics() [server streaming]  │
     ├──────────────────────────────────────>│ Real-time
     │                                       │ Monitoring
     │  ← metrics ← metrics ← metrics ←     │
     │                                       │
```

### 4. Scaling Flow (Auto-scaling)

```
NLB Container                          Module Container
     │                                       │
     │ Monitor metrics from heartbeats      │
     │ Detect threshold breach              │
     │                                       │
     │  Scale(scaling_config)               │
     ├──────────────────────────────────────>│
     │                                       │ Spawn new
     │                                       │ instances
     │                                       │
     │  ScaleResponse(current, desired)     │
     │<──────────────────────────────────────┤
     │                                       │
     │                            ┌──────────┴──────────┐
     │                            │                     │
     │  RegisterModule()     RegisterModule()     RegisterModule()
     │<───────────────────────────┤                     │
     │<──────────────────────────────────────────────────┤
     │                            │                     │
     │  Update routing table      │                     │
     │  with new instances        │                     │
     │                            │                     │
```

### 5. Blue/Green Deployment Flow

```
Module Container (Blue v1.0)           NLB Container            Module Container (Green v2.0)
     │                                       │                            │
     │                                       │  RegisterModule()         │
     │                                       │<───────────────────────────┤
     │                                       │  (version: "2.0")         │
     │                                       │                            │
     │                                       │  SetTrafficWeight()       │
     │                                       │<───────────────────────────┤
     │                                       │  (version: "2.0", 5%)     │
     │                                       │                            │
     │ ← 95% traffic ─────────────┬─────────┘                            │
     │                            │                                       │
     │                            └───── 5% traffic ──────────────────────>│
     │                                       │                            │
     │                                       │  Monitor metrics          │
     │                                       │  No errors detected        │
     │                                       │                            │
     │                                       │  PromoteVersion()         │
     │                                       │<───────────────────────────┤
     │                                       │  (gradual: 25%, 50%, 100%)│
     │                                       │                            │
     │ ← 50% traffic ─────────────┬─────────┘                            │
     │                            └───── 50% traffic ──────────────────────>│
     │                                       │                            │
     │                                       │  PromoteVersion()         │
     │                                       │<───────────────────────────┤
     │                                       │  (100%)                   │
     │                                       │                            │
     │ ← 0% traffic ──────────────┬─────────┘                            │
     │                            └───── 100% traffic ─────────────────────>│
     │                                       │                            │
     │  UnregisterModule()                  │                            │
     ├──────────────────────────────────────>│                            │
     │  (graceful: true)                    │                            │
     │                                       │                            │
```

## gRPC Service Interfaces

### NLBService (Implemented by NLB Container)

```
service NLBService {
  // Module Registration (5 RPCs)
  RegisterModule, UnregisterModule, Heartbeat,
  ListModules, GetModuleInfo

  // Routing Management (4 RPCs)
  UpdateRouting, GetRoutingTable,
  RouteRequest, ValidateRoute

  // Metrics & Monitoring (4 RPCs)
  ReportMetrics, GetNLBMetrics,
  GetModuleMetrics, StreamNLBMetrics

  // Health & Status (2 RPCs)
  CheckHealth, GetNLBStatus

  // Configuration (2 RPCs)
  UpdateNLBConfig, GetNLBConfig

  // Load Balancing & Scaling (3 RPCs)
  RebalanceLoad, GetLoadDistribution,
  TriggerScaling
}
```

### ModuleService (Implemented by All Modules)

```
service ModuleService {
  // Lifecycle (3 RPCs)
  GetStatus, Reload, Shutdown

  // Traffic Routing (4 RPCs)
  CanHandle, GetRoutes,
  UpdateRoutes, DeleteRoute

  // Rate Limiting (3 RPCs)
  GetRateLimits, SetRateLimit,
  RemoveRateLimit

  // Scaling (3 RPCs)
  GetMetrics, Scale, GetInstances

  // Blue/Green (4 RPCs)
  SetTrafficWeight, GetActiveVersion,
  Rollback, PromoteVersion

  // Health & Monitoring (3 RPCs)
  HealthCheck, GetStats, StreamMetrics

  // Configuration (2 RPCs)
  GetConfig, UpdateConfig
}
```

## Data Flow: Complete Request Lifecycle

```
1. Client Request
   │
   ↓
2. NLB Container (L4)
   ├─→ Check routing table
   ├─→ Call CanHandle() on candidate modules
   ├─→ Receive priority scores
   ├─→ Select highest priority module
   └─→ Forward connection
       │
       ↓
3. Module Container (L7)
   ├─→ Receive connection
   ├─→ Parse L7 protocol
   ├─→ Apply rate limits
   ├─→ Check route configuration
   ├─→ Forward to backend
   ├─→ Collect metrics
   └─→ Return response
       │
       ↓
4. Backend Processing
   │
   ↓
5. Response Path (reverse)
   │
   ↓
6. Metrics Reporting
   ├─→ Module tracks request stats
   ├─→ Report via Heartbeat (10s)
   ├─→ Report via ReportMetrics (batch)
   └─→ NLB aggregates and stores
```

## Module Type Routing Decision Tree

```
NLB Receives Connection
│
├─→ Protocol = HTTP/HTTPS?
│   └─→ Call ALB.CanHandle()
│       ├─→ Check path patterns
│       ├─→ Check host headers
│       └─→ Return priority score
│
├─→ Protocol = MySQL/PostgreSQL/MongoDB?
│   └─→ Call DBLB.CanHandle()
│       ├─→ Check database type
│       ├─→ Check connection metadata
│       └─→ Return priority score
│
├─→ Protocol = AI/ML specific?
│   └─→ Call AILB.CanHandle()
│       ├─→ Check model requirements
│       ├─→ Check GPU availability
│       └─→ Return priority score
│
└─→ Protocol = RTMP/HLS/DASH?
    └─→ Call RTMP.CanHandle()
        ├─→ Check stream type
        ├─→ Check bandwidth availability
        └─→ Return priority score
```

## Health Check Hierarchy

```
Manager Container (Web UI)
│
└─→ GET /healthz
    │
    └─→ Checks: Database, NLB status
        │
        └─→ NLB.CheckHealth()
            │
            ├─→ ALB.HealthCheck()
            │   ├─→ Upstream checks
            │   ├─→ Connection pool status
            │   └─→ Return HEALTHY/DEGRADED/UNHEALTHY
            │
            ├─→ DBLB.HealthCheck()
            │   ├─→ Database connection checks
            │   ├─→ Query performance
            │   └─→ Return HEALTHY/DEGRADED/UNHEALTHY
            │
            ├─→ AILB.HealthCheck()
            │   ├─→ Model availability
            │   ├─→ GPU status
            │   └─→ Return HEALTHY/DEGRADED/UNHEALTHY
            │
            └─→ RTMP.HealthCheck()
                ├─→ Stream availability
                ├─→ Bandwidth capacity
                └─→ Return HEALTHY/DEGRADED/UNHEALTHY
```

## Scaling Decision Flow

```
NLB Monitoring Loop (every 30s)
│
├─→ Aggregate metrics from all module heartbeats
│
├─→ For each module type:
│   │
│   ├─→ Calculate average metrics
│   │   ├─→ CPU usage
│   │   ├─→ Memory usage
│   │   ├─→ Connections per instance
│   │   └─→ Request latency
│   │
│   ├─→ Compare against scaling config thresholds
│   │
│   ├─→ Scale up if:
│   │   ├─→ CPU > scale_up_threshold (e.g., 80%)
│   │   ├─→ Memory > scale_up_threshold
│   │   ├─→ Connections > target_connections
│   │   └─→ Not in cooldown period
│   │
│   ├─→ Scale down if:
│   │   ├─→ CPU < scale_down_threshold (e.g., 30%)
│   │   ├─→ Memory < scale_down_threshold
│   │   ├─→ Connections < (target_connections / 2)
│   │   ├─→ current_instances > min_instances
│   │   └─→ Not in cooldown period
│   │
│   └─→ If scaling needed:
│       ├─→ Call Module.Scale(new_config)
│       ├─→ Module spawns/terminates instances
│       ├─→ New instances register with NLB
│       ├─→ NLB updates routing table
│       └─→ Set cooldown timer
```

## Error Handling Flow

```
Module Operation
│
├─→ Success?
│   ├─→ Yes: Return Response(success=true, message=...)
│   │
│   └─→ No: Return Response(
│           success=false,
│           error=Error(
│               code=STATUS_CODE_ERROR,
│               message="Error description",
│               details={"key": "value"},
│               occurred_at=timestamp
│           )
│       )
│
└─→ NLB receives response
    │
    ├─→ success=true: Continue normally
    │
    └─→ success=false:
        ├─→ Log error
        ├─→ Update module health status
        ├─→ Retry on different instance (if available)
        ├─→ Return error to client (if no retry)
        └─→ Trigger scaling if errors exceed threshold
```

## Network Ports

```
Container           Port    Protocol  Purpose
─────────────────────────────────────────────────────────
Manager             8000    HTTP      Web UI & API
NLB                 50050   gRPC      NLBService
ALB Module          50051   gRPC      ModuleService
DBLB Module         50052   gRPC      ModuleService
AILB Module         50053   gRPC      ModuleService
RTMP Module         50054   gRPC      ModuleService

Client Traffic Ports (handled by modules):
ALB                 80/443  HTTP(S)   Web traffic
DBLB                3306    MySQL     Database traffic
DBLB                5432    PostgreSQL Database traffic
RTMP                1935    RTMP      Streaming traffic
```

## Proto File Organization

```
proto/
├── marchproxy/
│   ├── types.proto      ← Shared types (enums, common messages)
│   ├── module.proto     ← ModuleService (25 RPCs)
│   └── nlb.proto        ← NLBService (21 RPCs)
│
└── Generated code:
    ├── Go → pkg/proto/marchproxy/
    │   ├── types.pb.go
    │   ├── module.pb.go
    │   ├── module_grpc.pb.go
    │   ├── nlb.pb.go
    │   └── nlb_grpc.pb.go
    │
    └── Python → manager/proto/marchproxy/
        ├── types_pb2.py
        ├── module_pb2.py
        ├── module_pb2_grpc.py
        ├── nlb_pb2.py
        └── nlb_pb2_grpc.py
```

## Summary

This architecture provides:
- **Clear separation**: L4 (NLB) vs L7 (modules)
- **Protocol-specific routing**: ALB, DBLB, AILB, RTMP
- **Dynamic registration**: Modules register at startup
- **Health monitoring**: Heartbeats and health checks
- **Auto-scaling**: Based on metrics and thresholds
- **Blue/green deployments**: Gradual traffic shifting
- **Comprehensive metrics**: Real-time and historical
- **Graceful operations**: Reload, shutdown, drain

All communication uses gRPC with 46 total RPC methods across 2 services.
