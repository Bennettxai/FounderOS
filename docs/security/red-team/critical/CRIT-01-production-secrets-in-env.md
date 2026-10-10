# CRIT-1: Live Production Credentials in `.env` File

**Severity:** CRITICAL | **Exploitability:** 10/10 | **CVSS:** 9.8

## Location
`desktop/backend-go/.env`

## Description
The `.env` file contains live production credentials including:
- `DATABASE_URL` with embedded Supabase password (`TheBusinessOS01!`)
- `GOOGLE_CLIENT_SECRET` (`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`)
- `MIOSA_API_KEY` (JWT valid until 2027 with admin role)
- `SECRET_KEY` (88-byte base64 key used for HMAC signing)
- `TOKEN_ENCRYPTION_KEY` (AES key for credential vault encryption)
- `REDIS_KEY_HMAC_SECRET` (HMAC key for Redis session cache)

## Impact
- **Confidentiality:** Full database read — all user data, sessions, OAuth tokens
- **Integrity:** Database write — create users, forge sessions, manipulate records
- **Availability:** DROP tables, exhaust connection pools

## PoC
```bash
cat desktop/backend-go/.env | grep -E "DATABASE_URL|SECRET|KEY|CLIENT"
# Returns all live credentials
```

## Fix
1. Rotate ALL credentials immediately
2. Purge from git history with `git filter-repo` or BFG
3. Use macOS Keychain, SOPS, or Vault for local development secrets
4. Add `.env` verification to CI that rejects committed secrets
