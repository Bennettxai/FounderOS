---
title: Client Roster
kind: agent
generated: founder-os
---

# Client Roster

Live Client List in [[pillar-clients]].

The single source of truth for who is a client: reconciles Ledger and PayKit against the funnel and keeps the roster current.

## Instructions

Executes [[sop-client-roster]] — Keep the client roster live.

1. Pull clients and deal states from Ledger and PayKit every morning
2. Reconcile them against the funnel journeys and payment records
3. Mark each account active, at risk, or churned with a reason
4. Flag stale records and missing fields to the owning lane
5. Publish the roster to the Clients pillar and note the deltas

## Harness

- Tier: lead
- Runs on: builtin · funnel + Ledger
- Status: active
- Sub-agents: [[client-success]] [[client-onboarding]]
- Human lead: [[person-sasha]]

## Tools

- [[ledger]]
- [[paykit]]
