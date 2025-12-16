# Phase 5 Documentation Index

## Overview

Phase 5 introduces **proxy-l3l4** - an enterprise-grade Layer 3/Layer 4 proxy with advanced networking features targeting 100+ Gbps throughput and sub-millisecond latency.

## Quick Navigation

### 🚀 Get Started
- **[Quick Start Guide](PHASE5_QUICKSTART.md)** - Step-by-step implementation instructions
- **[Implementation Summary](../PHASE5_SUMMARY.md)** - High-level overview and status
- **[Automated Script](../scripts/implement-phase5-l3l4.sh)** - One-command setup

### 📚 Complete Documentation
- **[Full Implementation Guide](PHASE5_L3L4_IMPLEMENTATION.md)** - Comprehensive 916-line specification

## What is proxy-l3l4?

proxy-l3l4 is an enhanced version of proxy-egress with four major enterprise features:

### 1. QoS Traffic Shaping 🎯
Advanced quality of service for bandwidth management:
- **Token Bucket**: Rate limiting with burst allowance
- **Priority Queues**: P0 (critical) through P3 (best-effort)
- **DSCP Marking**: Packet classification for QoS-aware networks
- **Bandwidth Guarantees**: Minimum bandwidth allocation per service

### 2. Multi-Cloud Routing ☁️
Intelligent routing across cloud providers:
- **Cloud Support**: AWS, GCP, Azure, on-premises
- **Smart Selection**: Latency-based, cost-based, weighted algorithms
- **Active-Active Failover**: Sub-second failover times
- **Health Monitoring**: RTT measurement and packet loss detection

### 3. Deep Observability 🔍
Comprehensive monitoring and tracing:
- **OpenTelemetry**: Distributed request tracing
- **Jaeger Integration**: Trace visualization and analysis
- **Custom Metrics**: Business-specific KPIs
- **Flow Analysis**: Detailed per-connection statistics

### 4. NUMA Optimization ⚡
Performance optimization for NUMA systems:
- **CPU Affinity**: Pin workers to NUMA nodes
- **Memory Locality**: NUMA-aware buffer allocation
- **Interrupt Distribution**: Balanced across cores
- **Cache Optimization**: Minimize cache line bouncing

## Performance Targets

| Metric | Target | Description |
|--------|--------|-------------|
| **Throughput** | 100+ Gbps | Per proxy instance |
| **Latency (p50)** | <100μs | Median latency |
| **Latency (p99)** | <1ms | 99th percentile |
| **Latency (p99.9)** | <5ms | 99.9th percentile |
| **Connections** | 10M+ | Concurrent connections |
| **Packet Rate** | 100M+ pps | Packets per second |
| **CPU Efficiency** | <5% @ 10 Gbps | CPU utilization |

## Implementation Timeline

### Week 1: QoS Traffic Shaping
- [ ] Implement token bucket algorithm
- [ ] Implement priority queues (P0-P3)
- [ ] Implement DSCP marker
- [ ] Implement bandwidth limiter
- [ ] Write unit tests (80%+ coverage)
- [ ] Benchmark performance

### Week 2: Multi-Cloud Routing & Observability
- [ ] Implement route table management
- [ ] Implement health probe system
- [ ] Implement routing algorithms
- [ ] Integrate OpenTelemetry
- [ ] Configure Jaeger exporter
- [ ] Write unit tests (80%+ coverage)

### Week 3: NUMA Optimization & Testing
- [ ] Implement CPU affinity management
- [ ] Implement NUMA-aware memory allocation
- [ ] Create integration test suite
- [ ] Perform load testing
- [ ] Validate performance targets
- [ ] Write documentation

### Week 4: Integration & Deployment
- [ ] Integrate features into main.go
- [ ] Build Docker image
- [ ] Create GitHub Actions workflow
- [ ] Update docker-compose.yml
- [ ] Final performance validation
- [ ] Code review and merge

## File Structure

```
proxy-l3l4/
├── cmd/
│   └── proxy/
│       └── main.go                 # Enhanced main with enterprise features
├── internal/
│   ├── qos/                        # QoS Traffic Shaping
│   │   ├── token_bucket.go         # Rate limiting
│   │   ├── priority_queue.go       # Priority management
│   │   ├── bandwidth_limiter.go    # Bandwidth control
│   │   ├── dscp_marker.go          # Packet marking
│   │   └── qos_test.go             # Unit tests
│   ├── multicloud/                 # Multi-Cloud Routing
│   │   ├── route_table.go          # Route management
│   │   ├── health_probe.go         # Health checking
│   │   ├── routing_algorithm.go    # Path selection
│   │   ├── failover.go             # Failover logic
│   │   └── multicloud_test.go      # Unit tests
│   ├── observability/              # Deep Observability
│   │   ├── otel_tracer.go          # OpenTelemetry
│   │   ├── jaeger_exporter.go      # Jaeger integration
│   │   ├── custom_metrics.go       # Custom metrics
│   │   └── observability_test.go   # Unit tests
│   └── acceleration/
│       └── numa/                   # NUMA Optimization
│           ├── numa_affinity.go    # CPU affinity (Linux)
│           ├── numa_affinity_fallback.go  # Fallback
│           ├── memory_allocation.go # Memory optimization
│           └── numa_test.go        # Unit tests
├── Dockerfile                      # Multi-stage build
├── go.mod                          # Module definition
├── go.sum                          # Dependencies
└── .version                        # Version tracking
```

## Quick Start Commands

### Option 1: Automated Setup (Recommended)

```bash
cd /home/penguin/code/MarchProxy
./scripts/implement-phase5-l3l4.sh
```

### Option 2: Manual Setup

```bash
# Copy baseline
cp -r proxy-egress proxy-l3l4
cd proxy-l3l4

# Update module
sed -i 's/marchproxy-egress/marchproxy-l3l4/g' go.mod
find . -name "*.go" -exec sed -i 's/marchproxy-egress/marchproxy-l3l4/g' {} \;

# Create directories
mkdir -p internal/{qos,multicloud,observability,acceleration/numa}

# Add dependencies
cat >> go.mod << 'EOF'
require (
    go.opentelemetry.io/otel v1.31.0
    go.opentelemetry.io/otel/exporters/jaeger v1.31.0
    go.opentelemetry.io/otel/sdk v1.31.0
    go.opentelemetry.io/otel/trace v1.31.0
    github.com/klauspost/compress v1.17.0
    golang.org/x/time v0.8.0
)
EOF

# Download dependencies
go mod tidy

# Test build
go build -o marchproxy-l3l4 cmd/proxy/main.go
```

## Building and Testing

### Build Binary

```bash
cd proxy-l3l4
go build -o marchproxy-l3l4 cmd/proxy/main.go
```

### Run Tests

```bash
# Unit tests
go test -v ./...

# With coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage
go tool cover -html=coverage.out
```

### Build Docker Image

```bash
docker build -t marchproxy/proxy-l3l4:v1.0.0 .
```

### Run Container

```bash
docker run -it --rm \
  --cap-add NET_ADMIN \
  --cap-add SYS_ADMIN \
  --privileged \
  -e MANAGER_URL=http://manager:8000 \
  -e CLUSTER_API_KEY=your-api-key \
  -e ENABLE_QOS=true \
  -e ENABLE_MULTICLOUD=true \
  -e ENABLE_OTEL=true \
  -e NUMA_OPTIMIZATION=true \
  -p 9080:8080 \
  -p 9081:8081 \
  marchproxy/proxy-l3l4:v1.0.0
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MANAGER_URL` | - | Manager API endpoint (required) |
| `CLUSTER_API_KEY` | - | Cluster authentication key (required) |
| `LOG_LEVEL` | INFO | Logging level (DEBUG, INFO, WARN, ERROR) |
| `ENABLE_QOS` | false | Enable QoS traffic shaping |
| `ENABLE_MULTICLOUD` | false | Enable multi-cloud routing |
| `ENABLE_OTEL` | false | Enable OpenTelemetry tracing |
| `NUMA_OPTIMIZATION` | false | Enable NUMA optimizations |
| `JAEGER_ENDPOINT` | - | Jaeger collector endpoint |

### Command-Line Flags

```bash
marchproxy-l3l4 \
  --manager-url http://manager:8000 \
  --cluster-api-key your-key \
  --listen-port 8080 \
  --admin-port 8081 \
  --log-level INFO \
  --enable-qos \
  --enable-multicloud \
  --enable-otel \
  --numa-optimization
```

## Code Examples

### QoS Token Bucket

```go
import "marchproxy-l3l4/internal/qos"

// Create token bucket: 100MB capacity, 10MB/s refill rate
bucket := qos.NewTokenBucket(100000000, 10000000)

// Try to consume 1MB
if bucket.Take(1000000) {
    // Allowed - process request
} else {
    // Denied - rate limited
}
```

### Multi-Cloud Routing

```go
import "marchproxy-l3l4/internal/multicloud"

// Create route table
routeTable := multicloud.NewRouteTable()

// Add route
route := &multicloud.Route{
    ID:          "aws-us-east-1",
    Destination: parseIPNet("10.0.0.0/8"),
    Gateway:     net.ParseIP("10.1.1.1"),
    Provider:    multicloud.ProviderAWS,
    Region:      "us-east-1",
    Cost:        0.01,
    Latency:     5 * time.Millisecond,
    Active:      true,
}
routeTable.AddRoute(route)

// Select best route
algorithm := &multicloud.LatencyBasedRouting{}
bestRoute := routeTable.GetBestRoute(destIP, algorithm)
```

### OpenTelemetry Tracing

```go
import "marchproxy-l3l4/internal/observability"

// Create tracer
tracer := observability.NewTracerManager("marchproxy-l3l4")

// Trace connection
ctx, span := tracer.TraceConnection(ctx, srcIP, dstIP, srcPort, dstPort)
defer span.End()

// Connection processing...
```

### NUMA Optimization

```go
import "marchproxy-l3l4/internal/acceleration/numa"

// Create NUMA manager
numaManager := numa.NewManager(true)

// Optimize for NUMA
if err := numaManager.OptimizeForNUMA(); err != nil {
    log.Printf("NUMA optimization failed: %v", err)
}

// Workers are now pinned to NUMA nodes
```

## Testing Examples

### QoS Unit Test

```go
func TestTokenBucket(t *testing.T) {
    bucket := qos.NewTokenBucket(1000, 100)

    // Should allow first request
    if !bucket.Take(100) {
        t.Error("Expected token bucket to allow request")
    }

    // Should deny after capacity exceeded
    if bucket.Take(1000) {
        t.Error("Expected token bucket to deny request")
    }
}
```

### Multi-Cloud Integration Test

```go
func TestMultiCloudFailover(t *testing.T) {
    routeTable := multicloud.NewRouteTable()

    // Add primary and backup routes
    // ...

    // Simulate primary failure
    routeTable.UpdateRouteHealth("primary", 0, 1.0)

    // Should failover to backup
    route := routeTable.GetBestRoute(destIP, algorithm)
    if route.ID != "backup" {
        t.Error("Expected failover to backup route")
    }
}
```

## Performance Benchmarking

### Throughput Benchmark

```bash
# Using iperf3
iperf3 -c proxy-l3l4 -p 9080 -t 60 -P 10

# Expected: >100 Gbps
```

### Latency Benchmark

```bash
# Using wrk
wrk -t 12 -c 400 -d 30s http://proxy-l3l4:9080/

# Expected: p99 <1ms
```

### Connection Benchmark

```bash
# Using Apache Bench
ab -n 1000000 -c 10000 http://proxy-l3l4:9080/

# Expected: >10M concurrent connections
```

## Troubleshooting

### Build Fails

**Problem**: Import errors or module not found

**Solution**:
```bash
go clean -modcache
go mod tidy
go build -v ./...
```

### Docker Build Fails

**Problem**: Dockerfile not found or build errors

**Solution**:
```bash
# Verify Dockerfile exists
ls -l Dockerfile

# Build with verbose output
docker build --no-cache --progress=plain -t marchproxy/proxy-l3l4:debug .
```

### Tests Fail

**Problem**: Unit tests failing

**Solution**:
```bash
# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestTokenBucket ./internal/qos/
```

### Performance Below Targets

**Problem**: Not achieving 100+ Gbps throughput

**Solution**:
- Enable NUMA optimization
- Verify eBPF/XDP acceleration is active
- Check network interface capabilities
- Review CPU affinity settings
- Monitor system resources

## Documentation Reference

### Phase 5 Specific
- [PHASE5_L3L4_IMPLEMENTATION.md](PHASE5_L3L4_IMPLEMENTATION.md) - Complete implementation (916 lines)
- [PHASE5_QUICKSTART.md](PHASE5_QUICKSTART.md) - Quick start guide
- [../PHASE5_SUMMARY.md](../PHASE5_SUMMARY.md) - Implementation summary
- [../scripts/implement-phase5-l3l4.sh](../scripts/implement-phase5-l3l4.sh) - Setup script

### General Documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [PERFORMANCE.md](PERFORMANCE.md) - Performance tuning
- [WORKFLOWS.md](WORKFLOWS.md) - CI/CD workflows
- [STANDARDS.md](STANDARDS.md) - Development standards
- [../README.md](../README.md) - Project overview

## CI/CD Integration

### GitHub Actions Workflow

Create `.github/workflows/proxy-l3l4-ci.yml`:

```yaml
name: Proxy L3/L4 CI/CD

on:
  push:
    branches: [main, develop]
    paths:
      - 'proxy-l3l4/**'
      - '.version'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          working-directory: proxy-l3l4

  test:
    runs-on: ubuntu-latest
    needs: [lint]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Run tests
        working-directory: proxy-l3l4
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./proxy-l3l4/coverage.out

  build:
    runs-on: ubuntu-latest
    needs: [test]
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: docker build -t marchproxy/proxy-l3l4:${GITHUB_SHA::8} proxy-l3l4
```

## Success Criteria

- [x] Documentation complete
- [x] Automated setup script created
- [ ] proxy-l3l4 directory created
- [ ] Module name updated
- [ ] QoS implementation complete
- [ ] Multi-cloud routing complete
- [ ] Observability integration complete
- [ ] NUMA optimization complete
- [ ] Unit tests passing (80%+ coverage)
- [ ] Integration tests passing
- [ ] Docker image builds
- [ ] Performance targets met
- [ ] GitHub Actions workflow passing
- [ ] Code review approved

## Additional Resources

### Go Libraries
- [OpenTelemetry Go](https://github.com/open-telemetry/opentelemetry-go)
- [Jaeger Client Go](https://github.com/jaegertracing/jaeger-client-go)
- [golang.org/x/time](https://pkg.go.dev/golang.org/x/time)
- [golang.org/x/sys](https://pkg.go.dev/golang.org/x/sys)

### External Documentation
- [OpenTelemetry Docs](https://opentelemetry.io/docs/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [NUMA Best Practices](https://www.kernel.org/doc/html/latest/vm/numa.html)
- [DSCP Values](https://www.iana.org/assignments/dscp-registry/dscp-registry.xhtml)

### Cloud Provider Docs
- [AWS VPC](https://docs.aws.amazon.com/vpc/)
- [GCP VPC](https://cloud.google.com/vpc/docs)
- [Azure Virtual Network](https://docs.microsoft.com/azure/virtual-network/)

## Support and Contribution

### Getting Help
1. Review this documentation
2. Check [PHASE5_QUICKSTART.md](PHASE5_QUICKSTART.md) troubleshooting
3. Review proxy-egress implementation for reference
4. Consult Go and OpenTelemetry documentation

### Contributing
1. Follow [STANDARDS.md](STANDARDS.md) development standards
2. Ensure 80%+ test coverage
3. Update documentation
4. Submit pull request for review

## Version History

- **v1.0.0** (2025-12-12) - Initial Phase 5 implementation
  - QoS traffic shaping
  - Multi-cloud routing
  - Deep observability
  - NUMA optimization

## License

Limited AGPL-3.0 with Contributor Employer Exception

See [LICENSE](../LICENSE) for details.

---

**Last Updated**: 2025-12-12
**Status**: Ready for Implementation
**Maintainer**: MarchProxy Team
