/**
 * Tests for useLicense hook (useLicense.ts)
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';

vi.mock('@services/licenseApi', () => ({
  getLicenseStatus: vi.fn(),
  checkFeature: vi.fn(),
  refreshLicenseCache: vi.fn(),
}));

import { useLicense } from '../useLicense';
import { getLicenseStatus, checkFeature, refreshLicenseCache } from '@services/licenseApi';

const mockGetLicenseStatus = getLicenseStatus as ReturnType<typeof vi.fn>;
const mockCheckFeature = checkFeature as ReturnType<typeof vi.fn>;
const mockRefreshLicenseCache = refreshLicenseCache as ReturnType<typeof vi.fn>;

const communityLicense = {
  is_enterprise: false,
  tier: 'community' as const,
  features: [],
  valid: false,
};

const enterpriseLicense = {
  is_enterprise: true,
  tier: 'enterprise' as const,
  features: ['traffic_shaping', 'advanced_analytics', 'sso'],
  valid: true,
};

describe('useLicense', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('initial state with autoLoad=true', () => {
    it('starts with loading=true', () => {
      mockGetLicenseStatus.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(() => useLicense(true));

      expect(result.current.loading).toBe(true);
    });

    it('calls getLicenseStatus on mount', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);

      renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(mockGetLicenseStatus).toHaveBeenCalled();
      });
    });
  });

  describe('after successful load', () => {
    it('sets license data', async () => {
      mockGetLicenseStatus.mockResolvedValue(enterpriseLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.license).toEqual(enterpriseLicense);
    });

    it('sets loading=false after load', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });

    it('keeps error=null on success', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('after load error', () => {
    it('sets error message', async () => {
      mockGetLicenseStatus.mockRejectedValue(new Error('API unavailable'));

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.error).toBe('API unavailable');
    });

    it('sets default community license on error', async () => {
      mockGetLicenseStatus.mockRejectedValue(new Error('API unavailable'));

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.license).toEqual({
        is_enterprise: false,
        tier: 'community',
        features: [],
        valid: false,
      });
    });

    it('sets loading=false after error', async () => {
      mockGetLicenseStatus.mockRejectedValue(new Error('API unavailable'));

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });
    });
  });

  describe('hasFeature', () => {
    it('returns true when feature is in license features list', async () => {
      mockGetLicenseStatus.mockResolvedValue(enterpriseLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.hasFeature('traffic_shaping')).toBe(true);
    });

    it('returns false when feature is not in license features list', async () => {
      mockGetLicenseStatus.mockResolvedValue(enterpriseLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.hasFeature('nonexistent_feature')).toBe(false);
    });

    it('returns false when license is null', async () => {
      mockGetLicenseStatus.mockImplementation(() => new Promise(() => {}));

      const { result } = renderHook(() => useLicense(false));

      expect(result.current.hasFeature('traffic_shaping')).toBe(false);
    });

    it('returns false for community license with no features', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.hasFeature('traffic_shaping')).toBe(false);
    });
  });

  describe('isEnterprise', () => {
    it('returns true when license.is_enterprise=true', async () => {
      mockGetLicenseStatus.mockResolvedValue(enterpriseLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.isEnterprise).toBe(true);
    });

    it('returns false when license.is_enterprise=false', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);

      const { result } = renderHook(() => useLicense(true));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.isEnterprise).toBe(false);
    });

    it('returns false when license is null (no autoLoad)', () => {
      const { result } = renderHook(() => useLicense(false));

      expect(result.current.isEnterprise).toBe(false);
    });
  });

  describe('autoLoad=false', () => {
    it('does not call getLicenseStatus', () => {
      renderHook(() => useLicense(false));

      expect(mockGetLicenseStatus).not.toHaveBeenCalled();
    });

    it('starts with loading=false', () => {
      const { result } = renderHook(() => useLicense(false));

      expect(result.current.loading).toBe(false);
    });

    it('starts with license=null', () => {
      const { result } = renderHook(() => useLicense(false));

      expect(result.current.license).toBeNull();
    });
  });

  describe('checkFeatureAsync', () => {
    it('returns true when feature is available', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);
      mockCheckFeature.mockResolvedValue({ feature: 'sso', available: true, tier: 'enterprise' });

      const { result } = renderHook(() => useLicense(false));

      await waitFor(async () => {
        const available = await result.current.checkFeatureAsync('sso');
        expect(available).toBe(true);
      });
    });

    it('returns false when checkFeature throws', async () => {
      mockCheckFeature.mockRejectedValue(new Error('API error'));

      const { result } = renderHook(() => useLicense(false));

      const available = await result.current.checkFeatureAsync('sso');
      expect(available).toBe(false);
    });
  });

  describe('refresh', () => {
    it('calls refreshLicenseCache', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);
      mockRefreshLicenseCache.mockResolvedValue(enterpriseLicense);

      const { result } = renderHook(() => useLicense(false));

      await act(async () => {
        await result.current.refresh();
      });

      expect(mockRefreshLicenseCache).toHaveBeenCalled();
    });

    it('updates license after refresh', async () => {
      mockGetLicenseStatus.mockResolvedValue(communityLicense);
      mockRefreshLicenseCache.mockResolvedValue(enterpriseLicense);

      const { result } = renderHook(() => useLicense(false));

      await act(async () => {
        await result.current.refresh();
      });

      expect(result.current.license).toEqual(enterpriseLicense);
    });
  });
});
