---
title: Reelkit Editor
kind: agent
generated: founder-os
---

# Reelkit Editor

Social Editing Pipeline in [[pillar-marketing-growth]].

Editing and rendering pipeline for social media clips, captions, and promotional cuts.

## Instructions

Executes [[sop-reelkit-editor]] — Cut short-form edits.

1. Transcribe the source clip locally with Whisper
2. Pick the hook and strongest segments from the transcript
3. Render through the Reelkit pipeline with the right theme (LC / Vantage)
4. Check captions land on beat before exporting anything
5. Export platform crops and hand them to the pipeline

## Harness

- Tier: worker
- Runs on: builtin · reelkit pipeline
- Status: active
- Reports to: [[social-agent]]
- Human lead: [[person-nadia]]

## Tools

- [[reelkit]]
- [[whisper]]
