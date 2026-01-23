/**
 * Authentication Service
 * Handles user login, logout, and token management
 */

import { apiClient, setAuthToken, clearAuthToken } from './api';
import { LoginRequest, LoginResponse, User } from './types';

export const authService = {
  /**
   * Login user with username and password
   */
  async login(credentials: LoginRequest): Promise<LoginResponse> {
    const response = await apiClient.post<LoginResponse>(
      '/api/v1/auth/login',
      credentials
    );

    if (response.data.access_token) {
      setAuthToken(response.data.access_token);
    }

    return response.data;
  },

  /**
   * Logout user and clear token
   */
  async logout(): Promise<void> {
    try {
      await apiClient.post('/api/v1/auth/logout');
    } finally {
      clearAuthToken();
    }
  },

  /**
   * Get current user profile
   */
  async getCurrentUser(): Promise<User> {
    const response = await apiClient.get<User>('/api/v1/auth/me');
    return response.data;
  },

  /**
   * Refresh authentication token
   */
  async refreshToken(): Promise<string> {
    const response = await apiClient.post<{ token: string }>(
      '/api/v1/auth/refresh'
    );

    if (response.data.token) {
      setAuthToken(response.data.token);
    }

    return response.data.token;
  },

  /**
   * Change user password
   */
  async changePassword(
    currentPassword: string,
    newPassword: string
  ): Promise<void> {
    await apiClient.post('/api/v1/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    });
  },

  /**
   * Enable 2FA for current user
   */
  async enable2FA(): Promise<{ secret: string; qr_code: string }> {
    const response = await apiClient.post<{
      secret: string;
      qr_code: string;
    }>('/api/v1/auth/2fa/enable');
    return response.data;
  },

  /**
   * Verify 2FA code
   */
  async verify2FA(code: string): Promise<void> {
    await apiClient.post('/api/v1/auth/2fa/verify', { code });
  },

  /**
   * Disable 2FA for current user
   */
  async disable2FA(code: string): Promise<void> {
    await apiClient.post('/api/v1/auth/2fa/disable', { code });
  },
};
