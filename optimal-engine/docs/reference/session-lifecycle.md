# Session Lifecycle

A session is a bounded human/agent work period. It should not become an
untracked chat log.

## Lifecycle

```text
start session
  -> identify actor
  -> choose workspace/node/task scope
  -> open or attach Active Memory Pool
  -> assemble Context Package
  -> work with tools/models/files
  -> record observations
  -> create pending Claims when useful
  -> close or archive pool
  -> refresh projections
```

## Session State

```text
session_id
actor_id
workspace_id
node_scope
active_memory_pool_id
context_package_ids
tool_call_run_ids
model_call_run_ids
observations
pending_claim_ids
artifacts
audit_event_ids
```

## Capture Rule

Conversation summaries are not Facts. They are source evidence or observations
until reviewed.

```text
session summary -> Source Package
useful assertion -> pending Claim
accepted assertion -> Fact
contextual meaning -> Memory Object
```

