---
title: Social Agent
kind: agent
generated: founder-os
---

# Social Agent

Social Media & Content Creation Instance in [[pillar-marketing-growth]].

Owns publishing and content production. Aggregates the Postly and Adsmith workers.

## Instructions

Executes [[sop-social-agent]] — Run the daily content pipeline.

1. Pull today’s slots from the content calendar
2. Brief the creative workers (Adsmith, Renderly, Reelkit) with hooks and formats
3. Collect finished assets and check them against the brief
4. Reject anything off-brand with a one-line reason so the fix is fast
5. Queue approved posts for the Postly publisher with per-platform captions
6. Log what shipped to the calendar so tomorrow’s brief starts warm

## Harness

- Tier: lead
- Runs on: builtin · aggregate of workers
- Status: active
- Sub-agents: [[adsmith-creative]] [[dmflow-mcp]] [[newsletter-agent]] [[postly-publisher]] [[reelkit-editor]] [[renderly-creative]]
- Human lead: [[person-nadia]]

## Tools

- [[postly]]
- [[adsmith]]
- [[reelkit]]
- [[renderly]]
- [[dmflow]]
