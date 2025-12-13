# MarchProxy Observability UI

Complete observability dashboard implementation for MarchProxy WebUI.

## Overview

This module provides comprehensive observability features including distributed tracing, metrics visualization, service dependency graphing, and alerting configuration.

## Components

### Pages

#### 1. Tracing (`Tracing.tsx`)
Distributed tracing dashboard with Jaeger integration.

**Features:**
- Embedded Jaeger UI iframe
- Trace search with advanced filters (service, operation, duration, tags, time range)
- Latency histogram visualization
- Error rate tracking over time
- Service dependency graph integration
- CSV export capability

**Routes:** `/observability/tracing`

#### 2. Metrics (`Metrics.tsx`)
Prometheus metrics dashboard with PromQL query builder.

**Features:**
- PromQL query editor with quick query templates
- Time-series charts (request rate, error rate, latency percentiles, throughput)
- Time range selector (5m, 15m, 1h, 6h, 24h, 7d)
- Auto-refresh with configurable intervals (10s, 30s, 1m, 5m)
- Real-time statistics cards (current, avg, min, max)
- CSV export capability

**Quick Queries:**
- Request Rate: `rate(http_requests_total[5m])`
- Error Rate: `rate(http_requests_total{status=~"5.."}[5m])`
- P95 Latency: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))`
- P99 Latency: `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))`
- Throughput: `rate(bytes_sent_total[5m])`
- Active Connections: `connections_active`

**Routes:** `/observability/metrics`

#### 3. Alerts (`Alerts.tsx`)
Alert rule management and active alerts monitoring.

**Features:**
- Alert rule CRUD operations
- Active alerts dashboard with status indicators
- Alert severity levels (critical, warning, info)
- PromQL-based alert conditions
- Threshold and duration configuration
- Alert acknowledgment and silencing
- Notification channel configuration (email, Slack, webhook, PagerDuty)
- Enable/disable alert rules

**Routes:** `/observability/alerts`

### Components

#### ServiceGraph (`ServiceGraph.tsx`)
Interactive service dependency graph visualization using React Flow.

**Features:**
- Real-time service health status (healthy, degraded, unhealthy)
- Request rate per edge visualization
- Error rate highlighting
- P95 latency metrics per service
- Interactive zoom/pan controls
- Auto-refresh capability
- Animated edges for high-traffic connections
- Grid-based auto-layout

**Props:**
- `timeRange?: TimeRange` - Time window for metrics
- `autoRefresh?: boolean` - Enable auto-refresh
- `refreshInterval?: number` - Refresh interval in seconds

## API Services

### `observabilityApi.ts`
Centralized API client for all observability operations.

**Tracing:**
- `getTraces(params)` - Search traces
- `getTraceById(id)` - Get single trace
- `getServices()` - List available services
- `getOperations(service)` - List operations for service
- `exportTraces(params)` - Export traces as CSV

**Metrics:**
- `queryMetrics(query)` - Execute PromQL query (instant)
- `queryRangeMetrics(query)` - Execute PromQL range query
- `getMetricNames()` - List available metrics
- `exportMetrics(query)` - Export metrics as CSV

**Service Graph:**
- `getServiceGraph(timeRange)` - Get service dependency graph

**Alerts:**
- `getAlerts()` - List alert rules
- `getActiveAlerts()` - List firing alerts
- `createAlert(alert)` - Create alert rule
- `updateAlert(id, alert)` - Update alert rule
- `deleteAlert(id)` - Delete alert rule
- `acknowledgeAlert(id, note)` - Acknowledge alert
- `silenceAlert(id, duration, note)` - Silence alert

**Configuration:**
- `getJaegerUrl()` - Get Jaeger UI URL

### `observabilityTypes.ts`
TypeScript type definitions for observability data structures.

**Key Types:**
- `Trace`, `Span`, `TraceSearchParams` - Distributed tracing
- `MetricQuery`, `MetricQueryResponse`, `MetricSeries` - Prometheus metrics
- `ServiceNode`, `ServiceEdge`, `ServiceDependency` - Service graphs
- `AlertRule`, `Alert`, `NotificationChannel` - Alerting
- `TimeSeriesData`, `ChartDataPoint` - Chart data

## Dependencies

### Required Packages
- `recharts` - Chart library for metrics visualization
- `reactflow` - Interactive graph visualization for service dependencies
- `@mui/material` - Material-UI components
- `@mui/x-date-pickers` - Date/time range pickers
- `axios` - HTTP client
- `react-router-dom` - Routing

### Peer Dependencies
- React 18+
- TypeScript 5+
- Vite (build system)

## Integration

### Backend API Requirements

The observability UI expects the following backend endpoints:

```
GET  /api/v1/observability/traces
GET  /api/v1/observability/traces/:id
GET  /api/v1/observability/traces/export
GET  /api/v1/observability/services
GET  /api/v1/observability/operations
GET  /api/v1/observability/service-graph
POST /api/v1/observability/metrics/query
POST /api/v1/observability/metrics/query_range
GET  /api/v1/observability/metrics/names
POST /api/v1/observability/metrics/export
GET  /api/v1/observability/alerts
POST /api/v1/observability/alerts
PUT  /api/v1/observability/alerts/:id
DELETE /api/v1/observability/alerts/:id
GET  /api/v1/observability/alerts/active
POST /api/v1/observability/alerts/:id/acknowledge
POST /api/v1/observability/alerts/:id/silence
GET  /api/v1/observability/jaeger-url
```

### Jaeger Integration

The Tracing page embeds the Jaeger UI via iframe. Configure the Jaeger URL via:
```
GET /api/v1/observability/jaeger-url
Response: { "url": "http://jaeger:16686" }
```

### Prometheus Integration

Metrics queries are executed against Prometheus via the backend API. The backend should proxy PromQL queries to Prometheus.

## Navigation

Observability pages are accessible via the sidebar:
- Tracing icon: Speed/Timeline
- Metrics icon: BarChart
- Alerts icon: NotificationsActive

## Styling

All components use Material-UI theming and follow the dark theme consistency requirement. Charts use Recharts with custom color schemes matching the application theme.

## Export Capabilities

Both Tracing and Metrics pages support CSV export:
- Tracing: Exports trace data with timestamps, services, operations, durations
- Metrics: Exports time-series data with timestamps and metric values

## Real-time Updates

- **Tracing**: Manual refresh or search-triggered updates
- **Metrics**: Auto-refresh with configurable intervals (10s - 5m)
- **Service Graph**: Auto-refresh every 30 seconds (configurable)
- **Alerts**: Manual refresh on tab view

## Error Handling

All components include:
- Loading states with spinners
- Error alerts with descriptive messages
- Graceful degradation when backend is unavailable
- Empty state handling with informative messages

## Accessibility

- Semantic HTML structure
- ARIA labels on interactive elements
- Keyboard navigation support
- Screen reader compatible
- Color contrast compliance

## Performance Considerations

- Lazy loading of large trace datasets (limit parameter)
- Debounced search inputs
- Memoized chart data transformations
- Efficient re-render prevention with React hooks
- Pagination support ready (backend implementation required)

## Future Enhancements

- Trace flamegraph visualization
- Custom dashboard creation
- Metric correlation analysis
- Alert rule templates
- Advanced PromQL query builder with autocomplete
- Service-level objectives (SLO) tracking
- Anomaly detection integration
- Log correlation with traces
