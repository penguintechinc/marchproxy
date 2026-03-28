/**
 * Tests for Authentication Service (auth.ts)
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('../api', () => ({
  apiClient: {
    post: vi.fn(),
    get: vi.fn(),
  },
  setAuthToken: vi.fn(),
  clearAuthToken: vi.fn(),
  getAuthToken: vi.fn(),
}));

import { authService } from '../auth';
import { apiClient, setAuthToken, clearAuthToken } from '../api';

const mockApiClient = apiClient as { post: ReturnType<typeof vi.fn>; get: ReturnType<typeof vi.fn> };
const mockSetAuthToken = setAuthToken as ReturnType<typeof vi.fn>;
const mockClearAuthToken = clearAuthToken as ReturnType<typeof vi.fn>;

describe('authService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('login', () => {
    it('calls POST /api/v1/auth/login with credentials', async () => {
      const credentials = { username: 'admin', password: 'secret' };
      mockApiClient.post.mockResolvedValue({
        data: { access_token: 'tok123', user_id: 1, username: 'admin', email: 'a@b.com', is_admin: true },
      });

      await authService.login(credentials);

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/login', credentials);
    });

    it('sets auth token when access_token is present', async () => {
      const credentials = { username: 'admin', password: 'secret' };
      mockApiClient.post.mockResolvedValue({
        data: { access_token: 'tok123', user_id: 1, username: 'admin', email: 'a@b.com', is_admin: false },
      });

      await authService.login(credentials);

      expect(mockSetAuthToken).toHaveBeenCalledWith('tok123');
    });

    it('does not set auth token when access_token is absent', async () => {
      const credentials = { username: 'admin', password: 'secret' };
      mockApiClient.post.mockResolvedValue({
        data: { requires_2fa: true },
      });

      await authService.login(credentials);

      expect(mockSetAuthToken).not.toHaveBeenCalled();
    });

    it('handles 2FA required response without setting token', async () => {
      const credentials = { username: 'admin', password: 'secret' };
      const responseData = { requires_2fa: true };
      mockApiClient.post.mockResolvedValue({ data: responseData });

      const result = await authService.login(credentials);

      expect(result).toEqual(responseData);
      expect(mockSetAuthToken).not.toHaveBeenCalled();
    });

    it('returns response data', async () => {
      const credentials = { username: 'admin', password: 'secret' };
      const responseData = { access_token: 'tok123', user_id: 1, username: 'admin', email: 'a@b.com', is_admin: true };
      mockApiClient.post.mockResolvedValue({ data: responseData });

      const result = await authService.login(credentials);

      expect(result).toEqual(responseData);
    });

    it('propagates errors from apiClient', async () => {
      const credentials = { username: 'admin', password: 'wrong' };
      const error = new Error('Unauthorized');
      mockApiClient.post.mockRejectedValue(error);

      await expect(authService.login(credentials)).rejects.toThrow('Unauthorized');
    });
  });

  describe('logout', () => {
    it('calls POST /api/v1/auth/logout', async () => {
      mockApiClient.post.mockResolvedValue({ data: {} });

      await authService.logout();

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/logout');
    });

    it('clears auth token after successful logout', async () => {
      mockApiClient.post.mockResolvedValue({ data: {} });

      await authService.logout();

      expect(mockClearAuthToken).toHaveBeenCalled();
    });

    it('clears auth token even when logout request fails', async () => {
      mockApiClient.post.mockRejectedValue(new Error('Network error'));

      // try/finally re-throws, but clearAuthToken runs before the throw
      await expect(authService.logout()).rejects.toThrow('Network error');

      expect(mockClearAuthToken).toHaveBeenCalled();
    });
  });

  describe('getCurrentUser', () => {
    it('calls GET /api/v1/auth/me', async () => {
      const user = { id: 1, username: 'admin', email: 'a@b.com', role: 'administrator', is_active: true, created_at: '', updated_at: '' };
      mockApiClient.get.mockResolvedValue({ data: user });

      await authService.getCurrentUser();

      expect(mockApiClient.get).toHaveBeenCalledWith('/api/v1/auth/me');
    });

    it('returns user data', async () => {
      const user = { id: 1, username: 'admin', email: 'a@b.com', role: 'administrator', is_active: true, created_at: '', updated_at: '' };
      mockApiClient.get.mockResolvedValue({ data: user });

      const result = await authService.getCurrentUser();

      expect(result).toEqual(user);
    });

    it('propagates errors from apiClient', async () => {
      mockApiClient.get.mockRejectedValue(new Error('Forbidden'));

      await expect(authService.getCurrentUser()).rejects.toThrow('Forbidden');
    });
  });

  describe('refreshToken', () => {
    it('calls POST /api/v1/auth/refresh', async () => {
      mockApiClient.post.mockResolvedValue({ data: { token: 'new-tok' } });

      await authService.refreshToken();

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/refresh');
    });

    it('sets new auth token', async () => {
      mockApiClient.post.mockResolvedValue({ data: { token: 'new-tok' } });

      await authService.refreshToken();

      expect(mockSetAuthToken).toHaveBeenCalledWith('new-tok');
    });

    it('returns new token string', async () => {
      mockApiClient.post.mockResolvedValue({ data: { token: 'new-tok' } });

      const result = await authService.refreshToken();

      expect(result).toBe('new-tok');
    });
  });

  describe('changePassword', () => {
    it('calls POST /api/v1/auth/change-password with correct payload', async () => {
      mockApiClient.post.mockResolvedValue({ data: {} });

      await authService.changePassword('oldpass', 'newpass');

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/change-password', {
        current_password: 'oldpass',
        new_password: 'newpass',
      });
    });
  });

  describe('enable2FA', () => {
    it('calls POST /api/v1/auth/2fa/enable', async () => {
      mockApiClient.post.mockResolvedValue({ data: { secret: 'ABCD', qr_code: 'data:image/png;base64,...' } });

      await authService.enable2FA();

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/2fa/enable');
    });

    it('returns secret and qr_code', async () => {
      const twoFAData = { secret: 'ABCD1234', qr_code: 'data:image/png;base64,xyz' };
      mockApiClient.post.mockResolvedValue({ data: twoFAData });

      const result = await authService.enable2FA();

      expect(result).toEqual(twoFAData);
    });
  });

  describe('verify2FA', () => {
    it('calls POST /api/v1/auth/2fa/verify with code', async () => {
      mockApiClient.post.mockResolvedValue({ data: {} });

      await authService.verify2FA('123456');

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/2fa/verify', { code: '123456' });
    });
  });

  describe('disable2FA', () => {
    it('calls POST /api/v1/auth/2fa/disable with code', async () => {
      mockApiClient.post.mockResolvedValue({ data: {} });

      await authService.disable2FA('654321');

      expect(mockApiClient.post).toHaveBeenCalledWith('/api/v1/auth/2fa/disable', { code: '654321' });
    });
  });
});
