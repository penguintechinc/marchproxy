# proxy-nlb Documentation

Network Load Balancer (NLB) proxy module for MarchProxy.

## Overview

The NLB proxy provides ultra-high-performance network-layer load balancing optimized for extreme throughput and minimal latency. It operates at Layer 4 with support for millions of concurrent connections.

## Features

- Ultra-high throughput
- Extreme connection scaling
- Ultra-low latency
- TCP/UDP load balancing
- Connection affinity
- Health-based failover
- eBPF acceleration

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## Hardware Acceleration

The NLB proxy supports optional hardware acceleration for maximum performance:
- DPDK (Data Plane Development Kit) - Kernel bypass
- XDP (eXpress Data Path) - Driver-level processing
- AF_XDP (AF_XDP sockets) - Zero-copy packet processing
- SR-IOV (Single Root I/O Virtualization) - Hardware offload

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for network setup and load balancing policies.

## Performance

Multi-tier performance optimization:
- Hardware acceleration options (DPDK/XDP/AF_XDP/SR-IOV)
- eBPF fast-path processing
- Go application logic for complex decisions
- Standard kernel networking fallback

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)
- Connection and throughput statistics

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [Proxy L3/L4](../proxy-l3l4) - Layer 3/4 processing
