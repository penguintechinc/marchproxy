# MarchProxy Egress Dockerfile
# High-performance Go egress proxy with eBPF/XDP acceleration, mTLS, and L7 (Envoy) support
#
# Features:
# - L4 TCP/UDP proxy with eBPF/XDP acceleration
# - L7 HTTP/1.1, HTTP/2 proxy via Envoy
# - HTTP/3 (QUIC) support (EXPERIMENTAL)
# - TLS interception (MITM and pre-configured modes)
# - Threat intelligence integration
# - External authorization for traffic control

# Envoy version to use
ARG ENVOY_VERSION=v1.28-latest

# Build stage
FROM golang:1.24-bookworm AS builder

# Install build dependencies for eBPF and networking
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    gcc \
    build-essential \
    linux-headers-generic \
    clang \
    llvm \
    libbpf-dev \
    libelf-dev \
    zlib1g-dev \
    pkg-config \
    openssl \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy go modules first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the egress proxy application with eBPF support
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo,osusergo \
    -o marchproxy-egress ./cmd/proxy

# Build health check binary if it exists
RUN if [ -d "./cmd/health" ]; then \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo,osusergo \
    -o marchproxy-health ./cmd/health; \
    fi

# Build metrics binary if it exists
RUN if [ -d "./cmd/metrics" ]; then \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo,osusergo \
    -o marchproxy-metrics ./cmd/metrics; \
    fi

# Build test client for testing if it exists
RUN if [ -d "./cmd/test-client" ]; then \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -tags netgo,osusergo \
    -o test-client ./cmd/test-client; \
    fi

# Production stage - using Envoy as base for L7 support
ARG ENVOY_VERSION=v1.28-latest
FROM envoyproxy/envoy:${ENVOY_VERSION} AS production

# Switch to root for package installation
USER root

# Install runtime dependencies for eBPF and networking
# Note: libbpf1 not available in Ubuntu Focal (Envoy base), skip for now
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    libelf1 \
    dumb-init \
    curl \
    iproute2 \
    iptables \
    ethtool \
    openssl \
    && rm -rf /var/lib/apt/lists/*

# Create marchproxy user (envoy user already exists in base image)
RUN groupadd -g 1000 marchproxy || true && \
    useradd -u 1000 -g marchproxy -s /bin/bash -m marchproxy || true

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/marchproxy-egress /app/
COPY --from=builder /build/test-client /app/

# Copy configuration files
COPY --chown=marchproxy:marchproxy configs/ /app/configs/

# Copy Envoy configuration
COPY --chown=marchproxy:marchproxy envoy/bootstrap.yaml /etc/envoy/envoy.yaml

# Create necessary directories
RUN mkdir -p /app/logs /app/certs /app/ebpf /etc/envoy && \
    chown -R marchproxy:marchproxy /app/logs /app/certs /app/ebpf && \
    chown -R marchproxy:marchproxy /etc/envoy && \
    chmod +x /app/marchproxy-egress /app/test-client

# Environment variables - L4 proxy
ENV PROXY_TYPE=egress
ENV PROXY_CONFIG_PATH=/app/configs
ENV PROXY_LOG_PATH=/app/logs
ENV PROXY_CERT_PATH=/app/certs
ENV MTLS_ENABLED=true
ENV MTLS_SERVER_CERT_PATH=/app/certs/egress-server.crt
ENV MTLS_SERVER_KEY_PATH=/app/certs/egress-server.key
ENV MTLS_CLIENT_CA_PATH=/app/certs/client-ca-bundle.crt

# Environment variables - L7 Envoy
ENV L7_ENABLED=true
ENV ENVOY_BINARY=/usr/local/bin/envoy
ENV ENVOY_CONFIG_PATH=/etc/envoy/envoy.yaml
ENV ENVOY_ADMIN_PORT=9901
ENV ENVOY_LISTEN_PORT=10000
ENV ENVOY_HTTPS_PORT=10443
ENV ENVOY_LOG_LEVEL=info

# HTTP/3 (QUIC) - EXPERIMENTAL - disabled by default
ENV ENVOY_HTTP3_ENABLED=false

# Threat intelligence
ENV THREAT_IP_BLOCKING_ENABLED=true
ENV THREAT_DOMAIN_BLOCKING_ENABLED=true
ENV THREAT_URL_MATCHING_ENABLED=true
ENV THREAT_DNS_CACHE_ENABLED=true

# External authorization server port
ENV EXTAUTH_PORT=9002

# Health check - checks both Go proxy and Envoy
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD /app/marchproxy-health && curl -sf http://localhost:9901/ready || exit 1

# Expose ports
# L4: 8080 (TCP proxy), 8081 (admin/metrics)
# L7: 10000 (HTTP), 10443 (HTTPS/QUIC), 9901 (Envoy admin)
# ext_authz: 9002
EXPOSE 8080 8081 10000 10443 9901 9002

# Switch to non-root user for security
USER marchproxy

# Start command - Go supervisor manages both L4 proxy and Envoy
CMD ["dumb-init", "/app/marchproxy-egress"]

# Development stage
FROM golang:1.21-bookworm AS development

# Install development dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    gcc \
    build-essential \
    linux-headers-generic \
    clang \
    llvm \
    libbpf-dev \
    libelf-dev \
    zlib1g-dev \
    vim \
    curl \
    htop \
    tcpdump \
    iproute2 \
    iptables \
    ethtool \
    openssl \
    && rm -rf /var/lib/apt/lists/*

# Install development tools
RUN go install github.com/cosmtrek/air@latest && \
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && \
    go install golang.org/x/tools/cmd/goimports@latest

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Download dependencies
RUN go mod download

# Development command with hot reload
CMD ["air", "-c", ".air.toml"]

# Testing stage
FROM development AS testing

# Install test dependencies
RUN go install github.com/onsi/ginkgo/v2/ginkgo@latest && \
    go install github.com/onsi/gomega/...@latest

# Run tests by default
CMD ["go", "test", "-v", "-race", "-coverprofile=coverage.out", "./..."]

# Cross-compilation stage for multi-platform builds
FROM --platform=$BUILDPLATFORM golang:1.21-bullseye AS cross-builder

ARG TARGETOS
ARG TARGETARCH

# Install build dependencies
RUN apt-get update && apt-get install -y \
    git \
    ca-certificates \
    tzdata \
    gcc \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build for target platform
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags='-w -s' \
    -tags netgo,osusergo \
    -o marchproxy-proxy ./cmd/proxy

# Distroless stage for minimal production image
FROM gcr.io/distroless/static:nonroot AS distroless

# Copy binary from cross-builder
COPY --from=cross-builder /build/marchproxy-proxy /

# Copy configuration
COPY --chown=nonroot:nonroot configs/ /configs/

# Expose ports
EXPOSE 80 443 8080 8081

# Use distroless nonroot user
USER nonroot

# Start command
ENTRYPOINT ["/marchproxy-proxy"]

# Debug stage with shell access
FROM debian:12-slim AS debug

# Install debugging tools
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    tcpdump \
    netcat-openbsd \
    strace \
    gdb \
    libbpf1 \
    libelf1 \
    dumb-init \
    bash \
    htop \
    iftop \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/marchproxy-egress /app/
COPY --from=builder /build/test-client /app/

# Copy configuration
COPY configs/ /app/configs/

# Copy Envoy configuration
COPY envoy/bootstrap.yaml /etc/envoy/envoy.yaml

# Make executable
RUN chmod +x /app/marchproxy-egress /app/test-client

# Debug entrypoint
CMD ["bash"]