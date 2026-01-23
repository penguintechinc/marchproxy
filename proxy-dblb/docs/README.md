# proxy-dblb Documentation

Dynamic Load Balancer (DBLB) proxy module for MarchProxy.

## Overview

The DBLB proxy provides dynamic load balancing with automatic backend discovery and health-based routing. It supports dynamic backend pool updates without service interruption.

## Features

- Dynamic backend discovery
- Health-based routing
- Automatic failover
- Load distribution algorithms
- Connection pooling
- Session persistence

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for cluster setup and backend pool configuration.

## Performance

Multi-tier packet processing for optimal performance:
- eBPF fast-path for health checks and connection filtering
- Go application logic for dynamic routing decisions
- Standard kernel networking for fallback paths

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)
- Backend health status available via API

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [Proxy L3/L4](../proxy-l3l4) - Layer 3/4 packet processing
