---
title: Comms Agent
kind: agent
generated: founder-os
---

# Comms Agent

Unified Communications Instance in [[pillar-communications]].

Owns the unified /comms feed. Aggregates its three channel workers and reports which are live.

## Instructions

Executes [[sop-comms-agent]] — Compose the unified comms feed.

1. Collect fresh output from the Gmail, WhatsApp and Slack workers
2. Dedupe and merge everything into one ordered timeline
3. Tag each entry with its contact tier
4. Bubble urgent and reply-needed items to the top of the feed
5. Publish the feed and report which channels are live

## Harness

- Tier: lead
- Runs on: builtin · aggregate of workers
- Status: active
- Sub-agents: [[gmail-worker]] [[slack-worker]] [[whatsapp-worker]]
- Human lead: [[person-mia]]

## Tools

- [[comms-feed]]
