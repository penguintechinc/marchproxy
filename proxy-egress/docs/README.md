# MarchProxy Egress Proxy

The MarchProxy Egress Proxy provides comprehensive egress traffic control with L4 and L7 capabilities, threat intelligence, and TLS interception.

## Features

### L4 Traffic Control
- TCP/UDP traffic management via eBPF
- Optional hardware acceleration (DPDK, XDP, AF_XDP, SR-IOV)
- High-performance packet filtering at the kernel level

### L7 Traffic Control (via Envoy)
- HTTP/1.1 and HTTP/2 support
- HTTP/3 (QUIC) support (**EXPERIMENTAL**)
- Request/response inspection and modification
- External authorization integration

### Threat Intelligence
- **IP Blocking**: Block individual IPs and CIDR ranges (IPv4/IPv6)
- **Domain Blocking**: Block domains by Host header with wildcard support (*.example.com)
- **URL Pattern Matching**: Regex-based URL blocking using RE2 engine
- **DNS-based Blocking**: Block resolved domains with DNS caching

### TLS Interception
- **MITM Mode**: Dynamic certificate generation for any domain
- **Preconfigured Mode**: Pre-loaded certificates for specific domains
- Per-domain and per-IP interception control

### Access Control
- JWT/Bearer token-based access restrictions
- Service-to-destination authorization
- Per-domain authentication requirements

## Architecture

```
                       ┌─────────────────────────────────────────┐
                       │            proxy-egress                 │
                       │                                         │
Incoming ──────────────┼──► L4 Path (eBPF/XDP)                   │
Traffic                │        │                                │
                       │        ▼                                │
                       │    TCP/UDP passthrough                  │
                       │                                         │
                       │    L7 Path (Envoy)                      │
                       │        │                                │
                       │        ▼                                │
                       │    ┌─────────────┐    ┌──────────────┐ │
                       │    │   Envoy     │◄──►│  ext_authz   │ │
                       │    │ HTTP/1.1/2/3│    │  (Go gRPC)   │ │
                       │    └─────────────┘    └──────────────┘ │
                       │                              │          │
                       │                              ▼          │
                       │                       ┌────────────┐   │
                       │                       │  Threat    │   │
                       │                       │  Engine    │   │
                       │                       └────────────┘   │
                       └─────────────────────────────────────────┘
```

## Quick Start

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `MANAGER_URL` | Manager API URL | Required |
| `CLUSTER_API_KEY` | Cluster API key | Required |
| `ENVOY_ENABLED` | Enable L7 proxy | `false` |
| `ENVOY_HTTP3_ENABLED` | Enable HTTP/3 (EXPERIMENTAL) | `false` |
| `THREAT_ENABLED` | Enable threat intelligence | `true` |
| `TLS_INTERCEPT_ENABLED` | Enable TLS interception | `false` |

### Running with Docker

```bash
docker run -d \
  --name marchproxy-egress \
  -e MANAGER_URL=http://manager:8000 \
  -e CLUSTER_API_KEY=your-api-key \
  -e ENVOY_ENABLED=true \
  -p 8080:8080 \
  -p 10000:10000 \
  marchproxy/proxy-egress:latest
```

### Running with Docker Compose

See the main `docker-compose.yml` in the project root.

## Configuration

See [CONFIGURATION.md](CONFIGURATION.md) for detailed configuration options.

## API

See [API.md](API.md) for the REST API reference.

## Testing

See [TESTING.md](TESTING.md) for testing information.

## Release Notes

See [RELEASE_NOTES.md](RELEASE_NOTES.md) for version history.
