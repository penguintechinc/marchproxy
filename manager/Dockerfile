# MarchProxy Manager Dockerfile
# Python/Quart async management server

# Build stage
FROM python:3.12-slim AS builder

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    libpq-dev \
    libxml2-dev \
    libxslt1-dev \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /build

# Copy requirements first for better caching
COPY requirements.txt requirements-core.txt requirements-dev.txt ./

# Install Python dependencies
RUN pip install --no-cache-dir --user -r requirements.txt

# Copy manager application
COPY . .

# Production stage
FROM python:3.12-slim AS production

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    libpq5 \
    libxml2 \
    libxslt1.1 \
    curl \
    dumb-init \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -r marchproxy && useradd -r -g marchproxy marchproxy

# Set working directory
WORKDIR /app

# Copy Python packages from builder
COPY --from=builder /root/.local /home/marchproxy/.local

# Copy application code
COPY --chown=marchproxy:marchproxy . .

# Create necessary directories
RUN mkdir -p /app/logs /app/certs && \
    chown -R marchproxy:marchproxy /app/logs /app/certs

# Make sure scripts are executable
RUN chmod +x scripts/*.sh 2>/dev/null || true

# Set Python path to include user packages
ENV PATH="/home/marchproxy/.local/bin:$PATH"
ENV PYTHONPATH="/app"

# Environment variables
ENV PYTHONUNBUFFERED=1

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8000/healthz || exit 1

# Expose port
EXPOSE 8000

# Switch to non-root user
USER marchproxy

# Start command - Quart with Hypercorn ASGI server
CMD ["dumb-init", "hypercorn", "quart_app:create_app()", "--bind", "0.0.0.0:8000", "--workers", "4"]

# Development stage
FROM production AS development

# Switch back to root for development tools
USER root

# Install development dependencies
RUN pip install --no-cache-dir \
    pytest \
    pytest-cov \
    pytest-asyncio \
    black \
    flake8 \
    mypy

# Install development tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    vim \
    less \
    && rm -rf /var/lib/apt/lists/*

# Switch back to marchproxy user
USER marchproxy

# Override command for development with auto-reload
CMD ["dumb-init", "hypercorn", "quart_app:create_app()", "--bind", "0.0.0.0:8000", "--reload"]

# Testing stage
FROM development AS testing

# Switch to root for test setup
USER root

# Copy test requirements
COPY requirements-dev.txt .
RUN pip install --no-cache-dir -r requirements-dev.txt

# Switch back to marchproxy user
USER marchproxy

# Run tests by default
CMD ["python", "-m", "pytest", "tests/", "-v", "--cov=.", "--cov-report=term-missing"]
