# Phase 7: Module Management API Implementation

## Overview

This implementation adds comprehensive module management APIs to the MarchProxy API server for Phase 7 Unified NLB Architecture. The module management system enables:

- **Module CRUD operations** - Create, read, update, delete modules
- **Route configuration** - Per-module routing rules with match conditions and backends
- **Auto-scaling policies** - CPU/memory/request-based auto-scaling
- **Blue/Green deployments** - Zero-downtime deployments with traffic shifting
- **gRPC communication** - Health checks and configuration updates via gRPC

## Architecture

### Components

1. **Database Models** (`app/models/sqlalchemy/module.py`)
   - `Module` - Module configuration and state
   - `ModuleRoute` - Route configuration per module
   - `ScalingPolicy` - Auto-scaling policy per module
   - `Deployment` - Blue/green deployment tracking

2. **Pydantic Schemas** (`app/schemas/module.py`)
   - Request/response models for all module operations
   - Validation for create, update, and promote/rollback operations

3. **Services**
   - `ModuleService` (`app/services/module_service.py`) - Module and route business logic
   - `ScalingService` (`app/services/module_service_scaling.py`) - Scaling policy logic
   - `DeploymentService` (`app/services/module_service_scaling.py`) - Deployment logic
   - `ModuleGRPCClient` (`app/services/grpc_client.py`) - gRPC communication

4. **API Routes**
   - `modules.py` - Module CRUD + health checks
   - `module_routes.py` - Route configuration
   - `scaling.py` - Auto-scaling policies
   - `deployments.py` - Blue/green deployments

## Database Schema

### modules
- Module configuration: name, type, config, image, replicas
- gRPC connection: host, port
- Health status: status, last_health_check
- Versioning: version, created_at, updated_at

### module_routes
- Route matching: match_rules (JSON)
- Backend config: backend_config (JSON)
- Traffic control: rate_limit, priority
- State: enabled

### scaling_policies
- Instance limits: min_instances, max_instances
- Thresholds: scale_up_threshold, scale_down_threshold
- Configuration: cooldown_seconds, metric
- State: enabled

### deployments
- Version info: version, image
- Traffic management: traffic_weight, status
- Rollback: previous_deployment_id
- Health: health_check_passed, health_check_message
- Audit: deployed_by, deployed_at, completed_at

## API Endpoints

### Modules (`/api/v1/modules`)
- `GET /modules` - List all modules
- `POST /modules` - Create module (Admin)
- `GET /modules/{id}` - Get module details
- `PATCH /modules/{id}` - Update module (Admin)
- `DELETE /modules/{id}` - Delete/disable module (Admin)
- `POST /modules/{id}/health` - Check module health
- `POST /modules/{id}/enable` - Enable module (Admin)
- `POST /modules/{id}/disable` - Disable module (Admin)

### Module Routes (`/api/v1/modules/{module_id}/routes`)
- `GET /routes` - List module routes
- `POST /routes` - Create route (Admin)
- `GET /routes/{id}` - Get route details
- `PATCH /routes/{id}` - Update route (Admin)
- `DELETE /routes/{id}` - Delete route (Admin)
- `POST /routes/{id}/enable` - Enable route (Admin)
- `POST /routes/{id}/disable` - Disable route (Admin)

### Auto-Scaling (`/api/v1/modules/{module_id}/scaling`)
- `GET /scaling` - Get scaling policy
- `POST /scaling` - Create scaling policy (Admin)
- `PUT /scaling` - Update scaling policy (Admin)
- `DELETE /scaling` - Delete scaling policy (Admin)
- `POST /scaling/enable` - Enable auto-scaling (Admin)
- `POST /scaling/disable` - Disable auto-scaling (Admin)

### Deployments (`/api/v1/modules/{module_id}/deployments`)
- `GET /deployments` - List deployments
- `POST /deployments` - Create deployment (Admin)
- `GET /deployments/{id}` - Get deployment details
- `PATCH /deployments/{id}` - Update deployment (Admin)
- `POST /deployments/{id}/promote` - Promote deployment (Admin)
- `POST /deployments/{id}/rollback` - Rollback deployment (Admin)
- `GET /deployments/{id}/health` - Check deployment health

## Usage Examples

### Create a Module

```bash
POST /api/v1/modules
{
  "name": "http-proxy",
  "type": "L7_HTTP",
  "description": "HTTP/HTTPS proxy module",
  "config": {
    "max_connections": 10000,
    "timeout": 30
  },
  "grpc_host": "http-proxy-service",
  "grpc_port": 50051,
  "version": "v1.0.0",
  "image": "marchproxy/http-proxy:v1.0.0",
  "replicas": 3,
  "enabled": true
}
```

### Create a Route

```bash
POST /api/v1/modules/1/routes
{
  "name": "api-route",
  "match_rules": {
    "host": "api.example.com",
    "path": "/v1/*",
    "method": ["GET", "POST"]
  },
  "backend_config": {
    "target": "http://backend:8080",
    "timeout": 30,
    "retries": 3,
    "load_balancing": "round_robin"
  },
  "rate_limit": 1000.0,
  "priority": 100,
  "enabled": true
}
```

### Configure Auto-Scaling

```bash
POST /api/v1/modules/1/scaling
{
  "min_instances": 2,
  "max_instances": 10,
  "scale_up_threshold": 80.0,
  "scale_down_threshold": 20.0,
  "cooldown_seconds": 300,
  "metric": "cpu",
  "enabled": true
}
```

### Create Blue/Green Deployment

```bash
# 1. Create new deployment (0% traffic)
POST /api/v1/modules/1/deployments
{
  "version": "v2.0.0",
  "image": "marchproxy/http-proxy:v2.0.0",
  "config": {"new_feature": true},
  "environment": {"ENV": "production"},
  "traffic_weight": 0.0
}

# 2. Check deployment health
GET /api/v1/modules/1/deployments/2/health

# 3. Gradually promote (canary)
POST /api/v1/modules/1/deployments/2/promote
{
  "traffic_weight": 100.0,
  "incremental": true
}

# 4. Rollback if needed
POST /api/v1/modules/1/deployments/2/rollback
{
  "reason": "High error rate detected"
}
```

## gRPC Integration

The module management system communicates with module containers via gRPC for:

1. **Health Checks** - Query module health status
2. **Configuration Updates** - Push config changes to modules
3. **Route Reloads** - Trigger route reload after changes
4. **Metrics Collection** - Gather CPU, memory, request metrics
5. **Control Operations** - Start, stop, reload modules

### gRPC Client Manager

The `ModuleGRPCClientManager` maintains a pool of gRPC clients for all modules:

```python
from app.services.grpc_client import grpc_client_manager

# Get client for module
client = grpc_client_manager.get_client(module_id, host, port)

# Health check
health = await client.health_check()

# Update config
success = await client.update_config(config_dict)

# Reload routes
success = await client.reload_routes()
```

## Module Types

Supported module types (enum `ModuleType`):
- `L7_HTTP` - HTTP/HTTPS/HTTP2/HTTP3 proxy
- `L4_TCP` - TCP proxy
- `L4_UDP` - UDP proxy
- `L3_NETWORK` - Network layer proxy
- `OBSERVABILITY` - Observability module
- `ZERO_TRUST` - Zero trust security
- `MULTI_CLOUD` - Multi-cloud routing

## Module Status Lifecycle

```
DISABLED → STARTING → ENABLED → STOPPING → DISABLED
              ↓           ↓
            ERROR      ERROR
```

- `DISABLED` - Module not running
- `STARTING` - Module initialization in progress
- `ENABLED` - Module active and processing traffic
- `STOPPING` - Module shutdown in progress
- `ERROR` - Module encountered an error

## Deployment Status Lifecycle

```
PENDING → ROLLING_OUT → ACTIVE
    ↓          ↓           ↓
  FAILED   ROLLED_BACK  INACTIVE
```

- `PENDING` - Deployment created, not yet active
- `ROLLING_OUT` - Traffic gradually shifting to deployment
- `ACTIVE` - Deployment receiving 100% traffic
- `INACTIVE` - Previous deployment, no longer active
- `ROLLED_BACK` - Deployment rolled back due to issues
- `FAILED` - Deployment failed health checks

## Database Migration

Alembic migration `002_phase7_module_tables.py` creates all module management tables.

To apply migration:
```bash
cd /home/penguin/code/MarchProxy/api-server
alembic upgrade head
```

To rollback:
```bash
alembic downgrade -1
```

## Security & Permissions

- **Admin-only operations**: Create, update, delete modules/routes/policies/deployments
- **User operations**: View modules, check health, view deployments
- **License validation**: Enterprise features can be gated via license checks

## Next Steps

1. **Implement gRPC protocol definitions** - Create .proto files for module communication
2. **Module container templates** - Docker templates for L3/L4/L7 modules
3. **Auto-scaling controller** - Background service to monitor metrics and scale
4. **Deployment orchestration** - Automate traffic shifting for canary deployments
5. **Metrics integration** - Collect and expose module metrics via Prometheus
6. **Health check automation** - Periodic health checks for all modules

## Files Created

1. `/home/penguin/code/MarchProxy/api-server/app/models/sqlalchemy/module.py`
2. `/home/penguin/code/MarchProxy/api-server/app/schemas/module.py`
3. `/home/penguin/code/MarchProxy/api-server/app/services/grpc_client.py`
4. `/home/penguin/code/MarchProxy/api-server/app/services/module_service.py`
5. `/home/penguin/code/MarchProxy/api-server/app/services/module_service_scaling.py`
6. `/home/penguin/code/MarchProxy/api-server/app/api/v1/routes/modules.py`
7. `/home/penguin/code/MarchProxy/api-server/app/api/v1/routes/module_routes.py`
8. `/home/penguin/code/MarchProxy/api-server/app/api/v1/routes/scaling.py`
9. `/home/penguin/code/MarchProxy/api-server/app/api/v1/routes/deployments.py`
10. `/home/penguin/code/MarchProxy/api-server/alembic/versions/002_phase7_module_tables.py`

## Files Modified

1. `/home/penguin/code/MarchProxy/api-server/app/api/v1/__init__.py` - Added Phase 7 route imports

## Testing

Test the API endpoints using the provided test script or FastAPI's built-in documentation:

```bash
# Start API server
cd /home/penguin/code/MarchProxy/api-server
uvicorn app.main:app --reload

# Open API docs
http://localhost:8000/api/docs

# Test endpoints
curl -X GET http://localhost:8000/api/v1/modules
```

## Documentation

- API documentation: http://localhost:8000/api/docs
- ReDoc: http://localhost:8000/api/redoc
- OpenAPI schema: http://localhost:8000/api/openapi.json
