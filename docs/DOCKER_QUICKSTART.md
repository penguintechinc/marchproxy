# MarchProxy Docker Quick Start

Fast track to running MarchProxy with Docker Compose.

## 30-Second Setup

```bash
# 1. Clone and enter directory
cd /home/penguin/code/MarchProxy

# 2. Copy environment config
cp .env.example .env

# 3. Edit environment (optional for development)
# nano .env

# 4. Start all services
./scripts/start.sh

# 5. Wait 60 seconds for initialization
sleep 60

# 6. Verify health
./scripts/health-check.sh

# 7. Access services
# Web UI:        http://localhost:3000
# API Server:    http://localhost:8000/docs
# Grafana:       http://localhost:3000 (same port, but grafana runs here)
```

## Common Tasks

### View Real-Time Logs

```bash
# API Server
./scripts/logs.sh -f api-server

# All services
./scripts/logs.sh -f all

# Critical services only
./scripts/logs.sh -f critical
```

### Restart Services

```bash
# Restart all
./scripts/restart.sh

# Restart specific service
./scripts/restart.sh api-server
./scripts/restart.sh webui
./scripts/restart.sh postgres
```

### Stop Services

```bash
./scripts/stop.sh
```

### Check Service Status

```bash
docker-compose ps

# Detailed health check
./scripts/health-check.sh
```

### Access Database

```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U marchproxy -d marchproxy

# Example queries:
# \dt                    # List tables
# SELECT * FROM users;  # Query users table
# \q                    # Exit
```

### Test API

```bash
# Health endpoint
curl http://localhost:8000/healthz

# Get API documentation
open http://localhost:8000/docs

# Make authenticated request
curl -H "Authorization: Bearer <token>" http://localhost:8000/api/v1/clusters
```

### View Metrics

```bash
# Prometheus metrics
open http://localhost:9090

# Grafana dashboards
open http://localhost:3000
# Login: admin / admin123

# Jaeger traces
open http://localhost:16686
```

### Check Resource Usage

```bash
docker stats

# Watch container stats live
watch -n 2 docker stats
```

## Environment Variables

Most important:

```bash
# .env file
POSTGRES_PASSWORD=marchproxy123      # Change this!
REDIS_PASSWORD=redis123              # Change this!
SECRET_KEY=your-secret-key-here      # Change this!
CLUSTER_API_KEY=default-api-key      # Change this!
LICENSE_KEY=                          # Add for Enterprise
```

For development, defaults are fine. For production, use strong passwords.

## Troubleshooting

### "docker-compose: command not found"

Install Docker Compose:
```bash
# macOS
brew install docker-compose

# Linux
sudo apt-get install docker-compose

# Or use docker with compose plugin
docker compose up -d  # (instead of docker-compose)
```

### Services not starting

```bash
# Check specific service logs
./scripts/logs.sh api-server

# See what went wrong
docker-compose ps          # Check status
docker-compose logs        # See all logs

# Common issues:
# - Port already in use: Change docker-compose.yml port mappings
# - Postgres not ready: Wait longer, it needs to initialize
# - Out of memory: Close other apps or reduce container limits
```

### Cannot connect to localhost:3000

```bash
# Check if services are running
docker-compose ps

# Check if port is actually open
curl http://localhost:3000

# Try accessing from inside container
docker-compose exec webui curl http://localhost:3000

# Check Docker network
docker inspect marchproxy_marchproxy-network
```

### High memory usage

```bash
# Check what's using memory
docker stats

# Reduce limits in docker-compose.override.yml or .env
# Elasticsearch: Reduce -Xmx to 512m or 256m
# Logstash: Reduce -Xmx to 256m or 128m
```

### Database permission denied

```bash
# Reset database connection
./scripts/restart.sh postgres

# Verify connection
docker-compose exec postgres psql -U marchproxy -d marchproxy -c "\dt"

# Check environment
echo $POSTGRES_PASSWORD
cat .env | grep POSTGRES
```

## Development Mode

The `docker-compose.override.yml` is automatically used for development:

```bash
# Development mode (automatic)
docker-compose up -d

# Production mode (skip override)
docker-compose -f docker-compose.yml up -d
```

Development features:
- Hot-reload for Python code
- Volume mounts for debugging
- Debug ports (5678 for Python, 6060 for Go)
- More verbose logging

## Useful Commands

```bash
# List all containers
docker-compose ps

# Show container details
docker-compose ps api-server

# Execute command in container
docker-compose exec api-server ps aux

# View environment in container
docker-compose exec api-server env

# Stop and remove everything (keep data)
docker-compose down

# Stop and remove everything (delete data)
docker-compose down -v

# Rebuild specific service
docker-compose build api-server

# Rebuild and restart
docker-compose up -d --build api-server

# Check logs with timestamps
docker-compose logs --timestamps api-server

# Follow logs with tail
docker-compose logs -f --tail 100 api-server
```

## Next Steps

1. **Access the Web UI**: http://localhost:3000
2. **Review API docs**: http://localhost:8000/docs
3. **Monitor metrics**: http://localhost:9090 (Prometheus)
4. **View dashboards**: http://localhost:3000 (Grafana - same port!)
5. **Check traces**: http://localhost:16686 (Jaeger)

## Full Documentation

See [DOCKER_COMPOSE_SETUP.md](./docs/DOCKER_COMPOSE_SETUP.md) for comprehensive documentation covering:
- Architecture overview
- Advanced configuration
- Troubleshooting
- Performance tuning
- Scaling and production deployment
- Backup and recovery strategies

## Default Credentials

| Service | Username | Password |
|---------|----------|----------|
| Grafana | admin | admin123 |
| PostgreSQL | marchproxy | marchproxy123 |
| WebUI | (varies) | (varies) |

**Always change passwords in production!**

## Port Reference

| Service | Port | Purpose |
|---------|------|---------|
| Web UI | 3000 | Frontend application |
| API Server | 8000 | REST API & health |
| API Server | 18000 | xDS gRPC control plane |
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache |
| Prometheus | 9090 | Metrics collection |
| Grafana | 3000 | Metrics dashboard |
| Jaeger | 16686 | Distributed tracing |
| Kibana | 5601 | Log visualization |
| AlertManager | 9093 | Alert routing |
| Envoy Admin | 9901 | Envoy proxy administration |
| Proxy L3/L4 | 8081-8082 | Transport layer proxy |

## Performance Tips

```bash
# Monitor during startup
watch -n 1 docker-compose ps
./scripts/logs.sh -f critical

# For better development performance:
# - Disable monitoring services in override.yml (uncomment profiles line)
# - Reduce Elasticsearch memory (if not needed for development)
# - Use local NVMe/SSD for better I/O

# Check performance bottlenecks
docker stats --no-stream

# Limit specific service memory (in docker-compose.override.yml)
services:
  api-server:
    environment:
      - PYTHONUNBUFFERED=1  # Real-time logging
```

## Support

Run health check to diagnose issues:
```bash
./scripts/health-check.sh
```

View logs:
```bash
./scripts/logs.sh <service-name>
```

Full documentation:
```bash
cat docs/DOCKER_COMPOSE_SETUP.md
```
