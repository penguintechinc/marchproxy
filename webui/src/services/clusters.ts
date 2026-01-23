/**
 * Clusters Service
 *
 * Provides HTTP client functions for cluster management operations.
 * Uses centralized apiClient for consistent request handling and authentication.
 */

import { apiClient } from './api';
import { Cluster, ApiResponse, PaginatedResponse } from './types';

export interface CreateClusterData {
  name: string;
  description?: string;
  syslog_server?: string;
  syslog_port?: number;
  auth_log_enabled?: boolean;
  netflow_log_enabled?: boolean;
  debug_log_enabled?: boolean;
}

export interface UpdateClusterData {
  name?: string;
  description?: string;
  syslog_server?: string;
  syslog_port?: number;
  auth_log_enabled?: boolean;
  netflow_log_enabled?: boolean;
  debug_log_enabled?: boolean;
}

export interface ClusterFilters {
  page?: number;
  page_size?: number;
  search?: string;
}

/**
 * Get all clusters
 * GET /api/v1/clusters
 */
export const getClusters = async (
  filters?: ClusterFilters
): Promise<PaginatedResponse<Cluster>> => {
  const response = await apiClient.get<PaginatedResponse<Cluster>>(
    '/api/v1/clusters',
    { params: filters }
  );
  return response.data;
};

/**
 * Get a single cluster by ID
 * GET /api/v1/clusters/{id}
 */
export const getCluster = async (id: number | string): Promise<Cluster> => {
  const response = await apiClient.get<Cluster>(`/api/v1/clusters/${id}`);
  return response.data;
};

/**
 * Create a new cluster
 * POST /api/v1/clusters
 */
export const createCluster = async (
  data: CreateClusterData
): Promise<Cluster> => {
  const response = await apiClient.post<ApiResponse<Cluster>>(
    '/api/v1/clusters',
    data
  );
  return response.data.data;
};

/**
 * Update an existing cluster
 * PATCH /api/v1/clusters/{id}
 */
export const updateCluster = async (
  id: number | string,
  data: UpdateClusterData
): Promise<Cluster> => {
  const response = await apiClient.patch<ApiResponse<Cluster>>(
    `/api/v1/clusters/${id}`,
    data
  );
  return response.data.data;
};

/**
 * Delete a cluster
 * DELETE /api/v1/clusters/{id}
 */
export const deleteCluster = async (id: number | string): Promise<void> => {
  await apiClient.delete(`/api/v1/clusters/${id}`);
};

/**
 * Rotate API key for a cluster
 * POST /api/v1/clusters/{id}/rotate-api-key
 */
export const rotateApiKey = async (
  id: number | string
): Promise<{ api_key: string }> => {
  const response = await apiClient.post<{ api_key: string }>(
    `/api/v1/clusters/${id}/rotate-api-key`
  );
  return response.data;
};
