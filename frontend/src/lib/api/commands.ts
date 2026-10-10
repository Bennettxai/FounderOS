/**
 * commands.ts
 * Shared service for fetching slash commands from the backend.
 *
 * Single source of truth for all slash-command consumers:
 *   - Chat page (chatAgentStore)
 *   - Spotlight search
 *   - Terminal AI chat
 *
 * The result is cached after the first successful fetch so that opening
 * multiple consumers in the same session does not trigger redundant requests.
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/**
 * Canonical SlashCommand shape returned by GET /api/ai/commands.
 * Matches the existing definition in $lib/stores/chat/types.ts and is the
 * authoritative type for all consumers.
 */
export interface SlashCommand {
  name: string;
  display_name: string;
  description: string;
  icon: string;
  category: string;
}

// ---------------------------------------------------------------------------
// Cache
// ---------------------------------------------------------------------------

let cachedCommands: SlashCommand[] | null = null;
let inFlightPromise: Promise<SlashCommand[]> | null = null;

// ---------------------------------------------------------------------------
// Fetch
// ---------------------------------------------------------------------------

/**
 * Fetch slash commands from the backend and return the list.
 *
 * - On first call: issues a real HTTP request and caches the result.
 * - On subsequent calls: returns the cached list immediately (no network).
 * - Concurrent calls while the first is in flight share the same Promise.
 * - On error: logs a warning and returns an empty array (no throw).
 *
 * Pass `force = true` to bypass the cache and re-fetch from the API.
 */
export async function fetchSlashCommands(
  force = false,
): Promise<SlashCommand[]> {
  if (!force && cachedCommands !== null) {
    return cachedCommands;
  }

  if (!force && inFlightPromise !== null) {
    return inFlightPromise;
  }

  inFlightPromise = (async (): Promise<SlashCommand[]> => {
    try {
      const response = await fetch("/api/ai/commands");
      if (response.ok) {
        const data = await response.json();
        cachedCommands = (data.commands as SlashCommand[]) ?? [];
        return cachedCommands;
      }
      console.warn("[commands] GET /api/ai/commands returned", response.status);
    } catch (e) {
      console.warn("[commands] Failed to fetch slash commands:", e);
    }
    // Do not cache a failure so the next call retries.
    return [];
  })();

  const result = await inFlightPromise;
  inFlightPromise = null;
  return result;
}

/**
 * Minimal fallback set shown while the async fetch is in flight.
 * Keeps the spotlight and terminal usable before the API responds.
 */
export const FALLBACK_COMMANDS: SlashCommand[] = [
  {
    name: "analyze",
    display_name: "Analyze",
    description: "Analyze content or data",
    icon: "📊",
    category: "general",
  },
  {
    name: "summarize",
    display_name: "Summarize",
    description: "Summarize text or document",
    icon: "📝",
    category: "general",
  },
  {
    name: "explain",
    display_name: "Explain",
    description: "Explain code or concept",
    icon: "💡",
    category: "general",
  },
  {
    name: "generate",
    display_name: "Generate",
    description: "Generate content or code",
    icon: "✨",
    category: "general",
  },
  {
    name: "task",
    display_name: "Task",
    description: "Create a new task",
    icon: "✅",
    category: "general",
  },
];
