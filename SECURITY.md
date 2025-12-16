# MarchProxy Security Policy

## Reporting Security Vulnerabilities

MarchProxy takes security seriously. We appreciate responsible vulnerability disclosure and are committed to working with security researchers to resolve issues quickly.

### How to Report a Vulnerability

**Please DO NOT report security vulnerabilities through public GitHub issues.** Instead, follow these steps:

1. **Email**: Send a detailed vulnerability report to [security@marchproxy.io](mailto:security@marchproxy.io)
2. **Include**:
   - Description of the vulnerability
   - Affected components and versions
   - Steps to reproduce
   - Potential impact assessment
   - Suggested remediation (if any)

3. **Response Time**: We aim to respond within 48 hours and provide status updates every 5 business days

### Coordinated Disclosure

We follow coordinated vulnerability disclosure practices:

- We will work with you to understand and verify the vulnerability
- We will develop and test a fix
- We will create a patch version and security advisory
- We will coordinate the public disclosure timing (typically 90 days from initial report)
- You will be credited in the security advisory (if desired)

## Security Features

### Authentication & Authorization

- **Multi-Factor Authentication (MFA)**: 2FA with TOTP support
- **Enterprise SSO**: SAML 2.0, OAuth2, and SCIM provisioning
- **Role-Based Access Control (RBAC)**: Administrator and Service-owner roles with cluster assignments
- **API Key Management**: Cluster-specific API keys for proxy registration
- **JWT Tokens**: Configurable token expiration and refresh tokens

### Encryption

- **Mutual TLS (mTLS)**: ECC P-384 cryptography with automatic certificate management
- **Certificate Authority**: Self-signed CA generation with 10-year validity
- **TLS 1.2+**: All connections encrypted with TLS 1.2 or higher
- **Perfect Forward Secrecy**: Ephemeral key exchange for all TLS connections
- **OCSP Stapling**: Optional OCSP certificate status checking

### Network Security

- **eBPF Firewall**: Kernel-level packet filtering for both proxies
- **XDP Acceleration**: Driver-level DDoS protection and rate limiting
- **Web Application Firewall (WAF)**: Protection against SQL injection, XSS, and command injection
- **Rate Limiting**: Multi-tier rate limiting (API level, proxy level, hardware level)
- **Circuit Breaker**: Automatic failure detection and recovery

### Data Protection

- **Database Encryption**: PostgreSQL SSL/TLS connections with encrypted credentials
- **Redis Security**: Password-protected Redis with optional SSL/TLS
- **Secrets Management**: Integration with HashiCorp Vault and Infisical
- **Data at Rest**: Encrypted backups and volume encryption options

### Logging & Auditing

- **Comprehensive Audit Logging**: All API calls logged with user context and outcomes
- **Immutable Logs**: Logs stored in Elasticsearch with tamper-proof configuration
- **Log Retention**: Configurable retention periods (default: 7-30 days)
- **Centralized Logging**: Syslog integration with ELK stack for security monitoring
- **Zero-Trust Audit**: Per-request RBAC decisions logged for compliance

## Vulnerability Management

### Dependency Scanning

- **Automated Scanning**: All dependencies scanned with Dependabot and Socket.dev
- **Regular Audits**: Weekly security audits via npm audit, pip-audit, and govulncheck
- **Immediate Updates**: Critical vulnerabilities patched within 24 hours
- **CVE Monitoring**: Active monitoring of NVD and package repositories

### Patch Management

- **Semantic Versioning**: Security patches in patch versions (X.Y.Z)
- **Backwards Compatibility**: Security patches maintain API compatibility
- **Long-term Support**: Bug fixes provided for 2 years after release
- **Emergency Patches**: 0-day vulnerabilities patched immediately

### Supported Versions

| Version | Release Date | End of Life | Security Fixes |
|---------|-------------|-------------|---|
| v1.0.x | 2025-12-12 | 2027-12-12 | Yes |
| v0.1.x | 2025-03-15 | 2025-12-12 | Critical only |

## Security Configuration

### Recommended Production Settings

```yaml
# Environment Configuration
MARCHPROXY_ENV: production
DEBUG: false
LOG_LEVEL: warn

# TLS/mTLS Configuration
TLS_MIN_VERSION: "1.2"
MTLS_ENABLED: true
MTLS_REQUIRE_CLIENT_CERT: true

# Database Security
DATABASE_SSL_MODE: require
DATABASE_POOL_SIZE: 20
DATABASE_TIMEOUT: 30

# API Security
CORS_ENABLED: false
RATE_LIMIT_ENABLED: true
RATE_LIMIT_RPS: 1000
API_KEY_ROTATION_DAYS: 90

# Authentication Security
ACCESS_TOKEN_EXPIRE_MINUTES: 60
REFRESH_TOKEN_EXPIRE_DAYS: 7
SESSION_TIMEOUT_MINUTES: 30
PASSWORD_MIN_LENGTH: 12
PASSWORD_REQUIRE_SPECIAL_CHARS: true

# Audit & Logging
AUDIT_LOGGING_ENABLED: true
LOGS_RETENTION_DAYS: 30
TRACES_RETENTION_DAYS: 7
SECURITY_EVENT_ALERT: true
```

### Hardening Checklist

- [ ] Change all default passwords and secrets
- [ ] Enable mTLS between all components
- [ ] Configure RBAC with least-privilege access
- [ ] Set up centralized logging (ELK stack)
- [ ] Enable audit logging for all API calls
- [ ] Configure rate limiting appropriate to your scale
- [ ] Set up network firewalls and network policies
- [ ] Use Vault/Infisical for secrets management
- [ ] Enable HTTPS/TLS for all external connections
- [ ] Configure monitoring and alerting
- [ ] Set up backups with encryption
- [ ] Enable WAF rules for application layer
- [ ] Configure DDoS protection (XDP if available)
- [ ] Implement certificate rotation policy
- [ ] Set up incident response procedures

## Security by Layer

### Application Layer

- Input validation on all API endpoints
- SQL injection prevention via parameterized queries (SQLAlchemy)
- XSS protection via output encoding
- CSRF tokens on all state-changing operations
- Secure session management with HTTPOnly, Secure cookies

### Transport Layer

- Mutual TLS authentication (mTLS) for all service-to-service communication
- Certificate pinning support for high-security deployments
- TLS 1.2+ with strong cipher suites
- Perfect forward secrecy enabled

### Network Layer

- eBPF firewall rules enforced at kernel level
- XDP early packet filtering (hardware-accelerated)
- Network policies for container communication
- DDoS protection with token bucket rate limiting

### Infrastructure Layer

- Container image scanning with Trivy
- Kubernetes RBAC and network policies
- Secrets encryption at rest
- Audit logging at container/infrastructure level

## Compliance

MarchProxy supports compliance with multiple standards:

- **SOC 2 Type II**: Comprehensive security and operational controls
- **HIPAA**: Healthcare data protection requirements
- **PCI-DSS**: Payment card data protection
- **GDPR**: Data privacy and protection
- **ISO 27001**: Information security management

Enterprise customers receive compliance documentation and audit-ready configurations.

## Security Research

We welcome security research and responsible disclosure from the security community. If you discover a vulnerability while conducting authorized security research, please follow our reporting process above.

## Security Updates

Subscribe to security updates:

- **GitHub Releases**: Watch for new security releases
- **Mailing List**: Subscribe at security@marchproxy.io for critical announcements
- **Blog**: Follow updates at blog.marchproxy.io

## Security Advisories Archive

### v1.0.0 Advisories

None at release time. Check [GitHub Advisories](https://github.com/marchproxy/marchproxy/security/advisories) for current advisories.

## PenguinTech Enterprise Support

Enterprise customers receive:

- 24/7 security incident response
- Priority vulnerability patching
- Security audits and penetration testing
- Compliance documentation generation
- Custom security configurations

Contact [enterprise@marchproxy.io](mailto:enterprise@marchproxy.io) for enterprise security support.

---

**Last Updated**: 2025-12-12
**Next Review**: 2026-01-12
