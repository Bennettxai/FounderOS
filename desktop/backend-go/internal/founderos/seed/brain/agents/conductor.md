---
title: Conductor
kind: agent
generated: founder-os
---

# Conductor

Broadcast & Orchestration in [[pillar-tech]].

Fans your message out to every agent at once and checks which instance hosts (Clawline, Ollama, tmux) are available for future bindings.

## Instructions

Executes [[sop-conductor]] — Broadcast directives across the fleet.

1. Receive the directive from the operator console
2. Resolve the target list: the whole fleet, or the pillar the directive names
3. Poll instance hosts (Clawline, Ollama, tmux) for availability before dispatch
4. Fan the message out to every target at once and stamp each send
5. Collect replies as they land and file the run to agent_runs
6. Report non-responders after sixty seconds so nothing fails silently

## Harness

- Tier: lead
- Runs on: builtin · fan-out runtime
- Status: active

## Tools

- [[broadcast]]
- [[clawline]]
- [[tmux]]
