# MarchProxy Manager API Documentation

**Version:** 1.0.0
**API Base URL:** `http://localhost:8000/api`
**OpenAPI Specification:** Available at `/openapi.json` or `/docs` (SwaggerUI)
**ReDoc Documentation:** Available at `/redoc`

## Table of Contents

- [Overview](#overview)
- [OpenAPI Specification](#openapi-specification)
- [Authentication](#authentication)
- [API Versioning](#api-versioning)
- [License Tiers & Feature Gating](#license-tiers--feature-gating)
- [Common Patterns](#common-patterns)
- [API Endpoints](#api-endpoints)
  - [Authentication](#authentication-endpoints)
  - [Users](#user-management)
  - [Services](#service-management)
  - [Mappings](#mapping-management)
  - [Clusters](#cluster-management-enterprise)
  - [Certificates](#certificate-management)
  - [Proxies](#proxy-management)
  - [License](#license-management)
  - [Monitoring](#monitoring-endpoints)
  - [xDS Control Plane](#xds-control-plane)
  - [Traffic Shaping (Enterprise)](#traffic-shaping-enterprise)
  - [Multi-Cloud Routing (Enterprise)](#multi-cloud-routing-enterprise)
  - [Observability (Enterprise)](#observability-enterprise)
  - [Zero-Trust (Enterprise)](#zero-trust-enterprise)
- [Request/Response Examples](#requestresponse-examples)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Webhooks](#webhooks)
- [Migration Notes](#migration-notes)

## Overview

The MarchProxy Manager API is a RESTful API for managing the complete dual proxy infrastructure (ingress and egress). It provides:

- **User Authentication**: Local, SAML, OAuth2, and SCIM integration
- **Service Configuration**: Backend service definitions with authentication
- **Traffic Mapping**: Granular routing rules with protocol and port control
- **Cluster Management**: Multi-cluster isolation (Enterprise)
- **Certificate Management**: TLS/mTLS certificate lifecycle with automated CA generation
- **Proxy Registration**: Automatic proxy discovery and configuration distribution
- **License Enforcement**: Community (3 proxies) vs Enterprise (unlimited) validation
- **Real-time Monitoring**: Health checks and Prometheus metrics
- **xDS Control Plane**: Dynamic Envoy proxy configuration
- **Enterprise Features**: Traffic shaping, multi-cloud routing, observability, zero-trust

## OpenAPI Specification

The complete API is documented in OpenAPI 3.0 format. Access the interactive documentation:

- **SwaggerUI:** `http://localhost:8000/docs`
- **ReDoc:** `http://localhost:8000/redoc`
- **OpenAPI JSON:** `http://localhost:8000/openapi.json`
- **OpenAPI YAML:** `http://localhost:8000/openapi.yaml`

### Example: Downloading OpenAPI Spec

```bash
# Download OpenAPI JSON
curl http://localhost:8000/openapi.json > marchproxy-openapi.json

# Generate client SDK (using openapi-generator)
docker run --rm -v ${PWD}:/local \
  openapitools/openapi-generator-cli generate \
  -i /local/marchproxy-openapi.json \
  -g python \
  -o /local/marchproxy-client
```

## API Versioning

The API uses URL-based versioning: `/api/v1/`, `/api/v2/`, etc.

- **Current Version:** v1
- **Backward Compatibility:** v0 (deprecated, removed in v2.0.0)
- **Deprecation Policy:** Old versions supported for 6 months after new major version

**Example:**
```bash
# v1 endpoints
GET /api/v1/services
POST /api/v1/services

# API version in response headers
X-API-Version: 1.0.0
```

## Authentication

The API supports three authentication methods:

### 1. JWT Authentication (Web Interface)

Used for browser-based sessions with automatic token refresh.

```bash
# Login
curl -X POST http://localhost:8000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "changeme",
    "totp_code": "123456"
  }'

# Response
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "user": {
      "id": 1,
      "username": "admin",
      "is_admin": true
    }
  }
}

# Use token in subsequent requests
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  http://localhost:8000/api/services
```

### 2. API Key Authentication (Automation)

For programmatic access and CI/CD integrations.

```bash
# Create API key
curl -X POST http://localhost:8000/api/auth/api-keys \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-cd-automation",
    "permissions": ["read", "write"]
  }'

# Use API key
curl -H "Authorization: Bearer mp_api_<key_value>" \
  http://localhost:8000/api/services
```

### 3. Cluster API Key (Proxy Operations)

For proxy registration and configuration retrieval.

```bash
# Proxies use cluster-specific API keys
curl -H "X-Cluster-API-Key: <cluster_api_key>" \
  http://localhost:8000/api/config/1
```

## License Tiers & Feature Gating

### Community Edition (Open Source)
- **Max Proxies**: 3 total (any combination of ingress/egress)
- **Clusters**: Single default cluster only
- **Authentication**: Basic (username/password), 2FA/TOTP, JWT
- **Certificates**: Manual upload
- **Acceleration**: eBPF only

### Enterprise Edition
- **Max Proxies**: Unlimited (based on license)
- **Clusters**: Multiple with isolation
- **Authentication**: + SAML, SCIM, OAuth2
- **Certificates**: + Automated CA generation, wildcard certificates
- **Acceleration**: + XDP, AF_XDP, DPDK, SR-IOV

### Feature-Gated Endpoints

Enterprise features return `402 Payment Required` when accessed without valid license:

- `POST /api/clusters` - Create additional clusters
- `POST /api/certificates/generate-wildcard` - Generate wildcard certificates
- SAML/OAuth2 authentication endpoints

## Common Patterns

### Standard Response Format

All API responses follow this structure:

```json
{
  "success": true,
  "data": { /* response payload */ },
  "message": "Operation completed successfully",
  "timestamp": "2025-12-12T10:30:00Z",
  "request_id": "req_abc123"
}
```

### Error Response Format

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input parameters",
    "details": {
      "field": "ip_fqdn",
      "issue": "Invalid IP address format"
    }
  },
  "timestamp": "2025-12-12T10:30:00Z",
  "request_id": "req_abc123"
}
```

### Pagination

List endpoints support pagination:

```bash
GET /api/users?page=1&limit=20&search=john
```

## API Endpoints

### Authentication Endpoints

#### POST /api/auth/login
Authenticate user with username, password, and optional 2FA.

**Authentication:** None (public endpoint)

**Request:**
```json
{
  "username": "admin",
  "password": "secure_password",
  "totp_code": "123456"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 3600,
    "user": {
      "id": 1,
      "username": "admin",
      "email": "admin@company.com",
      "is_admin": true,
      "clusters": [1]
    }
  }
}
```

#### POST /api/auth/refresh
Refresh access token using refresh token.

#### POST /api/auth/logout
Invalidate current session.

#### GET /api/auth/api-keys
List API keys for current user.

#### POST /api/auth/api-keys
Create new API key for programmatic access.

#### DELETE /api/auth/api-keys/{keyId}
Revoke and delete API key.

### User Management

#### GET /api/users
List users (admin only, paginated).

**Query Parameters:**
- `page` (integer): Page number (default: 1)
- `limit` (integer): Items per page (default: 20, max: 100)
- `search` (string): Search by username/email
- `cluster_id` (integer): Filter by cluster assignment

#### POST /api/users
Create new user account (admin only).

**Request:**
```json
{
  "username": "jdoe",
  "email": "jdoe@company.com",
  "password": "StrongPassword123!",
  "is_admin": false,
  "clusters": [1, 2]
}
```

#### GET /api/users/{userId}
Get user details by ID.

#### PUT /api/users/{userId}
Update user account (admin or self).

#### DELETE /api/users/{userId}
Delete user account (admin only).

### Service Management

#### GET /api/services
List backend services.

**Query Parameters:**
- `cluster_id` (integer): Filter by cluster
- `collection` (string): Filter by service collection
- `search` (string): Search by name

#### POST /api/services
Create new backend service.

**Request:**
```json
{
  "name": "web-backend",
  "ip_fqdn": "10.0.1.100",
  "collection": "web-services",
  "cluster_id": 1,
  "auth_type": "jwt",
  "auth_config": {
    "jwt_expiry": 3600
  }
}
```

**Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "id": 42,
    "name": "web-backend",
    "ip_fqdn": "10.0.1.100",
    "collection": "web-services",
    "cluster_id": 1,
    "auth_type": "jwt",
    "created_at": "2025-12-12T10:30:00Z",
    "updated_at": "2025-12-12T10:30:00Z"
  }
}
```

#### GET /api/services/{serviceId}
Get service details.

#### PUT /api/services/{serviceId}
Update service configuration.

#### DELETE /api/services/{serviceId}
Delete service and associated mappings.

#### POST /api/services/{serviceId}/rotate-jwt
Rotate JWT secret for service with zero-downtime.

**Response:**
```json
{
  "success": true,
  "data": {
    "new_secret": "new_jwt_secret_value",
    "old_secret_valid_until": "2025-12-12T11:30:00Z"
  }
}
```

### Mapping Management

#### GET /api/mappings
List traffic routing mappings.

#### POST /api/mappings
Create new mapping.

**Request:**
```json
{
  "source_services": ["web-frontend"],
  "dest_services": ["web-backend", "api-backend"],
  "cluster_id": 1,
  "protocols": ["tcp", "http", "https"],
  "ports": [80, 443, 8080],
  "auth_required": true,
  "comments": "Web tier to backend services"
}
```

#### GET /api/mappings/{mappingId}
Get mapping details.

#### PUT /api/mappings/{mappingId}
Update mapping configuration.

#### DELETE /api/mappings/{mappingId}
Delete mapping.

### Cluster Management (Enterprise)

#### GET /api/clusters
List all clusters.

**Enterprise Feature:** Returns `402` if Community edition.

#### POST /api/clusters
Create new cluster.

**Request:**
```json
{
  "name": "production-us-east",
  "description": "Production cluster in US East region",
  "syslog_endpoint": "syslog.company.com:514",
  "log_auth": true,
  "log_netflow": false,
  "log_debug": false
}
```

#### GET /api/clusters/{clusterId}
Get cluster details.

#### PUT /api/clusters/{clusterId}
Update cluster configuration.

#### DELETE /api/clusters/{clusterId}
Deactivate cluster.

#### POST /api/clusters/{clusterId}/rotate-key
Rotate cluster API key with zero-downtime.

**Response:**
```json
{
  "success": true,
  "data": {
    "new_api_key": "mp_cluster_<new_key>",
    "old_key_valid_until": "2025-12-12T11:30:00Z"
  }
}
```

#### PUT /api/clusters/{clusterId}/logging
Update cluster logging configuration.

### Certificate Management

#### GET /api/certificates
List TLS certificates.

#### POST /api/certificates
Upload certificate and private key.

**Request:** `multipart/form-data`
- `name`: Certificate name
- `cert_file`: Certificate file (PEM format)
- `key_file`: Private key file (PEM format)
- `ca_file`: CA certificate (optional)
- `auto_renew`: Enable auto-renewal

#### POST /api/certificates/generate-wildcard
Generate wildcard certificate (Enterprise).

**Request:**
```json
{
  "domain": "company.com",
  "validity_years": 10
}
```

**Response:** `201 Created`
```json
{
  "success": true,
  "data": {
    "id": 10,
    "name": "*.company.com",
    "common_name": "*.company.com",
    "source_type": "auto",
    "auto_renew": true,
    "expires_at": "2035-12-12T10:30:00Z",
    "status": "active"
  }
}
```

#### DELETE /api/certificates/{certificateId}
Delete certificate.

### Proxy Management

#### POST /api/proxy/register
Register new proxy instance.

**Authentication:** `X-Cluster-API-Key`

**Request:**
```json
{
  "hostname": "proxy-egress-01.company.com",
  "capabilities": ["ebpf", "xdp", "afxdp"],
  "version": "v1.0.0"
}
```

#### GET /api/config/{clusterId}
Get proxy configuration for cluster.

**Authentication:** `X-Cluster-API-Key`

**Response:**
```json
{
  "success": true,
  "data": {
    "cluster": {
      "id": 1,
      "name": "default",
      "syslog_endpoint": "syslog.company.com:514",
      "log_auth": true,
      "log_netflow": false,
      "log_debug": false
    },
    "services": [ /* array of services */ ],
    "mappings": [ /* array of mappings */ ],
    "certificates": [ /* array of certificates */ ]
  }
}
```

#### POST /api/proxy/heartbeat
Send proxy status and metrics.

**Authentication:** `X-Cluster-API-Key`

#### GET /api/proxies
List registered proxy instances.

**Query Parameters:**
- `cluster_id`: Filter by cluster
- `status`: Filter by status (healthy, unhealthy, offline)

### License Management

#### GET /api/license-status
Get current license status and limits.

**Response:**
```json
{
  "success": true,
  "data": {
    "license_type": "community",
    "valid": true,
    "features": {
      "unlimited_proxies": false,
      "multi_cluster": false,
      "saml_authentication": false,
      "oauth2_authentication": false
    },
    "limits": {
      "max_proxies": 3,
      "max_clusters": 1,
      "current_proxies": 2,
      "current_clusters": 1
    }
  }
}
```

#### POST /api/license/validate
Validate license key with license server.

**Request:**
```json
{
  "license_key": "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"
}
```

### Monitoring Endpoints

#### GET /api/healthz
Health check endpoint (no authentication required).

**Response:** `200 OK` (healthy) or `503 Service Unavailable` (unhealthy)
```json
{
  "status": "healthy",
  "timestamp": "2025-12-12T10:30:00Z",
  "checks": {
    "database": "healthy",
    "license_server": "healthy",
    "certificate_expiry": "healthy"
  },
  "version": "v1.0.0",
  "uptime": 86400
}
```

#### GET /api/metrics
Get Prometheus-formatted metrics (no authentication required).

**Response:** `200 OK` (text/plain)
```
# HELP marchproxy_users_total Total number of users
# TYPE marchproxy_users_total gauge
marchproxy_users_total 45

# HELP marchproxy_proxies_registered Total registered proxies
# TYPE marchproxy_proxies_registered gauge
marchproxy_proxies_registered{cluster="default",type="egress"} 1
marchproxy_proxies_registered{cluster="default",type="ingress"} 1

# HELP marchproxy_api_requests_total Total API requests
# TYPE marchproxy_api_requests_total counter
marchproxy_api_requests_total{method="GET",endpoint="/api/services",status="200"} 1234
```

## Error Handling

### Common Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `VALIDATION_ERROR` | Invalid input parameters |
| 401 | `AUTHENTICATION_REQUIRED` | Missing or invalid authentication |
| 403 | `INSUFFICIENT_PERMISSIONS` | User lacks required permissions |
| 404 | `RESOURCE_NOT_FOUND` | Requested resource not found |
| 409 | `RESOURCE_CONFLICT` | Resource already exists |
| 402 | `FEATURE_NOT_AVAILABLE` | Enterprise feature requires license |
| 429 | `RATE_LIMIT_EXCEEDED` | Rate limit exceeded |
| 500 | `INTERNAL_SERVER_ERROR` | Internal server error |
| 503 | `SERVICE_UNAVAILABLE` | Service temporarily unavailable |

### Error Response Example

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid service configuration",
    "details": {
      "field": "ip_fqdn",
      "issue": "Must be valid IP address or FQDN",
      "provided": "invalid..value"
    }
  },
  "timestamp": "2025-12-12T10:30:00Z",
  "request_id": "req_abc123xyz"
}
```

## Rate Limiting

API endpoints are rate limited to prevent abuse:

| Endpoint Category | Rate Limit |
|------------------|------------|
| Authentication | 5 requests/minute per IP |
| General API | 1000 requests/hour per API key |
| Configuration | 100 requests/minute per cluster |
| Monitoring | Unlimited |

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 742
X-RateLimit-Reset: 1702387200
```

## Complete OpenAPI Specification

For the complete API specification including all schemas, parameters, and response formats, see [api/openapi.yaml](api/openapi.yaml).

The OpenAPI spec can be used to:
- Generate client libraries in any language
- Import into Postman or Insomnia
- Generate API documentation with Swagger UI
- Validate requests and responses

---

**Need Help?**
- GitHub Issues: https://github.com/marchproxy/marchproxy/issues
- Documentation: https://github.com/marchproxy/marchproxy/tree/main/docs
- Enterprise Support: support@marchproxy.io
