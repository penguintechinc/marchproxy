/**
 * Services API Wrapper
 *
 * Provides a unified interface for service management operations.
 * Implements standard CRUD operations following RESTful conventions.
 *
 * Endpoints:
 * - GET /api/v1/services - List all services
 * - GET /api/v1/services/{id} - Get service details
 * - POST /api/v1/services - Create new service
 * - PUT /api/v1/services/{id} - Update service
 * - DELETE /api/v1/services/{id} - Delete service
 */

import { apiClient } from './api';
import {
  Service,
  PaginatedResponse,
  ApiResponse,
} from './types';

/**
 * Service creation request payload
 */
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

/**
 * Service update request payload
 */
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

/**
 * Service listing query parameters
 */
export interface ServiceListParams {
  page?: number;
  page_size?: number;
  cluster_id?: number;
  search?: string;
  protocol?: string;
  is_active?: boolean;
}

/**
 * Get all services with optional filtering and pagination
 *
 * @param params - Query parameters for filtering and pagination
 * @returns Promise resolving to paginated list of services
 *
 * @example
 * const services = await getServices({ cluster_id: 1, page: 1 });
 */
export const getServices = async (
  params?: ServiceListParams
): Promise<PaginatedResponse<Service>> => {
  const response = await apiClient.get<PaginatedResponse<Service>>(
    '/api/v1/services',
    { params }
  );
  return response.data;
};

/**
 * Get a single service by ID
 *
 * @param id - Service ID
 * @returns Promise resolving to service details
 *
 * @example
 * const service = await getService(42);
 */
export const getService = async (id: number): Promise<Service> => {
  const response = await apiClient.get<Service>(`/api/v1/services/${id}`);
  return response.data;
};

/**
 * Create a new service
 *
 * @param data - Service creation payload
 * @returns Promise resolving to created service with ID
 *
 * @example
 * const newService = await createService({
 *   cluster_id: 1,
 *   name: "API Service",
 *   description: "Backend API",
 *   destination_fqdn: "api.example.com",
 *   destination_port: "443",
 *   protocol: "HTTPS",
 *   auth_method: "jwt",
 *   is_active: true
 * });
 */
export const createService = async (
  data: CreateServiceRequest
): Promise<Service> => {
  const response = await apiClient.post<ApiResponse<Service>>(
    '/api/v1/services',
    data
  );
  return response.data.data;
};

/**
 * Update an existing service
 *
 * @param id - Service ID to update
 * @param data - Partial service update payload
 * @returns Promise resolving to updated service
 *
 * @example
 * const updated = await updateService(42, {
 *   description: "Updated description",
 *   is_active: false
 * });
 */
export const updateService = async (
  id: number,
  data: UpdateServiceRequest
): Promise<Service> => {
  const response = await apiClient.put<ApiResponse<Service>>(
    `/api/v1/services/${id}`,
    data
  );
  return response.data.data;
};

/**
 * Delete a service
 *
 * @param id - Service ID to delete
 * @returns Promise resolving when deletion is complete
 *
 * @example
 * await deleteService(42);
 */
export const deleteService = async (id: number): Promise<void> => {
  await apiClient.delete(`/api/v1/services/${id}`);
};

/**
 * Services API object - Legacy interface for backward compatibility
 * Provides CRUD operations via object methods
 *
 * @deprecated Use individual function exports instead
 *
 * @example
 * const services = await servicesApi.list({ cluster_id: 1 });
 * const service = await servicesApi.get(42);
 */
export const servicesApi = {
  /**
   * List services with optional filtering
   */
  list: async (params?: ServiceListParams): Promise<PaginatedResponse<Service>> => {
    return getServices(params);
  },

  /**
   * Get service details
   */
  get: async (id: number): Promise<Service> => {
    return getService(id);
  },

  /**
   * Create new service
   */
  create: async (data: CreateServiceRequest): Promise<Service> => {
    return createService(data);
  },

  /**
   * Update service
   */
  update: async (
    id: number,
    data: UpdateServiceRequest
  ): Promise<Service> => {
    return updateService(id, data);
  },

  /**
   * Delete service
   */
  delete: async (id: number): Promise<void> => {
    return deleteService(id);
  }
};

export default {
  getServices,
  getService,
  createService,
  updateService,
  deleteService,
  servicesApi
};
