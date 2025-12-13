# MarchProxy RTMP Container - CPU Version
# Supports x264 and x265 software encoding

FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rtmp-proxy ./cmd/rtmp

# Final stage
FROM alpine:3.18

# Install FFmpeg and runtime dependencies
RUN apk add --no-cache \
    ffmpeg \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 marchproxy && \
    adduser -D -u 1000 -G marchproxy marchproxy

# Create directories
RUN mkdir -p /var/lib/marchproxy/streams && \
    chown -R marchproxy:marchproxy /var/lib/marchproxy

# Copy binary from builder
COPY --from=builder /build/rtmp-proxy /usr/local/bin/rtmp-proxy

# Switch to non-root user
USER marchproxy

# Expose ports
EXPOSE 1935/tcp
EXPOSE 50053/tcp

# Set environment variables
ENV RTMP_HOST=0.0.0.0 \
    RTMP_PORT=1935 \
    RTMP_GRPC_PORT=50053 \
    RTMP_LOG_LEVEL=info \
    RTMP_ENCODER=auto \
    RTMP_PRESET=medium \
    RTMP_OUTPUT_DIR=/var/lib/marchproxy/streams \
    RTMP_ENABLE_HLS=true \
    RTMP_ENABLE_DASH=true \
    RTMP_SEGMENT_DURATION=6 \
    RTMP_MAX_BITRATE=10 \
    RTMP_MAX_STREAMS=100 \
    RTMP_MAX_RESOLUTION=1080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["sh", "-c", "nc -z localhost 1935 && nc -z localhost 50053 || exit 1"]

# Run the application
ENTRYPOINT ["/usr/local/bin/rtmp-proxy"]
CMD ["--config", "/etc/marchproxy/rtmp.yaml"]
