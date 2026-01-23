# Configuration Guide

## Overview

MarchProxy WebUI is configured through environment variables, application settings, and feature flags. This document describes all configuration options and how to apply them.

## Environment Variables

### Backend API Connection

```bash
# API Server URL (default: http://localhost:8000)
REACT_APP_API_URL=http://localhost:8000

# API request timeout in milliseconds (default: 30000)
REACT_APP_API_TIMEOUT=30000

# API version for compatibility (default: v1)
REACT_APP_API_VERSION=v1
```

### Application Environment

```bash
# Deployment environment
REACT_APP_ENV=development|staging|production

# Application name displayed in UI
REACT_APP_APP_NAME="MarchProxy"

# Company homepage
REACT_APP_COMPANY_URL=https://www.penguintech.io
```

### Feature Flags

```bash
# Enable/disable enterprise features
REACT_APP_ENABLE_ENTERPRISE=false

# Enable/disable debugging and logging
REACT_APP_ENABLE_DEBUG_LOGGING=false

# Enable/disable analytics
REACT_APP_ENABLE_ANALYTICS=true

# Enable/disable sentry error reporting
REACT_APP_ENABLE_SENTRY=false
REACT_APP_SENTRY_DSN=https://...
```

### UI Customization

```bash
# Theme: light, dark, auto
REACT_APP_THEME=auto

# Primary color (hex or named)
REACT_APP_PRIMARY_COLOR=#1976d2

# Secondary color
REACT_APP_SECONDARY_COLOR=#dc004e

# Sidebar mode: expanded, collapsed, compact
REACT_APP_SIDEBAR_DEFAULT=expanded

# Chart refresh interval (milliseconds)
REACT_APP_CHART_REFRESH_INTERVAL=5000

# Session timeout (milliseconds, 0 = never)
REACT_APP_SESSION_TIMEOUT=3600000  # 1 hour
```

### Authentication & License

```bash
# License server URL
REACT_APP_LICENSE_SERVER_URL=https://license.penguintech.io

# Product name for licensing
REACT_APP_PRODUCT_NAME=marchproxy

# Enable/disable license enforcement
REACT_APP_ENFORCE_LICENSE=false
```

### Development

```bash
# Enable CORS proxy for development
REACT_APP_USE_PROXY=true

# Mock API responses (for testing without backend)
REACT_APP_MOCK_API=false

# Show debug information in console
REACT_APP_DEBUG=false

# Enable source maps in production
REACT_APP_GENERATE_SOURCE_MAP=false
```

## Environment Files

### Structure

```
webui/
├── .env                  # Default configuration
├── .env.local            # Local overrides (git-ignored)
├── .env.development      # Development environment
├── .env.staging          # Staging environment
├── .env.production       # Production environment
└── .env.production.local # Production local overrides
```

### Example Files

**`.env`** (checked in)
```bash
REACT_APP_API_URL=http://localhost:8000
REACT_APP_ENV=development
REACT_APP_ENABLE_ENTERPRISE=false
REACT_APP_ENABLE_DEBUG_LOGGING=true
REACT_APP_THEME=auto
REACT_APP_SESSION_TIMEOUT=3600000
```

**`.env.production`** (checked in)
```bash
REACT_APP_API_URL=https://api.example.com
REACT_APP_ENV=production
REACT_APP_ENABLE_ENTERPRISE=true
REACT_APP_ENABLE_DEBUG_LOGGING=false
REACT_APP_ENABLE_ANALYTICS=true
REACT_APP_ENFORCE_LICENSE=true
REACT_APP_GENERATE_SOURCE_MAP=false
```

**`.env.local`** (git-ignored)
```bash
# Local development overrides
REACT_APP_API_URL=http://localhost:8000
REACT_APP_ENABLE_ENTERPRISE=true
REACT_APP_MOCK_API=false
REACT_APP_DEBUG=true
```

## Vite Configuration

### `vite.config.ts`

```typescript
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],

  // Path aliases
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@services': path.resolve(__dirname, './src/services'),
      '@pages': path.resolve(__dirname, './src/pages'),
      '@types': path.resolve(__dirname, './src/types'),
    },
  },

  // Server configuration
  server: {
    port: 3000,
    host: '0.0.0.0',
    proxy: {
      // Proxy API calls to backend (development)
      '/api': {
        target: process.env.REACT_APP_API_URL || 'http://localhost:8000',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, '/api'),
      },
    },
  },

  // Build configuration
  build: {
    target: 'ES2020',
    minify: 'terser',
    sourcemap: process.env.REACT_APP_GENERATE_SOURCE_MAP === 'true',
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'react-router-dom'],
          mui: ['@mui/material', '@mui/icons-material'],
          charts: ['recharts'],
          forms: ['react-hook-form'],
          queries: ['react-query'],
        },
      },
    },
  },

  // Preview server
  preview: {
    port: 3000,
    host: '0.0.0.0',
  },
});
```

## Application Configuration

### Theme Configuration

MUI theme customization in `src/theme/index.ts`:

```typescript
import { createTheme } from '@mui/material/styles';

export const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: process.env.REACT_APP_PRIMARY_COLOR || '#1976d2',
    },
    secondary: {
      main: process.env.REACT_APP_SECONDARY_COLOR || '#dc004e',
    },
  },

  typography: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    h1: { fontSize: '2.5rem', fontWeight: 500 },
    h2: { fontSize: '2rem', fontWeight: 500 },
  },

  shape: {
    borderRadius: 4,
  },

  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
        },
      },
    },
  },
});
```

### Feature Flags

Runtime feature management in `src/config/features.ts`:

```typescript
export interface FeatureFlags {
  enterpriseEnabled: boolean;
  debugLogging: boolean;
  analyticsEnabled: boolean;
  licenseEnforcement: boolean;
  darkModeSupport: boolean;
  advancedSecurityFeatures: boolean;
  multiClusterSupport: boolean;
}

export const features: FeatureFlags = {
  enterpriseEnabled: process.env.REACT_APP_ENABLE_ENTERPRISE === 'true',
  debugLogging: process.env.REACT_APP_ENABLE_DEBUG_LOGGING === 'true',
  analyticsEnabled: process.env.REACT_APP_ENABLE_ANALYTICS === 'true',
  licenseEnforcement: process.env.REACT_APP_ENFORCE_LICENSE === 'true',
  darkModeSupport: true,
  advancedSecurityFeatures: true,
  multiClusterSupport: true,
};

// Usage in components
import { features } from '@/config/features';

if (features.enterpriseEnabled) {
  // Show enterprise features
}
```

### API Configuration

Centralized API client setup in `src/services/api.ts`:

```typescript
const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8000';
const API_TIMEOUT = parseInt(process.env.REACT_APP_API_TIMEOUT || '30000', 10);

export const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: API_TIMEOUT,
  headers: {
    'Content-Type': 'application/json',
    'X-API-Version': process.env.REACT_APP_API_VERSION || 'v1',
  },
});
```

### Logging Configuration

Debug logging setup in `src/utils/logger.ts`:

```typescript
const DEBUG = process.env.REACT_APP_ENABLE_DEBUG_LOGGING === 'true';

export const logger = {
  debug: (message: string, data?: any) => {
    if (DEBUG) {
      console.debug(`[DEBUG] ${message}`, data);
    }
  },

  info: (message: string, data?: any) => {
    console.info(`[INFO] ${message}`, data);
  },

  warn: (message: string, data?: any) => {
    console.warn(`[WARN] ${message}`, data);
  },

  error: (message: string, error?: any) => {
    console.error(`[ERROR] ${message}`, error);
  },
};
```

## Docker Configuration

### Build-time Environment

`Dockerfile`:

```dockerfile
ARG REACT_APP_API_URL=http://localhost:8000
ARG REACT_APP_ENV=development
ARG REACT_APP_ENABLE_ENTERPRISE=false

ENV REACT_APP_API_URL=$REACT_APP_API_URL
ENV REACT_APP_ENV=$REACT_APP_ENV
ENV REACT_APP_ENABLE_ENTERPRISE=$REACT_APP_ENABLE_ENTERPRISE

RUN npm run build

EXPOSE 3000
CMD ["npm", "run", "serve"]
```

Build with custom environment:

```bash
docker build \
  --build-arg REACT_APP_API_URL=https://api.example.com \
  --build-arg REACT_APP_ENV=production \
  --build-arg REACT_APP_ENABLE_ENTERPRISE=true \
  -t marchproxy-webui:latest .
```

### Runtime Environment

Docker Compose:

```yaml
services:
  webui:
    image: marchproxy-webui:latest
    ports:
      - "3000:3000"
    environment:
      REACT_APP_API_URL: http://manager:8000
      REACT_APP_ENV: development
      REACT_APP_ENABLE_ENTERPRISE: "false"
      REACT_APP_SESSION_TIMEOUT: "3600000"
    depends_on:
      - manager
```

## Deployment Configurations

### Development

```bash
# .env.development
REACT_APP_API_URL=http://localhost:8000
REACT_APP_ENV=development
REACT_APP_ENABLE_ENTERPRISE=false
REACT_APP_ENABLE_DEBUG_LOGGING=true
REACT_APP_DEBUG=true
REACT_APP_MOCK_API=false
```

Run: `npm run dev`

### Staging

```bash
# .env.staging
REACT_APP_API_URL=https://api-staging.example.com
REACT_APP_ENV=staging
REACT_APP_ENABLE_ENTERPRISE=true
REACT_APP_ENABLE_DEBUG_LOGGING=true
REACT_APP_ENFORCE_LICENSE=false
REACT_APP_ENABLE_ANALYTICS=true
```

Build: `npm run build`

### Production

```bash
# .env.production
REACT_APP_API_URL=https://api.example.com
REACT_APP_ENV=production
REACT_APP_ENABLE_ENTERPRISE=true
REACT_APP_ENABLE_DEBUG_LOGGING=false
REACT_APP_ENFORCE_LICENSE=true
REACT_APP_ENABLE_ANALYTICS=true
REACT_APP_GENERATE_SOURCE_MAP=false
REACT_APP_SESSION_TIMEOUT=3600000
```

Build: `NODE_ENV=production npm run build`

## Performance Tuning

### Vite Build Optimization

```typescript
// vite.config.ts
build: {
  rollupOptions: {
    output: {
      // Code splitting for better caching
      manualChunks: {
        vendor: ['react', 'react-dom'],
        mui: ['@mui/material', '@mui/x-data-grid'],
        charts: ['recharts'],
      },
    },
  },
  // Minify with terser for better compression
  minify: 'terser',
  terserOptions: {
    compress: {
      drop_console: process.env.NODE_ENV === 'production',
    },
  },
},
```

### API Caching Configuration

```typescript
// src/config/query-client.ts
import { QueryClient } from 'react-query';

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 60000,      // 1 minute
      cacheTime: 300000,     // 5 minutes
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 1,
    },
  },
});
```

### Chart Refresh Interval

Control real-time updates:

```bash
# Refresh every 5 seconds
REACT_APP_CHART_REFRESH_INTERVAL=5000

# Or disable auto-refresh (manual only)
REACT_APP_CHART_REFRESH_INTERVAL=0
```

## Security Configuration

### Content Security Policy

Configure CSP headers (server-side):

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self' 'unsafe-inline';
  style-src 'self' 'unsafe-inline';
  img-src 'self' data: https:;
  font-src 'self' data:;
  connect-src 'self' https://license.penguintech.io
```

### CORS Configuration

Backend should configure CORS:

```
Access-Control-Allow-Origin: https://example.com
Access-Control-Allow-Methods: GET, POST, PATCH, DELETE
Access-Control-Allow-Credentials: true
```

### Session Security

```bash
# Session timeout (30 minutes)
REACT_APP_SESSION_TIMEOUT=1800000

# Secure cookies (HTTPS only)
# Configure in backend API
```

## Monitoring & Observability

### Sentry Error Reporting

```bash
REACT_APP_ENABLE_SENTRY=true
REACT_APP_SENTRY_DSN=https://key@sentry.io/project-id
```

Configuration:

```typescript
// src/utils/sentry.ts
import * as Sentry from "@sentry/react";

if (process.env.REACT_APP_ENABLE_SENTRY === 'true') {
  Sentry.init({
    dsn: process.env.REACT_APP_SENTRY_DSN,
    environment: process.env.REACT_APP_ENV,
    tracesSampleRate: 0.1,
  });
}
```

### Analytics

```bash
# Google Analytics or similar
REACT_APP_ENABLE_ANALYTICS=true
REACT_APP_GA_ID=UA-XXXXXXXXX-X
```

## Troubleshooting

### Environment Variables Not Loading

Check load order:
1. `.env` (base defaults)
2. `.env.local` (local overrides)
3. `.env.development` / `.env.production` (environment-specific)
4. Process environment variables (highest priority)

Verify with:
```bash
npm run env  # Shows all loaded environment variables
```

### API Connection Issues

Verify configuration:
```bash
# Check API URL
echo $REACT_APP_API_URL

# Test connectivity
curl -v http://localhost:8000/api/health
```

### Build Fails with Environment Variables

Ensure variables are prefixed with `REACT_APP_`:

```bash
# Works
REACT_APP_API_URL=http://localhost:8000

# Won't work (missing REACT_APP_ prefix)
API_URL=http://localhost:8000
```

### Theme Not Applying

Verify theme configuration:
```typescript
// Check if theme is loaded
console.log(process.env.REACT_APP_PRIMARY_COLOR);

// Verify ThemeProvider wraps App
<ThemeProvider theme={theme}>
  <App />
</ThemeProvider>
```
