# Mix Tasks Reference

All commands are prefixed with `mix optimal.*`.

Run any task with `--help` or inspect the matching file in `lib/mix/tasks/` for
full arguments.

## First-Run And Workspace Setup

| Task | Purpose |
| --- | --- |
| `mix optimal.initiate` | Start a workspace from a messy natural-language setup dump. Preserves the dump, creates a setup Claim, applies conservative Node candidates, and leaves integrations disabled until scoped. Use `--review-only` to keep topology proposals pending. |
| `mix optimal.setup` | Create or refresh a workspace with starter Node Types, Nodes, rhythm files, projections, and agent SOP. |
| `mix optimal.topology` | Inspect workspace topology: Nodes, Node Types, members, and skills. |
| `mix optimal.init` | Older/simple initialization helper. Prefer `optimal.initiate` or `optimal.setup` for new workspaces. |
| `mix optimal.bootstrap` | Compile/migrate/seed convenience path for local demos and smoke checks. |

Examples:

```bash
mix optimal.initiate my-workspace --name "My Workspace" --dump setup.md
mix optimal.initiate regulated-workspace --name "Regulated Workspace" --dump setup.md --review-only
mix optimal.setup my-workspace --node project:launch:"Launch"
mix optimal.topology --workspace default:my-workspace
mix optimal.topology approve <request-id> --workspace default:regulated-workspace --apply
```

## Source Intake And Signal Pipeline

| Task | Purpose |
| --- | --- |
| `mix optimal.ingest` | Classify, route, write projection files, and index a single text/file Signal. |
| `mix optimal.intake` | Interactive multi-line intake from stdin. |
| `mix optimal.ingest_workspace` | Ingest a workspace directory. |
| `mix optimal.index` | Reindex markdown/files under the root. |
| `mix optimal.classify` | Classify content without necessarily writing final state. |
| `mix optimal.parse` | Parse supported file/content inputs. |
| `mix optimal.decompose` | Decompose parsed documents into chunks. |
| `mix optimal.embed` | Generate/store embeddings where configured. |
| `mix optimal.cluster` | Cluster indexed chunks/signals. |
| `mix optimal.architectures` | List registered multimodal data architectures and processors. |

Examples:

```bash
mix optimal.ingest "Customer asked about pricing" --genre note
mix optimal.ingest --file notes.md --genre meeting_notes
mix optimal.architectures
```

## Retrieval And Context

| Task | Purpose |
| --- | --- |
| `mix optimal.search` | Hybrid search across indexed workspace context. |
| `mix optimal.grep` | Literal/content grep over indexed chunks. |
| `mix optimal.rag` | Ask a question using retrieved context. |
| `mix optimal.read` | Read a context by `optimal://` URI and tier. |
| `mix optimal.ls` | List contexts under an `optimal://` URI. |
| `mix optimal.l0` | Print the always-loaded structural inventory. |
| `mix optimal.assemble` | Build a tiered context bundle for a topic. |
| `mix optimal.context.refresh_stale` | Refresh stale stored Context Packages. |

Examples:

```bash
mix optimal.search "pricing decision" --limit 5
mix optimal.rag "what changed this week?"
mix optimal.context.refresh_stale --limit 25
```

## Wiki And Projection Surface

| Task | Purpose |
| --- | --- |
| `mix optimal.wiki list` | List wiki pages for a tenant/workspace. |
| `mix optimal.wiki view <slug>` | View a rendered wiki page. |
| `mix optimal.wiki verify <slug>` | Verify one wiki page. |
| `mix optimal.wiki verify-all` | Verify all wiki pages in a workspace. |
| `mix optimal.wiki render-node <node>` | Render a Node page from governed topology/memory state. |
| `mix optimal.wiki render-tree` | Render a workspace tree page. |
| `mix optimal.wiki check <slug>` | Run projection checks for a page. |

Examples:

```bash
mix optimal.wiki render-tree --workspace default:my-workspace
mix optimal.wiki render-node first-project --workspace default:my-workspace
mix optimal.wiki check node-first-project --workspace default:my-workspace
```

## Memory, Learning, And Graph Work

| Task | Purpose |
| --- | --- |
| `mix optimal.remember` | Store observations and feed learning loops. |
| `mix optimal.rethink` | Synthesize observations into actionable knowledge. |
| `mix optimal.knowledge` | Knowledge graph and learning operations. |
| `mix optimal.graph` | Graph statistics and analysis. |
| `mix optimal.reflect` | Find missing edges from co-occurrences. |
| `mix optimal.reweave` | Find stale contexts on a topic and suggest updates. |
| `mix optimal.simulate` | Run a "what if" scenario through graph/impact logic. |
| `mix optimal.impact` | Impact analysis for an entity or node. |

## Connectors, Tools, And Governance

| Task | Purpose |
| --- | --- |
| `mix optimal.connector` | Connector registry/sync operations. |
| `mix optimal.auth` | Mint, list, revoke, and delete API keys for apps, MCP servers, remote agents, and scripts. |
| `mix optimal.api` | Start the HTTP API. |
| `mix optimal.graph_ui` | Launch the graph visualizer against a running API. |
| `mix optimal.compliance` | Compliance workflows. |
| `mix optimal.backup` | Backup runtime state. |
| `mix optimal.migrate` | Run/verify store migrations. |

Tool, MCP, API, connector, script, and model outputs should flow through
governance before becoming evidence, observations, Claims, or Facts.

Local CLI use is trusted local access to the configured store. Use API keys for
HTTP/API clients, MCP servers, remote agents, external apps, and automation:

```bash
mix optimal.auth mint --name "Business OS" --workspace default:my-workspace
mix optimal.auth env --name "Local Agent" --workspace default:my-workspace
mix optimal.auth list
```

## Evaluation, Health, And Verification

| Task | Purpose |
| --- | --- |
| `mix optimal.reality_check` | Broad runtime probe across store, topology, memory, retrieval, pools, workflows, governance, connectors, evaluation, wiki, and compliance. |
| `mix optimal.eval.run` | Run evaluation datasets. |
| `mix optimal.health` | Diagnostic checks. |
| `mix optimal.verify` | Cold-read fidelity checks. |
| `mix optimal.stats` | Store statistics. |
| `mix optimal.status` | Runtime status. |

Recommended verification:

```bash
mix compile
mix optimal.reality_check
```

Focused workspace/wiki/initiation path:

```bash
mix test test/wiki/service_test.exs \
  test/wiki/store_test.exs \
  test/workspace_export_test.exs \
  test/mix_tasks/optimal_setup_test.exs \
  test/workspace_initiation_test.exs \
  test/mix_tasks/optimal_initiate_test.exs \
  --seed 0
```

## Spec Tooling

| Task | Purpose |
| --- | --- |
| `mix optimal.spec.init` | Scaffold `.spec/` templates. |
| `mix optimal.spec.check` | Validate spec files. |
| `mix optimal.spec.drift` | Detect code changes without corresponding spec updates. |
| `mix optimal.spec.report` | Report spec coverage and verification. |

## Runtime Configuration

| Variable | Meaning |
| --- | --- |
| `OPTIMAL_ENGINE_ROOT` | Root directory for workspace projections. |
| `OPTIMAL_ENGINE_DB` | SQLite database path. |
| `OPTIMAL_ENGINE_CACHE` | Cache directory. |
| `OPTIMAL_ENGINE_TOPOLOGY` | Routing/topology YAML path. |
| `OLLAMA_HOST` | Local embedding/generation host. |

Default local data shape:

```text
.optimal/index.db
.optimal/cache/
workspace folders and markdown projections
```
