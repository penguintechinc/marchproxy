/**
 * Kong Admin API Service
 * Direct communication with Kong Admin API (internal network only)
 */
import axios, { AxiosInstance } from 'axios';

// Kong Admin API base URL (internal Docker network)
const KONG_ADMIN_URL = import.meta.env.VITE_KONG_ADMIN_URL || 'http://kong:8001';

// Create axios instance for Kong Admin API
const kongClient: AxiosInstance = axios.create({
  baseURL: KONG_ADMIN_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Types
export interface KongService {
  id: string;
  name: string;
  protocol: string;
  host: string;
  port: number;
  path?: string;
  retries?: number;
  connect_timeout?: number;
  write_timeout?: number;
  read_timeout?: number;
  enabled?: boolean;
  tags?: string[];
  created_at?: number;
  updated_at?: number;
}

export interface KongRoute {
  id: string;
  name: string;
  protocols: string[];
  methods?: string[];
  hosts?: string[];
  paths?: string[];
  headers?: Record<string, string[]>;
  strip_path?: boolean;
  preserve_host?: boolean;
  regex_priority?: number;
  service?: { id: string };
  tags?: string[];
  created_at?: number;
  updated_at?: number;
}

export interface KongUpstream {
  id: string;
  name: string;
  algorithm?: string;
  hash_on?: string;
  hash_fallback?: string;
  slots?: number;
  healthchecks?: Record<string, unknown>;
  tags?: string[];
  created_at?: number;
}

export interface KongTarget {
  id: string;
  target: string;
  weight: number;
  upstream?: { id: string };
  tags?: string[];
  created_at?: number;
}

export interface KongConsumer {
  id: string;
  username?: string;
  custom_id?: string;
  tags?: string[];
  created_at?: number;
}

export interface KongPlugin {
  id: string;
  name: string;
  config: Record<string, unknown>;
  enabled: boolean;
  protocols?: string[];
  service?: { id: string };
  route?: { id: string };
  consumer?: { id: string };
  tags?: string[];
  created_at?: number;
}

export interface KongCertificate {
  id: string;
  cert: string;
  key: string;
  cert_alt?: string;
  key_alt?: string;
  tags?: string[];
  snis?: string[];
  created_at?: number;
}

export interface KongSNI {
  id: string;
  name: string;
  certificate: { id: string };
  tags?: string[];
  created_at?: number;
}

export interface KongListResponse<T> {
  data: T[];
  next?: string;
  offset?: string;
}

export interface KongStatus {
  database: { reachable: boolean };
  memory: { workers_lua_vms: unknown };
  server: { connections_reading: number; connections_writing: number };
}

// Kong API methods
export const kongApi = {
  // Status
  getStatus: () => kongClient.get<KongStatus>('/status'),

  // Services
  getServices: (params?: { offset?: number; size?: number }) =>
    kongClient.get<KongListResponse<KongService>>('/services', { params }),
  getService: (id: string) => kongClient.get<KongService>(`/services/${id}`),
  createService: (data: Partial<KongService>) =>
    kongClient.post<KongService>('/services', data),
  updateService: (id: string, data: Partial<KongService>) =>
    kongClient.patch<KongService>(`/services/${id}`, data),
  deleteService: (id: string) => kongClient.delete(`/services/${id}`),

  // Routes
  getRoutes: (params?: { offset?: number; size?: number }) =>
    kongClient.get<KongListResponse<KongRoute>>('/routes', { params }),
  getServiceRoutes: (serviceId: string) =>
    kongClient.get<KongListResponse<KongRoute>>(`/services/${serviceId}/routes`),
  getRoute: (id: string) => kongClient.get<KongRoute>(`/routes/${id}`),
  createRoute: (data: Partial<KongRoute>) =>
    kongClient.post<KongRoute>('/routes', data),
  updateRoute: (id: string, data: Partial<KongRoute>) =>
    kongClient.patch<KongRoute>(`/routes/${id}`, data),
  deleteRoute: (id: string) => kongClient.delete(`/routes/${id}`),

  // Upstreams
  getUpstreams: () => kongClient.get<KongListResponse<KongUpstream>>('/upstreams'),
  getUpstream: (id: string) => kongClient.get<KongUpstream>(`/upstreams/${id}`),
  createUpstream: (data: Partial<KongUpstream>) =>
    kongClient.post<KongUpstream>('/upstreams', data),
  updateUpstream: (id: string, data: Partial<KongUpstream>) =>
    kongClient.patch<KongUpstream>(`/upstreams/${id}`, data),
  deleteUpstream: (id: string) => kongClient.delete(`/upstreams/${id}`),

  // Targets
  getTargets: (upstreamId: string) =>
    kongClient.get<KongListResponse<KongTarget>>(`/upstreams/${upstreamId}/targets`),
  createTarget: (upstreamId: string, data: Partial<KongTarget>) =>
    kongClient.post<KongTarget>(`/upstreams/${upstreamId}/targets`, data),
  deleteTarget: (upstreamId: string, targetId: string) =>
    kongClient.delete(`/upstreams/${upstreamId}/targets/${targetId}`),

  // Consumers
  getConsumers: () => kongClient.get<KongListResponse<KongConsumer>>('/consumers'),
  getConsumer: (id: string) => kongClient.get<KongConsumer>(`/consumers/${id}`),
  createConsumer: (data: Partial<KongConsumer>) =>
    kongClient.post<KongConsumer>('/consumers', data),
  updateConsumer: (id: string, data: Partial<KongConsumer>) =>
    kongClient.patch<KongConsumer>(`/consumers/${id}`, data),
  deleteConsumer: (id: string) => kongClient.delete(`/consumers/${id}`),

  // Plugins
  getPlugins: () => kongClient.get<KongListResponse<KongPlugin>>('/plugins'),
  getEnabledPlugins: () =>
    kongClient.get<{ enabled_plugins: string[] }>('/plugins/enabled'),
  getPluginSchema: (pluginName: string) =>
    kongClient.get(`/plugins/schema/${pluginName}`),
  getPlugin: (id: string) => kongClient.get<KongPlugin>(`/plugins/${id}`),
  createPlugin: (data: Partial<KongPlugin>) =>
    kongClient.post<KongPlugin>('/plugins', data),
  updatePlugin: (id: string, data: Partial<KongPlugin>) =>
    kongClient.patch<KongPlugin>(`/plugins/${id}`, data),
  deletePlugin: (id: string) => kongClient.delete(`/plugins/${id}`),

  // Certificates
  getCertificates: () =>
    kongClient.get<KongListResponse<KongCertificate>>('/certificates'),
  getCertificate: (id: string) =>
    kongClient.get<KongCertificate>(`/certificates/${id}`),
  createCertificate: (data: Partial<KongCertificate>) =>
    kongClient.post<KongCertificate>('/certificates', data),
  updateCertificate: (id: string, data: Partial<KongCertificate>) =>
    kongClient.patch<KongCertificate>(`/certificates/${id}`, data),
  deleteCertificate: (id: string) => kongClient.delete(`/certificates/${id}`),

  // SNIs
  getSNIs: () => kongClient.get<KongListResponse<KongSNI>>('/snis'),
  createSNI: (data: Partial<KongSNI>) => kongClient.post<KongSNI>('/snis', data),
  deleteSNI: (id: string) => kongClient.delete(`/snis/${id}`),

  // Declarative Config
  getConfig: () => kongClient.get('/config'),
  postConfig: (yamlConfig: string) =>
    kongClient.post('/config', yamlConfig, {
      headers: { 'Content-Type': 'text/yaml' },
    }),
};

export default kongApi;
