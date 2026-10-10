---
title: Vector Auditor
kind: agent
generated: founder-os
---

# Vector Auditor

pgvector / Supabase Health in [[pillar-tech]].

Runs engine doctor: connection to Supabase pgvector, embedding checks, health score. Works today.

## Instructions

Executes [[sop-vector-auditor]] — Audit the vector index.

1. Ping the Supabase Second Brain project (free tier pauses on idle)
2. Wake the database and wait until it accepts queries before comparing
3. Compare pgvector chunk counts against brain-store files
4. Flag drift and paused-tier warnings on the /brain doctor card
5. Trigger bge-m3 re-embeds for drifted documents and verify counts after

## Harness

- Tier: worker
- Runs on: builtin · engine doctor
- Status: active
- Reports to: [[data-agent]]

## Tools

- [[supabase]]
- [[ollama]]
