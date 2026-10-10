# Redis Key Security: HMAC-SHA256 Token Hashing

## Overview

Session tokens are now hashed using HMAC-SHA256 before being used as Redis keys. This prevents enumeration attacks where attackers could scan Redis to discover valid session tokens.

## Security Problem Addressed

### Vulnerability: Token Enumeration
**Before**: Raw session tokens were used directly as Redis keys:
```text
auth_session:abc123def456...
user_sessions:user-id-123
```
**Attack vector**:
- Attacker gains read access to Redis (misconfiguration, network exposure, compromised service)
- Uses `SCAN` or `KEYS` commands to enumerate all session tokens
- Steals valid session tokens without needing to crack passwords
- Can impersonate users by using discovered tokens

### Solution: HMAC-SHA256 Key Derivation
**After**: Session tokens are hashed before use:
```text
auth_session:3f8a7b2c1e9d4a5f6b8c9e2a3d4f5e6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2
user_sessions:7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8
```
**Protection**:
- Tokens are hashed using HMAC-SHA256 with a secret key
- Resulting hashes are one-way (irreversible)
- Only the application with the secret can generate valid hashes
- Even with Redis access, attackers cannot discover original tokens
- Prevents token enumeration and session hijacking

## Implementation Details

### HMAC-SHA256 Algorithm
```go
func hashToken(token string, secret []byte) string {
    h := hmac.New(sha256.New, secret)
    h.Write([]byte(token))
    return hex.EncodeToString(h.Sum(nil))
}
```

**Properties**:
- **Deterministic**: Same token always produces same hash (required for key lookup)
- **One-way**: Cannot reverse hash to get original token
- **Secret-dependent**: Requires application's HMAC secret to compute valid hashes
- **Collision-resistant**: SHA256 provides 256-bit output space (64 hex characters)

### Configuration

#### Environment Variable
```bash
# Production (REQUIRED)
REDIS_KEY_HMAC_SECRET="your-strong-random-secret-at-least-32-bytes"

# Generate a secure secret:
openssl rand -hex 32
# or
head -c 32 /dev/urandom | base64
```

#### Minimum Requirements
- **Length**: Minimum 32 bytes (256 bits) recommended
- **Randomness**: Use cryptographically secure random generator
- **Uniqueness**: Different per environment (dev/staging/production)
- **Persistence**: Must be same across all application instances

#### Development vs Production

**Development** (auto-generated secret):
```go
// WARNING: Auto-generated on startup
// Different on each restart - sessions invalidated
// Different per instance - breaks horizontal scaling
hmacSecret := generateRandomSecret(32)
```

**Production** (configured secret):
```bash
# Set in environment (.env file or secrets manager)
REDIS_KEY_HMAC_SECRET="e8f3a9b2c7d6e1f4a8b3c9d2e7f1a6b8c3d9e2f7a1b6c8d3e9f2a7b1c6d8e3f9"
```

### Updated Components

1. **Config** (`internal/config/config.go`)
   - Added `RedisKeyHMACSecret` field
   - Loads from `REDIS_KEY_HMAC_SECRET` environment variable

2. **SessionCache** (`internal/middleware/redis_auth.go`)
   - Added `hmacSecret` field to SessionCache struct
   - Implemented `hashToken()` method for HMAC-SHA256
   - Updated `sessionKey()` to hash tokens
   - Updated `userSessionsKey()` to hash user IDs

3. **SessionStore** (`internal/redis/session.go`)
   - Added `hmacSecret` field to SessionStore struct
   - Implemented `hashKey()` method for HMAC-SHA256
   - Updated `sessionKey()` to hash session IDs
   - Updated `userSessionsKey()` to hash user IDs

4. **Server Initialization** (`cmd/server/main.go`)
   - Passes HMAC secret to SessionCache on startup
   - Logs security warning if secret not configured

## Migration Strategy

### Backwards Compatibility
⚠️ **WARNING**: This change is NOT backwards compatible with existing sessions!

**Why**: Old Redis keys use raw tokens, new keys use hashed tokens. They don't match.

**Impact**:
- Existing sessions in Redis will become inaccessible
- Users will be logged out and need to re-authenticate
- Session data will expire naturally (TTL=15 minutes for auth cache)

### Migration Options

#### Option 1: Accept Session Invalidation (Recommended)
**Best for**: Most applications with short-lived sessions

1. Deploy the update during low-traffic period
2. Sessions expire within 15 minutes
3. Users re-authenticate naturally
4. No data loss (PostgreSQL still has session data)

```bash
# Deploy
git pull
go build -o server cmd/server/main.go
systemctl restart businessos-backend

# Monitor
journalctl -u businessos-backend -f | grep "Session cache enabled"
# Should see: "Session cache enabled (TTL=15m, HMAC-secured keys)"
```

#### Option 2: Gradual Migration (Complex)
**Best for**: High-traffic apps requiring zero downtime

1. Deploy with dual-key support (check both old and new keys)
2. Warm new cache from PostgreSQL
3. Remove old key support after migration period

**NOT IMPLEMENTED** - Adds complexity without significant benefit for BusinessOS use case.

#### Option 3: Pre-Migration Cache Flush (Clean Slate)
**Best for**: Ensuring clean state

```bash
# Before deploying update
redis-cli SCAN 0 MATCH "auth_session:*" | xargs redis-cli DEL
redis-cli SCAN 0 MATCH "user_sessions:*" | xargs redis-cli DEL

# Then deploy update
```

## Security Best Practices

### Secret Management

1. **Never commit secrets to version control**
   ```bash
   # Add to .gitignore
   echo ".env" >> .gitignore
   ```

2. **Use environment variables or secrets manager**
   ```bash
   # In production environment
   export REDIS_KEY_HMAC_SECRET="$(openssl rand -hex 32)"

   # Or use secrets manager (AWS Secrets Manager, HashiCorp Vault, etc.)
   export REDIS_KEY_HMAC_SECRET="$(aws secretsmanager get-secret-value --secret-id redis-hmac-secret --query SecretString --output text)"
   ```

3. **Rotate secrets periodically**
   - Recommend quarterly rotation
   - Coordinate with session invalidation window
   - Update all instances simultaneously

4. **Use different secrets per environment**
   ```bash
   # Development
   REDIS_KEY_HMAC_SECRET="dev-secret-not-for-production-use-123"

   # Staging
   REDIS_KEY_HMAC_SECRET="staging-$(openssl rand -hex 32)"

   # Production
   REDIS_KEY_HMAC_SECRET="$(vault read -field=value secret/prod/redis-hmac)"
   ```

### Monitoring

Monitor for security warnings in logs:
```bash
# Check for missing secret warning
journalctl -u businessos-backend | grep "auto-generated HMAC secret"

# Check for weak secret warning
journalctl -u businessos-backend | grep "shorter than recommended 32 bytes"

# Verify secure initialization
journalctl -u businessos-backend | grep "HMAC-secured keys"
```

### Production Checklist

- [ ] Generate strong random secret (min 32 bytes)
- [ ] Store secret in secure secrets manager
- [ ] Configure `REDIS_KEY_HMAC_SECRET` environment variable
- [ ] Verify secret is same across all instances
- [ ] Test session creation and retrieval
- [ ] Monitor for security warnings in logs
- [ ] Document secret rotation procedure
- [ ] Set up secret rotation schedule (quarterly)

## Testing

### Verify Hashed Keys
```bash
# Check Redis keys (should see hashed values)
redis-cli KEYS "auth_session:*"
# Expected: auth_session:3f8a7b2c1e9d4a5f6b8c9e2a3d4f5e6a... (64 hex chars)

# Old format (BAD):
# auth_session:abc123def456ghi789... (raw token)

# New format (GOOD):
# auth_session:3f8a7b2c1e9d4a5f6b8c9e2a3d4f5e6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2
```

### Test Session Lifecycle
```bash
# 1. Login and capture cookie
TOKEN=$(curl -c cookies.txt -X POST http://localhost:8001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}' \
  | jq -r '.token')

# 2. Make authenticated request
curl -b cookies.txt http://localhost:8001/api/user/profile

# 3. Check Redis (should see hashed key)
redis-cli GET "auth_session:$(echo -n "$TOKEN" | openssl dgst -sha256 -hmac "$REDIS_KEY_HMAC_SECRET" | awk '{print $2}')"

# 4. Logout
curl -b cookies.txt -X POST http://localhost:8001/api/auth/logout

# 5. Verify Redis key deleted
redis-cli GET "auth_session:..." # Should be empty
```

## Performance Impact

### Hash Computation
- **Operation**: HMAC-SHA256 of session token
- **Time**: ~1-5 microseconds per hash
- **Impact**: Negligible (< 0.01% of total request time)

### Memory
- **Old**: ~40 bytes per key (prefix + raw token)
- **New**: ~85 bytes per key (prefix + 64 hex chars)
- **Increase**: ~45 bytes per session
- **Impact**: Negligible for typical workloads (< 1MB for 10,000 sessions)

### Network
- No change (key hashing is local operation)

## References

- OWASP: [Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- NIST: [HMAC-SHA256 Specification](https://csrc.nist.gov/publications/detail/fips/198/1/final)
- Redis Security: [Redis Security Best Practices](https://redis.io/docs/management/security/)

## Troubleshooting

### Sessions Not Persisting
**Symptom**: Users logged out on every request

**Cause**: Auto-generated HMAC secret changes on restart

**Solution**: Set `REDIS_KEY_HMAC_SECRET` environment variable

```bash
# Check logs for warning
journalctl -u businessos-backend | grep "auto-generated HMAC secret"

# Set permanent secret
echo "REDIS_KEY_HMAC_SECRET=$(openssl rand -hex 32)" >> .env
systemctl restart businessos-backend
```

### Horizontal Scaling Issues
**Symptom**: Session valid on some instances, not others

**Cause**: Different HMAC secrets across instances

**Solution**: Ensure same secret on all instances

```bash
# Check secret on each instance
# Instance 1
ssh instance-1 'grep REDIS_KEY_HMAC_SECRET /etc/businessos/.env'

# Instance 2
ssh instance-2 'grep REDIS_KEY_HMAC_SECRET /etc/businessos/.env'

# Should be identical
```

### Legacy Sessions Not Working
**Symptom**: Existing sessions invalid after update

**Expected**: This is normal behavior (see Migration Strategy)

**Solution**: Users re-authenticate (sessions expire in 15 minutes)

## Contact

For security concerns or questions:
- Security Team: security@businessos.com
- Backend Lead: backend@businessos.com
