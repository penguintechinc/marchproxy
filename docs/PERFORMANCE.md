# MarchProxy Performance Tuning Guide

**Version:** 1.0.0
**Last Updated:** 2025-12-12
**Target Audience:** DevOps, SREs, Performance Engineers

## Table of Contents

1. [Performance Architecture](#performance-architecture)
2. [Benchmarking](#benchmarking)
3. [Tuning Guidelines](#tuning-guidelines)
4. [Hardware Acceleration](#hardware-acceleration)
5. [Monitoring Performance](#monitoring-performance)
6. [Capacity Planning](#capacity-planning)
7. [Optimization Checklist](#optimization-checklist)

---

## Performance Architecture

### Multi-Tier Processing Model

MarchProxy uses a multi-tier performance architecture where different packet types are routed to optimal processing paths:

```
Packet Arrival (100%)
    ↓
Tier 1: Hardware Acceleration (100+ Gbps)
├─ DPDK: kernel bypass (5% of packets)
├─ SR-IOV: virtualization (5% of packets)
    ↓
Tier 2: XDP (40+ Gbps)
├─ Driver-level processing
├─ Early packet classification (40% of packets)
    ↓
Tier 3: AF_XDP (20+ Gbps)
├─ Zero-copy socket I/O
├─ User-space processing (35% of packets)
    ↓
Tier 4: eBPF (10+ Gbps)
├─ Kernel-level filtering
├─ Complex rules (14% of packets)
    ↓
Tier 5: Go Application (1+ Gbps)
├─ Business logic
├─ Policy evaluation (1% of packets)
```

### Component Performance Targets

| Component | Metric | Target | Notes |
|-----------|--------|--------|-------|
| **Proxy L7 (Envoy)** | Throughput | 40+ Gbps | HTTP/HTTPS/gRPC |
| | Requests/sec | 1M+ | Per instance |
| | Latency (p99) | <10ms | End-to-end |
| **Proxy L3/L4 (Go)** | Throughput | 100+ Gbps | TCP/UDP/ICMP |
| | Packets/sec | 10M+ | Per instance |
| | Latency (p99) | <1ms | Kernel-bypass |
| **API Server** | Requests/sec | 10K+ | Per instance |
| | Query latency (p99) | <10ms | Database queries |
| **WebUI** | Initial load | <2s | First contentful paint |
| | Lighthouse score | 90+ | Performance score |

---

## Benchmarking

### Load Testing Tools

#### 1. HTTP Load Testing (L7)

**Apache Bench:**
```bash
# 10,000 requests, 100 concurrent connections
ab -n 10000 -c 100 http://localhost:80/

# HTTPS with keep-alive disabled
ab -k -n 10000 -c 100 https://localhost:443/
```

**wrk (Modern alternative):**
```bash
# Download: https://github.com/wg/wrk
wrk -t12 -c400 -d30s http://localhost:80/

# With custom Lua script
wrk -t12 -c400 -d30s -s script.lua http://localhost:80/
```

#### 2. Network Load Testing (L3/L4)

**iperf3 (TCP/UDP):**
```bash
# Server
docker-compose exec proxy-l3l4 iperf3 -s

# Client
iperf3 -c localhost -p 5201 -P 8 -t 60

# UDP test
iperf3 -c localhost -p 5201 -u -b 10G
```

**netcat (Packet loss):**
```bash
# Generate packet stream
dd if=/dev/zero bs=1M count=1000 | nc localhost 8081

# Measure throughput
time nc localhost 8081 < /dev/zero
```

#### 3. gRPC Load Testing

**ghz:**
```bash
# Install: go install github.com/bojand/ghz@latest

# Simple RPC
ghz --insecure \
  --proto api/service.proto \
  --call mypackage.MyService/Method \
  localhost:8080

# Streaming RPC
ghz --insecure \
  --proto api/service.proto \
  --call mypackage.MyService/Stream \
  -n 100000 \
  localhost:8080
```

### Benchmark Commands

```bash
#!/bin/bash
# save as benchmark.sh

echo "=== MarchProxy Performance Benchmark ==="
echo

# HTTP throughput
echo "1. HTTP Throughput (L7 Proxy - Envoy)"
wrk -t8 -c100 -d30s http://localhost:80/ | grep "Requests/sec"
echo

# HTTPS throughput
echo "2. HTTPS Throughput (L7 Proxy - Envoy)"
wrk -t8 -c100 -d30s https://localhost:443/ | grep "Requests/sec"
echo

# TCP throughput
echo "3. TCP Throughput (L3/L4 Proxy - Go)"
iperf3 -c localhost -t 30 | grep "Sender"
echo

# Latency measurement
echo "4. HTTP Latency (p99)"
ab -n 1000 -c 1 http://localhost:80/ | grep "99%"
echo

# Concurrent connections
echo "5. Concurrent Connections"
wrk -t8 -c1000 -d30s http://localhost:80/ | grep "Requests/sec"
echo

# API response time
echo "6. API Response Time"
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8000/api/healthz
echo
```

### Baseline Benchmark Results (v1.0.0)

```
Test Environment:
- AWS c5.2xlarge (8 vCPU, 16GB RAM)
- Linux kernel 5.15
- Docker Compose deployment

Results:
─────────────────────────────────────────────────────
HTTP Throughput (1 connection):        ~25,000 req/s
HTTP Throughput (100 connections):     ~50,000 req/s
HTTP Latency (p50):                    2ms
HTTP Latency (p99):                    8ms
─────────────────────────────────────────────────────
TCP Throughput (single client):        ~5 Gbps
TCP Throughput (10 parallel clients):  ~15 Gbps
TCP Latency (p99):                     <1ms (eBPF)
─────────────────────────────────────────────────────
API Latency (p99):                     5ms
Prometheus Query Latency (p99):        3ms
─────────────────────────────────────────────────────
WebUI Load Time (Lighthouse):          90
Bundle Size (gzipped):                 350KB
─────────────────────────────────────────────────────
```

---

## Tuning Guidelines

### Operating System

```bash
# Increase file descriptor limits
sysctl -w fs.file-max=2097152
sysctl -w fs.nr_open=2097152

# Increase network buffer sizes
sysctl -w net.core.rmem_max=134217728
sysctl -w net.core.wmem_max=134217728
sysctl -w net.ipv4.tcp_rmem='4096 87380 67108864'
sysctl -w net.ipv4.tcp_wmem='4096 65536 67108864'

# Increase TCP backlog
sysctl -w net.ipv4.tcp_max_syn_backlog=10000
sysctl -w net.core.somaxconn=10000

# Enable TCP fast open
sysctl -w net.ipv4.tcp_fastopen=3

# Persist changes
echo 'net.core.rmem_max=134217728' >> /etc/sysctl.conf
sysctl -p
```

### Docker Compose

```yaml
services:
  proxy-l7:
    # Increase memory and CPU
    mem_limit: 4g
    cpus: "2"
    # Performance options
    cap_add:
      - NET_ADMIN
      - SYS_ADMIN
      - SYS_RESOURCE
    # Increase file descriptors
    ulimits:
      nofile: 1048576
      nproc: 1048576

  proxy-l3l4:
    mem_limit: 4g
    cpus: "2"
    cap_add:
      - NET_ADMIN
      - SYS_ADMIN
      - SYS_RESOURCE
    ulimits:
      nofile: 1048576
      nproc: 1048576

  api-server:
    # Database connection pooling
    environment:
      - DATABASE_POOL_SIZE=20
      - DATABASE_MAX_OVERFLOW=40
    mem_limit: 2g
    cpus: "1"
```

### Proxy Configuration

**Envoy (L7) - Tuning:**
```yaml
# bootstrap.yaml
admin:
  address:
    socket_address:
      address: 127.0.0.1
      port_value: 9901

# Connection pooling
http_connection_manager:
  codec_type: AUTO
  max_connection_duration: 3600s
  idle_timeout: 60s
  http_filters:
    - name: envoy.filters.network.http_connection_manager
      typed_config:
        '@type': type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
        common_http_protocol_options:
          idle_timeout: 60s
          max_requests: 10000
        upstream_http_protocol_options:
          auto_sni: true
```

**Go (L3/L4) - Environment Variables:**
```bash
# Connection pooling
GO_MAX_PROCS=8            # CPU affinity
GOMAXPROCS=8              # Go goroutine scheduler

# Network tuning
TCP_SOCKET_BUFFER=2097152 # 2MB
UDP_SOCKET_BUFFER=2097152

# Memory
GOMEMLIMIT=3500MiB       # Go 1.19+ memory limit
```

### Database

```bash
# PostgreSQL performance tuning
sudo -u postgres psql <<EOF
ALTER SYSTEM SET shared_buffers = '4GB';
ALTER SYSTEM SET effective_cache_size = '12GB';
ALTER SYSTEM SET maintenance_work_mem = '1GB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
SELECT pg_reload_conf();
EOF

# Rebuild indexes
docker-compose exec postgres psql -U marchproxy -c "REINDEX DATABASE marchproxy;"
```

### Redis

```bash
# redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru
save ""                    # Disable RDB
appendonly yes             # Enable AOF
appendfsync everysec       # AOF sync policy
```

---

## Hardware Acceleration

### XDP (eXpress Data Path)

**Requirements:**
- Linux kernel 5.10+ (5.15+ recommended)
- Network interface with XDP support (Intel, Mellanox, etc.)

**Enable XDP:**
```bash
# Check XDP support
ethtool -i eth0 | grep driver

# Load XDP program
docker-compose exec proxy-l3l4 ./scripts/enable-xdp.sh

# Verify
docker-compose exec proxy-l3l4 bpftool prog list
```

### AF_XDP (Zero-Copy)

**Requirements:**
- Linux kernel 5.10+
- NUMA-capable CPU recommended

**Enable AF_XDP:**
```bash
# Update .env
ENABLE_AF_XDP=true
ENABLE_NUMA=true

# Restart containers
docker-compose restart proxy-l3l4

# Check status
curl http://localhost:8082/metrics | grep af_xdp
```

### DPDK (Kernel Bypass)

**Requirements:**
- Linux kernel 5.15+
- IOMMU support (VT-d for Intel)
- 2MB or 1GB hugepages

**Configuration:**
```bash
# Configure hugepages
sudo sh -c 'echo 1024 > /sys/kernel/mm/hugepages/hugepages-2048kB/nr_hugepages'

# Verify
grep HugePages /proc/meminfo

# Enable DPDK in proxy
ENABLE_DPDK=true docker-compose restart proxy-l3l4
```

### Performance Comparison

| Method | Gbps | Latency | Complexity | License |
|--------|------|---------|-----------|---------|
| Standard | 1 | <10ms | Low | Community |
| eBPF | 10 | <5ms | Medium | Community |
| XDP | 40 | <1ms | High | Enterprise |
| AF_XDP | 20 | <0.5ms | High | Enterprise |
| DPDK | 100+ | <0.2ms | Very High | Enterprise |

---

## Monitoring Performance

### Key Metrics

**L7 Proxy (Envoy):**
```prometheus
# Requests per second
rate(envoy_http_ingress_http_requests_total[1m])

# Latency distribution
histogram_quantile(0.99, envoy_http_request_duration_ms)

# Active connections
envoy_http_connections_active

# Request errors
envoy_http_ingress_http_requests_total{response_code=~"5.."}
```

**L3/L4 Proxy (Go):**
```prometheus
# Bytes transferred
rate(marchproxy_tcp_bytes_total[1m])
rate(marchproxy_udp_bytes_total[1m])

# Packets per second
rate(marchproxy_tcp_packets_total[1m])
rate(marchproxy_udp_packets_total[1m])

# Connection count
marchproxy_tcp_connections_active

# Latency
histogram_quantile(0.99, marchproxy_latency_ms)
```

**API Server:**
```prometheus
# Request latency
histogram_quantile(0.99, http_request_duration_seconds)

# Database query time
histogram_quantile(0.99, db_query_duration_seconds)

# Cache hit rate
rate(cache_hits_total[5m]) / rate(cache_requests_total[5m])
```

### Grafana Dashboard

Pre-configured dashboards available at:
```
http://localhost:3000/d/marchproxy-performance
```

**Key panels:**
- Throughput (requests/sec, bytes/sec)
- Latency (p50, p95, p99)
- Error rate
- CPU/Memory usage
- Active connections
- Cache hit rate

### Real-Time Monitoring

```bash
#!/bin/bash
# Monitor real-time metrics
watch -n 1 'docker stats --no-stream'

# Or with top-like view
ctop

# Monitor network I/O
nethogs -d 1

# Monitor disk I/O
iostat -x 1
```

---

## Capacity Planning

### Sizing Formula

```
Required Proxies = (Peak Traffic in Gbps) / (Throughput per Proxy)

Example for 100 Gbps peak traffic:
- L7 Proxy (40 Gbps per instance): 100 / 40 = 2.5 → 3 instances
- L3/L4 Proxy (100 Gbps per instance): 100 / 100 = 1 instance

For high availability (N+1 redundancy):
- L7 Proxy: 3 + 1 = 4 instances
- L3/L4 Proxy: 1 + 1 = 2 instances
```

### Resource Requirements

**Per L7 Proxy Instance:**
- vCPU: 2-4
- RAM: 2-4 GB
- Network: 10 Gbps NIC minimum
- Storage: 100 MB (configuration)

**Per L3/L4 Proxy Instance:**
- vCPU: 4-8
- RAM: 4-8 GB
- Network: 25 Gbps NIC (for 100+ Gbps)
- Storage: 50 MB (configuration)

**Central Management:**
- vCPU: 2-4
- RAM: 8-16 GB
- Storage: 100-500 GB (databases)
- Network: 1 Gbps minimum

---

## Optimization Checklist

### Pre-Deployment

- [ ] Upgrade Linux kernel to 5.15+
- [ ] Enable hugepages for DPDK (if using)
- [ ] Tune OS network parameters (see Tuning section)
- [ ] Set up monitoring (Prometheus, Grafana)
- [ ] Benchmark hardware baseline

### Deployment

- [ ] Enable XDP acceleration
- [ ] Configure NUMA affinity (if multi-socket)
- [ ] Set appropriate resource limits
- [ ] Enable connection pooling
- [ ] Configure database connection pool

### Operational

- [ ] Monitor key metrics continuously
- [ ] Weekly performance reviews
- [ ] Monthly capacity planning
- [ ] Document custom tunings
- [ ] Test failover procedures quarterly

### Ongoing Optimization

- [ ] Analyze slow query logs (API)
- [ ] Review Prometheus metrics for anomalies
- [ ] Implement caching strategies
- [ ] Optimize database indexes
- [ ] Profile CPU usage (go pprof)

---

## Troubleshooting Performance Issues

### High CPU Usage

```bash
# Profile Go application
curl http://localhost:6060/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Check for hot functions
(pprof) top

# Check goroutine count
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

### High Memory Usage

```bash
# Heap profile
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Monitor memory growth
(pprof) top -cum

# Check for memory leaks
curl http://localhost:6060/debug/pprof/allocs > allocs.prof
```

### Slow Requests

```bash
# Check database query performance
docker-compose exec postgres psql -U marchproxy -c "
SELECT query, mean_time, max_time, calls
FROM pg_stat_statements
ORDER BY mean_time DESC LIMIT 10;
"

# Enable query logging
ALTER SYSTEM SET log_min_duration_statement = 100;  -- 100ms
```

### Packet Loss

```bash
# Monitor network errors
ethtool -S eth0 | grep -i error

# Check NIC RX/TX queues
ethtool -l eth0
ethtool -L eth0 rx 16 tx 16  # Increase queues

# Verify XDP drop counters
docker-compose exec proxy-l3l4 bpftool stat
```

---

## References

- [Linux kernel performance](https://www.kernel.org/doc/html/latest/admin-guide/sysctl/net.html)
- [Envoy performance tuning](https://www.envoyproxy.io/docs/envoy/latest/faq/performance/how_to_benchmarking)
- [Go performance optimization](https://pkg.go.dev/runtime/pprof)
- [PostgreSQL tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [XDP resources](https://github.com/xdp-project/xdp-tutorial)

---

**For deployment guidance, see:** [DEPLOYMENT.md](DEPLOYMENT.md)
**For troubleshooting, see:** [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
