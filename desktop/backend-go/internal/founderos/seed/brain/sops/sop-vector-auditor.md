---
title: Audit the vector index
kind: sop
generated: founder-os
---

# Audit the vector index

## Purpose

Embeddings in Supabase must mirror brain-store.

## Owner

[[vector-auditor]] — one worker, one job (monogamous by design).
Runs on: builtin · engine doctor.

## Trigger

Kicks off when it is time to "ping the supabase second brain project (free tier pauses on idle)" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Ping the Supabase Second Brain project (free tier pauses on idle)
2. Wake the database and wait until it accepts queries before comparing
3. Compare pgvector chunk counts against brain-store files
4. Flag drift and paused-tier warnings on the /brain doctor card
5. Trigger bge-m3 re-embeds for drifted documents and verify counts after

## Definition of done

The run is complete when "trigger bge-m3 re-embeds for drifted documents and verify counts after" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to the operator. Never fake a green run.

## Pillar

[[pillar-tech]]
