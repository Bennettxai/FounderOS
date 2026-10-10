/**
 * Voice Command Navigation Pattern Matchers
 *
 * Handles next/previous window navigation patterns.
 * Extracted from voiceCommandPatterns.ts due to large pattern arrays.
 */

import type { VoiceCommand } from "./voiceCommandTypes";
import { matchesPattern } from "./voiceCommandPatterns";

/**
 * Parse window navigation commands (next/previous)
 */
export function parseNavigationCommand(text: string): VoiceCommand | null {
  if (
    matchesPattern(text, [
      "next window",
      "next one",
      "next page",
      "next thing",
      "next module",
      "next app",
      "go to next",
      "go to the next",
      "go to next window",
      "go to the next window",
      "go to next page",
      "go to the next page",
      "go to next one",
      "go to the next one",
      "move to next",
      "move to the next",
      "move to next window",
      "move to next page",
      "switch to next",
      "switch to the next",
      "switch to next window",
      "switch to next page",
      "next",
      "forward",
      "go forward",
      "move forward",
      "right",
    ])
  ) {
    return { type: "next_window" };
  }

  if (
    matchesPattern(text, [
      "previous window",
      "previous one",
      "previous page",
      "previous thing",
      "previous module",
      "previous app",
      "last window",
      "last one",
      "last page",
      "go to previous",
      "go to the previous",
      "go to previous window",
      "go to the previous window",
      "go to previous page",
      "go to the previous page",
      "go to previous one",
      "go to the previous one",
      "go to last",
      "go to the last",
      "go to last window",
      "go to last page",
      "move to previous",
      "move to the previous",
      "move to previous window",
      "move to previous page",
      "move to last",
      "move to last window",
      "switch to previous",
      "switch to the previous",
      "switch to previous window",
      "switch to previous page",
      "switch to last",
      "go back",
      "back",
      "previous",
      "go backward",
      "move back",
      "left",
    ])
  ) {
    return { type: "previous_window" };
  }

  return null;
}
