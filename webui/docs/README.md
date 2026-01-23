# WebUI Documentation

Web-based user interface for MarchProxy management and monitoring.

## Overview

The WebUI is a React-based frontend application providing graphical access to MarchProxy cluster management, service configuration, monitoring, and user administration.

## Features

- Cluster management interface
- Service configuration UI
- Proxy server management
- User and role administration
- Real-time monitoring and metrics
- API key management
- TLS certificate management
- License management (Enterprise)

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Technology Stack

- **Framework**: React
- **Build Tool**: Vite or similar
- **Styling**: CSS/Tailwind CSS
- **API Communication**: REST with Manager backend
- **Node.js**: 20.x or 22.x LTS

## Project Structure

```
webui/
├── src/
│   ├── pages/
│   │   ├── Clusters.tsx
│   │   ├── Proxies.tsx
│   │   ├── Services.tsx
│   │   ├── Users.tsx
│   │   └── ...
│   ├── components/
│   ├── services/
│   └── App.tsx
├── tests/
├── docs/
└── package.json
```

## API Integration

The WebUI communicates with the Manager via REST API. See Manager documentation for API specifications.

## Development

Node.js 20.x or 22.x LTS required. See project package.json for dependencies.

## Testing

Comprehensive test suite including unit tests and integration tests. See tests/ folder for test coverage.

## Deployment

Deployed as static assets served by the Manager container with API routes proxied to the backend.

## Monitoring

- Error tracking and logging
- Performance metrics collection
- Real-time metric updates from proxy services

## Related Modules

- [Manager](../../manager) - Backend API and configuration
- [API Server](../../api-server) - API gateway
- [Proxy modules](../../) - All proxy implementations
