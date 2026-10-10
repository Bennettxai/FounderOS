# Quick Start: Secure Session Key Implementation

## What Changed?
Session tokens are now hashed using HMAC-SHA256 before being stored as Redis keys. This prevents attackers from enumerating valid session tokens.

## Required Action

### Development Setup
```bash
# 1. Add to your .env file
echo "REDIS_KEY_HMAC_SECRET=dev-hmac-secret-change-in-production" >> .env

# 2. Restart server
go run cmd/server/main.go

# 3. Verify in logs
# Look for: "Session cache enabled (TTL=15m, HMAC-secured keys)"
```

### Production Setup
```bash
# 1. Generate a strong secret (32+ bytes)
openssl rand -hex 32
# Example output: e8f3a9b2c7d6e1f4a8b3c9d2e7f1a6b8c3d9e2f7a1b6c8d3e9f2a7b1c6d8e3f9

# 2. Add to production environment
export REDIS_KEY_HMAC_SECRET="e8f3a9b2c7d6e1f4a8b3c9d2e7f1a6b8c3d9e2f7a1b6c8d3e9f2a7b1c6d8e3f9"

# Or add to .env (ensure .env is in .gitignore!)
echo "REDIS_KEY_HMAC_SECRET=e8f3a9b2c7d6e1f4a8b3c9d2e7f1a6b8c3d9e2f7a1b6c8d3e9f2a7b1c6d8e3f9" >> .env

# 3. Deploy and restart
./deploy.sh
# or
systemctl restart businessos-backend

# 4. Verify
journalctl -u businessos-backend -n 50 | grep "HMAC-secured keys"
```

## What Happens to Existing Sessions?

### Expected Behavior
- All existing Redis sessions will be inaccessible (different key hashes)
- Users will be logged out
- PostgreSQL still has session data (used as fallback)
- Sessions expire naturally within 15 minutes
- Users re-authenticate automatically

### Timeline
```text
T+0min:  Deploy new code
T+0min:  All Redis cache keys become stale (different hashes)
T+0min:  Requests fall back to PostgreSQL (slight performance hit)
T+1min:  Users re-authenticate, new hashed keys populate Redis
T+15min: All old unhashed keys expire (TTL)
T+15min: System fully migrated, back to normal performance
```
## Verification

### Check Configuration
```bash
# 1. Verify environment variable is set
echo $REDIS_KEY_HMAC_SECRET

# 2. Check it's loaded in config
grep REDIS_KEY_HMAC_SECRET .env

# 3. Verify it's not the default/empty
# Should NOT be empty or "dev-hmac-secret-change-in-production" in production
```

### Check Redis Keys
```bash
# 1. Login to application
# 2. Check Redis keys
redis-cli KEYS "auth_session:*"

# Expected: Long hashed keys (64 hex characters)
# auth_session:3f8a7b2c1e9d4a5f6b8c9e2a3d4f5e6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2

# NOT acceptable: Short unhashed keys
# auth_session:abc123def456...
```

### Check Logs
```bash
# Look for successful initialization
journalctl -u businessos-backend -n 100 | grep "Session cache enabled"
# Expected: "Session cache enabled (TTL=15m, HMAC-secured keys)"

# Check for warnings (should be none in production)
journalctl -u businessos-backend -n 100 | grep -E "WARNING.*HMAC|auto-generated"
# Expected: No output in production
```

## Common Issues

### Issue: "auto-generated HMAC secret" warning in logs
**Cause**: REDIS_KEY_HMAC_SECRET not set

**Fix**:
```bash
export REDIS_KEY_HMAC_SECRET="$(openssl rand -hex 32)"
systemctl restart businessos-backend
```

### Issue: Sessions don't persist across server restarts
**Cause**: Auto-generated secret changes on each restart

**Fix**: Set permanent REDIS_KEY_HMAC_SECRET (see Production Setup)

### Issue: Sessions work on one server, not another (multi-instance)
**Cause**: Different HMAC secrets on different instances

**Fix**: Ensure same secret on all instances
```bash
# Check each instance
ssh instance-1 'echo $REDIS_KEY_HMAC_SECRET'
ssh instance-2 'echo $REDIS_KEY_HMAC_SECRET'
# Should be identical
```

### Issue: "shorter than recommended 32 bytes" warning
**Cause**: HMAC secret is too short

**Fix**: Generate longer secret
```bash
# Old (too short)
REDIS_KEY_HMAC_SECRET="short123"

# New (recommended)
REDIS_KEY_HMAC_SECRET="$(openssl rand -hex 32)"
```

## Security Checklist

Production deployment checklist:

- [ ] Generate strong random secret (min 32 bytes)
  ```bash
  openssl rand -hex 32
  ```

- [ ] Store secret securely (secrets manager, NOT git)
  ```bash
  aws secretsmanager create-secret --name redis-hmac-secret --secret-string "..."
  ```

- [ ] Set REDIS_KEY_HMAC_SECRET on all instances
  ```bash
  export REDIS_KEY_HMAC_SECRET="..."
  ```

- [ ] Verify secret is same across all instances
  ```bash
  for i in instance-{1..3}; do ssh $i 'echo $REDIS_KEY_HMAC_SECRET'; done
  ```

- [ ] Deploy during low-traffic period
  ```bash
  # Schedule maintenance window
  ```

- [ ] Verify logs show "HMAC-secured keys"
  ```bash
  journalctl -u businessos-backend | grep "HMAC-secured keys"
  ```

- [ ] Verify no auto-generation warnings
  ```bash
  journalctl -u businessos-backend | grep "auto-generated" | wc -l  # Should be 0
  ```

- [ ] Test session creation and retrieval
  ```bash
  curl -X POST http://localhost:8001/api/auth/login -d '...'
  ```

- [ ] Verify Redis keys are hashed (64 hex chars)
  ```bash
  redis-cli KEYS "auth_session:*" | head -1
  ```

- [ ] Document secret location for team
  ```bash
  echo "HMAC secret stored in: AWS Secrets Manager - redis-hmac-secret" >> runbook.md
  ```

- [ ] Schedule quarterly secret rotation
  ```bash
  # Add to calendar/runbook
  ```

## Performance Notes

- Hash computation: ~1-5 microseconds (negligible)
- Memory overhead: +45 bytes per session key (negligible)
- No network impact
- Total overhead: < 0.01% of request time

## More Information

- Full documentation: `docs/REDIS_KEY_SECURITY.md`
- Implementation summary: `docs/SECURITY_IMPLEMENTATION_SUMMARY.md`
- Configuration example: `.env.example`

## Support

Questions or issues?
1. Check logs: `journalctl -u businessos-backend -f`
2. Read docs: `docs/REDIS_KEY_SECURITY.md`
3. Contact: security@businessos.com
