# Multi-stage build for MarchProxy Proxy (Go/eBPF)
# Multi-architecture support with proper platform detection  
FROM --platform=$BUILDPLATFORM golang:1.22-bullseye as base

# Build arguments for multi-arch support
ARG BUILDPLATFORM
ARG TARGETPLATFORM 
ARG TARGETOS
ARG TARGETARCH

# Install system dependencies for eBPF and networking (architecture-aware)
RUN apt-get update && apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    build-essential \
    curl \
    iproute2 \
    iptables \
    ca-certificates \
    && apt-get install -y linux-headers-$(dpkg --print-architecture) || echo "Headers not available for this architecture" \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Set working directory
WORKDIR /app

# Development stage - simplified without air
FROM base as development

# Copy go.mod and go.sum first for better caching
COPY proxy/go.mod proxy/go.sum /app/

# Download dependencies
RUN go mod download

# Copy source code
COPY proxy/ /app/
COPY ebpf/ /app/ebpf/

# Build eBPF programs (skip for now due to header issues)
# RUN cd ebpf && \
#     clang -O2 -target bpf -c programs/*.c -o programs/filter.o

# Build development binary
RUN CGO_ENABLED=1 go build -race -gcflags="all=-N -l" -o marchproxy-dev ./cmd/proxy

# Create non-root user
RUN useradd -m -u 1000 marchproxy && \
    mkdir -p /app/logs && \
    chown -R marchproxy:marchproxy /app

# Switch to non-root user for development
USER marchproxy

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8081/healthz || exit 1

# Development command
CMD ["./marchproxy-dev"]

# Builder stage for production
FROM base as builder

# Copy source code
COPY proxy/ /app/
COPY ebpf/ /app/ebpf/

# Download dependencies
RUN go mod download

# Build eBPF programs (skip for now due to header issues)
# RUN cd ebpf && \
#     clang -O2 -target bpf -c programs/*.c -o programs/filter.o

# Build production binary with optimizations
# Cross-compile Go binary for target architecture
RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags="-w -s -extldflags '-static'" \
    -tags netgo \
    -installsuffix cgo \
    -o marchproxy ./cmd/proxy

# Production stage
FROM debian:bullseye-slim as production

# Install runtime dependencies only
RUN apt-get update && apt-get install -y \
    ca-certificates \
    curl \
    iproute2 \
    iptables \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Create non-root user
RUN useradd -m -u 1000 marchproxy

# Set working directory
WORKDIR /app

# Copy binary and eBPF programs from builder
COPY --from=builder /app/marchproxy /app/marchproxy
# COPY --from=builder /app/ebpf/programs/*.o /app/ebpf/

# Create logs directory
RUN mkdir -p /app/logs && \
    chown -R marchproxy:marchproxy /app

# Switch to non-root user
USER marchproxy

# Expose ports
EXPOSE 8080 8081

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8081/healthz || exit 1

# Production command
CMD ["./marchproxy"]