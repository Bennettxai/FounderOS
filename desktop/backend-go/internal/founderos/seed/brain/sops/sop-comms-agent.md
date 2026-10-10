---
title: Compose the unified comms feed
kind: sop
generated: founder-os
---

# Compose the unified comms feed

## Purpose

Three channels, one timeline at /comms.

## Owner

[[comms-agent]] — one worker, one job (monogamous by design).
Runs on: builtin · aggregate of workers.

## Trigger

Kicks off when it is time to "collect fresh output from the gmail, whatsapp and slack workers" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Collect fresh output from the Gmail, WhatsApp and Slack workers
2. Dedupe and merge everything into one ordered timeline
3. Tag each entry with its contact tier
4. Bubble urgent and reply-needed items to the top of the feed
5. Publish the feed and report which channels are live

## Definition of done

The run is complete when "publish the feed and report which channels are live" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Mia Torres ([[person-mia]]). Never fake a green run.

## Pillar

[[pillar-communications]]
