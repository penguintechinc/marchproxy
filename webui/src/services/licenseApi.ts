/**
 * License API Service
 *
 * Handles license status checks and feature entitlement validation
 * with built-in caching to reduce API calls.
 */

import { apiClient } from './api';

export interface LicenseStatus {
  is_enterprise: boolean;
  tier: 'community' | 'enterprise';
  license_key?: string;
  expiration_date?: string;
  proxy_limit?: number;
  active_proxies?: number;
  features: string[];
  valid: boolean;
}

export interface FeatureCheck {
  feature: string;
  available: boolean;
  tier: 'community' | 'enterprise';
}

// Cache license status with 5-minute TTL
let cachedLicense: LicenseStatus | null = null;
let cacheTimestamp: number = 0;
const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

/**
 * Check if cached license is still valid
 */
const isCacheValid = (): boolean => {
  if (!cachedLicense) return false;
  const now = Date.now();
  return now - cacheTimestamp < CACHE_TTL_MS;
};

/**
 * Get current license status from API or cache
 */
export const getLicenseStatus = async (
  forceRefresh: boolean = false
): Promise<LicenseStatus> => {
  // Return cached license if valid and not forced refresh
  if (!forceRefresh && isCacheValid()) {
    return cachedLicense!;
  }

  try {
    const response = await apiClient.get<LicenseStatus>('/api/license/status');
    cachedLicense = response.data;
    cacheTimestamp = Date.now();
    return response.data;
  } catch (error: any) {
    console.error('Failed to fetch license status:', error);
    // If API fails and we have cached data, return it
    if (cachedLicense) {
      return cachedLicense;
    }
    // Default to community tier if API fails
    return {
      is_enterprise: false,
      tier: 'community',
      features: [],
      valid: false
    };
  }
};

/**
 * Check if a specific feature is available in current license
 */
export const checkFeature = async (
  featureName: string,
  forceRefresh: boolean = false
): Promise<FeatureCheck> => {
  try {
    const response = await apiClient.get<FeatureCheck>(
      `/api/license/features/${featureName}`,
      {
        params: { force_refresh: forceRefresh }
      }
    );
    return response.data;
  } catch (error: any) {
    console.error(`Failed to check feature ${featureName}:`, error);
    // Default to community tier
    return {
      feature: featureName,
      available: false,
      tier: 'community'
    };
  }
};

/**
 * Get list of all available features for current license
 */
export const getAvailableFeatures = async (
  forceRefresh: boolean = false
): Promise<string[]> => {
  const license = await getLicenseStatus(forceRefresh);
  return license.features;
};

/**
 * Refresh license cache (for when license is activated/changed)
 */
export const refreshLicenseCache = async (): Promise<LicenseStatus> => {
  return getLicenseStatus(true);
};

/**
 * Clear license cache (useful for testing or manual refresh)
 */
export const clearLicenseCache = (): void => {
  cachedLicense = null;
  cacheTimestamp = 0;
};
