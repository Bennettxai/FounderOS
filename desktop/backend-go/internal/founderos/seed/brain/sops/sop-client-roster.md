---
title: Keep the client roster live
kind: sop
generated: founder-os
---

# Keep the client roster live

## Purpose

One list of every client, always current.

## Owner

[[client-roster]] — one worker, one job (monogamous by design).
Runs on: builtin · funnel + Ledger.

## Trigger

Kicks off when it is time to "pull clients and deal states from ledger and paykit every morning" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull clients and deal states from Ledger and PayKit every morning
2. Reconcile them against the funnel journeys and payment records
3. Mark each account active, at risk, or churned with a reason
4. Flag stale records and missing fields to the owning lane
5. Publish the roster to the Clients pillar and note the deltas

## Definition of done

The run is complete when "publish the roster to the clients pillar and note the deltas" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Sasha Bell ([[person-sasha]]). Never fake a green run.

## Pillar

[[pillar-clients]]
