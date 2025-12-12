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

**For detailed changelog, see:** [CHANGELOG.md](CHANGELOG.md)
**For API changes, see:** [API.md](API.md)
**For migration instructions, see:** [MIGRATION.md](MIGRATION.md)
