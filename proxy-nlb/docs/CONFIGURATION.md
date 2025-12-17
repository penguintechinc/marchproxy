# MarchProxy NLB Configuration Guide

## Configuration Methods

The NLB supports configuration via:
1. **YAML configuration file** - Primary method
2. **Environment variables** - Override file settings
3. **Docker environment** - Container runtime configuration
4. **Command-line flags** - Application startup

Precedence (highest to lowest):
1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

---

## Configuration File (YAML)

The default configuration file is `config.yaml` in the working directory.

### Minimal Configuration

```yaml
# Required settings
bind_addr: ":8080"
grpc_port: 50051
metrics_addr: ":8082"
manager_url: "http://api-server:8000"
cluster_api_key: "your-cluster-api-key"
```

### Complete Configuration Example

```yaml
# Server settings
bind_addr: ":8080"           # Main traffic ingress address
grpc_addr: "0.0.0.0"        # gRPC server bind address
grpc_port: 50051            # gRPC server port
metrics_addr: ":8082"       # Metrics/health server address

# Manager connection
manager_url: "http://api-server:8000"
cluster_api_key: "your-cluster-api-key"
registration_url: "http://api-server:8000/api/v1/register"

# Rate limiting
enable_rate_limiting: true
default_rate_limit: 10000.0         # Requests per second
default_burst_size: 20000.0         # Burst capacity
rate_limit_buckets:
  - name: "http_global"
    protocol: "http"
    capacity: 50000.0
    refill_rate: 10000.0
  - name: "mysql_global"
    protocol: "mysql"
    capacity: 10000.0
    refill_rate: 2000.0
  - name: "postgres_service"
    protocol: "postgresql"
    capacity: 5000.0
    refill_rate: 1000.0

# Autoscaling
enable_autoscaling: true
autoscale_interval: 30s              # Evaluation interval
scale_up_cooldown: 3m                # Wait before scaling up again
scale_down_cooldown: 5m              # Wait before scaling down again
scale_up_threshold: 80               # CPU % to trigger scale up
scale_down_threshold: 20             # CPU % to trigger scale down
scale_up_policy: "cpu"               # Policy: cpu, memory, connections
scale_up_increment: 1                # Add N replicas per scale up
scale_down_decrement: 1              # Remove N replicas per scale down

# Blue/Green deployments
enable_bluegreen: true
canary_step_size: 10                 # Traffic shift increment (%)
canary_step_duration: 2m             # Duration between increments

# Module management
max_modules_per_protocol: 50
module_health_check_interval: 10s
max_connections_per_module: 10000

# Observability
enable_tracing: false
jaeger_endpoint: "http://jaeger:6831"
trace_sample_rate: 0.1               # 10% sampling
metrics_namespace: "marchproxy_nlb"

# Licensing (Enterprise)
license_key: "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"
license_server: "https://license.penguintech.io"
release_mode: false

# Advanced features
enable_connection_pooling: true
max_connections_per_module: 10000
grpc_connection_timeout: 5s
grpc_keepalive_interval: 30s
```

---

## Environment Variables

Environment variables override file settings. All environment variables use the `MARCHPROXY_NLB_` prefix.

### Server Configuration

```bash
# Main traffic ingress
MARCHPROXY_NLB_BIND_ADDR=:8080

# gRPC server settings
MARCHPROXY_NLB_GRPC_ADDR=0.0.0.0
MARCHPROXY_NLB_GRPC_PORT=50051

# Metrics and health server
MARCHPROXY_NLB_METRICS_ADDR=:8082
```

### Manager Connection

```bash
# Manager API endpoint
MARCHPROXY_NLB_MANAGER_URL=http://api-server:8000

# Cluster API key (also supports CLUSTER_API_KEY)
MARCHPROXY_NLB_CLUSTER_API_KEY=your-cluster-api-key
CLUSTER_API_KEY=your-cluster-api-key

# Module registration endpoint
MARCHPROXY_NLB_REGISTRATION_URL=http://api-server:8000/api/v1/register
```

### Rate Limiting

```bash
# Enable/disable rate limiting
MARCHPROXY_NLB_ENABLE_RATE_LIMITING=true

# Default limits
MARCHPROXY_NLB_DEFAULT_RATE_LIMIT=10000.0
MARCHPROXY_NLB_DEFAULT_BURST_SIZE=20000.0
```

### Autoscaling

```bash
# Enable/disable autoscaling
MARCHPROXY_NLB_ENABLE_AUTOSCALING=true

# Evaluation interval
MARCHPROXY_NLB_AUTOSCALE_INTERVAL=30s

# Cooldown periods
MARCHPROXY_NLB_SCALE_UP_COOLDOWN=3m
MARCHPROXY_NLB_SCALE_DOWN_COOLDOWN=5m

# Thresholds
MARCHPROXY_NLB_SCALE_UP_THRESHOLD=80
MARCHPROXY_NLB_SCALE_DOWN_THRESHOLD=20
MARCHPROXY_NLB_SCALE_UP_POLICY=cpu
```

### Blue/Green Deployments

```bash
# Enable/disable blue/green
MARCHPROXY_NLB_ENABLE_BLUEGREEN=true

# Canary rollout settings
MARCHPROXY_NLB_CANARY_STEP_SIZE=10
MARCHPROXY_NLB_CANARY_STEP_DURATION=2m
```

### Module Management

```bash
# Module limits
MARCHPROXY_NLB_MAX_MODULES_PER_PROTOCOL=50
MARCHPROXY_NLB_MAX_CONNECTIONS_PER_MODULE=10000

# Health check interval
MARCHPROXY_NLB_MODULE_HEALTH_CHECK_INTERVAL=10s
```

### Observability

```bash
# Tracing
MARCHPROXY_NLB_ENABLE_TRACING=false
MARCHPROXY_NLB_JAEGER_ENDPOINT=http://jaeger:6831
MARCHPROXY_NLB_TRACE_SAMPLE_RATE=0.1

# Metrics namespace
MARCHPROXY_NLB_METRICS_NAMESPACE=marchproxy_nlb
```

### Licensing

```bash
# Enterprise license
MARCHPROXY_NLB_LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
MARCHPROXY_NLB_LICENSE_SERVER=https://license.penguintech.io

# Release mode (enables license enforcement)
MARCHPROXY_NLB_RELEASE_MODE=false
```

### Advanced Features

```bash
# Connection pooling
MARCHPROXY_NLB_ENABLE_CONNECTION_POOLING=true
MARCHPROXY_NLB_MAX_CONNECTIONS_PER_MODULE=10000

# gRPC communication
MARCHPROXY_NLB_GRPC_CONNECTION_TIMEOUT=5s
MARCHPROXY_NLB_GRPC_KEEPALIVE_INTERVAL=30s
```

---

## Docker Configuration

### Environment Variables in Docker Run

```bash
docker run -d \
  --name marchproxy-nlb \
  -p 8080:8080 \
  -p 8082:8082 \
  -p 50051:50051 \
  -e CLUSTER_API_KEY=your-api-key \
  -e MARCHPROXY_NLB_MANAGER_URL=http://api-server:8000 \
  -e MARCHPROXY_NLB_ENABLE_AUTOSCALING=true \
  -e MARCHPROXY_NLB_ENABLE_RATE_LIMITING=true \
  -v /path/to/config.yaml:/app/config.yaml \
  marchproxy-nlb:latest
```

### Docker Compose Configuration

```yaml
version: '3.8'

services:
  nlb:
    image: marchproxy-nlb:latest
    container_name: marchproxy-nlb
    ports:
      - "8080:8080"
      - "8082:8082"
      - "50051:50051"
    environment:
      CLUSTER_API_KEY: ${CLUSTER_API_KEY}
      MARCHPROXY_NLB_MANAGER_URL: http://api-server:8000
      MARCHPROXY_NLB_ENABLE_AUTOSCALING: "true"
      MARCHPROXY_NLB_ENABLE_RATE_LIMITING: "true"
      MARCHPROXY_NLB_ENABLE_BLUEGREEN: "true"
    volumes:
      - ./config.yaml:/app/config.yaml
    command: ["--config", "/app/config.yaml"]
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    depends_on:
      - api-server
    restart: unless-stopped

  api-server:
    image: marchproxy-manager:latest
    ports:
      - "8000:8000"
    environment:
      DATABASE_URL: postgresql://user:pass@db:5432/marchproxy
    depends_on:
      - db
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3

  db:
    image: postgres:15-bookworm
    environment:
      POSTGRES_DB: marchproxy
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

---

## Configuration Validation

The NLB validates configuration on startup:

```bash
# Configuration validation rules
- manager_url: Required, must be valid URL
- cluster_api_key: Required, must not be empty
- grpc_port: Must be 1-65535
- default_rate_limit: Must be > 0 if rate limiting enabled
- default_burst_size: Must be > 0 if rate limiting enabled
- autoscale_interval: Must be > 0 if autoscaling enabled
- scale_up_cooldown: Must be > 0 if autoscaling enabled
- scale_down_cooldown: Must be > 0 if autoscaling enabled
- canary_step_size: Must be 1-100 if blue/green enabled
- canary_step_duration: Must be > 0 if blue/green enabled
- max_modules_per_protocol: Must be > 0
- max_connections_per_module: Must be > 0
```

---

## Protocol Configuration

### Rate Limiting Buckets

Configure rate limiting per protocol:

```yaml
rate_limit_buckets:
  - name: "http_global"
    protocol: "http"
    capacity: 50000.0         # Max tokens in bucket
    refill_rate: 10000.0      # Tokens per second

  - name: "mysql_service"
    protocol: "mysql"
    capacity: 10000.0
    refill_rate: 2000.0

  - name: "postgres_enterprise"
    protocol: "postgresql"
    capacity: 25000.0
    refill_rate: 5000.0

  - name: "redis_cache"
    protocol: "redis"
    capacity: 100000.0
    refill_rate: 20000.0

  - name: "mongodb_app"
    protocol: "mongodb"
    capacity: 15000.0
    refill_rate: 3000.0

  - name: "rtmp_stream"
    protocol: "rtmp"
    capacity: 5000.0
    refill_rate: 1000.0
```

---

## Performance Tuning

### For High Throughput

```yaml
# Increase default limits
default_rate_limit: 100000.0
default_burst_size: 200000.0

# Larger rate limit buckets
rate_limit_buckets:
  - name: "http_global"
    protocol: "http"
    capacity: 500000.0
    refill_rate: 100000.0

# Larger module capacity
max_connections_per_module: 100000
max_modules_per_protocol: 200

# Enable connection pooling
enable_connection_pooling: true
```

### For Low Latency

```yaml
# Reduce autoscaling interval
autoscale_interval: 5s

# Shorter cooldowns
scale_up_cooldown: 1m
scale_down_cooldown: 2m

# Enable tracing for debugging
enable_tracing: true
trace_sample_rate: 0.01
```

### For Stability

```yaml
# Conservative rate limiting
default_rate_limit: 5000.0
default_burst_size: 10000.0

# Longer autoscaling cooldowns
scale_up_cooldown: 10m
scale_down_cooldown: 15m

# Smaller canary steps
canary_step_size: 5
canary_step_duration: 5m

# Lower max modules
max_modules_per_protocol: 20
```

---

## Troubleshooting Configuration

### Validation Errors

**"manager_url is required"**: Set `MARCHPROXY_NLB_MANAGER_URL` or add to config.yaml

**"cluster_api_key is required"**: Set `CLUSTER_API_KEY` environment variable

**"invalid grpc_port"**: Port must be 1-65535

### Runtime Issues

**High memory usage**: Reduce `max_modules_per_protocol` and `max_connections_per_module`

**Slow routing**: Enable tracing to identify bottlenecks, adjust `autoscale_interval`

**Rate limit too aggressive**: Increase `default_rate_limit` and bucket `refill_rate`

---

## Configuration Reload

The NLB does not support hot reloading. Configuration changes require restart:

```bash
# Graceful restart
docker restart marchproxy-nlb

# Or with compose
docker compose up -d nlb
```

---

## Secrets Management

### Using Environment Variables

Store secrets in `.env` file:
```bash
CLUSTER_API_KEY=your-secure-api-key
MARCHPROXY_NLB_LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

Load with Docker Compose:
```yaml
services:
  nlb:
    env_file: .env
```

### Using Docker Secrets (Swarm)

```bash
docker secret create cluster_api_key your-api-key
docker service create \
  --secret cluster_api_key \
  -e CLUSTER_API_KEY_FILE=/run/secrets/cluster_api_key \
  marchproxy-nlb:latest
```

---

## Configuration Best Practices

1. **Use configuration files for standard settings** - Easier to version control
2. **Use environment variables for secrets** - Never commit API keys
3. **Set release_mode=false in development** - All features available for testing
4. **Monitor configuration changes** - Track file changes in version control
5. **Test configuration changes** - Validate before deploying to production
6. **Keep backups** - Save working configurations as reference
7. **Document custom buckets** - Add comments explaining rate limit choices
