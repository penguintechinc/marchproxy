/**
 * React hook for Observability and Distributed Tracing API
 *
 * Manages tracing configurations and observability settings.
 */

import { useState, useCallback } from 'react';
import { apiClient } from '../services/api';

export type TracingBackend = 'jaeger' | 'zipkin' | 'otlp';
export type SamplingStrategy = 'always' | 'never' | 'probabilistic' | 'rate_limit' | 'error_only' | 'adaptive';
export type SpanExporter = 'grpc' | 'http' | 'thrift';

export interface TracingStats {
  total_spans: number;
  sampled_spans: number;
  dropped_spans: number;
  error_spans: number;
  avg_span_duration_ms?: number;
  last_export?: string;
}

export interface TracingConfig {
  id: number;
  name: string;
  description?: string;
  cluster_id: number;
  backend: TracingBackend;
  endpoint: string;
  exporter: SpanExporter;
  sampling_strategy: SamplingStrategy;
  sampling_rate: number;
  max_traces_per_second?: number;
  include_request_headers: boolean;
  include_response_headers: boolean;
  include_request_body: boolean;
  include_response_body: boolean;
  max_attribute_length: number;
  service_name: string;
  custom_tags?: Record<string, string>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  stats?: TracingStats;
}

export interface TracingConfigCreate {
  name: string;
  description?: string;
  cluster_id: number;
  backend: TracingBackend;
  endpoint: string;
  exporter?: SpanExporter;
  sampling_strategy?: SamplingStrategy;
  sampling_rate?: number;
  max_traces_per_second?: number;
  include_request_headers?: boolean;
  include_response_headers?: boolean;
  include_request_body?: boolean;
  include_response_body?: boolean;
  max_attribute_length?: number;
  service_name?: string;
  custom_tags?: Record<string, string>;
  enabled?: boolean;
}

export interface TracingConfigUpdate {
  name?: string;
  description?: string;
  backend?: TracingBackend;
  endpoint?: string;
  exporter?: SpanExporter;
  sampling_strategy?: SamplingStrategy;
  sampling_rate?: number;
  max_traces_per_second?: number;
  include_request_headers?: boolean;
  include_response_headers?: boolean;
  include_request_body?: boolean;
  include_response_body?: boolean;
  max_attribute_length?: number;
  service_name?: string;
  custom_tags?: Record<string, string>;
  enabled?: boolean;
}

export interface UseObservabilityReturn {
  configs: TracingConfig[];
  loading: boolean;
  error: string | null;
  hasAccess: boolean;
  fetchConfigs: (clusterId?: number) => Promise<void>;
  createConfig: (config: TracingConfigCreate) => Promise<TracingConfig | null>;
  updateConfig: (id: number, config: TracingConfigUpdate) => Promise<TracingConfig | null>;
  deleteConfig: (id: number) => Promise<boolean>;
  getStats: (id: number) => Promise<TracingStats | null>;
  enableConfig: (id: number) => Promise<boolean>;
  disableConfig: (id: number) => Promise<boolean>;
  testConfig: (id: number, sendTestSpan?: boolean) => Promise<any>;
  searchSpans: (filters: any) => Promise<any>;
}

export const useObservability = (): UseObservabilityReturn => {
  const [configs, setConfigs] = useState<TracingConfig[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [hasAccess, setHasAccess] = useState<boolean>(true);

  const fetchConfigs = useCallback(async (clusterId?: number) => {
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams();
      if (clusterId) params.append('cluster_id', clusterId.toString());

      const response = await apiClient.get(
        `/api/v1/observability/tracing?${params.toString()}`
      );

      setConfigs(response.data);
      setHasAccess(true);
    } catch (err: any) {
      if (err.response?.status === 403) {
        setHasAccess(false);
        setError('Enterprise feature not available. Please upgrade your license.');
      } else {
        setError(err.message || 'Failed to fetch tracing configurations');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const createConfig = useCallback(async (
    config: TracingConfigCreate
  ): Promise<TracingConfig | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        '/api/v1/observability/tracing',
        config
      );

      const newConfig = response.data;
      setConfigs(prev => [...prev, newConfig]);
      return newConfig;
    } catch (err: any) {
      if (err.response?.status === 403) {
        setHasAccess(false);
        setError('Enterprise feature not available. Please upgrade your license.');
      } else {
        setError(err.message || 'Failed to create tracing configuration');
      }
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const updateConfig = useCallback(async (
    id: number,
    config: TracingConfigUpdate
  ): Promise<TracingConfig | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.put(
        `/api/v1/observability/tracing/${id}`,
        config
      );

      const updatedConfig = response.data;
      setConfigs(prev =>
        prev.map(c => c.id === id ? updatedConfig : c)
      );
      return updatedConfig;
    } catch (err: any) {
      setError(err.message || 'Failed to update tracing configuration');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const deleteConfig = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      await apiClient.delete(`/api/v1/observability/tracing/${id}`);
      setConfigs(prev => prev.filter(c => c.id !== id));
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to delete tracing configuration');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const getStats = useCallback(async (id: number): Promise<TracingStats | null> => {
    try {
      const response = await apiClient.get(
        `/api/v1/observability/tracing/${id}/stats`
      );
      return response.data;
    } catch (err: any) {
      setError(err.message || 'Failed to fetch tracing stats');
      return null;
    }
  }, []);

  const enableConfig = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        `/api/v1/observability/tracing/${id}/enable`
      );

      const updatedConfig = response.data;
      setConfigs(prev =>
        prev.map(c => c.id === id ? updatedConfig : c)
      );
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to enable tracing configuration');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const disableConfig = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        `/api/v1/observability/tracing/${id}/disable`
      );

      const updatedConfig = response.data;
      setConfigs(prev =>
        prev.map(c => c.id === id ? updatedConfig : c)
      );
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to disable tracing configuration');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const testConfig = useCallback(async (
    id: number,
    sendTestSpan: boolean = true
  ): Promise<any> => {
    try {
      const response = await apiClient.post(
        `/api/v1/observability/tracing/${id}/test`,
        null,
        { params: { send_test_span: sendTestSpan } }
      );
      return response.data;
    } catch (err: any) {
      setError(err.message || 'Failed to test tracing configuration');
      return null;
    }
  }, []);

  const searchSpans = useCallback(async (filters: any): Promise<any> => {
    try {
      const response = await apiClient.get(
        '/api/v1/observability/spans/search',
        { params: filters }
      );
      return response.data;
    } catch (err: any) {
      setError(err.message || 'Failed to search spans');
      return null;
    }
  }, []);

  return {
    configs,
    loading,
    error,
    hasAccess,
    fetchConfigs,
    createConfig,
    updateConfig,
    deleteConfig,
    getStats,
    enableConfig,
    disableConfig,
    testConfig,
    searchSpans
  };
};
