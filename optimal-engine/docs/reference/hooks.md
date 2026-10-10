# Agent Hooks

Hooks connect agent sessions to the governed engine lifecycle.

## Hook Model

| Hook | Fires when | Purpose |
| --- | --- | --- |
| SessionStart | Agent/session begins. | Load workspace scope, active pool, and initial Context Package. |
| PreWrite | Agent writes markdown, code, docs, or state. | Decide whether the write is projection, source evidence, topology change, or artifact. |
| PreToolCall | Agent calls MCP/API/script/model/tool. | Validate grant, schema, partition, and confirmation policy. |
| PostToolCall | Tool/model returns. | Record run, preserve output, create observation or Source Package. |
| SearchEnhance | Agent asks/searches. | Add workspace/node/time/modality hints and permission scope. |
| ModeTransition | Task or rhythm mode changes. | Save current pool state, refresh context, load next scope. |
| SessionCapture | Session ends. | Capture summary, observations, pending Claims, artifacts, and audit. |

## Rule

```text
Agent hooks should call engine interfaces.
They should not write durable truth directly.
```

## Expected Flow

```text
SessionStart
  -> load topology
  -> retrieve Context Package
  -> open Active Memory Pool
  -> work
  -> governed tool/model calls
  -> observations
  -> pending Claims
  -> SessionCapture
```

This gives Codex, Claude Code, MCP clients, scripts, and app agents the same
operating contract.

