# MarchProxy

[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Report Card](https://goreportcard.com/badge/github.com/marchproxy/marchproxy)](https://goreportcard.com/report/github.com/marchproxy/marchproxy)
[![Docker Pulls](https://img.shields.io/docker/pulls/marchproxy/manager)](https://hub.docker.com/r/marchproxy/manager)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-brightgreen)](https://kubernetes.io/)
[![Performance](https://img.shields.io/badge/Performance-100Gbps%2B-red)](https://github.com/marchproxy/marchproxy/blob/main/docs/performance.md)

**A high-performance, enterprise-grade proxy suite for managing egress traffic in data center environments with advanced eBPF acceleration and hardware optimization.**

MarchProxy is a next-generation proxy solution designed for enterprise data centers that need to control and monitor egress traffic to the internet. Built with a unique multi-tier performance architecture combining eBPF kernel programming, hardware acceleration (XDP, AF_XDP, DPDK, SR-IOV), and enterprise-grade management capabilities.

## Why MarchProxy?

- **Unmatched Performance**: Multi-tier acceleration from standard networking → eBPF → XDP/AF_XDP → DPDK supporting 100+ Gbps throughput
- **Enterprise Security**: Built-in WAF, DDoS protection, XDP-based rate limiting, and comprehensive authentication (SAML, OAuth2, SCIM)
- **Service-Centric**: Designed for service-to-service communication with granular access control and cluster isolation
- **Production Ready**: Comprehensive monitoring, centralized logging, automatic failover, and zero-downtime configuration updates
- **Open Source + Enterprise**: Community edition with core features, Enterprise edition with advanced acceleration and unlimited scaling

## 🚀 Quick Start

### Docker Compose (Recommended for Testing)

Get MarchProxy running in under 5 minutes with our comprehensive Docker Compose setup:

```bash
# Clone the repository
git clone https://github.com/marchproxy/marchproxy.git
cd marchproxy

# Start the complete stack (Manager + Proxy + Observability)
docker-compose up -d

# Verify all services are running
docker-compose ps

# Access the management interface
open http://localhost:8000

# Default credentials:
# Username: admin
# Password: changeme
# 2FA: Use any TOTP app to scan the QR code
```

**What you get out of the box:**
- ✅ Manager web interface with modern dashboard
- ✅ High-performance Go proxy with eBPF acceleration
- ✅ PostgreSQL database with sample data
- ✅ Prometheus metrics collection
- ✅ Grafana dashboards for monitoring
- ✅ ELK stack for centralized logging
- ✅ Jaeger for distributed tracing
- ✅ AlertManager for intelligent alerting

**Quick Configuration Test:**
```bash
# Create a simple service mapping
curl -X POST http://localhost:8000/api/services \
  -H "Authorization: Bearer $(cat .cluster-api-key)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-service",
    "ip_fqdn": "httpbin.org",
    "collection": "test",
    "auth_type": "none"
  }'

# Test proxy connectivity
curl -x http://localhost:8080 http://httpbin.org/ip
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
- [Contributing](#-contributing)
- [License](#-license)
- [Support](#-support)

## ✨ Features

### Core Features
- **High-Performance Proxy**: Multi-protocol support (TCP, UDP, ICMP, HTTP/HTTPS, WebSocket, QUIC/HTTP3)
- **eBPF Acceleration**: Kernel-level packet processing for maximum performance
- **Service-to-Service Mapping**: Granular traffic routing and access control
- **Multi-Cluster Support**: Enterprise-grade cluster management and isolation
- **Real-time Configuration**: Hot-reload configuration without downtime
- **Comprehensive Monitoring**: Prometheus metrics, health checks, and observability

### Performance Acceleration
- **eBPF Fast-path**: Programmable kernel-level packet filtering
- **Hardware Acceleration**: Optional DPDK, XDP, AF_XDP, and SR-IOV support
- **Advanced Caching**: Redis-backed and in-memory caching with multiple eviction policies
- **Circuit Breaker**: Automatic failure detection and recovery
- **Content Compression**: Gzip, Brotli, Zstandard, and Deflate support

### Security & Authentication
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

MarchProxy features a distributed architecture optimized for high-performance egress traffic management:

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   Data Center   │       │   Enterprise    │       │    Internet     │
│    Services     │       │   Management    │       │   Destinations  │
└─────────┬───────┘       └─────────┬───────┘       └─────────┬───────┘
          │                         │                         │
          │                         │                         │
┌─────────▼─────────────────────────▼─────────────────────────▼─────────┐
│                        MarchProxy Cluster                            │
│  ┌─────────────────┐                    ┌─────────────────────────┐  │
│  │     Manager     │◄──────────────────►│       Proxy Nodes       │  │
│  │ (py4web/pydal)  │   Configuration    │    (Go/eBPF/XDP)        │  │
│  │                 │      & Control     │                         │  │
│  │ • Web Dashboard │                    │ ┌─────┐ ┌─────┐ ┌─────┐ │  │
│  │ • API Server    │                    │ │ XDP │ │ XDP │ │ XDP │ │  │
│  │ • User Mgmt     │                    │ │ P1  │ │ P2  │ │ P3  │ │  │
│  │ • License Mgmt  │                    │ └─────┘ └─────┘ └─────┘ │  │
│  │ • TLS CA        │                    │                         │  │
│  └─────────────────┘                    └─────────────────────────┘  │
│           │                                         │                │
│           ▼                                         ▼                │
│  ┌─────────────────┐                    ┌─────────────────────────┐  │
│  │   PostgreSQL    │                    │    Observability        │  │
│  │   Database      │                    │                         │  │
│  │                 │                    │ • Prometheus/Grafana    │  │
│  │ • Clusters      │                    │ • ELK Stack             │  │
│  │ • Services      │                    │ • Jaeger Tracing        │  │
│  │ • Mappings      │                    │ • AlertManager          │  │
│  │ • Users         │                    │ • Health Checks         │  │
│  │ • Certificates  │                    │                         │  │
│  └─────────────────┘                    └─────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────┘
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
| **Proxy Instances** | Up to 3 | Unlimited* |
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
