---
title: Keep Ledger clean
kind: sop
generated: founder-os
---

# Keep Ledger clean

## Purpose

A CRM the numbers can be trusted from.

## Owner

[[crm-pulse]] — one worker, one job (monogamous by design).
Runs on: builtin · ledger api.

## Trigger

Kicks off when it is time to "scan records for missing fields and duplicates" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Scan records for missing fields and duplicates
2. Verify deal stages match what actually happened
3. Merge duplicates and backfill whatever can be backfilled safely
4. Nudge lane owners on records gone stale
5. Snapshot pipeline metrics for the dashboard

## Definition of done

The run is complete when "snapshot pipeline metrics for the dashboard" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Marco ([[person-marco]]). Never fake a green run.

## Pillar

[[pillar-sales]]
