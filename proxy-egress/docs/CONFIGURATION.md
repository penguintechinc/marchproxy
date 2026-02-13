# Proxy Egress Configuration

This document describes all configuration options for the MarchProxy Egress Proxy.

## Configuration Methods

Configuration can be provided via:
1. Configuration file (YAML)
2. Environment variables (prefixed with `MARCHPROXY_`)
3. Command line flags

Priority: CLI flags > Environment variables > Config file > Defaults

## Core Settings

### Manager Connection

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `manager_url` | `MANAGER_URL` | Required | Manager API URL |
| `cluster_api_key` | `CLUSTER_API_KEY` | Required | Cluster API key for authentication |

### Proxy Settings

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `proxy_name` | `MARCHPROXY_PROXY_NAME` | hostname | Name for this proxy instance |
| `hostname` | `MARCHPROXY_HOSTNAME` | auto-detected | Hostname of the proxy |
| `listen_port` | `MARCHPROXY_LISTEN_PORT` | `8080` | L4 proxy listen port |
| `admin_port` | `MARCHPROXY_ADMIN_PORT` | `8081` | Admin/metrics port |
| `log_level` | `MARCHPROXY_LOG_LEVEL` | `INFO` | Log level (DEBUG, INFO, WARN, ERROR) |

### Performance Settings

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `enable_ebpf` | `MARCHPROXY_ENABLE_EBPF` | `true` | Enable eBPF-based packet filtering |
| `enable_metrics` | `MARCHPROXY_ENABLE_METRICS` | `true` | Enable Prometheus metrics |
| `worker_threads` | `MARCHPROXY_WORKER_THREADS` | `0` | Worker threads (0=auto) |

### Hardware Acceleration (Optional)

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `enable_dpdk` | `MARCHPROXY_ENABLE_DPDK` | `false` | Enable DPDK acceleration |
| `enable_xdp` | `MARCHPROXY_ENABLE_XDP` | `false` | Enable XDP acceleration |
| `enable_af_xdp` | `MARCHPROXY_ENABLE_AF_XDP` | `false` | Enable AF_XDP acceleration |
| `enable_sriov` | `MARCHPROXY_ENABLE_SRIOV` | `false` | Enable SR-IOV acceleration |

## L7 Configuration

The L7 proxy uses Envoy for HTTP traffic handling.

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `l7.enabled` | `ENVOY_ENABLED` | `false` | Enable L7 HTTP proxy |
| `l7.envoy_binary` | `ENVOY_BINARY` | `/usr/local/bin/envoy` | Path to Envoy binary |
| `l7.envoy_config_path` | `ENVOY_CONFIG_PATH` | `/app/envoy/bootstrap.yaml` | Envoy config file |
| `l7.envoy_admin_port` | `ENVOY_ADMIN_PORT` | `9901` | Envoy admin port |
| `l7.http_listen_port` | `ENVOY_HTTP_PORT` | `10000` | HTTP listen port |
| `l7.https_listen_port` | `ENVOY_HTTPS_PORT` | `10443` | HTTPS listen port |
| `l7.http3_enabled` | `ENVOY_HTTP3_ENABLED` | `false` | Enable HTTP/3 (**EXPERIMENTAL**) |
| `l7.envoy_log_level` | `ENVOY_LOG_LEVEL` | `info` | Envoy log level |

### HTTP/3 (QUIC) Support

> **WARNING: HTTP/3 support is EXPERIMENTAL**
>
> HTTP/3 (QUIC) support is provided as an experimental feature. It may have:
> - Stability issues under high load
> - Incomplete feature support
> - Performance characteristics that differ from HTTP/1.1 and HTTP/2
>
> Enable with caution in production environments.

To enable HTTP/3:
```yaml
l7:
  enabled: true
  http3_enabled: true  # EXPERIMENTAL
```

Or via environment:
```bash
ENVOY_ENABLED=true
ENVOY_HTTP3_ENABLED=true
```

## Threat Intelligence Configuration

The threat intelligence engine provides multiple blocking mechanisms.

### Global Settings

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `threat.enabled` | `THREAT_ENABLED` | `true` | Enable threat intelligence |

### IP Blocking

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `threat.ip_blocking_enabled` | `THREAT_IP_BLOCKING_ENABLED` | `true` | Enable IP/CIDR blocking |
| `threat.ip_cache_size` | `THREAT_IP_CACHE_SIZE` | `100000` | IP cache size |

### Domain Blocking

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `threat.domain_blocking_enabled` | `THREAT_DOMAIN_BLOCKING_ENABLED` | `true` | Enable domain blocking |
| `threat.wildcard_support` | `THREAT_WILDCARD_SUPPORT` | `true` | Support wildcard patterns |

### URL Matching

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `threat.url_matching_enabled` | `THREAT_URL_MATCHING_ENABLED` | `true` | Enable URL pattern matching |
| `threat.url_match_engine` | `THREAT_URL_MATCH_ENGINE` | `re2` | Regex engine (re2, boost) |

### DNS Cache

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `threat.dns_cache_enabled` | `THREAT_DNS_CACHE_ENABLED` | `true` | Enable DNS caching |
| `threat.dns_positive_ttl` | `THREAT_DNS_POSITIVE_TTL` | `5m` | Positive TTL |
| `threat.dns_negative_ttl` | `THREAT_DNS_NEGATIVE_TTL` | `1m` | Negative TTL |
| `threat.dns_cache_size` | `THREAT_DNS_CACHE_SIZE` | `50000` | DNS cache size |
| `threat.dns_upstream` | `THREAT_DNS_UPSTREAM` | `8.8.8.8:53,1.1.1.1:53` | Upstream DNS servers |

### Feed Synchronization

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `threat.sync_mode` | `THREAT_SYNC_MODE` | `both` | Sync mode (grpc, poll, both) |
| `threat.sync_poll_interval` | `THREAT_SYNC_POLL_INTERVAL` | `60s` | Poll interval |
| `threat.sync_grpc_endpoint` | `THREAT_SYNC_GRPC_ENDPOINT` | - | gRPC streaming endpoint |

## TLS Interception Configuration

TLS interception allows deep packet inspection of HTTPS traffic.

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `tls_intercept.enabled` | `TLS_INTERCEPT_ENABLED` | `false` | Enable TLS interception |
| `tls_intercept.mode` | `TLS_INTERCEPT_MODE` | `mitm` | Mode (mitm, preconfigured) |
| `tls_intercept.ca_cert_path` | `TLS_INTERCEPT_CA_CERT` | `/app/certs/ca.crt` | CA certificate path |
| `tls_intercept.ca_key_path` | `TLS_INTERCEPT_CA_KEY` | `/app/certs/ca.key` | CA private key path |
| `tls_intercept.cert_cache_size` | `TLS_INTERCEPT_CACHE_SIZE` | `10000` | Certificate cache size |

### MITM Mode

In MITM mode, the proxy dynamically generates certificates for each destination domain, signed by the configured CA.

Requirements:
1. Generate a CA certificate and key
2. Install the CA certificate as trusted on client systems
3. Configure the CA paths in the proxy

### Preconfigured Mode

In preconfigured mode, certificates for specific domains must be pre-loaded via the Manager API.

## External Authorization

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `extauth.enabled` | `EXTAUTH_ENABLED` | `true` | Enable ext_authz server |
| `extauth.port` | `EXTAUTH_PORT` | `9002` | gRPC server port |
| `extauth.host` | `EXTAUTH_HOST` | `127.0.0.1` | gRPC server host |

## Access Control

| Option | Env Variable | Default | Description |
|--------|--------------|---------|-------------|
| `access_control.enabled` | `ACCESS_CONTROL_ENABLED` | `false` | Enable access control |
| `access_control.default_require_auth` | `ACCESS_CONTROL_DEFAULT_REQUIRE_AUTH` | `false` | Require auth by default |
| `access_control.default_allow` | `ACCESS_CONTROL_DEFAULT_ALLOW` | `true` | Allow by default |

## Example Configuration

```yaml
manager_url: "http://manager:8000"
cluster_api_key: "your-cluster-api-key"

l7:
  enabled: true
  http3_enabled: false  # Keep disabled unless needed (EXPERIMENTAL)

threat:
  enabled: true
  ip_blocking_enabled: true
  domain_blocking_enabled: true
  url_matching_enabled: true
  sync_mode: "both"

tls_intercept:
  enabled: false
  mode: "mitm"
  ca_cert_path: "/app/certs/ca.crt"
  ca_key_path: "/app/certs/ca.key"

access_control:
  enabled: true
  default_require_auth: false
  default_allow: true
```

## Health Check

The proxy provides health endpoints:
- `GET /healthz` - Basic health check
- `GET /metrics` - Prometheus metrics

## Ports Reference

| Port | Protocol | Description |
|------|----------|-------------|
| 8080 | TCP | L4 proxy (default) |
| 8081 | HTTP | Admin/metrics |
| 10000 | HTTP | L7 HTTP (when enabled) |
| 10443 | HTTPS | L7 HTTPS (when enabled) |
| 9901 | HTTP | Envoy admin (when L7 enabled) |
| 9002 | gRPC | External authorization |
