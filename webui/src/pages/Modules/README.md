# Module Management Pages

This directory contains the WebUI pages for managing MarchProxy's Unified NLB Architecture modules.

## Pages

### ModuleManager.tsx
Main dashboard for viewing and managing all modules (NLB, ALB, DBLB, AILB, RTMP).

**Features:**
- Grid view of module cards with status, health, and metrics
- Enable/disable module controls
- Tab-based filtering by module type
- Real-time metrics polling (10s interval)
- Quick navigation to route configuration

**Route:** `/modules`

**License:** Requires Enterprise feature `unified_nlb`

### ModuleRoutes.tsx
Configure routes for a specific module.

**Features:**
- Route table with CRUD operations
- Protocol-specific route editor
- Rate limiting configuration
- Priority settings (P0-P3)
- Backend URL and port configuration

**Route:** `/modules/:moduleId/routes`

**License:** Requires Enterprise feature `unified_nlb`

## Components Used

- `ModuleCard` - Visual card for module status display
- `RouteEditor` - Dialog form for route configuration
- `LicenseGate` - Enterprise feature protection

## API Integration

These pages use the `modulesApi` service which expects the following backend endpoints:

- `GET /api/v1/modules` - List all modules
- `POST /api/v1/modules/{id}/enable` - Enable module
- `POST /api/v1/modules/{id}/disable` - Disable module
- `GET /api/v1/modules/{id}/routes` - List routes
- `POST /api/v1/modules/{id}/routes` - Create route
- `PUT /api/v1/modules/{id}/routes/{route_id}` - Update route
- `DELETE /api/v1/modules/{id}/routes/{route_id}` - Delete route

## Module Types

| Type | Name | Description |
|------|------|-------------|
| NLB | Network Load Balancer | L3/L4 load balancer with XDP/eBPF |
| ALB | Application Load Balancer | L7 HTTP/HTTPS proxy with Envoy |
| DBLB | Database Load Balancer | Database proxy with ArticDBM |
| AILB | AI/LLM Load Balancer | AI inference proxy with WaddleAI |
| RTMP | Video Transcoder | Video streaming with FFmpeg |

## Usage

```typescript
import ModuleManager from './pages/Modules/ModuleManager';
import ModuleRoutes from './pages/Modules/ModuleRoutes';

// In App.tsx
<Route path="/modules" element={<ModuleManager />} />
<Route path="/modules/:moduleId/routes" element={<ModuleRoutes />} />
```

## Development

To test these pages during development:

1. Ensure backend API is running
2. Configure API endpoint in `.env`
3. Start WebUI dev server: `npm run dev`
4. Navigate to `/modules`

## Related Pages

- `/scaling/auto-scaling` - Auto-scaling policy management
- `/deployments/blue-green` - Blue/green deployment control
