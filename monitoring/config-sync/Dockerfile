FROM python:3.13-slim-bookworm@sha256:01f42367a0a94ad4bc17111776fd66e3500c1d87c15bbd6055b7371d39c124fb

WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy application
COPY sync.py .

# Create monitoring config directory and non-root user
RUN mkdir -p /etc/monitoring && \
    useradd -m -u 1000 appuser && \
    chown -R appuser:appuser /app /etc/monitoring

# Run as non-root user
USER appuser

# Run the sync service
CMD ["python3", "sync.py"]