---
title: Track PayKit income
kind: sop
generated: founder-os
---

# Track PayKit income

## Purpose

Month-to-date, split by venture, refunds flagged.

## Owner

[[paykit-sales]] — one worker, one job (monogamous by design).
Runs on: builtin · paykit api.

## Trigger

Kicks off when it is time to "pull month-to-date customers from the paykit api" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull month-to-date customers from the PayKit API
2. Split income by venture (LC vs Vantage)
3. Record the income snapshot for the Finances view
4. Flag refunds and disputes the day they land
5. Reconcile the running total against the month-end books

## Definition of done

The run is complete when "reconcile the running total against the month-end books" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Dana Whitfield ([[person-dana]]). Never fake a green run.

## Pillar

[[pillar-finances]]
