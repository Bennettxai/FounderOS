---
title: Vantage PayKit
kind: agent
generated: founder-os
---

# Vantage PayKit

Vantage PayKit Lane in [[pillar-sales]].

PayKit lane specifically under Vantage for offer, payment, and customer context.

## Instructions

Executes [[sop-vantage-paykit]] — Reconcile the Vantage PayKit lane.

1. Pull month-to-date customers from PayKit
2. Match each payment to its Ledger deal
3. Flag payments with no deal and deals with no payment
4. Chase every mismatch to a resolution, not just a flag
5. Post month-to-date totals to Finances

## Harness

- Tier: worker
- Runs on: builtin · paykit api
- Status: planned
- Reports to: [[vantage-sales]]
- Human lead: [[person-marco]]

## Tools

- [[paykit]]
