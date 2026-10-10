# Redis Security Configuration

## Overview

BusinessOS Go backend uses Redis for session storage and pub/sub messaging across horizontal scaling. This document covers the security configuration for Redis in both development and production environments.

## Security Features

### 1. Password Authentication (`requirepass`)
- **Purpose**: Prevents unauthorized access to Redis
- **Implementation**: Redis password set via `REDIS_PASSWORD` environment variable
- **Required**: YES (both dev and production)

### 2. TLS Encryption (`rediss://`)
- **Purpose**: Encrypts data in transit between application and Redis
- **Implementation**: Use `rediss://` URL scheme and set `REDIS_TLS_ENABLED=true`
- **Required**: NO for local dev, YES for production cloud deployments

### 3. HMAC Key Derivation
- **Purpose**: Prevents token enumeration attacks by hashing session tokens before storing as Redis keys
- **Implementation**: Set `REDIS_KEY_HMAC_SECRET` to a strong random value (min 32 bytes)
- **Required**: YES in production

## Development Setup

### Using Docker Compose (Recommended)

1. **Set Redis password in environment**:
```bash
# In your shell or .env file
export REDIS_PASSWORD=changeme_insecure_dev_password
```

2. **Start Redis**:
```bash
cd desktop/backend-go
docker-compose up -d redis
```

3. **Configure application (.env)**:
```env
REDIS_URL=redis://localhost:6379/0
REDIS_PASSWORD=changeme_insecure_dev_password
REDIS_TLS_ENABLED=false
REDIS_KEY_HMAC_SECRET=dev-hmac-secret-change-in-production
```

### Testing Connection

```bash
# Test Redis connection with password
docker exec -it businessos-redis redis-cli -a changeme_insecure_dev_password ping

# Should output: PONG
```

## Production Setup

### Google Cloud Memorystore (Redis)

1. **Create Memorystore instance with AUTH enabled**:
```bash
gcloud redis instances create businessos-redis \
  --size=1 \
  --region=us-central1 \
  --redis-version=redis_7_0 \
  --auth-enabled \
  --transit-encryption-mode=SERVER_AUTHENTICATION
```

2. **Get connection details**:
```bash
# Get Redis host
gcloud redis instances describe businessos-redis --region=us-central1 --format="get(host)"

# Get AUTH string (password)
gcloud redis instances get-auth-string businessos-redis --region=us-central1
```

3. **Configure application (.env.production)**:
```env
# Use rediss:// for TLS (GCP Memorystore supports TLS)
REDIS_URL=rediss://10.0.0.3:6378/0
REDIS_PASSWORD=your-memorystore-auth-string
REDIS_TLS_ENABLED=true

# CRITICAL: Generate strong HMAC secret
# openssl rand -base64 48
REDIS_KEY_HMAC_SECRET=your-strong-random-value-min-32-bytes
```

### AWS ElastiCache (Redis)

1. **Create ElastiCache cluster with encryption**:
```bash
aws elasticache create-replication-group \
  --replication-group-id businessos-redis \
  --replication-group-description "BusinessOS Redis" \
  --engine redis \
  --cache-node-type cache.t3.micro \
  --num-cache-clusters 1 \
  --auth-token your-strong-password-min-16-chars \
  --transit-encryption-enabled \
  --at-rest-encryption-enabled
```

2. **Configure application**:
```env
REDIS_URL=rediss://businessos-redis.abc123.cache.amazonaws.com:6379/0
REDIS_PASSWORD=your-strong-password-min-16-chars
REDIS_TLS_ENABLED=true
REDIS_KEY_HMAC_SECRET=your-strong-random-value-min-32-bytes
```

### Self-Hosted Production

1. **Generate strong passwords**:
```bash
# Redis password
openssl rand -base64 32

# HMAC secret
openssl rand -base64 48
```

2. **Generate TLS certificates** (if using self-signed):
```bash
mkdir -p certs/redis
cd certs/redis

# Generate CA key and certificate
openssl genrsa -out ca-key.pem 4096
openssl req -new -x509 -days 3650 -key ca-key.pem -out ca-cert.pem

# Generate Redis server key and certificate
openssl genrsa -out redis-server-key.pem 4096
openssl req -new -key redis-server-key.pem -out redis-server.csr
openssl x509 -req -days 3650 -in redis-server.csr -CA ca-cert.pem \
  -CAkey ca-key.pem -set_serial 01 -out redis-server-cert.pem

# Set permissions
chmod 600 redis-server-key.pem ca-key.pem
chmod 644 redis-server-cert.pem ca-cert.pem
```

3. **Update docker-compose.yml** for TLS:
```yaml
redis:
  image: redis:7-alpine
  environment:
    - REDIS_PASSWORD=${REDIS_PASSWORD}
  command: >
    sh -c '
    redis-server
    --requirepass "$${REDIS_PASSWORD}"
    --tls-port 6379
    --port 0
    --tls-cert-file /etc/redis/certs/redis-server-cert.pem
    --tls-key-file /etc/redis/certs/redis-server-key.pem
    --tls-ca-cert-file /etc/redis/certs/ca-cert.pem
    --tls-auth-clients no
    '
  volumes:
    - ./certs/redis:/etc/redis/certs:ro
    - redis-data:/data
```

4. **Configure application**:
```env
REDIS_URL=rediss://localhost:6379/0
REDIS_PASSWORD=your-generated-strong-password
REDIS_TLS_ENABLED=true
REDIS_KEY_HMAC_SECRET=your-generated-hmac-secret
```

## Security Best Practices

### 1. Strong Passwords
- **Minimum**: 16 characters for development, 32 for production
- **Generation**: Use `openssl rand -base64 32` or equivalent
- **Storage**: Use secrets management (GCP Secret Manager, AWS Secrets Manager, HashiCorp Vault)
- **Rotation**: Rotate passwords every 90 days in production

### 2. TLS Configuration
- **Development**: Optional (can use `REDIS_TLS_ENABLED=false` for simplicity)
- **Production**: MANDATORY (`REDIS_TLS_ENABLED=true`)
- **Certificate Validation**: Always validate certificates in production
- **Minimum TLS Version**: TLS 1.2 (enforced in code)

### 3. Network Security
- **Firewall**: Restrict Redis port (6379) to application servers only
- **VPC**: Use private networking (no public IP for Redis)
- **Cloud**: Use cloud provider's managed Redis with VPC peering

### 4. HMAC Secret
- **Purpose**: Prevents token enumeration by hashing session tokens
- **Length**: Minimum 32 bytes (48 bytes recommended)
- **Generation**: `openssl rand -base64 48`
- **Critical**: MUST be set in production

### 5. Connection Pooling
- **Default Pool Size**: 50 connections
- **Min Idle**: 10 connections
- **Timeouts**: Read/Write timeout of 3 seconds
- **Tune**: Adjust based on your application's load

## Monitoring & Alerts

### Health Check Endpoint
```bash
curl http://localhost:8001/health
```

Response includes Redis health:
```json
{
  "status": "healthy",
  "redis": {
    "connected": true,
    "latency_ms": 2,
    "pool_stats": {
      "total_conns": 10,
      "idle_conns": 8
    }
  }
}
```

### Metrics to Monitor
- **Connection failures**: Alert if Redis unavailable
- **Latency**: Alert if ping latency > 100ms
- **Pool exhaustion**: Alert if `idle_conns` = 0 frequently
- **Memory usage**: Monitor Redis memory consumption

## Troubleshooting

### Connection Refused
```text
Error: failed to ping Redis: dial tcp [::1]:6379: connect: connection refused
```
**Solutions**:
1. Check Redis is running: `docker ps | grep redis`
2. Verify port mapping: `docker-compose ps`
3. Check firewall rules

### Authentication Failed
```text
Error: failed to ping Redis: NOAUTH Authentication required
```
**Solutions**:
1. Verify `REDIS_PASSWORD` is set correctly
2. Check password in docker-compose.yml matches application config
3. Test with redis-cli: `redis-cli -a your-password ping`

### TLS Handshake Failed
```text
Error: failed to ping Redis: x509: certificate signed by unknown authority
```
**Solutions**:
1. For development: Ensure `REDIS_TLS_ENABLED=false` OR set proper certs
2. For production: Verify certificate chain is correct
3. Check TLS version compatibility (min TLS 1.2)

### Session Cache Not Working
```bash
Warning: Redis unavailable
Sessions will use direct DB auth (not optimal for horizontal scaling)
```
**Impact**: Application still works but uses database for every auth check (slower)

**Solutions**:
1. Fix Redis connection (see above)
2. Verify `REDIS_URL` is set in environment
3. Check Redis logs: `docker logs businessos-redis`

## Migration Checklist

### From No Auth to Password Auth
- [ ] Generate strong password: `openssl rand -base64 32`
- [ ] Set `REDIS_PASSWORD` in environment
- [ ] Update docker-compose.yml with `--requirepass`
- [ ] Restart Redis: `docker-compose restart redis`
- [ ] Update application config with password
- [ ] Test connection: `redis-cli -a password ping`

### From Plain to TLS
- [ ] Generate or obtain TLS certificates
- [ ] Update docker-compose.yml with TLS config
- [ ] Change URL scheme from `redis://` to `rediss://`
- [ ] Set `REDIS_TLS_ENABLED=true`
- [ ] Restart Redis with TLS enabled
- [ ] Test connection with TLS client

## Security Compliance

### PCI DSS
- ✅ Encryption in transit (TLS)
- ✅ Strong authentication (password + HMAC)
- ✅ Access logging (Redis logs all commands)
- ✅ Network segmentation (VPC/firewall)

### HIPAA
- ✅ Encryption at rest (Redis AOF/RDB)
- ✅ Encryption in transit (TLS)
- ✅ Access controls (password auth)
- ✅ Audit logging (Redis command logging)

### SOC 2
- ✅ Secure configuration management
- ✅ Secrets management integration
- ✅ Monitoring and alerting
- ✅ Password rotation procedures

## References

- [Redis Security Documentation](https://redis.io/docs/manual/security/)
- [Redis TLS Configuration](https://redis.io/docs/manual/security/encryption/)
- [Go Redis Client](https://github.com/redis/go-redis)
- [OWASP Session Management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
