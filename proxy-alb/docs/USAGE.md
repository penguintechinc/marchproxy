# MarchProxy ALB - Usage Guide

## Quick Start

### Prerequisites
- Docker installed
- Access to MarchProxy xDS control plane (api-server)
- License key for Enterprise features (optional)

### Running with Docker

Pull and run the latest ALB image:

```bash
docker run -d \
  --name marchproxy-alb \
  -p 10000:10000 \
  -p 50051:50051 \
  -p 8080:8080 \
  -p 9090:9090 \
  -e XDS_SERVER=api-server:18000 \
  -e MODULE_ID=alb-1 \
  marchproxy-alb:latest
```

Verify it's running:

```bash
curl http://localhost:8080/healthz
# Output: OK
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  alb:
    image: marchproxy-alb:latest
    ports:
      - "10000:10000"  # Traffic port
      - "50051:50051"  # gRPC control plane
      - "8080:8080"    # Health checks
      - "9090:9090"    # Metrics
    environment:
      XDS_SERVER: api-server:18000
      MODULE_ID: alb-1
      LOG_LEVEL: info
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 10s
    networks:
      - marchproxy

  api-server:
    image: marchproxy-api:latest
    ports:
      - "18000:18000"
    networks:
      - marchproxy

networks:
  marchproxy:
```

Start:

```bash
docker-compose up -d
```

## Configuration

The ALB is configured via environment variables. Key variables:

| Variable | Purpose | Example |
|----------|---------|---------|
| `MODULE_ID` | ALB instance identifier | `alb-1` |
| `XDS_SERVER` | Control plane address | `api-server:18000` |
| `GRPC_PORT` | gRPC API port | `50051` |
| `HEALTH_PORT` | Health check port | `8080` |
| `METRICS_PORT` | Prometheus metrics port | `9090` |
| `LOG_LEVEL` | Logging level | `info` |

See `docs/CONFIGURATION.md` for complete reference.

## Deployment Patterns

### Single ALB Instance

Ideal for development and small deployments:

```bash
docker run -d \
  -p 10000:10000 \
  -p 50051:50051 \
  -e XDS_SERVER=api-server:18000 \
  -e MODULE_ID=alb-1 \
  marchproxy-alb:latest
```

### Multiple ALBs for High Availability

Deploy multiple ALB instances with unique IDs:

```bash
# ALB Instance 1
docker run -d --name alb-1 \
  -p 10001:10000 \
  -p 50051:50051 \
  -e MODULE_ID=alb-1 \
  -e XDS_SERVER=api-server:18000 \
  marchproxy-alb:latest

# ALB Instance 2
docker run -d --name alb-2 \
  -p 10002:10000 \
  -p 50052:50051 \
  -e MODULE_ID=alb-2 \
  -e XDS_SERVER=api-server:18000 \
  marchproxy-alb:latest

# ALB Instance 3
docker run -d --name alb-3 \
  -p 10003:10000 \
  -p 50053:50051 \
  -e MODULE_ID=alb-3 \
  -e XDS_SERVER=api-server:18000 \
  marchproxy-alb:latest
```

Place an external load balancer (nginx, HAProxy) in front:

```nginx
upstream albs {
  server localhost:10001;
  server localhost:10002;
  server localhost:10003;
}

server {
  listen 80;
  location / {
    proxy_pass http://albs;
  }
}
```

### Kubernetes Deployment

Example Kubernetes StatefulSet:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: marchproxy-alb
spec:
  serviceName: alb
  replicas: 3
  selector:
    matchLabels:
      app: alb
  template:
    metadata:
      labels:
        app: alb
    spec:
      containers:
      - name: alb
        image: marchproxy-alb:v1.0.0
        ports:
        - containerPort: 10000
          name: traffic
        - containerPort: 50051
          name: grpc
        - containerPort: 8080
          name: health
        - containerPort: 9090
          name: metrics
        env:
        - name: MODULE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: XDS_SERVER
          value: "api-server.default.svc.cluster.local:18000"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: alb
spec:
  clusterIP: None
  selector:
    app: alb
  ports:
  - port: 10000
    name: traffic
---
apiVersion: v1
kind: Service
metadata:
  name: alb-public
spec:
  type: LoadBalancer
  selector:
    app: alb
  ports:
  - port: 80
    targetPort: 10000
    name: traffic
```

## API Usage Examples

### Check ALB Status

```bash
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus
```

Response:
```json
{
  "module_id": "alb-1",
  "module_type": "ALB",
  "version": "v1.0.0",
  "health": "HEALTHY",
  "uptime_seconds": 3600,
  "envoy_version": "v1.28.0",
  "metadata": {
    "xds_server": "api-server:18000",
    "admin_port": "9901",
    "listen_port": "10000"
  }
}
```

### Retrieve Metrics

```bash
curl -s http://localhost:9090/metrics | grep alb_
```

Output:
```
alb_total_connections 1000
alb_active_connections 45
alb_total_requests 50000
alb_requests_per_second 1234
alb_latency_ms{quantile="0.5"} 12.34
alb_latency_ms{quantile="0.9"} 45.67
alb_latency_ms{quantile="0.99"} 123.45
alb_responses_total{status="200"} 49500
alb_responses_total{status="404"} 400
alb_responses_total{status="500"} 100
```

### Apply Rate Limiting

```bash
grpcurl -plaintext \
  -d '{
    "route_name":"api-route",
    "config":{
      "requests_per_second":1000,
      "burst_size":2000,
      "enabled":true
    }
  }' \
  localhost:50051 marchproxy.ModuleService/ApplyRateLimit
```

### Set Traffic Weights (Blue/Green)

```bash
grpcurl -plaintext \
  -d '{
    "route_name":"web-service",
    "weights":[
      {"backend_name":"blue-backend","weight":100},
      {"backend_name":"green-backend","weight":0}
    ]
  }' \
  localhost:50051 marchproxy.ModuleService/SetTrafficWeight
```

Switch to green (canary at 10%):

```bash
grpcurl -plaintext \
  -d '{
    "route_name":"web-service",
    "weights":[
      {"backend_name":"blue-backend","weight":90},
      {"backend_name":"green-backend","weight":10}
    ]
  }' \
  localhost:50051 marchproxy.ModuleService/SetTrafficWeight
```

### Reload Configuration

```bash
grpcurl -plaintext \
  -d '{"force":false}' \
  localhost:50051 marchproxy.ModuleService/Reload
```

## Monitoring

### Health Checks

Liveness check (is ALB running?):

```bash
curl -i http://localhost:8080/healthz
# HTTP/1.1 200 OK
# OK
```

Readiness check (is ALB ready for traffic?):

```bash
curl -i http://localhost:8080/ready
# HTTP/1.1 200 OK
# Ready
```

### Prometheus Integration

Scrape metrics in Prometheus:

```yaml
scrape_configs:
  - job_name: 'marchproxy-alb'
    static_configs:
      - targets: ['localhost:9090']
```

### Docker Logs

View logs:

```bash
docker logs -f marchproxy-alb
```

Filter by severity:

```bash
docker logs marchproxy-alb | grep ERROR
docker logs marchproxy-alb | grep WARNING
```

## Troubleshooting

### ALB fails to start

Check logs for configuration errors:

```bash
docker logs marchproxy-alb
```

Verify required environment variables are set:

```bash
docker inspect marchproxy-alb | grep -A 20 '"Env"'
```

### Cannot connect to xDS server

Verify network connectivity:

```bash
docker exec marchproxy-alb nc -zv api-server 18000
```

Check xDS server is running:

```bash
curl -i http://api-server:18000/healthz
```

### Metrics not available

Check metrics endpoint responds:

```bash
curl -i http://localhost:9090/metrics
```

Verify Prometheus can scrape:

```bash
curl -s http://localhost:9090/metrics | head -20
```

### High latency or timeouts

Check active connections:

```bash
curl -s http://localhost:9090/metrics | grep active_connections
```

Retrieve detailed metrics via gRPC:

```bash
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetMetrics
```

## Performance Tuning

### Increase gRPC connection timeout

For slower networks, increase xDS connection timeout:

```bash
export XDS_CONNECT_TIMEOUT=10s
```

### Optimize for throughput

Reduce logging overhead in production:

```bash
export LOG_LEVEL=warn
export ENVOY_LOG_LEVEL=warn
```

### Optimize for latency

Use shorter shutdown timeouts for quicker response:

```bash
export SHUTDOWN_TIMEOUT=15s
```

## Support and Documentation

- **API Reference**: See `docs/API.md`
- **Configuration Guide**: See `docs/CONFIGURATION.md`
- **Testing**: See `docs/TESTING.md`
- **Release Notes**: See `docs/RELEASE_NOTES.md`
