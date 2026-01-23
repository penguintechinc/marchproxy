/**
 * Users API Service
 *
 * Handles all user-related API operations including CRUD,
 * user management, and role assignments.
 */

import { apiClient } from './api';
import { User, ApiResponse, PaginatedResponse } from './types';

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  role: 'administrator' | 'service_owner';
  clusters?: number[];
}

export interface UpdateUserRequest {
  username?: string;
  email?: string;
  role?: 'administrator' | 'service_owner';
  is_active?: boolean;
  clusters?: number[];
}

export interface UserListParams {
  page?: number;
  page_size?: number;
  search?: string;
  role?: 'administrator' | 'service_owner';
}

export const usersApi = {
  /**
   * Get all users with optional pagination and search
   */
  list: async (params?: UserListParams): Promise<PaginatedResponse<User>> => {
    const response = await apiClient.get<PaginatedResponse<User>>('/api/v1/users', {
      params
    });
    return response.data;
  },

  /**
   * Get a single user by ID
   */
  get: async (id: number): Promise<User> => {
    const response = await apiClient.get<User>(`/api/v1/users/${id}`);
    return response.data;
  },

  /**
   * Create a new user
   */
  create: async (data: CreateUserRequest): Promise<User> => {
    const response = await apiClient.post<ApiResponse<User>>('/api/v1/users', data);
    return response.data.data;
  },

  /**
   * Update an existing user
   */
  update: async (id: number, data: UpdateUserRequest): Promise<User> => {
    const response = await apiClient.put<ApiResponse<User>>(`/api/v1/users/${id}`, data);
    return response.data.data;
  },

  /**
   * Delete a user
   */
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/v1/users/${id}`);
  }
};
