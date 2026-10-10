---
title: Gmail Worker
kind: agent
generated: founder-os
---

# Gmail Worker

IMAP Inboxes ×4 in [[pillar-communications]].

Pulls unread counts and recent mail from up to four IMAP inboxes into /comms. Activates when INBOX_* creds land.

## Instructions

Executes [[sop-gmail-worker]] — Triage the four Gmail inboxes.

1. Connect the four configured IMAP inboxes on the sync cadence
2. Pull unread counts and every thread newer than the last sweep
3. Classify each thread: urgent, reply-needed, waiting-on-us, FYI
4. Draft suggested replies for reply-needed threads in Alex voice
5. Hand urgent threads to the escalation queue with a one-line summary
6. Surface anything from a client domain to the Clients pillar too

## Harness

- Tier: worker
- Runs on: builtin · imapflow
- Status: planned
- Reports to: [[comms-agent]]
- Human lead: [[person-mia]]

## Tools

- [[imap]]
