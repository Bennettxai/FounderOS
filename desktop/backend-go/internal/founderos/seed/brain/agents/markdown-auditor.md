---
title: Markdown Auditor
kind: agent
generated: founder-os
---

# Markdown Auditor

brain-store Health in [[pillar-tech]].

Audits the knowledge base: broken wikilinks, orphan pages, duplicate titles, and whether the index search reads still matches the store on disk.

## Instructions

Executes [[sop-markdown-auditor]] — Audit brain-store markdown health.

1. Walk every markdown file in knowledge/brain-store
2. Flag broken wiki-links, orphan notes and stale frontmatter
3. Check generated org docs still match the live agents, SOPs and tools
4. Write the health report with per-folder scores
5. Queue fix-ups for the worst offenders and track them to done

## Harness

- Tier: worker
- Runs on: builtin · link audit
- Status: active
- Reports to: [[data-agent]]

## Tools

- [[brain-store]]
