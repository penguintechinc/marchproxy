# MarchProxy API Server - Release Notes

Release history and version documentation for the MarchProxy API Server.

## Version Format

Versions follow semantic versioning: `vMajor.Minor.Patch.Build`

- **Major**: Breaking changes, API changes, removed features
- **Minor**: Significant new features and functionality additions
- **Patch**: Minor updates, bug fixes, security patches
- **Build**: Epoch64 timestamp of build time

## Current Version

**v1.0.0** - Initial Release

Released: 2024-01-15

### Overview

Enterprise-grade API server with integrated xDS control plane for dynamic Envoy proxy configuration.

### Core Features

#### ✓ FastAPI Backend
- High-performance async Python application
- Automatic API documentation (Swagger/ReDoc)
- Request validation and serialization
- CORS middleware for cross-origin requests

#### ✓ Authentication & Security
- JWT token-based authentication (HS256)
- 2FA/TOTP support for enhanced security
- bcrypt password hashing with 12-round cost factor
- Access token (30 min) and refresh token (7 days)
- HTTPBearer security scheme

#### ✓ Database & ORM
- SQLAlchemy 2.0 async ORM
- PostgreSQL with asyncpg driver
- Connection pooling (QueuePool production, NullPool development)
- Alembic for database migrations
- Comprehensive data models with relationships

#### ✓ API Endpoints
- Authentication endpoints (login, register, 2FA)
- User management (CRUD operations)
- Cluster management (multi-cluster support)
- Service management (create, read, update, delete)
- Proxy monitoring (list, status, metrics)
- Certificate management (upload, renewal, tracking)
- Traffic shaping (rate limiting, bandwidth control)

#### ✓ xDS Control Plane Integration
- Integrated Go xDS server
- gRPC protocol for Envoy communication
- Dynamic configuration snapshot generation
- Service change triggering xDS updates
- Snapshot caching and versioning

#### ✓ Monitoring & Observability
- Health check endpoint (`/healthz`)
- Prometheus metrics endpoint (`/metrics`)
- OpenTelemetry instrumentation (optional)
- Structured logging with correlation IDs
- Performance metrics collection

#### ✓ License Integration
- PenguinTech license server validation
- Enterprise feature gating
- License key rotation support
- Multi-tier licensing (Community/Enterprise)

#### ✓ Docker Support
- Multi-stage Dockerfile
- Minimal production image
- Non-root user execution
- Health check configuration
- Pre-built Docker images

### System Requirements

- **Python**: 3.12+
- **PostgreSQL**: 12+
- **Redis**: 6.0+
- **Docker**: 20.10+ (for containerized deployment)
- **Memory**: Minimum 512MB, recommended 2GB+
- **CPU**: Minimum 1 core, recommended 2+ cores

### Breaking Changes

None - Initial release.

### New Features

#### Authentication
- User registration with first-user admin promotion
- JWT token-based authentication
- 2FA/TOTP support with QR code generation
- Token refresh mechanism
- Change password functionality

#### User Management
- Create, read, update, delete users
- Admin and regular user roles
- User status tracking (active/inactive/verified)
- Last login tracking

#### Cluster Management
- Multi-cluster support (Enterprise feature)
- Cluster API key generation and rotation
- Per-cluster configuration
- Syslog integration (auth, netflow, debug logs)

#### Service Management
- Service creation with protocol support (TCP, UDP, HTTPS, HTTP3)
- Authentication types (None, Base64, JWT)
- Health check configuration
- Service token generation and rotation
- xDS configuration triggering

#### Proxy Management
- Proxy registration and heartbeat
- Real-time metrics (CPU, memory, connections, throughput)
- Latency tracking (average, p95)
- Error rate monitoring
- Configuration version tracking

#### Certificate Management
- TLS certificate upload and import
- Certificate chain support
- Expiry tracking and renewal status
- Auto-renewal configuration
- Multi-source certificate support (Infisical, Vault, Upload)

#### Traffic Shaping (Enterprise)
- Bandwidth limiting per service
- Connection limiting
- Request rate limiting
- Per-service traffic rules

#### xDS Control Plane
- Integrated Go gRPC server
- Dynamic Envoy configuration
- Snapshot-based configuration delivery
- Multi-node support
- Configuration versioning

### Improvements

#### Security
- Implemented comprehensive input validation
- bcrypt password hashing (4.0.1)
- JWT token security with configurable expiry
- CORS middleware configuration
- License key validation

#### Performance
- Async/await throughout application
- Database connection pooling
- Redis caching support
- xDS snapshot caching
- Multi-worker support (Uvicorn)

#### Reliability
- Comprehensive error handling
- Database transaction management
- Graceful shutdown handling
- Health check mechanisms
- Automatic database table creation

#### Observability
- Prometheus metrics endpoint
- Structured logging
- OpenTelemetry instrumentation ready
- Request/response logging
- Performance monitoring

### Fixed Issues

None - Initial release.

### Known Limitations

1. **Rate Limiting**: Basic rate limiting (to be enhanced in v1.1)
2. **SAML/OAuth**: Not implemented (Enterprise feature, planned v1.1)
3. **Hardware Acceleration**: Delegated to Go proxy (not applicable to API server)
4. **WebSocket Tunneling**: Planned for v1.1+
5. **Advanced Routing**: Basic service routing implemented, advanced rules planned v1.1+

### Deprecated Features

None - Initial release.

### Dependencies

Key dependencies:
- FastAPI 0.109.0
- SQLAlchemy 2.0.25
- Uvicorn 0.27.0
- asyncpg 0.29.0
- pydantic 2.5.3
- python-jose 3.3.0
- bcrypt 4.0.1
- pyotp 2.9.0

See `requirements.txt` for complete list.

### Security Advisories

No security advisories for this release.

### Migration Guide

Not applicable - Initial release.

### Upgrade Instructions

Not applicable - Initial release.

### Getting Started

1. **Install**: `docker pull marchproxy-api-server:v1.0.0`
2. **Configure**: Set environment variables (see CONFIGURATION.md)
3. **Run**: `docker run -d -p 8000:8000 ... marchproxy-api-server:v1.0.0`
4. **Register**: Create admin user via `/api/v1/auth/register`
5. **Access**: Open `http://localhost:8000/api/docs` for API documentation

### Support & Resources

- **Documentation**: See `docs/` folder
- **API Reference**: `/api/docs` (Swagger UI)
- **Issues**: https://github.com/penguintech/marchproxy/issues
- **Community**: https://www.penguintech.io

### Credits

Developed by PenguinTech for MarchProxy enterprise egress proxy management.

---

## Version History (Planned)

### v1.1.0 (Planned)

- SAML/OAuth2 authentication
- Advanced rate limiting and throttling
- WebSocket/HTTP upgrade tunneling
- QUIC/HTTP3 support
- Advanced routing rules
- Policy-based access control
- Multi-cloud integration enhancements

### v1.2.0 (Planned)

- Service mesh integration (Istio)
- Advanced observability (tracing, profiling)
- ML-based anomaly detection
- Automated backup and disaster recovery
- Multi-region deployment support

### v2.0.0 (Future)

- GraphQL API support
- Event-driven architecture
- Streaming analytics
- Advanced AI/ML features
- Zero-trust security model enhancements

---

## Changelog

### 2024-01-15: v1.0.0 Release

#### Added
- FastAPI-based REST API server
- User authentication with JWT and 2FA
- User management endpoints
- Cluster management with multi-cluster support
- Service management with xDS integration
- Proxy management and monitoring
- Certificate management
- Traffic shaping for Enterprise tier
- Health check and metrics endpoints
- Prometheus metrics support
- PostgreSQL database backend
- Redis caching support
- OpenTelemetry instrumentation hooks
- Docker containerization
- Comprehensive API documentation
- Unit and integration test framework
- CI/CD pipeline integration

#### Infrastructure
- Multi-stage Docker build for minimal images
- Non-root user execution in containers
- Health check configuration
- Connection pooling optimization
- Database transaction management

#### Documentation
- API reference (API.md)
- Testing guide (TESTING.md)
- Configuration reference (CONFIGURATION.md)
- Usage guide (USAGE.md)
- Release notes (this file)
- Architecture documentation
- Database migration guide

---

## Issues & Feedback

### Reporting Issues

Found a bug? Please report it:

1. Check existing issues: https://github.com/penguintech/marchproxy/issues
2. Create new issue with:
   - API Server version
   - Steps to reproduce
   - Expected vs actual behavior
   - Logs (if applicable)
   - Environment details

### Feature Requests

Have a feature idea? Share it:

1. Check planned features above
2. Create discussion or issue with details
3. Include use case and expected behavior

### Security Issues

Found a security vulnerability? Please report responsibly:

Email: security@penguintech.io (not public issues)

---

## Compatibility

### Python Versions
- ✓ 3.12
- ✓ 3.13 (when available)

### Database
- ✓ PostgreSQL 12+
- ✓ PostgreSQL 13+
- ✓ PostgreSQL 14+
- ✓ PostgreSQL 15+

### Operating Systems
- ✓ Linux (Ubuntu, Debian, CentOS, RHEL)
- ✓ macOS (Intel and Apple Silicon)
- ✓ Windows (Docker/WSL2)

### Container Runtimes
- ✓ Docker 20.10+
- ✓ Podman 3.0+
- ✓ Kubernetes 1.20+

---

## License

Limited AGPL3 with PenguinTech preamble for fair use.

See LICENSE file for full terms.

---

## Acknowledgments

Built with modern Python stack:
- FastAPI - Modern web framework
- SQLAlchemy - Database ORM
- Pydantic - Data validation
- Uvicorn - ASGI server
- PostgreSQL - Reliable database
- Redis - Caching and sessions

---

## Support & Contact

- **Website**: https://www.penguintech.io
- **GitHub**: https://github.com/penguintech/marchproxy
- **Issues**: https://github.com/penguintech/marchproxy/issues
- **Email**: support@penguintech.io

---

## Version History Timeline

```
2024-01-15  v1.0.0   Initial Release
2024-03-xx  v1.1.0   Advanced Features (Planned)
2024-06-xx  v1.2.0   Integrations & AI (Planned)
2025-00-xx  v2.0.0   Next Generation (Future)
```

Last updated: 2024-01-15
