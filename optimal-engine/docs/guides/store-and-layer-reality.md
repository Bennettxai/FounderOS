# Store And Layer Reality

This guide explains what actually runs inside Optimal Engine.
It avoids product language and focuses on the real stores, layers, checks, and failure modes.

## Short Version

Optimal Engine is not one database and it is not one RAG pipeline.
It is a runtime with one canonical local store today, plus several derived or optional stores around it.

```text
SQLite
  -> canonical local runtime rows

Raw files and artifacts
  -> preserved source evidence

FTS, vectors, chunks, cache
  -> rebuildable retrieval indexes

In-memory graph store
  -> hydrated knowledge graph view from stored edges

Optional graph backends such as RocksDB
  -> specialized graph or triple-store acceleration when installed and configured

API, CLI, markdown, wiki, app views
  -> surfaces that read and write through engine rules
```

If you remember one thing, remember this:

```text
SQLite owns local truth today.
Indexes and caches make retrieval faster.
The graph store makes relationships usable.
RAG is a query path across those stores.
RocksDB is the preferred local knowledge graph backend when the native NIF is installed.
ETS remains the in-memory fallback.
```

## Runtime Flow

This is the real flow from input to answer.

```text
Raw inputs
  -> meetings, docs, notes, files, transcripts, connector payloads, app data

Ingestion
  -> normalizes the source, preserves evidence, creates source records

SQLite canonical store
  -> stores the durable truth for contexts, chunks, entities, edges, vectors, claims, facts, memory objects, packages, and audit

Runtime graph store
  -> hydrates graph edges from SQLite into RocksDB for the full local setup
  -> uses ETS only as the fallback when RocksDB is unavailable or intentionally disabled

Retrieval layer
  -> uses search, chunks, vectors, graph relationships, ranking, and Context Packages

RAG path
  -> assembles an answer from retrieved governed context
  -> does not create truth by itself

App and API surfaces
  -> BusinessOS, CLI, wiki, markdown, and API clients read and write through engine rules
```

The practical mental model is:

```text
SQLite = durable truth
RocksDB = fast persistent graph runtime
Chunks and vectors = retrieval material
Search and graph = ways to find relevant context
RAG = answer assembly from retrieved context
BusinessOS = app surface that calls the engine with workspace scope
```

BusinessOS should not own this engine data.
BusinessOS should send the correct tenant, organization, and workspace scope to Optimal Engine.
Optimal Engine then decides what that workspace can read and returns scoped graph, search, or RAG results.

For example:

```text
BusinessOS workspace
  -> Optimal Engine API
  -> tenant, organization, and workspace scope check
  -> search, graph, or RAG query
  -> SQLite plus RocksDB plus retrieval projections
  -> scoped response back to BusinessOS
```

If Workspace A can read Workspace B private data, that is a bug.
If an app screen shows stale data but the engine API is correct, that is an app or mapping bug.
If the engine API returns the wrong workspace data, that is an engine isolation bug.

## What Each Store Does

| Store | Is it required locally? | What it does | How to verify |
| --- | --- | --- | --- |
| SQLite | Yes | Canonical local runtime rows for workspaces, sources, claims, facts, memories, context packages, tools, connectors, wiki, and audit. | `curl http://localhost:4200/api/health` shows `store` and `migrations` ok. |
| Raw artifact storage | Yes when files/media exist | Preserves uploaded files, attachments, transcripts, media, and source evidence before interpretation. | Reality check probes `source package preserved` and `connector sync payload preserves assets`. |
| FTS rows | Yes for text search | Full-text search projection built from contexts and memory rows. | `mix optimal.search "query"` returns scoped hits. |
| Vector rows | Optional but expected for semantic retrieval | Embedding-backed retrieval projection. | Reality check retrieval probes pass. Health stays degraded only if configured embedder is broken. |
| Chunk rows | Yes for document-scale retrieval | Breaks large sources into retrievable pieces. | Reality check store counts include `chunks`; ingesting files should increase chunk rows. |
| Cache directories | Optional | Rebuildable acceleration for parsing, embeddings, and runtime work. | Can be deleted and rebuilt without losing truth. |
| RocksDB graph store | Yes for the full local setup | Persistent knowledge graph backend hydrated from SQLite edges. | `lsof` shows `.optimal/knowledge-rocksdb` files open and startup logs show `Knowledge backend: RocksDB`. |
| ETS graph store | Fallback | In-memory graph backend when RocksDB is unavailable or disabled. | Startup logs show `Knowledge backend: ETS`. |
| Mnesia graph store | Optional | Distributed BEAM graph backend for specialized deployments. | Backend tests pass and startup logs show `Knowledge backend: Mnesia` when configured. |
| Postgres | No for local dev | Production target for canonical runtime rows. | Verify in production profile, not normal local boot. |

## Agent-Facing Logical Stores

`GET /api/stores` reports the current logical storage capabilities without
exposing raw secrets or arbitrary SQL access.

| Logical store | Local backing | Contents |
| --- | --- | --- |
| relational | SQLite | topology, sources, claims, facts, memories, episodes, and audit |
| full-text | SQLite FTS5 | exact text and phrase retrieval |
| vector | SQLite float32 BLOBs | context and chunk embeddings |
| graph | RocksDB plus SQLite lineage | relationship traversal and multi-hop reasoning |
| assets | filesystem plus SQLite metadata | uploaded files and multimodal extraction records |
| cache | ETS and filesystem cache | rebuildable hot context |
| jobs | SQLite | leases, retries, and dead letters |
| metrics | Telemetry, Prometheus, and SQLite history | operational measurements |
| backups | files plus SQLite catalog | checksum, integrity, retention, and offsite status |
| decomposition | SQLite plus optional DSPy RLM | deterministic, recursive, and fallback runs |
| models | SQLite metadata plus artifact storage | training examples, runs, and adapters |
| secrets | bcrypt and encrypted envelopes | API authentication and connector credentials |

Operational and model records can be inspected through bounded, sanitized,
workspace-scoped endpoints under `/api/stores/:id/records`. Secret record
inspection is intentionally rejected.

## What Each Layer Owns

Do not organize the system by database name.
Organize it by ownership.

| Layer | Owns | Does not own |
| --- | --- | --- |
| Topology | Tenants, organizations, workspaces, Nodes, relationships, memberships, aliases, and routing scope. | Final claims about the world. |
| Source Intake | Preserving raw evidence before interpretation. | Deciding that evidence is true. |
| Signal Pipeline | Classifying inputs by mode, genre, type, format, and structure. | Long-term truth. |
| Memory Core | Claims, Facts, Memory Objects, evidence links, lineage, and promotion policy. | App layout or random markdown edits. |
| Retrieval | Search, RAG, Context Packages, ranking, filtering, and scoped recall. | Creating facts without review. |
| Active Memory Pools | Task-local working context. | Permanent workspace truth. |
| Workflow and Skills | Repeatable procedures and skill packages. | Uncontrolled tool calls. |
| Tool and Model Governance | Permissions, tool/model runs, input/output validation, audit records, and rejection. | Secret storage in markdown. |
| Wiki and Export | Human-facing projections. | Canonical source of truth. |

## How To Prove The Engine Works

Start the engine:

```bash
brew install snappy
make install
make bootstrap
make dev
```

Check the live process:

```bash
curl http://localhost:4200/api/health
```

Expected shape:

```json
{
  "status": "up",
  "live": true,
  "ok?": true,
  "checks": {
    "store": ":ok",
    "credential_key": ":ok",
    "migrations": ":ok"
  }
}
```

Run the full backend probe:

```bash
mix optimal.reality_check
```

For storage-specific proof in an OptimalOS checkout, run:

```bash
.system/oe storage_check
```

The equivalent direct Engine endpoint is:

```bash
curl http://localhost:4200/api/stores/audit
```

Do not report storage as healthy from table existence alone. The audit must
show zero failures, FTS row parity, zero foreign-key violations, zero
legacy-default rows, valid vector dimensions, existing asset paths, and an
existing integrity-verified backup.

Expected summary:

```text
total probes: 126   ok: 126   warn: 0   fail: 0
```

That check proves the main runtime spine:

```text
store tables
health and supervision
architecture registry
workspace topology
source evidence
claims and facts
memory objects
context packages
workflow and skill lifecycle
tool and model governance
connector registry
asset preservation
wiki projections
retrieval and RAG
compliance workflows
```

To prove the live local graph is running on RocksDB:

```bash
pid=$(launchctl list | awk '/com.optimalos.engine/{print $1}')
lsof -p "$pid" | rg 'knowledge-rocksdb|liberocksdb|index.db'
```

Expected:

```text
.optimal/index.db
.optimal/knowledge-rocksdb/LOG
.optimal/knowledge-rocksdb/MANIFEST-...
deps/rocksdb/priv/liberocksdb.so
```

## How To Prove Retrieval Works

Use search for direct text recall:

```bash
mix optimal.search "pricing"
```

Use RAG when you want an answer assembled from governed context:

```bash
mix optimal.rag "what changed this week?"
```

Use topology when the question depends on workspace shape:

```bash
mix optimal.topology --workspace default:my-workspace
```

Use wiki render checks when humans or apps depend on projections:

```bash
mix optimal.wiki render-tree --workspace default:my-workspace
mix optimal.wiki check node-first-project --workspace default:my-workspace
```

## How To Prove Workspace Isolation Works

Use explicit workspace scope in every command or API call:

```bash
mix optimal.search "pricing" --workspace default:workspace-a
mix optimal.search "pricing" --workspace default:workspace-b
```

The result sets should differ when the workspaces contain different data.
If the same private result leaks across unrelated workspaces, that is a bug.

For API clients, send workspace scope explicitly in the request body or query string.
Do not rely on vague names.

## What Not To Promise

Do not tell users that every possible backend is always running.
That is not true.

Say this instead:

```text
The local engine runs SQLite as the canonical store.
It builds retrieval indexes and hydrates graph views from that store.
The full local setup uses RocksDB as the persistent knowledge graph backend.
ETS is the fallback backend when RocksDB is unavailable or intentionally disabled.
The reality check tells you which layers are working in this checkout.
```

Do not call indexes, cache, markdown, or app screens the source of truth.
They are surfaces or rebuildable projections unless a specific layer says otherwise.

## Common Confusions

| Confusion | Correct explanation |
| --- | --- |
| "RocksDB is the database." | No. SQLite is the local canonical store today. RocksDB is the persistent knowledge graph backend in the full local setup. |
| "RAG is the brain." | No. RAG is one retrieval path across governed stores and context packages. |
| "Markdown is the source of truth." | No. Markdown is a projection and editing surface. Important edits re-enter as source evidence or pending changes. |
| "The app owns the data." | No. Apps are surfaces. The engine owns lifecycle and policy. |
| "A vector DB is enough." | No. Vectors help retrieval, but they do not replace topology, source evidence, claims, facts, permissions, or audit. |
| "If search returns something, it is true." | No. Search returns candidates. Facts require review or policy promotion. |

## Agent Checklist

Before telling a user the engine is healthy, check:

```bash
git status --short --branch
curl http://localhost:4200/api/health
mix optimal.reality_check
git ls-files | rg '(^|/)\\.optimal|index\\.db|connector_key' || true
```

The last command should print nothing.
If it prints local DBs, keys, or runtime files, stop and fix `.gitignore` or tracked files before pushing.
