---
title: Broadcast directives across the fleet
kind: sop
generated: founder-os
---

# Broadcast directives across the fleet

## Purpose

One message in, every agent briefed, replies collected.

## Owner

[[conductor]] — one worker, one job (monogamous by design).
Runs on: builtin · fan-out runtime.

## Trigger

Kicks off when it is time to "receive the directive from the operator console" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Receive the directive from the operator console
2. Resolve the target list: the whole fleet, or the pillar the directive names
3. Poll instance hosts (Clawline, Ollama, tmux) for availability before dispatch
4. Fan the message out to every target at once and stamp each send
5. Collect replies as they land and file the run to agent_runs
6. Report non-responders after sixty seconds so nothing fails silently

## Definition of done

The run is complete when "report non-responders after sixty seconds so nothing fails silently" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to the operator. Never fake a green run.

## Pillar

[[pillar-tech]]
