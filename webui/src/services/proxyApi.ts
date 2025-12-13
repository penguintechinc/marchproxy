/**
 * Proxy API Service
 *
 * Handles all proxy-related API operations including listing,
 * status monitoring, and deregistration.
 */

import { apiClient } from './api';
import { Proxy, ProxyStats, PaginatedResponse } from './types';

export interface ProxyListParams {
  page?: number;
  page_size?: number;
  cluster_id?: number;
  status?: 'active' | 'inactive' | 'error';
  search?: string;
}

export interface ProxyMetrics {
  proxy_id: number;
  hostname: string;
  cpu_usage: number;
  memory_usage: number;
  connections_active: number;
  connections_total: number;
  bytes_sent: number;
  bytes_received: number;
  packets_sent: number;
  packets_received: number;
  errors: number;
  uptime: number;
  last_heartbeat: string;
}

export const proxyApi = {
  /**
   * Get all proxies with optional filtering
   */
  list: async (params?: ProxyListParams): Promise<PaginatedResponse<Proxy>> => {
    const response = await apiClient.get<PaginatedResponse<Proxy>>('/api/proxies', {
      params
    });
    return response.data;
  },

  /**
   * Get a single proxy by ID
   */
  get: async (id: number): Promise<Proxy> => {
    const response = await apiClient.get<Proxy>(`/api/proxies/${id}`);
    return response.data;
  },

  /**
   * Deregister a proxy
   */
  deregister: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/proxies/${id}`);
  },

  /**
   * Get proxy metrics
   */
  getMetrics: async (id: number): Promise<ProxyMetrics> => {
    const response = await apiClient.get<ProxyMetrics>(`/api/proxies/${id}/metrics`);
    return response.data;
  },

  /**
   * Get proxy statistics
   */
  getStats: async (id: number): Promise<ProxyStats> => {
    const response = await apiClient.get<ProxyStats>(`/api/proxies/${id}/stats`);
    return response.data;
  },

  /**
   * Get all proxy statistics for a cluster
   */
  getClusterStats: async (clusterId: number): Promise<ProxyStats[]> => {
    const response = await apiClient.get<ProxyStats[]>(
      `/api/clusters/${clusterId}/proxy-stats`
    );
    return response.data;
  },

  /**
   * Force heartbeat check for a proxy
   */
  checkHeartbeat: async (id: number): Promise<{ status: string; last_heartbeat: string }> => {
    const response = await apiClient.post(`/api/proxies/${id}/heartbeat`);
    return response.data;
  }
};
