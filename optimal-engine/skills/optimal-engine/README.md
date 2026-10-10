# Optimal Engine Skill

Claude Code skill for building with [Optimal Engine](../../README.md) — a
backend operating engine for source-backed workspace memory, agent context, and
projection surfaces.

## What This Skill Does

When activated, this skill gives Claude Code full working knowledge of:
- Every HTTP API endpoint (method, params, response shape)
- The workspace topology, source-first intake, memory lifecycle, and projection model
- When to use `ask` vs `search` vs `grep` vs `recall` vs `profile`
- Memory primitive: versioning, relations, forgetting
- Workspace isolation and config schema
- Runnable integration examples in TypeScript, Python, MCP, and curl

## Activation

The skill auto-activates when you say things like:
- "I need persistent memory for my agent"
- "Build a second brain for our team"
- "Set up RAG with citations"
- "My agent should remember what happened last week"
- "Knowledge base", "organizational memory", "workspace knowledge"

Or reference it directly with `@optimal-engine`.

## Contents

```
skills/optimal-engine/
├── SKILL.md                    Manifest (Claude Code reads this)
├── README.md                   This file
├── references/
│   ├── api-reference.md        Every endpoint with params + examples
│   ├── concepts.md             tiers, layers, stages, intents, Mode + Genre + Type + Format + Structure
│   ├── workspace-pattern.md    Multi-workspace design + config schema
│   ├── memory-pattern.md       Versioning, relations, forgetting
│   ├── retrieval-pattern.md    Decision tree: which endpoint to use
│   └── integration-examples.md TypeScript, Python, MCP, curl, Elixir
└── scripts/
    └── bootstrap.sh            Scaffold + smoke-test a workspace
```

## Quick Start

```bash
# 1. Start the engine
make install
make bootstrap
make dev

# 2. Bootstrap an API workspace
bash skills/optimal-engine/scripts/bootstrap.sh my-workspace

# 3. Optional: scaffold a markdown-operable workspace
mix optimal.init ~/my-workspace-files

# 4. Edit the generated files, then ingest your knowledge
mix optimal.ingest_workspace ~/my-workspace-files

# 5. Query the workspace
curl -X POST http://localhost:4200/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query":"what do we know?","workspace":"my-workspace","format":"markdown"}'
```

`make dev` starts the HTTP engine on `http://localhost:4200` and creates a local connector key in `.optimal/connector_key` if one is not already set.
The `.optimal/` directory is local runtime state and is ignored by git.

The skill should preserve all core architecture terms: company second brain,
tiers, layers, stages, rhythm, Nodes, Signals, wiki/curation, retrieval, and
agent operation. Do not flatten those into a generic RAG or task-runner model.

## Key References

- Engine README: [`../../README.md`](../../README.md)
- Architecture: [`../../docs/architecture/ENGINE-STRUCTURE.md`](../../docs/architecture/ENGINE-STRUCTURE.md)
- Storage/projections: [`../../docs/architecture/STORAGE-AND-PROJECTION-MAP.md`](../../docs/architecture/STORAGE-AND-PROJECTION-MAP.md)
- Signal Theory: [`../../docs/concepts/signal-theory.md`](../../docs/concepts/signal-theory.md)
- Mix tasks: [`../../docs/guides/mix-tasks.md`](../../docs/guides/mix-tasks.md)
