# MarchProxy DBLB (Database Load Balancer)

Database Load Balancer module for MarchProxy with support for multiple database protocols, connection pooling, rate limiting, and SQL injection detection.

## Features

- **Multi-Protocol Support**: MySQL, PostgreSQL, MongoDB, Redis, MSSQL
- **Connection Pooling**: Efficient connection reuse and management
- **Rate Limiting**: Per-route connection and query rate limiting
- **SQL Injection Detection**: Pattern-based security checking
- **gRPC ModuleService**: Full ModuleService interface implementation
- **Health & Metrics**: HTTP endpoints for monitoring and observability
- **Production Ready**: Containerized deployment with K8s support

## Supported Database Protocols

| Protocol   | Default Port | Features                          |
|------------|--------------|-----------------------------------|
| MySQL      | 3306         | TCP proxy, connection pooling     |
| PostgreSQL | 5432         | TCP proxy, connection pooling     |
| MongoDB    | 27017        | TCP proxy, connection pooling     |
| Redis      | 6379         | TCP proxy, connection pooling     |
| MSSQL      | 1433         | TCP proxy, connection pooling     |

## Quick Start

### Build

```bash
go build -o proxy-dblb ./cmd/main.go
```

### Run

```bash
./proxy-dblb --config config.yaml
```

### Docker

```bash
docker build -t marchproxy/dblb:latest .
docker run -p 3306:3306 -p 5432:5432 -p 7002:7002 marchproxy/dblb:latest
```

## Configuration

Example configuration (`config.yaml`):

```yaml
grpc_addr: "0.0.0.0"
grpc_port: 50052
metrics_addr: ":7002"

manager_url: "http://api-server:8000"
cluster_api_key: "${CLUSTER_API_KEY}"

# Connection pooling
max_connections_per_route: 100
connection_idle_timeout: 5m
connection_max_lifetime: 30m

# Rate limiting
enable_rate_limiting: true
default_connection_rate: 100.0  # connections/sec
default_query_rate: 1000.0      # queries/sec

# Security
enable_sql_injection_detection: true
block_suspicious_queries: true

# Routes
routes:
  - name: "mysql-primary"
    protocol: "mysql"
    listen_port: 3306
    backend_host: "mysql-server"
    backend_port: 3306
    max_connections: 100
    connection_rate: 50.0
    query_rate: 500.0
    enable_auth: false

  - name: "postgres-main"
    protocol: "postgresql"
    listen_port: 5432
    backend_host: "postgres-server"
    backend_port: 5432
    max_connections: 200
    connection_rate: 100.0
    query_rate: 1000.0
```

## Environment Variables

- `CLUSTER_API_KEY`: API key for cluster authentication (required)
- `MARCHPROXY_DBLB_*`: Override any config value (e.g., `MARCHPROXY_DBLB_GRPC_PORT`)

## API Endpoints

### Health Check
```bash
GET http://localhost:7002/healthz
```

### Metrics (Prometheus)
```bash
GET http://localhost:7002/metrics
```

### Status
```bash
GET http://localhost:7002/status
```

Returns JSON with handler and pool statistics.

## gRPC ModuleService

DBLB implements the full ModuleService interface for NLB integration:

- `GetStatus` - Module health and status
- `Reload` - Graceful configuration reload
- `Shutdown` - Graceful shutdown
- `GetMetrics` - Real-time metrics
- `HealthCheck` - Deep health verification
- `GetStats` - Detailed statistics

## Architecture

```
Client -> DBLB (TCP Proxy) -> Backend Database
           |
           +-> Connection Pool
           +-> Rate Limiter
           +-> SQL Injection Checker
           +-> Metrics Collection
```

## Development

### Prerequisites

- Go 1.24.7 or later
- Docker (for containerized builds)

### Testing

```bash
go test ./...
```

### Linting

```bash
golangci-lint run
```

## Deployment

### Kubernetes

See `k8s/unified/dblb/` for deployment manifests:

- `deployment.yaml` - DBLB deployment with 2 replicas
- `service.yaml` - Service exposure

```bash
kubectl apply -f k8s/unified/dblb/
```

## Monitoring

DBLB exposes Prometheus metrics on `:7002/metrics`:

- `dblb_connections_active` - Active database connections
- `dblb_connections_total` - Total connections (lifetime)
- `dblb_queries_blocked` - Queries blocked by security
- `dblb_pool_size` - Connection pool sizes

## Security

### SQL Injection Detection

DBLB includes pattern-based SQL injection detection:

- Common SQL injection patterns
- Comment injection detection
- Excessive SQL keyword heuristics
- Custom pattern support

Configure via:
```yaml
enable_sql_injection_detection: true
block_suspicious_queries: true
```

## License

Limited AGPL3 with Contributor Employer Exception

Copyright (c) 2024 PenguinTech.io

## Support

- Documentation: https://docs.marchproxy.io
- Issues: https://github.com/penguintech/marchproxy/issues
- Website: https://www.penguintech.io
