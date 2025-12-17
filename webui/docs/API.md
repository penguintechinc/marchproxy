# Frontend API Service Layer

## Overview

The MarchProxy WebUI uses a centralized API service layer built with Axios for all backend communication. This layer handles authentication, request/response interceptors, error handling, and provides typed service modules for different functional domains.

## Architecture

### Core Components

```
src/services/
├── api.ts              # Axios client configuration and interceptors
├── auth.ts             # Authentication service
├── types.ts            # TypeScript interfaces
├── clusterApi.ts       # Cluster management endpoints
├── serviceApi.ts       # Service management endpoints
├── proxyApi.ts         # Proxy management endpoints
├── certificateApi.ts   # TLS certificate management
├── modulesApi.ts       # Module management
├── observabilityApi.ts # Metrics, tracing, logs
├── securityApi.ts      # Security and compliance
├── licenseApi.ts       # License management
├── enterpriseApi.ts    # Enterprise features
└── clusters.ts         # Cluster service (legacy)
```

## API Client (api.ts)

### Configuration

```typescript
const API_BASE_URL = import.meta.env.REACT_APP_API_URL || 'http://localhost:8000';

export const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' }
});
```

### Request Interceptor

Automatically adds JWT Bearer token from localStorage to all requests:

```typescript
apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

### Response Interceptor

Handles HTTP errors with automatic redirects:

- **401 Unauthorized**: Clears token and redirects to `/login`
- **403 Forbidden**: Checks for license-related errors
- **400 Bad Request**: Returns error details for form validation
- **500 Server Error**: Logs error and shows generic error message

## Service Modules

### Authentication Service (auth.ts)

Manages user authentication and session:

```typescript
authService.login(credentials)      // POST /api/auth/login
authService.logout()                // POST /api/auth/logout
authService.getCurrentUser()        // GET /api/auth/me
authService.refreshToken()          // POST /api/auth/refresh
authService.enableTwoFactor()       // POST /api/auth/2fa/enable
authService.verifyTwoFactor(code)   // POST /api/auth/2fa/verify
authService.disableTwoFactor()      // POST /api/auth/2fa/disable
authService.changePassword(data)    // PUT /api/auth/password
authService.register(data)          // POST /api/auth/register
```

### Cluster API (clusterApi.ts)

Manages cluster configuration and operations:

```typescript
clusterApi.listClusters()           // GET /api/clusters
clusterApi.getCluster(id)           // GET /api/clusters/{id}
clusterApi.createCluster(data)      // POST /api/clusters
clusterApi.updateCluster(id, data)  // PATCH /api/clusters/{id}
clusterApi.deleteCluster(id)        // DELETE /api/clusters/{id}
clusterApi.rotateApiKey(id)         // POST /api/clusters/{id}/rotate-api-key
```

### Service API (serviceApi.ts)

Manages service definitions and proxying rules:

```typescript
serviceApi.listServices()           // GET /api/services
serviceApi.getService(id)           // GET /api/services/{id}
serviceApi.createService(data)      // POST /api/services
serviceApi.updateService(id, data)  // PATCH /api/services/{id}
serviceApi.deleteService(id)        // DELETE /api/services/{id}
serviceApi.rotateToken(id)          // POST /api/services/{id}/rotate-token
```

### Proxy API (proxyApi.ts)

Monitors proxy instances and health:

```typescript
proxyApi.listProxies()              // GET /api/proxies
proxyApi.getProxy(id)               // GET /api/proxies/{id}
proxyApi.deregisterProxy(id)        // DELETE /api/proxies/{id}
proxyApi.getProxyMetrics(id)        // GET /api/proxies/{id}/metrics
proxyApi.reportMetrics(id, data)    // POST /api/proxies/{id}/metrics
```

### Certificate API (certificateApi.ts)

Manages TLS certificates with multiple sources:

```typescript
certificateApi.listCertificates()           // GET /api/certificates
certificateApi.getCertificate(id)           // GET /api/certificates/{id}
certificateApi.uploadCertificate(data)      // POST /api/certificates
certificateApi.updateCertificate(id, data)  // PUT /api/certificates/{id}
certificateApi.deleteCertificate(id)        // DELETE /api/certificates/{id}
certificateApi.renewCertificate(id)         // POST /api/certificates/{id}/renew
certificateApi.listExpiringCerts()          // GET /api/certificates/expiring/list
certificateApi.batchRenew(ids)              // POST /api/certificates/batch-renew
```

Supports integration with:
- Direct certificate upload
- Infisical secrets management
- HashiCorp Vault
- Automatic renewal scheduling

### Modules API (modulesApi.ts)

Manages custom routing modules:

```typescript
modulesApi.listModules()            // GET /api/modules
modulesApi.getModule(id)            // GET /api/modules/{id}
modulesApi.createModule(data)       // POST /api/modules
modulesApi.updateModule(id, data)   // PATCH /api/modules/{id}
modulesApi.deleteModule(id)         // DELETE /api/modules/{id}
modulesApi.getModuleHealth(id)      // GET /api/modules/{id}/health
modulesApi.enableModule(id)         // POST /api/modules/{id}/enable
modulesApi.disableModule(id)        // POST /api/modules/{id}/disable

// Module Routes
modulesApi.listRoutes()             // GET /api/modules/routes
modulesApi.getRoute(id)             // GET /api/modules/routes/{id}
modulesApi.createRoute(data)        // POST /api/modules/routes
modulesApi.updateRoute(id, data)    // PATCH /api/modules/routes/{id}
modulesApi.deleteRoute(id)          // DELETE /api/modules/routes/{id}
modulesApi.enableRoute(id)          // POST /api/modules/routes/{id}/enable
modulesApi.disableRoute(id)         // POST /api/modules/routes/{id}/disable

// Deployments
modulesApi.listDeployments()        // GET /api/deployments
modulesApi.createDeployment(data)   // POST /api/deployments
modulesApi.updateDeployment(id, data) // PATCH /api/deployments/{id}
modulesApi.promoteDeployment(id)    // POST /api/deployments/{id}/promote
modulesApi.rollbackDeployment(id)   // POST /api/deployments/{id}/rollback
modulesApi.getDeploymentHealth(id)  // GET /api/deployments/{id}/health
```

### Observability API (observabilityApi.ts)

Handles metrics, tracing, and logging:

```typescript
// Metrics
observabilityApi.getMetrics(query)          // GET /api/observability/metrics
observabilityApi.queryMetrics(params)       // POST /api/observability/metrics/query

// Tracing (Distributed Tracing)
observabilityApi.listTracingConfig()        // GET /api/observability/tracing
observabilityApi.createTracingConfig(data)  // POST /api/observability/tracing
observabilityApi.updateTracingConfig(data)  // PATCH /api/observability/tracing/{id}
observabilityApi.deleteTracingConfig(id)    // DELETE /api/observability/tracing/{id}
observabilityApi.testTracing(config)        // POST /api/observability/tracing/test
observabilityApi.searchSpans(query)         // POST /api/observability/spans/search
observabilityApi.getSpanDetails(id)         // GET /api/observability/spans/{id}

// Logs
observabilityApi.queryLogs(params)          // POST /api/observability/logs/query
observabilityApi.streamLogs()               // WebSocket /api/observability/logs/stream

// Alerts
observabilityApi.listAlerts()               // GET /api/observability/alerts
observabilityApi.createAlert(data)          // POST /api/observability/alerts
observabilityApi.updateAlert(id, data)      // PATCH /api/observability/alerts/{id}
observabilityApi.deleteAlert(id)            // DELETE /api/observability/alerts/{id}
```

### Security API (securityApi.ts)

Manages security policies and compliance:

```typescript
// Zero-Trust Security
securityApi.getZeroTrustStatus()            // GET /api/security/zero-trust/status
securityApi.toggleZeroTrust(enabled)        // POST /api/security/zero-trust/toggle
securityApi.listOPAPolicies()               // GET /api/security/policies
securityApi.createOPAPolicy(policy)         // POST /api/security/policies
securityApi.updateOPAPolicy(id, policy)     // PATCH /api/security/policies/{id}
securityApi.deleteOPAPolicy(id)             // DELETE /api/security/policies/{id}
securityApi.validateOPAPolicy(policy)       // POST /api/security/policies/validate
securityApi.testOPAPolicy(policy, input)    // POST /api/security/policies/test

// mTLS Configuration
securityApi.getMTLSConfig()                 // GET /api/security/mtls
securityApi.updateMTLSConfig(config)        // PATCH /api/security/mtls

// Audit Logs
securityApi.queryAuditLogs(params)          // POST /api/security/audit-logs/query
securityApi.exportAuditLogs(params)         // POST /api/security/audit-logs/export
securityApi.verifyAuditChain()              // POST /api/security/audit-logs/verify

// Compliance
securityApi.generateComplianceReport(type)  // POST /api/security/compliance/generate
securityApi.exportComplianceReport(id)      // GET /api/security/compliance/{id}/export
```

Supports compliance standards:
- SOC2
- HIPAA
- PCI-DSS
- ISO 27001
- Custom policies

### License API (licenseApi.ts)

Manages product licensing and feature entitlements:

```typescript
licenseApi.validateLicense(key)     // POST /api/license/validate
licenseApi.getCurrentLicense()      // GET /api/license/current
licenseApi.getFeatures()            // GET /api/license/features
licenseApi.checkFeature(name)       // GET /api/license/features/{name}
```

### Enterprise API (enterpriseApi.ts)

Enterprise-only features:

```typescript
// Multi-Cloud Routing
enterpriseApi.listCloudRoutes()     // GET /api/enterprise/routes
enterpriseApi.createCloudRoute(data) // POST /api/enterprise/routes
enterpriseApi.updateCloudRoute(id, data) // PATCH /api/enterprise/routes/{id}
enterpriseApi.deleteCloudRoute(id)  // DELETE /api/enterprise/routes/{id}
enterpriseApi.testFailover(id)      // POST /api/enterprise/routes/{id}/test-failover
enterpriseApi.getCostAnalytics()    // GET /api/enterprise/cost-analytics

// Auto-Scaling
enterpriseApi.getScalingPolicy(module_id) // GET /api/modules/{id}/scaling
enterpriseApi.createScalingPolicy(data)   // POST /api/modules/{id}/scaling
enterpriseApi.updateScalingPolicy(data)   // PUT /api/modules/{id}/scaling
enterpriseApi.deleteScalingPolicy(id)     // DELETE /api/modules/{id}/scaling
enterpriseApi.enableScaling(id)           // POST /api/modules/{id}/scaling/enable
enterpriseApi.disableScaling(id)          // POST /api/modules/{id}/scaling/disable

// Traffic Shaping
enterpriseApi.listTrafficPolicies()       // GET /api/enterprise/traffic-policies
enterpriseApi.createTrafficPolicy(data)   // POST /api/enterprise/traffic-policies
enterpriseApi.updateTrafficPolicy(id, data) // PATCH /api/enterprise/traffic-policies/{id}
enterpriseApi.deleteTrafficPolicy(id)     // DELETE /api/enterprise/traffic-policies/{id}
enterpriseApi.enableTrafficPolicy(id)     // POST /api/enterprise/traffic-policies/{id}/enable
enterpriseApi.disableTrafficPolicy(id)    // POST /api/enterprise/traffic-policies/{id}/disable
```

## Type Definitions (types.ts)

Key TypeScript interfaces for API contracts:

```typescript
// Authentication
interface LoginRequest {
  username: string;
  password: string;
  totp_code?: string;  // For 2FA
}

interface LoginResponse {
  token: string;
  user: User;
  expires_in: number;
}

interface User {
  id: string;
  email: string;
  username: string;
  role: 'admin' | 'service-owner' | 'user';
  clusters: string[];
  created_at: string;
  last_login?: string;
}

// Clusters
interface Cluster {
  id: string;
  name: string;
  description: string;
  api_key: string;
  status: 'active' | 'inactive';
  created_at: string;
  proxy_count: number;
  license_tier: 'community' | 'enterprise';
}

// Services
interface Service {
  id: string;
  name: string;
  cluster_id: string;
  port: number | string;  // Can be single, range, or list
  protocol: 'tcp' | 'udp' | 'icmp';
  upstream_url: string;
  auth_token?: string;
  status: 'active' | 'inactive';
  created_at: string;
}

// Proxies
interface Proxy {
  id: string;
  cluster_id: string;
  hostname: string;
  status: 'online' | 'offline' | 'degraded';
  version: string;
  cpu_usage: number;
  memory_usage: number;
  connections: number;
  last_heartbeat: string;
}

// Certificates
interface Certificate {
  id: string;
  name: string;
  subject: string;
  issuer: string;
  valid_from: string;
  valid_until: string;
  thumbprint: string;
  source: 'upload' | 'infisical' | 'vault';
  auto_renew: boolean;
  status: 'valid' | 'expired' | 'expiring-soon';
}

// Modules
interface Module {
  id: string;
  name: string;
  version: string;
  type: 'routing' | 'plugin' | 'middleware';
  status: 'enabled' | 'disabled';
  routes: Route[];
  deployments: Deployment[];
  health: 'healthy' | 'degraded' | 'unhealthy';
}

interface Route {
  id: string;
  path: string;
  method: 'GET' | 'POST' | 'PATCH' | 'DELETE' | '*';
  upstream: string;
  enabled: boolean;
  rules: RouteRule[];
}

interface Deployment {
  id: string;
  module_id: string;
  version: string;
  status: 'active' | 'inactive' | 'rolling';
  created_at: string;
  traffic_weight: number;  // 0-100
}

// License
interface License {
  key: string;
  tier: 'community' | 'enterprise';
  product: string;
  issued_at: string;
  expires_at: string;
  proxy_limit: number;
  features: string[];
  is_valid: boolean;
}

// Observability
interface Metric {
  name: string;
  value: number;
  timestamp: string;
  labels: Record<string, string>;
}

interface Span {
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  operation: string;
  start_time: string;
  duration: number;  // milliseconds
  status: 'ok' | 'error';
  tags: Record<string, any>;
  logs: SpanLog[];
}

// Errors
interface ApiError {
  code: string;
  message: string;
  details?: Record<string, any>;
  request_id: string;
  timestamp: string;
}
```

## Usage Examples

### In React Components

```typescript
import { useEffect, useState } from 'react';
import { serviceApi } from '../services/serviceApi';

export function ServiceList() {
  const [services, setServices] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    serviceApi.listServices()
      .then(setServices)
      .catch((err: ApiError) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div>Loading...</div>;
  if (error) return <div>Error: {error}</div>;

  return (
    <ul>
      {services.map(service => (
        <li key={service.id}>{service.name}</li>
      ))}
    </ul>
  );
}
```

### With Error Handling

```typescript
async function createServiceWithErrorHandling(data: ServiceCreateRequest) {
  try {
    const response = await serviceApi.createService(data);
    return response;
  } catch (error) {
    if (axios.isAxiosError(error)) {
      const apiError = error.response?.data as ApiError;

      if (error.response?.status === 400) {
        // Handle validation errors
        console.error('Validation error:', apiError.details);
      } else if (error.response?.status === 409) {
        // Handle conflict (service already exists)
        console.error('Service already exists');
      } else {
        console.error('Server error:', apiError.message);
      }
    }
  }
}
```

### With React Query

```typescript
import { useQuery, useMutation } from 'react-query';
import { serviceApi } from '../services/serviceApi';

function useServices() {
  return useQuery('services', serviceApi.listServices);
}

function useCreateService() {
  return useMutation(
    (data: ServiceCreateRequest) => serviceApi.createService(data),
    {
      onSuccess: () => {
        // Invalidate and refetch
        queryClient.invalidateQueries('services');
      }
    }
  );
}
```

## Environment Variables

```bash
# API Configuration
REACT_APP_API_URL=http://localhost:8000
REACT_APP_API_TIMEOUT=30000

# Feature Flags
REACT_APP_ENABLE_ENTERPRISE=false
REACT_APP_ENABLE_DEBUG_LOGGING=false
```

## Error Handling

The API layer implements comprehensive error handling:

1. **Network Errors**: Timeout, connection refused, DNS resolution
2. **HTTP Errors**: 4xx client errors, 5xx server errors
3. **Validation Errors**: Form field validation from server
4. **Authorization Errors**: Token expiration, insufficient permissions
5. **License Errors**: Feature not available in tier

All errors are normalized to the `ApiError` interface for consistent handling across the application.

## Authentication Flow

1. User submits login credentials
2. `authService.login()` calls backend `/api/auth/login`
3. JWT token returned and stored in localStorage
4. Request interceptor adds token to all subsequent requests
5. Response interceptor checks for 401 (expired token)
6. Token refresh triggered automatically on 401
7. Logout clears token and redirects to `/login`

## Rate Limiting

API implements standard HTTP rate limiting headers:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1701720000
```

Client should respect these headers and implement exponential backoff on 429 responses.

## Performance Optimization

- Request deduplication for identical queries
- Response caching with React Query
- Lazy-loading of large datasets with pagination
- WebSocket connections for real-time data (metrics, logs)
- Batch operations for multiple resource operations

## Testing

All API services are testable with mocked responses:

```typescript
import { rest } from 'msw';
import { server } from './mocks/server';

test('should load services', async () => {
  server.use(
    rest.get('/api/services', (req, res, ctx) => {
      return res(ctx.json([
        { id: '1', name: 'Service 1' }
      ]));
    })
  );

  const services = await serviceApi.listServices();
  expect(services).toHaveLength(1);
});
```
