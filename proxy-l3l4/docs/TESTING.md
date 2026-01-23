# TESTING.md - proxy-l3l4 Testing Guide

## Overview

proxy-l3l4 uses comprehensive testing strategies including unit tests, integration tests, eBPF testing, and Docker-based testing.

## Prerequisites

- Go 1.24+
- Docker and Docker Compose
- Linux kernel 5.8+ (for eBPF features)
- libbpf development files
- clang and llvm toolchain

## Running Tests Locally

### Unit Tests

Run all unit tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

Run tests with race detection:
```bash
go test -v -race ./...
```

Run specific test:
```bash
go test -v ./internal/multicloud -run TestRouterSelection
```

### Test Coverage Requirements

Minimum coverage: 80%

Generate coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Docker-Based Testing

### Build and Run Tests

Using Docker testing stage:
```bash
docker build --target testing -t proxy-l3l4:test .
docker run --rm proxy-l3l4:test
```

### Development Container

Start development container with hot reload:
```bash
docker build --target development -t proxy-l3l4:dev .
docker run -it -v $(pwd):/app -p 8081:8081 -p 8082:8082 proxy-l3l4:dev
```

The development container uses `air` for hot reloading on code changes.

## eBPF Testing

### eBPF Compilation

Test eBPF program compilation:
```bash
cd internal/acceleration
make compile
```

### eBPF Unit Tests

Test eBPF programs with kernel module simulation:
```bash
go test -v ./internal/acceleration/... -run TestXDP
go test -v ./internal/acceleration/... -run TestAFXDP
```

### eBPF Kernel Tests

Test eBPF programs with actual kernel (requires elevated privileges):

1. Start debug container:
```bash
docker build --target debug -t proxy-l3l4:debug .
docker run -it --privileged --cap-add=SYS_RESOURCE --cap-add=NET_ADMIN \
    -v $(pwd):/app proxy-l3l4:debug bash
```

2. Load and test eBPF programs:
```bash
cd /app
go run ./internal/acceleration/test_ebpf.go
```

3. Verify with packet inspection:
```bash
tcpdump -i eth0 -n
```

## Integration Tests

### Docker Compose Integration Testing

Start full environment:
```bash
docker-compose -f docker-compose.test.yml up
```

Run integration tests against running services:
```bash
docker-compose -f docker-compose.test.yml exec proxy-l3l4 \
    go test -v -tags=integration ./...
```

## Test Structure

### Unit Test Locations

- `internal/acceleration/*_test.go` - eBPF and acceleration tests
- `internal/numa/*_test.go` - NUMA affinity tests
- `internal/multicloud/*_test.go` - Routing and health check tests
- `internal/qos/*_test.go` - QoS shaping tests
- `internal/zerotrust/*_test.go` - Zero-trust policy tests
- `internal/observability/*_test.go` - Metrics and tracing tests

### Test Patterns

Mocking HTTP clients:
```go
mockClient := &http.Client{
    Transport: &mockRoundTripper{
        response: &http.Response{
            StatusCode: 200,
            Body:       io.NopCloser(bytes.NewReader([]byte(`{"status":"ok"}`))),
        },
    },
}
```

Testing metrics:
```go
metrics := observability.NewMetrics("test")
gauge := metrics.ConnectionsActive
gauge.Add(1)
// Assert value increased
```

Testing eBPF programs:
```go
ebpf := acceleration.NewXDPProgram(cfg)
err := ebpf.Load()
require.NoError(t, err)
defer ebpf.Unload()
```

## Benchmark Tests

Run benchmarks:
```bash
go test -bench=. -benchmem ./...
```

Run specific benchmark:
```bash
go test -bench=BenchmarkRouting -benchmem ./internal/multicloud
```

## Performance Testing

### Load Testing with Docker

Build load test container:
```dockerfile
FROM golang:1.24-bookworm
RUN apt-get update && apt-get install -y hey
WORKDIR /app
COPY . .
RUN go build -o load-test ./cmd/loadtest
CMD ["./load-test"]
```

Run load test:
```bash
docker run --rm --network host -e TARGET=http://localhost:8081 \
    proxy-l3l4:loadtest
```

### Network Performance Testing

Test with iperf3:
```bash
# Start iperf3 server
docker run -d -p 5201:5201 networkstatic/iperf3 iperf3 -s

# Run iperf3 client
docker run --rm --network host networkstatic/iperf3 \
    iperf3 -c localhost -t 30
```

## CI/CD Testing

### GitHub Actions

Tests run automatically on:
- Pull requests to `main` and `develop`
- Commits to `main` and `develop`
- Manual trigger via workflow dispatch

Workflow file: `.github/workflows/proxy-l3l4-ci.yml`

**Testing stages**:
1. **Lint** - golangci-lint, gosec, go fmt, go vet
2. **Unit Tests** - go test -race -cover
3. **Integration Tests** - Docker Compose
4. **Security Scan** - Trivy, govulncheck
5. **Build** - Multi-architecture Docker builds

## Debugging Tests

### Enable Debug Logging

Set log level during tests:
```go
logger := logrus.New()
logger.SetLevel(logrus.DebugLevel)
```

Or via environment variable:
```bash
LOG_LEVEL=debug go test -v ./...
```

### Run Single Test with Debugger

Using dlv debugger:
```bash
dlv test ./internal/multicloud -- -test.run TestRouterSelection
```

### Inspect Test Container

Keep container running after tests fail:
```bash
docker build --target testing -t proxy-l3l4:test .
docker run -it --entrypoint bash proxy-l3l4:test
```

Inside container, run tests manually:
```bash
go test -v ./...
```

## Testing Checklist

Before marking tests as complete:
- [ ] All unit tests pass locally
- [ ] Code coverage >= 80%
- [ ] Race detector finds no issues
- [ ] Integration tests pass in Docker
- [ ] Benchmarks show acceptable performance
- [ ] No compiler warnings
- [ ] No security issues detected
- [ ] CI/CD pipeline passes all checks

## Common Issues

### eBPF Load Failures

**Issue**: Permission denied loading eBPF programs
**Solution**: Run with `--privileged` flag or in debug container

### NUMA Tests Fail

**Issue**: System doesn't have NUMA capability
**Solution**: Tests automatically skip on non-NUMA systems

### Port Already in Use

**Issue**: Metrics port 8082 already bound
**Solution**: Kill existing process or change port via config

### Docker Network Issues

**Issue**: Container can't reach manager service
**Solution**: Ensure docker-compose network is created and services are connected

## Advanced Testing

### Custom Test Configuration

Create test config file:
```yaml
# test-config.yaml
listen_port: 8081
metrics_addr: ":8082"
enable_numa: false
enable_tracing: false
enable_xdp: false
enable_afxdp: false
```

Run with custom config:
```bash
go test -v ./... -args -config test-config.yaml
```

### Mutation Testing

Detect weak tests using mutagen:
```bash
go install github.com/mdwhatley/mutagen@latest
mutagen test ./...
```

Identifies tests that don't catch bugs effectively.
