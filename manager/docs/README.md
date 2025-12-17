# Manager Documentation

Central management server for MarchProxy cluster operations.

## Overview

The Manager is a Python/py4web-based centralized control plane for MarchProxy. It handles configuration management, cluster administration, service definitions, user authentication, and licensing enforcement.

## Features

- Cluster management
- Service configuration
- User authentication (2FA, SAML, SCIM, OAuth2)
- License management (Community/Enterprise)
- API key generation and management
- Database schema and migrations
- Web UI backend API
- TLS/certificate management

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Technology Stack

- **Framework**: py4web
- **ORM**: pydal
- **Database**: PostgreSQL (configurable to any pydal-supported DB)
- **Authentication**: py4web native auth system + OAuth2/SAML/SCIM
- **API**: RESTful with native py4web decorators

## Database

PostgreSQL is the default database. Configuration via `DATABASE_URL` environment variable. See [MIGRATIONS.md](./MIGRATIONS.md) for schema management.

## Configuration

Key environment variables:
- `DATABASE_URL` - PostgreSQL connection string
- `LICENSE_KEY` - Enterprise license key (format: PENG-XXXX-XXXX-XXXX-XXXX-ABCD)
- `LICENSE_SERVER_URL` - License validation server
- `CLUSTER_API_KEY` - Default cluster API key

## API Documentation

See API reference in this documentation folder for endpoint specifications.

## Licensing

- **Community**: Maximum 3 proxy servers, single cluster, basic auth
- **Enterprise**: Unlimited proxies (license-based), multi-cluster, advanced auth

## Security

- Input validation on all endpoints
- CSRF protection via py4web
- Role-based access control (Administrator, Service-owner)
- Cluster-specific API key isolation

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)

## Related Modules

- [API Server](../../api-server) - API gateway
- [WebUI](../../webui) - Web interface
- [Proxy modules](../../) - All proxy implementations
