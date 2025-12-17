/**
 * Certificate API Service
 *
 * Handles all certificate-related API operations including upload,
 * Infisical and Vault integrations, and certificate management.
 *
 * Backend API uses a unified endpoint at /api/v1/certificates with
 * source_type parameter to determine the certificate source.
 */

import { apiClient } from './api';

export type CertificateSource = 'upload' | 'infisical' | 'vault';

export interface Certificate {
  id: number;
  name: string;
  description?: string;
  source_type: CertificateSource;
  common_name?: string;
  issuer?: string;
  valid_from?: string;
  valid_until: string;
  auto_renew: boolean;
  renew_before_days: number;
  is_active: boolean;
  is_expired: boolean;
  days_until_expiry: number;
  needs_renewal: boolean;
  last_renewal?: string;
  renewal_error?: string;
  created_at: string;
  updated_at?: string;
}

export interface CertificateDetail extends Certificate {
  cert_data: string;
  ca_chain?: string;
  subject_alt_names?: string;
  infisical_secret_path?: string;
  infisical_project_id?: string;
  infisical_environment?: string;
  vault_path?: string;
  vault_role?: string;
  vault_common_name?: string;
}

export interface UploadCertificateRequest {
  name: string;
  description?: string;
  source_type: 'upload';
  cert_data: string;
  key_data: string;
  ca_chain?: string;
  auto_renew?: boolean;
  renew_before_days?: number;
}

export interface InfisicalCertificateRequest {
  name: string;
  description?: string;
  source_type: 'infisical';
  infisical_secret_path: string;
  infisical_project_id: string;
  infisical_environment?: string;
  auto_renew?: boolean;
  renew_before_days?: number;
}

export interface VaultCertificateRequest {
  name: string;
  description?: string;
  source_type: 'vault';
  vault_path: string;
  vault_role: string;
  vault_common_name: string;
  auto_renew?: boolean;
  renew_before_days?: number;
}

export type CreateCertificateRequest =
  | UploadCertificateRequest
  | InfisicalCertificateRequest
  | VaultCertificateRequest;

export interface CertificateUpdateRequest {
  description?: string;
  auto_renew?: boolean;
  renew_before_days?: number;
  is_active?: boolean;
}

export interface CertificateListParams {
  skip?: number;
  limit?: number;
  include_expired?: boolean;
  source_type?: CertificateSource;
}

export interface CertificateRenewResponse {
  certificate_id: number;
  renewed: boolean;
  message: string;
  valid_until?: string;
  error?: string;
}

export interface BatchRenewResponse {
  total: number;
  renewed: number;
  failed: number;
  skipped: number;
  details: Array<{
    id: number;
    name: string;
    status: 'renewed' | 'failed' | 'skipped';
    reason?: string;
    error?: string;
  }>;
}

export const certificateApi = {
  /**
   * Get all certificates with optional filtering
   *
   * Backend returns a simple array, converted to object format for consistency
   */
  list: async (params?: CertificateListParams): Promise<{ items: Certificate[] }> => {
    const response = await apiClient.get<Certificate[]>(
      '/api/v1/certificates',
      { params }
    );
    return { items: response.data };
  },

  /**
   * Get a single certificate by ID
   */
  get: async (id: number): Promise<Certificate> => {
    const response = await apiClient.get<Certificate>(
      `/api/v1/certificates/${id}`
    );
    return response.data;
  },

  /**
   * Get certificate details including cert data (but not private key)
   */
  getDetails: async (id: number): Promise<CertificateDetail> => {
    const response = await apiClient.get<CertificateDetail>(
      `/api/v1/certificates/${id}`
    );
    return response.data;
  },

  /**
   * Upload a certificate
   *
   * Uses unified /api/v1/certificates endpoint with source_type='upload'
   */
  upload: async (data: UploadCertificateRequest): Promise<Certificate> => {
    const response = await apiClient.post<Certificate>(
      '/api/v1/certificates',
      data
    );
    return response.data;
  },

  /**
   * Configure Infisical integration
   *
   * Uses unified /api/v1/certificates endpoint with source_type='infisical'
   */
  configureInfisical: async (
    data: InfisicalCertificateRequest
  ): Promise<Certificate> => {
    const response = await apiClient.post<Certificate>(
      '/api/v1/certificates',
      data
    );
    return response.data;
  },

  /**
   * Configure Vault integration
   *
   * Uses unified /api/v1/certificates endpoint with source_type='vault'
   */
  configureVault: async (data: VaultCertificateRequest): Promise<Certificate> => {
    const response = await apiClient.post<Certificate>(
      '/api/v1/certificates',
      data
    );
    return response.data;
  },

  /**
   * Update certificate settings including auto_renew
   *
   * Uses PUT with auto_renew field instead of separate endpoint
   */
  update: async (
    id: number,
    data: CertificateUpdateRequest
  ): Promise<Certificate> => {
    const response = await apiClient.put<Certificate>(
      `/api/v1/certificates/${id}`,
      data
    );
    return response.data;
  },

  /**
   * Toggle auto-renewal for a certificate
   *
   * Convenience method that calls update() with auto_renew field
   */
  toggleAutoRenew: async (id: number, enabled: boolean): Promise<Certificate> => {
    return certificateApi.update(id, { auto_renew: enabled });
  },

  /**
   * Delete a certificate
   */
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/api/v1/certificates/${id}`);
  },

  /**
   * Force certificate renewal
   *
   * Only works for Infisical and Vault certificates
   */
  renew: async (id: number): Promise<CertificateRenewResponse> => {
    const response = await apiClient.post<CertificateRenewResponse>(
      `/api/v1/certificates/${id}/renew`
    );
    return response.data;
  },

  /**
   * List certificates expiring within specified days
   */
  listExpiring: async (days?: number): Promise<{ items: Certificate[] }> => {
    const response = await apiClient.get<Certificate[]>(
      '/api/v1/certificates/expiring/list',
      { params: { days } }
    );
    return { items: response.data };
  },

  /**
   * Batch renew all certificates expiring soon
   */
  batchRenew: async (days?: number): Promise<BatchRenewResponse> => {
    const response = await apiClient.post<BatchRenewResponse>(
      '/api/v1/certificates/batch-renew',
      {},
      { params: { days } }
    );
    return response.data;
  }
};
