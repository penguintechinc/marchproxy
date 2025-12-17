# DBLB Release Notes

Version history and release notes for MarchProxy Database Load Balancer (DBLB).

## Versioning

DBLB follows semantic versioning: `vMajor.Minor.Patch.Build`

- **Major**: Breaking changes, API changes, removed features
- **Minor**: Significant new features and functionality additions
- **Patch**: Minor updates, bug fixes, security patches
- **Build**: Epoch64 timestamp of build time

## v1.0.0 (TBD)

### Features

#### Multi-Protocol Support
- MySQL 5.7+ with full protocol support
- PostgreSQL 10+ with full protocol support
- MongoDB 4.0+ with wire protocol support
- Redis 6.0+ with full protocol support
- MSSQL 2016+ with full protocol support
- SQLite 3.x for testing and standalone deployments

#### Connection Pooling
- Efficient connection reuse and management
- Configurable pool sizes per route
- Idle connection timeout management
- Connection lifetime limits
- Automatic connection cleanup

#### Rate Limiting
- Per-route connection rate limiting
- Per-route query rate limiting
- Backpressure-based query queueing
- Configurable default rates
- Per-route rate overrides

#### Security Features
- SQL injection pattern detection
- Comment injection detection
- Stacked query detection
- Excessive SQL keyword heuristics
- Configurable security policies
- TLS/SSL support for backend connections

#### Observability
- HTTP health check endpoint (`/healthz`)
- Prometheus metrics endpoint (`/metrics`)
- Detailed status endpoint (`/status`)
- Structured JSON logging
- Distributed tracing support (Jaeger)
- Performance metrics collection

#### gRPC Integration
- Full ModuleService interface implementation
- GetStatus - Module health and status
- Reload - Graceful configuration reload
- Shutdown - Graceful shutdown with timeout
- GetMetrics - Real-time metrics
- HealthCheck - Deep health verification
- GetStats - Detailed statistics

#### Management Integration
- Manager API registration and discovery
- Cluster-based API key authentication
- Configuration synchronization
- Proxy lifecycle management

#### Enterprise Features (Licensed)
- License key validation
- Feature gating based on tier
- License server integration
- Release mode enforcement

### Improvements

- Multi-architecture Docker images (amd64, arm64, arm/v7)
- Production-ready containerization
- Comprehensive test coverage (80%+)
- Security scanning in CI/CD
- Performance benchmarking
- Database-specific optimizations

### Bug Fixes

- Initial release - no prior versions

### Security

- SQL injection detection with configurable patterns
- TLS/SSL support for secure backend connections
- Non-root Docker container execution
- Regular dependency vulnerability scanning
- Secure credential handling via environment variables

### Breaking Changes

- Initial release - no breaking changes

### Dependencies

Key dependencies:
- Go 1.24.x runtime
- sirupsen/logrus for logging
- Prometheus client for metrics
- gRPC for service communication
- Protocol-specific clients (mysql, postgres, mongodb, redis)

### Migration Notes

- Initial release - no migration required
- See CONFIGURATION.md for setup instructions

### Known Issues

None identified in initial release.

### Contributors

- PenguinTech Development Team

---

## Version History Template

Use this template for future releases:

```markdown
## vX.Y.Z (YYYY-MM-DD)

### Features

- Feature 1: Brief description
- Feature 2: Brief description

### Improvements

- Improvement 1: Brief description
- Improvement 2: Brief description

### Bug Fixes

- Bug fix 1: Description of fix
- Bug fix 2: Description of fix

### Security

- Security issue 1: Description and fix
- Security issue 2: Description and fix

### Breaking Changes

- Change 1: What changed and how to migrate
- Change 2: What changed and how to migrate

### Dependencies

- Updated package X from Y to Z
- Added package X for feature Y

### Deprecations

- Deprecated feature/function X (remove in vX.Y.Z)

### Migration Notes

Instructions for users upgrading from previous versions:
- Step 1: Description
- Step 2: Description

### Known Issues

- Issue 1: Description and workaround
- Issue 2: Description and workaround

### Contributors

- @github_username for contribution
- @github_username for contribution
```

## Release Process

### Preparing a Release

1. **Update Version**: Update `.version` file with new version
   ```bash
   echo "v1.0.0" > .version
   ```

2. **Update Release Notes**: Add new version section to RELEASE_NOTES.md
   ```markdown
   ## vX.Y.Z (YYYY-MM-DD)
   ```

3. **Commit Changes**:
   ```bash
   git add .version docs/RELEASE_NOTES.md
   git commit -m "Release vX.Y.Z"
   ```

4. **Push to Main**:
   ```bash
   git push origin main
   ```

### Automatic Release Process

1. **Pre-Release**: GitHub Actions automatically creates pre-release
   - Triggered by version change in `.version` file
   - Creates `vX.Y.Z-pre` GitHub release
   - Builds and pushes images tagged with version

2. **Final Release**: Optional tag for production release
   ```bash
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```
   - Creates production release
   - Tags images with `vX.Y.Z` and `latest`

### Docker Image Tags

**Development Builds:**
- `alpha-<epoch64>` - Feature branch builds
- `beta-<epoch64>` - Develop branch builds

**Version Releases:**
- `v<X.Y.Z>-alpha` - Alpha release
- `v<X.Y.Z>-beta` - Beta release
- `v<X.Y.Z>` - Production release
- `latest` - Latest production release

**Pull Request Builds:**
- `pr-<number>` - PR number
- `<branch>-<sha>` - Branch and commit

### Rollback Procedure

If a release has critical issues:

1. Identify the issue
2. Fix on a branch
3. Bump patch version
4. Follow release process again

Users can rollback by pulling previous image tag:
```bash
docker pull marchproxy/dblb:v1.0.0
docker run marchproxy/dblb:v1.0.0
```

## Upgrade Path

### From Community Edition

No upgrades needed - DBLB is always available at latest version.

### From Previous DBLB Version

Backward compatible within major versions:
- No database migration required
- Configuration files compatible
- gRPC interface stable

To upgrade:
```bash
docker pull marchproxy/dblb:latest
docker stop dblb
docker rm dblb
docker run -d -v /path/to/config.yaml:/app/config.yaml marchproxy/dblb:latest
```

## Support

- **Documentation**: /docs/ folder or https://docs.marchproxy.io
- **Issues**: https://github.com/penguintech/marchproxy/issues
- **Website**: https://www.penguintech.io
- **Email**: support@penguintech.io

## License

Limited AGPL3 with Contributor Employer Exception

Copyright (c) 2024 PenguinTech.io
