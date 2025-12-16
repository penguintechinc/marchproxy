/**
 * License Store using Zustand
 *
 * Manages license state across the application with automatic
 * initialization and periodic refresh.
 */

import { create } from 'zustand';
import {
  LicenseStatus,
  getLicenseStatus,
  refreshLicenseCache,
  checkFeature
} from '@services/licenseApi';

interface LicenseStore {
  // State
  license: LicenseStatus | null;
  isLoading: boolean;
  error: string | null;
  lastRefresh: number | null;

  // Actions
  loadLicense: () => Promise<void>;
  refreshLicense: () => Promise<void>;
  isFeatureAvailable: (featureName: string) => Promise<boolean>;
  clearError: () => void;
  resetStore: () => void;
}

export const useLicenseStore = create<LicenseStore>((set, get) => ({
  license: null,
  isLoading: false,
  error: null,
  lastRefresh: null,

  /**
   * Load license status from API
   */
  loadLicense: async () => {
    set({ isLoading: true, error: null });
    try {
      const license = await getLicenseStatus();
      set({
        license,
        isLoading: false,
        error: null,
        lastRefresh: Date.now()
      });
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to load license status';
      set({
        license: null,
        isLoading: false,
        error: errorMessage,
        lastRefresh: null
      });
    }
  },

  /**
   * Refresh license status (bypasses cache)
   */
  refreshLicense: async () => {
    set({ isLoading: true, error: null });
    try {
      const license = await refreshLicenseCache();
      set({
        license,
        isLoading: false,
        error: null,
        lastRefresh: Date.now()
      });
    } catch (error: any) {
      const errorMessage = error.message || 'Failed to refresh license status';
      set({
        isLoading: false,
        error: errorMessage,
        lastRefresh: null
      });
    }
  },

  /**
   * Check if a specific feature is available
   */
  isFeatureAvailable: async (featureName: string): Promise<boolean> => {
    const state = get();

    // If license is not loaded, try to load it first
    if (!state.license) {
      await get().loadLicense();
    }

    // Check features from current license
    const currentLicense = get().license;
    if (currentLicense && currentLicense.features) {
      return currentLicense.features.includes(featureName);
    }

    // Fallback: check via API if not in cache
    try {
      const featureCheck = await checkFeature(featureName);
      return featureCheck.available;
    } catch (error) {
      console.error(`Failed to check feature ${featureName}:`, error);
      return false;
    }
  },

  /**
   * Clear error message
   */
  clearError: () => {
    set({ error: null });
  },

  /**
   * Reset store to initial state
   */
  resetStore: () => {
    set({
      license: null,
      isLoading: false,
      error: null,
      lastRefresh: null
    });
  }
}));

/**
 * Initialize license store on app load
 */
export const initializeLicenseStore = async () => {
  try {
    await useLicenseStore.getState().loadLicense();
  } catch (error) {
    console.error('Failed to initialize license store:', error);
  }
};

/**
 * Setup automatic license refresh every 5 minutes
 */
export const setupLicenseRefreshInterval = () => {
  return setInterval(() => {
    useLicenseStore.getState().refreshLicense().catch((error) => {
      console.error('Failed to refresh license:', error);
    });
  }, 5 * 60 * 1000); // 5 minutes
};
