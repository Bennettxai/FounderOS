---
title: Automate DM funnels
kind: sop
generated: founder-os
---

# Automate DM funnels

## Purpose

Keyword triggers to booked conversations.

## Owner

[[dmflow-mcp]] — one worker, one job (monogamous by design).
Runs on: builtin · dmflow api.

## Trigger

Kicks off when it is time to "watch configured trigger keywords across platforms" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Watch configured trigger keywords across platforms
2. Fire the matching DMFlow flow for each trigger
3. Tag subscribers by intent as they move through the flow
4. Hand hot leads to the Sales pillar with their conversation history
5. Report conversions back to the growth dashboard

## Definition of done

The run is complete when "report conversions back to the growth dashboard" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Nadia ([[person-nadia]]). Never fake a green run.

## Pillar

[[pillar-marketing-growth]]
