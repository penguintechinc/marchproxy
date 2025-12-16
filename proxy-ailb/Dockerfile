# AILB Container Dockerfile
# Multi-stage build for AI Load Balancer

FROM python:3.11-slim as builder

# Set working directory
WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    make \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir --user -r requirements.txt

# Final stage
FROM python:3.11-slim

# Set working directory
WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy Python dependencies from builder
COPY --from=builder /root/.local /root/.local

# Add local bin to PATH
ENV PATH=/root/.local/bin:$PATH

# Copy application code
COPY . .

# Copy proto files (if generated)
COPY ../proto /app/proto

# Generate gRPC code from proto files
RUN python -m grpc_tools.protoc \
    -I/app/proto \
    --python_out=/app/proto \
    --grpc_python_out=/app/proto \
    /app/proto/marchproxy/module_service.proto \
    || echo "Proto generation failed, will skip gRPC server"

# Create directories for data persistence
RUN mkdir -p /app/ailb_memory /app/ailb_rag

# Expose ports
EXPOSE 8080 50051

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Set environment variables
ENV PYTHONUNBUFFERED=1
ENV HTTP_PORT=8080
ENV GRPC_PORT=50051
ENV ROUTING_STRATEGY=load_balanced
ENV ENABLE_MEMORY=true
ENV ENABLE_RAG=false
ENV MEMORY_BACKEND=chromadb
ENV RAG_BACKEND=chromadb

# Run the application
CMD ["python", "main.py"]
