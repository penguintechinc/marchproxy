# MarchProxy ALB - Release Notes

## [Unreleased]

### Added
- Placeholder for next features

### Changed
- Placeholder for next changes

### Fixed
- Placeholder for next fixes

### Deprecated
- None

### Removed
- None

### Security
- None

---

## [v1.0.0] - 2025-01-01

### Added
- Initial ALB supervisor with Envoy proxy management
- gRPC ModuleService API with full control plane interface
  - `GetStatus` - Retrieve health and operational status
  - `GetRoutes` - Fetch route configuration from xDS
  - `ApplyRateLimit` - Configure rate limiting per route
  - `GetMetrics` - Retrieve performance metrics
  - `SetTrafficWeight` - Configure blue/green and canary deployments
  - `Reload` - Graceful configuration reload
- HTTP health check endpoints
  - `/healthz` - Liveness probe
  - `/ready` - Readiness probe
- Prometheus metrics endpoint (`/metrics`)
  - Connection and request counters
  - Latency percentiles (P50, P90, P95, P99)
  - Per-route metrics
  - HTTP status code distribution
- xDS client for dynamic configuration updates
- Envoy process lifecycle management
  - Graceful startup and shutdown
  - Configuration reload support
  - Health monitoring
- Comprehensive logging with structured JSON output
- Docker container image with multi-stage build
  - Go supervisor compilation
  - Protobuf code generation
  - Envoy 1.28 base image
- Configuration via environment variables with validation
- gRPC keepalive and connection management
- Graceful shutdown with configurable timeout

### Infrastructure
- GitHub Actions CI/CD pipeline with linting and testing
- Dockerfile with security hardening (non-root user)
- Protobuf service definition and code generation

### Documentation
- API.md - gRPC and REST endpoint documentation
- TESTING.md - Unit and integration testing guide
- CONFIGURATION.md - Environment variables and Docker setup
- USAGE.md - Quick start and deployment guide

---

## Version Format

Version follows semantic versioning: `vMAJOR.MINOR.PATCH`

- **MAJOR** - Breaking changes, incompatible API changes
- **MINOR** - New features, backward compatible
- **PATCH** - Bug fixes, security patches, backward compatible

---

## Upgrading

### v1.0.0 from previous versions

This is the initial release. No upgrades available.

### Breaking Changes

None - initial release.

### Migration Guide

None - initial release.

---

## Known Issues

None currently identified.

---

## Future Roadmap

### Planned Features
- TLS/mTLS support for gRPC and Envoy communication
- Advanced traffic management (circuit breakers, retries)
- Enhanced metrics and distributed tracing integration
- Multi-ALB clustering and state synchronization
- Custom filter plugin support
- Performance optimization (AF_XDP, XDP acceleration)

### Under Investigation
- IPv6 support improvements
- UDP traffic handling enhancements
- Advanced routing rule patterns

---

## Support

For issues and questions:
- GitHub: [MarchProxy Issues](https://github.com/PenguinTech/MarchProxy/issues)
- Documentation: See `/docs` folder
- API Reference: See `docs/API.md`

---

## Contributors

Initial contributors:
- PenguinTech Engineering Team

---

## License

MarchProxy is licensed under the Limited AGPL3 with preamble for fair use.
See LICENSE file in repository root.
