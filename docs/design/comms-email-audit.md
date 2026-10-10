# Communications · Email — UX Audit

**Author:** Leah (UI/UX, Wave 1)
**Branch:** `feat/communications-module`
**Audit date:** 2026-05-02
**Subject:** [`frontend/src/routes/(app)/communication/email/+page.svelte`](../../frontend/src/routes/(app)/communication/email/+page.svelte) — 1,124 lines, single-file implementation
**Scope:** UX, visual design, interaction, copy, token compliance. Backend stubbing is documented in [`COMMUNICATIONS_AUDIT.md`](../COMMUNICATIONS_AUDIT.md) and is not duplicated here except where it changes what users see.

---

## TL;DR

The current Email tab is a competent three-pane Gmail clone built in a single 1,124-line file. The structure is sound — folders / list / preview is the right shape — but four problems compound:

1. **Single-provider single-account.** UI is hard-coded to Gmail. There is no surface for Outlook (the backend exists), no concept of "all inboxes," and no provider switcher.
2. **Decorative dead controls.** Paperclip, Archive button in preview, Search input outside Compose, and the entire Drafts folder render but do nothing meaningful.
3. **Errors are invisible.** `console.error` calls on five paths; the user sees a static "Failed to load emails" banner at most. No toasts, no retry affordance beyond inline "Try again."
4. **Token drift.** Three concrete violations: hardcoded `#3b82f6` fallbacks, hardcoded `rgba(224, 82, 82, …)` for error background, and a one-off compose modal that re-implements `.bos-modal` from scratch.

The single-file shape also blocks parallel work: there's nothing to import or compose against. Even read-only changes mean editing 1,124 lines.

---

## A. Architecture & file structure

### A.1 — Single 1,124-line page, no decomposition
**Severity:** High · **Where:** entire file

The whole feature lives in `email/+page.svelte`: state, helpers, three panels, the preview, the compose modal, and ~590 lines of `<style>`. The neighbouring [`calendar/+page.svelte`](../../frontend/src/routes/(app)/communication/calendar/+page.svelte) (566 lines, the proven template) decomposes into ten components in [`lib/components/calendar/`](../../frontend/src/lib/components/calendar/) — sidebar, toolbar, status banner, three views, two modals, plus `calendarUtils.ts`. The page itself orchestrates; components own rendering.

Email needs the same treatment: any redesign starts by extracting components, otherwise three engineers cannot work the file simultaneously without merge conflict on every change.

### A.2 — `$effect` re-fetches on every reactive read
**Severity:** Medium · **Where:** [`email/+page.svelte:227-231`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L227-L231)

```svelte
$effect(() => {
    if (accessStatus?.has_access) {
        loadEmails();
    }
});
```

Reads `accessStatus` and inside `loadEmails` reads `currentFolder`. There's no explicit dependency on folder, so the trigger is implicit — currently fine, but fragile. The calendar module's pattern (`selectedMeetingType; loadEvents();` at [`calendar/+page.svelte:140-142`](../../frontend/src/routes/(app)/communication/calendar/+page.svelte#L140-L142)) reads dependencies explicitly and is more predictable. Standardize on that.

### A.3 — Compose modal is an inline overlay, not a `.bos-modal`
**Severity:** Medium · **Where:** [`email/+page.svelte:441-536`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L441-L536), corresponding CSS [`:979-1113`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L979-L1113)

Re-implements modal scaffolding (overlay div, escape handler, click-outside, header, footer) instead of using `.bos-modal-overlay` / `.bos-modal` / `.bos-modal-header` / `.bos-modal-body` / `.bos-modal-footer` per the [design system](../COMMUNICATIONS_DESIGN_SYSTEM.md#modals). Every other modal in the app uses the standard. The redesign must adopt it.

---

## B. Information architecture & content

### B.1 — Six folders, one provider, no provider concept
**Severity:** High · **Where:** [`email/+page.svelte:42-49`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L42-L49)

`folderMeta` is a flat list of Gmail folders. The schema and Microsoft handler already support Outlook (`microsoft_mail_messages`, `microsoft/handler.go`), but this page can never surface it. Wave 2 needs a provider concept above the folder list — minimum: provider switcher (Gmail / Outlook / All) — and a folder taxonomy that maps both providers' canonical mailboxes onto a unified set.

### B.2 — Drafts folder is fully cosmetic
**Severity:** High · **Where:** [`email/+page.svelte:45`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L45) (folder list), the entire compose modal

There is no draft persistence — `handleSendEmail` either sends or stays open. `is_draft` on the `Email` type is set, but never written or read by this page. Clicking "Drafts" returns a stubbed empty list. Either:
- Wire it (auto-save while composing → write `is_draft=true` rows; click resumes), OR
- Remove the Drafts folder until it's wired.

The redesign chooses "wire it" because draft auto-save is in the Wave 2 brief.

### B.3 — Folder name shown verbatim, lowercase, in toolbar
**Severity:** Low · **Where:** [`email/+page.svelte:310`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L310)

```svelte
<h3 class="ch-inbox-toolbar__title">{currentFolder}</h3>
```

Renders `inbox` lowercase. Capitalised by CSS `text-transform: capitalize` ([`:661`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L661)). For multi-word folders (e.g. "All inboxes") this fails. Use `folderMeta` `label` field, which is already correctly cased.

### B.4 — Threads are flat, treated as individual messages
**Severity:** High · **Where:** rendering loop [`email/+page.svelte:351-374`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L351-L374)

`Email.thread_id` exists on the type ([`gmail/types.ts:20`](../../frontend/src/lib/api/gmail/types.ts#L20)) but is never used. A 30-message reply chain renders as 30 list rows, all with the same subject. This is the highest-impact UX defect — the inbox feels noisy and lossy. The redesign collapses into thread rows with a count badge and expands into a stacked thread view in the preview pane.

### B.5 — No search-within-thread, no advanced search, no labels
**Severity:** Low (Wave 1) / Medium (Wave 3) · **Where:** [`email/+page.svelte:322-331`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L322-L331)

Search is client-side only and scoped to the currently-loaded 50 rows (via `getEmails({ limit: 50 })`). Anything not in that page is invisible. Acceptable for Wave 1 if we add the limitation in copy ("Search this folder…") but the long-term fix is server-side search. Out of scope for this redesign except as a copy fix and an explicit "search server" note for Wave 3.

---

## C. Decorative / non-functional UI

These render but do nothing useful. Either wire or remove. The redesign removes any control that doesn't have a Wave 2 implementation owner.

| Control | Location | State | Action |
|---|---|---|---|
| Paperclip (compose attach) | [`:520-523`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L520-L523) | No `onclick`, no file input | **Wire** in Wave 2 (compose attachments are scoped) |
| Archive (preview actions) | [`:425-428`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L425-L428) | No `onclick` | **Wire** to `archiveEmail(id)` (Ghost wave 1 adds the route) |
| Drafts folder | [`:45`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L45) | Lists empty regardless | **Wire** to draft persistence (Wave 2) |
| Search input | [`:322-331`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L322-L331) | Filters loaded 50 rows only | **Keep but rescope copy** — `Search this folder…`, plus Wave 3 server-search note |
| "X emails synced" footer | [`:303`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L303) | Always reads `0` until Ghost wires `getGmailStats` | **Keep** — Ghost wave 1 fixes the data |

---

## D. Token violations

These violate [`COMMUNICATIONS_DESIGN_SYSTEM.md`](../COMMUNICATIONS_DESIGN_SYSTEM.md). Listed for Fantem's cleanup pass; they must not survive into Wave 2.

| File:line | Violation | Replace with |
|---|---|---|
| [`:616`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L616) `var(--accent-blue, #3b82f6)` | Hardcoded fallback to `#3b82f6` | `var(--bos-accent-blue)` (always defined, `bos-variables.css:40`) |
| [`:740`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L740) same | Same | Same |
| [`:896`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L896) same in preview links | Same | `var(--bos-accent-blue)` |
| [`:828`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L828) `var(--color-error, #e05252)` | Custom error token + hardcoded fallback | `var(--bos-status-error-text)` (`bos-variables.css:238`) |
| [`:1029`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L1029) same | Same | Same |
| [`:1022-1023`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L1022-L1023) `rgba(224, 82, 82, 0.1)` background, `rgba(224, 82, 82, 0.2)` border | Hardcoded error tint + arbitrary border colour | `background: var(--bos-status-error-bg)`; `border-color: var(--bos-status-error-bg)` (or define a `--bos-status-error-border` if reused; current scope: just the bg token) |
| [`:974`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L974) `var(--color-warning, #f59e0b)` | Custom warning token | `var(--bos-status-warning-text)` (`bos-variables.css:235`) |
| [`:982`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L982) `rgba(0, 0, 0, 0.5)` overlay | Inline overlay color | `var(--bos-modal-backdrop)` (`bos-variables.css:177`) |
| [`:986`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L986) `z-index: 50` | Off-scale z-index | `var(--bos-z-modal)` (`bos-variables.css:195`) |
| [`:992`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L992) `border-radius: 12px`, `:994` `box-shadow: 0 8px 32px rgba(0,0,0,0.18)` | Off-scale radius and ad-hoc shadow on compose modal | `var(--bos-modal-radius)` (`bos-variables.css:181`), `var(--bos-modal-shadow)` (`:182`) — or use the `.bos-modal` class outright (preferred) |
| [`:585`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L585), [`:714`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L714) `transition: background 0.15s` | Hardcoded duration | `var(--bos-transition-fast)` (`bos-variables.css:210`) |
| Multiple `0.81rem`, `0.83rem`, `0.85rem`, `0.88rem` font sizes throughout | Off-scale typography | Snap to `--text-xs` (0.75rem) / `--text-sm` (0.875rem) / `--text-base` (1rem). Pick one: **the redesign uses `--text-xs` for metadata, `--text-sm` for body and list rows, `--text-base` for preview body, `--text-lg` for subject**. |
| `width: 185px` sidebar, `width: 320px` list, `46px` label | Off-scale spacing | Use `--space-*` tokens or document the override centrally |
| Custom button class `.ch-inbox-compose-btn` ([`:558-565`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L558-L565)) | Reinvents `.btn-pill-primary .btn-pill-sm` with margin tweaks | Wrapper div for layout, button uses `<PillButton variant="primary" size="sm" block />` |

---

## E. State coverage — empty / loading / error

### E.1 — Empty states are inconsistent

Three empty states exist with three different shapes:

| State | Location | Shape | Notes |
|---|---|---|---|
| Not connected | [`:246-263`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L246-L263) | 64×64 round icon, title, body, primary CTA | This is the canonical pattern |
| Empty folder | [`:346-349`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L346-L349) | Tiny mail icon + "No emails in inbox" — no illustration, no CTA | Bare |
| No email selected | [`:431-434`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L431-L434) | Mail icon + "Select an email to read" | Bare |

The "not connected" empty state is the one to template. Empty folder and no-selection should match its rhythm: icon → title → one sentence of context → optional secondary action. The redesign normalises.

### E.2 — Loading is just a spinner; no skeleton
**Severity:** Medium · **Where:** [`:334-337`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L334-L337)

A `Loader2` spinner replaces the entire list area on every load — including folder switches, which jolts. The chat module uses `SkeletonLoader.svelte` ([`lib/components/ui/SkeletonLoader.svelte`](../../frontend/src/lib/components/ui/SkeletonLoader.svelte)). Email list deserves a skeleton row (avatar + sender + subject placeholders) and the preview pane deserves a body skeleton. Spinner remains for the "global initial bootstrap" only.

### E.3 — Errors silenced or buried
**Severity:** High · **Where:** five paths

| Source | Handling | User experience |
|---|---|---|
| `loadAccessStatus` ([`:73-79`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L73-L79)) | `console.error` only | Page renders the unconnected empty state regardless of the actual cause — could be auth expiry, network, or backend down |
| `loadEmails` ([`:81-98`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L81-L98)) | Sets `error` string, renders inline panel with "Try again" | OK pattern, but only here |
| `loadStats` ([`:100-108`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L100-L108)) | `console.error` only | Sidebar count silently `0` |
| `markAsRead` ([`:138-141`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L138-L141)) | `console.error` only | Email visually flips read; if the API fails, next refresh resurrects unread state — confusing |
| `handleRequestAccess` ([`:144-154`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L144-L154)) | Sets `error` but the user is mid-OAuth-redirect — they likely never see it |

The redesign standardises on a comms-namespaced toast (`cm-toast`, since `notifications/` exists but is for app-level notifications, not transient action feedback). All five paths above use it.

---

## F. Accessibility

### F.1 — Generally good
ARIA labels present on every icon-only button (folder buttons [`:281`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L281), sync [`:316`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L316), search input [`:329`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L329), compose toolbar buttons). Modal has `role="dialog"`, `aria-modal="true"`, `aria-label`. Escape key handled.

### F.2 — Live region missing for sync / send
**Severity:** Low · **Where:** [`:113-124`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L113-L124), [`:156-188`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L156-L188)

"Sending…" is a button label change only. Screen-reader users get no announcement when the send completes or fails. Add a polite live region for sync status, send status, errors. The toast pattern handles this naturally.

### F.3 — Click-outside to close uses a `<div>` overlay
**Severity:** Low · **Where:** [`:443`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L443)

`<!-- svelte-ignore a11y_no_static_element_interactions -->` on a div with click and keydown — implies it's not a real button. Switch to `.bos-modal` which has an actual button as overlay (or backdrop element with `role="presentation"` + a sibling close button). Lint suppression is the smell.

### F.4 — Focus trap on compose modal
**Severity:** Low · **Where:** entire compose modal

No focus trap. Tab past the Discard button and focus escapes into the page behind. Spec the use of an existing modal wrapper that does focus trapping (none exists today; Axis to confirm whether to add one or leave to platform — note this as a Wave 2 question, not a blocker).

### F.5 — Keyboard nav between list and preview
**Severity:** Low (Wave 3) · **Where:** —

Up/Down does nothing in the list; Enter on a row does nothing (it's a `<button>` with `onclick`, but visual focus styles are absent). Wave 3 Gmail-style shortcuts (`j/k/e/#`) are in the sprint plan; spec this as part of that pass with a stubbed key-handler hook in the components.

---

## G. Visual / layout

### G.1 — Sidebar width fixed at 185px, list at 320px
**Severity:** Low · **Where:** [`:549`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L549), [`:640`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L640)

Hard widths. On narrow viewports, preview pane shrinks below readability. No collapse. Calendar's pattern ([`calendar/+page.svelte:454`](../../frontend/src/routes/(app)/communication/calendar/+page.svelte#L454)) wraps the sidebar in a collapsible container with a transition. Redesign mirrors that.

### G.2 — Avatar is a one-letter initial on a flat grey
**Severity:** Low · **Where:** [`:748-760`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L748-L760)

Background is `--dt3` (tertiary text), color is `--dbg` — works but feels stamped. A consistent initial-avatar component used across chat / channels / email would be better. Out of scope to define here; spec calls out that compose v2 uses the same component as Channels, which Leon will spec.

### G.3 — Subject + snippet are concatenated into one ellipsised line
**Severity:** Low · **Where:** [`:367-370`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L367-L370)

```svelte
<span class="ch-inbox-subject">{email.subject || '(no subject)'}</span>
<span class="ch-inbox-preview"> — {email.snippet || ''}</span>
```

Read-only Gmail and Outlook both present this on two lines on desktop (subject bold, snippet on line 2). Single-line is cramped at 320px and silently drops the snippet. Redesign: two-line row, subject line + sender, snippet line below.

### G.4 — Time format inconsistent between row and preview
**Severity:** Low · **Where:** `formatDate` ([`:52-65`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L52-L65)) vs `toLocaleString()` ([`:391`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L391))

Row says "9:42 AM" or "May 1"; preview says "5/1/2026, 9:42:13 AM". Spec: define one date helper in `commsUtils.ts` with two formatters — relative-row and full-preview — used by both surfaces.

### G.5 — No HTML body styling
**Severity:** Medium · **Where:** [`:889-898`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L889-L898)

Sanitised HTML renders into `.ch-inbox-preview__html` with only `font-size`, `color`, `line-height` set. Marketing emails (tables, columns, images) overflow horizontally. Wave 2 needs:
- `max-width: 100%` clamp
- `img { max-width: 100%; height: auto; }`
- Quoted-text collapse (the `> on … wrote:` block)
- Link styling (currently relies on browser default + `--accent-blue` fallback)

### G.6 — Inline reply quote uses ASCII separators
**Severity:** Low · **Where:** [`:203`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L203)

```js
composeBody = `\n\n---\nOn ${...}, ${...} wrote:\n${...}`;
```

Plain-text only, no rich text. This stays plain-text in Wave 1 but Compose v2 in Wave 2 has a rich text body — quoting will be a styled block, not ASCII.

---

## H. Data shape & integration concerns (UX-visible)

These aren't backend bugs — they affect what the user sees.

### H.1 — `from_name` falls back to email
**Severity:** Low · **Where:** [`:364`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L364), [`:366`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L366), [`:386`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L386), [`:389`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L389)

Fine. Worth noting that `from_name` for newsletters is often "Stripe Notifications via …" — the avatar then shows "S" for everything that comes through Mailgun/Sendgrid. Wave 3 could derive avatar from the email domain. Not Wave 1/2 work — flag for Roberto.

### H.2 — `to_emails` / `cc_emails` / `bcc_emails` never displayed
**Severity:** Medium · **Where:** preview header [`:382-394`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L382-L394)

Preview shows sender only. Recipients aren't visible. For an email client this is a basic miss. Spec includes a "to: a@b, c@d (+ 2)" line in the preview header.

### H.3 — `attachments` field ignored
**Severity:** Medium · **Where:** preview body — never iterates `selectedEmail.attachments`

`Email.attachments` is on the type. Render under preview body, with filename + size + mime icon. Click → backend download endpoint (Ghost owns, out of scope here; spec includes the UI affordance).

### H.4 — No unread/star toggling from list
**Severity:** Low · **Where:** [`:351-374`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L351-L374)

`is_starred`, `is_important` exist on the type, never rendered or toggled. Drafts folder also ignores `is_draft` (see B.2). Spec adds star toggle on the list row (icon button positioned right of the time) and on the preview header.

---

## I. Light + dark theme

### I.1 — Generally honest
The page uses `--dt`/`--dbg`/`--dbd` aliases throughout (good). No `.dark` selectors — the alias layer handles theme switching. Verified by reading `bos-variables.css:299+` for the dark overrides — all the tokens we use are themed.

### I.2 — Two exceptions
The hardcoded `#3b82f6`, `#e05252`, `#f59e0b` fallbacks (see Section D) are not themed. In dark mode they're approximately right but never tuned — the canonical dark-mode versions of those tokens have different values (e.g. `--bos-accent-blue` is themed differently than the literal). Replacing fallbacks with the canonical tokens fixes this automatically.

### I.3 — Compose modal backdrop hardcoded
[`:982`](../../frontend/src/routes/(app)/communication/email/+page.svelte#L982) `rgba(0, 0, 0, 0.5)` — same in both themes. Token `--bos-modal-backdrop` is themed (`0.5` light, `0.7` dark per `bos-variables.css`). Fix: use the token.

---

## J. Summary — what the redesign must do

In priority order. Each item maps to a section in the spec doc.

1. **Decompose** the page into ~10 components mirroring the calendar pattern.
2. **Unified inbox** with provider switcher (Gmail / Outlook / All) above the folder list.
3. **Thread view** in the preview — collapse messages with the same `thread_id` into stacks.
4. **Compose v2** as a `.bos-modal` with rich text body, attachment chips, contact autocomplete, draft auto-save, send-later affordance.
5. **State coverage** — three empty states with consistent rhythm; skeletons for list and preview; toasts for all errors and async actions.
6. **Token compliance** — zero hex/rgb literals, zero off-scale typography, zero ad-hoc z-index. The checklist in the [design system doc](../COMMUNICATIONS_DESIGN_SYSTEM.md#design-token-enforcement-checklist) is the gate.
7. **Wired controls only** — every button has an `onclick` doing real work, or the button doesn't ship.

The spec at [`docs/design/comms-email-spec.md`](./comms-email-spec.md) is concrete enough that Fantem can implement without further design questions.