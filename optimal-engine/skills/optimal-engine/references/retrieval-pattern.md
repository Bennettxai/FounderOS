# Retrieval Pattern

Optimal Engine exposes five retrieval surfaces. Choosing the right one changes both latency and answer quality. This document is a decision guide.

---

## Decision Tree

```
Is the user asking an open-ended question about organizational knowledge?
  └── YES → use /api/rag (ask)

Does the user want to find specific documents or signals?
  └── YES → use /api/search

Does the user want to inspect chunks at a specific scale/intent/modality?
  └── YES → use /api/grep

Does the user want a typed memory-failure query (who/what/when/where/owns)?
  └── YES → use /api/recall/:type

Does the user want a full contextual snapshot to seed an agent system prompt?
  └── YES → use /api/profile
```

---

## /api/rag — Ask With Governed Context

**When:** any natural-language question where you want the best possible answer from organizational memory.

**How it works:**
1. Resolve actor, tenant, workspace, and receiver scope.
2. Decode the query intent, entities, and temporal scope.
3. Assemble governed context from Memory Core state, search/index projections,
   and wiki/export projections where useful.
4. Apply authorization and filtering before delivery.
5. Format the resulting Context Package for the receiver/model.

```bash
# Markdown output (good for humans + generic LLMs)
curl -X POST http://localhost:4200/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query":"what is the current pricing strategy?","workspace":"sales","format":"markdown","audience":"sales"}'

# Claude-optimized output (system prompt ready)
curl -X POST http://localhost:4200/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query":"what is the current pricing strategy?","workspace":"sales","format":"claude","bandwidth":"medium"}'

# OpenAI messages format
curl -X POST http://localhost:4200/api/rag \
  -H 'Content-Type: application/json' \
  -d '{"query":"pricing strategy","workspace":"sales","format":"openai"}'
```

**Format options:**

| `format` | What you get |
|---|---|
| `markdown` | Answer in markdown with citations |
| `text` | Plain text, no markup |
| `claude` | System prompt string ready to pass as `system:` |
| `openai` | `messages` array ready for OpenAI `/v1/chat/completions` |
| `json` | Structured envelope with `answer`, `sources`, `wiki_hit` |

**Bandwidth options:**

| `bandwidth` | Approximate tokens | Use when |
|---|---|---|
| `l0` | ~100 | Micro-summary for tight budgets |
| `medium` | ~2,000 | Standard agent context |
| `full` | Up to budget | Deep analysis, no compression |

---

## /api/search — Find Documents

**When:** the user wants to locate signals or documents by keyword or semantic similarity. Returns context-level metadata (not chunks).

```bash
# Basic hybrid search
curl 'http://localhost:4200/api/search?q=pricing+negotiation&workspace=sales&limit=10'

# Response shape
# {
#   "query": "pricing negotiation",
#   "results": [
#     { "id": "ctx:abc", "title": "Customer pricing call Q4", "slug": "sales/2026-04-28-pricing-call",
#       "score": 0.91, "snippet": "Alice confirmed $2K/seat...", "node": "03-sales" }
#   ]
# }
```

**Retrieval mechanics:** BM25 over FTS5 + vector cosine + graph_boost (connected entities score higher) + temporal_decay (recent signals score higher). Same pipeline as the Tier 2 fallback in `/api/rag`.

---

## /api/grep — Chunk-Level Inspection

**When:** you need chunk-level matches with full signal trace — scale, intent, sn_ratio, modality. Use this for:
- Debugging why a retrieval returned unexpected results
- Building filtered context (e.g., "only `record_fact` chunks from the `paragraph` scale")
- Auditing intent distribution across a topic
- Power-user search with explicit filters

```bash
# All record_fact chunks about pricing at paragraph scale
curl 'http://localhost:4200/api/grep?q=pricing&workspace=sales&intent=record_fact&scale=paragraph'

# Literal FTS match only (no vector)
curl 'http://localhost:4200/api/grep?q="$2000 per seat"&workspace=sales&literal=true'

# Restrict to a specific node
curl 'http://localhost:4200/api/grep?q=pricing&workspace=sales&path=03-sales'

# Response shape per result:
# { "slug": "sales/signals/2026-04-28-call", "scale": "paragraph",
#   "intent": "record_fact", "sn_ratio": 0.82, "modality": "text",
#   "snippet": "Alice confirmed $2K/seat/year...", "score": 0.88 }
```

**Intent filter values:** `request_info` / `propose_decision` / `record_fact` / `express_concern` / `commit_action` / `reference` / `narrate` / `reflect` / `specify` / `measure`

**Scale filter values:** `document` / `section` / `paragraph` / `chunk`

---

## /api/recall/:type — Typed Cued Recall

**When:** the question has a known memory-failure pattern. These endpoints build an intent-optimized query internally and route through the same `/api/rag` pipeline. The benefit: `IntentAnalyzer` decodes intent with maximum confidence, and the retrieval boost picks chunks that match the intent type directly.

```bash
# What did Alice decide about pricing after January?
curl 'http://localhost:4200/api/recall/actions?actor=Alice&topic=pricing&since=2026-01-01&workspace=sales'

# Who owns the pricing negotiation?
curl 'http://localhost:4200/api/recall/who?topic=pricing&role=owner&workspace=sales'

# When is the Q4 board review?
curl 'http://localhost:4200/api/recall/when?event=Q4+board+review&workspace=default'

# Where is the pricing deck kept?
curl 'http://localhost:4200/api/recall/where?thing=pricing+deck&workspace=sales'

# What is Alice currently committed to?
curl 'http://localhost:4200/api/recall/owns?actor=Alice&workspace=sales'
```

All five return the same envelope as `/api/rag` plus a `recall_query` field showing the synthesized query string.

---

## /api/profile — Contextual Snapshot

**When:** you want to seed an agent's system prompt with a full workspace context snapshot before it starts working. Returns all four tiers in one call.

```bash
# Full sales context for an executive
curl 'http://localhost:4200/api/profile?workspace=sales&audience=exec&bandwidth=l1'

# Restrict to one node
curl 'http://localhost:4200/api/profile?workspace=engineering&node=product-customer-portal&bandwidth=full'
```

**Response structure:**

```json
{
  "workspace_id": "sales",
  "tenant_id": "default",
  "audience": "exec",
  "static": "Base price is $2K/seat...",
  "dynamic": "Alice and Bob co-lead pricing. Open: Q4 deal with Example...",
  "curated": "## Requirements Review Strategy\n\nThe team has committed to...\n\n{{cite:...}}",
  "activity": [
    { "kind": "ingest", "signal": "2026-04-28-pricing-call", "at": "2026-04-28T14:00:00Z" }
  ],
  "entities": [
    { "name": "Alice", "type": "person", "connections": 12 }
  ],
  "generated_at": "2026-04-28T15:00:00Z"
}
```

---

## Audience Parameter

The `audience` parameter selects which variant of a wiki page to serve. The curator maintains separate variants per audience tag.

| Audience tag | Typical receiver | What changes |
|---|---|---|
| `default` | General | Standard wiki page |
| `engineering` | Engineers | Technical detail preserved |
| `sales` | Sales team | Customer-facing framing, deal context |
| `exec` | Leadership | Summary-first, metrics prominent |
| `legal` | Legal team | Compliance notes, contract refs surfaced |

If the requested audience variant doesn't exist for a page, the engine falls back to `"default"`.

Configure which audiences the curator maintains in `.optimal/config.yaml`:
```yaml
wiki:
  audiences: [default, engineering, exec, sales, legal]
```

---

## Bandwidth Parameter

Controls context density. Always prefer the lowest bandwidth that answers the question — smaller context = lower LLM cost + faster response.

| `bandwidth` | Token budget | When to use |
|---|---|---|
| `l0` | ~100 tokens | One-liner summary; tight prompt budgets; mobile |
| `l1` / `medium` | ~2,000 tokens | Standard agent turn; most use cases |
| `full` | Up to token limit | Deep analysis; multi-step reasoning; RAG with full doc |

**Rule:** default to `medium`. Only use `full` when the query requires reasoning over complete documents.

---

## Retrieval Method Comparison

| | ask (`/rag`) | search | grep | recall | profile |
|---|---|---|---|---|---|
| Unit returned | Answer + sources | Signals (metadata) | Chunks (with trace) | Answer + sources | 4-tier snapshot |
| Wiki-first | Yes | No | No | Yes | Yes (curated tier) |
| Intent-optimized | Yes (general) | No | Filter only | Yes (typed) | No |
| Audience-aware | Yes | No | No | Yes | Yes |
| Best for | Open questions | Finding docs | Debugging / filtering | Typed queries | System prompt seeding |
| Response size | Bandwidth-controlled | N results | N chunks | Bandwidth-controlled | Bandwidth-controlled |
