# Release Notes

Version history and release information for MarchProxy Manager.

## [v0.1.0] - 2025-01-16

### Initial Release

First release of MarchProxy Manager with core functionality.

#### Added

**Core Features:**
- User authentication with JWT tokens
- Two-factor authentication (2FA) support
- Role-based access control (RBAC)
- Cluster management and isolation
- Proxy server registration and heartbeat monitoring
- Service configuration and management
- Service-to-IP mappings
- mTLS certificate management
- API key generation and rotation
- License management (Community and Enterprise tiers)

**Authentication:**
- Local username/password authentication
- JWT token generation and refresh
- 2FA with TOTP and QR codes
- OAuth2 support (Google, GitHub, Azure)
- SAML 2.0 support for enterprise SSO
- SCIM user provisioning

**API Endpoints:**
- RESTful API with full CRUD operations
- Comprehensive error handling
- Rate limiting (10 req/min auth, 100 req/min standard)
- Pagination support
- Prometheus metrics endpoint (/metrics)
- Health check endpoint (/healthz)

**Database:**
- PostgreSQL support (any pydal-compatible DB)
- Automatic schema migration
- Connection pooling
- Transaction support

**Administration:**
- User management
- Cluster creation and configuration
- Proxy management dashboard
- Service configuration
- License status monitoring

**Monitoring:**
- Prometheus metrics integration
- Health check endpoint
- Syslog centralized logging
- Per-cluster log configuration
- Request/response logging

**Security:**
- Input validation on all endpoints
- CSRF protection
- SQL injection prevention (pydal ORM)
- XSS protection
- Password hashing with bcrypt
- Secure token generation
- Cluster-specific API key isolation

**Infrastructure:**
- Docker containerization with multi-stage builds
- Environment variable configuration
- Database connection pooling
- Health checks
- Non-root container execution
- Development and testing stages

#### Changed
- N/A (initial release)

#### Fixed
- N/A (initial release)

#### Removed
- N/A (initial release)

#### Deprecated
- N/A (initial release)

#### Security
- All authentication implementations include password strength validation
- JWT secrets required (32+ characters in production)
- Database passwords stored as environment variables
- API keys generated with cryptographically secure random source
- mTLS support for secure proxy communication

#### Known Limitations
- Community edition limited to 3 proxy servers
- Maximum 50 items per page in list endpoints
- License validation requires external network access
- SAML requires certificate configuration

---

## Compatibility Matrix

| Component | Version | Status |
|-----------|---------|--------|
| Python | 3.12+ | Supported |
| PostgreSQL | 12+ | Supported |
| py4web | 1.20230507.1+ | Required |
| pydal | 20230521.1+ | Required |

---

## Migration Guide

### From Community to Enterprise

To upgrade from Community to Enterprise edition:

1. Obtain valid Enterprise license key
2. Set environment variable:
   ```bash
   export LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
   ```
3. Set release mode:
   ```bash
   export RELEASE_MODE=true
   ```
4. Restart manager service
5. All enterprise features now available

### Database Upgrades

Manager uses automatic schema migration. Simply restart with new version:

```bash
docker pull marchproxy-manager:latest
docker-compose up manager  # Automatic migration runs
```

No manual migration scripts required.

---

## Breaking Changes

**None in v0.1.0** (initial release)

Future releases will follow semantic versioning:
- MAJOR: Breaking API changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes (backward compatible)

---

## Upgrade Instructions

### From Previous Version

1. Backup database:
   ```bash
   pg_dump -U marchproxy marchproxy > backup.sql
   ```

2. Update image:
   ```bash
   docker pull marchproxy-manager:v0.1.0
   ```

3. Update docker-compose.yml version tag

4. Restart service:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

5. Verify health:
   ```bash
   curl http://localhost:8000/healthz
   ```

### Rollback Instructions

If issues occur:

1. Restore database:
   ```bash
   psql -U marchproxy < backup.sql
   ```

2. Revert image version in docker-compose.yml

3. Restart with previous version:
   ```bash
   docker-compose up -d
   ```

---

## Versioning Policy

MarchProxy Manager follows Semantic Versioning 2.0.0:

**vMAJOR.MINOR.PATCH.BUILD**

- **MAJOR**: Incompatible API changes, breaking changes
- **MINOR**: New functionality (backward compatible)
- **PATCH**: Bug fixes and security patches
- **BUILD**: Epoch timestamp at build time

Build timestamp format: Unix epoch (seconds since Jan 1, 1970)

Example: `v1.2.3.1737706313` represents v1.2.3 built at 2025-01-24 09:51:53 UTC

---

## Security Updates

### Reporting Security Issues

Do NOT create public GitHub issues for security vulnerabilities.

Please email: security@penguintech.io

Include:
- Vulnerability description
- Affected versions
- Reproduction steps
- Suggested fix (if available)

### Security Commitment

- Critical security issues: Patched within 24 hours
- High severity issues: Patched within 1 week
- Medium severity issues: Patched in next release

---

## Feature Roadmap

### Planned for v0.2.0
- [ ] User audit logging
- [ ] Advanced clustering
- [ ] Backup and restore automation
- [ ] Webhook support
- [ ] GraphQL API endpoint
- [ ] Enhanced monitoring dashboard

### Planned for v0.3.0
- [ ] Multi-region support
- [ ] Advanced rate limiting policies
- [ ] Custom authentication providers
- [ ] Database query optimization
- [ ] API versioning strategy

### Under Consideration
- [ ] WebSocket support
- [ ] gRPC API endpoint
- [ ] Machine learning-based threat detection
- [ ] Advanced traffic analytics
- [ ] Custom protocol support

---

## Dependencies

### Core Dependencies
- py4web >= 1.20230507.1
- pydal >= 20230521.1
- psycopg2-binary >= 2.9.9
- PyJWT >= 2.8.0
- cryptography >= 41.0.7

### Authentication
- python-saml >= 2.5.0
- authlib >= 1.3.0
- pyotp >= 2.9.0

### Monitoring
- prometheus-client >= 0.19.0
- python-json-logger >= 2.0.7

See `requirements.txt` for complete list.

---

## Support

### Getting Help

- **Documentation**: `/manager/docs/`
- **Issues**: GitHub Issues
- **Security**: security@penguintech.io
- **Email**: support@penguintech.io

### Version Support

| Version | Released | LTS | Support Until |
|---------|----------|-----|---|
| v0.1.0 | 2025-01-16 | Yes | 2026-01-16 |

LTS versions receive critical security updates for 2 years.

---

## Changelog Details

### v0.1.0 Commits

- Initial project structure
- Database schema implementation
- Authentication API
- Cluster management API
- Proxy registration API
- mTLS certificate management
- License integration
- Docker containerization
- Comprehensive testing
- Documentation

---

## Contributors

MarchProxy Manager v0.1.0 created by PenguinTech team.

For contribution guidelines, see CONTRIBUTING.md in main repository.

---

## License

Limited AGPL3 - See LICENSE file in repository root

**Fair Use Preamble**: This software is available for personal and small-scale deployment at no cost. Commercial use or deployment beyond fair use requires a license from PenguinTech.
