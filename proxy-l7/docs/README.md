# proxy-l7 Documentation

Layer 7 (Application) proxy module for MarchProxy.

## Overview

The L7 proxy provides application-layer packet processing with deep packet inspection for HTTP, HTTPS, WebSocket, and QUIC/HTTP3 protocols. It enables sophisticated routing decisions based on application-specific criteria.

## Features

- HTTP/HTTPS processing
- WebSocket upgrade handling
- QUIC/HTTP3 support
- Deep packet inspection (DPI)
- Content-based routing
- Request/response modification
- Header-based routing

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Protocol Support

- **HTTP/HTTPS**: Full support with TLS termination
- **WebSockets**: Upgrade and proxying with persistent connections
- **QUIC/HTTP3**: High-performance HTTP3 with multiplexing
- **Custom Protocols**: Extensible framework for protocol handlers

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for service setup and routing rules.

## Performance

Multi-tier packet processing:
- eBPF fast-path for connection establishment
- Go application logic for L7 decision making
- Connection pooling and keep-alive optimization

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)
- Request/response statistics available via API

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [Proxy ALB](../proxy-alb) - Application load balancer
