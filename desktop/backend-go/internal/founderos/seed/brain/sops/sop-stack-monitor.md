---
title: Watch the local stack
kind: sop
generated: founder-os
---

# Watch the local stack

## Purpose

Honest status for every port, session and binary.

## Owner

[[stack-monitor]] — one worker, one job (monogamous by design).
Runs on: builtin · local checks.

## Trigger

Kicks off when it is time to "probe the command center :3100 and the worker gateway :8642" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Probe the command center :3100 and the worker gateway :8642
2. Check the brew binaries the agents shell out to (ffmpeg, pdftotext, whisper, gh) and the Optimal Engine
3. Record honest ConnectorStatus, never fake connected
4. Compare against the last sweep to catch flapping services
5. Alert the console when something that was up goes down

## Definition of done

The run is complete when "alert the console when something that was up goes down" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to the operator. Never fake a green run.

## Pillar

[[pillar-tech]]
