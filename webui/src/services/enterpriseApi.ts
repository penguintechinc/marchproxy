/**
 * Enterprise API Service
 *
 * API client for Enterprise-only features:
 * - Traffic Shaping & QoS
 * - Multi-Cloud Routing
 * - Cost Analytics
 * - NUMA Configuration
 */

import axios from 'axios';
import type {
  QoSPolicy,
  CloudRoute,
  BackendHealth,
  CloudBackendLocation,
  CostAnalytics,
  CostTimeSeries,
  CostOptimization,
  NUMATopology,
  NUMAConfig,
  NUMAMetrics,
  RoutingAlgorithm,
  ApiResponse,
} from './types';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '/api';

// Get auth token from localStorage
const getAuthToken = (): string | null => {
  return localStorage.getItem('auth_token');
};

// Create axios instance with auth
const apiClient = axios.create({
  baseURL: API_BASE,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Traffic Shaping & QoS API

export const getQoSPolicies = async (serviceId?: number): Promise<QoSPolicy[]> => {
  const url = serviceId ? `/qos?service_id=${serviceId}` : '/qos';
  const response = await apiClient.get<ApiResponse<QoSPolicy[]>>(url);
  return response.data.data;
};

export const getQoSPolicy = async (id: number): Promise<QoSPolicy> => {
  const response = await apiClient.get<ApiResponse<QoSPolicy>>(`/qos/${id}`);
  return response.data.data;
};

export const createQoSPolicy = async (policy: Partial<QoSPolicy>): Promise<QoSPolicy> => {
  const response = await apiClient.post<ApiResponse<QoSPolicy>>('/qos', policy);
  return response.data.data;
};

export const updateQoSPolicy = async (
  id: number,
  policy: Partial<QoSPolicy>
): Promise<QoSPolicy> => {
  const response = await apiClient.put<ApiResponse<QoSPolicy>>(`/qos/${id}`, policy);
  return response.data.data;
};

export const deleteQoSPolicy = async (id: number): Promise<void> => {
  await apiClient.delete(`/qos/${id}`);
};

// Multi-Cloud Routing API

export const getCloudRoutes = async (serviceId?: number): Promise<CloudRoute[]> => {
  const url = serviceId ? `/routes?service_id=${serviceId}` : '/routes';
  const response = await apiClient.get<ApiResponse<CloudRoute[]>>(url);
  return response.data.data;
};

export const getCloudRoute = async (id: number): Promise<CloudRoute> => {
  const response = await apiClient.get<ApiResponse<CloudRoute>>(`/routes/${id}`);
  return response.data.data;
};

export const createCloudRoute = async (route: Partial<CloudRoute>): Promise<CloudRoute> => {
  const response = await apiClient.post<ApiResponse<CloudRoute>>('/routes', route);
  return response.data.data;
};

export const updateCloudRoute = async (
  id: number,
  route: Partial<CloudRoute>
): Promise<CloudRoute> => {
  const response = await apiClient.put<ApiResponse<CloudRoute>>(`/routes/${id}`, route);
  return response.data.data;
};

export const deleteCloudRoute = async (id: number): Promise<void> => {
  await apiClient.delete(`/routes/${id}`);
};

export const getBackendHealth = async (serviceId?: number): Promise<BackendHealth[]> => {
  const url = serviceId ? `/routes/health?service_id=${serviceId}` : '/routes/health';
  const response = await apiClient.get<ApiResponse<BackendHealth[]>>(url);
  return response.data.data;
};

export const getCloudBackendLocations = async (): Promise<CloudBackendLocation[]> => {
  const response = await apiClient.get<ApiResponse<CloudBackendLocation[]>>(
    '/routes/locations'
  );
  return response.data.data;
};

export const getRoutingAlgorithms = async (): Promise<RoutingAlgorithm[]> => {
  const response = await apiClient.get<ApiResponse<RoutingAlgorithm[]>>(
    '/routes/algorithms'
  );
  return response.data.data;
};

export const updateRoutingAlgorithm = async (
  serviceId: number,
  algorithm: string
): Promise<void> => {
  await apiClient.put(`/routes/algorithm/${serviceId}`, { algorithm });
};

// Cost Analytics API

export const getCostAnalytics = async (
  startDate: string,
  endDate: string,
  serviceId?: number
): Promise<CostAnalytics> => {
  const params: Record<string, string> = {
    start_date: startDate,
    end_date: endDate,
  };
  if (serviceId) {
    params.service_id = serviceId.toString();
  }
  const response = await apiClient.get<ApiResponse<CostAnalytics>>('/analytics/cost', {
    params,
  });
  return response.data.data;
};

export const getCostTimeSeries = async (
  startDate: string,
  endDate: string,
  groupBy: 'hour' | 'day' | 'week' | 'month',
  provider?: string
): Promise<CostTimeSeries[]> => {
  const params: Record<string, string> = {
    start_date: startDate,
    end_date: endDate,
    group_by: groupBy,
  };
  if (provider) {
    params.provider = provider;
  }
  const response = await apiClient.get<ApiResponse<CostTimeSeries[]>>(
    '/analytics/cost/timeseries',
    { params }
  );
  return response.data.data;
};

export const getCostOptimizations = async (): Promise<CostOptimization[]> => {
  const response = await apiClient.get<ApiResponse<CostOptimization[]>>(
    '/analytics/cost/optimizations'
  );
  return response.data.data;
};

export const exportCostReport = async (
  startDate: string,
  endDate: string,
  format: 'csv' | 'pdf'
): Promise<Blob> => {
  const response = await apiClient.get('/analytics/cost/export', {
    params: {
      start_date: startDate,
      end_date: endDate,
      format,
    },
    responseType: 'blob',
  });
  return response.data;
};

// NUMA Configuration API

export const getNUMATopology = async (proxyId: number): Promise<NUMATopology> => {
  const response = await apiClient.get<ApiResponse<NUMATopology>>(
    `/numa/topology/${proxyId}`
  );
  return response.data.data;
};

export const getNUMAConfig = async (proxyId: number): Promise<NUMAConfig> => {
  const response = await apiClient.get<ApiResponse<NUMAConfig>>(`/numa/config/${proxyId}`);
  return response.data.data;
};

export const updateNUMAConfig = async (
  proxyId: number,
  config: Partial<NUMAConfig>
): Promise<NUMAConfig> => {
  const response = await apiClient.put<ApiResponse<NUMAConfig>>(
    `/numa/config/${proxyId}`,
    config
  );
  return response.data.data;
};

export const getNUMAMetrics = async (proxyId: number): Promise<NUMAMetrics[]> => {
  const response = await apiClient.get<ApiResponse<NUMAMetrics[]>>(
    `/numa/metrics/${proxyId}`
  );
  return response.data.data;
};

export const resetNUMAConfig = async (proxyId: number): Promise<void> => {
  await apiClient.post(`/numa/config/${proxyId}/reset`);
};
