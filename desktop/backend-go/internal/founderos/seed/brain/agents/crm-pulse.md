---
title: Ledger CRM
kind: agent
generated: founder-os
---

# Ledger CRM

ATTO / Ledger Deals Pipeline in [[pillar-sales]].

Vantage + LC deals from Ledger, key reused from the MCP config. Works today.

## Instructions

Executes [[sop-crm-pulse]] — Keep Ledger clean.

1. Scan records for missing fields and duplicates
2. Verify deal stages match what actually happened
3. Merge duplicates and backfill whatever can be backfilled safely
4. Nudge lane owners on records gone stale
5. Snapshot pipeline metrics for the dashboard

## Harness

- Tier: worker
- Runs on: builtin · ledger api
- Status: active
- Reports to: [[sales-agent]]
- Human lead: [[person-marco]]

## Tools

- [[ledger]]
