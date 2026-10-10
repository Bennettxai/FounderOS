---
title: Vantage
kind: agent
generated: founder-os
---

# Vantage

Sales Account Lane in [[pillar-sales]].

Vantage sales lane: account pipeline, PayKit context, payment confirmation, and call data.

## Instructions

Executes [[sop-vantage-lane]] — Run the Vantage lane.

1. Qualify inbound Vantage leads against the ICP
2. Book qualified leads onto Marco’s calendar with context attached
3. Sync stage changes back to Ledger
4. Reconcile payments across PayKit and Stripe
5. Report lane revenue to the pipeline brief

## Harness

- Tier: worker
- Runs on: builtin · account lane
- Status: planned
- Reports to: [[sales-agent]]
- Sub-agents: [[vantage-paykit]]
- Human lead: [[person-marco]]

## Tools

- [[ledger]]
- [[stripe]]
- [[paykit]]
