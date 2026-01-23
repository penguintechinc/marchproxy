# proxy-ailb Documentation

AI-Intelligent Load Balancer (AILB) proxy module for MarchProxy.

## Overview

The AILB proxy provides intelligent load balancing powered by machine learning and AI-driven decision making. It learns traffic patterns and optimizes routing decisions for improved performance and resource utilization.

## Features

- AI-powered routing optimization
- Traffic pattern learning
- Predictive load balancing
- Anomaly detection
- Dynamic backend selection
- Performance optimization

## Architecture

See [ARCHITECTURE.md](./ARCHITECTURE.md) for detailed architecture documentation.

## AI/ML Components

The AILB proxy integrates WaddleAI for:
- Traffic pattern analysis
- Predictive routing
- Anomaly detection
- Performance optimization recommendations

## Configuration

Configuration is managed through the Manager API. Refer to the Manager documentation for cluster setup and AI model configuration.

## Performance

Multi-tier packet processing with AI enhancement:
- eBPF fast-path for baseline routing
- Go application logic with AI-driven decisions
- WaddleAI integration for ML inference
- Standard kernel networking for fallback

## Monitoring

- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics` (Prometheus format)
- AI model performance metrics
- Routing decision analytics

## Related Modules

- [Manager](../../manager) - Configuration management
- [API Server](../../api-server) - API gateway
- [WaddleAI](https://github.com/penguintech/WaddleAI) - AI inference engine
