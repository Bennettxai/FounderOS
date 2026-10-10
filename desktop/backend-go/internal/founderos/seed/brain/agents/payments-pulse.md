---
title: Payments Pulse
kind: agent
generated: founder-os
---

# Payments Pulse

Processor Monitor in [[pillar-finances]].

Stripe balance + recent charges; PayPal/Square/Whop registered and awaiting keys.

## Instructions

Executes [[sop-payments-pulse]] — Watch processor health.

1. Ping each processor registered in the registry
2. Record honest ConnectorStatus, never fake connected
3. Alert Finances when a processor goes down
4. Re-check failed processors on a tighter cadence until they recover
5. Keep the uptime history for the analytics view

## Harness

- Tier: lead
- Runs on: builtin · stripe sdk
- Status: planned
- Sub-agents: [[flexpay-financing]] [[paykit-sales]] [[processor-confirmation]] [[stripe-sales]]
- Human lead: [[person-dana]]

## Tools

- [[stripe]]
- [[paypal]]
- [[square]]
- [[whop]]
