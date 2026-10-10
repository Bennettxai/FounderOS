# Communications Module — Elite UI Upgrades

**Branch:** `feat/communications-module`
**Date:** 2026-05-02
**Status:** Brainstorm + tier ranking, not yet sliced into tasks.

These upgrades are constrained by the existing design system (`docs/COMMUNICATIONS_DESIGN_SYSTEM.md`):

- One accent: `--bos-accent-blue` (`#3B82F6`). No second accent.
- Status colors only for icon/badge differentiation, never as a UI accent.
- Reuse existing classes: `btn-pill`, `btn-pill-cta`, `btn-cta`, `glass-card`, `bos-modal`, `bos-input`.
- Two themes — every change must work in light and dark via tokens.

---

## Email — 10 ideas

### 1. Triage as first-class status (Linear-style inbox)
Replace the binary `is_starred` toggle with an inbox disposition: `Inbox / Snoozed / Done / Pinned`. Each row gets a status pill on the right. Keyboard: `e` archives, `s` snoozes, `p` pins, `u` un-archives.

**Tokens:** existing status backgrounds (`--bos-status-info-bg`, `--bos-status-success-bg`, `--bos-status-accent-bg`).

### 2. "Catch me up" thread summary card
For threads with 4+ messages, render a collapsed AI summary at the top of `EmailThreadView`. Uses the comms-category purple icon to signal AI provenance. Click to expand individual messages.

**Component:** new `<CommsSummaryCard>` (shared with channels — see channel #4 below).

### 3. Inline action-item extraction
Detected tasks render as ✓/○ chips below the subject. Recipient can claim them — claiming creates a BusinessOS task linked back to the email. Sender's claimed items show with `--bos-status-success` tint; pending stay neutral.

**Reuses:** `CalendarEventModal` already has an action-item extractor for meeting notes — same backend call, same UI primitive.

### 4. Sender hover-card with CRM context
Hover any `from_name` → 280px popover with avatar, last meeting date, deal stage, recent threads, jump-to-CRM link. Pure positioning + `glass-card`. Solves the "who is this person" problem inline.

### 5. AI quick-reply pills
Three context-aware reply suggestions render as `btn-pill-soft` under the latest message. Click loads the suggestion into compose for editing. Suggestions are grounded in the user's own past replies and BusinessOS records (project, client).

### 6. Schedule-send + auto-follow-up in one surface
The Send button (`btn-pill-cta`) gets a chevron menu:
- Send now
- Send at 9am tomorrow
- Send & follow up if no reply in 3d
- Send & follow up if no reply in 7d

The follow-up ladder is the differentiator vs. plain schedule-send.

### 7. Thread timeline rail
Optional left rail in thread view: vertical avatar timeline with hour markers. 40-message threads become scannable. Same pattern Linear uses for issue history. Toggle on/off per user preference.

### 8. Smart compose mentions
`@`, `#`, `$` triggers BusinessOS autocomplete in compose:
- `@person` → contact (CRM)
- `#project` → project
- `$client` → client

Inserts a structured chip. On send, chips serialize to plain-text references for external recipients but keep structured data for internal recipients (links resolve in the BusinessOS reader).

### 9. Compose: split-preview
Right-pane shows the email exactly as the recipient will see it (HTML render with their dark/light mode variant if known). Catches signature/font issues before send. Toggle preview on/off.

### 10. Linked artifacts in compose
Drag a task, doc, or calendar event into compose → it embeds as a card preview with a deep link. External recipient sees a clickable summary card; internal BusinessOS users see a live, current-state object. Reuses existing wiki/doc card patterns.

---

## Channels — 10 ideas

### 1. "Pulse" hub at top of channels page
Horizontal strip above the channel list:

```
Mentions (3) · DMs (2) · Threads I'm in (5) · Saved (1)
```

Each tab is a `btn-pill-soft` with a status-bg unread count. One-click cross-workspace triage. Replaces the "scan every sidebar" loop.

### 2. Thread → Wiki page in one click
Any thread can be converted to a wiki doc (BusinessOS already has the wiki module). Header gets a "Convert to doc" action. Powerful for decisions that should outlive the thread — captures the rationale in a durable surface.

### 3. Reactions as workflow signals
Define a small reaction set with semantics:

- 🎯 "I'll handle this" → creates a task assigned to the reactor
- 👀 "watching" → adds to your follow list
- ✅ "approved" → marks the parent message resolved
- 🚫 "blocked" → flags the parent, surfaces in a "blocked" digest

Optimal Engine pattern: reaction-as-trigger.

### 4. Inline "Catch me up" summary
Above an unread run, show a dismissable AI summary card. **Same component as email idea #2** — `<CommsSummaryCard>` shared across email + channels + (eventually) calendar pre-meeting briefs.

### 5. Channel → automation rule
Channel header gear → "When messages match X, do Y". Surfaces the Optimal Engine right where users feel the pain ("we keep getting bug reports here, auto-create issues"). Three example presets:
- New mentions of `bug` / `error` → create issue
- Messages from `@boss` → push notification + star
- Files in this channel → mirror to wiki section

### 6. Presence-on-thread dots
Tiny avatar stack at the top of each thread showing who's currently reading. Updates live via the existing comms stream. Status colors: `--bos-status-success` (online), `--bos-status-neutral` (away).

### 7. Code-block actions
Detect language, render with syntax highlighting (already in place), add hover actions: `Copy / Open in osa terminal / Save as snippet`. Connects channels to the terminal module — copy a snippet straight into a workspace shell.

### 8. Voice notes with transcript
Attach a voice memo from compose; auto-transcribe to text shown inline; original audio plays on click. **Reuses the voice-notes module already in the routes tree.**

### 9. Channel modes
Per-channel mode selector: `Discussion (default) / Broadcast / Standup`.
- **Discussion:** what we have today.
- **Broadcast:** hides reactions, adds read receipts. For announcements.
- **Standup:** swaps the compose for a structured prompt (Yesterday / Today / Blockers). Auto-collates into a daily digest.

Same surface, different affordances based on channel intent.

### 10. Message as artifact
Drag a task, calendar event, or contact into compose → structured embed. Same primitive as email #10. Across email and channels, this is the BusinessOS-native message-attachment platforms like Slack lack.

---

## Cross-cutting opportunities

Three ideas (Email #2, Channel #4, plus future calendar pre-meeting briefs) all use the same "AI summary card" surface. Build it once as `<CommsSummaryCard>` with the comms-category icon and `glass-card` shell — every other thread/feed/channel can adopt it free.

Two ideas (Email #10, Channel #10) share the "linked artifact" embed primitive. Build a `<CommsArtifactEmbed>` once; both modules get it.

---

## Tier ranking

### Tier A — ship now, big-feel, low-build
- Email #1 — Triage status (existing tokens, mostly UI work)
- Email #6 — Schedule + follow-up (compose menu + a small backend cron)
- Channels #1 — Pulse hub (cross-tab activity strip)
- Channels #4 — Catch me up (shared summary card)

### Tier B — shippable but needs backend wiring
- Email #5 — AI quick-reply pills
- Email #10 / Channels #10 — Artifact embeds (shared primitive)
- Channels #3 — Reactions as workflow signals

### Tier C — depends on Optimal Engine surface
- Email #4 — CRM hover-card
- Channels #5 — Channel automation rules
- Channels #9 — Channel modes

---

## Design system anchors used

| Idea | Anchors |
|---|---|
| Email #1 | `--bos-status-*-bg`, `btn-pill-soft` |
| Email #2 / Channels #4 | `glass-card`, `--bos-category-communication`, comms-category icon |
| Email #3 | `--bos-status-success-bg`, neutral chip pattern |
| Email #4 | `glass-card`, popover positioning |
| Email #5 | `btn-pill-soft` |
| Email #6 | `btn-pill-cta` (the new blue CTA) |
| Email #7 | Avatar tokens, `--bos-divider-color` |
| Email #8 | `bos-input` autocomplete, structured-chip pattern |
| Email #9 | Two-pane modal, `bos-modal` |
| Email #10 / Channels #10 | Wiki/doc card preview pattern |
| Channels #1 | `btn-pill-soft`, status-bg counts |
| Channels #2 | Wiki module integration |
| Channels #3 | Reaction popover + status-tint feedback |
| Channels #5 | `bos-modal`, automation rule editor |
| Channels #6 | Avatar stack, status dots |
| Channels #7 | Code block + `btn-compact-ghost` action row |
| Channels #8 | Voice-notes module integration |
| Channels #9 | Tabs/segmented control pattern (`ct-view-toggle` analogue) |
