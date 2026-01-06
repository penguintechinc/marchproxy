# MarchProxy Deployment Guide

This guide covers deployment options for MarchProxy's unified NLB architecture across different environments.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [GPU-Accelerated Video Transcoding](#gpu-accelerated-video-transcoding)
- [Production Considerations](#production-considerations)

---

## Architecture Overview

MarchProxy uses a unified Network Load Balancer (NLB) architecture with modular components:

```
┌────────────────────────────────────────────┐
│        NLB (proxy-nlb)                     │
│    Single Entry Point (L3/L4)             │
│  - Protocol inspection                     │
│  - Traffic routing via gRPC                │
│  - Rate limiting                           │
│  - Auto-scaling orchestration              │
└──────────────┬─────────────────────────────┘
               │ gRPC Communication
               │
    ┌──────────┼──────────┬──────────┬────────────┐
    │          │          │          │            │
    ▼          ▼          ▼          ▼            ▼
┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐  ┌──────────┐
│  ALB   │ │  DBLB  │ │  AILB  │ │  RTMP  │  │  Direct  │
│ (L7)   │ │ (DB)   │ │ (AI)   │ │ (Video)│  │Passthru  │
└────────┘ └────────┘ └────────┘ └────────┘  └──────────┘
```

### Core Components

- **postgres**: PostgreSQL database
- **redis**: Redis cache
- **api-server**: Management API and xDS control plane (FastAPI)
- **webui**: Admin dashboard (React + Vite)
- **proxy-nlb**: Network Load Balancer (Go + eBPF)
- **proxy-alb**: Application Load Balancer (Envoy L7)
- **proxy-dblb**: Database Load Balancer (Go) - *Optional*
- **proxy-ailb**: AI/LLM Load Balancer (Python) - *Optional*
- **proxy-rtmp**: Video Transcoding (Go + FFmpeg) - *Optional*

---

## Docker Compose Deployment

### Basic Stack (Core + NLB + ALB)

```bash
# Start core services (postgres, redis, api-server, webui, nlb, alb)
docker-compose up -d

# View logs
docker-compose logs -f

# Check service health
docker-compose ps
```

### Full Stack with All Modules

```bash
# Start all services including optional modules
docker-compose --profile full up -d

# Or start specific module profiles
docker-compose --profile dblb up -d  # Database proxy
docker-compose --profile ailb up -d  # AI/LLM proxy
docker-compose --profile rtmp up -d  # Video transcoding
```

### Development Mode

Development mode automatically applies when `docker-compose.override.yml` is present:

```bash
# Start in development mode (auto-reload, debug logs, pprof)
docker-compose up -d

# Features enabled:
# - Source code volume mounts
# - Hot reload for Python/Node.js
# - Debug log levels
# - Go pprof profiling endpoints
# - Python debugpy support
```

### Environment Configuration

Create a `.env` file in the project root:

```bash
# Database
POSTGRES_PASSWORD=your-secure-password
REDIS_PASSWORD=your-redis-password

# API Server
SECRET_KEY=your-secret-key-min-32-chars
CLUSTER_API_KEY=your-cluster-api-key

# License (Enterprise)
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD

# AI Provider Keys (for AILB)
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Network Configuration
NLB_PORT=7000
LOG_LEVEL=info
```

### Docker Compose Files Reference

| File | Purpose | Usage |
|------|---------|-------|
| `docker-compose.yml` | Main production stack | Default |
| `docker-compose.override.yml` | Development overrides | Auto-applied in dev |
| `docker-compose.gpu-nvidia.yml` | NVIDIA GPU support | `-f docker-compose.yml -f docker-compose.gpu-nvidia.yml` |
| `docker-compose.gpu-amd.yml` | AMD GPU support | `-f docker-compose.yml -f docker-compose.gpu-amd.yml` |

### Port Mapping

| Service | Port | Description |
|---------|------|-------------|
| webui | 3000 | Admin Dashboard |
| api-server | 8000 | REST API |
| api-server | 18000 | xDS gRPC |
| proxy-nlb | 7000 | Main entry point |
| proxy-nlb | 7001 | Admin/Metrics |
| proxy-alb | 80 | HTTP |
| proxy-alb | 443 | HTTPS |
| proxy-alb | 9901 | Envoy admin |
| proxy-dblb | 3306 | MySQL proxy |
| proxy-dblb | 5433 | PostgreSQL proxy |
| proxy-dblb | 27017 | MongoDB proxy |
| proxy-ailb | 7003 | AI/LLM HTTP API |
| proxy-rtmp | 1935 | RTMP ingest |
| proxy-rtmp | 8081 | HLS/DASH output |
| grafana | 3001 | Metrics dashboard |
| prometheus | 9090 | Metrics storage |
| jaeger | 16686 | Tracing UI |

---

## Kubernetes Deployment

### Prerequisites

```bash
# Verify cluster
kubectl cluster-info

# Verify metrics server (required for HPA)
kubectl top nodes
```

### Quick Deploy

```bash
# 1. Create secrets
cp k8s/unified/base/secrets.yaml.example k8s/unified/base/secrets.yaml
vim k8s/unified/base/secrets.yaml  # Edit with your values

# 2. Deploy base infrastructure
kubectl apply -f k8s/unified/base/

# 3. Deploy core modules (NLB + ALB)
kubectl apply -f k8s/unified/nlb/
kubectl apply -f k8s/unified/alb/
kubectl apply -f k8s/unified/hpa/nlb-hpa.yaml
kubectl apply -f k8s/unified/hpa/alb-hpa.yaml

# 4. Deploy optional modules
kubectl apply -f k8s/unified/dblb/  # Database proxy
kubectl apply -f k8s/unified/ailb/  # AI proxy
kubectl apply -f k8s/unified/rtmp/  # Video transcoding

# Deploy all at once
kubectl apply -R -f k8s/unified/
```

### Verify Deployment

```bash
# Check pods
kubectl get pods -n marchproxy

# Check services
kubectl get svc -n marchproxy

# Check HPA
kubectl get hpa -n marchproxy

# Get NLB external IP
kubectl get svc proxy-nlb -n marchproxy -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

### Access Services

```bash
# Port forward to API server
kubectl port-forward -n marchproxy svc/api-server 8000:8000

# Port forward to Web UI
kubectl port-forward -n marchproxy svc/webui 3000:3000

# Port forward to NLB admin
kubectl port-forward -n marchproxy svc/proxy-nlb 7001:7001
```

### Scaling

```bash
# Manual scaling
kubectl scale deployment proxy-nlb --replicas=5 -n marchproxy

# Update HPA
kubectl edit hpa proxy-nlb-hpa -n marchproxy

# View current scaling metrics
kubectl get hpa -n marchproxy --watch
```

---

## GPU-Accelerated Video Transcoding

### NVIDIA GPU (NVENC)

**Requirements:**
- NVIDIA GPU with NVENC support (GTX 1050+, RTX series, Tesla series)
- NVIDIA Container Toolkit installed on host
- Docker >= 19.03

**Deploy with NVIDIA GPU:**

```bash
# Using Docker Compose
docker-compose -f docker-compose.yml -f docker-compose.gpu-nvidia.yml up -d

# Verify GPU access
docker exec marchproxy-proxy-rtmp-nvidia nvidia-smi

# Check encoding
docker logs marchproxy-proxy-rtmp-nvidia | grep nvenc
```

**Features:**
- Hardware H.264/H.265 encoding
- 10x faster than CPU encoding
- Lower power consumption
- Presets: p1 (fastest) to p7 (best quality)
- Tuning: hq, ll, ull

### AMD GPU (VCE/AMF)

**Requirements:**
- AMD GPU with VCE/AMF support (RX 400+, Vega, RDNA series)
- ROCm drivers installed
- /dev/kfd and /dev/dri accessible

**Deploy with AMD GPU:**

```bash
# Using Docker Compose
docker-compose -f docker-compose.yml -f docker-compose.gpu-amd.yml up -d

# Verify GPU access
docker exec marchproxy-proxy-rtmp-amd ls -l /dev/dri/

# Check encoding
docker logs marchproxy-proxy-rtmp-amd | grep amf
```

**Features:**
- Hardware H.264/H.265 encoding
- VAAPI acceleration
- Quality modes: speed, balanced, quality
- Usage modes: transcoding, lowlatency, ultralowlatency

### Performance Comparison

| Method | Encoding Speed | Quality | Power Usage |
|--------|----------------|---------|-------------|
| CPU (x264) | 1x (baseline) | Excellent | High |
| NVIDIA NVENC | 10-15x | Very Good | Low |
| AMD VCE/AMF | 8-12x | Very Good | Medium |

---

## Production Considerations

### High Availability

1. **Docker Swarm/Compose**
   ```bash
   # Deploy with replicas
   docker stack deploy -c docker-compose.yml marchproxy
   ```

2. **Kubernetes**
   - HPA enabled for auto-scaling
   - Pod anti-affinity for node distribution
   - Multiple replicas per service
   - Pod Disruption Budgets

### Resource Limits

**Docker Compose:**
```yaml
services:
  proxy-nlb:
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 1G
```

**Kubernetes:**
- Already configured in deployment manifests
- See `k8s/unified/README.md` for details

### Security

1. **Network Isolation**
   - Internal network for gRPC communication
   - External network for public access
   - Firewall rules limiting exposed ports

2. **Secrets Management**
   - Never commit secrets to git
   - Use Docker secrets or Kubernetes secrets
   - Rotate credentials regularly

3. **TLS/mTLS**
   - Enable TLS for all external endpoints
   - Configure mTLS for inter-service communication
   - Store certificates securely

### Monitoring

1. **Prometheus + Grafana**
   - Included in Docker Compose
   - Metrics from all services
   - Pre-built dashboards

2. **Jaeger Tracing**
   - Distributed tracing enabled
   - Request flow visualization
   - Performance bottleneck identification

3. **Logging**
   - Centralized logging with syslog
   - ELK stack integration (optional)
   - Structured JSON logs

### Backup & Recovery

1. **Database Backups**
   ```bash
   # Postgres backup
   docker exec marchproxy-postgres pg_dump -U marchproxy marchproxy > backup.sql

   # Restore
   cat backup.sql | docker exec -i marchproxy-postgres psql -U marchproxy marchproxy
   ```

2. **Volume Backups**
   ```bash
   # Backup named volumes
   docker run --rm -v postgres_data:/data -v $(pwd):/backup alpine tar czf /backup/postgres_data.tar.gz /data
   ```

3. **Configuration Backups**
   - Export ConfigMaps/Secrets from Kubernetes
   - Backup `.env` files
   - Version control infrastructure as code

---

## Troubleshooting

### Docker Compose

```bash
# View logs
docker-compose logs -f <service-name>

# Restart service
docker-compose restart <service-name>

# Rebuild and restart
docker-compose up -d --build <service-name>

# Check resource usage
docker stats

# Clean up
docker-compose down -v  # Warning: removes volumes!
```

### Kubernetes

```bash
# Describe pod
kubectl describe pod <pod-name> -n marchproxy

# View events
kubectl get events -n marchproxy --sort-by='.lastTimestamp'

# Check logs
kubectl logs -f <pod-name> -n marchproxy

# Execute commands in pod
kubectl exec -it <pod-name> -n marchproxy -- /bin/sh
```

### Common Issues

1. **Port conflicts**: Change port mappings in `.env` or docker-compose.yml
2. **Out of memory**: Increase Docker resource limits
3. **License errors**: Verify LICENSE_KEY in secrets
4. **GPU not detected**: Check NVIDIA/AMD drivers and container toolkit

---

## Support

- Documentation: See `docs/` folder
- Kubernetes Guide: See `k8s/unified/README.md`
- GitHub Issues: https://github.com/penguintech/marchproxy/issues
- Enterprise Support: support@penguintech.io
