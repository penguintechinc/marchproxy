/**
 * Authentication Store using Zustand
 * Manages authentication state across the application
 */

import { create } from 'zustand';
import { User, LoginRequest } from '@services/types';
import { authService } from '@services/auth';
import { getAuthToken } from '@services/api';

interface AuthStore {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  login: (credentials: LoginRequest) => Promise<void>;
  logout: () => Promise<void>;
  loadUser: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  user: null,
  token: getAuthToken(),
  isAuthenticated: !!getAuthToken(),
  isLoading: false,
  error: null,

  login: async (credentials: LoginRequest) => {
    set({ isLoading: true, error: null });
    try {
      const response = await authService.login(credentials);

      // Check if 2FA is required
      if (response.requires_2fa) {
        set({
          isLoading: false,
          error: 'Please enter your 2FA code',
        });
        return;
      }

      // Create user object from response
      const user: User = {
        id: response.user_id,
        username: response.username,
        email: response.email,
        role: response.is_admin ? 'administrator' : 'service_owner',
        is_active: true,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      set({
        user,
        token: response.access_token || null,
        isAuthenticated: !!response.access_token,
        isLoading: false,
        error: null,
      });
    } catch (error: any) {
      const errorMessage =
        error.response?.data?.detail || 'Login failed. Please try again.';
      set({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
        error: errorMessage,
      });
      throw error;
    }
  },

  logout: async () => {
    set({ isLoading: true });
    try {
      await authService.logout();
    } finally {
      set({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
        error: null,
      });
    }
  },

  loadUser: async () => {
    const token = getAuthToken();
    if (!token) {
      set({ isAuthenticated: false, user: null });
      return;
    }

    set({ isLoading: true });
    try {
      const user = await authService.getCurrentUser();
      set({
        user,
        token,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });
    } catch (error) {
      set({
        user: null,
        token: null,
        isAuthenticated: false,
        isLoading: false,
        error: 'Failed to load user',
      });
    }
  },

  clearError: () => {
    set({ error: null });
  },
}));
