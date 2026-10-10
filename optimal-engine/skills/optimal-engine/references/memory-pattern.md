# Memory Pattern

The Memory primitive is a first-class, versioned, relation-tracked entry. It is not a key-value store. Every memory carries provenance: who wrote it, what it cites, how it relates to other memories, and whether it has been superseded or forgotten.

---

## When to Add a Memory vs. Ingest a Signal vs. Curate a Wiki Page

| Situation | What to do |
|---|---|
| An agent observes a fact during a session and needs it to persist | `POST /api/memory` |
| A document, transcript, or signal file exists on disk | `mix optimal.ingest --file` or `mix optimal.ingest_workspace` |
| A human has written an authoritative document to add to the knowledge base | Ingest as a signal |
| The engine should maintain a curated summary of a topic | Let Stage 9 (Curator) handle it automatically on ingest |
| An agent wants to add a note to an existing wiki page | `POST /api/memory` with `citation_uri` pointing at the page |
| A fact was true last week but has changed | `POST /api/memory/:id/update` with the new content |
| A fact is a specialization or addendum to an existing memory | `POST /api/memory/:id/extend` |
| A conclusion has been drawn from a set of facts | `POST /api/memory/:id/derive` |

**Rule of thumb:** signals are what happened (append-only source documents). Memories are what was observed or concluded (first-class facts). The wiki is what the LLM curated from both.

---

## The Five Relation Types

Relations create a typed graph over memories. Every relation is directed: `source → target`.

| Relation | Semantics | Use when |
|---|---|---|
| `updates` | Source is a newer version of target | A fact changed; target is now stale |
| `extends` | Source adds detail to target without superseding it | An addendum, clarification, or specialization |
| `derives` | Source is a conclusion drawn from target | A summary, inference, or analysis |
| `contradicts` | Source conflicts with target | Explicitly flagging a known conflict |
| `cites` | Source references target as evidence | Attribution without version or derivation semantics |

```bash
# Create the original memory
ORIGINAL=$(curl -s -X POST http://localhost:4200/api/memory \
  -H 'Content-Type: application/json' \
  -d '{"content": "Alice owns the pricing negotiation", "workspace": "sales"}' | jq -r '.id')

# Update it (relation: updates)
curl -s -X POST "http://localhost:4200/api/memory/$ORIGINAL/update" \
  -H 'Content-Type: application/json' \
  -d '{"content": "Alice and Bob co-own the pricing negotiation as of 2026-04-28"}'

# Extend it (relation: extends)
curl -s -X POST "http://localhost:4200/api/memory/$ORIGINAL/extend" \
  -H 'Content-Type: application/json' \
  -d '{"content": "Bob is the fallback contact when Alice is OOO"}'

# Derive from it (relation: derives)
curl -s -X POST "http://localhost:4200/api/memory/$ORIGINAL/derive" \
  -H 'Content-Type: application/json' \
  -d '{"content": "Pricing negotiations have dual ownership — escalate to both Alice and Bob"}'
```

---

## Versioning Model

Every `update` call creates a new memory entry and links it to the original via the `updates` relation. The version chain fields:

| Field | Meaning |
|---|---|
| `version` | Monotonically increasing integer (1, 2, 3, ...) |
| `parent_memory_id` | ID of the immediately preceding version |
| `root_memory_id` | ID of the first version in the chain |
| `is_latest` | `true` on the newest version only |

When you call `GET /api/memory` (list), by default only `is_latest: true` entries are returned. Pass `include_old_versions=true` to see the full chain.

When you call `GET /api/memory/:id/versions`, you get the full chain sorted chronologically:

```json
{
  "memory_id": "mem:abc123",
  "root_id": "mem:root000",
  "versions": [
    { "id": "mem:root000", "version": 1, "content": "...", "is_latest": false },
    { "id": "mem:ver001",  "version": 2, "content": "...", "is_latest": false },
    { "id": "mem:abc123",  "version": 3, "content": "...", "is_latest": true }
  ]
}
```

---

## Forgetting

Forgetting is soft by default. The memory is marked `is_forgotten: true` but the data is preserved for audit and GDPR-erasure workflows.

```bash
# Soft forget immediately
curl -X POST http://localhost:4200/api/memory/mem:abc123/forget \
  -H 'Content-Type: application/json' \
  -d '{"reason": "decision reversed"}'

# Schedule future forgetting (GDPR Art. 17 erasure workflow)
curl -X POST http://localhost:4200/api/memory/mem:abc123/forget \
  -H 'Content-Type: application/json' \
  -d '{"reason": "retention policy", "forget_after": "2027-01-01T00:00:00Z"}'

# Hard delete (irreversible)
curl -X DELETE http://localhost:4200/api/memory/mem:abc123
```

Forgotten memories are excluded from `GET /api/memory` by default. Pass `include_forgotten=true` to retrieve them.

---

## is_static vs. Dynamic

The `is_static` boolean controls where a memory appears in the Profile response.

| Value | Profile tier | Meaning |
|---|---|---|
| `true` | `static` | Rarely-changing facts (team structure, product definitions, key decisions) |
| `false` (default) | `dynamic` | Rolling, session-scoped, or ephemeral observations |

Static memories form the stable baseline of the workspace. Dynamic memories are the working surface — they can be forgotten or versioned frequently without affecting the stable baseline.

```bash
# Create a static memory (stable fact)
curl -X POST http://localhost:4200/api/memory \
  -H 'Content-Type: application/json' \
  -d '{"content": "Our base price is $2,000/seat/year", "workspace": "sales", "is_static": true}'

# Create a dynamic memory (working observation)
curl -X POST http://localhost:4200/api/memory \
  -H 'Content-Type: application/json' \
  -d '{"content": "Alice mentioned willingness to do $1,800 in todays call", "workspace": "sales"}'
```

---

## Audience Scoping

Every memory can be tagged with an `audience`. When retrieving via `/api/profile?audience=exec`, only memories whose `audience` matches `"exec"`, `"default"`, or is unset are returned.

```bash
# Create an exec-only memory
curl -X POST http://localhost:4200/api/memory \
  -H 'Content-Type: application/json' \
  -d '{"content": "Board approved M&A budget at $50M", "workspace": "default", "audience": "exec"}'
```

---

## Citation URI

`citation_uri` links a memory to its source using the `optimal://` URI scheme.

```bash
# Memory citing a specific wiki page
curl -X POST http://localhost:4200/api/memory \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "Pricing decision confirmed — see wiki",
    "workspace": "sales",
    "citation_uri": "optimal://wiki/customer portal-pricing-decision"
  }'

# Memory citing a specific chunk
curl -X POST http://localhost:4200/api/memory \
  -H 'Content-Type: application/json' \
  -d '{
    "content": "Alice committed to follow-up by Friday",
    "workspace": "sales",
    "source_chunk_id": "chunk:sha256:abc123"
  }'
```

---

## Reading the Relation Graph

```bash
# All inbound and outbound relations for a memory
curl http://localhost:4200/api/memory/mem:abc123/relations | jq '.'

# Returns:
# {
#   "memory_id": "mem:abc123",
#   "inbound": [
#     { "relation": "updates", "source_id": "mem:ver001", "target_id": "mem:abc123" }
#   ],
#   "outbound": [
#     { "relation": "cites", "source_id": "mem:abc123", "target_id": "chunk:sha256:..." }
#   ]
# }
```

---

## Full Lifecycle Example

```bash
ENGINE=http://localhost:4200
WS=sales

# 1. Create the initial fact
ID=$(curl -s -X POST $ENGINE/api/memory \
  -H 'Content-Type: application/json' \
  -d "{\"content\":\"Alice leads the Q4 pricing negotiation\",\"workspace\":\"$WS\",\"is_static\":true}" \
  | jq -r '.id')

# 2. It evolves — update it
V2=$(curl -s -X POST "$ENGINE/api/memory/$ID/update" \
  -H 'Content-Type: application/json' \
  -d '{"content":"Alice and Bob co-lead Q4 pricing after org change on 2026-04-28"}' \
  | jq -r '.id')

# 3. Add a derived conclusion
curl -s -X POST "$ENGINE/api/memory/$V2/derive" \
  -H 'Content-Type: application/json' \
  -d '{"content":"Escalate pricing decisions to Alice AND Bob — either can approve"}'

# 4. Verify the chain
curl -s "$ENGINE/api/memory/$ID/versions" | jq '.versions | length'  # should be 2

# 5. The old fact is no longer latest
curl -s "$ENGINE/api/memory/$ID" | jq '.is_latest'  # false

# 6. When resolved, forget the derived conclusion
curl -s -X POST "$ENGINE/api/memory/$V2/forget" \
  -H 'Content-Type: application/json' \
  -d '{"reason":"Q4 closed, negotiation complete"}'
```
