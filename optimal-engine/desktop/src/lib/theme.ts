/**
 * Light/dark theme toggle persisted to localStorage.
 * Applies via `data-theme="light|dark"` on <html>; CSS variables in
 * app.css pick the appropriate palette.
 */

import { writable } from "svelte/store";

export type Theme = "light" | "dark";

const STORAGE_KEY = "oe-theme";

function initial(): Theme {
  if (typeof window === "undefined") return "dark";
  const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
  if (saved === "light" || saved === "dark") return saved;
  // Follow OS preference on first load.
  return window.matchMedia("(prefers-color-scheme: light)").matches
    ? "light"
    : "dark";
}

export const theme = writable<Theme>(initial());

export function apply(t: Theme) {
  if (typeof document === "undefined") return;
  document.documentElement.dataset.theme = t;
  localStorage.setItem(STORAGE_KEY, t);
}

export function toggle() {
  theme.update((t) => {
    const next: Theme = t === "dark" ? "light" : "dark";
    apply(next);
    return next;
  });
}

// Apply the initial theme immediately on client.
if (typeof window !== "undefined") {
  theme.subscribe(apply);
}
