---
title: Run the Launchpad Cohort lane
kind: sop
generated: founder-os
---

# Run the Launchpad Cohort lane

## Purpose

Webinar registrants to closed LC deals.

## Owner

[[launchpad-cohort-sales]] — one worker, one job (monogamous by design).
Runs on: builtin · account lane.

## Trigger

Kicks off when it is time to "track lc leads from webinar registration to booked call" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Track LC leads from webinar registration to booked call
2. Chase no-shows with the rebooking sequence within 24 hours
3. Sync every stage change back to Ledger
4. Reconcile LC payments against Stripe
5. Report lane revenue to the pipeline brief

## Definition of done

The run is complete when "report lane revenue to the pipeline brief" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Marco ([[person-marco]]). Never fake a green run.

## Pillar

[[pillar-sales]]
