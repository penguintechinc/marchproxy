# proxy-rtmp Documentation

RTMP (Real-Time Messaging Protocol) proxy module for MarchProxy.

## Overview

The RTMP proxy provides streaming media protocol support with real-time message routing. It enables efficient proxying of RTMP streams for video broadcasting, live streaming, and media delivery.

## Features

- RTMP protocol support
- Real-time message routing
- Stream multiplexing
- Bitrate adaptation
- Connection pooling
- Stream statistics

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Protocol Details

RTMP is a protocol for real-time communication. The proxy handles:
- RTMP message framing
- Command and data streams
- Stream creation and destruction
- Connection handshaking

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for stream setup and routing rules.

## Performance

Multi-tier packet processing:
- eBPF connection tracking
- Go application logic for RTMP message routing
- Buffering and stream management

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)
- Stream statistics and bitrate metrics

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [Proxy L7](../proxy-l7) - Application layer processing
