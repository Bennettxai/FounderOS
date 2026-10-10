---
title: Audit brain-store markdown health
kind: sop
generated: founder-os
---

# Audit brain-store markdown health

## Purpose

Keep the knowledge base clean and linkable.

## Owner

[[markdown-auditor]] — one worker, one job (monogamous by design).
Runs on: builtin · link audit.

## Trigger

Kicks off when it is time to "walk every markdown file in knowledge/brain-store" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Walk every markdown file in knowledge/brain-store
2. Flag broken wiki-links, orphan notes and stale frontmatter
3. Check generated org docs still match the live agents, SOPs and tools
4. Write the health report with per-folder scores
5. Queue fix-ups for the worst offenders and track them to done

## Definition of done

The run is complete when "queue fix-ups for the worst offenders and track them to done" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to the operator. Never fake a green run.

## Pillar

[[pillar-tech]]
