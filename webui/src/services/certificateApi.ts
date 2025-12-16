/**
 * Certificate API Service
 *
 * Handles all certificate-related API operations including upload,
 * Infisical and Vault integrations, and certificate management.
 */

import { apiClient } from './api';
import { ApiResponse, PaginatedResponse } from './types';

export interface Certificate {
  id: number;
  name: string;
  common_name: string;
  san: string[];
  issuer: string;
  valid_from: string;
  valid_until: string;
  is_expired: boolean;
  days_until_expiry: number;
  auto_renew: boolean;
  source: 'upload' | 'infisical' | 'vault';
  created_at: string;
  updated_at: string;
}

export interface UploadCertificateRequest {
  name: string;
  certificate: string;
  private_key: string;
  ca_chain?: string;
  auto_renew: boolean;
}

export interface InfisicalIntegrationRequest {
  name: string;
  infisical_url: string;
  infisical_token: string;
  project_id: string;
  secret_path: string;
  auto_renew: boolean;
}

export interface VaultIntegrationRequest {
  name: string;
  vault_url: string;
  vault_token: string;
  vault_path: string;
  auto_renew: boolean;
}

export interface CertificateListParams {
  page?: number;
  page_size?: number;
  search?: string;
  is_expired?: boolean;
  source?: 'upload' | 'infisical' | 'vault';
}

export const certificateApi = {
  /**
   * Get all certificates with optional filtering
   */
  list: async (params?: CertificateListParams): Promise<PaginatedResponse<Certificate>> => {
    const response = await apiClient.get<PaginatedResponse<Certificate>>('/api/certificates', {
      params
    });
    return response.data;
  },

  /**
   * Get a single certificate by ID
   */
  get: async (id: number): Promise<Certificate> => {
    const response = await apiClient.get<Certificate>(`/api/certificates/${id}`);
    return response.data;
  },

  /**
   * Upload a certificate
   */
  upload: async (data: UploadCertificateRequest): Promise<Certificate> => {
    const response = await apiClient.post<ApiResponse<Certificate>>(
      '/api/certificates/upload',
      data
    );
    return response.data.data;
  },

  /**
   * Configure Infisical integration
   */
  configureInfisical: async (data: InfisicalIntegrationRequest): Promise<Certificate> => {
    const response = await apiClient.post<ApiResponse<Certificate>>(
      '/api/certificates/infisical',
      data
    );
    return response.data.data;
  },

  /**
   * Configure Vault integration
   */
  configureVault: async (data: VaultIntegrationRequest): Promise<Certificate> => {
    const response = await apiClient.post<ApiResponse<Certificate>>(
      '/api/certificates/vault',
      data
    );
    return response.data.data;
  },

  /**
   * Delete a certificate
   */
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/certificates/${id}`);
  },

  /**
   * Toggle auto-renewal for a certificate
   */
  toggleAutoRenew: async (id: number, enabled: boolean): Promise<Certificate> => {
    const response = await apiClient.put<ApiResponse<Certificate>>(
      `/api/certificates/${id}/auto-renew`,
      { enabled }
    );
    return response.data.data;
  },

  /**
   * Force certificate renewal
   */
  renew: async (id: number): Promise<Certificate> => {
    const response = await apiClient.post<ApiResponse<Certificate>>(
      `/api/certificates/${id}/renew`
    );
    return response.data.data;
  },

  /**
   * Get certificate details including private key (requires admin)
   */
  getDetails: async (id: number): Promise<{
    certificate: string;
    private_key: string;
    ca_chain: string;
  }> => {
    const response = await apiClient.get(`/api/certificates/${id}/details`);
    return response.data;
  }
};
