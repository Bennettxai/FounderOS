/**
 * Chat Agent Store
 * Manages slash-command autocomplete, @agent mention autocomplete, custom agent
 * selection, focus mode state, and memory injection for the chat page.
 *
 * Follows the singleton factory pattern used by chatUIStore.svelte.ts and
 * cameraStore.svelte.ts. Uses Svelte 5 $state runes for fine-grained reactivity.
 *
 * All state that must be written from outside this module (page + child
 * components) exposes both a getter and a setter. Method-only mutations are
 * getter-only.
 */

import { api, getCustomAgents } from "$lib/api";
import type { CustomAgent } from "$lib/api";
import type { Memory } from "$lib/api/memory/types";
import { handleApiCall } from "$lib/utils/api-handler";
import type { SlashCommand, AgentPreset } from "./types";

// ── Return type for parseInputForSuggestions ────────────────────────────────

export interface ParseSuggestionsResult {
  /** True when the input is a fully-resolved @agent mention (e.g. "@osa "). */
  agentMentionComplete: boolean;
  /** True when the input is a fully-resolved /command (e.g. "/analyze "). */
  commandComplete: boolean;
}

// ── Factory ──────────────────────────────────────────────────────────────────

function createChatAgentStore() {
  // ── Slash commands ──────────────────────────────────────────────────────

  let availableCommands = $state<SlashCommand[]>([]);
  let showCommandSuggestions = $state(false);
  let filteredCommands = $state<SlashCommand[]>([]);
  let commandDropdownIndex = $state(0);
  let activeCommand = $state<SlashCommand | null>(null);

  // ── Agent @mentions ─────────────────────────────────────────────────────

  let availableAgents = $state<AgentPreset[]>([]);
  let detectedAgent = $state<AgentPreset | null>(null);
  let showAgentSuggestions = $state(false);
  let filteredAgents = $state<AgentPreset[]>([]);
  let agentDropdownIndex = $state(0);

  // ── Custom agent selector ───────────────────────────────────────────────

  let customAgents = $state<CustomAgent[]>([]);
  let selectedAgent = $state<CustomAgent | null>(null);
  let loadingAgents = $state(false);

  // ── Focus mode ──────────────────────────────────────────────────────────

  let focusModeEnabled = $state(true);
  let selectedFocusId = $state<string | null>(null);
  let focusOptions = $state<Record<string, string>>({});
  let showFocusDropdown = $state(false);
  let focusModeInitialInput = $state("");

  // ── Memory injection ────────────────────────────────────────────────────

  let selectedMemoryIds = $state<string[]>([]);
  let activeMemories = $state<Memory[]>([]);

  // ── Async loaders ───────────────────────────────────────────────────────

  /**
   * Fetch available slash commands from the backend and populate
   * availableCommands. Safe to call multiple times; each call replaces the
   * previous list.
   */
  async function loadCommands(): Promise<void> {
    try {
      const response = await fetch("/api/ai/commands");
      if (response.ok) {
        const data = await response.json();
        availableCommands = data.commands ?? [];
      }
    } catch (e) {
      console.error("[ChatAgentStore] Failed to load slash commands:", e);
    }
  }

  /**
   * Fetch built-in agent presets for @mention autocomplete.
   * Must be called before loadCustomAgents so the custom-agent merge step has
   * a stable baseline of built-in agents.
   */
  async function loadAgentPresets(): Promise<void> {
    try {
      const response = await fetch("/api/ai/agents/presets");
      if (response.ok) {
        const data = await response.json();
        availableAgents = data.presets ?? [];
      }
    } catch (e) {
      console.error("[ChatAgentStore] Failed to load agent presets:", e);
    }
  }

  /**
   * Fetch the user's custom agents, populate customAgents, and merge them
   * into availableAgents for @mention autocomplete.
   *
   * Merge strategy: keep all built-in agents (category !== 'custom'), then
   * append the fresh custom-agent list. This prevents duplicates when called
   * more than once (e.g. after the user creates a new agent).
   */
  async function loadCustomAgents(): Promise<void> {
    loadingAgents = true;
    try {
      const response = await getCustomAgents(false); // active agents only
      customAgents = response.agents ?? [];

      const customAgentPresets: AgentPreset[] = customAgents.map((agent) => ({
        id: agent.id,
        name: agent.name,
        display_name: agent.display_name,
        description: agent.description ?? null,
        avatar: agent.avatar ?? null,
        category: agent.category ?? "custom",
      }));

      // Preserve built-in agents; replace the custom-agent slice.
      const builtInAgents = availableAgents.filter(
        (a) => a.category !== "custom",
      );
      availableAgents = [...builtInAgents, ...customAgentPresets];
    } catch (e) {
      console.error("[ChatAgentStore] Failed to load custom agents:", e);
      customAgents = [];
    } finally {
      loadingAgents = false;
    }
  }

  // ── Memory ──────────────────────────────────────────────────────────────

  /**
   * Update the selected memory ID list and fetch the corresponding full
   * memory objects for context injection.
   *
   * Each memory is fetched individually (the list endpoint does not support
   * filtering by ID array). Failures for individual IDs are silently skipped
   * so a single bad ID does not abort the entire batch.
   */
  async function handleMemoriesSelected(memoryIds: string[]): Promise<void> {
    selectedMemoryIds = memoryIds;

    if (memoryIds.length === 0) {
      activeMemories = [];
      return;
    }

    const { data, error } = await handleApiCall(
      async () => {
        const results = await Promise.all(
          memoryIds.map((id) => api.getMemory(id).catch(() => null)),
        );
        return results.filter((m): m is NonNullable<typeof m> => m !== null);
      },
      {
        showErrorToast: true,
        errorMessage: "Failed to fetch memory objects",
      },
    );

    activeMemories = data ?? [];
  }

  // ── Input suggestion parsing ─────────────────────────────────────────────

  /**
   * Analyse the current raw input value and update all autocomplete state:
   * filtered lists, visibility flags, and dropdown cursor positions.
   *
   * Returns a result object that tells the page whether a complete @mention or
   * /command has been resolved — so the page can take further action (e.g.
   * clear the input prefix, move focus) without duplicating this logic.
   *
   * Side-effects: mutates showAgentSuggestions, filteredAgents,
   * agentDropdownIndex, detectedAgent, showCommandSuggestions,
   * filteredCommands, commandDropdownIndex, activeCommand.
   */
  function parseInputForSuggestions(
    inputValue: string,
  ): ParseSuggestionsResult {
    if (inputValue.startsWith("@")) {
      // ── @agent branch ──────────────────────────────────────────────────
      const spaceIndex = inputValue.indexOf(" ");

      if (spaceIndex > 0) {
        // User typed a full agent name followed by a space — try to resolve.
        const agentName = inputValue.slice(1, spaceIndex);
        const matchedAgent = availableAgents.find(
          (a) => a.name.toLowerCase() === agentName.toLowerCase(),
        );
        if (matchedAgent) {
          detectedAgent = matchedAgent;
          showAgentSuggestions = false;
          showCommandSuggestions = false;
          return { agentMentionComplete: true, commandComplete: false };
        }
      }

      // Still typing — show live suggestions.
      const query = inputValue.slice(1).toLowerCase().split(" ")[0];

      if (query.length === 0) {
        filteredAgents = availableAgents.slice(0, 8);
      } else {
        filteredAgents = availableAgents
          .filter(
            (a) =>
              a.name.toLowerCase().includes(query) ||
              a.display_name.toLowerCase().includes(query),
          )
          .slice(0, 8);
      }

      showAgentSuggestions = filteredAgents.length > 0;
      agentDropdownIndex = 0;
      showCommandSuggestions = false;

      return { agentMentionComplete: false, commandComplete: false };
    }

    if (inputValue.startsWith("/")) {
      // ── /command branch ───────────────────────────────────────────────
      const spaceIndex = inputValue.indexOf(" ");

      if (spaceIndex > 0) {
        // User typed a full command name followed by a space — try to resolve.
        const cmdName = inputValue.slice(1, spaceIndex);
        const matchedCmd = availableCommands.find((c) => c.name === cmdName);
        if (matchedCmd) {
          activeCommand = matchedCmd;
          showCommandSuggestions = false;
          showAgentSuggestions = false;
          return { agentMentionComplete: false, commandComplete: true };
        }
      }

      // Still typing — show live suggestions.
      const query = inputValue.slice(1).toLowerCase().split(" ")[0];

      if (query.length === 0) {
        filteredCommands = availableCommands.slice(0, 8);
      } else {
        filteredCommands = availableCommands
          .filter(
            (c) =>
              c.name.includes(query) ||
              c.display_name.toLowerCase().includes(query),
          )
          .slice(0, 8);
      }

      showCommandSuggestions = filteredCommands.length > 0;
      commandDropdownIndex = 0;
      showAgentSuggestions = false;

      return { agentMentionComplete: false, commandComplete: false };
    }

    // ── Neither @ nor / prefix ────────────────────────────────────────────
    showCommandSuggestions = false;
    showAgentSuggestions = false;
    activeCommand = null;
    detectedAgent = null;

    return { agentMentionComplete: false, commandComplete: false };
  }

  // ── Dropdown selection helpers ───────────────────────────────────────────

  /**
   * Commit a command selection from the dropdown.
   * Sets activeCommand, hides suggestions, and returns the input string the
   * page should place in the textarea (e.g. "/analyze ").
   */
  function selectCommand(cmd: SlashCommand): string {
    activeCommand = cmd;
    showCommandSuggestions = false;
    return "/" + cmd.name + " ";
  }

  /** Clear the active command and reset related state. */
  function clearActiveCommand(): void {
    activeCommand = null;
  }

  /**
   * Commit an agent selection from the dropdown.
   * Sets detectedAgent, hides suggestions, and returns the input string the
   * page should place in the textarea (e.g. "@osa ").
   */
  function selectAgent(agent: AgentPreset): string {
    detectedAgent = agent;
    showAgentSuggestions = false;
    return "@" + agent.name + " ";
  }

  /** Clear the detected agent and reset related state. */
  function clearDetectedAgent(): void {
    detectedAgent = null;
  }

  // ── Agent selector ───────────────────────────────────────────────────────

  /** Set the active custom agent (null means "use default"). */
  function handleAgentSelect(agent: CustomAgent | null): void {
    selectedAgent = agent;
  }

  /**
   * Resolve a fully-typed @agent mention from free text.
   * Matches the pattern `^@(\w[\w-]*)` against the provided text and sets
   * detectedAgent if a matching preset is found. Clears detectedAgent otherwise.
   * Used for detecting mentions that were pasted or typed without the
   * autocomplete dropdown.
   */
  function detectAgentMention(text: string): void {
    const agentMatch = text.match(/^@(\w[\w-]*)/);
    if (agentMatch && availableAgents.length > 0) {
      const agentName = agentMatch[1].toLowerCase();
      const agent = availableAgents.find(
        (a) => a.name.toLowerCase() === agentName,
      );
      if (agent) {
        detectedAgent = agent;
        return;
      }
    }
    detectedAgent = null;
  }

  // ── Public interface ─────────────────────────────────────────────────────

  return {
    // ── Slash commands ──────────────────────────────────────────────────

    get availableCommands() {
      return availableCommands;
    },
    set availableCommands(v: SlashCommand[]) {
      availableCommands = v;
    },

    get showCommandSuggestions() {
      return showCommandSuggestions;
    },
    set showCommandSuggestions(v: boolean) {
      showCommandSuggestions = v;
    },

    get filteredCommands() {
      return filteredCommands;
    },
    set filteredCommands(v: SlashCommand[]) {
      filteredCommands = v;
    },

    get commandDropdownIndex() {
      return commandDropdownIndex;
    },
    set commandDropdownIndex(v: number) {
      commandDropdownIndex = v;
    },

    get activeCommand() {
      return activeCommand;
    },
    set activeCommand(v: SlashCommand | null) {
      activeCommand = v;
    },

    // ── Agent @mentions ─────────────────────────────────────────────────

    get availableAgents() {
      return availableAgents;
    },
    set availableAgents(v: AgentPreset[]) {
      availableAgents = v;
    },

    get detectedAgent() {
      return detectedAgent;
    },
    set detectedAgent(v: AgentPreset | null) {
      detectedAgent = v;
    },

    get showAgentSuggestions() {
      return showAgentSuggestions;
    },
    set showAgentSuggestions(v: boolean) {
      showAgentSuggestions = v;
    },

    get filteredAgents() {
      return filteredAgents;
    },
    set filteredAgents(v: AgentPreset[]) {
      filteredAgents = v;
    },

    get agentDropdownIndex() {
      return agentDropdownIndex;
    },
    set agentDropdownIndex(v: number) {
      agentDropdownIndex = v;
    },

    // ── Custom agent selector ───────────────────────────────────────────

    get customAgents() {
      return customAgents;
    },
    set customAgents(v: CustomAgent[]) {
      customAgents = v;
    },

    get selectedAgent() {
      return selectedAgent;
    },
    set selectedAgent(v: CustomAgent | null) {
      selectedAgent = v;
    },

    get loadingAgents() {
      return loadingAgents;
    },

    // ── Focus mode ──────────────────────────────────────────────────────

    get focusModeEnabled() {
      return focusModeEnabled;
    },
    set focusModeEnabled(v: boolean) {
      focusModeEnabled = v;
    },

    get selectedFocusId() {
      return selectedFocusId;
    },
    set selectedFocusId(v: string | null) {
      selectedFocusId = v;
    },

    get focusOptions() {
      return focusOptions;
    },
    set focusOptions(v: Record<string, string>) {
      focusOptions = v;
    },

    get showFocusDropdown() {
      return showFocusDropdown;
    },
    set showFocusDropdown(v: boolean) {
      showFocusDropdown = v;
    },

    get focusModeInitialInput() {
      return focusModeInitialInput;
    },
    set focusModeInitialInput(v: string) {
      focusModeInitialInput = v;
    },

    // ── Memory injection ────────────────────────────────────────────────

    get selectedMemoryIds() {
      return selectedMemoryIds;
    },
    set selectedMemoryIds(v: string[]) {
      selectedMemoryIds = v;
    },

    get activeMemories() {
      return activeMemories;
    },
    set activeMemories(v: Memory[]) {
      activeMemories = v;
    },

    // ── Methods ─────────────────────────────────────────────────────────

    loadCommands,
    loadAgentPresets,
    loadCustomAgents,
    handleAgentSelect,
    detectAgentMention,
    selectCommand,
    clearActiveCommand,
    selectAgent,
    clearDetectedAgent,
    handleMemoriesSelected,
    parseInputForSuggestions,
  };
}

export const chatAgentStore = createChatAgentStore();
