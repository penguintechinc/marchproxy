# Manager Configuration Guide

This guide covers comprehensive configuration options for the MarchProxy Manager component.

## Overview

The MarchProxy Manager is the central control plane component built on py4web/pydal that handles:

- User authentication and authorization
- Service and mapping configuration
- Multi-cluster management (Enterprise)
- License validation
- TLS certificate management
- Web interface and API server

## Configuration File Structure

The manager uses a YAML configuration file with the following sections:

```yaml
# /etc/marchproxy/manager.yaml
database:        # Database connection settings
server:          # Web server configuration
security:        # Authentication and security settings
license:         # Enterprise license configuration
monitoring:      # Health checks and metrics
logging:         # Logging configuration
clustering:      # Multi-cluster settings (Enterprise)
authentication:  # External authentication providers
tls:            # TLS/SSL configuration
enterprise:     # Enterprise-specific features
```

## Database Configuration

### PostgreSQL (Recommended)

```yaml
database:
  # Connection URL format: postgresql://user:password@host:port/database
  url: "postgresql://marchproxy:secure_password@localhost:5432/marchproxy"

  # Connection pool settings
  pool_size: 20              # Number of persistent connections
  max_overflow: 30           # Additional connections when pool exhausted
  pool_timeout: 30           # Seconds to wait for connection from pool
  pool_recycle: 3600         # Seconds to recycle connections

  # Query settings
  echo: false                # Log all SQL queries (development only)
  echo_pool: false           # Log connection pool events

  # Performance tuning
  connect_args:
    sslmode: "prefer"        # SSL connection mode
    connect_timeout: 10      # Connection timeout in seconds
    application_name: "marchproxy-manager"
```

### Alternative Database Configurations

#### MySQL/MariaDB
```yaml
database:
  url: "mysql+pymysql://marchproxy:password@localhost:3306/marchproxy"
  connect_args:
    charset: "utf8mb4"
    ssl_disabled: false
```

#### SQLite (Development Only)
```yaml
database:
  url: "sqlite:///marchproxy.db"
  connect_args:
    check_same_thread: false
```

### Environment Variable Override

```bash
# Override database URL via environment variable
export DATABASE_URL="postgresql://user:pass@host:port/db"
```

## Server Configuration

### Basic Server Settings

```yaml
server:
  # Binding configuration
  host: "0.0.0.0"            # Interface to bind (0.0.0.0 for all)
  port: 8000                 # HTTP port

  # Process configuration
  workers: 4                 # Number of worker processes (auto-detect: 0)
  reload: false              # Auto-reload on code changes (development)

  # Request handling
  timeout: 30                # Request timeout in seconds
  max_request_size: "10MB"   # Maximum request body size

  # Static files
  static_folder: "/opt/marchproxy/static"
  upload_folder: "/opt/marchproxy/uploads"
  max_content_length: "100MB"  # Maximum file upload size
```

### Advanced Server Settings

```yaml
server:
  # Performance tuning
  backlog: 2048              # Socket listen backlog
  keepalive: 5               # Keep-alive timeout
  max_connections: 1000      # Maximum concurrent connections

  # Logging
  access_log: true           # Enable access logging
  access_log_format: "%(h)s %(l)s %(u)s %(t)s \"%(r)s\" %(s)s %(b)s"

  # Security headers
  security_headers:
    X-Frame-Options: "DENY"
    X-Content-Type-Options: "nosniff"
    X-XSS-Protection: "1; mode=block"
    Strict-Transport-Security: "max-age=31536000; includeSubDomains"
    Content-Security-Policy: "default-src 'self'"
```

## Security Configuration

### Authentication Settings

```yaml
security:
  # JWT configuration
  jwt_secret: "your-256-bit-secret-key-here"  # Generate with: openssl rand -base64 32
  jwt_algorithm: "HS256"                      # JWT signing algorithm
  jwt_expiry: 86400                          # Token expiry in seconds (24 hours)

  # Session configuration
  session_secret: "your-session-secret-key"   # Session encryption key
  session_timeout: 3600                      # Session timeout in seconds
  session_cookie_secure: true               # HTTPS-only session cookies
  session_cookie_httponly: true             # HTTP-only session cookies

  # Password security
  password_salt: "your-password-salt"        # Password hashing salt
  password_hash_rounds: 12                   # bcrypt rounds (4-31)
  password_min_length: 8                     # Minimum password length
  password_require_uppercase: true           # Require uppercase letters
  password_require_lowercase: true           # Require lowercase letters
  password_require_digits: true              # Require digits
  password_require_special: true             # Require special characters

  # 2FA/TOTP settings
  totp_issuer: "MarchProxy"                  # TOTP issuer name
  totp_validity_window: 1                    # TOTP validity window

  # Rate limiting
  login_rate_limit: "5/minute"               # Login attempt rate limit
  api_rate_limit: "1000/hour"                # API request rate limit

  # Security features
  enable_csrf: true                          # Enable CSRF protection
  enable_xss_protection: true                # Enable XSS protection
  enable_sql_injection_protection: true     # Enable SQL injection protection
```

### API Security

```yaml
security:
  # API key configuration
  api_key_length: 32                         # API key length in bytes
  api_key_expiry: 0                         # API key expiry (0 = never)
  api_key_rotation_interval: 86400          # Auto-rotation interval (24 hours)

  # Cluster API keys
  cluster_api_key_length: 32                # Cluster API key length
  cluster_api_key_prefix: "mp_"             # Cluster API key prefix

  # CORS configuration
  cors_enabled: true                        # Enable CORS support
  cors_origins: ["https://dashboard.company.com"]  # Allowed origins
  cors_methods: ["GET", "POST", "PUT", "DELETE"]   # Allowed methods
  cors_headers: ["Content-Type", "Authorization"]  # Allowed headers
```

## License Configuration

### Basic License Settings

```yaml
license:
  # License server configuration
  server: "https://license.penguintech.io"   # License validation server
  timeout: 10                               # Validation timeout in seconds
  retry_attempts: 3                         # Retry attempts on failure
  retry_delay: 5                           # Delay between retries

  # License key
  key: "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"     # Enterprise license key

  # Caching configuration
  cache_ttl: 3600                          # Cache TTL in seconds (1 hour)
  grace_period: 86400                      # Grace period during outages (24 hours)

  # Validation frequency
  validation_interval: 300                  # Periodic validation (5 minutes)
  startup_validation: true                 # Validate on startup
```

### License Features

```yaml
license:
  # Feature enforcement
  enforce_proxy_limits: true               # Enforce proxy count limits
  enforce_cluster_limits: true             # Enforce cluster limits
  enforce_feature_access: true             # Enforce feature access

  # Community edition limits
  community_max_proxies: 3                 # Maximum proxies for Community
  community_max_clusters: 1               # Maximum clusters for Community

  # License validation
  offline_mode: false                      # Allow offline operation
  strict_validation: true                  # Strict license validation
```

## Monitoring Configuration

### Health Checks

```yaml
monitoring:
  # Health check configuration
  enable_health: true                      # Enable /healthz endpoint
  health_port: 8000                       # Health check port (same as main)
  health_path: "/healthz"                 # Health check path

  # Health check components
  health_checks:
    database: true                         # Check database connectivity
    license_server: true                   # Check license server
    certificate_expiry: true               # Check certificate expiry
    disk_space: true                      # Check disk space
    memory_usage: true                    # Check memory usage

  # Health check thresholds
  health_thresholds:
    database_timeout: 5                   # Database check timeout
    disk_space_min: 10                    # Minimum disk space (%)
    memory_usage_max: 90                  # Maximum memory usage (%)
    certificate_expiry_days: 30           # Certificate expiry warning (days)
```

### Metrics Configuration

```yaml
monitoring:
  # Metrics configuration
  enable_metrics: true                     # Enable /metrics endpoint
  metrics_port: 8001                      # Metrics port
  metrics_path: "/metrics"                # Metrics path
  metrics_format: "prometheus"            # Metrics format

  # Metrics collection
  metrics_interval: 15                    # Collection interval in seconds
  metrics_retention: 86400                # Metrics retention in seconds

  # Custom metrics
  custom_metrics:
    - name: "user_logins_total"
      type: "counter"
      description: "Total user login attempts"
    - name: "api_requests_duration"
      type: "histogram"
      description: "API request duration"
    - name: "active_proxies"
      type: "gauge"
      description: "Number of active proxy instances"
```

## Logging Configuration

### Basic Logging

```yaml
logging:
  # Log level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
  level: "INFO"

  # Log format (text, json)
  format: "json"

  # Log destinations
  file: "/var/log/marchproxy/manager.log"  # Log file path
  syslog: false                          # Enable syslog output
  console: true                          # Enable console output

  # File rotation
  max_size: "100MB"                      # Maximum log file size
  max_files: 10                          # Number of rotated files to keep
  compress: true                         # Compress rotated files
```

### Advanced Logging

```yaml
logging:
  # Structured logging
  structured: true                       # Enable structured logging
  correlation_id: true                   # Add correlation IDs
  request_id: true                      # Add request IDs

  # Log filters
  filters:
    - "password"                         # Filter sensitive data
    - "api_key"
    - "token"
    - "secret"

  # Component-specific log levels
  loggers:
    "marchproxy.auth": "DEBUG"           # Authentication debug logs
    "marchproxy.license": "INFO"         # License validation logs
    "marchproxy.database": "WARNING"     # Database logs
    "sqlalchemy": "WARNING"              # SQLAlchemy logs

  # Audit logging
  audit:
    enabled: true                        # Enable audit logging
    file: "/var/log/marchproxy/audit.log" # Audit log file
    events:                              # Events to audit
      - "user_login"
      - "user_logout"
      - "api_key_created"
      - "cluster_created"
      - "service_created"
      - "mapping_created"
```

### Centralized Logging

```yaml
logging:
  # Syslog configuration
  syslog:
    enabled: true                        # Enable syslog
    host: "syslog.company.com"          # Syslog server
    port: 514                           # Syslog port
    protocol: "udp"                     # Protocol (udp, tcp)
    facility: "local0"                  # Syslog facility

  # ELK Stack integration
  elasticsearch:
    enabled: false                       # Enable Elasticsearch output
    hosts: ["elasticsearch:9200"]       # Elasticsearch hosts
    index: "marchproxy-manager"          # Index pattern

  # Fluentd integration
  fluentd:
    enabled: false                       # Enable Fluentd output
    host: "fluentd"                     # Fluentd host
    port: 24224                         # Fluentd port
```

## Multi-Cluster Configuration (Enterprise)

### Cluster Management

```yaml
clustering:
  # Multi-cluster support
  enabled: true                          # Enable multi-cluster features

  # Default cluster
  default_cluster:
    name: "default"                      # Default cluster name
    description: "Default cluster"       # Default cluster description
    auto_create: true                   # Auto-create default cluster

  # Cluster limits
  max_clusters: 0                       # Maximum clusters (0 = unlimited)

  # Cluster API keys
  api_key_rotation:
    enabled: true                       # Enable automatic rotation
    interval: 86400                     # Rotation interval (24 hours)
    overlap_period: 3600                # Old key valid period (1 hour)
```

### Cross-Cluster Configuration

```yaml
clustering:
  # Cross-cluster communication
  cross_cluster:
    enabled: true                       # Enable cross-cluster mappings
    require_approval: true              # Require admin approval
    audit_all: true                    # Audit all cross-cluster operations

  # Cluster federation
  federation:
    enabled: false                      # Enable cluster federation
    discovery_interval: 300             # Cluster discovery interval
    heartbeat_interval: 60              # Cluster heartbeat interval
```

## Authentication Providers (Enterprise)

### SAML Configuration

```yaml
authentication:
  saml:
    enabled: true                       # Enable SAML authentication

    # Identity Provider settings
    idp:
      metadata_url: "https://identity.company.com/saml/metadata"
      entity_id: "https://identity.company.com"
      sso_url: "https://identity.company.com/saml/sso"
      slo_url: "https://identity.company.com/saml/slo"
      x509_cert: |                      # IdP certificate
        -----BEGIN CERTIFICATE-----
        MIIDXTCCAkWgAwIBAgIJAKoK/heBjcOuMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
        ...
        -----END CERTIFICATE-----

    # Service Provider settings
    sp:
      entity_id: "https://marchproxy.company.com"
      acs_url: "https://marchproxy.company.com/auth/saml/acs"
      sls_url: "https://marchproxy.company.com/auth/saml/sls"
      x509_cert: ""                     # SP certificate (optional)
      private_key: ""                   # SP private key (optional)

    # Attribute mapping
    attributes:
      user_id: "NameID"                 # User ID attribute
      email: "email"                    # Email attribute
      first_name: "firstName"           # First name attribute
      last_name: "lastName"             # Last name attribute
      groups: "groups"                  # Group membership attribute

    # User provisioning
    provisioning:
      auto_create: true                 # Auto-create users
      auto_update: true                 # Auto-update user attributes
      default_role: "user"              # Default role for new users
```

### OAuth2 Configuration

```yaml
authentication:
  oauth2:
    enabled: true                       # Enable OAuth2 authentication

    # Provider configurations
    providers:
      google:
        enabled: true
        client_id: "your-google-client-id"
        client_secret: "your-google-client-secret"
        scope: ["openid", "email", "profile"]

      microsoft:
        enabled: true
        client_id: "your-microsoft-client-id"
        client_secret: "your-microsoft-client-secret"
        tenant_id: "your-tenant-id"
        scope: ["openid", "email", "profile"]

      github:
        enabled: false
        client_id: "your-github-client-id"
        client_secret: "your-github-client-secret"
        scope: ["user:email"]

    # User provisioning
    provisioning:
      auto_create: true                 # Auto-create users
      auto_update: true                 # Auto-update user attributes
      email_verification: true          # Require email verification
      default_role: "user"              # Default role for new users
```

### SCIM Configuration

```yaml
authentication:
  scim:
    enabled: true                       # Enable SCIM provisioning
    endpoint: "/scim/v2"                # SCIM endpoint path

    # Authentication
    auth_type: "bearer"                 # Authentication type (bearer, basic)
    bearer_token: "your-scim-token"     # Bearer token for authentication

    # User provisioning
    users:
      auto_create: true                 # Auto-create users
      auto_update: true                 # Auto-update users
      auto_delete: false                # Auto-delete users (soft delete)

    # Group provisioning
    groups:
      auto_create: true                 # Auto-create groups
      auto_update: true                 # Auto-update groups
      map_to_clusters: true             # Map groups to clusters
```

## TLS Configuration

### Basic TLS Settings

```yaml
tls:
  # TLS enablement
  enabled: true                         # Enable TLS/HTTPS
  port: 8443                           # HTTPS port

  # Certificate sources
  certificate_source: "file"            # Source: file, vault, infisical, auto

  # File-based certificates
  cert_file: "/etc/ssl/certs/marchproxy.crt"     # Certificate file
  key_file: "/etc/ssl/private/marchproxy.key"    # Private key file
  ca_file: "/etc/ssl/certs/ca.crt"               # CA certificate file

  # TLS settings
  min_version: "1.2"                    # Minimum TLS version (1.0, 1.1, 1.2, 1.3)
  max_version: "1.3"                    # Maximum TLS version
  ciphers: "ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256"

  # Certificate validation
  verify_certificates: true             # Verify client certificates
  client_cert_required: false          # Require client certificates
```

### Automatic Certificate Management

```yaml
tls:
  # Let's Encrypt integration
  letsencrypt:
    enabled: true                       # Enable Let's Encrypt
    email: "admin@company.com"          # Contact email
    domains: ["marchproxy.company.com"] # Domains to secure
    staging: false                      # Use staging environment

  # Certificate renewal
  auto_renewal:
    enabled: true                       # Enable auto-renewal
    check_interval: 86400               # Check interval (24 hours)
    renewal_threshold: 30               # Renew when <30 days left
```

### Vault Integration

```yaml
tls:
  vault:
    enabled: true                       # Enable Vault integration
    url: "https://vault.company.com"    # Vault server URL
    token: "your-vault-token"           # Vault token
    mount_path: "pki"                   # PKI mount path
    role: "marchproxy"                  # Vault role
    common_name: "marchproxy.company.com"  # Certificate CN
    ttl: "8760h"                        # Certificate TTL (1 year)
```

## Enterprise Features

### Advanced Features Configuration

```yaml
enterprise:
  # Feature enablement
  enabled: true                         # Enable enterprise features

  # Advanced networking
  networking:
    xdp_rate_limiting: true             # Enable XDP rate limiting
    tls_proxy: true                     # Enable TLS proxy with wildcard certs
    advanced_load_balancing: true       # Enable advanced load balancing

  # Advanced authentication
  authentication:
    saml: true                          # Enable SAML
    oauth2: true                        # Enable OAuth2
    scim: true                          # Enable SCIM
    ldap: true                          # Enable LDAP

  # Advanced monitoring
  monitoring:
    advanced_dashboards: true           # Enable advanced Grafana dashboards
    distributed_tracing: true           # Enable distributed tracing
    custom_metrics: true                # Enable custom metrics
    alerting: true                      # Enable alerting

  # Advanced security
  security:
    waf_enterprise: true                # Enable enterprise WAF features
    threat_intelligence: true           # Enable threat intelligence
    compliance_reporting: true          # Enable compliance reporting
```

## Environment Variables Override

All configuration options can be overridden using environment variables with the prefix `MARCHPROXY_`:

```bash
# Database configuration
export MARCHPROXY_DATABASE_URL="postgresql://user:pass@host:port/db"
export MARCHPROXY_DATABASE_POOL_SIZE="20"

# Server configuration
export MARCHPROXY_SERVER_HOST="0.0.0.0"
export MARCHPROXY_SERVER_PORT="8000"
export MARCHPROXY_SERVER_WORKERS="4"

# Security configuration
export MARCHPROXY_SECURITY_JWT_SECRET="your-secret-key"
export MARCHPROXY_SECURITY_SESSION_SECRET="your-session-secret"

# License configuration
export MARCHPROXY_LICENSE_KEY="PENG-XXXX-XXXX-XXXX-XXXX-ABCD"

# Monitoring configuration
export MARCHPROXY_MONITORING_ENABLE_METRICS="true"
export MARCHPROXY_MONITORING_METRICS_PORT="8001"

# Logging configuration
export MARCHPROXY_LOGGING_LEVEL="INFO"
export MARCHPROXY_LOGGING_FORMAT="json"
```

## Configuration Validation

The manager includes built-in configuration validation:

```bash
# Validate configuration
marchproxy-manager validate-config --config /etc/marchproxy/manager.yaml

# Check specific sections
marchproxy-manager validate-config --section database
marchproxy-manager validate-config --section security
marchproxy-manager validate-config --section license
```

Example validation output:
```
✅ Configuration validation successful
✅ Database connection: OK
✅ License server: Reachable
✅ TLS certificates: Valid
⚠️  Warning: JWT secret should be at least 256 bits
❌ Error: SAML IdP metadata URL is unreachable
```

## Configuration Examples

### Development Configuration

```yaml
# development.yaml - Development environment
database:
  url: "sqlite:///marchproxy_dev.db"

server:
  host: "127.0.0.1"
  port: 8000
  workers: 1
  reload: true

security:
  jwt_secret: "dev-secret-key-not-for-production"
  session_secret: "dev-session-secret"
  password_hash_rounds: 4

license:
  server: "https://license-staging.penguintech.io"

logging:
  level: "DEBUG"
  format: "text"
  console: true
  file: ""

monitoring:
  enable_metrics: false
  enable_health: true

tls:
  enabled: false
```

### Production Configuration

```yaml
# production.yaml - Production environment
database:
  url: "postgresql://marchproxy:${DB_PASSWORD}@postgres-cluster:5432/marchproxy"
  pool_size: 50
  max_overflow: 100

server:
  host: "0.0.0.0"
  port: 8000
  workers: 8

security:
  jwt_secret: "${JWT_SECRET}"
  session_secret: "${SESSION_SECRET}"
  password_hash_rounds: 12
  login_rate_limit: "3/minute"
  api_rate_limit: "10000/hour"

license:
  server: "https://license.penguintech.io"
  key: "${ENTERPRISE_LICENSE}"

logging:
  level: "INFO"
  format: "json"
  file: "/var/log/marchproxy/manager.log"
  syslog:
    enabled: true
    host: "syslog.company.com"

monitoring:
  enable_metrics: true
  enable_health: true
  health_checks:
    database: true
    license_server: true
    certificate_expiry: true

tls:
  enabled: true
  port: 8443
  certificate_source: "vault"
  vault:
    url: "https://vault.company.com"
    token: "${VAULT_TOKEN}"

enterprise:
  enabled: true
  networking:
    xdp_rate_limiting: true
    tls_proxy: true
  authentication:
    saml: true
    oauth2: true
```

This completes the comprehensive Manager configuration documentation.