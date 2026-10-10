---
title: Comms Digest
kind: agent
generated: founder-os
---

# Comms Digest

Morning Report · 09:00 daily in [[pillar-communications]].

Scrapes the last 24h across all four inboxes, WhatsApp and Slack and ranks who needs a reply: calls first, then clients, community members, brand deals, group chats, companies last. Also lists what to unsubscribe from.

## Instructions

Executes [[sop-comms-digest]] — Run the 09:00 comms report.

1. Pull the trailing 24 hours from all four inboxes, WhatsApp and Slack (one guarded call each — a dead channel degrades the report, it never cancels it)
2. Load the ranking context: calendar titles for upcoming calls, the Ledger roster for clients, contact tags for community members
3. Rank every message: calls first, then clients and proposal replies, then community questions, then brand deals, then group chats, companies and software last
4. Collect the automated senders into an unsubscribe worklist, noisiest first
5. Store the report so /comms renders it instantly, and log the run against the schedule

## Harness

- Tier: lead
- Runs on: builtin · rules + connectors
- Status: active
- Human lead: [[person-mia]]

## Tools

- [[comms-feed]]
- [[calendar]]
- [[ledger]]
