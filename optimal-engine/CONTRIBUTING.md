# Contributing to the Optimal Engine

The Optimal Engine is open-source (MIT). Contributions are welcome — bug fixes, new connectors, SDK improvements, and documentation. Read this before opening a PR.

---

## Development Setup

**Prerequisites**

- Elixir 1.17+ and Erlang/OTP 26+
- SQLite 3.35+ (FTS5 must be compiled in — verify with `sqlite3 :memory: "SELECT fts5()"`)
- Node.js 20+ (for the TypeScript SDK, MCP server, and site)
- Git

**Clone and install**

```bash
git clone https://github.com/Miosa-osa/OptimalEngine.git
cd OptimalEngine
mix deps.get
```

**Run the test suite**

```bash
mix test
```

The test alias wipes the test SQLite store before each run so schema drift
between migration versions cannot leak. The current local suite is over 1,600
tests; use CI as the source of truth for the exact count on your branch.

**Start the engine locally**

```bash
mix run --no-halt
# HTTP API available at http://localhost:4000
```

---

## Code Style

**Elixir**

```bash
mix format          # auto-format (enforced in CI)
mix credo --strict  # static analysis
```

All submitted code must pass `mix format` without diff and `mix credo --strict` without new warnings.

**TypeScript (SDK / MCP / site)**

```bash
cd sdks/typescript && npm run typecheck
cd apps/mcp       && npm run typecheck
```

Strict mode is on everywhere. No `any`. Explicit return types on public functions.

---

## Pull Request Process

1. **Fork** the repository and create a branch from `main`.
2. **Name your branch** using the scope prefix convention (see Commit Messages below).
3. **Write or update tests** — new behaviour needs coverage, bug fixes need a regression test.
4. **Run the safety checks** locally before pushing:
   `mix format --check-formatted && mix compile && mix test && mix optimal.reality_check`.
5. **Open a PR** against `main`. Keep the diff reviewable — aim for under 400 lines changed per PR. Split larger changes into a stack.
6. **Describe what changed and why** in the PR body. Link any related issues.
7. At least one maintainer approval is required before merge.

---

## Commit Message Convention

Use imperative mood. Prefix with a scope when the change is confined to one area.

```
feat(workspace): add per-workspace YAML config loader
fix(fts): correct SQL placeholder numbering in grep query
test(recall): add regression for who-endpoint entity resolution
docs(api): document X-RateLimit-* response headers
chore(ci): skip non-Elixir paths in test matrix
refactor(topology): rename Workspace module to Topology
```

Scopes in use: `topology`, `intake`, `signal`, `memory-core`, `retrieval`,
`pool`, `workflow`, `skill`, `governance`, `connector`, `asset`, `wiki`,
`export`, `api`, `auth`, `cli`, `sdk`, `mcp`, `docs`, `ci`, `deploy`.

---

## Test Requirements

- All existing tests must continue to pass — no regressions.
- New public functions need unit tests.
- New API endpoints need integration tests (see `test/optimal_engine/api/` for examples).
- Target 80%+ line coverage on new modules. Run `mix test --cover` to check.
- Edge cases to cover: empty inputs, missing workspace, permission boundary violations, concurrent writes.

---

## Architecture Overview

Optimal Engine is organized by lifecycle ownership, not by where bytes happen
to live.

```text
Tenant / Organization
  -> Workspaces
      -> Nodes
          -> Sources, Signals, Claims, Facts, Memories, Workflows, Skills
```

The runtime layers are:

```text
Workspace / Topology
Source Intake
Signal Pipeline
Memory Core
Retrieval / Context
Active Memory Pools
Workflow / Skill Runtime
Tool / Model Governance
Wiki / Export Surface
Evaluation / Audit / Recovery
```

The storage rule is:

```text
database rows own governed runtime state
raw artifacts own source evidence
indexes/caches speed up retrieval
markdown/wiki/API/app views project state
```

Full architecture documentation lives in `docs/architecture/`. Read
`docs/architecture/ENGINE-STRUCTURE.md`,
`docs/architecture/STORAGE-AND-PROJECTION-MAP.md`, and
`docs/reference/backend-readiness.md` before making changes to lifecycle,
storage, retrieval, or projection behavior.

Key modules:

| Module | Responsibility |
|--------|---------------|
| `OptimalEngine.WorkspaceTopology` | Workspaces, Nodes, Node Types, relationships, membership, and topology review. |
| `OptimalEngine.WorkspaceInitiation` | Messy setup dump to workspace topology candidates and projections. |
| `OptimalEngine.Pipeline` | Intake/classification/indexing compatibility pipeline. |
| `OptimalEngine.Signal` | Signal classification and dispatch surfaces. |
| `OptimalEngine.MemoryCore` | Source Packages, Claims, Facts, Memory Objects, Relationship Edges, context, pools, workflows, and skills. |
| `OptimalEngine.MemoryCore.RetrievalCoordinator` | Governed Context Package assembly and refresh. |
| `OptimalEngine.Connectors` | Connector registry, governed execution, and payload preservation. |
| `OptimalEngine.Wiki` | Wiki/export projection service over governed engine state. |
| `OptimalEngine.API.Router` | Plug.Router HTTP entry point |

---

## Filing Issues

Use GitHub Issues. Tag with one of: `bug`, `enhancement`, `documentation`, `question`, `connector`.

For bugs: include the Elixir/OTP version, the steps to reproduce, and the full error output (sanitize any sensitive data before pasting).

For connector requests: name the source system and describe what signals it produces.

---

## Code of Conduct

This project follows a standard contributor code of conduct. Be direct. Be respectful. Assume good intent. Harassment of any kind is not tolerated.
