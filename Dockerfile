# FounderOS (on GladOS) - Dockerfile for Coolify deployment
# Multi-stage build for smaller production image

# =============================================================================
# Base stage - install dependencies
# =============================================================================
FROM node:22-alpine AS base

# Install system dependencies needed for better-sqlite3 and other native modules
RUN apk add --no-cache \
    python3 \
    make \
    g++ \
    sqlite \
    sqlite-dev \
    ffmpeg \
    vips-dev \
    git \
    curl \
    wget

WORKDIR /app

# Copy package files
COPY package*.json ./

# =============================================================================
# Dependencies stage - install npm dependencies
# =============================================================================
FROM base AS deps

# Install all dependencies (including devDependencies for build)
RUN npm ci --prefer-offline --no-audit --progress=false

# =============================================================================
# Builder stage - build the Next.js application
# =============================================================================
FROM deps AS builder

# Copy source code
COPY . .

# Build the application
# NEXT_DIST_DIR is set to .next by default, but we can override
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# =============================================================================
# Runner stage - production image
# =============================================================================
FROM node:22-alpine AS runner

# Install runtime dependencies
RUN apk add --no-cache \
    sqlite \
    sqlite-dev \
    ffmpeg \
    vips \
    curl \
    wget \
    tzdata

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nextjs -u 1001 -G nodejs

WORKDIR /app

# Set production environment
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Copy built application from builder
COPY --from=builder --chown=nextjs:nodejs /app/.next ./.next
COPY --from=builder --chown=nextjs:nodejs /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/node_modules ./node_modules
COPY --from=builder --chown=nextjs:nodejs /app/package.json ./package.json
COPY --from=builder --chown=nextjs:nodejs /app/scripts ./scripts
COPY --from=builder --chown=nextjs:nodejs /app/lib ./lib
COPY --from=builder --chown=nextjs:nodejs /app/middleware.ts ./middleware.ts
COPY --from=builder --chown=nextjs:nodejs /app/next.config.mjs ./next.config.mjs
COPY --from=builder --chown=nextjs:nodejs /app/tsconfig.json ./tsconfig.json

# Create data directories with correct permissions
RUN mkdir -p /data /data/brain-store /data/creative-engine /data/Automation && \
    chown -R nextjs:nodejs /data

# Switch to non-root user
USER nextjs

# Expose port
EXPOSE 4100

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget -q --spider http://localhost:4100/api/auth/check || exit 1

# Start the application using the serve script
CMD ["node", "scripts/serve.mjs", "start"]