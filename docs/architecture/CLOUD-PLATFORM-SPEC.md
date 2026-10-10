# BusinessOS Cloud Platform Spec

**Status:** Proposed  
**Date:** 2026-04-11  
**Author:** Roberto H. Luna  
**Reviewers:** @architect, @devops-engineer, @backend-go  

---

## 1. Overview

BusinessOS Cloud turns a local desktop application into a platform where any user can
provision a full BusinessOS instance in a browser, without installing anything. The
mental model is "create a computer" — click one button, get a running BusinessOS workspace
accessible at `{slug}.bos.miosa.ai` within 90 seconds.

**What it is:**
- A hosted, isolated BusinessOS instance running inside a Firecracker microVM
- Accessible entirely through the browser — no Electron required
- Backed by MIOSA's existing compute infrastructure (Firecracker provisioning, compute proxy, Stripe billing)
- Each VM is single-tenant: one workspace per VM, no shared databases

**Who it is for:**
- Users who want BusinessOS without installing software
- Teams that need a shared workspace accessible from any device
- Users who want cloud-synced data without managing local infrastructure

**What it is not:**
- A multi-tenant SaaS app (no shared PostgreSQL, no row-level tenancy)
- A replacement for the Electron desktop app (local mode continues to work independently)
- A new product — it is BusinessOS running on MIOSA compute

**Infrastructure baseline (already exists):**

| Component | Status | Notes |
|-----------|--------|-------|
| Firecracker VM provisioning | Operational | `POST /api/v1/computers` |
| Compute proxy routing | Operational | `*.sandbox.miosa.ai` → VM |
| Stripe billing + credits | Operational | metered + subscription |
| Ollama (qwen3:32b) | Operational | server-side LLM |
| Sparse-clone rootfs boot | Operational | 30–90s cold start |
| Bare-metal server | Operational | AMD EPYC, 64GB RAM, ~14-15 concurrent 4GB VMs |

---

## 2. Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────┐
│  USER BROWSER                                                            │
│  {slug}.bos.miosa.ai                                                     │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ HTTPS
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  CADDY (TLS termination + reverse proxy)                                 │
│  *.bos.miosa.ai → localhost:8080 (compute proxy)                         │
│  Auto-cert: Let's Encrypt wildcard                                       │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ HTTP (internal)
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  MIOSA COMPUTE PROXY  :8080                                              │
│  Routes {slug} → VM private IP:8001                                      │
│  Handles: hibernate wake-on-request, idle detection, health gating       │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │ HTTP (VM private network /30 subnet)
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  FIRECRACKER MICROVM  (per-user)                                         │
│  RAM: 4GB   vCPUs: 2   Disk: 10GB ext4                                  │
│                                                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │  systemd init                                                     │   │
│  │   │                                                               │   │
│  │   ├── PostgreSQL 16 + pgvector  (port 5432, local only)          │   │
│  │   ├── Redis 7                   (port 6379, local only)           │   │
│  │   ├── Go backend (Gin)          (port 8001, external)             │   │
│  │   │    ├── /api/*  REST handlers                                  │   │
│  │   │    ├── /api/chat  SSE streaming                               │   │
│  │   │    └── /health   readiness probe                              │   │
│  │   └── SvelteKit (pre-built static, served by Go backend)         │   │
│  │                                                                   │   │
│  │   envd agent (port 7860, internal — MIOSA lifecycle control)      │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                │
                                │ MIOSA API calls (outbound from VM)
                                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│  MIOSA PLATFORM API  api.miosa.ai                                        │
│  ├── Ollama proxy (qwen3:32b AI calls, credit deduction)                 │
│  ├── Stripe billing (compute minutes, subscription enforcement)           │
│  └── Object storage (daily VM snapshots)                                 │
└─────────────────────────────────────────────────────────────────────────┘


ELECTRON APP (local mode — separate process, same user)
  ├── Sync agent (background)
  └── POST /api/sync/push → {slug}.bos.miosa.ai/api/sync/pull
```

---

## 3. VM Template

### Rootfs Contents (`businessos-desktop.ext4`)

The rootfs is a sparse-cloned ext4 image built once and copy-on-write cloned per VM.

```
/
├── usr/local/bin/
│   ├── businessos-backend          Go binary (statically compiled, linux/amd64)
│   └── envd-agent                  MIOSA lifecycle agent
├── etc/systemd/system/
│   ├── postgresql.service          Standard pg16 unit
│   ├── redis.service               Redis 7 unit (requirepass from /etc/bos/secrets)
│   ├── businessos.service          Go backend unit (after postgres + redis)
│   └── envd.service                MIOSA agent unit (after network-online.target)
├── etc/bos/
│   ├── config.env                  Runtime env vars (written by provisioner at boot)
│   └── secrets                     Redis password, JWT public key, DB password
├── var/lib/postgresql/             PostgreSQL data directory
├── var/lib/redis/                  Redis persistence
├── app/
│   └── static/                     Pre-built SvelteKit output (node build/ contents)
└── tmp/bos/                        Ephemeral working directory
```

### Build Image Process

```bash
# 1. Build Go binary for linux/amd64
GOOS=linux GOARCH=amd64 go build -o businessos-backend ./cmd/server

# 2. Build SvelteKit static output
cd frontend && npm run build

# 3. Assemble rootfs via packer or debootstrap
# Result: businessos-desktop.ext4 (~2GB sparse, ~400MB actual)
```

### Boot Sequence

```
kernel boot (< 1s)
  └── systemd
        ├── [1] postgresql.service      → pg_isready loop until healthy
        ├── [2] redis.service           → redis-cli ping until healthy
        ├── [3] businessos.service      → starts after pg + redis healthy
        │         Reads /etc/bos/config.env
        │         Runs database migrations (idempotent)
        │         Starts HTTP listener :8001
        │         Registers /health endpoint
        └── [4] envd.service            → MIOSA agent reports VM as ready

Total to /health 200: 30–90s (cold) | 5–10s (from snapshot)
```

### Health Endpoint

`GET /health` — returns 200 only when all subsystems are ready.

```json
{
  "status": "ok",
  "version": "1.2.0",
  "deployment_mode": "cloud",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "migrations": "ok"
  },
  "workspace_id": "ws_abc123"
}
```

Returns 503 with partial `checks` map if any subsystem is not ready. The provisioner
polls this endpoint and gates traffic until 200 is received.

### Resource Allocation

| Resource | Allocation | Rationale |
|----------|-----------|-----------|
| RAM | 4 GB | PostgreSQL (1GB) + Redis (512MB) + Go backend (512MB) + headroom |
| vCPUs | 2 | Single-user workload; AI calls offloaded to MIOSA Ollama |
| Disk | 10 GB | rootfs (2GB) + pg data (5GB) + redis (1GB) + app artifacts (2GB) |
| Network | /30 subnet | 2 usable IPs per VM (VM + gateway) — existing MIOSA pattern |

---

## 4. Provisioning Flow

### User-Facing Flow

```
1. User clicks "Create Workspace" in miosa.ai dashboard
2. Frontend opens SSE stream:  GET /api/v1/computers/{provision_id}/stream
3. Backend provisions VM asynchronously (provisioning worker)
4. SSE pushes status events: pending → booting → ready | failed
5. On ready: SSE delivers { sandbox_url, workspace_id }
6. Frontend redirects browser to {sandbox_url}
```

### API Contract

**Request:**
```http
POST /api/v1/computers
Authorization: Bearer {miosa_jwt}
Content-Type: application/json

{
  "template_type": "businessos",
  "name": "My Business",
  "region": "us-east-1",
  "tier": "pro"
}
```

**Response (202 Accepted):**
```json
{
  "provision_id": "prov_xyz789",
  "status": "pending",
  "stream_url": "/api/v1/computers/prov_xyz789/stream",
  "estimated_ready_seconds": 60
}
```

### Provisioning Worker Steps

```
[T+0s]   Receive provisioning job from queue
[T+1s]   Allocate /30 subnet from CIDR pool
[T+2s]   Sparse-clone businessos-desktop.ext4 (copy-on-write, ~2s)
[T+4s]   Write /etc/bos/config.env into rootfs:
           DEPLOYMENT_MODE=cloud
           WORKSPACE_ID={uuid}
           MIOSA_JWT_PUBLIC_KEY={key}
           DB_PASSWORD={random}
           REDIS_PASSWORD={random}
           MIOSA_WORKSPACE_SLUG={slug}
           MIOSA_API_KEY={provisioner_key}
[T+6s]   Boot Firecracker VM (kernel + rootfs)
[T+8s]   Register {slug} → VM_IP:8001 in compute proxy routing table
[T+10s]  Begin health polling: GET {vm_ip}:8001/health every 2s, timeout 90s
[T+35s]  /health returns 200 (typical cold start)
[T+36s]  Update VM record: status=active, sandbox_url={slug}.bos.miosa.ai
[T+36s]  Emit SSE event: { type: "ready", sandbox_url: "...", workspace_id: "..." }
[T+36s]  Provisioning complete
```

**Failure handling:**
- If health poll times out at 90s: emit `{ type: "failed", reason: "boot_timeout" }`, destroy VM, release subnet
- If rootfs clone fails: emit `{ type: "failed", reason: "clone_error" }`, release subnet
- Provisioning worker is idempotent — safe to retry with same `provision_id`

### SSE Event Schema

```
data: {"type":"pending","message":"Allocating resources...","progress":10}
data: {"type":"booting","message":"Starting VM...","progress":40}
data: {"type":"health_check","message":"Waiting for services...","progress":70}
data: {"type":"ready","sandbox_url":"https://my-business.bos.miosa.ai","workspace_id":"ws_abc123","progress":100}
```

---

## 5. Routing

### Subdomain Structure

```
{slug}.bos.miosa.ai
  └── slug: kebab-case workspace name, unique, immutable after creation
             e.g. "my-business" → my-business.bos.miosa.ai
```

### Caddy Configuration Addition

```caddyfile
# Add to existing Caddyfile alongside *.sandbox.miosa.ai block

*.bos.miosa.ai {
    tls {
        dns <provider> {env.DNS_API_KEY}    # wildcard cert via DNS-01
    }
    reverse_proxy localhost:8080 {          # compute proxy
        header_up X-Forwarded-Host {host}
        header_up X-BOS-Slug {labels.1}    # extract slug from subdomain
    }
}
```

### Compute Proxy Routing Table

The proxy maintains an in-memory routing table (persisted to Redis for restart recovery):

```
slug → { vm_ip, vm_port, status, last_request_at, hibernate_at }
```

On each inbound request:
1. Extract slug from `X-BOS-Slug` header
2. Look up routing entry
3. If `status == hibernated`: trigger wake, return 202 with `Retry-After: 10`
4. If `status == active`: forward to `{vm_ip}:{vm_port}`
5. Update `last_request_at`

### SSL

Caddy handles TLS automatically via Let's Encrypt DNS-01 challenge for the `*.bos.miosa.ai`
wildcard. No per-VM certificate provisioning required. DNS provider API key is set in
Caddy environment.

---

## 6. Auth Bridge

### Design Decision

MIOSA provisions the VM with the user's identity baked into `/etc/bos/config.env` at boot
time. BusinessOS validates MIOSA JWTs using a pinned public key loaded at startup. No
round-trip to MIOSA API is required per-request.

### Flow

```
1. User authenticates with MIOSA (miosa.ai login)
2. MIOSA issues JWT: { sub: user_id, email, workspace_id, slug, tier, exp }
3. Browser navigates to {slug}.bos.miosa.ai with JWT in Authorization header
   (or: MIOSA sets __Host-miosa-token cookie before redirect)
4. BusinessOS backend receives request
5. Middleware checks: is MIOSA JWT valid? (verify sig with pinned public key)
6. On first valid JWT for this workspace:
     - Upsert user record in local PostgreSQL (email, user_id from JWT claims)
     - Create better-auth session scoped to {slug}.bos.miosa.ai
     - Set better-auth.session_token cookie
7. Subsequent requests use better-auth cookie (no JWT required)
```

### JWT Validation

```go
// internal/middleware/miosa_auth.go

func (m *MIOSAAuthMiddleware) ValidateMIOSAJWT(tokenStr string) (*MIOSAClaims, error) {
    // Public key loaded from /etc/bos/config.env at startup
    // Key is MIOSA's ECDSA P-256 signing key (public half only)
    key, err := m.loadPublicKey()
    token, err := jwt.ParseWithClaims(tokenStr, &MIOSAClaims{}, func(t *jwt.Token) (any, error) {
        if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
        }
        return key, nil
    })
    // ...
}

type MIOSAClaims struct {
    Sub         string `json:"sub"`          // MIOSA user ID
    Email       string `json:"email"`
    WorkspaceID string `json:"workspace_id"`
    Slug        string `json:"slug"`
    Tier        string `json:"tier"`          // "pro" | "team" | "business"
    jwt.RegisteredClaims
}
```

### Session Scope

- better-auth session cookie: `Domain={slug}.bos.miosa.ai; HttpOnly; Secure; SameSite=Strict`
- Session TTL: 7 days (sliding)
- On JWT re-validation (workspace re-entry): refresh session

### First-Boot Seeding

At VM boot, if `MIOSA_OWNER_EMAIL` is set in `/etc/bos/config.env`, the businessos
backend runs a seeding step before accepting traffic:

```sql
INSERT INTO users (id, email, role) VALUES ($1, $2, 'admin')
ON CONFLICT (email) DO UPDATE SET role = 'admin';

INSERT INTO workspaces (id, name, owner_id) VALUES ($1, $2, $3)
ON CONFLICT (id) DO NOTHING;
```

This ensures the first JWT validation always finds a valid user record.

---

## 7. Data Isolation

### Isolation Model

Each VM contains its own complete PostgreSQL instance. There is no shared database
cluster. Isolation is enforced at three layers:

| Layer | Mechanism | Scope |
|-------|-----------|-------|
| Process | Firecracker KVM | Full kernel-level isolation |
| Network | /30 subnet per VM | No inter-VM IP routing |
| Data | PostgreSQL inside VM | No cross-VM DB access possible |

### Network Topology

```
bare-metal host
├── VM 1: 10.100.0.1/30  (gateway: 10.100.0.2)
├── VM 2: 10.100.0.5/30  (gateway: 10.100.0.6)
├── VM 3: 10.100.0.9/30  (gateway: 10.100.0.10)
└── ... (CIDR pool: 10.100.0.0/16 = 16,383 possible /30 subnets)

VM → host: via TAP device (existing MIOSA pattern)
VM → internet: via NAT on host (for MIOSA API calls, Ollama, Stripe)
inter-VM: NO route (not configured, blocked at host routing table)
```

### Filesystem Isolation

- Firecracker rootfs: copy-on-write clone per VM. Changes to one VM never touch another.
- Host kernel: VMs share the host kernel but are isolated via KVM + seccomp.
- `/tmp/bos/`: cleared on each VM boot.

---

## 8. Lifecycle Management

### States

```
pending → booting → active ─→ hibernated ─→ active
                      │                        ↑
                      │                 wake-on-request
                      └──→ terminated (permanent)
```

### Hibernate

**Trigger:** Compute proxy observes no HTTP request to a VM for 30 minutes.

**Mechanism:**
1. Proxy sets VM status = `hibernating`
2. Proxy calls MIOSA API: `POST /api/v1/computers/{vm_id}/suspend`
3. MIOSA: snapshot VM memory to object storage, pause Firecracker process
4. Proxy updates routing entry: `status=hibernated`
5. Subsequent requests receive 202 with `Retry-After: 10`

**RAM freed:** 4GB per hibernated VM returned to host pool.

### Wake

**Trigger:** Any HTTP request to `{slug}.bos.miosa.ai` when `status=hibernated`.

**Mechanism:**
1. Proxy receives request, sees `status=hibernated`
2. Proxy calls MIOSA API: `POST /api/v1/computers/{vm_id}/resume`
3. MIOSA: restore from snapshot (5–10s) or cold boot (30–90s)
4. Proxy polls `/health` until 200
5. Proxy updates status = `active`, replays original request

**User experience:** Browser receives 202 + progress page while VM wakes. Page auto-refreshes.

### Terminate

**Trigger:** User clicks "Delete Workspace" in MIOSA dashboard.

**Mechanism:**
1. Optional: user requests data export (ZIP of PostgreSQL dump + file attachments)
2. MIOSA waits for export completion (if requested, max 10 min)
3. MIOSA calls `DELETE /api/v1/computers/{vm_id}`
4. Firecracker process killed, rootfs deleted, subnet released, routing entry removed
5. Object storage snapshots deleted (or retained for 30 days per retention policy)

### Backup

**Schedule:** Daily at 03:00 UTC (host local time, staggered across VMs by VM ID)

**Mechanism:**
```
1. MIOSA snapshot agent calls Firecracker pause API
2. rsync rootfs to object storage: s3://miosa-vm-backups/{vm_id}/{date}/rootfs.ext4
3. Unpause VM
4. Retain: 7 daily, 4 weekly, 12 monthly (configurable per tier)
5. Total storage per VM: ~3GB compressed per daily snapshot
```

---

## 9. Billing Integration

### Pricing Tiers

| Tier | Price | VM | Compute | AI Credits | Seats |
|------|-------|----|---------|-----------|----|
| Free | $0 | None (local only) | — | None | 1 |
| Pro | $15/mo | 1 VM | 720 min/mo | 100k tokens/mo | 3 |
| Team | $25/user/mo | 1 VM | Unlimited | Included | Unlimited |
| Business | $50/user/mo | Dedicated VM | Unlimited | Included + priority | Unlimited + SSO |

### Metered Billing

**Compute minutes:** Timer starts when VM status = `active`. Paused when `hibernated`.
Reported to Stripe as a usage record every 5 minutes via metered subscription item.

```go
// Reported every 5 minutes for active VMs
stripe.UsageRecordNew(&stripe.UsageRecordParams{
    SubscriptionItem: stripe.String(sub.ComputeItemID),
    Quantity:         stripe.Int64(5), // minutes
    Timestamp:        stripe.Int64(time.Now().Unix()),
    Action:           stripe.String("increment"),
})
```

**AI credits:** Claude/GPT/Ollama calls made from the VM are proxied through the MIOSA
API. Each call deducts from the user's credit balance. Credits are checked before the
call is forwarded. Insufficient credits return 402.

**Credit proxy flow:**
```
VM → POST api.miosa.ai/proxy/ai  { model, messages, workspace_id }
  → MIOSA checks credit balance
  → MIOSA forwards to provider (Anthropic / OpenAI / local Ollama)
  → MIOSA deducts tokens from credit balance
  → MIOSA streams response back to VM
```

### Enforcement

| Limit | Enforcement |
|-------|-------------|
| Compute minutes exceeded | VM hibernated, user notified, upgrade prompt |
| AI credits at zero | 402 on next AI call, purchase prompt |
| VM count at tier limit | Provisioning rejected with 402 |
| Storage at 10GB | VM writes blocked until data deleted or tier upgraded |

---

## 10. Sync Layer (Local ↔ Cloud)

### Overview

The Electron desktop app can run alongside the cloud workspace. A background sync agent
keeps both in sync. Sync is bidirectional. Conflict resolution is Last-Write-Wins.

### Architecture

```
Electron App
  └── Sync Agent (background goroutine, polls every 30s + on local change)
        ├── Push: local changes → POST {slug}.bos.miosa.ai/api/sync/push
        └── Pull: GET {slug}.bos.miosa.ai/api/sync/pull?since={cursor}
```

### What Syncs

| Synced | Not Synced |
|--------|-----------|
| Projects | Terminal history |
| Tasks | Local file paths |
| Clients | Electron window state |
| CRM (companies, deals) | Local-only settings |
| Conversations (AI chat) | PTY sessions |
| Pages / documents | Electron-specific IPC state |
| Daily logs | |
| Module configurations | |
| Agent definitions | |
| Settings (shared subset) | |

### Sync API Contract

**Push:**
```http
POST /api/sync/push
Authorization: Bearer {miosa_jwt}
Content-Type: application/json

{
  "device_id": "local-mac-abc123",
  "cursor": "2026-04-10T15:30:00Z",
  "entities": [
    {
      "type": "task",
      "id": "task_xyz",
      "updated_at": "2026-04-11T09:00:00Z",
      "operation": "upsert",
      "payload": { ... }
    }
  ]
}
```

**Pull:**
```http
GET /api/sync/pull?since=2026-04-10T15:30:00Z&device_id=local-mac-abc123
Authorization: Bearer {miosa_jwt}

Response 200:
{
  "cursor": "2026-04-11T09:05:00Z",
  "entities": [ { "type": "task", "id": "...", "updated_at": "...", ... } ]
}
```

### Conflict Resolution

Strategy: **Last-Write-Wins** with `(device_id, updated_at)` vector.

```
If remote.updated_at > local.updated_at: accept remote version
If local.updated_at > remote.updated_at: local wins, push overrides remote
If timestamps equal: device_id hash used as tiebreaker (deterministic)
```

This is correct for business data (tasks, clients, deals) where concurrent editing
is rare and eventual consistency is acceptable. It is NOT used for Pages/documents —
those use operational-transform-aware block-level merge (existing Pages editor behavior).

### Encryption

- Transport: TLS 1.3 (Caddy enforces, no TLS 1.2)
- Sync payload: AES-256-GCM encrypted before transmission using a workspace-derived key
- Key derivation: `HKDF(MIOSA_API_KEY, workspace_id, "sync-v1")` — workspace key never
  transmitted, only used locally for encryption

---

## 11. Multi-Tenancy

### Isolation Model

Multi-tenancy at the VM level, not the database level. Each user gets their own VM with
their own PostgreSQL instance. This is stronger isolation than row-level security but
costs more compute.

### Team Access

Team members access the same VM through the same subdomain. The VM's PostgreSQL holds
all team members as users with role-based access.

**Invite flow:**
```
1. Owner visits Settings → Team → Invite
2. BusinessOS generates invite token (JWT: { workspace_id, slug, role, exp: 7 days })
3. Owner sends invite link: https://{slug}.bos.miosa.ai/join?token={invite_token}
4. Teammate clicks link → redirected to MIOSA signup/login
5. After MIOSA auth: teammate presented with invite acceptance screen
6. On accept: MIOSA JWT for teammate passed to BusinessOS
7. BusinessOS validates invite token + MIOSA JWT
8. Upserts teammate user record in VM PostgreSQL with specified role
9. Teammate gets better-auth session on {slug}.bos.miosa.ai
```

### Roles

| Role | Permissions |
|------|------------|
| admin | Full access, invite/remove members, billing |
| member | Full module access, cannot manage team or billing |
| viewer | Read-only access to all modules |

---

## 12. Scaling Plan

### Phase 1 — Now (single server, ≤15 VMs)

```
OVHcloud bare-metal (existing)
├── 14-15 concurrent VMs @ 4GB = ~60GB RAM utilization
├── Hibernate strategy extends effective capacity (cold pool)
├── Manual provisioning via MIOSA API
└── Capacity: ~50 users with hibernate cycle (avg 8hr/day active)
```

Bottleneck: RAM. Hibernate is the primary scaling lever. At full 64GB with aggressive
hibernation, effective capacity is 150–200 registered users.

### Phase 2 — 100 users (~3 months)

```
Add second bare-metal server
├── Implement VM placement algorithm (bin-pack by RAM)
├── Shared object storage (Hetzner/OVH S3-compatible) for snapshots
├── Compute proxy becomes cluster-aware (routing table shared via Redis)
└── Capacity: ~400 users
```

### Phase 3 — 1,000 users (~6 months)

```
Move compute proxy to Kubernetes (K8s)
├── Firecracker VMs still on bare-metal (VMs need KVM, not K8s pods)
├── K8s manages: compute proxy, MIOSA API, billing workers, snapshot workers
├── Bare-metal pool: 5-10 servers behind K8s-managed placement service
├── Auto-scaling: add bare-metal nodes when pool utilization > 70%
└── Capacity: ~4,000 users
```

### Phase 4 — 10,000 users (~12 months)

```
Dedicated compute fleet
├── Multi-region: US-East, EU-West, AP-Southeast
├── Edge CDN for SvelteKit static assets (users hit CDN, not VM for static)
├── VM images stored in regional object storage (fast clone from nearby store)
├── ClickHouse for billing analytics
└── Capacity: 40,000+ users
```

---

## 13. Migration Path (BusinessOS Codebase Changes)

### Deployment Mode Detection

Add `DEPLOYMENT_MODE` to config:

```go
// internal/config/config_types.go

type DeploymentMode string

const (
    ModeLocal DeploymentMode = "local"   // Electron desktop app
    ModeCloud DeploymentMode = "cloud"   // Firecracker VM on MIOSA
)

// Config addition:
DeploymentMode DeploymentMode `mapstructure:"DEPLOYMENT_MODE"`
```

Loaded from `/etc/bos/config.env` on VM boot. Defaults to `local` when not set.

### Backend Changes (cloud mode)

| Change | Local | Cloud |
|--------|-------|-------|
| Auth middleware | better-auth cookie only | MIOSA JWT → better-auth session bridge |
| AI provider | Local Ollama / user API key | MIOSA AI proxy |
| Terminal handler | PTY on local machine | Disabled (return 404 in cloud) |
| Electron IPC routes | Active | Disabled |
| Team features | Optional | Always enabled |
| Sync endpoint | Not present | `POST /api/sync/push`, `GET /api/sync/pull` |
| `/health` endpoint | Basic | Full subsystem check (required by provisioner) |

Conditional behavior:
```go
func (h *TerminalHandler) RegisterRoutes(rg *gin.RouterGroup) {
    if h.config.DeploymentMode == config.ModeCloud {
        rg.Any("/terminal/*path", h.DisabledInCloud)
        return
    }
    // ... existing terminal routes
}
```

### Frontend Changes (cloud mode)

The frontend detects mode from the `/health` API response field `deployment_mode`.

```typescript
// lib/stores/deploymentMode.ts
import { writable, derived } from 'svelte/store'

export const deploymentMode = writable<'local' | 'cloud'>('local')
export const isCloud = derived(deploymentMode, m => m === 'cloud')

// On app boot:
const health = await fetch('/health').then(r => r.json())
deploymentMode.set(health.deployment_mode)
```

Features gated on `isCloud`:

```svelte
{#if !$isCloud}
  <!-- Terminal window icon -->
  <!-- Local file picker -->
  <!-- Electron-specific settings -->
{/if}

{#if $isCloud}
  <!-- Sync status indicator -->
  <!-- Team invite UI (always visible in cloud) -->
  <!-- Hibernation warning banner (when approaching 30min idle) -->
{/if}
```

### Static Asset Serving (cloud mode)

In cloud mode, the Go backend serves the pre-built SvelteKit static output directly
(no separate Node.js process in the VM):

```go
// cmd/server/main.go
if cfg.DeploymentMode == config.ModeCloud {
    // Serve pre-built SvelteKit from /app/static/
    r.NoRoute(gin.WrapH(http.FileServer(http.Dir("/app/static"))))
}
```

This eliminates the need for a Node.js process in the VM, reducing RAM usage by ~150MB.

### Environment Variables Added to `/etc/bos/config.env` (cloud only)

```bash
DEPLOYMENT_MODE=cloud
WORKSPACE_ID=ws_abc123
WORKSPACE_SLUG=my-business
MIOSA_JWT_PUBLIC_KEY=-----BEGIN PUBLIC KEY-----\n...
MIOSA_API_KEY=msk_live_...
MIOSA_OWNER_EMAIL=user@example.com
MIOSA_TIER=pro
DB_URL=postgres://bos:${DB_PASSWORD}@localhost:5432/businessos
REDIS_URL=redis://:${REDIS_PASSWORD}@localhost:6379
```

---

## 14. Security

### Threat Model

| Threat | Mitigation |
|--------|-----------|
| Cross-VM data access | Firecracker KVM isolation + no inter-VM routing |
| MIOSA JWT forgery | ECDSA P-256 signature, public key pinned at boot |
| Session hijacking | HttpOnly + Secure cookie, SameSite=Strict, short TTL |
| Subdomain takeover | Slugs are immutable, routing entry held until explicit delete |
| Noisy neighbor | Firecracker vCPU + RAM hard limits (no balloon driver in cloud mode) |
| SQL injection | Parameterized queries throughout (SQLC-generated, no string concat) |
| Container escape | KVM-level isolation, not container-level |
| Credential leak | Secrets written to `/etc/bos/secrets` with 0600 perms, not in env |

### Defense in Depth

```
Layer 1: Caddy TLS (TLS 1.3 only, HSTS, OCSP stapling)
Layer 2: Compute proxy (rate limiting per slug, auth header required)
Layer 3: Firecracker KVM (hardware virtualization boundary)
Layer 4: /30 network isolation (no inter-VM routing)
Layer 5: BusinessOS middleware (MIOSA JWT validation + better-auth session)
Layer 6: PostgreSQL (workspace-scoped data, role-based access)
```

### Audit Logging

All API calls in cloud mode are logged with:

```json
{
  "timestamp": "2026-04-11T09:00:00Z",
  "tenant_id": "ws_abc123",
  "user_id": "usr_xyz",
  "method": "POST",
  "path": "/api/tasks",
  "status": 201,
  "latency_ms": 12,
  "ip": "10.100.0.1",
  "user_agent": "..."
}
```

Logs are streamed to MIOSA's central log aggregator via the envd agent. Retained 90 days.
Sensitive fields (passwords, tokens, PII) are never logged.

### Data at Rest

- VM disk (ext4): encrypted at the block device level by the Firecracker host using dm-crypt
- Object storage snapshots: AES-256-GCM encrypted before upload, key managed by MIOSA KMS
- PostgreSQL data files: protected by VM-level disk encryption (above)

### Secrets Rotation

- JWT public keys: rotated quarterly; VMs receive new key via envd agent config push + graceful reload
- Redis passwords: per-VM, randomized at provisioning, stored only in `/etc/bos/secrets`
- DB passwords: per-VM, randomized at provisioning, stored only in `/etc/bos/secrets`

---

## Open Questions

| # | Question | Owner | Blocking |
|---|----------|-------|---------|
| 1 | What is the exact MIOSA Firecracker API shape for `suspend`/`resume`? | MIOSA infra | Phase 1 |
| 2 | Does the compute proxy support `X-BOS-Slug` header extraction today? | MIOSA infra | Phase 1 |
| 3 | DNS-01 provider for `*.bos.miosa.ai` wildcard cert? | DevOps | Phase 1 |
| 4 | Object storage provider + bucket policy for VM snapshots? | DevOps | Phase 1 |
| 5 | Acceptable cold-start latency SLA for end users? | Product | UX |
| 6 | Does MIOSA JWT include `tier` claim or must we look it up? | MIOSA auth | Billing enforcement |

---

## References

- [OSA Architecture](./OSA-ARCHITECTURE.md) — Reactive vs proactive layer boundaries
- `desktop/backend-go/internal/integrations/miosa.ComputeClient` — MIOSA compute provisioning client
- [Docker Compose](../../docker-compose.yml) — Local service topology (mirrors VM stack)
- [Frontend Dockerfile](../../frontend/Dockerfile) — Static build output structure
- ADR to be created: `docs/adrs/ADR-CLOUD-001-vm-per-tenant.md` (VM-per-tenant vs shared DB)
