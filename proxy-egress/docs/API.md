# Proxy Egress API Reference

This document describes the internal APIs and endpoints exposed by the MarchProxy Egress Proxy.

## Admin Endpoints

### Health Check

```
GET /healthz
```

Returns the health status of the proxy.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T10:30:00Z",
  "components": {
    "l4_proxy": "healthy",
    "l7_proxy": "healthy",
    "threat_engine": "healthy",
    "extauth": "healthy"
  }
}
```

### Metrics

```
GET /metrics
```

Returns Prometheus-formatted metrics.

**Example Metrics:**
```
# HELP marchproxy_egress_requests_total Total number of requests processed
# TYPE marchproxy_egress_requests_total counter
marchproxy_egress_requests_total{status="allowed"} 12345
marchproxy_egress_requests_total{status="blocked"} 678

# HELP marchproxy_egress_threat_blocks_total Total requests blocked by threat intelligence
# TYPE marchproxy_egress_threat_blocks_total counter
marchproxy_egress_threat_blocks_total{type="ip"} 100
marchproxy_egress_threat_blocks_total{type="domain"} 50
marchproxy_egress_threat_blocks_total{type="url"} 25
```

### Statistics

```
GET /stats
```

Returns detailed statistics in JSON format.

**Response:**
```json
{
  "proxy": {
    "uptime_seconds": 3600,
    "requests_total": 13023,
    "bytes_in": 1234567890,
    "bytes_out": 9876543210
  },
  "threat": {
    "ip_blocks": 100,
    "domain_blocks": 50,
    "url_blocks": 25,
    "cache_hits": 10000,
    "cache_misses": 500
  },
  "tls_intercept": {
    "certs_generated": 150,
    "cache_hits": 1000,
    "cache_misses": 150
  },
  "extauth": {
    "total_requests": 5000,
    "allowed": 4900,
    "denied": 100,
    "errors": 0
  }
}
```

## Threat Intelligence APIs

These endpoints are used internally by the threat engine. Threat rules are primarily managed through the Manager API and synced via gRPC/polling.

### Get Threat Stats

```
GET /api/v1/threat/stats
```

Returns threat intelligence statistics.

**Response:**
```json
{
  "ip_blocker": {
    "rules_count": 1000,
    "cache_size": 100000,
    "blocks_total": 500
  },
  "domain_blocker": {
    "rules_count": 500,
    "wildcard_rules": 50,
    "blocks_total": 200
  },
  "url_matcher": {
    "patterns_count": 100,
    "matches_total": 75
  },
  "dns_cache": {
    "entries": 5000,
    "hits": 10000,
    "misses": 500
  },
  "feed_sync": {
    "last_sync": "2024-01-15T10:30:00Z",
    "sync_count": 100,
    "errors": 0,
    "version": "2024011510"
  }
}
```

### Force Feed Sync

```
POST /api/v1/threat/sync
```

Forces an immediate synchronization of threat feeds from the Manager.

**Response:**
```json
{
  "status": "success",
  "synced_at": "2024-01-15T10:30:00Z",
  "rules_processed": 1500
}
```

## TLS Interception APIs

### Get TLS Intercept Stats

```
GET /api/v1/tls/stats
```

Returns TLS interception statistics.

**Response:**
```json
{
  "enabled": true,
  "mode": "mitm",
  "certs_generated": 150,
  "cache_hits": 1000,
  "cache_misses": 150,
  "intercepted_conns": 5000,
  "passthrough_conns": 500
}
```

### Get Domain Config

```
GET /api/v1/tls/domains
```

Returns per-domain interception configuration.

**Response:**
```json
{
  "domains": {
    "*.google.com": false,
    "example.com": true,
    "internal.company.com": false
  }
}
```

### Get IP Config

```
GET /api/v1/tls/ips
```

Returns per-IP interception configuration.

**Response:**
```json
{
  "ips": {
    "10.0.0.0/8": false,
    "192.168.1.100": true
  }
}
```

## Access Control APIs

### Get Access Control Rules

```
GET /api/v1/access/rules
```

Returns all access control rules.

**Response:**
```json
{
  "rules": [
    {
      "id": "rule-1",
      "target_type": "domain",
      "target_pattern": "api.example.com",
      "mode": "allow",
      "allowed_services": ["service-a", "service-b"],
      "require_auth": true
    }
  ]
}
```

## Envoy Admin (when L7 enabled)

When L7 mode is enabled, Envoy's admin interface is available on the configured admin port (default: 9901).

### Envoy Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /ready` | Envoy readiness |
| `GET /stats` | Envoy statistics |
| `GET /clusters` | Cluster information |
| `GET /config_dump` | Full configuration dump |

## External Authorization Protocol

The proxy implements Envoy's external authorization gRPC protocol:

```protobuf
service Authorization {
  rpc Check(CheckRequest) returns (CheckResponse);
}
```

### Request Flow

1. Envoy sends `CheckRequest` with request attributes
2. ext_authz server queries threat engine and access controller
3. Returns `CheckResponse` with allow/deny decision
4. Envoy forwards or blocks the request accordingly

### Response Headers

On allowed requests:
- `x-marchproxy-checked: true`
- `x-marchproxy-check-time: <timestamp>`

On blocked requests:
- `x-marchproxy-blocked: true`
- `x-marchproxy-block-reason: <reason>`

## Error Codes

| Code | Description |
|------|-------------|
| `BLOCKED_IP` | Request blocked by IP blocklist |
| `BLOCKED_DOMAIN` | Request blocked by domain blocklist |
| `BLOCKED_URL` | Request blocked by URL pattern |
| `AUTH_REQUIRED` | Authentication required |
| `ACCESS_DENIED` | Access denied by access control |
| `SERVICE_NOT_AUTHORIZED` | Service not authorized for destination |

## Rate Limits

When rate limiting is enabled:
- Default: 1000 requests per second per source IP
- Configurable via `rate_limit_rps`
- Returns `429 Too Many Requests` when exceeded
