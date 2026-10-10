---
title: Run the daily content pipeline
kind: sop
generated: founder-os
---

# Run the daily content pipeline

## Purpose

Calendar → briefs → assets → publish queue.

## Owner

[[social-agent]] — one worker, one job (monogamous by design).
Runs on: builtin · aggregate of workers.

## Trigger

Kicks off when it is time to "pull today’s slots from the content calendar" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull today’s slots from the content calendar
2. Brief the creative workers (Adsmith, Renderly, Reelkit) with hooks and formats
3. Collect finished assets and check them against the brief
4. Reject anything off-brand with a one-line reason so the fix is fast
5. Queue approved posts for the Postly publisher with per-platform captions
6. Log what shipped to the calendar so tomorrow’s brief starts warm

## Definition of done

The run is complete when "log what shipped to the calendar so tomorrow’s brief starts warm" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Nadia ([[person-nadia]]). Never fake a green run.

## Pillar

[[pillar-marketing-growth]]
