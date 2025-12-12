# WebUI Implementation Guide

## Phase 1: Foundation Setup (Current)

This document describes the complete WebUI implementation for MarchProxy v1.0.0.

### Directory Structure

```
webui/
├── src/
│   ├── App.tsx                 # Main application component
│   ├── main.tsx               # React entry point
│   ├── components/            # Reusable components
│   │   ├── Layout/
│   │   │   ├── Header.tsx
│   │   │   ├── Sidebar.tsx
│   │   │   └── Footer.tsx
│   │   └── Auth/
│   │       └── ProtectedRoute.tsx
│   ├── pages/                 # Page components
│   │   ├── Login.tsx
│   │   ├── Dashboard.tsx
│   │   ├── Clusters.tsx
│   │   └── Services.tsx
│   ├── services/              # API client services
│   │   ├── api.ts            # Axios instance
│   │   └── auth.ts           # Auth service
│   ├── hooks/                 # Custom React hooks
│   │   └── useAuth.ts
│   ├── store/                 # Zustand state management
│   │   └── authStore.ts
│   └── utils/                 # Utilities
│       └── theme.ts          # MUI theme (Dark Grey/Navy/Gold)
├── public/
│   └── index.html
├── package.json
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts
├── Dockerfile
└── server.js                  # Production Express server
```

### Configuration Files

#### vite.config.ts
See reference in .PLAN-fresh for Vite configuration with React plugin and proxy setup.

#### tsconfig.json
Standard React + TypeScript configuration with strict mode and path aliases.

#### package.json
Dependencies:
- React 18.2
- TypeScript 5.3
- Vite 5.1
- Material-UI 5.15
- React Router DOM 6.22
- React Query 3.39
- Zustand 4.5
- Axios 1.6

### Theme Specification

Colors (from .PLAN-fresh):
- Background: Dark Grey (#1E1E1E, #2C2C2C)
- Primary: Navy Blue (#1E3A8A, #0F172A)
- Accent: Gold (#FFD700, #FDB813)

### Core Components

#### App.tsx
- React Router setup
- Protected routes
- Theme provider (MUI)
- Query client provider (React Query)
- Layout wrapper

#### pages/Login.tsx
- Username/password form
- 2FA code input (conditional)
- Error handling
- Redirect on success
- Integration with auth service

#### services/api.ts
- Axios instance with base URL
- Request/response interceptors
- JWT token injection
- Error handling

#### services/auth.ts
- login(username, password, totpCode?)
- register(credentials)
- logout()
- refreshToken()
- getCurrentUser()

#### store/authStore.ts
- User state
- Token management
- Login/logout actions
- Persistence to localStorage

### Dockerfile

Multi-stage build:
1. Builder: npm install + vite build
2. Production: Node.js 20 + Express server
3. Serve static files from /dist
4. Health check on port 3000

### Express Server (server.js)

Simple static file server:
- Serve from dist/
- SPA fallback to index.html
- Health check endpoint /health
- Port 3000

## Implementation Status

✅ Directory structure created
✅ package.json created
⏳ Configuration files (vite.config.ts, tsconfig.json) - PENDING
⏳ Source files (App.tsx, main.tsx, etc.) - PENDING
⏳ Dockerfile - PENDING
⏳ Express server - PENDING

## Next Steps

1. Create vite.config.ts with React plugin
2. Create tsconfig.json with strict TypeScript config
3. Create src/main.tsx (React entry point)
4. Create src/App.tsx with routing
5. Create src/utils/theme.ts with MUI theme
6. Create src/pages/Login.tsx
7. Create src/services/api.ts and auth.ts
8. Create Dockerfile for production build
9. Create server.js for production serving
10. Test build and deployment

## Docker Integration

Add to docker-compose.yml:
```yaml
webui:
  build:
    context: ./webui
    dockerfile: Dockerfile
  container_name: marchproxy-webui
  environment:
    - VITE_API_URL=http://api-server:8000
  ports:
    - "3000:3000"
  networks:
    - marchproxy-network
  depends_on:
    - api-server
```
