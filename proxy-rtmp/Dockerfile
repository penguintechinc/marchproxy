# MarchProxy RTMP Container - CPU Version
# Supports x264, x265, and AV1 (libaom, SVT-AV1) software encoding
# Includes SRT and WebRTC support

FROM golang:1.24-bookworm AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rtmp-proxy ./cmd/rtmp

# CPU stage (default)
FROM debian:bookworm-slim AS cpu

# Install FFmpeg with AV1 support, SRT library, and runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    libsrt1.5-openssl \
    libsvtav1enc1 \
    libaom3 \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1000 marchproxy && \
    useradd -u 1000 -g marchproxy -m -s /sbin/nologin marchproxy

# Create directories
RUN mkdir -p /var/lib/marchproxy/streams && \
    chown -R marchproxy:marchproxy /var/lib/marchproxy

# Copy binary from builder
COPY --from=builder /build/rtmp-proxy /usr/local/bin/rtmp-proxy

# Switch to non-root user
USER marchproxy

# Expose ports
# RTMP: 1935, gRPC: 50053, SRT: 8890, WHIP: 8080, WHEP: 8081
EXPOSE 1935/tcp
EXPOSE 50053/tcp
EXPOSE 8890/udp
EXPOSE 8080/tcp
EXPOSE 8081/tcp

# Set environment variables
ENV RTMP_HOST=0.0.0.0 \
    RTMP_PORT=1935 \
    RTMP_GRPC_PORT=50053 \
    RTMP_LOG_LEVEL=info \
    RTMP_ENCODER=auto \
    RTMP_PRESET=medium \
    RTMP_PREFER_AV1=false \
    RTMP_OUTPUT_DIR=/var/lib/marchproxy/streams \
    RTMP_ENABLE_HLS=true \
    RTMP_ENABLE_DASH=true \
    RTMP_SEGMENT_DURATION=6 \
    RTMP_MAX_BITRATE=10 \
    RTMP_MAX_STREAMS=100 \
    RTMP_MAX_RESOLUTION=1080 \
    RTMP_MAX_RESOLUTION_CPU=1440 \
    RTMP_MAX_RESOLUTION_GPU=4320 \
    RTMP_ENABLE_SRT=false \
    RTMP_SRT_PORT=8890 \
    RTMP_SRT_LATENCY=120 \
    RTMP_ENABLE_WEBRTC=false \
    RTMP_WHIP_PORT=8080 \
    RTMP_WHEP_PORT=8081

# Health check using Go binary (no nc dependency)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/usr/local/bin/rtmp-proxy", "--healthcheck"]

# Run the application
ENTRYPOINT ["/usr/local/bin/rtmp-proxy"]
CMD ["--config", "/etc/marchproxy/rtmp.yaml"]
