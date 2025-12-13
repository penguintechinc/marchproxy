/**
 * TypeScript Type Definitions for MarchProxy API
 */

// User and Authentication Types
export interface User {
  id: number;
  username: string;
  email: string;
  role: 'administrator' | 'service_owner';
  is_active: boolean;
  created_at: string;
  updated_at: string;
  clusters?: number[];
}

export interface LoginRequest {
  username: string;
  password: string;
  totp_code?: string;
}

export interface LoginResponse {
  token: string;
  user: User;
  expires_at: string;
}

export interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
}

// Cluster Types
export interface Cluster {
  id: number;
  name: string;
  description: string;
  api_key: string;
  syslog_server?: string;
  syslog_port?: number;
  auth_log_enabled: boolean;
  netflow_log_enabled: boolean;
  debug_log_enabled: boolean;
  created_at: string;
  updated_at: string;
  proxy_count?: number;
}

// Proxy Types
export interface Proxy {
  id: number;
  cluster_id: number;
  hostname: string;
  ip_address: string;
  status: 'active' | 'inactive' | 'error';
  last_heartbeat: string;
  version: string;
  capabilities: string[];
  created_at: string;
  updated_at: string;
}

// Service Types
export interface Service {
  id: number;
  cluster_id: number;
  name: string;
  description: string;
  destination_fqdn: string;
  destination_port: string;
  protocol: 'TCP' | 'UDP' | 'ICMP' | 'HTTPS' | 'HTTP3';
  auth_method: 'base64_token' | 'jwt';
  token_ttl?: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  owner_id?: number;
}

export interface ServiceToken {
  id: number;
  service_id: number;
  token: string;
  created_at: string;
  expires_at?: string;
}

// License Types
export interface License {
  key: string;
  tier: 'community' | 'enterprise';
  features: string[];
  max_proxies: number;
  valid_until: string;
  is_valid: boolean;
}

// Statistics Types
export interface DashboardStats {
  total_proxies: number;
  active_proxies: number;
  total_services: number;
  active_services: number;
  total_clusters: number;
  license_tier: 'community' | 'enterprise';
  license_valid: boolean;
}

export interface ProxyStats {
  proxy_id: number;
  hostname: string;
  connections_active: number;
  connections_total: number;
  bytes_sent: number;
  bytes_received: number;
  errors: number;
  timestamp: string;
}

// API Response Types
export interface ApiResponse<T> {
  data: T;
  message?: string;
  status: string;
}

export interface ApiError {
  detail: string;
  status_code: number;
  feature?: string;
}

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

// Health Check Types
export interface HealthCheck {
  status: 'healthy' | 'unhealthy' | 'degraded';
  version: string;
  database: boolean;
  redis?: boolean;
  timestamp: string;
}

// Metrics Types
export interface Metrics {
  [key: string]: number | string;
}

// WebSocket Message Types
export interface WSMessage {
  type: 'proxy_update' | 'service_update' | 'stats_update' | 'alert';
  data: any;
  timestamp: string;
}

// Enterprise Feature Types

// Traffic Shaping & QoS Types
export interface QoSPolicy {
  id: number;
  service_id: number;
  priority: 'P0' | 'P1' | 'P2' | 'P3';
  bandwidth_limit_mbps?: number;
  burst_size_kb?: number;
  rate_limit_pps?: number;
  dscp_marking?: number;
  token_bucket_enabled: boolean;
  token_bucket_rate?: number;
  token_bucket_burst?: number;
  created_at: string;
  updated_at: string;
}

export interface BandwidthAllocation {
  priority: string;
  allocated_mbps: number;
  percentage: number;
  color: string;
}

// Multi-Cloud Routing Types
export interface CloudRoute {
  id: number;
  service_id: number;
  cloud_provider: 'AWS' | 'GCP' | 'Azure' | 'Custom';
  backend_url: string;
  backend_ip: string;
  region: string;
  weight: number;
  priority: number;
  health_status: 'healthy' | 'unhealthy' | 'degraded';
  health_check_url?: string;
  health_check_interval_seconds: number;
  last_health_check?: string;
  rtt_ms?: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface RoutingAlgorithm {
  type: 'latency' | 'cost' | 'geo' | 'weighted_rr';
  name: string;
  description: string;
}

export interface BackendHealth {
  route_id: number;
  backend_url: string;
  status: 'healthy' | 'unhealthy' | 'degraded';
  last_check: string;
  rtt_ms: number;
  success_rate: number;
  error_count: number;
}

export interface CloudBackendLocation {
  id: number;
  name: string;
  latitude: number;
  longitude: number;
  cloud_provider: string;
  region: string;
  status: 'healthy' | 'unhealthy' | 'degraded';
  rtt_ms?: number;
}

// Cost Analytics Types
export interface CostAnalytics {
  total_cost_usd: number;
  period_start: string;
  period_end: string;
  breakdown_by_provider: CostBreakdown[];
  breakdown_by_service: CostBreakdown[];
  breakdown_by_region: CostBreakdown[];
  monthly_projection: number;
  yearly_projection: number;
}

export interface CostBreakdown {
  label: string;
  cost_usd: number;
  percentage: number;
  egress_gb: number;
}

export interface CostOptimization {
  recommendation: string;
  potential_savings_usd: number;
  priority: 'high' | 'medium' | 'low';
  implementation_effort: 'easy' | 'moderate' | 'complex';
}

export interface CostTimeSeries {
  timestamp: string;
  cost_usd: number;
  provider?: string;
}

// NUMA Configuration Types
export interface NUMATopology {
  node_count: number;
  nodes: NUMANode[];
  total_cpus: number;
  total_memory_gb: number;
}

export interface NUMANode {
  node_id: number;
  cpu_list: number[];
  cpu_count: number;
  memory_gb: number;
  worker_count: number;
  affinity_enabled: boolean;
  performance_score?: number;
}

export interface NUMAConfig {
  enabled: boolean;
  auto_affinity: boolean;
  worker_allocation: WorkerAllocation[];
  memory_locality_optimization: boolean;
}

export interface WorkerAllocation {
  numa_node: number;
  worker_count: number;
  cpu_pinning: number[];
}

export interface NUMAMetrics {
  node_id: number;
  cpu_usage_percent: number;
  memory_usage_percent: number;
  local_memory_access_percent: number;
  remote_memory_access_percent: number;
  throughput_mbps: number;
}

// Module Management Types (Unified NLB Architecture)
export interface Module {
  id: number;
  name: string;
  type: 'NLB' | 'ALB' | 'DBLB' | 'AILB' | 'RTMP';
  description: string;
  is_enabled: boolean;
  container_image: string;
  grpc_address: string;
  health_status: 'healthy' | 'unhealthy' | 'degraded';
  version: string;
  created_at: string;
  updated_at: string;
}

export interface ModuleRoute {
  id: number;
  module_id: number;
  name: string;
  protocol: string;
  backend_url: string;
  backend_port: number;
  is_active: boolean;
  rate_limit_rps?: number;
  rate_limit_connections?: number;
  rate_limit_bandwidth_mbps?: number;
  priority: 'P0' | 'P1' | 'P2' | 'P3';
  created_at: string;
  updated_at: string;
}

export interface AutoScalingPolicy {
  id: number;
  module_id: number;
  metric_type: 'cpu' | 'memory' | 'connections' | 'latency';
  scale_up_threshold: number;
  scale_down_threshold: number;
  min_instances: number;
  max_instances: number;
  cooldown_seconds: number;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface BlueGreenDeployment {
  id: number;
  module_id: number;
  blue_version: string;
  green_version: string;
  traffic_weight_blue: number;
  traffic_weight_green: number;
  health_check_url: string;
  auto_rollback_enabled: boolean;
  status: 'active' | 'transitioning' | 'rolled_back';
  created_at: string;
  updated_at: string;
}

export interface ModuleMetrics {
  module_id: number;
  cpu_percent: number;
  memory_percent: number;
  active_connections: number;
  requests_per_second: number;
  average_latency_ms: number;
  error_rate: number;
  timestamp: string;
}

export interface ModuleInstance {
  id: string;
  module_id: number;
  container_id: string;
  status: 'running' | 'stopped' | 'starting' | 'stopping';
  version: string;
  started_at: string;
}
