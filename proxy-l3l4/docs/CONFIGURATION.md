# CONFIGURATION.md - proxy-l3l4 Configuration Guide

## Overview

proxy-l3l4 is configured through:
1. YAML/TOML configuration file
2. Environment variables (override config file)
3. Default values

Configuration flow: Defaults → Config File → Environment Variables

## Configuration File

Load config file:
```bash
./proxy-l3l4 --config /etc/marchproxy/config.yaml
```

### Example Configuration

```yaml
# Basic proxy configuration
listen_port: 8081
metrics_addr: ":8082"
metrics_namespace: "marchproxy"

# Manager integration
manager_endpoint: "http://manager:8000"
cluster_api_key: "${CLUSTER_API_KEY}"
cluster_id: "cluster-1"

# NUMA configuration
enable_numa: true
numa_node_isolation: true
numa_memory_alignment: 2048

# Acceleration settings
enable_xdp: true
xdp_interface: "eth0"
enable_afxdp: false
xdp_mode: "skb"  # Options: skb, drv, hw

# QoS configuration
enable_qos: true
qos_default_priority: 3
qos_max_bandwidth: "1gbps"

# Observability
enable_tracing: true
jaeger_endpoint: "http://jaeger:14268/api/traces"
trace_sample_rate: 0.1
log_level: "info"

# Zero-trust security
enable_zerotrust: true
opa_policy_path: "/etc/marchproxy/policies"
zerotrust_audit_log: "/var/log/marchproxy/audit.log"

# Advanced routing
enable_multicloud: true
router_algorithm: "latency"  # Options: latency, cost, random
health_check_interval: 30s
health_check_timeout: 5s

# Connection settings
tcp_timeout: 900s
udp_timeout: 120s
connection_limit: 100000
```

## Environment Variables

Environment variables override config file settings.

### Core Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_PORT` | 8081 | Proxy listener port |
| `METRICS_ADDR` | :8082 | Metrics server address |
| `METRICS_NAMESPACE` | marchproxy | Prometheus metric namespace |
| `CONFIG_PATH` | (none) | Path to configuration file |

### Manager Integration

| Variable | Default | Description |
|----------|---------|-------------|
| `MANAGER_ENDPOINT` | http://manager:8000 | Manager API endpoint |
| `CLUSTER_API_KEY` | (required) | Cluster API key for registration |
| `CLUSTER_ID` | cluster-1 | Cluster identifier |
| `REGISTRATION_INTERVAL` | 300s | Registration heartbeat interval |

### NUMA Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_NUMA` | true | Enable NUMA support |
| `NUMA_NODE_ISOLATION` | true | Isolate NUMA nodes |
| `NUMA_MEMORY_ALIGNMENT` | 2048 | Memory alignment in bytes |
| `NUMA_CPU_PINNING` | false | Pin threads to NUMA nodes |

### XDP Acceleration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_XDP` | true | Enable XDP acceleration |
| `XDP_INTERFACE` | eth0 | Network interface for XDP |
| `XDP_MODE` | skb | XDP mode: skb, drv, or hw |
| `XDP_PROGRAM_PATH` | ./ebpf/xdp.o | Path to compiled XDP program |
| `XDP_DETACH_ON_EXIT` | true | Detach XDP on shutdown |

**XDP Modes**:
- `skb` - Software fast-path (default, compatible)
- `drv` - Driver-native mode (higher performance)
- `hw` - Hardware offload (requires NIC support)

### AF_XDP Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_AFXDP` | false | Enable AF_XDP acceleration |
| `AFXDP_INTERFACE` | eth0 | Network interface for AF_XDP |
| `AFXDP_QUEUE_ID` | 0 | Queue ID for AF_XDP socket |
| `AFXDP_FRAME_HEADROOM` | 0 | Frame headroom bytes |
| `AFXDP_RX_FRAMES` | 2048 | RX ring frame count |
| `AFXDP_TX_FRAMES` | 2048 | TX ring frame count |

### QoS Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_QOS` | true | Enable QoS traffic shaping |
| `QOS_DEFAULT_PRIORITY` | 3 | Default traffic priority (1-4) |
| `QOS_MAX_BANDWIDTH` | unlimited | Max bandwidth limit |
| `QOS_QUEUE_DEPTH` | 10000 | Packet queue depth |
| `QOS_SHAPER_ALGORITHM` | wrr | Scheduler: wrr, drr, or cbq |

**Priority Levels**:
1. Critical (system traffic)
2. High (real-time applications)
3. Normal (standard traffic)
4. Low (background traffic)

### Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_TRACING` | false | Enable distributed tracing |
| `JAEGER_ENDPOINT` | http://jaeger:14268 | Jaeger collector endpoint |
| `TRACE_SAMPLE_RATE` | 0.1 | Trace sampling rate (0.0-1.0) |
| `LOG_LEVEL` | info | Logging level: debug, info, warn, error |
| `METRICS_PUBLISH_INTERVAL` | 30s | Metrics publish interval |

### Zero-Trust Security

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_ZEROTRUST` | true | Enable zero-trust policies |
| `OPA_POLICY_PATH` | /etc/marchproxy/policies | OPA policy directory |
| `ZEROTRUST_AUDIT_LOG` | /var/log/marchproxy/audit.log | Audit log path |
| `ZEROTRUST_AUDIT_LEVEL` | info | Audit level: all, failures, or none |
| `ZEROTRUST_CACHE_TTL` | 300s | Policy evaluation cache TTL |

### Multi-Cloud Routing

| Variable | Default | Description |
|----------|---------|-------------|
| `ENABLE_MULTICLOUD` | true | Enable multi-cloud routing |
| `ROUTER_ALGORITHM` | latency | Routing algorithm |
| `HEALTH_CHECK_INTERVAL` | 30s | Health check interval |
| `HEALTH_CHECK_TIMEOUT` | 5s | Health check timeout |
| `BACKEND_RETRY_COUNT` | 3 | Retry count for failures |

**Router Algorithms**:
- `latency` - Route based on response latency
- `cost` - Route based on cloud provider costs
- `random` - Random backend selection
- `weighted` - Weighted round-robin

### Connection Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `TCP_TIMEOUT` | 900s | TCP connection timeout |
| `UDP_TIMEOUT` | 120s | UDP connection timeout |
| `ICMP_TIMEOUT` | 60s | ICMP timeout |
| `CONNECTION_LIMIT` | 100000 | Max concurrent connections |
| `GRACEFUL_SHUTDOWN_TIMEOUT` | 30s | Graceful shutdown timeout |

## Docker Environment Variables

### Docker Compose

```yaml
services:
  proxy-l3l4:
    image: proxy-l3l4:latest
    environment:
      LISTEN_PORT: "8081"
      METRICS_ADDR: ":8082"
      MANAGER_ENDPOINT: "http://manager:8000"
      CLUSTER_API_KEY: ${CLUSTER_API_KEY}
      CLUSTER_ID: "cluster-1"
      ENABLE_XDP: "true"
      XDP_INTERFACE: "eth0"
      ENABLE_ZEROTRUST: "true"
      LOG_LEVEL: "info"
    ports:
      - "8081:8081"
      - "8082:8082"
    volumes:
      - /etc/marchproxy/policies:/etc/marchproxy/policies
      - /var/log/marchproxy:/var/log/marchproxy
    cap_add:
      - NET_ADMIN
      - SYS_RESOURCE
      - SYS_PTRACE
    network_mode: "host"
```

### Required Capabilities

For eBPF and network features:
- `NET_ADMIN` - Network administration
- `SYS_RESOURCE` - Resource limits
- `SYS_PTRACE` - Process tracing (debug mode)

## eBPF Configuration

### eBPF Programs

eBPF programs must be compiled before deployment:

```bash
# Build eBPF programs
clang -O2 -target bpf \
    -c internal/acceleration/xdp/xdp.c \
    -o internal/acceleration/xdp/xdp.o

# Load program
sudo ip link set dev eth0 xdp obj xdp.o sec xdp
```

### eBPF Program Location

| Mode | Path | Description |
|------|------|-------------|
| XDP | /etc/marchproxy/ebpf/xdp.o | XDP kernel program |
| AF_XDP | /etc/marchproxy/ebpf/afxdp.o | AF_XDP program |

### eBPF Maps

Configuration maps accessible from userspace:

**Connection State Map**:
```bash
bpftool map dump name conn_state
```

**Routing Decision Map**:
```bash
bpftool map dump name routing_table
```

**QoS Priority Map**:
```bash
bpftool map dump name qos_priorities
```

## Configuration Validation

Validate configuration before deployment:

```bash
./proxy-l3l4 --config config.yaml --validate
```

Test with dry-run:
```bash
./proxy-l3l4 --config config.yaml --dry-run
```

## Reloading Configuration

Hot-reload configuration (select settings):

```bash
# Signal to reload config
kill -HUP $(pgrep proxy-l3l4)
```

Reloadable settings:
- Log level
- Metrics namespace
- QoS policies
- Zero-trust policies
- Health check intervals

Non-reloadable (require restart):
- Listen port
- Manager endpoint
- XDP interface
- NUMA settings

## Performance Tuning

### Recommended Settings for High Throughput

```yaml
# Production config for 100Gbps+
listen_port: 8081
metrics_addr: ":8082"

enable_numa: true
numa_node_isolation: true

enable_xdp: true
xdp_mode: "hw"

enable_qos: true
qos_max_bandwidth: "100gbps"
qos_queue_depth: 65536

enable_zerotrust: true
zerotrust_cache_ttl: 3600s

connection_limit: 1000000
```

### Low-Latency Settings

```yaml
# Low-latency configuration
enable_tracing: true
trace_sample_rate: 1.0
enable_numa: true
enable_xdp: true
qos_default_priority: 2
```

## Security Considerations

### Protected Configuration Values

Never commit to version control:
- `CLUSTER_API_KEY` - Use environment variable or secrets manager
- `JAEGER_ENDPOINT` credentials
- TLS certificates paths

### Configuration File Permissions

```bash
# Restrict config file access
chmod 600 /etc/marchproxy/config.yaml
chown marchproxy:marchproxy /etc/marchproxy/config.yaml
```

### Policy Directory Permissions

```bash
# OPA policies should be readable by proxy user
chmod 750 /etc/marchproxy/policies
chown root:marchproxy /etc/marchproxy/policies
```

## Configuration Examples

### Development Environment

```yaml
listen_port: 8081
metrics_addr: ":8082"
enable_xdp: false
enable_zerotrust: false
enable_tracing: true
trace_sample_rate: 1.0
log_level: "debug"
```

### Production Environment

```yaml
listen_port: 8081
metrics_addr: ":8082"
enable_xdp: true
xdp_mode: "hw"
enable_zerotrust: true
enable_tracing: true
trace_sample_rate: 0.1
log_level: "warn"
connection_limit: 1000000
```

### Edge/IoT Environment

```yaml
listen_port: 8081
metrics_addr: ":8082"
enable_xdp: true
xdp_mode: "skb"
enable_zerotrust: false
enable_tracing: false
enable_numa: false
log_level: "info"
```
