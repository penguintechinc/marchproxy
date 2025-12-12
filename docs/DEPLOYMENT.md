# MarchProxy Deployment Guide

**Version:** 1.0.0
**Last Updated:** 2025-12-12

## Table of Contents

- [Overview](#overview)
- [System Requirements](#system-requirements)
- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [Deployment Methods](#deployment-methods)
  - [Docker Compose](#docker-compose-recommended)
  - [Kubernetes](#kubernetes-deployment)
  - [Bare Metal](#bare-metal-deployment)
- [Environment Variables](#environment-variables)
- [Initial Configuration](#initial-configuration)
- [Health Check Validation](#health-check-validation)
- [Security Hardening](#security-hardening)
- [Monitoring Setup](#monitoring-setup)
- [Backup & Recovery](#backup--recovery)

## Overview

This guide provides step-by-step instructions for deploying MarchProxy v1.0.0 in production environments. MarchProxy supports three primary deployment methods:

1. **Docker Compose** - Recommended for single-node deployments and testing
2. **Kubernetes** - Recommended for multi-node production clusters
3. **Bare Metal** - For maximum performance and full control

## System Requirements

### Minimum Requirements (Community Edition)

| Component | Requirement |
|-----------|-------------|
| **CPU** | 2 cores (4 threads) |
| **Memory** | 4 GB RAM |
| **Storage** | 20 GB SSD |
| **Network** | 1 Gbps NIC |
| **OS** | Linux kernel 4.18+ (for eBPF support) |
| **Docker** | Docker 20.10+ and Docker Compose 2.0+ |

**Supported Operating Systems:**
- Ubuntu 20.04 LTS or later
- Debian 11 or later
- RHEL/CentOS 8 or later
- Fedora 35 or later

### Recommended Requirements (Enterprise Production)

| Component | Requirement |
|-----------|-------------|
| **CPU** | 8+ cores (16+ threads) |
| **Memory** | 32 GB RAM |
| **Storage** | 200 GB NVMe SSD |
| **Network** | 10+ Gbps NIC (preferably 25/40 Gbps) |
| **OS** | Ubuntu 22.04 LTS with kernel 5.15+ |
| **Docker** | Docker 24.0+ and Docker Compose 2.20+ |

**For XDP/AF_XDP acceleration:**
- Linux kernel 5.10+ (5.15+ recommended)
- Network card with XDP support (Intel i40e, ixgbe, mlx5, etc.)

**For DPDK/SR-IOV (Enterprise):**
- IOMMU-capable CPU (Intel VT-d or AMD-Vi)
- SR-IOV capable NIC
- Hugepages configured (2MB or 1GB pages)

### Network Requirements

| Port | Protocol | Purpose | Exposure |
|------|----------|---------|----------|
| 80 | TCP | HTTP (Ingress Proxy) | External |
| 443 | TCP | HTTPS/mTLS (Ingress Proxy) | External |
| 8000 | TCP | Manager API/Web UI | Internal |
| 8080 | TCP | Egress Proxy (Services) | Internal |
| 8081 | TCP | Egress Proxy Admin/Metrics | Internal |
| 8082 | TCP | Ingress Proxy Admin/Metrics | Internal |
| 5432 | TCP | PostgreSQL | Internal |
| 6379 | TCP | Redis | Internal |
| 9090 | TCP | Prometheus | Internal |
| 3000 | TCP | Grafana | Internal |
| 5601 | TCP | Kibana | Internal |

## Pre-Deployment Checklist

Before deploying MarchProxy, ensure you have:

- [ ] Verified system meets minimum requirements
- [ ] Obtained Enterprise license key (if applicable)
- [ ] Prepared SSL/TLS certificates (or will use auto-generation)
- [ ] Configured firewall rules
- [ ] Set up backup storage
- [ ] Prepared monitoring infrastructure (Prometheus, Grafana)
- [ ] Configured centralized logging (ELK stack or syslog)
- [ ] Reviewed security hardening requirements
- [ ] Tested DNS resolution for all service endpoints
- [ ] Allocated IP addresses and network segments

## Deployment Methods

### Docker Compose (Recommended)

Docker Compose provides the fastest deployment path for single-node setups.

#### Step 1: Clone Repository

```bash
# Clone the MarchProxy repository
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy

# Checkout v1.0.0 release
git checkout v1.0.0
```

#### Step 2: Configure Environment Variables

```bash
# Copy the example environment file
cp .env.example .env

# Edit the environment file
nano .env
```

**Required environment variables:**

```bash
# PostgreSQL Configuration
POSTGRES_PASSWORD=<strong_password>
POSTGRES_USER=marchproxy
POSTGRES_DB=marchproxy

# Redis Configuration
REDIS_PASSWORD=<strong_password>

# Manager Configuration
SECRET_KEY=<generate_random_64_char_string>
ADMIN_PASSWORD=<initial_admin_password>

# License Configuration (Enterprise)
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD

# Cluster API Key (for proxies)
CLUSTER_API_KEY=<cluster_specific_api_key>

# Syslog Configuration
SYSLOG_ENDPOINT=syslog.company.com:514

# Domain Configuration
DOMAIN=marchproxy.company.com
```

**Generate strong secrets:**

```bash
# Generate SECRET_KEY
openssl rand -hex 32

# Generate passwords
openssl rand -base64 32
```

#### Step 3: Deploy the Stack

```bash
# Start all services
docker-compose up -d

# Verify all containers are running
docker-compose ps

# Expected output:
# NAME                 STATUS      PORTS
# marchproxy-manager   Up 2m       0.0.0.0:8000->8000/tcp
# marchproxy-proxy-ingress   Up 2m   0.0.0.0:80->80/tcp, 0.0.0.0:443->443/tcp
# marchproxy-proxy-egress    Up 2m   0.0.0.0:8080-8081->8080-8081/tcp
# marchproxy-postgres  Up 2m       5432/tcp
# marchproxy-redis     Up 2m       6379/tcp
# ...
```

#### Step 4: Initialize Database

```bash
# The database initializes automatically on first run
# Verify initialization
docker-compose logs manager | grep "Database initialized"

# Check database connectivity
docker-compose exec manager python -c "from pydal import DAL; dal = DAL('postgres://marchproxy:<password>@postgres:5432/marchproxy'); print('Connected!')"
```

#### Step 5: Access Web Interface

```bash
# Open browser to manager URL
open http://localhost:8000

# Default credentials:
# Username: admin
# Password: <ADMIN_PASSWORD from .env>
# 2FA: Scan QR code with authenticator app
```

#### Step 6: Generate mTLS Certificates

```bash
# Option 1: Use the built-in certificate generator
docker-compose --profile tools run --rm cert-generator

# Option 2: Generate via Manager API
curl -X POST http://localhost:8000/api/certificates/generate-wildcard \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"domain": "company.com", "validity_years": 10}'

# Option 3: Upload existing certificates via Web UI
# Navigate to Certificates → Upload Certificate
```

#### Step 7: Configure Services and Mappings

```bash
# Example: Create backend service via API
curl -X POST http://localhost:8000/api/services \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "web-backend",
    "ip_fqdn": "10.0.1.100",
    "collection": "web-services",
    "cluster_id": 1,
    "auth_type": "jwt"
  }'

# Example: Create traffic mapping
curl -X POST http://localhost:8000/api/mappings \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "source_services": ["web-frontend"],
    "dest_services": ["web-backend"],
    "cluster_id": 1,
    "protocols": ["tcp", "http", "https"],
    "ports": [80, 443],
    "auth_required": true
  }'
```

### Kubernetes Deployment

For production multi-node deployments, Kubernetes provides orchestration, auto-scaling, and high availability.

#### Prerequisites

- Kubernetes cluster 1.24+ (EKS, GKE, AKS, or self-hosted)
- kubectl configured with cluster access
- Helm 3.10+ installed
- cert-manager installed (for automatic TLS)

#### Step 1: Create Namespace

```bash
# Create dedicated namespace
kubectl create namespace marchproxy

# Set as default namespace
kubectl config set-context --current --namespace=marchproxy
```

#### Step 2: Create Secrets

```bash
# Create database secrets
kubectl create secret generic marchproxy-db \
  --from-literal=postgres-password=<strong_password> \
  --from-literal=postgres-user=marchproxy \
  --from-literal=postgres-database=marchproxy

# Create application secrets
kubectl create secret generic marchproxy-secrets \
  --from-literal=secret-key=<64_char_secret> \
  --from-literal=admin-password=<admin_password> \
  --from-literal=redis-password=<redis_password>

# Create license secret (Enterprise)
kubectl create secret generic marchproxy-license \
  --from-literal=license-key=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

#### Step 3: Deploy with Helm

```bash
# Add MarchProxy Helm repository
helm repo add marchproxy https://charts.marchproxy.io
helm repo update

# Install MarchProxy
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --set manager.replicas=2 \
  --set proxy-ingress.replicas=3 \
  --set proxy-egress.replicas=3 \
  --set postgresql.persistence.size=100Gi \
  --set ingress.enabled=true \
  --set ingress.hostname=marchproxy.company.com \
  --set monitoring.enabled=true

# Verify deployment
kubectl get pods -n marchproxy
```

#### Step 4: Expose Services

```bash
# Option 1: Using LoadBalancer (Cloud providers)
kubectl apply -f - <<EOF
apiVersion: v1
kind: Service
metadata:
  name: marchproxy-ingress-lb
  namespace: marchproxy
spec:
  type: LoadBalancer
  ports:
    - name: http
      port: 80
      targetPort: 80
    - name: https
      port: 443
      targetPort: 443
  selector:
    app: marchproxy-proxy-ingress
EOF

# Option 2: Using Ingress Controller
kubectl apply -f k8s/ingress.yaml
```

#### Step 5: Configure Persistent Storage

```bash
# Apply persistent volume claims
kubectl apply -f k8s/pvc-postgres.yaml
kubectl apply -f k8s/pvc-redis.yaml

# Verify PVCs are bound
kubectl get pvc -n marchproxy
```

### Bare Metal Deployment

For maximum performance and full control, deploy directly on bare metal.

#### Prerequisites

- Linux server with kernel 5.10+
- Go 1.21+
- Python 3.12+
- PostgreSQL 15+
- Redis 7+
- systemd for service management

#### Step 1: Install Dependencies

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y \
  build-essential \
  clang \
  llvm \
  libelf-dev \
  linux-headers-$(uname -r) \
  postgresql-15 \
  redis-server \
  python3.12 \
  python3.12-venv \
  python3-pip

# RHEL/CentOS
sudo dnf install -y \
  gcc \
  clang \
  llvm \
  elfutils-libelf-devel \
  kernel-devel \
  postgresql15-server \
  redis \
  python3.12 \
  python3.12-devel
```

#### Step 2: Build Proxies

```bash
# Clone repository
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy

# Build proxy-egress
cd proxy-egress
go build -o /usr/local/bin/marchproxy-egress ./cmd/proxy

# Build proxy-ingress
cd ../proxy-ingress
go build -o /usr/local/bin/marchproxy-ingress ./cmd/proxy

# Verify builds
/usr/local/bin/marchproxy-egress --version
/usr/local/bin/marchproxy-ingress --version
```

#### Step 3: Setup Manager

```bash
# Create virtual environment
cd manager
python3.12 -m venv venv
source venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Initialize database
export DATABASE_URL="postgresql://marchproxy:<password>@localhost:5432/marchproxy"
python scripts/init_db.py

# Create systemd service
sudo cp deploy/marchproxy-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable marchproxy-manager
sudo systemctl start marchproxy-manager
```

#### Step 4: Configure Proxies

```bash
# Create proxy configuration directories
sudo mkdir -p /etc/marchproxy/{proxy-egress,proxy-ingress}
sudo mkdir -p /var/log/marchproxy

# Copy configuration templates
sudo cp config/proxy-egress.yaml /etc/marchproxy/proxy-egress/config.yaml
sudo cp config/proxy-ingress.yaml /etc/marchproxy/proxy-ingress/config.yaml

# Edit configurations
sudo nano /etc/marchproxy/proxy-egress/config.yaml
sudo nano /etc/marchproxy/proxy-ingress/config.yaml

# Create systemd services
sudo cp deploy/marchproxy-proxy-egress.service /etc/systemd/system/
sudo cp deploy/marchproxy-proxy-ingress.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable marchproxy-proxy-{egress,ingress}
sudo systemctl start marchproxy-proxy-{egress,ingress}
```

## Environment Variables

### Manager Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `REDIS_URL` | Yes | - | Redis connection string |
| `SECRET_KEY` | Yes | - | Application secret key (64 chars) |
| `ADMIN_PASSWORD` | Yes | - | Initial admin password |
| `LICENSE_KEY` | No | - | Enterprise license key |
| `SYSLOG_ENDPOINT` | No | - | Centralized syslog endpoint |
| `LOG_LEVEL` | No | `INFO` | Logging level (DEBUG, INFO, WARN, ERROR) |
| `ENABLE_2FA` | No | `true` | Require 2FA for all users |
| `SESSION_TIMEOUT` | No | `3600` | Session timeout in seconds |

### Proxy Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MANAGER_URL` | Yes | - | Manager API URL |
| `CLUSTER_API_KEY` | Yes | - | Cluster-specific API key |
| `PROXY_TYPE` | Yes | - | `ingress` or `egress` |
| `ENABLE_EBPF` | No | `true` | Enable eBPF acceleration |
| `ENABLE_XDP` | No | `true` | Enable XDP acceleration (Enterprise) |
| `ENABLE_AF_XDP` | No | `false` | Enable AF_XDP (Enterprise) |
| `NUMA_NODE` | No | `0` | NUMA node for CPU affinity |
| `CONFIG_REFRESH_INTERVAL` | No | `60` | Config refresh interval (seconds) |
| `HEARTBEAT_INTERVAL` | No | `30` | Heartbeat interval (seconds) |

## Initial Configuration

### Create First Service

After deployment, configure your first service via the web interface or API:

1. **Navigate to Services** → "Create Service"
2. **Fill in service details:**
   - Name: `web-backend`
   - IP/FQDN: `10.0.1.100`
   - Collection: `web-services`
   - Cluster: `default`
   - Auth Type: `jwt`

3. **Create Traffic Mapping:**
   - Source Services: `web-frontend`
   - Destination Services: `web-backend`
   - Protocols: `tcp`, `http`, `https`
   - Ports: `80`, `443`
   - Authentication Required: `Yes`

### Configure mTLS

Generate mTLS certificates for service-to-service authentication:

```bash
# Via API
curl -X POST http://localhost:8000/api/certificates/generate-wildcard \
  -H "Authorization: Bearer <jwt_token>" \
  -H "Content-Type: application/json" \
  -d '{"domain": "internal.company.com", "validity_years": 10}'

# Via Web UI
# Navigate to Certificates → Generate CA → Generate Wildcard Certificate
```

## Health Check Validation

Verify deployment health after installation:

```bash
# Manager health check
curl http://localhost:8000/api/healthz

# Expected response:
# {
#   "status": "healthy",
#   "checks": {
#     "database": "healthy",
#     "license_server": "healthy",
#     "certificate_expiry": "healthy"
#   },
#   "version": "v1.0.0"
# }

# Proxy-Egress health check
curl http://localhost:8081/healthz

# Proxy-Ingress health check
curl http://localhost:8082/healthz

# Metrics endpoint
curl http://localhost:8081/metrics
```

## Security Hardening

### SSL/TLS Configuration

```bash
# Use strong cipher suites only
# In manager config:
TLS_CIPHER_SUITES="TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"

# Disable TLS 1.0 and 1.1
MIN_TLS_VERSION="1.2"

# Enable HSTS
HSTS_MAX_AGE="31536000"
```

### Firewall Configuration

```bash
# Ubuntu/Debian (UFW)
sudo ufw allow from <internal_network> to any port 8000 proto tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# RHEL/CentOS (firewalld)
sudo firewall-cmd --permanent --add-service=http
sudo firewall-cmd --permanent --add-service=https
sudo firewall-cmd --permanent --add-port=8000/tcp --source=<internal_network>
sudo firewall-cmd --reload
```

### System Hardening

```bash
# Enable kernel hardening
sudo sysctl -w kernel.dmesg_restrict=1
sudo sysctl -w kernel.kptr_restrict=2
sudo sysctl -w kernel.unprivileged_bpf_disabled=1

# Limit core dumps
echo "* hard core 0" | sudo tee -a /etc/security/limits.conf

# Enable SELinux/AppArmor
sudo setenforce 1  # SELinux
sudo aa-enforce /etc/apparmor.d/*  # AppArmor
```

## Monitoring Setup

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'marchproxy-manager'
    static_configs:
      - targets: ['manager:8000']
    metrics_path: '/api/metrics'

  - job_name: 'marchproxy-proxy-egress'
    static_configs:
      - targets: ['proxy-egress:8081']
    metrics_path: '/metrics'

  - job_name: 'marchproxy-proxy-ingress'
    static_configs:
      - targets: ['proxy-ingress:8082']
    metrics_path: '/metrics'
```

### Grafana Dashboards

Pre-built Grafana dashboards are available in `monitoring/grafana/dashboards/`:

- `marchproxy-overview.json` - System overview
- `marchproxy-proxy-performance.json` - Proxy performance metrics
- `marchproxy-security.json` - Security and authentication metrics

Import via Grafana UI: Configuration → Dashboards → Import

## Backup & Recovery

### Database Backup

```bash
# Automated daily backup
cat > /usr/local/bin/marchproxy-backup.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/var/backups/marchproxy"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p $BACKUP_DIR

# Backup PostgreSQL
pg_dump -h localhost -U marchproxy marchproxy | gzip > $BACKUP_DIR/marchproxy_$TIMESTAMP.sql.gz

# Retain last 30 days
find $BACKUP_DIR -name "marchproxy_*.sql.gz" -mtime +30 -delete
EOF

chmod +x /usr/local/bin/marchproxy-backup.sh

# Add to cron
echo "0 2 * * * /usr/local/bin/marchproxy-backup.sh" | sudo crontab -
```

### Configuration Backup

```bash
# Backup certificates and configuration
tar -czf marchproxy-config-$(date +%Y%m%d).tar.gz \
  /etc/marchproxy \
  /var/lib/marchproxy/certs \
  .env

# Store off-site
aws s3 cp marchproxy-config-$(date +%Y%m%d).tar.gz s3://backups/marchproxy/
```

### Disaster Recovery

```bash
# Restore database
gunzip < marchproxy_20251212_020000.sql.gz | psql -h localhost -U marchproxy marchproxy

# Restore configuration
tar -xzf marchproxy-config-20251212.tar.gz -C /

# Restart services
docker-compose restart
# OR
sudo systemctl restart marchproxy-{manager,proxy-egress,proxy-ingress}
```

---

**Next Steps:**
- [Configure advanced features](configuration.md)
- [Set up monitoring and alerting](operations/monitoring.md)
- [Review troubleshooting guide](TROUBLESHOOTING.md)
- [Plan for high availability](operations/ha-setup.md)
