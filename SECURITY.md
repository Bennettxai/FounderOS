# Security Policy

> **Status:** ACTIVE
> **Owner:** Team
> **Last Updated:** 2026-02-24

---

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest `main` | Yes |
| Older releases | No |

BusinessOS is in active development. Only the current `main` branch receives security fixes.

---

## Reporting a Vulnerability

**DO NOT open a public GitHub issue for security vulnerabilities.**

To report a vulnerability, email: **security@businessos.app**

We will:
1. Acknowledge receipt within 24 hours
2. Investigate and provide an initial assessment within 72 hours
3. Release a fix within 7 days for critical vulnerabilities
4. Credit the reporter (unless they prefer anonymity)

For detailed incident response procedures, see [docs/security/INCIDENT-RESPONSE-PLAN.md](docs/security/INCIDENT-RESPONSE-PLAN.md).

---

## Security Architecture

### Authentication

BusinessOS uses session-based authentication via BetterAuth.

| Mechanism | Detail |
|-----------|--------|
| **Password storage** | bcrypt (adaptive hashing, cost factor 12) |
| **OAuth provider** | Google OAuth 2.0 (email + profile scopes) |
| **Session transport** | HttpOnly + Secure cookies; never accessible to JavaScript |
| **Session token storage** | Redis — tokens are stored as HMAC-SHA256 hashes, never as plaintext |
| **Session lifetime** | 7-day maximum inactivity window; 24-hour sliding refresh; 30-day absolute maximum |
| **Session invalidation** | Tokens are rotated on each auth event (login, OAuth callback) |

Multi-factor authentication (MFA) is planned for a future release.

### Authorization

Role-based access control (RBAC) is enforced at the workspace level.

| Role | Permissions |
|------|-------------|
| **Owner** | Full workspace control, billing, member management, deletion |
| **Admin** | Member management, all data operations, settings |
| **Manager** | Create/edit/delete workspace resources; cannot manage members |
| **Member** | Read and create resources scoped to own assignments |

All API endpoints verify:
1. A valid session exists (authenticated)
2. The session's user has the required role for the requested operation (authorized)
3. The resource belongs to the user's workspace (tenant isolation)

Resource ownership is validated on every request. Cross-workspace access is not permitted except through the explicit Cross-OS grant system documented in the architecture.

### Data Protection

| Layer | Control |
|-------|---------|
| **In transit** | TLS 1.3 enforced by Cloud Run; HTTP is redirected to HTTPS |
| **At rest (database)** | Cloud SQL encryption using Google-managed keys (AES-256) |
| **OAuth tokens** | Encrypted at the column level using AES-256-GCM before storage in the credential vault |
| **Passwords** | bcrypt hashed; plaintext is never stored or logged |
| **Session tokens** | HMAC-SHA256 hashed in Redis; plaintext only in the HttpOnly cookie |

For full data classification, retention schedules, and PII inventory, see [docs/security/DATA-CLASSIFICATION.md](docs/security/DATA-CLASSIFICATION.md).

For secret storage and rotation schedules, see [docs/security/SECRETS-MANAGEMENT.md](docs/security/SECRETS-MANAGEMENT.md).

### Network Security

See [docs/security/NETWORK-SECURITY.md](docs/security/NETWORK-SECURITY.md) for the full network topology diagram and verification commands.

**Summary:**

| Control | Implementation |
|---------|---------------|
| **CORS** | Restricted to `https://businessos.app` and `https://app.businessos.app` in production; `http://localhost:5173` in development only |
| **Rate limiting** | 20 req/sec default; 10 req/min on `POST /api/auth/*` (brute-force protection); 5 req/5-min on LLM generation endpoints |
| **CSRF protection** | Double-submit cookie pattern via BetterAuth |
| **Security headers** | HSTS, X-Frame-Options: DENY, X-Content-Type-Options: nosniff, Content-Security-Policy, Referrer-Policy |
| **Database network** | Cloud SQL has no public IP; accessible only via Cloud SQL Proxy from authorized Cloud Run services |
| **Redis network** | VPC-peered; no external access |

### Dependency Security

| Language | Integrity Mechanism | CI Scanning |
|----------|--------------------|-----------:|
| Go | `go.sum` checksum verification on every build | `govulncheck` planned for CI (not yet wired) |
| Node/TypeScript | `package-lock.json` lockfile integrity | `npm audit` planned for CI (not yet wired) |

Automated dependency scanning via Dependabot or Snyk is planned. Adding `govulncheck` and `npm audit` steps to CI workflows is a near-term priority.

---

## Audit Logging

All security-relevant events are written as structured JSON using Go's `slog` package and forwarded to Cloud Logging.

| Event Category | Events Logged |
|---------------|---------------|
| **Authentication** | Login success/failure, OAuth callback, session creation, session expiry, logout |
| **Authorization** | Permission denied events (endpoint, user ID, required role, actual role) |
| **Sensitive data access** | Access to restricted data classifications (as defined in DATA-CLASSIFICATION.md) |
| **API requests** | All requests: method, path, status code, latency, user ID (no request bodies) |
| **Secret/credential operations** | OAuth token storage/rotation (event only, never the token value) |

**What is never logged:** Passwords, plaintext session tokens, OAuth tokens, API keys, full request/response bodies, or any Restricted-classification data.

Log retention: 30 days in Cloud Run logs (GCP default). Audit logs (auth events, permission denials) are additionally written to the `audit_logs` PostgreSQL table with a 1-year retention.

Log access is restricted to GCP project editors (project maintainers).

---

## Compliance

### Data Classification

BusinessOS classifies all data as Public, Internal, Confidential, or Restricted. Full details are in [docs/security/DATA-CLASSIFICATION.md](docs/security/DATA-CLASSIFICATION.md).

### GDPR

The following technical controls support GDPR compliance:

| Requirement | Control |
|-------------|---------|
| Right to erasure | Account deletion cascades to all user data via PostgreSQL foreign keys |
| Data export | User data export endpoint is planned for Q2 2026 |
| Data minimization | Only email and optional display name are collected at registration |
| PII in transit | TLS 1.3 for all data in transit |
| PII at rest | Encrypted via Cloud SQL (Google-managed AES-256) |
| Breach notification | 72-hour notification target per INCIDENT-RESPONSE-PLAN.md |

BusinessOS is not yet certified under any compliance framework. GDPR technical controls are implemented; a formal DPA process and legal review are in progress.

### SOC 2

SOC 2 Type I preparation is planned for Q2 2026. BusinessOS is not currently SOC 2 certified.

### HIPAA

BusinessOS does not currently meet HIPAA requirements. Business Associate Agreements (BAAs), customer-managed encryption keys (CMEK), and enhanced access logging required for HIPAA readiness are planned for Q4 2026. **Do not store Protected Health Information (PHI) in BusinessOS at this time.**

For the full compliance roadmap, see [docs/security/COMPLIANCE-OVERVIEW.md](docs/security/COMPLIANCE-OVERVIEW.md).

---

*Last Updated: 2026-02-24*
