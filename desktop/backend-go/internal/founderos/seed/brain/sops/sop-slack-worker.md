---
title: Digest Slack channels
kind: sop
generated: founder-os
---

# Digest Slack channels

## Purpose

Joined channels summarized into the feed.

## Owner

[[slack-worker]] — one worker, one job (monogamous by design).
Runs on: builtin · @slack/web-api.

## Trigger

Kicks off when it is time to "list channels the bot has joined" — on cadence or on the upstream event, whichever lands first.

## Steps

1. List channels the bot has joined
2. Pull the latest messages per channel since the last sweep
3. Summarize each channel into a short digest
4. Call out direct mentions and unanswered questions separately
5. Push the digest into the unified feed

## Definition of done

The run is complete when "push the digest into the unified feed" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Mia Torres ([[person-mia]]). Never fake a green run.

## Pillar

[[pillar-communications]]
