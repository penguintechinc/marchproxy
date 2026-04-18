import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useLicenseStore, initializeLicenseStore, setupLicenseRefreshInterval } from '../licenseStore';
import * as licenseApi from '@services/licenseApi';

vi.mock('@services/licenseApi', () => ({
  getLicenseStatus: vi.fn(),
  refreshLicenseCache: vi.fn(),
  checkFeature: vi.fn(),
}));

const mockGetLicenseStatus = licenseApi.getLicenseStatus as ReturnType<typeof vi.fn>;
const mockRefreshLicenseCache = licenseApi.refreshLicenseCache as ReturnType<typeof vi.fn>;
const mockCheckFeature = licenseApi.checkFeature as ReturnType<typeof vi.fn>;

describe('License Store', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useLicenseStore.getState().resetStore();
  });

  describe('loadLicense', () => {
    it('loads license status from API', async () => {
      const mockLicense = {
        isValid: true,
        expiresAt: '2025-12-31',
        features: ['feature1', 'feature2'],
        tier: 'enterprise',
      };

      mockGetLicenseStatus.mockResolvedValue(mockLicense);

      await useLicenseStore.getState().loadLicense();

      const state = useLicenseStore.getState();
      expect(state.license).toEqual(mockLicense);
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });

    it('sets error on API failure', async () => {
      mockGetLicenseStatus.mockRejectedValue(new Error('API Error'));

      await useLicenseStore.getState().loadLicense();

      const state = useLicenseStore.getState();
      expect(state.license).toBeNull();
      expect(state.isLoading).toBe(false);
      expect(state.error).not.toBeNull();
    });

    it('sets loading state during fetch', async () => {
      mockGetLicenseStatus.mockImplementation(
        () =>
          new Promise((resolve) => {
            expect(useLicenseStore.getState().isLoading).toBe(true);
            resolve({
              isValid: true,
              expiresAt: '2025-12-31',
              features: [],
              tier: 'community',
            });
          })
      );

      await useLicenseStore.getState().loadLicense();

      const state = useLicenseStore.getState();
      expect(state.isLoading).toBe(false);
    });
  });

  describe('refreshLicense', () => {
    it('refreshes license from API', async () => {
      const mockLicense = {
        isValid: true,
        expiresAt: '2025-12-31',
        features: ['feature1'],
        tier: 'enterprise',
      };

      mockRefreshLicenseCache.mockResolvedValue(mockLicense);

      await useLicenseStore.getState().refreshLicense();

      const state = useLicenseStore.getState();
      expect(state.license).toEqual(mockLicense);
      expect(state.lastRefresh).not.toBeNull();
    });

    it('handles refresh errors', async () => {
      mockRefreshLicenseCache.mockRejectedValue(new Error('Refresh failed'));

      await useLicenseStore.getState().refreshLicense();

      const state = useLicenseStore.getState();
      expect(state.error).not.toBeNull();
    });
  });

  describe('isFeatureAvailable', () => {
    it('checks feature from loaded license', async () => {
      const mockLicense = {
        isValid: true,
        expiresAt: '2025-12-31',
        features: ['advanced-analytics', 'multi-cloud'],
        tier: 'enterprise',
      };

      mockGetLicenseStatus.mockResolvedValue(mockLicense);
      await useLicenseStore.getState().loadLicense();

      const available = await useLicenseStore.getState().isFeatureAvailable('advanced-analytics');
      expect(available).toBe(true);

      const unavailable = await useLicenseStore.getState().isFeatureAvailable('unknown-feature');
      expect(unavailable).toBe(false);
    });

    it('loads license if not already loaded', async () => {
      const mockLicense = {
        isValid: true,
        expiresAt: '2025-12-31',
        features: ['feature1'],
        tier: 'enterprise',
      };

      mockGetLicenseStatus.mockResolvedValue(mockLicense);

      const available = await useLicenseStore.getState().isFeatureAvailable('feature1');

      expect(available).toBe(true);
    });
  });

  describe('clearError', () => {
    it('clears error message', async () => {
      mockGetLicenseStatus.mockRejectedValue(new Error('Test error'));
      await useLicenseStore.getState().loadLicense();

      expect(useLicenseStore.getState().error).not.toBeNull();

      useLicenseStore.getState().clearError();

      expect(useLicenseStore.getState().error).toBeNull();
    });
  });

  describe('resetStore', () => {
    it('resets store to initial state', async () => {
      const mockLicense = {
        isValid: true,
        expiresAt: '2025-12-31',
        features: [],
        tier: 'enterprise',
      };

      mockGetLicenseStatus.mockResolvedValue(mockLicense);
      await useLicenseStore.getState().loadLicense();

      useLicenseStore.getState().resetStore();

      const state = useLicenseStore.getState();
      expect(state.license).toBeNull();
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
      expect(state.lastRefresh).toBeNull();
    });
  });

  describe('initializeLicenseStore', () => {
    it('initializes license on startup', async () => {
      const mockLicense = {
        isValid: true,
        expiresAt: '2025-12-31',
        features: [],
        tier: 'enterprise',
      };

      mockGetLicenseStatus.mockResolvedValue(mockLicense);

      await initializeLicenseStore();

      expect(mockGetLicenseStatus).toHaveBeenCalled();
    });
  });

  describe('setupLicenseRefreshInterval', () => {
    it('returns interval ID', () => {
      mockRefreshLicenseCache.mockResolvedValue({
        isValid: true,
        expiresAt: '2025-12-31',
        features: [],
        tier: 'enterprise',
      });

      const intervalId = setupLicenseRefreshInterval();

      expect(intervalId).toBeDefined();
      clearInterval(intervalId);
    });
  });
});
