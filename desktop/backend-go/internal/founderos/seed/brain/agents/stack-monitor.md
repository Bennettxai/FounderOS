---
title: Stack Monitor
kind: agent
generated: founder-os
---

# Stack Monitor

Local Stack Health in [[pillar-tech]].

Reelkit, Ollama, command-center, Clawline, tmux, whisper, ffmpeg, renderly, gh + Dictate Flow stats.

## Instructions

Executes [[sop-stack-monitor]] — Watch the local stack.

1. Probe the command center :3100 and the worker gateway :8642
2. Check the brew binaries the agents shell out to (ffmpeg, pdftotext, whisper, gh) and the Optimal Engine
3. Record honest ConnectorStatus, never fake connected
4. Compare against the last sweep to catch flapping services
5. Alert the console when something that was up goes down

## Harness

- Tier: lead
- Runs on: builtin · local checks
- Status: active

## Tools

- [[reelkit]]
- [[ollama]]
- [[tmux]]
- [[dictate]]
