---
title: Reconcile the Vantage PayKit lane
kind: sop
generated: founder-os
---

# Reconcile the Vantage PayKit lane

## Purpose

PayKit customers matched to CRM deals.

## Owner

[[vantage-paykit]] — one worker, one job (monogamous by design).
Runs on: builtin · paykit api.

## Trigger

Kicks off when it is time to "pull month-to-date customers from paykit" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull month-to-date customers from PayKit
2. Match each payment to its Ledger deal
3. Flag payments with no deal and deals with no payment
4. Chase every mismatch to a resolution, not just a flag
5. Post month-to-date totals to Finances

## Definition of done

The run is complete when "post month-to-date totals to finances" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Marco ([[person-marco]]). Never fake a green run.

## Pillar

[[pillar-sales]]
