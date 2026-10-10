# Communications Module — Design System Reference

**Branch:** `feat/communications-module`
**Audit date:** 2026-05-01
**Purpose:** Single source of truth for what tokens, classes, and components every UI change in this module MUST use. No hardcoded hex/rgb. No new patterns.

---

## TL;DR — the rules

1. **Use `--bos-*` tokens** from `frontend/src/lib/modules/theme/styles/bos-variables.css` (466 lines, single source of truth).g
2. **Or use shorthand aliases** `--dt`, `--dt2`, `--dbg`, `--dbg2`, `--dbd` from `frontend/src/app.css:66-74` (these alias into `--bos-*`).
3. **Reuse existing classes** from `app.css`: `.btn-pill-*`, `.btn-compact-*`, `.btn-cta`, `.bos-modal`, `.bos-input`, `.glass-card`. Don't write new ones unless reviewed.
4. **Existing components first.** `lib/components/ui/*`, `lib/components/calendar/*`, `lib/components/chat/*` already implement most of what comms needs. Mimic their structure.
5. **Two themes.** Everything must work in light and dark — tokens already handle this; if you have to write a `.dark` rule, you are likely doing it wrong.

---

## Token system (the canonical layer cake)

```
┌─────────────────────────────────────────────────────────────────┐
│  bos-variables.css   ← SOURCE OF TRUTH (light + dark)           │
│  --bos-{category}-{property}                                    │
│  466 lines, every color/shadow/radius/transition lives here     │
└────────────────────────────────┬────────────────────────────────┘
                                 │ aliased by
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│  app.css :root            ← shorthand aliases                   │
│  --dt, --dt2, --dt3, --dt4   (text)                             │
│  --dbg, --dbg2, --dbg3       (backgrounds)                      │
│  --dbd, --dbd2               (borders)                          │
│  --space-*, --radius-*, --text-*, --font-*  (scales)            │
└────────────────────────────────┬────────────────────────────────┘
                                 │ also aliased by
                                 ▼
┌─────────────────────────────────────────────────────────────────┐
│  variables.css            ← HSL compatibility layer for Tailwind │
│  --background, --foreground, --primary, --border, etc.          │
│  Used by `hsl(var(--border))` patterns. Don't add new ones.     │
└─────────────────────────────────────────────────────────────────┘
```

### Color tokens — what to grab when

**Text** (`bos-variables.css:43-56`)
```
--bos-text-primary-color    →  --dt    (headings, body)
--bos-text-secondary-color  →  --dt2   (subtitle, label, secondary)
--bos-text-tertiary-color   →  --dt3   (helper text, timestamps)
--bos-text-disable-color    →  --dt4   (disabled, placeholder)
--bos-text-emphasis-color           (titles needing extra weight)
```

**Backgrounds** (`bos-variables.css:65-72`)
```
--bos-v2-layer-background-primary    →  --dbg    (page surface)
--bos-v2-layer-background-secondary  →  --dbg2   (sidebar, card)
--bos-v2-layer-background-tertiary   →  --dbg3   (nested, hover)
--bos-v2-layer-background-hoverOverlay         (hover ghost)
--bos-v2-layer-background-modal                (modal backdrop)
--bos-v2-layer-background-overlayPanel         (popover)
```

**Borders & dividers** (`bos-variables.css:104-106`)
```
--bos-border-color   →  --dbd
--bos-divider-color
--bos-v2-layer-insideBorder-border       (semi-transparent)
```

**Status / semantic** (`bos-variables.css:230-247`)
```
--bos-status-success / -bg / -text       (#10b981 light, #4ade80 dark)
--bos-status-warning / -bg / -text       (amber)
--bos-status-error   / -bg / -text       (red)
--bos-status-info    / -bg / -text       (blue)
--bos-status-neutral / -bg / -text       (gray)
--bos-status-accent  / -bg / -text       (purple — for AI/agent labels)
```

**The ONE accent — blue glow** (`bos-variables.css:40-41, 285-287`)
The whole product has exactly one accent color: `#3B82F6` (blue). Used for active nav, CTA glow, unread dots, focus rings. **Don't introduce a second accent.**

```
--bos-accent-blue       #3B82F6
--bos-accent-blue-rgb   59, 130, 246
--bos-nav-active        active tab/nav color
--bos-nav-active-bg     6% blue tint background
--bos-nav-active-glow   blue glow shadow
```

**Category colors** (`bos-variables.css:272-279`) — for icon differentiation only, NOT general UI accents. The communications module's own category color is:
```
--bos-category-communication  #a855f7  (purple, dark: #c084fc)
```

### Scales

```
--space-0..24       0, 4px, 8px, 12px, 16px, 20px, 24px, 32px, 40px...
--radius-xs..3xl    4px, 8px, 12px, 16px, 20px, 24px, 32px
--radius-full       9999px
--text-xs..5xl      0.75rem .. 3rem
--font-light..black 300..900
--icon-xs..4xl      16, 20, 24, 32, 48, 64, 80, 96 (lucide-svelte sizes)
```

### Z-index (`bos-variables.css:190-199`) — use these, do not invent

```
--bos-z-base            0
--bos-z-dropdown        100
--bos-z-sticky          200
--bos-z-popover         300
--bos-z-modal-backdrop  900
--bos-z-modal           1000
--bos-z-modal-nested    1010
--bos-z-toast           2000
--bos-z-tooltip         2500
```

### Transitions

```
--bos-transition-fast   150ms ease-out   (hovers, micro)
--bos-transition-normal 200ms ease-out   (default)
--bos-transition-slow   300ms ease-out   (panels, drawers)
```

### Shadows (`bos-variables.css:136-141`)

```
--bos-shadow-1     subtle (4px blur)
--bos-shadow-2     standard with inset border (12px)
--bos-shadow-3     elevated (20px)
--bos-popover-shadow / --bos-menu-shadow / --bos-float-button-shadow
--bos-modal-shadow
```

### Modal tokens (`bos-variables.css:177-184`)

```
--bos-modal-backdrop, -blur, -bg, -border, -radius, -shadow
--bos-modal-header-bg, -border
```

---

## Reusable components — already built, use them

### UI primitives (`frontend/src/lib/components/ui/`)
| Component | When to use |
|---|---|
| `PillButton.svelte` | Any button — wraps `.btn-pill-*` classes. Variants: primary, secondary, ghost, danger, success, warning, outline, soft, link. Sizes: xs, sm, md, lg, xl, icon. |
| `GlassCard.svelte` | Any elevated card. Variants: default, frosted, subtle, panel, panel-dark, surface. |
| `LoadingSpinner.svelte` | Loading states |
| `SkeletonLoader.svelte` | Placeholder skeletons during fetch |
| `ErrorBoundary.svelte` | Wrap async sections |
| `Pagination.svelte` | List pagination |
| `Tooltip.svelte` | Hover explanations |

### CSS class patterns (in `app.css`)

**Buttons:**
- `.btn-pill` + `.btn-pill-{primary|secondary|ghost|danger|success|outline|soft|link}` (`app.css:1211-1564`) — iOS pill style, the standard.
- `.btn-pill-{xs|sm|lg|xl}` for size, `.btn-pill-icon` for icon-only.
- `.btn-compact` + `.btn-compact-{primary|secondary|ghost|outline|soft}` (`app.css:2212-2420`) — for toolbars, dense rows. THE CHANNELS AND EMAIL PAGES ALREADY USE THESE.
- `.btn-cta` (`app.css:1303-1358`) — black + blue glow. The signature element. Used for primary CTAs (Compose, New, Connect). Animated blue pulse heartbeat.
- `.btn` / `.btn-primary` / `.btn-secondary` / `.btn-ghost` (`app.css:249-298`) — older base. Prefer `.btn-pill-*` for new code.

**Modals:**
- `.bos-modal-overlay` + `.bos-modal` + sizes `--sm|md|lg|xl|full` (`app.css:3055-3163`)
- `.bos-modal-header` / `.bos-modal-title` / `.bos-modal-subtitle` / `.bos-modal-close` / `.bos-modal-body` / `.bos-modal-footer`

**Forms:**
- `.bos-label` / `.bos-label--required` (`app.css:3171-3182`)
- `.bos-input` / `.bos-textarea` / `.bos-select` (`app.css:3185-3248`) with `--error` modifiers

**Cards & glass:**
- `.glass-card` / `-frosted` / `-subtle` / `.glass-panel` / `.glass-surface` (`app.css:1188+`)
- `.card` (`app.css:330+`)

**Inputs (alt rounded variant):**
- `.input` / `.input-square` (`app.css:302-326`)
- `.input-rounded` (`app.css:2783+`)

### Domain components to mimic / reuse

The communications module sits next to TWO already-built reference implementations:

**Calendar (working, polished)** — `frontend/src/lib/components/calendar/`:
- `CalendarSidebar.svelte`, `CalendarToolbar.svelte`, `CalendarStatusBanner.svelte`
- Three view components: `CalendarWeekDayView`, `CalendarMonthView`, `CalendarAgendaView`
- Modals: `CalendarEventModal`, `CalendarEventForm`, `SchedulingModal`
- Utilities: `calendarUtils.ts`
- This is the **architectural template** for how to break up a comm tab — sidebar + toolbar + content + modals + utils.

**Chat (working)** — `frontend/src/lib/components/chat/`:
- `conversations/`, `focus/`, `input/`, `messages/`, `modals/`, `panels/`, `shared/`
- Reuse the **message bubble**, **input**, and **conversation list** primitives for Channels (Slack/Teams) and Email thread views.

**AI elements** — `frontend/src/lib/components/ai-elements/`:
- `Message.svelte`, `Conversation.svelte`, `PromptInput.svelte`, `Suggestion.svelte`, `Loader.svelte`
- For agent-driven actions inside the comm module (auto-draft, summarize thread, triage suggestions).

### Forms — `frontend/src/lib/components/forms/`
- `FormField.svelte`, `FormSection.svelte` — compose modal should use these.

---

## CSS-in-Svelte conventions (observed across calendar/email/channels pages)

These are de-facto rules across the existing comms code:

- **BEM-style local class names** scoped per component, e.g. `.ch-inbox-row`, `.ch-chat-msg__avatar`. The `ch-` prefix is communications-namespace.
- **`<style>` block at the bottom of every `+page.svelte`** with all colors referencing tokens (`var(--dbg)`, `var(--dt)` etc.).
- **Svelte 5 runes everywhere** — `$state`, `$derived`, `$effect`. No legacy `let` reactive declarations, no `createEventDispatcher`.
- **Callback props, not events** — `onclick={...}`, `onSelect={(x) => ...}`. Confirmed in `email/+page.svelte` and `channels/+page.svelte`.
- **Tailwind allowed** but compose with token utilities. Many places use both class-based and style-based.
- **DOMPurify for any user HTML** (`email/+page.svelte:399` is the gold standard — tight allowlist).
- **Lucide icons exclusively** — `import { Mail, Inbox, Send } from 'lucide-svelte'`. Sizes: typically 14, 16, 18 inside compact UI.

---

## What to avoid / known anti-patterns to fix

These show up in the current comm module and **must be cleaned up** as part of the sprint:

1. **`any[]` typing** in `channels/+page.svelte:8-15` — kills runtime safety.
2. **Hardcoded color values** — none in token files but watch for them in new component code; e.g. `var(--accent-blue, #3b82f6)` fallbacks (`email/+page.svelte:616, 740`) are unnecessary, the token is always defined.
3. **Decorative buttons that don't do anything** — Paperclip in compose, Archive in email actions, Search in channels header. Either wire or remove.
4. **Silent `catch { events = [] }`** — `calendar/+page.svelte:190`. Errors must reach the user (toast).
5. **Custom button styling per component** — `.ch-inbox-compose-btn` in email page should just be `.btn-pill-primary .btn-pill-sm` plus a layout wrapper.
6. **Inline opaque colors** — e.g. `rgba(224, 82, 82, 0.1)` for error backgrounds (`email/+page.svelte:1022`). Use `var(--bos-status-error-bg)` instead.
7. **Two competing API surfaces for Gmail** — see `COMMUNICATIONS_AUDIT.md`. This is the bigger architectural fix.

---

## Design token enforcement checklist

Before merging a comms PR, verify:

- [ ] Every color references a `--bos-*` or `--dt`/`--dbg`/`--dbd` token. Zero `#xxxxxx` or `rgb()` literals in component CSS.
- [ ] Every spacing uses `--space-*` or `var(--space-N)`. No arbitrary `12px` literals.
- [ ] Every radius uses `--radius-*`. No arbitrary `8px` literals.
- [ ] Every animation duration uses `--bos-transition-*`.
- [ ] Modal uses `.bos-modal*` classes OR existing `<Modal>` wrapper. No custom overlay div.
- [ ] Buttons use `.btn-pill-*`, `.btn-compact-*`, `.btn-cta`, OR the `<PillButton>` component. No raw `<button>` with custom styles.
- [ ] Inputs use `.bos-input` / `.bos-textarea` / `.bos-select` OR `<FormField>`.
- [ ] Status colors use `--bos-status-*` tokens, never raw `#10b981`.
- [ ] Light AND dark modes work without writing a `.dark` selector. If you wrote one, the token system can probably do it for you.
- [ ] `aria-label` on every icon-only button (existing comms pages do this consistently — keep it up).
- [ ] DOMPurify sanitization on any user-supplied HTML. Allowlist matches `email/+page.svelte:399`.
