# MarchProxy NLB Release Notes

## Version 1.0.0 (2025-12-16)

### Highlights

Initial release of the MarchProxy Network Load Balancer (NLB) as a unified traffic management solution for the MarchProxy architecture. The NLB serves as the single entry point for all network traffic with intelligent protocol detection, routing, rate limiting, and orchestration capabilities.

### Major Features

#### Protocol Detection and Routing
- Automatic detection of incoming protocol type by inspecting initial bytes
- Support for 6 protocols: HTTP, MySQL, PostgreSQL, MongoDB, Redis, RTMP
- Least-connections routing algorithm
- Health-aware routing with automatic failover

#### Rate Limiting
- Token bucket algorithm implementation
- Per-protocol and per-service rate limit buckets
- Configurable capacity and refill rates
- Real-time token availability tracking
- Graceful handling of rate limit exceeded conditions

#### Autoscaling Orchestration
- Metric-based autoscaling (CPU, memory, connection count)
- Configurable scaling policies per protocol
- Separate cooldown periods for scale-up and scale-down
- Min/max replica bounds with safety enforcement
- Stability through multi-period evaluation averages

#### Blue/Green Deployments
- Zero-downtime module version updates
- Instant traffic switching capability
- Canary rollout with configurable step sizes and durations
- Traffic splitting with weighted distribution
- Quick rollback to previous version on issues

#### gRPC Communication
- Module registration and management via gRPC
- Connection pooling for efficient module communication
- Health check integration with module containers
- Automatic reconnection on failures
- Keepalive support for stable connections

#### Observability
- Comprehensive health check endpoint (/healthz)
- Prometheus-compatible metrics endpoint
- JSON status endpoint with detailed subsystem statistics
- Support for distributed tracing (Jaeger)
- Structured JSON logging

### Technical Details

- **Language**: Go 1.24
- **Base Image**: Debian bookworm (no Alpine)
- **Container Platforms**: linux/amd64, linux/arm64, linux/arm/v7
- **Dependencies**: Managed via go.mod with security scanning
- **Testing**: 80%+ code coverage with unit and integration tests
- **Licensing**: Limited AGPL3 with fair use preamble

### Configuration

- YAML configuration file support with environment variable overrides
- Sensible defaults for all settings
- Comprehensive validation on startup
- Docker Compose example included

### Monitoring and Metrics

Key metrics exposed:
- `nlb_routed_connections_total` - Connections routed by protocol
- `nlb_active_connections` - Current connections per module
- `nlb_routing_errors_total` - Routing failures
- `nlb_ratelimit_allowed_total` - Allowed requests
- `nlb_ratelimit_denied_total` - Denied requests (rate limit exceeded)
- `nlb_ratelimit_tokens_available` - Current token availability
- `nlb_scale_operations_total` - Scaling operations executed
- `nlb_current_replicas` - Current replica count per protocol
- `nlb_bluegreen_traffic_split` - Traffic distribution during deployments

### Documentation

Comprehensive documentation included:
- API.md - REST and gRPC API documentation
- TESTING.md - Testing guide with benchmarks
- CONFIGURATION.md - Configuration reference
- USAGE.md - Operational usage guide
- README.md - Quick reference and architecture overview
- QUICK_REFERENCE.md - Common commands and examples

### Docker Builds

Provided multi-stage Docker build targets:
- **production** - Optimized for production use
- **development** - Includes debug tools
- **testing** - Includes test runner
- **debug** - Includes Delve debugger for development

### CI/CD Integration

- GitHub Actions workflow for automated testing
- Multi-architecture image building (amd64, arm64, arm/v7)
- Security scanning with govulncheck and Trivy
- Linting with golangci-lint, gosec
- 80%+ code coverage requirement
- Artifact caching for build speed

### Known Limitations

- Configuration changes require full restart (no hot reload)
- Protocol detection based on first bytes only
- Single NLB instance per deployment (clustering in roadmap)
- License enforcement available only in release mode

### Roadmap

Planned features for future releases:
- Multi-NLB clustering for high availability
- Additional protocol support (GRPC, QUIC, HTTP/3)
- Custom protocol detection plugins
- Advanced traffic shaping (weighted round-robin, IP-based routing)
- Enhanced observability (OpenTelemetry integration)
- Configuration hot-reload capability
- Web UI for management

### Breaking Changes

None - initial release

### Migration Guide

N/A - initial release

### Dependency Updates

All dependencies at latest stable versions as of release date:
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration management
- `github.com/prometheus/client_golang` - Metrics
- `github.com/sirupsen/logrus` - Logging
- `google.golang.org/grpc` - gRPC framework
- `google.golang.org/protobuf` - Protocol buffers

### Security Considerations

- API key authentication via `CLUSTER_API_KEY` environment variable
- Cluster-specific routing isolation
- Rate limiting protects against denial-of-service
- No credentials embedded in code or default config
- Security scanning in CI/CD pipeline
- Vulnerability scanning with govulncheck for Go dependencies

### Platform Support

- **Operating Systems**: Linux (amd64, arm64, armv7)
- **Kubernetes**: Compatible with any K8s environment
- **Container Runtimes**: Docker, containerd, CRI-O
- **Go Versions**: 1.24.x only
- **Python Manager**: 3.12+

### Performance

Typical performance metrics:
- **Throughput**: 100k+ requests/second per module
- **Latency**: <100µs average routing latency
- **Protocol Detection**: <50µs per connection
- **Memory**: ~50MB baseline + connection tracking
- **CPU**: <1% idle, scales with load

### Support

- Documentation: See `/proxy-nlb/docs/`
- Issue Tracking: GitHub Issues
- Website: https://www.penguintech.io
- Company: PenguinTech, LLC

### License

Licensed under Limited AGPL3 with preamble for fair use.

```
Copyright (c) 2025 PenguinTech, LLC.
All rights reserved.

This software is licensed under the Limited AGPL3 license with
a preamble for fair use. See LICENSE file for details.
```

### Contributors

- MarchProxy Development Team
- PenguinTech Engineering

### Version Information

- **Release Date**: 2025-12-16
- **Build**: epoch64 timestamp
- **Git Commit**: See version output
- **Next Release Target**: Q1 2026

---

## Version History

### v1.0.0 (2025-12-16)
- Initial production release
- Complete feature set for unified network load balancing
- Comprehensive testing and documentation
- Multi-platform container support

---

## Upgrade Path

For users coming from earlier development versions:

1. **Backup Configuration**: Save current config.yaml
2. **Verify Compatibility**: Check environment variables match new format
3. **Test in Staging**: Deploy to test environment first
4. **Plan Downtime**: Brief restart required for version update
5. **Deploy**: Update image and restart container
6. **Verify**: Check `/healthz` and `/status` endpoints

---

## Thank You

Thank you for using MarchProxy NLB. We're committed to making high-performance network load balancing accessible and reliable for everyone.

For feedback, feature requests, or bug reports, please open an issue on GitHub or contact support at info@penguintech.io.

---

## Next Steps

To get started:

1. Read [QUICK_REFERENCE.md](./QUICK_REFERENCE.md) for common commands
2. Review [CONFIGURATION.md](./CONFIGURATION.md) for setup options
3. Check [USAGE.md](./USAGE.md) for operational guidance
4. Explore [API.md](./API.md) for integration details
5. Review [TESTING.md](./TESTING.md) for validation and testing

Happy load balancing!
