# Configuration Guide

Complete configuration documentation for MarchProxy Manager.

## Environment Variables

### Database Configuration

**DATABASE_URL** (Required)
- PostgreSQL connection string
- Format: `postgres://username:password@host:port/database`
- Default: `postgres://marchproxy:password@localhost:5432/marchproxy`
- Supports any pydal-compatible database (MySQL, SQLite, etc.)

Example:
```bash
DATABASE_URL=postgres://marchproxy:secure_pass@db.internal:5432/marchproxy_prod
```

### Authentication Configuration

**JWT_SECRET** (Required in production)
- Secret key for JWT token signing
- Minimum 32 characters
- Default: `your-super-secret-jwt-key-change-in-production`
- **MUST be changed before deployment**

```bash
JWT_SECRET=$(openssl rand -base64 32)
export JWT_SECRET
```

**ADMIN_PASSWORD** (Optional)
- Initial admin user password
- Default: `admin123`
- Only used during first initialization
- Change via API after first login

```bash
ADMIN_PASSWORD=admin123
```

**JWT_EXPIRATION_HOURS** (Optional)
- JWT token validity duration
- Default: `24` hours
- Range: 1-168 hours

```bash
JWT_EXPIRATION_HOURS=12
```

### License Configuration

**LICENSE_KEY** (Optional for Community edition)
- Enterprise license key format: `PENG-XXXX-XXXX-XXXX-XXXX-ABCD`
- **Required** for Enterprise features
- Omit for Community edition (max 3 proxies)

```bash
LICENSE_KEY=PENG-1234-5678-9012-3456-ABCD
```

**LICENSE_SERVER_URL** (Optional)
- PenguinTech license validation server
- Default: `https://license.penguintech.io`
- Leave at default in most cases

```bash
LICENSE_SERVER_URL=https://license.penguintech.io
```

**LICENSE_CACHE_HOURS** (Optional)
- License validation cache duration
- Default: `24` hours
- Range: 1-168 hours

```bash
LICENSE_CACHE_HOURS=12
```

**RELEASE_MODE** (Optional)
- Enable strict license enforcement
- Default: `false` (development)
- Set to `true` for production

```bash
RELEASE_MODE=true
```

### Logging Configuration

**LOG_LEVEL** (Optional)
- Logging verbosity
- Options: `DEBUG`, `INFO`, `WARNING`, `ERROR`, `CRITICAL`
- Default: `INFO`

```bash
LOG_LEVEL=DEBUG
```

**SYSLOG_ENABLED** (Optional)
- Enable centralized syslog logging
- Default: `false`

```bash
SYSLOG_ENABLED=true
```

**SYSLOG_HOST** (Optional)
- Syslog server hostname
- Required if SYSLOG_ENABLED=true

```bash
SYSLOG_HOST=logs.internal.example.com
```

**SYSLOG_PORT** (Optional)
- Syslog server UDP port
- Default: `514`

```bash
SYSLOG_PORT=514
```

### Server Configuration

**PORT** (Optional)
- HTTP server port
- Default: `8000`

```bash
PORT=8080
```

**HOST** (Optional)
- Server bind address
- Default: `0.0.0.0`

```bash
HOST=127.0.0.1
```

**PYTHONUNBUFFERED** (Recommended)
- Disable Python output buffering
- Set to `1` for Docker

```bash
PYTHONUNBUFFERED=1
```

**PY4WEB_APPS_FOLDER** (Optional)
- Location of py4web apps
- Default: `/app/apps`

```bash
PY4WEB_APPS_FOLDER=/app/apps
```

### SAML Configuration (Enterprise)

**SAML_ENABLED** (Optional)
- Enable SAML authentication
- Default: `false`

```bash
SAML_ENABLED=true
```

**SAML_IDP_URL** (Optional)
- SAML Identity Provider URL
- Required if SAML_ENABLED=true

```bash
SAML_IDP_URL=https://idp.example.com/auth/realms/marchproxy
```

**SAML_ENTITY_ID** (Optional)
- SAML Service Provider Entity ID
- Default: `marchproxy`

```bash
SAML_ENTITY_ID=marchproxy-sp
```

**SAML_CERT_PATH** (Optional)
- Path to SAML certificate file
- Required if SAML_ENABLED=true

```bash
SAML_CERT_PATH=/app/certs/saml-cert.pem
```

### OAuth2 Configuration (Enterprise)

**OAUTH2_ENABLED** (Optional)
- Enable OAuth2 authentication
- Default: `false`

```bash
OAUTH2_ENABLED=true
```

**OAUTH2_PROVIDER** (Optional)
- OAuth2 provider name
- Options: `google`, `github`, `azure`, `custom`

```bash
OAUTH2_PROVIDER=google
```

**OAUTH2_CLIENT_ID** (Optional)
- OAuth2 client ID
- Required if OAUTH2_ENABLED=true

```bash
OAUTH2_CLIENT_ID=your-client-id.apps.googleusercontent.com
```

**OAUTH2_CLIENT_SECRET** (Optional)
- OAuth2 client secret
- Required if OAUTH2_ENABLED=true
- **Never commit to version control**

```bash
OAUTH2_CLIENT_SECRET=your-client-secret
```

### TLS/Certificate Configuration

**TLS_ENABLED** (Optional)
- Enable HTTPS
- Default: `false`

```bash
TLS_ENABLED=true
```

**TLS_CERT_PATH** (Optional)
- Path to TLS certificate file
- Required if TLS_ENABLED=true

```bash
TLS_CERT_PATH=/app/certs/server.crt
```

**TLS_KEY_PATH** (Optional)
- Path to TLS private key
- Required if TLS_ENABLED=true

```bash
TLS_KEY_PATH=/app/certs/server.key
```

### Redis Configuration (Optional)

**REDIS_ENABLED** (Optional)
- Enable Redis for caching and sessions
- Default: `false`

```bash
REDIS_ENABLED=true
```

**REDIS_URL** (Optional)
- Redis connection string
- Format: `redis://[:password]@host:port/db`
- Required if REDIS_ENABLED=true

```bash
REDIS_URL=redis://:password@redis.internal:6379/0
```

**REDIS_TTL_SECONDS** (Optional)
- Session cache TTL
- Default: `3600` (1 hour)

```bash
REDIS_TTL_SECONDS=1800
```

### Monitoring Configuration

**METRICS_ENABLED** (Optional)
- Enable Prometheus metrics endpoint
- Default: `true`

```bash
METRICS_ENABLED=true
```

**METRICS_PORT** (Optional)
- Metrics endpoint port (if separate)
- Default: Same as HTTP port

```bash
METRICS_PORT=9090
```

### Rate Limiting

**RATE_LIMIT_ENABLED** (Optional)
- Enable API rate limiting
- Default: `true`

```bash
RATE_LIMIT_ENABLED=true
```

**RATE_LIMIT_REQUESTS_PER_MINUTE** (Optional)
- Default rate limit for standard endpoints
- Default: `100`

```bash
RATE_LIMIT_REQUESTS_PER_MINUTE=60
```

**RATE_LIMIT_AUTH_PER_MINUTE** (Optional)
- Rate limit for authentication endpoints
- Default: `10`

```bash
RATE_LIMIT_AUTH_PER_MINUTE=5
```

## Configuration Files

### .env File

Create `.env` file in manager directory:

```bash
# Database
DATABASE_URL=postgres://marchproxy:password@localhost:5432/marchproxy

# Authentication
JWT_SECRET=your-32-character-minimum-secret-key-here
ADMIN_PASSWORD=admin123

# License
LICENSE_KEY=PENG-1234-5678-9012-3456-ABCD
RELEASE_MODE=false

# Logging
LOG_LEVEL=INFO

# Server
PORT=8000
HOST=0.0.0.0
PYTHONUNBUFFERED=1

# Monitoring
METRICS_ENABLED=true
```

Load with:
```bash
export $(cat .env | xargs)
```

### Docker Compose

```yaml
version: '3.9'

services:
  postgres:
    image: postgres:15-bookworm
    environment:
      POSTGRES_USER: marchproxy
      POSTGRES_PASSWORD: password
      POSTGRES_DB: marchproxy
    volumes:
      - postgres_data:/var/lib/postgresql/data

  manager:
    build:
      context: .
      target: production
    environment:
      DATABASE_URL: postgres://marchproxy:password@postgres:5432/marchproxy
      JWT_SECRET: your-secret-key-here
      ADMIN_PASSWORD: admin123
      LOG_LEVEL: INFO
      PORT: 8000
    ports:
      - "8000:8000"
    depends_on:
      - postgres
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: marchproxy-manager-config
  namespace: marchproxy
data:
  LOG_LEVEL: "INFO"
  METRICS_ENABLED: "true"
  RELEASE_MODE: "true"
```

Secrets (keep separate):
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: marchproxy-manager-secrets
  namespace: marchproxy
type: Opaque
stringData:
  DATABASE_URL: postgres://user:pass@postgres:5432/db
  JWT_SECRET: your-secret-key-here
  LICENSE_KEY: PENG-1234-5678-9012-3456-ABCD
```

## Configuration Profiles

### Development Profile

```bash
LOG_LEVEL=DEBUG
PYTHONUNBUFFERED=1
PORT=8000
RELEASE_MODE=false
METRICS_ENABLED=true
DATABASE_URL=postgres://marchproxy:password@localhost:5432/marchproxy_dev
```

### Staging Profile

```bash
LOG_LEVEL=INFO
PORT=8000
RELEASE_MODE=false
METRICS_ENABLED=true
LICENSE_KEY=PENG-1234-5678-9012-3456-ABCD
SYSLOG_ENABLED=true
SYSLOG_HOST=logs.staging.internal
```

### Production Profile

```bash
LOG_LEVEL=WARNING
PORT=8000
RELEASE_MODE=true
METRICS_ENABLED=true
TLS_ENABLED=true
TLS_CERT_PATH=/app/certs/server.crt
TLS_KEY_PATH=/app/certs/server.key
LICENSE_KEY=PENG-1234-5678-9012-3456-ABCD
SYSLOG_ENABLED=true
SYSLOG_HOST=logs.prod.internal
REDIS_ENABLED=true
REDIS_URL=redis://password@redis-cluster:6379/0
RATE_LIMIT_ENABLED=true
```

## Configuration Validation

Manager validates configuration on startup:

```python
# Example: Required settings check
required = ['DATABASE_URL', 'JWT_SECRET']
missing = [k for k in required if not os.environ.get(k)]

if missing:
    raise ConfigError(f"Missing required config: {', '.join(missing)}")
```

## Secrets Management

### Local Development

Use `.env` file (add to `.gitignore`):
```bash
echo ".env" >> .gitignore
```

### Docker/Kubernetes

Use secrets management:

**Docker:**
```bash
docker run --env-file .env.production ...
```

**Kubernetes:**
```bash
kubectl create secret generic manager-secrets \
  --from-literal=JWT_SECRET=... \
  --from-literal=LICENSE_KEY=...
```

### HashiCorp Vault

```bash
# Retrieve secrets
export JWT_SECRET=$(vault kv get -field=jwt_secret secret/marchproxy)
export LICENSE_KEY=$(vault kv get -field=license_key secret/marchproxy)
```

## Configuration Best Practices

1. **Never hardcode secrets** - Use environment variables
2. **Validate all configuration** - Check required values at startup
3. **Document all variables** - Keep README updated
4. **Use strong defaults** - Fail securely
5. **Separate by environment** - Dev/staging/prod profiles
6. **Rotate secrets regularly** - JWT_SECRET, database passwords
7. **Monitor configuration changes** - Audit config modifications
8. **Version control configs** - Exclude secrets only

## Troubleshooting Configuration

### "Missing DATABASE_URL"
Set database connection string:
```bash
export DATABASE_URL=postgres://user:pass@host/db
```

### "Invalid JWT_SECRET"
Secret must be at least 32 characters:
```bash
JWT_SECRET=$(openssl rand -base64 32)
```

### "License validation failed"
Check license server connectivity:
```bash
curl https://license.penguintech.io/health
```

### "Database connection failed"
Verify PostgreSQL is running:
```bash
psql -c "SELECT version();"
```
