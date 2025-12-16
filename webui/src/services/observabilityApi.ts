/**
 * Observability API Service
 *
 * Handles all observability-related API calls including tracing, metrics,
 * service graphs, and alerting.
 */

import { apiClient } from './api';
import {
  Trace,
  TraceSearchParams,
  ServiceDependency,
  MetricQuery,
  MetricQueryResponse,
  Alert,
  AlertRule,
  CreateAlertRequest,
  UpdateAlertRequest,
  TimeRange,
} from './observabilityTypes';

/**
 * Fetch traces from Jaeger backend
 */
export const getTraces = async (
  params: TraceSearchParams
): Promise<Trace[]> => {
  const response = await apiClient.get('/api/v1/observability/traces', {
    params,
  });
  return response.data.data;
};

/**
 * Fetch single trace by ID
 */
export const getTraceById = async (traceId: string): Promise<Trace> => {
  const response = await apiClient.get(
    `/api/v1/observability/traces/${traceId}`
  );
  return response.data.data;
};

/**
 * Get service dependency graph
 */
export const getServiceGraph = async (
  timeRange?: TimeRange
): Promise<ServiceDependency[]> => {
  const response = await apiClient.get('/api/v1/observability/service-graph', {
    params: timeRange,
  });
  return response.data.data;
};

/**
 * Query metrics from Prometheus
 */
export const queryMetrics = async (
  query: MetricQuery
): Promise<MetricQueryResponse> => {
  const response = await apiClient.post('/api/v1/observability/metrics/query', query);
  return response.data.data;
};

/**
 * Query range metrics from Prometheus
 */
export const queryRangeMetrics = async (
  query: MetricQuery
): Promise<MetricQueryResponse> => {
  const response = await apiClient.post(
    '/api/v1/observability/metrics/query_range',
    query
  );
  return response.data.data;
};

/**
 * Get available metric names
 */
export const getMetricNames = async (): Promise<string[]> => {
  const response = await apiClient.get('/api/v1/observability/metrics/names');
  return response.data.data;
};

/**
 * Get all alert rules
 */
export const getAlerts = async (): Promise<AlertRule[]> => {
  const response = await apiClient.get('/api/v1/observability/alerts');
  return response.data.data;
};

/**
 * Get alert rule by ID
 */
export const getAlertById = async (id: number): Promise<AlertRule> => {
  const response = await apiClient.get(`/api/v1/observability/alerts/${id}`);
  return response.data.data;
};

/**
 * Create new alert rule
 */
export const createAlert = async (
  alert: CreateAlertRequest
): Promise<AlertRule> => {
  const response = await apiClient.post('/api/v1/observability/alerts', alert);
  return response.data.data;
};

/**
 * Update existing alert rule
 */
export const updateAlert = async (
  id: number,
  alert: UpdateAlertRequest
): Promise<AlertRule> => {
  const response = await apiClient.put(
    `/api/v1/observability/alerts/${id}`,
    alert
  );
  return response.data.data;
};

/**
 * Delete alert rule
 */
export const deleteAlert = async (id: number): Promise<void> => {
  await apiClient.delete(`/api/v1/observability/alerts/${id}`);
};

/**
 * Get active alerts/incidents
 */
export const getActiveAlerts = async (): Promise<Alert[]> => {
  const response = await apiClient.get('/api/v1/observability/alerts/active');
  return response.data.data;
};

/**
 * Acknowledge alert
 */
export const acknowledgeAlert = async (
  id: number,
  note?: string
): Promise<void> => {
  await apiClient.post(`/api/v1/observability/alerts/${id}/acknowledge`, {
    note,
  });
};

/**
 * Silence alert
 */
export const silenceAlert = async (
  id: number,
  duration: number,
  note?: string
): Promise<void> => {
  await apiClient.post(`/api/v1/observability/alerts/${id}/silence`, {
    duration,
    note,
  });
};

/**
 * Get Jaeger UI URL
 */
export const getJaegerUrl = async (): Promise<string> => {
  const response = await apiClient.get('/api/v1/observability/jaeger-url');
  return response.data.data.url;
};

/**
 * Get service list for filtering
 */
export const getServices = async (): Promise<string[]> => {
  const response = await apiClient.get('/api/v1/observability/services');
  return response.data.data;
};

/**
 * Get operations for a service
 */
export const getOperations = async (service: string): Promise<string[]> => {
  const response = await apiClient.get('/api/v1/observability/operations', {
    params: { service },
  });
  return response.data.data;
};

/**
 * Export traces as CSV
 */
export const exportTraces = async (
  params: TraceSearchParams
): Promise<Blob> => {
  const response = await apiClient.get('/api/v1/observability/traces/export', {
    params,
    responseType: 'blob',
  });
  return response.data;
};

/**
 * Export metrics as CSV
 */
export const exportMetrics = async (query: MetricQuery): Promise<Blob> => {
  const response = await apiClient.post(
    '/api/v1/observability/metrics/export',
    query,
    {
      responseType: 'blob',
    }
  );
  return response.data;
};
