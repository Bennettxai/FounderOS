---
title: Slack Worker
kind: agent
generated: founder-os
---

# Slack Worker

Channel Digest in [[pillar-communications]].

Latest messages across joined channels into /comms. Needs SLACK_BOT_TOKEN.

## Instructions

Executes [[sop-slack-worker]] — Digest Slack channels.

1. List channels the bot has joined
2. Pull the latest messages per channel since the last sweep
3. Summarize each channel into a short digest
4. Call out direct mentions and unanswered questions separately
5. Push the digest into the unified feed

## Harness

- Tier: worker
- Runs on: builtin · @slack/web-api
- Status: planned
- Reports to: [[comms-agent]]
- Human lead: [[person-mia]]

## Tools

- [[slack]]
