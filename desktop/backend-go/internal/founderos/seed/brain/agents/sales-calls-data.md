---
title: Sales Calls Data
kind: agent
generated: founder-os
---

# Sales Calls Data

Call Intelligence in [[pillar-sales]].

Sales calls data lane for recordings, notes, outcomes, and follow-up context: Recall on the calls, Plaud in the room.

## Instructions

Executes [[sop-sales-calls-data]] — Mine sales-call recordings.

1. Ingest Recall notes after each recorded call
2. Ingest Plaud transcripts + AI notes after each in-person meeting or site walk
3. Extract objections, commitments and next steps
4. Write the extract back to the Ledger record
5. Tag calls where pricing or competitors came up
6. Feed recurring patterns into the pipeline brief

## Harness

- Tier: worker
- Runs on: builtin · recall + plaud + crm
- Status: planned
- Reports to: [[sales-agent]]
- Human lead: [[person-marco]]

## Tools

- [[recall]]
- [[plaud]]
- [[ledger]]
