# MarchProxy API Coverage Audit

**Generated**: 2025-12-16
**Audit Type**: API Endpoint to WebUI Component Coverage Analysis

## Executive Summary

This audit document maps all API endpoints defined in the backend API server to corresponding WebUI components for management and monitoring. The analysis identifies gaps where backend functionality lacks frontend management interfaces.

---

## API Coverage Matrix

| API Category | Endpoints Count | Primary Endpoints | WebUI Component | Coverage Status |
|---|---|---|---|---|
| **Authentication** | 9 | login, register, refresh, 2fa (enable/verify/disable), change-password, me, logout | Common/LicenseGate.tsx (partial) | ⚠️ Partial |
| **User Management** | 7 | list, create, get, update, delete, cluster-assignments, service-assignments | None | ❌ Missing |
| **Clusters** | 6 | list, create, get, update, delete, rotate-api-key | None | ❌ Missing |
| **Proxies** | 7 | register, heartbeat, list, get, deregister, metrics (get/post) | None | ❌ Missing |
| **Services** | 6 | list, create, get, update, delete, rotate-token | None | ❌ Missing |
| **Certificates** | 8 | list, create, get, update, delete, renew, expiring-list, batch-renew | None | ❌ Missing |
| **Modules** | 8 | list, create, get, update, delete, health, enable, disable | Modules/ModuleCard.tsx | ⚠️ Partial |
| **Module Routes** | 7 | list, create, get, update, delete, enable, disable | Modules/RouteEditor.tsx | ⚠️ Partial |
| **Module Deployments** | 6 | list, create, get, update, promote, rollback, health | Deployments/TrafficSlider.tsx | ⚠️ Partial |
| **Module Scaling** | 6 | get-policy, create-policy, update-policy, delete-policy, enable, disable | None | ❌ Missing |
| **Configuration** | 3 | get-cluster-config, validate-config, get-version | None | ❌ Missing |
| **Certificates (TLS)** | 8 | list, create, get, update, delete, renew, expiring-list, batch-renew | None | ❌ Missing |
| **Observability (Enterprise)** | 11 | list-tracing, create-tracing, get-tracing, update-tracing, delete-tracing, stats, enable, disable, test, search-spans, etc. | Observability/ServiceGraph.tsx | ⚠️ Partial |
| **Traffic Shaping (Enterprise)** | 7 | list-policies, create-policy, get-policy, update-policy, delete-policy, enable, disable | Enterprise.disabled/ | ❌ Missing (disabled) |
| **Multi-Cloud Routing (Enterprise)** | 10 | list-routes, create-route, get-route, update-route, delete-route, health, enable, disable, test-failover, cost-analytics | Enterprise/CloudHealthMap.tsx | ⚠️ Partial |
| **Zero-Trust Security (Enterprise)** | 14 | status, toggle, policies (CRUD), validate, test, audit-logs (query/export/verify), compliance-reports (generate/export) | Enterprise.disabled/ | ❌ Missing (disabled) |
| **TOTALS** | **129 endpoints** | — | — | **❌ 9/17 categories missing** |

---

## Detailed Coverage Analysis

### ✅ Full Coverage (0 categories)
*No API categories have complete WebUI coverage.*

### ⚠️ Partial Coverage (5 categories)

#### 1. Authentication (9 endpoints)
- **Status**: ⚠️ Partial
- **Component**: `Common/LicenseGate.tsx`
- **Covered**: License validation only
- **Gaps**:
  - No login/register UI
  - No 2FA setup/management UI
  - No password change UI
  - No user profile/account settings UI
- **Required Components**:
  - `LoginPage.tsx` - Login and registration
  - `AccountSettings.tsx` - Profile and password management
  - `TwoFactorSetup.tsx` - 2FA configuration

#### 2. Modules (8 endpoints)
- **Status**: ⚠️ Partial
- **Component**: `Modules/ModuleCard.tsx`
- **Covered**: Module display and basic card management
- **Gaps**:
  - No create/edit module UI
  - No enable/disable module UI
  - No detailed module configuration
- **Required Components**:
  - `ModuleEditor.tsx` - Create and update modules
  - `ModuleDetail.tsx` - Full module information and controls

#### 3. Module Routes (7 endpoints)
- **Status**: ⚠️ Partial
- **Component**: `Modules/RouteEditor.tsx`
- **Covered**: Route editing interface
- **Gaps**:
  - Limited to RouteEditor component
  - No full CRUD list view
  - No enable/disable toggle UI
- **Required Components**:
  - `RouteList.tsx` - List all routes with bulk operations
  - `RouteDetail.tsx` - Detailed route configuration

#### 4. Module Deployments (6 endpoints)
- **Status**: ⚠️ Partial
- **Component**: `Deployments/TrafficSlider.tsx`
- **Covered**: Traffic weight management for deployments
- **Gaps**:
  - No deployment creation UI
  - No deployment list view
  - No health monitoring UI
  - No rollback confirmation UI
- **Required Components**:
  - `DeploymentList.tsx` - List all deployments
  - `DeploymentCreate.tsx` - Create new deployments
  - `DeploymentHealth.tsx` - Health status monitoring
  - `RollbackConfirm.tsx` - Rollback workflow

#### 5. Observability/Distributed Tracing (11 endpoints)
- **Status**: ⚠️ Partial
- **Component**: `Observability/ServiceGraph.tsx`
- **Covered**: Service relationship visualization only
- **Gaps**:
  - No tracing configuration UI
  - No spans search/filtering
  - No tracing stats dashboard
  - No sampling configuration
- **Required Components**:
  - `TracingConfig.tsx` - Configure tracing backend
  - `SpansViewer.tsx` - Search and analyze spans
  - `TracingStats.tsx` - View tracing statistics

#### 6. Multi-Cloud Routing (10 endpoints)
- **Status**: ⚠️ Partial
- **Component**: `Enterprise/CloudHealthMap.tsx`
- **Covered**: Cloud health visualization
- **Gaps**:
  - No route table management UI
  - No failover testing UI
  - No cost analytics dashboard
  - No route creation/edit
- **Required Components**:
  - `RouteTableManager.tsx` - CRUD for route tables
  - `HealthProbeConfig.tsx` - Configure health checks
  - `CostAnalytics.tsx` - Cost optimization views
  - `FailoverTester.tsx` - Failover testing interface

### ❌ Missing Coverage (9 categories)

#### 1. User Management (7 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `AdminPanel/UserManagement.tsx`
- **Endpoints Not Managed**:
  - GET /users - List users
  - POST /users - Create user
  - GET /users/{user_id} - Get user details
  - PATCH /users/{user_id} - Update user
  - DELETE /users/{user_id} - Delete/deactivate user
  - POST /users/{user_id}/cluster-assignments - Assign to cluster
  - POST /users/{user_id}/service-assignments - Assign to service
- **Business Impact**: High - Admin users cannot manage user accounts and permissions

#### 2. Cluster Management (6 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `AdminPanel/ClusterManagement.tsx`
- **Endpoints Not Managed**:
  - GET /clusters - List clusters
  - POST /clusters - Create cluster
  - GET /clusters/{cluster_id} - Get cluster
  - PATCH /clusters/{cluster_id} - Update cluster
  - DELETE /clusters/{cluster_id} - Delete cluster
  - POST /clusters/{cluster_id}/rotate-api-key - Rotate API key
- **Business Impact**: Critical - Cannot manage clusters without direct API calls
- **Note**: Only 1 default cluster in Community tier; Enterprise supports multiple

#### 3. Proxy Management (7 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `Dashboard/ProxyMonitoring.tsx`
- **Endpoints Not Managed**:
  - POST /proxies/register - Register proxy (automated)
  - POST /proxies/heartbeat - Heartbeat (automated)
  - GET /proxies - List proxies
  - GET /proxies/{proxy_id} - Get proxy details
  - DELETE /proxies/{proxy_id} - Deregister proxy
  - GET /proxies/{proxy_id}/metrics - View proxy metrics
  - POST /proxies/{proxy_id}/metrics - Report metrics (automated)
- **Business Impact**: Critical - No visibility into proxy fleet status and health
- **Note**: Last 2 are automated but need UI dashboard

#### 4. Service Management (6 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `ServiceManagement/ServiceCRUD.tsx`
- **Endpoints Not Managed**:
  - GET /services - List services
  - POST /services - Create service
  - GET /services/{service_id} - Get service
  - PATCH /services/{service_id} - Update service
  - DELETE /services/{service_id} - Delete service
  - POST /services/{service_id}/rotate-token - Rotate auth token
- **Business Impact**: Critical - Core functionality requires direct API access
- **Note**: Services define what traffic the proxies forward

#### 5. Module Auto-Scaling (6 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `ModuleManagement/ScalingPolicy.tsx`
- **Endpoints Not Managed**:
  - GET /modules/{module_id}/scaling - Get scaling policy
  - POST /modules/{module_id}/scaling - Create policy
  - PUT /modules/{module_id}/scaling - Update policy
  - DELETE /modules/{module_id}/scaling - Delete policy
  - POST /modules/{module_id}/scaling/enable - Enable
  - POST /modules/{module_id}/scaling/disable - Disable
- **Business Impact**: Medium - Auto-scaling feature cannot be configured

#### 6. Configuration Management (3 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `AdminPanel/ConfigurationDashboard.tsx`
- **Endpoints Not Managed**:
  - GET /config/{cluster_id} - Get cluster config
  - GET /config/validate/{cluster_id} - Validate config
  - GET /config/version/{cluster_id} - Get version
- **Business Impact**: Low - These are primarily for proxy initialization
- **Note**: Diagnostic endpoints for troubleshooting

#### 7. Certificate Management (8 endpoints)
- **Status**: ❌ Missing
- **Required Component**: `CertificateManagement/CertificateCRUD.tsx`
- **Endpoints Not Managed**:
  - GET /certificates - List certificates
  - POST /certificates - Create/upload certificate
  - GET /certificates/{cert_id} - Get certificate
  - PUT /certificates/{cert_id} - Update certificate
  - DELETE /certificates/{cert_id} - Delete certificate
  - POST /certificates/{cert_id}/renew - Renew certificate
  - GET /certificates/expiring/list - List expiring certificates
  - POST /certificates/batch-renew - Batch renewal
- **Business Impact**: Critical - TLS certificate management is essential
- **Features**: Support for upload, Infisical, and HashiCorp Vault integration

#### 8. Traffic Shaping / QoS Policies (7 endpoints)
- **Status**: ❌ Missing (Disabled)
- **Path**: `Enterprise.disabled/TrafficShaping/`
- **Endpoints Not Managed**:
  - GET /policies - List QoS policies
  - POST /policies - Create policy
  - GET /policies/{policy_id} - Get policy
  - PUT /policies/{policy_id} - Update policy
  - DELETE /policies/{policy_id} - Delete policy
  - POST /policies/{policy_id}/enable - Enable
  - POST /policies/{policy_id}/disable - Disable
- **Business Impact**: Medium (Enterprise) - QoS and traffic management features
- **Note**: Enterprise-only feature; components exist in disabled state
- **Features**: Bandwidth limiting, priority queues, DSCP marking

#### 9. Zero-Trust Security & Compliance (14 endpoints)
- **Status**: ❌ Missing (Disabled)
- **Path**: `Enterprise.disabled/ZeroTrust/`
- **Endpoints Not Managed**:
  - GET /status - Get zero-trust status
  - POST /toggle - Enable/disable zero-trust
  - GET /policies - List OPA policies
  - POST /policies - Create policy
  - GET /policies/{policy_name} - Get policy
  - DELETE /policies/{policy_name} - Delete policy
  - POST /policies/validate - Validate policy
  - POST /policies/test - Test policy
  - GET /audit-logs - Query audit logs
  - GET /audit-logs/export - Export logs
  - POST /audit-logs/verify - Verify audit chain
  - POST /compliance-reports/generate - Generate report
  - POST /compliance-reports/export - Export report
  - More...
- **Business Impact**: High (Enterprise) - Compliance and security audit critical
- **Note**: Enterprise-only feature; components exist in disabled state
- **Features**: OPA policy management, audit logging, compliance reporting (SOC2, HIPAA, PCI-DSS)

---

## Implementation Roadmap

### Phase 1: Critical Path (Foundation)
Priority: High - Core system functionality

1. **User Management** (`AdminPanel/UserManagement.tsx`)
   - User CRUD operations
   - Cluster and service assignments
   - Role-based access control UI

2. **Cluster Management** (`AdminPanel/ClusterManagement.tsx`)
   - Cluster CRUD operations
   - API key rotation
   - License validation display

3. **Service Management** (`ServiceManagement/ServiceCRUD.tsx`)
   - Service CRUD operations
   - Authentication token management
   - Cluster selection

4. **Proxy Monitoring** (`Dashboard/ProxyMonitoring.tsx`)
   - Proxy fleet status
   - Metrics visualization
   - Health indicators

### Phase 2: Secondary Path (Essential Features)
Priority: High - Core operations completeness

5. **Certificate Management** (`CertificateManagement/CertificateCRUD.tsx`)
   - Certificate upload/import
   - Renewal scheduling
   - Expiration alerts
   - Integration with Infisical/Vault

6. **Module Completion**
   - `ModuleEditor.tsx` - Create/edit modules
   - `ModuleDetail.tsx` - Full configuration view

7. **Deployment Completion**
   - `DeploymentList.tsx`
   - `DeploymentCreate.tsx`
   - `DeploymentHealth.tsx`

### Phase 3: Advanced Features (Optimization)
Priority: Medium - Performance and observability

8. **Auto-Scaling Configuration** (`ModuleManagement/ScalingPolicy.tsx`)
   - Policy creation and management
   - Metric threshold configuration
   - Historical scaling view

9. **Configuration Dashboard** (`AdminPanel/ConfigurationDashboard.tsx`)
   - System configuration
   - Proxy version management
   - Configuration validation

### Phase 4: Enterprise Features (License-gated)
Priority: Medium - Enterprise customers

10. **Zero-Trust Security** (`Enterprise/ZeroTrust/*.tsx`)
    - Activate components in `Enterprise.disabled/ZeroTrust/`
    - OPA policy editor and validator
    - Audit log viewer and exporter
    - Compliance report generator

11. **Traffic Shaping** (`Enterprise/TrafficShaping/*.tsx`)
    - Activate components in `Enterprise.disabled/TrafficShaping/`
    - QoS policy management
    - Bandwidth and priority configuration

12. **Enhanced Observability**
    - `TracingConfig.tsx` - Tracing setup
    - `SpansViewer.tsx` - Span analysis
    - `TracingStats.tsx` - Statistics dashboard

---

## Coverage Statistics

```
Total API Categories:        17
Categories with Coverage:     5 (29%)
  - Full Coverage:            0 (0%)
  - Partial Coverage:         5 (29%)
  - Missing Coverage:         9 (53%)
  - Missing + Disabled:       3 (18%)

Total API Endpoints:          129
Estimated UI Coverage:        ~40% (52 endpoints have some UI)
Missing UI Endpoints:         ~60% (77 endpoints need UI)

Enterprise Features:          3 categories (24 endpoints)
  - Disabled/Missing:         2 categories
  - Partial Coverage:         1 category
```

---

## Key Gaps Summary

| Gap Category | Endpoint Count | Severity | User Impact |
|---|---|---|---|
| User/Admin Management | 7 | Critical | Cannot manage system users |
| Cluster Management | 6 | Critical | Multi-cluster setup impossible |
| Service Configuration | 6 | Critical | Core proxying feature unavailable |
| Proxy Monitoring | 5 | Critical | No visibility into proxy fleet |
| Certificate Management | 8 | High | Manual cert operations required |
| Auto-Scaling | 6 | Medium | Cannot configure auto-scaling |
| Enterprise Features (disabled) | 21 | Medium | Enterprise capabilities unavailable |
| Configuration Management | 3 | Low | Diagnostic access limited |

---

## Recommendations

### Immediate Actions (Week 1-2)
1. Prioritize WebUI for user/cluster/service/proxy management
2. These are blocking features for any meaningful system use
3. Start with admin dashboard as central hub

### Short-term Actions (Week 3-4)
1. Complete certificate management UI
2. Finalize module/deployment/scaling UIs
3. Add comprehensive monitoring dashboard

### Medium-term Actions (Week 5-6)
1. Enable enterprise feature components
2. Add zero-trust and traffic-shaping UIs
3. Implement advanced observability views

### Quality Assurance
- Ensure all API endpoints have corresponding error handling in UI
- Add input validation on all forms
- Implement proper loading and error states
- Add confirmation dialogs for destructive operations
- Include audit logging UI for compliance

---

## File References

### API Routes Location
`/home/penguin/code/MarchProxy/api-server/app/api/v1/routes/`

- `auth.py` - 9 endpoints
- `users.py` - 7 endpoints
- `clusters.py` - 6 endpoints
- `proxies.py` - 7 endpoints
- `services.py` - 6 endpoints
- `certificates.py` - 8 endpoints
- `modules.py` - 8 endpoints
- `module_routes.py` - 7 endpoints
- `deployments.py` - 6 endpoints
- `scaling.py` - 6 endpoints
- `config.py` - 3 endpoints
- `observability.py` - 11 endpoints (Enterprise)
- `traffic_shaping.py` - 7 endpoints (Enterprise)
- `multi_cloud.py` - 10 endpoints (Enterprise)
- `zero_trust.py` - 14 endpoints (Enterprise)

### WebUI Components Location
`/home/penguin/code/MarchProxy/webui/src/components/`

**Existing Components:**
- `Common/LicenseGate.tsx` - License validation
- `Layout/Header.tsx`, `Sidebar.tsx`, `MainLayout.tsx`, `ProtectedRoute.tsx` - Navigation
- `Modules/ModuleCard.tsx`, `RouteEditor.tsx` - Module management (partial)
- `Deployments/TrafficSlider.tsx` - Deployment control (partial)
- `Observability/ServiceGraph.tsx` - Service visualization (partial)
- `Enterprise/CloudHealthMap.tsx` - Cloud health (partial)
- `Enterprise.disabled/*` - Disabled enterprise components

**Missing Components:**
- AdminPanel (users, clusters, configuration)
- ServiceManagement (services CRUD)
- Dashboard (proxy monitoring, metrics)
- CertificateManagement (certificate CRUD)
- ModuleManagement (scaling policies)
- Full Enterprise features (when enabled)

---

## Notes

- This audit was generated on 2025-12-16
- API endpoints were extracted from FastAPI route decorators (@router.*)
- WebUI components analyzed for functional coverage
- Enterprise features marked as licensed
- Some components exist in `Enterprise.disabled/` awaiting license activation
- Recommend using this matrix to prioritize frontend development
