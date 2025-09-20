# Manager API Reference

This document provides comprehensive documentation for the MarchProxy Manager REST API.

## Overview

The MarchProxy Manager exposes a RESTful API for managing all aspects of the proxy system:

- **Authentication**: User management and API key operations
- **Services**: Backend service configuration
- **Mappings**: Traffic routing rules
- **Clusters**: Multi-cluster management (Enterprise)
- **Certificates**: TLS certificate management
- **Monitoring**: Health checks and metrics

## Base URL

```
http://localhost:8000/api
https://manager.company.com/api
```

## Authentication

### API Key Authentication

Include the API key in the request header:

```http
Authorization: Bearer your-api-key-here
```

### Cluster API Key Authentication

For proxy registration and configuration:

```http
X-Cluster-API-Key: your-cluster-api-key
```

### JWT Authentication

For web interface sessions:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## Common Response Format

All API responses follow this structure:

```json
{
  "success": true,
  "data": { /* response data */ },
  "message": "Operation completed successfully",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req_123456789"
}
```

Error responses:

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input parameters",
    "details": {
      "field": "ip_fqdn",
      "issue": "Invalid IP address or FQDN format"
    }
  },
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req_123456789"
}
```

## Authentication Endpoints

### Login

```http
POST /api/auth/login
```

**Request Body:**

```json
{
  "username": "admin",
  "password": "secure_password",
  "totp_code": "123456"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 86400,
    "user": {
      "id": 1,
      "username": "admin",
      "is_admin": true,
      "clusters": [1, 2, 3]
    }
  }
}
```

### Refresh Token

```http
POST /api/auth/refresh
```

**Request Body:**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

### Logout

```http
POST /api/auth/logout
```

**Headers:**

```http
Authorization: Bearer access_token
```

### Generate API Key

```http
POST /api/auth/api-keys
```

**Request Body:**

```json
{
  "name": "Production API Key",
  "expires_at": "2025-01-15T00:00:00Z",
  "permissions": ["read", "write"]
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": "key_123456789",
    "name": "Production API Key",
    "key": "mp_live_1234567890abcdef...",
    "expires_at": "2025-01-15T00:00:00Z",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

## User Management

### List Users

```http
GET /api/users
```

**Query Parameters:**

- `page` (integer): Page number (default: 1)
- `limit` (integer): Items per page (default: 20, max: 100)
- `search` (string): Search term for username or email
- `cluster_id` (integer): Filter by cluster assignment

**Response:**

```json
{
  "success": true,
  "data": {
    "users": [
      {
        "id": 1,
        "username": "admin",
        "email": "admin@company.com",
        "is_admin": true,
        "auth_provider": "local",
        "created_at": "2024-01-01T00:00:00Z",
        "last_login": "2024-01-15T10:30:00Z",
        "clusters": [1, 2, 3]
      }
    ],
    "total": 1,
    "page": 1,
    "limit": 20
  }
}
```

### Create User

```http
POST /api/users
```

**Request Body:**

```json
{
  "username": "newuser",
  "email": "newuser@company.com",
  "password": "secure_password",
  "is_admin": false,
  "clusters": [1, 2]
}
```

### Update User

```http
PUT /api/users/{user_id}
```

**Request Body:**

```json
{
  "email": "updated@company.com",
  "is_admin": true,
  "clusters": [1, 2, 3]
}
```

### Delete User

```http
DELETE /api/users/{user_id}
```

## Service Management

### List Services

```http
GET /api/services
```

**Query Parameters:**

- `cluster_id` (integer): Filter by cluster
- `collection` (string): Filter by service collection
- `search` (string): Search term for service name or IP/FQDN

**Response:**

```json
{
  "success": true,
  "data": {
    "services": [
      {
        "id": 1,
        "name": "web-backend",
        "ip_fqdn": "backend.internal.com",
        "collection": "web-services",
        "cluster_id": 1,
        "auth_type": "jwt",
        "auth_config": {
          "jwt_secret": "secret_key_here",
          "jwt_expiry": 3600
        },
        "created_at": "2024-01-15T10:00:00Z",
        "updated_at": "2024-01-15T10:30:00Z"
      }
    ],
    "total": 1
  }
}
```

### Create Service

```http
POST /api/services
```

**Request Body:**

```json
{
  "name": "api-service",
  "ip_fqdn": "api.internal.com",
  "collection": "api-services",
  "cluster_id": 1,
  "auth_type": "jwt",
  "auth_config": {
    "jwt_expiry": 3600
  }
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "id": 2,
    "name": "api-service",
    "ip_fqdn": "api.internal.com",
    "collection": "api-services",
    "cluster_id": 1,
    "auth_type": "jwt",
    "auth_config": {
      "jwt_secret": "generated_secret_key",
      "jwt_expiry": 3600
    },
    "created_at": "2024-01-15T11:00:00Z"
  }
}
```

### Update Service

```http
PUT /api/services/{service_id}
```

**Request Body:**

```json
{
  "ip_fqdn": "api-v2.internal.com",
  "auth_type": "api_key"
}
```

### Delete Service

```http
DELETE /api/services/{service_id}
```

### Rotate Service JWT

```http
POST /api/services/{service_id}/rotate-jwt
```

**Response:**

```json
{
  "success": true,
  "data": {
    "new_secret": "new_generated_secret",
    "old_secret_valid_until": "2024-01-15T12:00:00Z"
  }
}
```

## Mapping Management

### List Mappings

```http
GET /api/mappings
```

**Query Parameters:**

- `cluster_id` (integer): Filter by cluster
- `source_service` (string): Filter by source service
- `dest_service` (string): Filter by destination service

**Response:**

```json
{
  "success": true,
  "data": {
    "mappings": [
      {
        "id": 1,
        "source_services": ["web-frontend"],
        "dest_services": ["web-backend", "api-service"],
        "cluster_id": 1,
        "protocols": ["tcp", "http"],
        "ports": [80, 443],
        "auth_required": true,
        "comments": "Frontend to backend mapping",
        "created_at": "2024-01-15T10:00:00Z"
      }
    ],
    "total": 1
  }
}
```

### Create Mapping

```http
POST /api/mappings
```

**Request Body:**

```json
{
  "source_services": ["web-frontend"],
  "dest_services": ["web-backend"],
  "cluster_id": 1,
  "protocols": ["tcp", "http"],
  "ports": [80, 443],
  "auth_required": true,
  "comments": "Frontend to backend mapping"
}
```

### Update Mapping

```http
PUT /api/mappings/{mapping_id}
```

### Delete Mapping

```http
DELETE /api/mappings/{mapping_id}
```

## Cluster Management (Enterprise)

### List Clusters

```http
GET /api/clusters
```

**Response:**

```json
{
  "success": true,
  "data": {
    "clusters": [
      {
        "id": 1,
        "name": "production",
        "description": "Production cluster",
        "api_key": "mp_cluster_1234567890...",
        "syslog_endpoint": "syslog.company.com:514",
        "log_auth": true,
        "log_netflow": true,
        "log_debug": false,
        "is_active": true,
        "proxy_count": 5,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1
  }
}
```

### Create Cluster

```http
POST /api/clusters
```

**Request Body:**

```json
{
  "name": "staging",
  "description": "Staging environment cluster",
  "syslog_endpoint": "syslog-staging.company.com:514",
  "log_auth": true,
  "log_netflow": false,
  "log_debug": true
}
```

### Update Cluster

```http
PUT /api/clusters/{cluster_id}
```

### Rotate Cluster API Key

```http
POST /api/clusters/{cluster_id}/rotate-key
```

**Response:**

```json
{
  "success": true,
  "data": {
    "new_api_key": "mp_cluster_new_key_here",
    "old_key_valid_until": "2024-01-15T12:00:00Z"
  }
}
```

### Update Cluster Logging

```http
PUT /api/clusters/{cluster_id}/logging
```

**Request Body:**

```json
{
  "syslog_endpoint": "new-syslog.company.com:514",
  "log_auth": true,
  "log_netflow": true,
  "log_debug": false
}
```

## Certificate Management

### List Certificates

```http
GET /api/certificates
```

**Response:**

```json
{
  "success": true,
  "data": {
    "certificates": [
      {
        "id": 1,
        "name": "wildcard-company-com",
        "common_name": "*.company.com",
        "source_type": "vault",
        "auto_renew": true,
        "expires_at": "2025-01-15T00:00:00Z",
        "created_at": "2024-01-15T00:00:00Z",
        "status": "active"
      }
    ],
    "total": 1
  }
}
```

### Upload Certificate

```http
POST /api/certificates
```

**Request Body (multipart/form-data):**

```
name: my-certificate
cert_file: [certificate.crt file]
key_file: [private.key file]
ca_file: [ca.crt file] (optional)
auto_renew: false
```

### Generate Wildcard Certificate

```http
POST /api/certificates/generate-wildcard
```

**Request Body:**

```json
{
  "domain": "company.com",
  "validity_years": 1
}
```

### Delete Certificate

```http
DELETE /api/certificates/{cert_id}
```

## Proxy Management

### Register Proxy

```http
POST /api/proxy/register
```

**Headers:**

```http
X-Cluster-API-Key: cluster_api_key
```

**Request Body:**

```json
{
  "hostname": "proxy-node-01",
  "capabilities": ["ebpf", "xdp"],
  "version": "v0.1.1"
}
```

**Response:**

```json
{
  "success": true,
  "data": {
    "proxy_id": "proxy_123456789",
    "cluster_id": 1,
    "config_endpoint": "/api/config/1",
    "heartbeat_interval": 60
  }
}
```

### Get Configuration

```http
GET /api/config/{cluster_id}
```

**Headers:**

```http
X-Cluster-API-Key: cluster_api_key
```

**Response:**

```json
{
  "success": true,
  "data": {
    "cluster": {
      "id": 1,
      "name": "production",
      "syslog_endpoint": "syslog.company.com:514",
      "logging": {
        "auth": true,
        "netflow": true,
        "debug": false
      }
    },
    "services": [
      {
        "id": 1,
        "name": "web-backend",
        "ip_fqdn": "backend.internal.com",
        "auth_type": "jwt",
        "auth_config": { /* auth configuration */ }
      }
    ],
    "mappings": [
      {
        "id": 1,
        "source_services": ["web-frontend"],
        "dest_services": ["web-backend"],
        "protocols": ["tcp", "http"],
        "ports": [80, 443],
        "auth_required": true
      }
    ],
    "certificates": [
      {
        "id": 1,
        "name": "wildcard-company-com",
        "cert_data": "-----BEGIN CERTIFICATE-----...",
        "key_data": "-----BEGIN PRIVATE KEY-----..."
      }
    ]
  }
}
```

### Proxy Heartbeat

```http
POST /api/proxy/heartbeat
```

**Headers:**

```http
X-Cluster-API-Key: cluster_api_key
```

**Request Body:**

```json
{
  "proxy_id": "proxy_123456789",
  "status": "healthy",
  "metrics": {
    "active_connections": 1250,
    "requests_per_second": 850,
    "cpu_usage": 45.2,
    "memory_usage": 68.5
  }
}
```

### List Proxies

```http
GET /api/proxies
```

**Query Parameters:**

- `cluster_id` (integer): Filter by cluster
- `status` (string): Filter by status (healthy, unhealthy, offline)

**Response:**

```json
{
  "success": true,
  "data": {
    "proxies": [
      {
        "id": "proxy_123456789",
        "hostname": "proxy-node-01",
        "cluster_id": 1,
        "status": "healthy",
        "version": "v0.1.1",
        "capabilities": ["ebpf", "xdp"],
        "last_seen": "2024-01-15T11:00:00Z",
        "metrics": {
          "active_connections": 1250,
          "requests_per_second": 850
        }
      }
    ],
    "total": 1
  }
}
```

## License Management

### Get License Status

```http
GET /api/license-status
```

**Response:**

```json
{
  "success": true,
  "data": {
    "license_key": "PENG-XXXX-XXXX-XXXX-XXXX-ABCD",
    "license_type": "enterprise",
    "valid": true,
    "expires_at": "2025-12-31T23:59:59Z",
    "features": {
      "unlimited_proxies": true,
      "multi_cluster": true,
      "saml_authentication": true,
      "oauth2_authentication": true,
      "advanced_monitoring": true
    },
    "limits": {
      "max_proxies": -1,
      "max_clusters": -1,
      "current_proxies": 15,
      "current_clusters": 3
    }
  }
}
```

### Validate License

```http
POST /api/license/validate
```

**Request Body:**

```json
{
  "license_key": "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"
}
```

## Monitoring Endpoints

### Health Check

```http
GET /api/healthz
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T11:00:00Z",
  "checks": {
    "database": "healthy",
    "license_server": "healthy",
    "certificate_expiry": "healthy"
  },
  "version": "v0.1.1"
}
```

### Metrics

```http
GET /api/metrics
```

**Response (Prometheus format):**

```
# HELP marchproxy_users_total Total number of users
# TYPE marchproxy_users_total gauge
marchproxy_users_total 45

# HELP marchproxy_services_total Total number of services
# TYPE marchproxy_services_total gauge
marchproxy_services_total 127

# HELP marchproxy_proxies_active Number of active proxy instances
# TYPE marchproxy_proxies_active gauge
marchproxy_proxies_active{cluster="production"} 8
marchproxy_proxies_active{cluster="staging"} 3

# HELP marchproxy_requests_total Total number of API requests
# TYPE marchproxy_requests_total counter
marchproxy_requests_total{method="GET",endpoint="/api/services"} 1524
marchproxy_requests_total{method="POST",endpoint="/api/services"} 89
```

## Rate Limiting

The API implements rate limiting with the following limits:

- **Authentication endpoints**: 5 requests per minute per IP
- **General API endpoints**: 1000 requests per hour per API key
- **Configuration endpoints**: 100 requests per minute per cluster

Rate limit headers are included in responses:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642248600
```

## Error Codes

| Code | Description |
|------|-------------|
| `AUTHENTICATION_REQUIRED` | Authentication is required |
| `INVALID_CREDENTIALS` | Invalid username or password |
| `INVALID_API_KEY` | Invalid or expired API key |
| `INSUFFICIENT_PERMISSIONS` | User lacks required permissions |
| `VALIDATION_ERROR` | Request validation failed |
| `RESOURCE_NOT_FOUND` | Requested resource not found |
| `RESOURCE_CONFLICT` | Resource already exists or conflict |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded |
| `LICENSE_INVALID` | Invalid or expired license |
| `FEATURE_NOT_AVAILABLE` | Feature not available in current license |
| `INTERNAL_ERROR` | Internal server error |

## SDK Examples

### Python SDK

```python
import requests

class MarchProxyClient:
    def __init__(self, base_url, api_key):
        self.base_url = base_url
        self.headers = {
            'Authorization': f'Bearer {api_key}',
            'Content-Type': 'application/json'
        }

    def create_service(self, name, ip_fqdn, cluster_id=1):
        data = {
            'name': name,
            'ip_fqdn': ip_fqdn,
            'cluster_id': cluster_id,
            'auth_type': 'jwt'
        }
        response = requests.post(
            f'{self.base_url}/api/services',
            json=data,
            headers=self.headers
        )
        return response.json()

    def create_mapping(self, source, dest, ports=[80, 443]):
        data = {
            'source_services': [source],
            'dest_services': [dest],
            'protocols': ['tcp', 'http'],
            'ports': ports,
            'auth_required': True
        }
        response = requests.post(
            f'{self.base_url}/api/mappings',
            json=data,
            headers=self.headers
        )
        return response.json()

# Usage
client = MarchProxyClient('http://localhost:8000', 'your-api-key')
service = client.create_service('my-service', 'backend.company.com')
mapping = client.create_mapping('frontend', 'my-service')
```

### Go SDK

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

type MarchProxyClient struct {
    BaseURL string
    APIKey  string
    Client  *http.Client
}

type Service struct {
    Name     string `json:"name"`
    IPFQDN   string `json:"ip_fqdn"`
    ClusterID int   `json:"cluster_id"`
    AuthType string `json:"auth_type"`
}

func (c *MarchProxyClient) CreateService(service Service) (*http.Response, error) {
    data, _ := json.Marshal(service)
    req, _ := http.NewRequest("POST", c.BaseURL+"/api/services", bytes.NewBuffer(data))
    req.Header.Set("Authorization", "Bearer "+c.APIKey)
    req.Header.Set("Content-Type", "application/json")

    return c.Client.Do(req)
}

func main() {
    client := &MarchProxyClient{
        BaseURL: "http://localhost:8000",
        APIKey:  "your-api-key",
        Client:  &http.Client{},
    }

    service := Service{
        Name:      "my-service",
        IPFQDN:    "backend.company.com",
        ClusterID: 1,
        AuthType:  "jwt",
    }

    resp, err := client.CreateService(service)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    defer resp.Body.Close()

    fmt.Printf("Status: %s\n", resp.Status)
}
```

### cURL Examples

```bash
# Create a service
curl -X POST http://localhost:8000/api/services \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-service",
    "ip_fqdn": "backend.company.com",
    "cluster_id": 1,
    "auth_type": "jwt"
  }'

# Create a mapping
curl -X POST http://localhost:8000/api/mappings \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "source_services": ["frontend"],
    "dest_services": ["my-service"],
    "protocols": ["tcp", "http"],
    "ports": [80, 443],
    "auth_required": true
  }'

# Get proxy configuration
curl -X GET http://localhost:8000/api/config/1 \
  -H "X-Cluster-API-Key: cluster-api-key"
```

This completes the comprehensive Manager API documentation.