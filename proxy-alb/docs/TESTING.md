# MarchProxy ALB - Testing Guide

## Running Tests

### Prerequisites
- Go 1.24+
- Docker (for containerized builds)
- protobuf compiler (`protoc`)

### Unit Tests

Run all unit tests in the proxy-alb directory:

```bash
cd /home/penguin/code/MarchProxy/proxy-alb
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run tests with coverage:

```bash
go test -cover ./...
```

Generate coverage report:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Code Quality

#### Linting

Run golangci-lint:

```bash
golangci-lint run ./...
```

Format code with gofmt:

```bash
gofmt -w .
```

Go vet for static analysis:

```bash
go vet ./...
```

### Integration Tests

Integration tests verify ALB communication with Envoy and xDS server.

```bash
# Set up test environment
export XDS_SERVER="localhost:18000"
export ENVOY_BINARY="/usr/local/bin/envoy"
export ENVOY_CONFIG_PATH="./envoy/envoy.yaml"

# Run integration tests (if present)
go test -tags=integration ./...
```

### Building in Docker

Build the ALB container image:

```bash
docker build -f Dockerfile -t marchproxy-alb:latest .
```

Build with specific version:

```bash
docker build -f Dockerfile \
  --build-arg VERSION=v1.2.3 \
  --build-arg GIT_COMMIT=abc123def \
  -t marchproxy-alb:v1.2.3 .
```

### Running Tests in Container

Run tests inside Docker:

```bash
docker run --rm \
  -v $(pwd):/workspace \
  -w /workspace \
  golang:1.24-trixie \
  sh -c "go mod download && go test ./..."
```

### Manual Testing

#### Test gRPC API

Using grpcurl to test the gRPC service:

```bash
# Start ALB in one terminal
export GRPC_PORT=50051
export HEALTH_PORT=8080
export METRICS_PORT=9090
./alb-supervisor

# In another terminal, test endpoints:
grpcurl -plaintext localhost:50051 list

grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus

grpcurl -plaintext \
  -d '{"route_name":"test","config":{"requests_per_second":100,"burst_size":200,"enabled":true}}' \
  localhost:50051 marchproxy.ModuleService/ApplyRateLimit

grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetMetrics
```

#### Test Health Endpoints

```bash
# Liveness check
curl -i http://localhost:8080/healthz

# Readiness check
curl -i http://localhost:8080/ready

# Metrics
curl -s http://localhost:9090/metrics
```

#### Test Configuration Reload

```bash
grpcurl -plaintext \
  -d '{"force":false}' \
  localhost:50051 marchproxy.ModuleService/Reload
```

### CI/CD Pipeline

The project uses GitHub Actions for automated testing. See `.github/workflows/proxy-ci.yml` for the full test pipeline including:

- Linting (golangci-lint)
- Unit tests with coverage (80%+ required)
- Docker build verification
- Security scanning

## Troubleshooting

### Tests fail to import proto packages

Ensure proto files are generated:

```bash
cd ../proto/marchproxy
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       module_service.proto
```

### Envoy fails to start during tests

Check that Envoy binary is available at the configured path:

```bash
which envoy
export ENVOY_BINARY=$(which envoy)
```

Or use the Docker container path:

```bash
export ENVOY_BINARY=/usr/local/bin/envoy
```

### gRPC connection refused

Verify the ALB is running on the expected port:

```bash
netstat -tlnp | grep 50051
# or with ss:
ss -tlnp | grep 50051
```

### Coverage below threshold

View uncovered lines:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

Open `coverage.html` in browser to identify missing test cases.
