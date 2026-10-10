---
title: Mine sales-call recordings
kind: sop
generated: founder-os
---

# Mine sales-call recordings

## Purpose

Every Recall call and Plaud recording becomes CRM intelligence.

## Owner

[[sales-calls-data]] — one worker, one job (monogamous by design).
Runs on: builtin · recall + plaud + crm.

## Trigger

Kicks off when it is time to "ingest recall notes after each recorded call" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Ingest Recall notes after each recorded call
2. Ingest Plaud transcripts + AI notes after each in-person meeting or site walk
3. Extract objections, commitments and next steps
4. Write the extract back to the Ledger record
5. Tag calls where pricing or competitors came up
6. Feed recurring patterns into the pipeline brief

## Definition of done

The run is complete when "feed recurring patterns into the pipeline brief" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Marco ([[person-marco]]). Never fake a green run.

## Pillar

[[pillar-sales]]
