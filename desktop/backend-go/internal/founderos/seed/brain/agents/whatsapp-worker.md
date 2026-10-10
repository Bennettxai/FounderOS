---
title: WhatsApp Worker
kind: agent
generated: founder-os
---

# WhatsApp Worker

Chat Monitor in [[pillar-communications]].

Reads the local WhatsApp ChatStorage (local team chats) into /comms. Works today.

## Instructions

Executes [[sop-whatsapp-worker]] — Monitor WhatsApp chats.

1. Read the local ChatStorage.sqlite (read-only, nothing leaves the machine)
2. Surface new messages from the LC and Vantage team chats
3. Map senders to their contact tags
4. Flag messages that mention money, deadlines or blockers
5. Push tagged messages into the unified feed

## Harness

- Tier: worker
- Runs on: builtin · local sqlite (read-only)
- Status: active
- Reports to: [[comms-agent]]
- Human lead: [[person-mia]]

## Tools

- [[whatsapp]]
