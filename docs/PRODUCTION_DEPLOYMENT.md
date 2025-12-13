# MarchProxy Production Deployment Guide

**Version**: v1.0.0
**Last Updated**: 2025-12-12

This guide provides comprehensive instructions for deploying MarchProxy in production environments with security hardening, high availability configuration, and operational best practices.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Pre-Deployment Checklist](#pre-deployment-checklist)
3. [Infrastructure Setup](#infrastructure-setup)
4. [Installation Methods](#installation-methods)
5. [SSL/TLS Certificate Setup](#ssltls-certificate-setup)
6. [Secrets Management](#secrets-management)
7. [Monitoring Setup](#monitoring-setup)
8. [Backup and Disaster Recovery](#backup-and-disaster-recovery)
9. [Scaling Guidelines](#scaling-guidelines)
10. [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

#### Minimum (Single Node, Small Deployment)
- **CPU**: 4 cores (2.0+ GHz)
- **Memory**: 8GB RAM
- **Storage**: 50GB available space (SSD recommended)
- **Network**: 1 Gbps network interface
- **OS**: Ubuntu 20.04+ / RHEL 8+ / Debian 11+
- **Kernel**: 4.18+ (for eBPF support)

#### Recommended (Production Deployment)
- **CPU**: 16+ cores (3.0+ GHz)
- **Memory**: 64GB RAM
- **Storage**: 500GB+ SSD (RAID-1 or better)
- **Network**: 10+ Gbps network interface
- **OS**: Ubuntu 22.04 LTS or RHEL 9+
- **Kernel**: 5.10+ (for full XDP support)

#### High Performance (Enterprise, 100+ Gbps)
- **CPU**: 32+ cores (3.5+ GHz)
- **Memory**: 128GB+ RAM
- **Storage**: 1TB+ NVMe SSD (RAID-0 or better)
- **Network**: 100 Gbps network interface
- **Specialized Hardware**: DPDK-compatible NIC, SR-IOV capable hardware
- **OS**: Ubuntu 22.04 LTS (optimized kernel)
- **Kernel**: 6.0+ with DPDK support patches

### Required Software

```bash
# Docker and Docker Compose (v2.0+)
docker --version    # >= 20.10.0
docker-compose --version  # >= 2.0.0

# For Kubernetes deployments
kubectl version --client  # >= 1.24.0
helm version       # >= 3.10.0

# For bare metal
golang.org/x/sys  # >= 1.7.0
libelf-dev
libcap-dev
```

### Network Requirements

- **Outbound HTTPS** (443): License server (license.penguintech.io)
- **Outbound DNS** (53): Domain resolution
- **Inbound HTTP** (80): Health checks, optional metrics
- **Inbound HTTPS** (443): User traffic
- **Internal Ports**: 8000-8082 for internal services
- **Syslog UDP** (5514): Centralized logging
- **Database** (5432): PostgreSQL access
- **Cache** (6379): Redis access

## Pre-Deployment Checklist

### Security Checklist

- [ ] Security policy reviewed with security team
- [ ] SSL/TLS certificates obtained and validated
- [ ] Private keys secured and access restricted
- [ ] RBAC policy defined for users and services
- [ ] Firewall rules configured
- [ ] Secrets management solution selected (Vault/Infisical)
- [ ] Network segmentation planned
- [ ] DDoS protection strategy established
- [ ] Encryption at rest enabled for databases
- [ ] Audit logging enabled and reviewed

### Operational Checklist

- [ ] Hardware provisioned and tested
- [ ] Network connectivity verified
- [ ] DNS records created
- [ ] Monitoring and alerting configured
- [ ] Logging infrastructure prepared
- [ ] Backup procedures documented
- [ ] Runbooks created for common issues
- [ ] On-call rotation established
- [ ] Disaster recovery procedures tested
- [ ] Capacity planning completed

### Documentation Checklist

- [ ] Custom configuration documented
- [ ] Network topology diagram created
- [ ] Architecture diagram updated
- [ ] Change control process defined
- [ ] Runbook for daily operations
- [ ] Escalation procedures documented
- [ ] Contact list for critical issues
- [ ] Disaster recovery procedures
- [ ] Configuration backup process
- [ ] Knowledge base populated

## Infrastructure Setup

### 1. Virtual Machine or Bare Metal Preparation

#### Storage Setup

```bash
# Create LVM volumes for optimal performance
sudo lvm
lvm> pvcreate /dev/sdb
lvm> vgcreate marchproxy-vg /dev/sdb
lvm> lvcreate -L 100G -n postgres marchproxy-vg
lvm> lvcreate -L 100G -n app marchproxy-vg
lvm> lvcreate -L 50G -n logs marchproxy-vg
lvm> exit

# Format volumes
sudo mkfs.ext4 /dev/marchproxy-vg/postgres
sudo mkfs.ext4 /dev/marchproxy-vg/app
sudo mkfs.ext4 /dev/marchproxy-vg/logs

# Create mount points
sudo mkdir -p /mnt/postgres /mnt/app /mnt/logs
sudo mount /dev/marchproxy-vg/postgres /mnt/postgres
sudo mount /dev/marchproxy-vg/app /mnt/app
sudo mount /dev/marchproxy-vg/logs /mnt/logs

# Configure automatic mounting
echo "/dev/marchproxy-vg/postgres /mnt/postgres ext4 defaults 0 2" | sudo tee -a /etc/fstab
echo "/dev/marchproxy-vg/app /mnt/app ext4 defaults 0 2" | sudo tee -a /etc/fstab
echo "/dev/marchproxy-vg/logs /mnt/logs ext4 defaults 0 2" | sudo tee -a /etc/fstab
```

#### Network Configuration

```bash
# Configure static IP (example for Netplan on Ubuntu)
sudo cat > /etc/netplan/00-static.yaml <<EOF
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: no
      addresses:
        - 10.0.1.100/24
      gateway4: 10.0.1.1
      nameservers:
        addresses: [8.8.8.8, 8.8.4.4]
EOF

sudo netplan apply

# Configure MTU for performance
sudo ip link set dev eth0 mtu 9000  # Jumbo frames
echo "net.ipv4.tcp_mtu_probing = 1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

#### Kernel Tuning for Performance

```bash
# Add these to /etc/sysctl.conf
sudo cat >> /etc/sysctl.conf <<EOF
# Network Performance Tuning
net.core.rmem_max=134217728
net.core.wmem_max=134217728
net.ipv4.tcp_rmem=4096 87380 67108864
net.ipv4.tcp_wmem=4096 65536 67108864
net.core.netdev_max_backlog=5000
net.ipv4.tcp_max_syn_backlog=5000
net.ipv4.ip_local_port_range=1024 65535

# Connection Tracking
net.netfilter.nf_conntrack_max=1000000
net.netfilter.nf_conntrack_tcp_timeout_established=600

# eBPF/XDP Support
kernel.perf_event_paranoid=-1
kernel.unprivileged_bpf_disabled=0
kernel.unprivileged_userns_clone=1
EOF

sudo sysctl -p
```

### 2. Docker and Container Runtime Setup

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add current user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Configure Docker daemon for production
sudo cat > /etc/docker/daemon.json <<EOF
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "insecure-registries": [],
  "default-ulimits": {
    "nofile": {
      "Name": "nofile",
      "Hard": 65536,
      "Soft": 65536
    }
  }
}
EOF

sudo systemctl restart docker
```

### 3. Database Preparation

#### PostgreSQL Setup (if not using Docker)

```bash
# Install PostgreSQL 15+
sudo sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list'
wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo apt-key add -
sudo apt update && sudo apt install postgresql-15 -y

# Configure PostgreSQL for production
sudo cat >> /etc/postgresql/15/main/postgresql.conf <<EOF
# Production Configuration
max_connections = 200
shared_buffers = 16GB
effective_cache_size = 48GB
maintenance_work_mem = 4GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 10485kB
min_wal_size = 1GB
max_wal_size = 4GB

# SSL/TLS
ssl = on
ssl_cert_file = '/etc/postgresql/certs/server.crt'
ssl_key_file = '/etc/postgresql/certs/server.key'
EOF

sudo systemctl restart postgresql

# Create MarchProxy database and user
sudo -u postgres psql <<EOF
CREATE DATABASE marchproxy;
CREATE USER marchproxy WITH PASSWORD 'strong-password-here';
ALTER ROLE marchproxy SET client_encoding TO 'utf8';
ALTER ROLE marchproxy SET default_transaction_isolation TO 'read committed';
GRANT ALL PRIVILEGES ON DATABASE marchproxy TO marchproxy;
EOF
```

## Installation Methods

### Method 1: Docker Compose (Recommended)

```bash
# Clone repository
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy

# Create environment file
cp .env.example .env

# Edit configuration
nano .env  # Update with your settings

# Pull latest images
docker-compose pull

# Start services
docker-compose up -d

# Verify services
docker-compose ps
docker-compose logs -f api-server

# Run health checks
./scripts/health-check.sh
```

### Method 2: Kubernetes with Helm

#### Install Prerequisites

```bash
# Add Helm repository
helm repo add marchproxy https://charts.marchproxy.io
helm repo update

# Create namespace
kubectl create namespace marchproxy

# Create secrets for credentials
kubectl create secret generic marchproxy-secrets \
  --from-literal=postgres-password=your-password \
  --from-literal=redis-password=your-password \
  --from-literal=license-key=PENG-XXXX-XXXX-XXXX-XXXX-XXXX \
  -n marchproxy
```

#### Deploy with Helm

```bash
# Create values file
cat > values.yaml <<EOF
replicaCount: 3

apiServer:
  enabled: true
  resources:
    requests:
      memory: "2Gi"
      cpu: "2"
    limits:
      memory: "4Gi"
      cpu: "4"

webui:
  enabled: true
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"

proxyL7:
  enabled: true
  resources:
    requests:
      memory: "2Gi"
      cpu: "4"

proxyL3L4:
  enabled: true
  resources:
    requests:
      memory: "4Gi"
      cpu: "8"

postgresql:
  enabled: true
  persistence:
    size: 100Gi

redis:
  enabled: true
  persistence:
    size: 10Gi

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: marchproxy.example.com
      paths:
        - path: /
          pathType: Prefix
EOF

# Install MarchProxy
helm install marchproxy marchproxy/marchproxy \
  -f values.yaml \
  -n marchproxy

# Verify installation
kubectl get pods -n marchproxy
kubectl logs -f deployment/marchproxy-api-server -n marchproxy
```

### Method 3: Kubernetes Operator

```bash
# Install operator
kubectl apply -f https://github.com/marchproxy/marchproxy-operator/releases/latest/download/operator.yaml

# Create MarchProxy custom resource
kubectl apply -f - <<EOF
apiVersion: marchproxy.io/v1
kind: MarchProxy
metadata:
  name: production
spec:
  replicas: 3
  postgresql:
    enabled: true
    size: 100Gi
    backup:
      enabled: true
      schedule: "0 2 * * *"
  redis:
    enabled: true
    size: 10Gi
  monitoring:
    enabled: true
    prometheus:
      retention: 30d
    alerting:
      enabled: true
EOF

# Monitor deployment
kubectl watch deployment marchproxy-api-server
```

## SSL/TLS Certificate Setup

### Option 1: Let's Encrypt (Automatic)

```bash
# Install cert-manager (Kubernetes)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml

# Create ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Request certificate (automatic via ingress annotation)
# or manually:
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: marchproxy-cert
spec:
  secretName: marchproxy-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - marchproxy.example.com
EOF
```

### Option 2: Self-Signed Certificates

```bash
# Generate CA certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca.pem \
  -subj "/CN=MarchProxy-CA/O=YourOrg/C=US"

# Generate server certificate
openssl genrsa -out server-key.pem 4096
openssl req -new -key server-key.pem -out server.csr \
  -subj "/CN=marchproxy.example.com/O=YourOrg/C=US"

# Sign server certificate
openssl x509 -req -days 365 -in server.csr \
  -CA ca.pem -CAkey ca-key.pem -CAcreateserial \
  -out server.pem

# Create Kubernetes secret
kubectl create secret tls marchproxy-tls \
  --cert=server.pem \
  --key=server-key.pem \
  -n marchproxy
```

### Option 3: Commercial Certificates

```bash
# If you have certificates from a CA:

# For Docker Compose
cp server.pem docker/certs/
cp server-key.pem docker/certs/
cp ca.pem docker/certs/

# For Kubernetes
kubectl create secret tls marchproxy-tls \
  --cert=server.pem \
  --key=server-key.pem \
  -n marchproxy
```

## Secrets Management

### Using HashiCorp Vault

```bash
# Install and start Vault (example)
vault operator init
vault operator unseal

# Create secrets
vault kv put secret/marchproxy \
  postgres_password="strong-password" \
  redis_password="strong-password" \
  license_key="PENG-XXXX-XXXX-XXXX-XXXX-XXXX"

# Create authentication policy
vault policy write marchproxy - <<EOF
path "secret/data/marchproxy" {
  capabilities = ["read"]
}
EOF

# Generate token
vault token create -policy=marchproxy
```

### Using Infisical

```bash
# Create project in Infisical web interface
# Then deploy agent:

docker run -d \
  -e INFISICAL_TOKEN="your-token" \
  -e INFISICAL_PROJECT_ID="project-id" \
  -v /etc/marchproxy/.env:/etc/marchproxy/.env \
  infisical/infisical-agent:latest
```

### Manual Secrets Management (Less Recommended)

```bash
# Create encrypted .env file
cat > .env.encrypted <<EOF
# Database
DATABASE_URL=postgresql://marchproxy:password@postgres:5432/marchproxy
POSTGRES_PASSWORD=strong-password

# License
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-XXXX

# API Security
SECRET_KEY=very-long-random-key-min-32-chars

# Redis
REDIS_PASSWORD=redis-strong-password
EOF

# Encrypt with GPG
gpg -c .env.encrypted

# Restrict permissions
chmod 600 .env.encrypted
chmod 600 .env.encrypted.gpg

# On deployment, decrypt
gpg -d .env.encrypted.gpg > .env
```

## Monitoring Setup

### Prometheus Configuration

```yaml
# monitoring/prometheus/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s
  external_labels:
    cluster: 'production'
    environment: 'prod'

scrape_configs:
  - job_name: 'api-server'
    static_configs:
      - targets: ['api-server:8000']
    metrics_path: '/metrics'

  - job_name: 'proxy-l7'
    static_configs:
      - targets: ['proxy-l7:9901']
    metrics_path: '/stats/prometheus'

  - job_name: 'proxy-l3l4'
    static_configs:
      - targets: ['proxy-l3l4:8082']
    metrics_path: '/metrics'

  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']

# Alert rules
rule_files:
  - '/etc/prometheus/rules/*.yml'

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

### Grafana Dashboard Setup

```bash
# Import dashboards
curl -X POST http://localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @dashboards/api-server.json

curl -X POST http://localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @dashboards/proxy-l7.json

curl -X POST http://localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @dashboards/proxy-l3l4.json
```

### Alert Configuration

```yaml
# monitoring/alertmanager/alertmanager.yml
global:
  resolve_timeout: 5m

route:
  receiver: 'default'
  group_by: ['alertname', 'cluster']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h
  routes:
    - match:
        severity: critical
      receiver: 'critical'
      group_wait: 0s
      repeat_interval: 5m

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops-team@example.com'
        from: 'alerts@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alerts@example.com'
        auth_password: '${ALERT_EMAIL_PASSWORD}'

  - name: 'critical'
    email_configs:
      - to: 'critical-team@example.com'
        from: 'alerts@example.com'
        smarthost: 'smtp.example.com:587'
        auth_username: 'alerts@example.com'
        auth_password: '${ALERT_EMAIL_PASSWORD}'
    pagerduty_configs:
      - service_key: '${PAGERDUTY_SERVICE_KEY}'
```

## Backup and Disaster Recovery

### Database Backups

```bash
# Daily PostgreSQL backup
cat > /usr/local/bin/backup-postgres.sh <<'EOF'
#!/bin/bash
BACKUP_DIR="/mnt/backups/postgres"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Full backup
pg_dump -U marchproxy marchproxy | gzip > $BACKUP_DIR/marchproxy_$DATE.sql.gz

# Encrypt backup
gpg -e -r admin@example.com $BACKUP_DIR/marchproxy_$DATE.sql.gz

# Remove old backups
find $BACKUP_DIR -name "*.sql.gz.gpg" -mtime +$RETENTION_DAYS -delete

# Upload to S3 (optional)
aws s3 cp $BACKUP_DIR/marchproxy_$DATE.sql.gz.gpg s3://backup-bucket/marchproxy/
EOF

chmod +x /usr/local/bin/backup-postgres.sh

# Add to crontab
0 2 * * * /usr/local/bin/backup-postgres.sh
```

### Volume Snapshots (Docker Compose)

```bash
# Create snapshot of all volumes
docker run --rm \
  -v postgres_data:/data \
  -v /backup:/backup \
  alpine:latest \
  tar czf /backup/postgres_data_$(date +%s).tar.gz -C /data .

# For production, use storage-level snapshots
# Example: LVM snapshots
lvcreate -L10G -s -n postgres_data_snapshot /dev/marchproxy-vg/postgres_data
mount /dev/marchproxy-vg/postgres_data_snapshot /mnt/snapshot
tar czf /backup/postgres_data_$(date +%s).tar.gz -C /mnt/snapshot .
lvremove -f /dev/marchproxy-vg/postgres_data_snapshot
```

### Disaster Recovery Test

```bash
# Test recovery process monthly

# 1. Restore from backup
gunzip < /backup/marchproxy_backup.sql.gz | psql -U marchproxy

# 2. Verify data integrity
psql -U marchproxy marchproxy -c "SELECT COUNT(*) FROM clusters;"

# 3. Start services
docker-compose up -d

# 4. Verify health
curl -f http://localhost:8000/healthz
```

## Scaling Guidelines

### Vertical Scaling (Single Node)

**For 100K concurrent connections:**
- 8 CPU cores
- 32GB RAM
- 5 Gbps network

**For 500K concurrent connections:**
- 16 CPU cores
- 64GB RAM
- 10 Gbps network

**For 1M+ concurrent connections:**
- 32 CPU cores
- 128GB RAM
- 100 Gbps network

### Horizontal Scaling (Multiple Nodes)

```yaml
# Kubernetes example for 3-node cluster
apiVersion: apps/v1
kind: Deployment
metadata:
  name: marchproxy-api-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: marchproxy-api-server
  template:
    metadata:
      labels:
        app: marchproxy-api-server
    spec:
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app
                  operator: In
                  values:
                  - marchproxy-api-server
              topologyKey: kubernetes.io/hostname
      containers:
      - name: api-server
        image: marchproxy/api-server:v1.0.0
        resources:
          requests:
            cpu: 2
            memory: 4Gi
          limits:
            cpu: 4
            memory: 8Gi
```

## Troubleshooting

### Service Health Checks

```bash
# API Server
curl -v http://localhost:8000/healthz

# Proxy L7
curl -v http://localhost:9901/stats

# Proxy L3/L4
curl -v http://localhost:8082/healthz

# Database
pg_isready -h postgres -U marchproxy

# Redis
redis-cli -h redis ping
```

### Common Issues

#### Services Not Starting

```bash
# Check docker logs
docker-compose logs manager
docker-compose logs api-server

# Verify environment variables
docker-compose config

# Check resource availability
docker stats

# Restart services
docker-compose restart
```

#### Database Connection Failures

```bash
# Check PostgreSQL
docker-compose logs postgres

# Verify connection string
psql "postgresql://marchproxy:password@postgres:5432/marchproxy"

# Check connection pool
SELECT count(*) FROM pg_stat_activity;
```

#### High Memory Usage

```bash
# Check memory consumption
docker stats

# Check for memory leaks in application
docker-compose logs api-server | grep -i memory

# Increase memory limits
# Edit docker-compose.yml memory limits and restart
```

---

**Document Version**: v1.0.0
**Last Updated**: 2025-12-12
**Support**: [support@marchproxy.io](mailto:support@marchproxy.io)
