# Monitoring and Observability

This guide covers comprehensive monitoring and observability setup for MarchProxy in production environments.

## Overview

MarchProxy provides extensive monitoring capabilities through:

- **Prometheus metrics**: Detailed performance and operational metrics
- **Grafana dashboards**: Pre-built visualization and alerting
- **Health checks**: Service health monitoring and automated recovery
- **Distributed tracing**: Request flow tracking across components
- **Centralized logging**: Structured logging with ELK stack integration
- **Alert management**: Intelligent alerting with AlertManager

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   MarchProxy    │    │   MarchProxy    │    │   External      │
│    Manager      │    │    Proxies      │    │   Services      │
│                 │    │                 │    │                 │
│ • HTTP metrics  │    │ • HTTP metrics  │    │ • HTTP metrics  │
│ • Health checks │    │ • Health checks │    │ • Health checks │
│ • Logs          │    │ • Logs          │    │ • Logs          │
│ • Traces        │    │ • Traces        │    │ • Traces        │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Observability Stack                         │
│                                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ Prometheus  │  │   Grafana   │  │    Jaeger   │             │
│  │             │  │             │  │             │             │
│  │ • Metrics   │  │ • Dashboards│  │ • Tracing   │             │
│  │ • Alerting  │  │ • Alerting  │  │ • Debugging │             │
│  │ • Storage   │  │ • Reporting │  │ • Analysis  │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                 ELK Stack                               │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │   │
│  │  │Elasticsearch│ │ Logstash │ │  Kibana  │ │ Filebeat│ │   │
│  │  │             │ │          │ │          │ │         │ │   │
│  │  │ • Storage   │ │ • Process│ │ • Visualize│ • Collect│ │   │
│  │  │ • Search    │ │ • Filter │ │ • Search │ │ • Ship  │ │   │
│  │  │ • Analysis  │ │ • Enrich │ │ • Alert  │ │ • Monitor│ │   │
│  │  └──────────┘  └──────────┘  └──────────┘  └─────────┘ │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────┐                                               │
│  │AlertManager │                                               │
│  │             │                                               │
│  │ • Routing   │                                               │
│  │ • Grouping  │                                               │
│  │ • Silencing │                                               │
│  │ • Integration│                                              │
│  └─────────────┘                                               │
└─────────────────────────────────────────────────────────────────┘
```

## Prometheus Metrics

### Manager Metrics

The MarchProxy Manager exposes metrics on port 8001 (`/metrics`):

```bash
# Access manager metrics
curl http://localhost:8001/metrics
```

**Core Metrics:**

```prometheus
# User and authentication metrics
marchproxy_users_total{type="active"} 145
marchproxy_users_total{type="inactive"} 23
marchproxy_login_attempts_total{status="success"} 1250
marchproxy_login_attempts_total{status="failure"} 45
marchproxy_auth_tokens_active 89

# Service and mapping metrics
marchproxy_services_total{cluster="production"} 67
marchproxy_services_total{cluster="staging"} 23
marchproxy_mappings_total{cluster="production"} 134
marchproxy_mappings_total{cluster="staging"} 45

# API metrics
marchproxy_api_requests_total{method="GET",endpoint="/api/services",status="200"} 15234
marchproxy_api_requests_total{method="POST",endpoint="/api/services",status="201"} 567
marchproxy_api_request_duration_seconds{method="GET",endpoint="/api/services",quantile="0.5"} 0.023
marchproxy_api_request_duration_seconds{method="GET",endpoint="/api/services",quantile="0.95"} 0.156

# Database metrics
marchproxy_database_connections_active 15
marchproxy_database_connections_idle 5
marchproxy_database_query_duration_seconds{operation="select",quantile="0.5"} 0.012
marchproxy_database_query_duration_seconds{operation="insert",quantile="0.95"} 0.045

# License metrics
marchproxy_license_valid{license_type="enterprise"} 1
marchproxy_license_expires_in_days 245
marchproxy_license_features_enabled{feature="multi_cluster"} 1
marchproxy_license_features_enabled{feature="saml_auth"} 1

# Cluster metrics (Enterprise)
marchproxy_clusters_total 3
marchproxy_cluster_proxies{cluster="production"} 8
marchproxy_cluster_proxies{cluster="staging"} 3
marchproxy_cluster_api_key_rotations_total{cluster="production"} 12
```

### Proxy Metrics

Each MarchProxy Proxy exposes metrics on port 8081 (`/metrics`):

```bash
# Access proxy metrics
curl http://localhost:8081/metrics
```

**Core Metrics:**

```prometheus
# Connection metrics
marchproxy_proxy_connections_active{cluster="production"} 1245
marchproxy_proxy_connections_total{cluster="production"} 45789
marchproxy_proxy_connection_duration_seconds{quantile="0.5"} 45.2
marchproxy_proxy_connection_duration_seconds{quantile="0.95"} 120.8

# Request metrics
marchproxy_proxy_requests_total{method="GET",protocol="http",status="200"} 78456
marchproxy_proxy_requests_total{method="POST",protocol="http",status="201"} 3421
marchproxy_proxy_request_duration_seconds{method="GET",quantile="0.5"} 0.045
marchproxy_proxy_request_duration_seconds{method="GET",quantile="0.95"} 0.234

# Throughput metrics
marchproxy_proxy_bytes_sent_total{protocol="http"} 123456789
marchproxy_proxy_bytes_received_total{protocol="http"} 67891234
marchproxy_proxy_throughput_bytes_per_second{direction="sent"} 15678
marchproxy_proxy_throughput_bytes_per_second{direction="received"} 8934

# eBPF metrics
marchproxy_ebpf_programs_loaded 3
marchproxy_ebpf_map_entries{map="services"} 67
marchproxy_ebpf_map_entries{map="mappings"} 134
marchproxy_ebpf_packets_processed_total{program="filter"} 567890123
marchproxy_ebpf_packets_dropped_total{program="filter"} 12345

# XDP metrics (Enterprise)
marchproxy_xdp_programs_attached{interface="eth0"} 1
marchproxy_xdp_packets_processed_total{interface="eth0"} 789123456
marchproxy_xdp_packets_dropped_total{interface="eth0",reason="rate_limit"} 23456
marchproxy_xdp_rate_limit_hits_total{interface="eth0"} 5678

# Performance metrics
marchproxy_cpu_usage_percent 67.5
marchproxy_memory_usage_bytes 1073741824
marchproxy_memory_usage_percent 42.3
marchproxy_goroutines_active 156

# Authentication metrics
marchproxy_auth_requests_total{type="jwt",status="success"} 12345
marchproxy_auth_requests_total{type="jwt",status="failure"} 89
marchproxy_auth_cache_hits_total 9876
marchproxy_auth_cache_misses_total 234

# Circuit breaker metrics
marchproxy_circuit_breaker_state{service="backend-api"} 0  # 0=closed, 1=open, 2=half-open
marchproxy_circuit_breaker_requests_total{service="backend-api",state="success"} 5678
marchproxy_circuit_breaker_requests_total{service="backend-api",state="failure"} 23
```

### System Metrics

Additionally, install node_exporter for system-level metrics:

```bash
# Install node_exporter
wget https://github.com/prometheus/node_exporter/releases/latest/download/node_exporter-1.6.1.linux-amd64.tar.gz
tar -xzf node_exporter-1.6.1.linux-amd64.tar.gz
sudo mv node_exporter-1.6.1.linux-amd64/node_exporter /usr/local/bin/

# Create systemd service
sudo tee /etc/systemd/system/node_exporter.service <<EOF
[Unit]
Description=Node Exporter
After=network.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl enable node_exporter
sudo systemctl start node_exporter
```

## Prometheus Configuration

### Basic Prometheus Setup

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "marchproxy_rules.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # MarchProxy Manager
  - job_name: 'marchproxy-manager'
    static_configs:
      - targets: ['manager:8001']
    metrics_path: /metrics
    scrape_interval: 10s
    scrape_timeout: 5s

  # MarchProxy Proxies
  - job_name: 'marchproxy-proxy'
    static_configs:
      - targets:
          - 'proxy-1:8081'
          - 'proxy-2:8081'
          - 'proxy-3:8081'
    metrics_path: /metrics
    scrape_interval: 5s
    scrape_timeout: 3s

  # System metrics
  - job_name: 'node-exporter'
    static_configs:
      - targets:
          - 'manager:9100'
          - 'proxy-1:9100'
          - 'proxy-2:9100'
          - 'proxy-3:9100'

  # PostgreSQL metrics
  - job_name: 'postgres-exporter'
    static_configs:
      - targets: ['postgres:9187']

  # Additional infrastructure
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
```

### Service Discovery

For dynamic environments, use service discovery:

```yaml
# Kubernetes service discovery
scrape_configs:
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__

# Docker service discovery
  - job_name: 'docker-containers'
    dockerswarm_sd_configs:
      - host: unix:///var/run/docker.sock
        role: tasks
    relabel_configs:
      - source_labels: [__meta_dockerswarm_service_label_prometheus_job]
        target_label: job
```

## Grafana Dashboards

### Pre-built Dashboards

MarchProxy includes pre-built Grafana dashboards:

#### 1. MarchProxy Overview Dashboard

```json
{
  "dashboard": {
    "id": null,
    "title": "MarchProxy Overview",
    "tags": ["marchproxy"],
    "timezone": "browser",
    "panels": [
      {
        "title": "System Status",
        "type": "stat",
        "targets": [
          {
            "expr": "up{job=~\"marchproxy-.*\"}",
            "legendFormat": "{{instance}}"
          }
        ]
      },
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(marchproxy_proxy_requests_total[5m])",
            "legendFormat": "{{instance}}"
          }
        ]
      },
      {
        "title": "Active Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "marchproxy_proxy_connections_active",
            "legendFormat": "{{instance}}"
          }
        ]
      },
      {
        "title": "Response Times",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(marchproxy_proxy_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.5, rate(marchproxy_proxy_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      }
    ]
  }
}
```

#### 2. Performance Dashboard

Focus on detailed performance metrics:

```json
{
  "dashboard": {
    "title": "MarchProxy Performance",
    "panels": [
      {
        "title": "CPU Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "marchproxy_cpu_usage_percent",
            "legendFormat": "{{instance}}"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "marchproxy_memory_usage_percent",
            "legendFormat": "{{instance}}"
          }
        ]
      },
      {
        "title": "Network Throughput",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(marchproxy_proxy_bytes_sent_total[5m])",
            "legendFormat": "Sent - {{instance}}"
          },
          {
            "expr": "rate(marchproxy_proxy_bytes_received_total[5m])",
            "legendFormat": "Received - {{instance}}"
          }
        ]
      },
      {
        "title": "eBPF Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(marchproxy_ebpf_packets_processed_total[5m])",
            "legendFormat": "Processed - {{program}}"
          },
          {
            "expr": "rate(marchproxy_ebpf_packets_dropped_total[5m])",
            "legendFormat": "Dropped - {{program}}"
          }
        ]
      }
    ]
  }
}
```

#### 3. Security Dashboard

Monitor security-related metrics:

```json
{
  "dashboard": {
    "title": "MarchProxy Security",
    "panels": [
      {
        "title": "Authentication Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(marchproxy_auth_requests_total{status=\"success\"}[5m]) / rate(marchproxy_auth_requests_total[5m]) * 100"
          }
        ]
      },
      {
        "title": "Failed Login Attempts",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(marchproxy_login_attempts_total{status=\"failure\"}[5m])",
            "legendFormat": "Failed logins"
          }
        ]
      },
      {
        "title": "Rate Limiting",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(marchproxy_xdp_packets_dropped_total{reason=\"rate_limit\"}[5m])",
            "legendFormat": "Rate limited - {{interface}}"
          }
        ]
      },
      {
        "title": "WAF Blocks",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(marchproxy_waf_requests_blocked_total[5m])",
            "legendFormat": "{{rule_type}}"
          }
        ]
      }
    ]
  }
}
```

### Dashboard Provisioning

Automatically provision dashboards:

```yaml
# grafana/provisioning/dashboards/dashboards.yml
apiVersion: 1

providers:
  - name: 'marchproxy'
    orgId: 1
    folder: 'MarchProxy'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards/marchproxy
```

```yaml
# grafana/provisioning/datasources/prometheus.yml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
```

## Health Checks

### Manager Health Checks

The Manager provides comprehensive health checks:

```bash
# Basic health check
curl http://localhost:8000/healthz

# Detailed health check with component status
curl http://localhost:8000/healthz?detailed=true
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T11:00:00Z",
  "checks": {
    "database": {
      "status": "healthy",
      "response_time": "5ms",
      "last_check": "2024-01-15T11:00:00Z"
    },
    "license_server": {
      "status": "healthy",
      "response_time": "120ms",
      "last_check": "2024-01-15T10:59:55Z"
    },
    "certificate_expiry": {
      "status": "warning",
      "message": "Certificate expires in 25 days",
      "certificates": [
        {
          "name": "wildcard-company-com",
          "expires_at": "2024-02-09T00:00:00Z",
          "days_remaining": 25
        }
      ]
    },
    "disk_space": {
      "status": "healthy",
      "usage": "45%",
      "available": "500GB"
    },
    "memory_usage": {
      "status": "healthy",
      "usage": "68%",
      "available": "2.5GB"
    }
  },
  "version": "v0.1.1",
  "uptime": 86400
}
```

### Proxy Health Checks

Each proxy provides health status:

```bash
# Basic health check
curl http://localhost:8081/healthz

# Detailed health with eBPF status
curl http://localhost:8081/healthz?detailed=true
```

**Response:**

```json
{
  "status": "healthy",
  "timestamp": "2024-01-15T11:00:00Z",
  "checks": {
    "manager_connectivity": {
      "status": "healthy",
      "last_sync": "2024-01-15T10:59:45Z",
      "config_version": "v1.2.3"
    },
    "ebpf_programs": {
      "status": "healthy",
      "programs_loaded": 3,
      "programs": [
        {
          "name": "packet_filter",
          "status": "loaded",
          "id": 123
        },
        {
          "name": "connection_tracker",
          "status": "loaded",
          "id": 124
        },
        {
          "name": "rate_limiter",
          "status": "loaded",
          "id": 125
        }
      ]
    },
    "network_interfaces": {
      "status": "healthy",
      "interfaces": [
        {
          "name": "eth0",
          "status": "up",
          "xdp_attached": true
        }
      ]
    },
    "performance": {
      "cpu_usage": 45.2,
      "memory_usage": 68.5,
      "active_connections": 1245,
      "requests_per_second": 850
    }
  },
  "cluster_id": 1,
  "proxy_id": "proxy_123456789",
  "version": "v0.1.1"
}
```

### Kubernetes Health Checks

Configure Kubernetes probes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: marchproxy-manager
spec:
  template:
    spec:
      containers:
      - name: manager
        image: marchproxy/manager:v0.1.1
        ports:
        - containerPort: 8000
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 1
        startupProbe:
          httpGet:
            path: /healthz
            port: 8000
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 30
```

## Alerting

### Prometheus Alerting Rules

```yaml
# marchproxy_rules.yml
groups:
  - name: marchproxy.rules
    rules:
      # Manager alerts
      - alert: MarchProxyManagerDown
        expr: up{job="marchproxy-manager"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MarchProxy Manager is down"
          description: "Manager instance {{ $labels.instance }} has been down for more than 1 minute."

      - alert: MarchProxyHighErrorRate
        expr: rate(marchproxy_api_requests_total{status=~"5.."}[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate in MarchProxy Manager"
          description: "Manager {{ $labels.instance }} has error rate above 10% for 2 minutes."

      - alert: MarchProxyDatabaseConnectionFailed
        expr: marchproxy_database_connections_active == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Database connection lost"
          description: "Manager {{ $labels.instance }} has lost database connectivity."

      # Proxy alerts
      - alert: MarchProxyProxyDown
        expr: up{job="marchproxy-proxy"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "MarchProxy Proxy is down"
          description: "Proxy instance {{ $labels.instance }} has been down for more than 1 minute."

      - alert: MarchProxyHighLatency
        expr: histogram_quantile(0.95, rate(marchproxy_proxy_request_duration_seconds_bucket[5m])) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency in MarchProxy Proxy"
          description: "Proxy {{ $labels.instance }} has 95th percentile latency above 1 second."

      - alert: MarchProxyHighConnectionCount
        expr: marchproxy_proxy_connections_active > 5000
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High connection count"
          description: "Proxy {{ $labels.instance }} has more than 5000 active connections."

      # Performance alerts
      - alert: MarchProxyHighCPU
        expr: marchproxy_cpu_usage_percent > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High CPU usage"
          description: "Instance {{ $labels.instance }} has CPU usage above 80% for 5 minutes."

      - alert: MarchProxyHighMemory
        expr: marchproxy_memory_usage_percent > 90
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "High memory usage"
          description: "Instance {{ $labels.instance }} has memory usage above 90%."

      # Security alerts
      - alert: MarchProxyHighFailedLogins
        expr: rate(marchproxy_login_attempts_total{status="failure"}[5m]) > 10
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "High number of failed login attempts"
          description: "More than 10 failed login attempts per minute detected."

      - alert: MarchProxyRateLimitTriggered
        expr: rate(marchproxy_xdp_packets_dropped_total{reason="rate_limit"}[5m]) > 1000
        for: 30s
        labels:
          severity: warning
        annotations:
          summary: "Rate limiting triggered"
          description: "High rate of packets being dropped due to rate limiting on {{ $labels.interface }}."

      # License alerts
      - alert: MarchProxyLicenseExpiring
        expr: marchproxy_license_expires_in_days < 30
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "License expiring soon"
          description: "MarchProxy license expires in {{ $value }} days."

      - alert: MarchProxyLicenseInvalid
        expr: marchproxy_license_valid == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Invalid license"
          description: "MarchProxy license is invalid or expired."
```

### AlertManager Configuration

```yaml
# alertmanager.yml
global:
  smtp_smarthost: 'mail.company.com:587'
  smtp_from: 'alerts@company.com'
  smtp_auth_username: 'alerts@company.com'
  smtp_auth_password: 'password'

templates:
  - '/etc/alertmanager/templates/*.tmpl'

route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
    # Critical alerts go to PagerDuty
    - match:
        severity: critical
      receiver: 'pagerduty'
      continue: true

    # Warning alerts go to Slack
    - match:
        severity: warning
      receiver: 'slack'

    # Security alerts go to security team
    - match_re:
        alertname: '^MarchProxy.*Login.*|.*Rate.*Limit.*'
      receiver: 'security-team'

receivers:
  - name: 'default'
    email_configs:
      - to: 'ops-team@company.com'
        subject: '[MarchProxy] {{ .GroupLabels.alertname }}'
        body: |
          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Instance: {{ .Labels.instance }}
          Severity: {{ .Labels.severity }}
          {{ end }}

  - name: 'pagerduty'
    pagerduty_configs:
      - routing_key: 'your-pagerduty-integration-key'
        description: '{{ .GroupLabels.alertname }} - {{ .GroupLabels.instance }}'

  - name: 'slack'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'
        channel: '#ops-alerts'
        title: 'MarchProxy Alert'
        text: |
          {{ range .Alerts }}
          *Alert:* {{ .Annotations.summary }}
          *Description:* {{ .Annotations.description }}
          *Instance:* {{ .Labels.instance }}
          *Severity:* {{ .Labels.severity }}
          {{ end }}

  - name: 'security-team'
    email_configs:
      - to: 'security@company.com'
        subject: '[SECURITY] MarchProxy Alert'
        body: |
          SECURITY ALERT DETECTED

          {{ range .Alerts }}
          Alert: {{ .Annotations.summary }}
          Description: {{ .Annotations.description }}
          Time: {{ .StartsAt }}
          {{ end }}

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'instance']
```

## Distributed Tracing

### Jaeger Integration

Configure distributed tracing with Jaeger:

```yaml
# Manager tracing configuration
tracing:
  enabled: true
  jaeger:
    agent_host: "jaeger-agent"
    agent_port: 6831
    sampler:
      type: "probabilistic"
      param: 0.1  # Sample 10% of traces
  service_name: "marchproxy-manager"
  tags:
    version: "v0.1.1"
    environment: "production"
```

```yaml
# Proxy tracing configuration
tracing:
  enabled: true
  jaeger:
    agent_host: "jaeger-agent"
    agent_port: 6831
    sampler:
      type: "probabilistic"
      param: 0.1
  service_name: "marchproxy-proxy"
  tags:
    version: "v0.1.1"
    environment: "production"
    cluster_id: "${CLUSTER_ID}"
```

### Custom Spans

Add custom tracing spans in application code:

```go
// Go proxy tracing example
func (p *Proxy) handleRequest(ctx context.Context, req *http.Request) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "proxy.handleRequest")
    defer span.Finish()

    span.SetTag("http.method", req.Method)
    span.SetTag("http.url", req.URL.String())

    // Authentication span
    authSpan, ctx := opentracing.StartSpanFromContext(ctx, "proxy.authenticate")
    authSpan.SetTag("auth.type", "jwt")

    if err := p.authenticate(ctx, req); err != nil {
        authSpan.SetTag("error", true)
        authSpan.LogFields(log.Error(err))
        return
    }
    authSpan.Finish()

    // Proxy span
    proxySpan, ctx := opentracing.StartSpanFromContext(ctx, "proxy.forward")
    resp, err := p.forward(ctx, req)
    if err != nil {
        proxySpan.SetTag("error", true)
        span.SetTag("error", true)
    }
    proxySpan.SetTag("http.status_code", resp.StatusCode)
    proxySpan.Finish()

    span.SetTag("http.status_code", resp.StatusCode)
}
```

```python
# Python manager tracing example
from jaeger_client import Config
from opentracing.ext import tags

def create_service(service_data):
    with tracer.start_span('manager.create_service') as span:
        span.set_tag(tags.COMPONENT, 'manager')
        span.set_tag('service.name', service_data['name'])

        # Database span
        with tracer.start_span('manager.database.insert', child_of=span) as db_span:
            db_span.set_tag(tags.DATABASE_TYPE, 'postgresql')
            try:
                service = db.services.insert(service_data)
                db_span.set_tag('service.id', service.id)
            except Exception as e:
                db_span.set_tag(tags.ERROR, True)
                db_span.log_kv({'error.message': str(e)})
                raise

        # License validation span
        with tracer.start_span('manager.license.validate', child_of=span) as license_span:
            if not validate_license_limits():
                license_span.set_tag(tags.ERROR, True)
                raise LicenseExceededError()

        span.set_tag('service.id', service.id)
        return service
```

## Centralized Logging

### ELK Stack Configuration

#### Elasticsearch Configuration

```yaml
# elasticsearch.yml
cluster.name: marchproxy-logs
node.name: elasticsearch-1
path.data: /var/lib/elasticsearch
path.logs: /var/log/elasticsearch
network.host: 0.0.0.0
http.port: 9200
discovery.type: single-node

# Performance settings
bootstrap.memory_lock: true
indices.fielddata.cache.size: 40%
indices.memory.index_buffer_size: 10%

# Index settings
action.auto_create_index: "marchproxy-*"
```

#### Logstash Configuration

```ruby
# logstash.conf
input {
  # Syslog input from MarchProxy components
  syslog {
    port => 514
    type => "syslog"
  }

  # Filebeat input
  beats {
    port => 5044
  }

  # Direct JSON logs
  tcp {
    port => 5000
    codec => json_lines
  }
}

filter {
  # Parse MarchProxy logs
  if [program] == "marchproxy-manager" {
    json {
      source => "message"
    }

    mutate {
      add_field => { "component" => "manager" }
    }

    # Parse API logs
    if [logger] == "api" {
      grok {
        match => { "message" => "%{COMBINEDAPACHELOG}" }
      }

      date {
        match => [ "timestamp", "ISO8601" ]
      }
    }
  }

  if [program] == "marchproxy-proxy" {
    json {
      source => "message"
    }

    mutate {
      add_field => { "component" => "proxy" }
    }

    # Parse performance metrics from logs
    if [logger] == "performance" {
      ruby {
        code => "
          if event.get('metrics')
            event.set('cpu_usage', event.get('metrics')['cpu_usage'])
            event.set('memory_usage', event.get('metrics')['memory_usage'])
            event.set('active_connections', event.get('metrics')['active_connections'])
          end
        "
      }
    }
  }

  # GeoIP lookup for external IPs
  if [client_ip] and [client_ip] !~ /^10\./ and [client_ip] !~ /^192\.168\./ {
    geoip {
      source => "client_ip"
      target => "geoip"
    }
  }

  # Anonymize sensitive data
  mutate {
    gsub => [
      "message", "password=\S+", "password=***",
      "message", "api_key=\S+", "api_key=***",
      "message", "token=\S+", "token=***"
    ]
  }
}

output {
  elasticsearch {
    hosts => ["elasticsearch:9200"]
    index => "marchproxy-%{component}-%{+YYYY.MM.dd}"
    template_name => "marchproxy"
    template => "/etc/logstash/templates/marchproxy.json"
    template_overwrite => true
  }

  # Debug output
  if [loglevel] == "DEBUG" {
    stdout {
      codec => rubydebug
    }
  }
}
```

#### Kibana Configuration

```yaml
# kibana.yml
server.host: "0.0.0.0"
server.port: 5601
elasticsearch.hosts: ["http://elasticsearch:9200"]
server.name: "marchproxy-kibana"

# Default index patterns
kibana.defaultAppId: "discover"
kibana.index: ".kibana"

# Security (if X-Pack is enabled)
xpack.security.enabled: false
xpack.monitoring.enabled: true
```

### Log Parsing and Indexing

#### Index Templates

```json
{
  "index_patterns": ["marchproxy-*"],
  "template": {
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 1,
      "index.refresh_interval": "5s",
      "index.codec": "best_compression"
    },
    "mappings": {
      "properties": {
        "@timestamp": {
          "type": "date"
        },
        "component": {
          "type": "keyword"
        },
        "level": {
          "type": "keyword"
        },
        "logger": {
          "type": "keyword"
        },
        "message": {
          "type": "text",
          "analyzer": "standard"
        },
        "request_id": {
          "type": "keyword"
        },
        "user_id": {
          "type": "keyword"
        },
        "cluster_id": {
          "type": "keyword"
        },
        "service_name": {
          "type": "keyword"
        },
        "client_ip": {
          "type": "ip"
        },
        "response_time": {
          "type": "float"
        },
        "http_status": {
          "type": "short"
        },
        "cpu_usage": {
          "type": "float"
        },
        "memory_usage": {
          "type": "float"
        },
        "active_connections": {
          "type": "integer"
        },
        "geoip": {
          "properties": {
            "location": {
              "type": "geo_point"
            },
            "country_name": {
              "type": "keyword"
            },
            "city_name": {
              "type": "keyword"
            }
          }
        }
      }
    }
  }
}
```

### Log Retention and Management

```bash
# Curator configuration for log retention
# curator.yml
client:
  hosts:
    - elasticsearch
  port: 9200
  url_prefix:
  use_ssl: False
  certificate:
  client_cert:
  client_key:
  ssl_no_validate: False
  http_auth:
  timeout: 30
  master_only: False

logging:
  loglevel: INFO
  logfile:
  logformat: default
  blacklist: ['elasticsearch', 'urllib3']

# Delete indices older than 30 days
actions:
  1:
    action: delete_indices
    description: "Delete marchproxy indices older than 30 days"
    options:
      ignore_empty_list: True
      timeout_override:
      continue_if_exception: False
      disable_action: False
    filters:
    - filtertype: pattern
      kind: prefix
      value: marchproxy-
      exclude:
    - filtertype: age
      source: name
      direction: older
      timestring: '%Y.%m.%d'
      unit: days
      unit_count: 30
      exclude:

  2:
    action: forcemerge
    description: "Optimize indices older than 2 days"
    options:
      max_num_segments: 1
      delay: 120
      timeout_override: 21600
      continue_if_exception: False
      disable_action: False
    filters:
    - filtertype: pattern
      kind: prefix
      value: marchproxy-
      exclude:
    - filtertype: age
      source: name
      direction: older
      timestring: '%Y.%m.%d'
      unit: days
      unit_count: 2
      exclude:
```

This completes the comprehensive monitoring and observability documentation for MarchProxy.