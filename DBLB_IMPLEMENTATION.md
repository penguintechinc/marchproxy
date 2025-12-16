# DBLB (Database Load Balancer) Container Implementation

## Overview

The DBLB container is ported from ArticDBM to provide database proxy functionality within the MarchProxy unified NLB architecture. It implements the ModuleService gRPC interface and supports multiple database protocols.

## Architecture

```
proxy-dblb/
├── Dockerfile
├── go.mod
├── go.sum
├── main.go                          # Entry point
├── cmd/
├── internal/
│   ├── config/
│   │   └── config.go                # Configuration management
│   ├── pool/
│   │   └── pool.go                  # Connection pooling
│   ├── security/
│   │   └── checker.go               # SQL injection detection
│   ├── handlers/
│   │   ├── base.go                  # Base handler interface
│   │   ├── mysql.go                 # MySQL protocol handler
│   │   ├── postgresql.go            # PostgreSQL protocol handler
│   │   ├── mongodb.go               # MongoDB protocol handler
│   │   ├── redis.go                 # Redis protocol handler
│   │   └── mssql.go                 # MSSQL protocol handler
│   └── grpc/
│       └── server.go                # ModuleService gRPC implementation
└── proto/
    └── marchproxy/
        └── module.proto             # gRPC service definitions
```

## Supported Databases

1. **MySQL** - Connection pooling, SQL injection detection
2. **PostgreSQL** - TLS support, read/write splitting
3. **MongoDB** - BSON protocol handling, dangerous operation blocking
4. **Redis** - RESP protocol, command filtering
5. **MSSQL** - TDS protocol, Windows authentication support

## Key Features

### 1. Connection Pooling
- Pre-warmed connections for low latency
- Configurable pool sizes per database
- Automatic connection lifecycle management
- Health monitoring and recovery

### 2. Security
- SQL injection pattern detection
- Shell command execution prevention
- Blocked database/user/table lists
- Threat intelligence integration
- Default resource blocking (test, admin, system databases)

### 3. gRPC Integration
- Implements ModuleService for NLB communication
- Health checks and metrics reporting
- Dynamic route configuration
- Blue/green deployment support
- Rate limiting per route

### 4. Multiple Routes
- Support for multiple database backends per type
- Round-robin load balancing
- Read/write splitting
- Blue/green failover

## Implementation Files

### 1. Proto Definition (proto/marchproxy/module.proto)

Complete gRPC service definition with:
- ModuleStatus, ModuleMetrics
- Route configuration
- Rate limiting
- Scaling operations
- Blue/green deployment support
- Health checks

### 2. Configuration (internal/config/config.go)

```go
type Config struct {
    // Module info
    ModuleName    string
    Version       string
    GRPCPort      int

    // Database configurations
    MySQLEnabled      bool
    PostgreSQLEnabled bool
    MongoDBEnabled    bool
    RedisEnabled      bool
    MSSQLEnabled      bool

    // Backend configurations
    MySQLBackends      []Backend
    PostgreSQLBackends []Backend
    MongoDBBackends    []Backend
    RedisBackends      []Backend
    MSSQLBackends      []Backend

    // Connection pooling
    MaxConnections int

    // Security
    SQLInjectionDetection bool
    BlockingEnabled       bool
    SeedDefaultBlocked    bool

    // Redis (for config storage)
    RedisAddr     string
    RedisPassword string
    RedisDB       int
}

type Backend struct {
    Host     string
    Port     int
    User     string
    Password string
    Database string
    TLS      bool
    Weight   int  // For blue/green
}
```

### 3. Connection Pool (internal/pool/pool.go)

Ported directly from ArticDBM with optimizations:
- 80% idle connections for faster access
- 3-minute connection max lifetime
- 60-second idle timeout
- Pre-warming 30% of connections on startup
- Context-aware connection acquisition

### 4. Security Checker (internal/security/checker.go)

Comprehensive security features:
- 40+ SQL injection patterns
- Shell command detection
- Default blocked resources (system databases, admin accounts)
- Threat intelligence checking
- Query classification (read/write detection)

### 5. Database Handlers

Each handler implements:
- Protocol-specific handshaking
- Authentication integration
- Security checking (SQL injection, blocked resources)
- Threat intelligence verification
- Connection pooling
- Metrics reporting

**MySQL Handler:**
- MySQL protocol parsing
- Galera cluster detection
- Binary protocol support

**PostgreSQL Handler:**
- Startup message parsing
- TLS encryption support
- Extended query protocol

**MongoDB Handler:**
- BSON document parsing
- Dangerous operation blocking (eval, mapReduce, etc.)
- Collection-level security

**Redis Handler:**
- RESP protocol parsing
- Dangerous command blocking (FLUSHALL, EVAL, CONFIG, etc.)
- Pub/sub support

**MSSQL Handler:**
- TDS protocol support
- Windows authentication
- Encrypted connections

### 6. gRPC Server (internal/grpc/server.go)

ModuleService implementation:

```go
type DBLBServer struct {
    config    *config.Config
    handlers  map[string]Handler
    metrics   *MetricsCollector
    routes    []*Route
    instances []*Instance
    version   string
}

// Lifecycle
func (s *DBLBServer) GetStatus(ctx context.Context, req *Empty) (*ModuleStatus, error)
func (s *DBLBServer) Reload(ctx context.Context, req *ModuleConfig) (*ReloadResponse, error)

// Traffic handling
func (s *DBLBServer) CanHandle(ctx context.Context, req *TrafficProbe) (*HandleDecision, error)
func (s *DBLBServer) GetRoutes(ctx context.Context, req *Empty) (*RouteList, error)

// Rate limiting
func (s *DBLBServer) GetRateLimits(ctx context.Context, req *Empty) (*RateLimitConfig, error)
func (s *DBLBServer) SetRateLimit(ctx context.Context, req *RateLimitRequest) (*RateLimitResponse, error)

// Scaling
func (s *DBLBServer) GetMetrics(ctx context.Context, req *Empty) (*ModuleMetrics, error)
func (s *DBLBServer) Scale(ctx context.Context, req *ScaleRequest) (*ScaleResponse, error)
func (s *DBLBServer) GetInstances(ctx context.Context, req *Empty) (*InstanceList, error)

// Blue/Green
func (s *DBLBServer) SetTrafficWeight(ctx context.Context, req *TrafficWeightRequest) (*TrafficWeightResponse, error)
func (s *DBLBServer) GetActiveVersion(ctx context.Context, req *Empty) (*VersionInfo, error)
func (s *DBLBServer) Rollback(ctx context.Context, req *RollbackRequest) (*RollbackResponse, error)

// Health
func (s *DBLBServer) HealthCheck(ctx context.Context, req *Empty) (*HealthStatus, error)
func (s *DBLBServer) GetStats(ctx context.Context, req *Empty) (*ModuleStats, error)
```

### 7. Main Entry Point (main.go)

```go
func main() {
    // Load configuration
    cfg := config.LoadConfig()

    // Initialize logger
    logger := initLogger()

    // Create Redis client for coordination
    redisClient := createRedisClient(cfg)

    // Initialize database handlers based on enabled protocols
    handlers := make(map[string]Handler)

    if cfg.MySQLEnabled {
        handlers["mysql"] = handlers.NewMySQLHandler(cfg, redisClient, logger)
    }
    if cfg.PostgreSQLEnabled {
        handlers["postgresql"] = handlers.NewPostgreSQLHandler(cfg, redisClient, logger)
    }
    if cfg.MongoDBEnabled {
        handlers["mongodb"] = handlers.NewMongoDBHandler(cfg, redisClient, logger)
    }
    if cfg.RedisEnabled {
        handlers["redis"] = handlers.NewRedisHandler(cfg, redisClient, logger)
    }
    if cfg.MSSQLEnabled {
        handlers["mssql"] = handlers.NewMSSQLHandler(cfg, redisClient, logger)
    }

    // Start gRPC server for ModuleService
    grpcServer := grpc.NewDBLBServer(cfg, handlers, logger)
    go grpcServer.Start(cfg.GRPCPort)

    // Start database proxy listeners
    ctx := context.Background()
    var wg sync.WaitGroup

    for name, handler := range handlers {
        listener := startListener(name, cfg, &wg)
        go handler.Start(ctx, listener)
    }

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    logger.Info("Shutting down DBLB")
    grpcServer.Stop()
    wg.Wait()
}
```

### 8. Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /dblb main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary
COPY --from=builder /dblb .

# Expose gRPC port and database ports
EXPOSE 50051 3306 5432 27017 6379 1433

# Run the binary
ENTRYPOINT ["./dblb"]
```

### 9. go.mod

```go
module github.com/penguintechinc/marchproxy/proxy-dblb

go 1.21

require (
    github.com/go-redis/redis/v8 v8.11.5
    github.com/go-sql-driver/mysql v1.7.1
    github.com/lib/pq v1.10.9
    github.com/denisenkom/go-mssqldb v0.12.3
    go.uber.org/zap v1.26.0
    google.golang.org/grpc v1.59.0
    google.golang.org/protobuf v1.31.0
)

require (
    github.com/cespare/xxhash/v2 v2.2.0 // indirect
    github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
    github.com/golang/protobuf v1.5.3 // indirect
    go.uber.org/multierr v1.11.0 // indirect
    golang.org/x/net v0.17.0 // indirect
    golang.org/x/sys v0.13.0 // indirect
    golang.org/x/text v0.13.0 // indirect
    google.golang.org/genproto/googleapis/rpc v0.0.0-20231030173426-d783a09b4405 // indirect
)
```

## Integration with NLB

The DBLB container integrates with the NLB (proxy-nlb) through:

1. **Service Registration**: On startup, DBLB registers with NLB via gRPC
2. **Traffic Routing**: NLB detects database protocols and routes to DBLB
3. **Health Monitoring**: NLB periodically calls HealthCheck RPC
4. **Metrics Reporting**: DBLB reports metrics for auto-scaling decisions
5. **Configuration Updates**: NLB can reload DBLB config via Reload RPC

## Protocol Detection

The NLB detects database traffic by:
- **Port-based**: 3306 (MySQL), 5432 (PostgreSQL), 27017 (MongoDB), 6379 (Redis), 1433 (MSSQL)
- **Protocol inspection**: First packet analysis for protocol signatures
- **CanHandle RPC**: NLB can query DBLB if it can handle specific traffic

## Blue/Green Deployment Example

```
# Initial state - all traffic to v1
mysql-prod-v1: weight=100
mysql-prod-v2: weight=0

# Canary deployment - 10% to v2
mysql-prod-v1: weight=90
mysql-prod-v2: weight=10

# Full deployment - all traffic to v2
mysql-prod-v1: weight=0
mysql-prod-v2: weight=100

# Rollback if needed
mysql-prod-v1: weight=100
mysql-prod-v2: weight=0
```

## Rate Limiting Example

```
# Global limits
global_rps: 10000
global_max_connections: 1000

# Per-route limits
route: mysql-prod
  requests_per_second: 5000
  max_connections: 500
  burst_size: 100

route: mysql-analytics
  requests_per_second: 2000
  max_connections: 200
  burst_size: 50
```

## Auto-Scaling Example

```
# Scale up when:
- CPU > 70% for 2 minutes
- Active connections > 400
- Average latency > 100ms

# Scale down when:
- CPU < 30% for 5 minutes
- Active connections < 100
- After 10-minute cooldown period

# Limits:
min_instances: 2
max_instances: 10
```

## Environment Variables

```bash
# Module configuration
MODULE_NAME=dblb
MODULE_VERSION=1.0.0
GRPC_PORT=50051

# Database enablement
MYSQL_ENABLED=true
POSTGRESQL_ENABLED=true
MONGODB_ENABLED=true
REDIS_ENABLED=true
MSSQL_ENABLED=false

# Connection pooling
MAX_CONNECTIONS=1000

# Security
SQL_INJECTION_DETECTION=true
BLOCKING_ENABLED=true
SEED_DEFAULT_BLOCKED=true

# Redis coordination
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

# MySQL backends (JSON array)
MYSQL_BACKENDS='[{"host":"mysql-1","port":3306,"user":"proxy","password":"secret","database":"","tls":false,"weight":50},{"host":"mysql-2","port":3306,"user":"proxy","password":"secret","database":"","tls":false,"weight":50}]'

# PostgreSQL backends
POSTGRESQL_BACKENDS='[{"host":"pg-1","port":5432,"user":"proxy","password":"secret","database":"postgres","tls":true,"weight":100}]'

# MongoDB backends
MONGODB_BACKENDS='[{"host":"mongo-1","port":27017,"user":"proxy","password":"secret","database":"admin","tls":false,"weight":100}]'

# Redis backends
REDIS_BACKENDS='[{"host":"redis-1","port":6379,"user":"","password":"secret","database":"0","tls":false,"weight":100}]'
```

## Docker Compose Integration

```yaml
services:
  dblb:
    image: marchproxy-dblb:latest
    container_name: marchproxy-dblb
    environment:
      - MODULE_NAME=dblb
      - MODULE_VERSION=1.0.0
      - GRPC_PORT=50051
      - MYSQL_ENABLED=true
      - POSTGRESQL_ENABLED=true
      - MONGODB_ENABLED=true
      - REDIS_ENABLED=true
      - MAX_CONNECTIONS=1000
      - SQL_INJECTION_DETECTION=true
      - BLOCKING_ENABLED=true
      - REDIS_ADDR=redis:6379
      - MYSQL_BACKENDS=${MYSQL_BACKENDS}
      - POSTGRESQL_BACKENDS=${POSTGRESQL_BACKENDS}
    ports:
      - "50051:50051"  # gRPC
      - "3306:3306"    # MySQL
      - "5432:5432"    # PostgreSQL
      - "27017:27017"  # MongoDB
      - "6379:6379"    # Redis
      - "1433:1433"    # MSSQL
    networks:
      - marchproxy
    depends_on:
      - redis
    restart: unless-stopped

networks:
  marchproxy:
    driver: bridge
```

## Testing

### Unit Tests
```bash
cd proxy-dblb
go test ./internal/... -v
```

### Integration Tests
```bash
# Start test databases
docker-compose -f docker-compose.test.yml up -d

# Run integration tests
go test ./test/integration/... -v

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### gRPC Testing
```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Test HealthCheck
grpcurl -plaintext localhost:50051 marchproxy.module.ModuleService/HealthCheck

# Test GetStatus
grpcurl -plaintext localhost:50051 marchproxy.module.ModuleService/GetStatus

# Test GetRoutes
grpcurl -plaintext localhost:50051 marchproxy.module.ModuleService/GetRoutes

# Test GetMetrics
grpcurl -plaintext localhost:50051 marchproxy.module.ModuleService/GetMetrics
```

## Security Considerations

1. **SQL Injection Prevention**: 40+ patterns, shell command detection
2. **Blocked Resources**: System databases, admin accounts, dangerous operations
3. **Threat Intelligence**: IP blocklists, malicious patterns
4. **TLS Encryption**: Support for encrypted database connections
5. **Authentication**: Backend credential management
6. **Audit Logging**: All security events logged

## Performance Optimizations

1. **Connection Pooling**: Pre-warmed connections, optimized pool sizes
2. **Protocol Parsing**: Minimal overhead, efficient buffer management
3. **Round-Robin LB**: Atomic operations, lock-free where possible
4. **Metrics Collection**: Async reporting, batched updates
5. **gRPC Streaming**: Efficient bidirectional communication

## Monitoring and Metrics

The DBLB exposes these metrics via gRPC:
- Active connections per database
- Requests per second
- Average/P95/P99 latency
- Error rates
- SQL injection blocks
- Threat intelligence blocks
- Connection pool stats

## Next Steps

1. Generate proto code: `protoc --go_out=. --go-grpc_out=. proto/marchproxy/module.proto`
2. Implement all handler files
3. Create Dockerfile and build image
4. Integration testing with NLB
5. Performance benchmarking
6. Documentation updates

## References

- ArticDBM source: `~/code/ArticDBM/proxy/`
- MarchProxy architecture: `.PLAN-micro`
- gRPC documentation: https://grpc.io/docs/languages/go/
- Protocol specifications: MySQL, PostgreSQL, MongoDB, Redis, MSSQL wire protocols
