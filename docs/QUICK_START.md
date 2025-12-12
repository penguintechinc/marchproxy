# MarchProxy v1.0.0 Quick Start Guide

**Time to First Proxy:** 5-10 minutes (Docker Compose)
**Version:** 1.0.0
**Last Updated:** 2025-12-12

## Quick Links

- [5-Minute Docker Setup](#5-minute-docker-setup)
- [Initial Configuration](#initial-configuration)
- [Verification](#verification)
- [Next Steps](#next-steps)
- [Troubleshooting](#troubleshooting)

---

## 5-Minute Docker Setup

### Step 1: Clone Repository

```bash
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy
git checkout v1.0.0
```

### Step 2: Start Services

```bash
# Copy example environment
cp .env.example .env

# Start all containers (database, API server, WebUI, proxies)
docker-compose up -d

# Verify all services are running
docker-compose ps
```

**Expected Output:**
```
NAME              STATUS
marchproxy-postgres      Up
marchproxy-redis         Up
marchproxy-api-server    Up (healthy)
marchproxy-webui         Up
marchproxy-proxy-l7      Up
marchproxy-proxy-l3l4    Up
prometheus               Up
grafana                  Up
jaeger                   Up
```

### Step 3: Access Dashboards

| Service | URL | Credentials |
|---------|-----|-------------|
| **WebUI** | http://localhost:3000 | admin / changeme |
| **API Docs** | http://localhost:8000/docs | (API key) |
| **Prometheus** | http://localhost:9090 | - |
| **Grafana** | http://localhost:3000 | admin / admin |
| **Jaeger** | http://localhost:16686 | - |

### Step 4: Verify Health

```bash
# Check API server health
curl http://localhost:8000/api/healthz

# Check L7 proxy (Envoy)
curl http://localhost:9901/stats | head -20

# Check L3/L4 proxy (Go)
curl http://localhost:8082/metrics | head -20

# Expected responses: JSON with "healthy": true
```

---

## Initial Configuration

### Add Your First Service

```bash
# Login to get JWT token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"changeme","totp_code":"000000"}' \
  | jq -r '.data.access_token')

# Create a service pointing to your backend
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

### Create a Traffic Route

```bash
# Create mapping (traffic rule)
curl -X POST http://localhost:8000/api/v1/mappings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "source_services": ["external-clients"],
    "dest_services": ["my-backend"],
    "protocols": ["tcp"],
    "ports": [80, 443],
    "auth_required": false,
    "cluster_id": 1
  }'
```

### Configure mTLS (Optional)

```bash
# Generate mTLS CA
curl -X POST http://localhost:8000/api/v1/certificates/ca/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"validity_years": 10}'

# Generate wildcard certificate
curl -X POST http://localhost:8000/api/v1/certificates/wildcard/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"domain": "*.example.com","validity_years": 1}'
```

---

## Verification

### Test L7 Proxy (Envoy)

```bash
# HTTP request through Envoy
curl -v http://localhost:80/

# HTTPS request (self-signed cert)
curl -k -v https://localhost:443/

# gRPC request
grpcurl -plaintext -d '{}' localhost:8080 mypackage.MyService/Method
```

### Test L3/L4 Proxy (Go)

```bash
# TCP connectivity
nc -zv localhost 8081

# UDP test
echo "test" | nc -u localhost 8081

# Metrics
curl http://localhost:8082/metrics | grep -E "(tcp|udp)_(bytes|packets)_total"
```

### Check Logs

```bash
# API Server logs
docker-compose logs api-server | tail -50

# L7 Proxy logs
docker-compose logs proxy-l7 | tail -50

# L3/L4 Proxy logs
docker-compose logs proxy-l3l4 | tail -50

# Follow logs in real-time
docker-compose logs -f <service_name>
```

### Monitor Metrics

```bash
# View Prometheus metrics
curl http://localhost:9090/api/v1/query?query=up

# View specific proxy metrics
curl http://localhost:9090/api/v1/query?query=marchproxy_proxy_uptime_seconds

# Check license status
curl http://localhost:9090/api/v1/query?query=marchproxy_license_proxies_allowed
```

---

## Next Steps

### 1. Production Deployment

For production deployment with high availability:
- Review [DEPLOYMENT.md](DEPLOYMENT.md) for Kubernetes and bare metal options
- Configure SSL/TLS certificates
- Set up monitoring and alerting
- Enable XDP/AF_XDP acceleration (Enterprise)

### 2. Advanced Configuration

- **Traffic Shaping:** See [Traffic Shaping Configuration](configuration/traffic_shaping.md)
- **Multi-Cloud Routing:** See [Multi-Cloud Routing](configuration/multi_cloud.md)
- **Zero-Trust Security:** See [Zero-Trust Policies](configuration/zero_trust.md)

### 3. Enterprise Features

Unlock with license key from [license.penguintech.io](https://license.penguintech.io):

```bash
# Set license key in .env
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD

# Restart services to apply
docker-compose restart
```

### 4. Scale Horizontally

```bash
# Deploy additional proxy instances
docker-compose up -d --scale proxy-l7=3 --scale proxy-l3l4=3

# Configure external load balancer to distribute traffic
# See DEPLOYMENT.md section "Load Balancing"
```

### 5. Integrate with Monitoring

```bash
# Prometheus is already configured at http://localhost:9090
# Grafana dashboards auto-imported at http://localhost:3000

# Add custom alerts:
# See docs/operations/alerting.md for alert configuration examples
```

---

## Common Tasks

### Change Admin Password

```bash
# Access manager container
docker-compose exec api-server python -c "
from app.core.security import hash_password
pwd = input('New password: ')
print(hash_password(pwd))
"

# Update in database
docker-compose exec postgres psql -U marchproxy <<EOF
UPDATE users SET password_hash='<hash>' WHERE username='admin';
EOF
```

### Backup Configuration

```bash
# Backup database and configuration
tar -czf marchproxy-backup-$(date +%Y%m%d).tar.gz \
  .env \
  docker-compose.override.yml \
  /var/lib/docker/volumes/marchproxy_*

# Store securely
mv marchproxy-backup-*.tar.gz /backup/location/
```

### Enable XDP Acceleration

```bash
# For Go L3/L4 proxy (requires Linux 5.10+)
docker-compose exec proxy-l3l4 ./scripts/enable-xdp.sh

# Verify XDP is loaded
docker-compose exec proxy-l3l4 bpftool prog list
```

### View Distributed Tracing

```bash
# Jaeger UI (already running)
# Open browser: http://localhost:16686

# Query recent traces
curl http://localhost:16686/api/traces?service=api-server | jq '.data[0]'
```

---

## Troubleshooting

### Services won't start

```bash
# Check Docker logs
docker-compose logs

# Verify Docker socket is accessible
ls -la /var/run/docker.sock

# Increase Docker memory if needed
# Edit docker-compose.yml: mem_limit: 2g
```

### Can't connect to proxies

```bash
# Check network connectivity
docker-compose exec api-server curl -v http://proxy-l3l4:8082/metrics

# Verify port mappings
docker-compose ps

# Check firewall rules
sudo ufw status

# Temporarily disable firewall for testing
sudo ufw disable
```

### License errors

```bash
# Verify license key is set
docker-compose exec api-server echo $LICENSE_KEY

# Check license status
curl http://localhost:8000/api/v1/license/status

# Validate license online
curl -X POST https://license.penguintech.io/api/v2/validate \
  -d "{\"key\":\"$LICENSE_KEY\"}"
```

### Performance issues

```bash
# Check resource usage
docker stats

# Increase memory/CPU in docker-compose.yml
# Restart containers
docker-compose up -d --force-recreate

# Monitor in Prometheus
curl http://localhost:9090/api/v1/query?query=container_memory_usage_bytes
```

### Database connection errors

```bash
# Check PostgreSQL health
docker-compose exec postgres pg_isready

# Verify credentials
cat .env | grep DATABASE

# Reset database (WARNING: Deletes data)
docker-compose down -v
docker-compose up -d
```

---

## Getting Help

### Documentation

- **Architecture:** [ARCHITECTURE.md](ARCHITECTURE.md)
- **Deployment:** [DEPLOYMENT.md](DEPLOYMENT.md)
- **API Reference:** [API.md](API.md)
- **Troubleshooting:** [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
- **Performance:** [PERFORMANCE.md](PERFORMANCE.md)

### Community Support

- **GitHub Issues:** https://github.com/marchproxy/marchproxy/issues
- **Discussions:** https://github.com/marchproxy/marchproxy/discussions

### Enterprise Support

- **Email:** support@marchproxy.io
- **SLA:** 24/7 for critical issues
- **Slack:** Available for Enterprise customers

---

**Ready to deploy?** Continue to [DEPLOYMENT.md](DEPLOYMENT.md) for production deployment options.
