# Migration Guide: v0.1.x → v1.0.0

**Target Version:** 1.0.0
**Source Versions:** 0.1.0, 0.1.1
**Migration Date:** 2025-12-12

## Overview

This guide provides step-by-step instructions for migrating from MarchProxy v0.1.x to v1.0.0. The v1.0.0 release includes significant improvements to the dual proxy architecture, enhanced mTLS support, and production-ready features.

## Breaking Changes

### Architecture Changes

1. **Dual Proxy Architecture Finalized**
   - v0.1.x: Single proxy or separate ingress/egress (experimental)
   - v1.0.0: Production-ready dual proxy with mTLS

2. **mTLS Certificate Management**
   - v0.1.x: Manual certificate upload only
   - v1.0.0: Automated CA generation with ECC P-384

3. **License Enforcement**
   - v0.1.x: License validation but limited enforcement
   - v1.0.0: Strict enforcement (Community: 3 proxies max)

### Database Schema Changes

New tables in v1.0.0:
- `mtls_cas`: Certificate Authority management
- `mtls_server_certs`: Server certificates for mTLS
- `mtls_client_certs`: Client certificates for services
- `mtls_crl`: Certificate revocation lists

Modified tables:
- `proxy_servers`: Added `proxy_type` field (ingress/egress)
- `clusters`: Enhanced logging configuration fields

### Configuration Changes

**Environment Variables:**
- `PROXY_TYPE` is now required (values: `ingress` or `egress`)
- `ENABLE_MTLS` defaults to `true` (was `false` in v0.1.x)
- `MANAGER_URL` renamed from `MANAGER_HOST`

## Pre-Migration Checklist

Before starting the migration:

- [ ] Backup database: `pg_dump marchproxy > backup_v0.1.x.sql`
- [ ] Backup configuration files and certificates
- [ ] Document current proxy count and cluster setup
- [ ] Test migration in staging environment first
- [ ] Schedule maintenance window (recommended: 1-2 hours)
- [ ] Notify users of planned downtime
- [ ] Verify v1.0.0 system requirements are met

## Migration Steps

### Step 1: Backup Current System

```bash
# Stop services
docker-compose stop

# Backup database
docker-compose exec postgres pg_dump -U marchproxy marchproxy > backup_$(date +%Y%m%d_%H%M%S).sql

# Backup configuration
tar -czf marchproxy-backup-$(date +%Y%m%d).tar.gz \
  .env \
  docker-compose.yml \
  /var/lib/docker/volumes/marchproxy_postgres_data/ \
  /var/lib/docker/volumes/marchproxy_certs/

# Store backup securely
mv marchproxy-backup-*.tar.gz /path/to/secure/location/
```

### Step 2: Update Repository

```bash
# Pull latest code
git fetch origin
git checkout v1.0.0

# Review changes
git log v0.1.1..v1.0.0 --oneline
```

### Step 3: Update Environment Variables

```bash
# Edit .env file
nano .env

# Add new required variables:
PROXY_TYPE_INGRESS=ingress
PROXY_TYPE_EGRESS=egress

# Update renamed variables:
# OLD: MANAGER_HOST=http://manager:8000
# NEW: MANAGER_URL=http://manager:8000

# Add mTLS configuration:
ENABLE_MTLS=true
MTLS_CA_VALIDITY_YEARS=10
```

### Step 4: Update Docker Compose Configuration

```bash
# v1.0.0 includes updated docker-compose.yml
# If you have custom modifications, merge them carefully

# Compare your customizations
diff docker-compose.yml.backup docker-compose.yml

# Key changes to note:
# - Proxy services now require PROXY_TYPE environment variable
# - New mTLS certificate volumes
# - Updated health check endpoints
```

### Step 5: Database Migration

```bash
# Start only database service
docker-compose up -d postgres

# Wait for database to be ready
sleep 10

# Run migration script
docker-compose run --rm manager python scripts/migrate_v0.1_to_v1.0.py

# Expected output:
# ✓ Backing up current schema
# ✓ Creating new tables (mtls_cas, mtls_server_certs, mtls_client_certs, mtls_crl)
# ✓ Adding proxy_type column to proxy_servers
# ✓ Migrating existing proxy records
# ✓ Updating cluster logging configuration
# ✓ Migration completed successfully
```

### Step 6: Certificate Migration

```bash
# Generate new mTLS CA (recommended for v1.0.0)
docker-compose run --rm manager python scripts/generate_mtls_ca.py

# OR import existing certificates
docker-compose run --rm manager python scripts/import_certificates.py \
  --ca /path/to/ca.pem \
  --ca-key /path/to/ca-key.pem
```

### Step 7: Update Proxy Configuration

```bash
# Update proxy-egress configuration
docker-compose run --rm proxy-egress --validate-config

# Update proxy-ingress configuration
docker-compose run --rm proxy-ingress --validate-config

# If validation passes, start all services
docker-compose up -d
```

### Step 8: Verify Migration

```bash
# Check all services are healthy
docker-compose ps

# Verify manager health
curl http://localhost:8000/api/healthz

# Verify proxy-egress health
curl http://localhost:8081/healthz

# Verify proxy-ingress health
curl http://localhost:8082/healthz

# Check database migration version
docker-compose exec postgres psql -U marchproxy -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"

# Expected: v1.0.0
```

### Step 9: Test Core Functionality

```bash
# Test authentication
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "<your_password>", "totp_code": "123456"}'

# Test service listing
curl -H "Authorization: Bearer <jwt_token>" http://localhost:8000/api/services

# Test mTLS certificate generation
curl -X POST http://localhost:8000/api/certificates/generate-wildcard \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"domain": "test.local", "validity_years": 1}'

# Test proxy registration
docker-compose logs proxy-egress | grep "Successfully registered"
docker-compose logs proxy-ingress | grep "Successfully registered"
```

## Rollback Procedure

If migration fails or issues occur:

### Step 1: Stop v1.0.0 Services

```bash
docker-compose down
```

### Step 2: Restore v0.1.x Code

```bash
git checkout v0.1.1
```

### Step 3: Restore Database

```bash
# Start database only
docker-compose up -d postgres

# Wait for database
sleep 10

# Drop current database
docker-compose exec postgres psql -U postgres -c "DROP DATABASE marchproxy;"

# Recreate database
docker-compose exec postgres psql -U postgres -c "CREATE DATABASE marchproxy OWNER marchproxy;"

# Restore backup
docker-compose exec -T postgres psql -U marchproxy marchproxy < backup_<timestamp>.sql
```

### Step 4: Restore Configuration

```bash
# Restore .env and docker-compose.yml from backup
tar -xzf marchproxy-backup-<date>.tar.gz
```

### Step 5: Restart v0.1.x Services

```bash
docker-compose up -d
```

## Post-Migration Tasks

After successful migration to v1.0.0:

### 1. Update Documentation

- Update internal runbooks with v1.0.0 procedures
- Document new mTLS certificate locations
- Update monitoring dashboards for new metrics

### 2. Reconfigure Monitoring

```bash
# Update Prometheus scrape configs
# v1.0.0 includes new metrics:
# - marchproxy_mtls_certificates_total
# - marchproxy_proxy_type_info
# - marchproxy_license_status
```

### 3. Update Client Applications

If client applications connect to proxies:
- Update mTLS client certificates
- Update connection endpoints if changed
- Test connectivity from all client applications

### 4. Performance Tuning

```bash
# Enable XDP acceleration (Enterprise)
docker-compose exec proxy-egress ./scripts/enable_xdp.sh
docker-compose exec proxy-ingress ./scripts/enable_xdp.sh

# Verify XDP programs loaded
docker-compose exec proxy-egress cat /sys/kernel/debug/bpf/prog_id
```

### 5. Security Hardening

- Rotate all API keys and tokens
- Review and update firewall rules
- Enable audit logging
- Configure SIEM integration

## Common Migration Issues

### Issue 1: Database Migration Fails

**Symptom:** Migration script errors with "column already exists"

**Solution:**
```bash
# Check if partial migration occurred
docker-compose exec postgres psql -U marchproxy -c "\d proxy_servers"

# If proxy_type column exists, skip that part
docker-compose run --rm manager python scripts/migrate_v0.1_to_v1.0.py --skip-schema-changes
```

### Issue 2: Proxy Registration Fails

**Symptom:** Proxies cannot register with manager after migration

**Solution:**
```bash
# Regenerate cluster API keys
docker-compose exec manager python scripts/rotate_cluster_keys.py

# Update proxy environment variables
nano .env
# Update CLUSTER_API_KEY with new value

# Restart proxies
docker-compose restart proxy-egress proxy-ingress
```

### Issue 3: mTLS Certificate Issues

**Symptom:** "certificate verify failed" errors in proxy logs

**Solution:**
```bash
# Regenerate mTLS certificates
docker-compose run --rm manager python scripts/regenerate_mtls.py

# Restart proxies to load new certificates
docker-compose restart proxy-egress proxy-ingress
```

## Support

If you encounter issues during migration:

1. **Check logs:** `docker-compose logs -f <service>`
2. **Review migration script output:** Look for specific error messages
3. **Consult troubleshooting guide:** [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
4. **Community support:** GitHub Issues
5. **Enterprise support:** support@marchproxy.io (Enterprise customers)

---

**Estimated Migration Time:**
- Small deployment (<5 proxies): 30-60 minutes
- Medium deployment (5-20 proxies): 1-2 hours
- Large deployment (>20 proxies): 2-4 hours

**Recommended Migration Window:** Off-peak hours with full maintenance window
