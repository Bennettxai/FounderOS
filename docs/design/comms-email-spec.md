# Communications · Email — Redesign Spec

**Author:** Leah (UI/UX, Wave 1)
**Branch:** `feat/communications-module`
**Spec date:** 2026-05-02
**Audience:** Fantem (frontend implementation, Wave 2). Ghost for backend touch-points.
**Companion docs:** [`comms-email-audit.md`](./comms-email-audit.md) · [`COMMUNICATIONS_DESIGN_SYSTEM.md`](../COMMUNICATIONS_DESIGN_SYSTEM.md) · [`COMMUNICATIONS_AUDIT.md`](../COMMUNICATIONS_AUDIT.md) · [`COMMUNICATIONS_SPRINT_PLAN.md`](../COMMUNICATIONS_SPRINT_PLAN.md)

---

## 0. How to read this spec

Each section has:

- **Goal** — why this surface exists
- **Layout** — ASCII wireframe and structural anatomy
- **Tokens** — every color / space / radius / typography by name
- **Component** — which file, which props, which existing primitives it composes
- **Behavior** — interactions, states, edge cases
- **Acceptance** — what Leah checks against in Wave 2 visual QA

If a section is silent on a token or behaviour, default to the [design system doc](../COMMUNICATIONS_DESIGN_SYSTEM.md). No new tokens are introduced. If implementation needs one, raise to Axis before adding it.

**Naming convention:** all new components live under [`frontend/src/lib/components/comms/email/`](../../frontend/src/lib/components/comms/email/) (new directory). All new CSS classes are `cm-email-*` (sub-feature namespace).

---

## 1. Module architecture

### 1.1 — Component breakdown

The page decomposes into ten components, mirroring the [calendar template](../../frontend/src/routes/(app)/communication/calendar/+page.svelte) at `:420-509`.

```
routes/(app)/communication/email/+page.svelte    ← orchestrator only (~150 lines)
├─ EmailStatusBanner.svelte         (connection state, sync status, account switcher)
├─ EmailToolbar.svelte              (folder name, search, sync, compose CTA, refresh, view switch)
├─ EmailSidebar.svelte              (provider switcher, folder list, account list, "compose" button)
├─ EmailList.svelte                 (thread rows, list virtualization later, empty/loading states)
│   └─ EmailListRow.svelte          (single thread row, two-line)
├─ EmailThreadView.svelte           (preview pane: header + collapsed message stack + actions)
│   ├─ EmailMessageCard.svelte      (single message in a thread)
│   └─ EmailAttachmentList.svelte   (chip list under message body)
├─ EmailComposeModal.svelte         (.bos-modal wrapper for compose v2)
│   ├─ EmailRecipientField.svelte   (chip-style To/Cc/Bcc with autocomplete)
│   └─ EmailRichTextBody.svelte     (composes existing DocumentEditor, see §6)
├─ EmailEmptyState.svelte           (3 variants: not-connected, empty-folder, no-selection)
└─ commsEmailUtils.ts               (formatDate, groupByThread, foldersByProvider, providerLabel)
```

The page-level state (selected email, current folder, current provider, compose modal state, etc.) lives in `+page.svelte`. Children receive callbacks (`onSelectThread`, `onChangeFolder`, …) and props — no `createEventDispatcher`. This matches the calendar module's contract.

### 1.2 — Reuse map

Reused without modification:

| Existing | Used in | Why |
|---|---|---|
| [`PillButton.svelte`](../../frontend/src/lib/components/ui/PillButton.svelte) | every button surface | Standard. Variants: `primary` (Compose CTA), `ghost` (toolbar), `danger` (Delete), `soft` (Reply/Forward) |
| `.btn-cta` class (`app.css:1303`) | Compose button in sidebar | Signature blue-glow pulse — primary action of the tab |
| `.btn-compact-*` classes (`app.css:2212+`) | toolbar, message-card actions | Already used by current code; correct for dense rows |
| [`SkeletonLoader.svelte`](../../frontend/src/lib/components/ui/SkeletonLoader.svelte) | EmailList loading rows, EmailThreadView body skeleton | Existing skeleton primitive |
| [`ErrorBoundary.svelte`](../../frontend/src/lib/components/ui/ErrorBoundary.svelte) | wraps the three panes individually | One pane crash doesn't kill the others |
| [`Tooltip.svelte`](../../frontend/src/lib/components/ui/Tooltip.svelte) | icon-only buttons | Hover hint. Don't replace `aria-label` — both |
| `.bos-modal*` classes (`app.css:3055-3163`) | EmailComposeModal | Replaces the current bespoke `.ch-compose-overlay` |
| `.bos-input` / `.bos-textarea` / `.bos-label` (`app.css:3171-3248`) | recipient/subject inputs | Standard form chrome |
| [`DocumentEditor.svelte`](../../frontend/src/lib/components/editor/DocumentEditor.svelte) | EmailRichTextBody, in **read-only-formatting-toolbar** mode | Existing block editor. Wave 2 risk note: confirm it can be embedded without document persistence; if not, fall back to a contenteditable + minimal toolbar. Decision rests with Fantem after a spike — Leah signs off either path so long as the formatting toolbar uses the design tokens. |
| [`ChatAttachments.svelte`](../../frontend/src/lib/components/chat/input/ChatAttachments.svelte) pattern | EmailAttachmentList chip styling | Mirror the file-chip layout (don't import — different data shape, but same visual) |
| `Inbox`, `Send`, `FileEdit`, `Star`, `Archive`, `Trash2`, `Plus`, `Search`, `RefreshCw`, `Reply`, `Forward`, `Paperclip`, `Mail`, `X`, `Loader2`, `MoreHorizontal`, `Check`, `Clock` from `lucide-svelte` | everywhere | Existing icon vocabulary |

**Not reused (don't try):**

- `lib/components/chat/messages/UserMessage.svelte` / `AssistantMessage.svelte` — these are bubble-style, wrong shape for an email thread (which is card-stacked). Mimic the *layout pattern* for the message stack but build new components.
- `lib/components/chat/conversations/ConversationListItem.svelte` — wrong shape for an email row (different metadata).
- `lib/components/forms/FormField.svelte` — uses daisyUI classes (`.input input-bordered`), not the `.bos-input` system. The compose modal uses raw `.bos-input` directly.

---

## 2. Page layout

### 2.1 — Three-pane structure (desktop ≥ 1024px)

```
┌──────────────────────────────────────────────────────────────────────────┐
│  StatusBanner    [ All inboxes (3) · 247 unread · last sync 2 min ago ]  │  ← optional, only when notable
├──────────────────────────────────────────────────────────────────────────┤
│ Sidebar    │ Toolbar  Inbox · 247           [↻] [⚙]                      │
│ (220px)    ├───────────────────────────────────────────────────────────  │
│            │ Search   [ 🔍 Search this folder…              ]            │
│ Provider   ├───────────┬──────────────────────────────────────────────── │
│ ◉ All      │ List      │ Thread Preview                                  │
│ ○ Gmail    │ (340px)   │                                                 │
│ ○ Outlook  │           │ Subject (large)                                 │
│            │ ● Sender  │ ───────────────────────────────                 │
│ ── Folders │   Subject │ avatar  Sender Name   Wed 10:42                 │
│ ▼ Inbox 247│   snippet │         to me, alice                            │
│   Sent     │   Wed 10:42                                                 │
│   Drafts 3 │           │ ─── (3 earlier messages, click to expand) ───   │
│   Starred  │ ○ Sender  │                                                 │
│   Archive  │   Subject │ <body content>                                  │
│   Trash    │   ...     │                                                 │
│            │           │ <attachments chips>                             │
│ ── Accounts│           ├───────────────────────────────────────────────  │
│ • Gmail    │           │ [Reply] [Reply all] [Forward] [Archive] [⋯]    │
│   javaris@…│           │                                                 │
│ • Outlook  │           │                                                 │
│   me@miosa │           │                                                 │
│            │           │                                                 │
│ ─────────  │           │                                                 │
│ [+ Compose]│           │                                                 │
│            │           │                                                 │
└────────────┴───────────┴─────────────────────────────────────────────────┘
```

Top to bottom in the sidebar, the order is **provider → folders → accounts → compose CTA at footer**. Compose lives at the footer (sticky) so it's visible when the folder list is long; this is the same placement Gmail web uses and is established in the current code.

### 2.2 — Tokens

| Surface | Property | Token |
|---|---|---|
| Page background | bg | `var(--dbg)` |
| Sidebar | bg | `var(--dbg2)` |
| Sidebar | border-right | `1px solid var(--dbd)` |
| Sidebar | width | `220px` (was 185 — needs more room for unread counts on multiple folders) |
| Sidebar | padding | `var(--space-3) 0` |
| List pane | bg | `var(--dbg)` |
| List pane | border-right | `1px solid var(--dbd)` |
| List pane | width | `340px` (was 320 — two-line rows need more breathing room) |
| Preview pane | bg | `var(--dbg)` |
| Preview pane | flex | `1` |
| All panes | min-height | `0` (allow internal scroll) |

### 2.3 — Responsive

- **≥ 1024px:** three panes as above.
- **640–1023px:** two panes — sidebar collapses to icon rail (folder icons only, no labels); list + preview share the rest. Compose CTA collapses to icon-only `<PillButton variant="primary" size="icon"><Plus /></PillButton>` at footer of icon rail.
- **< 640px:** single pane — list takes the screen; selecting a thread navigates to a stacked preview (back chevron in toolbar). Compose becomes a floating `.btn-cta` at the bottom-right (44×44).

Wave 2 ships desktop only. Mobile is Wave 3, but the components are built mobile-aware (avoid hard widths in the inner content).

---

## 3. Sidebar

### 3.1 — Anatomy

```
┌─────────────────────┐
│ ▼ View              │  ← collapsed by default if only 1 provider
│   ◉ All inboxes 247 │  active state: bg = --bos-nav-active-bg, fg = --bos-nav-active
│   ○ Gmail        12 │
│   ○ Outlook       3 │
│                     │
│ ▼ Folders           │
│   📥 Inbox      247 │
│   ➤ Sent            │
│   📝 Drafts       3 │
│   ⭐ Starred         │
│   📦 Archive         │
│   🗑️ Trash           │
│                     │
│ ▼ Accounts          │
│   • javaris@miosa.ai│  Gmail dot color = --bos-status-success when synced healthy
│   • me@miosa.ai     │  Outlook dot color = --bos-status-warning if last sync >24h
│   [+ Add account]   │  small text-xs link
│                     │
├─────────────────────┤  ← sticky footer
│ [+ Compose] btn-cta │
└─────────────────────┘
```

**Section headers** ("View", "Folders", "Accounts") render as collapsed-state click targets that hide/show the section. State persists in `localStorage` with key `comms.email.sidebarSections`. Default: all expanded.

### 3.2 — Provider switcher (View section)

`SlateChoice` rendering — three radio rows. Active = filled blue dot + bg tint.

| Option | When shown | Filter applied |
|---|---|---|
| All inboxes | Always when ≥ 2 providers connected | No filter |
| Gmail | Gmail account exists | `provider=gmail` |
| Outlook | Outlook account exists | `provider=outlook` |

If only one provider is connected, the View section auto-collapses and is replaced by an "Add Outlook" / "Add Gmail" affordance at the bottom of Accounts.

### 3.3 — Folder list

| Folder | Icon | Provider mapping |
|---|---|---|
| Inbox | `Inbox` | Gmail `INBOX` / Outlook `Inbox` |
| Sent | `Send` | Gmail `SENT` / Outlook `Sent Items` |
| Drafts | `FileEdit` | Gmail `DRAFT` + local `is_draft=true` rows / Outlook `Drafts` |
| Starred | `Star` | Gmail `STARRED` / Outlook flagged |
| Archive | `Archive` | Gmail (no `INBOX` label, not Trash) / Outlook `Archive` |
| Trash | `Trash2` | Gmail `TRASH` / Outlook `Deleted Items` |

Unread count badge appears only on folders where `unread > 0`. Inbox shows total unread for the active provider scope; Drafts shows total drafts (not "unread"); others stay countless.

**Tokens:**

| Surface | Property | Token |
|---|---|---|
| Folder row | font | `var(--text-sm)` / `var(--font-medium)` |
| Folder row | padding | `var(--space-2) var(--space-4)` |
| Folder row | text color | `var(--dt2)` |
| Folder row hover | bg | `var(--dbg3)` |
| Folder row active | bg | `var(--bos-nav-active-bg)` |
| Folder row active | text color | `var(--bos-nav-active)` |
| Folder row active | font-weight | `var(--font-semibold)` |
| Folder row icon | size | `16px` (Lucide) |
| Unread badge | bg | `var(--bos-accent-blue)` |
| Unread badge | text | `var(--bos-surface-on-color)` |
| Unread badge | font | `var(--text-xs)` / `var(--font-bold)` |
| Unread badge | radius | `var(--radius-full)` |
| Unread badge | padding | `1px var(--space-2)` |
| Section divider | border-top | `1px solid var(--dbd)` |
| Section divider | margin | `var(--space-3) 0` |
| Section header | font | `var(--text-xs)` / `var(--font-semibold)` / uppercase / `letter-spacing: 0.04em` |
| Section header | color | `var(--dt3)` |

### 3.4 — Compose CTA (sidebar footer)

```svelte
<PillButton variant="primary" size="md" block onclick={openCompose}>
  <Plus size={16} /> Compose
</PillButton>
```

Footer wrapper has `padding: var(--space-3)`, `border-top: 1px solid var(--dbd)`, `background: var(--dbg2)`. **Don't** introduce a custom `.ch-inbox-compose-btn` rule — `block` prop on `<PillButton>` makes it stretch.

For the *signature blue-glow CTA* described in [the design system](../COMMUNICATIONS_DESIGN_SYSTEM.md#buttons), use `.btn-cta` directly inside the wrapper instead — Axis to confirm whether sidebar Compose ranks high enough for the CTA glow or stays as a regular pill primary. Default in this spec: **regular `btn-pill-primary`**. Reason: the calendar module reserves `.btn-cta` for the toolbar create-event button, and `.btn-cta` glow is visually loud at sidebar scale.

---

## 4. Toolbar

### 4.1 — Anatomy

```
┌────────────────────────────────────────────────────────────────────┐
│ Inbox · 247 unread          [ 🔍 Search this folder… ]  [↻] [⋯]   │
└────────────────────────────────────────────────────────────────────┘
```

- Left: folder label (capitalised, `--text-base` / `--font-semibold`) + dot separator + unread count in muted text.
- Center: search input (collapses on hover-out if empty; expands on focus). Uses `.bos-input` with prepended `<Search size={14} />`.
- Right: refresh (`RefreshCw`, animates on sync), overflow menu (`MoreHorizontal`) for "Mark all as read", "Empty trash" (only on Trash folder), "Settings".

### 4.2 — Tokens

| Surface | Property | Token |
|---|---|---|
| Toolbar | bg | `var(--dbg)` |
| Toolbar | border-bottom | `1px solid var(--dbd)` |
| Toolbar | padding | `var(--space-3) var(--space-4)` |
| Toolbar | gap | `var(--space-3)` |
| Folder label | font | `var(--text-base)` / `var(--font-semibold)` |
| Folder label | color | `var(--dt)` |
| Unread count | font | `var(--text-sm)` |
| Unread count | color | `var(--dt3)` |
| Search input | base | `.bos-input` (existing class, do not re-style) |
| Search input | width | `flex: 1`, `max-width: 280px` |
| Refresh button | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |
| Refresh icon (syncing) | animation | existing `ch-spin` keyframes (rename to `cm-spin` in this module) |

### 4.3 — Sync state

Three states for the refresh icon:

| State | Visual | Trigger |
|---|---|---|
| Idle | static `RefreshCw` | default |
| Syncing | spinning + `--dt3` color | sync in progress |
| Error | static `AlertCircle` + `--bos-status-error` color, tooltip with message | sync just failed |

Error state auto-clears on next successful sync, or after 10s into idle. Success doesn't show a checkmark — silent success is the rule.

---

## 5. Email list

### 5.1 — Row anatomy

Two-line row. Replaces the current single-line subject+snippet concatenation.

```
┌──────────────────────────────────────────────────────────────────┐
│ ●  ┌──┐  Sender Name · Reply count                     Wed 10:42 │  line 1
│    │SN│  Subject of the thread                           [📎]    │
│    └──┘  Snippet preview text…                             ⭐    │  line 2
└──────────────────────────────────────────────────────────────────┘
   │  │   │                                                │
   │  │   │                                                └─ time, attachment chip, star
   │  │   └─ sender + (reply count if thread has > 1 msg)
   │  └─ avatar (32×32, initial)
   └─ unread dot (7×7) or empty circle if read
```

- Hover: `bg: var(--dbg2)`.
- Selected: `bg: var(--dbg3)`.
- Unread: sender + subject in `--font-semibold` and `--dt`. Read: `--font-normal` and `--dt2`.
- Provider badge (only when "All inboxes" is active): a 12×12 colored dot to the left of the unread dot — red dot for Gmail, blue dot for Outlook. Use `--bos-status-error` for Gmail (matches Google's brand red) and `--bos-status-info` for Outlook (matches Microsoft blue). Spec note: these are status tokens used for visual identity here, which is acceptable — agreed in advance with Axis.

### 5.2 — Tokens

| Surface | Property | Token |
|---|---|---|
| Row | padding | `var(--space-3) var(--space-4)` |
| Row | gap | `var(--space-3)` |
| Row | border-bottom | `1px solid var(--dbd)` |
| Row hover | bg | `var(--dbg2)` |
| Row selected | bg | `var(--dbg3)` |
| Row transition | bg | `var(--bos-transition-fast)` |
| Unread dot | size | `7px` |
| Unread dot | bg | `var(--bos-accent-blue)` |
| Unread dot | radius | `var(--radius-full)` |
| Avatar | size | `32px` |
| Avatar | bg | `var(--dt3)` |
| Avatar | text color | `var(--dbg)` |
| Avatar | font | `var(--text-xs)` / `var(--font-bold)` |
| Avatar | radius | `var(--radius-full)` |
| Sender name | font | `var(--text-sm)` |
| Sender name (unread) | color, weight | `var(--dt)` / `var(--font-semibold)` |
| Sender name (read) | color, weight | `var(--dt2)` / `var(--font-normal)` |
| Subject | font | `var(--text-sm)` |
| Snippet | font | `var(--text-xs)` |
| Snippet | color | `var(--dt3)` |
| Time | font | `var(--text-xs)` |
| Time | color | `var(--dt3)` |
| Attachment chip | icon | `Paperclip size={12}` |
| Attachment chip | color | `var(--dt3)` |
| Star button (idle) | color | `var(--dt4)` |
| Star button (active) | color | `var(--bos-status-warning)` (gold-ish amber) |
| Provider dot | size | `8px` |
| Provider dot (Gmail) | bg | `var(--bos-status-error)` |
| Provider dot (Outlook) | bg | `var(--bos-status-info)` |

### 5.3 — Threading

Backend (Ghost wave 1+2) returns `Email` rows. The frontend groups by `thread_id` client-side via `commsEmailUtils.groupByThread(emails)`. Each list row represents the latest message in a thread; row metadata is:

- **Sender:** the most recent participant.
- **Subject:** the most recent subject (strip leading `Re:` / `Fwd:` runs to one prefix max).
- **Snippet:** the most recent message's snippet.
- **Time:** the most recent message's `date`.
- **Reply count:** thread length, displayed as `· 4` after sender name when > 1.
- **Unread:** thread is unread if *any* message is unread.
- **Star:** thread is starred if *any* message is starred.

When the user opens a thread, mark all its messages read.

### 5.4 — Empty state

Renders inside the list pane (not full-page).

| Subscope | Title | Body | Action |
|---|---|---|---|
| Empty folder | "No emails in {Folder}" | "When you receive emails here, they'll show up." | none |
| Search no-match | "No matching emails" | "Try different keywords or clear search." | "Clear search" `<PillButton variant="soft" size="sm">` |
| Folder unsupported | "Drafts coming with provider sync" | "Outlook drafts will appear here once sync completes." | none (only when filtering to a provider that doesn't yet support that folder) |

Tokens:

| Surface | Property | Token |
|---|---|---|
| Empty wrapper | padding | `var(--space-12) var(--space-4)` |
| Empty wrapper | gap | `var(--space-3)` |
| Empty wrapper | text-align | `center` |
| Icon | `Mail`, stroke 1.5 | size `28px`, color `var(--dt3)` |
| Title | font | `var(--text-sm)` / `var(--font-semibold)` |
| Title | color | `var(--dt2)` |
| Body | font | `var(--text-xs)` |
| Body | color | `var(--dt3)` |
| Body | max-width | `280px` |

### 5.5 — Loading skeleton

While the first page of emails is loading (or after a folder change), render five skeleton rows. Each skeleton row mirrors the real row geometry: 7px circle, 32px avatar block, two text lines with widths `60%`/`90%`, plus a small time block on the right. Use [`SkeletonLoader.svelte`](../../frontend/src/lib/components/ui/SkeletonLoader.svelte) for each shape.

Subsequent loads (folder switches that re-render the same content shape) keep the previous list in place with a top progress bar instead of a full skeleton — avoids the jarring "wipe and re-paint" of the current implementation.

### 5.6 — Error state

```
┌──────────────────────────────────────────────┐
│  ⚠  Couldn't load this folder                 │
│     Network problem. Check your connection.  │
│     [Try again]                              │
└──────────────────────────────────────────────┘
```

Tokens: same as empty state, but icon is `AlertCircle` color `var(--bos-status-error-text)`, title color `var(--dt)`. Action button is `<PillButton variant="soft" size="sm">Try again</PillButton>`.

---

## 6. Thread view (preview pane)

### 6.1 — Header

```
┌──────────────────────────────────────────────────────────────────┐
│ Subject of the thread                                            │
│ Inbox · gmail · 5 messages                                       │  meta line, --dt3
│                                                                  │
│ ┌──┐ Sender Name (sender@…)                       Wed 10:42      │
│ │SN│ to me, alice@…, bob@… · cc carol@… (+ 2 more)               │
│ └──┘                                                             │
└──────────────────────────────────────────────────────────────────┘
```

Recipient line truncates with "(+ N more)" when total recipients > 4. Click expands inline.

### 6.2 — Message stack

Default state: latest message expanded, others collapsed to one-line cards.

```
┌──────────────────────────────────────────────────────────────────┐
│ ▾  Latest sender   Wed 10:42                                     │  expanded
│    <body content rendered here>                                   │
│    [📎 attachment.pdf · 1.2 MB] [📎 image.png · 320 KB]          │
│ ─────────────────────────────────────────────────────────────────│
│ ▸ Earlier sender · "earlier snippet..."  Mon 14:01               │  collapsed
│ ▸ First sender   · "first snippet..."   Sun 09:00                │  collapsed
└──────────────────────────────────────────────────────────────────┘
```

Click a collapsed row → expand it (others stay open; multi-expand). Each card stores `expanded` in local component state.

### 6.3 — Message card (expanded) tokens

| Surface | Property | Token |
|---|---|---|
| Card | padding | `var(--space-5) var(--space-6)` |
| Card | border-bottom | `1px solid var(--dbd)` |
| Card sender row | gap | `var(--space-3)` |
| Sender name | font | `var(--text-sm)` / `var(--font-semibold)` |
| Sender name | color | `var(--dt)` |
| Sender email | font | `var(--text-xs)` |
| Sender email | color | `var(--dt3)` |
| Date | font | `var(--text-xs)` |
| Date | color | `var(--dt3)` |
| Body wrapper | font | `var(--text-base)` |
| Body wrapper | color | `var(--dt2)` |
| Body wrapper | line-height | `1.65` |
| Body wrapper | max-width | `680px` (readable measure) |
| Body links | color | `var(--bos-accent-blue)` (`a` selector inside body) |
| Body images | max-width | `100%`, height `auto` |
| Body quote | bg | `var(--dbg2)` |
| Body quote | border-left | `2px solid var(--dbd)` |
| Body quote | padding | `var(--space-2) var(--space-4)` |
| Body quote | color | `var(--dt3)` |

**Body content rules:**

- Render `body_html` via `DOMPurify.sanitize` with the existing allowlist at [`email/+page.svelte:399`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L399). Don't widen.
- If `body_html` missing, render `body_text` in `<pre>` with `white-space: pre-wrap` and the same body color/measure tokens.
- Quoted text (the `> ` prefix runs in plain text, the `<blockquote>` runs in HTML) collapses to a "Show quoted text" affordance after the first non-quoted block. Click expands.

### 6.4 — Message card (collapsed) tokens

| Surface | Property | Token |
|---|---|---|
| Collapsed row | padding | `var(--space-3) var(--space-6)` |
| Collapsed row | bg | `var(--dbg)` (matches expanded — collapsed is just shorter, not visually different) |
| Collapsed row hover | bg | `var(--dbg2)` |
| Collapsed row | font | `var(--text-sm)` |
| Sender name | color | `var(--dt2)` |
| Snippet | color | `var(--dt3)` |
| Date | font | `var(--text-xs)` |
| Date | color | `var(--dt3)` |
| Chevron icon | size | `14px` |

### 6.5 — Attachments

`EmailAttachmentList.svelte` renders a horizontal flex row of chips below the body, before the next card border.

```
[📄 attachment.pdf · 1.2 MB ⬇] [🖼 image.png · 320 KB ⬇]
```

Chip:

| Surface | Property | Token |
|---|---|---|
| Chip wrapper | flex | `gap: var(--space-2)`, `wrap` |
| Chip | bg | `var(--dbg2)` |
| Chip | border | `1px solid var(--dbd)` |
| Chip | radius | `var(--radius-md)` |
| Chip | padding | `var(--space-2) var(--space-3)` |
| Chip | font | `var(--text-xs)` |
| Chip | color | `var(--dt2)` |
| Chip hover | bg | `var(--dbg3)` |
| Chip transition | bg | `var(--bos-transition-fast)` |
| Chip icon | size | `14px` |
| Filename | max-width | `180px`, ellipsis |
| Size | color | `var(--dt3)` |
| Download icon | hover-only, color `var(--dt2)` |

Click chip → call backend download endpoint (Ghost owns; spec calls out the affordance — Fantem expects `GET /api/integrations/.../emails/:emailId/attachments/:attachmentId`).

### 6.6 — Action bar

Sticky footer of the preview pane.

```
[Reply] [Reply all] [Forward]    [Star] [Archive] [Trash] [⋯]
```

Left group is primary action set (uses `<PillButton variant="soft" size="sm">` — soft fill, distinct from list-row icons). Right group is icon-only `.btn-compact-ghost .btn-compact-icon`. Overflow `⋯` opens a menu with: "Mark as unread", "Move to…", "Add label" (Wave 3), "Mute thread" (Wave 3), "Print", "Show original".

| Surface | Property | Token |
|---|---|---|
| Action bar | padding | `var(--space-3) var(--space-6)` |
| Action bar | border-top | `1px solid var(--dbd)` |
| Action bar | bg | `var(--dbg2)` |
| Action bar | gap | `var(--space-2)` |
| Right group | margin-left | `auto` |

### 6.7 — Thread view empty state

When no thread is selected:

```
┌──────────────────────────────────────────────┐
│              [Mail icon, 40px]                │
│            Select an email to read            │
│         or [Compose] a new message            │
└──────────────────────────────────────────────┘
```

The "Compose" link triggers the modal (same as sidebar CTA).

Tokens: same shape as List empty state at §5.4. Icon size `40px` (was `28px` for list — preview is bigger).

---

## 7. Compose v2

### 7.1 — Anatomy

`.bos-modal` (size `lg`) — replaces the bespoke `.ch-compose-overlay`. Header / Body / Footer slots use the standard `.bos-modal-*` classes.

```
┌─────────────────────────────────────────────────────────────┐
│ New Message                            [_]  [⛶]  [✕]        │  ← header (standard)
├─────────────────────────────────────────────────────────────┤
│ From   ▼ Gmail · javaris@miosa.ai                            │  account picker
│ To     [chip] [chip] [type to add…]                          │  recipient field
│ Cc     [type to add…]                            [Bcc ▾]    │  Cc, Bcc toggle on right
│ Subject  Email subject                                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   <rich text body editor — DocumentEditor or fallback>      │  body
│                                                             │
│   [📎 file.pdf · 1.2 MB ✕]  [📎 image.png · 320 KB ✕]      │  attachment chips above body? See §7.6
│                                                             │
├─────────────────────────────────────────────────────────────┤
│ [Send]  [⏰ Send later ▾]   [📎] [✏ Format]    [Discard]   │  ← footer
└─────────────────────────────────────────────────────────────┘
   [⌘+Enter to send · auto-saved 2s ago]                       (status line, --dt3)
```

### 7.2 — Header buttons

- `[_]` minimize → docks the modal to the bottom-right as a 240×40 chip showing the subject. Click to restore. Out of scope for Wave 2 if it adds risk; mark as "if cheap" — Fantem's call.
- `[⛶]` toggle full-screen (max-width: 100% + max-height: 100%).
- `[✕]` close → if there are unsaved changes, prompt "Save as draft?" with [Save draft] [Discard] [Cancel].

### 7.3 — From / account picker

A `<select>` rendered as a dropdown trigger. Lists every connected account with its provider tag. Default: most recently used. If only one account, render as static text (no dropdown).

| Surface | Property | Token |
|---|---|---|
| Field row | padding | `var(--space-2) var(--space-4)` |
| Field row | border-bottom | `1px solid var(--dbd)` |
| Label | width | `64px` (was 46 — needs room for "Subject") |
| Label | font | `var(--text-xs)` / `var(--font-medium)` |
| Label | color | `var(--dt3)` |
| Value | font | `var(--text-sm)` |
| Value | color | `var(--dt)` |

### 7.4 — Recipient field (`EmailRecipientField`)

Chip-style input. Each typed-or-pasted email becomes a chip; backspace deletes the last; click chip to remove.

Autocomplete sources (in priority order):
1. `crm_contacts` table (existing schema).
2. `microsoft_contacts` (existing schema).
3. `hubspot_contacts` (existing schema).
4. Local frequency cache: distinct `from_email` / `to_emails` from the `emails` table, weighted by send count.

Backend endpoint Fantem expects: `GET /api/comms/contacts/search?q=…&limit=8`. Ghost adds in Wave 2 (already noted in audit). Spec includes this so Fantem can stub it locally first.

Chip tokens:

| Surface | Property | Token |
|---|---|---|
| Chip | bg | `var(--dbg2)` |
| Chip | border | `1px solid var(--dbd)` |
| Chip | radius | `var(--radius-full)` |
| Chip | padding | `2px var(--space-2)` |
| Chip | font | `var(--text-xs)` |
| Chip valid | color | `var(--dt2)` |
| Chip invalid | color | `var(--bos-status-error-text)`, border `var(--bos-status-error)` |
| Chip remove icon | size | `12px`, color `var(--dt3)` on hover `var(--dt)` |
| Suggestion list | bg | `var(--dbg)` |
| Suggestion list | border | `1px solid var(--dbd)` |
| Suggestion list | radius | `var(--radius-md)` |
| Suggestion list | shadow | `var(--bos-popover-shadow)` |
| Suggestion list | z-index | `var(--bos-z-popover)` |
| Suggestion row | padding | `var(--space-2) var(--space-3)` |
| Suggestion row hover | bg | `var(--dbg2)` |

`Cc` / `Bcc` are hidden by default — single "Add Cc · Bcc" toggle on the right of the To row reveals them.

### 7.5 — Subject

Standard `.bos-input` styling but borderless inside the field row. Font `var(--text-sm)` / `var(--font-semibold)` (same as the original `.ch-compose-subject` which was correct in spirit). Placeholder color `var(--dt4)`.

### 7.6 — Body (`EmailRichTextBody`)

Two-mode component:

**Mode A: rich text (default).** Embeds `DocumentEditor` with: paragraph, bold, italic, underline, link, list (bullet/numbered), quote, code-inline, undo/redo. **No** code blocks, headings, callouts, tables, or block-bookmarks — those features in DocumentEditor target documents, not email. Hide them.

**Mode B: plain text fallback.** A `<textarea>` styled as `.bos-textarea`. User toggles via the "✏ Format" button in the footer.

If embedding `DocumentEditor` carries risk (it might require a document persistence model), Fantem spikes for one half-day; if blocked, fallback to a thin contenteditable + minimal toolbar (rendered via `lib/components/ai-elements/` shape if any matches; otherwise hand-built). Either path uses the design tokens. **Mode B remains regardless** as an explicit option for power users who hate rich text.

Body area tokens:

| Surface | Property | Token |
|---|---|---|
| Body wrapper | padding | `var(--space-4)` |
| Body wrapper | min-height | `240px` |
| Body wrapper | max-height | `60vh` (then internal scroll) |
| Body wrapper | bg | `var(--dbg)` |
| Body text | font | `var(--text-sm)` |
| Body text | color | `var(--dt)` |
| Body text | line-height | `1.65` |
| Toolbar (rich text) | bg | `var(--dbg2)` |
| Toolbar | border-bottom | `1px solid var(--dbd)` |
| Toolbar | padding | `var(--space-2) var(--space-3)` |
| Toolbar | gap | `var(--space-1)` |
| Toolbar button | base | `.btn-compact .btn-compact-ghost .btn-compact-icon` |
| Toolbar button active | bg | `var(--dbg3)`, color `var(--dt)` |

Attachment chips render *inside* the modal body, between the body editor and the footer (not above the body, despite the wireframe — the wireframe is illustrative). Same chip styling as §6.5.

### 7.7 — Footer actions

| Button | Variant | Behaviour |
|---|---|---|
| Send | `<PillButton variant="primary" size="sm">` with `<Send size={15} />` | sends; ⌘/Ctrl+Enter shortcut |
| Send later ▾ | `<PillButton variant="ghost" size="sm">` next to Send, opens schedule popover | options: "Tomorrow morning", "Tomorrow afternoon", "Monday morning", "Pick date & time…" |
| Attach | `.btn-compact .btn-compact-ghost .btn-compact-icon` `<Paperclip />` | opens file picker, supports drag-drop onto body |
| Format | toggle | switches body Mode A ↔ Mode B |
| Discard | `.btn-compact .btn-compact-ghost` | "Discard draft? This can't be undone." prompt; on confirm clears state and closes |

Right group = Send + Send later. Center group = Attach + Format. Right edge = Discard. Status line below ("⌘+Enter to send · auto-saved 2s ago") in `var(--text-xs)` `var(--dt3)`.

### 7.8 — Draft auto-save

- Debounced 2s after last keystroke or attachment change.
- Persists to `emails` row with `is_draft=true`. Backend endpoint Ghost exposes: `POST /api/integrations/.../drafts` (create), `PUT /api/integrations/.../drafts/:id` (update).
- Status line text: "Saving…" → "Auto-saved {relative time}" → "Couldn't save — Retry".
- Re-open from Drafts folder: opens the compose modal pre-populated with the draft, including attachments.

### 7.9 — Send-later

Popover above the "Send later ▾" button.

```
┌────────────────────────────────────┐
│ Tomorrow morning      Tue 8:00 AM   │
│ Tomorrow afternoon    Tue 1:00 PM   │
│ Monday morning        Mon 8:00 AM   │
│ ─────────────                      │
│ Pick date & time…                  │
└────────────────────────────────────┘
```

| Surface | Property | Token |
|---|---|---|
| Popover | bg | `var(--dbg)` |
| Popover | border | `1px solid var(--dbd)` |
| Popover | radius | `var(--radius-md)` |
| Popover | shadow | `var(--bos-popover-shadow)` |
| Popover | z-index | `var(--bos-z-popover)` |
| Row | padding | `var(--space-2) var(--space-3)` |
| Row hover | bg | `var(--dbg2)` |
| Row label | font | `var(--text-sm)` / color `var(--dt2)` |
| Row time | font | `var(--text-xs)` / color `var(--dt3)` / margin-left auto |

"Pick date & time…" opens a native `<input type="datetime-local">` rendered inside the popover. Wave 2 acceptable; Wave 3 can replace with a styled date picker.

Backend: schedules persist as a draft with `scheduled_send_at` column (Ghost — note for Wave 2 schema add).

### 7.10 — Validation

- "To" must have at least one valid email. Otherwise Send is disabled and a small inline error shows under the field: "Add at least one recipient" in `var(--bos-status-error-text)`.
- Subject empty: prompt "Send without a subject?" on Send click.
- Body empty: same prompt.
- Attachment over provider's size limit: chip turns to error styling and tooltip explains.

---

## 8. Status banner

### 8.1 — Anatomy & visibility

Renders at the top of the page, above the toolbar. Only shows when there's something *notable*. Three trigger conditions, in priority order:

1. **No accounts connected:** banner says "Connect Gmail or Outlook to start." with two `<PillButton size="sm">` buttons. Banner takes the *whole page* (not just the strip) — uses the existing not-connected pattern at [`email/+page.svelte:246-263`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L246-L263). The other panes don't render.
2. **Reauth needed for an account:** strip with warning icon: "Outlook authorization expired · [Reconnect]". One per affected account.
3. **Last sync > 1 hour ago and folder is Inbox:** info strip "Last synced {relative} · [Sync now]".

Otherwise: not rendered.

### 8.2 — Strip tokens

| Surface | Property | Token |
|---|---|---|
| Strip | padding | `var(--space-2) var(--space-4)` |
| Strip | border-bottom | `1px solid var(--dbd)` |
| Strip | gap | `var(--space-3)` |
| Strip (warning) | bg | `var(--bos-status-warning-bg)` |
| Strip (warning) text | color | `var(--bos-status-warning-text)` |
| Strip (info) bg | `var(--bos-status-info-bg)` |
| Strip (info) text | color | `var(--bos-status-info-text)` |
| Icon | size | `16px` |
| Action button | base | `.btn-compact .btn-compact-ghost .btn-compact-sm` |

---

## 9. Empty / loading / error — the matrix

Single-source-of-truth list so Fantem doesn't miss any state.

| Surface | Empty | Loading | Error |
|---|---|---|---|
| Page (no accounts) | "Connect Gmail / Outlook" full-page (§8.1) | "Connecting…" full-page spinner during OAuth callback | "Couldn't connect. {message} · [Try again]" full-page |
| Sidebar | n/a (folders are static) | (no skeleton — sidebar ships immediately) | n/a |
| List | "No emails in {Folder}" / "No matching emails" / "Drafts coming with provider sync" (§5.4) | 5 skeleton rows on first load (§5.5); top progress bar on subsequent loads | "Couldn't load this folder · [Try again]" (§5.6) |
| Thread view | "Select an email to read" / "Compose a new message" (§6.7) | Header skeleton + 3 message-card skeletons | "Couldn't load this email · [Back to list] [Try again]" |
| Compose | (closed by default) | "Sending…" overlay + spinner inside modal | inline error banner above body, dismissible |

### 9.1 — Toast (`cm-toast`)

A new comms-namespaced toast helper. Lives at [`frontend/src/lib/components/comms/CmToast.svelte`](../../frontend/src/lib/components/comms/CmToast.svelte) + a tiny `commsToast.ts` store. Rendered in the comms hub layout (`+layout.svelte`) so it floats above all three tabs.

| Variant | Color tokens | Trigger examples |
|---|---|---|
| success | bg `var(--bos-status-success-bg)`, text `var(--bos-status-success-text)`, border `var(--bos-status-success)` | "Email sent", "Archived", "Marked as read", "Draft saved" |
| error | bg `var(--bos-status-error-bg)`, text `var(--bos-status-error-text)`, border `var(--bos-status-error)` | "Couldn't send: {message}", "Couldn't sync: {message}" |
| info | bg `var(--bos-status-info-bg)`, text `var(--bos-status-info-text)`, border `var(--bos-status-info)` | "Reconnecting…" |

Tokens:

| Surface | Property | Token |
|---|---|---|
| Toast container | position | `fixed; bottom: var(--space-6); right: var(--space-6)` |
| Toast container | z-index | `var(--bos-z-toast)` |
| Toast container | gap | `var(--space-2)` (vertical stack) |
| Toast | padding | `var(--space-3) var(--space-4)` |
| Toast | border-radius | `var(--radius-md)` |
| Toast | shadow | `var(--bos-shadow-2)` |
| Toast | font | `var(--text-sm)` |
| Toast | min-width | `280px`, max-width `400px` |
| Toast icon | size | `16px` |
| Toast close | size | `14px`, color `--dt3` on hover `--dt` |
| Toast transition | enter | slide-up + fade, `var(--bos-transition-normal)` |
| Auto-dismiss | success: 3s · error: never · info: 5s |

Action: optional `[Undo]` link (`.btn-compact-link` style, color `--bos-accent-blue`) — used by Archive, Delete, Mark-as-read for a 5s grace window.

---

## 10. Interaction flows

### 10.1 — Read an email (the happy path)

1. User clicks a row in `EmailList`.
2. Row gets `selected` class; `selectedThreadId` set in page state.
3. `EmailThreadView` mounts with `threadId` prop.
4. If thread had unread messages: optimistic update — toggle `is_read=true` locally on all messages, mark thread row read. Backend call `markAsRead(threadId)` runs in background.
5. On backend failure: toast "Couldn't mark as read · [Retry]"; revert optimistic update if user taps Retry.

### 10.2 — Reply

1. User clicks `Reply` (or `Reply all`, `Forward`).
2. Modal opens pre-populated:
    - To: original sender (Reply) / sender + all recipients (Reply all) / empty (Forward)
    - Subject: `Re: …` (deduplicated) / `Fwd: …`
    - Body: original body in a quote block at the bottom; cursor positioned above the quote.
    - Account picker: original recipient account (sticky).
3. Compose flow continues as §7.

### 10.3 — Archive / Delete / Mark unread

Optimistic; toast with Undo. On Undo within 5s, revert. On Undo after 5s, the action is final and Undo doesn't show.

### 10.4 — Send

1. Validation per §7.10. If invalid, focus first invalid field.
2. Click Send → button enters loading state (disabled, label "Sending…", spinner).
3. Modal stays open while sending.
4. On success: modal closes, toast "Email sent · [View in Sent]". `View in Sent` switches folder to Sent and selects the new message.
5. On failure: modal stays open, inline error above body with retry CTA. Draft is already saved (auto-save), so closing the modal doesn't lose work.

### 10.5 — Drag attachment onto body

1. User drags file from OS over the modal.
2. Body wrapper highlights (border-color `var(--bos-accent-blue)`, bg `var(--bos-nav-active-bg)`) and shows centered overlay text "Drop to attach".
3. Drop → file added to attachment list, upload begins, chip shows progress.

### 10.6 — Provider switch

1. User clicks "Outlook" in sidebar View section.
2. List clears, skeleton renders, fetch starts with `provider=outlook` filter.
3. Folder counts in sidebar refresh for Outlook scope.
4. Active folder stays the same (Inbox stays Inbox); selection clears.

### 10.7 — Server search (Wave 3 hook)

Search input copy is "Search this folder…" in Wave 2. In Wave 3, copy changes to "Search all email…" and triggers a server-side search across all providers and folders. Spec marks the input component as ready to accept a `searchScope` prop now ("folder" | "all") so Wave 3 swap is one prop change.

---

## 11. Engine-sync surface

The page itself doesn't render engine signals, but the components must avoid blocking Axis's `OnEmailSaved → ModuleEmail` hook (Axis Wave 1 task in [the sprint plan](../COMMUNICATIONS_SPRINT_PLAN.md#-axis--wave-1-task-engine-sync-hooks-spec--module-level-wiring)).

Implication for design: when a thread is opened and its messages are marked as read, the optimistic update path must include the same backend mutation that fires the `OnEmailSaved` hook — otherwise read-state never reaches the engine. Fantem confirms with Ghost that the `markAsRead` route fires the hook (it should — Ghost's plumbing handles this).

A future Wave 3 surface (out of scope here, mentioned for completeness): an "AI insights" strip in the thread view that surfaces Signal Theory triage hints ("Low-priority newsletter", "Customer waiting on reply since {time}"). The component slot exists in `EmailThreadView` between header and message stack — pass through a `<slot name="insights" />` so Wave 3 fills it.

---

## 12. Accessibility commitments

- Every icon-only button keeps `aria-label`. (Continuing the existing standard.)
- Compose modal traps focus. First focused element on open: To field if empty, Body if pre-populated.
- Live region (polite) at `role="status"` in `+page.svelte` announces: "Synced N new messages", "Email sent", "Couldn't send email".
- Keyboard:
    - In list: `↑/↓` move selection, `Enter` opens, `e` archive, `#` delete, `Shift+u` mark unread, `j/k` Gmail-style aliases for ↑/↓. (All Wave 3.)
    - In compose: `⌘/Ctrl+Enter` send, `⌘/Ctrl+K` insert link, `⌘/Ctrl+Shift+A` attach.
- Color contrast: every text/bg combination above uses tokens that are LC in both themes (verified by spot-checking the bos-variables values against the contrast ratio calculator — Leah confirms during Wave 2 visual QA).
- Reduced motion: all transitions wrap in `@media (prefers-reduced-motion: reduce) { transition: none; animation: none; }` at the component level. Apply to every `cm-` class that animates.

---

## 13. Light + dark theme commitments

- **Zero `.dark` selectors.** Tokens already theme. If Fantem catches themself writing `.dark .cm-…`, stop and use a token.
- The hardcoded `#3b82f6`, `#e05252`, `#f59e0b`, `rgba(0,0,0,0.5)` literals listed in [the audit Section D](./comms-email-audit.md#d-token-violations) must not appear in Wave 2 code.
- Visual QA in both themes is part of Leah's Wave 2 task. Leah signs off only after toggling the theme.

---

## 14. Out of scope (deliberately)

So Fantem doesn't accidentally pull these in:

- **Threading server-side.** Frontend groups by `thread_id` client-side. Server-side merge is Wave 3 if we hit performance issues.
- **Server search.** "Search all email" is a Wave 3 capability; the input prop is ready, the backend is not.
- **Labels and custom folders.** Gmail labels and Outlook categories are surfaced as a flat list in Wave 3; Wave 2 ships only the canonical six folders.
- **Calendar invite parsing.** Out of scope. Wave 3 may add an inline RSVP card.
- **Email signature management.** Plain text signature stored in user settings is Wave 3.
- **Encryption / S/MIME.** Out of all roadmap.
- **Bulk select.** Multi-select rows + bulk actions are Wave 3.
- **List virtualization.** Wave 2 supports 50–100 rows comfortably without virtualization. If `getEmails(limit > 200)` becomes a thing, add virtualization in Wave 3.

---

## 15. Acceptance — what Leah checks in Wave 2 QA

Per-component checklist applied to every commit Fantem ships:

- [ ] Component file lives under [`lib/components/comms/email/`](../../frontend/src/lib/components/comms/email/).
- [ ] Class names use `cm-email-*` namespace.
- [ ] Zero `#xxxxxx`, `rgb()`, `rgba()` literals in `<style>` blocks (grep gate). Exceptions documented inline only if approved by Leah.
- [ ] Every color resolves to a `--bos-*` or `--dt`/`--dbg`/`--dbd` token.
- [ ] Every space resolves to a `--space-*` token.
- [ ] Every radius resolves to a `--radius-*` token.
- [ ] Every transition uses `--bos-transition-*`.
- [ ] Every z-index uses `--bos-z-*`.
- [ ] Modals use `.bos-modal*` classes.
- [ ] Buttons use `<PillButton>`, `.btn-pill-*`, `.btn-compact-*`, or `.btn-cta`. No raw `<button>` with custom styles.
- [ ] Inputs use `.bos-input` / `.bos-textarea` (or composed via FormField with the `.bos-input` skin if FormField is updated to that).
- [ ] Lucide icons only.
- [ ] Light AND dark theme verified.
- [ ] Every icon-only button has `aria-label`.
- [ ] Empty / loading / error variants implemented per §9.
- [ ] DOMPurify allowlist matches [the original](../../frontend/src/routes/(app)/communication/email/+page.svelte#L399).
- [ ] No new design tokens introduced. (If one was unavoidable, raise to Axis with rationale before Leah signs off.)
- [ ] No `any` types. (Strict-TS gate, separate from Leah but a non-negotiable.)

---

## 16. Open questions for Axis / Roberto

Marked here so they don't block Fantem and Leah resolves before Wave 2 starts.

1. **Sidebar Compose CTA — `.btn-cta` glow or regular `btn-pill-primary`?** Spec defaults to regular pill primary. Axis confirm.
2. **Provider dot color in list rows.** Using `--bos-status-error` (red) for Gmail and `--bos-status-info` (blue) for Outlook — repurposing status tokens for brand identity. Acceptable per Axis pre-agreement; flag for Roberto.
3. **`DocumentEditor` embeddability.** Spec assumes one half-day spike to confirm. If blocked, fallback to plain contenteditable + minimal toolbar — no design difference.
4. **`cm-toast` placement.** Spec puts the toast helper in the comms-tab layout so it spans Calendar/Email/Channels. Confirm whether the app already has a global toast (search of `lib/components/notifications/` returned NotificationDropdown/Item/List — those are persistent app notifications, not transient toasts). If a global toast exists I missed, use it instead.
5. **Send-later schema.** Adds `scheduled_send_at` column to `emails`. Ghost's call whether this lands in Wave 2 backend or slips to Wave 3.
6. **Minimize-to-chip in compose.** Marked "if cheap". Confirm whether Fantem should attempt or strip from Wave 2.

If any of these flip, the spec changes in three places at most: the relevant section + the §15 acceptance + the §14 out-of-scope. Easy to maintain.