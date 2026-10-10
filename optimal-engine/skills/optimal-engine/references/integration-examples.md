# Integration Examples

All examples assume Optimal Engine is running at `http://localhost:4200`. Override with the `OPTIMAL_ENGINE_URL` environment variable.

---

## TypeScript — Vercel AI SDK

```typescript
// lib/optimal-engine.ts
const ENGINE = process.env.OPTIMAL_ENGINE_URL ?? 'http://localhost:4200';

export interface RagResponse {
  answer: string;
  sources: Array<{ slug: string; score: number; snippet: string }>;
  wiki_hit: boolean;
  citations: string[];
}

export async function ask(
  query: string,
  options: {
    workspace?: string;
    audience?: string;
    bandwidth?: 'l0' | 'medium' | 'full';
    format?: 'markdown' | 'claude' | 'openai' | 'json';
  } = {}
): Promise<RagResponse> {
  const res = await fetch(`${ENGINE}/api/rag`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query,
      workspace: options.workspace ?? 'default',
      audience: options.audience ?? 'default',
      bandwidth: options.bandwidth ?? 'medium',
      format: options.format ?? 'json',
    }),
  });
  if (!res.ok) throw new Error(`Optimal Engine error: ${res.status}`);
  return res.json() as Promise<RagResponse>;
}

export async function storeMemory(
  content: string,
  options: { workspace?: string; isStatic?: boolean; audience?: string } = {}
): Promise<{ id: string }> {
  const res = await fetch(`${ENGINE}/api/memory`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      content,
      workspace: options.workspace ?? 'default',
      is_static: options.isStatic ?? false,
      audience: options.audience,
    }),
  });
  if (!res.ok) throw new Error(`Memory store error: ${res.status}`);
  return res.json() as Promise<{ id: string }>;
}
```

```typescript
// app/api/chat/route.ts — Vercel AI SDK streaming endpoint
import { streamText } from 'ai';
import { anthropic } from '@ai-sdk/anthropic';
import { ask } from '@/lib/optimal-engine';

export async function POST(req: Request) {
  const { messages, workspace = 'default', audience = 'default' } = await req.json();
  const lastMessage = messages[messages.length - 1].content as string;

  // Fetch organizational context before streaming
  const memory = await ask(lastMessage, { workspace, audience, format: 'claude' });

  return streamText({
    model: anthropic('claude-opus-4-5'),
    system: `You have access to organizational memory. Use it to answer accurately.\n\n${memory.answer}`,
    messages,
    onFinish: async ({ text }) => {
      // Persist the agent's conclusion as a dynamic memory
      await storeMemory(text, { workspace });
    },
  }).toDataStreamResponse();
}
```

---

## TypeScript — Mastra

```typescript
// src/mastra/tools/optimal-engine.ts
import { createTool } from '@mastra/core/tools';
import { z } from 'zod';

const ENGINE = process.env.OPTIMAL_ENGINE_URL ?? 'http://localhost:4200';

export const recallTool = createTool({
  id: 'recall',
  description: 'Retrieve organizational memory for a query. Use before answering any question about company knowledge.',
  inputSchema: z.object({
    query: z.string().describe('Natural-language question'),
    workspace: z.string().default('default').describe('Workspace to query'),
    audience: z.string().default('default').describe('Audience tag (sales, engineering, exec, legal)'),
  }),
  outputSchema: z.object({
    answer: z.string(),
    wiki_hit: z.boolean(),
    sources: z.array(z.object({ slug: z.string(), score: z.number() })),
  }),
  execute: async ({ context }) => {
    const res = await fetch(`${ENGINE}/api/rag`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: context.query,
        workspace: context.workspace,
        audience: context.audience,
        format: 'json',
        bandwidth: 'medium',
      }),
    });
    return res.json();
  },
});

export const rememberTool = createTool({
  id: 'remember',
  description: 'Store a fact or observation in organizational memory for future retrieval.',
  inputSchema: z.object({
    content: z.string().describe('The fact or observation to store'),
    workspace: z.string().default('default'),
    isStatic: z.boolean().default(false).describe('True for stable facts, false for ephemeral observations'),
  }),
  outputSchema: z.object({ id: z.string() }),
  execute: async ({ context }) => {
    const res = await fetch(`${ENGINE}/api/memory`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: context.content,
        workspace: context.workspace,
        is_static: context.isStatic,
      }),
    });
    return res.json();
  },
});
```

```typescript
// src/mastra/agents/company-brain.ts
import { Agent } from '@mastra/core/agent';
import { anthropic } from '@ai-sdk/anthropic';
import { recallTool, rememberTool } from '../tools/optimal-engine';

export const companyBrainAgent = new Agent({
  name: 'Company Brain',
  instructions: `You are a company knowledge assistant with access to organizational memory.
ALWAYS call recall() before answering any question about company decisions, people, projects, or history.
ALWAYS call remember() to store important facts or conclusions you derive during the conversation.`,
  model: anthropic('claude-opus-4-5'),
  tools: { recall: recallTool, remember: rememberTool },
});
```

---

## Python — OpenAI Agents SDK

```python
# agents/company_brain.py
import os
import requests
from agents import Agent, Runner, function_tool

ENGINE = os.getenv("OPTIMAL_ENGINE_URL", "http://localhost:4200")


@function_tool
def recall(query: str, workspace: str = "default", audience: str = "default") -> str:
    """Retrieve organizational memory for a natural-language question.

    Always call this before answering questions about company knowledge,
    decisions, people, or projects.
    """
    r = requests.post(
        f"{ENGINE}/api/rag",
        json={"query": query, "workspace": workspace, "audience": audience,
              "format": "markdown", "bandwidth": "medium"},
        timeout=30,
    )
    r.raise_for_status()
    data = r.json()
    return data.get("answer", "No relevant memory found.")


@function_tool
def remember(content: str, workspace: str = "default", is_static: bool = False) -> str:
    """Store a fact or observation in organizational memory for future retrieval.

    Use is_static=True for stable facts (decisions, definitions).
    Use is_static=False for working observations (call notes, session context).
    Returns the memory ID.
    """
    r = requests.post(
        f"{ENGINE}/api/memory",
        json={"content": content, "workspace": workspace, "is_static": is_static},
        timeout=10,
    )
    r.raise_for_status()
    return r.json()["id"]


@function_tool
def who_owns(topic: str, workspace: str = "default") -> str:
    """Look up who owns or leads a topic in the organization."""
    r = requests.get(
        f"{ENGINE}/api/recall/who",
        params={"topic": topic, "workspace": workspace},
        timeout=30,
    )
    r.raise_for_status()
    return r.json().get("answer", "Unknown owner.")


agent = Agent(
    name="Company Brain",
    instructions=(
        "You are a company knowledge assistant. "
        "ALWAYS call recall() before answering questions about company knowledge. "
        "ALWAYS call remember() to persist important facts or conclusions."
    ),
    tools=[recall, remember, who_owns],
)

if __name__ == "__main__":
    result = Runner.run_sync(
        agent,
        "What did we decide about Q4 pricing, and who is responsible for closing it?"
    )
    print(result.final_output)
```

---

## Python — LangChain Tool

```python
# tools/optimal_engine.py
import os
from typing import Optional
import requests
from langchain_core.tools import BaseTool
from pydantic import BaseModel, Field

ENGINE = os.getenv("OPTIMAL_ENGINE_URL", "http://localhost:4200")


class RecallInput(BaseModel):
    query: str = Field(description="Natural-language question about organizational knowledge")
    workspace: str = Field(default="default", description="Workspace to query")
    audience: str = Field(default="default", description="Audience tag (sales, engineering, exec, legal)")


class OptimalEngineRecallTool(BaseTool):
    name: str = "recall_organizational_memory"
    description: str = (
        "Retrieve organizational memory for a question. "
        "Use this before answering any question about company decisions, people, "
        "projects, history, or knowledge. Returns curated context with citations."
    )
    args_schema: type[BaseModel] = RecallInput

    def _run(self, query: str, workspace: str = "default", audience: str = "default") -> str:
        r = requests.post(
            f"{ENGINE}/api/rag",
            json={"query": query, "workspace": workspace, "audience": audience,
                  "format": "markdown", "bandwidth": "medium"},
            timeout=30,
        )
        r.raise_for_status()
        return r.json().get("answer", "No relevant memory found.")


class RememberInput(BaseModel):
    content: str = Field(description="The fact or observation to store")
    workspace: str = Field(default="default")
    is_static: bool = Field(default=False, description="True for stable facts")


class OptimalEngineRememberTool(BaseTool):
    name: str = "store_organizational_memory"
    description: str = (
        "Store a fact or observation in organizational memory for future retrieval. "
        "Use for important conclusions, decisions, or facts discovered during the session."
    )
    args_schema: type[BaseModel] = RememberInput

    def _run(self, content: str, workspace: str = "default", is_static: bool = False) -> str:
        r = requests.post(
            f"{ENGINE}/api/memory",
            json={"content": content, "workspace": workspace, "is_static": is_static},
            timeout=10,
        )
        r.raise_for_status()
        mem_id = r.json()["id"]
        return f"Memory stored: {mem_id}"


# Usage with LangChain agent
from langchain_anthropic import ChatAnthropic
from langchain.agents import AgentExecutor, create_tool_calling_agent
from langchain_core.prompts import ChatPromptTemplate

tools = [OptimalEngineRecallTool(), OptimalEngineRememberTool()]
llm = ChatAnthropic(model="claude-opus-4-5")

prompt = ChatPromptTemplate.from_messages([
    ("system", "You are a company knowledge assistant. Always recall memory before answering."),
    ("human", "{input}"),
    ("placeholder", "{agent_scratchpad}"),
])

agent = create_tool_calling_agent(llm, tools, prompt)
executor = AgentExecutor(agent=agent, tools=tools, verbose=True)

result = executor.invoke({"input": "Who owns the Q4 pricing negotiation?"})
print(result["output"])
```

---

## MCP — Claude Desktop / Cursor / Windsurf

Add to your MCP config file:

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "optimal-engine": {
      "command": "npx",
      "args": ["-y", "@optimal-engine/mcp-server"],
      "env": {
        "OPTIMAL_ENGINE_URL": "http://localhost:4200",
        "OPTIMAL_WORKSPACE": "default",
        "OPTIMAL_AUDIENCE": "default"
      }
    }
  }
}
```

**Cursor** (`.cursor/mcp.json` in project root or `~/.cursor/mcp.json` globally):

```json
{
  "mcpServers": {
    "optimal-engine": {
      "command": "npx",
      "args": ["-y", "@optimal-engine/mcp-server"],
      "env": {
        "OPTIMAL_ENGINE_URL": "http://localhost:4200",
        "OPTIMAL_WORKSPACE": "engineering"
      }
    }
  }
}
```

**Windsurf** (`.windsurf/mcp_config.json`):

```json
{
  "mcpServers": {
    "optimal-engine": {
      "command": "npx",
      "args": ["-y", "@optimal-engine/mcp-server"],
      "env": {
        "OPTIMAL_ENGINE_URL": "http://localhost:4200"
      }
    }
  }
}
```

The MCP server exposes these tools:
- `recall` — `POST /api/rag`
- `search` — `GET /api/search`
- `remember` — `POST /api/memory`
- `forget` — `POST /api/memory/:id/forget`
- `who_owns` — `GET /api/recall/who`
- `what_decided` — `GET /api/recall/actions`
- `profile` — `GET /api/profile`

---

## curl — Raw HTTP

```bash
ENGINE=http://localhost:4200
WS=sales

# Ask with governed context (markdown output)
curl -s -X POST $ENGINE/api/rag \
  -H 'Content-Type: application/json' \
  -d "{\"query\":\"current pricing strategy\",\"workspace\":\"$WS\",\"format\":\"markdown\"}" \
  | jq '.answer'

# Ask with Claude-optimized output
curl -s -X POST $ENGINE/api/rag \
  -H 'Content-Type: application/json' \
  -d "{\"query\":\"Q4 pricing decision\",\"workspace\":\"$WS\",\"format\":\"claude\"}"

# Hybrid search
curl -s "$ENGINE/api/search?q=pricing+negotiation&workspace=$WS&limit=5" | jq '.results'

# Chunk-level grep (only record_fact chunks)
curl -s "$ENGINE/api/grep?q=pricing&workspace=$WS&intent=record_fact&scale=paragraph" \
  | jq '.results[] | {slug, snippet, sn_ratio}'

# Store a memory
MEM_ID=$(curl -s -X POST $ENGINE/api/memory \
  -H 'Content-Type: application/json' \
  -d "{\"content\":\"Alice confirmed \$2K/seat pricing on 2026-04-28\",\"workspace\":\"$WS\",\"is_static\":true}" \
  | jq -r '.id')
echo "Stored: $MEM_ID"

# Update the memory
curl -s -X POST "$ENGINE/api/memory/$MEM_ID/update" \
  -H 'Content-Type: application/json' \
  -d '{"content":"Alice confirmed $1,900/seat after negotiation on 2026-04-29"}' | jq '.version'

# Typed recall — who owns pricing?
curl -s "$ENGINE/api/recall/who?topic=pricing&workspace=$WS" | jq '.answer'

# Profile snapshot for sales audience
curl -s "$ENGINE/api/profile?workspace=$WS&audience=sales&bandwidth=l1" \
  | jq '{static, dynamic, curated}'

# Create a workspace
curl -s -X POST $ENGINE/api/workspaces \
  -H 'Content-Type: application/json' \
  -d '{"slug":"ma","name":"M&A Brain"}' | jq '.id'

# Subscribe to surfacing events
SUB_ID=$(curl -s -X POST $ENGINE/api/subscriptions \
  -H 'Content-Type: application/json' \
  -d "{\"workspace\":\"$WS\",\"scope\":\"topic\",\"scope_value\":\"pricing\",\"categories\":[\"recent_actions\",\"contradictions\"],\"principal_id\":\"alice\"}" \
  | jq -r '.id')

# Stream surfacing events (blocks; use in background)
curl -s -N "$ENGINE/api/surface/stream?subscription=$SUB_ID"

# Wiki list
curl -s "$ENGINE/api/wiki?workspace=$WS" | jq '.pages[] | .slug'

# Wiki page render
curl -s "$ENGINE/api/wiki/customer portal-pricing-decision?workspace=$WS&audience=sales" \
  | jq '.body'

# Graph hubs
curl -s "$ENGINE/api/graph/hubs" | jq '.hubs[:3]'

# Engine status
curl -s "$ENGINE/api/status" | jq '{status, ok?}'
```

---

## Elixir — In-Process API

When running inside the same BEAM node:

```elixir
# Direct module calls — same code path as HTTP API
alias OptimalEngine

# RAG
{:ok, result} = OptimalEngine.Retrieval.ask(
  "current pricing strategy",
  receiver: OptimalEngine.Retrieval.Receiver.new(%{format: :claude, bandwidth: :medium, audience: "sales"}),
  workspace_id: "sales"
)

# Search
{:ok, contexts} = OptimalEngine.search("pricing negotiation", limit: 10, workspace_id: "sales")

# Memory
{:ok, mem} = OptimalEngine.Memory.create(%{
  content: "Alice owns pricing",
  workspace_id: "sales",
  is_static: true
})

{:ok, mem_v2} = OptimalEngine.Memory.update(mem.id, %{content: "Alice and Bob co-own pricing"})
:ok = OptimalEngine.Memory.forget(mem.id, reason: "resolved")

# Wiki
{:ok, pages} = OptimalEngine.Wiki.list("default", "sales")
{:ok, page} = OptimalEngine.Wiki.latest("default", "customer portal-pricing-decision", "sales", "sales")
```

All four surfaces (CLI, HTTP, Elixir API, Mix tasks) route through the same internal modules. There is no separate code path per surface.
