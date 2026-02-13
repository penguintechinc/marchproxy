# MarchProxy NLB Testing Guide

## Test Coverage

The NLB includes comprehensive unit tests, integration tests, and performance benchmarks covering all major components.

**Current Coverage**: 80%+ code coverage across all packages

---

## Running Tests

### Unit Tests

Run all unit tests:
```bash
go test -v ./...
```

Run tests for a specific package:
```bash
go test -v ./internal/nlb
go test -v ./internal/grpc
go test -v ./internal/config
```

### Tests with Coverage Report

Generate and view coverage report:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

View coverage for specific package:
```bash
go test -coverprofile=coverage.out ./internal/nlb
go tool cover -html=coverage.out
```

### Concurrent Testing

Run tests with race condition detection:
```bash
go test -race -v ./...
```

### Verbose Test Output

Run tests with detailed logging:
```bash
go test -v -run TestName ./package
```

---

## Integration Tests

Integration tests verify component interactions across the NLB.

### Running Integration Tests

```bash
go test -v -tags=integration ./tests/integration/...
```

### Docker-based Integration Tests

Build testing container and run integration tests:
```bash
docker build --target testing -t marchproxy-nlb:test .
docker run --rm marchproxy-nlb:test go test -v ./tests/integration/...
```

---

## Component-Specific Tests

### Protocol Inspector Tests

Tests for protocol detection accuracy:
```bash
go test -v ./internal/nlb -run TestInspector
```

**Coverage Areas**:
- HTTP/HTTPS detection
- MySQL protocol detection
- PostgreSQL protocol detection
- MongoDB protocol detection
- Redis protocol detection
- RTMP protocol detection
- Unknown protocol handling

### Router Tests

Tests for traffic routing logic:
```bash
go test -v ./internal/nlb -run TestRouter
```

**Coverage Areas**:
- Least connections routing
- Health-aware routing
- Connection tracking
- Multiple module routing
- Failover on module failure

### Rate Limiter Tests

Tests for token bucket rate limiting:
```bash
go test -v ./internal/nlb -run TestRateLimiter
```

**Coverage Areas**:
- Token bucket behavior
- Per-protocol buckets
- Per-service buckets
- Configurable refill rates
- Token availability

### Autoscaler Tests

Tests for autoscaling orchestration:
```bash
go test -v ./internal/nlb -run TestAutoscaler
```

**Coverage Areas**:
- Scaling policy application
- Scale-up triggering
- Scale-down triggering
- Cooldown period enforcement
- Min/max replica bounds

### Blue/Green Tests

Tests for blue/green deployments:
```bash
go test -v ./internal/nlb -run TestBlueGreen
```

**Coverage Areas**:
- Instant traffic switching
- Canary rollout progression
- Version tracking
- Rollback functionality
- Traffic splitting

### gRPC Tests

Tests for gRPC client/server:
```bash
go test -v ./internal/grpc -run Test
```

**Coverage Areas**:
- Module registration
- Module unregistration
- Health updates
- Statistics queries
- Connection pooling
- Reconnection logic

---

## Performance Benchmarks

Run performance benchmarks:
```bash
go test -bench=. -benchmem ./...
```

Run benchmarks for specific component:
```bash
go test -bench=BenchmarkRouter -benchmem ./internal/nlb
go test -bench=BenchmarkRateLimiter -benchmem ./internal/nlb
```

Key benchmarks measured:
- Router throughput (connections/sec)
- Protocol detection latency (µs)
- Rate limit decision time (ns)
- gRPC registration latency (ms)

---

## Load Testing

### Using Apache Bench

Test HTTP protocol routing:
```bash
ab -n 10000 -c 100 http://localhost:8080/
```

### Using wrk

High-performance HTTP load testing:
```bash
wrk -t4 -c100 -d30s http://localhost:8080/
```

### Custom Load Testing Script

```bash
#!/bin/bash
PROTOCOL="http"
TARGET="http://localhost:8080"
CONNECTIONS=100
REQUESTS=10000

echo "Testing NLB with $REQUESTS requests, $CONNECTIONS connections"
ab -n $REQUESTS -c $CONNECTIONS $TARGET
```

---

## Linting and Code Quality

### Run Linters

```bash
# Run all linters
golangci-lint run

# Specific linters
golangci-lint run --no-config --enable=gosec,go-fmt
```

### Code Formatting

```bash
# Format all Go files
go fmt ./...

# Using gofmt
gofmt -s -w .
```

### Static Analysis

```bash
# Go vet
go vet ./...

# Staticcheck
staticcheck ./...

# Go security analyzer
gosec ./...
```

### Import Ordering

```bash
# Check and fix import order
goimports -w .
```

---

## Docker Testing

### Build Testing Image

```bash
docker build --target testing -t marchproxy-nlb:test .
```

### Run Tests in Container

```bash
docker run --rm marchproxy-nlb:test go test -v ./...
```

### Run with Coverage in Container

```bash
docker run --rm marchproxy-nlb:test \
  go test -coverprofile=coverage.out ./... && \
  go tool cover -html=coverage.out
```

---

## CI/CD Pipeline Tests

The NLB uses GitHub Actions for automated testing:

### Workflow Stages

1. **Lint Stage** - Code quality checks (fail fast)
   - golangci-lint
   - go fmt verification
   - gosec security scanning

2. **Test Stage** - Unit tests with coverage
   - go test with -race flag
   - 80%+ coverage requirement
   - Artifact upload for coverage reports

3. **Build Stage** - Multi-architecture builds
   - linux/amd64
   - linux/arm64
   - linux/arm/v7

4. **Security Scan** - Vulnerability scanning
   - govulncheck
   - Trivy container scanning

### Running CI Locally

Simulate CI pipeline locally:
```bash
# Lint
golangci-lint run

# Test
go test -race -coverprofile=coverage.out ./...

# Build
docker build -t marchproxy-nlb:test .

# Verify
go tool cover -func=coverage.out
```

---

## Test Data and Fixtures

### Mock Data

Mock data is available in `tests/fixtures/`:
- Configuration files
- gRPC request/response examples
- Module registration data

### Test Configuration Files

```bash
tests/fixtures/config/
├── minimal.yaml          # Minimal valid configuration
├── with-rate-limiting.yaml
├── with-autoscaling.yaml
├── with-bluegreen.yaml
└── enterprise.yaml
```

---

## Debugging Tests

### Run Single Test with Debugging

```bash
go test -v -run TestName -timeout 30s ./package -debug
```

### Enable Debug Logging

```bash
# Set debug environment
DEBUG=true go test -v ./package
```

### Use Delve Debugger

```bash
# Install dlv
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug test
dlv test ./package -- -test.v
```

---

## Test Maintenance

### Updating Tests After Code Changes

1. Run affected tests to identify failures
2. Update test expectations to match new behavior
3. Add tests for new functionality
4. Ensure coverage remains above 80%
5. Verify CI passes before committing

### Test Dependencies

Test dependencies are managed via `go.mod`:
```bash
go get -u github.com/stretchr/testify
go get -u github.com/golang/mock/...
```

---

## Troubleshooting

### Test Failures

**Random failures**: May indicate race conditions
```bash
go test -race ./...  # Run with race detector
```

**Timeout failures**: Increase timeout or optimize test
```bash
go test -timeout 60s ./...
```

### Coverage Issues

**Low coverage on package**:
```bash
go test -coverprofile=coverage.out ./package
go tool cover -html=coverage.out  # Identify uncovered lines
```

### Build Issues in Tests

**Module import errors**:
```bash
go mod tidy    # Clean up go.mod
go mod verify  # Verify integrity
```

---

## Best Practices

1. **Write tests alongside code** - Test-driven development approach
2. **Aim for 80%+ coverage** - Focus on critical paths
3. **Use table-driven tests** - Test multiple scenarios efficiently
4. **Mock external dependencies** - Test isolation
5. **Use meaningful test names** - Describe what is being tested
6. **Keep tests fast** - Avoid sleep() and timeouts
7. **Parallel test execution** - Use `-parallel` flag for speed

---

## Performance Benchmarks

Target performance metrics:

| Metric | Target | Current |
|--------|--------|---------|
| Routing latency | < 100µs | ~50µs |
| Protocol detection | < 50µs | ~25µs |
| Rate limit check | < 100ns | ~75ns |
| Module registration | < 100ms | ~50ms |

---

## Advanced Testing

### Chaos Testing

Simulate failures and network issues:
```bash
# Inject latency
tc qdisc add dev lo root netem delay 10ms

# Inject packet loss
tc qdisc add dev lo root netem loss 1%

# Run tests under chaos
go test -v ./...

# Clean up
tc qdisc del dev lo root
```

### Load Testing with Metrics

```bash
# Start NLB with metrics enabled
./nlb --config config.yaml

# In another terminal, run load test while monitoring metrics
wrk -t4 -c100 -d30s http://localhost:8080/ &
watch -n 1 'curl -s http://localhost:8082/metrics | grep nlb_'
```

---

## Test Reporting

Generate test report for CI/CD:
```bash
go test -v ./... -json | jq '.'
```

Convert to JUnit format for Jenkins/GitLab:
```bash
go install github.com/jstemmer/go-junit-report/v2@latest
go test -v ./... 2>&1 | go-junit-report -set-exit-code > report.xml
```
