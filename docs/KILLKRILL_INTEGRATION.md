# KillKrill Integration

This document describes the KillKrill integration for MarchProxy, enabling centralized logging and metrics collection.

## Overview

KillKrill is a centralized log and metrics ingestion platform that provides:

- **HTTP3/QUIC and UDP Syslog** for high-performance log ingestion
- **HTTP3/QUIC API** for metrics collection in Prometheus format
- **Redis Streams** for guaranteed delivery with zero duplication
- **ELK Stack** integration for log storage and analysis
- **Prometheus** integration for metrics storage and monitoring

## Configuration

### Environment Variables

Both the Go proxy and Python manager support the following environment variables:

```bash
# Core KillKrill settings
KILLKRILL_ENABLED=true
KILLKRILL_LOG_ENDPOINT=https://killkrill.example.com/api/v1/logs
KILLKRILL_METRICS_ENDPOINT=https://killkrill.example.com/api/v1/metrics
KILLKRILL_API_KEY=your-api-key-here
KILLKRILL_SOURCE_NAME=marchproxy-instance-1

# Optional settings
KILLKRILL_APPLICATION=proxy  # or "manager"
KILLKRILL_BATCH_SIZE=100     # Number of entries to batch
KILLKRILL_FLUSH_INTERVAL=10  # Seconds between flushes
KILLKRILL_TIMEOUT=30         # Request timeout in seconds
KILLKRILL_USE_HTTP3=true     # Use HTTP/3 for transport
KILLKRILL_TLS_INSECURE=false # Skip TLS verification (dev only)
```

### Go Proxy Configuration

The proxy reads configuration from environment variables and config files using Viper.

```yaml
# config.yaml
killkrill_enabled: true
killkrill_log_endpoint: "https://killkrill.example.com/api/v1/logs"
killkrill_metrics_endpoint: "https://killkrill.example.com/api/v1/metrics"
killkrill_api_key: "your-api-key-here"
killkrill_source_name: "marchproxy-proxy"
```

### Python Manager Configuration

The manager stores configuration in the database with environment variable fallbacks.

## Components

### Go Proxy Integration

#### KillKrill Client (`proxy-egress/internal/killkrill/`)

- **client.go**: HTTP3/QUIC client with batching and automatic flushing
- **converter.go**: Converts between logrus/Prometheus and KillKrill formats
- **hook.go**: Logrus hook for automatic log forwarding

#### Enhanced Logging (`proxy-egress/internal/logging/`)

- Automatic KillKrill integration via logrus hooks
- Structured logging with ECS 8.0 format
- Configurable log levels and output targets

#### Enhanced Metrics (`proxy-egress/internal/metrics/`)

- Dual export: Prometheus (local) + KillKrill (centralized)
- Automatic conversion of all Prometheus metrics
- Periodic batch export with configurable intervals

### Python Manager Integration

#### KillKrill Service (`manager/services/killkrill_service.py`)

- **KillKrillService**: Main service class with batching and HTTP client
- **KillKrillLogHandler**: Python logging handler for automatic forwarding
- **Health Checks**: Monitor KillKrill endpoint availability
- **Thread Safety**: Concurrent-safe buffering and transmission

#### Configuration Management (`manager/config/settings.py`)

- Database-first configuration with environment fallbacks
- `get_killkrill_config()` method for retrieving settings
- Integration with existing configuration management

## Usage Examples

### Go Proxy

```go
// Create KillKrill-integrated logger
killKrillConfig := &killkrill.Config{
    Enabled:         true,
    LogEndpoint:     "https://killkrill.example.com/api/v1/logs",
    MetricsEndpoint: "https://killkrill.example.com/api/v1/metrics",
    APIKey:          "your-api-key",
    SourceName:      "marchproxy-proxy",
}

logger, err := logging.NewLoggerWithKillKrill("info", "", killKrillConfig)
if err != nil {
    log.Fatal(err)
}
defer logger.Close()

// Logging automatically goes to both console and KillKrill
logger.Info("Proxy starting", "version", "1.0.0")

// Create metrics collector with KillKrill export
metricsConfig := metrics.MetricsConfig{
    KillKrillConfig: killKrillConfig,
}
collector := metrics.NewMetricsCollector(metricsConfig)
collector.StartCollection() // Exports to both Prometheus and KillKrill
```

### Python Manager

```python
from services.killkrill_service import KillKrillService, setup_killkrill_logging

# Create KillKrill service
config = {
    'enabled': True,
    'log_endpoint': 'https://killkrill.example.com/api/v1/logs',
    'metrics_endpoint': 'https://killkrill.example.com/api/v1/metrics',
    'api_key': 'your-api-key',
    'source_name': 'marchproxy-manager',
}
killkrill_service = KillKrillService(config)

# Setup automatic logging
logger = logging.getLogger('marchproxy')
setup_killkrill_logging(logger, killkrill_service)

# Logging automatically goes to KillKrill
logger.info("Manager starting", extra={'component': 'startup'})

# Direct metrics
killkrill_service.send_metric(
    name='active_connections',
    metric_type='gauge',
    value=42,
    labels={'instance': 'manager-1'}
)
```

## Log Format

All logs sent to KillKrill follow the ECS (Elastic Common Schema) 8.0 format:

```json
{
  "timestamp": "2023-12-01T10:00:00.000Z",
  "log_level": "info",
  "message": "User authenticated successfully",
  "service_name": "marchproxy-proxy",
  "hostname": "proxy-01",
  "logger_name": "auth.login",
  "ecs_version": "8.0",
  "labels": {
    "user_id": "12345",
    "session_id": "abc123"
  },
  "tags": ["authentication", "success"],
  "trace_id": "trace-123",
  "span_id": "span-456"
}
```

## Metrics Format

Metrics are converted from Prometheus format to KillKrill's JSON format:

```json
{
  "name": "http_requests_total",
  "type": "counter",
  "value": 1245.0,
  "labels": {
    "method": "GET",
    "status": "200",
    "endpoint": "/api/users"
  },
  "timestamp": "2023-12-01T10:00:00.000Z",
  "help": "Total number of HTTP requests"
}
```

## Benefits

1. **Centralized Observability**: All MarchProxy logs and metrics in one platform
2. **High Performance**: HTTP3/QUIC transport with batching reduces overhead
3. **Reliability**: Redis Streams guarantee delivery without duplication
4. **Compatibility**: Maintains existing Prometheus and logging functionality
5. **Enterprise Integration**: Leverages existing KillKrill infrastructure
6. **Zero Downtime**: Fallback to local logging/metrics if KillKrill unavailable

## Deployment

### Docker Environment Variables

```bash
docker run -e KILLKRILL_ENABLED=true \
           -e KILLKRILL_LOG_ENDPOINT=https://killkrill.example.com/api/v1/logs \
           -e KILLKRILL_METRICS_ENDPOINT=https://killkrill.example.com/api/v1/metrics \
           -e KILLKRILL_API_KEY=your-api-key \
           -e KILLKRILL_SOURCE_NAME=marchproxy-proxy-01 \
           marchproxy/proxy:latest
```

### Kubernetes ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: marchproxy-killkrill-config
data:
  KILLKRILL_ENABLED: "true"
  KILLKRILL_LOG_ENDPOINT: "https://killkrill.example.com/api/v1/logs"
  KILLKRILL_METRICS_ENDPOINT: "https://killkrill.example.com/api/v1/metrics"
  KILLKRILL_SOURCE_NAME: "marchproxy-k8s"
```

## Troubleshooting

### Common Issues

1. **Connection Failures**: Check `KILLKRILL_TLS_INSECURE=true` for development
2. **Authentication Errors**: Verify `KILLKRILL_API_KEY` is correct
3. **High Memory Usage**: Reduce `KILLKRILL_BATCH_SIZE` or increase `KILLKRILL_FLUSH_INTERVAL`
4. **Missing Logs/Metrics**: Ensure `KILLKRILL_ENABLED=true` and endpoints are reachable

### Health Checks

Both components provide health check capabilities:

```go
// Go: KillKrill client automatically handles connection failures
// Logs will show connection issues but won't stop operation

// Python: Explicit health check
health := killkrill_service.health_check()
fmt.Printf("KillKrill health: %v\n", health)
```

## Security Considerations

1. **API Keys**: Store `KILLKRILL_API_KEY` securely (environment variables, secrets management)
2. **TLS**: Always use HTTPS endpoints in production (`KILLKRILL_TLS_INSECURE=false`)
3. **Network**: Ensure KillKrill endpoints are accessible from MarchProxy instances
4. **Data Privacy**: Review log content for sensitive information before enabling

## Performance Impact

The KillKrill integration is designed for minimal performance impact:

- **Asynchronous**: All KillKrill operations are non-blocking
- **Batched**: Reduces network overhead through intelligent batching
- **HTTP3**: Uses modern protocols for optimal performance
- **Fallback**: Continues normal operation if KillKrill is unavailable
- **Configurable**: Batch sizes and intervals can be tuned for your environment