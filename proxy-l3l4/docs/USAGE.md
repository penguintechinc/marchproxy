# USAGE.md - proxy-l3l4 Operational Guide

## Quick Start

### Docker Quick Start

```bash
# Start with default configuration
docker run -d --name proxy-l3l4 \
    -e CLUSTER_API_KEY=test-key-12345 \
    -p 8081:8081 \
    -p 8082:8082 \
    marchproxy:latest

# Check health
curl http://localhost:8082/healthz

# View metrics
curl http://localhost:8082/metrics
```

### Local Development

```bash
# Build from source
git clone https://github.com/penguintech/marchproxy.git
cd proxy-l3l4
go build -o proxy-l3l4 ./cmd/proxy

# Run with default configuration
./proxy-l3l4

# Run with custom config
./proxy-l3l4 --config /etc/marchproxy/config.yaml
```

## Deployment

### Docker Compose Deployment

```yaml
version: '3.8'

services:
  proxy-l3l4:
    image: marchproxy:latest
    container_name: proxy-l3l4
    environment:
      LISTEN_PORT: "8081"
      METRICS_ADDR: ":8082"
      MANAGER_ENDPOINT: "http://manager:8000"
      CLUSTER_API_KEY: "${CLUSTER_API_KEY}"
      CLUSTER_ID: "cluster-1"
      ENABLE_XDP: "true"
      XDP_INTERFACE: "eth0"
      LOG_LEVEL: "info"
    ports:
      - "8081:8081"
      - "8082:8082"
    volumes:
      - /etc/marchproxy/config.yaml:/etc/marchproxy/config.yaml:ro
      - /etc/marchproxy/policies:/etc/marchproxy/policies:ro
      - /var/log/marchproxy:/var/log/marchproxy
    cap_add:
      - NET_ADMIN
      - SYS_RESOURCE
    network_mode: "host"
    restart: always
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
```

Start services:
```bash
CLUSTER_API_KEY=test-key docker-compose up -d
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: proxy-l3l4
  namespace: marchproxy
spec:
  selector:
    matchLabels:
      app: proxy-l3l4
  template:
    metadata:
      labels:
        app: proxy-l3l4
    spec:
      hostNetwork: true
      hostPID: true
      containers:
      - name: proxy-l3l4
        image: marchproxy:latest
        securityContext:
          privileged: true
          capabilities:
            add:
              - NET_ADMIN
              - SYS_RESOURCE
        env:
        - name: CLUSTER_API_KEY
          valueFrom:
            secretKeyRef:
              name: marchproxy-secret
              key: cluster-api-key
        - name: CLUSTER_ID
          value: "k8s-cluster-1"
        - name: MANAGER_ENDPOINT
          value: "http://manager:8000"
        - name: LOG_LEVEL
          value: "info"
        ports:
        - name: proxy
          containerPort: 8081
        - name: metrics
          containerPort: 8082
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8082
          initialDelaySeconds: 5
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8082
          initialDelaySeconds: 3
          periodSeconds: 10
        volumeMounts:
        - name: policies
          mountPath: /etc/marchproxy/policies
          readOnly: true
        - name: logs
          mountPath: /var/log/marchproxy
      volumes:
      - name: policies
        configMap:
          name: marchproxy-policies
      - name: logs
        emptyDir: {}
```

Deploy to Kubernetes:
```bash
kubectl create namespace marchproxy
kubectl create secret generic marchproxy-secret \
    --from-literal=cluster-api-key=$CLUSTER_API_KEY \
    -n marchproxy
kubectl apply -f proxy-l3l4-daemonset.yaml
```

## Monitoring

### Health Checks

Check proxy health:
```bash
curl -s http://localhost:8082/healthz | jq .
```

Response indicates component status:
```json
{
  "status": "healthy",
  "components": {
    "ebpf": "ready",
    "numa": "initialized",
    "acceleration": "active"
  }
}
```

### Metrics Collection

Prometheus scrape configuration:
```yaml
scrape_configs:
  - job_name: 'marchproxy-l3l4'
    static_configs:
      - targets: ['localhost:8082']
```

Key metrics to monitor:
```bash
# Active connections
curl -s http://localhost:8082/metrics | grep marchproxy_connections_active

# Traffic throughput (bytes/sec)
curl -s http://localhost:8082/metrics | grep marchproxy_bytes_sent

# Packet drop rate
curl -s http://localhost:8082/metrics | grep marchproxy_packets_dropped

# Routing errors
curl -s http://localhost:8082/metrics | grep marchproxy_route_errors

# XDP performance
curl -s http://localhost:8082/metrics | grep marchproxy_xdp_packets_processed
```

### Alerting

Recommended alerts:

```yaml
groups:
  - name: marchproxy
    interval: 30s
    rules:
    - alert: ProxyUnhealthy
      expr: up{job="marchproxy-l3l4"} == 0
      for: 2m
      labels:
        severity: critical

    - alert: HighPacketDropRate
      expr: rate(marchproxy_packets_dropped[5m]) > 1000
      for: 5m
      labels:
        severity: warning

    - alert: HighConnectionCount
      expr: marchproxy_connections_active > 900000
      for: 5m
      labels:
        severity: warning

    - alert: XDPErrors
      expr: rate(marchproxy_xdp_errors[5m]) > 10
      for: 5m
      labels:
        severity: warning
```

## Log Management

### Log Levels

Set log level via configuration:
```bash
./proxy-l3l4 --config config.yaml
```

In config.yaml:
```yaml
log_level: "info"  # Options: debug, info, warn, error
```

Or environment variable:
```bash
LOG_LEVEL=debug ./proxy-l3l4
```

### Log Locations

- **Container**: Logs to stdout/stderr
- **Host**: Configure volume mount to `/var/log/marchproxy`
- **Audit logs**: `/var/log/marchproxy/audit.log` (zero-trust enabled)

### Log Collection

Example Filebeat configuration:
```yaml
filebeat.inputs:
- type: log
  enabled: true
  paths:
    - /var/log/marchproxy/audit.log
  tags: ["marchproxy", "audit"]

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
```

## Troubleshooting

### Service Won't Start

**Check logs**:
```bash
docker logs proxy-l3l4
```

**Common issues**:
- Port 8081/8082 already in use
- Missing CLUSTER_API_KEY
- Network connectivity to Manager

**Solutions**:
```bash
# Find process on port 8081
lsof -i :8081
kill -9 <pid>

# Check Manager connectivity
curl http://manager:8000/api/v2/health
```

### High CPU Usage

**Check processes**:
```bash
top -p $(pgrep proxy-l3l4)
```

**Reduce load**:
- Lower trace sample rate: `TRACE_SAMPLE_RATE=0.01`
- Disable NUMA isolation: `NUMA_NODE_ISOLATION=false`
- Check for routing loops

### Memory Leak Detection

**Monitor memory**:
```bash
# Watch memory growth
watch -n 1 'docker stats proxy-l3l4'
```

**Enable memory profiling**:
```bash
# In config
enable_profiling: true
profile_addr: ":6060"

# Check profile
go tool pprof http://localhost:6060/debug/pprof/heap
```

### XDP Issues

**Check XDP status**:
```bash
# Inside container
ip link show dev eth0
cat /sys/kernel/debug/tracing/trace_pipe
```

**Disable XDP if failing**:
```bash
ENABLE_XDP=false ./proxy-l3l4
```

**Verify XDP compatibility**:
```bash
# Check kernel version
uname -r  # Should be 5.8+

# Check NIC driver support
ethtool -i eth0
```

### Connection Tracking Issues

**Check connection count**:
```bash
curl -s http://localhost:8082/metrics | \
    grep marchproxy_connections
```

**Increase connection limit**:
```bash
CONNECTION_LIMIT=500000 ./proxy-l3l4
```

## Performance Tuning

### For High Throughput (100+ Gbps)

```yaml
# config.yaml
listen_port: 8081
enable_xdp: true
xdp_mode: "hw"
enable_numa: true
numa_node_isolation: true
enable_qos: true
qos_max_bandwidth: "100gbps"
connection_limit: 1000000
log_level: "warn"
enable_tracing: false
```

### For Low Latency

```yaml
# config.yaml
listen_port: 8081
enable_xdp: true
xdp_mode: "hw"
enable_numa: true
enable_zerotrust: true
qos_default_priority: 2
log_level: "warn"
```

### For Memory Efficiency

```yaml
# config.yaml
enable_numa: false
enable_xdp: true
connection_limit: 50000
log_level: "error"
enable_tracing: false
metrics_publish_interval: 60s
```

## Backup & Recovery

### Configuration Backup

```bash
# Backup configuration
tar czf /backup/marchproxy-config.tar.gz \
    /etc/marchproxy/config.yaml \
    /etc/marchproxy/policies/

# Restore configuration
tar xzf /backup/marchproxy-config.tar.gz -C /
```

### Audit Log Retention

```bash
# Archive old logs
find /var/log/marchproxy -name "audit*.log" -mtime +30 \
    -exec gzip {} \; \
    -exec mv {}.gz /backup/ \;

# Compress logs
logrotate /etc/logrotate.d/marchproxy
```

## Maintenance

### Regular Tasks

**Daily**:
- Monitor health and metrics
- Check error logs
- Review audit logs

**Weekly**:
- Verify backup success
- Update threat intelligence (OPA policies)
- Performance baseline trending

**Monthly**:
- Security updates (kernel, dependencies)
- Policy review and updates
- Capacity planning assessment

### Updates & Patching

```bash
# Check for updates
docker pull marchproxy:latest

# Apply patch release (e.g., 1.0.1)
docker pull marchproxy:v1.0.1
docker stop proxy-l3l4
docker rm proxy-l3l4
docker run -d --name proxy-l3l4 \
    -e CLUSTER_API_KEY=$CLUSTER_API_KEY \
    -v /etc/marchproxy:/etc/marchproxy \
    marchproxy:v1.0.1

# Verify health
curl http://localhost:8082/healthz
```

## Advanced Usage

### Custom Routing Policies

Update routing algorithm:
```bash
# Change to cost-based routing
curl -X POST http://manager:8000/api/v2/clusters/cluster-1/config \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"router_algorithm": "cost"}'
```

### QoS Policy Configuration

Modify QoS rules:
```bash
# Create QoS profile
curl -X POST http://manager:8000/api/v2/qos/profiles \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "video-streaming",
        "priority": 3,
        "max_bandwidth": "10gbps",
        "burst_size": "100mb"
    }'
```

### Zero-Trust Policy Management

Deploy OPA policies:
```bash
# Upload policy
curl -X PUT http://manager:8000/api/v2/policies/deny-default \
    -H "Authorization: Bearer $TOKEN" \
    --data-binary @policies/deny-default.opa
```

## Support & Documentation

- **Full Docs**: /docs (included in repository)
- **API Reference**: [API.md](./API.md)
- **Configuration**: [CONFIGURATION.md](./CONFIGURATION.md)
- **Testing**: [TESTING.md](./TESTING.md)
- **Release Info**: [RELEASE_NOTES.md](./RELEASE_NOTES.md)

## Getting Help

- Check logs: `docker logs proxy-l3l4`
- View metrics: `curl http://localhost:8082/metrics`
- Check health: `curl http://localhost:8082/healthz`
- Report issues: GitHub Issues
- Contact support: support@penguintech.io
