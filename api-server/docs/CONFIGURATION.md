# MarchProxy API Server - Configuration Guide

Complete configuration reference for the MarchProxy API Server.

## Overview

Configuration is managed through environment variables using Pydantic Settings. All values can be overridden via environment variables at runtime.

## Configuration File Structure

The main configuration is in `app/core/config.py`:

```python
from pydantic_settings import BaseSettings

class Settings(BaseSettings):
    # Application settings
    APP_NAME: str
    APP_VERSION: str
    DEBUG: bool
    # ... more settings
```

## Environment Variables

### Application Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `APP_NAME` | string | "MarchProxy API Server" | Application display name |
| `APP_VERSION` | string | "1.0.0" | Application version |
| `DEBUG` | bool | false | Enable debug mode |
| `PORT` | int | 8000 | FastAPI server port |
| `WORKERS` | int | 4 | Number of Uvicorn workers |
| `ENVIRONMENT` | string | "production" | Environment name (production/staging/development) |

### Database Configuration

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `DATABASE_URL` | string | None | YES | PostgreSQL async connection string |
| `DATABASE_POOL_SIZE` | int | 20 | NO | Connection pool size |
| `DATABASE_POOL_RECYCLE` | int | 3600 | NO | Pool connection recycle time (seconds) |
| `DATABASE_POOL_PRE_PING` | bool | true | NO | Test connections before using |
| `DATABASE_ECHO` | bool | false | NO | Log all SQL statements |

**Database URL Format:**
```
postgresql+asyncpg://username:password@host:port/database
```

**Example:**
```bash
export DATABASE_URL="postgresql+asyncpg://marchproxy:secure_password@db.example.com:5432/marchproxy"
```

### Security Settings

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `SECRET_KEY` | string | None | YES | JWT signing secret (min 32 chars) |
| `ALGORITHM` | string | "HS256" | NO | JWT algorithm |
| `ACCESS_TOKEN_EXPIRE_MINUTES` | int | 30 | NO | Access token expiry in minutes |
| `REFRESH_TOKEN_EXPIRE_DAYS` | int | 7 | NO | Refresh token expiry in days |
| `BCRYPT_ROUNDS` | int | 12 | NO | Bcrypt hashing rounds |
| `PASSWORD_MIN_LENGTH` | int | 8 | NO | Minimum password length |
| `ALLOW_REGISTRATION` | bool | true | NO | Allow new user registration |
| `REQUIRE_EMAIL_VERIFICATION` | bool | false | NO | Require email verification |

**SECRET_KEY Requirements:**
- Minimum 32 characters
- Use random, cryptographically-secure string
- Change in production

**Generate SECRET_KEY:**
```bash
python -c "import secrets; print(secrets.token_urlsafe(32))"
```

### Redis Configuration

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `REDIS_URL` | string | redis://redis:6379/0 | NO | Redis connection URL |
| `REDIS_CACHE_TTL` | int | 3600 | NO | Cache time-to-live (seconds) |
| `ENABLE_CACHE` | bool | true | NO | Enable response caching |

**Redis URL Format:**
```
redis://[password@]host:port/database
redis+sentinel://password@sentinel1:26379/service-name/database
redis+unix:///tmp/redis.sock
```

### CORS Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `CORS_ORIGINS` | list | ["http://localhost:3000"] | Allowed origins (comma-separated) |
| `CORS_ALLOW_CREDENTIALS` | bool | true | Allow cookies in cross-origin |
| `CORS_ALLOW_METHODS` | list | ["*"] | Allowed HTTP methods |
| `CORS_ALLOW_HEADERS` | list | ["*"] | Allowed headers |

**Example:**
```bash
export CORS_ORIGINS="https://app.example.com,https://admin.example.com"
```

### License Integration

| Variable | Type | Default | Required | Description |
|----------|------|---------|----------|-------------|
| `LICENSE_KEY` | string | None | NO | PenguinTech license key |
| `LICENSE_SERVER_URL` | string | https://license.penguintech.io | NO | License server endpoint |
| `RELEASE_MODE` | bool | false | NO | Enable license enforcement |
| `LICENSE_CHECK_INTERVAL` | int | 86400 | NO | License check interval (seconds) |

**License Key Format:**
```
PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

### xDS Control Plane

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `XDS_GRPC_PORT` | int | 18000 | xDS gRPC server port |
| `XDS_HTTP_PORT` | int | 19000 | xDS HTTP admin port |
| `XDS_SERVER_URL` | string | http://localhost:19000 | xDS server HTTP URL |
| `XDS_SNAPSHOT_CACHE_SIZE` | int | 1000 | Max cached snapshots |
| `XDS_ENABLE_DEBUG_LOGS` | bool | false | Enable xDS debug logging |

### Monitoring Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENABLE_METRICS` | bool | true | Enable Prometheus metrics |
| `METRICS_PORT` | int | 8000 | Metrics endpoint port |
| `ENABLE_TRACING` | bool | false | Enable OpenTelemetry tracing |
| `TRACE_SAMPLE_RATE` | float | 1.0 | Trace sampling rate (0.0-1.0) |
| `JAEGER_AGENT_HOST` | string | localhost | Jaeger agent host |
| `JAEGER_AGENT_PORT` | int | 6831 | Jaeger agent port |

### Logging Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `LOG_LEVEL` | string | "INFO" | Logging level (DEBUG/INFO/WARNING/ERROR) |
| `LOG_FORMAT` | string | "json" | Log format (json/text) |
| `SYSLOG_ENABLED` | bool | false | Enable syslog forwarding |
| `SYSLOG_HOST` | string | localhost | Syslog server host |
| `SYSLOG_PORT` | int | 514 | Syslog server port |

### Advanced Features

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `ENABLE_MULTI_CLOUD` | bool | false | Enable multi-cloud routing |
| `ENABLE_TRAFFIC_SHAPING` | bool | false | Enable traffic shaping |
| `ENABLE_ZERO_TRUST` | bool | false | Enable zero-trust security |
| `ENABLE_OBSERVABILITY` | bool | false | Enable advanced observability |

## Configuration Examples

### Development Environment

```bash
# Development configuration
export APP_NAME="MarchProxy API Server"
export DEBUG=true
export ENVIRONMENT=development
export PORT=8000
export DATABASE_URL="postgresql+asyncpg://postgres:postgres@localhost:5432/marchproxy_dev"
export SECRET_KEY="dev-secret-key-minimum-32-characters-long"
export REDIS_URL="redis://localhost:6379/0"
export CORS_ORIGINS="http://localhost:3000,http://localhost:5173"
export RELEASE_MODE=false
```

### Staging Environment

```bash
# Staging configuration
export APP_NAME="MarchProxy API Server"
export DEBUG=false
export ENVIRONMENT=staging
export PORT=8000
export DATABASE_URL="postgresql+asyncpg://user:password@db.staging.internal:5432/marchproxy"
export SECRET_KEY="$(python -c 'import secrets; print(secrets.token_urlsafe(32))')"
export REDIS_URL="redis://redis.staging.internal:6379/0"
export CORS_ORIGINS="https://api-staging.example.com,https://admin-staging.example.com"
export LICENSE_KEY="PENG-XXXX-XXXX-XXXX-XXXX-XXXX"
export RELEASE_MODE=true
export LOG_LEVEL="INFO"
```

### Production Environment

```bash
# Production configuration (use secrets manager!)
export APP_NAME="MarchProxy API Server"
export DEBUG=false
export ENVIRONMENT=production
export PORT=8000
export WORKERS=8
export DATABASE_URL="postgresql+asyncpg://user:$(cat /run/secrets/db_password)@db.production.internal:5432/marchproxy"
export SECRET_KEY="$(cat /run/secrets/secret_key)"
export REDIS_URL="redis://:$(cat /run/secrets/redis_password)@redis.production.internal:6379/0"
export CORS_ORIGINS="https://api.example.com,https://admin.example.com"
export LICENSE_KEY="$(cat /run/secrets/license_key)"
export RELEASE_MODE=true
export LOG_LEVEL="WARNING"
export ENABLE_TRACING=true
export TRACE_SAMPLE_RATE=0.1
```

## Docker Configuration

### Environment File (.env)

Create `.env` file in api-server directory:

```env
# Application
APP_NAME=MarchProxy API Server
DEBUG=false
PORT=8000
WORKERS=4

# Database
DATABASE_URL=postgresql+asyncpg://user:password@postgres:5432/marchproxy

# Security
SECRET_KEY=your-secret-key-minimum-32-characters

# Redis
REDIS_URL=redis://redis:6379/0

# License
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-XXXX
RELEASE_MODE=true

# xDS
XDS_SERVER_URL=http://xds-server:19000
```

### Docker Run

```bash
# Load from .env file
docker run --env-file .env \
  -p 8000:8000 \
  marchproxy-api-server:latest

# Override specific variables
docker run --env-file .env \
  -e DEBUG=true \
  -e LOG_LEVEL=DEBUG \
  -p 8000:8000 \
  marchproxy-api-server:latest
```

### Docker Compose

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-bookworm
    environment:
      POSTGRES_USER: ${DB_USER:-marchproxy}
      POSTGRES_PASSWORD: ${DB_PASSWORD:-password}
      POSTGRES_DB: ${DB_NAME:-marchproxy}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-bookworm
    ports:
      - "6379:6379"

  api-server:
    build: .
    environment:
      DATABASE_URL: postgresql+asyncpg://${DB_USER:-marchproxy}:${DB_PASSWORD:-password}@postgres:5432/${DB_NAME:-marchproxy}
      SECRET_KEY: ${SECRET_KEY}
      REDIS_URL: redis://redis:6379/0
      LICENSE_KEY: ${LICENSE_KEY}
      RELEASE_MODE: ${RELEASE_MODE:-false}
    ports:
      - "8000:8000"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_started

volumes:
  postgres_data:
```

## Kubernetes Configuration

### ConfigMap for Non-Sensitive Data

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-server-config
  namespace: marchproxy
data:
  APP_NAME: "MarchProxy API Server"
  DEBUG: "false"
  PORT: "8000"
  WORKERS: "4"
  LOG_LEVEL: "INFO"
  ENVIRONMENT: "production"
  XDS_SERVER_URL: "http://xds-server:19000"
```

### Secret for Sensitive Data

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-server-secrets
  namespace: marchproxy
type: Opaque
stringData:
  SECRET_KEY: "your-secret-key-minimum-32-characters"
  DATABASE_URL: "postgresql+asyncpg://user:password@postgres:5432/marchproxy"
  REDIS_URL: "redis://:password@redis:6379/0"
  LICENSE_KEY: "PENG-XXXX-XXXX-XXXX-XXXX-XXXX"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-server
  namespace: marchproxy
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api-server
        image: marchproxy-api-server:v1.0.0
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8000
        envFrom:
        - configMapRef:
            name: api-server-config
        - secretRef:
            name: api-server-secrets
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "2Gi"
```

## Configuration Validation

### Validate Configuration at Startup

```python
from app.core.config import settings

# Print configuration
print(f"Environment: {settings.ENVIRONMENT}")
print(f"Database: {settings.DATABASE_URL.split('@')[1] if '@' in settings.DATABASE_URL else 'N/A'}")
print(f"Debug: {settings.DEBUG}")
print(f"Release Mode: {settings.RELEASE_MODE}")
```

### Check Configuration in Application

```bash
# Check current configuration
curl -s http://localhost:8000/healthz | jq .

# View environment
docker exec <container> printenv | grep -E "^(DATABASE_|SECRET_|REDIS_|LICENSE_)"
```

## Sensitive Configuration Management

### Using Secrets Managers

#### HashiCorp Vault

```bash
# Install hvac
pip install hvac

# Load secrets from Vault
export VAULT_ADDR="https://vault.example.com"
export VAULT_TOKEN="$(cat /run/secrets/vault_token)"

# In Python
import hvac

client = hvac.Client(url=os.getenv('VAULT_ADDR'))
secret = client.secrets.kv.v2.read_secret_version(path='marchproxy/api-server')
```

#### AWS Secrets Manager

```bash
# Install boto3
pip install boto3

# Load secrets
import boto3

client = boto3.client('secretsmanager', region_name='us-east-1')
secret = client.get_secret_value(SecretId='marchproxy/api-server')
```

#### Kubernetes Secrets

```bash
# Mount secrets as environment variables
env:
- name: SECRET_KEY
  valueFrom:
    secretKeyRef:
      name: api-server-secrets
      key: SECRET_KEY
```

## Configuration Best Practices

1. **Never Commit Secrets**: Add `.env` and secret files to `.gitignore`
2. **Use Environment Variables**: All configuration via env vars, not config files
3. **Validate at Startup**: Fail fast if required config is missing
4. **Separate by Environment**: Different configs for dev/staging/production
5. **Document Defaults**: Make defaults explicit in code
6. **Rotate Secrets Regularly**: Especially SECRET_KEY and PASSWORD
7. **Use Secrets Managers**: Don't store secrets in plain text anywhere
8. **Audit Access**: Log all configuration changes
9. **Secure Communication**: Use TLS for database and Redis connections
10. **Principle of Least Privilege**: Only grant required permissions

## Troubleshooting Configuration Issues

### Database Connection Failed

```bash
# Test database URL
export DATABASE_URL="postgresql+asyncpg://user:pass@host:5432/db"
python -c "
from sqlalchemy import create_engine
engine = create_engine('$DATABASE_URL')
engine.connect()
print('Database connection successful')
"
```

### Invalid SECRET_KEY

```bash
# Verify SECRET_KEY length
echo -n "$SECRET_KEY" | wc -c  # Should be >= 32

# Generate new SECRET_KEY
python -c "import secrets; print(secrets.token_urlsafe(32))"
```

### Redis Connection Issues

```bash
# Test Redis connection
redis-cli -u "$REDIS_URL" ping  # Should respond with PONG

# View current Redis config
redis-cli -u "$REDIS_URL" CONFIG GET "*"
```

### License Key Invalid

```bash
# Verify license key format
grep -E "^PENG-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$" <<< "$LICENSE_KEY"

# Test license validation
curl -X POST https://license.penguintech.io/api/v2/validate \
  -H "Content-Type: application/json" \
  -d "{\"license_key\": \"$LICENSE_KEY\"}"
```
