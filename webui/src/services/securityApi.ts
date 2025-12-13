/**
 * Security API Service
 *
 * API service for Zero-Trust security features including OPA policies,
 * audit logs, compliance reports, and mTLS certificate management.
 */

import { apiClient } from './api';
import { PaginatedResponse } from './types';

// OPA Policy Types
export interface OPAPolicy {
  id: number;
  name: string;
  description: string;
  rego_code: string;
  version: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  created_by: string;
  cluster_id?: number;
}

export interface PolicyVersion {
  id: number;
  policy_id: number;
  version: number;
  rego_code: string;
  created_at: string;
  created_by: string;
  diff?: string;
}

export interface PolicyTestRequest {
  policy_id?: number;
  rego_code: string;
  input_json: object;
}

export interface PolicyTestResponse {
  decision: 'allow' | 'deny';
  result: any;
  evaluation_trace: string[];
  evaluation_time_ms: number;
  errors?: string[];
}

export interface PolicyTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  rego_code: string;
  example_input: object;
}

// Audit Log Types
export interface AuditLog {
  id: number;
  timestamp: string;
  user_id?: number;
  username: string;
  service_id?: number;
  service_name?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  status: 'success' | 'failure';
  ip_address: string;
  user_agent?: string;
  details: object;
  cluster_id?: number;
  hash: string;
  previous_hash?: string;
  tamper_detected: boolean;
}

export interface AuditLogFilter {
  user_id?: number;
  service_id?: number;
  action?: string;
  status?: 'success' | 'failure';
  start_date?: string;
  end_date?: string;
  cluster_id?: number;
  search?: string;
  page?: number;
  page_size?: number;
}

export interface AuditLogExport {
  format: 'csv' | 'json';
  filters: AuditLogFilter;
}

// Compliance Types
export interface ComplianceStatus {
  framework: 'soc2' | 'hipaa' | 'pci_dss';
  overall_score: number;
  passing_controls: number;
  total_controls: number;
  last_assessment: string;
  next_assessment: string;
  requirements: ComplianceRequirement[];
}

export interface ComplianceRequirement {
  id: string;
  name: string;
  description: string;
  status: 'compliant' | 'non_compliant' | 'not_applicable';
  evidence_count: number;
  last_verified: string;
  automated: boolean;
}

export interface ComplianceEvidence {
  id: number;
  requirement_id: string;
  framework: string;
  evidence_type: string;
  description: string;
  file_url?: string;
  collected_at: string;
  collected_by: string;
}

export interface ComplianceReportRequest {
  framework: 'soc2' | 'hipaa' | 'pci_dss';
  start_date: string;
  end_date: string;
  include_evidence: boolean;
}

// mTLS Certificate Types
export interface Certificate {
  id: number;
  name: string;
  type: 'client' | 'ca' | 'server';
  subject: string;
  issuer: string;
  serial_number: string;
  not_before: string;
  not_after: string;
  status: 'valid' | 'expired' | 'revoked' | 'pending';
  fingerprint: string;
  cluster_id?: number;
  auto_rotation_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface CertificateUpload {
  name: string;
  type: 'client' | 'ca' | 'server';
  certificate_pem: string;
  private_key_pem?: string;
  cluster_id?: number;
  auto_rotation_enabled?: boolean;
}

export interface CRLEntry {
  serial_number: string;
  revocation_date: string;
  reason: string;
}

export interface CertificateValidation {
  is_valid: boolean;
  errors: string[];
  warnings: string[];
  chain_valid: boolean;
  expires_in_days: number;
}

// Policy Management API
export const getPolicies = async (
  cluster_id?: number
): Promise<OPAPolicy[]> => {
  const params = cluster_id ? { cluster_id } : {};
  const response = await apiClient.get('/api/v1/security/policies', {
    params
  });
  return response.data;
};

export const getPolicy = async (id: number): Promise<OPAPolicy> => {
  const response = await apiClient.get(`/api/v1/security/policies/${id}`);
  return response.data;
};

export const savePolicy = async (
  policy: Partial<OPAPolicy>
): Promise<OPAPolicy> => {
  if (policy.id) {
    const response = await apiClient.put(
      `/api/v1/security/policies/${policy.id}`,
      policy
    );
    return response.data;
  } else {
    const response = await apiClient.post(
      '/api/v1/security/policies',
      policy
    );
    return response.data;
  }
};

export const deletePolicy = async (id: number): Promise<void> => {
  await apiClient.delete(`/api/v1/security/policies/${id}`);
};

export const getPolicyVersions = async (
  policy_id: number
): Promise<PolicyVersion[]> => {
  const response = await apiClient.get(
    `/api/v1/security/policies/${policy_id}/versions`
  );
  return response.data;
};

export const testPolicy = async (
  request: PolicyTestRequest
): Promise<PolicyTestResponse> => {
  const response = await apiClient.post(
    '/api/v1/security/policies/test',
    request
  );
  return response.data;
};

export const getPolicyTemplates = async (): Promise<PolicyTemplate[]> => {
  const response = await apiClient.get('/api/v1/security/policy-templates');
  return response.data;
};

export const validatePolicy = async (
  rego_code: string
): Promise<{ valid: boolean; errors: string[] }> => {
  const response = await apiClient.post(
    '/api/v1/security/policies/validate',
    { rego_code }
  );
  return response.data;
};

// Audit Log API
export const getAuditLogs = async (
  filters: AuditLogFilter
): Promise<PaginatedResponse<AuditLog>> => {
  const response = await apiClient.get('/api/v1/security/audit-logs', {
    params: filters
  });
  return response.data;
};

export const exportAuditLogs = async (
  exportRequest: AuditLogExport
): Promise<Blob> => {
  const response = await apiClient.post(
    '/api/v1/security/audit-logs/export',
    exportRequest,
    { responseType: 'blob' }
  );
  return response.data;
};

export const verifyAuditLogIntegrity = async (): Promise<{
  verified: boolean;
  total_logs: number;
  tampered_logs: number[];
  last_verified: string;
}> => {
  const response = await apiClient.get(
    '/api/v1/security/audit-logs/verify'
  );
  return response.data;
};

// Compliance API
export const getComplianceStatus = async (
  framework: 'soc2' | 'hipaa' | 'pci_dss'
): Promise<ComplianceStatus> => {
  const response = await apiClient.get(
    `/api/v1/security/compliance/${framework}`
  );
  return response.data;
};

export const runComplianceCheck = async (
  framework: 'soc2' | 'hipaa' | 'pci_dss'
): Promise<ComplianceStatus> => {
  const response = await apiClient.post(
    `/api/v1/security/compliance/${framework}/check`
  );
  return response.data;
};

export const uploadComplianceEvidence = async (
  framework: string,
  requirement_id: string,
  evidence: FormData
): Promise<ComplianceEvidence> => {
  const response = await apiClient.post(
    `/api/v1/security/compliance/${framework}/evidence/${requirement_id}`,
    evidence,
    {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    }
  );
  return response.data;
};

export const generateComplianceReport = async (
  request: ComplianceReportRequest
): Promise<Blob> => {
  const response = await apiClient.post(
    '/api/v1/security/compliance/report',
    request,
    { responseType: 'blob' }
  );
  return response.data;
};

// mTLS Certificate Management API
export const getCertificates = async (
  cluster_id?: number
): Promise<Certificate[]> => {
  const params = cluster_id ? { cluster_id } : {};
  const response = await apiClient.get('/api/v1/security/certificates', {
    params
  });
  return response.data;
};

export const getCertificate = async (id: number): Promise<Certificate> => {
  const response = await apiClient.get(
    `/api/v1/security/certificates/${id}`
  );
  return response.data;
};

export const uploadCertificate = async (
  certificate: CertificateUpload
): Promise<Certificate> => {
  const response = await apiClient.post(
    '/api/v1/security/certificates',
    certificate
  );
  return response.data;
};

export const deleteCertificate = async (id: number): Promise<void> => {
  await apiClient.delete(`/api/v1/security/certificates/${id}`);
};

export const revokeCertificate = async (
  id: number,
  reason: string
): Promise<void> => {
  await apiClient.post(`/api/v1/security/certificates/${id}/revoke`, {
    reason
  });
};

export const validateCertificate = async (
  id: number
): Promise<CertificateValidation> => {
  const response = await apiClient.get(
    `/api/v1/security/certificates/${id}/validate`
  );
  return response.data;
};

export const getCRL = async (cluster_id?: number): Promise<CRLEntry[]> => {
  const params = cluster_id ? { cluster_id } : {};
  const response = await apiClient.get('/api/v1/security/crl', { params });
  return response.data;
};

export const updateCertificateRotation = async (
  id: number,
  enabled: boolean
): Promise<Certificate> => {
  const response = await apiClient.patch(
    `/api/v1/security/certificates/${id}/rotation`,
    { auto_rotation_enabled: enabled }
  );
  return response.data;
};

export const getExpiringCertificates = async (
  days: number = 30
): Promise<Certificate[]> => {
  const response = await apiClient.get(
    '/api/v1/security/certificates/expiring',
    {
      params: { days }
    }
  );
  return response.data;
};
