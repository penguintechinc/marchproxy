# MarchProxy API Server - REST API Reference

Complete REST API documentation for the MarchProxy API Server. All endpoints require authentication (JWT tokens) unless otherwise specified.

## Base URL

```
http://localhost:8000
```

## Authentication

All endpoints (except `/`, `/healthz`) require a Bearer token in the `Authorization` header:

```http
Authorization: Bearer <JWT_TOKEN>
```

### Obtaining a Token

**POST** `/api/v1/auth/login`

```json
{
  "email": "admin@example.com",
  "password": "securepassword",
  "totp_code": "123456"  // Optional if 2FA enabled
}
```

Response:
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "token_type": "bearer",
  "expires_in": 1800
}
```

## Health & Information Endpoints

### Get API Information

**GET** `/`

Returns basic API information.

**Response:**
```json
{
  "service": "MarchProxy API Server",
  "version": "1.0.0",
  "environment": "production",
  "features": ["xds", "multi-cloud", "zero-trust"]
}
```

### Health Check

**GET** `/healthz`

Returns service health status.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "service": "marchproxy-api-server"
}
```

### Metrics

**GET** `/metrics`

Returns Prometheus metrics in text format.

## Authentication Endpoints

Base path: `/api/v1/auth`

### User Registration

**POST** `/api/v1/auth/register`

Create a new user account. First user automatically becomes admin.

**Request:**
```json
{
  "email": "newuser@example.com",
  "username": "newuser",
  "password": "securepassword123",
  "full_name": "New User"
}
```

**Response:**
```json
{
  "id": "uuid",
  "email": "newuser@example.com",
  "username": "newuser",
  "is_admin": false,
  "is_active": true,
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Status Codes:**
- `201` - User created
- `400` - Invalid input or user already exists
- `409` - Email/username conflict

### Login

**POST** `/api/v1/auth/login`

Authenticate user and obtain JWT tokens.

**Request:**
```json
{
  "email": "admin@example.com",
  "password": "securepassword",
  "totp_code": "123456"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "token_type": "bearer",
  "expires_in": 1800,
  "user": {
    "id": "uuid",
    "email": "admin@example.com",
    "username": "admin",
    "is_admin": true
  }
}
```

**Status Codes:**
- `200` - Login successful
- `401` - Invalid credentials
- `403` - User not verified or account disabled

### Refresh Token

**POST** `/api/v1/auth/refresh`

Refresh an expired access token using the refresh token.

**Request:**
```json
{
  "refresh_token": "eyJhbGc..."
}
```

**Response:**
```json
{
  "access_token": "eyJhbGc...",
  "token_type": "bearer",
  "expires_in": 1800
}
```

### Current User

**GET** `/api/v1/auth/me`

Get current authenticated user information.

**Response:**
```json
{
  "id": "uuid",
  "email": "admin@example.com",
  "username": "admin",
  "full_name": "Administrator",
  "is_admin": true,
  "is_active": true,
  "is_verified": true,
  "totp_enabled": true,
  "last_login": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-14T09:00:00Z"
}
```

### Change Password

**POST** `/api/v1/auth/change-password`

Change the current user's password.

**Request:**
```json
{
  "current_password": "oldpassword",
  "new_password": "newpassword123"
}
```

**Response:**
```json
{
  "message": "Password changed successfully"
}
```

**Status Codes:**
- `200` - Password changed
- `400` - Invalid current password

### Enable 2FA

**POST** `/api/v1/auth/2fa/enable`

Generate a TOTP secret for two-factor authentication.

**Response:**
```json
{
  "totp_secret": "JBSWY3DPEBLW64TMMQ======",
  "qr_code_uri": "otpauth://totp/MarchProxy:admin@example.com?secret=...",
  "backup_codes": ["code1", "code2", "code3", ...]
}
```

### Verify 2FA

**POST** `/api/v1/auth/2fa/verify`

Verify and activate TOTP code.

**Request:**
```json
{
  "totp_code": "123456"
}
```

**Response:**
```json
{
  "message": "2FA enabled successfully",
  "backup_codes": ["code1", "code2", ...]
}
```

### Disable 2FA

**POST** `/api/v1/auth/2fa/disable`

Disable two-factor authentication.

**Request:**
```json
{
  "password": "userpassword"
}
```

**Response:**
```json
{
  "message": "2FA disabled successfully"
}
```

### Logout

**POST** `/api/v1/auth/logout`

Invalidate the current session.

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

## User Management Endpoints

Base path: `/api/v1/users`

### List Users (Admin Only)

**GET** `/api/v1/users`

List all users in the system.

**Query Parameters:**
- `skip` (int, default: 0) - Number of records to skip
- `limit` (int, default: 10) - Number of records to return
- `is_admin` (bool, optional) - Filter by admin status
- `is_active` (bool, optional) - Filter by active status

**Response:**
```json
{
  "total": 5,
  "items": [
    {
      "id": "uuid",
      "email": "user1@example.com",
      "username": "user1",
      "is_admin": true,
      "is_active": true,
      "is_verified": true,
      "created_at": "2024-01-14T09:00:00Z"
    }
  ]
}
```

### Get User (Admin Only)

**GET** `/api/v1/users/{user_id}`

Get specific user details.

**Response:**
```json
{
  "id": "uuid",
  "email": "user1@example.com",
  "username": "user1",
  "full_name": "User One",
  "is_admin": true,
  "is_active": true,
  "is_verified": true,
  "totp_enabled": true,
  "last_login": "2024-01-15T10:30:00Z",
  "created_at": "2024-01-14T09:00:00Z"
}
```

### Update User (Admin Only)

**PUT** `/api/v1/users/{user_id}`

Update user information.

**Request:**
```json
{
  "full_name": "New Name",
  "is_active": true,
  "is_admin": false
}
```

**Response:**
```json
{
  "id": "uuid",
  "email": "user1@example.com",
  "username": "user1",
  "full_name": "New Name",
  "is_admin": false,
  "is_active": true,
  "updated_at": "2024-01-15T11:00:00Z"
}
```

### Delete User (Admin Only)

**DELETE** `/api/v1/users/{user_id}`

Delete a user account.

**Response:**
```json
{
  "message": "User deleted successfully"
}
```

## Cluster Management Endpoints

Base path: `/api/v1/clusters`

### Create Cluster

**POST** `/api/v1/clusters`

Create a new cluster (Enterprise feature).

**Request:**
```json
{
  "name": "cluster-1",
  "description": "Production cluster",
  "max_proxies": 10,
  "syslog_server": "syslog.example.com:514",
  "enable_auth_logs": true,
  "enable_netflow_logs": true,
  "enable_debug_logs": false
}
```

**Response:**
```json
{
  "id": "uuid",
  "name": "cluster-1",
  "description": "Production cluster",
  "api_key_hash": "sha256:...",
  "api_key": "CLUSTER_KEY_...",  // Only shown on creation
  "max_proxies": 10,
  "proxy_count": 0,
  "syslog_server": "syslog.example.com:514",
  "enable_auth_logs": true,
  "enable_netflow_logs": true,
  "enable_debug_logs": false,
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Status Codes:**
- `201` - Cluster created
- `400` - Invalid input
- `403` - Unauthorized (not admin or enterprise license)

### List Clusters

**GET** `/api/v1/clusters`

List all clusters accessible to the user.

**Query Parameters:**
- `skip` (int, default: 0) - Pagination offset
- `limit` (int, default: 10) - Items per page

**Response:**
```json
{
  "total": 2,
  "items": [
    {
      "id": "uuid",
      "name": "cluster-1",
      "description": "Production cluster",
      "proxy_count": 5,
      "max_proxies": 10,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Get Cluster

**GET** `/api/v1/clusters/{cluster_id}`

Get cluster details.

**Response:**
```json
{
  "id": "uuid",
  "name": "cluster-1",
  "description": "Production cluster",
  "api_key_hash": "sha256:...",
  "max_proxies": 10,
  "proxy_count": 5,
  "syslog_server": "syslog.example.com:514",
  "enable_auth_logs": true,
  "enable_netflow_logs": true,
  "enable_debug_logs": false,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T11:00:00Z"
}
```

### Update Cluster

**PUT** `/api/v1/clusters/{cluster_id}`

Update cluster configuration.

**Request:**
```json
{
  "description": "Updated description",
  "max_proxies": 20,
  "syslog_server": "newsyslog.example.com:514",
  "enable_debug_logs": true
}
```

**Response:** Updated cluster object

### Delete Cluster

**DELETE** `/api/v1/clusters/{cluster_id}`

Delete a cluster (requires no active proxies).

**Response:**
```json
{
  "message": "Cluster deleted successfully"
}
```

**Status Codes:**
- `204` - Cluster deleted
- `409` - Cluster has active proxies

### Rotate Cluster API Key

**POST** `/api/v1/clusters/{cluster_id}/rotate-key`

Generate a new API key for the cluster.

**Response:**
```json
{
  "api_key": "CLUSTER_KEY_...",
  "message": "API key rotated successfully"
}
```

## Service Management Endpoints

Base path: `/api/v1/services`

### Create Service

**POST** `/api/v1/services`

Create a new service within a cluster.

**Request:**
```json
{
  "cluster_id": "uuid",
  "name": "api-service",
  "description": "Internal API",
  "destination_ip": "10.0.1.100",
  "destination_port": 443,
  "protocol": "https",
  "auth_type": "jwt",
  "enable_health_check": true,
  "health_check_interval": 30,
  "health_check_path": "/health"
}
```

**Response:**
```json
{
  "id": "uuid",
  "cluster_id": "uuid",
  "name": "api-service",
  "description": "Internal API",
  "destination_ip": "10.0.1.100",
  "destination_port": 443,
  "protocol": "https",
  "auth_type": "jwt",
  "service_token": "TOKEN_...",
  "enable_health_check": true,
  "health_check_interval": 30,
  "health_check_path": "/health",
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Status Codes:**
- `201` - Service created
- `400` - Invalid input
- `404` - Cluster not found

### List Services

**GET** `/api/v1/services`

List all services.

**Query Parameters:**
- `cluster_id` (uuid, optional) - Filter by cluster
- `protocol` (string, optional) - Filter by protocol
- `skip` (int, default: 0) - Pagination offset
- `limit` (int, default: 10) - Items per page

**Response:**
```json
{
  "total": 3,
  "items": [
    {
      "id": "uuid",
      "cluster_id": "uuid",
      "name": "api-service",
      "description": "Internal API",
      "destination_ip": "10.0.1.100",
      "destination_port": 443,
      "protocol": "https",
      "auth_type": "jwt",
      "health_check_status": "healthy",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Get Service

**GET** `/api/v1/services/{service_id}`

Get service details.

**Response:**
```json
{
  "id": "uuid",
  "cluster_id": "uuid",
  "name": "api-service",
  "description": "Internal API",
  "destination_ip": "10.0.1.100",
  "destination_port": 443,
  "protocol": "https",
  "auth_type": "jwt",
  "enable_health_check": true,
  "health_check_interval": 30,
  "health_check_path": "/health",
  "health_check_status": "healthy",
  "last_health_check": "2024-01-15T11:00:00Z",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Update Service

**PUT** `/api/v1/services/{service_id}`

Update service configuration.

**Request:**
```json
{
  "description": "Updated description",
  "destination_port": 8443,
  "enable_health_check": false
}
```

**Response:** Updated service object

### Delete Service

**DELETE** `/api/v1/services/{service_id}`

Delete a service.

**Response:**
```json
{
  "message": "Service deleted successfully"
}
```

### Rotate Service Token

**POST** `/api/v1/services/{service_id}/rotate-token`

Generate a new authentication token for the service.

**Response:**
```json
{
  "service_token": "TOKEN_...",
  "message": "Service token rotated successfully"
}
```

### Reload xDS Configuration

**POST** `/api/v1/services/{service_id}/reload-xds`

Force a configuration reload to Envoy proxies.

**Response:**
```json
{
  "message": "xDS configuration reloaded successfully",
  "proxies_updated": 5
}
```

## Proxy Management Endpoints

Base path: `/api/v1/proxies`

### List Proxies

**GET** `/api/v1/proxies`

List all proxies in the system.

**Query Parameters:**
- `cluster_id` (uuid, optional) - Filter by cluster
- `status` (string, optional) - Filter by status (active, inactive, error)
- `skip` (int, default: 0) - Pagination offset
- `limit` (int, default: 10) - Items per page

**Response:**
```json
{
  "total": 5,
  "items": [
    {
      "id": "uuid",
      "cluster_id": "uuid",
      "name": "proxy-1",
      "status": "active",
      "ip_address": "10.0.2.1",
      "version": "1.0.0",
      "capabilities": ["ebpf", "hardware_acceleration"],
      "connections": 1250,
      "throughput_mbps": 850,
      "cpu_percent": 45.2,
      "memory_percent": 62.1,
      "last_heartbeat": "2024-01-15T11:00:00Z"
    }
  ]
}
```

### Get Proxy

**GET** `/api/v1/proxies/{proxy_id}`

Get proxy details.

**Response:**
```json
{
  "id": "uuid",
  "cluster_id": "uuid",
  "name": "proxy-1",
  "status": "active",
  "ip_address": "10.0.2.1",
  "version": "1.0.0",
  "capabilities": ["ebpf", "xdp", "hardware_acceleration"],
  "config_version": "v1.2.3",
  "connections": 1250,
  "throughput_mbps": 850,
  "cpu_percent": 45.2,
  "memory_percent": 62.1,
  "latency_avg_ms": 12.5,
  "latency_p95_ms": 28.3,
  "error_rate": 0.05,
  "last_heartbeat": "2024-01-15T11:00:00Z",
  "registered_at": "2024-01-10T10:00:00Z"
}
```

## Certificate Management Endpoints

Base path: `/api/v1/certificates`

### Create Certificate

**POST** `/api/v1/certificates`

Create or import a TLS certificate.

**Request:**
```json
{
  "cluster_id": "uuid",
  "service_id": "uuid",
  "name": "api-cert",
  "source": "upload",
  "certificate": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "private_key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
  "chain": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
  "auto_renewal": true
}
```

**Response:**
```json
{
  "id": "uuid",
  "cluster_id": "uuid",
  "name": "api-cert",
  "source": "upload",
  "common_name": "api.example.com",
  "san": ["api.example.com", "*.api.example.com"],
  "issuer": "Let's Encrypt",
  "issued_at": "2023-01-15T00:00:00Z",
  "expires_at": "2024-01-15T00:00:00Z",
  "auto_renewal": true,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### List Certificates

**GET** `/api/v1/certificates`

List all certificates.

**Query Parameters:**
- `cluster_id` (uuid, optional) - Filter by cluster
- `status` (string, optional) - Filter by status (valid, expiring, expired)
- `skip` (int, default: 0) - Pagination offset
- `limit` (int, default: 10) - Items per page

**Response:**
```json
{
  "total": 3,
  "items": [
    {
      "id": "uuid",
      "cluster_id": "uuid",
      "name": "api-cert",
      "common_name": "api.example.com",
      "status": "valid",
      "expires_at": "2024-01-15T00:00:00Z",
      "auto_renewal": true
    }
  ]
}
```

### Get Certificate

**GET** `/api/v1/certificates/{certificate_id}`

Get certificate details.

**Response:**
```json
{
  "id": "uuid",
  "cluster_id": "uuid",
  "name": "api-cert",
  "source": "upload",
  "common_name": "api.example.com",
  "san": ["api.example.com", "*.api.example.com"],
  "issuer": "Let's Encrypt",
  "issued_at": "2023-01-15T00:00:00Z",
  "expires_at": "2024-01-15T00:00:00Z",
  "auto_renewal": true,
  "renewal_count": 2,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Update Certificate

**PUT** `/api/v1/certificates/{certificate_id}`

Update certificate metadata.

**Request:**
```json
{
  "auto_renewal": false
}
```

**Response:** Updated certificate object

### Delete Certificate

**DELETE** `/api/v1/certificates/{certificate_id}`

Delete a certificate.

**Response:**
```json
{
  "message": "Certificate deleted successfully"
}
```

## Traffic Shaping Endpoints (Enterprise)

Base path: `/api/v1/traffic-shaping`

### Create Traffic Shape

**POST** `/api/v1/traffic-shaping`

Configure rate limiting and traffic shaping.

**Request:**
```json
{
  "service_id": "uuid",
  "name": "peak-hours-limit",
  "enabled": true,
  "bandwidth_limit_mbps": 100,
  "connection_limit": 1000,
  "requests_per_second": 5000
}
```

**Response:**
```json
{
  "id": "uuid",
  "service_id": "uuid",
  "name": "peak-hours-limit",
  "enabled": true,
  "bandwidth_limit_mbps": 100,
  "connection_limit": 1000,
  "requests_per_second": 5000,
  "created_at": "2024-01-15T10:30:00Z"
}
```

### List Traffic Shapes

**GET** `/api/v1/traffic-shaping`

**Query Parameters:**
- `service_id` (uuid, optional) - Filter by service
- `skip` (int, default: 0) - Pagination offset
- `limit` (int, default: 10) - Items per page

**Response:**
```json
{
  "total": 2,
  "items": [...]
}
```

### Update Traffic Shape

**PUT** `/api/v1/traffic-shaping/{shape_id}`

Update traffic shaping rules.

**Request:**
```json
{
  "bandwidth_limit_mbps": 150,
  "enabled": true
}
```

**Response:** Updated traffic shape object

### Delete Traffic Shape

**DELETE** `/api/v1/traffic-shaping/{shape_id}`

Delete traffic shaping rules.

## Error Handling

All endpoints return consistent error responses:

```json
{
  "detail": "Error message describing the issue",
  "error_code": "ERROR_CODE",
  "timestamp": "2024-01-15T11:00:00Z"
}
```

**Common Status Codes:**
- `200` - Success
- `201` - Created
- `204` - No content
- `400` - Bad request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not found
- `409` - Conflict
- `422` - Validation error
- `500` - Server error

## Rate Limiting

API rate limits (per minute):
- Authentication endpoints: 10 requests/minute
- Standard endpoints: 100 requests/minute
- Admin endpoints: 50 requests/minute

Rate limit headers:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`

## Pagination

List endpoints support standard pagination:

```json
{
  "total": 100,
  "skip": 0,
  "limit": 10,
  "items": [...]
}
```

## API Documentation

Interactive API documentation available at:
- **Swagger UI**: `http://localhost:8000/api/docs`
- **ReDoc**: `http://localhost:8000/api/redoc`
- **OpenAPI JSON**: `http://localhost:8000/api/openapi.json`
