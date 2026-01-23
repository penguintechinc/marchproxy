# MarchProxy Security Documentation

## Overview

MarchProxy implements defense-in-depth security across multiple layers: application, transport, network, and infrastructure. This document provides comprehensive security implementation details.

**For vulnerability reporting, see the [main Security Policy](/SECURITY.md).**

## Table of Contents

1. [Authentication Methods](#authentication-methods)
2. [Encryption & mTLS](#encryption--mtls)
3. [API Security](#api-security)
4. [Network Security](#network-security)
5. [Data Protection](#data-protection)
6. [Vulnerability Reporting](#vulnerability-reporting)

## Authentication Methods

### Multi-Factor Authentication (MFA)

- **TOTP Support**: Time-based one-time passwords for user accounts
- **Hardware Keys**: Support for FIDO2/WebAuthn hardware security keys
- **Backup Codes**: Emergency access codes for account recovery
- **Enrollment**: Optional MFA with enforcement policies for Enterprise tier

### Enterprise SSO

**SAML 2.0**: Industry-standard single sign-on
- Service provider configuration via SAML metadata
- Just-in-time (JIT) user provisioning
- Attribute mapping for role assignment
- Signed assertions and response validation

**OAuth2/OpenID Connect**: Modern identity delegation
- Authorization code flow with PKCE
- Token refresh and revocation
- User info endpoint integration
- SAML attribute federation

**SCIM Provisioning**: System for Cross-domain Identity Management
- User provisioning and de-provisioning
- Group membership synchronization
- Real-time directory updates
- Bidirectional sync with identity providers

### Role-Based Access Control (RBAC)

Two core roles with cluster-level assignments:

**Administrator Role**
- Full system configuration access
- License management and enforcement
- User and cluster management
- Audit log access
- Service creation and deletion

**Service-Owner Role**
- Service-specific permissions (assign to services or clusters)
- Read/write service configurations
- View associated metrics and logs
- Cannot modify user accounts or cluster settings

Roles are assigned at cluster scope, enabling multi-tenant isolation in Enterprise deployments.

### API Key Management

**Cluster-Specific Keys**: Each proxy cluster receives unique API key
- Registered during proxy initialization
- Used for proxy-to-manager authentication
- Rotated via policy (default: 90 days)
- Immediate revocation without downtime

**Token Types**:
- **Bearer Tokens**: Stateless JWT with configurable expiration
- **Base64 Tokens**: Legacy format for backward compatibility (mutually exclusive with JWT)

**Token Expiration & Rotation**:
- Access tokens: 60 minutes (configurable)
- Refresh tokens: 7 days (configurable)
- Automatic rotation on deployment

## Encryption & mTLS

### TLS Configuration

**Protocol Versions**: TLS 1.2+ enforced (TLS 1.3 recommended)
- TLS 1.0/1.1: Disabled by default
- Perfect forward secrecy: Enabled via ephemeral key exchange
- OCSP stapling: Optional certificate status validation

**Cipher Suites**: Strong modern ciphers
```
TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256
TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 (mTLS)
```

### Mutual TLS (mTLS)

**Certificate Authority**:
- Self-signed CA generation with 10-year validity (3650 days)
- ECC P-384 cryptography for signature
- RSA-2048 or ECC P-256 for service certificates
- Annual renewal with 60-day overlap window

**Service Certificates**:
- Auto-generated on first startup
- 1-year validity with automatic renewal
- Common Name: `<service-name>.marchproxy.local`
- Subject Alternative Names (SANs) for service discovery

**Certificate Pinning**: Optional for high-security environments
- Pin issuer CA certificate in client configuration
- Pin service certificate by thumbprint
- Prevent certificate substitution attacks

### Database Encryption

**Connection Security**:
- PostgreSQL SSL/TLS: `sslmode=require`
- Connection pooling with encrypted credentials
- Statement timeout protection (30s default)

**Credentials Management**:
- Encrypted in transit via TLS
- Stored encrypted in secrets manager
- Never logged or exposed in errors
- Rotated via policy (90 days recommended)

## API Security

### Input Validation

**Field Validation**: All API inputs validated server-side
- Type checking: Enforce correct data types
- Length limits: Prevent buffer overflow/DoS
- Format validation: Email, IP, port, etc.
- Whitelist validation: Only allow known values

**SQL Injection Prevention**:
- Parameterized queries via SQLAlchemy ORM
- No string concatenation in queries
- Input escaping for all untrusted data

**XSS Protection**:
- Output encoding in all responses
- Content-Security-Policy headers
- HTML entity encoding
- JavaScript context encoding

**CSRF Protection**:
- Synchronization token pattern for form submissions
- Same-site cookie attribute (Strict)
- Origin/Referer header validation
- Token rotation per request (optional)

### Rate Limiting

**Multi-Tier Strategy**:

1. **API Gateway Level**: Per-IP rate limiting
   - 1000 requests/second per IP (configurable)
   - Token bucket algorithm
   - Sliding window enforcement

2. **Service Level**: Per-API endpoint
   - Route-specific limits
   - User-based limits (authenticated)
   - Burst tolerance with gradual backoff

3. **Proxy Level**: Per-service connection limits
   - Max concurrent connections per service
   - Connection timeout: 5 minutes idle
   - Max requests per connection

### Circuit Breaker

**Automatic Failure Detection**:
- Track failed API calls
- Thresholds: 50 consecutive failures or 90% error rate
- States: Closed (normal) → Open (failing) → Half-Open (recovering)
- Recovery: 30-second timeout before retry

**Graceful Degradation**:
- Return cached responses when circuit open
- Failover to backup services
- User-facing error messages without internals
- Automatic status page updates

## Network Security

### eBPF Firewall

**Kernel-Level Filtering**:
- TCP/UDP protocol filtering
- Port-based rules (allow/deny)
- Rate limiting at packet level
- Low-level DDoS detection

**Rule Engine**:
- Per-service rules
- Per-cluster default rules
- Dynamic rule reloading without restart
- Efficient memory usage for large rulesets

### XDP Acceleration

**Driver-Level Processing**:
- Packet processing before kernel stack
- Early DDoS protection
- High-speed traffic shaping
- Hardware offload when available

**DDoS Detection**:
- Traffic anomaly detection
- Connection state tracking
- Rate limiting with token bucket
- Adaptive thresholds

### Web Application Firewall (WAF)

**OWASP Protection**:
- SQL injection detection and blocking
- XSS payload detection
- Command injection prevention
- Path traversal blocking
- Protocol violation detection

**Rule Management**:
- Rule updates via policy
- Custom rule support (Enterprise)
- Log all blocked requests
- False positive analysis

## Data Protection

### Database Security

**Encryption Options**:
- PostgreSQL native encryption: pgcrypto extension
- Volume-level encryption (dm-crypt, BitLocker)
- Backup encryption: AES-256
- Column-level encryption for sensitive fields

**Access Control**:
- Row-level security via PostgreSQL policies
- User credential isolation per cluster
- Service credential isolation per service
- Immutable audit records

### Secrets Management

**Infisical Integration**:
- Centralized secret storage
- Automatic rotation policies
- Audit trail for all access
- Team-based access control

**HashiCorp Vault Integration**:
- Dynamic secret generation
- Automatic credential rotation
- Encryption key management
- Audit logging

**Environment Variables** (Development only):
- Never commit secrets to version control
- Use `.env.local` for local development
- Rotate before any deployment

### Logging Security

**Comprehensive Audit Logging**:
- All API calls logged with user context
- Request/response (sanitized of secrets)
- Success/failure outcome
- Timestamp and user agent
- Source IP and session ID

**Log Storage**:
- Elasticsearch for centralized storage
- Immutable index configuration
- Tamper-evident logging
- Configurable retention (7-30 days)

**Log Access Control**:
- RBAC for log viewing
- Sensitive field masking (passwords, tokens)
- Read-only access from audit system
- Encryption at rest and in transit

### Data Retention

**Policy Defaults**:
- Audit logs: 30 days
- Application logs: 7 days
- Metrics retention: 7 days
- User activity traces: 30 days
- Backup retention: 30 days minimum

**Secure Deletion**:
- Cryptographic erasure
- Multi-pass overwrite not required (encryption key deletion sufficient)
- Verified deletion for PII
- Compliance with GDPR right-to-be-forgotten

## Vulnerability Reporting

### Responsible Disclosure

**Process**:
1. Email: security@marchproxy.io
2. Include: description, affected versions, steps to reproduce, impact
3. Response within 48 hours
4. Status updates every 5 business days
5. Coordinated disclosure (typically 90 days)
6. Credit in security advisory (optional)

**Do NOT**:
- Publicly disclose before patch available
- Test on production systems
- Access unauthorized data beyond proof-of-concept
- Disrupt service availability

### Vulnerability Management

**Scanning**:
- Dependabot for dependency vulnerabilities
- Socket.dev for supply chain risks
- govulncheck (Go) weekly
- safety/pip-audit (Python) weekly
- Semgrep for code vulnerabilities
- Trivy for container images

**Patching Policy**:
- Critical: 24 hours
- High: 1 week
- Medium: 2 weeks
- Low: Next release cycle

**Supported Versions**:
- v1.0.x: Through 2027-12-12
- v0.1.x: Critical fixes only through 2025-12-12

## Security Compliance

MarchProxy supports multiple compliance frameworks:

**SOC 2 Type II**: Security controls, availability, processing integrity
**HIPAA**: Healthcare data protection and audit logging
**PCI-DSS**: Payment card data protection and encryption
**GDPR**: Data privacy, retention, and subject access requests
**ISO 27001**: Information security management system

Enterprise customers receive audit-ready configurations and compliance documentation.

## Related Documentation

- [Main Security Policy](/SECURITY.md) - Vulnerability reporting and supported versions
- [Architecture Documentation](/docs/ARCHITECTURE.md) - System design and components
- [Deployment Guide](/docs/DEPLOYMENT.md) - Production security configuration
- [API Documentation](/docs/API.md) - Authentication and authorization details

---

**Last Updated**: 2025-12-16
**Status**: Complete
