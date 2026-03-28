/**
 * Tests for Authentication Store (authStore.ts)
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@services/auth', () => ({
  authService: {
    login: vi.fn(),
    logout: vi.fn(),
    getCurrentUser: vi.fn(),
  },
}));

vi.mock('@services/api', () => ({
  getAuthToken: vi.fn(() => null),
  setAuthToken: vi.fn(),
  clearAuthToken: vi.fn(),
}));

import { useAuthStore } from '../authStore';
import { authService } from '@services/auth';
import { getAuthToken } from '@services/api';

const mockAuthService = authService as {
  login: ReturnType<typeof vi.fn>;
  logout: ReturnType<typeof vi.fn>;
  getCurrentUser: ReturnType<typeof vi.fn>;
};
const mockGetAuthToken = getAuthToken as ReturnType<typeof vi.fn>;

const initialState = {
  user: null,
  token: null,
  isAuthenticated: false,
  isLoading: false,
  error: null,
};

describe('useAuthStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetAuthToken.mockReturnValue(null);
    useAuthStore.setState(initialState);
  });

  describe('initial state', () => {
    it('has null user', () => {
      expect(useAuthStore.getState().user).toBeNull();
    });

    it('has isAuthenticated false', () => {
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('has null token', () => {
      expect(useAuthStore.getState().token).toBeNull();
    });

    it('has null error', () => {
      expect(useAuthStore.getState().error).toBeNull();
    });

    it('has isLoading false', () => {
      expect(useAuthStore.getState().isLoading).toBe(false);
    });
  });

  describe('login', () => {
    const credentials = { username: 'admin', password: 'secret' };

    it('sets user and isAuthenticated=true on success', async () => {
      mockAuthService.login.mockResolvedValue({
        access_token: 'tok123',
        user_id: 1,
        username: 'admin',
        email: 'admin@example.com',
        is_admin: true,
      });

      await useAuthStore.getState().login(credentials);

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(true);
      expect(state.user).not.toBeNull();
      expect(state.user?.username).toBe('admin');
    });

    it('sets token on success', async () => {
      mockAuthService.login.mockResolvedValue({
        access_token: 'tok123',
        user_id: 1,
        username: 'admin',
        email: 'admin@example.com',
        is_admin: false,
      });

      await useAuthStore.getState().login(credentials);

      expect(useAuthStore.getState().token).toBe('tok123');
    });

    it('sets user role to administrator when is_admin=true', async () => {
      mockAuthService.login.mockResolvedValue({
        access_token: 'tok123',
        user_id: 1,
        username: 'admin',
        email: 'admin@example.com',
        is_admin: true,
      });

      await useAuthStore.getState().login(credentials);

      expect(useAuthStore.getState().user?.role).toBe('administrator');
    });

    it('sets user role to service_owner when is_admin=false', async () => {
      mockAuthService.login.mockResolvedValue({
        access_token: 'tok123',
        user_id: 2,
        username: 'user',
        email: 'user@example.com',
        is_admin: false,
      });

      await useAuthStore.getState().login(credentials);

      expect(useAuthStore.getState().user?.role).toBe('service_owner');
    });

    it('sets error and isAuthenticated=false on failure', async () => {
      const error = Object.assign(new Error('Unauthorized'), {
        response: { data: { detail: 'Invalid credentials' } },
      });
      mockAuthService.login.mockRejectedValue(error);

      await expect(useAuthStore.getState().login(credentials)).rejects.toThrow();

      const state = useAuthStore.getState();
      expect(state.isAuthenticated).toBe(false);
      expect(state.error).toBe('Invalid credentials');
    });

    it('uses fallback error message when response detail is absent', async () => {
      mockAuthService.login.mockRejectedValue(new Error('Network error'));

      await expect(useAuthStore.getState().login(credentials)).rejects.toThrow();

      expect(useAuthStore.getState().error).toBe('Login failed. Please try again.');
    });

    it('throws error on failure', async () => {
      mockAuthService.login.mockRejectedValue(new Error('Unauthorized'));

      await expect(useAuthStore.getState().login(credentials)).rejects.toThrow();
    });

    it('sets error message about 2FA when requires_2fa is true', async () => {
      mockAuthService.login.mockResolvedValue({ requires_2fa: true });

      await useAuthStore.getState().login(credentials);

      const state = useAuthStore.getState();
      expect(state.error).toBe('Please enter your 2FA code');
      expect(state.isAuthenticated).toBe(false);
    });

    it('sets isLoading=false after successful login', async () => {
      mockAuthService.login.mockResolvedValue({
        access_token: 'tok123',
        user_id: 1,
        username: 'admin',
        email: 'admin@example.com',
        is_admin: false,
      });

      await useAuthStore.getState().login(credentials);

      expect(useAuthStore.getState().isLoading).toBe(false);
    });
  });

  describe('logout', () => {
    beforeEach(() => {
      useAuthStore.setState({
        user: { id: 1, username: 'admin', email: 'a@b.com', role: 'administrator', is_active: true, created_at: '', updated_at: '' },
        token: 'tok123',
        isAuthenticated: true,
        isLoading: false,
        error: null,
      });
    });

    it('clears user on logout', async () => {
      mockAuthService.logout.mockResolvedValue(undefined);

      await useAuthStore.getState().logout();

      expect(useAuthStore.getState().user).toBeNull();
    });

    it('clears token on logout', async () => {
      mockAuthService.logout.mockResolvedValue(undefined);

      await useAuthStore.getState().logout();

      expect(useAuthStore.getState().token).toBeNull();
    });

    it('sets isAuthenticated=false on logout', async () => {
      mockAuthService.logout.mockResolvedValue(undefined);

      await useAuthStore.getState().logout();

      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('clears state even if logout request throws', async () => {
      mockAuthService.logout.mockRejectedValue(new Error('Network error'));

      // try/finally re-throws the error, but state is still cleared in finally
      await expect(useAuthStore.getState().logout()).rejects.toThrow('Network error');

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.token).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe('loadUser', () => {
    it('sets isAuthenticated=false when no token exists', async () => {
      mockGetAuthToken.mockReturnValue(null);

      await useAuthStore.getState().loadUser();

      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('sets user=null when no token exists', async () => {
      mockGetAuthToken.mockReturnValue(null);

      await useAuthStore.getState().loadUser();

      expect(useAuthStore.getState().user).toBeNull();
    });

    it('loads user and sets isAuthenticated=true when token exists', async () => {
      mockGetAuthToken.mockReturnValue('tok123');
      const user = { id: 1, username: 'admin', email: 'a@b.com', role: 'administrator', is_active: true, created_at: '', updated_at: '' };
      mockAuthService.getCurrentUser.mockResolvedValue(user);

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.user).toEqual(user);
      expect(state.isAuthenticated).toBe(true);
    });

    it('clears state on getCurrentUser error', async () => {
      mockGetAuthToken.mockReturnValue('tok123');
      mockAuthService.getCurrentUser.mockRejectedValue(new Error('Unauthorized'));

      await useAuthStore.getState().loadUser();

      const state = useAuthStore.getState();
      expect(state.user).toBeNull();
      expect(state.token).toBeNull();
      expect(state.isAuthenticated).toBe(false);
    });
  });

  describe('clearError', () => {
    it('sets error to null', () => {
      useAuthStore.setState({ ...initialState, error: 'Some error' });

      useAuthStore.getState().clearError();

      expect(useAuthStore.getState().error).toBeNull();
    });
  });
});
