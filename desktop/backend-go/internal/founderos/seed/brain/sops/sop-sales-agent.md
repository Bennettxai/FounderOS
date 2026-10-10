---
title: Keep the pipeline moving
kind: sop
generated: founder-os
---

# Keep the pipeline moving

## Purpose

Deals inspected daily, nothing stalls silently.

## Owner

[[sales-agent]] — one worker, one job (monogamous by design).
Runs on: builtin · aggregate of workers.

## Trigger

Kicks off when it is time to "pull every open deal and its stage from ledger each morning" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull every open deal and its stage from Ledger each morning
2. Rank deals by value and days-in-stage; anything past 7 days is stalled
3. Attach a concrete next action and owner to every stalled deal
4. Prepare payment links across PayKit, Stripe and FlexPay before calls
5. Brief Marco with the top five deals and their objections before each call
6. Log stage changes back to Ledger the same day they happen

## Definition of done

The run is complete when "log stage changes back to ledger the same day they happen" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Marco ([[person-marco]]). Never fake a green run.

## Pillar

[[pillar-sales]]
