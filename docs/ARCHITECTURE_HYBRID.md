# MarchProxy Hybrid Architecture (v1.0.0)

## Overview

MarchProxy v1.0.0 introduces a revolutionary **4-container hybrid architecture** combining separate proxies for different network layers with a centralized API-driven control plane. This architecture exceeds cloud-native ALB capabilities with enterprise-grade features and unmatched performance.

```
┌─────────────────────────────────────────────────────────────────┐
│                        MarchProxy Hybrid                         │
│                      v1.0.0 Architecture                         │
└─────────────────────────────────────────────────────────────────┘

                        ┌──────────────┐
                        │   Web UI     │
                        │  (React 18)  │
                        │ :3000        │
                        └──────────────┘
                               │
                               ▼
                        ┌──────────────┐
                        │ API Server   │
                        │ (FastAPI)    │
                        │ :8000        │
                        └──────────────┘
                               │
                ┌──────────────┬─────────────┐
                ▼              ▼             ▼
         ┌──────────┐  ┌──────────┐  ┌──────────┐
         │ Proxy L7 │  │Proxy L3/L4│ │ xDS gRPC │
         │ (Envoy)  │  │   (Go)    │ │ :18000   │
         │ :80,443  │  │ :8081     │ │          │
         └──────────┘  └──────────┘  └──────────┘

        Database    Cache        Observability
        (Postgres)  (Redis)      (Jaeger, Loki)
```

## Container Architecture

### 1. API Server (FastAPI)

**Purpose**: Central control plane and REST API
- Configuration management for all proxies
- xDS gRPC control plane for Envoy (L7)
- License validation and feature gating
- Database schema management with Alembic
- User authentication and RBAC

**Technology**:
- FastAPI (Python 3.11+)
- SQLAlchemy 2.0 with asyncpg
- Alembic for migrations
- Go-based xDS server (optional)

**Ports**:
- 8000: REST API
- 18000: xDS gRPC

**Responsibilities**:
- REST API for CRUD operations
- xDS configuration distribution
- Proxy registration and monitoring
- License validation integration
- Configuration generation and caching

### 2. Web UI (React + Node.js)

**Purpose**: Modern administrative dashboard
- Service and cluster management
- Observability visualization
- Enterprise feature configuration
- Real-time monitoring

**Technology**:
- React 18 with TypeScript
- Vite build tool
- Material-UI or Ant Design
- Zustand state management
- Node.js 20 backend

**Ports**:
- 3000: HTTP/HTTPS

**Features**:
- Real-time dashboards with WebSocket
- Traffic shaping configuration
- Multi-cloud routing visualization
- Zero-trust policy editor
- Embedded Jaeger tracing viewer

**Theme**:
- Background: Dark Grey (#1E1E1E, #2C2C2C)
- Primary: Navy Blue (#1E3A8A, #0F172A)
- Accent: Gold (#FFD700, #FDB813)

### 3. Proxy L7 (Envoy)

**Purpose**: Application-layer proxy for HTTP/HTTPS/gRPC
- HTTP/HTTPS/HTTP2 protocol handling
- gRPC support with multiplexing
- WebSocket upgrade handling
- TLS termination with SNI routing
- Custom WASM filters for auth/licensing

**Technology**:
- Envoy Proxy (latest stable)
- XDP integration for early packet filtering
- Custom WASM filters in Rust

**Ports**:
- 80: HTTP
- 443: HTTPS/TLS
- 8080: HTTP/2
- 9901: Admin interface

**Performance Targets**:
- Throughput: 40+ Gbps for HTTP/HTTPS
- Requests/sec: 1M+ for gRPC
- Latency: p99 < 10ms

**Features**:
- Dynamic configuration via xDS
- Circuit breaker pattern
- Advanced routing (path, host, header-based)
- Request/response transformation
- Distributed tracing integration
- Rate limiting and DDoS protection

### 4. Proxy L3/L4 (Go)

**Purpose**: Transport-layer proxy for TCP/UDP with enterprise features
- Raw TCP/UDP packet forwarding
- Advanced traffic shaping (QoS)
- Multi-cloud intelligent routing
- Zero-trust security enforcement
- NUMA-aware processing for performance

**Technology**:
- Go 1.21+
- eBPF for kernel-level filtering
- XDP/AF_XDP for hardware acceleration
- OpenTelemetry for distributed tracing

**Ports**:
- 8081: Proxy listen port
- 8082: Admin/Metrics port

**Performance Targets**:
- Throughput: 100+ Gbps for TCP/UDP
- Packets/sec: 10M+ pps
- Latency: p99 < 1ms

**Performance Stack (Tiered)**:
```
XDP (40+ Gbps)          ← Hardware-accelerated, wire-speed filtering
    ↓
AF_XDP (20+ Gbps)       ← Zero-copy sockets
    ↓
eBPF (5+ Gbps)          ← Kernel filtering
    ↓
Go App (1+ Gbps)        ← Complex business logic
```

**Enterprise Features**:
1. **Advanced Traffic Shaping**:
   - Per-service bandwidth limits (ingress/egress)
   - Priority queues (P0-P3) with SLAs
   - Token bucket algorithm for burst handling
   - DSCP/ECN marking

2. **Multi-Cloud Routing**:
   - Health probes with RTT measurement
   - Latency-based routing
   - Cost-optimized routing
   - Geo-proximity routing
   - Active-passive failover

3. **Deep Observability**:
   - OpenTelemetry distributed tracing
   - Custom metrics (histograms, heatmaps)
   - Request/response header logging
   - Sampling strategies

4. **Zero-Trust Security**:
   - Mutual TLS enforcement
   - Per-request RBAC via OPA
   - Immutable audit logging
   - Certificate rotation

## Supporting Services

### Database & Caching

**PostgreSQL 15**:
- Primary datastore for all configuration
- User management and RBAC
- Service/cluster definitions
- License caching
- Audit logs

**Redis 7**:
- Session caching
- Rate limit counters
- Configuration caching
- Distributed locks
- Queue for async tasks

### Observability Stack

**Jaeger**:
- Distributed tracing
- Trace visualization
- Service dependency graphs
- Latency analysis

**Prometheus**:
- Metrics collection
- Time-series database
- Alerting rules evaluation

**Grafana**:
- Dashboard visualization
- Alert management
- Multi-datasource support

**ELK Stack** (Elasticsearch, Logstash, Kibana):
- Centralized log aggregation
- Log searching and filtering
- Dashboard creation
- Compliance reporting

**Loki + Promtail**:
- Alternative log aggregation
- Log-based alerting
- Resource-efficient storage

### Networking

**Network Configuration**:
- Bridge network (172.20.0.0/16)
- Service discovery via Docker DNS
- Volume-based certificate sharing

## Traffic Flow

### Incoming Client Request (Ingress)

```
External Client
    │
    ├─── HTTP/HTTPS ────► Proxy L7 (Envoy)
    │                      │
    │                    XDP ─► Rate limit/DDoS check
    │                      │
    │                      ├─► WASM Filter (Auth)
    │                      │
    │                      └─► Route to backend
    │
    └─── TCP/UDP ────────► Proxy L3/L4 (Go)
                             │
                           XDP ─► Wire-speed classification
                             │
                             ├─► Traffic Shaping
                             │
                             ├─► Multi-Cloud Router
                             │
                             └─► Backend/Internet
```

### Control Plane Updates

```
Admin (Web UI)
    │
    ▼
API Server (FastAPI)
    │
    ├─► Alembic Migration (DB Schema)
    │
    ├─► Configuration Caching (Redis)
    │
    ├─► xDS Snapshot Update
    │      │
    │      └──────────────────────┐
    │                             │
    ▼                             ▼
Proxy L3/L4 ◄──────────────── Proxy L7 (Envoy)
(Pulls config)                (xDS client)
```

## Data Flow

### Configuration Update Sequence

1. **Admin Action**: User updates cluster configuration in Web UI
2. **API Validation**: FastAPI validates input and applies business rules
3. **Database Persistence**: Alembic migration tracks schema changes
4. **Cache Invalidation**: Redis cache keys updated
5. **xDS Update**: Control plane generates new configuration
6. **Distribution**:
   - Envoy: Receives via gRPC xDS push
   - Go Proxy: Polls API every 30-90 seconds (randomized)
7. **Proxy Application**: Configuration applied with zero-downtime restart

### Monitoring & Observability

```
Proxies (L7 & L3/L4)
    │
    ├─► Metrics ────────────► Prometheus ──► Grafana
    │
    ├─► Traces ─────────────► Jaeger ──────► Web UI
    │
    └─► Logs ───────────────► Logstash ───► Elasticsearch ──► Kibana
                                │
                                └──────────► Loki
```

## Performance Characteristics

### L7 (Envoy) Performance

| Metric | Target | Typical |
|--------|--------|---------|
| Throughput | 40+ Gbps | 45 Gbps |
| Requests/sec | 1M+ | 1.2M |
| Latency p50 | <1ms | 0.8ms |
| Latency p99 | <10ms | 8ms |
| Connection Setup | <5ms | 3ms |

### L3/L4 (Go) Performance

| Metric | Target | Typical |
|--------|--------|---------|
| Throughput | 100+ Gbps | 120 Gbps |
| Packets/sec | 10M+ | 12M |
| Latency p50 | <0.1ms | 0.08ms |
| Latency p99 | <1ms | 0.8ms |
| Connection Setup | <2ms | 1.5ms |

### API Server Performance

| Metric | Target |
|--------|--------|
| Requests/sec | 10K+ |
| xDS Update Propagation | <100ms |
| Database Query (p99) | <10ms |
| Configuration Caching Hit Rate | >95% |

## Deployment Options

### Docker Compose (Development/Testing)

Recommended for:
- Local development
- Testing and validation
- Small deployments
- Learning the system

**Components**: 14 containers, ~2GB RAM, requires docker-compose

### Kubernetes (Production)

Recommended for:
- Production deployments
- High availability (HA)
- Multi-region setup
- Auto-scaling

**Requirements**:
- Kubernetes 1.24+
- Helm 3.0+
- Persistent storage
- Service mesh support (optional)

### Bare Metal (Maximum Performance)

Recommended for:
- Extreme performance requirements
- Hardware acceleration (DPDK, SR-IOV)
- Dedicated infrastructure
- Custom networking

**Requirements**:
- Linux kernel 5.8+
- Compatible NIC for XDP
- NUMA-aware systems
- Custom networking setup

## Security Architecture

### Network Security

- **Mutual TLS**: Service-to-service mTLS with certificate rotation
- **Network Isolation**: Docker bridge network with internal communication
- **Rate Limiting**: XDP-based rate limiting at wire speed
- **DDoS Protection**: XDP early packet filtering

### Application Security

- **Authentication**: JWT tokens with refresh capability
- **Authorization**: RBAC with cluster isolation
- **Encryption**: AES-256 for sensitive data
- **Audit Logging**: Immutable append-only logs

### Enterprise Security

- **SAML/OAuth2**: Enterprise SSO integration
- **SCIM**: Automated user provisioning
- **Zero-Trust**: Per-request policy evaluation via OPA
- **Compliance**: SOC2, HIPAA, PCI-DSS reporting

## Technology Stack Summary

| Component | Technology | Version |
|-----------|-----------|---------|
| API Server | FastAPI | 0.100+ |
| Web UI | React | 18.0+ |
| Proxy L7 | Envoy | Latest |
| Proxy L3/L4 | Go | 1.21+ |
| Database | PostgreSQL | 15+ |
| Cache | Redis | 7+ |
| Tracing | Jaeger | Latest |
| Metrics | Prometheus | Latest |
| Visualization | Grafana | Latest |
| Logs | ELK/Loki | Latest |

## Future Enhancements

### v1.1.0 (Planned)

- WebAssembly plugin system
- GraphQL API alongside REST
- Advanced ML-based anomaly detection
- Service mesh integration (Istio, Linkerd)

### v1.2.0 (Planned)

- Full IPv6 support
- QUIC/HTTP3 in L3/L4 proxy
- Hardware crypto acceleration
- Multi-region geo-replication

### v2.0.0 (Planned)

- Kubernetes operator
- GitOps integration
- Full eBPF-based networking stack
- Edge computing support

## References

- [Envoy Proxy Documentation](https://www.envoyproxy.io/docs)
- [FastAPI Documentation](https://fastapi.tiangolo.com/)
- [React Documentation](https://react.dev/)
- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/otel/)
- [xDS Protocol](https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol)
- [eBPF Documentation](https://ebpf.io/)
- [XDP Tutorial](https://github.com/xdp-project/xdp-tutorial)
