# Failure Modes

Optimal Engine is designed to avoid a few predictable failures in agent and
workspace systems.

## Canonical Failures

| Failure | What happens | Prevention |
| --- | --- | --- |
| Unsupported truth | A note, model output, or tool result becomes accepted knowledge without evidence. | Source Package -> Claim -> review -> Fact. |
| Projection drift | Markdown/wiki/app views stop matching engine state. | Export records, freshness checks, rebuild commands. |
| Context leakage | Retrieval exposes objects outside the actor's scope. | Authorization envelope before and during candidate expansion. |
| Stale recall | An answer uses superseded or expired context. | valid time, transaction time, stale_after, supersession status. |
| Agent overreach | An agent changes topology, calls tools, or promotes Facts without permission. | tool/model governance, topology change requests, review gates. |
| Store confusion | Files, caches, indexes, and database rows all pretend to be truth. | storage/projection ownership rules. |
| Lost provenance | Derived summaries, embeddings, or workflows cannot be traced back. | derivation ledger and source links. |
| Workflow fossilization | An old procedure keeps being reused after context changes. | validation state, stale checks, exception logs, retirement status. |

## Operational Checks

Before shipping a new feature, ask:

```text
Who owns this lifecycle?
Where is the source evidence?
Is this durable truth or a projection?
Can it be rebuilt?
What invalidates it?
Who is allowed to retrieve it?
What audit event proves it happened?
```

If a feature cannot answer those questions, it is not ready to become part of
the governed engine.

