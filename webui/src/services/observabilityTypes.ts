/**
 * Observability Type Definitions
 *
 * TypeScript types for tracing, metrics, alerts, and service graphs.
 */

// Time Range Types
export interface TimeRange {
  start: string; // ISO 8601 timestamp
  end: string; // ISO 8601 timestamp
}

// Trace Types
export interface Span {
  traceID: string;
  spanID: string;
  operationName: string;
  references: SpanReference[];
  startTime: number;
  duration: number;
  tags: Tag[];
  logs: Log[];
  processID: string;
  warnings?: string[];
}

export interface SpanReference {
  refType: 'CHILD_OF' | 'FOLLOWS_FROM';
  traceID: string;
  spanID: string;
}

export interface Tag {
  key: string;
  type: 'string' | 'bool' | 'int64' | 'float64' | 'binary';
  value: any;
}

export interface Log {
  timestamp: number;
  fields: Tag[];
}

export interface Process {
  serviceName: string;
  tags: Tag[];
}

export interface Trace {
  traceID: string;
  spans: Span[];
  processes: { [key: string]: Process };
  warnings?: string[];
}

export interface TraceSearchParams {
  service?: string;
  operation?: string;
  tags?: { [key: string]: string };
  minDuration?: string; // e.g., "100ms"
  maxDuration?: string; // e.g., "5s"
  start: string; // ISO 8601 timestamp
  end: string; // ISO 8601 timestamp
  limit?: number;
  lookback?: string; // e.g., "1h", "24h"
}

// Service Graph Types
export interface ServiceNode {
  id: string;
  name: string;
  type: 'service' | 'external';
  health: 'healthy' | 'degraded' | 'unhealthy';
  requestRate: number; // requests per second
  errorRate: number; // percentage
  p95Latency: number; // milliseconds
}

export interface ServiceEdge {
  source: string;
  target: string;
  requestRate: number; // requests per second
  errorRate: number; // percentage
  p95Latency: number; // milliseconds
}

export interface ServiceDependency {
  nodes: ServiceNode[];
  edges: ServiceEdge[];
}

// Metrics Types
export interface MetricQuery {
  query: string; // PromQL query
  start?: string; // ISO 8601 timestamp
  end?: string; // ISO 8601 timestamp
  step?: string; // e.g., "15s", "1m"
  timeout?: string; // e.g., "30s"
}

export interface MetricValue {
  timestamp: number;
  value: number;
}

export interface MetricSeries {
  metric: { [key: string]: string };
  values: MetricValue[];
}

export interface MetricQueryResponse {
  resultType: 'matrix' | 'vector' | 'scalar' | 'string';
  result: MetricSeries[];
}

// Alert Types
export interface AlertRule {
  id: number;
  name: string;
  description: string;
  enabled: boolean;
  severity: 'critical' | 'warning' | 'info';
  query: string; // PromQL query
  threshold: number;
  operator: '>' | '<' | '=' | '>=' | '<=';
  duration: string; // e.g., "5m" - alert fires after threshold breached
  labels: { [key: string]: string };
  annotations: { [key: string]: string };
  notification_channels: NotificationChannel[];
  created_at: string;
  updated_at: string;
  created_by: string;
}

export interface NotificationChannel {
  id: number;
  type: 'email' | 'slack' | 'webhook' | 'pagerduty';
  name: string;
  config: EmailConfig | SlackConfig | WebhookConfig | PagerDutyConfig;
  enabled: boolean;
}

export interface EmailConfig {
  to: string[];
  subject_template?: string;
  body_template?: string;
}

export interface SlackConfig {
  webhook_url: string;
  channel?: string;
  username?: string;
  icon_emoji?: string;
}

export interface WebhookConfig {
  url: string;
  method: 'POST' | 'PUT';
  headers?: { [key: string]: string };
  body_template?: string;
}

export interface PagerDutyConfig {
  integration_key: string;
  severity?: string;
}

export interface Alert {
  id: number;
  rule_id: number;
  rule_name: string;
  severity: 'critical' | 'warning' | 'info';
  status: 'firing' | 'acknowledged' | 'resolved' | 'silenced';
  started_at: string;
  resolved_at?: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  silenced_until?: string;
  labels: { [key: string]: string };
  annotations: { [key: string]: string };
  value: number;
  message: string;
}

export interface CreateAlertRequest {
  name: string;
  description: string;
  enabled: boolean;
  severity: 'critical' | 'warning' | 'info';
  query: string;
  threshold: number;
  operator: '>' | '<' | '=' | '>=' | '<=';
  duration: string;
  labels?: { [key: string]: string };
  annotations?: { [key: string]: string };
  notification_channel_ids: number[];
}

export interface UpdateAlertRequest {
  name?: string;
  description?: string;
  enabled?: boolean;
  severity?: 'critical' | 'warning' | 'info';
  query?: string;
  threshold?: number;
  operator?: '>' | '<' | '=' | '>=' | '<=';
  duration?: string;
  labels?: { [key: string]: string };
  annotations?: { [key: string]: string };
  notification_channel_ids?: number[];
}

// Chart Data Types
export interface ChartDataPoint {
  timestamp: number;
  value: number;
  label?: string;
}

export interface TimeSeriesData {
  name: string;
  data: ChartDataPoint[];
  color?: string;
}

// Dashboard Types
export interface DashboardWidget {
  id: string;
  type: 'chart' | 'stat' | 'table' | 'graph';
  title: string;
  query: string;
  timeRange?: TimeRange;
  refreshInterval?: number; // seconds
  position: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
}

export interface Dashboard {
  id: number;
  name: string;
  description: string;
  widgets: DashboardWidget[];
  created_at: string;
  updated_at: string;
}

// Filter Types
export interface TraceFilter {
  services: string[];
  operations: string[];
  minDuration?: number;
  maxDuration?: number;
  tags: { [key: string]: string };
  timeRange: TimeRange;
}

export interface MetricFilter {
  metrics: string[];
  services: string[];
  timeRange: TimeRange;
  aggregation?: 'avg' | 'sum' | 'min' | 'max' | 'count';
}
