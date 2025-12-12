# Phase 5 Quick Start Guide

## Overview

This guide provides quick commands to implement Phase 5: Enhanced Go Proxy L3/L4 with Enterprise Features.

## Automated Implementation

### Option 1: Run the Implementation Script (Recommended)

```bash
cd /home/penguin/code/MarchProxy
./scripts/implement-phase5-l3l4.sh
```

This script will:
- Copy proxy-egress to proxy-l3l4
- Update all module names and imports
- Create enterprise feature directories
- Add placeholder implementations
- Update dependencies
- Validate the build

### Option 2: Manual Implementation

```bash
# Navigate to MarchProxy directory
cd /home/penguin/code/MarchProxy

# Copy proxy-egress to proxy-l3l4
cp -r proxy-egress proxy-l3l4

# Navigate to new directory
cd proxy-l3l4

# Update module name
sed -i 's/marchproxy-egress/marchproxy-l3l4/g' go.mod

# Update all imports
find . -name "*.go" -type f -exec sed -i 's/marchproxy-egress/marchproxy-l3l4/g' {} \;

# Update descriptions
sed -i 's/MarchProxy Egress/MarchProxy L3L4/g' cmd/proxy/main.go
sed -i 's/egress proxy/L3\/L4 proxy/g' cmd/proxy/main.go

# Create feature directories
mkdir -p internal/qos
mkdir -p internal/multicloud
mkdir -p internal/observability
mkdir -p internal/acceleration/numa

# Update version
echo "v1.0.0.$(date +%s)" > .version

# Add dependencies to go.mod
cat >> go.mod << 'EOF'

// Phase 5 Enterprise Dependencies
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

## Feature Implementation Order

### 1. QoS Traffic Shaping (Week 1)

Implement in `internal/qos/`:

```
qos/
├── token_bucket.go       # Token bucket rate limiting
├── priority_queue.go     # P0-P3 priority queues
├── bandwidth_limiter.go  # Per-service bandwidth limits
├── dscp_marker.go        # DSCP/ECN packet marking
└── qos_test.go           # Unit tests
```

**Key Files to Create:**
- Copy code from `docs/PHASE5_L3L4_IMPLEMENTATION.md` sections 125-364
- Run tests: `go test ./internal/qos/...`

### 2. Multi-Cloud Routing (Week 2)

Implement in `internal/multicloud/`:

```
multicloud/
├── route_table.go        # Cloud route management
├── health_probe.go       # RTT measurement
├── routing_algorithm.go  # Path selection algorithms
├── failover.go           # Active-active failover
└── multicloud_test.go    # Unit tests
```

**Key Files to Create:**
- Copy code from `docs/PHASE5_L3L4_IMPLEMENTATION.md` sections 366-648
- Run tests: `go test ./internal/multicloud/...`

### 3. Deep Observability (Week 2)

Implement in `internal/observability/`:

```
observability/
├── otel_tracer.go        # OpenTelemetry integration
├── jaeger_exporter.go    # Jaeger exporter
├── custom_metrics.go     # Custom business metrics
└── observability_test.go # Unit tests
```

**Key Files to Create:**
- Copy code from `docs/PHASE5_L3L4_IMPLEMENTATION.md` sections 650-691
- Configure Jaeger endpoint
- Run tests: `go test ./internal/observability/...`

### 4. NUMA Optimization (Week 3)

Implement in `internal/acceleration/numa/`:

```
numa/
├── numa_affinity.go      # Thread affinity (Linux)
├── numa_affinity_fallback.go  # Fallback (non-Linux)
├── memory_allocation.go  # NUMA-aware allocation
├── interrupt_handler.go  # Interrupt distribution
└── numa_test.go          # Unit tests
```

**Key Files to Create:**
- Copy code from `docs/PHASE5_L3L4_IMPLEMENTATION.md` sections 693-725
- Implement platform-specific optimizations
- Run tests: `go test ./internal/acceleration/numa/...`

## Integration with Main

Update `cmd/proxy/main.go` to integrate enterprise features:

```go
// Add imports
import (
    "marchproxy-l3l4/internal/qos"
    "marchproxy-l3l4/internal/multicloud"
    "marchproxy-l3l4/internal/observability"
    "marchproxy-l3l4/internal/acceleration/numa"
)

// Add command-line flags
rootCmd.Flags().BoolP("enable-qos", "", false, "Enable QoS traffic shaping")
rootCmd.Flags().BoolP("enable-multicloud", "", false, "Enable multi-cloud routing")
rootCmd.Flags().BoolP("enable-otel", "", false, "Enable OpenTelemetry tracing")
rootCmd.Flags().BoolP("numa-optimization", "", false, "Enable NUMA optimizations")

// Initialize managers in runProxy()
qosEnabled, _ := cmd.Flags().GetBool("enable-qos")
qosManager := qos.NewManager(qosEnabled)

multicloudEnabled, _ := cmd.Flags().GetBool("enable-multicloud")
multicloudManager := multicloud.NewManager(multicloudEnabled)

otelEnabled, _ := cmd.Flags().GetBool("enable-otel")
observabilityManager := observability.NewManager("marchproxy-l3l4", otelEnabled)

numaEnabled, _ := cmd.Flags().GetBool("numa-optimization")
numaManager := numa.NewManager(numaEnabled)

// Optimize for NUMA
if numaEnabled {
    if err := numaManager.OptimizeForNUMA(); err != nil {
        fmt.Printf("Warning: NUMA optimization failed: %v\n", err)
    }
}
```

## Docker Build

### Create Dockerfile

Use the Dockerfile from `docs/PHASE5_L3L4_IMPLEMENTATION.md` (lines 741-769).

### Build Image

```bash
cd /home/penguin/code/MarchProxy/proxy-l3l4
docker build -t marchproxy/proxy-l3l4:v1.0.0 .
```

### Test Container

```bash
docker run -it --rm \
  --cap-add NET_ADMIN \
  --cap-add SYS_ADMIN \
  -e MANAGER_URL=http://manager:8000 \
  -e CLUSTER_API_KEY=test-key \
  -e ENABLE_QOS=true \
  -e ENABLE_MULTICLOUD=true \
  -e ENABLE_OTEL=true \
  -e NUMA_OPTIMIZATION=true \
  marchproxy/proxy-l3l4:v1.0.0
```

## GitHub Actions

Create `.github/workflows/proxy-l3l4-ci.yml`:

Copy the workflow from `docs/PHASE5_L3L4_IMPLEMENTATION.md` (lines 804-851).

## Testing

### Run Unit Tests

```bash
cd /home/penguin/code/MarchProxy/proxy-l3l4

# Test all packages
go test -v ./...

# Test with coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

### Run Integration Tests

```bash
# Start docker-compose environment
docker-compose up -d manager postgres

# Wait for services to be ready
sleep 10

# Run integration tests
go test -v -tags=integration ./test/...
```

### Performance Testing

```bash
# Build optimized binary
go build -o marchproxy-l3l4 -ldflags="-s -w" cmd/proxy/main.go

# Run benchmark
go test -bench=. -benchmem ./...

# Profile CPU
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Profile memory
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## Update docker-compose.yml

Add proxy-l3l4 service to main `docker-compose.yml`:

```yaml
services:
  proxy-l3l4:
    build:
      context: ./proxy-l3l4
      dockerfile: Dockerfile
    image: marchproxy/proxy-l3l4:latest
    container_name: marchproxy-proxy-l3l4
    environment:
      - MANAGER_URL=http://manager:8000
      - CLUSTER_API_KEY=${CLUSTER_API_KEY:-default-key}
      - LOG_LEVEL=INFO
      - ENABLE_QOS=true
      - ENABLE_MULTICLOUD=true
      - ENABLE_OTEL=true
      - NUMA_OPTIMIZATION=true
      - JAEGER_ENDPOINT=http://jaeger:14268/api/traces
    ports:
      - "9080:8080"  # Proxy port
      - "9081:8081"  # Metrics port
    networks:
      - marchproxy
    depends_on:
      - manager
      - jaeger
    cap_add:
      - NET_ADMIN
      - SYS_ADMIN
    privileged: true
    restart: unless-stopped
```

## Verification Checklist

- [ ] proxy-l3l4 directory created
- [ ] Module name updated to marchproxy-l3l4
- [ ] All imports updated
- [ ] Enterprise feature directories created
- [ ] Dependencies added and downloaded
- [ ] QoS implementation complete
- [ ] Multi-cloud routing implementation complete
- [ ] Observability integration complete
- [ ] NUMA optimization complete
- [ ] Unit tests passing (80%+ coverage)
- [ ] Integration tests passing
- [ ] Docker image builds successfully
- [ ] Docker container runs without errors
- [ ] GitHub Actions workflow created
- [ ] Documentation updated
- [ ] Performance targets validated

## Troubleshooting

### Build Fails

```bash
# Clean and rebuild
cd proxy-l3l4
go clean -modcache
go mod tidy
go build -v ./...
```

### Import Errors

```bash
# Verify module name
grep "^module" go.mod

# Should output: module marchproxy-l3l4

# Re-run import updates
find . -name "*.go" -type f -exec sed -i 's/marchproxy-egress/marchproxy-l3l4/g' {} \;
```

### Docker Build Fails

```bash
# Check Dockerfile exists
ls -l Dockerfile

# Build with verbose output
docker build --no-cache --progress=plain -t marchproxy/proxy-l3l4:debug .
```

### Tests Fail

```bash
# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestTokenBucket ./internal/qos/

# Check test coverage
go test -cover ./...
```

## Performance Validation

### Throughput Test

```bash
# Use iperf3 for throughput testing
# On server:
iperf3 -s -p 9080

# On client:
iperf3 -c proxy-l3l4 -p 9080 -t 60 -P 10
```

### Latency Test

```bash
# Use wrk for HTTP latency testing
wrk -t 12 -c 400 -d 30s http://proxy-l3l4:9080/

# Check p99 latency in output
```

### Connection Test

```bash
# Test concurrent connections
ab -n 1000000 -c 10000 http://proxy-l3l4:9080/
```

## Next Steps After Implementation

1. **Merge to develop branch**: Create PR with all changes
2. **Run CI/CD pipeline**: Ensure all checks pass
3. **Update main README.md**: Add proxy-l3l4 to architecture
4. **Create release**: Tag version v1.0.0
5. **Deploy to staging**: Test in staging environment
6. **Performance benchmark**: Validate 100+ Gbps target
7. **Documentation review**: Ensure all docs are current
8. **Security audit**: Run security scans
9. **Production deployment**: Deploy to production clusters

## Resources

- **Full Implementation Guide**: `docs/PHASE5_L3L4_IMPLEMENTATION.md`
- **Architecture Docs**: `docs/ARCHITECTURE.md`
- **Performance Guide**: `docs/PERFORMANCE.md`
- **API Reference**: `docs/api/`
- **Project README**: `README.md`

## Support

For questions or issues:
- Review implementation guide
- Check troubleshooting section
- Review existing proxy-egress code
- Consult Go documentation
- Check OpenTelemetry docs
