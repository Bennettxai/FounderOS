---
title: Cut short-form edits
kind: sop
generated: founder-os
---

# Cut short-form edits

## Purpose

Raw footage to platform-ready crops.

## Owner

[[reelkit-editor]] — one worker, one job (monogamous by design).
Runs on: builtin · reelkit pipeline.

## Trigger

Kicks off when it is time to "transcribe the source clip locally with whisper" — on cadence or on the upstream event, whichever lands first.

## Steps

1. Transcribe the source clip locally with Whisper
2. Pick the hook and strongest segments from the transcript
3. Render through the Reelkit pipeline with the right theme (LC / Vantage)
4. Check captions land on beat before exporting anything
5. Export platform crops and hand them to the pipeline

## Definition of done

The run is complete when "export platform crops and hand them to the pipeline" has verifiably happened and is logged to the run history.

## Escalation

If any step fails twice in a row, or the data looks wrong, stop and escalate to Nadia ([[person-nadia]]). Never fake a green run.

## Pillar

[[pillar-marketing-growth]]
