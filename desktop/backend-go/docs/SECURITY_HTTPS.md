# HTTPS Enforcement - Production Deployment

## Overview

BusinessOS **REQUIRES HTTPS** in production to protect against:
- Man-in-the-middle (MITM) attacks
- Credential interception
- Session hijacking
- Data tampering

## Deployment Options

### Option 1: GCP Cloud Run (Recommended - Automatic TLS)

Cloud Run provides automatic TLS termination:

- HTTPS is enforced by default
- Certificates are managed automatically
- All HTTP traffic is redirected to HTTPS
- No additional configuration needed

**Deploy command:**
```bash
gcloud run deploy businessos-backend \
  --source . \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated
```

Cloud Run handles all HTTPS enforcement automatically. Your application receives traffic over HTTP internally, but all external traffic is HTTPS-only.

### Option 2: Reverse Proxy (nginx, Caddy, Traefik)

If deploying behind a reverse proxy, the proxy handles TLS termination.

**nginx configuration:**
```nginx
server {
    listen 443 ssl http2;
    server_name api.businessos.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Modern TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256';
    ssl_prefer_server_ciphers off;

    location / {
        proxy_pass http://localhost:8001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name api.businessos.com;
    return 301 https://$server_name$request_uri;
}
```

**Environment variable:**
```bash
BEHIND_REVERSE_PROXY=true
```

This tells BusinessOS that HTTPS is handled upstream.

### Option 3: Direct TLS Termination (Not Recommended)

For direct TLS termination (without a proxy), you need to:

1. Obtain TLS certificates (Let's Encrypt recommended)
2. Configure Gin to serve HTTPS directly
3. Manage certificate renewal

This is **not recommended** because:
- Certificate renewal is manual or requires additional tooling
- Reverse proxies handle this better
- Cloud platforms provide this automatically

## Verification Checklist

Before going to production, verify:

- [ ] Application is accessible only via HTTPS
- [ ] HTTP requests are redirected to HTTPS
- [ ] TLS certificate is valid and not self-signed
- [ ] Certificate expiry is monitored (Cloud Run handles this automatically)
- [ ] Security headers are set (X-Frame-Options, CSP, etc.)
- [ ] CORS is configured with explicit origins (no wildcards)

## Testing HTTPS Configuration

### Test HTTPS accessibility:
```bash
curl -I https://your-domain.com/health
```

Should return `200 OK` with valid TLS.

### Test HTTP redirect:
```bash
curl -I http://your-domain.com/health
```

Should return `301` or `308` redirect to HTTPS.

### Verify certificate:
```bash
openssl s_client -connect your-domain.com:443 -servername your-domain.com
```

Should show valid certificate chain.

## Production Environment Variables

**Required:**
```bash
ENVIRONMENT=production
ALLOWED_ORIGINS=https://app.businessos.com,https://admin.businessos.com
JWT_SECRET=<secure-random-value>
TOKEN_ENCRYPTION_KEY=<secure-random-value-32-bytes>
REDIS_HMAC_KEY=<secure-random-value-32-bytes>
```

**For reverse proxy deployments:**
```bash
BEHIND_REVERSE_PROXY=true
```

## Security Headers

BusinessOS automatically sets these security headers in production:

- **Secure cookies:** `HttpOnly; Secure; SameSite=Strict`
- **CSRF protection:** Double submit cookie pattern
- **Rate limiting:** 100 req/s per IP, 200 req/s per user

Additional headers should be set at the reverse proxy or Cloud Run level:
- `X-Frame-Options: DENY`
- `X-Content-Type-Options: nosniff`
- `Content-Security-Policy: default-src 'self'`

## Troubleshooting

### "Mixed content" errors in frontend

**Cause:** Frontend making HTTP requests to HTTPS page

**Fix:** Ensure all API calls use HTTPS or relative URLs:
```typescript
// Good:
const API_BASE = import.meta.env.VITE_API_URL || '/api'

// Avoid:
const API_BASE = 'http://localhost:8001/api'
```

### CORS errors after enabling HTTPS

**Cause:** ALLOWED_ORIGINS still has HTTP origins

**Fix:** Update allowed origins to use HTTPS:
```bash
ALLOWED_ORIGINS=https://app.businessos.com,https://admin.businessos.com
```

### Certificate errors

**Cloud Run:** Automatic certificate management - no action needed

**nginx/reverse proxy:** Check certificate renewal is working:
```bash
certbot certificates
certbot renew --dry-run
```

## References

- [OWASP Transport Layer Protection Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Transport_Layer_Protection_Cheat_Sheet.html)
- [Google Cloud Run TLS](https://cloud.google.com/run/docs/securing/using-https)
- [Let's Encrypt](https://letsencrypt.org/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
