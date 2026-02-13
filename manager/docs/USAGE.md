# Usage Guide

Practical guide for using MarchProxy Manager.

## Table of Contents

1. [Starting the Manager](#starting-the-manager)
2. [User Management](#user-management)
3. [Authentication](#authentication)
4. [Cluster Management](#cluster-management)
5. [Proxy Management](#proxy-management)
6. [Service Configuration](#service-configuration)
7. [Certificate Management](#certificate-management)
8. [Monitoring](#monitoring)
9. [Common Tasks](#common-tasks)

---

## Starting the Manager

### Docker

**Start with Docker Compose:**

```bash
cd manager
docker-compose up -d
```

**Start with Docker directly:**

```bash
docker build -t marchproxy-manager:latest .
docker run -d \
  --name marchproxy-manager \
  -e DATABASE_URL=postgres://user:pass@postgres:5432/db \
  -e JWT_SECRET=$(openssl rand -base64 32) \
  -p 8000:8000 \
  marchproxy-manager:latest
```

### Verify Startup

Check health endpoint:

```bash
curl http://localhost:8000/healthz
```

Expected response:
```json
{
  "status": "healthy",
  "database": "connected",
  "license": "community"
}
```

### Initial Login

Default credentials after first startup:
- **Username**: `admin`
- **Password**: `admin123` (or value of `ADMIN_PASSWORD` env var)

**Change immediately in production:**
```bash
curl -X PUT http://localhost:8000/api/auth/profile \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"password": "NewSecurePassword123!"}'
```

---

## User Management

### Create New User

```bash
curl -X POST http://localhost:8000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john.doe",
    "email": "john@example.com",
    "password": "SecurePassword123!"
  }'
```

Response:
```json
{
  "user_id": 2,
  "username": "john.doe",
  "email": "john@example.com"
}
```

### Get User Profile

```bash
curl -X GET http://localhost:8000/api/auth/profile \
  -H "Authorization: Bearer <jwt_token>"
```

### Update User Profile

```bash
curl -X PUT http://localhost:8000/api/auth/profile \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "newemail@example.com",
    "display_name": "John Doe"
  }'
```

### Reset User Password

As admin, create new user and have them update password on first login, or:

```bash
curl -X POST http://localhost:8000/api/auth/reset-password \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id": 2, "temporary_password": "TempPass123!"}'
```

---

## Authentication

### Login Flow

**Step 1: Send credentials**

```bash
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": 1,
  "username": "admin",
  "expires_in": 3600
}
```

**Step 2: Use token in subsequent requests**

```bash
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  http://localhost:8000/api/clusters
```

### Token Refresh

Refresh before expiration:

```bash
curl -X POST http://localhost:8000/api/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"token": "current_expired_token"}'
```

### Logout

```bash
curl -X POST http://localhost:8000/api/auth/logout \
  -H "Authorization: Bearer <token>"
```

### Enable Two-Factor Authentication

**Step 1: Request 2FA setup**

```bash
curl -X POST http://localhost:8000/api/auth/2fa/enable \
  -H "Authorization: Bearer <token>"
```

Response includes QR code and backup codes:
```json
{
  "qr_code": "data:image/png;base64,...",
  "secret": "JBSWY3DPEBLW64TMMQ6XSZLUL======",
  "backup_codes": ["123456", "234567", "345678", ...]
}
```

**Step 2: Scan QR code with authenticator app (Google Authenticator, Authy, etc.)**

**Step 3: Verify 2FA setup**

```bash
curl -X POST http://localhost:8000/api/auth/2fa/verify \
  -H "Content-Type: application/json" \
  -d '{"code": "123456"}'
```

**Save backup codes securely** - use them to recover account if authenticator is lost.

---

## Cluster Management

### List All Clusters

```bash
curl -X GET "http://localhost:8000/api/clusters?page=1&per_page=50" \
  -H "Authorization: Bearer <token>"
```

### Get Cluster Details

```bash
curl -X GET http://localhost:8000/api/clusters/1 \
  -H "Authorization: Bearer <token>"
```

### Create New Cluster (Admin)

```bash
curl -X POST http://localhost:8000/api/clusters \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Production",
    "description": "Production egress cluster",
    "logging_config": {
      "syslog_enabled": false,
      "syslog_host": null,
      "log_level": "INFO"
    }
  }'
```

Response:
```json
{
  "cluster_id": 2,
  "name": "Production",
  "api_key": "cluster-key-uuid-1234...",
  "is_active": true
}
```

**Save the API key** - needed for proxy registration.

### Update Cluster

```bash
curl -X PUT http://localhost:8000/api/clusters/1 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Updated cluster description",
    "is_active": true
  }'
```

### Rotate Cluster API Key

If API key is compromised:

```bash
curl -X POST http://localhost:8000/api/clusters/1/rotate-key \
  -H "Authorization: Bearer <admin_token>"
```

Response:
```json
{
  "new_api_key": "new-cluster-key-uuid...",
  "expires_old_key_at": "2025-01-23T12:00:00Z"
}
```

Old key valid for 7 days to allow proxy updates.

### Assign User to Cluster

```bash
curl -X POST http://localhost:8000/api/clusters/1/assign-user \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 2,
    "role": "admin"
  }'
```

Roles:
- `admin` - Full cluster access
- `service_owner` - Service management only

### Configure Cluster Logging

```bash
curl -X PUT http://localhost:8000/api/clusters/1/logging \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "syslog_enabled": true,
    "syslog_host": "logs.example.com",
    "syslog_port": 514,
    "log_level": "DEBUG"
  }'
```

---

## Proxy Management

### Register Proxy

Proxies self-register with cluster API key:

```bash
curl -X POST http://localhost:8000/api/proxy/register \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_api_key": "cluster-key-uuid",
    "hostname": "proxy-01.prod.internal",
    "proxy_type": "egress",
    "version": "v0.1.0",
    "ip_address": "192.168.1.100"
  }'
```

Response:
```json
{
  "proxy_id": 1,
  "registration_token": "proxy-token-uuid",
  "expires_at": "2025-01-23T12:00:00Z"
}
```

Proxy stores this token and uses it for heartbeats and config requests.

### List All Proxies

```bash
curl -X GET "http://localhost:8000/api/proxies?cluster_id=1&status=active" \
  -H "Authorization: Bearer <token>"
```

### Get Proxy Details

```bash
curl -X GET http://localhost:8000/api/proxies/1 \
  -H "Authorization: Bearer <token>"
```

### Monitor Proxy Health

```bash
curl -X GET http://localhost:8000/api/proxies/1/metrics \
  -H "Authorization: Bearer <token>"
```

### View All Proxy Statistics

```bash
curl -X GET http://localhost:8000/api/proxies/stats \
  -H "Authorization: Bearer <token>"
```

Response:
```json
{
  "total_proxies": 5,
  "active_proxies": 4,
  "unhealthy_proxies": 1,
  "avg_cpu_usage": 30.2,
  "avg_memory_usage": 600,
  "total_connections": 6250
}
```

### Clean Up Stale Proxies

```bash
curl -X POST http://localhost:8000/api/proxies/cleanup \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{"inactive_hours": 24}'
```

Removes proxies inactive for 24+ hours.

---

## Service Configuration

### Create Service

```bash
curl -X POST http://localhost:8000/api/services \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": 1,
    "name": "web-service",
    "description": "Web service egress",
    "ports": "80,443,8080",
    "protocols": ["TCP", "UDP"]
  }'
```

### List Services

```bash
curl -X GET "http://localhost:8000/api/services?cluster_id=1" \
  -H "Authorization: Bearer <token>"
```

### Create Service Mapping

Map service IPs to backend destinations:

```bash
curl -X POST http://localhost:8000/api/mappings \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "service_id": 1,
    "source_ip": "10.0.0.100",
    "dest_ip": "8.8.8.8",
    "dest_port": 443,
    "protocol": "TCP"
  }'
```

### List Mappings

```bash
curl -X GET "http://localhost:8000/api/mappings?service_id=1" \
  -H "Authorization: Bearer <token>"
```

---

## Certificate Management

### List Certificates

```bash
curl -X GET "http://localhost:8000/api/mtls/certificates?cluster_id=1" \
  -H "Authorization: Bearer <token>"
```

### Upload Certificate

```bash
curl -X POST http://localhost:8000/api/mtls/certificates \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": 1,
    "type": "server",
    "certificate_pem": "-----BEGIN CERTIFICATE-----\n...",
    "private_key_pem": "-----BEGIN PRIVATE KEY-----\n...",
    "ca_certificate": "-----BEGIN CERTIFICATE-----\n..."
  }'
```

### Generate CA Certificate

Create self-signed CA for cluster:

```bash
curl -X POST http://localhost:8000/api/mtls/ca/generate \
  -H "Authorization: Bearer <admin_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": 1,
    "common_name": "MarchProxy-CA",
    "days_valid": 365
  }'
```

### Configure mTLS for Proxies

```bash
curl -X PUT http://localhost:8000/api/mtls/config/1/egress \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "require_client_cert": true,
    "ca_certificate_id": 1,
    "server_certificate_id": 2
  }'
```

### Test mTLS Connection

```bash
curl -X POST http://localhost:8000/api/mtls/test/connection \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"proxy_id": 1, "cluster_id": 1}'
```

---

## Monitoring

### View Health Status

```bash
curl http://localhost:8000/healthz
```

### Export Prometheus Metrics

```bash
curl http://localhost:8000/metrics
```

Or configure Prometheus scrape job:

```yaml
scrape_configs:
  - job_name: 'marchproxy-manager'
    static_configs:
      - targets: ['localhost:8000']
    metrics_path: '/metrics'
```

### Check License Status

```bash
curl http://localhost:8000/license-status
```

Response:
```json
{
  "tier": "enterprise",
  "is_valid": true,
  "max_proxies": 100,
  "active_proxies": 5,
  "expires_at": "2026-01-16T00:00:00Z"
}
```

---

## Common Tasks

### Complete Setup Workflow

1. **Start Manager**
   ```bash
   docker-compose up -d
   ```

2. **Login with admin credentials**
   ```bash
   curl -X POST http://localhost:8000/api/auth/login \
     -d '{"username": "admin", "password": "admin123"}'
   ```

3. **Create cluster**
   ```bash
   curl -X POST http://localhost:8000/api/clusters \
     -d '{"name": "Production"}'
   ```

4. **Configure logging** (optional)
   ```bash
   curl -X PUT http://localhost:8000/api/clusters/1/logging \
     -d '{"syslog_enabled": true, "syslog_host": "logs.internal"}'
   ```

5. **Create service**
   ```bash
   curl -X POST http://localhost:8000/api/services \
     -d '{"cluster_id": 1, "name": "web-service", "ports": "80,443"}'
   ```

6. **Add service mapping**
   ```bash
   curl -X POST http://localhost:8000/api/mappings \
     -d '{"service_id": 1, "source_ip": "10.0.0.100", "dest_ip": "8.8.8.8"}'
   ```

7. **Register proxy** (from proxy container)
   ```bash
   curl -X POST http://manager:8000/api/proxy/register \
     -d '{"cluster_api_key": "...", "hostname": "proxy-01", ...}'
   ```

### Backup Configuration

Export all cluster configuration:

```bash
curl -X GET http://localhost:8000/api/config/1 \
  -H "Authorization: Bearer <token>" | jq . > cluster-1-backup.json
```

### Migrate Cluster

1. Create new cluster
2. Export services and mappings from old cluster
3. Create services and mappings in new cluster
4. Migrate proxies via API key rotation or restart with new cluster key

---

## Troubleshooting

### Cannot Connect to Database

Check DATABASE_URL and ensure PostgreSQL is running:
```bash
psql $DATABASE_URL -c "SELECT 1"
```

### Invalid JWT Token

Token may be expired. Refresh it:
```bash
curl -X POST http://localhost:8000/api/auth/refresh \
  -d '{"token": "old_token"}'
```

### Proxy Registration Fails

Verify cluster API key:
```bash
curl http://localhost:8000/license-status
```

### License Validation Fails

Check license server connectivity:
```bash
curl https://license.penguintech.io/health
```

See [CONFIGURATION.md](./CONFIGURATION.md) for complete configuration details.
