---
title: Run the 09:00 comms report
kind: sop
generated: founder-os
---

# Run the 09:00 comms report

## Purpose

Every morning: 24h of email, WhatsApp and Slack, ranked by who needs a reply.

## Owner

[[comms-digest]] — one worker, one job (monogamous by design).
Runs on: builtin · rules + connectors.

## Trigger

Kicks off when it is time to "pull the trailing 24 hours from all four inboxes, whatsapp and slack (one guarded call each — a dead channel degrades the report, it never cancels it)" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull the trailing 24 hours from all four inboxes, WhatsApp and Slack (one guarded call each — a dead channel degrades the report, it never cancels it)
2. Load the ranking context: calendar titles for upcoming calls, the Ledger roster for clients, contact tags for community members
3. Rank every message: calls first, then clients and proposal replies, then community questions, then brand deals, then group chats, companies and software last
4. Collect the automated senders into an unsubscribe worklist, noisiest first
5. Store the report so /comms renders it instantly, and log the run against the schedule

## Definition of done

The run is complete when "store the report so /comms renders it instantly, and log the run against the schedule" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Mia Torres ([[person-mia]]). Never fake a green run.

## Pillar

[[pillar-communications]]
