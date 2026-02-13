# Kubernetes Deployment Guide for MarchProxy

Comprehensive guide for deploying MarchProxy on Kubernetes using kubectl, Helm charts, operators, and scaling strategies.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [kubectl Deployment](#kubectl-deployment)
4. [Helm Chart Installation](#helm-chart-installation)
5. [MarchProxy Operator](#marchproxy-operator)
6. [Scaling and High Availability](#scaling-and-high-availability)
7. [Configuration Management](#configuration-management)
8. [Monitoring and Observability](#monitoring-and-observability)
9. [Troubleshooting](#troubleshooting)
10. [Production Checklist](#production-checklist)

## Prerequisites

- Kubernetes cluster v1.24+ with eBPF support
- kubectl v1.24+ configured to access your cluster
- Helm 3.x (for Helm installation)
- Metrics Server installed (for HPA autoscaling)
- Docker images built and pushed to registry
- At least 3 nodes with 4+ CPU cores and 8GB RAM each
- Network CNI supporting eBPF (Cilium recommended)

Verify prerequisites:
```bash
kubectl cluster-info
kubectl version --client
helm version
kubectl get deployment metrics-server -n kube-system
```

## Quick Start

### Minimal Deployment (5 minutes)

```bash
# Create namespace
kubectl create namespace marchproxy

# Apply base configurations
kubectl apply -f k8s/unified/base/namespace.yaml
kubectl apply -f k8s/unified/base/configmap.yaml

# Create secrets from template
cp k8s/unified/base/secrets.yaml.example k8s/unified/base/secrets.yaml
# Edit secrets.yaml with your credentials
kubectl apply -f k8s/unified/base/secrets.yaml

# Deploy core NLB proxy
kubectl apply -f k8s/unified/nlb/

# Verify deployment
kubectl get pods -n marchproxy
kubectl get svc -n marchproxy
```

Access the NLB proxy:
```bash
kubectl get svc proxy-nlb -n marchproxy
# Use the EXTERNAL-IP for external traffic
```

## kubectl Deployment

Direct Kubernetes manifest deployment for maximum control.

### Directory Structure

```
k8s/unified/
├── base/
│   ├── namespace.yaml          # Namespace and resource quotas
│   ├── configmap.yaml          # Configuration for all modules
│   └── secrets.yaml.example    # Template for sensitive data
├── nlb/                        # Network Load Balancer (L3/L4)
│   ├── deployment.yaml
│   └── service.yaml
├── alb/                        # Application Load Balancer (L7)
│   ├── deployment.yaml
│   └── service.yaml
├── dblb/                       # Database Load Balancer
│   ├── deployment.yaml
│   └── service.yaml
├── ailb/                       # AI/LLM Load Balancer
│   ├── deployment.yaml
│   └── service.yaml
├── rtmp/                       # RTMP Video Transcoding
│   ├── deployment.yaml
│   └── service.yaml
└── hpa/                        # Horizontal Pod Autoscalers
    ├── nlb-hpa.yaml
    ├── alb-hpa.yaml
    ├── dblb-hpa.yaml
    ├── ailb-hpa.yaml
    └── rtmp-hpa.yaml
```

### Step-by-Step Deployment

**1. Set Up Secrets**

```bash
# Copy and customize the secrets template
cp k8s/unified/base/secrets.yaml.example k8s/unified/base/secrets.yaml

# Generate secure random values
CLUSTER_API_KEY=$(openssl rand -hex 32)
JWT_SECRET=$(openssl rand -base64 48)
POSTGRES_PASSWORD=$(openssl rand -base64 32)
REDIS_PASSWORD=$(openssl rand -base64 32)

# Update secrets.yaml with your values, then apply
kubectl apply -f k8s/unified/base/secrets.yaml
```

**2. Deploy Base Infrastructure**

```bash
# Create namespace and resource quotas
kubectl apply -f k8s/unified/base/namespace.yaml

# Apply global configuration
kubectl apply -f k8s/unified/base/configmap.yaml
```

**3. Deploy NLB (Entry Point)**

```bash
kubectl apply -f k8s/unified/nlb/deployment.yaml
kubectl apply -f k8s/unified/nlb/service.yaml

# Wait for NLB to be ready
kubectl rollout status deployment/proxy-nlb -n marchproxy
```

**4. Deploy Additional Modules**

```bash
# HTTP/HTTPS Proxy (L7)
kubectl apply -f k8s/unified/alb/

# Database Proxy (optional)
kubectl apply -f k8s/unified/dblb/

# AI/LLM Proxy (optional)
kubectl apply -f k8s/unified/ailb/

# RTMP Transcoding (optional)
kubectl apply -f k8s/unified/rtmp/
```

**5. Enable Auto-Scaling**

```bash
# Requires Metrics Server
kubectl apply -f k8s/unified/hpa/

# Verify HPA configuration
kubectl get hpa -n marchproxy
```

### Complete One-Command Deployment

```bash
# Deploy all manifests recursively
kubectl apply -R -f k8s/unified/

# Watch deployment progress
kubectl get pods -n marchproxy -w
```

## Helm Chart Installation

Helm charts provide templating, version management, and simplified configuration.

### Add MarchProxy Helm Repository

```bash
helm repo add marchproxy https://charts.marchproxy.io
helm repo update

# Search for available charts
helm search repo marchproxy
```

### Community Edition Installation

```bash
# Minimal deployment with defaults
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace

# Verify installation
helm list -n marchproxy
kubectl get pods -n marchproxy
```

### Enterprise Edition Installation

```bash
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace \
  --set enterprise.enabled=true \
  --set enterprise.licenseKey="PENG-XXXX-XXXX-XXXX-XXXX-ABCD" \
  --set enterprise.multiCluster.enabled=true \
  --set enterprise.authentication.saml.enabled=true \
  --set enterprise.authentication.oauth2.enabled=true
```

### Production Configuration with values.yaml

Create `production-values.yaml`:

```yaml
# Global settings
global:
  imageRegistry: "registry.marchproxy.io"
  imagePullSecrets:
    - name: registry-credentials
  environment: production

# Manager (Control Plane)
manager:
  replicaCount: 3
  image:
    repository: marchproxy/manager
    tag: "v0.1.1"
    pullPolicy: IfNotPresent

  resources:
    requests:
      cpu: "1"
      memory: "2Gi"
    limits:
      cpu: "2"
      memory: "4Gi"

  # Database configuration
  postgresql:
    enabled: true
    primary:
      persistence:
        enabled: true
        size: "100Gi"
        storageClass: "fast-ssd"
    primary_replica_count: 2

  # High availability
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: marchproxy-manager
          topologyKey: kubernetes.io/hostname

# Proxy Layer (Data Plane)
proxy:
  replicaCount: 6
  image:
    repository: marchproxy/proxy
    tag: "v0.1.1"

  resources:
    requests:
      cpu: "2"
      memory: "4Gi"
    limits:
      cpu: "4"
      memory: "8Gi"

  # Performance optimizations
  performance:
    enableEBPF: true
    enableXDP: true
    enableDPDK: false
    numaAffinity: true
    hugepages: true
    cpuPinning: true

  # Network configuration
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"

# Auto-scaling configuration
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 20
  targetCPUUtilization: 70
  targetMemoryUtilization: 80

# Enterprise features
enterprise:
  enabled: true
  licenseKey: "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"

  # Multi-cluster support
  clusters:
    - name: "production"
      region: "us-west-2"
      apiKey: "generated-by-manager"
    - name: "staging"
      region: "us-east-1"
      apiKey: "generated-by-manager"

  # Authentication methods
  authentication:
    saml:
      enabled: true
      metadataURL: "https://identity.company.com/saml/metadata"
      entityID: "https://marchproxy.company.com"
    oauth2:
      enabled: true
      providers:
        - name: google
          clientID: "your-client-id"
          clientSecret: "your-client-secret"
        - name: azure
          clientID: "your-client-id"
          clientSecret: "your-client-secret"
    scim:
      enabled: true
      endpoint: "https://identity.company.com/scim/v2"

# Monitoring and observability
monitoring:
  prometheus:
    enabled: true
    retention: 30d
    serviceMonitor:
      enabled: true
      interval: 30s

  grafana:
    enabled: true
    adminPassword: "secure-password"
    dashboards:
      enabled: true

  jaeger:
    enabled: true
    samplerType: probabilistic
    samplerParam: 0.1

# Logging configuration
logging:
  syslog:
    enabled: true
    host: "syslog.company.com"
    port: 514
    protocol: udp

  elasticsearch:
    enabled: true
    host: "elasticsearch.company.com"
    port: 9200
    persistence:
      enabled: true
      size: "500Gi"

# TLS/SSL configuration
tls:
  enabled: true
  certificateSource: letsencrypt
  issuer: letsencrypt-prod
  email: admin@company.com
```

Install with production configuration:

```bash
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace \
  --values production-values.yaml \
  --timeout 15m \
  --wait
```

### Helm Upgrade and Rollback

```bash
# Upgrade to new version
helm upgrade marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --values production-values.yaml \
  --timeout 10m \
  --wait

# View release history
helm history marchproxy -n marchproxy

# Rollback to previous release
helm rollback marchproxy 1 -n marchproxy

# Uninstall
helm uninstall marchproxy -n marchproxy
```

## MarchProxy Operator

The Kubernetes Operator provides declarative management and advanced features.

### Install Operator

```bash
# Install CRD (Custom Resource Definition)
kubectl apply -f https://raw.githubusercontent.com/marchproxy/marchproxy/main/operator/config/crd/marchproxy.yaml

# Install operator controller
kubectl apply -f https://raw.githubusercontent.com/marchproxy/marchproxy/main/operator/config/manager/manager.yaml

# Verify operator is running
kubectl get deployment -n marchproxy-system
kubectl logs -f deployment/marchproxy-operator -n marchproxy-system
```

### Deploy MarchProxy Instance via Operator

Create `marchproxy-instance.yaml`:

```yaml
apiVersion: proxy.marchproxy.io/v1alpha1
kind: MarchProxy
metadata:
  name: production-deployment
  namespace: marchproxy
spec:
  # Version management
  version: "v0.1.1"
  updateStrategy: Rolling

  # Manager configuration
  manager:
    replicas: 3
    image: marchproxy/manager:v0.1.1
    resources:
      requests:
        cpu: "1"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "4Gi"

    # Database
    database:
      type: postgresql
      host: postgres.marchproxy.svc.cluster.local
      port: 5432
      name: marchproxy
      credentialsSecret: db-credentials
      persistence:
        enabled: true
        size: 100Gi
        storageClass: fast-ssd

    # TLS
    tls:
      enabled: true
      certificateSecret: manager-tls

    # Service configuration
    service:
      type: ClusterIP
      port: 8000

  # Proxy configuration
  proxy:
    replicas: 6
    minReplicas: 3
    maxReplicas: 20
    image: marchproxy/proxy:v0.1.1
    resources:
      requests:
        cpu: "2"
        memory: "4Gi"
      limits:
        cpu: "4"
        memory: "8Gi"

    # Performance optimization
    performance:
      enableEBPF: true
      enableXDP: true
      enableDPDK: false
      numaAffinity: true
      hugepages: true

    # Service configuration
    service:
      type: LoadBalancer
      port: 8080
      targetPort: 8080

    # Auto-scaling
    autoscaling:
      enabled: true
      metrics:
        - type: cpu
          target: 70
        - type: memory
          target: 80

  # Enterprise configuration
  enterprise:
    enabled: true
    licenseKey: PENG-XXXX-XXXX-XXXX-XXXX-ABCD

    clusters:
      - name: production
        region: us-west-2
      - name: staging
        region: us-east-1

    authentication:
      saml:
        enabled: true
        metadataURL: https://identity.company.com/saml/metadata
      oauth2:
        enabled: true

  # Monitoring configuration
  monitoring:
    enabled: true
    prometheus:
      enabled: true
      retention: 30d
    grafana:
      enabled: true
    jaeger:
      enabled: true

  # Logging configuration
  logging:
    syslog:
      enabled: true
      host: syslog.company.com
      port: 514
```

Deploy the instance:

```bash
kubectl apply -f marchproxy-instance.yaml

# Monitor deployment
kubectl describe marchproxy production-deployment -n marchproxy
kubectl logs -f deployment/marchproxy-manager -n marchproxy
```

## Scaling and High Availability

### Horizontal Pod Autoscaler (HPA)

Auto-scale based on CPU and memory usage:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: proxy-nlb-hpa
  namespace: marchproxy
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: proxy-nlb
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 15
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
        - type: Percent
          value: 100
          periodSeconds: 30
```

### Manual Scaling

```bash
# Scale NLB to 5 replicas
kubectl scale deployment proxy-nlb --replicas=5 -n marchproxy

# Scale ALB to 10 replicas
kubectl scale deployment proxy-alb --replicas=10 -n marchproxy

# Verify scaling
kubectl get deployment -n marchproxy
```

### Pod Disruption Budget (PDB)

Ensure minimum availability during disruptions:

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: proxy-nlb-pdb
  namespace: marchproxy
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app.kubernetes.io/component: proxy-nlb
```

### Multi-Zone Deployment

Spread pods across availability zones:

```yaml
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            app: proxy-nlb
        topologyKey: topology.kubernetes.io/zone
```

## Configuration Management

### ConfigMap Operations

```bash
# View current configuration
kubectl get configmap marchproxy-config -n marchproxy -o yaml

# Edit configuration
kubectl edit configmap marchproxy-config -n marchproxy

# Update from file
kubectl create configmap marchproxy-config --from-file=config/ -n marchproxy --dry-run=client -o yaml | kubectl apply -f -

# Restart pods to apply changes
kubectl rollout restart deployment/proxy-nlb -n marchproxy
```

### Secret Management

```bash
# Create secrets from literals
kubectl create secret generic marchproxy-secrets \
  --from-literal=cluster-api-key="your-key" \
  --from-literal=jwt-secret="your-secret" \
  -n marchproxy \
  --dry-run=client -o yaml | kubectl apply -f -

# Update specific secret values
kubectl patch secret marchproxy-secrets -n marchproxy -p '{"data":{"cluster-api-key":"'$(echo -n "new-key" | base64)'"}}'

# Use external secret manager (recommended)
# Deploy ESO (External Secrets Operator)
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets-system --create-namespace
```

### Environment Variables

Key environment variables for MarchProxy services:

```bash
# NLB (Network Load Balancer)
CLUSTER_API_KEY=your-cluster-key
MANAGER_URL=http://marchproxy-manager:8000
ENABLE_EBPF=true
ENABLE_XDP=true
LOG_LEVEL=info

# ALB (Application Load Balancer)
ENVOY_LOG_LEVEL=info
MAX_CONNECTIONS=10000
ENABLE_TRACING=true

# Manager
DATABASE_URL=postgresql://user:pass@postgres:5432/marchproxy
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
JWT_SECRET=your-jwt-secret
SESSION_SECRET=your-session-secret
```

## Monitoring and Observability

### Prometheus Integration

```bash
# Apply ServiceMonitor for metrics scraping
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: marchproxy-metrics
  namespace: marchproxy
spec:
  selector:
    matchLabels:
      app.kubernetes.io/part-of: marchproxy
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
EOF

# View metrics in Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Access http://localhost:9090
```

### Grafana Dashboards

```bash
# Deploy Grafana
helm repo add grafana https://grafana.github.io/helm-charts
helm install grafana grafana/grafana -n monitoring --create-namespace

# Port forward to access
kubectl port-forward -n monitoring svc/grafana 3000:80

# Import MarchProxy dashboards from dashboard.grafana.com
```

### Centralized Logging

```bash
# Deploy ELK Stack
helm repo add elastic https://helm.elastic.co
helm install elasticsearch elastic/elasticsearch -n logging --create-namespace
helm install logstash elastic/logstash -n logging
helm install kibana elastic/kibana -n logging

# View logs in Kibana
kubectl port-forward -n logging svc/kibana 5601:5601
```

### Trace Collection with Jaeger

```bash
# Deploy Jaeger
kubectl apply -f https://raw.githubusercontent.com/jaegertracing/jaeger-kubernetes/main/jaeger-production-template.yml

# Port forward to Jaeger UI
kubectl port-forward -n jaeger svc/jaeger-query 16686:16686
```

## Troubleshooting

### Pod Status Checks

```bash
# Check pod status
kubectl get pods -n marchproxy

# Describe problematic pod
kubectl describe pod <pod-name> -n marchproxy

# View pod logs
kubectl logs -f <pod-name> -n marchproxy

# Check previous logs (if pod crashed)
kubectl logs --previous <pod-name> -n marchproxy
```

### Common Issues and Solutions

| Issue | Cause | Solution |
|-------|-------|----------|
| `ImagePullBackOff` | Image not found in registry | Verify image path, registry credentials, and tag |
| `CrashLoopBackOff` | Container exiting immediately | Check logs, verify configuration, check resources |
| `Pending` | Insufficient resources | Scale cluster, adjust resource requests, check quotas |
| `NotReady` | Liveness/readiness probe failing | Check service health endpoints, verify configuration |
| `Service has no endpoints` | No ready pods behind service | Wait for pods to start, check pod status |

### Network Debugging

```bash
# Test DNS resolution
kubectl exec -it <pod-name> -n marchproxy -- nslookup proxy-alb

# Test connectivity between pods
kubectl exec -it <pod-name> -n marchproxy -- curl http://proxy-alb:50051

# Check service endpoints
kubectl get endpoints -n marchproxy

# Verify network policies
kubectl get networkpolicies -n marchproxy
kubectl describe networkpolicy <policy-name> -n marchproxy
```

### Performance Issues

```bash
# Check resource usage
kubectl top pods -n marchproxy
kubectl top nodes

# Check HPA status
kubectl describe hpa -n marchproxy

# View HPA events
kubectl get events -n marchproxy --sort-by=.metadata.creationTimestamp

# Check for throttling
kubectl get pods -n marchproxy -o json | jq '.items[].status.containerStatuses[]'
```

### Database Connectivity

```bash
# Check PostgreSQL service
kubectl get svc postgres -n marchproxy

# Test database connection from pod
kubectl exec -it <manager-pod> -n marchproxy -- psql -h postgres -U marchproxy -d marchproxy

# Check database logs
kubectl logs -f deployment/postgres -n marchproxy
```

## Production Checklist

Before deploying to production:

- [ ] **Resources**: Cluster has minimum 3 nodes with 4+ CPU cores and 8GB RAM each
- [ ] **Networking**: eBPF-capable CNI (Cilium) installed
- [ ] **Storage**: Persistent volumes provisioned for databases
- [ ] **Secrets**: All sensitive data in Kubernetes secrets, not ConfigMaps
- [ ] **RBAC**: Service accounts with least privilege permissions
- [ ] **Image Registry**: Private registry configured with credentials
- [ ] **Monitoring**: Prometheus and Grafana deployed and configured
- [ ] **Logging**: Centralized logging (ELK/EFK) configured
- [ ] **Backups**: Database backup strategy implemented
- [ ] **High Availability**: Multiple replicas of critical components
- [ ] **HPA**: Auto-scaling configured based on metrics
- [ ] **PDB**: Pod Disruption Budgets for critical workloads
- [ ] **Network Policies**: Restrict inter-pod communication as needed
- [ ] **TLS**: Certificate management configured (Let's Encrypt)
- [ ] **License**: Enterprise license key configured and validated
- [ ] **Resource Quotas**: Namespace quotas enforced
- [ ] **Limits**: Memory and CPU limits set on all containers
- [ ] **Health Checks**: Liveness and readiness probes configured
- [ ] **Load Balancer**: Type and annotations configured for cloud provider
- [ ] **Security Context**: Pod and container security contexts configured
- [ ] **Ingress**: Kubernetes Ingress configured for manager access
- [ ] **DNS**: Proper DNS configuration for service discovery
- [ ] **Testing**: Load testing completed and validated
- [ ] **Documentation**: Runbooks and procedures documented
- [ ] **Alerting**: Prometheus alerts configured and tested
- [ ] **Disaster Recovery**: Recovery procedures tested
- [ ] **Performance Tuning**: Kernel parameters optimized for eBPF

### Resource Requirements Summary

| Component | Min Replicas | CPU Request | Memory Request | Storage |
|-----------|--------------|-------------|----------------|---------|
| Manager | 3 | 1 core | 2 GB | 100 GB |
| NLB | 2 | 1 core | 1 GB | N/A |
| ALB | 3 | 500m | 512 MB | N/A |
| DBLB | 2 | 1 core | 1 GB | N/A |
| AILB | 2 | 500m | 1 GB | N/A |
| RTMP | 1 | 2 cores | 2 GB | N/A |
| PostgreSQL | 1 | 500m | 1 GB | 100 GB |
| **Total** | **14** | **7.5 cores** | **10.5 GB** | **200 GB** |

---

For detailed information on specific components, see:
- [Installation Guide](/docs/installation.md)
- [Configuration Reference](/docs/configuration.md)
- [Operations Guide](/docs/operations/monitoring.md)
- [Architecture Overview](/docs/architecture.md)
