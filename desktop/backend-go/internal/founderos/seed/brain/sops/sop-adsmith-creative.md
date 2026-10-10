---
title: Generate UGC ad variants
kind: sop
generated: founder-os
---

# Generate UGC ad variants

## Purpose

Vantage ad angles rendered as UGC actors.

## Owner

[[adsmith-creative]] — one worker, one job (monogamous by design).
Runs on: builtin · adsmith api.

## Trigger

Kicks off when it is time to "take the ad brief with hook, angle and offer" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Take the ad brief with hook, angle and offer
2. Generate actor variants across Veo / Sora / Kling
3. Cull the takes that break the brief before rendering finals
4. Render finals and name them by angle
5. Deliver the batch to creative review with a variant sheet

## Definition of done

The run is complete when "deliver the batch to creative review with a variant sheet" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Nadia ([[person-nadia]]). Never fake a green run.

## Pillar

[[pillar-marketing-growth]]
