---
title: Postly Publisher
kind: agent
generated: founder-os
---

# Postly Publisher

Six-Platform Publishing in [[pillar-marketing-growth]].

Publishes and monitors six platforms under @founderos.ai via Postly. Live once the Postly key is set.

## Instructions

Executes [[sop-postly-publisher]] — Publish to six platforms.

1. Take the next queued post from the pipeline
2. Adapt the caption per platform (IG, TikTok, X, YouTube, LinkedIn, Facebook)
3. Publish through the Postly API
4. Record post ids and verify each went live
5. Retry failed platforms once, then flag them to the Social Agent

## Harness

- Tier: worker
- Runs on: builtin · postly api
- Status: active
- Reports to: [[social-agent]]
- Human lead: [[person-nadia]]

## Tools

- [[postly]]
