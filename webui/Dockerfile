# MarchProxy WebUI Dockerfile
# Multi-stage build for React + TypeScript + Vite application

FROM node:20-alpine as builder

WORKDIR /build

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production=false

# Copy source code
COPY . .

# Build application
RUN npm run build

# Production stage
FROM node:20-alpine

LABEL maintainer="MarchProxy Contributors"
LABEL description="MarchProxy WebUI - React Frontend"

WORKDIR /app

# Install production server dependencies
RUN npm install -g serve

# Copy built application from builder
COPY --from=builder /build/dist ./dist

# Create non-root user
RUN adduser -D -u 1000 marchproxy && \
    chown -R marchproxy:marchproxy /app

USER marchproxy

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

# Serve application
CMD ["serve", "-s", "dist", "-l", "3000"]
