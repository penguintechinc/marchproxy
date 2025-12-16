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
  description: string;
  syslog_server?: string;
  syslog_port?: number;
  auth_log_enabled: boolean;
  netflow_log_enabled: boolean;
  debug_log_enabled: boolean;
}

export interface UpdateClusterRequest {
  name?: string;
  description?: string;
  syslog_server?: string;
  syslog_port?: number;
  auth_log_enabled?: boolean;
  netflow_log_enabled?: boolean;
  debug_log_enabled?: boolean;
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
    const response = await apiClient.get<PaginatedResponse<Cluster>>('/api/clusters', {
      params
    });
    return response.data;
  },

  /**
   * Get a single cluster by ID
   */
  get: async (id: number): Promise<Cluster> => {
    const response = await apiClient.get<Cluster>(`/api/clusters/${id}`);
    return response.data;
  },

  /**
   * Create a new cluster
   */
  create: async (data: CreateClusterRequest): Promise<Cluster> => {
    const response = await apiClient.post<ApiResponse<Cluster>>('/api/clusters', data);
    return response.data.data;
  },

  /**
   * Update an existing cluster
   */
  update: async (id: number, data: UpdateClusterRequest): Promise<Cluster> => {
    const response = await apiClient.put<ApiResponse<Cluster>>(`/api/clusters/${id}`, data);
    return response.data.data;
  },

  /**
   * Delete a cluster
   */
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/clusters/${id}`);
  },

  /**
   * Rotate cluster API key
   */
  rotateApiKey: async (id: number): Promise<{ api_key: string }> => {
    const response = await apiClient.post<{ api_key: string }>(
      `/api/clusters/${id}/rotate-key`
    );
    return response.data;
  },

  /**
   * Get cluster statistics
   */
  getStats: async (id: number): Promise<{
    proxy_count: number;
    service_count: number;
    active_connections: number;
  }> => {
    const response = await apiClient.get(`/api/clusters/${id}/stats`);
    return response.data;
  }
};
