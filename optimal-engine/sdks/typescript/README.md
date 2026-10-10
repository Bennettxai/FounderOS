# @optimal-engine/client

TypeScript client for the [Optimal Engine](https://github.com/OptimalEngine/OptimalEngine) — open-source memory infrastructure for AI agents.

- Native `fetch` — works in Node 20+, browsers, Cloudflare Workers, Deno
- Strict TypeScript with branded ID types and `exactOptionalPropertyTypes`
- Dual ESM + CJS output, fully tree-shakable
- Vercel AI SDK (v6) adapter included
- OpenAI Agents SDK adapter included
- Zero runtime dependencies (peer deps only: `ai`, `zod`, `openai` — all optional)

## Install

```bash
npm install @optimal-engine/client
# peer deps — install only what you use
npm install ai zod        # for Vercel AI SDK adapter
npm install openai zod    # for OpenAI Agents adapter
```

## Quickstart

```typescript
import { OptimalEngine } from "@optimal-engine/client";

const client = new OptimalEngine({
  baseUrl: "http://localhost:4200", // default
  workspace: "my-workspace",        // applied to every call unless overridden
  apiKey: process.env.ENGINE_KEY,   // optional — future-proofing
});

// Ask the second brain
const result = await client.ask("What did we decide about the auth flow?", {
  bandwidth: "l1",   // l0 = headline, l1 = summary, full = complete
});
console.log(result.answer);

// Store a memory
const mem = await client.memory.create({
  content: "We decided to use refresh-token rotation on 2025-04-28.",
  citationUri: "https://notion.so/decision-log#auth",
});

// Retrieve by id
const fetched = await client.memory.get(mem.id);

// Forget it (soft-delete, audit trail preserved)
await client.memory.forget(mem.id, { reason: "superseded by ADR-12" });

// Recall past decisions on a topic
const decisions = await client.recall.actions({
  topic: "auth",
  since: "2025-01-01T00:00:00Z",
});

// Search
const hits = await client.search("refresh token rotation", { limit: 5 });

// Profile
const profile = await client.profile({ bandwidth: "full" });
```

## API Reference

### `new OptimalEngine(config?)`

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `baseUrl` | `string` | `http://localhost:4200` | Engine base URL |
| `apiKey` | `string` | — | Sent as `Bearer` token when provided |
| `workspace` | `string` | — | Default workspace for all calls |

### Retrieval

```typescript
await client.ask(query, { workspace?, audience?, format?, bandwidth? })
await client.search(query, { workspace?, limit? })
await client.grep(query, { workspace?, intent?, scale?, modality?, limit?, literal? })
await client.profile({ workspace?, audience?, bandwidth?, node? })
```

### Memory

```typescript
await client.memory.create({ content, workspace?, isStatic?, audience?, citationUri?, metadata? })
await client.memory.get(id)
await client.memory.list({ workspace?, audience?, includeForgotten?, limit? })
await client.memory.forget(id, { reason?, forgetAfter? })
await client.memory.update(id, { content, ...rest })
await client.memory.extend(id, { content, ...rest })
await client.memory.derive(id, { content, ...rest })
await client.memory.versions(id)
await client.memory.relations(id)
await client.memory.delete(id)
```

### Recall

```typescript
await client.recall.actions({ topic, actor?, since?, workspace? })
await client.recall.who({ topic, role?, workspace? })
await client.recall.when({ event, workspace? })
await client.recall.where({ thing, workspace? })
await client.recall.owns({ actor, workspace? })
```

### Workspaces

```typescript
await client.workspaces.list({ tenant? })
await client.workspaces.create({ slug, name, description?, tenant? })
await client.workspaces.get(id)
await client.workspaces.config(id)
await client.workspaces.updateConfig(id, patch)
```

### Wiki

```typescript
await client.wiki.list({ workspace? })
await client.wiki.get(slug, { workspace?, audience?, format? })
await client.wiki.contradictions({ workspace? })
```

### Subscriptions & SSE surface

```typescript
// Create and list subscriptions
await client.subscriptions.create({ workspace, scope, scopeValue?, categories? })
await client.subscriptions.list({ workspace? })

// Open a live SSE stream
const stream = client.surface.stream(subscriptionId)
stream
  .on((event) => console.log(event.type, event.data))
  .onError((err) => console.error(err))

// Disconnect
stream.close()
```

### System

```typescript
await client.status()        // { status: "ok" | "degraded" | "down", version? }
await client.architectures() // list of registered architectures
```

## Vercel AI SDK Integration

```typescript
import { OptimalEngine } from "@optimal-engine/client";
import { optimalEngineTools } from "@optimal-engine/client/adapters/ai-sdk";
import { generateText } from "ai";
import { openai } from "@ai-sdk/openai";

const engine = new OptimalEngine({
  baseUrl: "http://localhost:4200",
  workspace: "sample",
});

const { text } = await generateText({
  model: openai("gpt-4o"),
  tools: optimalEngineTools(engine),
  maxSteps: 5,
  prompt: "What did the team decide about the database migration last quarter?",
});

console.log(text);
```

Available tools:

| Tool | Description |
|------|-------------|
| `askEngine` | Ask the second brain. Curated wiki first, hybrid search second. |
| `searchMemory` | Hybrid semantic + keyword search across memory. |
| `grepMemory` | Structured grep with intent, scale, and modality filters. |
| `recallActions` | Recover past actions / decisions / commitments by topic. |
| `recallWho` | Find who owns / is accountable for a topic. |
| `addMemory` | Add a fact or decision to long-term memory. |
| `forgetMemory` | Soft-forget a memory (audit trail preserved). |
| `getProfile` | Get the 4-tier workspace profile. |

## OpenAI Agents SDK Integration

```typescript
import { OptimalEngine } from "@optimal-engine/client";
import { optimalEngineAgentTools } from "@optimal-engine/client/adapters/openai-agents";

const engine = new OptimalEngine({ workspace: "sample" });
const tools = optimalEngineAgentTools(engine);

// Pass `tools` to your Agent constructor
```

## Error Handling

Non-2xx responses throw `OptimalEngineError`:

```typescript
import { OptimalEngineError } from "@optimal-engine/client";

try {
  await client.memory.get(id);
} catch (err) {
  if (err instanceof OptimalEngineError) {
    console.error(err.status, err.code, err.message);
  }
}
```

## Bandwidth Levels

| Level | Description |
|-------|-------------|
| `l0` | Headline only — minimal tokens |
| `l1` | Summary — balanced |
| `full` | Complete envelope with all tiers and citations |
