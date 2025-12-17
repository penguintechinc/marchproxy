# Release Notes - MarchProxy WebUI

## Latest Release

---

## v1.0.0 - 2025-12-16

### Major Milestone
Initial stable release of MarchProxy Web UI with core functionality for managing proxies, services, clusters, and certificates.

### New Features

#### Dashboard & Monitoring
- Real-time dashboard with summary statistics (clusters, services, proxies, active connections)
- Service status monitoring with health indicators
- Cluster overview and health status
- Proxy fleet monitoring with per-proxy metrics

#### User Management (Partial)
- User login/logout authentication
- User profile management
- 2FA setup and management
- Password change functionality
- Session management with configurable timeout

#### Cluster Management (Partial)
- View cluster list and details
- Create new clusters
- Update cluster configuration
- API key rotation
- Support for both Community and Enterprise tiers

#### Service Management (Partial)
- View all services
- Service creation wizard
- Service configuration management
- Port configuration (single, range, comma-separated)
- Protocol selection (TCP, UDP, ICMP)
- Authentication token management
- Service status controls (enable/disable)

#### Proxy Management (Partial)
- Proxy fleet overview
- Per-proxy status and health indicators
- Real-time metrics (CPU, memory, connections)
- Proxy heartbeat monitoring
- Historical metrics visualization

#### Certificate Management (Partial)
- Certificate list and search
- Certificate details view
- Expiration tracking with visual alerts
- Certificate upload/import UI
- Renewal scheduling
- Integration support for Infisical and HashiCorp Vault

#### Module Management (Partial)
- Module listing and overview
- Module health monitoring
- Route editor for custom routing rules
- Deployment management with blue-green support
- Traffic weight management for canary deployments

#### Advanced Features (Partial)
- Service graph visualization
- Multi-cloud health mapping
- Module auto-scaling configuration (disabled)
- Deployment orchestration (blue-green, canary)
- NUMA configuration (enterprise)
- Cost analytics (enterprise)

#### Observability
- Metrics dashboard with real-time updates
- Distributed tracing interface
- Alert configuration and management
- Activity feed on dashboard
- Audit log viewer

#### Security
- mTLS configuration management
- Zero-trust security policy editor (enterprise)
- OPA policy validation and testing
- Compliance report generation
- Audit log search and export
- Access control enforcement

### Technology Stack
- **Frontend**: React 18.2, TypeScript 5.3
- **UI Framework**: Material-UI (MUI) 5.15
- **State Management**: Zustand 4.5
- **Data Fetching**: React Query 3.39, Axios 1.6
- **Forms**: React Hook Form 7.68
- **Routing**: React Router DOM 6.22
- **Charts**: Recharts 3.5
- **Build Tool**: Vite 5.1
- **Testing**: Playwright 1.40
- **Styling**: Emotion

### API Coverage
- Authentication: 9/9 endpoints (100%)
- User Management: 2/7 endpoints (29%)
- Cluster Management: 4/6 endpoints (67%)
- Service Management: 4/6 endpoints (67%)
- Proxy Management: 3/7 endpoints (43%)
- Certificate Management: 3/8 endpoints (38%)
- Module Management: 6/8 endpoints (75%)
- Observability: 6/11 endpoints (55%)
- Security: 7/8 endpoints (88%)
- License: 4/4 endpoints (100%)

### Known Limitations
- User/cluster/service deletion UI not implemented
- Advanced module editing features pending
- Enterprise Zero-Trust UI disabled pending license validation
- Traffic Shaping UI disabled pending enterprise tier
- Bulk operations not implemented
- WebSocket real-time logs not connected
- SAML/SCIM authentication not exposed in UI
- Configuration management dashboard missing
- Scaling policy CRUD incomplete

### Components Implemented
- `Layout/MainLayout.tsx` - Application shell
- `Layout/Header.tsx` - Top navigation
- `Layout/Sidebar.tsx` - Sidebar navigation
- `Layout/ProtectedRoute.tsx` - Auth guard
- `Pages/Dashboard.tsx` - Main dashboard
- `Pages/Clusters.tsx` - Cluster management (partial)
- `Pages/Services.tsx` - Service management (partial)
- `Pages/Proxies.tsx` - Proxy monitoring
- `Pages/Certificates.tsx` - Certificate management (partial)
- `Pages/Modules/Manager.tsx` - Module overview
- `Pages/Modules/RouteEditor.tsx` - Route editing
- `Pages/Deployments/BlueGreen.tsx` - Deployment control
- `Pages/Scaling/AutoScaling.tsx` - Scaling configuration
- `Pages/Observability/Metrics.tsx` - Metrics dashboard
- `Pages/Observability/Tracing.tsx` - Tracing interface
- `Pages/Security/MTLSConfig.tsx` - mTLS management
- `Pages/Security/PolicyEditor.tsx` - OPA policy editor
- `Pages/Enterprise/*` - Enterprise features (partial)
- `Common/LicenseGate.tsx` - License validation
- Plus 20+ additional component modules

### Components Missing/Pending
- `AdminPanel/UserManagement.tsx` - User CRUD
- `AdminPanel/ClusterManagement.tsx` - Full cluster management
- `AdminPanel/ConfigurationDashboard.tsx` - System configuration
- `ServiceManagement/ServiceCRUD.tsx` - Full service CRUD
- `CertificateManagement/CertificateCRUD.tsx` - Full certificate CRUD
- `Dashboard/ProxyMonitoring.tsx` - Comprehensive proxy dashboard
- `ModuleManagement/ScalingPolicy.tsx` - Scaling policy UI
- Login/Registration pages
- Account settings pages
- More enterprise feature UIs

### Breaking Changes
None - first release

### Bug Fixes
None - initial release

### Performance Improvements
- Code splitting for faster initial load
- Lazy loading of route components
- React Query caching for API responses
- Memoized components to prevent unnecessary re-renders

### Dependencies Upgraded
- All dependencies pinned to latest stable versions
- Node 20+ required
- npm 9+ required

### Documentation
- API.md - Frontend API service layer documentation
- TESTING.md - Playwright E2E testing guide
- CONFIGURATION.md - Environment and application configuration
- USAGE.md - User guide and feature documentation
- RELEASE_NOTES.md - This file

### Testing Coverage
- Integration tests for page load across 12+ pages
- Playwright test suite with ~40 test cases
- Component rendering tests
- API client mocking setup
- Test coverage: ~50% of application workflows

### Deployment
- Multi-container deployment with Docker Compose
- Kubernetes manifests available
- Environment-based configuration (dev, staging, prod)
- Health check endpoints implemented
- Graceful shutdown support

### Known Issues
1. **API Coverage Gap**: 60% of backend APIs lack UI
   - Impacts: User/cluster deletion, bulk operations, advanced configuration
   - Workaround: Use API directly or backend admin tools
   - Target Fix: v1.1.0

2. **Enterprise Features Disabled**: Zero-Trust and Traffic Shaping UI disabled
   - Impacts: Enterprise customers unable to use features through UI
   - Workaround: API integration with license server
   - Target Fix: v1.1.0

3. **Real-time Updates**: WebSocket connections for logs not implemented
   - Impacts: Manual refresh required for log viewing
   - Workaround: Refresh page or use API polling
   - Target Fix: v1.2.0

### Migration Guide
First release - no migration needed.

### Upgrade Instructions
Standard installation and startup.

### Next Steps (v1.1.0)
- Complete remaining API coverage (user/cluster/service CRUD)
- Enable enterprise feature UIs
- Implement bulk operations
- Add configuration management dashboard
- Complete WebSocket real-time features
- SAML/SCIM UI integration
- Advanced module editor

### Contributors
- MarchProxy Development Team
- Community contributors

### Support
- Documentation: `/webui/docs/`
- Issues: GitHub Issues (pending)
- Discussions: GitHub Discussions (pending)
- Email: support@penguintech.io

---

## v0.9.0-beta - 2025-12-10

### Beta Release
Initial beta release with core features for testing.

### Features
- Basic dashboard
- Service and cluster views
- Certificate management UI
- Module overview
- Security page stubs

### Known Limitations
- Incomplete API coverage
- Enterprise features disabled
- Testing coverage partial
- Documentation in progress

---

## v0.5.0-alpha - 2025-12-01

### Alpha Release
Early alpha with basic React setup and MUI integration.

### Features
- React 18 + TypeScript setup
- Vite build configuration
- MUI component library
- Basic routing structure
- API client setup

---

## Version History Summary

| Version | Date | Type | Status | API Coverage |
|---------|------|------|--------|--------------|
| 1.0.0 | 2025-12-16 | Stable | Active | 52% |
| 0.9.0-beta | 2025-12-10 | Beta | Archive | 40% |
| 0.5.0-alpha | 2025-12-01 | Alpha | Archive | 15% |

---

## Changelog Format

Each release includes:
- **Major Milestone**: High-level overview
- **New Features**: User-facing additions
- **Technology Stack**: Framework and dependency versions
- **API Coverage**: Endpoints implemented/total
- **Known Limitations**: Features not yet implemented
- **Components**: Implemented React components
- **Breaking Changes**: Backwards-incompatible changes
- **Bug Fixes**: Issues resolved
- **Performance Improvements**: Optimizations made
- **Dependencies**: Upgrade notes
- **Documentation**: Updated docs
- **Testing**: Coverage metrics
- **Deployment**: Infrastructure notes
- **Known Issues**: Current bugs with workarounds
- **Migration Guide**: Upgrade instructions
- **Next Steps**: Planned features

---

## Versioning Policy

### Version Format
`vMajor.Minor.Patch-BuildType` (e.g., `v1.0.0`, `v1.1.0-beta`)

### Version Increments

**Major (X._._ )**
- Breaking API changes
- Significant UI overhaul
- Major new feature categories
- Removed features

**Minor (_.X._ )**
- New features
- API additions
- UI improvements
- Component additions

**Patch (_._.X)**
- Bug fixes
- Security patches
- Minor improvements
- Documentation updates

**Build Type**
- `-alpha`: Early development
- `-beta`: Feature complete, testing phase
- `-rc`: Release candidate
- (none): Stable release

### Release Schedule
- Major: 2-3 months
- Minor: 2-4 weeks
- Patch: As needed (weekly average)
- Beta: 1-2 weeks before release

---

## Support Lifecycle

| Version | Released | Supported Until | Security Only |
|---------|----------|-----------------|---------------|
| 1.0.x | 2025-12-16 | 2026-06-16 | 2026-12-16 |
| 0.9.x-beta | 2025-12-10 | 2025-12-30 | N/A |
| 0.5.x-alpha | 2025-12-01 | 2025-12-10 | N/A |

---

## Reporting Issues

Found a bug? Please report:
1. Version number
2. Steps to reproduce
3. Expected vs actual behavior
4. Environment (OS, browser, Node version)
5. Screenshots/logs if applicable

---

## Feature Request Process

To request a feature:
1. Check existing issues/discussions
2. Describe the use case
3. Explain why it's needed
4. Suggest implementation approach
5. Vote on proposed features

---

## Security Reports

Report security issues to: security@penguintech.io

Do not publicly disclose security vulnerabilities.

---

## License

MarchProxy WebUI is dual-licensed:
- **Community Edition**: Limited AGPL3 with preamble
- **Enterprise Edition**: Commercial license

See LICENSE file for details.
