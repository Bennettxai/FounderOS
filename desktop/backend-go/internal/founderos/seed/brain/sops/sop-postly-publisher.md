---
title: Publish to six platforms
kind: sop
generated: founder-os
---

# Publish to six platforms

## Purpose

One queue out to every @founderos.ai surface.

## Owner

[[postly-publisher]] — one worker, one job (monogamous by design).
Runs on: builtin · postly api.

## Trigger

Kicks off when it is time to "take the next queued post from the pipeline" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Take the next queued post from the pipeline
2. Adapt the caption per platform (IG, TikTok, X, YouTube, LinkedIn, Facebook)
3. Publish through the Postly API
4. Record post ids and verify each went live
5. Retry failed platforms once, then flag them to the Social Agent

## Definition of done

The run is complete when "retry failed platforms once, then flag them to the social agent" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Nadia ([[person-nadia]]). Never fake a green run.

## Pillar

[[pillar-marketing-growth]]
