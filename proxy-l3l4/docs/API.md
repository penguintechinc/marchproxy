# API.md - proxy-l3l4 API Documentation

## Overview

The L3/L4 proxy exposes health check and metrics endpoints for observability and monitoring. Configuration and service registration is managed through the Manager API.

## Health Check Endpoint

### GET /healthz

Health check endpoint for service liveness and readiness probes.

**Port**: 8082 (metrics address, configurable)

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-12-16T10:30:45Z",
  "version": "1.0.0",
  "components": {
    "ebpf": "ready",
    "numa": "initialized",
    "acceleration": "active",
    "tracing": "enabled"
  }
}
```

**Status Codes**:
- `200 OK` - All components healthy
- `503 Service Unavailable` - One or more components unhealthy

**Docker Health Check**:
```bash
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8082/healthz || exit 1
```

## Metrics Endpoint

### GET /metrics

Prometheus-compatible metrics endpoint for monitoring proxy performance and health.

**Port**: 8082 (metrics address, configurable)

**Format**: Prometheus text format

**Key Metrics**:

#### Connection Metrics
- `marchproxy_connections_active` - Number of active connections
- `marchproxy_connections_total` - Total connections processed
- `marchproxy_connections_closed` - Closed connections counter
- `marchproxy_connections_reset` - Reset connections counter

#### Traffic Metrics
- `marchproxy_bytes_received` - Total bytes received
- `marchproxy_bytes_sent` - Total bytes sent
- `marchproxy_packets_received` - Total packets received
- `marchproxy_packets_sent` - Total packets sent
- `marchproxy_packets_dropped` - Dropped packets counter

#### Protocol Metrics
- `marchproxy_tcp_connections` - Active TCP connections
- `marchproxy_udp_connections` - Active UDP connections
- `marchproxy_icmp_packets` - ICMP packets processed

#### Routing Metrics
- `marchproxy_routing_decisions` - Routing decisions made
- `marchproxy_route_errors` - Routing errors encountered
- `marchproxy_multicloud_selection` - Multi-cloud backend selections

#### QoS Metrics
- `marchproxy_qos_packets_shaper` - Packets passed through shaper
- `marchproxy_qos_packets_dropped` - Packets dropped by QoS
- `marchproxy_qos_latency_buckets` - Latency histogram buckets
- `marchproxy_qos_bandwidth_limit` - Current bandwidth limit

#### Acceleration Metrics
- `marchproxy_xdp_packets_processed` - Packets processed by XDP
- `marchproxy_xdp_errors` - XDP processing errors
- `marchproxy_afxdp_packets` - Packets processed by AF_XDP
- `marchproxy_acceleration_mode` - Current acceleration mode (0=disabled, 1=xdp, 2=afxdp)

#### NUMA Metrics
- `marchproxy_numa_cpu_affinity` - NUMA CPU affinity status
- `marchproxy_numa_memory_local` - Locally allocated memory
- `marchproxy_numa_memory_remote` - Remotely allocated memory
- `marchproxy_numa_node_count` - Number of NUMA nodes

#### Zero-Trust Metrics
- `marchproxy_zerotrust_policy_evaluations` - Policy evaluations
- `marchproxy_zerotrust_policy_denials` - Denied requests
- `marchproxy_zerotrust_audit_events` - Audit events logged

**Example Query**:
```bash
curl http://localhost:8082/metrics | grep marchproxy_
```

## Manager API Integration

Service registration and configuration is handled through the Manager API. The proxy registers itself during startup using the cluster API key.

### Registration Flow

1. Proxy starts with `CLUSTER_API_KEY` environment variable
2. Proxy makes HTTP request to Manager: `POST /api/v2/proxy/register`
3. Manager returns cluster configuration
4. Proxy applies configuration and begins forwarding

## Configuration API

Configuration is loaded from:
1. Configuration file (path specified via `--config` flag)
2. Environment variables (override config file)
3. Default values

See [CONFIGURATION.md](./CONFIGURATION.md) for detailed environment variables.

## Service Discovery

The proxy discovers services through the Manager API:

**Endpoint**: `GET /api/v2/services`

**Response**:
```json
{
  "services": [
    {
      "id": "svc-001",
      "name": "web-api",
      "cluster_id": "cluster-1",
      "protocol": "tcp",
      "port_mapping": [
        {
          "source_port": 443,
          "destination": "10.0.0.1:8443"
        }
      ]
    }
  ]
}
```

## Observability

### Distributed Tracing

Traces are exported to Jaeger if enabled. Configure via:
- `JAEGER_ENDPOINT` - Jaeger endpoint URL
- `ENABLE_TRACING` - Enable/disable tracing

Traces include:
- Packet classification timing
- Route selection decisions
- eBPF program execution
- Connection state transitions

### Logging

Logs are output as JSON with fields:
- `timestamp` - ISO8601 timestamp
- `level` - Log level (info, warn, error)
- `msg` - Log message
- `component` - Source component
- `trace_id` - Distributed trace ID

## gRPC Services (Future)

Reserved for future expansion:
- Service definition port: 50051
- Expected services:
  - Policy management
  - Real-time analytics
  - Advanced configuration

## Rate Limiting

Health checks and metrics endpoints have no rate limits. The proxy prioritizes observability to ensure monitoring can always assess proxy health.

## Error Responses

### 400 Bad Request
Invalid request parameters or malformed JSON.

### 401 Unauthorized
Missing or invalid authentication credentials.

### 404 Not Found
Requested resource or endpoint not found.

### 500 Internal Server Error
Internal server error. Check logs for details.

## Security

All endpoints should be protected in production:
- Run behind authentication layer
- Restrict network access to management VLAN
- Use TLS for remote access
- Rotate credentials regularly
