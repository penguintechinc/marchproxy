/**
 * API Client Service
 *
 * Centralized axios client for API requests with JWT authentication.
 */

import axios, { AxiosInstance, AxiosError } from 'axios';

// Base API URL from environment or default
const API_BASE_URL = import.meta.env.REACT_APP_API_URL || 'http://localhost:8000';

// Create axios instance
export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
});

// Request interceptor to add JWT token
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => {
    return response;
  },
  (error: AxiosError) => {
    // Handle 401 Unauthorized
    if (error.response?.status === 401) {
      // Clear token and redirect to login
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }

    // Handle 403 Forbidden (license issues)
    if (error.response?.status === 403) {
      const data: any = error.response.data;
      if (data?.detail?.feature) {
        console.warn(
          `Enterprise feature not available: ${data.detail.feature}`
        );
      }
    }

    return Promise.reject(error);
  }
);

// Helper functions
export const setAuthToken = (token: string) => {
  localStorage.setItem('auth_token', token);
};

export const clearAuthToken = () => {
  localStorage.removeItem('auth_token');
};

export const getAuthToken = (): string | null => {
  return localStorage.getItem('auth_token');
};
