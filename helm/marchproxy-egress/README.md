# MarchProxy Egress Helm Chart

High-performance egress proxy with eBPF/XDP, L7 (Envoy), TLS interception, and threat intelligence for Kubernetes.

## Overview

MarchProxy Egress is a comprehensive egress proxy solution designed for secure and performant outbound traffic management. It provides:

- **Dual-Layer Architecture**: L4 (Go) + L7 (Envoy) for complete traffic control
- **Security**: mTLS, TLS interception (MITM), threat intelligence integration
- **Performance**: eBPF/XDP acceleration, HTTP/3 (QUIC) support
- **Flexible Deployment**: Deployment or DaemonSet modes
- **Traffic Control**: Rate limiting, external authorization, network policies
- **Observability**: Prometheus metrics, Envoy admin interface

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
# Basic installation (Deployment mode)
helm install my-egress marchproxy-egress

# DaemonSet mode (one pod per node)
helm install my-egress marchproxy-egress \
  --set deploymentMode=daemonset

# With custom values
helm install my-egress marchproxy-egress \
  --set config.l7.enabled=true \
  --set config.threatIntel.ipBlockingEnabled=true

# From local directory
helm install my-egress ./helm/marchproxy-egress
```

## Configuration

### Core Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `deploymentMode` | Deployment mode (deployment/daemonset) | `deployment` |
| `replicaCount` | Number of replicas (deployment mode) | `3` |
| `image.repository` | Image repository | `marchproxy/proxy-egress` |
| `image.tag` | Image tag | `v1.0.0` |
| `service.type` | Service type | `ClusterIP` |

### L7 Envoy Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.l7.enabled` | Enable L7 (Envoy) proxy | `true` |
| `config.l7.envoyLogLevel` | Envoy log level | `info` |
| `config.l7.http3Enabled` | Enable HTTP/3 (QUIC) - EXPERIMENTAL | `false` |

### Threat Intelligence

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.threatIntel.ipBlockingEnabled` | Block malicious IPs | `true` |
| `config.threatIntel.domainBlockingEnabled` | Block malicious domains | `true` |
| `config.threatIntel.urlMatchingEnabled` | URL pattern matching | `true` |
| `config.threatIntel.dnsCacheEnabled` | DNS caching | `true` |

### TLS Interception

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.tlsInterception.enabled` | Enable TLS interception | `false` |
| `config.tlsInterception.mode` | Mode (mitm/preconfigured) | `mitm` |

### Network Policy

| Parameter | Description | Default |
|-----------|-------------|---------|
| `networkPolicy.enabled` | Enable NetworkPolicy | `true` |
| `networkPolicy.policyTypes` | Policy types | `["Egress"]` |

## Examples

### DaemonSet Mode (One Pod Per Node)

```yaml
deploymentMode: "daemonset"

daemonset:
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1

nodeSelector:
  node-role.kubernetes.io/worker: "true"
```

### L7 HTTP/3 (QUIC) Support

```yaml
config:
  l7:
    enabled: true
    envoyLogLevel: "info"
    http3Enabled: true  # EXPERIMENTAL

service:
  httpsPort: 10443  # QUIC port
```

### Threat Intelligence with Feed Integration

```yaml
config:
  threatIntel:
    ipBlockingEnabled: true
    domainBlockingEnabled: true
    urlMatchingEnabled: true
    dnsCacheEnabled: true
    feeds:
      - url: "https://feeds.threatintel.com/ips.json"
        type: "ip"
      - url: "https://feeds.threatintel.com/domains.json"
        type: "domain"
```

### TLS Interception (MITM Mode)

```yaml
config:
  tlsInterception:
    enabled: true
    mode: "mitm"
    caCertPath: "/app/certs/mitm-ca.crt"
    caKeyPath: "/app/certs/mitm-ca.key"

tls:
  createSecret: true
  mitmCaCert: "LS0tLS1CRUdJTi..."  # Base64-encoded CA cert
  mitmCaKey: "LS0tLS1CRUdJTi..."   # Base64-encoded CA key
```

### External Authorization

```yaml
config:
  extAuth:
    enabled: true
    port: 9002

# Deploy your own ext_authz gRPC service
# and point Envoy to localhost:9002
```

### Network Policy (Restricted Egress)

```yaml
networkPolicy:
  enabled: true
  policyTypes:
    - Egress
  egress:
    # Allow DNS
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
      ports:
        - protocol: UDP
          port: 53
    # Allow specific external IPs
    - to:
        - ipBlock:
            cidr: 52.1.2.3/32
      ports:
        - protocol: TCP
          port: 443
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
    requestsPerSecond: 25000
    burstSize: 50000

resources:
  requests:
    cpu: 1000m
    memory: 1Gi
  limits:
    cpu: 4000m
    memory: 4Gi

autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 50
```

## Monitoring

The chart exposes multiple metrics endpoints:

- **Go Proxy Metrics**: Port `8081` at `/metrics`
- **Envoy Admin**: Port `9901` at `/stats/prometheus`

### ServiceMonitor

```yaml
monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s
```

## Deployment Modes

### Deployment (Default)

Best for:
- Centralized egress control
- Dynamic scaling based on traffic
- Cost optimization

```yaml
deploymentMode: "deployment"
replicaCount: 3
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
```

### DaemonSet

Best for:
- Per-node egress control
- Consistent egress IP per node
- Network policy enforcement

```yaml
deploymentMode: "daemonset"
nodeSelector:
  node-role.kubernetes.io/worker: "true"
```

## Troubleshooting

### Envoy not starting

Check Envoy logs:

```bash
kubectl logs <pod-name> -c egress
kubectl port-forward <pod-name> 9901:9901
curl http://localhost:9901/config_dump
```

### TLS interception failing

Verify MITM CA certificate:

```bash
kubectl exec <pod-name> -- cat /app/certs/mitm-ca.crt
```

### Threat intelligence not blocking

Check threat feed URLs are accessible:

```bash
kubectl exec <pod-name> -- curl -I https://feeds.threatintel.com/ips.json
```

### Network policy blocking legitimate traffic

Review NetworkPolicy rules:

```bash
kubectl describe networkpolicy <networkpolicy-name>
```

## Uninstallation

```bash
helm uninstall my-egress
```

## Support

- GitHub: https://github.com/marchproxy/marchproxy
- Documentation: https://docs.marchproxy.io
- Email: support@marchproxy.io
