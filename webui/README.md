# MarchProxy WebUI

Modern React + TypeScript web interface for MarchProxy enterprise egress proxy management.

## Features

- **Modern Stack**: React 18, TypeScript, Vite, Material-UI
- **Dark Theme**: Custom dark grey/navy blue/gold color scheme
- **Authentication**: JWT-based auth with 2FA support
- **Responsive Design**: Mobile-first approach
- **Type Safety**: Full TypeScript coverage
- **State Management**: Zustand for global state
- **Routing**: React Router v6 for navigation

## Quick Start

### Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Access at http://localhost:3000
```

### Production Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

### Docker Deployment

```bash
# Build Docker image
docker build -t marchproxy-webui:latest .

# Run container
docker run -p 3000:3000 marchproxy-webui:latest
```

## Environment Configuration

Create a `.env` file based on `.env.example`:

```bash
VITE_API_URL=http://localhost:8000
VITE_WS_URL=ws://localhost:8000/ws
VITE_LICENSE_SERVER_URL=https://license.penguintech.io
VITE_PRODUCT_NAME=marchproxy
```

## Project Structure

```
webui/
├── src/
│   ├── components/        # React components
│   │   ├── Layout/       # Layout components (Header, Sidebar, etc.)
│   │   └── Common/       # Shared components
│   ├── pages/            # Page components
│   ├── services/         # API services
│   │   ├── api.ts       # Axios client
│   │   ├── auth.ts      # Auth service
│   │   └── types.ts     # TypeScript types
│   ├── store/           # Zustand stores
│   ├── utils/           # Utilities
│   │   └── theme.ts     # MUI theme config
│   ├── App.tsx          # Main app component
│   └── main.tsx         # Entry point
├── public/              # Static assets
├── Dockerfile          # Production container
├── vite.config.ts      # Vite configuration
├── tsconfig.json       # TypeScript configuration
└── package.json        # Dependencies
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run type-check` - TypeScript type checking

## Theme

The application uses a custom dark theme with:

- **Background**: Dark Grey (#1E1E1E, #2C2C2C)
- **Primary**: Navy Blue (#1E3A8A, #0F172A)
- **Accent**: Gold (#FFD700, #FDB813)

## Authentication

The WebUI supports:

- Username/password login
- Two-factor authentication (2FA)
- JWT token management
- Automatic token refresh
- Protected routes

## API Integration

All API calls go through the centralized `apiClient` in `src/services/api.ts`:

- Automatic JWT token injection
- Request/response interceptors
- Error handling
- License validation

## License

AGPL-3.0-or-later

## Support

For issues and support, visit [PenguinTech](https://www.penguintech.io)
