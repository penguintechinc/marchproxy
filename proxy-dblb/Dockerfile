# Multi-stage build for MarchProxy DBLB
FROM golang:1.24.7-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=${VERSION:-1.0.0} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.gitCommit=${GIT_COMMIT:-unknown}" \
    -o proxy-dblb \
    ./cmd/main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 dblb && \
    adduser -D -u 1000 -G dblb dblb

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/proxy-dblb /app/proxy-dblb

# Copy configuration example (optional)
COPY --from=builder /build/*.yaml /app/ 2>/dev/null || true

# Set ownership
RUN chown -R dblb:dblb /app

# Switch to non-root user
USER dblb

# Expose ports
# MySQL
EXPOSE 3306
# PostgreSQL
EXPOSE 5432
# MongoDB
EXPOSE 27017
# Redis
EXPOSE 6379
# MSSQL
EXPOSE 1433
# Admin/Metrics
EXPOSE 7002
# gRPC
EXPOSE 50052

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:7002/healthz || exit 1

# Set entrypoint
ENTRYPOINT ["/app/proxy-dblb"]

# Default command (can be overridden)
CMD ["--config", "/app/config.yaml"]
