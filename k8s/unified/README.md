# MarchProxy Unified NLB Architecture - Kubernetes Deployment

This directory contains Kubernetes manifests for deploying MarchProxy's unified NLB (Network Load Balancer) architecture with all optional proxy modules.

## Architecture Overview

```
┌─────────────┐
│   Ingress   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  NLB (L3/L4) - Entry Point          │
│  - Protocol inspection              │
│  - Traffic routing to modules       │
│  - Rate limiting & auto-scaling     │
└──────┬──────────────────────────────┘
       │ gRPC Communication
       │
       ├──► ALB (L7 Envoy) - HTTP/HTTPS
       ├──► DBLB - Database Proxy
       ├──► AILB - AI/LLM Proxy
       └──► RTMP - Video Transcoding
```

## Directory Structure

```
k8s/unified/
├── base/
│   ├── namespace.yaml          # Namespace, quotas, limits
│   ├── configmap.yaml          # Global and module-specific configs
│   └── secrets.yaml.example    # Secret template (copy to secrets.yaml)
├── nlb/
│   ├── deployment.yaml         # NLB deployment
│   └── service.yaml            # LoadBalancer service
├── alb/
│   ├── deployment.yaml         # ALB (Envoy) deployment
│   └── service.yaml            # ClusterIP service
├── dblb/
│   ├── deployment.yaml         # Database LB deployment
│   └── service.yaml            # ClusterIP service
├── ailb/
│   ├── deployment.yaml         # AI/LLM LB deployment
│   └── service.yaml            # ClusterIP service
├── rtmp/
│   ├── deployment.yaml         # RTMP transcoding deployment
│   └── service.yaml            # ClusterIP service
└── hpa/
    ├── nlb-hpa.yaml            # NLB auto-scaling (2-10 pods)
    ├── alb-hpa.yaml            # ALB auto-scaling (3-20 pods)
    ├── dblb-hpa.yaml           # DBLB auto-scaling (2-15 pods)
    ├── ailb-hpa.yaml           # AILB auto-scaling (2-10 pods)
    └── rtmp-hpa.yaml           # RTMP auto-scaling (1-5 pods)
```

## Quick Start

### 1. Prerequisites

- Kubernetes cluster (v1.24+)
- kubectl configured
- Metrics Server installed (for HPA)
- Docker images built and pushed to registry

```bash
# Verify cluster access
kubectl cluster-info

# Verify metrics server
kubectl top nodes
```

### 2. Create Secrets

```bash
# Copy secrets template
cp k8s/unified/base/secrets.yaml.example k8s/unified/base/secrets.yaml

# Edit secrets.yaml with your actual values
vim k8s/unified/base/secrets.yaml

# Generate random secrets
CLUSTER_API_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -base64 48)
POSTGRES_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)

# Update secrets.yaml with generated values
```

### 3. Deploy Base Infrastructure

```bash
# Create namespace and basic resources
kubectl apply -f k8s/unified/base/namespace.yaml
kubectl apply -f k8s/unified/base/configmap.yaml
kubectl apply -f k8s/unified/base/secrets.yaml
```

### 4. Deploy Core Modules (NLB + ALB)

```bash
# Deploy NLB (entry point)
kubectl apply -f k8s/unified/nlb/

# Deploy ALB (L7 HTTP proxy)
kubectl apply -f k8s/unified/alb/

# Deploy HPA for auto-scaling
kubectl apply -f k8s/unified/hpa/nlb-hpa.yaml
kubectl apply -f k8s/unified/hpa/alb-hpa.yaml
```

### 5. Deploy Optional Modules

```bash
# Deploy DBLB (database proxy) - Optional
kubectl apply -f k8s/unified/dblb/
kubectl apply -f k8s/unified/hpa/dblb-hpa.yaml

# Deploy AILB (AI/LLM proxy) - Optional
kubectl apply -f k8s/unified/ailb/
kubectl apply -f k8s/unified/hpa/ailb-hpa.yaml

# Deploy RTMP (video transcoding) - Optional
kubectl apply -f k8s/unified/rtmp/
kubectl apply -f k8s/unified/hpa/rtmp-hpa.yaml
```

### 6. Verify Deployment

```bash
# Check all pods
kubectl get pods -n marchproxy

# Check services
kubectl get svc -n marchproxy

# Check HPA status
kubectl get hpa -n marchproxy

# Get NLB external IP
kubectl get svc proxy-nlb -n marchproxy
```

## Complete Deployment

Deploy everything at once:

```bash
# Deploy all manifests
kubectl apply -R -f k8s/unified/

# Watch deployment progress
kubectl get pods -n marchproxy -w
```

## Configuration

### Update ConfigMaps

```bash
# Edit global config
kubectl edit configmap marchproxy-config -n marchproxy

# Edit module-specific config
kubectl edit configmap marchproxy-nlb-config -n marchproxy
kubectl edit configmap marchproxy-alb-config -n marchproxy

# Restart deployments to apply changes
kubectl rollout restart deployment proxy-nlb -n marchproxy
kubectl rollout restart deployment proxy-alb -n marchproxy
```

### Update Secrets

```bash
# Edit secrets
kubectl edit secret marchproxy-secrets -n marchproxy

# Or replace from file
kubectl delete secret marchproxy-secrets -n marchproxy
kubectl apply -f k8s/unified/base/secrets.yaml
```

### Scaling

```bash
# Manual scaling (overrides HPA)
kubectl scale deployment proxy-nlb --replicas=5 -n marchproxy

# Update HPA limits
kubectl edit hpa proxy-nlb-hpa -n marchproxy

# Disable HPA temporarily
kubectl delete hpa proxy-nlb-hpa -n marchproxy
```

## Monitoring

### View Logs

```bash
# NLB logs
kubectl logs -f deployment/proxy-nlb -n marchproxy

# ALB logs
kubectl logs -f deployment/proxy-alb -n marchproxy

# All pods
kubectl logs -f -l app.kubernetes.io/name=marchproxy -n marchproxy --max-log-requests=20
```

### Metrics

```bash
# Pod resource usage
kubectl top pods -n marchproxy

# Node resource usage
kubectl top nodes

# HPA status
kubectl describe hpa -n marchproxy
```

### Port Forwarding

```bash
# Access NLB admin interface
kubectl port-forward -n marchproxy svc/proxy-nlb 7001:7001

# Access ALB Envoy admin
kubectl port-forward -n marchproxy svc/proxy-alb 9901:9901

# Access AILB HTTP API
kubectl port-forward -n marchproxy svc/proxy-ailb 7003:7003
```

## Troubleshooting

### Pods Not Starting

```bash
# Describe pod to see events
kubectl describe pod <pod-name> -n marchproxy

# Check for image pull errors
kubectl get events -n marchproxy --sort-by='.lastTimestamp'

# Check resource constraints
kubectl describe node <node-name>
```

### Network Issues

```bash
# Test gRPC connectivity between pods
kubectl exec -it -n marchproxy <nlb-pod> -- curl -v http://proxy-alb:50051

# Check service endpoints
kubectl get endpoints -n marchproxy

# Verify DNS resolution
kubectl exec -it -n marchproxy <pod-name> -- nslookup proxy-alb
```

### HPA Not Scaling

```bash
# Check metrics server
kubectl get apiservice v1beta1.metrics.k8s.io -o yaml

# Check HPA status
kubectl describe hpa proxy-nlb-hpa -n marchproxy

# View current metrics
kubectl get hpa -n marchproxy --watch
```

## Resource Requirements

### Minimum Cluster Size

- **Nodes:** 3 (for high availability)
- **CPU:** 20 cores total
- **Memory:** 40 GB total
- **Storage:** 200 GB

### Per-Module Requirements

| Module | Min Replicas | CPU Request | Memory Request | CPU Limit | Memory Limit |
|--------|--------------|-------------|----------------|-----------|--------------|
| NLB    | 2            | 1000m       | 1Gi            | 4000m     | 4Gi          |
| ALB    | 3            | 500m        | 512Mi          | 2000m     | 2Gi          |
| DBLB   | 2            | 1000m       | 1Gi            | 4000m     | 4Gi          |
| AILB   | 2            | 500m        | 1Gi            | 2000m     | 4Gi          |
| RTMP   | 1            | 2000m       | 2Gi            | 8000m     | 8Gi          |

## Production Considerations

### High Availability

1. **Pod Disruption Budgets**
   ```bash
   kubectl create pdb proxy-nlb-pdb --selector=app.kubernetes.io/component=proxy-nlb --min-available=1 -n marchproxy
   ```

2. **Anti-Affinity Rules** - Already configured in deployments to spread pods across nodes

3. **Multiple Availability Zones** - Ensure nodes span multiple AZs

### Security

1. **Network Policies**
   - Restrict inter-pod communication
   - Allow only necessary external access

2. **Pod Security Policies**
   - Non-root containers (already configured)
   - Read-only root filesystem where possible
   - Drop unnecessary capabilities

3. **Secrets Management**
   - Use external secret managers (Vault, AWS Secrets Manager)
   - Rotate secrets regularly
   - Never commit secrets to git

### Monitoring & Alerting

1. **Prometheus Integration**
   - All pods expose `/metrics` endpoint
   - Service monitors configured

2. **Recommended Alerts**
   - Pod restart count > 5
   - CPU/Memory usage > 90%
   - HPA at max replicas
   - Service endpoint count = 0

## Cleanup

### Remove Specific Module

```bash
# Remove RTMP module
kubectl delete -f k8s/unified/rtmp/
kubectl delete -f k8s/unified/hpa/rtmp-hpa.yaml
```

### Remove All Resources

```bash
# Delete entire namespace (removes everything)
kubectl delete namespace marchproxy

# Or delete selectively
kubectl delete -R -f k8s/unified/
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/penguintech/marchproxy/issues
- Documentation: https://marchproxy.penguintech.io
- Enterprise Support: support@penguintech.io
