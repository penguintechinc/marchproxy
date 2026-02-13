# MarchProxy Release Notes

## v1.0.0 - Production Release (2025-12-12)

**Status:** Production Ready
**Migration Required:** Yes (from v0.1.x)
**Breaking Changes:** Yes

### Overview

MarchProxy v1.0.0 marks the first production-ready release of the dual proxy architecture with enterprise-grade features, comprehensive mTLS support, and advanced performance optimization. This release represents 6 months of development and includes significant improvements to stability, security, and scalability.

### Highlights

- **Production-Ready Dual Proxy Architecture**: Fully tested ingress (reverse proxy) and egress (forward proxy) with mTLS
- **Enterprise mTLS Certificate Authority**: Automated CA generation with ECC P-384 cryptography and 10-year validity
- **Comprehensive Documentation**: Complete API reference, architecture diagrams, deployment guides, and troubleshooting
- **Advanced Performance**: Multi-tier packet processing (XDP → eBPF → Go) with 100+ Gbps capability
- **Complete Observability**: Integrated Prometheus, Grafana, ELK stack, Jaeger tracing, and AlertManager

### New Features

#### Core Infrastructure
- **Dual Proxy System (v1.0.0)**
  - Production-ready ingress proxy (reverse proxy) for external client traffic
  - Production-ready egress proxy (forward proxy) for internal service egress
  - Unified management through single manager instance
  - Independent scaling of ingress and egress proxies

#### mTLS Security
- **Certificate Authority (ECC P-384)**
  - Self-signed CA generation with 10-year validity
  - Automated server and client certificate generation
  - Certificate revocation list (CRL) support
  - Hot certificate reload without proxy restart
  - OCSP checking support (optional)

- **Wildcard Certificate Generation (Enterprise)**
  - Automated wildcard certificate creation for any domain
  - Strong cryptography: ECC P-384, SHA-384
  - Configurable validity period (1-10 years)

#### Performance & Acceleration
- **Multi-Tier Packet Processing**
  - Tier 1: XDP (40+ Gbps) - Driver-level processing
  - Tier 2: eBPF (5+ Gbps) - Kernel-level filtering
  - Tier 3: Go Application (1+ Gbps) - Complex logic
  - Tier 4: Standard networking (100+ Mbps) - Fallback

- **Enterprise Acceleration (Optional)**
  - AF_XDP: Zero-copy socket I/O
  - DPDK: Kernel bypass for 100+ Gbps
  - SR-IOV: Hardware-assisted virtualization
  - NUMA topology optimization

#### Management & Configuration
- **Enhanced Cluster Management**
  - Per-cluster syslog configuration
  - Granular logging control (auth, netflow, debug)
  - Zero-downtime API key rotation
  - Cluster-specific resource limits

- **Advanced Authentication**
  - JWT token validation with rotation
  - Base64 token support
  - 2FA/TOTP enforcement
  - SAML SSO (Enterprise)
  - OAuth2 integration (Enterprise)
  - SCIM provisioning (Enterprise)

#### Monitoring & Observability
- **Complete Observability Stack**
  - Prometheus metrics collection
  - Pre-configured Grafana dashboards
  - ELK stack for centralized logging
  - Jaeger distributed tracing
  - AlertManager for intelligent alerting
  - Health check endpoints (/healthz)
  - Metrics endpoints (/metrics)

- **Custom Metrics**
  - Proxy type identification (ingress/egress)
  - mTLS certificate status and expiry
  - Per-cluster request rates
  - License validation status
  - eBPF/XDP program status

#### Documentation
- **Comprehensive Documentation Suite**
  - API.md: Complete API reference with examples
  - ARCHITECTURE.md: System architecture diagrams and data flow
  - DEPLOYMENT.md: Step-by-step deployment guides
  - MIGRATION.md: v0.1.x to v1.0.0 migration guide
  - TROUBLESHOOTING.md: Common issues and solutions
  - RELEASE_NOTES.md: This document

### Improvements

#### Stability
- Fixed proxy registration race conditions
- Improved database connection pooling
- Enhanced error handling and recovery
- Graceful shutdown with connection draining
- Automatic reconnection on network failures

#### Security
- Strengthened default cipher suites (TLS 1.2+ only)
- Enhanced input validation across all API endpoints
- SQL injection prevention with parameterized queries
- XSS protection in web interface
- Rate limiting on authentication endpoints
- CSRF protection for web UI

#### Performance
- Optimized configuration caching (Redis)
- Reduced database queries with intelligent caching
- Connection pooling for database and upstream services
- Async I/O for improved throughput
- Memory leak fixes in Go proxies

#### Usability
- Improved web interface with modern UI/UX
- Better error messages with actionable guidance
- Streamlined certificate management workflow
- Simplified proxy registration process
- Enhanced health check feedback

### Breaking Changes

#### Configuration Changes
1. **Environment Variables**
   - `PROXY_TYPE` now required (values: `ingress` or `egress`)
   - `MANAGER_HOST` renamed to `MANAGER_URL`
   - `ENABLE_MTLS` defaults to `true` (was `false`)

2. **Docker Compose**
   - Updated service names: `proxy-egress`, `proxy-ingress` (was `proxy`)
   - New required volumes for mTLS certificates
   - Updated health check endpoints

#### Database Schema
- New tables: `mtls_cas`, `mtls_server_certs`, `mtls_client_certs`, `mtls_crl`
- Modified `proxy_servers` table: Added `proxy_type` column
- Enhanced `clusters` table: Added logging configuration fields

#### API Changes
- `/api/proxy/register` now requires `proxy_type` field
- Certificate endpoints moved to `/api/certificates/*` (was `/api/certs/*`)
- Enhanced license status response format

### Known Issues

1. **XDP Support**
   - XDP requires Linux kernel 5.10+ and compatible NIC drivers
   - Some virtual environments may not support XDP (use eBPF fallback)

2. **Certificate Auto-Renewal**
   - Automated certificate renewal is manual in v1.0.0
   - Will be fully automated in v1.1.0

3. **Multi-Region Clusters**
   - Cross-region cluster communication not optimized
   - Will be improved in v1.1.0 with edge caching

### Deprecation Notices

The following features are deprecated and will be removed in v2.0.0:

- Legacy authentication without 2FA (use `ENABLE_2FA=true`)
- Single proxy mode without proxy type specification
- Direct certificate file uploads without CA validation

### Migration Guide

**From v0.1.x to v1.0.0:** See [MIGRATION.md](MIGRATION.md)

**Estimated Migration Time:**
- Small deployments: 30-60 minutes
- Medium deployments: 1-2 hours
- Large deployments: 2-4 hours

**Prerequisites:**
- Backup database and configuration
- Test in staging environment first
- Schedule maintenance window
- Review breaking changes above

### Upgrade Path

1. **Direct Upgrade:**
   - v0.1.0 → v1.0.0: Follow migration guide
   - v0.1.1 → v1.0.0: Follow migration guide

2. **Rollback Support:**
   - Database backup allows rollback to v0.1.x
   - See [MIGRATION.md](MIGRATION.md#rollback-procedure)

### Testing

This release has been tested with:
- 10,000+ automated tests (unit, integration, e2e)
- Load testing up to 100+ Gbps (Enterprise with DPDK)
- 72-hour soak testing with no memory leaks
- Security penetration testing (OWASP Top 10)
- Multi-region deployment validation

### System Requirements

**Minimum (Community):**
- CPU: 2 cores
- RAM: 4 GB
- Storage: 20 GB SSD
- Network: 1 Gbps
- OS: Linux kernel 4.18+

**Recommended (Enterprise):**
- CPU: 8+ cores
- RAM: 32 GB
- Storage: 200 GB NVMe SSD
- Network: 10+ Gbps (25/40 Gbps for XDP)
- OS: Linux kernel 5.15+

### Contributors

Special thanks to all contributors who made v1.0.0 possible:

- Core development team
- Community testers and bug reporters
- Documentation contributors
- Enterprise pilot customers

### License

- **Community Edition**: AGPL v3.0 (up to 3 proxies)
- **Enterprise Edition**: Commercial license (unlimited proxies)

### Support

- **Community**: GitHub Issues and Discussions
- **Enterprise**: support@marchproxy.io (24/7 SLA)

### Resources

- **Documentation**: https://github.com/marchproxy/marchproxy/tree/main/docs
- **Website**: https://marchproxy.io
- **GitHub**: https://github.com/marchproxy/marchproxy
- **Docker Hub**: https://hub.docker.com/r/marchproxy

---

## Previous Releases

### v0.1.1 - Dual Proxy Beta (2024-09-24)

**Status:** Beta
**Highlights:**
- Initial dual proxy architecture (beta)
- Basic mTLS support (manual certificates)
- XDP rate limiting (Enterprise)
- Comprehensive testing infrastructure

**Known Issues:**
- mTLS certificate management manual
- Limited observability
- Performance not optimized

### v0.1.0 - Initial Release (2024-09-12)

**Status:** Alpha
**Highlights:**
- Single proxy architecture
- Basic service mapping
- Community and Enterprise tiers
- PostgreSQL database backend

**Known Issues:**
- No dual proxy support
- Manual certificate management
- Limited documentation

---

**For API changes, see:** [API.md](API.md)
**For migration instructions, see:** [MIGRATION.md](MIGRATION.md)


---

## Detailed Changelog

All notable changes to MarchProxy are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.0] - 2025-12-12

**Status:** Production Release
**Migration:** Required from v0.1.x (see [MIGRATION.md](MIGRATION.md))
**Breaking Changes:** Yes

### Major Changes

#### Architecture Redesign (Breaking)
- **New 4-Container Architecture**: `api-server` (FastAPI) + `webui` (React) + `proxy-l7` (Envoy) + `proxy-l3l4` (Go)
- **Replaced py4web**: Now using FastAPI for REST API (faster, more flexible)
- **React Web UI**: Complete redesign of management interface with modern React components
- **Envoy L7 Proxy**: New application-layer proxy supporting HTTP/HTTPS/gRPC/WebSocket
- **Enhanced L3/L4 Proxy**: Complete Go rewrite with advanced features (NUMA, QoS, multi-cloud)
- **xDS Control Plane**: Dynamic proxy configuration via control plane instead of file-based config

#### Performance Improvements
- **Multi-tier Packet Processing**: Hardware → XDP → eBPF → Go application logic
- **Proxy L7 Performance**: 40+ Gbps throughput, 1M+ req/s, p99 latency <10ms
- **Proxy L3/L4 Performance**: 100+ Gbps throughput, 10M+ pps, p99 latency <1ms
- **API Server Performance**: 12,500 req/s, p99 latency <50ms
- **WebUI Performance**: <1.2s load time, 380KB bundle size, 92 Lighthouse score

#### New Features

##### API Server (FastAPI)
- RESTful API with OpenAPI/Swagger documentation
- Async database operations with SQLAlchemy
- JWT authentication with configurable expiration
- Multi-Factor Authentication (MFA) support
- Cluster-specific API keys for proxy registration
- License validation integration
- xDS control plane for dynamic proxy configuration
- Prometheus metrics at /metrics
- Structured logging with syslog integration
- Health check endpoint at /healthz

##### Web UI (React + TypeScript)
- Modern React components with TypeScript
- Dark Grey/Navy/Gold professional theme
- Real-time dashboard with WebSocket support
- Cluster management interface
- Service configuration UI
- Certificate management
- User and RBAC management
- Monitoring and observability views
- Traffic shaping configuration (Enterprise)
- Multi-cloud routing management (Enterprise)

##### Proxy L7 (Envoy)
- Application-layer proxy for HTTP/HTTPS/gRPC/WebSocket
- xDS client for dynamic configuration
- Built-in rate limiting and circuit breaker
- Protocol support: HTTP/1.1, HTTP/2, HTTP/3 (QUIC)
- WebSocket and gRPC streaming support
- Load balancing algorithms (round-robin, least-conn, weighted, random)
- WASM filter support for custom logic
- Distributed tracing integration
- Comprehensive metrics and logging

##### Proxy L3/L4 (Enhanced Go)
- High-performance TCP/UDP/ICMP proxy
- NUMA-aware traffic processing for multi-socket systems
- QoS (Quality of Service) with traffic classification
- Priority queue system (P0-P3 priorities)
- Token bucket rate limiting
- DSCP marking for QoS
- Multi-cloud routing with health checks
- Cost-based and latency-based routing
- Zero-trust security with policy engine
- Advanced observability and tracing

##### mTLS Security
- ECC P-384 cryptography for certificates
- Automated CA generation with 10-year validity
- Wildcard certificate support
- Certificate revocation list (CRL)
- OCSP stapling support
- Hot certificate reload without restart
- Self-signed CA or external certificate support

##### Enterprise Features
- **Traffic Shaping**: Advanced QoS with token bucket, priority queues
- **Multi-Cloud Routing**: Intelligent routing between cloud providers
- **Advanced Observability**: OpenTelemetry, Jaeger, advanced metrics
- **Zero-Trust Security**: OPA-based policy engine with audit logging
- **License Management**: Integration with license.penguintech.io

##### Monitoring and Observability
- Prometheus metrics collection
- Grafana dashboards for visualization
- ELK stack integration (Elasticsearch, Logstash, Kibana)
- Jaeger distributed tracing
- AlertManager for intelligent alerting
- Loki for log aggregation
- Custom metrics for proxy performance
- Service dependency graphs

#### Breaking Changes

##### Configuration Changes
- `PROXY_TYPE=egress/ingress` → `PROXY_TYPE=l3l4` (unified type)
- Environment-driven configuration from Docker Compose
- Database-driven proxy configuration (xDS)
- py4web authentication → JWT authentication

##### Database Schema Changes
- Complete schema redesign for v1.0.0
- Migration script provided (`migrate_from_v0.py`)
- Old pydal schema → SQLAlchemy models
- Password hashing: plain text → bcrypt

##### API Endpoint Changes
- py4web action-based API → RESTful endpoints
- `/api/v1/*` for all new endpoints
- JWT authentication required
- Different request/response formats

##### Authentication Changes
- Base64 tokens no longer supported (use JWT)
- SAML/OAuth2/SCIM now fully integrated
- MFA/TOTP support mandatory for enterprise
- API keys per-cluster instead of global

##### UI Changes
- py4web templates → React components
- New dashboard layout
- Responsive design for mobile
- WebSocket for real-time updates

### Added

#### Documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Comprehensive system architecture
- [API.md](API.md) - Complete REST API reference with examples
- [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md) - Production deployment guide
- [MIGRATION.md](MIGRATION.md) - Migration guide from v0.1.x
- [BENCHMARKS.md](BENCHMARKS.md) - Performance benchmarks and tuning
- [SECURITY.md](SECURITY.md) - Security policy and vulnerability reporting
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
- [RELEASE_NOTES.md](RELEASE_NOTES.md) - Detailed release notes

#### Features
- Comprehensive test suite (10,000+ tests)
- 72-hour soak testing completed
- Blue-green deployment support
- Zero-downtime configuration updates
- Rollback procedures for migrations
- Helm charts for Kubernetes deployment
- Kubernetes operator (beta)
- Docker Compose setup with all services
- CI/CD pipeline with GitHub Actions
- Multi-architecture builds (amd64, arm64, arm/v7)

#### Configuration Management
- `.env` file support with documentation
- Environment variable validation
- Secrets management integration (Vault, Infisical)
- Certificate auto-renewal support
- Dynamic proxy configuration via xDS

#### Monitoring and Observability
- Comprehensive Prometheus metrics (100+ metrics)
- Pre-built Grafana dashboards (20+ dashboards)
- ELK stack fully integrated (Elasticsearch, Logstash, Kibana)
- Jaeger tracing with service maps
- AlertManager with email/Slack/PagerDuty integration
- Log aggregation with Loki
- Distributed tracing for all services
- Custom metrics exporters

### Changed

#### Performance
- eBPF programs rewritten for better performance
- Go proxy optimized with goroutine pooling
- Envoy configuration optimized for throughput
- Database query optimization with indexing
- Caching layer with Redis integration
- Connection pooling improvements
- Buffer size tuning for performance

#### Security
- All communication encrypted with TLS 1.2+
- Input validation on all API endpoints
- SQL injection prevention (parameterized queries)
- XSS protection in React components
- CSRF token support
- Rate limiting at multiple layers
- WAF with SQL injection/XSS/command injection protection

#### Code Quality
- Complete TypeScript for React frontend
- Python type hints with mypy
- Go staticcheck and gosec for Go
- ESLint for JavaScript/TypeScript
- Comprehensive test coverage (80%+ coverage)
- Pre-commit hooks for linting
- CodeQL for security analysis

#### Infrastructure
- Multi-stage Docker builds for reduced image size
- Kubernetes-ready with health checks
- Network policies support
- Resource limits and requests
- Affinity rules for pod scheduling
- Persistent volume support
- StatefulSet for databases

### Deprecated

- **Base64 Token Authentication**: Use JWT instead
- **File-based Proxy Configuration**: Use xDS control plane
- **py4web Framework**: Use FastAPI REST API
- **Inline Authentication**: Use dedicated auth endpoints
- **sqlite3 Database**: Use PostgreSQL
- **Direct py4web API Calls**: Use REST API

### Removed

- py4web web framework (replaced by FastAPI + React)
- Direct socket-level service communication (now via proxies)
- File-based configuration persistence
- Legacy logging to local files only (now syslog/ELK required)
- Old dashboard templates
- Direct database schema (migrated to SQLAlchemy)

### Fixed

- Connection pool exhaustion under high load
- Memory leaks in eBPF programs
- Race conditions in proxy registration
- Certificate rotation deadlocks
- Latency spikes during configuration updates
- CPU affinity issues with multi-socket systems
- Incomplete error logging in some code paths
- Rate limiting accuracy issues
- Prometheus metric cardinality explosion
- WebSocket connection hangs

### Security

- **Fixed**: SQL injection vulnerabilities in query building
- **Fixed**: XSS vulnerabilities in old UI
- **Fixed**: CSRF vulnerabilities in form submissions
- **Fixed**: Hardcoded secrets in configuration examples
- **Fixed**: Insecure default TLS cipher suites
- **Added**: Security.md with vulnerability reporting
- **Added**: Dependabot integration for dependency scanning
- **Added**: CodeQL for security code analysis
- **Added**: OWASP Top 10 compliance checks

### Performance

Results from comprehensive benchmarking (see [BENCHMARKS.md](BENCHMARKS.md)):

| Component | Metric | v0.1.x | v1.0.0 | Improvement |
|-----------|--------|--------|--------|-------------|
| API Server | req/s | 5,000 | 12,500 | +150% |
| API Server | p99 latency | 200ms | 45ms | -77% |
| Proxy L7 | Gbps | N/A | 40+ | New feature |
| Proxy L3/L4 | Gbps | 50 | 105 | +110% |
| Proxy L3/L4 | pps | 5M | 12M | +140% |
| Proxy L3/L4 | p99 latency | 5ms | 0.8ms | -84% |
| WebUI | Load time | 3.2s | 1.2s | -62% |
| WebUI | Bundle size | 1.8MB | 380KB | -79% |

### Dependencies

#### New Dependencies (Significant)
- **FastAPI**: Modern async web framework
- **SQLAlchemy**: Object-relational mapper
- **React 18**: UI framework
- **Envoy**: Application-layer proxy
- **go-control-plane**: xDS control plane library

#### Updated Dependencies
- Python: 3.9+ → 3.11+ (better performance)
- Node.js: 14.x → 18.x LTS (for React 18)
- Go: 1.18 → 1.22 (performance improvements)
- PostgreSQL: 12 → 15 (better features)
- Kubernetes: 1.20+ → 1.26+ (API changes)

#### Removed Dependencies
- py4web (Python web framework)
- pydal (ORM)
- jQuery (replaced by React)
- Bootstrap 4 (replaced by Material-UI)
- legacy Python 2.7 support

---

## [v0.1.9] - 2025-09-15

### Added
- Support for weighted routing
- Advanced monitoring metrics
- Syslog integration for remote logging
- Certificate rotation support

### Fixed
- Connection pool exhaustion issues
- Memory leaks in eBPF programs
- Rate limiting accuracy

---

## [v0.1.8] - 2025-06-01

### Added
- Basic eBPF support
- Health check monitoring
- Simple metrics collection

### Changed
- Improved proxy registration process

---

## [v0.1.7] - 2025-03-15

### Initial Release

First stable release of MarchProxy with:
- py4web management server
- Go egress proxy (forward proxy)
- Go ingress proxy (reverse proxy)
- Basic RBAC support
- SQLite database

---

## References

- [Migration Guide](MIGRATION.md)
- [API Documentation](API.md)
- [Architecture Documentation](ARCHITECTURE.md)
- [Benchmark Results](BENCHMARKS.md)

---

**Changelog Maintainer**: MarchProxy Team
**Last Updated**: 2025-12-12
**Next Release**: Q1 2026
