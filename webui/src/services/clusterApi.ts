/**
 * Cluster API Service
 *
 * Handles all cluster-related API operations including CRUD,
 * API key rotation, and cluster statistics.
 */

import { apiClient } from './api';
import { Cluster, ApiResponse, PaginatedResponse } from './types';

export interface CreateClusterRequest {
  name: string;
  description?: string;
  syslog_endpoint?: string;
  log_auth?: boolean;
  log_netflow?: boolean;
  log_debug?: boolean;
  max_proxies?: number;
}

export interface UpdateClusterRequest {
  name?: string;
  description?: string;
  syslog_endpoint?: string;
  log_auth?: boolean;
  log_netflow?: boolean;
  log_debug?: boolean;
  max_proxies?: number;
  is_active?: boolean;
}

export interface ClusterListParams {
  page?: number;
  page_size?: number;
  search?: string;
}

export const clusterApi = {
  /**
   * Get all clusters with optional pagination and search
   */
  list: async (params?: ClusterListParams): Promise<PaginatedResponse<Cluster>> => {
    const response = await apiClient.get<{ total: number; clusters: Cluster[] }>('/api/v1/clusters', {
      params
    });
    // Map backend response to PaginatedResponse format
    return {
      items: response.data.clusters,
      total: response.data.total,
      page: params?.page || 1,
      page_size: params?.page_size || 100,
      total_pages: Math.ceil(response.data.total / (params?.page_size || 100))
    };
  },

  /**
   * Get a single cluster by ID
   */
  get: async (id: number): Promise<Cluster> => {
    const response = await apiClient.get<Cluster>(`/api/v1/clusters/${id}`);
    return response.data;
  },

  /**
   * Create a new cluster
   */
  create: async (data: CreateClusterRequest): Promise<Cluster> => {
    const response = await apiClient.post<Cluster>('/api/v1/clusters', data);
    return response.data;
  },

  /**
   * Update an existing cluster
   */
  update: async (id: number, data: UpdateClusterRequest): Promise<Cluster> => {
    const response = await apiClient.patch<Cluster>(`/api/v1/clusters/${id}`, data);
    return response.data;
  },

  /**
   * Delete a cluster
   */
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/v1/clusters/${id}`);
  },

  /**
   * Rotate cluster API key
   */
  rotateApiKey: async (id: number): Promise<{ api_key: string }> => {
    const response = await apiClient.post<{ new_api_key: string }>(
      `/api/v1/clusters/${id}/rotate-api-key`
    );
    return { api_key: response.data.new_api_key };
  },

  /**
   * Get cluster statistics
   */
  getStats: async (id: number): Promise<{
    proxy_count: number;
    service_count: number;
    active_connections: number;
  }> => {
    const response = await apiClient.get(`/api/v1/clusters/${id}/stats`);
    return response.data;
  }
};
