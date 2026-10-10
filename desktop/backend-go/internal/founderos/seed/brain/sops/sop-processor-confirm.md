---
title: Confirm payments across processors
kind: sop
generated: founder-os
---

# Confirm payments across processors

## Purpose

No deal marked paid without an API receipt.

## Owner

[[processor-confirmation]] — one worker, one job (monogamous by design).
Runs on: builtin · processor registry.

## Trigger

Kicks off when it is time to "receive the payment claim from a sales lane" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Receive the payment claim from a sales lane
2. Check the claimed processor’s API (Stripe / PayPal / Square / Whop / PayKit)
3. Confirm the charge or flag the mismatch loudly
4. Write the confirmation onto the deal record
5. Keep an audit trail of every confirmation for month-end close

## Definition of done

The run is complete when "keep an audit trail of every confirmation for month-end close" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Dana Whitfield ([[person-dana]]). Never fake a green run.

## Pillar

[[pillar-finances]]
