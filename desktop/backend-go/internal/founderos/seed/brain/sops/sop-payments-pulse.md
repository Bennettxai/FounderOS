---
title: Watch processor health
kind: sop
generated: founder-os
---

# Watch processor health

## Purpose

Every processor pinged, status recorded honestly.

## Owner

[[payments-pulse]] — one worker, one job (monogamous by design).
Runs on: builtin · stripe sdk.

## Trigger

Kicks off when it is time to "ping each processor registered in the registry" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Ping each processor registered in the registry
2. Record honest ConnectorStatus, never fake connected
3. Alert Finances when a processor goes down
4. Re-check failed processors on a tighter cadence until they recover
5. Keep the uptime history for the analytics view

## Definition of done

The run is complete when "keep the uptime history for the analytics view" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Dana Whitfield ([[person-dana]]). Never fake a green run.

## Pillar

[[pillar-finances]]
