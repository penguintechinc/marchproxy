# MarchProxy Envoy L7 Proxy - Multi-stage Dockerfile
# Stage 1: XDP Build
# Stage 2: WASM Build
# Stage 3: Envoy Production

# ==================== Stage 1: XDP Build ====================
FROM debian:12-slim AS xdp-builder

LABEL stage=xdp-builder

# Install build dependencies for XDP
RUN apt-get update && apt-get install -y --no-install-recommends \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-amd64 \
    make \
    gcc \
    file \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build/xdp

# Copy XDP source and Makefile
COPY xdp/envoy_xdp.c xdp/Makefile ./

# Create symlinks for architecture-specific headers
RUN KERNEL_VER=$(ls /usr/src | grep linux-headers | grep -v common | head -n1 | sed 's/linux-headers-//') && \
    ln -sf /usr/src/linux-headers-${KERNEL_VER} /lib/modules/${KERNEL_VER}/build 2>/dev/null || true

# Build XDP program
RUN make

# Verify build
RUN ls -lh envoy_xdp.o && file envoy_xdp.o

# ==================== Stage 2: WASM Build ====================
FROM rust:1.83-slim AS wasm-builder

LABEL stage=wasm-builder

# Install build dependencies for WASM (ring crate needs clang)
RUN apt-get update && apt-get install -y --no-install-recommends \
    clang \
    llvm \
    && rm -rf /var/lib/apt/lists/*

# Install wasm32 target
RUN rustup target add wasm32-unknown-unknown

WORKDIR /build/filters

# Copy filter source code
COPY filters/auth_filter ./auth_filter
COPY filters/license_filter ./license_filter
COPY filters/metrics_filter ./metrics_filter

# Build auth filter
WORKDIR /build/filters/auth_filter
RUN cargo build --target wasm32-unknown-unknown --release

# Build license filter
WORKDIR /build/filters/license_filter
RUN cargo build --target wasm32-unknown-unknown --release

# Build metrics filter
WORKDIR /build/filters/metrics_filter
RUN cargo build --target wasm32-unknown-unknown --release

# Verify WASM builds
RUN ls -lh \
    /build/filters/auth_filter/target/wasm32-unknown-unknown/release/*.wasm \
    /build/filters/license_filter/target/wasm32-unknown-unknown/release/*.wasm \
    /build/filters/metrics_filter/target/wasm32-unknown-unknown/release/*.wasm

# ==================== Stage 3: Envoy Production ====================
FROM envoyproxy/envoy:v1.28-latest

LABEL maintainer="MarchProxy Contributors"
LABEL description="MarchProxy L7 Proxy - Envoy with XDP and WASM filters"
LABEL version="1.0.0"

# Install runtime dependencies
USER root
RUN apt-get update && apt-get install -y --no-install-recommends \
    iproute2 \
    iptables \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Create directories
RUN mkdir -p /etc/envoy /var/lib/envoy/xdp /var/lib/envoy/wasm /var/log/envoy

# Copy XDP program from builder
COPY --from=xdp-builder /build/xdp/envoy_xdp.o /var/lib/envoy/xdp/

# Copy WASM filters from builder
COPY --from=wasm-builder \
    /build/filters/auth_filter/target/wasm32-unknown-unknown/release/marchproxy_auth_filter.wasm \
    /var/lib/envoy/wasm/auth_filter.wasm

COPY --from=wasm-builder \
    /build/filters/license_filter/target/wasm32-unknown-unknown/release/marchproxy_license_filter.wasm \
    /var/lib/envoy/wasm/license_filter.wasm

COPY --from=wasm-builder \
    /build/filters/metrics_filter/target/wasm32-unknown-unknown/release/marchproxy_metrics_filter.wasm \
    /var/lib/envoy/wasm/metrics_filter.wasm

# Copy Envoy bootstrap configuration
COPY envoy/bootstrap.yaml /etc/envoy/envoy.yaml

# Copy startup script
COPY scripts/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh && \
    chown envoy:envoy /usr/local/bin/entrypoint.sh

# Set permissions
RUN chown -R envoy:envoy /var/lib/envoy /var/log/envoy /etc/envoy

# Expose ports
# 10000: HTTP/HTTPS listener
# 9901: Envoy admin interface
EXPOSE 10000 9901

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9901/ready || exit 1

# Environment variables
ENV ENVOY_UID=101
ENV LOGLEVEL=info

# Run as root initially (entrypoint may need privileges for XDP)
# The entrypoint script will drop to envoy user after setup
USER root

# Run Envoy
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["-c", "/etc/envoy/envoy.yaml", "--service-cluster", "marchproxy-l7", "--service-node", "marchproxy-node"]
