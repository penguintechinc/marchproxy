# MarchProxy L7 Proxy Configuration Guide

## Overview

The MarchProxy L7 proxy uses Envoy as the core proxy engine, with dynamic configuration via xDS control plane and static bootstrap configuration. This guide covers Envoy configuration, environment variables, and Docker deployment.

## Envoy Configuration

### Bootstrap Configuration (`/etc/envoy/envoy.yaml`)

The bootstrap configuration is static and defines how Envoy connects to the xDS control plane.

```yaml
# Admin interface configuration
admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      protocol: TCP
      address: 0.0.0.0
      port_value: 9901

# Node identification
node:
  id: marchproxy-node
  cluster: marchproxy-cluster
  locality:
    zone: us-central1-a

# Dynamic resources from xDS
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
    set_node_on_first_message_only: true

# Static resources
static_resources:
  clusters:
    - name: xds_cluster
      type: STRICT_DNS
      typed_extension_protocol_options:
        envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
          "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
          explicit_http_config:
            http2_protocol_options: {}
      load_assignment:
        cluster_name: xds_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: api-server
                      port_value: 18000
      connect_timeout: 5s
      http2_protocol_options:
        connection_keepalive:
          interval: 30s
          timeout: 5s

# Runtime overrides
layered_runtime:
  layers:
    - name: static_layer_0
      static_layer:
        overload:
          global_downstream_max_connections: 50000
```

### Dynamic Configuration via xDS

Dynamic resources are provided by the xDS control plane at `api-server:18000`. The control plane pushes configuration updates for:

- **Listeners (LDS)**: Define listening ports and protocols
- **Routes (RDS)**: Define routing rules and virtual hosts
- **Clusters (CDS)**: Define upstream services
- **Endpoints (EDS)**: Define individual endpoints within clusters

#### Example Configuration Update
```bash
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{
    "version": "1",
    "services": [
      {
        "name": "api-backend",
        "hosts": ["api1.internal", "api2.internal"],
        "port": 8080,
        "protocol": "http"
      },
      {
        "name": "web-backend",
        "hosts": ["web1.internal", "web2.internal"],
        "port": 80,
        "protocol": "http"
      }
    ],
    "routes": [
      {
        "name": "api-route",
        "prefix": "/api",
        "cluster_name": "api-backend",
        "hosts": ["api.example.com"],
        "timeout": 30
      },
      {
        "name": "web-route",
        "prefix": "/",
        "cluster_name": "web-backend",
        "hosts": ["example.com", "www.example.com"],
        "timeout": 30
      }
    ]
  }'
```

### WASM Filter Configuration

WASM filters are loaded into Envoy and configured via the xDS control plane configuration payload.

#### Auth Filter Configuration
```json
{
  "auth_filter": {
    "jwt_secret": "your-secret-key-here",
    "jwt_algorithm": "HS256",
    "require_auth": true,
    "base64_tokens": ["token1", "token2"],
    "exempt_paths": ["/healthz", "/metrics", "/public/*"]
  }
}
```

**Available JWT Algorithms**: HS256, HS384, HS512

**Exempt Paths**: Paths that don't require authentication (supports wildcards)

#### License Filter Configuration
```json
{
  "license_filter": {
    "license_key": "PENG-XXXX-XXXX-XXXX-XXXX-ABCD",
    "is_enterprise": true,
    "features": {
      "rate_limiting": true,
      "multi_cloud": true,
      "distributed_tracing": true,
      "advanced_routing": true
    },
    "max_proxies": 100,
    "license_server": "https://license.penguintech.io"
  }
}
```

#### Metrics Filter Configuration
```json
{
  "metrics_filter": {
    "histogram_buckets": [1, 5, 10, 50, 100, 500, 1000],
    "enable_request_size": true,
    "enable_response_size": true,
    "enable_latency": true
  }
}
```

## Environment Variables

Configuration via environment variables for Docker deployments.

### Required Variables

#### XDS Server
```bash
XDS_SERVER=api-server:18000
```
Address and port of the xDS control plane gRPC server.

#### Cluster API Key
```bash
CLUSTER_API_KEY=your-api-key-here
```
Authentication token for xDS server communication. Used to identify the proxy cluster.

### Optional Variables

#### XDP Configuration
```bash
# Network interface for XDP program loading
XDP_INTERFACE=eth0

# XDP mode: native (driver-level), skb (generic), hw (hardware)
XDP_MODE=native
```

#### Logging
```bash
# Log level: debug, info, warn, error
LOGLEVEL=info

# Access log format
ACCESS_LOG_FORMAT=[%START_TIME%] "%REQ(:METHOD)% %REQ(X-ENVOY-ORIGINAL-PATH?:PATH)% %PROTOCOL%" %RESPONSE_CODE% %RESPONSE_FLAGS% %BYTES_RECEIVED% %BYTES_SENT% "%DURATION%" "%RESP(X-ENVOY-UPSTREAM-SERVICE-TIME)%" "%REQ(USER-AGENT)%" "%REQ(X-REQUEST-ID)%" "%REQ(:AUTHORITY)%" "%UPSTREAM_HOST%"
```

#### Performance Tuning
```bash
# Maximum concurrent connections
MAX_CONNECTIONS=50000

# Connection timeout (milliseconds)
CONNECT_TIMEOUT=5000

# Request timeout (seconds)
REQUEST_TIMEOUT=30

# HTTP/2 settings
HTTP2_SETTINGS_MAX_CONCURRENT_STREAMS=100
```

#### Feature Flags
```bash
# Enable/disable XDP (default: enabled if NET_ADMIN capability present)
ENABLE_XDP=true

# Enable WASM filters (default: true)
ENABLE_WASM=true

# Enable admin interface (default: true, disable in production)
ENABLE_ADMIN=true
```

#### Rate Limiting (XDP)
```bash
# Rate limit window in nanoseconds (1 second = 1000000000)
RATE_LIMIT_WINDOW_NS=1000000000

# Maximum packets per source IP per window
RATE_LIMIT_MAX_PACKETS=10000

# Enable rate limiting (default: true)
ENABLE_RATE_LIMITING=true
```

## Docker Configuration

### Environment Variable Example
```bash
docker run -d \
  --name proxy-l7 \
  -p 10000:10000 \
  -p 9901:9901 \
  --cap-add=NET_ADMIN \
  -e XDS_SERVER=api-server:18000 \
  -e CLUSTER_API_KEY=test-key \
  -e XDP_INTERFACE=eth0 \
  -e XDP_MODE=native \
  -e LOGLEVEL=info \
  -e MAX_CONNECTIONS=50000 \
  marchproxy/proxy-l7:latest
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  api-server:
    image: marchproxy/api-server:latest
    container_name: api-server
    environment:
      - DATABASE_URL=postgresql://marchproxy:password@postgres:5432/marchproxy
      - XDS_GRPC_PORT=18000
      - XDS_HTTP_PORT=19000
      - SECRET_KEY=your-secret-key
    ports:
      - "8000:8000"    # API server
      - "18000:18000"  # xDS gRPC
      - "19000:19000"  # xDS HTTP
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - marchproxy-network

  proxy-l7:
    image: marchproxy/proxy-l7:latest
    container_name: proxy-l7
    environment:
      - XDS_SERVER=api-server:18000
      - CLUSTER_API_KEY=cluster-key-123
      - XDP_INTERFACE=eth0
      - XDP_MODE=native
      - LOGLEVEL=info
      - MAX_CONNECTIONS=50000
    ports:
      - "10000:10000"  # HTTP/HTTPS
      - "9901:9901"    # Admin
    cap_add:
      - NET_ADMIN      # Required for XDP
    networks:
      - marchproxy-network
    depends_on:
      api-server:
        condition: service_started
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9901/ready"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s

  postgres:
    image: postgres:15-bookworm
    container_name: postgres
    environment:
      - POSTGRES_USER=marchproxy
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=marchproxy
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - marchproxy-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U marchproxy"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:

networks:
  marchproxy-network:
    driver: bridge
```

### Required Docker Capabilities

```yaml
cap_add:
  - NET_ADMIN  # Required for XDP program loading
```

## Configuration Validation

### Check Envoy Configuration
```bash
# Dump current configuration
curl http://localhost:9901/config_dump | jq .

# Check specific resource types
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Listener"))'
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Route"))'
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Cluster"))'
```

### Verify xDS Connectivity
```bash
# Check if xDS updates are being received
curl -s http://localhost:9901/stats | grep xds

# Expected output shows xds.server_state as CONNECTED
```

### Test Configuration Changes
```bash
# Make configuration change
curl -X POST http://api-server:19000/v1/config \
  -H "Content-Type: application/json" \
  -d '{...}'

# Verify change was applied (may take a few seconds)
sleep 2
curl http://localhost:9901/config_dump | jq '.configs[] | select(.["@type"] | contains("Route"))'
```

## Performance Configuration

### Connection Settings
```yaml
# In bootstrap.yaml or via xDS
connection_settings:
  keepalive:
    timeout: 5s
    interval: 30s
  connect_timeout: 5s
```

### HTTP/2 Settings
```yaml
http2_protocol_options:
  max_concurrent_streams: 100
  initial_connection_window_size: 65536
  initial_stream_window_size: 65536
```

### Upstream Cluster Settings
```json
{
  "connect_timeout": "5s",
  "per_connection_buffer_limit_bytes": 32768,
  "http2_protocol_options": {
    "max_concurrent_streams": 100
  }
}
```

## TLS/HTTPS Configuration

Certificates can be managed via:

1. **Infisical Integration**: Secret management
2. **HashiCorp Vault**: Dynamic secrets
3. **Direct Upload**: Manual certificate management

Configuration is done via xDS control plane with listener configuration including SDS (Secret Discovery Service) for dynamic secret retrieval.

## Load Balancing Policy

Default load balancing: **Round Robin**

Configure via xDS cluster definition:
```json
{
  "name": "my-cluster",
  "lb_policy": "ROUND_ROBIN",
  "hosts": ["host1", "host2", "host3"]
}
```

Other options:
- `ROUND_ROBIN`: Default
- `LEAST_REQUEST`: Fewest active connections
- `RANDOM`: Random selection
- `MAGLEV`: Consistent hashing

## Advanced Configuration

### Circuit Breaking
```json
{
  "circuit_breakers": {
    "thresholds": [
      {
        "priority": "DEFAULT",
        "max_connections": 1024,
        "max_pending_requests": 1024,
        "max_requests": 1024,
        "max_retries": 3
      }
    ]
  }
}
```

### Retry Policy
```json
{
  "retry_policy": {
    "retry_on": "5xx",
    "num_retries": 3,
    "per_try_timeout": "5s"
  }
}
```

### Outlier Detection
```json
{
  "outlier_detection": {
    "consecutive_5xx": 5,
    "interval": "30s",
    "base_ejection_time": "30s"
  }
}
```

## Troubleshooting Configuration

### Configuration Not Applied
```bash
# Check xDS connectivity
curl -s http://localhost:9901/stats | grep -i xds

# Check configuration version
curl http://api-server:19000/v1/version

# Increase log level for debugging
-e LOGLEVEL=debug
```

### Invalid Configuration
```bash
# Check Envoy logs
docker logs proxy-l7

# Validate configuration
curl http://localhost:9901/config_dump | jq . | head -50
```

## Related Documentation

- [API.md](./API.md) - Admin API reference
- [USAGE.md](./USAGE.md) - Operational guide
- [TESTING.md](./TESTING.md) - Testing guide
