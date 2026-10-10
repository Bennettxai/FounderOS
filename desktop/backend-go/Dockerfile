# =============================================================================
# BusinessOS Backend - Multi-stage Dockerfile for Cloud Run
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build
# ---------------------------------------------------------------------------
FROM golang:1.26.4-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy only necessary source files
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s" \
    -o /server ./cmd/server

# ---------------------------------------------------------------------------
# Stage 2: Production
# ---------------------------------------------------------------------------
FROM alpine:3.19

# Install runtime dependencies (ca-certificates for HTTPS, tzdata for TZ,
# curl for Docker HEALTHCHECK)
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /server /app/server

# Own the workdir as non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Cloud Run injects PORT env var; default to 8080
ENV SERVER_PORT=8080

EXPOSE 8080

# Health check: Cloud Run uses its own HTTP health probes, but this is useful
# for local Docker and docker-compose testing
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -f http://localhost:${SERVER_PORT:-8080}/health || exit 1

CMD ["/app/server"]
