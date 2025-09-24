# Kubernetes Installation Guide

This guide covers deploying MarchProxy on Kubernetes using multiple methods: Helm charts, Kubernetes Operator, and manual YAML manifests.

## Prerequisites

- Kubernetes cluster 1.19+ with eBPF support
- kubectl configured to access your cluster
- Helm 3.x (for Helm installation method)
- At least 4 CPU cores and 8GB RAM available
- Network CNI that supports eBPF (Cilium recommended)

## Quick Start with Helm

### 1. Add MarchProxy Helm Repository

```bash
helm repo add marchproxy https://charts.marchproxy.io
helm repo update
```

### 2. Install with Default Configuration

```bash
# Community Edition (up to 3 proxies)
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace

# Enterprise Edition (requires license)
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace \
  --set enterprise.enabled=true \
  --set enterprise.licenseKey="PENG-XXXX-XXXX-XXXX-XXXX-ABCD"
```

### 3. Verify Installation

```bash
# Check pod status
kubectl get pods -n marchproxy

# Check services
kubectl get svc -n marchproxy

# View logs
kubectl logs -n marchproxy deployment/marchproxy-manager
kubectl logs -n marchproxy deployment/marchproxy-proxy
```

### 4. Access the Web Interface

```bash
# Port forward to access locally
kubectl port-forward -n marchproxy svc/marchproxy-manager 8000:8000

# Access at http://localhost:8000
# Default credentials: admin/changeme
```

## Production Helm Configuration

### values.yaml for Production

```yaml
# Production values.yaml
global:
  imageRegistry: "registry.marchproxy.io"
  imagePullSecrets:
    - name: marchproxy-registry

# Manager configuration
manager:
  replicaCount: 3
  image:
    tag: "v0.1.1"
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
        size: "100Gi"
        storageClass: "fast-ssd"

  # TLS configuration
  tls:
    enabled: true
    certificateSource: "letsencrypt"

  # High availability
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app.kubernetes.io/name: marchproxy-manager
          topologyKey: kubernetes.io/hostname

# Proxy configuration
proxy:
  replicaCount: 6
  image:
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
    enableSRIOV: false
    numaAffinity: true
    hugepages: true

  # Network configuration
  service:
    type: LoadBalancer
    annotations:
      service.beta.kubernetes.io/aws-load-balancer-type: "nlb"
      service.beta.kubernetes.io/aws-load-balancer-scheme: "internal"

# Enterprise features
enterprise:
  enabled: true
  licenseKey: "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"

  # Multi-cluster support
  clusters:
    - name: "production"
      region: "us-west-2"
    - name: "staging"
      region: "us-east-1"

  # Authentication
  authentication:
    saml:
      enabled: true
      metadataURL: "https://identity.company.com/saml/metadata"
    oauth2:
      enabled: true
      providers:
        - name: "google"
          clientId: "your-google-client-id"
          clientSecret: "your-google-client-secret"

# Monitoring
monitoring:
  prometheus:
    enabled: true
    serviceMonitor:
      enabled: true
  grafana:
    enabled: true
    dashboards:
      enabled: true
  jaeger:
    enabled: true

# Logging
logging:
  elasticsearch:
    enabled: true
    persistence:
      size: "500Gi"
  logstash:
    enabled: true
  kibana:
    enabled: true
```

### Install with Production Configuration

```bash
helm install marchproxy marchproxy/marchproxy \
  --namespace marchproxy \
  --create-namespace \
  --values values.yaml \
  --timeout 10m
```

## Kubernetes Operator Installation

### 1. Install the Operator

```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/marchproxy/marchproxy/main/operator/config/crd/marchproxy.yaml

# Install the operator
kubectl apply -f https://raw.githubusercontent.com/marchproxy/marchproxy/main/operator/config/manager/manager.yaml
```

### 2. Deploy MarchProxy Instance

Create a MarchProxy custom resource:

```yaml
# marchproxy-instance.yaml
apiVersion: proxy.marchproxy.io/v1alpha1
kind: MarchProxy
metadata:
  name: production-proxy
  namespace: marchproxy
spec:
  # Manager configuration
  manager:
    replicas: 3
    resources:
      requests:
        cpu: "1"
        memory: "2Gi"
      limits:
        cpu: "2"
        memory: "4Gi"

  # Proxy configuration
  proxy:
    replicas: 6
    resources:
      requests:
        cpu: "2"
        memory: "4Gi"
      limits:
        cpu: "4"
        memory: "8Gi"
    performance:
      enableEBPF: true
      enableXDP: true

  # Database configuration
  database:
    type: "postgresql"
    persistence:
      size: "100Gi"
      storageClass: "fast-ssd"

  # Enterprise configuration
  enterprise:
    enabled: true
    licenseKey: "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"

  # Monitoring
  monitoring:
    enabled: true
    prometheus: true
    grafana: true
```

```bash
kubectl apply -f marchproxy-instance.yaml
```

## Manual YAML Installation

For complete control, you can deploy using raw Kubernetes manifests:

### 1. Namespace and RBAC

```yaml
# 01-namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: marchproxy
  labels:
    name: marchproxy

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: marchproxy
  namespace: marchproxy

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: marchproxy
rules:
- apiGroups: [""]
  resources: ["pods", "services", "endpoints"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: marchproxy
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: marchproxy
subjects:
- kind: ServiceAccount
  name: marchproxy
  namespace: marchproxy
```

### 2. ConfigMap and Secrets

```yaml
# 02-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: marchproxy-config
  namespace: marchproxy
data:
  manager.yaml: |
    database:
      url: "postgresql://marchproxy:changeme@postgres:5432/marchproxy"
    license:
      server: "https://license.penguintech.io"
    auth:
      jwt_secret: "your-jwt-secret-here"
      session_secret: "your-session-secret-here"

  proxy.yaml: |
    manager:
      url: "http://marchproxy-manager:8000"
    performance:
      enable_ebpf: true
      enable_xdp: true

---
apiVersion: v1
kind: Secret
metadata:
  name: marchproxy-secrets
  namespace: marchproxy
type: Opaque
stringData:
  cluster-api-key: "your-cluster-api-key-here"
  database-password: "changeme"
  enterprise-license: "PENG-XXXX-XXXX-XXXX-XXXX-ABCD"
```

### 3. Database Deployment

```yaml
# 03-database.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres
  namespace: marchproxy
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_DB
          value: "marchproxy"
        - name: POSTGRES_USER
          value: "marchproxy"
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: marchproxy-secrets
              key: database-password
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: postgres-data
          mountPath: /var/lib/postgresql/data
      volumes:
      - name: postgres-data
        persistentVolumeClaim:
          claimName: postgres-pvc

---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: marchproxy
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 100Gi

---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: marchproxy
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
    targetPort: 5432
```

### 4. Manager Deployment

```yaml
# 04-manager.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: marchproxy-manager
  namespace: marchproxy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: marchproxy-manager
  template:
    metadata:
      labels:
        app: marchproxy-manager
    spec:
      serviceAccountName: marchproxy
      containers:
      - name: manager
        image: marchproxy/manager:v0.1.1
        ports:
        - containerPort: 8000
        env:
        - name: DATABASE_URL
          value: "postgresql://marchproxy:changeme@postgres:5432/marchproxy"
        - name: ENTERPRISE_LICENSE
          valueFrom:
            secretKeyRef:
              name: marchproxy-secrets
              key: enterprise-license
        volumeMounts:
        - name: config
          mountPath: /app/config
        resources:
          requests:
            cpu: "1"
            memory: "2Gi"
          limits:
            cpu: "2"
            memory: "4Gi"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: marchproxy-config

---
apiVersion: v1
kind: Service
metadata:
  name: marchproxy-manager
  namespace: marchproxy
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8000"
    prometheus.io/path: "/metrics"
spec:
  selector:
    app: marchproxy-manager
  ports:
  - name: http
    port: 8000
    targetPort: 8000
  - name: metrics
    port: 8001
    targetPort: 8001
  type: ClusterIP
```

### 5. Proxy Deployment

```yaml
# 05-proxy.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: marchproxy-proxy
  namespace: marchproxy
spec:
  replicas: 6
  selector:
    matchLabels:
      app: marchproxy-proxy
  template:
    metadata:
      labels:
        app: marchproxy-proxy
    spec:
      serviceAccountName: marchproxy
      securityContext:
        capabilities:
          add:
            - NET_ADMIN
            - SYS_ADMIN
      containers:
      - name: proxy
        image: marchproxy/proxy:v0.1.1
        ports:
        - containerPort: 8080
        - containerPort: 8081
        env:
        - name: CLUSTER_API_KEY
          valueFrom:
            secretKeyRef:
              name: marchproxy-secrets
              key: cluster-api-key
        - name: MANAGER_URL
          value: "http://marchproxy-manager:8000"
        - name: ENABLE_EBPF
          value: "true"
        - name: ENABLE_XDP
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /app/config
        - name: bpf-maps
          mountPath: /sys/fs/bpf
        resources:
          requests:
            cpu: "2"
            memory: "4Gi"
          limits:
            cpu: "4"
            memory: "8Gi"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8081
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: marchproxy-config
      - name: bpf-maps
        hostPath:
          path: /sys/fs/bpf
          type: DirectoryOrCreate

---
apiVersion: v1
kind: Service
metadata:
  name: marchproxy-proxy
  namespace: marchproxy
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8081"
    prometheus.io/path: "/metrics"
spec:
  selector:
    app: marchproxy-proxy
  ports:
  - name: proxy
    port: 8080
    targetPort: 8080
  - name: metrics
    port: 8081
    targetPort: 8081
  type: LoadBalancer
```

### 6. Deploy All Components

```bash
# Apply all manifests
kubectl apply -f 01-namespace.yaml
kubectl apply -f 02-config.yaml
kubectl apply -f 03-database.yaml
kubectl apply -f 04-manager.yaml
kubectl apply -f 05-proxy.yaml

# Wait for deployment
kubectl wait --for=condition=ready pod -l app=marchproxy-manager -n marchproxy --timeout=300s
kubectl wait --for=condition=ready pod -l app=marchproxy-proxy -n marchproxy --timeout=300s
```

## Post-Installation Configuration

### Access the Web Interface

```bash
# Get the manager service URL
kubectl get svc -n marchproxy marchproxy-manager

# For LoadBalancer:
export MANAGER_URL=$(kubectl get svc -n marchproxy marchproxy-manager -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
echo "Access MarchProxy at: http://$MANAGER_URL:8000"

# For port-forward:
kubectl port-forward -n marchproxy svc/marchproxy-manager 8000:8000
echo "Access MarchProxy at: http://localhost:8000"
```

### Initial Setup

1. **Login**: Use default credentials `admin/changeme`
2. **Enable 2FA**: Scan QR code with authenticator app
3. **Configure License**: Add your Enterprise license key
4. **Create Services**: Define your backend services
5. **Configure Mappings**: Set up traffic routing rules
6. **Monitor**: Check Grafana dashboards and logs

### Get Proxy Endpoint

```bash
# Get the proxy service URL
kubectl get svc -n marchproxy marchproxy-proxy

# Test proxy connectivity
export PROXY_URL=$(kubectl get svc -n marchproxy marchproxy-proxy -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
curl -x http://$PROXY_URL:8080 http://httpbin.org/ip
```

## Troubleshooting

### Common Issues

1. **eBPF not working**: Ensure kernel version 4.18+ and proper capabilities
2. **XDP not loading**: Check network interface supports XDP
3. **Database connection issues**: Verify PostgreSQL is running and accessible
4. **License validation fails**: Check internet connectivity to license.penguintech.io

### Debugging Commands

```bash
# Check pod logs
kubectl logs -n marchproxy deployment/marchproxy-manager -f
kubectl logs -n marchproxy deployment/marchproxy-proxy -f

# Check events
kubectl get events -n marchproxy --sort-by=.metadata.creationTimestamp

# Check configurations
kubectl describe configmap -n marchproxy marchproxy-config
kubectl describe secret -n marchproxy marchproxy-secrets

# Check eBPF programs (inside proxy pod)
kubectl exec -n marchproxy deployment/marchproxy-proxy -- bpftool prog list
kubectl exec -n marchproxy deployment/marchproxy-proxy -- bpftool map list
```

### Performance Tuning

```bash
# Check node capabilities
kubectl describe nodes | grep -A 10 -B 5 "cpu\|memory"

# Enable huge pages on nodes
kubectl patch node $NODE_NAME -p '{"spec":{"taints":[{"key":"node.kubernetes.io/memory","value":"hugepages-2Mi","effect":"NoSchedule"}]}}'

# Configure CPU affinity
kubectl patch deployment -n marchproxy marchproxy-proxy -p '{"spec":{"template":{"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"node.kubernetes.io/instance-type","operator":"In","values":["c5n.large","c5n.xlarge"]}]}]}}}}}}}'
```

## Scaling and High Availability

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: marchproxy-proxy-hpa
  namespace: marchproxy
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: marchproxy-proxy
  minReplicas: 3
  maxReplicas: 20
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
```

### Pod Disruption Budget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: marchproxy-proxy-pdb
  namespace: marchproxy
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: marchproxy-proxy
```

## Backup and Restore

### Database Backup

```bash
# Create backup job
kubectl create job -n marchproxy --from=cronjob/postgres-backup postgres-backup-manual

# Restore from backup
kubectl exec -n marchproxy deployment/postgres -- psql -U marchproxy -d marchproxy < backup.sql
```

This completes the comprehensive Kubernetes installation guide for MarchProxy.