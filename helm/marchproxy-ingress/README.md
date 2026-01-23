# MarchProxy Ingress Helm Chart

High-performance reverse proxy with eBPF/XDP acceleration and mTLS support for Kubernetes.

## Overview

MarchProxy Ingress is a high-performance ingress proxy designed for cloud-native environments. It provides:

- **High Performance**: eBPF/XDP acceleration for kernel-level packet processing
- **Security**: Built-in mTLS support with client certificate verification
- **Scalability**: Horizontal Pod Autoscaler (HPA) for automatic scaling
- **Cloud Native**: Native Kubernetes integration with Service, Deployment, and ConfigMap resources
- **Observability**: Prometheus metrics, health checks, and distributed tracing

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- NET_ADMIN and NET_RAW capabilities (for eBPF/XDP)

## Installation

### Add the Helm repository (if applicable)

```bash
helm repo add marchproxy https://charts.marchproxy.io
helm repo update
```

### Install the chart

```bash
# Basic installation
helm install my-ingress marchproxy-ingress

# With custom values
helm install my-ingress marchproxy-ingress \
  --set replicaCount=5 \
  --set ingress.mode=alb

# From local directory
helm install my-ingress ./helm/marchproxy-ingress
```

## Configuration

### Core Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `3` |
| `image.repository` | Image repository | `marchproxy/proxy-ingress` |
| `image.tag` | Image tag | `v1.0.0` |
| `service.type` | Service type | `LoadBalancer` |
| `ingress.mode` | Ingress mode (alb/nlb) | `nlb` |

### mTLS Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.mtls.enabled` | Enable mTLS | `true` |
| `config.mtls.verifyClient` | Verify client certificates | `true` |
| `tls.createSecret` | Create TLS secret from values | `false` |

### Performance Tuning

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.ebpf.enabled` | Enable eBPF acceleration | `true` |
| `config.xdp.enabled` | Enable XDP acceleration | `false` |
| `config.rateLimit.requestsPerSecond` | Rate limit (RPS) | `10000` |
| `resources.requests.cpu` | CPU request | `1000m` |
| `resources.limits.cpu` | CPU limit | `4000m` |

### Autoscaling

| Parameter | Description | Default |
|-----------|-------------|---------|
| `autoscaling.enabled` | Enable HPA | `true` |
| `autoscaling.minReplicas` | Minimum replicas | `3` |
| `autoscaling.maxReplicas` | Maximum replicas | `20` |
| `autoscaling.targetCPUUtilizationPercentage` | Target CPU % | `70` |

## Examples

### AWS NLB with Auto-scaling

```yaml
service:
  type: LoadBalancer
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"

ingress:
  mode: "nlb"

autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 50
  targetCPUUtilizationPercentage: 70
```

### AWS ALB with mTLS

```yaml
ingress:
  mode: "alb"
  alb:
    enabled: true
    className: "alb"
    annotations:
      alb.ingress.kubernetes.io/scheme: internet-facing
      alb.ingress.kubernetes.io/target-type: ip
    hosts:
      - host: api.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: api-tls
        hosts:
          - api.example.com

config:
  mtls:
    enabled: true
    verifyClient: true

tls:
  createSecret: true
  cert: "LS0tLS1CRUdJTi..."  # Base64-encoded
  key: "LS0tLS1CRUdJTi..."   # Base64-encoded
  clientCa: "LS0tLS1CRUdJTi..." # Base64-encoded CA bundle
```

### High-Performance Configuration

```yaml
config:
  ebpf:
    enabled: true
  xdp:
    enabled: true
    interface: "eth0"
  rateLimit:
    requestsPerSecond: 50000
    burstSize: 100000

resources:
  requests:
    cpu: 2000m
    memory: 2Gi
  limits:
    cpu: 8000m
    memory: 8Gi

nodeSelector:
  node.kubernetes.io/instance-type: c5.4xlarge
```

## Monitoring

The chart exposes Prometheus metrics on port `8082` at `/metrics` endpoint.

### ServiceMonitor

```yaml
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
```

## Troubleshooting

### Pods not starting

Check if NET_ADMIN capability is granted:

```bash
kubectl describe pod <pod-name>
```

### XDP not working

Ensure the node kernel supports XDP (4.8+):

```bash
kubectl exec <pod-name> -- uname -r
```

### Health check failing

Check the health endpoint:

```bash
kubectl port-forward <pod-name> 8083:8083
curl http://localhost:8083/healthz
```

## Uninstallation

```bash
helm uninstall my-ingress
```

## Support

- GitHub: https://github.com/marchproxy/marchproxy
- Documentation: https://docs.marchproxy.io
- Email: support@marchproxy.io
