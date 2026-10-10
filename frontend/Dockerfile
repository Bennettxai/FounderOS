# syntax=docker/dockerfile:1

FROM node:22-alpine AS base
WORKDIR /app/frontend

ENV PNPM_HOME=/pnpm
ENV PATH=$PNPM_HOME:$PATH
RUN corepack enable && corepack prepare pnpm@9.15.4 --activate

# The frontend depends on local OptimalEngine SDK packages via file:../optimal-engine.
# Build from the repository root context so those SDKs are available to pnpm.
COPY frontend/package.json frontend/pnpm-lock.yaml ./
COPY optimal-engine/sdks /app/optimal-engine/sdks

WORKDIR /app/optimal-engine/sdks/typescript
RUN npm ci && npm run build

WORKDIR /app/optimal-engine/sdks/miosa
RUN npm ci && npm run build

WORKDIR /app/frontend

FROM base AS builder
ENV NODE_OPTIONS=--max-old-space-size=4096
RUN pnpm install --frozen-lockfile
COPY frontend/ ./

ARG PUBLIC_API_URL
ARG PUBLIC_ENVIRONMENT=production
ENV PUBLIC_API_URL=$PUBLIC_API_URL
ENV PUBLIC_ENVIRONMENT=$PUBLIC_ENVIRONMENT

RUN pnpm run build

FROM base AS deps
RUN pnpm install --prod --frozen-lockfile

FROM node:22-alpine
WORKDIR /app/frontend

COPY --from=deps /app/frontend/package.json ./package.json
COPY --from=deps /app/frontend/node_modules ./node_modules
COPY --from=deps /app/optimal-engine /app/optimal-engine
COPY --from=builder /app/frontend/build ./build

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 sveltekit
USER sveltekit

ENV NODE_ENV=production
ENV PORT=3000
# ORIGIN must be set at deploy time to match the actual domain (e.g., https://app.example.com).
# This default is for local/container use only; override it in production.
ENV ORIGIN=http://localhost:3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:3000/ || exit 1

EXPOSE 3000

CMD ["node", "build"]
