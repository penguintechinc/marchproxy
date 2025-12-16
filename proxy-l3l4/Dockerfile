# Multi-stage Dockerfile for MarchProxy L3/L4 Proxy with Zero-Trust
# Supports: production, development, testing, debug targets

# Build stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-generic \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the proxy
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o proxy-l3l4 ./cmd/proxy/main.go

# Production stage
FROM debian:12-slim AS production

# Install runtime dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    libbpf1 \
    iproute2 \
    iputils-ping \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/proxy-l3l4 .

# Copy OPA policies
COPY policies/ ./policies/

# Create directories for audit logs and certificates
RUN mkdir -p /var/log/marchproxy/audit \
    && mkdir -p /etc/marchproxy/certs \
    && chmod 755 /var/log/marchproxy/audit

# Expose ports
EXPOSE 8081 8082

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8082/healthz || exit 1

# Run as non-root user
RUN useradd -r -u 1000 marchproxy
USER marchproxy

ENTRYPOINT ["./proxy-l3l4"]

# Development stage
FROM builder AS development

WORKDIR /app

# Install development tools
RUN go install github.com/cosmtrek/air@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Copy source
COPY . .

EXPOSE 8081 8082

CMD ["air", "-c", ".air.toml"]

# Testing stage
FROM builder AS testing

WORKDIR /app

# Copy source and tests
COPY . .

# Install test dependencies
RUN go install gotest.tools/gotestsum@latest

# Run tests with coverage
RUN go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

CMD ["gotestsum", "--junitfile", "test-results.xml", "--", "-v", "-race", "./..."]

# Debug stage
FROM production AS debug

# Install debugging tools
USER root
RUN apt-get update && apt-get install -y \
    gdb \
    strace \
    tcpdump \
    net-tools \
    vim \
    && rm -rf /var/lib/apt/lists/*

# Build with debug symbols
COPY --from=builder /build/proxy-l3l4 .

USER marchproxy

ENTRYPOINT ["./proxy-l3l4"]
