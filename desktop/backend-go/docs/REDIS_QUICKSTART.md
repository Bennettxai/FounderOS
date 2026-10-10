# Redis Security Quick Start

## 5-Minute Setup

### Development (Local Docker)

1. **Copy environment file**:
```bash
cd desktop/backend-go
cp .env.example .env
```

2. **Set Redis password in .env**:
```env
REDIS_PASSWORD=changeme_insecure_dev_password
```

3. **Start Redis with Docker Compose**:
```bash
docker-compose up -d redis
```

4. **Verify connection**:
```bash
# Should return "PONG"
docker exec -it businessos-redis redis-cli -a changeme_insecure_dev_password ping
```

5. **Start application**:
```bash
go run cmd/server/main.go
```

✅ **Done!** Redis is now running with password authentication.

---

## Production (Cloud)

### Option 1: Google Cloud Memorystore

1. **Create Redis instance**:
```bash
gcloud redis instances create businessos-redis \
  --size=1 \
  --region=us-central1 \
  --redis-version=redis_7_0 \
  --auth-enabled \
  --transit-encryption-mode=SERVER_AUTHENTICATION
```

2. **Get credentials**:
```bash
# Get host
HOST=$(gcloud redis instances describe businessos-redis --region=us-central1 --format="get(host)")

# Get password
PASSWORD=$(gcloud redis instances get-auth-string businessos-redis --region=us-central1)

echo "REDIS_URL=rediss://${HOST}:6378/0"
echo "REDIS_PASSWORD=${PASSWORD}"
```

3. **Configure .env.production**:
```env
REDIS_URL=rediss://10.0.0.3:6378/0
REDIS_PASSWORD=your-memorystore-auth-string
REDIS_TLS_ENABLED=true

# Generate with: openssl rand -base64 48
REDIS_KEY_HMAC_SECRET=your-strong-random-hmac-secret
```

### Option 2: AWS ElastiCache

1. **Create cluster** (via AWS Console or CLI):
   - Enable "Encryption in-transit"
   - Enable "Encryption at-rest"
   - Set AUTH token (min 16 characters)

2. **Configure .env.production**:
```env
REDIS_URL=rediss://your-cluster.cache.amazonaws.com:6379/0
REDIS_PASSWORD=your-elasticache-auth-token
REDIS_TLS_ENABLED=true
REDIS_KEY_HMAC_SECRET=your-strong-random-hmac-secret
```

---

## Security Checklist

### Development
- [x] Set `REDIS_PASSWORD` (can be simple for dev)
- [x] `REDIS_TLS_ENABLED=false` (OK for local)
- [x] Set `REDIS_KEY_HMAC_SECRET` (can be simple for dev)

### Production
- [ ] Generate strong password: `openssl rand -base64 32`
- [ ] Set `REDIS_TLS_ENABLED=true`
- [ ] Use `rediss://` URL scheme (not `redis://`)
- [ ] Generate HMAC secret: `openssl rand -base64 48`
- [ ] Store secrets in secrets manager (GCP Secret Manager, AWS Secrets Manager)
- [ ] Restrict Redis network access (VPC/firewall)
- [ ] Enable Redis persistence (AOF + RDB)
- [ ] Set up monitoring and alerts

---

## Verification

### Test Redis Connection
```bash
# Development
docker exec -it businessos-redis redis-cli -a changeme_insecure_dev_password ping

# Production (from app server)
redis-cli -h your-redis-host -p 6379 --tls -a your-password ping
```

### Test Application
```bash
# Health check should show Redis connected
curl http://localhost:8001/health

# Expected response includes:
# "redis": {"connected": true, "latency_ms": 2}
```

### Check Logs
```bash
# Docker logs
docker logs businessos-redis

# Application logs should show:
# "Redis: Password authentication enabled"
# "Redis connected: redis://***:***@*** (pool_size=50, protocol=plain)"
```

---

## Troubleshooting

### Common Issues

**"NOAUTH Authentication required"**
- Solution: Set `REDIS_PASSWORD` in .env to match docker-compose.yml

**"Connection refused"**
- Solution: Start Redis with `docker-compose up -d redis`

**"x509: certificate signed by unknown authority"** (TLS)
- Solution: For dev, set `REDIS_TLS_ENABLED=false`. For prod, verify certificates.

**Sessions not cached**
- Solution: Check Redis is running and password is correct

---

## Next Steps

- Review [REDIS_SECURITY.md](./REDIS_SECURITY.md) for detailed configuration
- Set up monitoring and alerting
- Configure automated backups
- Implement password rotation (every 90 days)
