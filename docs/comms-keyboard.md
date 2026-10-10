# Communications · Keyboard shortcuts

**Module:** `feat/communications-module` · **Wave:** 3 · **Author:** Leah
**Helper:** [`frontend/src/lib/components/comms/commsKeyboard.ts`](../frontend/src/lib/components/comms/commsKeyboard.ts) (shared between email and channels tabs)

---

## Why a shared helper

Both tabs need Gmail/Slack-style hotkeys. Without a single dispatcher we end up duplicating focus-detection, modifier-resolution (Cmd vs Ctrl), and key-spec parsing in two places. `bindShortcuts` is the one place all of that lives.

Drop in via Svelte 5 `$effect` — the function returns a teardown so cleanup is automatic on unmount.

```ts
import { bindShortcuts } from '$lib/components/comms/commsKeyboard';

$effect(() => {
  if (modalOpen) return; // suspend while a modal owns its own keymap
  return bindShortcuts([
    { key: 'j', description: 'Next item', handler: nextItem },
    { key: 'Mod+Enter', description: 'Submit', handler: submit, allowInInput: true },
  ]);
});
```

## Spec format

Specs are case-insensitive. Modifiers separate with `+`.

| Token | Meaning |
|---|---|
| Single character | Literal key — `j`, `k`, `e`, `r`, `c`, `/`, `?`, `#` |
| `Enter` | Enter / Return |
| `Escape` | Esc |
| `Mod` | Cmd on macOS, Ctrl elsewhere |
| `Shift`, `Alt` (or `Option`), `Ctrl`, `Cmd` | Explicit modifier |

### Symbol-key rule

Symbol keys (`#`, `?`, `/`) carry shift implicitly in `event.key` on US keyboards. Write the literal symbol, not `Shift+3`. Letter and digit keys still discriminate on shift — `u` and `Shift+u` are different bindings.

## Default behaviors

- **Typing in inputs is respected.** While focus is on `<input>`, `<textarea>`, `<select>`, or `[contenteditable]`, shortcuts skip — except when `allowInInput: true`. (Compose's `Mod+Enter` to send is the canonical exception.)
- **First match wins.** If two bindings would fire on the same event, the earlier one runs.
- **Honors `event.defaultPrevented`.** If something downstream already handled the key, we don't double-fire.
- **Calls `preventDefault` by default.** Set `preventDefault: false` if your handler is purely additive.

## Email tab bindings

Bound at the orchestrator level — suspended when the compose modal is open, since the modal owns Esc and Mod+Enter inside its own scope.

| Key | Action |
|---|---|
| `j` | Next thread (selects + opens) |
| `k` | Previous thread |
| `Enter` | Re-open the selected thread (no-op if already loaded) |
| `Escape` | Close the open thread (clears preview pane) |
| `e` | Archive the selected thread |
| `#` | Move the selected thread to Trash |
| `r` | Reply to the selected thread |
| `c` | Open compose |
| `/` | Focus the search input |
| `?` | Show shortcuts cheat sheet (toast) |

Inside the **compose modal**:

| Key | Action |
|---|---|
| `Escape` | Close compose (auto-saved draft persists) |
| `Mod+Enter` | Send |

These are handled by the modal's own keydown listener, not the global helper.

## Channels tab bindings (Wave 3 — Leon)

Channels reuses the same helper. Page-level keymap is suspended while the thread drawer is open; the drawer binds its own.

| Key | Action |
|---|---|
| `j` / `k` | Next / previous channel |
| `Enter` | Open selected channel |
| `Escape` | Close thread drawer (when open) |
| `r` | Focus reply input in current view |
| `Mod+Enter` | Send message (in input) |

## Adding a new shortcut

1. Pick a key that doesn't conflict with browser/OS reserved keys (avoid `Mod+W`, `Mod+T`, `Tab`, `Mod+Shift+R`, etc.).
2. Add an entry to the `bindShortcuts` array in the orchestrator (or relevant scope).
3. Provide a `description` so the cheat-sheet toast and any future help dialog can pick it up.
4. If the binding should fire while typing, set `allowInInput: true` — and double-check it doesn't capture keys the user is trying to type.
5. Update this doc.

## Testing

The helper is pure — no DOM access except `window.addEventListener`. A unit test wraps `window.dispatchEvent(new KeyboardEvent(...))` and asserts `handler` ran. See `commsKeyboard.test.ts` (TODO — Wave 3 follow-up if test coverage is requested).

## Display helper

`formatShortcut(spec)` returns a label suitable for tooltips and help dialogs. Renders `Mod+Enter` as `⌘⏎` on macOS and `Ctrl+Enter` elsewhere.

```ts
import { formatShortcut } from '$lib/components/comms/commsKeyboard';

const label = formatShortcut('Mod+Enter');  // "⌘⏎" on Mac, "Ctrl+Enter" elsewhere
```
