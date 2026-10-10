---
title: Answer questions from the Brain
kind: sop
generated: founder-os
---

# Answer questions from the Brain

## Purpose

Hybrid search over the second brain, honest fallbacks.

## Owner

[[data-agent]] — one worker, one job (monogamous by design).
Runs on: builtin · Optimal Engine.

## Trigger

Kicks off when it is time to "parse the incoming question into a Brain query" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Parse the incoming question into a Brain query
2. Run engine hybrid search (--no-expand) against Supabase
3. Fall back to local brain-store grep when the database is paused
4. Rank passages and keep only the ones that actually answer the question
5. Return cited passages with their source notes, never invented ones
6. Log unanswerable questions as gaps for the Markdown Auditor to fill

## Definition of done

The run is complete when "log unanswerable questions as gaps for the markdown auditor to fill" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to the operator. Never fake a green run.

## Pillar

[[pillar-tech]]
