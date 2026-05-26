# AILB Container Dockerfile — Multi-stage build for AI Load Balancer

FROM python:3.13-slim-bookworm@sha256:e4fa1f978c539608a10cdf74700ac32a3f719dfc6e8b6b6001da82deb36302a2 AS builder

WORKDIR /build

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    make \
    && rm -rf /var/lib/apt/lists/*

COPY proxy-ailb/requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

FROM python:3.13-slim-bookworm@sha256:e4fa1f978c539608a10cdf74700ac32a3f719dfc6e8b6b6001da82deb36302a2 AS production

RUN groupadd -r ailb && useradd -r -g ailb -u 1000 ailb

WORKDIR /app

COPY --from=builder /root/.local /home/ailb/.local
ENV PATH=/home/ailb/.local/bin:$PATH

COPY proxy-ailb/ .
COPY proto/ /app/proto/

RUN python -m grpc_tools.protoc \
    -I/app/proto \
    --python_out=/app/proto \
    --grpc_python_out=/app/proto \
    /app/proto/marchproxy/module_service.proto \
    || echo "Proto generation skipped"

RUN chown -R ailb:ailb /app /home/ailb

USER ailb

EXPOSE 7003 7004 50053

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python3 -c "import urllib.request; urllib.request.urlopen('http://localhost:7004/healthz')" || exit 1

ENV PYTHONUNBUFFERED=1 \
    HTTP_PORT=7003 \
    ADMIN_PORT=7004 \
    GRPC_PORT=50053 \
    ROUTING_STRATEGY=load_balanced \
    ENABLE_MEMORY=true \
    ENABLE_RAG=false

CMD ["python3", "main.py"]
