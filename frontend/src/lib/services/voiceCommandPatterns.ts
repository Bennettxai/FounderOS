/**
 * Voice Command Pattern Matchers
 *
 * Each function tries to match a normalized text string to a specific
 * command category. Returns the matched VoiceCommand or null.
 */

import type { VoiceCommand } from "./voiceCommandTypes";
import { VOICE_MODULE_IDS } from "./voiceCommandTypes";

/**
 * Check if text matches any of the given literal patterns (word-boundary aware)
 */
export function matchesPattern(text: string, patterns: string[]): boolean {
  for (const pattern of patterns) {
    const regex = new RegExp(`\\b${escapeRegex(pattern)}\\b`, "i");
    if (regex.test(text)) {
      return true;
    }
  }
  return false;
}

/**
 * Escape special regex characters in a literal string
 */
export function escapeRegex(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * Clean up a layout name extracted from transcript
 */
export function cleanLayoutName(name: string): string {
  return name
    .replace(/\b(called|named|as)\b/gi, "")
    .trim()
    .replace(/\s+/g, " ");
}

/**
 * Parse layout management commands
 */
export function parseLayoutCommand(text: string): VoiceCommand | null {
  if (
    matchesPattern(text, [
      "edit layout",
      "edit mode",
      "start editing",
      "enter edit",
    ])
  ) {
    return { type: "enter_edit_mode" };
  }

  if (
    matchesPattern(text, [
      "exit edit",
      "stop editing",
      "done editing",
      "cancel edit",
      "leave edit",
    ])
  ) {
    return { type: "exit_edit_mode" };
  }

  const saveMatch = text.match(
    /save\s+(?:the\s+)?layout(?:\s+as)?(?:\s+called)?(?:\s+named)?\s+(.+)/i,
  );
  if (saveMatch) {
    const name = cleanLayoutName(saveMatch[1]);
    if (name) return { type: "save_layout", name };
  }

  if (matchesPattern(text, ["save layout", "save this layout"])) {
    return {
      type: "save_layout",
      name: `Layout ${new Date().toLocaleDateString()}`,
    };
  }

  const loadMatch = text.match(
    /(?:load|switch\s+to|open)\s+(?:the\s+)?layout\s+(.+)/i,
  );
  if (loadMatch) {
    const name = cleanLayoutName(loadMatch[1]);
    if (name) return { type: "load_layout", name };
  }

  const deleteMatch = text.match(/delete\s+(?:the\s+)?layout\s+(.+)/i);
  if (deleteMatch) {
    const name = cleanLayoutName(deleteMatch[1]);
    if (name) return { type: "delete_layout", name };
  }

  if (
    matchesPattern(text, [
      "manage layouts",
      "layout manager",
      "show layouts",
      "open layout manager",
    ])
  ) {
    return { type: "open_layout_manager" };
  }

  if (
    matchesPattern(text, [
      "reset layout",
      "default layout",
      "restore default",
      "reset to default",
    ])
  ) {
    return { type: "reset_layout" };
  }

  return null;
}

/**
 * Parse module navigation commands (open/focus/close)
 */
export function parseModuleCommand(text: string): VoiceCommand | null {
  for (const module of VOICE_MODULE_IDS) {
    const moduleName = module.replace(/-/g, " ");
    const openPatterns = [
      `open ${moduleName}`,
      `open up ${moduleName}`,
      `open the ${moduleName}`,
      `show ${moduleName}`,
      `show me ${moduleName}`,
      `show the ${moduleName}`,
      `focus ${moduleName}`,
      `focus on ${moduleName}`,
      `go ${moduleName}`,
      `go to ${moduleName}`,
      `go to the ${moduleName}`,
      `switch to ${moduleName}`,
      `switch to the ${moduleName}`,
      `change to ${moduleName}`,
      `change to the ${moduleName}`,
      `switch me to ${moduleName}`,
      `pull up ${moduleName}`,
      `pull up the ${moduleName}`,
      `bring up ${moduleName}`,
      `bring up the ${moduleName}`,
      `load ${moduleName}`,
      `load the ${moduleName}`,
      `start ${moduleName}`,
      `launch ${moduleName}`,
    ];

    if (matchesPattern(text, openPatterns)) {
      return { type: "focus_module", module };
    }
  }

  for (const module of VOICE_MODULE_IDS) {
    const moduleName = module.replace(/-/g, " ");
    const closePatterns = [
      `close ${moduleName}`,
      `close the ${moduleName}`,
      `hide ${moduleName}`,
      `hide the ${moduleName}`,
      `exit ${moduleName}`,
      `quit ${moduleName}`,
      `shut ${moduleName}`,
      `shut down ${moduleName}`,
      `close down ${moduleName}`,
    ];

    if (matchesPattern(text, closePatterns)) {
      return { type: "close_module", module };
    }
  }

  if (
    matchesPattern(text, [
      "close all",
      "close all windows",
      "close everything",
      "hide all",
      "hide everything",
      "clear all",
      "clear workspace",
      "clear desktop",
    ])
  ) {
    return { type: "close_all_windows" };
  }

  if (
    matchesPattern(text, [
      "minimize",
      "minimize window",
      "minimize this",
      "hide this window",
      "minimize current",
    ])
  ) {
    return { type: "minimize_window" };
  }

  if (
    matchesPattern(text, [
      "maximize",
      "maximize window",
      "maximize this",
      "full screen",
      "fullscreen",
      "maximize current",
      "make full screen",
    ])
  ) {
    return { type: "maximize_window" };
  }

  return null;
}

/**
 * Parse view control commands (orb/grid, rotation, zoom, spacing)
 */
export function parseViewCommand(text: string): VoiceCommand | null {
  if (
    matchesPattern(text, [
      "orb view",
      "switch to orb",
      "sphere view",
      "circular view",
    ])
  ) {
    return { type: "switch_view", view: "orb" };
  }
  if (matchesPattern(text, ["grid view", "switch to grid", "table view"])) {
    return { type: "switch_view", view: "grid" };
  }
  if (
    matchesPattern(text, [
      "toggle auto-rotate",
      "auto rotate",
      "start rotating",
      "toggle rotation",
    ])
  ) {
    return { type: "toggle_auto_rotate" };
  }
  if (
    matchesPattern(text, [
      "rotate left",
      "turn left",
      "spin left",
      "rotate counter clockwise",
      "counterclockwise",
    ])
  ) {
    return { type: "rotate_left" };
  }
  if (
    matchesPattern(text, [
      "rotate right",
      "turn right",
      "spin right",
      "rotate clockwise",
      "clockwise",
    ])
  ) {
    return { type: "rotate_right" };
  }
  if (
    matchesPattern(text, [
      "stop rotating",
      "stop rotation",
      "freeze",
      "pause rotation",
      "halt rotation",
    ])
  ) {
    return { type: "stop_rotation" };
  }
  if (
    matchesPattern(text, [
      "rotate faster",
      "speed up",
      "faster rotation",
      "increase speed",
    ])
  ) {
    return { type: "rotate_faster" };
  }
  if (
    matchesPattern(text, [
      "rotate slower",
      "slow down",
      "slower rotation",
      "decrease speed",
    ])
  ) {
    return { type: "rotate_slower" };
  }
  if (
    matchesPattern(text, [
      "zoom in",
      "closer",
      "move closer",
      "come closer",
      "get closer",
      "bring closer",
    ])
  ) {
    return { type: "zoom_in" };
  }
  if (
    matchesPattern(text, [
      "zoom out",
      "farther",
      "move back",
      "go back",
      "back up",
      "move away",
      "pull back",
    ])
  ) {
    return { type: "zoom_out" };
  }
  if (
    matchesPattern(text, [
      "reset zoom",
      "default zoom",
      "normal zoom",
      "reset view",
      "normal view",
    ])
  ) {
    return { type: "reset_zoom" };
  }
  if (
    matchesPattern(text, [
      "expand",
      "expand orb",
      "expand out",
      "make bigger",
      "bigger",
      "enlarge",
      "spread out",
      "open up",
    ])
  ) {
    return { type: "expand_orb" };
  }
  if (
    matchesPattern(text, [
      "contract",
      "contract orb",
      "unexpand",
      "expand in",
      "expand back",
      "go back",
      "undo expand",
      "make smaller",
      "smaller",
      "shrink",
      "shrink orb",
      "close up",
      "bring together",
    ])
  ) {
    return { type: "contract_orb" };
  }
  if (
    matchesPattern(text, [
      "unfocus",
      "exit focus",
      "back to orb",
      "back to desktop",
      "show all",
    ])
  ) {
    return { type: "unfocus" };
  }
  if (
    matchesPattern(text, [
      "increase spacing",
      "more spacing",
      "spread apart",
      "more space",
      "looser grid",
    ])
  ) {
    return { type: "increase_grid_spacing" };
  }
  if (
    matchesPattern(text, [
      "decrease spacing",
      "less spacing",
      "bring closer",
      "less space",
      "tighter grid",
      "compact grid",
    ])
  ) {
    return { type: "decrease_grid_spacing" };
  }
  if (
    matchesPattern(text, [
      "more columns",
      "increase columns",
      "add columns",
      "more per row",
    ])
  ) {
    return { type: "more_grid_columns" };
  }
  if (
    matchesPattern(text, [
      "less columns",
      "fewer columns",
      "decrease columns",
      "remove columns",
      "less per row",
    ])
  ) {
    return { type: "less_grid_columns" };
  }

  return null;
}

/**
 * Parse window resize commands
 */
export function parseResizeCommand(text: string): VoiceCommand | null {
  if (matchesPattern(text, ["make wider", "expand width", "wider", "widen"])) {
    return { type: "resize_window", direction: "wider" };
  }
  if (
    matchesPattern(text, [
      "make narrower",
      "reduce width",
      "narrower",
      "shrink width",
    ])
  ) {
    return { type: "resize_window", direction: "narrower" };
  }
  if (
    matchesPattern(text, [
      "make taller",
      "expand height",
      "taller",
      "higher",
      "increase height",
    ])
  ) {
    return { type: "resize_window", direction: "taller" };
  }
  if (
    matchesPattern(text, [
      "make shorter",
      "reduce height",
      "shorter",
      "lower",
      "decrease height",
    ])
  ) {
    return { type: "resize_window", direction: "shorter" };
  }
  return null;
}

// Navigation command patterns are in a dedicated file due to large pattern arrays.
// Re-export here to keep voiceCommandPatterns.ts as the single import point.
export { parseNavigationCommand } from "./voiceCommandNavigationPatterns";
