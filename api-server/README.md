# MarchProxy API Server

Enterprise-grade API server with integrated xDS control plane for dynamic Envoy proxy configuration.

## Features

- **FastAPI Backend**: High-performance async Python API server
- **xDS Control Plane**: Go-based gRPC server for Envoy dynamic configuration
- **SQLAlchemy ORM**: PostgreSQL database with Alembic migrations
- **JWT Authentication**: Secure token-based authentication
- **License Integration**: PenguinTech license server validation
- **OpenTelemetry**: Distributed tracing and observability
- **Prometheus Metrics**: Built-in metrics endpoint

## Architecture

```
┌─────────────────────────────────────────────┐
│          API Server Container               │
│                                             │
│  ┌──────────────┐      ┌─────────────────┐ │
│  │   FastAPI    │◄────►│   xDS Server    │ │
│  │  (Python)    │      │      (Go)       │ │
│  │              │      │                 │ │
│  │  Port: 8000  │      │  Port: 18000    │ │
│  │              │      │  HTTP: 19000    │ │
│  └──────┬───────┘      └─────────┬───────┘ │
│         │                        │         │
└─────────┼────────────────────────┼─────────┘
          │                        │
          ▼                        ▼
    PostgreSQL              Envoy Proxies
                           (Dynamic Config)
```

## Quick Start

### Build

```bash
docker build -t marchproxy-api-server:latest .
```

### Run

```bash
docker run -d \
  --name marchproxy-api \
  -p 8000:8000 \
  -p 18000:18000 \
  -p 19000:19000 \
  -e DATABASE_URL=postgresql://user:pass@postgres:5432/marchproxy \
  -e SECRET_KEY=your-secret-key \
  -e XDS_SERVER_URL=http://localhost:19000 \
  marchproxy-api-server:latest
```

### Verify

```bash
# Check API health
curl http://localhost:8000/healthz

# Check xDS server
curl http://localhost:19000/healthz

# Check xDS stats
curl http://localhost:8000/xds/stats

# View API docs
open http://localhost:8000/api/docs
```

## Environment Variables

### Required

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgresql+asyncpg://user:pass@host:5432/db` |
| `SECRET_KEY` | JWT signing key | `your-secret-key-here` |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `DEBUG` | `false` | Enable debug mode |
| `PORT` | `8000` | FastAPI server port |
| `WORKERS` | `4` | Number of Uvicorn workers |
| `XDS_GRPC_PORT` | `18000` | xDS gRPC server port |
| `XDS_SERVER_URL` | `http://localhost:19000` | xDS HTTP API URL |
| `REDIS_URL` | `redis://redis:6379/0` | Redis cache URL |
| `LICENSE_KEY` | - | PenguinTech license key |
| `RELEASE_MODE` | `false` | Enable license enforcement |

## API Endpoints

### Core

- `GET /` - API information
- `GET /healthz` - Health check
- `GET /metrics` - Prometheus metrics
- `GET /api/docs` - Swagger UI
- `GET /api/redoc` - ReDoc UI

### Services (with xDS integration)

- `POST /api/v1/services` - Create service (triggers xDS update)
- `GET /api/v1/services` - List services
- `GET /api/v1/services/{id}` - Get service
- `PUT /api/v1/services/{id}` - Update service (triggers xDS update)
- `DELETE /api/v1/services/{id}` - Delete service (triggers xDS update)
- `POST /api/v1/services/{id}/reload-xds` - Force xDS reload

### xDS Control Plane

- `GET /xds/stats` - xDS bridge statistics
- `POST http://localhost:19000/v1/config` - Update xDS configuration (internal)
- `GET http://localhost:19000/v1/version` - Get configuration version (internal)

## Development

### Local Setup

```bash
# Install Python dependencies
pip install -r requirements.txt

# Build xDS server
cd xds
go build -o xds-server .
cd ..

# Run migrations
alembic upgrade head

# Start services
./start.sh
```

### Testing

```bash
# Python tests
pytest

# Go tests
cd xds && go test -v

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

### Linting

```bash
# Python
flake8 app/
black app/
isort app/
mypy app/

# Go
cd xds
golangci-lint run
go fmt ./...
```

## Directory Structure

```
api-server/
├── app/                    # FastAPI application
│   ├── api/               # API routes
│   │   └── v1/
│   │       └── routes/
│   │           └── services.py
│   ├── core/              # Core functionality
│   │   ├── database.py
│   │   ├── security.py
│   │   └── license.py
│   ├── models/            # Database models
│   │   └── sqlalchemy/
│   ├── services/          # Business logic
│   │   └── xds_bridge.py
│   ├── config.py          # Configuration
│   └── main.py            # Application entry point
├── xds/                   # Go xDS server
│   ├── main.go
│   ├── cache.go
│   ├── callbacks.go
│   ├── snapshot.go
│   ├── api.go
│   └── go.mod
├── alembic/               # Database migrations
├── Dockerfile             # Multi-stage build
├── requirements.txt       # Python dependencies
├── start.sh              # Startup script
└── README.md             # This file
```

## xDS Control Plane

The integrated xDS control plane provides dynamic configuration for Envoy proxies using the gRPC-based xDS protocol.

### How It Works

1. **Configuration Change**: Admin updates service via FastAPI
2. **Database Update**: Changes saved to PostgreSQL
3. **xDS Trigger**: Python bridge notifies xDS server
4. **Snapshot Generation**: Go xDS server generates Envoy config
5. **Push to Envoy**: Configuration streamed to connected proxies
6. **Apply**: Envoy applies configuration without restart

### Envoy Integration

Envoy proxies connect to the xDS server on port 18000:

```yaml
# Envoy bootstrap.yaml
node:
  id: "cluster-1"

dynamic_resources:
  lds_config:
    api_config_source:
      api_type: GRPC
      grpc_services:
        - envoy_grpc:
            cluster_name: xds_cluster

static_resources:
  clusters:
    - name: xds_cluster
      type: STRICT_DNS
      load_assignment:
        cluster_name: xds_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: api-server
                      port_value: 18000
      http2_protocol_options: {}
```

## Monitoring

### Health Checks

```bash
# API server health
curl http://localhost:8000/healthz

# Expected response:
{
  "status": "healthy",
  "version": "1.0.0",
  "xds_server": "healthy"
}
```

### Metrics

Prometheus metrics available at `http://localhost:8000/metrics`

### Logs

```bash
# View logs
docker logs -f marchproxy-api

# Follow xDS updates
docker logs -f marchproxy-api | grep xDS
```

## Troubleshooting

### Database Connection Failed

```bash
# Test database connectivity
psql $DATABASE_URL -c "SELECT 1"

# Check database logs
docker logs postgres
```

### xDS Server Not Starting

```bash
# Check xDS server binary
docker exec marchproxy-api /usr/local/bin/xds-server -help

# Test xDS server directly
/usr/local/bin/xds-server -port 18000 -debug
```

### Envoy Not Receiving Updates

1. Verify Envoy can reach port 18000
2. Check Envoy admin: `http://envoy:9901/config_dump`
3. Review xDS server logs for connection errors
4. Verify node ID matches in Envoy and xDS cache

## Security

- **JWT Authentication**: All API endpoints protected
- **HTTPS Ready**: Configure SSL/TLS certificates
- **License Validation**: Integration with PenguinTech license server
- **Input Validation**: Comprehensive request validation
- **Rate Limiting**: TODO - implement rate limiting
- **CORS**: Configurable cross-origin policies

## Performance

- **Async I/O**: FastAPI with async/await
- **Connection Pooling**: Database and Redis connections
- **Caching**: Redis-backed response caching
- **xDS Efficiency**: Snapshot caching and versioning
- **Multi-worker**: Uvicorn with configurable workers

## License

Limited AGPL3 with PenguinTech preamble for fair use.

## Support

For issues and questions:
- GitHub: https://github.com/penguintech/marchproxy
- Website: https://www.penguintech.io
- Docs: See `docs/` folder

## See Also

- [XDS_IMPLEMENTATION.md](XDS_IMPLEMENTATION.md) - Detailed xDS implementation guide
- [../docs/WORKFLOWS.md](../docs/WORKFLOWS.md) - CI/CD workflows
- [../docs/STANDARDS.md](../docs/STANDARDS.md) - Development standards
