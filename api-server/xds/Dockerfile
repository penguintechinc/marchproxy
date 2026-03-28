# MarchProxy xDS Server Dockerfile
FROM golang:1.24-bookworm@sha256:01f42367a0a94ad4bc17111776fd66e3500c1d87c15bbd6055b7371d39c124fb AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY *.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xds-server .

# Production stage
FROM debian:12-slim@sha256:66f47f9e94302e1e1ab2baf19c9a0107b75e4afb915b12a757575e33ce18cfb3

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/xds-server .

# Expose ports
EXPOSE 18000 19000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:19000/healthz || exit 1

# Run the server
ENTRYPOINT ["./xds-server"]
CMD ["-port=18000", "-http-port=19000", "-nodeID=marchproxy-node"]
