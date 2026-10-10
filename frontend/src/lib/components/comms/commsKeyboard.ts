// Shared keyboard-shortcut helper for the comms module.
// Both email and channels tabs use this to wire Gmail/Slack-style hotkeys
// without duplicating dispatch logic. Bind in a Svelte 5 component via
// $effect(() => bindShortcuts([...])).

export interface ShortcutBinding {
  // Key spec — case-insensitive. Examples:
  //   "j", "k", "Enter", "Escape", "/", "?"
  //   "Mod+Enter"  → Cmd on macOS, Ctrl elsewhere
  //   "Shift+u"
  //   "Mod+Shift+a"
  key: string;
  handler: (event: KeyboardEvent) => void;
  // Optional human description for the help dialog / docs.
  description?: string;
  // Whether the binding fires while an input/textarea/contenteditable has focus.
  // Default: false. Compose's Mod+Enter is the obvious exception.
  allowInInput?: boolean;
  // If true, calls preventDefault on match. Default: true.
  preventDefault?: boolean;
}

// Detect mac for Mod resolution. Falls back to navigator.platform when
// userAgentData is unavailable. Server-side renders return false.
function isMac(): boolean {
  if (typeof navigator === "undefined") return false;
  const ua = (navigator as { userAgentData?: { platform?: string } })
    .userAgentData?.platform;
  if (ua) return /mac/i.test(ua);
  return /Mac|iPhone|iPad|iPod/.test(navigator.platform || "");
}

interface ParsedSpec {
  key: string;
  shift: boolean;
  alt: boolean;
  meta: boolean;
  ctrl: boolean;
}

function parseSpec(spec: string, mac: boolean): ParsedSpec {
  const parts = spec.split("+").map((p) => p.trim());
  const key = parts.pop() ?? "";
  const flags = { shift: false, alt: false, meta: false, ctrl: false };
  for (const mod of parts) {
    const lower = mod.toLowerCase();
    if (lower === "shift") flags.shift = true;
    else if (lower === "alt" || lower === "option") flags.alt = true;
    else if (lower === "meta" || lower === "cmd") flags.meta = true;
    else if (lower === "ctrl" || lower === "control") flags.ctrl = true;
    else if (lower === "mod") {
      if (mac) flags.meta = true;
      else flags.ctrl = true;
    }
  }
  return { key: normalizeKey(key), ...flags };
}

// Browser KeyboardEvent.key uses "ArrowDown", "Escape", " " for space etc.
// Single-char keys are returned as-is (lowercase for letter normalization).
function normalizeKey(k: string): string {
  if (!k) return "";
  if (k.length === 1) return k.toLowerCase();
  return k;
}

// Letter keys discriminate on Shift (`u` vs `Shift+u` are different actions).
// Digits and named keys do too. Symbol keys like `#`, `?`, `/` already carry
// the shift in event.key on US layouts, so we don't double-check it.
function isShiftSensitive(specKey: string): boolean {
  if (specKey.length !== 1) return true;
  return /[a-z0-9]/i.test(specKey);
}

function eventMatches(event: KeyboardEvent, spec: ParsedSpec): boolean {
  if (normalizeKey(event.key) !== spec.key) return false;
  if (event.altKey !== spec.alt) return false;
  if (event.metaKey !== spec.meta) return false;
  if (event.ctrlKey !== spec.ctrl) return false;
  if (isShiftSensitive(spec.key) && event.shiftKey !== spec.shift) return false;
  return true;
}

// Returns true when the user is actively typing into an editable surface.
// We skip shortcuts in this case unless the binding opts in via allowInInput.
function isTypingInEditable(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  const tag = target.tagName;
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return true;
  if (target.isContentEditable) return true;
  return false;
}

// Bind a set of shortcuts to window keydown. Returns a teardown function for
// $effect's cleanup return.
//
// First match wins — earlier bindings take precedence over later ones if both
// would fire on the same event.
export function bindShortcuts(bindings: ShortcutBinding[]): () => void {
  if (typeof window === "undefined") return () => {};
  const mac = isMac();
  const compiled = bindings.map((b) => ({ binding: b, spec: parseSpec(b.key, mac) }));

  function onKeydown(event: KeyboardEvent) {
    // Modifier-only presses report key like "Shift" — let those through unmatched.
    if (event.defaultPrevented) return;
    const typing = isTypingInEditable(event.target);
    for (const { binding, spec } of compiled) {
      if (typing && !binding.allowInInput) continue;
      if (!eventMatches(event, spec)) continue;
      if (binding.preventDefault !== false) event.preventDefault();
      binding.handler(event);
      return;
    }
  }

  window.addEventListener("keydown", onKeydown);
  return () => window.removeEventListener("keydown", onKeydown);
}

// Convenience: format a key spec for display in tooltips / help dialogs.
// "Mod+Enter" → "⌘+Enter" on macOS, "Ctrl+Enter" elsewhere.
export function formatShortcut(spec: string, mac = isMac()): string {
  return spec
    .split("+")
    .map((p) => {
      const lower = p.trim().toLowerCase();
      if (lower === "mod") return mac ? "⌘" : "Ctrl";
      if (lower === "shift") return mac ? "⇧" : "Shift";
      if (lower === "alt" || lower === "option") return mac ? "⌥" : "Alt";
      if (lower === "meta" || lower === "cmd") return "⌘";
      if (lower === "ctrl" || lower === "control") return "Ctrl";
      if (lower === "enter") return mac ? "⏎" : "Enter";
      if (lower === "escape" || lower === "esc") return "Esc";
      return p.length === 1 ? p.toUpperCase() : p;
    })
    .join(mac ? "" : "+");
}
