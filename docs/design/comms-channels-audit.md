# Communications · Channels — UX Audit

**Author:** Leon (UI/UX, Wave 1)
**Branch:** `feat/communications-module`
**Audit date:** 2026-05-02
**Subject:** [`frontend/src/routes/(app)/communication/channels/+page.svelte`](../../frontend/src/routes/(app)/communication/channels/+page.svelte) — 692 lines, single-file implementation
**Scope:** UX, visual design, interaction, copy, token compliance. Backend stubbing is documented in [`COMMUNICATIONS_AUDIT.md`](../COMMUNICATIONS_AUDIT.md) and is not duplicated here except where it changes what users see.

> **Note on prior cleanup.** Some of the issues an audit would normally flag (`any[]` typing, three decorative icon-only buttons, inline color literals, silent errors) were resolved on 2026-05-02 in a Wave 1 hygiene pass that crossed lanes — Leon (this author) shipped them under the wrong scope before recognising the actual Wave 1 deliverable was this doc. Those items are still listed in the relevant sections marked **`[Resolved 2026-05-02]`** so the redesign captures the full context Fantem should know about. They do not require re-doing.

---

## TL;DR

The current Channels tab is a competent two-pane Slack viewer built in a single 692-line file. The structure (sidebar of channels + main message stream + bottom input) is the right shape — but six problems compound:

1. **Single-provider single-workspace.** UI is hard-coded to Slack. The schema and audit both call out Microsoft Teams as the second provider, and the generic [`channels` / `channel_messages`](../../desktop/backend-go/internal/database/schema.sql#L2336) tables already model both — but no Teams handler/types exist today, and the frontend has no concept of "workspace" or "provider".
2. **DMs invisible-as-such.** The Slack backend returns DMs through `GET /channels?dms=true` with `is_dm=true` flag — but the page doesn't filter or partition them. DMs and channels render in one undifferentiated list.
3. **Threads invisible.** `Message.thread_ts` and `Message.reply_count` are returned by the backend ([`messages.go:18-33`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33)) but ignored by the renderer. A 50-message reply thread shows as 50 top-level messages, polluting the channel.
4. **Reactions are dead UI.** The page renders three hardcoded reaction icons (`thumbs-up`/`party`/`eye`) but the Go [`Message`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33) struct doesn't expose `reactions` over HTTP. The schema's `channel_messages.reactions` JSONB column ([`schema.sql:2387`](../../desktop/backend-go/internal/database/schema.sql#L2387)) exists but isn't surfaced. Dead branch.
5. **Backend metadata ignored.** `unread_count`, `last_activity`, `topic`, `purpose`, `is_edited`, `attachments` are all on the wire and dropped on the floor.
6. **Single 692-line file.** Same blocker the email page had: nothing is composable, parallel work conflicts on every change.

The single-file shape and the missing thread/DM/workspace concepts are the core of the redesign. Token compliance is fine post-cleanup; multi-provider IA is the work.

---

## A. Architecture & file structure

### A.1 — Single 692-line page, no decomposition
**Severity:** High · **Where:** entire file

The whole feature lives in `channels/+page.svelte`: state, helpers, sidebar, header, message list, input, and ~360 lines of `<style>`. The neighbouring [`calendar/+page.svelte`](../../frontend/src/routes/(app)/communication/calendar/+page.svelte) (567 lines, the proven template) decomposes into ten components in [`lib/components/calendar/`](../../frontend/src/lib/components/calendar/) — sidebar, toolbar, status banner, three views, two modals, plus `calendarUtils.ts`. The page itself orchestrates; components own rendering. Leah's spec for [`comms-email-spec.md §1.1`](./comms-email-spec.md#11--component-breakdown) follows the same shape. Channels needs the same treatment.

### A.2 — Page directly calls the typed API; no service layer for provider abstraction
**Severity:** High · **Where:** [`channels/+page.svelte:5-15`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L5-L15)

The page imports Slack-specific functions directly (`getSlackConnectionStatus`, `getSlackChannels`, `sendSlackMessage`). When Wave 2 adds Teams (Ghost wave 2 — see [`COMMUNICATIONS_SPRINT_PLAN.md`](../COMMUNICATIONS_SPRINT_PLAN.md)), every call-site duplicates. There needs to be a provider-agnostic client (`$lib/api/comms/channels`) that the page consumes; per-provider clients live one layer below it. Spec describes the shape.

### A.3 — Init runs once in `onMount`; no `$effect` for state-driven reloads
**Severity:** Medium · **Where:** [`channels/+page.svelte:128-130`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L128-L130)

`onMount(() => { checkSlackConnection(); })` is the only trigger. Switching providers (Wave 2), filtering DMs/Channels/Threads, or re-loading after a reconnect all need explicit reactive paths. Calendar's pattern at [`calendar/+page.svelte:140-142`](../../frontend/src/routes/(app)/communication/calendar/+page.svelte#L140-L142) reads dependencies inside an `$effect`. Standardise on that. Email spec's [§A.2](./comms-email-audit.md#a2--effect-re-fetches-on-every-reactive-read) makes the same call.

### A.4 — `previewChannels` is a literal mock inside the page
**Severity:** Low · **Where:** [`channels/+page.svelte:132-136`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L132-L136)

A three-row literal used only by the not-connected empty state. Acceptable for v1; for the redesign it moves into the empty-state component as a prop default. Cosmetic.

---

## B. Information architecture & content

### B.1 — Single workspace, single provider
**Severity:** High · **Where:** [`channels/+page.svelte:139-330`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L139-L330) (entire UI)

Sidebar reads `Channels` (literal), the only connect path is `Connect Slack`, the only disconnect path is `Disconnect`. Schema ([`channels.provider`, `external_workspace_id`, `external_workspace_name`](../../desktop/backend-go/internal/database/schema.sql#L2342-L2345)) supports multiple providers and multiple workspaces per provider; the UI doesn't.

Wave 2 needs:
- A workspace switcher above the channel groups (analog to Leah's [§3.2 provider switcher](./comms-email-spec.md#32--provider-switcher-view-section)). Multiple workspaces per provider are out of scope for v2 — Slack's OAuth is per-workspace and the average user has one — but the structure must accept multiples without re-shaping the sidebar.
- Each Slack/Teams workspace renders as a section with channel + DM groups underneath.
- "All conversations" pseudo-workspace at the top when ≥ 2 connected.

### B.2 — DMs not separated from channels
**Severity:** High · **Where:** [`channels/+page.svelte:202-222`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L202-L222) (sidebar list), [`slack/handler.go:48`](../../desktop/backend-go/internal/integrations/slack/handler.go#L48) (`GET /channels?dms=true`)

The backend partitions via the `dms` query param (and `is_dm=true` field on `Channel`); the frontend never reads either. Channels and DMs intermix in one alphabetical list. This is the second-highest-impact UX defect after threads.

DMs are categorically different conversations (1:1 / group chat, no topic/purpose, no member browser, presence-relevant). Slack and Teams both surface them as a separate sidebar section. Redesign matches.

### B.3 — Threads invisible
**Severity:** High · **Where:** message-list rendering [`channels/+page.svelte:267-297`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L267-L297)

Backend exposes `Message.thread_ts` (parent message timestamp) and `Message.reply_count` ([`messages.go:18-33`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33)). Renderer ignores both. A 50-message reply thread shows as 50 top-level messages.

This is the top single-UX-defect — it makes channels feel noisy and lossy in the same way unthreaded email did. Redesign collapses threads in the main stream and opens them in a side drawer (see [`comms-channels-spec.md §6`](./comms-channels-spec.md#6--message-stream--thread-drawer)).

### B.4 — Reactions are dead UI
**Severity:** High · **Where:** [`channels/+page.svelte:279-294`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L279-L294)

The page renders `message.reactions` with three hardcoded icon variants (`thumbs-up`, `party`, `eye`). The TypeScript [`SlackMessage.reactions`](../../frontend/src/lib/api/slack/types.ts#L60) is optional. The Go [`Message`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33) struct **does not include `reactions`** — they're never on the JSON response. Schema's `channel_messages.reactions` JSONB column ([`schema.sql:2387`](../../desktop/backend-go/internal/database/schema.sql#L2387)) is populated by the sync path but not exposed.

Redesign keeps reactions in the UI (they're critical to channel UX) but requires Ghost to expose the JSONB column on the Message JSON response in Wave 2. Without that, render branch deletes.

### B.5 — Mentions, saved, threads, activity — none of the canonical sections
**Severity:** Medium · **Where:** sidebar layout [`channels/+page.svelte:179-230`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L179-L230)

Slack and Teams both surface these as top-level groups: **Activity** (mentions, replies to your messages, reactions to your messages), **Threads** (every thread you've replied to), **Saved** (bookmarked messages), **Drafts** (per-channel drafts). All four are missing. Wave 2 spec includes Activity and Threads as required; Saved + Drafts deferred to Wave 3 (no schema yet for either).

### B.6 — Channel metadata dropped: topic, purpose, last_activity, unread_count
**Severity:** Medium · **Where:** various

| Field | On wire | Used |
|---|---|---|
| `topic` ([`channels.go:23`](../../desktop/backend-go/internal/integrations/slack/channels.go#L23)) | yes | no |
| `purpose` ([`channels.go:24`](../../desktop/backend-go/internal/integrations/slack/channels.go#L24)) | yes | no |
| `last_activity` ([`channels.go:26`](../../desktop/backend-go/internal/integrations/slack/channels.go#L26)) | yes | no — channels render in name order, not activity order |
| `unread_count` ([`channels.go:25`](../../desktop/backend-go/internal/integrations/slack/channels.go#L25)) | yes | no — every row is treated as zero-unread |
| `member_count` | yes | yes (header only, [`+page.svelte:249`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L249)) |

The redesign uses all five: header shows topic and member count, sidebar sorts by `last_activity`, sidebar shows unread badges per channel.

### B.7 — Sidebar has no search
**Severity:** Low · **Where:** sidebar [`channels/+page.svelte:179-230`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L179-L230)

When the user has > 30 channels (typical), scrolling to find one is the only path. Slack and Teams both ship a sidebar filter input. Spec adds one.

### B.8 — Channel selection lost on reload
**Severity:** Low · **Where:** [`channels/+page.svelte:100-103`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L100-L103)

`selectedChannel` lives in component state with no URL/localStorage persistence. Refreshing the page lands you back at the channel-not-selected empty state. Calendar persists `viewMode` to localStorage at [`calendar/+page.svelte:111-116`](../../frontend/src/routes/(app)/communication/calendar/+page.svelte#L111-L116); spec mirrors that pattern with key `comms.channels.lastChannelId` per workspace.

---

## C. Decorative / non-functional UI

These rendered but did nothing useful in the pre-Wave-1 file. Tracked here for the audit record. Redesign re-introduces only those with a Wave 2 owner.

| Control | Original location | Pre-cleanup state | Action |
|---|---|---|---|
| Search messages (header) | `:235-237` (pre-cleanup) | No `onclick` | **Removed** Wave 1 (Leon, 2026-05-02). Spec re-introduces in v2 wired to in-channel search. |
| More options (header) | `:238-240` (pre-cleanup) | No `onclick` | **Removed** Wave 1. Spec re-introduces as a kebab menu with channel actions (mute, leave, view members). |
| Attach file (input) | `:301-303` (pre-cleanup) | No `onclick`, no file input | **Removed** Wave 1. Spec re-introduces in v2 wired to multipart upload (Ghost wave 2 owns). |
| Reactions row (message) | [`:279-294`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L279-L294) | Renders, but data never arrives | **Keep**, but require Ghost to expose `reactions` on Message JSON in Wave 2 (see B.4). If Ghost can't ship it in Wave 2, the render branch deletes. |
| `Disconnect` button (sidebar footer) | [`:226-228`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L226-L228) | Wired and works | **Keep**, but in v2 it moves into a workspace-level overflow menu — disconnecting one workspace shouldn't blow away the entire sidebar. |
| `Sync Channels` button (sidebar header) | [`:185-192`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L185-L192) | Wired and works | **Keep**, becomes per-workspace in v2. |

---

## D. Token violations

All concrete violations that existed in the pre-Wave-1 file were resolved in the 2026-05-02 hygiene pass. Listed for record so Fantem's redesign doesn't accidentally regress.

> **Divergence from Leah's structure.** The email audit's [§D table](./comms-email-audit.md#d-token-violations) enumerates *current* violations Fantem must fix. The channels equivalent is *resolved violations* Fantem must not regress. Same enforcement bar, different state.

| File:line (pre-cleanup) | Original violation | Replaced with | Status |
|---|---|---|---|
| `:483` | `color: #fff;` on message avatar | `var(--bos-avatar-default-text)` | Resolved |
| `:391`, `:557`, `:645` | `transition: background 0.15s;`, `transition: border-color 0.2s;` | Off-scale duration; should be `var(--bos-transition-fast)` (`bos-variables.css:210`) | **Not fixed** — Wave 2 cleanup |
| `:355`, `:435`, `:467`, `:541` | `padding: 14px 16px`, `12px 20px`, `16px 20px` | Off-scale spacing; should be `--space-*` tokens | **Not fixed** — Wave 2 |
| `:482`, `:553` | `border-radius: 8px`, `12px` | Off-scale radius; should be `--radius-md` (8px) / `--radius-lg` (12px) | **Not fixed** — Wave 2 |
| `:360`, `:367`, `:388`, `:449`, `:459`, `:487`, `:504`, `:510`, `:515`, `:535`, `:557`, `:601`, `:608`, `:625`, `:653`, `:661`, `:670` | `font-size: 0.7rem`, `0.78rem`, `0.82rem`, `0.83rem`, `0.85rem`, `0.88rem`, `0.95rem`, `1.15rem`, `0.72rem` | Off-scale typography; should snap to `--text-xs` (0.75rem) / `--text-sm` (0.875rem) / `--text-base` (1rem) / `--text-lg` (1.125rem) | **Not fixed** — Wave 2 |
| `:343` | `width: 220px` | Off-scale; document the override centrally or move to `--bos-sidebar-width` token | **Not fixed** — Wave 2 (note: matches Leah's email-spec sidebar width — they should share a token) |

The post-Wave-1 page is hex/rgba-clean (verified by grep). What remains is **off-scale durations, spacings, radii, and typography** — same anti-patterns Leah flagged for email. Spec resolves all of them by routing every CSS value through tokens.

---

## E. State coverage — empty / loading / error

### E.1 — Empty states are inconsistent
**Severity:** Medium · **Where:** four locations

| State | Location | Shape | Notes |
|---|---|---|---|
| Not connected | [`:144-176`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L144-L176) | 64×64 round icon + title + body + CTA + preview list | Canonical pattern — keep |
| Sidebar with no channels | [`:195-201`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L195-L201) | Inline text + Sync button | Bare |
| Channel with no messages | [`:261-264`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L261-L264) | Tiny icon + "No messages yet in this channel" | Bare |
| No channel selected | [`:323-326`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L323-L326) | Tiny icon + "Select a channel to view messages" | Bare |

Same problem the email audit flagged: only the "not connected" state has a coherent design language. The other three are placeholder text. Redesign normalises against the not-connected pattern's rhythm: icon → title → one sentence → optional secondary action.

### E.2 — Loading is just a spinner; no skeletons
**Severity:** Medium · **Where:** three locations ([`:140-143`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L140-L143) initial, [`:256-260`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L256-L260) message load, no sidebar skeleton)

Initial load wipes the entire pane to show a centred spinner. Switching channels wipes the message stream to "Loading messages…" + spinner. No skeleton anywhere. Spec adds skeletons for sidebar (3 channel rows) and message stream (5 message blocks).

### E.3 — Errors went to console (resolved)
**Severity:** Was High · **Where:** all six async paths

> **`[Resolved 2026-05-02]`** Wave 1 hygiene pass replaced every `console.error` with `toast.error` from `svelte-sonner` ([`:46`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L46), `:57`, `:68`, `:82`, `:94`, `:112`, `:124`).

> **Divergence from Leah's spec.** The email spec [§9.1](./comms-email-spec.md#91--toast-cm-toast) introduces a comms-namespaced `cm-toast` helper. The channels page is currently using `svelte-sonner` (the toast lib already installed in the repo, used by other modules). One of two paths must be chosen at the module level — it's wasteful for one tab to use one and the other to use the other. **Open question for Axis:** standardise on `svelte-sonner` repo-wide and skip `cm-toast`, or build `cm-toast` for the comms module and migrate the channels Wave 1 fix? Recorded in [`comms-channels-spec.md §9.1`](./comms-channels-spec.md#91--toast-channel) and Section 16 (open questions).

### E.4 — Error inside a channel → blank pane, no retry
**Severity:** High · **Where:** [`:67-70`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L67-L70)

`loadMessages` failure leaves `channelMessages = []` and renders the empty-state ("No messages yet in this channel"). The toast appears, but it's transient — the user lands on a misleading "this channel is empty" message that contradicts reality. Spec adds a dedicated error state with `Try again` action, mirroring email [§5.6](./comms-email-spec.md#56--error-state).

### E.5 — Send failure: optimistic clear is reverted, but no toast detail
**Severity:** Low · **Where:** [`:74-86`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L74-L86)

If `sendSlackMessage` rejects, the input clears (line 79) before the await finishes — actually no, looking again: the clear happens after the await, so on rejection the input keeps its content. Good. But the toast says "Failed to send message" with no retry. Spec adds `[Retry]` action like Leah's email §10.3.

---

## F. Accessibility

### F.1 — Generally good
ARIA labels present on every icon-only button (sync at [`:187`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L187), connect at [`:157`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L157), channel rows at [`:208`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L208), disconnect at [`:226`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L226), input at [`:309`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L309), send at [`:316`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L316)). Good baseline — keep.

### F.2 — No live region for sync / send
**Severity:** Low · **Where:** [`:88-98`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L88-L98), [`:74-86`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L74-L86)

`isSyncing` / `isSending` are visual state only. Screen readers get no announcement. Same gap email has — Leah's [§F.2](./comms-email-audit.md#f2--live-region-missing-for-sync--send). Toast pattern handles it implicitly when wired.

### F.3 — No keyboard navigation between channels or messages
**Severity:** Low (Wave 3) · **Where:** —

`↑/↓` does nothing in the sidebar; in messages, the only keystroke is `Enter` in the input. No `Esc` to leave the channel, no `j/k` for sidebar navigation. Slack-style shortcuts are Wave 3; spec stubs the key-handler hook in `ChannelsSidebar` and `ChannelMessageList` so Wave 3 fills it.

### F.4 — Focus management on channel switch
**Severity:** Low · **Where:** [`:100-103`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L100-L103)

After clicking a sidebar channel, focus stays on the sidebar button. For a chat surface, focus should move to the message input on each new channel selection (matches Slack/Teams behaviour). Spec adds a `tick()` + `inputEl.focus()` in the channel-select handler.

### F.5 — Disconnect button has no confirmation
**Severity:** Medium · **Where:** [`:226`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L226)

`Disconnect` immediately revokes OAuth and clears the sidebar — no confirmation. A misclick wipes the connection. Spec adds a "Disconnect Slack? You'll need to re-authorise to see channels again." prompt.

---

## G. Visual / layout

### G.1 — Sidebar 220px fixed, no collapse
**Severity:** Low · **Where:** [`:343`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L343)

Hard width. On narrow viewports, the message stream cramps. Calendar's pattern wraps the sidebar in a collapsible container with a transition; spec mirrors that. Same as Leah [§G.1](./comms-email-audit.md#g1--sidebar-width-fixed-at-185px-list-at-320px).

### G.2 — Avatars: one-letter initial; backend has no avatar URL for Slack
**Severity:** Medium · **Where:** [`:269-272`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L269-L272), [`messages.go:18-33`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33)

The channel-message renderer takes the first character of the sender name and stamps it on `var(--bos-avatar-default)`. Looks acceptable but every conversation feels stripped of identity.

The schema's [`channel_messages.sender_avatar`](../../desktop/backend-go/internal/database/schema.sql#L2381) column exists, but the Slack `Message` Go struct doesn't write it (or expose it). Slack's `users.info` API returns `image_72`/`image_192` URLs; sync should populate the column.

Redesign: `MessageAvatar` component takes `sender_avatar` URL and falls back to the initial-on-color pattern. Ghost adds avatar fetching to the Slack sync (Wave 2, small scope) — flagged in Section H.

### G.3 — Time format inconsistent across surfaces (resolved)

> **`[Resolved 2026-05-02]`** Wave 1 added `formatMessageTime(message.sent_at)` returning a localised `HH:MM` string. Sidebar last-activity timestamps would also benefit from a relative formatter (`2m ago` / `Yesterday` / `Tue`); spec defines `formatLastActivity` in `commsChannelsUtils.ts`.

### G.4 — Reactions UI shape is heavy
**Severity:** Low · **Where:** [`:521-537`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L521-L537)

Each reaction is a pill with a 1px border, padding `2px 8px`, full bg surface. On a busy message, the reaction row visually dominates. Slack uses a flatter chip with `--dbg2` background, no border, smaller padding. Spec lightens.

### G.5 — Own messages indistinguishable
**Severity:** Low · **Where:** [`:267-296`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L267-L296)

No visual differentiation between messages from the user vs. others. Slack and Teams both subtly ghost-highlight your own posts (right-aligned bubble, lighter bg, or just an indented "(you)" tag). Spec adopts the right-aligned-bubble pattern for own messages (matches the chat module's `UserMessage.svelte` shape, see [chat/messages/UserMessage.svelte](../../frontend/src/lib/components/chat/messages/UserMessage.svelte)). Mimics the layout, doesn't reuse the component (different metadata).

### G.6 — Edited indicator missing
**Severity:** Low · **Where:** message rendering ignores [`Message.is_edited`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33)

`is_edited` is on the wire. Slack/Teams show a small "(edited)" suffix next to the time. Spec adds.

### G.7 — Consecutive messages from same sender don't group
**Severity:** Low · **Where:** [`:267-297`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L267-L297)

Five messages in a row from the same person each render with full avatar + sender + time. Slack collapses consecutive sender messages into a single avatar+sender header followed by stacked bodies. Spec adds a `groupBySender` rule in `commsChannelsUtils.ts` with a 5-minute boundary.

### G.8 — No date dividers between days
**Severity:** Low · **Where:** message stream

Scrolling through yesterday's and today's messages, there's no `── Today ──` divider. Slack/Teams both insert one when the calendar date changes between consecutive messages. Spec adds.

---

## H. Data shape & integration concerns (UX-visible)

These aren't backend bugs — they affect what the user sees.

### H.1 — `attachments` field ignored
**Severity:** Medium · **Where:** [`messages.go:18-33`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33), renderer

`Message.attachments` is on the type. Slack messages with file uploads, link previews, and rich attachments all silently lose them. Spec adds `MessageAttachmentList` component (mirrors email's `EmailAttachmentList` shape from Leah [§6.5](./comms-email-spec.md#65--attachments)). Initially v2 only renders file attachments; rich Slack-block attachments (link previews, polls, etc.) are Wave 3.

### H.2 — `reactions` not on the wire (covered B.4)
**Severity:** High · **Where:** Ghost wave 2

Audit dependency: Ghost adds `reactions []byte (JSONB pass-through)` to the Slack `Message` JSON response. Without it, the reactions UI deletes.

### H.3 — `topic` / `purpose` ignored on header
**Severity:** Low · **Where:** [`channels/+page.svelte:235-251`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L235-L251)

Slack channels' "what is this channel about" lives in `topic` (short, often emoji-decorated) and `purpose` (longer, multi-sentence). Header shows neither. Spec adds a single-line topic under the channel name, with a hover/click to expand to the full purpose.

### H.4 — `unread_count` ignored in sidebar
**Severity:** Medium · **Where:** sidebar rows ignore `Channel.unread_count`

Backend returns it; sidebar treats every channel as zero-unread, no badge. Spec wires per-row badges (small blue dot for non-zero unread, count chip for ≥ 5).

### H.5 — `last_activity` not used for ordering
**Severity:** Low · **Where:** sidebar list

Channels render in insertion order. Slack/Teams default to "channels you've recently engaged with" first; both expose this through `last_activity` or its analog. Spec sorts by `last_activity DESC` within each section.

### H.6 — No avatar on Slack `Message` (covered G.2)
Ghost wave 2: extend Slack sync to populate `channel_messages.sender_avatar` (or the equivalent on `slack_messages` if Ghost keeps the provider-specific table — see audit Section J on the dual-write decision).

### H.7 — Send payload field divergence (resolved)

> **`[Resolved 2026-05-02]`** Pre-cleanup, the page used `api.sendSlackMessage` (umbrella) which sent `{ content }`; the Slack handler binds `text` as required ([`handler.go:226`](../../desktop/backend-go/internal/integrations/slack/handler.go#L226)). Send was failing 400 silently. Wave 1 switched to the typed `$lib/api/slack` client which sends `{ text }`. Sends work now.

---

## I. Light + dark theme

### I.1 — Generally honest (post-cleanup)
The page uses `--dt`/`--dbg`/`--dbd` aliases throughout. No `.dark` selectors — the alias layer themes correctly. Verified post Wave 1 hygiene.

### I.2 — Pre-cleanup exception (resolved)

> **`[Resolved 2026-05-02]`** A single `color: #fff` on the message avatar at the original `:483` was replaced with `var(--bos-avatar-default-text)` — themed token in both modes.

### I.3 — Off-scale typography may look thin in dark mode
**Severity:** Low · **Where:** various font-size literals

The off-scale `0.7rem`/`0.72rem` body sizes (G.6 in section D) are at the edge of legibility in dark mode where contrast is lower. Snapping to `--text-xs` (0.75rem) restores readability. Cosmetic but real.

---

## J. Architecture decision — where Slack's data lives

This is a Wave 1 audit *flag for Axis* rather than a bug. Worth surfacing because the spec depends on the answer.

The repo has two Slack data tables:
- [`slack_channels`](../../desktop/backend-go/internal/database/schema.sql#L2484), [`slack_messages`](../../desktop/backend-go/internal/database/schema.sql#L2518) — provider-specific, populated by the current sync, exposed by the current API.
- [`channels`](../../desktop/backend-go/internal/database/schema.sql#L2336), [`channel_messages`](../../desktop/backend-go/internal/database/schema.sql#L2372) — provider-generic, with a `provider` column, designed for the unified multi-provider story. Populated by **nothing** today.

When Wave 2 adds Teams, Ghost has three options:

| Option | Description | Cost |
|---|---|---|
| **A. Dual-write** | Slack sync writes both `slack_*` (raw cache) and `channels`/`channel_messages` (unified). Teams sync writes only `channels`/`channel_messages`. New `GET /api/comms/channels` reads the unified tables. | Low. Backwards-compatible — existing Slack endpoints keep working. |
| **B. Migrate** | Backfill `slack_*` → `channels`/`channel_messages`. Drop `slack_*`. Both providers write to the unified tables. New endpoint reads the unified tables. | Medium. One-time backfill. Existing Slack endpoints retire. |
| **C. Query-time union** | Keep tables as-is. New `GET /api/comms/channels` does a UNION ALL across `slack_channels` + (future) `microsoft_teams_channels`. | Low to write, high cost at query time once data scales. Accumulates schema debt. |

Frontend doesn't care which Ghost picks **as long as the response shape matches the spec**. Spec [`comms-channels-spec.md §1.3`](./comms-channels-spec.md#13--unified-response-shape) defines the response. **My recommendation:** Option A — dual-write. Lowest risk, no migration, leaves the Slack-specific table intact for raw cache use cases. Axis confirms before Ghost commits.

---

## K. Summary — what the redesign must do

In priority order. Each item maps to a section in the spec doc.

1. **Decompose** the page into ~10 components mirroring the calendar pattern and Leah's email decomposition.
2. **Workspace + provider concept** at the top of the sidebar, with channels/DMs grouped under each workspace.
3. **DMs as a separate section** in every workspace's sidebar group.
4. **Threads as a side drawer** — main stream collapses replies into a "View thread (N replies)" affordance; clicking opens a right-side drawer with the full thread.
5. **Reactions on the wire** — Ghost exposes `reactions` JSONB on the Slack Message JSON response. UI keeps the reaction row.
6. **Channel header reflects topic + purpose + member count + actions menu**.
7. **State coverage** — three empty states with consistent rhythm, skeletons for sidebar and message stream, dedicated error state with retry.
8. **Provider/workspace metadata used** — unread badges, activity sorting, real avatars (Ghost wave 2 backfills `sender_avatar`).
9. **Token compliance** — every off-scale value replaced with the appropriate token. Zero hex/rgb literals (already true post-cleanup; spec keeps it that way).
10. **Toast story aligned** with the email tab — Axis decides between `svelte-sonner` (current) and `cm-toast` (Leah's spec).

The spec at [`docs/design/comms-channels-spec.md`](./comms-channels-spec.md) is concrete enough that Fantem can implement without further design questions. Three open questions are surfaced there as Section 16 for Axis to resolve before Wave 2 starts.

---

## Appendix · Where this audit diverges from Leah's email audit

For Fantem's cross-reference. These are the structural deltas — the section letters/order are intentionally identical.

| Leah (email) | Leon (channels) | Why different |
|---|---|---|
| §B has `B.1` "no provider concept" and `B.4` "threads flat" as separate items | §B has `B.1` "no workspace/provider", `B.2` "no DM separation", `B.3` "threads invisible", `B.4` "reactions dead UI" — *four* IA items, not two | Channels has more axes: provider × workspace × channel-type (channel/DM) × thread-state. Email collapses to provider × folder × thread |
| §C lists *current* decorative buttons that need wiring or removal | §C lists *removed* decorative buttons (Wave 1 did the removal early, in Leon's misfire) and *re-introduction* requirements | Order of operations difference |
| §D enumerates current token violations | §D enumerates resolved violations + remaining off-scale value violations | Wave 1 hygiene happened on channels but not email |
| §E `cm-toast` is introduced as the canonical toast | §E flags that channels already uses `svelte-sonner` and asks Axis to align | The cleanup pass standardised channels on `svelte-sonner` ahead of the email spec being written |
| §G covers email-specific concerns (HTML body styling, two-line row, ASCII reply quote) | §G covers channels-specific concerns (own-message styling, edited indicator, sender grouping, date dividers, reactions chip shape) | Different domains |
| No equivalent | §J added — schema decision Axis must make about `slack_*` vs unified `channels` tables | Email has only one source-of-truth table (`emails` with `provider` col); channels has divergent state Ghost must reconcile |

Section letters A, B, C, D, E, F, G, H, I, K (was J in Leah's) match Leah's. The new §J is inserted between I and K (Leah's "Summary"). Fantem reading both side-by-side will find the same shape with Leon's added decision flag.
