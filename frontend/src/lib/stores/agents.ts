import { writable, derived } from "svelte/store";
import {
  getCustomAgents,
  getCustomAgent,
  createCustomAgent,
  updateCustomAgent,
  deleteCustomAgent,
  getAgentsByCategory,
  getAgentPresets,
  getAgentPreset,
  createFromPreset,
  testAgent,
  testSandbox,
} from "$lib/api/ai/ai";
import type { CustomAgent, AgentPreset } from "$lib/api/ai/types";

interface AgentFilters {
  category: string | null;
  search: string;
  status: "active" | "inactive" | null;
}

interface AgentsState {
  agents: CustomAgent[];
  currentAgent: CustomAgent | null;
  presets: AgentPreset[];
  loading: boolean;
  error: string | null;
  filters: AgentFilters;
}

function readResponseArray<T>(response: unknown, key: string): T[] {
  if (Array.isArray(response)) {
    return response as T[];
  }

  if (response && typeof response === "object") {
    const value = (response as Record<string, unknown>)[key];
    return Array.isArray(value) ? (value as T[]) : [];
  }

  return [];
}

function createAgentsStore() {
  const { subscribe, update } = writable<AgentsState>({
    agents: [],
    currentAgent: null,
    presets: [],
    loading: false,
    error: null,
    filters: {
      category: null,
      search: "",
      status: null,
    },
  });

  // Request versioning to prevent race conditions
  let loadRequestId = 0;

  return {
    subscribe,

    // ============ Agent Methods ============

    async loadAgents(filters?: Partial<AgentFilters>) {
      // Increment request ID to track latest request
      const thisRequestId = ++loadRequestId;
      update((s) => ({ ...s, loading: true, error: null }));

      // Apply filters immediately if provided
      if (filters) {
        update((s) => ({
          ...s,
          filters: { ...s.filters, ...filters },
        }));
      }

      // Snapshot current filters
      let currentFilters: AgentFilters = {
        category: null,
        search: "",
        status: null,
      };
      update((s) => {
        currentFilters = { ...s.filters };
        return s;
      });

      try {
        const includeInactive =
          currentFilters.status === null ||
          currentFilters.status === "inactive";

        const fetchPromise = getCustomAgents(includeInactive);
        const timeoutPromise = new Promise<never>((_, reject) =>
          setTimeout(() => reject(new Error("Request timed out")), 6000),
        );

        const response = await Promise.race([fetchPromise, timeoutPromise]);

        // Safely access agents array — backend may return different shapes
        let agents = readResponseArray<CustomAgent>(response, "agents");

        // Apply client-side filters
        if (currentFilters.category) {
          agents = agents.filter((a) => a.category === currentFilters.category);
        }

        if (currentFilters.search) {
          const searchLower = currentFilters.search.toLowerCase();
          agents = agents.filter(
            (a) =>
              a.name?.toLowerCase().includes(searchLower) ||
              a.display_name?.toLowerCase().includes(searchLower) ||
              a.description?.toLowerCase().includes(searchLower),
          );
        }

        if (currentFilters.status === "active") {
          agents = agents.filter((a) => a.is_active);
        } else if (currentFilters.status === "inactive") {
          agents = agents.filter((a) => !a.is_active);
        }

        // Only update if this is still the latest request
        if (thisRequestId === loadRequestId) {
          update((s) => ({ ...s, agents, loading: false, error: null }));
        }
      } catch (error) {
        console.error("Failed to load agents:", error);

        // Only update if this is still the latest request
        if (thisRequestId === loadRequestId) {
          update((s) => ({
            ...s,
            loading: false,
            agents: [],
            error:
              error instanceof Error ? error.message : "Failed to load agents",
          }));
        }
      }
    },

    async loadAgent(id: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const agent = await getCustomAgent(id);
        update((s) => ({ ...s, currentAgent: agent, loading: false }));
        return agent;
      } catch (error) {
        console.error("Failed to load agent:", error);
        update((s) => ({
          ...s,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load agent",
        }));
        return null;
      }
    },

    async createAgent(data: Partial<CustomAgent>) {
      try {
        const agent = await createCustomAgent(data);
        update((s) => ({ ...s, agents: [agent, ...s.agents] }));
        return agent;
      } catch (error) {
        console.error("Failed to create agent:", error);
        throw error;
      }
    },

    async updateAgent(id: string, data: Partial<CustomAgent>) {
      try {
        const agent = await updateCustomAgent(id, data);
        update((s) => ({
          ...s,
          agents: s.agents.map((a) => (a.id === id ? agent : a)),
          currentAgent: s.currentAgent?.id === id ? agent : s.currentAgent,
        }));
        return agent;
      } catch (error) {
        console.error("Failed to update agent:", error);
        throw error;
      }
    },

    async deleteAgent(id: string) {
      try {
        await deleteCustomAgent(id);
        update((s) => ({
          ...s,
          agents: s.agents.filter((a) => a.id !== id),
          currentAgent: s.currentAgent?.id === id ? null : s.currentAgent,
        }));
      } catch (error) {
        console.error("Failed to delete agent:", error);
        throw error;
      }
    },

    // ============ Current Agent Methods ============

    setCurrentAgent(agent: CustomAgent | null) {
      update((s) => ({ ...s, currentAgent: agent }));
    },

    clearCurrent() {
      update((s) => ({ ...s, currentAgent: null }));
    },

    // ============ Filter Methods ============

    setFilters(filters: Partial<AgentFilters>) {
      update((s) => ({
        ...s,
        filters: { ...s.filters, ...filters },
      }));
    },

    clearFilters() {
      update((s) => ({
        ...s,
        filters: {
          category: null,
          search: "",
          status: null,
        },
      }));
    },

    // ============ Error Methods ============

    clearError() {
      update((s) => ({ ...s, error: null }));
    },

    // ============ Test Utilities ============

    reset() {
      update((s) => ({
        agents: [],
        currentAgent: null,
        presets: [],
        loading: false,
        error: null,
        filters: {
          category: null,
          search: "",
          status: null,
        },
      }));
    },

    // ============ Preset Methods ============

    async loadPresets() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const fetchPromise = getAgentPresets();
        const timeoutPromise = new Promise<never>((_, reject) =>
          setTimeout(() => reject(new Error("Presets timeout")), 6000),
        );
        const response = await Promise.race([fetchPromise, timeoutPromise]);
        const presets = readResponseArray<AgentPreset>(response, "presets");
        update((s) => ({ ...s, presets, loading: false }));
      } catch (error) {
        console.error("Failed to load agent presets:", error);
        update((s) => ({
          ...s,
          loading: false,
          presets: [],
          error:
            error instanceof Error
              ? error.message
              : "Failed to load agent presets",
        }));
      }
    },

    async loadPreset(id: string) {
      try {
        return await getAgentPreset(id);
      } catch (error) {
        console.error("Failed to load agent preset:", error);
        throw error;
      }
    },

    async createFromPreset(presetId: string, name?: string) {
      try {
        const agent = await createFromPreset(presetId, name);
        update((s) => ({ ...s, agents: [agent, ...s.agents] }));
        return agent;
      } catch (error) {
        console.error("Failed to create agent from preset:", error);
        throw error;
      }
    },

    // ============ Testing Methods ============

    async testAgent(
      id: string,
      message: string,
    ): Promise<ReadableStream<Uint8Array> | null> {
      try {
        return await testAgent(id, message);
      } catch (error) {
        console.error("Failed to test agent:", error);
        throw error;
      }
    },

    async testSandbox(config: {
      system_prompt: string;
      message: string;
      model?: string;
      temperature?: number;
    }): Promise<ReadableStream<Uint8Array> | null> {
      try {
        // Convert message to test_message for API
        const apiConfig = {
          system_prompt: config.system_prompt,
          test_message: config.message,
          model: config.model,
          temperature: config.temperature,
        };
        return await testSandbox(apiConfig);
      } catch (error) {
        console.error("Failed to test in sandbox:", error);
        throw error;
      }
    },
  };
}

export const agents = createAgentsStore();

// ============ Derived Stores ============

export const selectedAgent = derived(agents, ($agents) => $agents.currentAgent);

export const agentsByCategory = derived(agents, ($agents) => {
  const byCategory: Record<string, CustomAgent[]> = {};

  for (const agent of $agents.agents) {
    const category = agent.category || "uncategorized";
    if (!byCategory[category]) {
      byCategory[category] = [];
    }
    byCategory[category].push(agent);
  }

  return byCategory;
});

export const activeAgents = derived(agents, ($agents) =>
  $agents.agents.filter((a) => a.is_active),
);

// ============ UI Constants ============

export const categoryColors: Record<string, string> = {
  general: "agm-cat agm-cat--general",
  coding: "agm-cat agm-cat--coding",
  writing: "agm-cat agm-cat--writing",
  analysis: "agm-cat agm-cat--analysis",
  research: "agm-cat agm-cat--research",
  support: "agm-cat agm-cat--support",
  sales: "agm-cat agm-cat--sales",
  marketing: "agm-cat agm-cat--marketing",
  uncategorized: "agm-cat",
};

export const categoryLabels: Record<string, string> = {
  general: "General",
  coding: "Coding",
  writing: "Writing",
  analysis: "Analysis",
  research: "Research",
  support: "Support",
  sales: "Sales",
  marketing: "Marketing",
  uncategorized: "Other",
};
