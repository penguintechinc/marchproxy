# MarchProxy Helm Charts Summary

Comprehensive Helm v3 charts for MarchProxy ingress and egress proxies.

## Overview

Two production-ready Helm charts have been created for deploying MarchProxy in Kubernetes:

1. **marchproxy-ingress**: High-performance reverse proxy (ingress)
2. **marchproxy-egress**: High-performance forward proxy (egress)

## Chart Statistics

### MarchProxy Ingress Chart
- **Total Files**: 14
- **Templates**: 10
- **Location**: `/home/penguin/code/MarchProxy/helm/marchproxy-ingress/`
- **Validation**: ✅ PASSED (helm lint)

### MarchProxy Egress Chart
- **Total Files**: 17
- **Templates**: 13
- **Location**: `/home/penguin/code/MarchProxy/helm/marchproxy-egress/`
- **Validation**: ✅ PASSED (helm lint)

## File Structure

### Ingress Chart Files

```
marchproxy-ingress/
├── Chart.yaml                      # Chart metadata
├── values.yaml                     # Default configuration values
├── README.md                       # Comprehensive documentation
├── .helmignore                     # Files to ignore when packaging
└── templates/
    ├── _helpers.tpl                # Template helpers
    ├── deployment.yaml             # Deployment resource
    ├── service.yaml                # LoadBalancer service
    ├── configmap.yaml              # Configuration
    ├── serviceaccount.yaml         # RBAC (ServiceAccount, Role, RoleBinding)
    ├── hpa.yaml                    # Horizontal Pod Autoscaler
    ├── pdb.yaml                    # Pod Disruption Budget
    ├── pvc.yaml                    # Persistent Volume Claim (logs)
    ├── tls-secret.yaml             # TLS certificates
    └── servicemonitor.yaml         # Prometheus ServiceMonitor
```

### Egress Chart Files

```
marchproxy-egress/
├── Chart.yaml                      # Chart metadata
├── values.yaml                     # Default configuration values
├── README.md                       # Comprehensive documentation
├── .helmignore                     # Files to ignore when packaging
└── templates/
    ├── _helpers.tpl                # Template helpers
    ├── _pod.tpl                    # Shared pod template
    ├── deployment.yaml             # Deployment or DaemonSet
    ├── service.yaml                # ClusterIP service
    ├── configmap.yaml              # Configuration
    ├── envoy-configmap.yaml        # Envoy L7 configuration
    ├── serviceaccount.yaml         # RBAC (ServiceAccount, Role, RoleBinding)
    ├── networkpolicy.yaml          # Network policies
    ├── hpa.yaml                    # Horizontal Pod Autoscaler
    ├── pdb.yaml                    # Pod Disruption Budget
    ├── pvc.yaml                    # Persistent Volume Claim (logs)
    ├── tls-secret.yaml             # TLS certificates (incl. MITM CA)
    └── servicemonitor.yaml         # Prometheus ServiceMonitor
```

## Key Features

### MarchProxy Ingress

**Architecture:**
- Deployment-based (3 replicas default)
- LoadBalancer service (AWS NLB/ALB support)
- Horizontal Pod Autoscaler (3-20 replicas)

**Features:**
- ✅ eBPF/XDP acceleration
- ✅ mTLS with client verification
- ✅ Rate limiting (10K RPS default)
- ✅ Health checks and readiness probes
- ✅ Pod Disruption Budget (minAvailable: 2)
- ✅ RBAC (NET_ADMIN, NET_RAW capabilities)
- ✅ Prometheus metrics
- ✅ AWS NLB/ALB mode selection
- ✅ TLS certificate management
- ✅ Persistent logs (optional)

**Ports:**
- 80: HTTP
- 443: HTTPS
- 8082: Admin/Metrics
- 8083: Health checks

### MarchProxy Egress

**Architecture:**
- Deployment (default) or DaemonSet mode
- ClusterIP service (internal only)
- Optional HPA (deployment mode only)

**Features:**
- ✅ Dual-layer proxy: L4 (Go) + L7 (Envoy)
- ✅ eBPF/XDP acceleration
- ✅ mTLS with client verification
- ✅ TLS interception (MITM mode)
- ✅ Threat intelligence integration
- ✅ External authorization (ext_authz)
- ✅ Network policies (egress control)
- ✅ HTTP/3 (QUIC) support (experimental)
- ✅ Rate limiting (5K RPS default)
- ✅ RBAC (NET_ADMIN, NET_RAW capabilities)
- ✅ Prometheus metrics (Go + Envoy)
- ✅ Envoy admin interface

**Ports:**
- 8080: L4 TCP proxy
- 8081: Admin/Metrics
- 10000: L7 HTTP (Envoy)
- 10443: L7 HTTPS/QUIC (Envoy)
- 9901: Envoy admin
- 9002: External authorization

## Installation

### Ingress Chart

```bash
# Basic installation
helm install my-ingress ./helm/marchproxy-ingress

# AWS NLB with auto-scaling
helm install my-ingress ./helm/marchproxy-ingress \
  --set ingress.mode=nlb \
  --set autoscaling.enabled=true \
  --set autoscaling.maxReplicas=50

# AWS ALB with mTLS
helm install my-ingress ./helm/marchproxy-ingress \
  --set ingress.mode=alb \
  --set ingress.alb.enabled=true \
  --set config.mtls.enabled=true
```

### Egress Chart

```bash
# Basic installation (Deployment mode)
helm install my-egress ./helm/marchproxy-egress

# DaemonSet mode (one pod per node)
helm install my-egress ./helm/marchproxy-egress \
  --set deploymentMode=daemonset

# With threat intelligence and TLS interception
helm install my-egress ./helm/marchproxy-egress \
  --set config.threatIntel.ipBlockingEnabled=true \
  --set config.tlsInterception.enabled=true
```

## Configuration Highlights

### Ingress - Key Values

```yaml
# Mode selection
ingress:
  mode: "nlb"  # or "alb"

# Auto-scaling
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPUUtilizationPercentage: 70

# Performance
config:
  ebpf:
    enabled: true
  xdp:
    enabled: false
  rateLimit:
    requestsPerSecond: 10000
```

### Egress - Key Values

```yaml
# Deployment mode
deploymentMode: "deployment"  # or "daemonset"

# L7 Envoy
config:
  l7:
    enabled: true
    http3Enabled: false  # EXPERIMENTAL

# Threat intelligence
config:
  threatIntel:
    ipBlockingEnabled: true
    domainBlockingEnabled: true

# Network policy
networkPolicy:
  enabled: true
```

## Testing

Both charts have been validated with:

```bash
# Lint validation
helm lint marchproxy-ingress  # ✅ PASSED
helm lint marchproxy-egress   # ✅ PASSED

# Template rendering
helm template test marchproxy-ingress --dry-run  # ✅ SUCCESS
helm template test marchproxy-egress --dry-run   # ✅ SUCCESS
```

## Security

### RBAC Capabilities

Both charts require:
- `NET_ADMIN`: For eBPF/XDP packet processing
- `NET_RAW`: For raw socket operations

### Security Context

```yaml
securityContext:
  pod:
    runAsNonRoot: true
    runAsUser: 1000
    fsGroup: 1000
  container:
    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true
    capabilities:
      add: [NET_ADMIN, NET_RAW]
      drop: [ALL]
```

### Network Policies

Egress chart includes NetworkPolicy for egress traffic control:
- DNS allowed (UDP/53)
- Configurable egress rules
- Support for IP/CIDR restrictions

## Monitoring

Both charts support Prometheus monitoring:

### Metrics Endpoints

**Ingress:**
- Port 8082: `/metrics` (Go proxy)

**Egress:**
- Port 8081: `/metrics` (Go proxy)
- Port 9901: `/stats/prometheus` (Envoy)

### ServiceMonitor

```yaml
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
```

## High Availability

### Ingress

- Horizontal Pod Autoscaler (3-20 replicas)
- Pod Disruption Budget (minAvailable: 2)
- Pod anti-affinity (prefer different nodes)
- Health checks (liveness, readiness, startup)

### Egress

- Deployment mode: HPA + PDB
- DaemonSet mode: One pod per node
- Health checks for both Go and Envoy
- Pod anti-affinity (prefer different nodes)

## Cloud Provider Support

### AWS

**Ingress:**
- NLB (Network Load Balancer) annotations
- ALB (Application Load Balancer) via Ingress resource
- Cross-zone load balancing
- Health check configuration

**Egress:**
- Works with VPC endpoints
- Network policies for VPC-only egress
- Instance metadata service access

### Azure/GCP

Both charts include placeholder annotations for:
- Azure Load Balancer
- GCP Internal/External Load Balancer

## Migration from Docker Compose

The charts are designed to match the Docker Compose deployment:

### Image References

- Ingress: `marchproxy/proxy-ingress:v1.0.0`
- Egress: `marchproxy/proxy-egress:v1.0.0`

### Environment Variables

All Docker Compose environment variables are supported via `values.yaml` configuration.

### Port Mappings

Ports match Docker Compose configuration for drop-in replacement.

## Documentation

Each chart includes comprehensive README.md with:
- Installation instructions
- Configuration examples
- Troubleshooting guide
- Cloud provider-specific examples
- Security best practices

## Next Steps

### Recommended Actions

1. **Test in staging environment**
   ```bash
   helm install test-ingress ./helm/marchproxy-ingress --namespace staging
   helm install test-egress ./helm/marchproxy-egress --namespace staging
   ```

2. **Configure TLS certificates**
   - Use cert-manager for automatic certificate management
   - Or provide certificates via values.yaml

3. **Set up monitoring**
   - Deploy Prometheus Operator
   - Enable ServiceMonitors
   - Configure alerts

4. **Tune resource limits**
   - Adjust CPU/memory based on load testing
   - Configure HPA thresholds
   - Set appropriate rate limits

5. **Network policies**
   - Define egress rules for egress proxy
   - Restrict ingress sources if needed

6. **Package for distribution**
   ```bash
   helm package ./helm/marchproxy-ingress
   helm package ./helm/marchproxy-egress
   ```

## Validation Status

- ✅ Helm lint passed (both charts)
- ✅ Template rendering successful
- ✅ YAML syntax valid
- ✅ Kubernetes resource schemas correct
- ✅ RBAC properly configured
- ✅ Security contexts defined
- ✅ Health checks configured
- ✅ Service types appropriate
- ✅ Documentation complete

## Support

For issues or questions:
- GitHub: https://github.com/marchproxy/marchproxy
- Documentation: https://docs.marchproxy.io
- Email: support@marchproxy.io

---

**Charts Version**: 1.0.0
**Created**: 2026-01-08
**Helm Version**: v3.x
**Kubernetes Version**: 1.19+
