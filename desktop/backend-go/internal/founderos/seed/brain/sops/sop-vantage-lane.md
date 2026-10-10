---
title: Run the Vantage lane
kind: sop
generated: founder-os
---

# Run the Vantage lane

## Purpose

Local-business inbound worked end to end.

## Owner

[[vantage-sales]] — one worker, one job (monogamous by design).
Runs on: builtin · account lane.

## Trigger

Kicks off when it is time to "qualify inbound vantage leads against the icp" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Qualify inbound Vantage leads against the ICP
2. Book qualified leads onto Marco’s calendar with context attached
3. Sync stage changes back to Ledger
4. Reconcile payments across PayKit and Stripe
5. Report lane revenue to the pipeline brief

## Definition of done

The run is complete when "report lane revenue to the pipeline brief" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Marco ([[person-marco]]). Never fake a green run.

## Pillar

[[pillar-sales]]
