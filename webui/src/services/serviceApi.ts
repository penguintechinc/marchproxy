/**
 * Service API Service
 *
 * Handles all service-related API operations including CRUD,
 * token management, and service-to-service mappings.
 */

import { apiClient } from './api';
import { Service, ServiceToken, ApiResponse, PaginatedResponse } from './types';

export interface CreateServiceRequest {
  cluster_id: number;
  name: string;
  description: string;
  destination_fqdn: string;
  destination_port: string;
  protocol: 'TCP' | 'UDP' | 'ICMP' | 'HTTPS' | 'HTTP3';
  auth_method: 'base64_token' | 'jwt';
  token_ttl?: number;
  is_active: boolean;
}

export interface UpdateServiceRequest {
  name?: string;
  description?: string;
  destination_fqdn?: string;
  destination_port?: string;
  protocol?: 'TCP' | 'UDP' | 'ICMP' | 'HTTPS' | 'HTTP3';
  auth_method?: 'base64_token' | 'jwt';
  token_ttl?: number;
  is_active?: boolean;
}

export interface ServiceListParams {
  page?: number;
  page_size?: number;
  cluster_id?: number;
  search?: string;
  protocol?: string;
  is_active?: boolean;
}

export const serviceApi = {
  /**
   * Get all services with optional filtering
   */
  list: async (params?: ServiceListParams): Promise<PaginatedResponse<Service>> => {
    const response = await apiClient.get<PaginatedResponse<Service>>('/api/services', {
      params
    });
    return response.data;
  },

  /**
   * Get a single service by ID
   */
  get: async (id: number): Promise<Service> => {
    const response = await apiClient.get<Service>(`/api/services/${id}`);
    return response.data;
  },

  /**
   * Create a new service
   */
  create: async (data: CreateServiceRequest): Promise<Service> => {
    const response = await apiClient.post<ApiResponse<Service>>('/api/services', data);
    return response.data.data;
  },

  /**
   * Update an existing service
   */
  update: async (id: number, data: UpdateServiceRequest): Promise<Service> => {
    const response = await apiClient.put<ApiResponse<Service>>(`/api/services/${id}`, data);
    return response.data.data;
  },

  /**
   * Delete a service
   */
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/services/${id}`);
  },

  /**
   * Regenerate service token
   */
  regenerateToken: async (id: number): Promise<ServiceToken> => {
    const response = await apiClient.post<ServiceToken>(
      `/api/services/${id}/regenerate-token`
    );
    return response.data;
  },

  /**
   * Get service token
   */
  getToken: async (id: number): Promise<ServiceToken> => {
    const response = await apiClient.get<ServiceToken>(`/api/services/${id}/token`);
    return response.data;
  },

  /**
   * Get service mappings (source services that can access this service)
   */
  getMappings: async (id: number): Promise<Service[]> => {
    const response = await apiClient.get<Service[]>(`/api/services/${id}/mappings`);
    return response.data;
  },

  /**
   * Add service mapping
   */
  addMapping: async (serviceId: number, targetServiceId: number): Promise<void> => {
    await apiClient.post(`/api/services/${serviceId}/mappings`, {
      target_service_id: targetServiceId
    });
  },

  /**
   * Remove service mapping
   */
  removeMapping: async (serviceId: number, targetServiceId: number): Promise<void> => {
    await apiClient.delete(`/api/services/${serviceId}/mappings/${targetServiceId}`);
  }
};
