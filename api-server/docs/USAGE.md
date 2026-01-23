# MarchProxy API Server - Usage Guide

Practical guide to using the MarchProxy API Server for common operations and workflows.

## Quick Start

### 1. Start the Server

```bash
# Using Docker
docker run -d \
  --name marchproxy-api \
  -p 8000:8000 \
  -e DATABASE_URL="postgresql+asyncpg://user:pass@db:5432/marchproxy" \
  -e SECRET_KEY="your-secret-key-minimum-32-characters" \
  marchproxy-api-server:latest

# Using local Python
pip install -r requirements.txt
./start.sh
```

### 2. Register Admin User

```bash
# Register the first user (automatically becomes admin)
curl -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@localhost.local",
    "username": "admin",
    "password": "admin123",
    "full_name": "Administrator"
  }'
```

### 3. Login and Get Token

```bash
# Login to get JWT token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@localhost.local",
    "password": "admin123"
  }' | jq -r '.access_token')

echo $TOKEN
```

### 4. Verify Installation

```bash
# Check health
curl http://localhost:8000/healthz | jq .

# View API documentation
# Open browser: http://localhost:8000/api/docs
```

## Common Workflows

### Workflow 1: Setting Up a New Cluster

This workflow shows how to create a cluster and register services.

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@localhost.local", "password": "admin123"}' \
  | jq -r '.access_token')

# 1. Create a new cluster
CLUSTER=$(curl -s -X POST http://localhost:8000/api/v1/clusters \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production-cluster-1",
    "description": "Primary production cluster",
    "max_proxies": 20,
    "syslog_server": "syslog.example.com:514",
    "enable_auth_logs": true,
    "enable_netflow_logs": true
  }')

CLUSTER_ID=$(echo $CLUSTER | jq -r '.id')
CLUSTER_KEY=$(echo $CLUSTER | jq -r '.api_key')

echo "Cluster ID: $CLUSTER_ID"
echo "Cluster API Key: $CLUSTER_KEY"

# 2. Create a service in the cluster
SERVICE=$(curl -s -X POST http://localhost:8000/api/v1/services \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"cluster_id\": \"$CLUSTER_ID\",
    \"name\": \"internal-api\",
    \"description\": \"Internal API service\",
    \"destination_ip\": \"10.0.1.100\",
    \"destination_port\": 443,
    \"protocol\": \"https\",
    \"auth_type\": \"jwt\",
    \"enable_health_check\": true,
    \"health_check_interval\": 30,
    \"health_check_path\": \"/health\"
  }")

SERVICE_ID=$(echo $SERVICE | jq -r '.id')
SERVICE_TOKEN=$(echo $SERVICE | jq -r '.service_token')

echo "Service ID: $SERVICE_ID"
echo "Service Token: $SERVICE_TOKEN"

# 3. Configure traffic shaping (optional)
curl -s -X POST http://localhost:8000/api/v1/traffic-shaping \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"service_id\": \"$SERVICE_ID\",
    \"name\": \"rate-limit\",
    \"enabled\": true,
    \"bandwidth_limit_mbps\": 100,
    \"connection_limit\": 5000,
    \"requests_per_second\": 10000
  }"

echo "Cluster setup complete!"
```

### Workflow 2: Managing Users and Permissions

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@localhost.local", "password": "admin123"}' \
  | jq -r '.access_token')

# 1. List all users
curl -s -X GET "http://localhost:8000/api/v1/users?limit=20" \
  -H "Authorization: Bearer $TOKEN" | jq .

# 2. Create a new user (not admin)
USER=$(curl -s -X POST http://localhost:8000/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "username": "user",
    "password": "SecurePass456!",
    "full_name": "Regular User"
  }')

USER_ID=$(echo $USER | jq -r '.id')

# 3. Update user permissions (promote to admin)
curl -s -X PUT "http://localhost:8000/api/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "is_admin": true
  }' | jq .

# 4. Disable user account
curl -s -X PUT "http://localhost:8000/api/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "is_active": false
  }' | jq .
```

### Workflow 3: TLS Certificate Management

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@localhost.local", "password": "admin123"}' \
  | jq -r '.access_token')

CLUSTER_ID="your-cluster-id"
SERVICE_ID="your-service-id"

# 1. Upload a certificate
CERT=$(curl -s -X POST http://localhost:8000/api/v1/certificates \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": "'$CLUSTER_ID'",
    "service_id": "'$SERVICE_ID'",
    "name": "api-cert-2024",
    "source": "upload",
    "certificate": "-----BEGIN CERTIFICATE-----\nMIID...\n-----END CERTIFICATE-----",
    "private_key": "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
    "auto_renewal": true
  }')

CERT_ID=$(echo $CERT | jq -r '.id')
echo "Certificate ID: $CERT_ID"

# 2. List certificates
curl -s -X GET "http://localhost:8000/api/v1/certificates?status=valid" \
  -H "Authorization: Bearer $TOKEN" | jq .

# 3. Check certificate expiry
curl -s -X GET "http://localhost:8000/api/v1/certificates/$CERT_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.expires_at'

# 4. Update auto-renewal
curl -s -X PUT "http://localhost:8000/api/v1/certificates/$CERT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "auto_renewal": false
  }'
```

### Workflow 4: Monitoring Proxies

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@localhost.local", "password": "admin123"}' \
  | jq -r '.access_token')

CLUSTER_ID="your-cluster-id"

# 1. List all proxies
curl -s -X GET "http://localhost:8000/api/v1/proxies?cluster_id=$CLUSTER_ID" \
  -H "Authorization: Bearer $TOKEN" | jq '.items[] | {name, status, cpu_percent, memory_percent}'

# 2. Get proxy details
PROXY_ID="your-proxy-id"
curl -s -X GET "http://localhost:8000/api/v1/proxies/$PROXY_ID" \
  -H "Authorization: Bearer $TOKEN" | jq .

# 3. Monitor proxy health
while true; do
  clear
  echo "=== Proxy Health Status ==="
  curl -s -X GET "http://localhost:8000/api/v1/proxies/$PROXY_ID" \
    -H "Authorization: Bearer $TOKEN" | jq '{
      name: .name,
      status: .status,
      cpu: .cpu_percent,
      memory: .memory_percent,
      connections: .connections,
      throughput_mbps: .throughput_mbps,
      latency_avg: .latency_avg_ms,
      error_rate: .error_rate
    }'
  sleep 5
done
```

### Workflow 5: Rotating Secrets

```bash
# Get token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@localhost.local", "password": "admin123"}' \
  | jq -r '.access_token')

CLUSTER_ID="your-cluster-id"
SERVICE_ID="your-service-id"

# 1. Rotate cluster API key
echo "Rotating cluster API key..."
NEW_KEY=$(curl -s -X POST "http://localhost:8000/api/v1/clusters/$CLUSTER_ID/rotate-key" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.api_key')

echo "New cluster API key: $NEW_KEY"
echo "WARNING: Update any proxies using the old key!"

# 2. Rotate service authentication token
echo "Rotating service token..."
NEW_TOKEN=$(curl -s -X POST "http://localhost:8000/api/v1/services/$SERVICE_ID/rotate-token" \
  -H "Authorization: Bearer $TOKEN" | jq -r '.service_token')

echo "New service token: $NEW_TOKEN"
echo "WARNING: Update any clients using the old token!"

# 3. Change password
echo "Changing user password..."
curl -s -X POST "http://localhost:8000/api/v1/auth/change-password" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "current_password": "SecurePass123!",
    "new_password": "NewSecurePass456!"
  }' | jq .
```

### Workflow 6: Enable Two-Factor Authentication (2FA)

```bash
# 1. Login with your password
curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@localhost.local",
    "password": "admin123"
  }' | jq -r '.access_token' > token.txt

TOKEN=$(cat token.txt)

# 2. Enable 2FA (get QR code)
TOTP=$(curl -s -X POST "http://localhost:8000/api/v1/auth/2fa/enable" \
  -H "Authorization: Bearer $TOKEN")

echo "QR Code URI:"
echo $TOTP | jq -r '.qr_code_uri'

echo "TOTP Secret:"
TOTP_SECRET=$(echo $TOTP | jq -r '.totp_secret')
echo $TOTP_SECRET

echo "Backup codes:"
echo $TOTP | jq -r '.backup_codes[]'

# 3. Verify 2FA (use authenticator app)
# After scanning QR code and getting code from your authenticator:
curl -s -X POST "http://localhost:8000/api/v1/auth/2fa/verify" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "totp_code": "123456"  # Replace with actual code from authenticator
  }' | jq .

# 4. Test 2FA login
curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@localhost.local",
    "password": "admin123",
    "totp_code": "123456"
  }' | jq .
```

## Advanced Usage

### Using Python SDK

```python
import requests
from typing import Optional

class MarchProxyClient:
    """Python client for MarchProxy API"""

    def __init__(self, base_url: str, email: str, password: str):
        self.base_url = base_url
        self.session = requests.Session()
        self.login(email, password)

    def login(self, email: str, password: str):
        """Authenticate and store token"""
        response = self.session.post(
            f"{self.base_url}/api/v1/auth/login",
            json={"email": email, "password": password}
        )
        response.raise_for_status()
        token = response.json()["access_token"]
        self.session.headers.update({"Authorization": f"Bearer {token}"})

    def create_cluster(self, name: str, max_proxies: int = 10) -> dict:
        """Create a new cluster"""
        response = self.session.post(
            f"{self.base_url}/api/v1/clusters",
            json={
                "name": name,
                "max_proxies": max_proxies,
                "enable_auth_logs": True,
                "enable_netflow_logs": True
            }
        )
        response.raise_for_status()
        return response.json()

    def create_service(
        self,
        cluster_id: str,
        name: str,
        destination_ip: str,
        destination_port: int,
        protocol: str = "https"
    ) -> dict:
        """Create a service in a cluster"""
        response = self.session.post(
            f"{self.base_url}/api/v1/services",
            json={
                "cluster_id": cluster_id,
                "name": name,
                "destination_ip": destination_ip,
                "destination_port": destination_port,
                "protocol": protocol,
                "auth_type": "jwt",
                "enable_health_check": True
            }
        )
        response.raise_for_status()
        return response.json()

    def list_proxies(self, cluster_id: str) -> list:
        """List proxies in a cluster"""
        response = self.session.get(
            f"{self.base_url}/api/v1/proxies",
            params={"cluster_id": cluster_id}
        )
        response.raise_for_status()
        return response.json()["items"]

    def get_proxy_metrics(self, proxy_id: str) -> dict:
        """Get detailed metrics for a proxy"""
        response = self.session.get(
            f"{self.base_url}/api/v1/proxies/{proxy_id}"
        )
        response.raise_for_status()
        return response.json()


# Usage example
client = MarchProxyClient(
    base_url="http://localhost:8000",
    email="admin@localhost.local",
    password="admin123"
)

# Create cluster
cluster = client.create_cluster(name="production-1", max_proxies=20)
cluster_id = cluster["id"]

# Create service
service = client.create_service(
    cluster_id=cluster_id,
    name="api-backend",
    destination_ip="10.0.1.100",
    destination_port=443,
    protocol="https"
)

# List proxies
proxies = client.list_proxies(cluster_id)
for proxy in proxies:
    print(f"{proxy['name']}: {proxy['status']}")
```

### Using Terraform

```hcl
# Configure the MarchProxy provider
terraform {
  required_providers {
    marchproxy = {
      source = "penguintech/marchproxy"
      version = "~> 1.0"
    }
  }
}

provider "marchproxy" {
  api_url = "http://localhost:8000"
  api_key = var.api_key
}

# Create a cluster
resource "marchproxy_cluster" "production" {
  name        = "production-cluster-1"
  description = "Primary production cluster"
  max_proxies = 20

  syslog_config {
    server           = "syslog.example.com:514"
    enable_auth_logs = true
  }
}

# Create a service
resource "marchproxy_service" "api" {
  cluster_id        = marchproxy_cluster.production.id
  name              = "internal-api"
  destination_ip    = "10.0.1.100"
  destination_port  = 443
  protocol          = "https"
  auth_type         = "jwt"
  enable_health_check = true
}

# Create traffic shaping rule
resource "marchproxy_traffic_shape" "api_limit" {
  service_id           = marchproxy_service.api.id
  name                 = "rate-limit"
  bandwidth_limit_mbps = 100
  connection_limit     = 5000
}

output "cluster_api_key" {
  value     = marchproxy_cluster.production.api_key
  sensitive = true
}
```

## CLI Tools

### Using curl with Helper Functions

```bash
# Save to ~/.bashrc or ~/.zshrc
marchproxy_api_url="http://localhost:8000"
marchproxy_email="admin@localhost.local"
marchproxy_password="admin123"

# Get token
mpx_token() {
  curl -s -X POST "$marchproxy_api_url/api/v1/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"$marchproxy_email\",
      \"password\": \"$marchproxy_password\"
    }" | jq -r '.access_token'
}

# List clusters
mpx_clusters() {
  local token=$(mpx_token)
  curl -s -X GET "$marchproxy_api_url/api/v1/clusters" \
    -H "Authorization: Bearer $token" | jq '.items[] | {id, name, proxy_count}'
}

# List services in cluster
mpx_services() {
  local cluster_id=$1
  local token=$(mpx_token)
  curl -s -X GET "$marchproxy_api_url/api/v1/services?cluster_id=$cluster_id" \
    -H "Authorization: Bearer $token" | jq '.items[] | {id, name, protocol, destination_port}'
}

# Get proxy status
mpx_proxy_status() {
  local proxy_id=$1
  local token=$(mpx_token)
  curl -s -X GET "$marchproxy_api_url/api/v1/proxies/$proxy_id" \
    -H "Authorization: Bearer $token" | jq '{status, cpu_percent, memory_percent, connections, throughput_mbps}'
}

# Usage:
# mpx_clusters
# mpx_services <cluster-id>
# mpx_proxy_status <proxy-id>
```

## Troubleshooting

### API Endpoint Returns 401 Unauthorized

```bash
# Check if token is expired
TOKEN="your_token"
echo $TOKEN | cut -d. -f2 | base64 -d | jq .

# Get new token
curl -s -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin@localhost.local", "password": "admin123"}' | jq -r '.access_token'
```

### 2FA Code Not Working

```bash
# Verify time sync on server
date

# Check TOTP secret is correct
# Disable and re-enable 2FA with correct secret
TOKEN="your_token"
curl -s -X POST "http://localhost:8000/api/v1/auth/2fa/disable" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"password": "your_password"}'
```

### Cluster API Key Expired

```bash
# Rotate cluster API key
TOKEN="your_token"
CLUSTER_ID="cluster_id"
curl -s -X POST "http://localhost:8000/api/v1/clusters/$CLUSTER_ID/rotate-key" \
  -H "Authorization: Bearer $TOKEN" | jq '.api_key'
```

## Performance Tips

1. **Use Pagination**: Don't fetch all items, use `skip` and `limit`
2. **Cache Tokens**: Reuse tokens instead of logging in repeatedly
3. **Batch Operations**: Group related operations when possible
4. **Use Connection Pooling**: Configure database pool settings
5. **Enable Caching**: Set `ENABLE_CACHE=true` for frequently accessed data
6. **Monitor Metrics**: Check `/metrics` endpoint for performance insights
