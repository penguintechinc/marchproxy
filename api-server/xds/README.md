# MarchProxy xDS Control Plane

## Overview

The MarchProxy xDS Control Plane is a Go-based implementation of the Envoy xDS (Discovery Service) protocol, providing dynamic configuration management for Envoy proxies used in the MarchProxy L7 infrastructure.

## Architecture

### Components

1. **xDS Server** (`main.go`, `server.go`)
   - gRPC server implementing Envoy's xDS protocol
   - Supports ADS (Aggregated Discovery Service)
   - Handles LDS, RDS, CDS, EDS, and SDS

2. **Snapshot Builder** (`snapshot.go`)
   - Translates MarchProxy configuration to Envoy resources
   - Generates consistent snapshots
   - Supports versioning for rollback capability

3. **HTTP API Bridge** (`api.go`)
   - RESTful API for Python FastAPI integration
   - Configuration update endpoints
   - Rollback support
   - Health checks and metrics

4. **Filter Configurations** (`filters.go`)
   - HTTP filter chains
   - WebSocket upgrade support
   - HTTP/2 protocol options
   - CORS and gRPC-Web filters

5. **TLS Management** (`tls_config.go`)
   - Downstream TLS contexts (listener)
   - Upstream TLS contexts (cluster)
   - Certificate validation

6. **Snapshot Cache** (`cache.go`)
   - Thread-safe cache for xDS snapshots
   - Version tracking
   - Statistics and debugging

7. **Callbacks** (`callbacks.go`)
   - xDS server lifecycle hooks
   - Request/response logging
   - Metrics collection

## Features

### xDS Resources Supported

- **LDS (Listener Discovery Service)**: HTTP/HTTPS listeners with WebSocket and HTTP/2 support
- **RDS (Route Discovery Service)**: Virtual hosts and route configurations
- **CDS (Cluster Discovery Service)**: Backend clusters with health checks
- **EDS (Endpoint Discovery Service)**: Backend endpoint assignments
- **SDS (Secret Discovery Service)**: TLS certificates and validation contexts

### Protocol Support

- HTTP/1.1
- HTTP/2 (including gRPC)
- WebSocket upgrade
- TLS/SSL with certificate management

### Advanced Features

- **Health Checks**: HTTP/gRPC health checking for upstream clusters
- **TLS Support**: Full TLS configuration for both downstream and upstream
- **WebSocket**: Native WebSocket upgrade support
- **HTTP/2**: HTTP/2 protocol with ALPN negotiation
- **Rollback**: Configuration version history and rollback capability
- **Metrics**: Prometheus-compatible metrics endpoint
- **Validation**: Comprehensive configuration validation

## Configuration Format

### JSON Configuration Structure

```json
{
  "version": "1",
  "services": [
    {
      "name": "service-name",
      "hosts": ["backend.example.com"],
      "port": 8080,
      "protocol": "http",
      "tls_enabled": false,
      "tls_cert_name": "cert-name",
      "tls_verify": true,
      "health_check_path": "/healthz",
      "timeout_seconds": 30,
      "http2_enabled": false,
      "websocket_upgrade": false
    }
  ],
  "routes": [
    {
      "name": "route-name",
      "prefix": "/api",
      "cluster_name": "service-name",
      "hosts": ["api.example.com"],
      "timeout": 30
    }
  ],
  "certificates": [
    {
      "name": "cert-name",
      "cert_chain": "-----BEGIN CERTIFICATE-----\n...",
      "private_key": "-----BEGIN PRIVATE KEY-----\n...",
      "ca_cert": "-----BEGIN CERTIFICATE-----\n...",
      "require_client": false
    }
  ]
}
```

## API Endpoints

### Configuration Management

#### POST /v1/config
Update Envoy configuration with new snapshot

**Request Body**: JSON configuration (see format above)

**Response**:
```json
{
  "status": "success",
  "version": "123",
  "message": "Configuration updated successfully"
}
```

#### GET /v1/version
Get current configuration version

**Response**:
```json
{
  "version": 123,
  "node_id": "marchproxy-control-plane"
}
```

#### GET /v1/snapshot/{version}
Get information about a specific snapshot version

**Response**:
```json
{
  "version": 5,
  "current_version": 10,
  "available": true
}
```

#### POST /v1/rollback/{version}
Rollback to a previous configuration version

**Response**:
```json
{
  "status": "success",
  "rolled_back_to": 5,
  "new_version": 11,
  "version_string": "5",
  "message": "Successfully rolled back to version 5"
}
```

### Health and Metrics

#### GET /healthz
Health check endpoint

**Response**:
```json
{
  "status": "healthy",
  "service": "marchproxy-xds-server"
}
```

#### GET /metrics
Prometheus metrics

**Response** (text/plain):
```
# HELP xds_requests_total Total number of xDS requests
# TYPE xds_requests_total counter
xds_requests_total 42
# HELP xds_fetches_total Total number of xDS fetches
# TYPE xds_fetches_total counter
xds_fetches_total 10
# HELP xds_cache_version Current cache version
# TYPE xds_cache_version gauge
xds_cache_version 5
```

## Building and Running

### Build

```bash
cd /home/penguin/code/MarchProxy/api-server/xds
go mod tidy
go build -o xds-server .
```

### Run

```bash
# Default settings (gRPC on 18000, HTTP API on 19000)
./xds-server

# Custom ports and debug mode
./xds-server -port 18000 -metrics 19000 -debug
```

### Command-line Options

- `-port <int>`: xDS gRPC server port (default: 18000)
- `-metrics <int>`: HTTP API and metrics port (default: 19000)
- `-debug`: Enable debug logging
- `-nodeID <string>`: Node ID for xDS (default: "marchproxy-control-plane")

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xds-server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/xds-server .
EXPOSE 18000 19000
CMD ["./xds-server"]
```

### Build Docker Image

```bash
cd /home/penguin/code/MarchProxy/api-server/xds
docker build -t marchproxy-xds:latest .
```

### Run Docker Container

```bash
docker run -d \
  --name marchproxy-xds \
  -p 18000:18000 \
  -p 19000:19000 \
  marchproxy-xds:latest
```

## Integration with Python FastAPI

### Using the xDS Service

```python
from app.services.xds_service import get_xds_service, trigger_xds_update
from sqlalchemy.orm import Session

# Get xDS service instance
xds_service = get_xds_service(xds_server_url="http://localhost:19000")

# Trigger configuration update
success = await trigger_xds_update(cluster_id=1, db=db_session)

# Check xDS server health
is_healthy = await xds_service.health_check()

# Get current version
version = await xds_service.get_current_version()

# Rollback to previous version
success = await xds_service.rollback_to_version(version=5)

# Get service statistics
stats = xds_service.get_stats()
```

### Automatic Updates on Database Changes

The xDS service automatically builds configurations from database models:
- Services with TLS settings
- Mappings for routing rules
- Certificates for TLS/SSL

## Troubleshooting

### Common Issues

#### 1. xDS Server Not Starting

```bash
# Check if ports are available
netstat -tuln | grep -E '18000|19000'

# Check logs
./xds-server -debug
```

#### 2. Envoy Not Connecting

```bash
# Verify xDS server is listening
curl http://localhost:19000/healthz

# Check Envoy logs for connection errors
docker logs envoy-proxy
```

#### 3. Configuration Not Updating

```bash
# Check current version
curl http://localhost:19000/v1/version

# Manually trigger update via Python
from app.services.xds_service import trigger_xds_update
await trigger_xds_update(cluster_id=1, db=session)
```

#### 4. TLS Errors

- Verify certificate format (PEM)
- Check certificate validity dates
- Ensure CA chain is complete
- Validate private key matches certificate

## Performance Considerations

### Resource Usage

- **Memory**: ~50MB base + ~1MB per active snapshot
- **CPU**: Minimal (~0.1% idle, ~5% during updates)
- **Network**: Low bandwidth, primarily push-based updates

### Scaling

- Single xDS server supports 1000+ Envoy proxies
- Stateless design allows horizontal scaling
- Use load balancer for high-availability deployments

### Best Practices

1. **Snapshot Versioning**: Use meaningful version strings (timestamps, git commits)
2. **Gradual Rollout**: Test configurations on staging before production
3. **Monitoring**: Monitor xDS metrics and Envoy connection status
4. **Rollback Strategy**: Keep last 10 versions for quick rollback
5. **Health Checks**: Configure health checks for all upstream clusters

## Security

### TLS Configuration

- Use strong cipher suites
- Enable client certificate validation when needed
- Rotate certificates before expiration
- Store private keys securely (use SDS with external secrets manager)

### API Security

- Deploy xDS server behind firewall or VPN
- Use network policies to restrict access
- Consider adding authentication to HTTP API
- Enable audit logging for configuration changes

## Development

### Adding New Features

1. Update configuration structs in `snapshot.go`
2. Implement resource builders in appropriate files
3. Add validation in snapshot generation
4. Update Python bridge in `xds_service.py`
5. Add tests and documentation

### Testing

```bash
# Unit tests
go test -v ./...

# Integration tests with Envoy
docker-compose up -d envoy
curl http://localhost:10000/  # Test through Envoy

# Load testing
hey -n 10000 -c 100 http://localhost:19000/v1/version
```

## References

- [Envoy xDS Protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol)
- [go-control-plane](https://github.com/envoyproxy/go-control-plane)
- [Envoy Proxy Documentation](https://www.envoyproxy.io/docs)
