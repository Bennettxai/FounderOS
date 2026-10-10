---
title: Stripe
kind: agent
generated: founder-os
---

# Stripe

Sales Payment Processor in [[pillar-finances]].

Stripe payment confirmation lane for sales workflows and account-level revenue checks.

## Instructions

Executes [[sop-stripe]] — Track Stripe income.

1. Pull balance and recent charges from Stripe
2. Label income to Launchpad Cohort
3. Record the snapshot for the income chart
4. Flag anomalies against the trailing average
5. Note upcoming payouts so cash flow is never a surprise

## Harness

- Tier: worker
- Runs on: builtin · stripe sdk
- Status: planned
- Reports to: [[payments-pulse]]
- Human lead: [[person-dana]]

## Tools

- [[stripe]]
