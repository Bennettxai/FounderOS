---
title: Processor Confirm
kind: agent
generated: founder-os
---

# Processor Confirm

Payment API Confirmation in [[pillar-finances]].

APIs to payment processors for confirming paid, failed, disputed, and pending states.

## Instructions

Executes [[sop-processor-confirm]] — Confirm payments across processors.

1. Receive the payment claim from a sales lane
2. Check the claimed processor’s API (Stripe / PayPal / Square / Whop / PayKit)
3. Confirm the charge or flag the mismatch loudly
4. Write the confirmation onto the deal record
5. Keep an audit trail of every confirmation for month-end close

## Harness

- Tier: worker
- Runs on: builtin · processor registry
- Status: planned
- Reports to: [[payments-pulse]]
- Human lead: [[person-dana]]

## Tools

- [[stripe]]
- [[paypal]]
- [[square]]
- [[whop]]
- [[paykit]]
