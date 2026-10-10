---
name: optimal-engine
description: |
  Optimal Engine is a self-hosted second brain and operating engine for human and
  AI workspaces. It preserves messy sources, organizes them into workspaces and
  Nodes, classifies Signals, turns extracted assertions into reviewable Claims,
  promotes accepted Claims into source-backed Facts and Memory Objects, assembles
  governed Context Packages, and projects the same state into markdown, wiki/API
  views, CLIs, MCP tools, and app surfaces. Use this skill when building or using
  persistent organizational memory, workspace topology, source-backed retrieval,
  agent work loops, multimodal intake, packages/exports, or governed tool/model
  execution. Architectural commitments: source-first evidence, workspace/Node
  scope, Signal = Mode + Genre + Type + Format + Structure, Claim -> Fact gate,
  Memory Core lineage, Context Packages, Active Memory Pools, Skill Packages,
  and projection surfaces over canonical runtime state. Self-hosted, MIT, runs
  locally first.
triggers:
  - persistent memory
  - organizational memory
  - memory across sessions
  - agent should remember
  - knowledge base
  - second brain
  - wiki for the team
  - curated context
  - retrieval augmented generation
  - workspace knowledge
  - signal classification
  - source-backed memory
  - workspace topology
  - context packages
  - active memory pool
---

# Optimal Engine Skill

## What Optimal Engine Is

Optimal Engine is a **second brain and operating engine** for a person, team, or
company. It is the backend runtime underneath markdown workspaces, agent
sessions, retrieval, workflows, packages, integrations, and app surfaces.

It is not just a wiki, vector database, desktop UI, or agent task runner. It is
the system that decides where information belongs, what can become truth, what
context an agent is allowed to see, and which projection should be rendered for a
human or application.

**Core promises:**

1. **Source integrity.** Preserve raw evidence before interpretation.
2. **Topology discipline.** Work happens inside tenant, workspace, Node, and pool scope.
3. **Signal clarity.** Classify noisy input as `Mode + Genre + Type + Format + Structure`.
4. **Truth gate.** Observations and extracted assertions become Claims first; accepted truth becomes Facts through review or policy.
5. **Governed context.** Humans and agents receive Context Packages, not random chunks.
6. **Projection discipline.** Markdown/wiki/API/app views are projections over governed runtime state.

**Hard invariants:**

- No durable Fact without source lineage.
- No cross-workspace recall without an authorization envelope.
- No agent observation becomes truth silently.
- No connector/tool/model output bypasses Source Package, Claim, audit, or governance when it affects memory.
- No package/export is written outside the owning workspace/Node projection unless it is a receiver-level export job.

---

## When to Use This Skill

Activate when the user says or implies:

- "I need persistent context across sessions"
- "My agent should remember what happened last week"
- "I want a knowledge base the whole team can query"
- "Build me a second brain / company wiki"
- "I need memory that survives conversation resets"
- "My agent needs to know who owns X / what Y decided"
- "I want to search all our documents semantically"
- "I want RAG with citations I can trust"
- "Multi-tenant / workspace-isolated knowledge"
- "Help me organize a messy company or personal workspace"
- "My agent needs authorized context before doing work"
- "Turn repeated work into workflows or skills"
- "Connect tools, CLIs, MCP servers, or scripts without bypassing governance"

---

## Core Capabilities

### 1. Workspace / Topology

Create an organization/workspace, define Nodes, attach aliases, set
relationships, and keep projects as one Node type inside a workspace.

```bash
mix optimal.setup company-os --name "Company OS"
mix optimal.initiate company-os --name "Company OS" --dump setup.md
mix optimal.topology --workspace default:company-os
```

### 2. Source-First Memory Core

Preserve source evidence, extract Claims, review/promote Facts, build Memory
Objects, and keep derivation lineage.

```bash
# Ingest a workspace projection or source dump
mix optimal.ingest_workspace sample-workspace

# Ask through governed retrieval
mix optimal.rag "what changed before the weekly review?"
```

Truth lifecycle:

```text
Source Package -> Signal -> Claim -> Fact -> Memory Object
```

### 3. Retrieval / Context Packages

Retrieve scoped context with evidence links and authorization metadata.

```bash
# Keyword + semantic search
GET /api/search?q=pricing+negotiation&workspace=sales

# Contextual answer
POST /api/rag  {"query": "what's the pricing decision?", "workspace": "sales", "format": "markdown"}

# Grep-style inspection
GET /api/grep?q=pricing&workspace=sales&intent=record_fact&scale=paragraph
```

### 4. Agent Work Loops

Agents should load a Context Package, work inside an Active Memory Pool, call
governed tools, record observations, and promote only reviewed knowledge.

```text
Task -> Context Package -> Active Memory Pool -> Tool/model calls
     -> Observations -> pending Claims -> reviewed Facts
```

### 5. Projections And Packages

Markdown/wiki/API/app views are projections. Packages are receiver/channel
bundles, usually owned by a Node, such as proposals, contracts, SOP bundles,
requirements packets, reports, or handoff bundles.

---

## Quick Integration Examples

### TypeScript — Vercel AI SDK

```typescript
import Anthropic from '@anthropic-ai/sdk';

const ENGINE = process.env.OPTIMAL_ENGINE_URL ?? 'http://localhost:4200';

async function getContext(query: string, workspace: string): Promise<string> {
  const res = await fetch(`${ENGINE}/api/rag`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, workspace, format: 'claude', bandwidth: 'medium' }),
  });
  const data = await res.json() as { answer?: string; context?: string };
  return data.answer ?? data.context ?? '';
}

// Use with Vercel AI SDK
import { streamText } from 'ai';
import { anthropic } from '@ai-sdk/anthropic';

export async function POST(req: Request) {
  const { messages, workspace = 'default' } = await req.json();
  const lastMessage = messages[messages.length - 1].content as string;
  const memory = await getContext(lastMessage, workspace);

  return streamText({
    model: anthropic('claude-opus-4-5'),
    system: `You have access to organizational memory:\n\n${memory}`,
    messages,
  }).toDataStreamResponse();
}
```

### Python — OpenAI Agents SDK

```python
import os, requests
from agents import Agent, Runner, function_tool

ENGINE = os.getenv("OPTIMAL_ENGINE_URL", "http://localhost:4200")

@function_tool
def recall(query: str, workspace: str = "default") -> str:
    """Retrieve organizational memory for a query."""
    r = requests.post(f"{ENGINE}/api/rag", json={
        "query": query,
        "workspace": workspace,
        "format": "markdown",
        "bandwidth": "medium",
    })
    r.raise_for_status()
    return r.json().get("answer", "")

agent = Agent(
    name="Company Brain",
    instructions="You are a company assistant. Use recall() before answering any question about company decisions, people, or projects.",
    tools=[recall],
)

result = Runner.run_sync(agent, "What did Alice decide about Q4 pricing?")
print(result.final_output)
```

### MCP — Claude Desktop / Cursor / Windsurf

```json
{
  "mcpServers": {
    "optimal-engine": {
      "command": "npx",
      "args": ["-y", "@optimal-engine/mcp-server"],
      "env": {
        "OPTIMAL_ENGINE_URL": "http://localhost:4200",
        "OPTIMAL_WORKSPACE": "default"
      }
    }
  }
}
```

### curl — raw HTTP

```bash
ENGINE=http://localhost:4200

# Ask the wiki
curl -s -X POST $ENGINE/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query":"current pricing strategy","workspace":"sales","format":"markdown"}' \
  | jq '.answer'

# Store a memory
curl -s -X POST $ENGINE/api/memory \
  -H 'Content-Type: application/json' \
  -d '{"content":"Alice confirmed $2K/seat pricing on 2026-04-28","workspace":"sales"}' \
  | jq '.id'

# Recall who owns something
curl -s "$ENGINE/api/recall/who?topic=pricing&workspace=sales" | jq '.answer'
```

---

## Key Concepts Cheat Sheet

| Concept | What it is | Why it matters |
|---|---|---|
| **Organization / Tenant** | Governance, identity, credential, and policy boundary | Prevents cross-company bleed |
| **Workspace** | Bounded operating area inside an organization | A company OS, personal OS, team OS, client workspace, or research workspace |
| **Node** | Governed unit of context, purpose, relationships, state, decisions, and lifecycle | A project, person, product, operation, company, learning area, or custom type |
| **Project Node** | Optional bounded initiative Node | Projects live inside workspaces; they are not peers of workspaces |
| **Source Package** | Preserved raw evidence | The engine can always prove where a Claim came from |
| **Signal** | `Mode + Genre + Type + Format + Structure` classification | Routes noisy input without flattening everything into notes |
| **Claim** | Reviewable assertion extracted from evidence or observation | Prevents generated output from becoming truth automatically |
| **Fact** | Accepted, source-backed truth | Used by retrieval, memory, packages, and projections |
| **Memory Object** | Durable institutional memory built from Facts and evidence | Makes accepted knowledge reusable |
| **Context Package** | Authorized task/query context envelope | Agents get scoped context, not random files |
| **Active Memory Pool** | Task-scoped working state for humans and agents | Captures observations and keeps loops auditable |
| **Skill Package** | Reviewed repeatable procedure | Turns repeated work into reusable agent/human workflows |
| **Package / Export** | Receiver/channel bundle or rendered projection | Keeps proposals, SOP bundles, reports, and handoffs organized |

---

## References

| Doc | Contents |
|---|---|
| [`references/api-reference.md`](references/api-reference.md) | Every endpoint — method, path, params, response, use-when |
| [`references/concepts.md`](references/concepts.md) | Legacy concept reference; prefer the current docs tree for architecture-critical work |
| [`references/workspace-pattern.md`](references/workspace-pattern.md) | Multi-workspace design, config schema, isolation guarantees |
| [`references/memory-pattern.md`](references/memory-pattern.md) | When to add a memory, 5 relation types, versioning, forgetting |
| [`references/retrieval-pattern.md`](references/retrieval-pattern.md) | Decision tree: ask vs search vs grep vs recall vs profile |
| [`references/integration-examples.md`](references/integration-examples.md) | TypeScript, Python, MCP, curl — complete runnable examples |
| [`scripts/bootstrap.sh`](scripts/bootstrap.sh) | Scaffold a workspace and verify the pipeline |
