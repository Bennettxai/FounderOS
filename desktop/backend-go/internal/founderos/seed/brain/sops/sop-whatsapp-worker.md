---
title: Monitor WhatsApp chats
kind: sop
generated: founder-os
---

# Monitor WhatsApp chats

## Purpose

Local team chats surfaced.

## Owner

[[whatsapp-worker]] — one worker, one job (monogamous by design).
Runs on: builtin · local sqlite (read-only).

## Trigger

Kicks off when it is time to "read the local chatstorage.sqlite (read-only, nothing leaves the machine)" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Read the local ChatStorage.sqlite (read-only, nothing leaves the machine)
2. Surface new messages from the LC and Vantage team chats
3. Map senders to their contact tags
4. Flag messages that mention money, deadlines or blockers
5. Push tagged messages into the unified feed

## Definition of done

The run is complete when "push tagged messages into the unified feed" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Mia Torres ([[person-mia]]). Never fake a green run.

## Pillar

[[pillar-communications]]
