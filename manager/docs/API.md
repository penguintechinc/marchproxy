# Manager API Reference

Complete API endpoint documentation for MarchProxy Manager.

## Base URL

```
http://localhost:8000/api
```

## Authentication

All API endpoints (except `/health` and `/metrics`) require Bearer token authentication:

```
Authorization: Bearer <jwt_token>
```

Obtain JWT tokens via the authentication endpoints below.

## Endpoints

### Authentication

#### POST /auth/login
Login and receive JWT token.

**Request:**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "eyJhbGc...",
  "user_id": 1,
  "username": "admin",
  "expires_in": 3600
}
```

#### POST /auth/register
Register new user.

**Request:**
```json
{
  "username": "newuser",
  "email": "user@example.com",
  "password": "secure_password123"
}
```

**Response:**
```json
{
  "user_id": 2,
  "username": "newuser",
  "email": "user@example.com"
}
```

#### POST /auth/refresh
Refresh JWT token.

**Request:**
```json
{
  "token": "current_jwt_token"
}
```

**Response:**
```json
{
  "token": "new_jwt_token",
  "expires_in": 3600
}
```

#### POST /auth/logout
Logout current user.

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

#### GET /auth/profile
Get current user profile.

**Response:**
```json
{
  "user_id": 1,
  "username": "admin",
  "email": "admin@localhost",
  "is_admin": true,
  "is_active": true
}
```

#### PUT /auth/profile
Update user profile.

**Request:**
```json
{
  "email": "newemail@example.com",
  "display_name": "Admin User"
}
```

#### POST /auth/2fa/enable
Enable two-factor authentication.

**Response:**
```json
{
  "qr_code": "data:image/png;base64,...",
  "secret": "JBSWY3DPEBLW64TMMQ...",
  "backup_codes": ["123456", "234567", ...]
}
```

#### POST /auth/2fa/verify
Verify 2FA code during login.

**Request:**
```json
{
  "code": "123456"
}
```

**Response:**
```json
{
  "verified": true,
  "token": "eyJhbGc..."
}
```

#### POST /auth/2fa/disable
Disable two-factor authentication.

**Request:**
```json
{
  "password": "user_password"
}
```

---

### Clusters

#### GET /clusters
List all clusters (paginated).

**Query Parameters:**
- `page` (optional, default: 1)
- `per_page` (optional, default: 50)

**Response:**
```json
{
  "clusters": [
    {
      "cluster_id": 1,
      "name": "Production",
      "description": "Production cluster",
      "api_key": "***masked***",
      "is_active": true,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ],
  "total": 1,
  "page": 1
}
```

#### POST /clusters
Create new cluster (Admin only).

**Request:**
```json
{
  "name": "Staging",
  "description": "Staging cluster",
  "logging_config": {
    "syslog_enabled": false,
    "syslog_host": null,
    "syslog_port": 514,
    "log_level": "INFO"
  }
}
```

**Response:**
```json
{
  "cluster_id": 2,
  "name": "Staging",
  "api_key": "cluster-key-uuid",
  "is_active": true
}
```

#### GET /clusters/{cluster_id}
Get cluster details.

**Response:**
```json
{
  "cluster_id": 1,
  "name": "Production",
  "description": "Production cluster",
  "is_active": true,
  "proxy_count": 5,
  "service_count": 10,
  "created_at": "2025-01-01T00:00:00Z",
  "logging_config": {
    "syslog_enabled": false,
    "log_level": "INFO"
  }
}
```

#### PUT /clusters/{cluster_id}
Update cluster (Admin only).

**Request:**
```json
{
  "description": "Updated description",
  "is_active": true
}
```

#### POST /clusters/{cluster_id}/rotate-key
Rotate cluster API key (Admin only).

**Response:**
```json
{
  "new_api_key": "new-cluster-key-uuid",
  "expires_old_key_at": "2025-01-16T12:00:00Z"
}
```

#### POST /clusters/{cluster_id}/assign-user
Assign user to cluster (Admin only).

**Request:**
```json
{
  "user_id": 2,
  "role": "admin"
}
```

**Response:**
```json
{
  "assignment_id": 1,
  "user_id": 2,
  "cluster_id": 1,
  "role": "admin"
}
```

#### PUT /clusters/{cluster_id}/logging
Update cluster logging configuration.

**Request:**
```json
{
  "syslog_enabled": true,
  "syslog_host": "logs.example.com",
  "syslog_port": 514,
  "log_level": "DEBUG"
}
```

#### GET /config/{cluster_id}
Get complete cluster configuration (for proxies).

**Response:**
```json
{
  "cluster_id": 1,
  "name": "Production",
  "api_key": "cluster-key",
  "services": [...],
  "mappings": [...],
  "certificates": [...]
}
```

---

### Proxies

#### POST /proxy/register
Register new proxy (Proxy authentication via API key).

**Request:**
```json
{
  "cluster_api_key": "cluster-key-uuid",
  "hostname": "proxy-01.example.com",
  "proxy_type": "egress",
  "version": "v0.1.0",
  "ip_address": "192.168.1.100"
}
```

**Response:**
```json
{
  "proxy_id": 1,
  "registration_token": "proxy-token-uuid",
  "expires_at": "2025-01-23T12:00:00Z"
}
```

#### POST /proxy/heartbeat
Send proxy heartbeat (Proxy authentication).

**Request:**
```json
{
  "proxy_id": 1,
  "registration_token": "proxy-token-uuid",
  "status": "healthy",
  "metrics": {
    "cpu_usage": 25.5,
    "memory_usage": 512,
    "active_connections": 1250
  }
}
```

**Response:**
```json
{
  "acknowledged": true,
  "next_heartbeat_interval": 30
}
```

#### POST /proxy/config
Get proxy configuration (Proxy authentication).

**Request:**
```json
{
  "proxy_id": 1,
  "registration_token": "proxy-token-uuid"
}
```

**Response:**
```json
{
  "cluster_id": 1,
  "services": [...],
  "mappings": [...],
  "certificates": [...]
}
```

#### GET /proxies
List all proxies (paginated).

**Query Parameters:**
- `cluster_id` (optional)
- `status` (optional: active, inactive, unhealthy)
- `page` (optional)

**Response:**
```json
{
  "proxies": [
    {
      "proxy_id": 1,
      "hostname": "proxy-01.example.com",
      "proxy_type": "egress",
      "status": "active",
      "cluster_id": 1,
      "version": "v0.1.0",
      "last_heartbeat": "2025-01-16T12:30:00Z"
    }
  ],
  "total": 5
}
```

#### GET /proxies/{proxy_id}
Get proxy details.

**Response:**
```json
{
  "proxy_id": 1,
  "hostname": "proxy-01.example.com",
  "proxy_type": "egress",
  "status": "active",
  "cluster_id": 1,
  "version": "v0.1.0",
  "ip_address": "192.168.1.100",
  "registered_at": "2025-01-01T00:00:00Z",
  "last_heartbeat": "2025-01-16T12:30:00Z",
  "metrics": {
    "cpu_usage": 25.5,
    "memory_usage": 512
  }
}
```

#### GET /proxies/stats
Get aggregate proxy statistics.

**Response:**
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

#### GET /proxies/{proxy_id}/metrics
Get proxy Prometheus metrics.

**Response:**
```
# HELP proxy_cpu_usage CPU usage percentage
# TYPE proxy_cpu_usage gauge
proxy_cpu_usage{proxy_id="1"} 25.5

# HELP proxy_memory_usage Memory usage in MB
# TYPE proxy_memory_usage gauge
proxy_memory_usage{proxy_id="1"} 512
```

#### POST /proxies/cleanup
Remove stale proxies (Admin only).

**Request:**
```json
{
  "inactive_hours": 24
}
```

**Response:**
```json
{
  "cleaned_count": 2,
  "removed_proxies": [1, 3]
}
```

---

### mTLS Certificates

#### GET /mtls/certificates
List all certificates.

**Query Parameters:**
- `cluster_id` (optional)
- `type` (optional: ca, server, client)

**Response:**
```json
{
  "certificates": [
    {
      "certificate_id": 1,
      "cluster_id": 1,
      "type": "ca",
      "subject": "CN=MarchProxy-CA",
      "expires_at": "2026-01-16T00:00:00Z",
      "is_valid": true
    }
  ]
}
```

#### POST /mtls/certificates
Upload or create certificate.

**Request (Upload):**
```json
{
  "cluster_id": 1,
  "type": "server",
  "certificate_pem": "-----BEGIN CERTIFICATE-----...",
  "private_key_pem": "-----BEGIN PRIVATE KEY-----...",
  "ca_certificate": "-----BEGIN CERTIFICATE-----..."
}
```

**Response:**
```json
{
  "certificate_id": 2,
  "cluster_id": 1,
  "type": "server",
  "thumbprint": "abc123...",
  "expires_at": "2026-01-16T00:00:00Z"
}
```

#### POST /mtls/certificates/validate
Validate certificate chain.

**Request:**
```json
{
  "certificate_pem": "-----BEGIN CERTIFICATE-----...",
  "ca_certificate": "-----BEGIN CERTIFICATE-----..."
}
```

**Response:**
```json
{
  "is_valid": true,
  "subject": "CN=proxy.example.com",
  "issuer": "CN=MarchProxy-CA",
  "expires_at": "2026-01-16T00:00:00Z"
}
```

#### GET /mtls/config/{cluster_id}/{proxy_type}
Get mTLS configuration for proxy type.

**Response:**
```json
{
  "enabled": true,
  "require_client_cert": true,
  "ca_certificate_id": 1,
  "server_certificate_id": 2,
  "crl_enabled": false
}
```

#### PUT /mtls/config/{cluster_id}/{proxy_type}
Update mTLS configuration.

**Request:**
```json
{
  "enabled": true,
  "require_client_cert": true,
  "ca_certificate_id": 1,
  "server_certificate_id": 2
}
```

#### POST /mtls/ca/generate
Generate new CA certificate (Admin only).

**Request:**
```json
{
  "cluster_id": 1,
  "common_name": "MarchProxy-CA",
  "days_valid": 365
}
```

**Response:**
```json
{
  "certificate_id": 3,
  "certificate_pem": "-----BEGIN CERTIFICATE-----...",
  "private_key_pem": "-----BEGIN PRIVATE KEY-----..."
}
```

#### GET /mtls/certificates/{certificate_id}/download
Download certificate and key.

**Response:** Binary file download

#### POST /mtls/test/connection
Test mTLS connection to proxy.

**Request:**
```json
{
  "proxy_id": 1,
  "cluster_id": 1
}
```

**Response:**
```json
{
  "connection_successful": true,
  "certificate_valid": true,
  "server_certificate_expires_in_days": 365
}
```

---

## Health & Monitoring

#### GET /healthz
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-16T12:00:00Z",
  "database": "connected",
  "license": "enterprise"
}
```

#### GET /metrics
Prometheus metrics endpoint.

**Response:**
```
# HELP marchproxy_users_total Total number of users
# TYPE marchproxy_users_total gauge
marchproxy_users_total 5

# HELP marchproxy_clusters_total Total number of clusters
# TYPE marchproxy_clusters_total gauge
marchproxy_clusters_total 2
```

#### GET /license-status
License status information (no auth required).

**Response:**
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

## Error Responses

All endpoints return standardized error responses:

```json
{
  "error": "Unauthorized",
  "message": "Invalid or expired token",
  "status_code": 401
}
```

### Common Status Codes
- `200`: Success
- `201`: Created
- `400`: Bad Request
- `401`: Unauthorized
- `403`: Forbidden
- `404`: Not Found
- `409`: Conflict
- `500`: Internal Server Error
- `503`: Service Unavailable

---

## Rate Limiting

API endpoints are rate limited:
- Authentication endpoints: 10 requests/minute per IP
- Standard endpoints: 100 requests/minute per user
- Proxy registration: Unlimited (via API key)

Rate limit headers included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 99
X-RateLimit-Reset: 1642349400
```

---

## Pagination

List endpoints support pagination:

**Query Parameters:**
- `page`: Page number (default: 1)
- `per_page`: Items per page (default: 50, max: 500)

**Response:**
```json
{
  "items": [...],
  "total": 150,
  "page": 1,
  "per_page": 50,
  "pages": 3
}
```
