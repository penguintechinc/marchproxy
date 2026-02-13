# MarchProxy
<p align="center">
  <img src="logo.png" alt="MarchProxy Logo" width="400" />
</p>


[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/marchproxy/marchproxy)](https://goreportcard.com/report/github.com/marchproxy/marchproxy)
[![Docker Pulls](https://img.shields.io/docker/pulls/marchproxy/manager)](https://hub.docker.com/r/marchproxy/manager)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-brightgreen)](https://kubernetes.io/)
[![Performance](https://img.shields.io/badge/Performance-100Gbps%2B-red)](https://github.com/marchproxy/marchproxy/blob/main/docs/performance.md)
[![Version](https://img.shields.io/badge/version-v1.0.0-blue)](https://github.com/marchproxy/marchproxy/releases/tag/v1.0.0)

**A high-performance, enterprise-grade proxy suite for managing traffic in data center environments with advanced eBPF acceleration, optional hardware optimization, Kong API Gateway integration, and comprehensive management capabilities.**

**🎉 v1.0.0 Production Release** - Production-ready release with comprehensive documentation, enterprise features, multiple specialized proxy modules, and breakthrough performance. See [Release Notes](docs/RELEASE_NOTES.md) for details.

MarchProxy is a next-generation proxy solution designed for enterprise data centers that need to control and monitor network traffic. Built with a unique multi-tier performance architecture combining eBPF kernel programming, optional hardware acceleration (XDP, AF_XDP, DPDK, SR-IOV), Kong API Gateway for service orchestration, and enterprise-grade management with gRPC inter-container communication and REST APIs for external integration. Modern React-based web interface provides comprehensive traffic visibility and control.

## Why MarchProxy?

- **Multiple Specialized Proxies**: NLB (L3/L4 load balancing), ALB (L7 application), **Egress (secure egress traffic control)**, DBLB (database load balancing), AILB (Artificial Intelligence load balancing), RTMP (media streaming) and more
- **Unmatched Performance**: Multi-tier acceleration from standard networking → eBPF → XDP/AF_XDP → DPDK supporting 100+ Gbps throughput
- **Enterprise API Gateway**: Kong-based APILB wrapper for service orchestration, authentication, and rate limiting
- **Secure Egress Control**: Comprehensive egress proxy with IP/domain/URL blocking, TLS interception, and threat intelligence integration
- **Service-Centric**: Designed for service-to-service communication with granular access control and cluster isolation
- **Production Ready**: Comprehensive monitoring (Prometheus/Grafana), distributed tracing (Jaeger), and zero-downtime configuration updates
- **Modern Architecture**: gRPC for inter-container communication, REST APIs for external integration, React-based management interface
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
- **Web UI**: http://localhost:3000 (React management interface)
- **REST API Server**: http://localhost:8000 (API for external integration)
- **Kong APILB**: http://localhost:8001 (Kong Admin), http://localhost:8000 (Kong Proxy)
- **NLB (L3/L4)**: :9091 (admin)
- **ALB (L7)**: :9092 (admin)
- **DBLB (Database)**: :9093 (admin)
- **AILB (AI Load Balancer)**: :9094 (admin)
- **RTMP (Media)**: :1935 (RTMP)
- **Grafana**: http://localhost:3001 (Monitoring dashboards)
- **Jaeger**: http://localhost:16686 (Distributed tracing)
- **Prometheus**: http://localhost:9090 (Metrics)

**What you get out of the box:**
- ✅ REST API Server for configuration management and external integration
- ✅ React Web UI with modern dashboard (dark theme)
- ✅ Kong-based APILB for API gateway and service orchestration
- ✅ Specialized proxy modules:
  - NLB: L3/L4 load balancing with 100+ Gbps throughput
  - ALB: L7 application proxy with 40+ Gbps throughput
  - **Egress: Secure egress traffic control with threat intelligence**
  - DBLB: Database load balancing
  - AILB: Artificial Intelligence load balancing
  - RTMP: Media streaming support
- ✅ eBPF acceleration across all proxy modules
- ✅ PostgreSQL database with optimized schema
- ✅ Redis caching for performance
- ✅ gRPC inter-container communication
- ✅ Prometheus metrics collection from all services
- ✅ Grafana dashboards for visualization
- ✅ Jaeger for distributed tracing

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
- [Documentation](#-documentation)
- [v1.0.0 Release](#-v100-release-highlights)
- [Contributing](#-contributing)
- [License](#-license)
- [Support](#-support)

## ✨ Features

### Core Features
- **Multiple Specialized Proxies**: NLB (L3/L4), ALB (L7), Egress (secure egress), DBLB (database), AILB (AI), RTMP (media) modules
- **High-Performance**: Multi-protocol support (TCP, UDP, ICMP, HTTP/HTTPS, WebSocket, QUIC/HTTP3, RTMP)
- **eBPF Acceleration**: Kernel-level packet processing across all proxy modules
- **Kong API Gateway**: APILB wrapper for service orchestration, request routing, and authentication
- **Service-to-Service Mapping**: Granular traffic routing and access control
- **Multi-Cluster Support**: Enterprise-grade cluster management and isolation
- **Real-time Configuration**: Hot-reload configuration without downtime
- **Comprehensive Monitoring**: Prometheus metrics, Grafana dashboards, Jaeger tracing, and observability

### Egress Proxy Features
- **Threat Intelligence**: IP/CIDR blocking, domain blocking (with wildcard support), URL pattern matching
- **TLS Interception**: MITM mode with dynamic cert generation, or preconfigured certificates
- **L7 Control**: HTTP/1.1, HTTP/2, and HTTP/3 (QUIC) support (**EXPERIMENTAL**)
- **Access Control**: JWT/Bearer token-based restrictions per destination
- **DNS Caching**: Resolved domain blocking with TTL-based caching
- **Real-time Updates**: gRPC streaming and polling for threat feed synchronization

### Performance Acceleration
- **eBPF Fast-path**: Programmable kernel-level packet filtering
- **Hardware Acceleration**: Optional DPDK, XDP, AF_XDP, and SR-IOV support
- **Advanced Caching**: Redis-backed and in-memory caching with multiple eviction policies
- **Circuit Breaker**: Automatic failure detection and recovery
- **Content Compression**: Gzip, Brotli, Zstandard, and Deflate support

### Security & Authentication
- **Kong-based Authentication**: OAuth2, SAML, SCIM, API key validation
- **Certificate Management**: Centralized TLS certificate management across all proxies
- **Multiple Auth Methods**: Base64 tokens, JWT, 2FA/TOTP (via Kong plugins)
- **Enterprise Authentication**: SAML SSO, SCIM provisioning, OAuth2 integration
- **Network Access Control**: Granular service-to-service access policies
- **Rate Limiting & DDoS Protection**: Advanced traffic shaping and attack mitigation

### Enterprise Features
- **Multi-Cluster Management**: Unlimited clusters with separate API keys and configurations
- **Advanced Authentication**: SAML SSO, SCIM provisioning, OAuth2 integration
- **Centralized Logging**: Per-cluster syslog configuration and structured logging
- **License Management**: Integration with license.penguintech.io
- **High Availability**: Auto-scaling, load balancing, and fault tolerance

## 🏗️ Architecture

MarchProxy features a microservice architecture with independent proxy modules optimized for different traffic types, allowing resources to be scaled individually based on traffic demands:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        External Traffic (All Types)                     │
└──────────────────────────┬──────────────────────────────────────────────┘
                           │
      ┌────────────────────▼────────────────────────────────────┐
      │        NLB (L3/L4 Entry Point & Traffic Control)       │
      │  • Initial traffic distribution                        │
      │  • Protocol detection & routing decision               │
      │  • Traffic throttling & rate limiting (100+ Gbps)      │
      │  • DDoS protection & traffic shaping                   │
      │  • eBPF-accelerated filtering                          │
      └────────────┬────────┬──────────────────────────────────┘
                   │        │
        Direct     │ gRPC   │ Internal routing
        apps       │ routing│ (optimized modules)
                   │        │
   ┌───────────────┼────────┼──────────────┐
   │               │        │              │
┌──▼────┐  ┌──────▼────┐ ┌─▼────────┐ ┌──▼────────────┐
│Direct │  │    ALB    │ │ DBLB    │ │ Specialized:  │
│Apps   │  │  (L7 Apps)│ │Database │ │ • AILB (AI LB)│
│(Scale │  │  (Scale)  │ │ (Scale) │ │ • RTMP (x265) │
│ ↑↓)   │  │           │ │         │ │ • Others      │
└───────┘  └───────────┘ └─────────┘ └───────────────┘

    Control Plane & Management
         │
      ┌──▼──────────────────────────┐
      │   API Server (REST/gRPC)   │
      │  • Configuration mgmt       │
      │  • License validation       │
      │  • Multi-cluster support    │
      │  • Service discovery        │
      └──────────┬─────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
   ┌────▼──────┐    ┌─────▼──────┐
   │PostgreSQL │    │ Redis      │
   │ Database  │    │  Cache     │
   └───────────┘    └────────────┘

      ┌────────────────────────────┐
      │   Observability Stack      │
      │  • Prometheus/Grafana      │
      │  • Jaeger (distributed)    │
      │  • Metrics export          │
      └────────────────────────────┘
```

### Component Architecture

#### API Server (REST/gRPC)
- **Configuration Management**: Centralized proxy configuration and service mapping
- **Multi-Cluster Support**: Enterprise cluster isolation with separate credentials
- **License Validation**: Real-time license checking via license.penguintech.io
- **Service Discovery**: Dynamic proxy registration and heartbeat health checks
- **REST API**: External integration with JSON payloads
- **gRPC Communication**: Internal inter-container communication

#### NLB (Network Load Balancer - L3/L4 Entry Point)
- **Traffic Distribution**: Routes traffic to appropriate modules or direct to applications
- **Protocol Detection**: Identifies traffic type for intelligent routing decisions
- **Centralized Control**: All traffic throttling, rate limiting, and DDoS protection
- **eBPF Acceleration**: Kernel-level packet processing at entry point
- **Lightweight Downstream**: Keeps other modules focused on their specialized functions
- **100+ Gbps Capacity**: High-performance entry point with minimal overhead

#### Specialized Proxy Modules (Go/eBPF)
Each module independently scalable based on traffic demands (traffic control handled by NLB):
- **ALB (Application L7)**: HTTP/HTTPS/gRPC applications, 40+ Gbps throughput
- **Egress (Secure Egress)**: Egress traffic control with threat intelligence, TLS interception, IP/domain/URL blocking
- **DBLB (Database)**: Database traffic load balancing with query awareness
- **AILB (Artificial Intelligence)**: AI model inference routing and optimization
- **RTMP (Media Streaming)**: x265 codec by default with x264 backwards compatibility

**All downstream modules share:**
- Multi-Tier Processing: Hardware → XDP → eBPF → Go application logic
- eBPF Acceleration: Kernel-level packet filtering and processing
- Configuration Sync: Hot-reload without connection drops
- Zero-Copy Networking: AF_XDP support for ultra-low latency
- Lightweight Design: Traffic control handled upstream by NLB

### Performance Tiers
1. **Standard Networking**: Traditional kernel socket processing (~1 Gbps)
2. **eBPF Acceleration**: Programmable kernel-level packet filtering (~10 Gbps)
3. **XDP/AF_XDP**: Driver-level processing and zero-copy I/O (~40 Gbps)
4. **DPDK/SR-IOV**: Kernel bypass + hardware isolation (~100+ Gbps)

## 💼 Community vs Enterprise

**Community Edition** includes all core proxy modules (NLB, ALB, Egress, DBLB, AILB, RTMP) with:
- Up to 3 proxy module instances total
- Single cluster
- eBPF acceleration
- REST API and Web UI
- Prometheus/Grafana monitoring
- Jaeger distributed tracing
- Open source (AGPL v3)

**Enterprise Edition** adds (see [marchproxy.io](https://marchproxy.io) for complete details):
- Unlimited proxy module instances
- Multi-cluster support with isolation
- Advanced hardware acceleration (XDP/AF_XDP/DPDK/SR-IOV)
- SAML/SCIM/OAuth2 integration
- Advanced Kong APILB features
- Auto-scaling policies
- 24/7 professional support
- Commercial licensing
- Enhanced performance optimization and customization

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

### Basic Configuration via REST API

Configure services and traffic routing:

```bash
# Create a service endpoint
curl -X POST http://localhost:8000/api/v1/services \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "name": "web-backend",
    "ip_address": "10.0.1.50",
    "port": 80,
    "protocol": "http",
    "health_check_path": "/health",
    "cluster_id": 1
  }'

# Route traffic through NLB to backend
curl -X POST http://localhost:8000/api/v1/routes \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "source_port": 80,
    "destination_service_id": "web-backend",
    "load_balance_strategy": "round_robin",
    "cluster_id": 1
  }'
```

Alternatively, use the Web UI at `http://localhost:3000` for visual configuration.

## 📚 Documentation

MarchProxy comprehensive documentation is organized in the `docs/` folder:

### Getting Started
- **[QUICKSTART.md](docs/QUICKSTART.md)** - Getting started with MarchProxy in 5 minutes
- **[KUBERNETES.md](docs/KUBERNETES.md)** - Kubernetes deployment guide with Helm and Operators

### Core Documentation
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System design, component architecture, and data flows
- **[SECURITY.md](docs/SECURITY.md)** - Security policy, threat models, and hardening guidance
- **[STANDARDS.md](docs/STANDARDS.md)** - Development standards, code style, and best practices
- **[WORKFLOWS.md](docs/WORKFLOWS.md)** - CI/CD pipelines, GitHub Actions, and deployment workflows

### Development & Operations
- **[TESTING.md](docs/TESTING.md)** - Testing strategy, test coverage, and running tests
- **[development/contributing.md](docs/development/contributing.md)** - Local development setup and environment
- **[DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Production deployment, scaling, and operations
- **[TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)** - Common issues and debugging

### Contributing & Attribution
- **[CONTRIBUTION.md](docs/CONTRIBUTION.md)** - Contributing guidelines and developer setup
- **[ATTRIBUTION.md](docs/ATTRIBUTION.md)** - Credits, third-party libraries, and acknowledgments
- **[RELEASE_NOTES.md](docs/RELEASE_NOTES.md)** - Version history, feature releases, and breaking changes

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](docs/CONTRIBUTION.md) for details.

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
