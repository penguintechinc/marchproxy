# MarchProxy API Server Dockerfile
# Multi-stage build for FastAPI + Go xDS server

# Stage 1: Build Go xDS server
FROM golang:1.21-alpine AS go-builder

WORKDIR /build

# Copy Go module files
COPY xds/go.mod xds/go.sum* ./

# Download dependencies
RUN go mod download

# Copy Go source files
COPY xds/*.go ./

# Build the xDS server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o xds-server .

# Stage 2: Build Python dependencies
FROM python:3.11-slim as python-builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

# Stage 3: Production stage
FROM python:3.11-slim

LABEL maintainer="MarchProxy Contributors"
LABEL description="MarchProxy API Server - FastAPI Backend with xDS Control Plane"

WORKDIR /app

# Copy Python dependencies from builder
COPY --from=python-builder /root/.local /root/.local

# Copy Go xDS server from builder
COPY --from=go-builder /build/xds-server /usr/local/bin/xds-server

# Make sure scripts in .local are usable
ENV PATH=/root/.local/bin:$PATH

# Copy application code
COPY app/ ./app/

# Create non-root user
RUN useradd -m -u 1000 marchproxy && \
    chown -R marchproxy:marchproxy /app && \
    chmod +x /usr/local/bin/xds-server

USER marchproxy

# Expose ports
# 8000: FastAPI REST API
# 18000: xDS gRPC server
# 19000: xDS metrics/health
EXPOSE 8000 18000 19000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://localhost:8000/healthz')"

# Start script to run both FastAPI and xDS server
COPY --chown=marchproxy:marchproxy start.sh /app/start.sh
RUN chmod +x /app/start.sh

CMD ["/app/start.sh"]
