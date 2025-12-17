# MarchProxy NLB API Documentation

## Overview

The NLB provides a comprehensive REST API for monitoring, health checks, and status queries, plus a gRPC API for module registration and management.

## REST API Endpoints

### Health Check Endpoint

**Endpoint**: `GET /healthz`
**Port**: 8082 (metrics port)
**Description**: Simple liveness probe for container health checks

**Response**:
```
200 OK
OK
```

**Example**:
```bash
curl http://localhost:8082/healthz
```

---

### Metrics Endpoint

**Endpoint**: `GET /metrics`
**Port**: 8082 (metrics port)
**Description**: Prometheus-compatible metrics endpoint for monitoring

**Response**: Text format with Prometheus metrics

**Key Metrics**:
- `nlb_routed_connections_total` - Counter: Connections routed by protocol and module
- `nlb_routing_errors_total` - Counter: Routing errors by protocol and error type
- `nlb_active_connections` - Gauge: Active connections per module
- `nlb_ratelimit_allowed_total` - Counter: Requests allowed by rate limiter
- `nlb_ratelimit_denied_total` - Counter: Requests denied by rate limiter
- `nlb_ratelimit_tokens_available` - Gauge: Available tokens per rate limit bucket
- `nlb_scale_operations_total` - Counter: Scaling operations by direction (up/down)
- `nlb_current_replicas` - Gauge: Current replica count per protocol
- `nlb_bluegreen_traffic_split` - Gauge: Traffic split percentage between blue/green

**Example**:
```bash
curl http://localhost:8082/metrics | grep nlb_routed
```

---

### Status Endpoint

**Endpoint**: `GET /status`
**Port**: 8082 (metrics port)
**Description**: Comprehensive status information about all NLB subsystems

**Response Format**: JSON

**Response Structure**:
```json
{
  "status": {
    "version": "1.0.0",
    "uptime": 12345.67,
    "rate_limiting": true,
    "autoscaling": true,
    "bluegreen": true,
    "connection_pooling": true,
    "router_stats": {
      "protocols": {
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
      },
      "errors": [
        {
          "protocol": "http",
          "error_type": "no_module",
          "count": 5
        }
      ]
    },
    "ratelimit_stats": {
      "buckets": {
        "http_global": {
          "tokens_available": 45000.0,
          "capacity": 50000.0,
          "refill_rate": 10000.0,
          "allowed_total": 1000000,
          "denied_total": 250
        }
      }
    },
    "autoscaler_stats": {
      "enabled": true,
      "last_evaluation": "2025-12-16T10:30:45Z",
      "recent_scaling_history": [
        {
          "timestamp": "2025-12-16T10:25:00Z",
          "protocol": "http",
          "direction": "up",
          "previous_replicas": 2,
          "new_replicas": 3,
          "reason": "high_cpu"
        }
      ]
    },
    "bluegreen_stats": {
      "active_deployments": [
        {
          "protocol": "http",
          "version": "v1.0.0",
          "blue_version": "v0.9.0",
          "green_version": "v1.0.0",
          "traffic_split_percent": 50,
          "status": "in_progress"
        }
      ]
    },
    "client_pool_stats": {
      "total_connections": 25,
      "active_connections": 20,
      "failed_connections": 2,
      "reconnect_attempts": 45
    }
  }
}
```

**Example**:
```bash
curl http://localhost:8082/status | jq '.status.router_stats'
```

---

## gRPC API

### Service Definition

The NLB gRPC server provides the following service on port 50051 (configurable):

```protobuf
service NLBService {
  rpc RegisterModule(RegisterModuleRequest) returns (RegisterModuleResponse);
  rpc UnregisterModule(UnregisterModuleRequest) returns (UnregisterModuleResponse);
  rpc UpdateHealth(HealthUpdateRequest) returns (HealthUpdateResponse);
  rpc GetStats(StatsRequest) returns (StatsResponse);
}
```

### RegisterModule RPC

**Description**: Register a module container with the NLB

**Request**:
```protobuf
message RegisterModuleRequest {
  string module_id = 1;           // Unique module identifier
  string protocol = 2;             // Protocol type (http, mysql, etc.)
  string address = 3;              // Module address (ip:port)
  int32 port = 4;                  // Module port
  string version = 5;              // Module version
  map<string, string> labels = 6;  // Custom labels/metadata
}
```

**Response**:
```protobuf
message RegisterModuleResponse {
  bool success = 1;
  string message = 2;
  string module_id = 3;
}
```

**Example Usage**:
```go
conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
client := nlb.NewNLBServiceClient(conn)

resp, _ := client.RegisterModule(context.Background(), &nlb.RegisterModuleRequest{
  ModuleId: "http-1",
  Protocol: "http",
  Address:  "127.0.0.1",
  Port:     50052,
  Version:  "1.0.0",
})
```

---

### UnregisterModule RPC

**Description**: Unregister a module container from the NLB

**Request**:
```protobuf
message UnregisterModuleRequest {
  string module_id = 1;  // Module identifier to remove
}
```

**Response**:
```protobuf
message UnregisterModuleResponse {
  bool success = 1;
  string message = 2;
}
```

---

### UpdateHealth RPC

**Description**: Update module health status

**Request**:
```protobuf
message HealthUpdateRequest {
  string module_id = 1;
  enum HealthStatus {
    HEALTHY = 0;
    DEGRADED = 1;
    UNHEALTHY = 2;
  }
  HealthStatus status = 2;
  string reason = 3;  // Optional health status reason
}
```

**Response**:
```protobuf
message HealthUpdateResponse {
  bool success = 1;
  string message = 2;
}
```

---

### GetStats RPC

**Description**: Retrieve statistics for a specific module

**Request**:
```protobuf
message StatsRequest {
  string module_id = 1;
}
```

**Response**:
```protobuf
message StatsResponse {
  string module_id = 1;
  int64 active_connections = 2;
  int64 total_connections_processed = 3;
  float cpu_usage_percent = 4;
  float memory_usage_bytes = 5;
  int64 bytes_in = 6;
  int64 bytes_out = 7;
  string health_status = 8;
  map<string, string> custom_metrics = 9;
}
```

---

## API Response Codes

### HTTP Status Codes

- **200 OK**: Request successful
- **400 Bad Request**: Invalid request parameters
- **401 Unauthorized**: Missing or invalid authentication
- **404 Not Found**: Resource not found
- **500 Internal Server Error**: Server error
- **503 Service Unavailable**: NLB not ready

### gRPC Status Codes

- **OK (0)**: Success
- **INVALID_ARGUMENT (3)**: Invalid request parameters
- **NOT_FOUND (5)**: Module not found
- **INTERNAL (13)**: Internal server error
- **UNAVAILABLE (14)**: Service unavailable

---

## Error Handling

### REST API Errors

Standard error response format:
```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "additional context"
  }
}
```

### gRPC Errors

gRPC errors include status code and message. Client libraries automatically decode these.

---

## Rate Limiting Headers

Rate limit information may be included in response headers:

```
X-RateLimit-Limit: 10000
X-RateLimit-Remaining: 9500
X-RateLimit-Reset: 1703064645
```

---

## Authentication

The NLB uses the `CLUSTER_API_KEY` environment variable for authentication. This key must be present for:
- Module registration via gRPC
- Administrative status queries

Requests without a valid key will be rejected with a 401 Unauthorized response.

---

## Request/Response Examples

### Check NLB Status with jq

```bash
# Get router statistics only
curl -s http://localhost:8082/status | jq '.status.router_stats'

# Get rate limit status
curl -s http://localhost:8082/status | jq '.status.ratelimit_stats'

# Get autoscaling history
curl -s http://localhost:8082/status | jq '.status.autoscaler_stats.recent_scaling_history'

# Get module health status for specific protocol
curl -s http://localhost:8082/status | jq '.status.router_stats.protocols.http'
```

### Monitor Metrics

```bash
# Get all NLB metrics
curl http://localhost:8082/metrics | grep nlb_

# Get routed connections count
curl http://localhost:8082/metrics | grep nlb_routed_connections_total

# Get rate limit denied requests
curl http://localhost:8082/metrics | grep nlb_ratelimit_denied_total
```

---

## Troubleshooting

### "No modules available" Error

- Verify modules are registered via gRPC
- Check module health status in `/status` endpoint
- Ensure modules have `protocol` matching incoming traffic

### Rate Limit Rejections

- Monitor `nlb_ratelimit_tokens_available` metric
- Adjust `default_rate_limit` and `default_burst_size` in configuration
- Check per-bucket token availability in `/status`

### Connection Failures

- Verify module addresses in router stats
- Check gRPC client pool connection count
- Review routing error counts by error type

---

## API Versioning

Current API Version: **1.0.0**

Future API versions will be provided at `/api/v2/...` endpoints while maintaining backward compatibility with v1.
