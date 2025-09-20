# Security Overview

This document provides a comprehensive overview of MarchProxy's security architecture, features, and best practices.

## Security Architecture

MarchProxy implements defense-in-depth security with multiple layers of protection:

```
┌─────────────────────────────────────────────────────────────────┐
│                        Internet Traffic                          │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                 Layer 1: Network Security                      │
│                                                                 │
│ • DDoS Protection          • Rate Limiting (XDP)               │
│ • IP Filtering             • GeoIP Blocking                    │
│ • Network Segmentation     • Traffic Shaping                   │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                 Layer 2: Application Security                  │
│                                                                 │
│ • Web Application Firewall • Input Validation                  │
│ • SQL Injection Protection • XSS Prevention                    │
│ • Command Injection Block  • Path Traversal Protection         │
│ • CSRF Protection          • Content Security Policy           │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│              Layer 3: Authentication & Authorization           │
│                                                                 │
│ • Multi-Factor Authentication • Role-Based Access Control      │
│ • SAML/OAuth2/SCIM Integration • API Key Management           │
│ • JWT Token Validation        • Session Management             │
│ • Certificate-based Auth      • Directory Integration          │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│                Layer 4: Data Security                          │
│                                                                 │
│ • TLS/SSL Encryption       • Data at Rest Encryption           │
│ • Certificate Management   • Key Rotation                      │
│ • Secure Configuration     • Secrets Management                │
│ • Database Encryption      • Backup Encryption                 │
└─────────────────────┬───────────────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────────────┐
│             Layer 5: Monitoring & Compliance                   │
│                                                                 │
│ • Security Event Logging   • Threat Detection                  │
│ • Audit Trails            • Compliance Reporting               │
│ • Anomaly Detection        • Incident Response                 │
│ • Vulnerability Scanning   • Security Metrics                  │
└─────────────────────────────────────────────────────────────────┘
```

## Core Security Features

### 1. Multi-Factor Authentication (MFA)

MarchProxy requires MFA for all administrative access:

- **TOTP-based 2FA**: Time-based One-Time Passwords using standard authenticator apps
- **Hardware tokens**: FIDO2/WebAuthn support for hardware security keys
- **Backup codes**: One-time recovery codes for emergency access
- **Grace period**: Configurable grace period for MFA setup

```yaml
# MFA Configuration
security:
  mfa:
    required: true
    totp:
      issuer: "MarchProxy"
      validity_window: 1
    backup_codes:
      enabled: true
      count: 10
    grace_period: 86400  # 24 hours for new users
```

### 2. Enterprise Authentication Integration

**SAML SSO Integration:**
- Support for major identity providers (Okta, Azure AD, Google Workspace)
- Just-in-time user provisioning
- Attribute-based role mapping
- Single logout support

**OAuth2/OpenID Connect:**
- Multi-provider support (Google, Microsoft, GitHub, custom)
- Automatic user provisioning
- Token refresh and validation
- Scope-based permissions

**SCIM Provisioning:**
- Automated user lifecycle management
- Group synchronization
- Real-time user updates
- Audit logging for all provisioning events

### 3. Role-Based Access Control (RBAC)

MarchProxy implements granular RBAC with cluster isolation:

```
Administrator
├── Full system access
├── User management
├── License management
└── Global configuration

Service Owner (per cluster)
├── Service management (assigned clusters only)
├── Mapping configuration (assigned clusters only)
├── Certificate management (assigned clusters only)
└── Read-only access to cluster metrics

Read-Only User (per cluster)
├── View services and mappings
├── View metrics and logs
└── No configuration changes
```

### 4. API Security

**API Key Management:**
- Cryptographically secure key generation
- Configurable expiration and rotation
- Granular permissions (read, write, admin)
- Automatic key rotation for clusters

**Rate Limiting:**
- Per-IP rate limiting
- Per-API key rate limiting
- Burst protection
- DDoS mitigation

**Request Validation:**
- Input sanitization and validation
- Schema validation for all API requests
- SQL injection prevention
- Command injection protection

### 5. Network Security

**XDP-based Rate Limiting (Enterprise):**
- Kernel-level packet filtering
- Hardware-accelerated processing
- Configurable rate limits per source IP
- Real-time attack mitigation

**Traffic Analysis:**
- Deep packet inspection
- Protocol validation
- Anomaly detection
- Threat intelligence integration

## Encryption and TLS

### TLS Configuration

MarchProxy uses TLS 1.2+ for all communications:

```yaml
tls:
  min_version: "1.2"
  max_version: "1.3"
  ciphers:
    - "TLS_AES_256_GCM_SHA384"
    - "TLS_CHACHA20_POLY1305_SHA256"
    - "TLS_AES_128_GCM_SHA256"
    - "ECDHE-RSA-AES256-GCM-SHA384"
    - "ECDHE-RSA-AES128-GCM-SHA256"

  # Certificate validation
  verify_certificates: true
  client_cert_required: false

  # Security headers
  hsts:
    enabled: true
    max_age: 31536000
    include_subdomains: true
    preload: true
```

### Certificate Management

**Automated Certificate Management:**
- Integration with Let's Encrypt
- HashiCorp Vault PKI integration
- Infisical certificate management
- Automatic renewal and rotation

**Enterprise TLS Proxy (Enterprise):**
- Self-signed CA generation with ECC P-384 and SHA-512
- Wildcard certificate generation for any domain
- 10-year certificate lifetimes
- User-provided CA upload and management

### Data Encryption

**Data at Rest:**
- Database encryption using AES-256
- Configuration file encryption
- Log file encryption
- Backup encryption

**Data in Transit:**
- TLS 1.3 for all API communications
- mTLS for service-to-service communication
- Encrypted database connections
- Secure syslog transmission

## Web Application Firewall (WAF)

MarchProxy includes a comprehensive WAF with multiple protection layers:

### SQL Injection Protection

```yaml
waf:
  sql_injection:
    enabled: true
    patterns:
      - "(?i)(union|select|insert|update|delete|drop|create|alter).*"
      - "(?i)(or|and)\\s+\\d+\\s*=\\s*\\d+"
      - "(?i)(exec|execute|sp_).*"
    block_threshold: 5
    action: "block"  # block, monitor, sanitize
```

### XSS Prevention

```yaml
waf:
  xss:
    enabled: true
    patterns:
      - "(?i)<script[^>]*>.*?</script>"
      - "(?i)javascript:"
      - "(?i)vbscript:"
      - "(?i)on(load|error|click|mouse)\\s*="
    sanitize_responses: true
    content_security_policy:
      enabled: true
      policy: "default-src 'self'; script-src 'self' 'unsafe-inline'"
```

### Command Injection Protection

```yaml
waf:
  command_injection:
    enabled: true
    patterns:
      - "(?i)(;|\\||&|`|\\$\\(|\\${)"
      - "(?i)(cat|ls|pwd|id|whoami|uname)\\s"
      - "(?i)(curl|wget|nc|netcat)\\s"
    block_threshold: 1
    action: "block"
```

### Custom Security Rules

```yaml
waf:
  custom_rules:
    - name: "Block admin paths from external IPs"
      pattern: "/admin/*"
      condition: "ip_not_in_range"
      value: ["10.0.0.0/8", "192.168.0.0/16"]
      action: "block"

    - name: "Rate limit API endpoints"
      pattern: "/api/*"
      condition: "rate_limit"
      value: "100/minute"
      action: "rate_limit"

    - name: "Block suspicious user agents"
      pattern: ".*"
      condition: "user_agent_contains"
      value: ["sqlmap", "nikto", "scanner"]
      action: "block"
```

## DDoS Protection

### Multi-Layer DDoS Protection

**Layer 3/4 Protection:**
- SYN flood protection
- UDP amplification mitigation
- ICMP flood filtering
- Connection rate limiting

**Layer 7 Protection:**
- HTTP flood detection
- Slowloris protection
- POST/PUT flood mitigation
- Browser validation challenges

### Configuration

```yaml
ddos_protection:
  enabled: true

  # Layer 3/4 protection
  network:
    syn_flood:
      enabled: true
      threshold: 1000  # packets per second
      window: 10       # seconds

    connection_limits:
      max_connections_per_ip: 100
      max_new_connections_per_second: 10
      connection_timeout: 30

  # Layer 7 protection
  application:
    http_flood:
      enabled: true
      threshold: 100   # requests per second
      window: 60       # seconds

    slowloris:
      enabled: true
      header_timeout: 10
      body_timeout: 30
```

## Security Monitoring and Alerting

### Security Event Logging

MarchProxy logs all security-relevant events:

```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "event_type": "authentication_failure",
  "severity": "warning",
  "source_ip": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "username": "admin",
  "failure_reason": "invalid_password",
  "session_id": "sess_123456789",
  "request_id": "req_987654321"
}
```

### Threat Detection

**Anomaly Detection:**
- Unusual login patterns
- Abnormal traffic volumes
- Suspicious user behavior
- Geographic access anomalies

**Signature-based Detection:**
- Known attack patterns
- Malicious IP addresses
- Bot detection
- Vulnerability scanning detection

### Security Metrics

```prometheus
# Authentication metrics
marchproxy_auth_attempts_total{status="success",method="password"} 1250
marchproxy_auth_attempts_total{status="failure",method="password"} 23
marchproxy_auth_mfa_challenges_total{status="success"} 145
marchproxy_auth_mfa_challenges_total{status="failure"} 5

# WAF metrics
marchproxy_waf_requests_total{action="allow"} 45678
marchproxy_waf_requests_total{action="block"} 234
marchproxy_waf_rules_triggered_total{rule="sql_injection"} 12
marchproxy_waf_rules_triggered_total{rule="xss"} 8

# DDoS metrics
marchproxy_ddos_attacks_detected_total{type="syn_flood"} 3
marchproxy_ddos_packets_dropped_total{reason="rate_limit"} 12345
marchproxy_ddos_connections_rejected_total{reason="ip_limit"} 67

# Security scanning metrics
marchproxy_security_scans_detected_total{scanner="nmap"} 5
marchproxy_security_scans_detected_total{scanner="nikto"} 2
```

## Compliance and Auditing

### Audit Logging

MarchProxy maintains comprehensive audit logs for compliance:

**User Actions:**
- Login/logout events
- Configuration changes
- Administrative actions
- Permission changes

**System Events:**
- Service starts/stops
- Configuration reloads
- Certificate updates
- License changes

**API Operations:**
- All API requests and responses
- Authentication events
- Authorization decisions
- Rate limiting actions

### Compliance Standards

MarchProxy supports compliance with:

- **SOC 2 Type II**
- **ISO 27001**
- **PCI DSS** (when handling payment data)
- **GDPR** (data protection and privacy)
- **HIPAA** (healthcare data protection)
- **FedRAMP** (US government compliance)

### Data Retention and Privacy

```yaml
compliance:
  data_retention:
    audit_logs: 2557  # 7 years in days
    access_logs: 90   # 90 days
    security_logs: 365 # 1 year

  privacy:
    data_minimization: true
    anonymization: true
    right_to_deletion: true
    data_export: true
```

## Security Best Practices

### Deployment Security

1. **Network Segmentation:**
   - Deploy components in separate network segments
   - Use firewalls between tiers
   - Implement zero-trust networking

2. **Least Privilege:**
   - Run services with minimal required permissions
   - Use dedicated service accounts
   - Implement capability dropping

3. **Infrastructure Hardening:**
   - Regular OS updates and patching
   - Disable unnecessary services
   - Configure secure defaults

### Operational Security

1. **Access Management:**
   - Regular access reviews
   - Automated user deprovisioning
   - Strong password policies

2. **Monitoring and Response:**
   - 24/7 security monitoring
   - Automated incident response
   - Regular security assessments

3. **Backup and Recovery:**
   - Encrypted backups
   - Regular backup testing
   - Disaster recovery procedures

### Development Security

1. **Secure Development:**
   - Security code reviews
   - Dependency scanning
   - Static analysis

2. **Testing:**
   - Security testing in CI/CD
   - Penetration testing
   - Vulnerability assessments

## Incident Response

### Security Incident Classification

**Critical (P0):**
- Active data breach
- Complete system compromise
- Widespread service outage

**High (P1):**
- Failed authentication attempts exceeding threshold
- WAF blocking significant traffic
- Suspicious administrative activity

**Medium (P2):**
- Individual account compromise
- Non-critical vulnerabilities
- Policy violations

**Low (P3):**
- Security awareness issues
- Minor configuration issues
- Information gathering attempts

### Response Procedures

1. **Detection and Analysis:**
   - Automated alert generation
   - Initial triage and assessment
   - Impact and scope determination

2. **Containment:**
   - Immediate threat isolation
   - System quarantine if necessary
   - Evidence preservation

3. **Eradication and Recovery:**
   - Root cause analysis
   - System remediation
   - Service restoration

4. **Post-Incident:**
   - Lessons learned documentation
   - Process improvement
   - Stakeholder communication

### Emergency Contacts

```yaml
incident_response:
  contacts:
    security_team: "security@company.com"
    on_call: "+1-555-SECURITY"
    management: "ciso@company.com"

  external:
    legal: "legal@company.com"
    pr: "communications@company.com"
    law_enforcement: "911"
```

## Security Updates and Maintenance

### Regular Security Tasks

**Daily:**
- Review security alerts and logs
- Monitor threat intelligence feeds
- Verify backup completion

**Weekly:**
- Security patch assessment
- Access review for new/changed accounts
- Vulnerability scan analysis

**Monthly:**
- Security metrics review
- Incident response plan testing
- Security training updates

**Quarterly:**
- Penetration testing
- Security policy review
- Compliance assessment

### Automated Security Maintenance

```bash
#!/bin/bash
# Security maintenance script

# Update threat intelligence feeds
curl -H "Authorization: Bearer $THREAT_INTEL_KEY" \
  https://api.threatintel.com/feeds/malicious-ips > /etc/marchproxy/blocked-ips.txt

# Rotate API keys older than 90 days
python3 /opt/marchproxy/scripts/rotate-old-keys.py --days 90

# Update WAF rules
wget https://updates.marchproxy.io/waf-rules/latest.yml -O /etc/marchproxy/waf-rules.yml

# Restart services if configuration changed
systemctl reload marchproxy-manager
systemctl reload marchproxy-proxy

# Generate security report
python3 /opt/marchproxy/scripts/security-report.py --email security@company.com
```

This completes the comprehensive security overview for MarchProxy, covering all aspects of the security architecture and operational procedures.