# MarchProxy Docker Compose Setup Guide

Complete documentation for MarchProxy's multi-container Docker architecture with 4 core services plus infrastructure and observability components.

## Architecture Overview

MarchProxy uses a sophisticated multi-tier architecture with the following components:

### Core Services (4-Container Architecture)
1. **API Server** (FastAPI/Python) - REST API + xDS gRPC control plane
2. **Web UI** (React/Vite) - Frontend application
3. **Proxy L7** (Envoy) - Application layer (HTTP/HTTPS)
4. **Proxy L3/L4** (Go) - Transport layer (TCP/UDP)

### Infrastructure Services
- **PostgreSQL** - Primary data store
- **Redis** - Session cache and real-time data

### Observability & Monitoring
- **Jaeger** - Distributed tracing
- **Prometheus** - Metrics collection
- **Grafana** - Metrics visualization
- **AlertManager** - Alert routing
- **Loki** - Log aggregation
- **Promtail** - Log collection
- **Kibana** - Log visualization (ELK Stack)
- **Elasticsearch** - Log storage
- **Logstash** - Log processing

### Supporting Services
- **Config Sync** - Configuration synchronization
- Legacy Proxies (proxy-egress, proxy-ingress)

## Quick Start

### 1. Clone and Initialize

```bash
cd /home/penguin/code/MarchProxy

# Copy environment template
cp .env.example .env

# Review and update .env with your settings
nano .env
```

### 2. Start Services

```bash
# Start all services
./scripts/start.sh

# Or use docker-compose directly
docker-compose up -d
```

### 3. Verify Health

```bash
# Check service status
./scripts/health-check.sh

# View logs
./scripts/logs.sh -f api-server
```

### 4. Access Services

After startup (~30-60 seconds for full initialization):

| Service | URL | Credentials |
|---------|-----|-------------|
| **Web UI** | http://localhost:3000 | (See Grafana) |
| **API Server** | http://localhost:8000/docs | N/A (Swagger) |
| **Grafana** | http://localhost:3000 | admin / admin123 |
| **Prometheus** | http://localhost:9090 | N/A |
| **Jaeger Tracing** | http://localhost:16686 | N/A |
| **Kibana (Logs)** | http://localhost:5601 | N/A |
| **AlertManager** | http://localhost:9093 | N/A |
| **Envoy Admin** | http://localhost:9901/stats | N/A |

## Environment Configuration

### Required Environment Variables

```bash
# Database
POSTGRES_PASSWORD=your-secure-password
REDIS_PASSWORD=your-secure-password

# Security
SECRET_KEY=your-secure-random-key-min-32-chars

# API Server
CLUSTER_API_KEY=your-cluster-api-key

# License (optional, for Enterprise)
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

### Optional Environment Variables

```bash
# Debugging
DEBUG=false
LOG_LEVEL=info

# Observability
JAEGER_ENABLED=true
ENABLE_METRICS=true

# Feature Flags (Enterprise)
MULTI_CLOUD_ENABLED=false
ZERO_TRUST_ENABLED=false
TRAFFIC_SHAPING_ENABLED=false

# Performance Tuning
RATE_LIMIT_RPS=10000
CONNECTION_POOL_SIZE=1000
```

See `.env.example` for complete configuration options.

## Service Management Scripts

### Start Services

```bash
./scripts/start.sh
```

Starts all containers in proper dependency order:
1. Infrastructure (postgres, redis, elasticsearch)
2. Observability (jaeger, prometheus, logstash, etc.)
3. Core services (api-server, webui)
4. Proxy services (proxy-l7, proxy-l3l4)

### Stop Services

```bash
./scripts/stop.sh
```

Gracefully stops all containers, preserving volumes and data.

### Restart Services

```bash
# Restart all services
./scripts/restart.sh

# Restart specific service
./scripts/restart.sh api-server
```

### View Logs

```bash
# View logs for specific service
./scripts/logs.sh api-server

# Follow logs in real-time
./scripts/logs.sh -f api-server

# Show last 50 lines
./scripts/logs.sh -n 50 api-server

# View critical services logs
./scripts/logs.sh -f critical

# View all service logs
./scripts/logs.sh -f all
```

### Health Checks

```bash
./scripts/health-check.sh
```

Verifies all critical services are healthy and responding.

## Docker Compose File Structure

### Main Configuration (`docker-compose.yml`)

Production-ready configuration with:
- Health checks for all services
- Proper service dependencies
- Resource limits and constraints
- Named volumes for persistence
- Security configuration
- Logging configuration

### Development Override (`docker-compose.override.yml`)

Automatically loaded for development with:
- Hot-reload volume mounts
- Debug ports enabled
- Development-only environment variables
- Reduced resource limits
- Additional pprof/debugger ports

### Special Compose Files

- **`docker-compose.ci.yml`** - CI/CD pipeline configuration
- **`docker-compose.test.yml`** - Integration test configuration

## Network Architecture

All services communicate via the `marchproxy-network` bridge network:

```
172.20.0.0/16 - Bridge Network
├── api-server (8000, 18000)
├── webui (3000)
├── postgres (5432)
├── redis (6379)
├── proxy-l7 (80, 443, 8080, 9901)
├── proxy-l3l4 (8081, 8082)
├── jaeger (16686, 6831, 14250)
├── prometheus (9090)
├── grafana (3000)
├── elasticsearch (9200)
├── logstash (5514)
├── kibana (5601)
└── ... other services
```

## Volume Management

### Named Volumes (Persistent Data)

```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect marchproxy_postgres_data

# Remove unused volumes
docker volume prune
```

Key volumes:
- `postgres_data` - Database files
- `redis_data` - Cache data
- `prometheus_data` - Metrics data
- `grafana_data` - Grafana configuration
- `elasticsearch_data` - Log storage
- `manager_logs`, `proxy_*_logs` - Application logs

### Backup Volumes

```bash
# Backup postgres data
docker run --rm -v marchproxy_postgres_data:/data \
  -v /backup:/backup alpine tar czf /backup/postgres.tar.gz -C /data .

# Restore postgres data
docker run --rm -v marchproxy_postgres_data:/data \
  -v /backup:/backup alpine tar xzf /backup/postgres.tar.gz -C /data
```

## Development Workflow

### Development Setup

The `docker-compose.override.yml` file is automatically used in development:

```bash
# Development mode (override.yml automatically loaded)
docker-compose up -d

# Source code is mounted as volumes for hot-reload
# Changes to Python code reflect without container restart
# JavaScript changes require manual rebuild
```

### Debugging

#### Python (Manager/API Server)

```bash
# Access Python debugger (debugpy) on port 5678
# In your IDE, attach remote debugger to localhost:5678
docker-compose logs -f api-server  # View debug output
```

#### Go (Proxy L3/L4)

```bash
# Access pprof on port 6060 (l3l4) or 6061 (other proxies)
curl http://localhost:6060/debug/pprof/profile?seconds=30 > profile.prof
go tool pprof profile.prof
```

#### Building During Development

```bash
# Rebuild container after dependency changes
docker-compose build api-server

# Rebuild and restart
docker-compose up -d --build api-server
```

## Troubleshooting

### Services Not Starting

```bash
# Check service status
docker-compose ps

# View detailed logs
docker-compose logs api-server

# Restart specific service
./scripts/restart.sh api-server

# Check for port conflicts
lsof -i :3000  # WebUI
lsof -i :8000  # API Server
lsof -i :5432  # Postgres
```

### Database Connection Issues

```bash
# Test PostgreSQL connection
docker-compose exec postgres psql -U marchproxy -d marchproxy -c "\dt"

# Test Redis connection
docker-compose exec redis redis-cli ping

# Check database logs
./scripts/logs.sh postgres
```

### High Memory Usage

```bash
# Check container resource usage
docker stats

# Reduce memory limits in docker-compose.yml
# Reduce Elasticsearch Java heap: -Xmx512m
# Reduce Logstash Java heap: -Xmx256m
```

### Slow Startup

The startup sequence takes approximately 60-120 seconds:
1. Infrastructure services initialize (20-30s)
2. Database migrations run (10-20s)
3. Observability services start (20-30s)
4. Core services initialize (20-30s)
5. Proxies register and configure (10-20s)

Monitor progress with:
```bash
watch -n 2 docker-compose ps
./scripts/logs.sh -f api-server
```

### Port Conflicts

If services fail to start due to port conflicts:

```bash
# Check what's using port 3000 (WebUI)
lsof -i :3000

# Change port mapping in docker-compose.override.yml
# Or use docker-compose with different ports:
docker-compose -f docker-compose.yml \
  -f docker-compose.override.yml \
  -f <(echo 'services:\n  webui:\n    ports:\n      - "3001:3000"') \
  up -d
```

### Dependency Failures

Services depend on health checks. If dependencies fail:

```bash
# Check health status
docker-compose ps  # Look at STATUS column

# View dependency container logs
./scripts/logs.sh postgres   # Database
./scripts/logs.sh redis      # Cache
./scripts/logs.sh jaeger     # Tracing

# Restart failed dependency
./scripts/restart.sh postgres
./scripts/restart.sh api-server  # Will wait for postgres to be healthy
```

## Production Considerations

### Security

```bash
# Generate strong secrets
openssl rand -base64 32  # For SECRET_KEY
openssl rand -hex 16     # For API keys

# Update .env with production values
POSTGRES_PASSWORD=<strong-password>
REDIS_PASSWORD=<strong-password>
SECRET_KEY=<strong-key-min-32-chars>
CLUSTER_API_KEY=<secure-cluster-key>

# Enable TLS
MTLS_ENABLED=true
```

### Resource Limits

Add to docker-compose.yml for production:

```yaml
services:
  api-server:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### Logging

Production logging configuration:

```bash
# Syslog setup
SYSLOG_ENABLED=true
SYSLOG_HOST=logstash
SYSLOG_PORT=5514

# Log retention
LOGS_RETENTION_DAYS=30
METRICS_RETENTION_DAYS=30
TRACES_RETENTION_DAYS=7
```

### Monitoring

Enable comprehensive monitoring:

```bash
# Prometheus scrape interval
docker-compose exec prometheus curl -X POST http://localhost:9090/-/reload

# Alert rules
docker-compose exec alertmanager curl -X POST http://localhost:9093/-/reload

# Grafana provisioning
docker-compose exec grafana /bin/bash -c 'grafana-cli admin reset-admin-password <new-password>'
```

### Backup Strategy

```bash
# Daily backup script
#!/bin/bash
BACKUP_DIR="/backups/marchproxy/$(date +%Y-%m-%d)"
mkdir -p "$BACKUP_DIR"

# Backup database
docker-compose exec -T postgres pg_dump -U marchproxy marchproxy | \
  gzip > "$BACKUP_DIR/database.sql.gz"

# Backup volumes
docker-compose down  # Graceful shutdown
tar czf "$BACKUP_DIR/volumes.tar.gz" -C /var/lib/docker/volumes \
  marchproxy_postgres_data marchproxy_elasticsearch_data
docker-compose up -d  # Restart
```

## Advanced Configuration

### Custom Network Settings

```yaml
networks:
  marchproxy-network:
    driver: bridge
    driver_opts:
      com.docker.network.bridge.name: br-marchproxy
      com.docker.network.bridge.enable_icc: "true"
```

### Custom Storage Drivers

```yaml
volumes:
  postgres_data:
    driver: local
    driver_opts:
      type: tmpfs
      device: tmpfs
      o: size=1G
```

### Multiple Clusters

For multi-cluster deployments:

```bash
# Create environment-specific overrides
docker-compose -f docker-compose.yml \
  -f docker-compose.override.yml \
  -f docker-compose.prod.yml \
  --project-name marchproxy-prod \
  up -d
```

## Performance Tuning

### Database Performance

```bash
# Increase connection pool
DB_POOL_SIZE=50
DB_MAX_OVERFLOW=20

# Enable pgBouncer for connection pooling
docker-compose exec postgres apt-get update && apt-get install pgbouncer
```

### Cache Performance

```bash
# Increase Redis memory
docker-compose exec redis redis-cli CONFIG SET maxmemory 512mb

# Enable Redis persistence
APPENDONLY=yes
APPENDFSYNC=everysec
```

### Network Performance

```bash
# Enable jumbo frames
docker network inspect marchproxy-network
# Requires host network configuration

# Use host network for high-performance scenarios
network_mode: host  # In docker-compose.yml
```

## Scaling

### Horizontal Scaling

Deploy multiple instances:

```bash
# Scale webui service (requires load balancer)
docker-compose up -d --scale webui=3

# Update Nginx/load balancer config
# Point to :3000, :3001, :3002
```

### Proxy Scaling

Add more proxy instances:

```bash
# Edit docker-compose.yml to add proxy-l3l4-2, proxy-l3l4-3
services:
  proxy-l3l4-2:
    extends: proxy-l3l4
    container_name: marchproxy-proxy-l3l4-2
    ports:
      - "8083:8081"
      - "8084:8082"
    environment:
      - PROXY_NAME=proxy-l3l4-2

# Register with load balancer
```

## Container Inspection

```bash
# View container details
docker-compose ps api-server
docker inspect marchproxy-api-server

# Check container resource usage
docker stats marchproxy-api-server

# View container environment
docker-compose exec api-server env | sort

# Get container IP address
docker inspect marchproxy-api-server -f '{{.NetworkSettings.IPAddress}}'
```

## Integration Testing

Run integration tests with test composition:

```bash
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

See `docker-compose.test.yml` for test-specific configuration.

## Continuous Integration

CI/CD pipeline configuration:

```bash
docker-compose -f docker-compose.ci.yml build
docker-compose -f docker-compose.ci.yml up --abort-on-container-exit
```

See `docker-compose.ci.yml` for CI-specific setup.

## Further Documentation

- **[Docker Compose Reference](https://docs.docker.com/compose/compose-file/)** - Official Docker docs
- **[Health Checks](https://docs.docker.com/engine/reference/builder/#healthcheck)** - Container health configuration
- **[Networks](https://docs.docker.com/engine/reference/commandline/network/)** - Docker networking
- **[Volumes](https://docs.docker.com/engine/storage/volumes/)** - Persistent storage

## Support

For issues or questions:

1. Check logs: `./scripts/logs.sh -f <service>`
2. Health check: `./scripts/health-check.sh`
3. Review configuration: `.env` and `docker-compose.yml`
4. Check Docker documentation: https://docs.docker.com/

## Version Information

Current MarchProxy Version: v1.0.0.1734019200

Update version with: `./scripts/update-version.sh [major|minor|patch]`
