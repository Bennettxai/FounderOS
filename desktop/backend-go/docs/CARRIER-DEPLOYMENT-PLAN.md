# CARRIER Protocol Bridge — Deployment Plan

**Date:** 2026-02-24
**Status:** Accepted
**Author:** Architect Agent (OSA)
**Relates to:** `internal/carrier/` package, SORX engine integration

---

## Table of Contents

1. [Overview and Constraints](#1-overview-and-constraints)
2. [Infrastructure Option Evaluation](#2-infrastructure-option-evaluation)
3. [Decision Matrix](#3-decision-matrix)
4. [Recommended Topology](#4-recommended-topology)
5. [Local Development Setup](#5-local-development-setup)
6. [Production Deployment (CloudAMQP Fast-Start)](#6-production-deployment-cloudamqp-fast-start)
7. [Production Deployment (Self-Hosted HA)](#7-production-deployment-self-hosted-ha)
8. [Environment Variables Reference](#8-environment-variables-reference)
9. [TLS Certificate Setup](#9-tls-certificate-setup)
10. [Monitoring and Observability](#10-monitoring-and-observability)
11. [Connection Pooling Strategy](#11-connection-pooling-strategy)
12. [Runbook: Common Operations](#12-runbook-common-operations)
13. [ADR-003: RabbitMQ Broker Selection](#13-adr-003-rabbitmq-broker-selection)

---

## 1. Overview and Constraints

### What CARRIER Does

The CARRIER bridge is an AMQP 0-9-1 message layer between:

- **BOS (Go)** — the BusinessOS backend running the SORX engine. Each BOS
  instance is a `carrier.Client` that publishes requests and consumes replies.
- **SorxMain (Elixir)** — the advanced reasoning engine consuming from five
  durable queues and publishing responses back on per-OS reply queues.

### Fixed Topology Requirements

These requirements are non-negotiable and constrain broker choice:

| Requirement | Detail |
|---|---|
| Protocol | AMQP 0-9-1 (not AMQP 1.0) |
| Exchange type | `topic` (durable) |
| Priority queues | `x-max-priority: 10` per queue |
| Durable queues | 5: `sorx.boardroom`, `sorx.critic`, `sorx.pddl`, `sorx.mcts`, `sorx.events` |
| Ephemeral queues | Per-OS `sorx.responses.{os_instance_id}` (exclusive, auto-delete) |
| Delivery mode | Transient (speed over durability for reasoning requests) |
| Message TTL | 60 000 ms default |
| Reconnection | Exponential backoff 1 s → 30 s (implemented in `client.go`) |

### Why Broker Choice Is Constrained

The `x-max-priority` queue argument and native AMQP 0-9-1 topic exchanges are
RabbitMQ-specific features. They eliminate two cloud options immediately:

- **Azure Service Bus**: Uses AMQP 1.0, not 0-9-1. Topic subscriptions do not
  map to RabbitMQ `topic` exchanges. Priority levels max out at 7. Rejected.
- **Amazon SQS / SNS**: No native AMQP. Would require a translation layer.
  Rejected.

Viable brokers: RabbitMQ (self-hosted), CloudAMQP (managed RabbitMQ),
Amazon MQ for RabbitMQ (managed RabbitMQ on AWS).

---

## 2. Infrastructure Option Evaluation

### Option A: Self-Hosted RabbitMQ via Docker Compose

Run RabbitMQ in Docker on the same host or a dedicated VM alongside BOS and
SorxMain.

**Cost:** Near zero (Docker host cost only). A 2 vCPU / 4 GB Hetzner CX22
instance costs ~$5/mo and is sufficient for CARRIER traffic volumes at early
stage.

**Latency:** Sub-millisecond within the same host or same LAN. Network hops are
zero when co-located.

**Reliability / HA:** Single point of failure. No automatic failover. If the
host dies, BOS falls back to local LLM (the `FallbackError` / `ReasonDisabled`
path in `client.go`). Acceptable for development and early production.

**Ease of setup:** One `docker-compose` stanza. The RabbitMQ management image
includes the management UI and `rabbitmq_prometheus` plugin.

**Monitoring:** Management UI on port 15672. Prometheus scrape on port 15692
(requires `rabbitmq_prometheus` plugin enabled).

**TLS:** Requires manual cert provisioning. Can use Let's Encrypt with certbot
or self-signed certs for internal services.

**Verdict:** Best for local dev and early single-tenant production. Upgrade path
to clustered when needed.

---

### Option B: CloudAMQP (Managed RabbitMQ)

CloudAMQP operates managed RabbitMQ clusters on AWS/GCP/Azure. You get an
`amqps://` URL and delegate all broker operations.

**Pricing (2026):**

| Plan | Monthly | Connections | Messages/mo | HA |
|---|---|---|---|---|
| Lemur (free) | $0 | 20 | 1 M | No |
| Little Lemur | $19 | 100 | Unlimited | No (1 node) |
| Tiger | $99 | 500 | Unlimited | Yes (3-node cluster) |
| Bunny | $299 | 3000 | Unlimited | Yes (3-node) |

For a small team: Lemur (free) for staging/dev, Little Lemur ($19) for
single-tenant production, Tiger ($99) for HA production.

**Latency:** 1-10 ms RTT within the same cloud region as BOS. Cross-region adds
30-100 ms. Co-locate CloudAMQP and BOS in the same AWS region.

**Reliability / HA:** Lemur/Little Lemur are single-node (no HA). Tiger and
above use 3-node mirrored clusters with automatic failover. Uptime SLA: 99.95%
on paid plans.

**Ease of setup:** Create account, copy URL, set `CARRIER_AMQP_URL`. Done in
under 5 minutes. No infrastructure to manage.

**Monitoring:** Built-in CloudAMQP console. Also exposes the standard RabbitMQ
management API and Prometheus endpoint on paid plans.

**TLS:** All CloudAMQP URLs are `amqps://` by default. Certificates are
managed by CloudAMQP (Let's Encrypt). No action required.

**Verdict:** Best "start here" recommendation. Free tier covers development and
testing. $19/mo covers early production. Scales to HA at $99/mo.

---

### Option C: Amazon MQ for RabbitMQ

AWS-managed RabbitMQ. Runs inside your VPC.

**Pricing (2026):**

| Instance | Monthly (single-AZ) | Monthly (multi-AZ) |
|---|---|---|
| mq.t3.micro | ~$15 | ~$30 |
| mq.m5.large | ~$200 | ~$400 |

Multi-AZ requires a 3-node cluster and is not available on micro instances.

**Latency:** Sub-millisecond within the same VPC. Lowest latency option if BOS
and SorxMain already run on AWS.

**Reliability / HA:** Single-AZ is a single broker (no HA). Multi-AZ deploys
a 3-node cluster across availability zones.

**Ease of setup:** More complex than CloudAMQP. Requires VPC configuration,
security groups, and IAM for AWS authentication. No free tier.

**Monitoring:** CloudWatch metrics. Prometheus requires a sidecar or custom
metric exporter.

**TLS:** TLS is enforced. Uses AWS Certificate Manager certificates.

**Verdict:** Recommended only if BOS is already deeply integrated with AWS
(ECS/EKS, VPC peering requirements, compliance mandates). Costs 2-4x more than
CloudAMQP for equivalent HA setup. Skip for early stage.

---

### Option D: Self-Hosted RabbitMQ Cluster (3-node, Fly.io or Hetzner)

Three RabbitMQ nodes configured as a cluster with quorum queues. Suitable for
production at scale.

**Cost (Hetzner CX22, 3 nodes):** ~$15/mo total for modest workloads.
**Cost (Fly.io, 3 shared-cpu-1x):** ~$15-20/mo.

**Latency:** Sub-millisecond within same datacenter / region. Fly.io private
networking provides 1-3 ms intra-region.

**Reliability / HA:** True HA with quorum queues (Raft consensus). Tolerates 1
node failure in a 3-node cluster. Requires ops knowledge to maintain.

**Ease of setup:** Moderate. Requires cluster configuration, peer discovery, and
volume management. Fly.io simplifies this with their native clustering support.

**Monitoring:** Full Prometheus + Grafana stack. RabbitMQ ships
`rabbitmq_prometheus` plugin.

**TLS:** Manual cert management or Fly.io's built-in TLS termination.

**Verdict:** Best long-term production option once you have ops capacity.
Migrate to this from CloudAMQP when monthly message volume or connection count
justifies it, or when data residency requirements preclude managed services.

---

## 3. Decision Matrix

Criteria weights reflect the priorities of a small, early-stage team: ship fast,
keep costs low, maintain reliability sufficient for an AI reasoning bridge where
`FallbackError` is a graceful degradation path (not a hard failure).

| Criterion (weight) | A: Docker Self-Host | B: CloudAMQP | C: Amazon MQ | D: Self-Host Cluster |
|---|---|---|---|---|
| Time to running (0.30) | 4 (1.20) | 5 (1.50) | 2 (0.60) | 2 (0.60) |
| Cost at early stage (0.25) | 5 (1.25) | 5 (1.25) | 2 (0.50) | 4 (1.00) |
| HA / reliability (0.20) | 1 (0.20) | 3 (0.60) | 3 (0.60) | 5 (1.00) |
| Operational burden (0.15) | 3 (0.45) | 5 (0.75) | 4 (0.60) | 2 (0.30) |
| Monitoring quality (0.10) | 3 (0.30) | 4 (0.40) | 3 (0.30) | 5 (0.50) |
| **TOTAL** | **3.40** | **4.50** | **2.60** | **3.40** |

**Winner: CloudAMQP for fast-start. Self-Hosted Cluster for production scale.**

---

## 4. Recommended Topology

### Fast-Start (Team < 5, single BOS instance)

```
┌─────────────────┐     amqps://        ┌─────────────────────┐
│   BOS (Go)      │ ──────────────────► │  CloudAMQP          │
│  carrier.Client │                     │  (Little Lemur $19) │
│  OS_INSTANCE_ID │ ◄────────────────── │                     │
│  = bos-prod-1   │   sorx.responses.   │  Exchange:          │
└─────────────────┘   bos-prod-1        │  sorx.carrier       │
                                        │                     │
┌─────────────────┐     amqps://        │  Queues:            │
│  SorxMain       │ ──────────────────► │  sorx.boardroom     │
│  (Elixir)       │                     │  sorx.critic        │
│                 │ ◄────────────────── │  sorx.pddl          │
└─────────────────┘   publishes to      │  sorx.mcts          │
                      sorx.responses.*  │  sorx.events        │
                                        └─────────────────────┘
```

- BOS and SorxMain can be in the same region as CloudAMQP (e.g. all on AWS
  us-east-1 or Fly.io iad).
- TLS is provided by CloudAMQP. Use `amqps://` URL.
- Single CloudAMQP node. If it goes down, BOS falls back to local LLM via
  `FallbackError` — by design.

### Production-Ready (Multi-tenant, HA required)

```
                        ┌──────────────────────────────────┐
                        │  RabbitMQ Cluster (3 nodes)      │
                        │  Node 1  Node 2  Node 3          │
┌─────────────────┐     │  ┌────┐  ┌────┐  ┌────┐         │
│  BOS instance 1 │────►│  │ R1 │  │ R2 │  │ R3 │         │
│  bos-prod-1     │     │  └────┘  └────┘  └────┘         │
└─────────────────┘     │       Quorum Queues              │
                        │  ─────────────────────────────   │
┌─────────────────┐     │  sorx.boardroom (replicated)     │
│  BOS instance 2 │────►│  sorx.critic    (replicated)     │
│  bos-prod-2     │     │  sorx.pddl      (replicated)     │
└─────────────────┘     │  sorx.mcts      (replicated)     │
                        │  sorx.events    (replicated)     │
┌─────────────────┐     │                                  │
│  SorxMain       │────►│  + N ephemeral reply queues      │
│  (Elixir)       │     └──────────────────────────────────┘
└─────────────────┘              │
                         ┌───────┴────────┐
                         │  HAProxy / NLB │
                         │  (VIP endpoint)│
                         └────────────────┘
                         Single AMQP endpoint
                         clients connect to
```

- HAProxy or a cloud NLB provides a single virtual endpoint.
- Clients connect to the VIP; failover is transparent.
- Quorum queues replace classic mirrored queues (RabbitMQ 3.8+).
  The `x-max-priority` argument is compatible with quorum queues as of
  RabbitMQ 3.13.
- Each BOS instance has its own `OS_INSTANCE_ID` and ephemeral reply queue.

### Network Placement Rules

1. RabbitMQ must be in the **same cloud region** as both BOS and SorxMain.
   Cross-region AMQP adds 30-100 ms per hop and approaches the 60 s TTL
   under load.
2. Firewall rules must allow AMQP (5672) and AMQPS (5671) inbound only from
   BOS and SorxMain IP ranges. Never expose the management UI (15672) to
   the public internet.
3. For Fly.io deployments, use Fly private networking (`.internal` DNS).
   For AWS, use VPC security groups — no public endpoint.

---

## 5. Local Development Setup

### Step 1: Add RabbitMQ to docker-compose.yml

Add the following service to the existing
`/Users/rhl/Desktop/MIOSA/_ACTIVE/BOS/desktop/backend-go/docker-compose.yml`:

```yaml
  rabbitmq:
    image: rabbitmq:3.13-management-alpine
    container_name: businessos-rabbitmq
    restart: unless-stopped
    ports:
      - "5672:5672"    # AMQP
      - "15672:15672"  # Management UI (dev only - never expose in prod)
      - "15692:15692"  # Prometheus metrics
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER:-bos}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASS:?RABBITMQ_PASS must be set}
      RABBITMQ_DEFAULT_VHOST: /
    volumes:
      - rabbitmq-data:/var/lib/rabbitmq
      - ./docker/rabbitmq/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro
      - ./docker/rabbitmq/enabled_plugins:/etc/rabbitmq/enabled_plugins:ro
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "-q", "ping"]
      interval: 15s
      timeout: 10s
      retries: 5
    networks:
      - businessos-network
```

Also add to the `volumes:` block:

```yaml
  rabbitmq-data:
    driver: local
```

### Step 2: Create RabbitMQ configuration files

Create `docker/rabbitmq/enabled_plugins`:

```
[rabbitmq_management,rabbitmq_prometheus].
```

Create `docker/rabbitmq/rabbitmq.conf`:

```ini
# RabbitMQ configuration for local development
# For production, this file is replaced by cloud provider config or Kubernetes ConfigMap

loopback_users.guest = false
listeners.tcp.default = 5672

# Management UI
management.tcp.port = 15672

# Prometheus metrics endpoint
prometheus.tcp.port = 15692

# Log level
log.console = true
log.console.level = info

# Memory threshold
vm_memory_high_watermark.relative = 0.7

# Disk free limit
disk_free_limit.absolute = 512MB
```

### Step 3: Add CARRIER environment variables to .env

Add to `.env` (and `.env.example`):

```bash
# ═══════════════════════════════════════════════════════════════
# [16] CARRIER - SORX Reasoning Bridge
# ═══════════════════════════════════════════════════════════════
# AMQP broker URL. Use amqps:// in production.
# For local dev with docker-compose:
CARRIER_AMQP_URL=amqp://bos:YOUR_RABBITMQ_PASS@localhost:5672/

# Unique identifier for this BOS instance.
# Must be stable across restarts (responses route back by this ID).
OS_INSTANCE_ID=bos-local-1

# Enable CARRIER bridge (false = fallback to local LLM only)
CARRIER_ENABLED=false

# Timeout for synchronous Send() calls (default: 60s)
# CARRIER_SEND_TIMEOUT=60s

# Prefetch count for reply queue consumer (default: 10)
# CARRIER_PREFETCH=10

# RabbitMQ credentials (must match docker-compose.yml)
RABBITMQ_USER=bos
RABBITMQ_PASS=
```

### Step 4: Start the stack

```bash
# Generate a RabbitMQ password
openssl rand -base64 32  # paste into .env as RABBITMQ_PASS and CARRIER_AMQP_URL

# Start all services
docker compose up -d

# Verify RabbitMQ is healthy
docker compose ps rabbitmq

# Open management UI
open http://localhost:15672
# Login: bos / YOUR_RABBITMQ_PASS

# Enable CARRIER and test
CARRIER_ENABLED=true go run ./cmd/server
```

### Step 5: Verify topology declaration

After BOS starts with `CARRIER_ENABLED=true`, the `carrier.Client` calls
`declareTopology()` which creates the exchange and all five queues. Verify in
the management UI under "Exchanges" and "Queues":

- Exchange `sorx.carrier` (topic, durable)
- Queues: `sorx.boardroom`, `sorx.critic`, `sorx.pddl`, `sorx.mcts`,
  `sorx.events` (all durable, max-priority=10)
- Queue: `sorx.responses.bos-local-1` (auto-delete, exclusive — disappears
  when BOS shuts down)

---

## 6. Production Deployment (CloudAMQP Fast-Start)

### Step 1: Create CloudAMQP instance

1. Sign up at https://www.cloudamqp.com
2. Create a new instance:
   - **Plan:** Little Lemur ($19/mo) for single BOS instance production
   - **Region:** Same as your BOS deployment (e.g. AWS us-east-1, GCP us-central1)
   - **Name:** `businessos-sorx-prod`
3. Copy the AMQP URL from the instance detail page. It will look like:
   `amqps://user:pass@hefty-pelican.rmq.cloudamqp.com/vhost`

### Step 2: Set production environment variables

```bash
# In your production environment (Railway, Fly.io, Render, ECS, etc.)
CARRIER_AMQP_URL=amqps://user:pass@hefty-pelican.rmq.cloudamqp.com/vhost
CARRIER_ENABLED=true
OS_INSTANCE_ID=bos-prod-1     # stable, unique per BOS instance
CARRIER_SEND_TIMEOUT=60s
CARRIER_PREFETCH=10
```

`OS_INSTANCE_ID` must be stable across restarts. If BOS restarts with the
same `OS_INSTANCE_ID`, it re-creates the same exclusive reply queue name.
Since the queue is `exclusive`, a new connection can re-declare it cleanly.
Use the deployment name or pod name if running in Kubernetes.

### Step 3: Enable Prometheus plugin on CloudAMQP

In the CloudAMQP console, navigate to your instance and enable the
"RabbitMQ Prometheus" plugin under "Plugins". This exposes:

```
https://hefty-pelican.rmq.cloudamqp.com/metrics
```

Add HTTP Basic Auth credentials (same as management API) for scraping.

### Step 4: TLS verification

The `amqps://` scheme in `CARRIER_AMQP_URL` activates TLS in `amqp.Dial`.
CloudAMQP uses Let's Encrypt certificates which are trusted by the Go
standard `crypto/tls` library. No additional configuration needed.

Verify TLS is active:

```bash
openssl s_client -connect hefty-pelican.rmq.cloudamqp.com:5671 -quiet
# Should show: Certificate chain with Let's Encrypt CA
```

### Step 5: Smoke test

```bash
# From a BOS instance with CARRIER_ENABLED=true
# Tail logs and look for:
# carrier: connected to RabbitMQ reply_queue=sorx.responses.bos-prod-1

# Send a test boardroom request via the Go REPL or a test endpoint
```

---

## 7. Production Deployment (Self-Hosted HA)

Use this path when you need data residency, cost at scale, or CloudAMQP
connection limits become a bottleneck.

### Option 7A: Fly.io 3-node cluster

Fly.io supports RabbitMQ clustering via Fly Volumes and private networking.

**fly.toml** for RabbitMQ:

```toml
app = "businessos-rabbitmq"
primary_region = "iad"

[build]
  image = "rabbitmq:3.13-management-alpine"

[env]
  RABBITMQ_ERLANG_COOKIE = "replace-with-secure-random-64-char-string"

[[services]]
  internal_port = 5672
  protocol = "tcp"

  [[services.ports]]
    port = 5671
    handlers = ["tls"]
    tls_options = { alpn = ["amqp"] }

[[mounts]]
  source = "rabbitmq_data"
  destination = "/var/lib/rabbitmq"
  initial_size = "10gb"
```

Scale to 3 nodes:

```bash
fly scale count 3 --app businessos-rabbitmq
```

Enable clustering via `rabbitmq.conf` peer discovery using Fly DNS:

```ini
cluster_formation.peer_discovery_backend = rabbit_peer_discovery_dns
cluster_formation.dns.hostname = businessos-rabbitmq.internal
cluster_name = bos-sorx-cluster
```

### Option 7B: Hetzner 3-node cluster

1. Provision three CX22 VMs (2 vCPU, 4 GB, ~$5/mo each) in the same datacenter.
2. Use Docker Swarm or Ansible to deploy RabbitMQ with the cluster configuration.
3. Use a private network (Hetzner Cloud Networks) to isolate AMQP traffic.
4. Deploy HAProxy on a separate small VM as the AMQP VIP.

**HAProxy configuration** (`/etc/haproxy/haproxy.cfg`):

```
frontend amqp_front
    bind *:5671 ssl crt /etc/ssl/rabbitmq/combined.pem
    mode tcp
    default_backend amqp_back

backend amqp_back
    mode tcp
    balance roundrobin
    timeout connect 5s
    timeout server 60s
    server rmq1 10.0.0.1:5672 check inter 5s
    server rmq2 10.0.0.2:5672 check inter 5s
    server rmq3 10.0.0.3:5672 check inter 5s
```

### Quorum Queue Compatibility

The existing `declareTopology()` in `client.go` declares classic durable queues
with `x-max-priority`. For the self-hosted HA path, you have two options:

**Option 1 (recommended):** Keep classic queues with mirroring policy.

```bash
# Set via rabbitmqctl after cluster is up
rabbitmqctl set_policy ha-sorx "^sorx\." \
  '{"ha-mode":"all","ha-sync-mode":"automatic"}' \
  --priority 1 --apply-to queues
```

This mirrors all `sorx.*` queues across all nodes with no code change.

**Option 2:** Migrate to quorum queues (RabbitMQ 3.13+).

Quorum queues support `x-max-priority` as of RabbitMQ 3.13. Change the queue
declaration in `client.go` to add:

```go
amqp.Table{
    "x-max-priority": int32(MaxPriority),
    "x-queue-type":   "quorum",
},
```

Quorum queues provide stronger durability guarantees (Raft replication) but
have slightly higher latency than classic mirrored queues. For the CARRIER
workload (reasoning requests with 60 s TTL, transient delivery mode), either
is acceptable.

---

## 8. Environment Variables Reference

Complete reference for all CARRIER-related environment variables consumed by
`internal/carrier/config.go`:

| Variable | Required | Default | Description |
|---|---|---|---|
| `CARRIER_ENABLED` | No | `false` | Master switch. `false` = fallback-only mode. |
| `CARRIER_AMQP_URL` | When enabled | `amqp://guest:guest@localhost:5672/` | Full AMQP/AMQPS connection string. Credentials are masked in logs. |
| `OS_INSTANCE_ID` | When enabled | (none) | Stable unique identifier for this BOS instance. Determines reply queue name: `sorx.responses.{id}`. |
| `CARRIER_SEND_TIMEOUT` | No | `60s` | Max wait for synchronous `Send()` response. Go duration string. |
| `CARRIER_PREFETCH` | No | `10` | AMQP QoS prefetch count for reply consumer. Positive integer. |

### AMQP URL Format

```
amqp[s]://[user[:password]@]host[:port][/vhost]

Examples:
  amqp://bos:pass@localhost:5672/           # local dev, no TLS
  amqps://user:pass@mq.example.com/        # production with TLS
  amqps://user:pass@mq.example.com:5671/   # explicit TLS port
  amqps://user:pass@rmq.cloudamqp.com/vhost  # CloudAMQP
```

### OS_INSTANCE_ID Guidelines

- Must be unique across all BOS instances connecting to the same broker.
- Must be stable across restarts (the reply queue is re-created on reconnect).
- May contain alphanumerics, hyphens, and underscores.
- Maximum length: 64 characters (RabbitMQ queue name limit is 255, prefix is
  `sorx.responses.` = 16 chars).

Suggested values by deployment type:

| Deployment | Example OS_INSTANCE_ID |
|---|---|
| Local dev | `bos-local-dev` |
| Docker Compose | `bos-compose-1` |
| Single production VM | `bos-prod-1` |
| Kubernetes pod | Use `$(POD_NAME)` env var |
| Railway | `bos-railway-$(RAILWAY_REPLICA_ID)` |
| Fly.io | `bos-fly-$(FLY_ALLOC_ID)` |

---

## 9. TLS Certificate Setup

### CloudAMQP (Managed)

No action required. Use the `amqps://` URL provided by CloudAMQP. Certificates
are managed by the provider.

### Self-Hosted with Let's Encrypt

If RabbitMQ has a public DNS name (even for internal use, via split-horizon DNS):

```bash
# Install certbot
apt install certbot

# Obtain certificate (standalone mode — stop RabbitMQ briefly)
certbot certonly --standalone -d mq.yourdomain.com

# Certificates land at:
# /etc/letsencrypt/live/mq.yourdomain.com/fullchain.pem
# /etc/letsencrypt/live/mq.yourdomain.com/privkey.pem

# Configure RabbitMQ to use them
# rabbitmq.conf:
listeners.ssl.default = 5671
ssl_options.cacertfile = /etc/letsencrypt/live/mq.yourdomain.com/fullchain.pem
ssl_options.certfile   = /etc/letsencrypt/live/mq.yourdomain.com/fullchain.pem
ssl_options.keyfile    = /etc/letsencrypt/live/mq.yourdomain.com/privkey.pem
ssl_options.verify     = verify_peer
ssl_options.fail_if_no_peer_cert = false
```

Auto-renew via cron and reload RabbitMQ:

```bash
# /etc/cron.d/rabbitmq-cert-renew
0 3 * * * root certbot renew --quiet && rabbitmqctl eval 'ssl:clear_pem_cache().'
```

### Self-Hosted with Internal CA (No Public DNS)

For internal-only deployments where RabbitMQ has no public DNS name:

```bash
# Generate CA
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
  -subj "/CN=CARRIER Internal CA"

# Generate server cert
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr \
  -subj "/CN=rabbitmq.internal"
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt \
  -extfile <(echo "subjectAltName=DNS:rabbitmq.internal,IP:10.0.0.1")

# In RabbitMQ:
ssl_options.verify = verify_peer
ssl_options.fail_if_no_peer_cert = false
```

For BOS (Go) to trust the internal CA, set the `CARRIER_AMQP_URL` to use
`amqps://` and configure the TLS config by wrapping `amqp.DialTLS`:

```go
// If using internal CA, customize TLS in your carrier initialization:
tlsCfg := &tls.Config{
    RootCAs: certPool,  // load ca.crt into certPool
}
conn, err := amqp.DialTLS(cfg.URL, tlsCfg)
```

This requires a small addition to `client.go`'s `connect()` method. See the
`amqp.DialTLS` function in the `amqp091-go` library.

---

## 10. Monitoring and Observability

### Key Metrics to Track

The RabbitMQ `rabbitmq_prometheus` plugin exposes all critical metrics. The
following are most important for CARRIER:

| Metric | Alert Threshold | Meaning |
|---|---|---|
| `rabbitmq_queue_messages{queue="sorx.boardroom"}` | > 1000 | SorxMain consumer is lagging |
| `rabbitmq_queue_messages_ready` | > 500 per queue | Unprocessed backlog building |
| `rabbitmq_queue_messages_unacknowledged` | > 100 per queue | Consumer processing slowdown |
| `rabbitmq_connections` | > 80% of plan limit | Approaching connection cap |
| `rabbitmq_channel_messages_published_total` | Rate drop > 50% | BOS publishing stopped |
| `rabbitmq_queue_message_bytes_ready{queue~"sorx.responses.*"}` | > 0 for > 120s | Reply queue leaking (orphaned) |
| `rabbitmq_node_mem_used` / `rabbitmq_node_mem_limit` | > 0.7 | Memory pressure |

### Prometheus Scrape Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: rabbitmq-carrier
    scheme: https                          # or http for local dev
    basic_auth:
      username: bos
      password_file: /etc/prometheus/rabbitmq-pass
    static_configs:
      - targets:
          - hefty-pelican.rmq.cloudamqp.com:443  # CloudAMQP
          # or:
          # - rabbitmq.internal:15692              # self-hosted
    metrics_path: /metrics
    scrape_interval: 15s
    scrape_timeout: 10s
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
        replacement: rabbitmq-carrier
```

### Grafana Dashboard

Import the official RabbitMQ Grafana dashboard (ID: `10991`) from
grafana.com/grafana/dashboards. It covers all the metrics above and provides
queue depth visualizations.

Add a custom panel for CARRIER-specific latency tracking using BOS application
logs. The `carrier.Client` logs `DurationMS` from `Response.DurationMS` — ship
these to your log aggregator and plot percentiles.

### Application-Level Observability

Add structured log parsing rules for these log lines emitted by `client.go`:

```
carrier: connected to RabbitMQ              # Startup success
carrier: AMQP connection closed unexpectedly # Reconnection event
carrier: attempting reconnect               # Backoff state
carrier: reconnected successfully           # Recovery
```

Alert on: more than 3 reconnect attempts in 5 minutes (suggests broker
instability).

### Health Check Endpoint

Expose a `/health/carrier` endpoint in BOS that returns:

```json
{
  "carrier_enabled": true,
  "carrier_connected": true,
  "os_instance_id": "bos-prod-1"
}
```

Use `client.IsConnected()` for the `carrier_connected` field. Wire this into
your existing health check infrastructure.

---

## 11. Connection Pooling Strategy

### Current Architecture

The current `carrier.Client` uses a **single connection, single channel**
model (see `client.go`). This is correct for the CARRIER workload because:

1. **Publishing** is serialized through one channel with an `RWMutex`, which
   prevents channel-level flow control issues.
2. **Consuming** runs in a dedicated goroutine on the same channel.
3. **Correlation-ID routing** (`pending sync.Map`) handles concurrent
   in-flight requests safely.

AMQP 0-9-1 channels are multiplexed over one TCP connection. At CARRIER's
expected throughput (10-1000 reasoning requests/minute), a single connection
is not a bottleneck. Do not add connection pooling prematurely.

### When to Add a Second Channel

If you observe channel-level blocking under high publish rates (> 10 000
msg/sec), split publishing and consuming onto separate channels:

```go
type Client struct {
    publishCh *amqp.Channel  // dedicated publish channel
    consumeCh *amqp.Channel  // dedicated consume channel
    ...
}
```

This separation prevents a slow consumer from blocking the publisher. For
CARRIER's current workload, this is not necessary.

### When to Add Connection Pooling

Add a pool only when:
- You have > 50 concurrent BOS instances sharing one broker
- You need publisher confirms (not currently used — delivery mode is transient)
- You are processing > 100 000 msg/sec

For a pool, use a `sync.Pool` of `*amqp.Connection` objects or a dedicated
connection pool library. Each channel in the pool should have its own
`QueueDeclare` call on first use to ensure idempotent topology.

### Multiple BOS Instances

Each BOS instance has its own `OS_INSTANCE_ID` and therefore its own
`sorx.responses.{id}` reply queue. There is no shared state between BOS
instances at the AMQP layer. Multiple BOS instances can run against the same
broker without coordination — the five durable queues are idempotently
declared by each client on connect.

---

## 12. Runbook: Common Operations

### Drain and inspect a queue

```bash
# Via management API
curl -u bos:PASS https://mq.example.com/api/queues/%2F/sorx.boardroom

# Get queue depth
curl -u bos:PASS https://mq.example.com/api/queues/%2F/sorx.boardroom \
  | jq '.messages_ready'
```

### Purge a stuck queue

```bash
# WARNING: destroys all unprocessed messages
curl -u bos:PASS -X DELETE \
  https://mq.example.com/api/queues/%2F/sorx.boardroom/contents
```

### Force-close an orphaned reply queue

Reply queues are `exclusive + auto-delete` and self-clean when the connection
closes. If an orphan persists (BOS crashed without clean disconnect):

```bash
# Delete via management API
curl -u bos:PASS -X DELETE \
  "https://mq.example.com/api/queues/%2F/sorx.responses.bos-prod-1"
```

### Test connectivity from BOS host

```bash
# Install amqp client tool
go install github.com/rabbitmq/amqp091-go/cmd/amqp-consume@latest

# Or use netcat for raw TCP check
nc -zv mq.example.com 5671 && echo "TLS port reachable"
nc -zv mq.example.com 5672 && echo "AMQP port reachable"
```

### Rolling restart of RabbitMQ cluster (self-hosted)

```bash
# Stop one node at a time — do NOT stop the cluster all at once
# Quorum queues tolerate one node down in a 3-node cluster

# On node 3:
systemctl stop rabbitmq-server
# ... perform maintenance ...
systemctl start rabbitmq-server
rabbitmqctl cluster_status  # verify node rejoined

# Repeat for node 2, then node 1
```

### Rotate RabbitMQ credentials

1. Create a new user with the same permissions.
2. Update `CARRIER_AMQP_URL` in all BOS instances (blue/green deploy or
   rolling restart).
3. Delete the old user.
4. The `carrier.Client` will reconnect on next restart (or immediately if the
   connection drops and `watchConnection` triggers).

---

## 13. ADR-003: RabbitMQ Broker Selection for CARRIER

**Status:** Accepted
**Date:** 2026-02-24

### Context

The CARRIER bridge requires AMQP 0-9-1 with native topic exchanges, priority
queues (`x-max-priority`), and per-client ephemeral reply queues. Three cloud
broker options were evaluated against these hard requirements plus the team's
early-stage constraints (low ops burden, low cost, fast time-to-running).

### Decision

**Fast-start path:** CloudAMQP managed RabbitMQ (Little Lemur, $19/mo).
**Production-HA path:** Self-hosted 3-node RabbitMQ cluster on Fly.io or
Hetzner with classic mirrored queues and an HAProxy VIP.

CloudAMQP wins the fast-start path with a score of 4.50/5.00 in the weighted
decision matrix, primarily because it eliminates all infrastructure ops, ships
with TLS on by default, and has a zero-friction free tier for staging.

### Consequences

#### Positive
- Development to production parity: same RabbitMQ version, same topology.
- TLS out of the box on CloudAMQP — no cert management in early stage.
- `FallbackError` / `ReasonDisconnected` in `client.go` means broker downtime
  degrades gracefully to local LLM, not a hard failure.
- Migrations to self-hosted HA require only a URL change (`CARRIER_AMQP_URL`)
  and a topology re-declaration on first connect (idempotent).

#### Negative
- CloudAMQP Little Lemur (single node) is a SPOF. Broker downtime triggers
  local fallback. Upgrade to Tiger ($99/mo) for HA before go-live with paying
  customers.
- Self-hosted HA requires ops expertise (Erlang clustering, quorum queue
  tuning, HAProxy health checks). Plan for this before hitting CloudAMQP
  connection limits.

#### Neutral
- `OS_INSTANCE_ID` must be managed as a first-class configuration value. In
  Kubernetes, bind it to `POD_NAME`. In serverless/ephemeral environments,
  use a stable identifier from the platform (e.g. `FLY_ALLOC_ID`).

### Alternatives Rejected

- **Azure Service Bus:** AMQP 1.0 only. Priority queue max is 7 (not 10).
  Topic exchange semantics differ. Incompatible without a translation layer.
- **Amazon MQ for RabbitMQ:** Technically compatible but 2-4x more expensive
  than CloudAMQP for equivalent HA tier. No free tier. Complex VPC setup.
  Recommended only if BOS is mandated to run entirely within AWS and data
  residency requirements prohibit third-party managed services.
- **Amazon SQS:** No AMQP support. Would require replacing the entire
  `carrier.Client` with SQS SDK calls. Rejected.

### References

- `internal/carrier/config.go` — `ConfigFromEnv`, env var names
- `internal/carrier/client.go` — `declareTopology`, reconnection logic
- `internal/carrier/routing.go` — queue names, topic constants
- CloudAMQP pricing: https://www.cloudamqp.com/plans.html
- RabbitMQ Quorum Queues: https://www.rabbitmq.com/docs/quorum-queues
- RabbitMQ Priority Queues: https://www.rabbitmq.com/docs/priority
