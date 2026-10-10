/**
 * useKeyboardShortcuts
 *
 * Svelte 5 runes hook that returns the `handleKeydown` handler
 * for the Desktop3D global keyboard shortcuts.
 *
 * Attach the returned function to `<svelte:window onkeydown={...} />`.
 */

import { desktop3dStore, openWindows } from "$lib/stores/desktop3dStore";
import { get } from "svelte/store";

interface UseKeyboardShortcutsOptions {
  /** Called when Escape is pressed and no window is focused */
  onExit?: () => void;
}

interface UseKeyboardShortcutsReturn {
  handleKeydown: (e: KeyboardEvent) => void;
}

export function useKeyboardShortcuts(
  opts: UseKeyboardShortcutsOptions,
): UseKeyboardShortcutsReturn {
  function handleKeydown(e: KeyboardEvent): void {
    const target = e.target as HTMLElement;
    const activeEl = document.activeElement;

    const isInteractiveElement =
      target?.tagName === "INPUT" ||
      target?.tagName === "TEXTAREA" ||
      target?.isContentEditable ||
      target?.closest("iframe") !== null ||
      activeEl?.tagName === "IFRAME";

    // Escape — unfocus or exit (ALWAYS allow, even in terminal)
    if (e.key === "Escape") {
      e.preventDefault();
      const state = get(desktop3dStore);
      if (state.focusedWindowId) {
        desktop3dStore.unfocusWindow();
      } else {
        opts.onExit?.();
      }
      return;
    }

    // Don't handle any other shortcuts when user is in terminal/inputs
    if (isInteractiveElement) {
      return;
    }

    const state = get(desktop3dStore);

    // Space — toggle view mode (only when not focused)
    if (e.key === " " && !state.focusedWindowId) {
      e.preventDefault();
      desktop3dStore.toggleViewMode();
    }

    // Arrow keys / resize keys — only when a window is focused
    if (state.focusedWindowId) {
      if (e.key === "ArrowRight") {
        e.preventDefault();
        desktop3dStore.focusNext();
      } else if (e.key === "ArrowLeft") {
        e.preventDefault();
        desktop3dStore.focusPrevious();
      }

      if (e.key === "+" || e.key === "=") {
        e.preventDefault();
        desktop3dStore.resizeFocusedWindow(100, 75);
      } else if (e.key === "-") {
        e.preventDefault();
        desktop3dStore.resizeFocusedWindow(-100, -75);
      }
    }

    // Number keys 1-9 — focus window by index (only when nothing is focused)
    if (e.key >= "1" && e.key <= "9" && !state.focusedWindowId) {
      const index = parseInt(e.key) - 1;
      const windows = get(openWindows);
      if (windows[index]) {
        desktop3dStore.focusWindow(windows[index].id);
      }
    }
  }

  return { handleKeydown };
}
