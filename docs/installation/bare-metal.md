# Bare Metal Installation Guide

This guide covers installing MarchProxy directly on bare metal servers for maximum performance and control.

## Prerequisites

### System Requirements

#### Minimum Requirements
- **CPU**: 4 cores, 2.0+ GHz (Intel/AMD x86_64)
- **Memory**: 8 GB RAM
- **Storage**: 50 GB available space (SSD recommended)
- **Network**: 1 Gbps network interface
- **OS**: Ubuntu 20.04+, RHEL 8+, CentOS 8+, or Debian 11+

#### Recommended for Production
- **CPU**: 16+ cores, 3.0+ GHz with hardware acceleration support
- **Memory**: 32+ GB RAM
- **Storage**: 500+ GB NVMe SSD
- **Network**: 10+ Gbps network interface with XDP/DPDK support
- **OS**: Ubuntu 22.04 LTS or RHEL 9

#### Network Interface Requirements

For maximum performance, verify your network interface supports:

```bash
# Check interface capabilities
ethtool -i eth0
ethtool -k eth0 | grep -E "(xdp|bpf)"

# Recommended interfaces for high performance:
# - Intel X710, XXV710, E810 series
# - Mellanox ConnectX-4, ConnectX-5, ConnectX-6 series
# - Broadcom NetXtreme-E series
```

### Kernel Requirements

```bash
# Check kernel version (4.18+ required, 5.4+ recommended)
uname -r

# Verify eBPF support
cat /proc/sys/kernel/unprivileged_bpf_disabled

# Check required kernel features
grep -E "(BPF|XDP)" /boot/config-$(uname -r)
```

### Software Dependencies

#### Ubuntu/Debian

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y \
    curl \
    wget \
    git \
    build-essential \
    python3 \
    python3-pip \
    python3-venv \
    postgresql-14 \
    postgresql-client-14 \
    redis-server \
    nginx \
    supervisor \
    htop \
    iotop \
    tcpdump \
    wireshark-common \
    bpftrace

# Install eBPF development tools
sudo apt install -y \
    clang \
    llvm \
    libbpf-dev \
    bpfcc-tools \
    linux-tools-$(uname -r) \
    linux-tools-common

# Install Go (1.21+)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

#### RHEL/CentOS

```bash
# Update system
sudo dnf update -y

# Enable additional repositories
sudo dnf install -y epel-release
sudo dnf config-manager --set-enabled crb

# Install required packages
sudo dnf install -y \
    curl \
    wget \
    git \
    gcc \
    gcc-c++ \
    make \
    python3 \
    python3-pip \
    postgresql-server \
    postgresql \
    redis \
    nginx \
    supervisor \
    htop \
    iotop \
    tcpdump \
    wireshark \
    bpftrace

# Install eBPF development tools
sudo dnf install -y \
    clang \
    llvm \
    libbpf-devel \
    bcc-tools \
    kernel-devel \
    kernel-headers

# Install Go
sudo dnf install -y golang

# Initialize PostgreSQL
sudo postgresql-setup --initdb
sudo systemctl enable postgresql
sudo systemctl start postgresql
```

## Installation Steps

### 1. Create System User

```bash
# Create marchproxy user
sudo useradd -r -s /bin/bash -d /opt/marchproxy -m marchproxy
sudo usermod -aG sudo marchproxy

# Add to required groups for eBPF
sudo usermod -aG netdev marchproxy
sudo usermod -aG bpf marchproxy 2>/dev/null || true
```

### 2. Download and Extract MarchProxy

```bash
# Switch to marchproxy user
sudo su - marchproxy

# Download latest release
cd /opt/marchproxy
wget https://github.com/marchproxy/marchproxy/releases/latest/download/marchproxy-linux-amd64.tar.gz
tar -xzf marchproxy-linux-amd64.tar.gz

# Alternatively, build from source
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy
```

### 3. Build from Source (Optional)

```bash
# Build manager
cd manager
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt

# Build proxy
cd ../proxy
go mod download
go build -o bin/proxy ./cmd/proxy

# Build eBPF programs
cd ebpf
make all

# Return to main directory
cd ..
```

### 4. Configure PostgreSQL Database

```bash
# Configure PostgreSQL
sudo -u postgres psql <<EOF
CREATE USER marchproxy WITH PASSWORD 'marchproxy_secure_password';
CREATE DATABASE marchproxy OWNER marchproxy;
GRANT ALL PRIVILEGES ON DATABASE marchproxy TO marchproxy;
\q
EOF

# Configure PostgreSQL settings for performance
sudo tee -a /etc/postgresql/14/main/postgresql.conf <<EOF

# MarchProxy Performance Settings
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 4MB
min_wal_size = 1GB
max_wal_size = 4GB
max_worker_processes = 8
max_parallel_workers_per_gather = 2
max_parallel_workers = 8
max_parallel_maintenance_workers = 2
EOF

# Restart PostgreSQL
sudo systemctl restart postgresql
```

### 5. Configure Manager

```bash
# Create configuration directory
sudo mkdir -p /etc/marchproxy
sudo chown marchproxy:marchproxy /etc/marchproxy

# Create manager configuration
cat > /etc/marchproxy/manager.yaml <<EOF
# MarchProxy Manager Configuration

# Database configuration
database:
  url: "postgresql://marchproxy:marchproxy_secure_password@localhost:5432/marchproxy"
  pool_size: 20
  max_overflow: 30
  echo: false

# Server configuration
server:
  host: "0.0.0.0"
  port: 8000
  workers: 4
  reload: false

# Security configuration
security:
  jwt_secret: "$(openssl rand -base64 32)"
  session_secret: "$(openssl rand -base64 32)"
  password_salt: "$(openssl rand -base64 16)"

# License configuration
license:
  server: "https://license.penguintech.io"
  key: "${ENTERPRISE_LICENSE:-}"
  cache_ttl: 3600

# Logging configuration
logging:
  level: "INFO"
  format: "json"
  file: "/var/log/marchproxy/manager.log"
  max_size: "100MB"
  max_files: 10

# Monitoring configuration
monitoring:
  enable_metrics: true
  enable_health: true
  metrics_port: 8001

# TLS configuration
tls:
  enabled: false
  cert_file: ""
  key_file: ""
  ca_file: ""

# Enterprise features
enterprise:
  enabled: ${ENTERPRISE_ENABLED:-false}
  features:
    multi_cluster: true
    saml_auth: true
    oauth2_auth: true
    advanced_monitoring: true
EOF
```

### 6. Configure Proxy

```bash
# Create proxy configuration
cat > /etc/marchproxy/proxy.yaml <<EOF
# MarchProxy Proxy Configuration

# Manager connection
manager:
  url: "http://localhost:8000"
  api_key: "${CLUSTER_API_KEY}"
  cluster_id: 1
  timeout: 30s
  retry_interval: 10s

# Server configuration
server:
  proxy_port: 8080
  metrics_port: 8081
  admin_port: 8082

# Performance configuration
performance:
  enable_ebpf: true
  enable_xdp: true
  enable_af_xdp: false
  enable_dpdk: false
  enable_sriov: false
  worker_threads: 0  # Auto-detect based on CPU cores
  buffer_size: 65536
  connection_pool_size: 1000

# Network configuration
network:
  interface: "eth0"
  bind_to_device: false
  reuse_port: true
  tcp_fastopen: true
  tcp_nodelay: true
  keepalive_timeout: 300s

# Security configuration
security:
  enable_waf: true
  enable_rate_limiting: true
  max_connections_per_ip: 1000
  request_timeout: 30s
  header_timeout: 10s

# Logging configuration
logging:
  level: "INFO"
  format: "json"
  file: "/var/log/marchproxy/proxy.log"
  max_size: "100MB"
  max_files: 10
  enable_access_log: true
  enable_debug_log: false

# Monitoring configuration
monitoring:
  enable_metrics: true
  enable_health: true
  enable_tracing: false
  metrics_interval: 10s

# Circuit breaker configuration
circuit_breaker:
  failure_threshold: 5
  reset_timeout: 60s
  timeout: 30s

# Cache configuration
cache:
  enabled: true
  type: "memory"  # or "redis"
  size: "100MB"
  ttl: 300s
EOF
```

### 7. Set Up System Services

#### Manager Service

```bash
# Create systemd service for manager
sudo tee /etc/systemd/system/marchproxy-manager.service <<EOF
[Unit]
Description=MarchProxy Manager
Documentation=https://github.com/marchproxy/marchproxy
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=marchproxy
Group=marchproxy
WorkingDirectory=/opt/marchproxy
Environment=PYTHONPATH=/opt/marchproxy/manager
Environment=DATABASE_URL=postgresql://marchproxy:marchproxy_secure_password@localhost:5432/marchproxy
ExecStart=/opt/marchproxy/manager/venv/bin/python -m py4web run apps
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=10
LimitNOFILE=65536
LimitNPROC=4096

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/marchproxy /var/log/marchproxy /tmp

[Install]
WantedBy=multi-user.target
EOF
```

#### Proxy Service

```bash
# Create systemd service for proxy
sudo tee /etc/systemd/system/marchproxy-proxy.service <<EOF
[Unit]
Description=MarchProxy Proxy
Documentation=https://github.com/marchproxy/marchproxy
After=network.target marchproxy-manager.service
Wants=marchproxy-manager.service

[Service]
Type=simple
User=marchproxy
Group=marchproxy
WorkingDirectory=/opt/marchproxy/proxy
Environment=CONFIG_FILE=/etc/marchproxy/proxy.yaml
Environment=CLUSTER_API_KEY=your-cluster-api-key
ExecStart=/opt/marchproxy/proxy/bin/proxy
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=10
LimitNOFILE=65536
LimitNPROC=4096

# Required capabilities for eBPF/XDP
CapabilityBoundingSet=CAP_NET_ADMIN CAP_SYS_ADMIN CAP_NET_RAW CAP_BPF
AmbientCapabilities=CAP_NET_ADMIN CAP_SYS_ADMIN CAP_NET_RAW CAP_BPF

# Security settings
NoNewPrivileges=false
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/marchproxy /var/log/marchproxy /sys/fs/bpf

[Install]
WantedBy=multi-user.target
EOF
```

### 8. Create Log Directories

```bash
# Create log directories
sudo mkdir -p /var/log/marchproxy
sudo chown marchproxy:marchproxy /var/log/marchproxy

# Configure log rotation
sudo tee /etc/logrotate.d/marchproxy <<EOF
/var/log/marchproxy/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 marchproxy marchproxy
    postrotate
        systemctl reload marchproxy-manager marchproxy-proxy
    endscript
}
EOF
```

### 9. Configure Firewall

```bash
# Configure firewall rules
sudo ufw allow 8000/tcp comment "MarchProxy Manager"
sudo ufw allow 8080/tcp comment "MarchProxy Proxy"
sudo ufw allow 8081/tcp comment "MarchProxy Metrics"

# For enterprise monitoring
sudo ufw allow 9090/tcp comment "Prometheus"
sudo ufw allow 3000/tcp comment "Grafana"
```

### 10. Start Services

```bash
# Reload systemd configuration
sudo systemctl daemon-reload

# Enable and start services
sudo systemctl enable marchproxy-manager
sudo systemctl enable marchproxy-proxy

# Start manager first
sudo systemctl start marchproxy-manager

# Wait for manager to be ready
sleep 30

# Start proxy
sudo systemctl start marchproxy-proxy

# Check service status
sudo systemctl status marchproxy-manager
sudo systemctl status marchproxy-proxy
```

### 11. Verify Installation

```bash
# Check service health
curl http://localhost:8000/healthz
curl http://localhost:8081/healthz

# Check metrics
curl http://localhost:8000/metrics
curl http://localhost:8081/metrics

# Check logs
sudo journalctl -u marchproxy-manager -f
sudo journalctl -u marchproxy-proxy -f

# Test proxy functionality
curl -x http://localhost:8080 http://httpbin.org/ip
```

## Performance Optimization

### 1. Kernel Tuning

```bash
# Create kernel tuning configuration
sudo tee /etc/sysctl.d/99-marchproxy.conf <<EOF
# Network performance tuning
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.core.rmem_default = 131072
net.core.wmem_default = 131072
net.ipv4.tcp_rmem = 4096 131072 134217728
net.ipv4.tcp_wmem = 4096 131072 134217728
net.ipv4.tcp_congestion_control = bbr
net.core.netdev_max_backlog = 30000
net.ipv4.tcp_max_syn_backlog = 8192
net.ipv4.tcp_slow_start_after_idle = 0
net.ipv4.tcp_tw_reuse = 1

# eBPF/XDP tuning
kernel.unprivileged_bpf_disabled = 0
net.core.bpf_jit_enable = 1
net.core.bpf_jit_harden = 0

# Memory management
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5

# File descriptor limits
fs.file-max = 1048576
fs.nr_open = 1048576
EOF

# Apply settings
sudo sysctl -p /etc/sysctl.d/99-marchproxy.conf
```

### 2. CPU Affinity and NUMA

```bash
# Install NUMA tools
sudo apt install -y numactl

# Check NUMA topology
numactl --hardware

# Pin services to specific NUMA nodes
sudo systemctl edit marchproxy-proxy
```

Add to the override file:
```ini
[Service]
ExecStart=
ExecStart=/usr/bin/numactl --cpunodebind=0 --membind=0 /opt/marchproxy/proxy/bin/proxy
```

### 3. Huge Pages Configuration

```bash
# Configure huge pages
echo 1024 | sudo tee /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# Make persistent
echo 'vm.nr_hugepages = 1024' | sudo tee -a /etc/sysctl.d/99-marchproxy.conf

# Verify huge pages
cat /proc/meminfo | grep Huge
```

### 4. Network Interface Optimization

```bash
# Optimize network interface
sudo ethtool -K eth0 rx on tx on
sudo ethtool -K eth0 gro on gso on
sudo ethtool -K eth0 tso on
sudo ethtool -G eth0 rx 4096 tx 4096

# For XDP-capable interfaces
sudo ethtool -K eth0 hw-tc-offload on
sudo ethtool -K eth0 generic-receive-offload on
```

## Monitoring Setup

### 1. Install Prometheus

```bash
# Create prometheus user
sudo useradd -r -s /bin/false prometheus

# Download and install Prometheus
cd /tmp
wget https://github.com/prometheus/prometheus/releases/latest/download/prometheus-2.45.0.linux-amd64.tar.gz
tar -xzf prometheus-2.45.0.linux-amd64.tar.gz
sudo mv prometheus-2.45.0.linux-amd64 /opt/prometheus
sudo chown -R prometheus:prometheus /opt/prometheus

# Create configuration
sudo mkdir -p /etc/prometheus
sudo tee /etc/prometheus/prometheus.yml <<EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'marchproxy-manager'
    static_configs:
      - targets: ['localhost:8001']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'marchproxy-proxy'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']
EOF

# Create systemd service
sudo tee /etc/systemd/system/prometheus.service <<EOF
[Unit]
Description=Prometheus
After=network.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/opt/prometheus/prometheus \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/var/lib/prometheus \
  --web.console.templates=/opt/prometheus/consoles \
  --web.console.libraries=/opt/prometheus/console_libraries \
  --web.listen-address=0.0.0.0:9090

[Install]
WantedBy=multi-user.target
EOF

# Start Prometheus
sudo systemctl enable prometheus
sudo systemctl start prometheus
```

### 2. Install Grafana

```bash
# Add Grafana repository
curl -fsSL https://packages.grafana.com/gpg.key | sudo apt-key add -
echo "deb https://packages.grafana.com/oss/deb stable main" | sudo tee /etc/apt/sources.list.d/grafana.list

# Install Grafana
sudo apt update
sudo apt install -y grafana

# Start Grafana
sudo systemctl enable grafana-server
sudo systemctl start grafana-server
```

## Backup and Maintenance

### 1. Database Backup

```bash
# Create backup script
sudo tee /opt/marchproxy/backup.sh <<EOF
#!/bin/bash
BACKUP_DIR="/opt/marchproxy/backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Database backup
pg_dump -h localhost -U marchproxy marchproxy > $BACKUP_DIR/db_backup_$DATE.sql

# Configuration backup
tar -czf $BACKUP_DIR/config_backup_$DATE.tar.gz /etc/marchproxy

# Keep only last 30 days
find $BACKUP_DIR -name "*.sql" -mtime +30 -delete
find $BACKUP_DIR -name "*.tar.gz" -mtime +30 -delete
EOF

chmod +x /opt/marchproxy/backup.sh

# Schedule backup
echo "0 2 * * * /opt/marchproxy/backup.sh" | sudo crontab -u marchproxy -
```

### 2. Log Monitoring

```bash
# Monitor logs for errors
sudo tee /opt/marchproxy/monitor.sh <<EOF
#!/bin/bash
# Check for critical errors in logs
if journalctl -u marchproxy-manager --since "5 minutes ago" | grep -i "error\|critical\|fatal"; then
    echo "Critical errors detected in manager logs" | mail -s "MarchProxy Alert" admin@company.com
fi

if journalctl -u marchproxy-proxy --since "5 minutes ago" | grep -i "error\|critical\|fatal"; then
    echo "Critical errors detected in proxy logs" | mail -s "MarchProxy Alert" admin@company.com
fi
EOF

chmod +x /opt/marchproxy/monitor.sh

# Schedule monitoring
echo "*/5 * * * * /opt/marchproxy/monitor.sh" | sudo crontab -u marchproxy -
```

## Troubleshooting

### Common Issues

1. **eBPF Programs Not Loading**
   ```bash
   # Check kernel support
   ls /sys/kernel/debug/tracing/

   # Check capabilities
   sudo -u marchproxy bpftool prog list
   ```

2. **XDP Not Working**
   ```bash
   # Check interface support
   sudo ethtool -i eth0 | grep driver

   # Load XDP program manually
   sudo ip link set dev eth0 xdp obj proxy.o sec xdp_prog
   ```

3. **Performance Issues**
   ```bash
   # Check CPU usage
   htop

   # Check network statistics
   ss -tuln
   netstat -i

   # Check eBPF map statistics
   sudo bpftool map dump id $(sudo bpftool map list | grep rate_limit | awk '{print $1}' | sed 's/://g')
   ```

4. **Database Connection Issues**
   ```bash
   # Test database connection
   psql -h localhost -U marchproxy -d marchproxy -c "SELECT version();"

   # Check PostgreSQL logs
   sudo tail -f /var/log/postgresql/postgresql-14-main.log
   ```

### Performance Debugging

```bash
# Check system performance
sudo perf top
sudo iotop -a

# Check eBPF program performance
sudo bpftrace -e 'kprobe:xdp_do_generic_redirect { @[comm] = count(); }'

# Monitor network traffic
sudo tcpdump -i eth0 -n -c 100

# Check proxy statistics
curl -s http://localhost:8081/metrics | grep -E "(connections|requests|latency)"
```

This completes the comprehensive bare metal installation guide for MarchProxy.