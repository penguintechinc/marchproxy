# Multi-stage build for MarchProxy Manager (Python/py4web)
# Multi-architecture support with proper platform detection
FROM --platform=$BUILDPLATFORM python:3.12-slim as base

# Build arguments for multi-arch support
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH

# Install system dependencies with architecture-aware packages
RUN apt-get update && apt-get install -y \
    curl \
    gcc \
    g++ \
    libpq-dev \
    postgresql-client \
    git \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Set working directory
WORKDIR /app

# Create non-root user
RUN useradd -m -u 1000 marchproxy && \
    chown -R marchproxy:marchproxy /app

# Copy requirements first for better caching
COPY manager/requirements.txt /app/requirements.txt

# Install Python dependencies
RUN pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir -r requirements.txt

# Development stage
FROM base as development

# Install development dependencies
RUN pip install --no-cache-dir \
    pytest \
    pytest-cov \
    black \
    flake8 \
    mypy \
    ipython \
    debugpy

# Copy application code
COPY manager/ /app/

# Copy entrypoint script
COPY docker/manager/entrypoint.sh /usr/local/bin/entrypoint.sh

# Create necessary directories
RUN mkdir -p /app/logs /app/certs /app/static /app/templates && \
    chown -R marchproxy:marchproxy /app && \
    chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER marchproxy

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8000/healthz || exit 1

# Default command
CMD ["/usr/local/bin/entrypoint.sh"]

# Production stage
FROM base as production

# Copy only necessary files
COPY manager/ /app/
COPY docker/manager/entrypoint.sh /usr/local/bin/entrypoint.sh

# Remove development files
RUN find /app -name "*.pyc" -delete && \
    find /app -name "__pycache__" -type d -exec rm -rf {} + || true

# Create directories and set permissions
RUN mkdir -p /app/logs /app/certs && \
    chown -R marchproxy:marchproxy /app && \
    chmod +x /usr/local/bin/entrypoint.sh

# Switch to non-root user
USER marchproxy

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8000/healthz || exit 1

# Production command
CMD ["/usr/local/bin/entrypoint.sh"]