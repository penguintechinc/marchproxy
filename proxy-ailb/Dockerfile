FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Copy module files first for layer caching.
COPY go.mod go.sum* ./
COPY ../proto/go.mod ../proto/go.mod

# Download dependencies.
RUN go mod download || true

# Copy source.
COPY . .
COPY ../proto ../proto

# Build with optional XDP tags.
ARG XDP_BUILD_TAGS=noxdp
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags "${XDP_BUILD_TAGS}" \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /ailb ./cmd/ailb

# Build health check binary.
RUN CGO_ENABLED=0 go build -o /healthcheck -ldflags="-s -w" \
    -trimpath ./cmd/ailb && \
    echo '#!/bin/sh' > /healthcheck.sh && \
    echo 'exec wget -q --spider http://localhost:8080/healthz || exit 1' >> /healthcheck.sh

# Runtime image.
FROM debian:bookworm-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates wget && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -r ailb && useradd -r -g ailb -s /sbin/nologin ailb

COPY --from=builder /ailb /usr/local/bin/ailb

USER ailb

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
    CMD wget -q --spider http://localhost:8080/healthz || exit 1

EXPOSE 8080 50051

CMD ["/usr/local/bin/ailb"]
