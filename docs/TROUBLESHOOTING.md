# MarchProxy Troubleshooting Guide

**Version:** 1.0.0
**Last Updated:** 2025-12-12

## Table of Contents

- [Common Issues](#common-issues)
- [Manager Issues](#manager-issues)
- [Proxy Issues](#proxy-issues)
- [mTLS Certificate Issues](#mtls-certificate-issues)
- [Performance Issues](#performance-issues)
- [Network Issues](#network-issues)
- [Database Issues](#database-issues)
- [License Issues](#license-issues)
- [Diagnostic Commands](#diagnostic-commands)
- [Getting Support](#getting-support)

## Common Issues

### Service Won't Start

**Symptom:** Docker containers fail to start or immediately exit

**Diagnosis:**
```bash
# Check container status
docker-compose ps

# View container logs
docker-compose logs <service_name>

# Check for port conflicts
sudo netstat -tulpn | grep -E ':(80|443|8000|8080|8081|8082|5432|6379)'
```

**Solutions:**
1. **Port conflicts:**
   ```bash
   # Find process using port
   sudo lsof -i :8000

   # Kill conflicting process
   sudo kill -9 <PID>

   # OR change port in docker-compose.yml
   ```

2. **Insufficient permissions:**
   ```bash
   # Add user to docker group
   sudo usermod -aG docker $USER
   newgrp docker

   # Restart Docker daemon
   sudo systemctl restart docker
   ```

3. **Insufficient resources:**
   ```bash
   # Check system resources
   free -h
   df -h

   # Increase Docker memory limit
   # Edit /etc/docker/daemon.json
   {
     "default-runtime": "runc",
     "default-memory": "4g"
   }
   ```

### Configuration Not Loading

**Symptom:** Proxies show stale configuration or fail to update

**Diagnosis:**
```bash
# Check proxy logs for config fetch errors
docker-compose logs proxy-egress | grep "config"
docker-compose logs proxy-ingress | grep "config"

# Verify manager API is accessible
curl http://localhost:8000/api/healthz

# Check cluster API key
docker-compose exec proxy-egress printenv | grep CLUSTER_API_KEY
```

**Solutions:**
1. **Invalid API key:**
   ```bash
   # Regenerate cluster API key in manager
   curl -X POST http://localhost:8000/api/clusters/1/rotate-key \
     -H "Authorization: Bearer <jwt_token>"

   # Update .env with new key
   nano .env

   # Restart proxies
   docker-compose restart proxy-egress proxy-ingress
   ```

2. **Network connectivity:**
   ```bash
   # Test connectivity from proxy to manager
   docker-compose exec proxy-egress ping -c 3 manager
   docker-compose exec proxy-egress curl http://manager:8000/api/healthz
   ```

3. **Configuration cache issues:**
   ```bash
   # Clear Redis cache
   docker-compose exec redis redis-cli FLUSHDB

   # Restart manager
   docker-compose restart manager
   ```

## Manager Issues

### Database Connection Failures

**Symptom:** Manager logs show "Could not connect to database"

**Diagnosis:**
```bash
# Check PostgreSQL status
docker-compose ps postgres

# View PostgreSQL logs
docker-compose logs postgres

# Test database connection
docker-compose exec postgres psql -U marchproxy -c "SELECT version();"
```

**Solutions:**
```bash
# Restart database
docker-compose restart postgres

# Wait for database to be ready
sleep 10

# Verify connection string in .env
nano .env
# Ensure: DATABASE_URL=postgresql://marchproxy:<password>@postgres:5432/marchproxy

# Restart manager
docker-compose restart manager
```

### Authentication Failures

**Symptom:** Unable to login to web interface

**Solutions:**

1. **Forgot password:**
   ```bash
   # Reset admin password
   docker-compose exec manager python scripts/reset_admin_password.py

   # Enter new password when prompted
   ```

2. **2FA issues:**
   ```bash
   # Disable 2FA temporarily
   docker-compose exec manager python scripts/disable_2fa.py --user admin

   # Re-enable after login
   ```

3. **Session timeout:**
   ```bash
   # Increase session timeout in .env
   SESSION_TIMEOUT=7200  # 2 hours

   # Restart manager
   docker-compose restart manager
   ```

### License Validation Fails

**Symptom:** "License validation failed" in manager logs

**Diagnosis:**
```bash
# Check license server connectivity
curl https://license.penguintech.io/api/v2/health

# Verify license key format
echo $LICENSE_KEY
# Should match: PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

**Solutions:**
```bash
# Validate license manually
curl -X POST https://license.penguintech.io/api/v2/validate \
  -H "Content-Type: application/json" \
  -d '{"license_key": "'"$LICENSE_KEY"'", "product": "marchproxy"}'

# Clear license cache
docker-compose exec redis redis-cli DEL "license:*"

# Restart manager
docker-compose restart manager
```

## Proxy Issues

### Proxy Won't Register with Manager

**Symptom:** Proxy logs show "Failed to register with manager"

**Diagnosis:**
```bash
# Check proxy logs
docker-compose logs proxy-egress | grep "register"

# Verify manager API is accessible
docker-compose exec proxy-egress curl http://manager:8000/api/healthz

# Check cluster API key
docker-compose exec proxy-egress printenv CLUSTER_API_KEY
```

**Solutions:**
```bash
# Verify API key is valid
curl -H "X-Cluster-API-Key: $CLUSTER_API_KEY" \
  http://localhost:8000/api/config/1

# Regenerate API key if needed
curl -X POST http://localhost:8000/api/clusters/1/rotate-key \
  -H "Authorization: Bearer <jwt_token>"

# Update proxy environment
nano .env
docker-compose restart proxy-egress proxy-ingress
```

### High CPU/Memory Usage

**Symptom:** Proxy consumes excessive resources

**Diagnosis:**
```bash
# Check resource usage
docker stats

# View proxy metrics
curl http://localhost:8081/metrics | grep -E '(cpu|memory)'

# Check for memory leaks
docker-compose exec proxy-egress pprof heap
```

**Solutions:**
```bash
# Enable resource limits in docker-compose.yml
services:
  proxy-egress:
    deploy:
      resources:
        limits:
          cpus: '4.0'
          memory: 8G
        reservations:
          cpus: '2.0'
          memory: 4G

# Restart with limits
docker-compose up -d
```

### eBPF/XDP Programs Won't Load

**Symptom:** "Failed to load eBPF program" in logs

**Diagnosis:**
```bash
# Check kernel version
uname -r  # Should be 4.18+ for eBPF, 5.10+ for XDP

# Verify BPF filesystem is mounted
mount | grep bpf

# Check for loaded programs
sudo bpftool prog list
```

**Solutions:**
```bash
# Mount BPF filesystem if not mounted
sudo mount -t bpf none /sys/fs/bpf

# Install required kernel headers
sudo apt install linux-headers-$(uname -r)  # Ubuntu/Debian
sudo dnf install kernel-devel-$(uname -r)   # RHEL/CentOS

# Rebuild eBPF programs
cd proxy-egress/ebpf
make clean && make

# Add NET_ADMIN capability
# In docker-compose.yml:
cap_add:
  - NET_ADMIN
  - SYS_ADMIN

# Restart proxy
docker-compose restart proxy-egress
```

## mTLS Certificate Issues

### Certificate Validation Failures

**Symptom:** "certificate verify failed" errors

**Diagnosis:**
```bash
# Check certificate expiry
docker-compose exec manager python scripts/check_cert_expiry.py

# Verify certificate chain
openssl verify -CAfile ca.pem server-cert.pem

# Check certificate details
openssl x509 -in server-cert.pem -text -noout
```

**Solutions:**
```bash
# Regenerate expired certificates
docker-compose exec manager python scripts/regenerate_mtls.py

# Update certificate in proxy
docker-compose restart proxy-ingress

# Verify new certificate
curl --cacert /path/to/ca.pem https://localhost:443
```

### Client Certificate Rejected

**Symptom:** Clients cannot connect with mTLS

**Solutions:**
```bash
# Generate new client certificate
curl -X POST http://localhost:8000/api/mtls/client-cert/generate \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"common_name": "client-service", "validity_years": 1}'

# Test client certificate
curl --cert client-cert.pem \
     --key client-key.pem \
     --cacert ca.pem \
     https://localhost:443
```

## Performance Issues

### Low Throughput

**Symptom:** Throughput below expected levels

**Diagnosis:**
```bash
# Check current performance
docker-compose exec proxy-egress curl http://localhost:8081/metrics | grep throughput

# Verify hardware acceleration
docker-compose logs proxy-egress | grep -E '(XDP|eBPF)'

# Check for bottlenecks
docker stats
htop
```

**Solutions:**
```bash
# Enable XDP (Enterprise)
export ENABLE_XDP=true
docker-compose restart proxy-egress proxy-ingress

# Enable AF_XDP (Enterprise)
export ENABLE_AF_XDP=true
docker-compose restart proxy-egress proxy-ingress

# Tune NUMA affinity
export NUMA_NODE=0
docker-compose restart proxy-egress proxy-ingress

# Increase worker threads
export WORKER_THREADS=8
docker-compose restart proxy-egress proxy-ingress
```

### High Latency

**Symptom:** Request latency higher than expected

**Solutions:**
```bash
# Enable connection pooling
export ENABLE_CONNECTION_POOLING=true
export MAX_CONNECTIONS_PER_BACKEND=100

# Reduce config refresh interval
export CONFIG_REFRESH_INTERVAL=30  # seconds

# Enable caching
export ENABLE_CACHE=true
export CACHE_TTL=300  # seconds

# Restart proxies
docker-compose restart proxy-egress proxy-ingress
```

## Network Issues

### Port Already in Use

**Symptom:** "Address already in use" error

**Solutions:**
```bash
# Find process using port
sudo lsof -i :443

# Kill process
sudo kill -9 <PID>

# OR change port in docker-compose.yml
ports:
  - "8443:443"  # Use 8443 externally instead
```

### DNS Resolution Failures

**Symptom:** Cannot resolve service hostnames

**Solutions:**
```bash
# Check Docker DNS
docker-compose exec proxy-egress cat /etc/resolv.conf

# Add custom DNS servers in docker-compose.yml
services:
  proxy-egress:
    dns:
      - 8.8.8.8
      - 8.8.4.4

# Test DNS resolution
docker-compose exec proxy-egress nslookup manager
```

## Database Issues

### Database Disk Full

**Symptom:** "No space left on device" errors

**Solutions:**
```bash
# Check disk usage
df -h

# Clean up old WAL files
docker-compose exec postgres bash -c "find /var/lib/postgresql/data/pg_wal -type f -mtime +7 -delete"

# Vacuum database
docker-compose exec postgres psql -U marchproxy -c "VACUUM FULL;"

# Increase volume size
docker volume inspect marchproxy_postgres_data
# Extend underlying storage, then restart
```

### Slow Queries

**Symptom:** Database queries taking too long

**Solutions:**
```bash
# Enable query logging
docker-compose exec postgres psql -U postgres -c \
  "ALTER SYSTEM SET log_min_duration_statement = 1000;"  # Log queries > 1s

# Restart PostgreSQL
docker-compose restart postgres

# Analyze slow queries
docker-compose exec postgres psql -U marchproxy -c "SELECT * FROM pg_stat_statements ORDER BY mean_time DESC LIMIT 10;"

# Add missing indexes
docker-compose exec manager python scripts/optimize_db.py
```

## License Issues

### Proxy Count Exceeded

**Symptom:** "Maximum proxy count reached" error

**Solutions:**
```bash
# Check current proxy count
curl -H "Authorization: Bearer <jwt_token>" http://localhost:8000/api/license-status

# Community: Maximum 3 proxies
# Remove unused proxies
curl -X DELETE -H "Authorization: Bearer <jwt_token>" \
  http://localhost:8000/api/proxies/<proxy_id>

# OR upgrade to Enterprise license
# Contact sales@marchproxy.io
```

## Diagnostic Commands

### Comprehensive Health Check

```bash
#!/bin/bash
# Save as check-health.sh

echo "=== MarchProxy Health Check ==="
echo

echo "1. Container Status:"
docker-compose ps
echo

echo "2. Manager Health:"
curl -s http://localhost:8000/api/healthz | jq .
echo

echo "3. Proxy-Egress Health:"
curl -s http://localhost:8081/healthz | jq .
echo

echo "4. Proxy-Ingress Health:"
curl -s http://localhost:8082/healthz | jq .
echo

echo "5. Database Status:"
docker-compose exec -T postgres psql -U marchproxy -c "SELECT version();"
echo

echo "6. Redis Status:"
docker-compose exec -T redis redis-cli PING
echo

echo "7. Resource Usage:"
docker stats --no-stream
echo

echo "8. Recent Errors:"
docker-compose logs --tail=50 | grep -i error
```

### Performance Benchmark

```bash
#!/bin/bash
# Save as benchmark.sh

echo "=== MarchProxy Performance Benchmark ==="

# HTTP benchmark
echo "HTTP throughput test:"
ab -n 10000 -c 100 http://localhost:80/

# HTTPS benchmark
echo "HTTPS throughput test:"
ab -n 10000 -c 100 https://localhost:443/

# Proxy metrics
echo "Current metrics:"
curl -s http://localhost:8081/metrics | grep -E '(requests_total|throughput|latency)'
```

## Getting Support

### Community Support

1. **GitHub Issues:** https://github.com/marchproxy/marchproxy/issues
2. **Discussions:** https://github.com/marchproxy/marchproxy/discussions
3. **Documentation:** https://github.com/marchproxy/marchproxy/tree/main/docs

### Enterprise Support

**Email:** support@marchproxy.io
**SLA:** 24/7 for critical issues (Enterprise customers)

**When opening a support ticket, include:**
1. MarchProxy version: `docker-compose exec manager cat .version`
2. System information: `uname -a`
3. Docker version: `docker --version`
4. Relevant logs: `docker-compose logs --tail=100 <service>`
5. Configuration (sanitized): Remove secrets before sharing
6. Steps to reproduce the issue

---

**Last Updated:** 2025-12-12
**Feedback:** Please report documentation issues via GitHub
