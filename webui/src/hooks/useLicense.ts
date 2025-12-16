/**
 * useLicense Hook
 *
 * React hook for checking license status and enterprise feature availability.
 * Provides cached license status and feature checking utilities.
 */

import { useState, useEffect, useCallback } from 'react';
import {
  getLicenseStatus,
  checkFeature,
  refreshLicenseCache,
  LicenseStatus,
  FeatureCheck
} from '../services/licenseApi';

interface UseLicenseResult {
  license: LicenseStatus | null;
  loading: boolean;
  error: string | null;
  isEnterprise: boolean;
  hasFeature: (featureName: string) => boolean;
  checkFeatureAsync: (featureName: string) => Promise<boolean>;
  refresh: () => Promise<void>;
}

/**
 * Hook to access license status and check enterprise features
 *
 * @param autoLoad - Whether to automatically load license on mount (default: true)
 * @returns License state and utility functions
 *
 * @example
 * const { isEnterprise, hasFeature, loading } = useLicense();
 *
 * if (loading) return <Loading />;
 * if (!isEnterprise) return <UpgradePrompt />;
 * if (!hasFeature('traffic_shaping')) return <FeatureGate />;
 */
export function useLicense(autoLoad: boolean = true): UseLicenseResult {
  const [license, setLicense] = useState<LicenseStatus | null>(null);
  const [loading, setLoading] = useState<boolean>(autoLoad);
  const [error, setError] = useState<string | null>(null);

  const loadLicense = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const status = await getLicenseStatus();
      setLicense(status);
    } catch (err: any) {
      setError(err.message || 'Failed to load license status');
      // Set default community license on error
      setLicense({
        is_enterprise: false,
        tier: 'community',
        features: [],
        valid: false
      });
    } finally {
      setLoading(false);
    }
  }, []);

  const refresh = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const status = await refreshLicenseCache();
      setLicense(status);
    } catch (err: any) {
      setError(err.message || 'Failed to refresh license status');
    } finally {
      setLoading(false);
    }
  }, []);

  const hasFeature = useCallback((featureName: string): boolean => {
    if (!license) return false;
    return license.features.includes(featureName);
  }, [license]);

  const checkFeatureAsync = useCallback(async (featureName: string): Promise<boolean> => {
    try {
      const result: FeatureCheck = await checkFeature(featureName);
      return result.available;
    } catch {
      return false;
    }
  }, []);

  useEffect(() => {
    if (autoLoad) {
      loadLicense();
    }
  }, [autoLoad, loadLicense]);

  return {
    license,
    loading,
    error,
    isEnterprise: license?.is_enterprise ?? false,
    hasFeature,
    checkFeatureAsync,
    refresh
  };
}

/**
 * Hook to check a specific enterprise feature
 *
 * @param featureName - The feature to check
 * @returns Object with feature availability status
 *
 * @example
 * const { available, loading } = useFeature('traffic_shaping');
 */
export function useFeature(featureName: string) {
  const [available, setAvailable] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const check = async () => {
      try {
        setLoading(true);
        const result = await checkFeature(featureName);
        setAvailable(result.available);
      } catch (err: any) {
        setError(err.message || 'Failed to check feature');
        setAvailable(false);
      } finally {
        setLoading(false);
      }
    };

    check();
  }, [featureName]);

  return { available, loading, error };
}

export default useLicense;
