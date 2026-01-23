# DBLB Configuration Guide

Complete configuration reference for MarchProxy Database Load Balancer (DBLB) including all environment variables, Docker configuration, and database backend setup.

## Configuration Files

DBLB is configured using YAML files. The main configuration file is specified via the `--config` flag:

```bash
./proxy-dblb --config /path/to/config.yaml
```

Default location: `/app/config.yaml` (in Docker)

## Configuration File Structure

```yaml
# gRPC server settings
grpc_addr: "0.0.0.0"
grpc_port: 50052

# Metrics and health endpoint
metrics_addr: ":7002"

# Manager connection
manager_url: "http://api-server:8000"
cluster_api_key: "${CLUSTER_API_KEY}"
registration_url: "http://api-server:8000/api/proxy/register"

# Connection pooling
max_connections_per_route: 100
connection_idle_timeout: 5m
connection_max_lifetime: 30m

# Rate limiting
enable_rate_limiting: true
default_connection_rate: 100.0  # connections/sec
default_query_rate: 1000.0      # queries/sec

# Security settings
enable_sql_injection_detection: true
block_suspicious_queries: true

# Observability
enable_tracing: false
jaeger_endpoint: "http://jaeger:14268/api/traces"
trace_sample_rate: 0.1
metrics_namespace: "marchproxy_dblb"

# Licensing (Enterprise)
license_key: "${LICENSE_KEY}"
license_server: "https://license.penguintech.io"
release_mode: false

# Routes (see below)
routes:
  - name: "mysql-primary"
    # ... route config ...
```

## Environment Variables

Environment variables override corresponding YAML settings. Use `${VAR_NAME}` syntax in YAML to reference environment variables.

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CLUSTER_API_KEY` | API key for cluster authentication | `abc123def456ghi789` |

### Optional Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DBLB_GRPC_ADDR` | gRPC server address | `0.0.0.0` |
| `DBLB_GRPC_PORT` | gRPC server port | `50052` |
| `DBLB_METRICS_ADDR` | Metrics HTTP address | `:7002` |
| `DBLB_CONFIG` | Config file path | `/app/config.yaml` |
| `DBLB_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `DBLB_LOG_FORMAT` | Log format (json, text) | `json` |
| `MANAGER_URL` | Manager API base URL | `http://api-server:8000` |
| `LICENSE_KEY` | Enterprise license key | (none) |
| `LICENSE_SERVER` | License server URL | `https://license.penguintech.io` |
| `RELEASE_MODE` | Enable license enforcement | `false` |

### Database-Specific Variables

For route authentication, use environment variables:

```yaml
routes:
  - name: "mysql-primary"
    password: "${MYSQL_PASSWORD}"
```

Environment variables for databases:
- `MYSQL_PASSWORD`
- `POSTGRESQL_PASSWORD`
- `MONGODB_PASSWORD`
- `REDIS_PASSWORD`
- `MSSQL_PASSWORD`

## Core Settings

### gRPC Server Configuration

```yaml
grpc_addr: "0.0.0.0"    # Listen address (0.0.0.0 for all interfaces)
grpc_port: 50052        # Listen port
```

gRPC is used for:
- ModuleService integration with NLB
- Configuration management
- Health checks
- Metrics reporting

### Metrics Endpoint

```yaml
metrics_addr: ":7002"   # HTTP metrics server address
```

Endpoints available:
- `GET http://localhost:7002/healthz` - Health check
- `GET http://localhost:7002/metrics` - Prometheus metrics
- `GET http://localhost:7002/status` - Detailed status

### Manager Integration

```yaml
manager_url: "http://api-server:8000"
cluster_api_key: "${CLUSTER_API_KEY}"
registration_url: "http://api-server:8000/api/proxy/register"
```

DBLB registers itself with the Manager API on startup:
1. Connects to Manager at `manager_url`
2. Uses `cluster_api_key` for authentication
3. Registers via `registration_url`

## Connection Pooling

### Pool Configuration

```yaml
max_connections_per_route: 100     # Max connections per route
connection_idle_timeout: 5m         # Idle connection timeout
connection_max_lifetime: 30m        # Max connection lifetime
```

**Parameter Meanings:**

- **max_connections_per_route**: Maximum concurrent connections per route. Set based on backend capacity.
- **connection_idle_timeout**: How long idle connections are kept. Lower = faster cleanup, higher = reduced reconnections.
- **connection_max_lifetime**: Maximum lifetime of connection. Prevents connection staleness.

**Per-Route Overrides:**

```yaml
routes:
  - name: "mysql-primary"
    max_connections: 200              # Override global setting
    connection_idle_timeout: 10m      # Override global setting
    connection_max_lifetime: 60m      # Override global setting
```

## Rate Limiting

### Global Rate Limiting

```yaml
enable_rate_limiting: true
default_connection_rate: 100.0  # connections per second
default_query_rate: 1000.0      # queries per second
```

### Per-Route Rate Limiting

```yaml
routes:
  - name: "mysql-primary"
    enable_auth: false
    connection_rate: 50.0       # Override global
    query_rate: 500.0           # Override global
```

**Rate Limiting Behavior:**

- **Connection Rate**: New connections are rejected if rate exceeded
- **Query Rate**: Queries are queued if rate exceeded (backpressure)

Metrics:
- `marchproxy_dblb_connection_rate_limited{route="..."}` - Rejected connections
- `marchproxy_dblb_query_rate_limited{route="..."}` - Queued queries

## Security Configuration

### SQL Injection Detection

```yaml
enable_sql_injection_detection: true   # Enable detection
block_suspicious_queries: true         # Block detected queries
```

**Detection Patterns:**
- UNION-based injection
- Error-based injection
- Time-based blind injection
- Comment injection
- Stacked queries
- Excessive SQL keywords

**Blocked Query Response:**
- HTTP 403 Forbidden (for REST API)
- Query rejected with log entry
- Metric: `marchproxy_dblb_queries_blocked{route="..."}`

### TLS/SSL Configuration

```yaml
routes:
  - name: "postgres-main"
    enable_ssl: true                # Enable SSL to backend
    ssl_verify: true                # Verify backend certificate
    ssl_ca_cert: "/path/to/ca.crt"  # CA certificate path
```

## Observability Configuration

### Logging

```yaml
log_level: "info"          # debug, info, warn, error
log_format: "json"         # json or text
```

Environment variable:
```bash
DBLB_LOG_LEVEL=debug ./proxy-dblb
```

### Metrics

```yaml
metrics_namespace: "marchproxy_dblb"    # Prometheus metric prefix
```

Exposed metrics:
- `marchproxy_dblb_connections_active{route="..."}`
- `marchproxy_dblb_connections_total{route="..."}`
- `marchproxy_dblb_queries_blocked{route="..."}`
- `marchproxy_dblb_pool_size{route="..."}`

### Distributed Tracing

```yaml
enable_tracing: false                           # Enable tracing
jaeger_endpoint: "http://jaeger:14268/api/traces"  # Jaeger collector
trace_sample_rate: 0.1                          # 10% sampling
```

Requires Jaeger collector to be running.

## Licensing Configuration (Enterprise)

```yaml
license_key: "${LICENSE_KEY}"
license_server: "https://license.penguintech.io"
release_mode: false                    # false=dev, true=production
```

**Development Mode (release_mode: false):**
- All routes enabled
- No license validation
- Suitable for testing

**Production Mode (release_mode: true):**
- License validation required
- Feature gating based on license
- Upstream endpoints blocked without valid license

**License Key Format:**
```
PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

Obtain from https://license.penguintech.io

## Route Configuration

Routes define database proxies. Each route listens on a port and proxies to a backend.

### Route Structure

```yaml
routes:
  - name: "unique-route-name"
    protocol: "mysql|postgresql|mongodb|redis|mssql"
    listen_port: 3306
    backend_host: "backend-server"
    backend_port: 3306
    max_connections: 100
    connection_rate: 50.0
    query_rate: 500.0
    enable_auth: false
    enable_ssl: false
    health_check_sql: "SELECT 1"
```

### MySQL Route Example

```yaml
- name: "mysql-primary"
  protocol: "mysql"
  listen_port: 3306
  backend_host: "mysql.internal"
  backend_port: 3306
  max_connections: 100
  connection_rate: 50.0
  query_rate: 500.0
  enable_auth: false
  enable_ssl: false
  health_check_sql: "SELECT 1"
```

### PostgreSQL Route Example

```yaml
- name: "postgres-main"
  protocol: "postgresql"
  listen_port: 5432
  backend_host: "postgres.internal"
  backend_port: 5432
  max_connections: 200
  connection_rate: 100.0
  query_rate: 1000.0
  enable_auth: true
  username: "dbuser"
  password: "${POSTGRESQL_PASSWORD}"
  enable_ssl: true
  ssl_verify: true
```

### MongoDB Route Example

```yaml
- name: "mongodb-cluster"
  protocol: "mongodb"
  listen_port: 27017
  backend_host: "mongodb.internal"
  backend_port: 27017
  max_connections: 150
  connection_rate: 75.0
  query_rate: 750.0
  enable_auth: true
  username: "dbuser"
  password: "${MONGODB_PASSWORD}"
  enable_ssl: true
```

### Redis Route Example

```yaml
- name: "redis-cache"
  protocol: "redis"
  listen_port: 6379
  backend_host: "redis.internal"
  backend_port: 6379
  max_connections: 500
  connection_rate: 200.0
  query_rate: 5000.0
  enable_auth: true
  password: "${REDIS_PASSWORD}"
  enable_ssl: false
```

### MSSQL Route Example

```yaml
- name: "mssql-enterprise"
  protocol: "mssql"
  listen_port: 1433
  backend_host: "mssql.internal"
  backend_port: 1433
  max_connections: 100
  connection_rate: 50.0
  query_rate: 500.0
  enable_auth: true
  username: "sa"
  password: "${MSSQL_PASSWORD}"
  enable_ssl: true
  ssl_verify: true
```

## Docker Configuration

### Docker Build

```dockerfile
# Build arguments
docker build \
  --build-arg VERSION=1.0.0 \
  --build-arg GIT_COMMIT=abc123 \
  -t marchproxy/dblb:1.0.0 .
```

Build arguments:
- `VERSION`: Application version
- `GIT_COMMIT`: Git commit hash

### Docker Run

```bash
docker run -d \
  --name dblb \
  -p 3306:3306 \
  -p 5432:5432 \
  -p 27017:27017 \
  -p 6379:6379 \
  -p 1433:1433 \
  -p 7002:7002 \
  -p 50052:50052 \
  -v /path/to/config.yaml:/app/config.yaml \
  -e CLUSTER_API_KEY=abc123 \
  marchproxy/dblb:latest
```

### Environment Variables in Docker

```bash
docker run -d \
  -e CLUSTER_API_KEY=abc123def456 \
  -e DBLB_LOG_LEVEL=debug \
  -e MYSQL_PASSWORD=mysql123 \
  -e POSTGRESQL_PASSWORD=postgres123 \
  marchproxy/dblb:latest
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  dblb:
    image: marchproxy/dblb:latest
    container_name: dblb
    environment:
      CLUSTER_API_KEY: abc123def456
      DBLB_LOG_LEVEL: info
      MYSQL_PASSWORD: mysql123
      POSTGRESQL_PASSWORD: postgres123
    ports:
      - "3306:3306"    # MySQL
      - "5432:5432"    # PostgreSQL
      - "27017:27017"  # MongoDB
      - "6379:6379"    # Redis
      - "1433:1433"    # MSSQL
      - "7002:7002"    # Metrics
      - "50052:50052"  # gRPC
    volumes:
      - ./config.yaml:/app/config.yaml:ro
    healthcheck:
      test: ["/app/proxy-dblb", "--healthcheck"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
    restart: unless-stopped
```

## Database Backend Configuration

### SQLite (Built-in)

SQLite is bundled for testing and standalone deployments:

```yaml
- name: "sqlite-local"
  protocol: "sqlite"
  listen_port: 5433
  backend_path: "/data/app.db"
  enable_auth: false
```

Environment:
- No external database needed
- Single file-based database
- Suitable for development/testing

### MySQL Backend

```bash
# Connect to MySQL
mysql -h localhost -P 3306 -u root -p

# Create user for DBLB
CREATE USER 'dblb'@'%' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON *.* TO 'dblb'@'%';
FLUSH PRIVILEGES;
```

Configuration:
```yaml
- name: "mysql-primary"
  protocol: "mysql"
  backend_host: "mysql.internal"
  backend_port: 3306
  username: "dblb"
  password: "${MYSQL_PASSWORD}"
```

### PostgreSQL Backend

```bash
# Connect to PostgreSQL
psql -h localhost -U postgres

# Create user for DBLB
CREATE USER dblb WITH PASSWORD 'password';
ALTER ROLE dblb CREATEDB;
```

Configuration:
```yaml
- name: "postgres-main"
  protocol: "postgresql"
  backend_host: "postgres.internal"
  backend_port: 5432
  username: "dblb"
  password: "${POSTGRESQL_PASSWORD}"
```

### MongoDB Backend

```bash
# Connect to MongoDB
mongosh mongodb://localhost:27017

# Create user for DBLB
db.createUser({
  user: "dblb",
  pwd: "password",
  roles: ["root"]
})
```

Configuration:
```yaml
- name: "mongodb-cluster"
  protocol: "mongodb"
  backend_host: "mongodb.internal"
  backend_port: 27017
  username: "dblb"
  password: "${MONGODB_PASSWORD}"
```

### Redis Backend

```bash
# Connect to Redis
redis-cli -h localhost

# Set password (if using auth)
CONFIG SET requirepass password
```

Configuration:
```yaml
- name: "redis-cache"
  protocol: "redis"
  backend_host: "redis.internal"
  backend_port: 6379
  password: "${REDIS_PASSWORD}"
```

### MSSQL Backend

```sql
-- Connect to MSSQL
sqlcmd -S localhost,1433 -U sa -P password

-- Create user for DBLB
CREATE LOGIN dblb WITH PASSWORD = 'password';
CREATE USER dblb FOR LOGIN dblb;
GRANT CONTROL SERVER TO dblb;
```

Configuration:
```yaml
- name: "mssql-enterprise"
  protocol: "mssql"
  backend_host: "mssql.internal"
  backend_port: 1433
  username: "sa"
  password: "${MSSQL_PASSWORD}"
```

## Health Checks

### HTTP Health Check

```bash
curl http://localhost:7002/healthz
```

Response:
```json
{
  "status": "healthy",
  "module": "dblb"
}
```

### Database Connection Health Checks

Each route can have a health check SQL query:

```yaml
routes:
  - name: "mysql-primary"
    health_check_sql: "SELECT 1"
    health_check_interval: 30s
```

Health check runs periodically to verify backend connectivity.

## Performance Tuning

### Connection Pool Tuning

**For High Throughput:**
```yaml
max_connections_per_route: 500
connection_idle_timeout: 10m
connection_max_lifetime: 60m
```

**For Low Latency:**
```yaml
max_connections_per_route: 50
connection_idle_timeout: 1m
connection_max_lifetime: 10m
```

### Rate Limiting Tuning

**For Development:**
```yaml
default_connection_rate: 1000.0
default_query_rate: 10000.0
enable_rate_limiting: false
```

**For Production:**
```yaml
default_connection_rate: 100.0
default_query_rate: 1000.0
enable_rate_limiting: true
```

### Metrics Collection Overhead

Lower metrics resolution for high-performance deployments:

```yaml
metrics_namespace: "marchproxy_dblb"
# Omit detailed metrics collection
```

## Troubleshooting Configuration

### Verify Configuration

```bash
./proxy-dblb --config config.yaml --validate
```

### Debug Configuration Loading

```bash
DBLB_LOG_LEVEL=debug ./proxy-dblb --config config.yaml
```

### Common Issues

**Issue: "Connection refused" to manager**
- Verify `manager_url` is correct
- Check network connectivity
- Verify `cluster_api_key` is set

**Issue: "Port already in use"**
- Check if another process is using the port
- Change `listen_port` in route configuration

**Issue: "Invalid database connection"**
- Verify `backend_host` and `backend_port`
- Check database credentials
- Verify database is running and accessible
