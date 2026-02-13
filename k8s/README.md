# MarchProxy Kubernetes Deployment

This directory contains standardized Kubernetes deployment configurations for MarchProxy using Kustomize for environment-specific customization.

## Directory Structure

```
k8s/
├── kustomize/
│   ├── base/                    # Base manifests (shared across all environments)
│   │   ├── kustomization.yaml   # Base kustomization configuration
│   │   ├── namespace.yaml       # Namespace, LimitRange, ResourceQuota
│   │   ├── services.yaml        # Service definitions (manager, proxy, postgres, redis, etc.)
│   │   ├── configmap.yaml       # Application configuration
│   │   └── secrets.yaml         # Sensitive data (passwords, API keys)
│   │
│   └── overlays/                # Environment-specific customizations
│       ├── alpha/               # Alpha/Development environment
│       │   └── kustomization.yaml
│       └── beta/                # Beta/Staging environment
│           └── kustomization.yaml
│
├── manifests/                   # Raw Kubernetes manifests (for reference)
│   ├── namespace.yaml
│   └── services.yaml
│
└── README.md                    # This file
```

## Configuration Overview

### Base Configuration (`k8s/kustomize/base/`)

The base kustomization contains:

- **Namespace**: `marchproxy` with resource quotas and limits
- **Services**: Manager (8000), Proxy (8080 TCP/UDP, 8443 QUIC), Database, Cache, Monitoring
- **ConfigMaps**: Application configuration for proxy, manager, prometheus, grafana
- **Secrets**: Default credentials (must be overridden in production)
- **Replicas**: Manager (2), Proxy (3) - overridden by overlays
- **Images**: Using `registry-dal2.penguintech.io` registry

### Alpha Overlay (`k8s/kustomize/overlays/alpha/`)

Development/testing environment:

- **Namespace**: `marchproxy-alpha`
- **Replicas**: Manager (1), Proxy (1)
- **Log Level**: DEBUG
- **Environment**: alpha
- **Resources**: Minimal (128Mi memory, 100m CPU requests)
- **Suffix**: `-alpha` appended to resource names
- **Images**: Using `latest` tags

### Beta Overlay (`k8s/kustomize/overlays/beta/`)

Staging/production environment:

- **Namespace**: `marchproxy-beta`
- **Replicas**: Manager (2), Proxy (2)
- **Log Level**: INFO
- **Environment**: beta
- **Resources**: Standard (256Mi memory, 250m CPU requests)
- **Suffix**: `-beta` appended to resource names
- **Images**: Using stable version tags (v1.0.4, v1.0.0)
- **Ingress**: Configured for `marchproxy.penguintech.io`
- **Host**: `marchproxy.penguintech.io`

## Deployment Instructions

### Prerequisites

```bash
# Install/verify tools
kubectl version --client
kustomize version

# Set Kubernetes context
kubectl config use-context <your-cluster>
```

### Deploy to Alpha Environment

```bash
# Preview manifests
kustomize build k8s/kustomize/overlays/alpha

# Apply deployment
kustomize build k8s/kustomize/overlays/alpha | kubectl apply -f -

# Or use the provided script
./scripts/deploy-beta.sh  # (Can be adapted for alpha)
```

### Deploy to Beta Environment

```bash
# Preview manifests
kustomize build k8s/kustomize/overlays/beta

# Apply deployment
kustomize build k8s/kustomize/overlays/beta | kubectl apply -f -

# Or use the provided deployment script
./scripts/deploy-beta.sh
```

## Deploy Script

The `scripts/deploy-beta.sh` script automates the deployment process:

```bash
#!/bin/bash
# Configuration
RELEASE_NAME="marchproxy"
NAMESPACE="marchproxy"
IMAGE_REGISTRY="registry-dal2.penguintech.io"
KUBE_CONTEXT="dal2-beta"
APP_HOST="marchproxy.penguintech.io"

# Features:
# - Validates prerequisites (kubectl, kustomize)
# - Checks Kubernetes context
# - Validates kustomize build
# - Creates namespace
# - Applies manifests
# - Waits for deployment rollout
# - Verifies endpoints
# - Provides rollback on failure

./scripts/deploy-beta.sh
```

## Customization

### Update Image Versions

Edit the overlay's `kustomization.yaml`:

```yaml
images:
  - name: marchproxy-manager
    newTag: v1.0.5  # Update version
  - name: marchproxy-proxy
    newTag: v1.0.1
```

### Change Replica Count

Edit the overlay's `kustomization.yaml`:

```yaml
replicas:
  - name: manager-deployment
    count: 3  # Scale manager
  - name: proxy-deployment
    count: 5  # Scale proxy
```

### Override Resources

In overlay `kustomization.yaml`, use patches:

```yaml
patches:
  - target:
      kind: Deployment
      name: manager-deployment
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/0/resources/requests/memory
        value: "512Mi"
```

### Update Configuration

Add to overlay's `configMapGenerator`:

```yaml
configMapGenerator:
  - name: marchproxy-config
    behavior: merge
    literals:
      - CUSTOM_SETTING=value
```

### Manage Secrets

**WARNING**: Secrets in YAML are base64 encoded, not encrypted. Use Kubernetes secrets management:

```bash
# Create secret from file
kubectl create secret generic my-secret --from-file=config.yaml

# Or use Sealed Secrets / Vault
```

## Monitoring and Verification

### Check Deployment Status

```bash
# Watch rollout progress
kubectl rollout status deployment/manager-deployment-beta -n marchproxy-beta

# View pods
kubectl get pods -n marchproxy-beta

# View services
kubectl get svc -n marchproxy-beta

# View events
kubectl get events -n marchproxy-beta
```

### Access Services

```bash
# Port forward to manager
kubectl port-forward -n marchproxy-beta svc/marchproxy-manager-beta 8000:8000

# Port forward to prometheus
kubectl port-forward -n marchproxy-beta svc/prometheus-beta 9090:9090

# Access web UI (via ingress)
curl https://marchproxy.penguintech.io
```

### View Logs

```bash
# View manager logs
kubectl logs -n marchproxy-beta deployment/manager-deployment-beta -f

# View proxy logs
kubectl logs -n marchproxy-beta deployment/proxy-deployment-beta -f

# View specific pod logs
kubectl logs -n marchproxy-beta <pod-name> -f
```

## Maintenance

### Scale Deployment

```bash
# Scale manager replicas
kubectl scale deployment/manager-deployment-beta --replicas=3 -n marchproxy-beta

# Scale proxy replicas
kubectl scale deployment/proxy-deployment-beta --replicas=5 -n marchproxy-beta
```

### Update Deployment

```bash
# Update manager image
kubectl set image deployment/manager-deployment-beta \
  manager=registry-dal2.penguintech.io/marchproxy-manager:v1.0.5 \
  -n marchproxy-beta

# Or reapply kustomize with updated image
kustomize build k8s/kustomize/overlays/beta | kubectl apply -f -
```

### Rollback Deployment

```bash
# Check rollout history
kubectl rollout history deployment/manager-deployment-beta -n marchproxy-beta

# Rollback to previous version
kubectl rollout undo deployment/manager-deployment-beta -n marchproxy-beta
```

### Delete Deployment

```bash
# Delete all resources in namespace
kubectl delete namespace marchproxy-beta

# Or delete using kustomize
kustomize build k8s/kustomize/overlays/beta | kubectl delete -f -
```

## Service Architecture

### Manager Service (8000)
- REST API for configuration and management
- Metrics endpoint (9090)
- Service: `marchproxy-manager-beta` (ClusterIP)
- Headless: `marchproxy-manager-headless-beta` (for StatefulSets)

### Proxy Service (8080/8443)
- Main proxy entry point
- TCP (8080), UDP (8080), QUIC (8443)
- Metrics (9090), Health check (8888)
- Service: `marchproxy-proxy-beta` (LoadBalancer with source ranges)
- Headless: `marchproxy-proxy-headless-beta`

### Database Service (5432)
- PostgreSQL database
- Service: `postgresql-beta` (ClusterIP)

### Cache Service (6379)
- Redis cache
- Service: `redis-beta` (ClusterIP)

### Monitoring Services
- **Prometheus** (9090): Metrics collection
- **Grafana** (3000): Visualization

## Network Policies

By default, network policies are disabled in alpha and can be enabled in beta:

```yaml
security:
  networkPolicy:
    enabled: true
```

To add custom policies, create additional manifests or patch existing ones.

## Resource Quotas

Default quotas per namespace:

- **CPU Requests**: 4 cores
- **CPU Limits**: 8 cores
- **Memory Requests**: 8Gi
- **Memory Limits**: 16Gi
- **Persistent Volumes**: 10
- **Pods**: 20
- **Services**: 10

Edit in `base/namespace.yaml` to adjust for your cluster.

## Troubleshooting

### Pods not starting

```bash
# Check pod status and events
kubectl describe pod <pod-name> -n marchproxy-beta

# Check resource constraints
kubectl top nodes
kubectl top pods -n marchproxy-beta
```

### Image pull errors

```bash
# Verify image registry access
kubectl create secret docker-registry regcred \
  --docker-server=registry-dal2.penguintech.io \
  --docker-username=<username> \
  --docker-password=<password>

# Add to deployment's imagePullSecrets
```

### Service not accessible

```bash
# Check service endpoints
kubectl get endpoints -n marchproxy-beta

# Test connectivity
kubectl run -it debug --image=alpine --rm -- sh
# Inside pod: wget http://marchproxy-manager-beta:8000
```

## CI/CD Integration

The deployment script can be integrated into CI/CD pipelines:

```bash
#!/bin/bash
./scripts/deploy-beta.sh && \
  kubectl get all -n marchproxy-beta
```

## References

- [Kustomize Documentation](https://kustomize.io/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [MarchProxy Documentation](https://docs.marchproxy.io/)
