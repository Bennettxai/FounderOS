---
title: PayKit
kind: agent
generated: founder-os
---

# PayKit

Offer & Payment Platform in [[pillar-finances]].

PayKit sales platform connection for offers and customer/payment context.

## Instructions

Executes [[sop-paykit]] — Track PayKit income.

1. Pull month-to-date customers from the PayKit API
2. Split income by venture (LC vs Vantage)
3. Record the income snapshot for the Finances view
4. Flag refunds and disputes the day they land
5. Reconcile the running total against the month-end books

## Harness

- Tier: worker
- Runs on: builtin · paykit api
- Status: planned
- Reports to: [[payments-pulse]]
- Human lead: [[person-dana]]

## Tools

- [[paykit]]
