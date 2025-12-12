# MarchProxy

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/marchproxy/marchproxy)](https://goreportcard.com/report/github.com/marchproxy/marchproxy)
[![Docker Pulls](https://img.shields.io/docker/pulls/marchproxy/manager)](https://hub.docker.com/r/marchproxy/manager)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-brightgreen)](https://kubernetes.io/)
[![Performance](https://img.shields.io/badge/Performance-100Gbps%2B-red)](https://github.com/marchproxy/marchproxy/blob/main/docs/performance.md)
[![Version](https://img.shields.io/badge/version-v1.0.0-blue)](https://github.com/marchproxy/marchproxy/releases/tag/v1.0.0)

**A high-performance, enterprise-grade dual proxy suite for managing both egress and ingress traffic in data center environments with advanced eBPF acceleration, mTLS authentication, and hardware optimization.**

**🎉 v1.0.0 Production Release** - First production-ready release with comprehensive documentation, enhanced mTLS support, and enterprise features. See [Release Notes](docs/RELEASE_NOTES.md) for details.

MarchProxy is a next-generation dual proxy solution designed for enterprise data centers that need to control and monitor both egress traffic to the internet and ingress traffic from external clients. Built with a unique multi-tier performance architecture combining eBPF kernel programming, mTLS mutual authentication, hardware acceleration (XDP, AF_XDP, DPDK, SR-IOV), and enterprise-grade management capabilities.

## Why MarchProxy?

- **Dual Proxy Architecture**: Complete solution with both egress (forward proxy) and ingress (reverse proxy) functionality
- **Unmatched Performance**: Multi-tier acceleration from standard networking → eBPF → XDP/AF_XDP → DPDK supporting 100+ Gbps throughput
- **Enterprise Security**: Built-in mTLS authentication, WAF, DDoS protection, XDP-based rate limiting, and comprehensive authentication (SAML, OAuth2, SCIM)
- **Service-Centric**: Designed for service-to-service communication with granular access control and cluster isolation
- **mTLS by Default**: Mutual TLS authentication with automated certificate management and ECC P-384 cryptography
- **Production Ready**: Comprehensive monitoring, centralized logging, automatic failover, and zero-downtime configuration updates
- **Open Source + Enterprise**: Community edition with core features, Enterprise edition with advanced acceleration and unlimited scaling

## 🚀 Quick Start

### Docker Compose (Recommended for Testing)

Get MarchProxy running in under 5 minutes with our comprehensive Docker Compose setup:

```bash
# Clone the repository
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy

# Copy environment configuration
cp .env.example .env

# Start all services using the provided script
./scripts/start.sh

# Or use docker-compose directly
docker-compose up -d

# Verify all services are running
docker-compose ps

# Check health status
./scripts/health-check.sh
```

**Access Points:**
- **Web UI**: http://localhost:3000 (React frontend)
- **API Server**: http://localhost:8000 (FastAPI REST API)
- **Envoy Admin**: http://localhost:9901 (Proxy L7 admin)
- **Grafana**: http://localhost:3000 (Monitoring dashboards)
- **Jaeger**: http://localhost:16686 (Distributed tracing)
- **Prometheus**: http://localhost:9090 (Metrics)
- **Kibana**: http://localhost:5601 (Log viewer)
- **AlertManager**: http://localhost:9093 (Alert management)

**What you get out of the box:**
- ✅ FastAPI REST API Server for configuration management
- ✅ React Web UI with modern dashboard (Dark Grey/Navy/Gold theme)
- ✅ Proxy L7 (Envoy) for HTTP/HTTPS/gRPC with 40+ Gbps throughput
- ✅ Proxy L3/L4 (Go) for TCP/UDP with 100+ Gbps throughput
- ✅ Legacy proxy-egress (forward proxy) with eBPF acceleration
- ✅ Legacy proxy-ingress (reverse proxy) with load balancing
- ✅ Complete mTLS authentication with automated certificate generation
- ✅ PostgreSQL database with optimized schema
- ✅ Redis caching for performance
- ✅ Prometheus metrics collection from all services
- ✅ Grafana dashboards for visualization
- ✅ ELK stack for centralized logging
- ✅ Jaeger for distributed tracing
- ✅ AlertManager for intelligent alerting
- ✅ Loki for log aggregation

**Integration Scripts:**
```bash
# Start all services with dependency ordering
./scripts/start.sh

# Stop all services gracefully
./scripts/stop.sh

# Check health of all services
./scripts/health-check.sh

# Run database migrations
./scripts/migrate.sh
```

**Quick Configuration Test:**
```bash
# Check health of all services
./scripts/health-check.sh

# View logs from specific service
docker-compose logs -f api-server
docker-compose logs -f webui
docker-compose logs -f proxy-l7
docker-compose logs -f proxy-l3l4

# Test L7 proxy health
curl http://localhost:9901/stats

# Test L3/L4 proxy health
curl http://localhost:8082/healthz

# Access Jaeger tracing
open http://localhost:16686

# Access Grafana dashboards
open http://localhost:3000
```

### Kubernetes with Helm

```bash
# Add Helm repository
helm repo add marchproxy https://charts.marchproxy.io
helm repo update

# Install MarchProxy
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace
```

### Kubernetes with Operator

```bash
# Install the operator
kubectl apply -f https://raw.githubusercontent.com/marchproxy/marchproxy/main/operator/config/crd/marchproxy.yaml
kubectl apply -f https://raw.githubusercontent.com/marchproxy/marchproxy/main/operator/config/manager/manager.yaml

# Deploy MarchProxy instance
kubectl apply -f examples/simple-marchproxy.yaml
```

## 📋 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Quick Start](#-quick-start)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Performance](#-performance)
- [Security](#-security)
- [Documentation](#-documentation)
- [v1.0.0 Release](#-v100-release-highlights)
- [Contributing](#-contributing)
- [License](#-license)
- [Support](#-support)

## ✨ Features

### Core Features
- **Dual Proxy Architecture**: Both egress (forward) and ingress (reverse) proxy functionality
- **High-Performance Proxies**: Multi-protocol support (TCP, UDP, ICMP, HTTP/HTTPS, WebSocket, QUIC/HTTP3)
- **mTLS Authentication**: Mutual TLS with automated certificate management and ECC P-384 cryptography
- **eBPF Acceleration**: Kernel-level packet processing for maximum performance on both proxies
- **Service-to-Service Mapping**: Granular traffic routing and access control
- **Multi-Cluster Support**: Enterprise-grade cluster management and isolation
- **Real-time Configuration**: Hot-reload configuration without downtime
- **Comprehensive Monitoring**: Prometheus metrics, health checks, and observability for both proxies

### Performance Acceleration
- **eBPF Fast-path**: Programmable kernel-level packet filtering
- **Hardware Acceleration**: Optional DPDK, XDP, AF_XDP, and SR-IOV support
- **Advanced Caching**: Redis-backed and in-memory caching with multiple eviction policies
- **Circuit Breaker**: Automatic failure detection and recovery
- **Content Compression**: Gzip, Brotli, Zstandard, and Deflate support

### Security & Authentication
- **mTLS Mutual Authentication**: Client certificate validation with ECC P-384 cryptography
- **Certificate Management**: Automated CA generation or upload existing certificate chains
- **Multiple Auth Methods**: Base64 tokens, JWT, 2FA/TOTP
- **Enterprise Authentication**: SAML, SCIM, OAuth2 (Google, Microsoft, etc.)
- **TLS Management**: Automatic certificate management via Infisical/Vault or manual upload
- **Web Application Firewall**: SQL injection, XSS, and command injection protection
- **Rate Limiting & DDoS Protection**: Advanced traffic shaping and attack mitigation

### Enterprise Features
- **Multi-Cluster Management**: Unlimited clusters with separate API keys and configurations
- **Advanced Authentication**: SAML SSO, SCIM provisioning, OAuth2 integration
- **Centralized Logging**: Per-cluster syslog configuration and structured logging
- **License Management**: Integration with license.penguintech.io
- **High Availability**: Auto-scaling, load balancing, and fault tolerance

## 🏗️ Architecture

MarchProxy features a distributed dual proxy architecture optimized for both egress and ingress traffic management:

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   External      │       │   Enterprise    │       │   Data Center   │
│    Clients      │       │   Management    │       │    Services     │
└─────────┬───────┘       └─────────┬───────┘       └─────────┬───────┘
          │                         │                         │
          │ HTTPS/mTLS              │                         │ Egress
          │                         │                         │
┌─────────▼─────────────────────────▼─────────────────────────▼─────────┐
│                     MarchProxy Dual Proxy Cluster                   │
│                                                                      │
│  ┌─────────────────┐          ┌─────────────────────────────────┐   │
│  │     Manager     │◄────────►│        Proxy Architecture       │   │
│  │ (py4web/pydal)  │          │                                 │   │
│  │                 │          │  ┌─────────────┐ ┌─────────────┐│   │
│  │ • Web Dashboard │          │  │   Ingress   │ │   Egress    ││   │
│  │ • API Server    │          │  │  (Reverse)  │ │  (Forward)  ││   │
│  │ • User Mgmt     │          │  │             │ │             ││   │
│  │ • License Mgmt  │          │  │ :80 (HTTP)  │ │ :8080 (TCP) ││   │
│  │ • mTLS CA Mgmt  │          │  │ :443 (TLS)  │ │ :8081 (ADM) ││   │
│  │ • Cert Mgmt     │          │  │ :8082 (ADM) │ │             ││   │
│  └─────────────────┘          │  │             │ │             ││   │
│           │                   │  │ ┌─────────┐ │ │ ┌─────────┐ ││   │
│           │                   │  │ │ mTLS    │ │ │ │ mTLS    │ ││   │
│           │                   │  │ │ eBPF    │ │ │ │ eBPF    │ ││   │
│           │                   │  │ │ XDP     │ │ │ │ XDP     │ ││   │
│           │                   │  │ └─────────┘ │ │ └─────────┘ ││   │
│           │                   │  └─────────────┘ └─────────────┘│   │
│           │                   └─────────────────────────────────┘   │
│           │                                     │                   │
│           ▼                                     ▼                   │
│  ┌─────────────────┐                    ┌─────────────────────────┐ │
│  │   PostgreSQL    │                    │    Observability        │ │
│  │   Database      │                    │                         │ │
│  │                 │                    │ • Prometheus/Grafana    │ │
│  │ • Clusters      │                    │ • ELK Stack             │ │
│  │ • Services      │                    │ • Jaeger Tracing        │ │
│  │ • Mappings      │                    │ • AlertManager          │ │
│  │ • Users         │                    │ • mTLS Metrics          │ │
│  │ • Certificates  │                    │ • Dual Proxy Dashboards│ │
│  │ • Ingress Routes│                    │                         │ │
│  └─────────────────┘                    └─────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────┘
```

### Component Architecture

#### Manager (Python/py4web)
- **Configuration Management**: Centralized service and mapping configuration
- **Multi-Cluster Support**: Enterprise cluster isolation with separate API keys
- **Authentication Hub**: SAML, OAuth2, SCIM integration for enterprise SSO
- **License Validation**: Real-time license checking with license.penguintech.io
- **TLS Certificate Authority**: Self-signed CA generation and wildcard certificates
- **Web Interface**: Modern multi-page dashboard with real-time monitoring

#### Proxy Nodes (Go/eBPF)
- **Multi-Tier Processing**: Hardware → XDP → eBPF → Go application logic
- **Protocol Support**: TCP, UDP, ICMP, HTTP/HTTPS, WebSocket, QUIC/HTTP3
- **Enterprise Rate Limiting**: XDP-based packet-per-second rate limiting
- **Advanced Security**: WAF, DDoS protection, circuit breakers
- **Zero-Copy Networking**: AF_XDP for ultra-low latency packet processing
- **Configuration Sync**: Hot-reload configuration without connection drops

### Performance Tiers
1. **Standard Networking**: Traditional kernel socket processing (~1 Gbps)
2. **eBPF Acceleration**: Programmable kernel-level packet filtering (~10 Gbps)
3. **XDP/AF_XDP**: Driver-level processing and zero-copy I/O (~40 Gbps)
4. **DPDK/SR-IOV**: Kernel bypass + hardware isolation (~100+ Gbps)

## 💼 Edition Comparison

| Feature | Community | Enterprise |
|---------|-----------|------------|
| **Proxy Instances** | Up to 3 total (any combination of ingress/egress) | Unlimited* |
| **Clusters** | Single default | Multiple with isolation |
| **Performance Tier** | Standard + eBPF | + XDP/AF_XDP + DPDK |
| **Rate Limiting** | Basic application-level | + XDP-based HW acceleration |
| **Authentication** | Basic, 2FA, JWT | + SAML, SCIM, OAuth2 |
| **TLS Management** | Manual certificates | + Wildcard CA generation |
| **Network Acceleration** | eBPF fast-path | + SR-IOV, NUMA optimization |
| **Web Application Firewall** | Basic protection | + Advanced threat detection |
| **Monitoring & Analytics** | Prometheus metrics | + Advanced dashboards, alerting |
| **Centralized Logging** | Local logging | + Per-cluster syslog, ELK stack |
| **Load Balancing** | Round-robin | + Weighted, least-conn, geo-aware |
| **Content Processing** | Basic compression | + Brotli, Zstandard, smart caching |
| **Circuit Breaker** | Basic | + Advanced patterns, auto-recovery |
| **Distributed Tracing** | Basic | + OpenTelemetry integration |
| **Support** | Community forums | 24/7 enterprise support |
| **License** | AGPL v3 | Commercial license available |

*Based on license entitlements from license.penguintech.io

### Proxy Instance Limits

**Community Edition:**
- **3 total proxy instances maximum** across all types
- Examples of valid configurations:
  - 1 ingress + 2 egress proxies
  - 2 ingress + 1 egress proxy
  - 3 egress proxies (no ingress)
  - 3 ingress proxies (no egress)
- All proxies share the same default cluster

**Enterprise Edition:**
- **Unlimited proxy instances** of both types
- Multiple clusters with separate quotas and isolation
- License determines specific limits per deployment

## 🚀 Installation

### System Requirements

#### Minimum Requirements
- **CPU**: 2 cores
- **Memory**: 4 GB RAM
- **Storage**: 10 GB available space
- **Network**: 1 Gbps network interface
- **OS**: Linux kernel 4.18+ (for eBPF support)

#### Recommended for Production
- **CPU**: 8+ cores
- **Memory**: 16+ GB RAM
- **Storage**: 100+ GB SSD
- **Network**: 10+ Gbps network interface
- **OS**: Ubuntu 20.04+ or RHEL 8+

### Installation Methods

#### 1. Docker Compose (Quickest)
```bash
curl -sSL https://raw.githubusercontent.com/marchproxy/marchproxy/main/docker-compose.yml | \
  docker-compose -f - up -d
```

#### 2. Kubernetes with Helm
```bash
helm repo add marchproxy https://charts.marchproxy.io
helm install marchproxy marchproxy/marchproxy
```

#### 3. Kubernetes with Operator
```bash
kubectl apply -f https://github.com/marchproxy/marchproxy/releases/latest/download/operator.yaml
```

## ⚙️ Configuration

### Basic Configuration

Create a service and mapping:

```yaml
# Service definition
services:
  - name: "web-backend"
    ip_fqdn: "backend.internal.com"
    collection: "web-services"
    auth_type: "jwt"
    cluster_id: 1

# Mapping definition
mappings:
  - source_services: ["web-frontend"]
    dest_services: ["web-backend"]
    protocols: ["tcp", "http"]
    ports: [80, 443]
    auth_required: true
    cluster_id: 1
```

## 🔧 Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy

# Build manager
cd manager
pip install -r requirements.txt

# Build proxy
cd ../proxy
go build -o proxy ./cmd/proxy

# Run tests
cd ..
./test/run_tests.sh --all
```

## 📚 v1.0.0 Release Highlights

**MarchProxy v1.0.0** is now production-ready with comprehensive documentation and enterprise features:

### New Documentation
- **[API.md](docs/API.md)** - Complete API reference with authentication flows and examples
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System architecture diagrams and data flow
- **[DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Step-by-step deployment guides (Docker, Kubernetes, Bare Metal)
- **[MIGRATION.md](docs/MIGRATION.md)** - Migration guide from v0.1.x to v1.0.0
- **[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** - Common issues and solutions
- **[RELEASE_NOTES.md](docs/RELEASE_NOTES.md)** - Complete release notes and changelog

### Key Improvements
- ✅ Production-ready dual proxy architecture (ingress + egress)
- ✅ Enterprise mTLS Certificate Authority with ECC P-384
- ✅ Complete observability stack (Prometheus, Grafana, ELK, Jaeger)
- ✅ Multi-tier performance architecture (100+ Gbps capability)
- ✅ Comprehensive testing (10,000+ tests, 72-hour soak testing)
- ✅ Enhanced security and hardening
- ✅ Zero-downtime configuration updates

### Breaking Changes
- Configuration updates required (see [MIGRATION.md](docs/MIGRATION.md))
- Database schema migration needed
- `PROXY_TYPE` environment variable now required

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](docs/development/contributing.md) for details.

### Quick Start for Contributors

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

## 📄 License

### Community Edition
MarchProxy Community Edition is licensed under the [GNU Affero General Public License v3.0](LICENSE).

**License Highlights:**
- **Personal & Internal Use**: Free under AGPL-3.0
- **Commercial Use**: Requires commercial license
- **SaaS Deployment**: Requires commercial license if providing as a service

### Contributor Employer Exception (GPL-2.0 Grant)

Companies employing official contributors receive GPL-2.0 access to community features:

- **Perpetual for Contributed Versions**: GPL-2.0 rights to versions where the employee contributed remain valid permanently, even after the employee leaves the company
- **Attribution Required**: Employee must be credited in CONTRIBUTORS, AUTHORS, commit history, or release notes
- **Future Versions**: New versions released after employment ends require standard licensing
- **Community Only**: Enterprise features still require a commercial license

This exception rewards contributors by providing lasting fair use rights to their employers.

### Enterprise Edition
Enterprise features require a commercial license. Contact [sales@marchproxy.io](mailto:sales@marchproxy.io) for licensing information.

## 🆘 Support

### Community Support
- **GitHub Issues**: Bug reports and feature requests
- **Discussions**: Community Q&A and discussions

### Enterprise Support
- **24/7 Support**: Emergency response and critical issue resolution
- **Professional Services**: Implementation assistance and consulting

---

**Made with ❤️ by the MarchProxy team**
