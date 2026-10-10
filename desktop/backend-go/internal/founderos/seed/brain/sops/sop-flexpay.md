---
title: Quote financing options
kind: sop
generated: founder-os
---

# Quote financing options

## Purpose

Payment plans attached to live offers.

## Owner

[[flexpay-financing]] — one worker, one job (monogamous by design).
Runs on: builtin · flexpay api.

## Trigger

Kicks off when it is time to "take the deal size and buyer profile from the lane" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Take the deal size and buyer profile from the lane
2. Pull matching plan options from FlexPay
3. Attach terms to the offer before the call
4. Track which plans get accepted and which stall deals
5. Report acceptance rates so pricing keeps getting sharper

## Definition of done

The run is complete when "report acceptance rates so pricing keeps getting sharper" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Dana Whitfield ([[person-dana]]). Never fake a green run.

## Pillar

[[pillar-finances]]
