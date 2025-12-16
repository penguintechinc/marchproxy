# MarchProxy Deployment Files Reference

This document provides a complete reference of all deployment configuration files created for the MarchProxy unified NLB architecture.

## File Tree

```
MarchProxy/
│
├── Docker Compose Configurations
│   ├── docker-compose.yml                      # Main production stack (578 lines)
│   ├── docker-compose.override.yml             # Development overrides (141 lines)
│   ├── docker-compose.gpu-nvidia.yml           # NVIDIA GPU support (119 lines)
│   └── docker-compose.gpu-amd.yml              # AMD GPU support (121 lines)
│
├── Kubernetes Configurations
│   └── k8s/unified/
│       │
│       ├── base/
│       │   ├── namespace.yaml                  # Namespace, quotas, limits (54 lines)
│       │   ├── configmap.yaml                  # Global & module configs (166 lines)
│       │   └── secrets.yaml.example            # Secret template (33 lines)
│       │
│       ├── nlb/
│       │   ├── deployment.yaml                 # NLB deployment (163 lines)
│       │   └── service.yaml                    # LoadBalancer service (43 lines)
│       │
│       ├── alb/
│       │   ├── deployment.yaml                 # ALB deployment (95 lines)
│       │   └── service.yaml                    # ClusterIP service (26 lines)
│       │
│       ├── dblb/
│       │   ├── deployment.yaml                 # DBLB deployment (67 lines)
│       │   └── service.yaml                    # ClusterIP service (22 lines)
│       │
│       ├── ailb/
│       │   ├── deployment.yaml                 # AILB deployment (79 lines)
│       │   └── service.yaml                    # ClusterIP service (17 lines)
│       │
│       ├── rtmp/
│       │   ├── deployment.yaml                 # RTMP deployment (70 lines)
│       │   └── service.yaml                    # ClusterIP service (18 lines)
│       │
│       ├── hpa/
│       │   ├── nlb-hpa.yaml                    # NLB auto-scaling (46 lines)
│       │   ├── alb-hpa.yaml                    # ALB auto-scaling (39 lines)
│       │   ├── dblb-hpa.yaml                   # DBLB auto-scaling (39 lines)
│       │   ├── ailb-hpa.yaml                   # AILB auto-scaling (39 lines)
│       │   └── rtmp-hpa.yaml                   # RTMP auto-scaling (37 lines)
│       │
│       └── README.md                           # K8s deployment guide (408 lines)
│
└── Documentation
    ├── DEPLOYMENT.md                           # Complete deployment guide (441 lines)
    └── PHASE8_DOCKER_K8S_SUMMARY.md           # Implementation summary (this file)

Total Files: 28
Total Lines: ~2,900+
```

## Quick Reference by Use Case

### I want to... deploy with Docker Compose

**Basic stack (core + NLB + ALB):**
```bash
docker-compose up -d
```

**Files used:**
- `docker-compose.yml` - Main configuration
- `docker-compose.override.yml` - Auto-applied in development

---

**Full stack with all modules:**
```bash
docker-compose --profile full up -d
```

**Files used:**
- `docker-compose.yml` - All services defined
- `docker-compose.override.yml` - Development mode

---

**GPU-accelerated video transcoding (NVIDIA):**
```bash
docker-compose -f docker-compose.yml -f docker-compose.gpu-nvidia.yml --profile gpu-nvidia up -d
```

**Files used:**
- `docker-compose.yml` - Base configuration
- `docker-compose.gpu-nvidia.yml` - NVIDIA GPU overrides

---

**GPU-accelerated video transcoding (AMD):**
```bash
docker-compose -f docker-compose.yml -f docker-compose.gpu-amd.yml --profile gpu-amd up -d
```

**Files used:**
- `docker-compose.yml` - Base configuration
- `docker-compose.gpu-amd.yml` - AMD GPU overrides

---

### I want to... deploy with Kubernetes

**Quick deploy everything:**
```bash
# 1. Create secrets
cp k8s/unified/base/secrets.yaml.example k8s/unified/base/secrets.yaml
vim k8s/unified/base/secrets.yaml

# 2. Deploy all
kubectl apply -R -f k8s/unified/
```

**Files used:** All Kubernetes files (22 total)

---

**Deploy only core services (NLB + ALB):**
```bash
# 1. Base infrastructure
kubectl apply -f k8s/unified/base/

# 2. Core proxies
kubectl apply -f k8s/unified/nlb/
kubectl apply -f k8s/unified/alb/

# 3. Auto-scaling
kubectl apply -f k8s/unified/hpa/nlb-hpa.yaml
kubectl apply -f k8s/unified/hpa/alb-hpa.yaml
```

**Files used:**
- `k8s/unified/base/*` (3 files)
- `k8s/unified/nlb/*` (2 files)
- `k8s/unified/alb/*` (2 files)
- `k8s/unified/hpa/nlb-hpa.yaml`
- `k8s/unified/hpa/alb-hpa.yaml`

---

**Add database proxy module:**
```bash
kubectl apply -f k8s/unified/dblb/
kubectl apply -f k8s/unified/hpa/dblb-hpa.yaml
```

**Files used:**
- `k8s/unified/dblb/*` (2 files)
- `k8s/unified/hpa/dblb-hpa.yaml`

---

**Add AI/LLM proxy module:**
```bash
kubectl apply -f k8s/unified/ailb/
kubectl apply -f k8s/unified/hpa/ailb-hpa.yaml
```

**Files used:**
- `k8s/unified/ailb/*` (2 files)
- `k8s/unified/hpa/ailb-hpa.yaml`

---

**Add video transcoding module:**
```bash
kubectl apply -f k8s/unified/rtmp/
kubectl apply -f k8s/unified/hpa/rtmp-hpa.yaml
```

**Files used:**
- `k8s/unified/rtmp/*` (2 files)
- `k8s/unified/hpa/rtmp-hpa.yaml`

---

## File Descriptions

### Docker Compose Files

#### docker-compose.yml
**Purpose:** Complete production stack with all services
**Services:** 9 core + 4 observability + 3 optional modules
**Networks:** Internal (gRPC) + External (public)
**Profiles:** `full`, `dblb`, `ailb`, `rtmp`
**Key Features:**
- Health checks for all services
- Resource limits and reservations
- Capability configurations (NET_ADMIN, SYS_RESOURCE)
- Volume management
- Environment variable configuration

#### docker-compose.override.yml
**Purpose:** Development mode overrides (auto-applies)
**Key Features:**
- Source code volume mounts for hot-reload
- Debug logging (LOG_LEVEL=debug)
- Go pprof profiling endpoints (ports 6060-6064)
- Python debugpy support (port 5678)
- Reduced resource requirements
- Development-specific command overrides

#### docker-compose.gpu-nvidia.yml
**Purpose:** NVIDIA GPU acceleration for RTMP
**Key Features:**
- NVENC hardware encoding (h264_nvenc, hevc_nvenc)
- CUDA acceleration (hwaccel=cuda)
- Multi-bitrate transcoding
- Quality presets (p1-p7)
- Tuning modes (hq, ll, ull)
- GPU resource allocation

#### docker-compose.gpu-amd.yml
**Purpose:** AMD GPU acceleration for RTMP
**Key Features:**
- AMF hardware encoding (h264_amf, hevc_amf)
- VAAPI acceleration
- Device mappings (/dev/kfd, /dev/dri)
- ROCm configuration
- Multi-bitrate transcoding
- Quality modes (speed, balanced, quality)

### Kubernetes Base Files

#### k8s/unified/base/namespace.yaml
**Purpose:** Namespace isolation and resource management
**Includes:**
- Namespace: `marchproxy`
- ResourceQuota (CPU, memory, storage, pods, services)
- LimitRange (container and pod limits/defaults)

#### k8s/unified/base/configmap.yaml
**Purpose:** Non-sensitive configuration data
**Includes:**
- Global config (database, Redis, API server, observability)
- Module-specific configs (NLB, ALB, DBLB, AILB, RTMP)
- gRPC endpoint mappings
- Feature flags and tuning parameters

#### k8s/unified/base/secrets.yaml.example
**Purpose:** Template for sensitive configuration
**Includes:**
- Cluster API key
- Database passwords (Postgres, Redis)
- License key (Enterprise)
- AI provider API keys (OpenAI, Anthropic)
- JWT secret
- TLS certificates (optional)
- Secret generation commands

### Kubernetes Module Deployments

#### NLB (proxy-nlb)
**Files:** deployment.yaml, service.yaml
**Replicas:** 2 (min) - 10 (max with HPA)
**Resources:** 1-4 CPU, 1-4Gi RAM
**Service Type:** LoadBalancer (external) + Headless (internal)
**Key Features:**
- Session affinity (ClientIP)
- Anti-affinity rules
- Security context (non-root, capabilities)
- AWS NLB annotations

#### ALB (proxy-alb)
**Files:** deployment.yaml, service.yaml
**Replicas:** 3 (min) - 20 (max with HPA)
**Resources:** 500m-2 CPU, 512Mi-2Gi RAM
**Service Type:** ClusterIP
**Key Features:**
- Envoy container with xDS
- Prometheus scrape annotations
- Health checks on Envoy admin port
- Multiple port mappings (HTTP, HTTPS, HTTP/2, admin, gRPC)

#### DBLB (proxy-dblb)
**Files:** deployment.yaml, service.yaml
**Replicas:** 2 (min) - 15 (max with HPA)
**Resources:** 1-4 CPU, 1-4Gi RAM
**Service Type:** ClusterIP
**Key Features:**
- Multi-protocol support (MySQL, Postgres, MongoDB, Redis, MSSQL)
- Connection pooling configuration
- SQL injection detection
- Per-protocol port mappings

#### AILB (proxy-ailb)
**Files:** deployment.yaml, service.yaml
**Replicas:** 2 (min) - 10 (max with HPA)
**Resources:** 500m-2 CPU, 1-4Gi RAM
**Service Type:** ClusterIP
**Key Features:**
- AI provider API key management
- Redis for conversation memory
- RAG configuration
- Rate limiting per provider

#### RTMP (proxy-rtmp)
**Files:** deployment.yaml, service.yaml
**Replicas:** 1 (min) - 5 (max with HPA)
**Resources:** 2-8 CPU, 2-8Gi RAM
**Service Type:** ClusterIP
**Key Features:**
- EmptyDir volume for streams (50Gi)
- FFmpeg configuration
- HLS/DASH output
- Higher resource limits for transcoding

### Kubernetes HPA Files

All HPA files include:
- CPU and memory utilization targets
- Custom metrics (connections, requests, streams)
- Intelligent scale-up/down policies
- Stabilization windows

**Scaling Policies Summary:**

| Module | Min | Max | CPU Target | Memory Target | Custom Metric |
|--------|-----|-----|------------|---------------|---------------|
| NLB | 2 | 10 | 70% | 80% | active_connections (1000) |
| ALB | 3 | 20 | 60% | 75% | http_requests_per_second (1000) |
| DBLB | 2 | 15 | 65% | 80% | database_connections (500) |
| AILB | 2 | 10 | 60% | 75% | ai_requests_per_minute (30) |
| RTMP | 1 | 5 | 75% | 80% | active_streams (5) |

### Documentation Files

#### DEPLOYMENT.md
**Purpose:** Complete deployment guide for all platforms
**Sections:**
- Architecture overview
- Docker Compose deployment
- Kubernetes deployment
- GPU acceleration setup
- Production considerations
- Troubleshooting

#### k8s/unified/README.md
**Purpose:** Kubernetes-specific deployment guide
**Sections:**
- Directory structure
- Quick start
- Configuration management
- Scaling and monitoring
- Resource requirements
- Production considerations
- Troubleshooting

#### PHASE8_DOCKER_K8S_SUMMARY.md
**Purpose:** Implementation summary and reference
**Sections:**
- Files created
- Architecture summary
- Key features
- Port mappings
- Resource requirements
- Usage examples
- Testing checklist

---

## Environment Variables Reference

### Required Variables

```bash
# Database (Docker Compose)
POSTGRES_PASSWORD=your-password
REDIS_PASSWORD=your-password

# API Server
SECRET_KEY=min-32-characters
CLUSTER_API_KEY=your-api-key

# License (Enterprise)
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

### Optional Variables

```bash
# AI Providers (for AILB)
OPENAI_API_KEY=sk-...
ANTHROPIC_API_KEY=sk-ant-...

# Network
NLB_PORT=7000
LOG_LEVEL=info

# Feature Flags
ENABLE_EBPF=true
ENABLE_XDP=false
RATE_LIMIT_ENABLED=true

# GPU (RTMP)
GPU_ENABLED=true
GPU_TYPE=nvidia  # or amd
NVENC_PRESET=p4
```

---

## Common Commands

### Docker Compose

```bash
# Start core stack
docker-compose up -d

# Start with all modules
docker-compose --profile full up -d

# Start with NVIDIA GPU
docker-compose -f docker-compose.yml -f docker-compose.gpu-nvidia.yml --profile gpu-nvidia up -d

# View logs
docker-compose logs -f <service>

# Restart service
docker-compose restart <service>

# Stop and remove
docker-compose down

# Stop and remove volumes (destructive!)
docker-compose down -v
```

### Kubernetes

```bash
# Deploy all
kubectl apply -R -f k8s/unified/

# Deploy core only
kubectl apply -f k8s/unified/base/
kubectl apply -f k8s/unified/nlb/
kubectl apply -f k8s/unified/alb/

# Check status
kubectl get all -n marchproxy
kubectl get pods -n marchproxy
kubectl get svc -n marchproxy
kubectl get hpa -n marchproxy

# View logs
kubectl logs -f deployment/proxy-nlb -n marchproxy

# Scale manually
kubectl scale deployment proxy-nlb --replicas=5 -n marchproxy

# Port forward
kubectl port-forward -n marchproxy svc/proxy-nlb 7001:7001

# Delete all
kubectl delete namespace marchproxy
```

---

## Support Resources

- **Main Documentation:** [DEPLOYMENT.md](DEPLOYMENT.md)
- **Kubernetes Guide:** [k8s/unified/README.md](k8s/unified/README.md)
- **Implementation Summary:** [PHASE8_DOCKER_K8S_SUMMARY.md](PHASE8_DOCKER_K8S_SUMMARY.md)
- **GitHub Issues:** https://github.com/penguintech/marchproxy/issues
- **Enterprise Support:** support@penguintech.io
