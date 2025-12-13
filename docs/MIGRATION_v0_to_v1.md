# MarchProxy v0.1.x → v1.0.0 Migration Guide

**Version**: v1.0.0
**Release Date**: 2025-12-12
**Migration Difficulty**: Moderate
**Estimated Duration**: 2-4 hours
**Downtime**: 30-60 minutes (with blue-green deployment: 0 minutes)

This guide provides step-by-step instructions for migrating from MarchProxy v0.1.x to v1.0.0, including breaking changes, configuration mapping, and rollback procedures.

## Breaking Changes

### Architecture Changes

**v0.1.x Architecture:**
- Single py4web application for management
- Separate egress and ingress proxy applications (Go)
- Direct service-to-service communication
- Basic eBPF support

**v1.0.0 Architecture:**
- FastAPI backend API server
- React web UI (separate from API)
- Envoy L7 proxy (replaces some py4web functions)
- Enhanced Go L3/L4 proxy with NUMA, QoS, multi-cloud routing
- xDS control plane for dynamic proxy configuration

### Configuration Format Changes

| v0.1.x | v1.0.0 | Impact |
|--------|--------|--------|
| `PROXY_TYPE=egress/ingress` | `PROXY_TYPE=l3l4` | All proxies now use unified L3/L4 type |
| PYDAL database models | SQLAlchemy models | Database schema changed |
| Inline authentication | JWT + MFA support | New auth endpoints required |
| Manager API | FastAPI endpoints | REST API endpoints changed |
| py4web templates | React components | UI completely redesigned |

### Database Schema Changes

**User Credentials:**
- Old: Plain password in database
- New: Bcrypt hashing + optional MFA secret
- Migration: All passwords must be reset by users

**Service Configuration:**
- Old: Inline rules and routes
- New: Hierarchical cluster → service → mapping structure
- Migration: Services must be re-created under clusters

**Proxy Configuration:**
- Old: File-based configuration
- New: Database-driven with xDS propagation
- Migration: Configuration pulled from database on startup

### Environment Variables

#### Removed in v1.0.0
```bash
# No longer used
PYDAL_MIGRATE=auto
AUTH_METHOD=base64  # Use JWT now
PROXY_LISTEN_IP=0.0.0.0
PROXY_LISTEN_PORT=8080
```

#### New in v1.0.0
```bash
# Required for API Server
API_BIND_ADDRESS=0.0.0.0:8000
XDS_GRPC_PORT=18000

# Required for WebUI
VITE_API_URL=http://api-server:8000

# License configuration (enhanced)
RELEASE_MODE=true  # Enables license validation
LICENSE_VALIDATION_INTERVAL=24  # hours
```

#### Changed Behavior
```bash
# Old behavior: Debug=true enabled everything
DEBUG=false        # v1.0.0: Use LOG_LEVEL instead

# Old behavior: log_level string
LOG_LEVEL=info     # v1.0.0: Now uses: debug, info, warn, error
```

## Pre-Migration Checklist

### Pre-Migration Validation

- [ ] Current v0.1.x version documented
- [ ] Complete database backup taken
- [ ] Configuration files backed up
- [ ] SSL/TLS certificates backed up
- [ ] Maintenance window scheduled
- [ ] Rollback procedure tested
- [ ] Team trained on new UI
- [ ] External service dependencies verified
- [ ] DNS entries ready for update (if needed)
- [ ] Load balancer rules reviewed

### Capacity Assessment

```bash
# Document current state
docker stats  # Capture CPU/Memory usage
curl localhost:9091/metrics | grep proxy_  # Capture metrics

# Record database size
du -sh /var/lib/postgresql/marchproxy

# Check number of services/clusters
sqlite3 storage.db "SELECT COUNT(*) FROM services;"
sqlite3 storage.db "SELECT COUNT(*) FROM clusters;"
sqlite3 storage.db "SELECT COUNT(*) FROM proxies;"
```

## Migration Paths

### Path 1: Blue-Green Deployment (Zero-Downtime)

This is the recommended approach for production deployments.

#### Step 1: Prepare v1.0.0 Environment (Blue)

```bash
# Clone environment to new server
# This is your "blue" environment with v1.0.0

# Install v1.0.0
git clone https://github.com/marchproxy/marchproxy.git \
  --branch v1.0.0 \
  /opt/marchproxy-v1

cd /opt/marchproxy-v1

# Copy configuration from v0.1.x
cp /opt/marchproxy-v0.1/.env .env.new

# Update environment variables for v1.0.0
cat > .env.new.patch <<EOF
# New v1.0.0 variables
API_BIND_ADDRESS=0.0.0.0:8000
XDS_GRPC_PORT=18000
RELEASE_MODE=true
LOG_LEVEL=info
EOF
```

#### Step 2: Prepare New Database

```bash
# Create parallel PostgreSQL instance (or use same with separate database)
createdb marchproxy_v1

# Initialize v1.0.0 schema
cd /opt/marchproxy-v1
alembic upgrade head

# Manually migrate data (see Data Migration section)
python scripts/migrate_from_v0.py
```

#### Step 3: Test v1.0.0 (Offline Testing)

```bash
# Start v1.0.0 on test ports (not affecting v0.1.x)
docker-compose \
  -f docker-compose.v1.yml \
  -p marchproxy-blue \
  up -d

# Run validation tests
./scripts/validate-migration.sh

# Test all critical functions
curl http://localhost:8001/healthz  # API Server
curl http://localhost:3001          # WebUI
```

#### Step 4: Traffic Cutover

```bash
# Update DNS/load balancer to point to v1.0.0
# Gradual: Send 10% traffic, monitor for 30 minutes
curl -X POST http://load-balancer:8080/api/weight \
  -d '{"v0.1x": 90, "v1.0.0": 10}'

# After 30 minutes: 50% traffic
curl -X POST http://load-balancer:8080/api/weight \
  -d '{"v0.1x": 50, "v1.0.0": 50}'

# After 30 minutes: 100% traffic
curl -X POST http://load-balancer:8080/api/weight \
  -d '{"v0.1x": 0, "v1.0.0": 100}'
```

#### Step 5: Verify Stability

```bash
# Monitor error rates for 1 hour
watch -n 10 "curl -s localhost:8000/metrics | grep http_requests_total"

# Check latency
curl -s localhost:8000/metrics | grep http_request_duration_seconds

# If stable: Mark migration as complete
# If issues: Revert to v0.1.x
```

### Path 2: Direct Migration (With Downtime)

For smaller deployments or scheduled maintenance windows.

#### Step 1: Backup Everything

```bash
# Full database backup
pg_dump marchproxy > /backup/marchproxy_v0.1_$(date +%s).sql

# Configuration backup
tar czf /backup/config_v0.1_$(date +%s).tar.gz \
  /etc/marchproxy \
  ~/.config/marchproxy \
  ./certs/

# Volume backup (Docker)
docker-compose exec -T postgres pg_dump -U marchproxy marchproxy | \
  gzip > /backup/postgres_v0.1_$(date +%s).sql.gz
```

#### Step 2: Stop v0.1.x

```bash
# Stop all v0.1.x services
docker-compose down

# Or if bare metal
systemctl stop marchproxy-manager
systemctl stop marchproxy-proxy-egress
systemctl stop marchproxy-proxy-ingress
```

#### Step 3: Migrate Data

```bash
# See Data Migration section below
python scripts/migrate_from_v0.py --backup-old

# Verify migration
python scripts/validate-migration.py
```

#### Step 4: Start v1.0.0

```bash
# Pull latest v1.0.0
git fetch origin v1.0.0
git checkout v1.0.0

# Update environment
cp .env.example .env
# Edit .env with your settings

# Start services
docker-compose up -d

# Verify health
./scripts/health-check.sh
```

## Data Migration

### Database Schema Mapping

#### Users Table

**v0.1.x:**
```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY,
  email TEXT UNIQUE,
  password TEXT,
  name TEXT,
  role TEXT
);
```

**v1.0.0:**
```sql
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) UNIQUE,
  password_hash VARCHAR(255),  -- bcrypt hash
  full_name VARCHAR(255),
  role VARCHAR(50),
  mfa_enabled BOOLEAN DEFAULT false,
  mfa_secret VARCHAR(255),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

**Migration Script:**
```python
# scripts/migrate_users.py
import bcrypt
from sqlalchemy import create_engine, text

engine = create_engine(os.getenv('DATABASE_URL'))

# Copy user data with password hashing
with engine.connect() as conn:
    # Get old users
    result = conn.execute(
        text("SELECT id, email, password, name, role FROM old_schema.users")
    )

    for user in result:
        # Hash password
        hashed = bcrypt.hashpw(
            user.password.encode(),
            bcrypt.gensalt()
        )

        # Insert into new schema
        conn.execute(
            text("""
            INSERT INTO users (email, password_hash, full_name, role)
            VALUES (:email, :password, :name, :role)
            """),
            {
                "email": user.email,
                "password": hashed.decode(),
                "name": user.name,
                "role": user.role
            }
        )

    conn.commit()
```

#### Services and Clusters

**v0.1.x** (flat structure):
```sql
services (id, name, ip, port, type, auth_method)
```

**v1.0.0** (hierarchical):
```sql
clusters (id, name, description)
services (id, cluster_id, name, ip, ports, type)
mappings (id, source_service_id, dest_service_id, ...)
```

**Migration Strategy:**
```python
# 1. Create default cluster
default_cluster = Cluster(
    name="default",
    description="Migrated from v0.1.x"
)
db.add(default_cluster)
db.commit()

# 2. Migrate services to cluster
for old_service in old_services:
    new_service = Service(
        cluster_id=default_cluster.id,
        name=old_service.name,
        ip=old_service.ip,
        # Convert single port to port list
        ports=[old_service.port] if old_service.port else [],
        type=old_service.type,
        auth_type="jwt"  # Default to new auth
    )
    db.add(new_service)

db.commit()
```

### Configuration Migration

#### Authentication Configuration

**v0.1.x:**
```yaml
auth:
  method: base64
  token_format: custom
  2fa_enabled: false
```

**v1.0.0:**
```yaml
auth:
  method: jwt
  jwt_algorithm: HS256
  jwt_secret_key: "your-secret"
  access_token_expire_minutes: 60
  refresh_token_expire_days: 7
  mfa_enabled: true
  mfa_issuer: "MarchProxy"
```

**Migration:**
```bash
# Old tokens will be invalidated
# Users must log in again with new JWT authentication
# Export old API keys if needed for automated systems:

# v0.1.x: Extract and document API keys
grep -r "api_key" /opt/marchproxy-v0.1/ > /backup/api_keys_v0.1.txt

# v1.0.0: Recreate API keys
curl -X POST http://localhost:8000/api/v1/clusters/1/api-keys \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "exported_key", "description": "Migrated from v0.1.x"}'
```

#### Proxy Configuration

**v0.1.x:**
```yaml
proxies:
  - name: proxy-egress-1
    type: egress
    listen_port: 8080
    admin_port: 8081
    enable_ebpf: true
```

**v1.0.0:**
```yaml
proxies:
  - id: uuid
    name: proxy-l3l4-1
    type: l3l4
    cluster_id: 1
    status: active
    capabilities: [ebpf, numa, qos, multi_cloud]
    config:
      listen_port: 8081
      admin_port: 8082
      enable_ebpf: true
      enable_xdp: false
```

**Migration:**
```bash
# Configuration is now database-driven via xDS
# Proxies pull config on registration

# Old file-based config:
cat /etc/marchproxy/proxy-config.yml

# v1.0.0: Configuration stored in database
# Proxy registers and receives config from API server
# via xDS control plane
```

### Secrets and Certificates Migration

```bash
# Back up all certificates
cp /etc/marchproxy/certs/* /backup/certs_v0.1/

# In v1.0.0, certificates are managed via API
# Upload certificates to new system:
curl -X POST http://localhost:8000/api/v1/certificates \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -F "cert=@/backup/certs_v0.1/server.crt" \
  -F "key=@/backup/certs_v0.1/server.key" \
  -F "ca=@/backup/certs_v0.1/ca.crt"
```

## Rollback Procedures

### Rollback to v0.1.x (Full Restoration)

```bash
# Stop v1.0.0
docker-compose down

# If using new database, restore old database
psql < /backup/marchproxy_v0.1_$(date +%s).sql

# Restore old configuration
tar xzf /backup/config_v0.1_$(date +%s).tar.gz -C /

# Start v0.1.x services
git checkout v0.1.x
docker-compose up -d

# Verify
curl http://localhost:8000/health
```

### Partial Rollback (Specific Component)

If a specific component fails:

```bash
# Database rollback to specific point
pg_dump marchproxy | psql marchproxy_backup
pg_restore --dbname=marchproxy /backup/marchproxy_before_migration.dump

# API server rollback
docker-compose down api-server
docker-compose up -d api-server:v0.1.0

# Check service health
curl http://localhost:8000/healthz
```

## Testing Migration

### Pre-Migration Testing

```bash
# Test data migration in staging
python -m pytest tests/migration/ -v

# Test database compatibility
alembic downgrade base
alembic upgrade head

# Test service startup
docker-compose up -d
./scripts/health-check.sh
```

### Post-Migration Testing

#### Functional Testing

```bash
# Test API endpoints
curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password"}'

# Test WebUI
curl -s http://localhost:3000 | grep -c "MarchProxy"

# Test proxy registration
curl -s http://localhost:8000/api/v1/proxies | grep '"status":"active"'

# Test service mapping
curl -s http://localhost:8000/api/v1/services | jq '.length'
```

#### Performance Testing

```bash
# Compare latency
# v0.1.x baseline
ab -n 1000 -c 10 http://old.example.com/health

# v1.0.0 new
ab -n 1000 -c 10 http://new.example.com/health

# Should be similar or better
```

#### Integration Testing

```bash
# Test actual traffic flow
# Source service → Proxy → Destination service

# Create test service
curl -X POST http://localhost:8000/api/v1/services \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": 1,
    "name": "test-service",
    "ip": "10.0.1.100",
    "ports": [8080]
  }'

# Create mapping
curl -X POST http://localhost:8000/api/v1/mappings \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "source_service_id": 1,
    "dest_service_id": 2,
    "protocols": ["tcp"]
  }'

# Test connectivity
curl http://localhost:8081/test-service/
```

## Known Issues and Workarounds

### Issue 1: JWT Token Expiration

**Problem**: Users session tokens from v0.1.x are invalidated
**Solution**: Users must log in again; implement session migration script

```python
# Migrate v0.1.x sessions to v1.0.0 JWT
from datetime import datetime, timedelta
import jwt

for old_session in old_sessions:
    token = jwt.encode(
        {
            "sub": old_session.user_id,
            "exp": datetime.utcnow() + timedelta(hours=1),
            "iat": datetime.utcnow()
        },
        settings.SECRET_KEY
    )
    # Store token mapping for users to migrate
```

### Issue 2: Database Connection Pooling

**Problem**: Old connections remain open
**Solution**: Restart all connected services

```bash
# Close all connections to old database
# (forced by system)
pg_terminate_backend(pid);

# Verify no active connections
SELECT count(*) FROM pg_stat_activity WHERE datname='marchproxy';
```

### Issue 3: Port Conflicts

**Problem**: v0.1.x and v1.0.0 proxy ports may conflict
**Solution**: Use different ports during migration

```yaml
# v0.1.x uses ports 8080, 8081
# v1.0.0 uses ports 8081, 8082
# Adjust in docker-compose.yml before starting
```

## Downtime Estimation

| Scenario | Downtime | Approach |
|----------|----------|----------|
| Small deployment (<10 services) | 15-30 min | Direct migration |
| Medium deployment (10-100 services) | 30-60 min | Direct migration |
| Large deployment (100+ services) | 0 min | Blue-green deployment |
| With performance tuning | +15-30 min | Any approach |

## Support and Troubleshooting

### Common Migration Issues

**Issue: "Services not appearing after migration"**
```bash
# Check if services were migrated to correct cluster
curl http://localhost:8000/api/v1/clusters/1/services | jq '.length'

# If empty, re-run migration script
python scripts/migrate_from_v0.py --force
```

**Issue: "Proxies not registering"**
```bash
# Verify proxy can reach API server
curl http://api-server:8000/healthz

# Check proxy logs
docker-compose logs proxy-l3l4

# Verify CLUSTER_API_KEY
echo $CLUSTER_API_KEY
```

**Issue: "Authentication failing"**
```bash
# Old base64 tokens no longer work
# Create new API key in v1.0.0:
curl -X POST http://localhost:8000/api/v1/clusters/1/api-keys \
  -H "Authorization: Bearer $NEW_JWT"
```

### Getting Help

- **Documentation**: https://docs.marchproxy.io
- **GitHub Issues**: https://github.com/marchproxy/marchproxy/issues
- **Community Forum**: https://community.marchproxy.io
- **Enterprise Support**: support@marchproxy.io

---

**Document Version**: v1.0.0
**Last Updated**: 2025-12-12
**Migration Hotline**: migration-support@marchproxy.io
