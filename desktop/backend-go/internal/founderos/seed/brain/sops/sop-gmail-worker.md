---
title: Triage the four Gmail inboxes
kind: sop
generated: founder-os
---

# Triage the four Gmail inboxes

## Purpose

IMAP slots 1–4 read, classified, escalated.

## Owner

[[gmail-worker]] — one worker, one job (monogamous by design).
Runs on: builtin · imapflow.

## Trigger

Kicks off when it is time to "connect the four configured imap inboxes on the sync cadence" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Connect the four configured IMAP inboxes on the sync cadence
2. Pull unread counts and every thread newer than the last sweep
3. Classify each thread: urgent, reply-needed, waiting-on-us, FYI
4. Draft suggested replies for reply-needed threads in Alex voice
5. Hand urgent threads to the escalation queue with a one-line summary
6. Surface anything from a client domain to the Clients pillar too

## Definition of done

The run is complete when "surface anything from a client domain to the clients pillar too" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Mia Torres ([[person-mia]]). Never fake a green run.

## Pillar

[[pillar-communications]]
