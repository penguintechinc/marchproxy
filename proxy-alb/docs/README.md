# proxy-alb Documentation

Application Load Balancer (ALB) proxy module for MarchProxy.

## Overview

The ALB proxy provides application-level load balancing with support for Layer 7 (application layer) routing decisions based on HTTP/HTTPS headers, paths, hostnames, and other application-specific criteria.

## Features

- HTTP/HTTPS load balancing
- Path-based routing
- Hostname-based routing
- Header-based routing
- WebSocket support
- TLS termination

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for cluster setup and service configuration.

## Performance

The ALB proxy is designed for maximum throughput using multi-tier packet processing:
- eBPF fast-path for connection filtering
- Go application logic for L7 routing decisions
- Standard kernel networking for fallback paths

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [Proxy L7](../proxy-l7) - Layer 7 packet processing
