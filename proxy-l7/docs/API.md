# MarchProxy L7 Proxy API Documentation

## Overview

The MarchProxy L7 Proxy (Envoy-based) provides both admin APIs for operational control and xDS endpoints for dynamic configuration. The proxy communicates with the control plane via gRPC xDS protocol.

## Envoy Admin API

The Admin API provides operational control, debugging, and metrics access to the Envoy proxy instance.

### Base URL
```
http://localhost:9901
```

### Health Check Endpoints

#### Server Ready Status
```bash
GET /ready
```
Returns 200 when server is ready to accept traffic.

**Response:**
```
HTTP/1.1 200 OK
```

#### Server Info
```bash
GET /server_info
```
Returns detailed information about the Envoy instance.

**Response:**
```json
{
  "version": "1.28.0",
  "state": "LIVE",
  "uptime_current_epoch": "3600s",
  "uptime_all_epochs": "3600s",
  "hot_restart_version": "11",
  "command_line_options": {
    "config_path": "/etc/envoy/envoy.yaml"
  },
  "node": {
    "id": "marchproxy-node",
    "cluster": "marchproxy-cluster"
  }
}
```

### Metrics Endpoints

#### Prometheus Format Metrics
```bash
GET /stats/prometheus
```
Returns all metrics in Prometheus format for scraping.

**Response:**
```
# HELP envoy_http_ingress_http_requests_total
# TYPE envoy_http_ingress_http_requests_total counter
envoy_http_ingress_http_requests_total 1000

# HELP envoy_http_ingress_http_request_duration_seconds
# TYPE envoy_http_ingress_http_request_duration_seconds histogram
envoy_http_ingress_http_request_duration_seconds_bucket{le="0.005"} 100
```

#### JSON Format Statistics
```bash
GET /stats
```
Returns all metrics in JSON format with detailed values.

### Configuration Endpoints

#### Configuration Dump
```bash
GET /config_dump
```
Returns the current Envoy configuration including all dynamic resources.

**Response:**
```json
{
  "configs": [
    {
      "@type": "type.googleapis.com/google.protobuf.Timestamp",
      "timestamp": "2024-12-16T10:30:00Z"
    },
    {
      "@type": "type.googleapis.com/envoy.admin.v3.ListenerConfigDump",
      "static_listeners": [...],
      "dynamic_listeners": [...]
    },
    {
      "@type": "type.googleapis.com/envoy.admin.v3.RouteConfigDump",
      "static_route_configs": [...],
      "dynamic_route_configs": [...]
    },
    {
      "@type": "type.googleapis.com/envoy.admin.v3.ClusterConfigDump",
      "static_clusters": [...],
      "dynamic_clusters": [...]
    }
  ]
}
```

#### Configuration Dump by Type
```bash
GET /config_dump?mask=listeners
GET /config_dump?mask=routes
GET /config_dump?mask=clusters
GET /config_dump?mask=endpoints
```

#### Reset Statistics
```bash
POST /stats/prometheus/reset
```
Resets all counters and stats.

### Cluster Endpoints

#### List All Clusters
```bash
GET /clusters
```
Returns list of all clusters (services).

**Response:**
```
clusters::test-backend::default_priority::max_connections::1024
clusters::test-backend::default_priority::max_pending_requests::1024
clusters::test-backend::default_priority::max_requests::1024
clusters::test-backend::default_priority::max_retries::3
clusters::test-backend::high_priority::max_connections::1024
...
```

#### Cluster Details
```bash
GET /clusters?format=json
```
Returns cluster information in JSON format.

### Listener Endpoints

#### List All Listeners
```bash
GET /listeners
```
Returns list of active listeners.

**Response:**
```
listener_manager.listener_create_success: 2
listener_manager.listener_create_failure: 0
listener_manager.listener_destroy_success: 0
listener_manager.listener_destroy_failure: 0
listener_manager.tcp_cx: 5000
listener_manager.tcp_cx_v6: 0
```

### Route Endpoints

#### List All Routes
```bash
GET /routes
```
Returns all active routes.

### Drain Control

#### Start Graceful Drain
```bash
POST /drain_listeners?inboundonly
```
Initiates graceful shutdown, stopping new connections while draining existing ones.

### xDS Debug

#### xDS Configuration Status
```bash
GET /config_dump | jq '.configs[] | select(.["@type"] | contains("Listener"))'
```
View listener configuration from xDS.

#### Check xDS Connectivity
```bash
curl -s http://localhost:9901/config_dump | jq '.configs[0]'
```
If timestamp is recent, xDS is connected and receiving updates.

## xDS Control Plane Interface

The proxy communicates with the control plane via gRPC xDS protocol. This is not directly called but is used internally by Envoy.

### Bootstrap Configuration

The proxy connects to the xDS control plane via the bootstrap configuration:

```yaml
# /etc/envoy/envoy.yaml
dynamic_resources:
  lds_config:
    resource_api_version: V3
    ads: {}
  cds_config:
    resource_api_version: V3
    ads: {}
  ads_config:
    api_type: GRPC
    transport_api_version: V3
    grpc_services:
      - envoy_grpc:
          cluster_name: xds_cluster

static_resources:
  clusters:
    - name: xds_cluster
      type: STRICT_DNS
      load_assignment:
        cluster_name: xds_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: api-server  # xDS control plane
                      port_value: 18000    # xDS gRPC port
```

### xDS Endpoints (Served by api-server)

Configuration updates are sent to the xDS server at `api-server:18000`.

#### Configuration Update API
```bash
POST http://api-server:19000/v1/config
Content-Type: application/json

{
  "version": "1",
  "services": [
    {
      "name": "backend-service",
      "hosts": ["backend1.local", "backend2.local"],
      "port": 8080,
      "protocol": "http"
    }
  ],
  "routes": [
    {
      "name": "api-route",
      "prefix": "/api",
      "cluster_name": "backend-service",
      "hosts": ["api.example.com"],
      "timeout": 30
    }
  ]
}
```

**Response:**
```json
{
  "status": "success",
  "version": "1",
  "timestamp": "2024-12-16T10:30:00Z"
}
```

#### Get Current Configuration Version
```bash
GET http://api-server:19000/v1/version
```

**Response:**
```json
{
  "version": "1",
  "timestamp": "2024-12-16T10:30:00Z",
  "services_count": 5,
  "routes_count": 12
}
```

## WASM Filter Metrics

Custom WASM filters expose metrics via the Envoy stats system.

### Auth Filter Metrics
```
wasm.auth_filter.total_requests: Counter
wasm.auth_filter.authorized_requests: Counter
wasm.auth_filter.denied_requests: Counter
wasm.auth_filter.jwt_validation_errors: Counter
wasm.auth_filter.token_validation_errors: Counter
```

### License Filter Metrics
```
wasm.license_filter.total_requests: Counter
wasm.license_filter.licensed_requests: Counter
wasm.license_filter.blocked_requests: Counter
wasm.license_filter.feature_checks: Counter
```

### Metrics Filter Metrics
```
wasm.metrics_filter.request_bytes: Histogram
wasm.metrics_filter.response_bytes: Histogram
wasm.metrics_filter.latency_ms: Histogram
```

## Query Metrics Examples

### Total Requests
```bash
curl -s http://localhost:9901/stats/prometheus | grep 'envoy_http_ingress_http_requests_total'
```

### Request Duration (p99)
```bash
curl -s http://localhost:9901/stats/prometheus | grep 'envoy_http_ingress_http_request_duration_seconds_bucket{le="10"}'
```

### Upstream Cluster Health
```bash
curl -s http://localhost:9901/stats/prometheus | grep 'envoy_cluster_membership_healthy'
```

### WASM Filter Rejections
```bash
curl -s http://localhost:9901/stats/prometheus | grep 'wasm_auth_filter_denied_requests'
```

## Debugging

### View Current Configuration
```bash
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("HttpConnectionManager"))'
```

### Check Route Matching
```bash
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Route"))'
```

### View Cluster Endpoints
```bash
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("ClusterLoadAssignment"))'
```

### Monitor xDS Updates
```bash
docker logs proxy-l7 | grep -i xds
```

## Rate Limiting

### Admin API Rate Limits
- No built-in rate limiting on admin API (runs on separate port)
- Restrict network access to port 9901 in production
- Use network policies or firewall rules for access control

### Monitoring Rate Limits
Watch the following metrics to detect issues:
```bash
curl -s http://localhost:9901/stats/prometheus | grep -E 'envoy_http.*rate_limit'
```

## Security Considerations

1. **Admin Port (9901)**: Runs on separate port, should not be exposed to untrusted networks
2. **No Authentication**: Admin API has no built-in authentication
3. **Sensitive Information**: Config dump includes route patterns and cluster details
4. **Drain Support**: Use `/drain_listeners` for graceful shutdown

## Related Documentation

- [CONFIGURATION.md](./CONFIGURATION.md) - Envoy configuration details
- [USAGE.md](./USAGE.md) - Operational guidance
- [README.md](../README.md) - Project overview
