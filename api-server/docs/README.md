# API Server Documentation

Central API gateway and service orchestration for MarchProxy.

## Overview

The API Server is the central gateway for all MarchProxy services. It handles request routing to proxy services, cluster coordination, metrics aggregation, and provides unified API access for all proxy modules.

## Features

- Unified API gateway
- Service routing and discovery
- Cluster coordination
- Metrics aggregation
- Health monitoring
- Request/response logging
- API rate limiting

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Database Migrations

See [MIGRATIONS.md](./MIGRATIONS.md) for database schema and migration documentation.

## Technology Stack

- **Language**: Go
- **Database**: PostgreSQL (via Manager)
- **API Format**: RESTful JSON
- **Service Communication**: gRPC/REST

## Configuration

Key environment variables:
- `MANAGER_URL` - Manager service endpoint
- `DATABASE_URL` - Database connection string
- `API_PORT` - API server listen port
- `LOG_LEVEL` - Logging verbosity level

## API Endpoints

- `/health` - Service health status
- `/metrics` - Prometheus metrics
- `/api/v1/*` - API v1 endpoints
- `/api/v2/*` - API v2 endpoints

## Security

- API key authentication (cluster-specific)
- TLS/HTTPS for all communications
- Input validation and sanitization
- Rate limiting per client
- Request logging and audit trails

## Monitoring

- Health check endpoint: `/health`
- Metrics endpoint: `/metrics` (Prometheus format)
- Structured logging with correlation IDs
- Request/response timing metrics

## Related Modules

- [Manager](../../manager) - Configuration management
- [WebUI](../../webui) - Web interface
- [Proxy modules](../../) - All proxy implementations
