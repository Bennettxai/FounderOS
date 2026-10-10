# Communications · Channels — Redesign Spec

**Author:** Leon (UI/UX, Wave 1)
**Branch:** `feat/communications-module`
**Spec date:** 2026-05-02
**Audience:** Fantem (frontend implementation, Wave 2). Ghost for backend touch-points.
**Companion docs:** [`comms-channels-audit.md`](./comms-channels-audit.md) · [`comms-email-spec.md`](./comms-email-spec.md) · [`COMMUNICATIONS_DESIGN_SYSTEM.md`](../COMMUNICATIONS_DESIGN_SYSTEM.md) · [`COMMUNICATIONS_AUDIT.md`](../COMMUNICATIONS_AUDIT.md) · [`COMMUNICATIONS_SPRINT_PLAN.md`](../COMMUNICATIONS_SPRINT_PLAN.md)

---

## 0. How to read this spec

Each section has:

- **Goal** — why this surface exists
- **Layout** — ASCII wireframe and structural anatomy
- **Tokens** — every color / space / radius / typography by name
- **Component** — which file, which props, which existing primitives it composes
- **Behavior** — interactions, states, edge cases
- **Acceptance** — what Leon checks against in Wave 2 visual QA

If a section is silent on a token or behaviour, default to the [design system doc](../COMMUNICATIONS_DESIGN_SYSTEM.md). No new tokens are introduced. If implementation needs one, raise to Axis before adding it.

**Naming convention:** all new components live under [`frontend/src/lib/components/comms/channels/`](../../frontend/src/lib/components/comms/channels/) (new directory, parallel to Leah's [`comms/email/`](../../frontend/src/lib/components/comms/email/)). All new CSS classes are `cm-channels-*` (sub-feature namespace).

**Cross-reference convention.** This spec is structurally parallel to [`comms-email-spec.md`](./comms-email-spec.md). Every divergence from Leah's structure is called out inline with `**Divergence from email spec:**` and rolled up in [Appendix A](#appendix-a--divergences-from-the-email-spec). Section numbers match Leah's wherever possible — if Leon adds a sub-section that doesn't exist for email, it's marked.

---

## 1. Module architecture

### 1.1 — Component breakdown

The page decomposes into eleven components, mirroring the [calendar template](../../frontend/src/routes/(app)/communication/calendar/+page.svelte) at `:420-509` and Leah's email pattern at [§1.1](./comms-email-spec.md#11--component-breakdown).

```
routes/(app)/communication/channels/+page.svelte    ← orchestrator only (~150 lines)
├─ ChannelsStatusBanner.svelte           (workspace re-auth, sync status, multi-workspace state)
├─ ChannelsToolbar.svelte                (channel header: name + topic + member count + actions menu)
├─ ChannelsSidebar.svelte                (workspace switcher, per-workspace sections, DM section, system sections)
│   ├─ WorkspaceSwitcher.svelte          (radio-group of connected workspaces + "All conversations")
│   ├─ ChannelsSidebarSection.svelte     (collapsible group: Channels / DMs / Threads / Activity)
│   ├─ ChannelsSidebarRow.svelte         (single channel or DM row, with unread badge)
│   └─ ChannelsSidebarSearch.svelte      (filter input above all sections)
├─ ChannelMessageList.svelte             (infinite-scroll message stream, grouping, date dividers)
│   ├─ ChannelMessageGroup.svelte        (consecutive same-sender messages stacked under one avatar+sender header)
│   ├─ ChannelMessage.svelte             (single message body + reactions row + reply-link)
│   └─ ChannelMessageDateDivider.svelte  (── Today ── / ── Mon May 12 ── separators)
├─ ChannelThreadDrawer.svelte            (right-side slide-in drawer, collapses main stream's right edge)
│   └─ ChannelThreadReplyInput.svelte    (compact compose bar inside the drawer)
├─ ChannelComposeBar.svelte              (the bottom-of-stream input row: message field + send + attach + emoji)
├─ ChannelsEmptyState.svelte             (4 variants: not-connected, no-channels-synced, no-messages, no-channel-selected, error-channel-load)
└─ commsChannelsUtils.ts                 (formatLastActivity, groupBySender, splitByDate, partitionByType, channelLabel, workspaceLabel)
```

The page-level state (selected workspace, selected channel/DM, open thread, compose draft, sidebar section collapse states) lives in `+page.svelte`. Children receive callbacks (`onSelectChannel`, `onOpenThread`, `onSwitchWorkspace`, …) and props — no `createEventDispatcher`. This matches the calendar module's contract.

> **Divergence from email spec:** Leah's [§1.1](./comms-email-spec.md#11--component-breakdown) has ten components anchored around a three-pane layout (sidebar, list, preview). Channels has eleven and a different split: there's no separate "list" pane — the sidebar acts as the navigation, and the main pane is a single message stream. The thread drawer is the second pane only when active. See [§2.1](#21--two-pane-primary-thread-drawer-as-third).

### 1.2 — Reuse map

Reused without modification:

| Existing | Used in | Why |
|---|---|---|
| [`PillButton.svelte`](../../frontend/src/lib/components/ui/PillButton.svelte) | every button surface | Standard. Variants: `primary` (Connect, Send), `ghost` (sidebar disconnect, kebab options), `danger` (Disconnect confirm), `soft` (View thread, Reply) |
| `.btn-cta` class (`app.css:1303`) | none in Wave 2 (channels has no signature CTA — see note below) | Reserved for Wave 3 `New message` global compose if it lands |
| `.btn-compact-*` classes (`app.css:2212+`) | toolbar, message hover actions, drawer header | Already used by current code; correct for dense rows |
| [`SkeletonLoader.svelte`](../../frontend/src/lib/components/ui/SkeletonLoader.svelte) | sidebar loading rows, message stream skeleton, drawer skeleton | Existing skeleton primitive |
| [`ErrorBoundary.svelte`](../../frontend/src/lib/components/ui/ErrorBoundary.svelte) | wraps sidebar + main pane + drawer individually | One pane crash doesn't kill the others |
| [`Tooltip.svelte`](../../frontend/src/lib/components/ui/Tooltip.svelte) | icon-only buttons, sender avatars | Hover hint. Don't replace `aria-label` — both |
| `.bos-input` / `.bos-textarea` (`app.css:3185-3248`) | compose bar input, sidebar search, drawer reply input | Standard form chrome |
| [`ChatAttachments.svelte`](../../frontend/src/lib/components/chat/input/ChatAttachments.svelte) pattern | `ChannelComposeBar` attachment chip row (Wave 2 if Ghost ships uploads, else Wave 3) | Mirror the file-chip layout (don't import — different upload contract, but same visual) |
| [`ChatInputBar.svelte`](../../frontend/src/lib/components/chat/input/ChatInputBar.svelte) shape | reference only | Provides the textarea + actions-row layout. Channels needs a thinner version; mimic, don't import (different submit contract, no model selector, no voice) |
| `Hash`, `Lock`, `MessageSquare`, `Send`, `Loader2`, `Users`, `RefreshCw`, `Smile`, `Paperclip`, `AtSign`, `MoreHorizontal`, `X`, `ChevronRight`, `Plus`, `Search`, `Bell`, `BellOff`, `LogOut`, `Reply`, `Bookmark` from `lucide-svelte` | everywhere | Existing icon vocabulary. Leah uses some of these too — see [§ icons](#icon-inventory) for the channels-specific add-ons |

**Not reused (don't try):**

- [`lib/components/chat/messages/UserMessage.svelte`](../../frontend/src/lib/components/chat/messages/UserMessage.svelte) / [`AssistantMessage.svelte`](../../frontend/src/lib/components/chat/messages/AssistantMessage.svelte) — these target the AI chat surface, with `role`-driven styling and tool-call rendering. Wrong shape for a Slack/Teams message (different metadata: channel, thread_ts, reactions, edited, reply_count). Mimic the bubble layout; build new components.
- [`lib/components/chat/conversations/ConversationListItem.svelte`](../../frontend/src/lib/components/chat/conversations/ConversationListItem.svelte) — wrong shape for a sidebar channel row (different metadata: provider, is_dm, unread_count, member_count, topic). Build a new `ChannelsSidebarRow.svelte`.
- [`lib/components/chat/messages/TypingIndicator.svelte`](../../frontend/src/lib/components/chat/messages/TypingIndicator.svelte) — Wave 3 will adopt it once realtime presence ships. Out of scope for Wave 2.
- [`lib/components/forms/FormField.svelte`](../../frontend/src/lib/components/forms/FormField.svelte) — uses daisyUI classes (`.input input-bordered`), not the `.bos-input` system. The compose bar uses `.bos-textarea` directly.

> **Divergence from email spec:** Leah's [§1.2](./comms-email-spec.md#12--reuse-map) reuses `DocumentEditor` for rich-text email body. Channels v2 is **plain text + emoji shortcodes + @mentions only** — no rich text editor. `DocumentEditor` is intentionally absent from the channels reuse map. Wave 3 may revisit if channel formatting parity becomes a goal.

### 1.3 — Unified response shape (frontend contract)

> **New section, no email-spec equivalent.** Email's backend already has one source-of-truth table (`emails` with a `provider` column). Channels has divergent state — Slack writes to `slack_*`, Teams doesn't exist. Frontend needs the response shape committed to *now* so it can render against either backing-table choice Ghost makes (see [`comms-channels-audit.md §J`](./comms-channels-audit.md#j--architecture-decision--where-slacks-data-lives)).

Frontend consumes a new provider-agnostic API client at [`frontend/src/lib/api/comms/channels/`](../../frontend/src/lib/api/comms/channels/) with this shape:

```typescript
export interface CommsChannel {
  id: string;                    // local UUID (channels.id)
  provider: 'slack' | 'teams';
  workspace_id: string;          // external_workspace_id
  workspace_name: string;        // external_workspace_name
  external_id: string;           // provider's channel id (Slack channel id, Teams channel id)
  name: string;
  description?: string;
  topic?: string;
  is_private: boolean;
  is_archived: boolean;
  is_dm: boolean;
  member_count: number;
  unread_count: number;
  last_message_at?: string;      // ISO 8601, drives sidebar ordering
}

export interface CommsMessage {
  id: string;                    // local UUID (channel_messages.id)
  channel_id: string;
  external_id: string;
  sender_id?: string;
  sender_name?: string;
  sender_avatar?: string;        // URL — Ghost backfills from Slack users.info / Graph
  content: string;               // plain text
  content_html?: string;         // rendered HTML (mentions, channel links, formatting marks)
  attachments: CommsAttachment[];
  reactions: CommsReaction[];    // empty array if none
  mentions: string[];            // user ids mentioned in content
  thread_ts?: string;            // timestamp of the thread root if this is a reply
  parent_message_id?: string;    // local UUID of the parent if reply
  reply_count: number;
  is_thread_root: boolean;
  is_edited: boolean;
  is_deleted: boolean;
  sent_at: string;               // ISO 8601
  edited_at?: string;
}

export interface CommsAttachment {
  id: string;
  name: string;
  size_bytes: number;
  mime_type: string;
  url?: string;                  // signed download URL (Ghost wave 2)
  thumbnail_url?: string;
}

export interface CommsReaction {
  emoji: string;                 // canonical name e.g. "thumbsup", "tada", "eyes"
  count: number;
  reacted_by_me: boolean;
}
```

Routes Fantem expects (Ghost owns implementation; spec defines contract):

| Method · path | Returns | Notes |
|---|---|---|
| `GET /api/comms/workspaces` | `{ workspaces: { provider, workspace_id, workspace_name, connected_at }[] }` | Drives the workspace switcher |
| `GET /api/comms/channels?workspace_id=&type=channel\|dm\|all` | `{ channels: CommsChannel[] }` | Type filter is server-side; default `all`. Sorted by `last_message_at DESC`. |
| `POST /api/comms/channels/sync?workspace_id=` | `{ synced: number, failed: number }` | Per-workspace |
| `GET /api/comms/channels/:id/messages?limit=&before=` | `{ messages: CommsMessage[], has_more: boolean }` | Paginated, descending. `before` = ISO timestamp. |
| `GET /api/comms/channels/:id/threads/:thread_ts/messages` | `{ messages: CommsMessage[] }` | Thread root + replies, ascending. |
| `POST /api/comms/channels/:id/messages` | `{ id: string }` | Body: `{ text: string, thread_ts?: string }`. `thread_ts` present means reply-in-thread. |
| `POST /api/comms/channels/:id/messages/:message_id/reactions` | `{ reactions: CommsReaction[] }` | Body: `{ emoji: string }`. Toggles. |
| `DELETE /api/comms/channels/:id/messages/:message_id/reactions` | `{ reactions: CommsReaction[] }` | Same body shape. |
| `GET /api/comms/activity?workspace_id=` | `{ items: ActivityItem[] }` | Mentions + thread-replies-to-you (Wave 3 hook; Wave 2 stub returns empty array) |

If any of those routes don't ship in Wave 2, Fantem stubs them and the corresponding UI section renders the empty state. Spec marks each route's required-for-Wave-2 status in the schedule mini-table at [§14](#14--out-of-scope-deliberately).

---

## 2. Page layout

### 2.1 — Two-pane primary, thread drawer as third

```
┌──────────────────────────────────────────────────────────────────────────┐
│  StatusBanner    [ Slack workspace expired · [Reconnect] ]              │  ← optional, only when notable
├──────────────────────────────────────────────────────────────────────────┤
│ Sidebar       │ Toolbar  # general                          [↻] [⋯]    │
│ (240px)       │            5 members · "Engineering announcements"      │
│               ├───────────────────────────────────────────────────────  │
│ Workspace     │ ─── Mon May 12 ─────────────────────────────────────    │
│ ◉ All        │                                                          │
│ ○ Acme Slack  │ ┌──┐ Alice Chen          10:42                          │
│ ○ Beta Teams  │ │AC│ Hey, can someone review the deploy plan?           │
│               │ └──┘                                                     │
│ ── Channels   │       💬 3 replies · last reply 11:03                   │
│ # general 12  │                                                          │
│ # design  3   │ ┌──┐ Bob Park              10:48                         │
│ # eng         │ │BP│ Looking now.                                        │
│ 🔒 launch     │ │  │ Done — left comments.                                │
│               │ └──┘   👍 2  🎉 1                                        │
│ ── DMs        │                                                          │
│ • Alice 1     │ ─── Today ───────────────────────────────────────────   │
│ • Bob, Carol  │                                                          │
│               │ ┌──┐ Alice Chen          09:14                           │
│ ── Activity   │ │AC│ ↪ replied in thread                                 │
│   3 mentions  │ └──┘                                                     │
│               │                                                          │
│ ── Threads 4  │                                                          │
│               ├───────────────────────────────────────────────────────  │
│ [🔍 Search]   │ Compose: [Message #general                ] [@] [😀] [📎]│
│               │                                              [↑ Send]   │
└────────────────┴──────────────────────────────────────────────────────────┘

When a thread is opened, the layout becomes three panes:

┌──────────────────────────────────────────────────────────────────────────┐
│ Sidebar       │ Toolbar  # general          │ Thread                    │
│ (240px)       │             ...              │ Reply to Alice's plan    │
│               │                              │ ────────────────────────  │
│               │ <main stream continues>      │ ┌──┐ Alice  10:42        │
│               │                              │ │AC│ Hey, can…           │
│               │                              │ └──┘                     │
│               │                              │                          │
│               │                              │ ┌──┐ Bob    10:50        │
│               │                              │ │BP│ Looking…            │
│               │                              │ └──┘                     │
│               │                              │                          │
│               │                              │ [Reply in thread…]       │
│               │                              │ [↑ Send]      [✕ Close]  │
└────────────────┴──────────────────────────────┴──────────────────────────┘
```

Sidebar is always present. Main pane is the message stream. Thread drawer slides in from the right when a thread is opened, sharing the main pane's horizontal space (main shrinks). Drawer width: `380px`. On viewports < 1280px the drawer becomes a full-screen overlay (covers main); on ≥ 1280px it slides alongside.

> **Divergence from email spec:** Leah's [§2.1](./comms-email-spec.md#21--three-pane-structure-desktop--1024px) is a static three-pane layout (sidebar + list + preview). Channels is two panes by default, with a third drawer that's only present when a thread is open. The reasoning: channel UX is dominated by the live stream; threads are episodic and shouldn't permanently consume horizontal space.

### 2.2 — Tokens

| Surface | Property | Token |
|---|---|---|
| Page background | bg | `var(--dbg)` |
| Sidebar | bg | `var(--dbg2)` |
| Sidebar | border-right | `1px solid var(--dbd)` |
| Sidebar | width | `240px` (was 220 — workspace switcher and Activity badge need more room) |
| Sidebar | padding | `var(--space-3) 0` |
| Main pane | bg | `var(--dbg)` |
| Main pane | flex | `1` |
| Thread drawer | bg | `var(--dbg)` |
| Thread drawer | border-left | `1px solid var(--dbd)` |
| Thread drawer | width | `380px` (≥ 1280px viewports), `100%` (< 1280px overlay mode) |
| Thread drawer | shadow (overlay mode only) | `var(--bos-shadow-3)` |
| Thread drawer transition | width / transform | `var(--bos-transition-slow)` |
| All panes | min-height | `0` (allow internal scroll) |

> **Divergence from email spec:** Sidebar width is `240px` here vs Leah's `220px`. The channels sidebar has more rows (workspace switcher + four section types) and unread badges that are more numerous. This is one of the open-questions for Axis at [§16](#16--open-questions-for-axis--roberto) — settle on a single shared `--bos-sidebar-width` token across both tabs.

### 2.3 — Responsive

- **≥ 1280px:** sidebar + main + drawer (when active) side-by-side.
- **1024–1279px:** sidebar + main side-by-side; drawer becomes a full-pane overlay covering the main stream when active.
- **640–1023px:** sidebar collapses to icon rail (channel-icon + dm-icon + activity-icon, no labels). Main + drawer behave as in the previous tier. Compose bar's actions condense (`Send` stays as a primary; `Attach` and `Emoji` move into a `+` overflow).
- **< 640px:** single pane. Sidebar opens as a left-drawer via a hamburger button; main stream takes the screen; drawer covers main. Mobile is Wave 3, but components avoid hard widths in inner content.

Wave 2 ships desktop only. Same expectation as email.

---

## 3. Sidebar

### 3.1 — Anatomy

```
┌──────────────────────┐
│ [🔍 Filter…]         │  ← search input, sticky at top
├──────────────────────┤
│ ▼ Workspace          │  ← collapsed by default if only 1 workspace
│   ◉ All conversations│  active state: bg = --bos-nav-active-bg, fg = --bos-nav-active
│   ○ Acme Slack       │
│   ○ Beta Teams       │
│                      │
│ ▼ Channels       18  │
│   # general      12  │
│   # design        3  │
│   # eng              │
│   🔒 launch          │  private channel — Lock icon, not Hash
│   # marketing        │
│                      │
│ ▼ Direct Messages 4  │
│   • Alice Chen    1  │  unread dot at left, name, optional unread count
│   • Bob, Carol       │  group DM — comma-separated names
│   • Sarah Wong       │
│                      │
│ ▼ Activity        3  │  ← all mentions and replies-to-you across workspaces
│ ▼ Threads         4  │  ← every thread you've replied to, with unread reply badge
│                      │
├──────────────────────┤  ← divider
│ + Add workspace      │  small text-xs link
└──────────────────────┘
```

**Section headers** ("Workspace", "Channels", "Direct Messages", "Activity", "Threads") render as collapsed-state click targets that hide/show the section. State persists in `localStorage` with key `comms.channels.sidebarSections` (mirrors Leah's `comms.email.sidebarSections`). Default: all expanded.

> **Divergence from email spec:** Email has three sections (View, Folders, Accounts). Channels has five (Workspace, Channels, DMs, Activity, Threads). The IA difference is real and intentional — channel-style messaging has more axes than email.

### 3.2 — Workspace switcher (`WorkspaceSwitcher`)

`SlateChoice` rendering — radio rows with provider icon + workspace name + connection state dot.

| Option | When shown | Filter applied |
|---|---|---|
| All conversations | Always when ≥ 2 workspaces connected | No filter |
| {Workspace name} | Workspace exists | `workspace_id=…` |

If only one workspace is connected, the section auto-collapses and is replaced by an "Add another workspace" affordance at the bottom of the sidebar.

Each row tokens:

| Surface | Property | Token |
|---|---|---|
| Row | padding | `var(--space-2) var(--space-4)` |
| Row | font | `var(--text-sm)` |
| Row | color (idle) | `var(--dt2)` |
| Row | color (active) | `var(--bos-nav-active)` |
| Row hover | bg | `var(--dbg3)` |
| Row active | bg | `var(--bos-nav-active-bg)` |
| Row active | font-weight | `var(--font-semibold)` |
| Provider icon | size | `14px` (Slack 'S' or Teams 'T' in a small colored square) |
| Workspace name | flex | `1`, ellipsis at `--text-sm` |
| Connection dot | size | `8px`, color = `--bos-status-success` (healthy) / `--bos-status-warning` (re-auth needed) / `--bos-status-error` (failed last sync) |

> **Divergence from email spec:** Leah's [§3.2](./comms-email-spec.md#32--provider-switcher-view-section) is a provider switcher (Gmail / Outlook). Leon's is a workspace switcher (one row per OAuth grant). The reasoning: a user typically has one Gmail account and one Outlook account, but might be in three Slack workspaces (work, side project, customer). Per-workspace is the right granularity. "All conversations" subsumes Leah's "All inboxes".

### 3.3 — Channels list (per active workspace scope)

Section header: `Channels` + total channel count. Below, one row per channel, sorted by `last_message_at DESC`.

| Field | Source | Notes |
|---|---|---|
| Icon | Hash if public, Lock if `is_private`, Archive if `is_archived` (faded) | 13px |
| Name | `channel.name` | Truncate with ellipsis |
| Unread count | `channel.unread_count` | Badge if > 0; small dot only if 1–4, count chip if ≥ 5 |
| Active indicator | derived from selected state | bg `--dbg3`, text `--dt`, font-weight semibold |

Tokens:

| Surface | Property | Token |
|---|---|---|
| Row | padding | `var(--space-1) var(--space-4)` (denser than email folder rows — channels are more numerous) |
| Row | font | `var(--text-sm)` |
| Row | color (idle) | `var(--dt2)` |
| Row | color (unread) | `var(--dt)` / `var(--font-semibold)` |
| Row hover | bg | `var(--dbg3)` |
| Row active | bg | `var(--dbg3)` |
| Row active | color | `var(--dt)` |
| Row active | font-weight | `var(--font-semibold)` |
| Row icon | size | `13px` |
| Row icon color | `var(--dt3)` |
| Unread dot | size | `7px` |
| Unread dot | bg | `var(--bos-accent-blue)` |
| Unread count chip | bg | `var(--bos-accent-blue)` |
| Unread count chip | text | `var(--bos-surface-on-color)` |
| Unread count chip | font | `var(--text-xs)` / `var(--font-bold)` |
| Unread count chip | radius | `var(--radius-full)` |
| Unread count chip | padding | `1px var(--space-2)` |

### 3.4 — DMs list (per active workspace scope)

Section header: `Direct Messages` + total DM count. Below, one row per DM, sorted by `last_message_at DESC`.

DM row anatomy is *almost* the same as the channel row but with two differences:

1. **Icon is a presence dot or initial avatar.** For 1:1 DMs: the other person's initial in a `12px` circle. For group DMs: the first three initials overlapped. (Initial-first; `sender_avatar` is a Wave 3 enhancement for DMs since Slack doesn't surface the participant avatar on the channel object.)
2. **Name is participants.** For 1:1: the other person's name. For group: comma-separated names truncated to fit (`Alice, Bob, Carol +2`).

Tokens — same as channel row, plus:

| Surface | Property | Token |
|---|---|---|
| DM avatar | size | `18px` (smaller than message avatars; the row is denser) |
| DM avatar | bg | `var(--bos-avatar-default)` |
| DM avatar | text color | `var(--bos-avatar-default-text)` |
| DM avatar | font | `var(--text-xs)` / `var(--font-bold)` |
| DM avatar | radius | `var(--radius-full)` |

> **Divergence from email spec:** Leah's email sidebar has no DM analog. Channels' DM section has no email parallel.

### 3.5 — Activity section

Single section row that, when expanded, lists mentions and thread-replies-to-you across the active workspace scope. Maximum 20 items shown; "View all" at the bottom navigates to a dedicated Activity view (Wave 3).

Wave 2 ships the section header and badge. Items inside the expanded section depend on `GET /api/comms/activity` (Wave 2 stub OK if backend not ready — the item list renders empty with a helper message "We'll show mentions and thread replies here.").

Header tokens — same as section headers across the sidebar.
Item tokens — same as channel row, with a small icon on the left indicating type:

| Type | Icon | Source |
|---|---|---|
| Mention | `AtSign size={12}` color `--bos-accent-blue` | `mentions` array on `CommsMessage` |
| Thread reply | `Reply size={12}` color `--dt3` | `parent_message_id` resolves to a message you authored |

### 3.6 — Threads section

Section row. Expanded state lists every thread the current user has participated in across the active workspace scope, with an unread-replies badge.

Same shape as Activity. Items are clickable: clicking opens the thread drawer for that thread, navigating to the parent channel as well.

Wave 2 acceptable behaviour: Threads section ships visible but empty if the activity endpoint is stubbed.

### 3.7 — Sidebar search (`ChannelsSidebarSearch`)

A `.bos-input` at the top of the sidebar (sticky), placeholder "Filter channels & DMs…". Filters the channel and DM lists client-side by `name` + `participants` substring match. Does **not** filter Activity or Threads sections.

| Surface | Property | Token |
|---|---|---|
| Search wrapper | padding | `var(--space-2) var(--space-3)` |
| Search wrapper | border-bottom | `1px solid var(--dbd)` |
| Search wrapper | bg (sticky) | `var(--dbg2)` |
| Search input | base | `.bos-input` |
| Search input | font | `var(--text-xs)` |
| Search input | padding | `var(--space-1) var(--space-2)` |
| Search icon | size | `12px`, color `var(--dt3)` |

### 3.8 — Footer

A single `+ Add workspace` link at the bottom. Triggers OAuth for the not-yet-connected provider (Slack if Teams is connected, Teams if Slack is connected, or a chooser modal if both).

> **Divergence from email spec:** Leah's email sidebar has a *Compose CTA* at the footer ([§3.4](./comms-email-spec.md#34--compose-cta-sidebar-footer)). Channels has no global compose — sending happens inside an open channel via the bottom-bar compose. So the footer is just the "Add workspace" affordance. **No `.btn-cta` glow surface in the channels tab.** Reserved for Wave 3 if a "New direct message" command palette is added.

---

## 4. Toolbar

### 4.1 — Anatomy

The toolbar is the channel header — the equivalent of email's folder header, but channel-specific.

```
┌────────────────────────────────────────────────────────────────────┐
│ # general · 12 members                                  [↻] [⋯]   │
│ "Engineering announcements and water-cooler talk"                   │
└────────────────────────────────────────────────────────────────────┘
```

- Left: channel name (with hash/lock icon prefix), separator, member count clickable to open member list (Wave 3 — Wave 2 just renders the count, no click handler).
- Below name: channel topic (single line, ellipsised; click expands to show purpose).
- Right: refresh (`RefreshCw`, animates on sync), kebab menu (`MoreHorizontal`) with: "Mute channel" (Wave 3), "Pin to sidebar" (Wave 3), "View members" (Wave 3 — but include the menu item, dimmed, with a "Coming soon" tooltip), "Leave channel" (Wave 3 — same), "Workspace settings" (opens the Settings → Integrations page).

For DMs, the header simplifies:

```
┌────────────────────────────────────────────────────────────────────┐
│ • Alice Chen                                            [↻] [⋯]   │
│ Last active 12 minutes ago                                          │
└────────────────────────────────────────────────────────────────────┘
```

### 4.2 — Tokens

| Surface | Property | Token |
|---|---|---|
| Toolbar | bg | `var(--dbg)` |
| Toolbar | border-bottom | `1px solid var(--dbd)` |
| Toolbar | padding | `var(--space-3) var(--space-5)` |
| Toolbar | gap | `var(--space-3)` |
| Channel label | font | `var(--text-base)` / `var(--font-semibold)` |
| Channel label | color | `var(--dt)` |
| Member count | font | `var(--text-sm)` |
| Member count | color | `var(--dt3)` |
| Channel topic | font | `var(--text-xs)` |
| Channel topic | color | `var(--dt3)` |
| Channel topic | max-width | `60vw`, ellipsis |
| Refresh button | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |
| Refresh icon (syncing) | animation | existing `ch-spin` keyframes (rename to `cm-spin` in this module — same as Leah did) |
| Kebab menu | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |

### 4.3 — Sync state

Three states for the refresh icon, mirroring Leah's [§4.3](./comms-email-spec.md#43--sync-state) verbatim:

| State | Visual | Trigger |
|---|---|---|
| Idle | static `RefreshCw` | default |
| Syncing | spinning + `--dt3` color | sync in progress |
| Error | static `AlertCircle` + `--bos-status-error` color, tooltip with message | sync just failed |

Auto-clears on next successful sync, or after 10s into idle. Silent success.

> **Divergence from email spec:** Leah's toolbar has a search input centered. Channels' toolbar does **not** — search-within-channel is a Wave 3 feature, and the sidebar already has a global filter. Adding a per-channel search would visually crowd the topic line. Spec leaves the slot for Wave 3.

---

## 5. Sidebar list rows

> **Divergence from email spec:** Leah's [§5](./comms-email-spec.md#5--email-list) is "Email list" — the middle pane that doesn't exist in channels. The equivalent rendering work for channels is split across [§3.3 (Channels)](#33--channels-list-per-active-workspace-scope) and [§3.4 (DMs)](#34--dms-list-per-active-workspace-scope) — both of which are *inside the sidebar*. There is no separate "list pane".

For Fantem cross-referencing: Section 5 is intentionally short here. The work that Leah's §5 describes is folded into Leon's §3.3 + §3.4. **Use those sections in place of §5 when implementing the channel/DM list.** Section number 5 is preserved here so cross-document references line up.

The skeleton-row shape used during sidebar load is described in [§9](#9--empty--loading--error--the-matrix).

---

## 6. Message stream + thread drawer

### 6.1 — Main stream anatomy (`ChannelMessageList` + `ChannelMessageGroup`)

The main pane is an infinite-scroll virtualised stream from oldest-loaded to newest, with date dividers and grouped consecutive sender messages.

```
─── Mon May 12 ───────────────────────────────────────────────────────

┌──┐ Alice Chen        10:42
│AC│ Hey, can someone review the deploy plan?
└──┘
        💬 3 replies · last reply 11:03 ↪

┌──┐ Bob Park          10:48
│BP│ Looking now.
│  │ Done — left comments. (edited)
└──┘     👍 2  🎉 1  +

─── Today ───────────────────────────────────────────────────────────

┌──┐ Alice Chen        09:14
│AC│ Pushed v2 to staging.
└──┘
```

**Grouping rule** (`commsChannelsUtils.groupBySender`): consecutive messages from the same `sender_id` within 5 minutes collapse under one avatar+sender header. The 5-minute window resets on:
- Sender change.
- Date change (a date divider always breaks groups).
- A reply-in-thread message (thread replies don't collapse with their parent).

**Date divider rule** (`commsChannelsUtils.splitByDate`): insert a divider when `sent_at`'s calendar date differs from the previous message's calendar date in the user's timezone. Format:
- `Today` for the current calendar day.
- `Yesterday` for one day before.
- `Day · Mon DD` (e.g. `Mon · May 12`) for ≤ 6 days ago.
- `Mon DD, YYYY` for older.

> **Divergence from email spec:** Email's "preview pane" is a single thread's contents ([§6](./comms-email-spec.md#6--thread-view-preview-pane)). Channels' main pane is the *entire channel's* infinite stream. Threads collapse inline (see §6.3) and expand into the side drawer (§6.4).

### 6.2 — Message anatomy (`ChannelMessage`)

Single message inside a group. The first message in a group has an avatar+sender+time header above the body; subsequent messages in the group are "tail" messages (just body, indented to align with the body of the head message).

**Head message:**
```
┌──┐ Sender Name       10:42 (edited)
│SN│ <body content here, plain text or HTML-rendered>
└──┘     👍 2  🎉 1  +    💬 3 replies · last reply 11:03 ↪
```

**Tail message** (no avatar/sender, time on hover only):
```
     <body content here>
```

**Own messages** are visually distinct:
- Right-aligned bubble layout: avatar on right, name/time right-aligned, body in a soft-tinted bubble (`var(--bos-nav-active-bg)`).
- Hover shows quick-actions on the *left* (opposite side from default).

> **Divergence from email spec:** Leah's email message-card has no own/other distinction (every email has its own card uniformly). Slack/Teams users expect a self-vs-other visual cue, so channels has it.

Tokens — head message:

| Surface | Property | Token |
|---|---|---|
| Group | padding | `var(--space-3) var(--space-5)` |
| Group | gap | `var(--space-2)` (between head and tail messages) |
| Group hover | bg | `var(--dbg2)` (subtle hover for hover-actions) |
| Avatar | size | `36px` |
| Avatar | radius | `var(--radius-md)` |
| Avatar | bg fallback | `var(--bos-avatar-default)` |
| Avatar | text fallback | `var(--bos-avatar-default-text)` |
| Avatar | font (initial) | `var(--text-xs)` / `var(--font-bold)` |
| Sender name | font | `var(--text-sm)` / `var(--font-semibold)` |
| Sender name | color | `var(--dt)` |
| Time | font | `var(--text-xs)` |
| Time | color | `var(--dt3)` |
| Edited tag | font | `var(--text-xs)` / `var(--font-normal)` |
| Edited tag | color | `var(--dt4)` |
| Body wrapper | font | `var(--text-sm)` |
| Body wrapper | color | `var(--dt)` |
| Body wrapper | line-height | `1.55` |
| Body wrapper | max-width | `680px` (readable measure, mirrors Leah's email body) |

Tokens — own message bubble (additions/overrides):

| Surface | Property | Token |
|---|---|---|
| Group (own) | bg | `var(--bos-nav-active-bg)` |
| Group (own) | border-radius | `var(--radius-md)` |
| Group (own) | margin-left | `auto` (right-align the bubble) |
| Group (own) | max-width | `min(680px, 70%)` |
| Group (own) avatar | order | reversed (right side) |

### 6.3 — Reply summary (collapsed thread)

When a message has `reply_count > 0`, render a clickable reply summary under the message body:

```
💬 3 replies · last reply at 11:03   ↪
```

Tokens:

| Surface | Property | Token |
|---|---|---|
| Summary row | padding | `var(--space-2) var(--space-3)` |
| Summary row | font | `var(--text-xs)` |
| Summary row | color | `var(--dt3)` |
| Summary row | bg | `var(--dbg2)` |
| Summary row hover | bg | `var(--dbg3)` |
| Summary row hover | color | `var(--dt2)` |
| Summary row | border-radius | `var(--radius-sm)` |
| Summary row | display | `inline-flex`, gap `var(--space-2)` |
| Summary row | margin-top | `var(--space-1)` |
| Summary icon (`MessageSquare`) | size | `12px` |
| Chevron icon | size | `12px`, color `var(--dt3)` |

Click → opens the thread drawer (§6.4). Same affordance is the keyboard `Enter` action when the message is keyboard-focused.

### 6.4 — Thread drawer (`ChannelThreadDrawer`)

A right-side drawer that slides in when a thread is opened. Width `380px` on ≥ 1280px viewports, full-pane overlay below.

```
┌──────────────────────────────┐
│ Thread                  [✕]   │  header
│ Reply to Alice's plan         │  one-line context — root-message subject (or first 60 chars of body)
├──────────────────────────────┤
│ ┌──┐ Alice Chen   10:42       │
│ │AC│ Hey, can someone review  │  ← root message, always shown
│ └──┘ the deploy plan?         │
│                               │
│ ── 3 replies ─────────────── │  divider
│                               │
│ ┌──┐ Bob Park    10:50        │
│ │BP│ Looking now.              │
│ └──┘                          │
│ ┌──┐ Bob Park    10:54        │
│ │BP│ Done — left comments.    │
│ └──┘                          │
│ ┌──┐ Alice Chen  11:03        │
│ │AC│ Thanks! Merging.         │
│ └──┘                          │
├──────────────────────────────┤
│ Reply to thread…              │  reply input (sticky bottom)
│ [@] [😀] [📎]        [↑ Send] │
└──────────────────────────────┘
```

Tokens:

| Surface | Property | Token |
|---|---|---|
| Drawer | bg | `var(--dbg)` |
| Drawer | border-left | `1px solid var(--dbd)` |
| Drawer | width | `380px` (≥ 1280px) / `100%` (< 1280px overlay) |
| Drawer header | padding | `var(--space-3) var(--space-4)` |
| Drawer header | border-bottom | `1px solid var(--dbd)` |
| Drawer header label "Thread" | font | `var(--text-base)` / `var(--font-semibold)` |
| Drawer header context | font | `var(--text-xs)`, color `var(--dt3)`, ellipsis |
| Drawer close button | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |
| Replies divider | border-top | `1px solid var(--dbd)` |
| Replies divider label | font | `var(--text-xs)` / color `var(--dt3)` / margin-top `var(--space-3)` |
| Reply input wrapper | bg | `var(--dbg2)` |
| Reply input wrapper | border-top | `1px solid var(--dbd)` |
| Reply input wrapper | padding | `var(--space-3) var(--space-4)` |
| Reply input | base | `.bos-textarea` with `min-height: 60px`, `max-height: 200px` |
| Reply send button | base | `<PillButton variant="primary" size="sm">` with `Send size={14}` |

Behaviour:

- Open → backend call `GET /api/comms/channels/:id/threads/:thread_ts/messages`, render skeleton meanwhile.
- Close → drawer slides out, clears local state. The triggering main-stream message stays scrolled-in-view.
- Sending a reply → optimistic append, backend call `POST /api/comms/channels/:id/messages` with `thread_ts` set. On success: replace optimistic message with real one; on failure: keep the optimistic message in error state with retry.
- Drawer state is **non-routed** in Wave 2: refreshing the page closes the drawer. Wave 3 may add `?thread=…` query param for deep-linking.

> **Divergence from email spec:** Email's "preview pane" *is* the thread view, always present. Channels' thread drawer is opened on demand. Reasoning: 90% of channel messages are non-threaded; permanently dedicating horizontal space would waste it.

### 6.5 — Reactions

Below the message body, a horizontal row of reaction chips. The current page already renders something like this — the Wave 2 redesign tightens the styling and wires it to real data.

```
👍 2   🎉 1   👀 3   +
```

Each chip:

| Surface | Property | Token |
|---|---|---|
| Chip wrapper row | gap | `var(--space-1)`, wrap, margin-top `var(--space-2)` |
| Chip | bg | `var(--dbg2)` |
| Chip | border | `none` (was 1px in current code — flatter looks better) |
| Chip | radius | `var(--radius-full)` |
| Chip | padding | `2px var(--space-2)` |
| Chip | font | `var(--text-xs)` |
| Chip | color | `var(--dt3)` |
| Chip (reacted-by-me) | bg | `var(--bos-nav-active-bg)` |
| Chip (reacted-by-me) | color | `var(--bos-nav-active)` |
| Chip hover | bg | `var(--dbg3)` |
| Chip transition | bg | `var(--bos-transition-fast)` |
| Emoji glyph | font | inherit, no special icon font |
| Count | font | `var(--text-xs)`, margin-left `var(--space-1)` |
| `+` add reaction | same chip styling, plus `Smile size={12}` icon, no count | hover reveals only when group is hovered |

Click a chip → toggles your reaction (`POST` if not reacted, `DELETE` if already). `+` opens an emoji picker (Wave 2: a small popover with the eight Slack defaults: 👍 👎 ✅ ❌ 🎉 👀 🙏 🚀; Wave 3 swaps for a full picker).

Emoji picker tokens:

| Surface | Property | Token |
|---|---|---|
| Popover | bg | `var(--dbg)` |
| Popover | border | `1px solid var(--dbd)` |
| Popover | radius | `var(--radius-md)` |
| Popover | shadow | `var(--bos-popover-shadow)` |
| Popover | z-index | `var(--bos-z-popover)` |
| Popover | padding | `var(--space-2)` |
| Picker grid | display | `grid`, columns `repeat(4, 1fr)`, gap `var(--space-1)` |
| Picker button | size | `28×28` |
| Picker button hover | bg | `var(--dbg2)` |

> **Divergence from email spec:** Reactions are a channels-only concept. No email-spec equivalent.

### 6.6 — Hover actions

When hovering a message group, a small floating action bar appears at the top-right of the group (top-left for own messages):

```
[😀] [↩] [⋯]
```

- `😀` quick-react (opens emoji picker).
- `↩` reply in thread (opens drawer with that message as root).
- `⋯` overflow with: "Copy link", "Copy text", "Edit" (own messages only, Wave 3), "Delete" (own messages only, Wave 3), "Save" (Wave 3 — bookmark/Saved section).

Tokens:

| Surface | Property | Token |
|---|---|---|
| Action bar | position | `absolute`, top `-12px`, right `var(--space-3)` |
| Action bar | bg | `var(--dbg2)` |
| Action bar | border | `1px solid var(--dbd)` |
| Action bar | radius | `var(--radius-md)` |
| Action bar | shadow | `var(--bos-shadow-1)` |
| Action bar | padding | `2px var(--space-1)` |
| Action bar | gap | `0` (buttons abut) |
| Action button | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |
| Action button | size | `24×24` |
| Action button icon | size | `14px` |
| Visibility | opacity `0`, `pointer-events: none` by default; `opacity: 1`, `pointer-events: auto` on group hover |

### 6.7 — Attachments

Render below the message body as a vertical stack of attachment cards (one card per file). Same chip styling as Leah's email [§6.5](./comms-email-spec.md#65--attachments) — Fantem can lift the visual treatment.

| Surface | Property | Token |
|---|---|---|
| Attachment list | display | `flex`, direction `column`, gap `var(--space-2)`, margin-top `var(--space-2)` |
| Card | bg | `var(--dbg2)` |
| Card | border | `1px solid var(--dbd)` |
| Card | radius | `var(--radius-md)` |
| Card | padding | `var(--space-2) var(--space-3)` |
| Card | font | `var(--text-xs)` |
| Card | color | `var(--dt2)` |
| Card hover | bg | `var(--dbg3)` |
| Card transition | bg | `var(--bos-transition-fast)` |
| File icon | size | `16px`, color `var(--dt3)` |
| Filename | max-width | `260px`, ellipsis |
| Size | color | `var(--dt3)`, margin-left `var(--space-2)` |
| Download icon | hover-only, color `var(--dt2)` |
| Image attachment thumb | max-width | `360px`, max-height `240px`, radius `var(--radius-md)` |

Click → download (uses `attachment.url` from the response shape). Image thumbnails are inline-rendered above the row.

### 6.8 — Empty state (channel with no messages)

```
┌──────────────────────────────────────────────┐
│              [MessageSquare 28px]             │
│             No messages in #general           │
│       Be the first to say hello              │
└──────────────────────────────────────────────┘
```

Tokens — same shape as Leah's empty state at [§5.4](./comms-email-spec.md#54--empty-state). Subtitle copy varies by channel type:
- Public channel: "Be the first to say hello."
- Private channel: "Start the conversation when you're ready."
- DM (1:1): "Send {Name} a message."
- DM (group): "Say hello to the group."

### 6.9 — Loading skeleton

While the first page of messages is loading, render five skeleton groups. Each skeleton group mirrors the real geometry: 36px avatar block, sender-name text-block (60% width), three body lines with widths `90%`/`80%`/`50%`. Use [`SkeletonLoader.svelte`](../../frontend/src/lib/components/ui/SkeletonLoader.svelte).

When paginating older messages on scroll, prepend three skeleton groups at the top with a short fade-in transition.

### 6.10 — Error state (channel load failure)

```
┌──────────────────────────────────────────────┐
│  ⚠  Couldn't load this channel                │
│     {error message}                            │
│     [Try again]                                │
└──────────────────────────────────────────────┘
```

Tokens — same as Leah's [§5.6](./comms-email-spec.md#56--error-state).

---

## 7. Compose: inline message bar

> **Divergence from email spec:** Leah's [§7](./comms-email-spec.md#7--compose-v2) is a full Compose modal — recipient field, subject, rich text body, attachments, send-later, draft auto-save, account picker, validation. **Channels has none of those.** Compose lives at the bottom of the message stream as an always-visible bar. Plain text only (with @mentions and emoji shortcodes), one optional thread context, optional attachments. Section 7 is therefore *significantly shorter* than Leah's.

### 7.1 — Anatomy (`ChannelComposeBar`)

```
┌──────────────────────────────────────────────────────────────────┐
│ [Message #general                                              ] │  ← textarea, autoresizes 1–6 rows
│  [@] [😀] [📎]   ⌘+Enter to send             [↑ Send]            │  ← actions row
└──────────────────────────────────────────────────────────────────┘
   [↑ chip] file.pdf · 1.2 MB ✕                                      pending attachment chips above
```

For the thread drawer's reply input, the same component is used with a different label ("Reply to thread…").

### 7.2 — Tokens

| Surface | Property | Token |
|---|---|---|
| Compose wrapper | bg | `var(--dbg2)` |
| Compose wrapper | border-top | `1px solid var(--dbd)` |
| Compose wrapper | padding | `var(--space-3) var(--space-4)` |
| Textarea | base | `.bos-textarea` |
| Textarea | font | `var(--text-sm)` |
| Textarea | min-height | `40px` (single row), max-height `160px` |
| Textarea | autoresize | `field-sizing: content` if supported, fallback to JS height calc |
| Actions row | display | `flex`, gap `var(--space-1)`, align-items `center`, margin-top `var(--space-2)` |
| Mention button | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |
| Mention icon | `AtSign size={14}` |
| Emoji button | base | same |
| Emoji icon | `Smile size={14}` |
| Attach button | base | same |
| Attach icon | `Paperclip size={14}` |
| Hint text "⌘+Enter to send" | font | `var(--text-xs)` / color `var(--dt3)` / margin-left `auto` (pushes Send to right) |
| Send button | base | `<PillButton variant="primary" size="sm">` with `Send size={14}` |
| Send button (disabled) | opacity | `0.5` (only when input empty or sending) |

### 7.3 — Behaviour

- **Submit:** `Enter` sends, `Shift+Enter` newline, `⌘/Ctrl+Enter` also sends (for users used to chat clients with reversed defaults).
- **Optimistic send:** message renders immediately at the bottom of the stream with a "sending" indicator (small spinner next to time). On success: indicator clears. On failure: message stays in error state with `[Retry]` and `[Discard]` actions.
- **Drafts:** per-channel draft persists to `localStorage` under `comms.channels.draft.{channelId}`. Restored when the channel is reopened. **No backend draft persistence in Wave 2** (Slack draft API exists but is per-app — out of scope; localStorage matches the current chat module's behaviour).
- **@mentions:** typing `@` in the textarea opens an autocomplete popover (workspace members; backed by `GET /api/comms/workspaces/:id/members?q=…`). Selecting inserts the mention as `@{user_id}` in the raw payload, rendered inline as `@Name`. Wave 2 acceptable: if backend members endpoint isn't ready, the popover is a stub showing "Type a handle and we'll send it raw" — manually-typed `@handle` strings are sent through verbatim.
- **Emoji shortcodes:** typing `:` opens an emoji shortcode autocomplete (`:smile:`, `:tada:`, `:rocket:`...). Wave 2: hardcoded list of 32 most common shortcodes; Wave 3 swaps for a full picker.
- **Attachments:** clicking Attach opens a file picker; drag-drop onto the textarea also works (highlight the wrapper border). Files appear as chips above the textarea. Multi-file accepted. Per-file size limit shown in the chip if exceeded (chip styled `var(--bos-status-error)` border, tooltip explains).
- **Send disabled state:** when textarea is empty AND no attachments, send is visibly disabled. When sending: disabled + spinner inside the button.

### 7.4 — Compose error state

If a send fails (network or backend), the optimistic message stays in place with:

```
┌──────────────────────────────────────────────┐
│ ┌──┐ You             10:42 ·  ⚠ Not sent     │
│ │YO│ original message text                    │
│ └──┘     [Retry]  [Discard]                   │
└──────────────────────────────────────────────┘
```

| Surface | Property | Token |
|---|---|---|
| Sender row label "Not sent" | font | `var(--text-xs)` |
| Sender row label "Not sent" | color | `var(--bos-status-error-text)` |
| Group bg | bg | `var(--bos-status-error-bg)` |
| Retry / Discard buttons | base | `<PillButton variant="soft" size="xs">` |

> **Divergence from email spec:** Leah's email has Send-later, draft auto-save to backend, account picker, recipient chip field, validation prompts. Channels has none. The shorter spec is intentional.

---

## 8. Status banner

### 8.1 — Anatomy & visibility

Renders at the top of the page, above the toolbar. Only when notable. Three trigger conditions, in priority order:

1. **No workspaces connected:** banner takes the *whole page* (uses the existing not-connected pattern from [`channels/+page.svelte:144-176`](../../frontend/src/routes/(app)/communication/channels/+page.svelte#L144-L176)) with two `<PillButton size="md">` buttons "Connect Slack" / "Connect Teams" (both shown; the user picks). The other panes don't render.
2. **Reauth needed for a workspace:** strip with warning icon: "Slack workspace 'Acme' authorization expired · [Reconnect]". One per affected workspace.
3. **Last sync > 1 hour ago AND active scope is a channel/DM with stale data:** info strip "Last synced {relative} · [Sync now]".

Otherwise: not rendered. Mirrors Leah's [§8](./comms-email-spec.md#8--status-banner) shape verbatim.

### 8.2 — Strip tokens

Identical to Leah's [§8.2](./comms-email-spec.md#82--strip-tokens). No reason to differ.

| Surface | Property | Token |
|---|---|---|
| Strip | padding | `var(--space-2) var(--space-4)` |
| Strip | border-bottom | `1px solid var(--dbd)` |
| Strip | gap | `var(--space-3)` |
| Strip (warning) | bg | `var(--bos-status-warning-bg)` |
| Strip (warning) text | color | `var(--bos-status-warning-text)` |
| Strip (info) | bg | `var(--bos-status-info-bg)` |
| Strip (info) text | color | `var(--bos-status-info-text)` |
| Icon | size | `16px` |
| Action button | base | `.btn-compact .btn-compact-ghost .btn-compact-sm` |

---

## 9. Empty / loading / error — the matrix

Single-source-of-truth list so Fantem doesn't miss any state.

| Surface | Empty | Loading | Error |
|---|---|---|---|
| Page (no workspaces) | "Connect Slack / Teams" full-page (§8.1) | "Connecting…" full-page spinner during OAuth callback | "Couldn't connect: {message} · [Try again]" full-page |
| Sidebar | "No channels synced — [Sync]" inline at sidebar empty | 3 skeleton rows on first load; no skeleton on subsequent loads | n/a (sidebar errors propagate via toast) |
| Main pane (no channel selected) | "Select a channel to view messages" (§6.8 layout) | n/a | n/a |
| Main pane (channel with no messages) | "No messages in #channel · Be the first…" (§6.8) | 5 skeleton groups on first load; 3 prepend-skeleton on scroll-up | "Couldn't load this channel · [Try again]" (§6.10) |
| Thread drawer | n/a (drawer doesn't open without a thread to view) | header skeleton + 3 message skeletons | "Couldn't load this thread · [Close] [Try again]" |
| Compose | n/a (always present in selected channel) | "Sending…" optimistic indicator on the message | inline error on the optimistic message with `[Retry]` `[Discard]` (§7.4) |

### 9.1 — Toast (channel)

> **Divergence from email spec.** Leah's [§9.1](./comms-email-spec.md#91--toast-cm-toast) introduces a comms-namespaced `cm-toast` helper at `lib/components/comms/CmToast.svelte`. Channels' Wave 1 hygiene pass already wired `svelte-sonner` (the toast lib already used elsewhere in the repo). Both can't be the standard.
>
> **Spec position:** **Use `svelte-sonner` for both tabs.** Reasoning: it's already installed, already used by other modules in this repo, the channels page is already on it, and `cm-toast` would duplicate functionality without a clear benefit — `svelte-sonner` supports custom variants, custom positions, and undo actions natively.
>
> **Action for Axis:** confirm `svelte-sonner` standardisation; Leah's spec rewrites §9.1 to point at `svelte-sonner` instead of `cm-toast`. Recorded in [§16](#16--open-questions-for-axis--roberto).

Tokens (when wiring `svelte-sonner` themes — set once at app shell, applied across both tabs):

| Variant | Color tokens (theme override on `<Toaster>` element) |
|---|---|
| success | bg `var(--bos-status-success-bg)`, text `var(--bos-status-success-text)`, border `var(--bos-status-success)` |
| error | bg `var(--bos-status-error-bg)`, text `var(--bos-status-error-text)`, border `var(--bos-status-error)` |
| info | bg `var(--bos-status-info-bg)`, text `var(--bos-status-info-text)`, border `var(--bos-status-info)` |

Position: bottom-right, `var(--bos-z-toast)`. Auto-dismiss: success 3s · error 8s · info 5s. Action slot: `[Undo]` link, `var(--bos-accent-blue)`, `--text-xs`. Used by reaction add/remove, leave channel, mute channel.

---

## 10. Interaction flows

### 10.1 — Open a channel

1. User clicks a sidebar row in `ChannelsSidebar`.
2. Row gets `active` class; page state sets `selectedChannelId`.
3. Main pane mounts `ChannelMessageList` with `channelId` prop. Skeleton renders.
4. Backend call `GET /api/comms/channels/:id/messages?limit=50`. Latest 50 messages render bottom-aligned.
5. URL updates to `?channel={id}` for refresh persistence (Wave 2 ships this small piece).
6. Focus moves to the compose textarea.
7. If channel has unread, mark as read in background (optimistic UI: clear the unread badge in sidebar; backend call `POST /api/comms/channels/:id/read` — Ghost wave 2 owns the route; spec defines the call).
8. On error: replace skeleton with error state (§6.10).

### 10.2 — Open a DM

Same as 10.1, with two differences: the toolbar renders DM-shape (presence dot + name, no topic), and on opening, the empty-state copy varies (§6.8).

### 10.3 — Open a thread

1. User clicks the reply summary on a message (§6.3) or the `↩` hover-action (§6.6).
2. `selectedThread` state in page is set with `{ channelId, thread_ts }`.
3. Drawer mounts `ChannelThreadDrawer`. Header renders immediately with the parent message's sender + first 60 chars. Skeleton renders below.
4. Backend call `GET /api/comms/channels/:id/threads/:thread_ts/messages`.
5. Drawer slides in (`var(--bos-transition-slow)` on width).
6. On error: drawer renders error state with [Close] and [Try again].

### 10.4 — Send a message

1. User types in compose, hits Enter (or `⌘+Enter`).
2. Validate: textarea trimmed must be non-empty OR at least one attachment.
3. Optimistic append: a message group with `id: 'temp-{nanoid}'`, sender = current user, sent_at = now, status = sending, renders at the bottom of the stream.
4. Compose textarea clears. Attachments clear.
5. Backend call `POST /api/comms/channels/:id/messages` (with `thread_ts` if in drawer).
6. On success: backend returns `{ id }`. Local state replaces the temp message id with the real one; status clears.
7. On failure: optimistic message stays, status switches to error (§7.4). User can `[Retry]` or `[Discard]`.

### 10.5 — Add / remove a reaction

1. User clicks an existing reaction chip OR the `+` button → emoji picker.
2. Optimistic toggle: chip count +1/-1 immediately, `reacted_by_me` flips.
3. Backend call `POST` (add) or `DELETE` (remove) with `{ emoji }`.
4. Backend returns the updated `reactions` array; local state replaces.
5. On failure: revert and toast "Couldn't react: {message} · [Retry]".

### 10.6 — Switch workspace

1. User clicks a workspace row in `WorkspaceSwitcher`.
2. Page state sets `selectedWorkspaceId` (or `null` for "All conversations").
3. Sidebar refilters channels and DMs by workspace_id.
4. Main pane stays on the currently selected channel if that channel belongs to the new scope; otherwise clears to "Select a channel".
5. Active selection is preserved per-workspace in localStorage (`comms.channels.lastChannelId.{workspaceId}`).

### 10.7 — Re-auth flow

1. Sidebar workspace row shows a `--bos-status-warning` dot.
2. Status banner renders with [Reconnect] action.
3. Clicking [Reconnect] re-runs the OAuth flow for that specific workspace (provider's `/auth?workspace_id=…` if Ghost supports re-auth scoping; otherwise full re-auth and show a "We'll merge your existing data" message — Ghost call).
4. On success, banner disappears; sidebar dot returns green.

### 10.8 — Disconnect a workspace

1. User opens the workspace row's overflow menu in the sidebar (Wave 2 small overflow, Wave 3 expands).
2. Selects "Disconnect…".
3. Modal: `.bos-modal` size `sm` with title "Disconnect {workspace name}?", body "You'll need to re-authorise to see channels and messages from this workspace again. Your existing local data won't be deleted.", footer buttons `[Cancel]` `<PillButton variant="danger" size="sm">Disconnect</PillButton>`.
4. On confirm: backend call `POST /api/integrations/{provider}/disconnect?workspace_id=…`, sidebar workspace section disappears, toast "Disconnected from {workspace} · [Undo not available]".

> **Divergence from email spec:** Leah's email tab has compose-side flows (Reply, Forward, Drag attachment, Send-later) at [§10.2 / 10.4 / 10.5](./comms-email-spec.md#10--interaction-flows). Channels' equivalents — react, open thread, switch workspace — are channels-specific. No 1:1 mapping.

---

## 11. Engine-sync surface

The page itself doesn't render engine signals, but the components must avoid blocking Axis's `OnMessageSaved → ModuleMessage` hook (Axis Wave 1 task in [the sprint plan](../COMMUNICATIONS_SPRINT_PLAN.md#-axis--wave-1-task-engine-sync-hooks-spec--module-level-wiring)).

The hook structure already partly exists in [`slack/messages.go:44-47`](../../desktop/backend-go/internal/integrations/slack/messages.go#L44-L47) as `OnMessageSaved` but isn't wired to `EngineSync.Enqueue()`. Axis adds the wiring; spec assumes it's in place by Wave 2.

Implication for design: when a channel is opened and its messages are marked read, the optimistic update path must include the same backend mutation that fires `OnMessageSaved` (or a sibling `OnMessageRead` hook if Axis adds one for read-state). Without it, read-state never reaches the engine. Fantem confirms with Ghost when wiring `markAsRead`.

A future Wave 3 surface (out of scope here): a "Triage hints" strip in the toolbar that surfaces Signal Theory hints from the engine — "This channel has 47 unread, 3 mentions you · [Catch up]" — derived from `ModuleMessage` signals. The component slot exists in `ChannelsToolbar` between the topic line and actions — pass through a `<slot name="triage" />` so Wave 3 fills it.

> **Mirrors Leah's [§11](./comms-email-spec.md#11--engine-sync-surface) deliberately.** Same engine, same module conceptual model, just `ModuleMessage` instead of `ModuleEmail`.

---

## 12. Accessibility commitments

- Every icon-only button keeps `aria-label`. (Continuing the existing standard from the channels Wave 1 cleanup.)
- Drawer (`role="complementary"`) has `aria-labelledby` pointing at its "Thread" header.
- Modal (`role="dialog"`, `aria-modal="true"`) for the disconnect confirmation.
- Live region (polite) at `role="status"` in `+page.svelte` announces: "New message from {sender}", "Synced N new messages", "Couldn't send".
- Keyboard:
    - In sidebar: `↑/↓` move focus through channel rows (Wave 3); `Enter` opens; `/` focuses the sidebar search.
    - In message stream: `↑/↓` move focus through messages (Wave 3); `Enter` opens hover actions; `T` opens thread on focused message; `R` quick-reply.
    - In compose: `Enter` send, `Shift+Enter` newline, `⌘/Ctrl+Enter` also send (for users used to chat clients with reversed defaults), `⌘/Ctrl+K` insert link (Wave 3).
    - In drawer: `Esc` closes drawer.
- Color contrast: every text/bg combination uses tokens that are LC in both themes (verified by spot-checking; Leon confirms during Wave 2 visual QA).
- Reduced motion: every transition wraps in `@media (prefers-reduced-motion: reduce) { transition: none; animation: none; }` at the component level. Apply to every `cm-channels-*` class that animates.
- Avatar fallback (initial-on-color) maintains LC contrast in both themes via the existing `--bos-avatar-default` / `--bos-avatar-default-text` token pair.

---

## 13. Light + dark theme commitments

- **Zero `.dark` selectors.** Tokens already theme. If Fantem catches themself writing `.dark .cm-channels-…`, stop and use a token.
- The pre-Wave-1 inline `#fff` on the avatar (now `--bos-avatar-default-text`) and the off-scale durations / paddings flagged in [`comms-channels-audit.md §D`](./comms-channels-audit.md#d--token-violations) must be tokenised in Wave 2 code.
- Visual QA in both themes is part of Leon's Wave 2 task. Leon signs off only after toggling the theme on every redesigned surface.

---

## 14. Out of scope (deliberately)

So Fantem doesn't accidentally pull these in:

- **Realtime / presence / typing.** Wave 2 is polling-based. SSE / Slack Events / Graph subscriptions are Wave 3. Components are realtime-aware (`<TypingIndicator>` slot exists, presence dot uses `--bos-status-success` / etc. with a TODO marker), but no events flow yet.
- **Edit / delete own messages.** Wave 3.
- **Saved messages / bookmarks.** Schema doesn't exist; Wave 3.
- **Per-channel drafts on the backend.** Wave 2 uses localStorage only.
- **Server-side message search.** Sidebar search filters local rows; per-channel search is Wave 3.
- **Slack rich attachment blocks** (link previews, polls, message blocks). Wave 3 with a renderer.
- **Microsoft Teams adaptive cards.** Wave 3.
- **Workspace member browser.** "View members" is in the kebab menu, dimmed.
- **Mute / leave channel.** Same — kebab menu, dimmed.
- **Thread routing.** `?thread=` query param for deep-linking is Wave 3.
- **Bulk react / bulk delete.** Wave 3.
- **Voice messages / video calls.** Out of all roadmap.
- **List virtualization.** Wave 2 uses standard scroll for ≤ 200 messages per channel; if a channel has more loaded, virtualise in Wave 3.

> **Divergence from email spec:** Leah's [§14](./comms-email-spec.md#14--out-of-scope-deliberately) defers labels, calendar invite parsing, signature management, encryption. Different domain — most of those aren't channel concerns. Channels' "out of scope" set centres on realtime, message edit/delete, and rich block rendering.

---

## 15. Acceptance — what Leon checks in Wave 2 QA

Per-component checklist applied to every commit Fantem ships:

- [ ] Component file lives under [`lib/components/comms/channels/`](../../frontend/src/lib/components/comms/channels/).
- [ ] Class names use `cm-channels-*` namespace.
- [ ] Zero `#xxxxxx`, `rgb()`, `rgba()` literals in `<style>` blocks (grep gate). Exceptions documented inline only if approved by Leon.
- [ ] Every color resolves to a `--bos-*` or `--dt`/`--dbg`/`--dbd` token.
- [ ] Every space resolves to a `--space-*` token.
- [ ] Every radius resolves to a `--radius-*` token.
- [ ] Every transition uses `--bos-transition-*`.
- [ ] Every z-index uses `--bos-z-*`.
- [ ] Modal (disconnect confirm) uses `.bos-modal*` classes.
- [ ] Buttons use `<PillButton>`, `.btn-pill-*`, `.btn-compact-*`. No raw `<button>` with custom styles.
- [ ] Inputs use `.bos-input` / `.bos-textarea`.
- [ ] Lucide icons only.
- [ ] Light AND dark theme verified.
- [ ] Every icon-only button has `aria-label`.
- [ ] Empty / loading / error variants implemented per [§9](#9--empty--loading--error--the-matrix).
- [ ] Reactions render only when backend exposes the `reactions` field; render branch is dead code if Ghost doesn't ship it.
- [ ] Thread drawer slides without horizontal scroll glitches in both themes and both viewport tiers (≥ 1280px slide-in, < 1280px overlay).
- [ ] Optimistic send / react / read mutations all have failure-state recovery paths.
- [ ] No new design tokens introduced. (If one was unavoidable, raise to Axis with rationale before Leon signs off.)
- [ ] No `any` types. (Strict-TS gate, separate from Leon but a non-negotiable.)
- [ ] DOMPurify sanitization on any HTML rendered from `content_html` (Slack/Teams may include block-kit-formatted HTML in Wave 3 — defensive baseline now). Allowlist matches the email allowlist at [`email/+page.svelte:399`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L399).

> **Divergence from email spec.** Leah's [§15](./comms-email-spec.md#15--acceptance--what-leah-checks-in-wave-2-qa) acceptance list is structurally the same; the channels-specific items (reactions render branch, thread drawer geometry, optimistic recovery) are added.

---

## 16. Open questions for Axis / Roberto

Marked here so they don't block Fantem and Leon resolves before Wave 2 starts.

1. **Where Slack data lives in Wave 2.** Audit Section J calls this out: dual-write Slack into the unified `channels` / `channel_messages` tables (option A, Leon's recommendation), migrate (B), or query-time UNION (C). Frontend doesn't care which — but Ghost does, and it changes the API contract surface area. **Axis confirm.**
2. **Toast standard: `svelte-sonner` or `cm-toast`.** Email spec [§9.1](./comms-email-spec.md#91--toast-cm-toast) introduces `cm-toast`; channels Wave 1 already uses `svelte-sonner`. Both can't ship. Leon recommends `svelte-sonner` (already in use, no duplicate work). **Axis confirm; Leah's §9.1 may need a rewrite.**
3. **Reactions on the wire.** Backend currently doesn't expose `Message.reactions` ([`messages.go:18-33`](../../desktop/backend-go/internal/integrations/slack/messages.go#L18-L33)). The schema's `channel_messages.reactions` JSONB column ([`schema.sql:2387`](../../desktop/backend-go/internal/database/schema.sql#L2387)) is populated by the sync but not exposed. Spec assumes Ghost adds it in Wave 2. **Ghost confirm scope; if punted, the reactions branch deletes from the spec.**
4. **Sidebar width shared with email.** Leah uses 220px; Leon spec'd 240px. Define a single `--bos-sidebar-width` (or `--bos-comms-sidebar-width`) shared by both tabs. Recommended: 240px (channels' need is wider; email won't suffer at 240). **Axis arbitrate.**
5. **Workspace re-auth scoping.** Current Slack disconnect is global (`POST /integrations/slack/disconnect`). For multi-workspace, we need `POST /integrations/slack/disconnect?workspace_id=…`. **Ghost: feasible in Wave 2?** If not, single-workspace-Slack constraint stays — UI hides the "Add another workspace" affordance for Slack until Wave 3.
6. **Provider dot colors in the workspace switcher.** Spec uses `--bos-status-success` (healthy), `--bos-status-warning` (re-auth), `--bos-status-error` (failed). These tokens are repurposed for status communication, which is fine — no clash with their use in toasts since context differs. **Roberto OK.**
7. **`channel_messages.sender_avatar` backfill.** Schema column exists. Slack sync doesn't populate it today; users.info call needed. **Ghost wave 2 small scope add — or punt to Wave 3 with initial-only avatars in Wave 2.** Spec ships either way; backfilled URL is a strict UX upgrade.
8. **Activity & Threads endpoints.** Spec defines `GET /api/comms/activity` and the threads-section data source. Wave 2 minimum: stub returning empty arrays so the sections render correctly empty. **Ghost wave 2 effort?** If real shipping is feasible, both sections become live; otherwise Wave 3.

If any of these flip, the spec changes in three places at most: the relevant section + §15 acceptance + §14 out-of-scope. Easy to maintain.

---

## Icon inventory

`lucide-svelte` icons used by the channels redesign. **Bold** = added beyond the current page; *italic* = used by Leah's email spec too (consistency check).

- `Hash`, `Lock` — channel type indicators (existing)
- *`MessageSquare`* — empty-state icon (existing)
- *`Send`* — compose send button (existing)
- *`Loader2`* — loading spinner (existing)
- *`Users`* — member count icon (existing)
- *`RefreshCw`* — sync (existing)
- **`Smile`** — emoji picker / quick-react
- *`Paperclip`* — attach (existing in email)
- **`AtSign`** — mention / activity icon
- **`MoreHorizontal`** — kebab menus (also in email)
- *`X`* — close drawer / modal
- **`ChevronRight`** — reply summary chevron
- *`Plus`* — add workspace (also email's Compose)
- *`Search`* — sidebar filter (also email)
- **`Bell` / `BellOff`** — channel mute (Wave 3 wiring; Wave 2 ships dimmed)
- **`LogOut`** — disconnect workspace
- **`Reply`** — thread quick-reply hover action
- **`Bookmark`** — Saved messages (Wave 3 wiring; Wave 2 ships dimmed)
- **`AlertCircle`** — sync error / message error states

No icons outside `lucide-svelte`. No custom SVG.

---

## Appendix A — Divergences from the email spec

For Fantem's cross-reference. Fantem will be implementing both tabs and will inevitably cross-pollinate; this list calls out where the patterns intentionally part ways so cross-pollination doesn't introduce inconsistencies.

| § | Email (Leah) | Channels (Leon) | Why |
|---|---|---|---|
| §1.1 | 10 components, three-pane orchestration | 11 components, two-pane + drawer orchestration | Drawer adds one more component; sidebar adds workspace switcher |
| §1.2 | Reuses `DocumentEditor` for rich-text body | No `DocumentEditor`; plain text + emoji shortcodes only | Channels v2 has no rich text |
| **§1.3** | (no equivalent) | Unified frontend response shape because backend has divergent state | Email has `emails.provider`; channels has `slack_*` + (Teams TBD) |
| §2.1 | Static three-pane layout (sidebar + list + preview) | Two-pane primary, third drawer on demand | Different IA: channels has stream + episodic threads |
| §2.2 | Sidebar 220px | Sidebar 240px | More sections; settle in [§16-4](#16--open-questions-for-axis--roberto) |
| §3 | Provider switcher (Gmail/Outlook), Folders, Accounts | Workspace switcher, Channels, DMs, Activity, Threads | More axes |
| §3.4 | Compose CTA at sidebar footer (`btn-pill-primary`) | No global compose CTA; "Add workspace" link instead | Channels compose is in-stream |
| §4 | Toolbar centers a search input | Toolbar centers nothing; topic line below name | Channels' sidebar already has filter; per-channel search is Wave 3 |
| §5 | Email list (the middle pane) | (intentional pointer to §3.3 + §3.4 — channels has no middle pane) | Different layout |
| §6 | Thread view permanently in preview pane; collapsed-by-default cards | Inline reply summary; thread opens in drawer | Channels stream is the page primary; threads are episodic |
| §6.x | (no equivalent) | Reactions section, hover actions, date dividers, sender grouping, own/other distinction | Channels-specific UX |
| §7 | Full Compose modal: recipients, subject, rich text, attachments, send-later, drafts, validation | Inline compose bar: textarea, mention, emoji, attach, send | Channel sends are conversational, not document-shaped |
| §9.1 | Introduces `cm-toast` helper | Standardises on existing `svelte-sonner` | Wave 1 hygiene already used `svelte-sonner`; resolve in [§16-2](#16--open-questions-for-axis--roberto) |
| §10 | Read, Reply, Archive/Delete, Send, Drag attachment, Provider switch, Search | Open channel/DM/thread, Send, React, Workspace switch, Re-auth, Disconnect | Different surface |
| §11 | `ModuleEmail` engine sync; `OnEmailSaved` hook | `ModuleMessage` engine sync; `OnMessageSaved` hook | Same engine, different module |
| §14 | Out of scope: labels, signature, encryption | Out of scope: realtime, message edit/delete, rich blocks | Different domain |
| §15 | Acceptance: standard list | Acceptance: standard list + reactions/drawer/optimistic recovery additions | Channels-specific gates |
| §16 | 6 open questions | 8 open questions | One more for the data-table decision and one more for the workspace re-auth scoping |
| Appendix A | (no equivalent — Leah's spec doesn't cross-reference) | Present | Leon's audit + spec are second in the pair, so they bear the cross-ref weight |

---

**End of spec.** Fantem: this should be enough to start implementing the decomposition + sidebar + main stream. The drawer and compose can land in subsequent passes. Open questions in §16 should be resolved before Day 1 of Wave 2.
