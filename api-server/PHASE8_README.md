# Phase 8: Enterprise Feature APIs & UI

Implementation of enterprise-only features for MarchProxy v1.0.0, including advanced traffic shaping, multi-cloud routing, and distributed tracing.

## Overview

Phase 8 adds three major enterprise feature categories:
1. **Traffic Shaping & QoS** - Bandwidth limits, priority queues, DSCP marking
2. **Multi-Cloud Routing** - Intelligent routing across AWS/GCP/Azure with health monitoring
3. **Distributed Tracing** - OpenTelemetry integration with Jaeger/Zipkin

All features require an **Enterprise license** and gracefully degrade for Community users with upgrade prompts.

## Architecture

### API Server (FastAPI)

**Location:** `/api-server/app/`

#### Pydantic Schemas
- `app/schemas/traffic_shaping.py` - QoS policy validation models
- `app/schemas/multi_cloud.py` - Route table and health probe models
- `app/schemas/observability.py` - Tracing configuration models

#### API Routes
- `app/api/v1/routes/traffic_shaping.py` - Traffic shaping CRUD endpoints
- `app/api/v1/routes/multi_cloud.py` - Multi-cloud routing endpoints
- `app/api/v1/routes/observability.py` - Tracing configuration endpoints

#### Database Models
- `app/models/sqlalchemy/enterprise.py` - SQLAlchemy ORM models:
  - `QoSPolicy` - Traffic shaping policies
  - `RouteTable` - Multi-cloud route tables
  - `RouteHealthStatus` - Health monitoring data
  - `TracingConfig` - Distributed tracing configuration
  - `TracingStats` - Runtime tracing statistics

#### License Enforcement
All endpoints use the `check_enterprise_license()` dependency:
```python
@router.get("/policies")
async def list_qos_policies(
    _: None = Depends(check_enterprise_license)
):
    # Returns 403 Forbidden for Community users
```

### WebUI (React + TypeScript)

**Location:** `/webui/src/`

#### React Hooks
- `hooks/useTrafficShaping.ts` - QoS policy management hook
- `hooks/useMultiCloud.ts` - Route table management hook
- `hooks/useObservability.ts` - Tracing configuration hook

#### Page Components
- `pages/Enterprise/TrafficShaping.tsx` - Traffic shaping dashboard
- `pages/Enterprise/MultiCloudRouting.tsx` - Multi-cloud routing UI with cost analytics

#### Common Components
- `components/Common/LicenseGate.tsx` - Enterprise feature gate with upgrade prompt

#### Services
- `services/api.ts` - Axios client with JWT authentication and license error handling

## Features

### 1. Traffic Shaping & QoS

**Priority Levels:**
- **P0** - Interactive (<1ms latency SLA)
- **P1** - Real-time (<10ms latency SLA)
- **P2** - Bulk (<100ms latency SLA)
- **P3** - Best effort (no SLA)

**Bandwidth Configuration:**
- Ingress/egress rate limits (Mbps)
- Token bucket burst handling
- Per-service bandwidth allocation

**DSCP Marking:**
- EF (Expedited Forwarding)
- AF41, AF31, AF21, AF11 (Assured Forwarding classes)
- BE (Best Effort)

**API Endpoints:**
```
GET    /api/v1/traffic-shaping/policies
POST   /api/v1/traffic-shaping/policies
GET    /api/v1/traffic-shaping/policies/{id}
PUT    /api/v1/traffic-shaping/policies/{id}
DELETE /api/v1/traffic-shaping/policies/{id}
POST   /api/v1/traffic-shaping/policies/{id}/enable
POST   /api/v1/traffic-shaping/policies/{id}/disable
```

### 2. Multi-Cloud Intelligent Routing

**Routing Algorithms:**
- **Latency** - Route to lowest RTT endpoint
- **Cost** - Route to cheapest egress costs
- **Geo-Proximity** - Route to nearest region
- **Weighted Round-Robin** - Distribute traffic by weight
- **Failover** - Active-passive with automatic failover

**Cloud Providers:**
- AWS (Amazon Web Services)
- GCP (Google Cloud Platform)
- Azure (Microsoft Azure)
- On-Premise

**Health Monitoring:**
- TCP/HTTP/HTTPS/ICMP health probes
- Configurable intervals and thresholds
- RTT (Round-Trip Time) measurement
- Automatic unhealthy endpoint removal

**Cost Analytics:**
- Per-provider cost tracking
- Per-service cost breakdown
- Historical cost trends

**API Endpoints:**
```
GET    /api/v1/multi-cloud/routes
POST   /api/v1/multi-cloud/routes
GET    /api/v1/multi-cloud/routes/{id}
PUT    /api/v1/multi-cloud/routes/{id}
DELETE /api/v1/multi-cloud/routes/{id}
GET    /api/v1/multi-cloud/routes/{id}/health
POST   /api/v1/multi-cloud/routes/{id}/enable
POST   /api/v1/multi-cloud/routes/{id}/disable
POST   /api/v1/multi-cloud/routes/{id}/test-failover
GET    /api/v1/multi-cloud/analytics/cost
```

### 3. Distributed Tracing & Observability

**Tracing Backends:**
- Jaeger
- Zipkin
- OTLP (OpenTelemetry Protocol)

**Sampling Strategies:**
- **Always** - Sample all requests (100%)
- **Never** - No sampling (0%)
- **Probabilistic** - Random percentage (e.g., 10%)
- **Rate Limit** - Maximum traces per second
- **Error Only** - Only sample failed requests
- **Adaptive** - Dynamic sampling based on load

**Span Exporters:**
- gRPC
- HTTP
- Thrift

**Data Collection:**
- Request/response headers (optional)
- Request/response bodies (optional, privacy concern)
- Custom tags and metadata
- Service dependency graphs
- Latency histograms

**API Endpoints:**
```
GET    /api/v1/observability/tracing
POST   /api/v1/observability/tracing
GET    /api/v1/observability/tracing/{id}
PUT    /api/v1/observability/tracing/{id}
DELETE /api/v1/observability/tracing/{id}
GET    /api/v1/observability/tracing/{id}/stats
POST   /api/v1/observability/tracing/{id}/enable
POST   /api/v1/observability/tracing/{id}/disable
POST   /api/v1/observability/tracing/{id}/test
GET    /api/v1/observability/spans/search
```

## Database Schema

### QoS Policies Table
```sql
CREATE TABLE qos_policies (
    id INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    service_id INTEGER NOT NULL,
    cluster_id INTEGER NOT NULL,
    bandwidth_config JSON NOT NULL,
    priority_config JSON NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_qos_service_cluster ON qos_policies(service_id, cluster_id);
```

### Route Tables Table
```sql
CREATE TABLE route_tables (
    id INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    service_id INTEGER NOT NULL,
    cluster_id INTEGER NOT NULL,
    algorithm VARCHAR(20) NOT NULL,
    routes JSON NOT NULL,
    health_probe_config JSON NOT NULL,
    enable_auto_failover BOOLEAN NOT NULL DEFAULT TRUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_route_service_cluster ON route_tables(service_id, cluster_id);
```

### Route Health Status Table
```sql
CREATE TABLE route_health_status (
    id INTEGER PRIMARY KEY,
    route_table_id INTEGER NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    is_healthy BOOLEAN NOT NULL DEFAULT TRUE,
    last_check TIMESTAMP NOT NULL,
    rtt_ms FLOAT,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    last_error TEXT
);
CREATE INDEX idx_health_route_endpoint ON route_health_status(route_table_id, endpoint);
```

### Tracing Configs Table
```sql
CREATE TABLE tracing_configs (
    id INTEGER PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    cluster_id INTEGER NOT NULL,
    backend VARCHAR(20) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    exporter VARCHAR(20) NOT NULL,
    sampling_strategy VARCHAR(20) NOT NULL,
    sampling_rate FLOAT NOT NULL,
    max_traces_per_second INTEGER,
    include_request_headers BOOLEAN NOT NULL DEFAULT FALSE,
    include_response_headers BOOLEAN NOT NULL DEFAULT FALSE,
    include_request_body BOOLEAN NOT NULL DEFAULT FALSE,
    include_response_body BOOLEAN NOT NULL DEFAULT FALSE,
    max_attribute_length INTEGER NOT NULL DEFAULT 512,
    service_name VARCHAR(100) NOT NULL,
    custom_tags JSON,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

## License Enforcement

### Development Mode
```bash
# In development, all features are available
RELEASE_MODE=false
```

### Production Mode
```bash
# In production, license validation enforces Enterprise features
RELEASE_MODE=true
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
```

### License Features
Enterprise license must include these features:
- `traffic_shaping` - Traffic shaping & QoS
- `multi_cloud_routing` - Multi-cloud routing
- `distributed_tracing` - Distributed tracing

### Community User Experience
When a Community user tries to access an enterprise feature:
1. API returns `403 Forbidden` with upgrade information
2. WebUI shows `LicenseGate` component with:
   - Feature description
   - Benefits list
   - "Upgrade to Enterprise" CTA button
   - Link to pricing page

## Usage Examples

### Create QoS Policy (API)
```bash
curl -X POST http://localhost:8000/api/v1/traffic-shaping/policies \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "High Priority Web Traffic",
    "service_id": 1,
    "cluster_id": 1,
    "bandwidth": {
      "ingress_mbps": 1000,
      "egress_mbps": 1000,
      "burst_size_kb": 2048
    },
    "priority_config": {
      "priority": "P1",
      "weight": 10,
      "max_latency_ms": 10,
      "dscp_marking": "EF"
    }
  }'
```

### Create Route Table (API)
```bash
curl -X POST http://localhost:8000/api/v1/multi-cloud/routes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Multi-Region Failover",
    "service_id": 1,
    "cluster_id": 1,
    "algorithm": "latency",
    "routes": [
      {
        "provider": "aws",
        "region": "us-east-1",
        "endpoint": "https://api.us-east-1.example.com",
        "weight": 100,
        "cost_per_gb": 0.09,
        "is_active": true
      },
      {
        "provider": "gcp",
        "region": "us-central1",
        "endpoint": "https://api.us-central1.example.com",
        "weight": 100,
        "cost_per_gb": 0.08,
        "is_active": true
      }
    ],
    "health_probe": {
      "protocol": "https",
      "port": 443,
      "path": "/health",
      "interval_seconds": 30,
      "timeout_seconds": 5,
      "unhealthy_threshold": 3,
      "healthy_threshold": 2
    },
    "enable_auto_failover": true
  }'
```

## Testing

### Run API Tests
```bash
cd /home/penguin/code/MarchProxy/api-server
pytest tests/test_enterprise_features.py -v
```

### Test License Enforcement
```bash
# Test without license (should return 403)
RELEASE_MODE=true LICENSE_KEY="" pytest tests/test_license_gate.py

# Test with valid license (should succeed)
RELEASE_MODE=true LICENSE_KEY=PENG-TEST-TEST-TEST-TEST-ABCD pytest tests/test_license_gate.py
```

### Test WebUI Components
```bash
cd /home/penguin/code/MarchProxy/webui
npm test -- --testPathPattern=Enterprise
```

## Deployment

### Environment Variables
```bash
# API Server
DATABASE_URL=postgresql+asyncpg://marchproxy:pass@postgres:5432/marchproxy
REDIS_URL=redis://redis:6379/0
SECRET_KEY=your-secret-key-here
LICENSE_KEY=PENG-XXXX-XXXX-XXXX-XXXX-ABCD
RELEASE_MODE=true

# WebUI
REACT_APP_API_URL=http://api-server:8000
NODE_ENV=production
```

### Docker Compose
```yaml
services:
  api-server:
    build: ./api-server
    environment:
      - LICENSE_KEY=${LICENSE_KEY}
      - RELEASE_MODE=true
    ports:
      - "8000:8000"

  webui:
    build: ./webui
    environment:
      - REACT_APP_API_URL=http://api-server:8000
    ports:
      - "3000:3000"
```

## Future Enhancements (Post-v1.0.0)

1. **Real-time Health Monitoring Dashboard** - Live route health visualization
2. **Cost Optimization Recommendations** - AI-powered cost savings suggestions
3. **Trace Analysis Tools** - Service dependency graphs and bottleneck detection
4. **Custom Routing Algorithms** - User-defined routing logic via WASM
5. **Advanced QoS Policies** - Time-based policies, conditional rules

## Support

- **Documentation:** https://docs.penguintech.io/marchproxy/enterprise
- **License Issues:** license@penguintech.io
- **Technical Support:** support@penguintech.io
- **Upgrade:** https://www.penguintech.io/marchproxy/pricing
