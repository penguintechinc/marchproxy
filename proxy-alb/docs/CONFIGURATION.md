# MarchProxy ALB - Configuration Guide

## Environment Variables

The ALB is configured entirely through environment variables. All have sensible defaults.

### Module Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MODULE_ID` | `alb-1` | Unique identifier for this ALB instance |
| `VERSION` | `v1.0.0` | ALB version (embedded in binary at build time) |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |

### Envoy Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVOY_BINARY` | `/usr/local/bin/envoy` | Path to Envoy executable |
| `ENVOY_CONFIG_PATH` | `/etc/envoy/envoy.yaml` | Path to Envoy configuration file |
| `ENVOY_ADMIN_PORT` | `9901` | Envoy admin interface port |
| `ENVOY_LISTEN_PORT` | `10000` | Port Envoy listens for traffic (HTTP/HTTPS) |
| `ENVOY_LOG_LEVEL` | `info` | Envoy log level |

### xDS Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `XDS_SERVER` | `api-server:18000` | Address of xDS control plane server |
| `XDS_NODE_ID` | `alb-node` | Node ID for xDS discovery service |
| `XDS_CLUSTER` | `marchproxy-cluster` | Cluster name in xDS configuration |
| `XDS_CONNECT_TIMEOUT` | `5s` | Timeout for xDS server connections (Go duration format) |

### gRPC Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `GRPC_PORT` | `50051` | Port for gRPC ModuleService API |
| `GRPC_MAX_CONN_AGE` | `30m` | Maximum connection age (Go duration format) |

### Monitoring Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `METRICS_PORT` | `9090` | Port for Prometheus metrics endpoint |
| `HEALTH_PORT` | `8080` | Port for health check endpoints (/healthz, /ready) |

### Lifecycle Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown timeout (Go duration format) |
| `RELOAD_GRACE_PERIOD` | `5s` | Grace period for configuration reload |

### Licensing Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LICENSE_KEY` | empty | Enterprise license key (format: `PENG-XXXX-XXXX-XXXX-XXXX-ABCD`) |
| `CLUSTER_API_KEY` | empty | API key for cluster authentication |

## Docker Configuration

### Environment File

Create a `.env` file for Docker Compose or `docker run`:

```bash
# Module
MODULE_ID=alb-1
VERSION=v1.0.0
LOG_LEVEL=info

# Envoy
ENVOY_BINARY=/usr/local/bin/envoy
ENVOY_CONFIG_PATH=/etc/envoy/envoy.yaml
ENVOY_ADMIN_PORT=9901
ENVOY_LISTEN_PORT=10000
ENVOY_LOG_LEVEL=info

# xDS Server
XDS_SERVER=api-server:18000
XDS_NODE_ID=alb-node-1
XDS_CLUSTER=marchproxy-cluster

# Services
GRPC_PORT=50051
METRICS_PORT=9090
HEALTH_PORT=8080

# Licensing
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
CLUSTER_API_KEY=your-cluster-api-key
```

### Docker Run

Start ALB with environment variables:

```bash
docker run -d \
  --name marchproxy-alb \
  --network marchproxy \
  -p 10000:10000 \
  -p 50051:50051 \
  -p 8080:8080 \
  -p 9090:9090 \
  -p 9901:9901 \
  -e MODULE_ID=alb-1 \
  -e XDS_SERVER=api-server:18000 \
  -e LOG_LEVEL=info \
  --env-file .env \
  marchproxy-alb:latest
```

### Docker Compose

Example `docker-compose.yml`:

```yaml
version: '3.8'

services:
  alb:
    image: marchproxy-alb:latest
    container_name: marchproxy-alb
    networks:
      - marchproxy
    ports:
      - "10000:10000"  # Envoy traffic
      - "50051:50051"  # gRPC API
      - "8080:8080"    # Health checks
      - "9090:9090"    # Metrics
      - "9901:9901"    # Envoy admin
    environment:
      MODULE_ID: alb-1
      VERSION: v1.0.0
      LOG_LEVEL: info
      XDS_SERVER: api-server:18000
      XDS_NODE_ID: alb-node-1
      GRPC_PORT: 50051
      METRICS_PORT: 9090
      HEALTH_PORT: 8080
      LICENSE_KEY: ${LICENSE_KEY}
      CLUSTER_API_KEY: ${CLUSTER_API_KEY}
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 30s
    depends_on:
      - api-server
    restart: unless-stopped

  api-server:
    image: api-server:latest
    container_name: api-server
    networks:
      - marchproxy
    ports:
      - "18000:18000"
    restart: unless-stopped

networks:
  marchproxy:
    driver: bridge
```

## Configuration Validation

The ALB validates configuration on startup. Required fields:

- `MODULE_ID` - Cannot be empty
- `ENVOY_BINARY` - Cannot be empty
- `ENVOY_CONFIG_PATH` - Cannot be empty
- `XDS_SERVER` - Cannot be empty
- `GRPC_PORT` - Must be between 1-65535
- `ENVOY_ADMIN_PORT` - Must be between 1-65535

If validation fails, the ALB exits with error logs.

## Go Duration Format

Duration strings use Go's duration format:

| Unit | Symbol | Example |
|------|--------|---------|
| Nanosecond | `ns` | `100ns` |
| Microsecond | `us` | `100us` |
| Millisecond | `ms` | `100ms` |
| Second | `s` | `30s` |
| Minute | `m` | `5m` |
| Hour | `h` | `1h` |

Examples: `5s`, `30m`, `1h30m`, `100ms`

## Configuration Examples

### Development Environment

```bash
export LOG_LEVEL=debug
export ENVOY_LOG_LEVEL=debug
export XDS_SERVER=localhost:18000
export GRPC_PORT=50051
export METRICS_PORT=9090
export HEALTH_PORT=8080
```

### Production Environment

```bash
export LOG_LEVEL=info
export ENVOY_LOG_LEVEL=warn
export XDS_SERVER=api-server.production:18000
export GRPC_PORT=50051
export METRICS_PORT=9090
export HEALTH_PORT=8080
export LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
export CLUSTER_API_KEY=your-production-key
export SHUTDOWN_TIMEOUT=60s
```

### High Performance Configuration

```bash
export LOG_LEVEL=warn
export ENVOY_LOG_LEVEL=warn
export XDS_CONNECT_TIMEOUT=10s
export GRPC_MAX_CONN_AGE=5m
export SHUTDOWN_TIMEOUT=60s
```

## Troubleshooting Configuration

### ALB fails to start with "config validation failed"

Check the error message for the specific field. Ensure all required fields are set:

```bash
echo $MODULE_ID
echo $ENVOY_BINARY
echo $ENVOY_CONFIG_PATH
echo $XDS_SERVER
```

### Envoy fails to start

Verify Envoy binary path and configuration file exist:

```bash
ls -la /usr/local/bin/envoy
ls -la /etc/envoy/envoy.yaml
```

### Cannot connect to xDS server

Check network connectivity and server address:

```bash
echo $XDS_SERVER
# Test connection
nc -zv $(echo $XDS_SERVER | cut -d: -f1) $(echo $XDS_SERVER | cut -d: -f2)
```

### Health checks failing

Check ALB logs and verify Envoy is running:

```bash
curl -v http://localhost:8080/healthz
curl -v http://localhost:8080/ready
```
