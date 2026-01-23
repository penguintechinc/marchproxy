# DBLB API Documentation

Database Load Balancer (DBLB) provides both HTTP REST API endpoints and a gRPC interface for database protocol proxying and management.

## HTTP API Endpoints

All HTTP endpoints are available on the metrics address (default: `:7002`).

### Health Check

Performs a basic health check of the DBLB module.

```http
GET /healthz
```

**Response (200 OK):**
```json
{
  "status": "healthy",
  "module": "dblb",
  "timestamp": 1703068800
}
```

**Response (503 Service Unavailable):**
```json
{
  "status": "unhealthy",
  "error": "connection pool initialization failed"
}
```

### Metrics

Returns Prometheus-formatted metrics for monitoring and observability.

```http
GET /metrics
```

**Response (200 OK):**
```
# HELP marchproxy_dblb_connections_active Active database connections
# TYPE marchproxy_dblb_connections_active gauge
marchproxy_dblb_connections_active{route="mysql-primary"} 15
marchproxy_dblb_connections_active{route="postgres-main"} 8

# HELP marchproxy_dblb_connections_total Total connections (lifetime)
# TYPE marchproxy_dblb_connections_total counter
marchproxy_dblb_connections_total{route="mysql-primary"} 250
marchproxy_dblb_connections_total{route="postgres-main"} 180

# HELP marchproxy_dblb_queries_blocked Queries blocked by security checks
# TYPE marchproxy_dblb_queries_blocked counter
marchproxy_dblb_queries_blocked{route="mysql-primary"} 2
marchproxy_dblb_queries_blocked{route="postgres-main"} 0

# HELP marchproxy_dblb_pool_size Current connection pool size
# TYPE marchproxy_dblb_pool_size gauge
marchproxy_dblb_pool_size{route="mysql-primary"} 20
marchproxy_dblb_pool_size{route="postgres-main"} 25
```

### Status

Returns detailed status and statistics for all handlers and connection pools.

```http
GET /status
```

**Response (200 OK):**
```json
{
  "module": "dblb",
  "status": "healthy",
  "uptime_seconds": 3600,
  "timestamp": 1703068800,
  "handlers": {
    "mysql-primary": {
      "protocol": "mysql",
      "listen_port": 3306,
      "status": "active",
      "connections": {
        "active": 15,
        "total": 250,
        "idle": 5
      },
      "rates": {
        "connection_rate": 2.5,
        "query_rate": 45.3
      },
      "errors": {
        "connection_errors": 0,
        "query_errors": 3
      },
      "blocked_queries": 2
    },
    "postgres-main": {
      "protocol": "postgresql",
      "listen_port": 5432,
      "status": "active",
      "connections": {
        "active": 8,
        "total": 180,
        "idle": 3
      },
      "rates": {
        "connection_rate": 1.2,
        "query_rate": 20.1
      },
      "errors": {
        "connection_errors": 0,
        "query_errors": 1
      },
      "blocked_queries": 0
    }
  }
}
```

## gRPC API

DBLB implements the ModuleService gRPC interface for integration with NLB (Network Load Balancer) and other MarchProxy components.

### Service Definition

```protobuf
service ModuleService {
  rpc GetStatus(Empty) returns (StatusResponse);
  rpc Reload(ReloadRequest) returns (Empty);
  rpc Shutdown(ShutdownRequest) returns (Empty);
  rpc GetMetrics(Empty) returns (MetricsResponse);
  rpc HealthCheck(Empty) returns (HealthCheckResponse);
  rpc GetStats(Empty) returns (StatsResponse);
}
```

### GetStatus

Returns the current operational status of the DBLB module.

**Request:**
```protobuf
message Empty {}
```

**Response:**
```protobuf
message StatusResponse {
  string module_type = 1;      // "DBLB"
  string status = 2;           // "healthy", "degraded", "unhealthy"
  double uptime = 3;           // Uptime in seconds
  int64 timestamp = 4;         // Unix timestamp
}
```

### Reload

Gracefully reloads DBLB configuration without dropping connections.

**Request:**
```protobuf
message ReloadRequest {
  bool graceful = 1;  // If true, allow in-flight queries to complete
}
```

**Response:**
```protobuf
message Empty {}
```

### Shutdown

Gracefully shuts down the DBLB module, optionally waiting for in-flight queries.

**Request:**
```protobuf
message ShutdownRequest {
  bool graceful = 1;        // If true, wait for queries to complete
  int32 timeout_seconds = 2; // Max wait time (0 = wait indefinitely)
}
```

**Response:**
```protobuf
message Empty {}
```

### GetMetrics

Returns real-time metrics for the DBLB module.

**Request:**
```protobuf
message Empty {}
```

**Response:**
```protobuf
message MetricsResponse {
  string module_type = 1;           // "DBLB"
  double uptime = 2;                // Uptime in seconds
  int64 timestamp = 3;              // Unix timestamp
  map<string, HandlerMetrics> handlers = 4;
}

message HandlerMetrics {
  int32 active_connections = 1;
  int64 total_connections = 2;
  double connection_rate = 3;
  double query_rate = 4;
  int64 queries_blocked = 5;
  int32 pool_size = 6;
}
```

### HealthCheck

Performs a deep health verification of the DBLB module and all backend connections.

**Request:**
```protobuf
message Empty {}
```

**Response:**
```protobuf
message HealthCheckResponse {
  string status = 1;  // "healthy", "degraded", "unhealthy"
  string message = 2; // Detailed health information
}
```

### GetStats

Returns detailed statistics for all handlers and connection pools.

**Request:**
```protobuf
message Empty {}
```

**Response:**
```protobuf
message StatsResponse {
  map<string, HandlerStats> handlers = 1;
  double total_uptime = 2;
}

message HandlerStats {
  string protocol = 1;
  int32 listen_port = 2;
  string status = 3;
  ConnectionStats connections = 4;
  RateLimitStats rates = 5;
  ErrorStats errors = 6;
}

message ConnectionStats {
  int32 active = 1;
  int64 total = 2;
  int32 idle = 3;
}

message RateLimitStats {
  double connection_rate = 1;
  double query_rate = 2;
}

message ErrorStats {
  int32 connection_errors = 1;
  int32 query_errors = 2;
}
```

## Database Protocol Proxying

DBLB acts as a transparent TCP proxy for database protocols. Clients connect to DBLB on the configured listen ports and are transparently proxied to backend servers.

### Supported Protocols

| Protocol   | Port | Authentication | SSL/TLS | Notes |
|------------|------|----------------|---------|-------|
| MySQL      | 3306 | Optional       | Yes     | Full protocol support |
| PostgreSQL | 5432 | Optional       | Yes     | Full protocol support |
| MongoDB    | 27017| Optional       | Yes     | Wire protocol support |
| Redis      | 6379 | Optional       | No      | Full protocol support |
| MSSQL      | 1433 | Required       | Yes     | Full protocol support |

### Example Connections

**MySQL:**
```bash
mysql -h localhost -P 3306 -u user -p
```

**PostgreSQL:**
```bash
psql -h localhost -p 5432 -U user -d database
```

**MongoDB:**
```bash
mongosh "mongodb://localhost:27017"
```

**Redis:**
```bash
redis-cli -h localhost -p 6379
```

**MSSQL:**
```bash
sqlcmd -S localhost,1433 -U sa -P password
```

## Security APIs

### SQL Injection Detection

When `enable_sql_injection_detection: true` is set in configuration, DBLB analyzes SQL queries for suspicious patterns.

**Blocked Query Patterns:**
- UNION-based injection patterns
- Error-based injection patterns
- Time-based blind injection patterns
- Stacked queries (multiple statements)
- Comment injection (/* */ and --)
- Excessive SQL keywords in user input

**Response on Blocked Query:**
```
403 Forbidden - Query blocked due to security policy
```

The blocked query is logged and counted in metrics:
```
marchproxy_dblb_queries_blocked{route="mysql-primary"} += 1
```

## Rate Limiting APIs

DBLB enforces per-route rate limits on both connections and queries.

**Connection Rate Limit:** Maximum new connections per second
- Exceeding limit: Connection rejected with TCP RST

**Query Rate Limit:** Maximum queries per second (protocol-dependent)
- Exceeding limit: Query held/queued until capacity available

**Metrics:**
```
marchproxy_dblb_connection_rate_limited{route="mysql-primary"}
marchproxy_dblb_query_rate_limited{route="mysql-primary"}
```

## Connection Pool APIs

The connection pool manages backend database connections efficiently.

**Configuration Parameters:**
- `max_connections`: Maximum connections per route (default: 100)
- `connection_idle_timeout`: Idle connection timeout (default: 5m)
- `connection_max_lifetime`: Maximum connection lifetime (default: 30m)

**Pool Metrics:**
```
marchproxy_dblb_pool_size{route="mysql-primary"}
marchproxy_dblb_pool_idle_connections{route="mysql-primary"}
marchproxy_dblb_pool_wait_time{route="mysql-primary"}
```

## Error Handling

All API endpoints return standard HTTP status codes:

- `200 OK`: Request successful
- `400 Bad Request`: Invalid request format
- `403 Forbidden`: Security policy violation (e.g., SQL injection detected)
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Unexpected server error
- `503 Service Unavailable`: Service unhealthy or unavailable

Error responses include a JSON body:
```json
{
  "error": "error message",
  "code": "ERROR_CODE",
  "timestamp": 1703068800
}
```

## Integration Examples

### Monitoring with Prometheus

Configure Prometheus to scrape DBLB metrics:

```yaml
scrape_configs:
  - job_name: 'marchproxy-dblb'
    static_configs:
      - targets: ['localhost:7002']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Health Checks with Kubernetes

Configure Kubernetes liveness and readiness probes:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 7002
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthz
    port: 7002
  initialDelaySeconds: 5
  periodSeconds: 5
```

### gRPC Client Usage

Example Go client connecting to DBLB gRPC service:

```go
import "google.golang.org/grpc"

conn, err := grpc.Dial("localhost:50052", grpc.WithInsecure())
if err != nil {
  log.Fatal(err)
}
defer conn.Close()

client := pb.NewModuleServiceClient(conn)
status, err := client.GetStatus(context.Background(), &pb.Empty{})
if err != nil {
  log.Fatal(err)
}
log.Printf("Status: %v", status)
```
