# MarchProxy ALB Quick Start Guide

Get the ALB up and running in 5 minutes.

## Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for local builds)
- protoc (Protocol Buffers compiler)

## Quick Start with Docker

### 1. Build the Image

```bash
cd /home/penguin/code/MarchProxy/proxy-alb
make build-docker
```

### 2. Run with Docker Compose

```bash
# Start complete stack (ALB + API Server + Postgres)
docker-compose -f docker-compose.example.yml up -d

# Check status
docker ps
```

### 3. Verify Health

```bash
# Health check
curl http://localhost:8080/healthz
# Expected: OK

# Readiness check
curl http://localhost:8080/ready
# Expected: Ready

# Envoy admin
curl http://localhost:9901/ready
# Expected: LIVE
```

### 4. Test gRPC Interface

```bash
# Install grpcurl if needed
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# Get ALB status
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus

# Expected output:
# {
#   "moduleId": "alb-1",
#   "moduleType": "ALB",
#   "version": "v1.0.0",
#   "health": "HEALTHY",
#   "uptimeSeconds": "45",
#   ...
# }
```

### 5. View Metrics

```bash
# Prometheus metrics
curl http://localhost:9090/metrics

# Envoy stats
curl http://localhost:9901/stats/prometheus
```

### 6. Send Test Traffic

```bash
# Send HTTP request through ALB
curl -i http://localhost:10000/

# Load test
ab -n 10000 -c 100 http://localhost:10000/
```

## Quick Start - Local Development

### 1. Generate Protobuf Code

```bash
cd /home/penguin/code/MarchProxy/proxy-alb
make proto
```

### 2. Build Go Binary

```bash
make build
```

### 3. Run Locally

**Note**: Requires Envoy binary installed locally.

```bash
# Set environment variables
export XDS_SERVER=localhost:18000
export MODULE_ID=alb-dev
export LOG_LEVEL=debug

# Run supervisor
./bin/alb-supervisor
```

## Testing the ModuleService API

### Get Status

```bash
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetStatus
```

### Get Routes

```bash
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetRoutes
```

### Get Metrics

```bash
grpcurl -plaintext localhost:50051 marchproxy.ModuleService/GetMetrics
```

### Apply Rate Limit

```bash
grpcurl -plaintext -d '{
  "routeName": "api-route",
  "config": {
    "requestsPerSecond": 1000,
    "burstSize": 100,
    "enabled": true
  }
}' localhost:50051 marchproxy.ModuleService/ApplyRateLimit
```

### Set Traffic Weights (Blue/Green)

```bash
grpcurl -plaintext -d '{
  "routeName": "api-route",
  "weights": [
    {"backendName": "blue", "weight": 90, "version": "blue"},
    {"backendName": "green", "weight": 10, "version": "green"}
  ]
}' localhost:50051 marchproxy.ModuleService/SetTrafficWeight
```

### Reload Configuration

```bash
grpcurl -plaintext -d '{"force": false}' \
  localhost:50051 marchproxy.ModuleService/Reload
```

## Monitoring with Prometheus & Grafana

### 1. Access Prometheus

```bash
# Open in browser
open http://localhost:9091

# Or via curl
curl http://localhost:9091/api/v1/query?query=alb_total_requests
```

### 2. Access Grafana

```bash
# Open in browser
open http://localhost:3001

# Login: admin / admin
```

### 3. Add Prometheus Data Source

1. Go to Configuration → Data Sources
2. Add Prometheus
3. URL: `http://prometheus:9090`
4. Save & Test

### 4. Create Dashboard

Import example queries:

```promql
# Total requests
alb_total_requests

# Requests per second
rate(alb_total_requests[1m])

# Active connections
alb_active_connections

# Latency (p99)
alb_latency_ms{quantile="0.99"}
```

## Common Tasks

### View Logs

```bash
# ALB supervisor logs
docker logs -f marchproxy-alb

# Envoy logs only
docker logs marchproxy-alb 2>&1 | grep envoy
```

### Restart ALB

```bash
# Graceful restart
docker restart marchproxy-alb

# Or via gRPC
grpcurl -plaintext -d '{"force": false}' \
  localhost:50051 marchproxy.ModuleService/Reload
```

### Stop ALB

```bash
docker stop marchproxy-alb
```

### Check Configuration

```bash
# View Envoy config
docker exec marchproxy-alb cat /etc/envoy/envoy.yaml

# View runtime config
curl http://localhost:9901/config_dump
```

### Debug Issues

```bash
# Check environment variables
docker exec marchproxy-alb env | grep -E 'XDS|MODULE|ENVOY'

# Check Envoy connectivity
docker exec marchproxy-alb wget -O- http://api-server:18000/healthz

# Check gRPC server
grpcurl -plaintext localhost:50051 list
```

## Integration with NLB (Example)

### Go Client

```go
package main

import (
    "context"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/PenguinTech/MarchProxy/proto/marchproxy"
)

func main() {
    // Connect
    conn, _ := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()))
    defer conn.Close()

    client := pb.NewModuleServiceClient(conn)

    // Get status
    status, _ := client.GetStatus(context.Background(), &pb.StatusRequest{})
    log.Printf("ALB Status: %v", status)
}
```

### Python Client

```python
import grpc
from proto.marchproxy import module_service_pb2_grpc

channel = grpc.insecure_channel('localhost:50051')
stub = module_service_pb2_grpc.ModuleServiceStub(channel)

# Get status
status = stub.GetStatus(module_service_pb2.StatusRequest())
print(f"ALB Health: {status.health}")
```

## Environment Variables Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `MODULE_ID` | `alb-1` | Unique ALB identifier |
| `XDS_SERVER` | `api-server:18000` | xDS server address |
| `GRPC_PORT` | `50051` | ModuleService gRPC port |
| `ENVOY_ADMIN_PORT` | `9901` | Envoy admin port |
| `ENVOY_LISTEN_PORT` | `10000` | HTTP/HTTPS port |
| `METRICS_PORT` | `9090` | Prometheus metrics |
| `HEALTH_PORT` | `8080` | Health checks |
| `LOG_LEVEL` | `info` | Log level |

## Troubleshooting

### Port Already in Use

```bash
# Check what's using the port
sudo lsof -i :50051

# Change port
docker run ... -e GRPC_PORT=50052 -p 50052:50052 ...
```

### Cannot Connect to xDS Server

```bash
# Test connectivity
docker exec marchproxy-alb ping api-server

# Check xDS server logs
docker logs api-server | grep xDS
```

### Envoy Won't Start

```bash
# Check Envoy logs
docker logs marchproxy-alb | grep envoy

# Verify config
docker exec marchproxy-alb cat /etc/envoy/envoy.yaml
```

### gRPC Connection Refused

```bash
# Check if gRPC server is running
grpcurl -plaintext localhost:50051 list

# Check logs
docker logs marchproxy-alb | grep grpc
```

## Next Steps

1. **Read Integration Guide**: See `INTEGRATION.md` for NLB integration
2. **Review Architecture**: See `README.md` for detailed architecture
3. **Check Implementation**: See `IMPLEMENTATION_SUMMARY.md` for details
4. **Run Tests**: `make test` to run unit tests
5. **Deploy to Production**: Review security settings and TLS configuration

## Useful Commands

```bash
# Complete rebuild
make clean && make build-docker

# View all running containers
docker ps

# Stop all containers
docker-compose -f docker-compose.example.yml down

# View metrics in real-time
watch -n 1 'curl -s http://localhost:9090/metrics | grep alb_'

# Continuous health check
watch -n 5 'curl -s http://localhost:8080/healthz'
```

## Getting Help

- **Documentation**: See `README.md` and `INTEGRATION.md`
- **Issues**: Check logs with `docker logs marchproxy-alb`
- **Health**: Monitor `/healthz` and `/ready` endpoints
- **Metrics**: Check Prometheus at http://localhost:9091

## Clean Up

```bash
# Stop all containers
docker-compose -f docker-compose.example.yml down

# Remove volumes
docker-compose -f docker-compose.example.yml down -v

# Remove images
docker rmi marchproxy/proxy-alb:latest
```

---

**That's it!** You now have a running ALB instance ready for NLB integration.
