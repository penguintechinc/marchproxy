# MarchProxy Quick Start Guide

**Time to First Proxy:** 5 minutes with Docker Compose
**Version:** 1.0.0
**Last Updated:** 2025-12-16

## Quick Links

- [30-Second Setup](#30-second-setup)
- [Service Access](#service-access)
- [Verify Health](#verify-health)
- [Initial Configuration](#initial-configuration)
- [Common Operations](#common-operations)
- [Troubleshooting](#troubleshooting)

---

## 30-Second Setup

```bash
# 1. Navigate to project
cd /home/penguin/code/MarchProxy

# 2. Copy environment config
cp .env.example .env

# 3. Start all services
./scripts/start.sh

# 4. Wait for initialization
sleep 60

# 5. Verify everything is running
./scripts/health-check.sh
```

That's it! Your MarchProxy instance is running.

---

## Service Access

After startup (~60 seconds for full initialization):

| Service | URL | Default Credentials |
|---------|-----|-------------------|
| **Web UI** | http://localhost:3000 | admin / admin123 |
| **API Docs** | http://localhost:8000/docs | N/A (Swagger UI) |
| **Prometheus** | http://localhost:9090 | N/A |
| **Grafana** | http://localhost:3000 | admin / admin123 |
| **Jaeger Tracing** | http://localhost:16686 | N/A |
| **AlertManager** | http://localhost:9093 | N/A |
| **Envoy Admin** | http://localhost:9901/stats | N/A |

---

## Verify Health

```bash
# Quick health check
./scripts/health-check.sh

# Or manually test critical endpoints
curl http://localhost:8000/healthz          # API Server
curl http://localhost:9090/api/v1/status    # Prometheus
curl http://localhost:3000/api/health       # Web UI
```

---

## Initial Configuration

### Add First Service via API

```bash
# Get authentication token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme"}' \
  | jq -r '.data.access_token')

# Create a service
curl -X POST http://localhost:8000/api/v1/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-backend",
    "ip_fqdn": "10.0.1.100",
    "port": 8080,
    "protocol": "http",
    "cluster_id": 1,
    "auth_type": "none"
  }'
```

### Configure via Web UI

1. Open http://localhost:3000
2. Login with admin / admin123
3. Navigate to Services
4. Click "Add Service" and fill in details

---

## Common Operations

### View Service Status

```bash
docker-compose ps
```

**Expected output shows running containers:**
```
NAME                    STATUS
marchproxy-postgres     Up (healthy)
marchproxy-redis        Up (healthy)
marchproxy-api-server   Up (healthy)
marchproxy-webui        Up
marchproxy-prometheus   Up
marchproxy-grafana      Up
```

### View Logs

```bash
# Specific service
./scripts/logs.sh api-server

# Follow in real-time
./scripts/logs.sh -f api-server

# All critical services
./scripts/logs.sh -f critical

# All services
./scripts/logs.sh -f all
```

### Restart Services

```bash
# Restart all services
./scripts/restart.sh

# Restart specific service
./scripts/restart.sh api-server
./scripts/restart.sh proxy-l3l4
```

### Stop Services

```bash
# Graceful stop
./scripts/stop.sh

# Stop and remove containers (keep volumes)
docker-compose down

# Stop and remove containers and volumes (WARNING: deletes data)
docker-compose down -v
```

### Access Database

```bash
# Connect to PostgreSQL
docker-compose exec postgres psql -U marchproxy -d marchproxy

# Useful commands inside psql:
# \dt                    List tables
# SELECT * FROM users;  List users
# \q                    Exit
```

### Monitor Performance

```bash
# Real-time container stats
docker stats

# Watch stats continuously
watch -n 2 docker stats

# View Prometheus metrics
curl http://localhost:9090/api/v1/query?query=up

# Check proxy uptime
curl http://localhost:9090/api/v1/query?query=marchproxy_proxy_uptime_seconds
```

### Test Proxy Connectivity

```bash
# Health endpoint
curl http://localhost:8000/healthz

# Get metrics
curl http://localhost:8082/metrics | head -20

# TCP connectivity test
nc -zv localhost 8081

# UDP test
echo "test" | nc -u localhost 8081
```

---

## Environment Configuration

Key variables in `.env`:

```bash
# Database (change for production)
POSTGRES_PASSWORD=marchproxy123
REDIS_PASSWORD=redis123

# Security (change for production)
SECRET_KEY=your-secret-key-here
CLUSTER_API_KEY=default-api-key

# License (Enterprise only)
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD

# Debugging (optional)
DEBUG=false
LOG_LEVEL=info
```

For production, update these with strong values.

---

## Common Tasks

### Change Admin Password

```bash
# Generate password hash
PASSWORD_HASH=$(docker-compose exec -T api-server python -c "
from app.core.security import hash_password
print(hash_password(input('New password: ')))
")

# Update database
docker-compose exec postgres psql -U marchproxy <<EOF
UPDATE users SET password_hash='$PASSWORD_HASH' WHERE username='admin';
EOF
```

### Backup Configuration

```bash
# Backup database and config
tar -czf marchproxy-backup-$(date +%Y%m%d).tar.gz \
  .env \
  docker-compose.override.yml

# Backup PostgreSQL data
docker-compose exec -T postgres pg_dump -U marchproxy marchproxy | \
  gzip > postgres-backup-$(date +%Y%m%d).sql.gz
```

### Restore from Backup

```bash
# Restore PostgreSQL
docker-compose exec -T postgres psql -U marchproxy marchproxy < backup.sql
```

### Enable XDP Acceleration (Optional)

```bash
# For high-performance proxy (requires Linux 5.10+)
docker-compose exec proxy-l3l4 ./scripts/enable-xdp.sh

# Verify XDP is loaded
docker-compose exec proxy-l3l4 bpftool prog list
```

### Scale Horizontally

```bash
# Scale proxy instances
docker-compose up -d --scale proxy-l7=3 --scale proxy-l3l4=3

# Configure load balancer to distribute traffic
```

---

## Troubleshooting

### Services Won't Start

```bash
# Check Docker logs
docker-compose logs

# Verify Docker socket is accessible
ls -la /var/run/docker.sock

# Check if ports are in use
lsof -i :3000   # WebUI
lsof -i :8000   # API Server
lsof -i :5432   # Postgres
```

**Solutions:**
- Kill process using port: `lsof -ti:3000 | xargs kill -9`
- Increase Docker memory limit
- Check disk space: `df -h`

### Can't Connect to Services

```bash
# Test from inside network
docker-compose exec api-server curl -v http://localhost:8000/healthz

# Check network connectivity
docker network inspect marchproxy_marchproxy-network

# Check firewall rules
sudo ufw status
sudo ufw allow 3000  # Allow port 3000
```

### Database Connection Errors

```bash
# Check PostgreSQL health
docker-compose exec postgres pg_isready

# Verify credentials
cat .env | grep POSTGRES

# Test connection
docker-compose exec postgres psql -U marchproxy -d marchproxy -c "\dt"

# Reset database (WARNING: Deletes all data)
docker-compose down -v
docker-compose up -d
```

### License Errors

```bash
# Verify license key is set
docker-compose exec api-server echo $LICENSE_KEY

# Check license status
curl http://localhost:8000/api/v1/license/status

# Validate online
curl -X POST https://license.penguintech.io/api/v2/validate \
  -H "Content-Type: application/json" \
  -d "{\"key\":\"$LICENSE_KEY\"}"
```

### High Memory Usage

```bash
# Check what's consuming memory
docker stats

# Reduce memory limits in docker-compose.yml
# Default Elasticsearch uses 1GB, can reduce to 512MB

# Restart with new limits
docker-compose up -d --force-recreate
```

### Slow Startup

Initialization takes 60-120 seconds:
1. Infrastructure startup (20-30s)
2. Database migrations (10-20s)
3. Observability services (20-30s)
4. Core services (20-30s)
5. Proxy registration (10-20s)

Monitor with:
```bash
watch -n 2 docker-compose ps
./scripts/logs.sh -f api-server
```

### "docker-compose: command not found"

```bash
# Install Docker Compose
# Linux
sudo apt-get install docker-compose

# Or use Docker's built-in compose
docker compose up -d  # (instead of docker-compose)
```

---

## Development Mode

Development configuration is automatic via `docker-compose.override.yml`:

```bash
# Development mode (override loaded automatically)
docker-compose up -d

# Features:
# - Volume mounts for hot-reload
# - Debug ports enabled (5678 for Python, 6060 for Go)
# - Verbose logging
# - Reduced resource limits
```

### Debugging Python Code

```bash
# Remote debugger on port 5678
# In your IDE: Attach to localhost:5678

# View debug output
docker-compose logs -f api-server
```

### Debugging Go Code

```bash
# pprof on port 6060
curl http://localhost:6060/debug/pprof/profile?seconds=30 > profile.prof
go tool pprof profile.prof
```

---

## Next Steps

1. **Production Deployment** - See [DEPLOYMENT.md](DEPLOYMENT.md)
2. **Advanced Configuration** - See [DOCKER_COMPOSE_SETUP.md](DOCKER_COMPOSE_SETUP.md)
3. **API Reference** - See [API.md](API.md)
4. **Troubleshooting** - See [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
5. **Architecture** - See [ARCHITECTURE.md](ARCHITECTURE.md)

---

## Docker Compose File Structure

- **`docker-compose.yml`** - Production configuration
- **`docker-compose.override.yml`** - Development overrides (auto-loaded)
- **`docker-compose.test.yml`** - Test configuration
- **`docker-compose.ci.yml`** - CI/CD configuration

---

## Useful Commands Reference

```bash
# Status and monitoring
docker-compose ps                              # List containers
docker stats                                   # Resource usage
watch -n 2 docker-compose ps                  # Monitor changes

# Logs
docker-compose logs                           # All logs
docker-compose logs -f <service>              # Follow service logs
docker-compose logs --tail 100 <service>      # Last 100 lines
docker-compose logs --timestamps <service>    # With timestamps

# Service management
docker-compose up -d                          # Start all
docker-compose down                           # Stop all (keep data)
docker-compose down -v                        # Stop all (delete data)
docker-compose restart <service>              # Restart service
docker-compose build <service>                # Rebuild image
docker-compose up -d --build <service>        # Rebuild and start

# Execution
docker-compose exec <service> <command>       # Run command in container
docker-compose exec postgres psql -U marchproxy  # Database access
docker-compose exec api-server bash           # Interactive shell

# Inspection
docker inspect <container>                    # Container details
docker-compose exec <service> env             # Environment variables
docker-compose ps <service>                   # Service details
```

---

## Getting Help

- **Health Check** - Run `./scripts/health-check.sh`
- **View Logs** - Run `./scripts/logs.sh <service>`
- **GitHub Issues** - https://github.com/marchproxy/marchproxy/issues
- **Documentation** - See `/docs` folder
- **Enterprise Support** - Email support@marchproxy.io

---

## Important Notes

- **Default credentials** (admin/admin123) must be changed in production
- **Passwords in .env** must be changed for production use
- **License key** required for Enterprise features
- **Data persistence** uses Docker volumes in `/var/lib/docker/volumes/`
- **Network** uses bridge network `marchproxy_marchproxy-network`
- **Development mode** automatically enabled if `docker-compose.override.yml` exists

**Ready to deploy?** Continue to [DEPLOYMENT.md](DEPLOYMENT.md) for production options.
