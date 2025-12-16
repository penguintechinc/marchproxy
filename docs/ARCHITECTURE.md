# MarchProxy v1.0.0 Architecture Documentation

**Version:** 1.0.0
**Last Updated:** 2025-12-12
**Status:** Production Ready

## Table of Contents

- [Executive Summary](#executive-summary)
- [System Architecture Overview](#system-architecture-overview)
- [Component Architecture](#component-architecture)
- [Data Flow Diagrams](#data-flow-diagrams)
- [Performance Architecture](#performance-architecture)
- [Security Architecture](#security-architecture)
- [Enterprise Features Architecture](#enterprise-features-architecture)
- [Database Schema](#database-schema)
- [Container Communication](#container-communication)
- [Scalability & High Availability](#scalability--high-availability)
- [Network Topology](#network-topology)

## Executive Summary

MarchProxy v1.0.0 is a hybrid, enterprise-grade dual proxy system featuring a 4-container architecture designed to exceed cloud-native ALB capabilities. The system separates concerns between management (API Server + WebUI), application-layer proxying (Envoy L7), and network-layer proxying (Go L3/L4).

### Key Architectural Decisions

- **Dual Proxy Model**: Separate L7 (HTTP/HTTPS/gRPC) and L3/L4 (TCP/UDP/ICMP) processing tiers
- **Hybrid Stack**: Envoy for L7 with custom WASM filters + Go for L3/L4 with eBPF/XDP optimization
- **FastAPI API Server**: Async, xDS control plane, SQLAlchemy ORM, Alembic migrations
- **React WebUI**: Modern dashboard with real-time updates via WebSocket
- **Enterprise Features**: Traffic shaping (QoS), multi-cloud routing, observability, zero-trust security

---

## System Architecture Overview

### 4-Container Deployment Model

| Container | Purpose | Technology | Port |
|-----------|---------|-----------|------|
| **api-server** | REST API, xDS control plane, config management | FastAPI, SQLAlchemy, PostgreSQL | 8000, 18000 |
| **webui** | Modern management dashboard | React 18, TypeScript, Material-UI | 3000 |
| **proxy-l7** | Application-layer (HTTP/HTTPS/gRPC) | Envoy 1.27+, WASM filters, XDP | 80, 443, 8080, 9901 |
| **proxy-l3l4** | Network-layer (TCP/UDP/ICMP) | Go, eBPF, XDP, AF_XDP, NUMA | 8081, 8082 |

## Overview

MarchProxy is a high-performance dual proxy system designed for enterprise data centers, providing comprehensive control over both ingress (incoming external traffic) and egress (outgoing internet traffic). The architecture employs a centralized management plane with distributed data plane proxies, enabling multi-tier packet processing from standard networking through eBPF, XDP, AF_XDP, to full kernel bypass with DPDK.

### Key Architecture Principles

1. **Separation of Concerns**: Management plane (configuration) separate from data plane (packet processing)
2. **Stateless Proxies**: Horizontal scalability without session affinity requirements
3. **Multi-Tier Performance**: Automatic workload classification for optimal processing path
4. **Zero-Downtime Updates**: Configuration hot-reload and certificate rotation without connection drops
5. **Cluster Isolation**: Enterprise multi-cluster support with strict boundary enforcement

## System Architecture

### High-Level Architecture Diagram

```
┌───────────────────────────────────────────────────────────────────────────┐
│                         MARCHPROXY v1.0.0                                 │
│                    Dual Proxy Architecture                                │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ┌─────────────────────┐              ┌─────────────────────┐           │
│  │  External Clients   │              │  Internal Services  │           │
│  │  (Internet)         │              │  (Data Center)      │           │
│  └──────────┬──────────┘              └──────────┬──────────┘           │
│             │                                    │                       │
│             │ HTTPS/mTLS                         │ Egress Traffic       │
│             ▼                                    ▼                       │
│  ┌──────────────────────────────────────────────────────────────┐       │
│  │                   PROXY LAYER                                │       │
│  ├──────────────────────────────────────────────────────────────┤       │
│  │                                                              │       │
│  │  ┌────────────────────┐      ┌─────────────────────┐        │       │
│  │  │  Proxy-Ingress     │      │   Proxy-Egress      │        │       │
│  │  │  (Reverse Proxy)   │      │   (Forward Proxy)   │        │       │
│  │  ├────────────────────┤      ├─────────────────────┤        │       │
│  │  │ • HTTP/HTTPS       │      │ • HTTP/HTTPS        │        │       │
│  │  │ • WebSocket        │      │ • TCP/UDP/ICMP      │        │       │
│  │  │ • mTLS Auth        │      │ • mTLS Auth         │        │       │
│  │  │ • Load Balancing   │      │ • Service-to-Web    │        │       │
│  │  │ • SSL Termination  │      │ • JWT/Token Auth    │        │       │
│  │  │                    │      │                     │        │       │
│  │  │ Performance Stack: │      │ Performance Stack:  │        │       │
│  │  │ XDP → eBPF → Go    │      │ XDP → eBPF → Go     │        │       │
│  │  │                    │      │                     │        │       │
│  │  │ Ports:             │      │ Ports:              │        │       │
│  │  │ :80 (HTTP)         │      │ :8080 (Proxy)       │        │       │
│  │  │ :443 (HTTPS/mTLS)  │      │ :8081 (Admin)       │        │       │
│  │  │ :8082 (Admin)      │      │ :8081/metrics       │        │       │
│  │  └────────────────────┘      └─────────────────────┘        │       │
│  └──────────────────┬───────────────────┬──────────────────────┘       │
│                     │                   │                              │
│                     │ Config Pull       │ Config Pull                  │
│                     │ Heartbeat         │ Heartbeat                    │
│                     ▼                   ▼                              │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │                MANAGEMENT LAYER                              │      │
│  ├──────────────────────────────────────────────────────────────┤      │
│  │                                                              │      │
│  │  ┌────────────────────────────────────────────────────────┐  │      │
│  │  │              Manager (py4web + pydal)                  │  │      │
│  │  ├────────────────────────────────────────────────────────┤  │      │
│  │  │  • REST API (port 8000)                                │  │      │
│  │  │  • Web Dashboard UI                                    │  │      │
│  │  │  • Configuration Management                            │  │      │
│  │  │  • User Authentication (2FA, SAML, OAuth2)             │  │      │
│  │  │  • License Validation & Enforcement                    │  │      │
│  │  │  • mTLS Certificate Authority                          │  │      │
│  │  │  • Proxy Registration & Health Monitoring              │  │      │
│  │  │  • Cluster Management (Enterprise)                     │  │      │
│  │  │  • Centralized Logging Configuration                   │  │      │
│  │  └────────────────────────────────────────────────────────┘  │      │
│  └──────────────────┬───────────────────────────────────────────┘      │
│                     │                                                  │
│                     ▼                                                  │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │                  DATA LAYER                                  │      │
│  ├──────────────────────────────────────────────────────────────┤      │
│  │  ┌────────────────┐  ┌────────────────┐  ┌───────────────┐  │      │
│  │  │  PostgreSQL    │  │     Redis      │  │   Secrets     │  │      │
│  │  │  (Primary DB)  │  │   (Cache)      │  │   (Vault)     │  │      │
│  │  ├────────────────┤  ├────────────────┤  ├───────────────┤  │      │
│  │  │ • Users        │  │ • Sessions     │  │ • Certs       │  │      │
│  │  │ • Clusters     │  │ • Config Cache │  │ • API Keys    │  │      │
│  │  │ • Services     │  │ • Rate Limits  │  │ • JWT Secrets │  │      │
│  │  │ • Mappings     │  │                │  │               │  │      │
│  │  │ • Certificates │  │                │  │               │  │      │
│  │  │ • Proxies      │  │                │  │               │  │      │
│  │  │ • Audit Logs   │  │                │  │               │  │      │
│  │  └────────────────┘  └────────────────┘  └───────────────┘  │      │
│  └──────────────────────────────────────────────────────────────┘      │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────┐      │
│  │             OBSERVABILITY LAYER                              │      │
│  ├──────────────────────────────────────────────────────────────┤      │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │      │
│  │  │  Prometheus  │  │   Grafana    │  │  ELK Stack   │       │      │
│  │  │  (Metrics)   │  │ (Dashboards) │  │   (Logs)     │       │      │
│  │  └──────────────┘  └──────────────┘  └──────────────┘       │      │
│  │  ┌──────────────┐  ┌──────────────┐                         │      │
│  │  │   Jaeger     │  │ AlertManager │                         │      │
│  │  │  (Tracing)   │  │  (Alerts)    │                         │      │
│  │  └──────────────┘  └──────────────┘                         │      │
│  └──────────────────────────────────────────────────────────────┘      │
└───────────────────────────────────────────────────────────────────────────┘
```

### Container Architecture

MarchProxy runs as a multi-container application with the following services:

| Container | Purpose | Technology | Ports |
|-----------|---------|------------|-------|
| **manager** | Configuration & API | Python 3.12 + py4web + pydal | 8000 (HTTP) |
| **proxy-ingress** | Reverse proxy (external → internal) | Go 1.21 + eBPF/XDP | 80, 443, 8082 |
| **proxy-egress** | Forward proxy (internal → external) | Go 1.21 + eBPF/XDP | 8080, 8081 |
| **postgres** | Primary database | PostgreSQL 15 | 5432 (internal) |
| **redis** | Cache & sessions | Redis 7 | 6379 (internal) |
| **prometheus** | Metrics collection | Prometheus | 9090 |
| **grafana** | Metrics visualization | Grafana | 3000 |
| **elasticsearch** | Log storage | Elasticsearch 8 | 9200 (internal) |
| **logstash** | Log processing | Logstash 8 | 5044 (internal) |
| **kibana** | Log visualization | Kibana 8 | 5601 |
| **jaeger** | Distributed tracing | Jaeger | 16686 |
| **alertmanager** | Alert routing | AlertManager | 9093 |

## Component Architecture

### Container 1: API Server (FastAPI)

**Purpose:** REST API, xDS control plane, configuration management, licensing, authentication

**Technology Stack:**
- FastAPI 0.104+ (async ASGI)
- SQLAlchemy 2.0 ORM with Alembic migrations
- PostgreSQL database (pydal bridge for v0.1.x compatibility)
- Python 3.11+
- Go microservice for xDS (using envoyproxy/go-control-plane)

**Key Responsibilities:**

1. **REST API Endpoints**
   - Authentication & authorization (JWT-based)
   - Cluster CRUD operations
   - Service mapping management
   - Proxy registration and heartbeat
   - Certificate lifecycle management
   - License validation

2. **xDS Control Plane**
   - Listener Discovery Service (LDS) - defines Envoy listeners
   - Route Discovery Service (RDS) - defines virtual hosts and routes
   - Cluster Discovery Service (CDS) - defines upstream clusters
   - Endpoint Discovery Service (EDS) - defines endpoints in clusters
   - Push-based configuration updates (gRPC streaming)

3. **Core Services**
   - Authentication (JWT-based with HS256/RS256)
   - License validation (via license.penguintech.io)
   - Configuration translation to xDS format
   - Database schema management (Alembic)
   - Health checks (/healthz) and metrics (/metrics)

**Database Schema:**
- Users (authentication, RBAC)
- Clusters (logical groupings with API keys)
- Services (backend definitions)
- Proxies (registration, status, metrics)
- Certificates (mTLS CA, server, client)
- AuditLogs (immutable append-only)
- Mappings (traffic routing rules)

### Container 2: WebUI (React + Node.js)

**Purpose:** Modern management dashboard with real-time monitoring and configuration

**Technology Stack:**
- React 18 + TypeScript
- Material-UI or Ant Design (Material-UI recommended)
- Vite (build tool, <2s cold start)
- React Query + Zustand (state management)
- Node.js 20 + Express (production hosting)
- WebSocket for real-time updates

**Theme:**
- Background: Dark Grey (#1E1E1E, #2C2C2C)
- Primary: Navy Blue (#1E3A8A, #0F172A)
- Accent: Gold (#FFD700, #FDB813)

**Key Pages:**
1. **Dashboard** - Real-time metrics, proxy status, traffic overview
2. **Clusters** - Create/edit/delete clusters, manage API keys
3. **Services** - Define services, configure authentication, set routes
4. **Proxies** - Register proxies, monitor health, view logs
5. **Traffic Shaping** - Define QoS policies (L7 and L3/L4)
6. **Multi-Cloud Routing** - Configure cloud routes, failover strategies
7. **Observability** - Tracing viewer, metrics dashboard, sampling config
8. **Zero-Trust** - OPA policy editor, audit logs, compliance reports
9. **Certificates** - Manage mTLS CA, certificates, key rotation
10. **Audit Log** - Immutable audit trail with filtering and search

**Performance Targets:**
- Initial load: <2s
- Lighthouse score: 90+
- Bundle size: <500KB gzipped
- Real-time updates: <500ms latency

### Container 3: Proxy L7 (Envoy)

**Purpose:** Application-layer (L7) traffic handling with advanced filtering

**Technology Stack:**
- Envoy Proxy 1.27+ (CNCF graduated)
- WASM Filters (Rust, proxy-wasm spec)
- XDP early packet classification
- Bootstrap configuration + dynamic xDS

**Key Features:**

1. **Listener Configuration**
   - HTTP listener (port 80) with automatic HTTPS redirect
   - HTTPS listener (port 443) with mTLS termination (ECC P-384)
   - gRPC listener (port 8080) for application protocols
   - Admin listener (port 9901) for metrics and debugging

2. **Custom WASM Filters**
   - **Authentication Filter**: JWT/Base64 token validation
   - **License Filter**: Enterprise feature gating
   - **Metrics Filter**: Custom MarchProxy metrics collection
   - Built in Rust using proxy-wasm spec

3. **Performance Tiers**
   - **XDP**: Early packet classification, DDoS protection (40+ Gbps)
   - **Envoy Core**: HTTP/gRPC/WebSocket processing with mTLS termination
   - **Metrics**: Prometheus format on /stats endpoint (port 9901)

4. **xDS Integration**
   - Static xDS cluster pointing to api-server:18000
   - Dynamic resource updates via gRPC streaming
   - Snapshot-based configuration caching
   - <100ms propagation time for config updates

### Container 4: Proxy L3/L4 (Enhanced Go)

**Purpose:** Network-layer (L3/L4) traffic handling with maximum performance

**Technology Stack:**
- Go 1.21+
- eBPF (Linux kernel programs, compiled with LLVM)
- XDP, AF_XDP for packet acceleration
- NUMA-aware processing
- OpenTelemetry for distributed tracing

**Key Features:**

1. **Multi-Tier Performance Stack**
   ```
   Packet Arrival (100%)
      ↓
   XDP Layer          (40+ Gbps) - 95% of packets
   AF_XDP Layer       (20+ Gbps) - 4% of packets
   eBPF Layer         (10+ Gbps) - 0.9% of packets
   Go Application     (1+ Gbps)  - 0.1% of packets (complex logic)
   ```

2. **Traffic Shaping (QoS)**
   - Priority queues: P0 (<1ms), P1 (<10ms), P2 (<100ms), P3 (best effort)
   - Token bucket rate limiting per-service
   - DSCP/ECN marking for network coordination
   - Per-service bandwidth limits (ingress/egress)

3. **Multi-Cloud Routing**
   - Route tables for AWS, GCP, Azure
   - Health probes (TCP, HTTP, ICMP)
   - RTT measurement and cost analysis
   - Automatic failover (active-active, active-passive)
   - Routing algorithms: latency-based, cost-optimized, geo-proximity, weighted RR

4. **Observability**
   - OpenTelemetry distributed tracing
   - Jaeger/Zipkin exporter
   - Custom metrics (histograms, heatmaps)
   - Request/response header logging
   - Sampling strategies (always, rate-based, error-based)

5. **Zero-Trust Security**
   - Mutual TLS enforcement (all service-to-service)
   - Per-request RBAC evaluation
   - OPA (Open Policy Agent) integration
   - Immutable audit logging (append-only)
   - Automated certificate rotation

6. **NUMA Optimization**
   - Topology detection (all nodes, cores, caches)
   - CPU affinity binding per socket
   - Local memory allocation
   - Per-NUMA-node packet processing queues

## Component Details

### Manager (Python/py4web)

**Responsibilities:**
- Centralized configuration management for all proxies
- User authentication and authorization (local, SAML, OAuth2, SCIM)
- License validation with license.penguintech.io
- Certificate Authority for mTLS (ECC P-384 self-signed CA)
- Cluster management and isolation (Enterprise)
- Proxy registration and health monitoring
- Configuration distribution via REST API
- Audit logging and compliance reporting

**Technology Stack:**
- **Framework**: py4web (async WSGI framework)
- **ORM**: pydal (database abstraction layer)
- **Database**: PostgreSQL 15 (default, configurable)
- **Cache**: Redis 7 for session storage and config caching
- **Authentication**: bcrypt for passwords, pyotp for 2FA

**Key Features:**
- Hot-reload configuration without proxy restart
- Zero-downtime JWT/API key rotation
- Automated certificate generation and renewal
- Role-based access control (admin, service-owner)
- Per-cluster logging configuration

### Proxy-Ingress (Go/eBPF)

**Responsibilities:**
- Reverse proxy for external client traffic
- HTTPS/TLS termination with mTLS client validation
- Load balancing across backend services
- Host-based and path-based routing
- DDoS protection and rate limiting (XDP layer)
- WebSocket upgrade handling
- Health checks for backend services

**Technology Stack:**
- **Language**: Go 1.21
- **Packet Processing**: XDP (driver-level) → eBPF (kernel) → Go (userspace)
- **Acceleration**: AF_XDP for zero-copy I/O (Enterprise)
- **Protocols**: HTTP/1.1, HTTP/2, WebSocket

**Performance Tiers:**
1. **XDP Layer**: DDoS protection, rate limiting (40+ Gbps)
2. **eBPF Layer**: Connection tracking, simple routing (5+ Gbps)
3. **Go Layer**: mTLS validation, load balancing, WebSocket (1+ Gbps)

### Proxy-Egress (Go/eBPF)

**Responsibilities:**
- Forward proxy for internal service egress traffic
- Service-to-service authentication (JWT, Base64 tokens)
- Protocol support: TCP, UDP, ICMP, HTTP/HTTPS
- Traffic filtering and access control
- Connection pooling and multiplexing
- mTLS for service authentication

**Technology Stack:**
- Same as Proxy-Ingress with additional TCP/UDP/ICMP support
- Stateless design for horizontal scaling
- Connection tracking in eBPF for performance

**Performance Tiers:**
1. **XDP Layer**: Early packet filtering (40+ Gbps)
2. **eBPF Layer**: Protocol detection, simple rules (5+ Gbps)
3. **Go Layer**: Authentication, complex routing (1+ Gbps)

### Database (PostgreSQL)

**Schema Overview:**

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│    Users     │────>│   Clusters   │<────│   Proxies    │
│              │     │              │     │              │
│ • id         │     │ • id         │     │ • id         │
│ • username   │     │ • name       │     │ • hostname   │
│ • password   │     │ • api_key    │     │ • cluster_id │
│ • is_admin   │     │ • syslog     │     │ • status     │
│ • totp_secret│     │              │     │ • version    │
└──────────────┘     └──────────────┘     └──────────────┘
       │                     │
       │                     │
       ▼                     ▼
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  Services    │<────│   Mappings   │────>│ Certificates │
│              │     │              │     │              │
│ • id         │     │ • id         │     │ • id         │
│ • name       │     │ • source[]   │     │ • name       │
│ • ip_fqdn    │     │ • dest[]     │     │ • cert_data  │
│ • cluster_id │     │ • protocols  │     │ • key_data   │
│ • auth_type  │     │ • ports      │     │ • expiry     │
└──────────────┘     └──────────────┘     └──────────────┘
```

**Key Tables:**
- `users`: User accounts and authentication
- `clusters`: Cluster definitions and configuration
- `services`: Backend service definitions
- `mappings`: Traffic routing rules
- `proxy_servers`: Registered proxy instances
- `certificates`: TLS/mTLS certificates
- `mtls_cas`: Certificate authorities for mTLS
- `license_cache`: Cached license validation results

## Data Flow

### Typical HTTP Request Flow (L7/Envoy)

```
1. Client → External Load Balancer
2. Load Balancer → Envoy L7 Proxy (Listener :443 HTTPS)
3. Envoy XDP → Early packet classification
4. Envoy WASM Auth Filter → Validate JWT token
5. Envoy WASM License Filter → Check feature gate
6. Envoy Route Matching → Select upstream cluster
7. Envoy → Go L3/L4 Proxy (if routed for TCP)
8. Go L3/L4 → Destination Service
9. Response flows back through same path
```

### Configuration Update Flow

```
1. Admin → WebUI (React)
2. WebUI → API Server REST (HTTP POST)
3. API Server → Database (UPDATE operation)
4. API Server → xDS Server (Go microservice)
5. xDS Server → Envoy (gRPC streaming update)
6. Envoy → Applies new configuration (atomic)
7. API Server → Go L3/L4 Proxy (HTTP heartbeat)
8. Go L3/L4 → Applies new configuration
9. Both proxies → Metrics endpoint updated
```

### Proxy Registration Flow

```
1. Go L3/L4 Proxy → API Server /register
   - Provides: cluster_api_key, type (l3l4), version, capabilities
2. API Server → Validates license (license.penguintech.io)
3. API Server → Stores proxy record in database
4. API Server → Generates registration token (JWT)
5. API Server → Returns token to proxy
6. Proxy → Uses token for future heartbeat/config requests
7. Proxy → Periodic heartbeat (every 30s)
8. API Server → Updates proxy status and metrics
```

### Observability Data Flow

```
1. Envoy L7 → Prometheus metrics (port 9901/stats)
2. Go L3/L4 → Prometheus metrics (port 8082/metrics)
3. Both → OpenTelemetry tracing → Jaeger (port 14250)
4. Prometheus → Scrapes metrics (every 15s)
5. Grafana → Queries Prometheus (displays dashboards)
6. ELK Stack → Collects logs (via syslog or stdout)
7. WebUI → Real-time updates via WebSocket (from api-server)
```

## Data Flow Diagrams

### Configuration Update Flow

```
┌─────────────┐                ┌─────────────┐                ┌─────────────┐
│   Admin     │                │   Manager   │                │   Proxies   │
│   User      │                │             │                │             │
└──────┬──────┘                └──────┬──────┘                └──────┬──────┘
       │                              │                              │
       │ 1. Update Service Config     │                              │
       ├─────────────────────────────>│                              │
       │                              │                              │
       │                              │ 2. Validate & Save to DB     │
       │                              ├──────────────┐               │
       │                              │              │               │
       │                              │<─────────────┘               │
       │                              │                              │
       │ 3. Success Response          │                              │
       │<─────────────────────────────┤                              │
       │                              │                              │
       │                              │ 4. Config Change Event       │
       │                              ├─────────────────────────────>│
       │                              │                              │
       │                              │ 5. Pull New Config           │
       │                              │<─────────────────────────────┤
       │                              │                              │
       │                              │ 6. Return Updated Config     │
       │                              ├─────────────────────────────>│
       │                              │                              │
       │                              │                              │ 7. Hot Reload
       │                              │                              │    (No Restart)
       │                              │                              ├───────────────┐
       │                              │                              │               │
       │                              │                              │<──────────────┘
       │                              │                              │
       │                              │ 8. Heartbeat + Status        │
       │                              │<─────────────────────────────┤
       │                              │                              │
```

### Ingress Traffic Flow (External Client → Backend Service)

```
┌──────────────┐                                          ┌──────────────┐
│   External   │                                          │   Backend    │
│    Client    │                                          │   Service    │
└──────┬───────┘                                          └──────┬───────┘
       │                                                         │
       │ 1. HTTPS Request (mTLS)                                │
       ▼                                                         │
┌──────────────────────────────────────────┐                    │
│         Proxy-Ingress                    │                    │
├──────────────────────────────────────────┤                    │
│  XDP Layer                               │                    │
│  ├─ Rate Limit Check                     │                    │
│  ├─ DDoS Protection                      │                    │
│  └─ Early Packet Classification          │                    │
│                                          │                    │
│  eBPF Layer                              │                    │
│  ├─ Connection Tracking                  │                    │
│  ├─ Fast-path Routing (Simple Rules)     │                    │
│  └─ Statistics Collection                │                    │
│                                          │                    │
│  Go Layer                                │                    │
│  ├─ TLS Termination                      │                    │
│  ├─ mTLS Client Certificate Validation   │                    │
│  ├─ Host/Path-based Routing              │                    │
│  ├─ Load Balancer Selection              │                    │
│  └─ Backend Health Check                 │                    │
└────────────────────┬─────────────────────┘                    │
                     │                                          │
                     │ 2. Forward to Backend (HTTP/HTTPS)       │
                     ├─────────────────────────────────────────>│
                     │                                          │
                     │ 3. Backend Response                      │
                     │<─────────────────────────────────────────┤
                     │                                          │
                     │ 4. Return to Client (HTTPS)              │
                     ▼                                          │
┌──────────────┐                                                │
│   External   │                                                │
│    Client    │                                                │
└──────────────┘                                                │
```

### Egress Traffic Flow (Internal Service → Internet)

```
┌──────────────┐                                          ┌──────────────┐
│   Internal   │                                          │   Internet   │
│   Service    │                                          │  Destination │
└──────┬───────┘                                          └──────┬───────┘
       │                                                         │
       │ 1. Outbound Request (JWT/Token Auth)                   │
       ▼                                                         │
┌──────────────────────────────────────────┐                    │
│         Proxy-Egress                     │                    │
├──────────────────────────────────────────┤                    │
│  XDP Layer                               │                    │
│  ├─ Early Packet Filtering               │                    │
│  └─ DDoS Protection                      │                    │
│                                          │                    │
│  eBPF Layer                              │                    │
│  ├─ Service Identification               │                    │
│  ├─ Protocol Detection (TCP/UDP/ICMP)    │                    │
│  └─ Connection Tracking                  │                    │
│                                          │                    │
│  Go Layer                                │                    │
│  ├─ JWT/Token Validation                 │                    │
│  ├─ Mapping Rule Check                   │                    │
│  ├─ mTLS Service Authentication          │                    │
│  ├─ Connection Pooling                   │                    │
│  └─ Protocol Proxy (HTTP/TCP/UDP/ICMP)   │                    │
└────────────────────┬─────────────────────┘                    │
                     │                                          │
                     │ 2. Forward to Internet                   │
                     ├─────────────────────────────────────────>│
                     │                                          │
                     │ 3. Internet Response                     │
                     │<─────────────────────────────────────────┤
                     │                                          │
                     │ 4. Return to Service                     │
                     ▼                                          │
┌──────────────┐                                                │
│   Internal   │                                                │
│   Service    │                                                │
└──────────────┘                                                │
```

## Performance Architecture

### Multi-Tier Packet Processing

MarchProxy implements a hierarchical packet processing pipeline for optimal performance:

```
┌───────────────────────────────────────────────────────────────────┐
│                     PERFORMANCE TIERS                             │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Tier 1: Hardware Acceleration (100+ Gbps) - Enterprise          │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  DPDK (Data Plane Development Kit)                          │ │
│  │  • Kernel bypass for maximum throughput                     │ │
│  │  • Direct NIC hardware access                               │ │
│  │  • Zero-copy packet processing                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                               │                                   │
│                               ▼                                   │
│  Tier 2: XDP - eXpress Data Path (40+ Gbps)                      │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  Driver-Level Packet Processing                             │ │
│  │  • Programmable with eBPF                                   │ │
│  │  • DDoS protection                                          │ │
│  │  • Rate limiting                                            │ │
│  │  • Early packet drop/redirect                               │ │
│  │  • Connection tracking (simple)                             │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                               │                                   │
│                               ▼                                   │
│  Tier 3: eBPF Fast-Path (5+ Gbps)                                │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  Kernel-Level Packet Filtering                              │ │
│  │  • Simple rule matching                                     │ │
│  │  • Protocol detection                                       │ │
│  │  • Statistics collection                                    │ │
│  │  • Fast-path classification                                 │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                               │                                   │
│                               ▼                                   │
│  Tier 4: Go Application Logic (1+ Gbps)                          │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  Complex Packet Processing                                  │ │
│  │  • JWT/Token authentication                                 │ │
│  │  • mTLS certificate validation                              │ │
│  │  • TLS termination                                          │ │
│  │  • WebSocket upgrade handling                               │ │
│  │  • Advanced routing and load balancing                      │ │
│  │  • Full protocol feature support                            │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                               │                                   │
│                               ▼                                   │
│  Tier 5: Standard Networking (100+ Mbps)                         │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │  Traditional Kernel Socket Processing                       │ │
│  │  • Fallback for unsupported scenarios                       │ │
│  │  • Compatibility mode for older systems                     │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

### Automatic Workload Classification

The system automatically classifies traffic for optimal processing:

| Traffic Type | Processing Tier | Performance |
|--------------|----------------|-------------|
| Simple allow/drop rules | XDP | 40+ Gbps |
| Basic IP/port filtering | eBPF | 5+ Gbps |
| mTLS authentication | Go | 1+ Gbps |
| WebSocket upgrade | Go | 1+ Gbps |
| Complex routing logic | Go | 1+ Gbps |

## Security Architecture

### mTLS Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                    mTLS CERTIFICATE HIERARCHY                    │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│                  ┌───────────────────────┐                       │
│                  │    Root CA (ECC P-384) │                       │
│                  │  Self-signed by Manager│                       │
│                  │  10-year validity      │                       │
│                  └───────────┬───────────┘                       │
│                              │                                   │
│              ┌───────────────┴───────────────┐                  │
│              │                               │                  │
│      ┌───────▼────────┐             ┌────────▼───────┐         │
│      │ Server Certs   │             │  Client Certs  │         │
│      │ (Ingress Proxy)│             │  (Services)    │         │
│      │                │             │                │         │
│      │ • SANs support │             │ • CN=service   │         │
│      │ • 1-year       │             │ • 1-year       │         │
│      └────────────────┘             └────────────────┘         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**mTLS Features:**
- ECC P-384 elliptic curve cryptography (stronger than RSA 2048)
- SHA-384 hashing algorithm
- Automated certificate generation and renewal
- Certificate revocation list (CRL) support
- OCSP checking (optional)
- Hot certificate reload without proxy restart

### Authentication Flow

```
┌────────────┐                ┌──────────┐                ┌──────────┐
│   Client   │                │  Manager │                │  Proxy   │
└──────┬─────┘                └─────┬────┘                └────┬─────┘
       │                            │                          │
       │ 1. Login (user/pass/2FA)   │                          │
       ├───────────────────────────>│                          │
       │                            │                          │
       │ 2. Generate JWT            │                          │
       │    (expires in 1h)         │                          │
       │<───────────────────────────┤                          │
       │                            │                          │
       │ 3. Service Request         │                          │
       │    (JWT in header)         │                          │
       ├────────────────────────────┼─────────────────────────>│
       │                            │                          │
       │                            │ 4. Validate JWT          │
       │                            │    (check signature,     │
       │                            │     expiry, permissions) │
       │                            │<─────────────────────────┤
       │                            │                          │
       │                            │ 5. JWT Valid             │
       │                            ├─────────────────────────>│
       │                            │                          │
       │                            │                          │ 6. Forward Request
       │                            │                          ├────────────────>
       │                            │                          │
       │ 7. Response                │                          │
       │<───────────────────────────┼──────────────────────────┤
       │                            │                          │
```

## Scalability & High Availability

### Horizontal Scaling

Proxies are stateless and can be scaled horizontally:

```
                    ┌─────────────┐
                    │   Manager   │
                    │  (Stateful) │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│ Proxy Egress 1│  │ Proxy Egress 2│  │ Proxy Egress 3│
│  (Stateless)  │  │  (Stateless)  │  │  (Stateless)  │
└───────────────┘  └───────────────┘  └───────────────┘

        ▼                  ▼                  ▼
┌───────────────┐  ┌───────────────┐  ┌───────────────┐
│Proxy Ingress 1│  │Proxy Ingress 2│  │Proxy Ingress 3│
│  (Stateless)  │  │  (Stateless)  │  │  (Stateless)  │
└───────────────┘  └───────────────┘  └───────────────┘
```

**Community Edition**: Maximum 3 total proxy instances (any combination)
**Enterprise Edition**: Unlimited based on license

### High Availability

- **Manager**: Active-passive failover with shared database
- **Proxies**: Active-active load balancing (no session affinity)
- **Database**: PostgreSQL streaming replication
- **Redis**: Redis Sentinel for automatic failover

## Network Topology

### Typical Deployment Topology

```
Internet
    │
    │ (Port 443/HTTPS)
    ▼
┌───────────────────┐
│   Load Balancer   │ (External LB, optional)
└─────────┬─────────┘
          │
    ┌─────┴──────┐
    │            │
    ▼            ▼
┌────────┐  ┌────────┐
│ Proxy  │  │ Proxy  │
│Ingress │  │Ingress │
│   1    │  │   2    │
└────┬───┘  └───┬────┘
     │          │
     │  DMZ     │
     └────┬─────┘
          │
     ┌────▼─────┐
     │ Firewall │
     └────┬─────┘
          │
     Internal Network
     │
     ├──> Backend Services (10.0.1.0/24)
     │
     ├──> Manager (10.0.2.10)
     │
     ├──> Database (10.0.2.20)
     │
     └──> Proxy Egress (10.0.3.0/24)
           │
           └──> Internet (Outbound)
```

---

**For detailed deployment instructions, see [DEPLOYMENT.md](DEPLOYMENT.md)**
**For migration guidance, see [MIGRATION.md](MIGRATION.md)**
**For troubleshooting, see [TROUBLESHOOTING.md](TROUBLESHOOTING.md)**
