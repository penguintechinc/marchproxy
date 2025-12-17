# MarchProxy ALB - API Documentation

## Overview

The MarchProxy ALB (Application Load Balancer) exposes two primary interfaces:

1. **gRPC API** (ModuleService) - Configuration and control plane
2. **REST Endpoints** - Health checks and metrics

## gRPC API (ModuleService)

The ALB implements the `ModuleService` interface defined in `proto/marchproxy/module_service.proto`.

### Service Port
- Default: `50051`
- Environment variable: `GRPC_PORT`

### Endpoints

#### GetStatus
Retrieve the current health and operational status of the ALB.

**Request:**
```protobuf
message StatusRequest {
}
```

**Response:**
```protobuf
message StatusResponse {
  string module_id = 1;
  string module_type = 2;
  string version = 3;
  HealthStatus health = 4;
  int64 uptime_seconds = 5;
  string envoy_version = 6;
  map<string, string> metadata = 7;
}

enum HealthStatus {
  HEALTHY = 0;
  UNHEALTHY = 1;
}
```

**Example:**
```bash
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus
```

#### GetRoutes
Fetch the current route configuration from xDS server.

**Request:**
```protobuf
message RoutesRequest {
  string cluster_id = 1;
}
```

**Response:**
```protobuf
message RoutesResponse {
  repeated RouteConfig routes = 1;
  int64 version = 2;
}

message RouteConfig {
  string name = 1;
  string prefix = 2;
  string cluster_name = 3;
  repeated string hosts = 4;
  int32 timeout_seconds = 5;
  map<string, string> headers = 6;
  bool enabled = 7;
  RateLimitConfig rate_limit = 8;
}
```

#### ApplyRateLimit
Apply or update rate limiting configuration for a route.

**Request:**
```protobuf
message RateLimitRequest {
  string route_name = 1;
  RateLimitConfig config = 2;
}

message RateLimitConfig {
  int32 requests_per_second = 1;
  int32 burst_size = 2;
  bool enabled = 3;
}
```

**Response:**
```protobuf
message RateLimitResponse {
  bool success = 1;
  string message = 2;
}
```

**Example:**
```bash
grpcurl -plaintext -d '{"route_name":"api-route","config":{"requests_per_second":1000,"burst_size":2000,"enabled":true}}' \
  localhost:50051 marchproxy.ModuleService/ApplyRateLimit
```

#### GetMetrics
Retrieve current performance metrics from Envoy.

**Request:**
```protobuf
message MetricsRequest {
}
```

**Response:**
```protobuf
message MetricsResponse {
  int64 timestamp = 1;
  int64 total_connections = 2;
  int64 active_connections = 3;
  int64 total_requests = 4;
  int64 requests_per_second = 5;
  LatencyMetrics latency = 6;
  map<string, int64> status_codes = 7;
  map<string, RouteMetrics> routes = 8;
}

message LatencyMetrics {
  double p50_ms = 1;
  double p90_ms = 2;
  double p95_ms = 3;
  double p99_ms = 4;
  double avg_ms = 5;
}

message RouteMetrics {
  int64 requests = 1;
  int64 errors = 2;
  double avg_latency_ms = 3;
}
```

#### SetTrafficWeight
Set traffic weights for blue/green or canary deployments.

**Request:**
```protobuf
message TrafficWeightRequest {
  string route_name = 1;
  repeated TrafficWeight weights = 2;
}

message TrafficWeight {
  string backend_name = 1;
  int32 weight = 2;
}
```

**Response:**
```protobuf
message TrafficWeightResponse {
  bool success = 1;
  string message = 2;
  repeated TrafficWeight applied_weights = 3;
}
```

#### Reload
Trigger a graceful configuration reload.

**Request:**
```protobuf
message ReloadRequest {
  bool force = 1;
}
```

**Response:**
```protobuf
message ReloadResponse {
  bool success = 1;
  string message = 2;
  int64 reload_timestamp = 3;
}
```

## REST API Endpoints

### Health Check
Liveness probe endpoint.

```
GET /healthz
```

**Response:** `200 OK` with body `OK` when Envoy is running, `503 Service Unavailable` otherwise.

**Environment variable:** `HEALTH_PORT` (default: `8080`)

### Readiness Check
Readiness probe endpoint.

```
GET /ready
```

**Response:** `200 OK` when Envoy is running and has been stable for >5 seconds, `503 Service Unavailable` otherwise.

### Metrics
Prometheus-compatible metrics endpoint.

```
GET /metrics
```

**Response:** Plain text Prometheus format metrics including:
- `alb_total_connections` - Total connections received
- `alb_active_connections` - Currently active connections
- `alb_total_requests` - Total requests processed
- `alb_requests_per_second` - Current request rate
- `alb_latency_ms` - Request latency (P50, P90, P95, P99)
- `alb_responses_total` - Response count by HTTP status code

**Environment variable:** `METRICS_PORT` (default: `9090`)

## Client Libraries

### Go gRPC Client
```go
conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
if err != nil {
  log.Fatal(err)
}
defer conn.Close()

client := pb.NewModuleServiceClient(conn)
status, err := client.GetStatus(context.Background(), &pb.StatusRequest{})
```

### grpcurl
```bash
# Ensure protobuf files are in current directory
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus
```

## Error Handling

All gRPC methods return standard gRPC error codes:
- `0` - OK
- `1` - CANCELLED
- `2` - UNKNOWN
- `3` - INVALID_ARGUMENT
- `4` - DEADLINE_EXCEEDED
- `5` - NOT_FOUND
- `13` - INTERNAL
- `14` - UNAVAILABLE

## TLS/mTLS

Currently, the ALB operates in plaintext mode. TLS support can be added via environment configuration of the gRPC server.
