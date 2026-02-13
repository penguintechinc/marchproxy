# proxy-l3l4 Documentation

Layer 3/Layer 4 (Network/Transport) proxy module for MarchProxy.

## Overview

The L3/L4 proxy provides network and transport layer packet processing including TCP, UDP, and ICMP protocol handling. It forms the foundation for efficient packet forwarding and filtering at the kernel level using eBPF.

## Features

- TCP/UDP/ICMP processing
- Connection tracking
- Stateful packet filtering
- NAT (Network Address Translation)
- Port forwarding
- eBPF-accelerated packet processing

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## eBPF Components

The L3/L4 proxy utilizes eBPF programs for:
- Fast-path packet classification
- Connection state tracking
- Early packet filtering decisions
- Performance optimization

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for network setup and port configuration.

## Performance

Multi-tier packet processing architecture:
- eBPF programs for hardware-accelerated fast-path
- Go application for complex routing logic
- Standard kernel networking for fallback

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)
- Connection statistics available via API

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [Proxy Egress/Ingress](../proxy-egress) - Advanced packet processing
