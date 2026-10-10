---
title: Track Stripe income
kind: sop
generated: founder-os
---

# Track Stripe income

## Purpose

Balance and charges labeled Launchpad Cohort.

## Owner

[[stripe-sales]] — one worker, one job (monogamous by design).
Runs on: builtin · stripe sdk.

## Trigger

Kicks off when it is time to "pull balance and recent charges from stripe" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Pull balance and recent charges from Stripe
2. Label income to Launchpad Cohort
3. Record the snapshot for the income chart
4. Flag anomalies against the trailing average
5. Note upcoming payouts so cash flow is never a surprise

## Definition of done

The run is complete when "note upcoming payouts so cash flow is never a surprise" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Dana Whitfield ([[person-dana]]). Never fake a green run.

## Pillar

[[pillar-finances]]
