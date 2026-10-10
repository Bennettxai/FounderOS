/**
 * spotlightSearch.ts
 * Search data, filtering logic, and API loaders for SpotlightSearch.
 */

import { apiClient } from "$lib/api";
import {
  fetchSlashCommands,
  FALLBACK_COMMANDS,
  type SlashCommand as ApiSlashCommand,
} from "$lib/api/commands";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface SearchItem {
  id: string;
  type: "app";
  name: string;
  description: string;
  icon: string;
  color: string;
}

/**
 * SlashCommand shape used by Spotlight UI components.
 * Re-exported from the shared API type so all consumers share one definition.
 * The `id` field is derived from `name` for backward compatibility with
 * SpotlightCommandsDropdown which keys each row on `cmd.id`.
 */
export interface SlashCommand extends ApiSlashCommand {
  /** Derived from `name` for keying in templates. */
  id: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
}

export interface AIModel {
  id: string;
  name: string;
  provider: string;
  size?: string;
}

// ---------------------------------------------------------------------------
// Static data
// ---------------------------------------------------------------------------

export const SEARCH_ITEMS: SearchItem[] = [
  {
    id: "dashboard",
    type: "app",
    name: "Dashboard",
    description: "Overview and analytics",
    icon: "M4 5a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v2a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zm0 6a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1h-4a1 1 0 01-1-1v-5zm-10 1a1 1 0 011-1h4a1 1 0 011 1v3a1 1 0 01-1 1H5a1 1 0 01-1-1v-3z",
    color: "#1E88E5",
  },
  {
    id: "chat",
    type: "app",
    name: "Chat",
    description: "AI Assistant",
    icon: "M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z",
    color: "#43A047",
  },
  {
    id: "tasks",
    type: "app",
    name: "Tasks",
    description: "Task management",
    icon: "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4",
    color: "#FB8C00",
  },
  {
    id: "projects",
    type: "app",
    name: "Projects",
    description: "Project management",
    icon: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z",
    color: "#8E24AA",
  },
  {
    id: "team",
    type: "app",
    name: "Team",
    description: "Team members",
    icon: "M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z",
    color: "#00ACC1",
  },
  {
    id: "clients",
    type: "app",
    name: "Clients",
    description: "Client management",
    icon: "M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4",
    color: "#7B1FA2",
  },
  {
    id: "calendar",
    type: "app",
    name: "Calendar",
    description: "Schedule and events",
    icon: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    color: "#E53935",
  },
  {
    id: "contexts",
    type: "app",
    name: "Contexts",
    description: "Work contexts",
    icon: "M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10",
    color: "#5E35B1",
  },
  {
    id: "nodes",
    type: "app",
    name: "Nodes",
    description: "Node management",
    icon: "M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z",
    color: "#E53935",
  },
  {
    id: "settings",
    type: "app",
    name: "Settings",
    description: "System settings",
    icon: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
    color: "#546E7A",
  },
  {
    id: "terminal",
    type: "app",
    name: "Terminal",
    description: "OS Agent terminal",
    icon: "M4 17l6-6-6-6M12 19h8",
    color: "#00FF00",
  },
  {
    id: "files",
    type: "app",
    name: "Files",
    description: "File browser",
    icon: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z",
    color: "#3B82F6",
  },
];

// ---------------------------------------------------------------------------
// Dynamic slash commands (loaded from /api/ai/commands)
// ---------------------------------------------------------------------------

/**
 * Module-level cache of loaded commands, normalised to the Spotlight shape.
 * Starts as the fallback set so the dropdown works immediately on first open.
 */
let _loadedCommands: SlashCommand[] = normaliseFallback();

/** Convert the shared API fallback list to the Spotlight SlashCommand shape. */
function normaliseFallback(): SlashCommand[] {
  return FALLBACK_COMMANDS.map((c) => ({ ...c, id: c.name }));
}

/**
 * Normalise a raw API command to the Spotlight SlashCommand shape.
 * The `name` field from the API is the bare command word (e.g. "analyze");
 * we keep that as `id` for template keying and as `name` for filtering.
 */
function normaliseCommand(c: ApiSlashCommand): SlashCommand {
  return { ...c, id: c.name };
}

/**
 * Fetch slash commands from the shared service and update the module cache.
 * Safe to call multiple times; the shared service deduplicates in-flight
 * requests and caches the result after the first successful fetch.
 *
 * Call this once when Spotlight opens (inside the `$effect` that reacts to
 * `open === true`).
 */
export async function initSlashCommands(): Promise<void> {
  const commands = await fetchSlashCommands();
  if (commands.length > 0) {
    _loadedCommands = commands.map(normaliseCommand);
  }
}

/**
 * The currently loaded slash commands. Exported for consumers that need the
 * full list (e.g. to display a static command palette). Prefer using
 * `filterSlashCommands` for input-driven filtering.
 */
export function getLoadedSlashCommands(): SlashCommand[] {
  return _loadedCommands;
}

// ---------------------------------------------------------------------------
// Filtering helpers
// ---------------------------------------------------------------------------

export function filterSearchItems(
  query: string,
  mode: "search" | "chat",
): SearchItem[] {
  if (mode === "chat") return [];
  if (!query.trim()) return SEARCH_ITEMS.slice(0, 6);
  const q = query.toLowerCase();
  return SEARCH_ITEMS.filter(
    (item) =>
      item.name.toLowerCase().includes(q) ||
      item.description.toLowerCase().includes(q),
  );
}

export function filterSlashCommands(input: string): SlashCommand[] {
  if (!input.startsWith("/")) return [];
  const query = input.slice(1).toLowerCase();
  if (query.length === 0) return _loadedCommands.slice(0, 8);
  return _loadedCommands.filter(
    (cmd) =>
      cmd.id.includes(query) ||
      cmd.display_name.toLowerCase().includes(query) ||
      cmd.description.toLowerCase().includes(query),
  );
}

// ---------------------------------------------------------------------------
// API loaders
// ---------------------------------------------------------------------------

export async function loadProjects(): Promise<Project[]> {
  try {
    const response = await apiClient.get("/projects");
    if (response.ok) {
      const data = await response.json();
      return data.projects || data || [];
    }
  } catch (e) {
    console.error("Failed to load projects:", e);
  }
  return [];
}

export interface ModelsResult {
  models: AIModel[];
  activeProvider: string;
  defaultModel: string;
}

export async function loadModels(): Promise<ModelsResult> {
  let activeProvider = "ollama_local";
  let defaultModel = "";
  let models: AIModel[] = [];

  try {
    const providersRes = await apiClient.get("/ai/providers");
    if (providersRes.ok) {
      const data = await providersRes.json();
      activeProvider = data.active_provider || "ollama_local";
      defaultModel = data.default_model || "";
    }

    const modelsRes = await apiClient.get("/ai/models");
    if (modelsRes.ok) {
      const data = await modelsRes.json();
      models = data.models || [];
    }
  } catch (e) {
    console.error("Failed to load models:", e);
  }

  return { models, activeProvider, defaultModel };
}
