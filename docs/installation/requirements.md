# System Requirements and Prerequisites

This document outlines the system requirements and prerequisites for running MarchProxy in different environments and performance tiers.

## Overview

MarchProxy supports multiple deployment tiers with different performance characteristics and system requirements:

1. **Basic Tier**: Standard networking with eBPF acceleration
2. **Enhanced Tier**: XDP/AF_XDP with zero-copy networking
3. **Ultra Tier**: DPDK with kernel bypass and SR-IOV

## Minimum System Requirements

### Hardware Requirements

| Component | Minimum | Recommended | Ultra Performance |
|-----------|---------|-------------|-------------------|
| **CPU** | 2 cores, 2.0 GHz | 8 cores, 3.0 GHz | 16+ cores, 3.5+ GHz |
| **Memory** | 4 GB RAM | 16 GB RAM | 64+ GB RAM |
| **Storage** | 20 GB HDD | 100 GB SSD | 1+ TB NVMe SSD |
| **Network** | 1 Gbps NIC | 10 Gbps NIC | 25+ Gbps NIC with hardware acceleration |

### Operating System Requirements

#### Supported Operating Systems

| OS | Version | eBPF Support | XDP Support | DPDK Support |
|----|---------|--------------|-------------|--------------|
| **Ubuntu** | 20.04+ | ✅ | ✅ | ✅ |
| **RHEL** | 8.0+ | ✅ | ✅ | ✅ |
| **CentOS** | 8.0+ | ✅ | ✅ | ✅ |
| **Debian** | 11+ | ✅ | ✅ | ✅ |
| **Fedora** | 35+ | ✅ | ✅ | ✅ |
| **Amazon Linux** | 2022+ | ✅ | ✅ | ✅ |

#### Kernel Requirements

```bash
# Minimum kernel version
uname -r  # Should be 4.18+ (5.4+ recommended)

# Required kernel features
grep -E "(CONFIG_BPF=|CONFIG_XDP_SOCKETS=|CONFIG_CGROUPS_BPF=)" /boot/config-$(uname -r)

# Check eBPF support
ls /sys/kernel/debug/tracing/events/bpf/
cat /proc/sys/kernel/unprivileged_bpf_disabled
```

**Required Kernel Options:**
- `CONFIG_BPF=y`
- `CONFIG_BPF_SYSCALL=y`
- `CONFIG_BPF_JIT=y`
- `CONFIG_HAVE_EBPF_JIT=y`
- `CONFIG_XDP_SOCKETS=y` (for XDP acceleration)
- `CONFIG_CGROUPS_BPF=y`
- `CONFIG_NET_CLS_BPF=y`
- `CONFIG_NET_SCH_INGRESS=y`
- `CONFIG_VFIO=y` (for SR-IOV/DPDK)

## Network Interface Requirements

### Basic Tier - eBPF Acceleration

**Supported Interfaces:**
- Any standard Ethernet interface
- Virtual interfaces (for testing)
- Cloud instance network interfaces

**Verification:**
```bash
# Check interface status
ip link show
ethtool eth0

# Verify eBPF attachment capability
tc qdisc add dev eth0 clsact
tc filter add dev eth0 ingress bpf direct-action obj dummy.o sec ingress
```

### Enhanced Tier - XDP/AF_XDP

**Supported Interfaces:**
- Intel: X710, XXV710, E810 series
- Mellanox: ConnectX-4, ConnectX-5, ConnectX-6 series
- Broadcom: NetXtreme-E series
- Netronome: Agilio series
- Amazon: ENA (Enhanced Networking)

**Verification:**
```bash
# Check XDP support
ethtool -k eth0 | grep -E "(xdp|bpf)"

# Test XDP program loading
ip link set dev eth0 xdp obj dummy.o sec xdp || echo "XDP not supported"

# Check driver support
ethtool -i eth0
dmesg | grep -i xdp
```

**Driver-Specific Requirements:**

| Vendor | Driver | XDP Mode | AF_XDP | Hardware Offload |
|--------|--------|----------|--------|------------------|
| Intel | i40e | Native | ✅ | Limited |
| Intel | ice | Native | ✅ | ✅ |
| Intel | ixgbe | Generic | ✅ | ❌ |
| Mellanox | mlx5_core | Native | ✅ | ✅ |
| Mellanox | mlx4_core | Generic | ✅ | ❌ |
| Broadcom | bnxt_en | Native | ✅ | Limited |

### Ultra Tier - DPDK with SR-IOV

**Hardware Requirements:**
- IOMMU support (Intel VT-d or AMD-Vi)
- SR-IOV capable network interface
- Dedicated CPU cores for DPDK threads
- Huge page support (2MB or 1GB pages)

**Verification:**
```bash
# Check IOMMU support
dmesg | grep -i iommu
ls /sys/kernel/iommu_groups/

# Check SR-IOV capability
lspci -vvv | grep -i sriov

# Check huge page support
cat /proc/meminfo | grep Huge
ls /sys/kernel/mm/hugepages/
```

## Software Dependencies

### Core Dependencies

#### Manager (Python)

```bash
# Python requirements
python3 --version  # 3.8+ required, 3.11+ recommended

# System packages (Ubuntu/Debian)
sudo apt install -y \
    python3 \
    python3-pip \
    python3-venv \
    python3-dev \
    libpq-dev \
    libssl-dev \
    libffi-dev

# System packages (RHEL/CentOS)
sudo dnf install -y \
    python3 \
    python3-pip \
    python3-devel \
    postgresql-devel \
    openssl-devel \
    libffi-devel
```

#### Proxy (Go)

```bash
# Go requirements
go version  # 1.21+ required

# Install Go (if not present)
curl -LO https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

#### eBPF Development Tools

```bash
# Ubuntu/Debian
sudo apt install -y \
    clang \
    llvm \
    libbpf-dev \
    bpfcc-tools \
    linux-tools-$(uname -r) \
    linux-tools-common

# RHEL/CentOS
sudo dnf install -y \
    clang \
    llvm \
    libbpf-devel \
    bcc-tools \
    kernel-devel \
    kernel-headers
```

### Database Requirements

#### PostgreSQL (Recommended)

```bash
# Minimum version: PostgreSQL 12+
# Recommended: PostgreSQL 14+

# Ubuntu/Debian
sudo apt install -y postgresql-14 postgresql-client-14

# RHEL/CentOS
sudo dnf install -y postgresql-server postgresql

# Performance requirements
# - Minimum: 2GB RAM, 20GB storage
# - Recommended: 8GB RAM, 100GB SSD
# - Production: 16GB RAM, 500GB NVMe SSD
```

#### Alternative Database Support

| Database | Version | Support Level | Performance |
|----------|---------|---------------|-------------|
| PostgreSQL | 12+ | Full | Excellent |
| MySQL | 8.0+ | Full | Good |
| MariaDB | 10.5+ | Full | Good |
| SQLite | 3.35+ | Development only | Limited |

### Container Requirements

#### Docker

```bash
# Docker Engine 20.10+
docker --version

# Docker Compose 2.0+
docker-compose --version

# Required for eBPF in containers
sudo sysctl kernel.unprivileged_bpf_disabled=0
```

#### Kubernetes

```bash
# Kubernetes 1.19+
kubectl version --client

# Required features
# - CNI with eBPF support (Cilium recommended)
# - Privileged containers
# - HostNetwork access
# - Custom capabilities (CAP_NET_ADMIN, CAP_SYS_ADMIN, CAP_BPF)
```

## Performance Tier Requirements

### Basic Tier (1-10 Gbps)

**Minimum Requirements:**
- 4 CPU cores
- 8 GB RAM
- Standard NIC
- Kernel 4.18+

**Software:**
- eBPF support
- Basic networking stack
- Standard memory allocation

```bash
# Verification script
#!/bin/bash
echo "=== Basic Tier Requirements Check ==="

# CPU cores
cores=$(nproc)
echo "CPU cores: $cores"
[ $cores -ge 4 ] && echo "✅ CPU: OK" || echo "❌ CPU: Need 4+ cores"

# Memory
mem_gb=$(free -g | awk '/^Mem:/{print $2}')
echo "Memory: ${mem_gb}GB"
[ $mem_gb -ge 8 ] && echo "✅ Memory: OK" || echo "❌ Memory: Need 8GB+"

# Kernel version
kernel_version=$(uname -r | cut -d. -f1-2)
echo "Kernel: $kernel_version"
if [ "$(echo "$kernel_version >= 4.18" | bc)" -eq 1 ]; then
    echo "✅ Kernel: OK"
else
    echo "❌ Kernel: Need 4.18+"
fi

# eBPF support
if [ -f /sys/kernel/debug/tracing/trace ]; then
    echo "✅ eBPF: Supported"
else
    echo "❌ eBPF: Not available"
fi
```

### Enhanced Tier (10-40 Gbps)

**Requirements:**
- 8+ CPU cores
- 16+ GB RAM
- XDP-capable NIC
- Kernel 5.4+
- Huge pages support

**Optimization:**
```bash
# Huge pages configuration
echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages

# CPU isolation for network processing
echo "isolcpus=2-7" >> /proc/cmdline  # Reboot required

# Network tuning
echo 'net.core.netdev_max_backlog = 30000' >> /etc/sysctl.conf
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
```

### Ultra Tier (40-100+ Gbps)

**Requirements:**
- 16+ CPU cores (dedicated)
- 64+ GB RAM
- DPDK-capable NIC with SR-IOV
- Kernel 5.8+
- 1GB huge pages
- NUMA optimization

**Hardware Setup:**
```bash
# IOMMU configuration
echo "intel_iommu=on iommu=pt" >> /proc/cmdline  # Intel
echo "amd_iommu=on iommu=pt" >> /proc/cmdline    # AMD

# Huge pages (1GB)
echo "hugepagesz=1G hugepages=16" >> /proc/cmdline

# CPU isolation (example for 32-core system)
echo "isolcpus=8-31 nohz_full=8-31 rcu_nocbs=8-31" >> /proc/cmdline

# After reboot, verify DPDK requirements
modprobe vfio-pci
echo 1 > /sys/module/vfio/parameters/enable_unsafe_noiommu_mode
```

## Cloud Platform Requirements

### AWS

**Instance Types:**
- Minimum: t3.large (Basic tier)
- Recommended: c5n.xlarge (Enhanced tier)
- Ultra performance: c5n.18xlarge, m5dn.24xlarge

**Network Features:**
- Enhanced networking (SR-IOV)
- Placement groups for low latency
- Elastic Network Adapter (ENA) support

```bash
# Check ENA support
modinfo ena
ethtool -i eth0 | grep driver
```

### Azure

**Instance Types:**
- Minimum: Standard_D4s_v3 (Basic tier)
- Recommended: Standard_F8s_v2 (Enhanced tier)
- Ultra performance: Standard_M128s, Standard_HB120rs_v2

**Network Features:**
- Accelerated networking
- InfiniBand (for HPC instances)
- Single Root I/O Virtualization (SR-IOV)

### Google Cloud

**Instance Types:**
- Minimum: n2-standard-4 (Basic tier)
- Recommended: c2-standard-8 (Enhanced tier)
- Ultra performance: c2-standard-60, n2-highmem-128

**Network Features:**
- gVNIC (Google Virtual NIC)
- Tier 1 networking
- Custom machine types with high network bandwidth

## Security Requirements

### User Permissions

```bash
# Required capabilities for eBPF/XDP
# CAP_NET_ADMIN - Network administration
# CAP_SYS_ADMIN - System administration
# CAP_BPF - eBPF operations (kernel 5.8+)

# Check current capabilities
capsh --print

# Required groups
usermod -aG netdev marchproxy
usermod -aG bpf marchproxy  # If group exists
```

### SELinux/AppArmor

```bash
# SELinux configuration (RHEL/CentOS)
setsebool -P domain_can_mmap_files 1
setsebool -P domain_fd_use 1

# AppArmor configuration (Ubuntu/Debian)
sudo aa-disable /usr/bin/marchproxy-proxy  # If profile exists
```

### Firewall Requirements

**Required Ports:**
- 8000/tcp: Manager web interface
- 8080/tcp: Proxy traffic
- 8081/tcp: Proxy metrics
- 5432/tcp: PostgreSQL (internal)
- 9090/tcp: Prometheus (optional)
- 3000/tcp: Grafana (optional)

```bash
# UFW (Ubuntu)
sudo ufw allow 8000/tcp
sudo ufw allow 8080/tcp
sudo ufw allow 8081/tcp

# firewalld (RHEL/CentOS)
sudo firewall-cmd --permanent --add-port=8000/tcp
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=8081/tcp
sudo firewall-cmd --reload
```

## Verification Scripts

### Complete System Check

```bash
#!/bin/bash
# comprehensive_check.sh - Complete MarchProxy requirements verification

echo "=== MarchProxy System Requirements Check ==="
echo

# Function to check command availability
check_command() {
    if command -v $1 &> /dev/null; then
        echo "✅ $1: Available"
        return 0
    else
        echo "❌ $1: Not found"
        return 1
    fi
}

# Function to check kernel feature
check_kernel_feature() {
    if grep -q "CONFIG_$1=y" /boot/config-$(uname -r) 2>/dev/null; then
        echo "✅ $1: Enabled"
        return 0
    else
        echo "❌ $1: Not enabled or unknown"
        return 1
    fi
}

# System information
echo "System Information:"
echo "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo "CPU cores: $(nproc)"
echo "Memory: $(free -h | awk '/^Mem:/{print $2}')"
echo

# Hardware requirements
echo "Hardware Requirements:"
cores=$(nproc)
mem_gb=$(free -g | awk '/^Mem:/{print $2}')

[ $cores -ge 4 ] && echo "✅ CPU cores: $cores (≥4)" || echo "❌ CPU cores: $cores (<4)"
[ $mem_gb -ge 8 ] && echo "✅ Memory: ${mem_gb}GB (≥8GB)" || echo "❌ Memory: ${mem_gb}GB (<8GB)"

# Software dependencies
echo
echo "Software Dependencies:"
check_command python3
check_command go
check_command psql
check_command docker

# Kernel features
echo
echo "Kernel Features:"
check_kernel_feature "BPF"
check_kernel_feature "BPF_SYSCALL"
check_kernel_feature "XDP_SOCKETS"
check_kernel_feature "CGROUPS_BPF"

# eBPF support
echo
echo "eBPF Support:"
if [ -d /sys/kernel/debug/tracing ]; then
    echo "✅ Debug filesystem: Mounted"
else
    echo "❌ Debug filesystem: Not mounted"
fi

if [ "$(cat /proc/sys/kernel/unprivileged_bpf_disabled 2>/dev/null)" = "0" ]; then
    echo "✅ Unprivileged eBPF: Enabled"
else
    echo "⚠️  Unprivileged eBPF: Disabled (may need privileges)"
fi

# Network interface check
echo
echo "Network Interfaces:"
for iface in $(ip link show | grep -E "^[0-9]+:" | cut -d: -f2 | tr -d ' '); do
    if [ "$iface" != "lo" ]; then
        echo "Interface: $iface"
        if ethtool -k $iface 2>/dev/null | grep -q "generic-receive-offload.*on"; then
            echo "  ✅ Hardware acceleration: Available"
        else
            echo "  ⚠️  Hardware acceleration: Limited"
        fi
    fi
done

echo
echo "=== Check Complete ==="
```

### Performance Tier Detection

```bash
#!/bin/bash
# detect_performance_tier.sh - Detect maximum supported performance tier

echo "=== MarchProxy Performance Tier Detection ==="

tier="basic"
max_throughput="1-10 Gbps"

# Check for XDP support
if ip link set dev $(ip route | grep default | awk '{print $5}') xdp obj /dev/null 2>/dev/null; then
    tier="enhanced"
    max_throughput="10-40 Gbps"
    echo "✅ XDP support detected"
else
    echo "❌ XDP not supported"
fi

# Check for DPDK requirements
if [ -d /sys/kernel/iommu_groups ] && [ $(find /sys/kernel/iommu_groups -type d | wc -l) -gt 1 ]; then
    if lspci | grep -i ethernet | head -1 | grep -E "(Intel|Mellanox|Broadcom)" > /dev/null; then
        tier="ultra"
        max_throughput="40-100+ Gbps"
        echo "✅ DPDK support detected"
    fi
else
    echo "❌ DPDK not supported (IOMMU required)"
fi

echo
echo "Detected Performance Tier: $tier"
echo "Expected Throughput: $max_throughput"
echo
echo "Tier Capabilities:"
case $tier in
    "basic")
        echo "- Standard networking with eBPF acceleration"
        echo "- Up to 10 Gbps throughput"
        echo "- Suitable for most deployments"
        ;;
    "enhanced")
        echo "- XDP/AF_XDP zero-copy networking"
        echo "- Up to 40 Gbps throughput"
        echo "- Recommended for high-traffic environments"
        ;;
    "ultra")
        echo "- DPDK kernel bypass"
        echo "- 40-100+ Gbps throughput"
        echo "- Maximum performance for extreme workloads"
        ;;
esac
```

Run these scripts to verify your system meets the requirements for your desired MarchProxy deployment tier.