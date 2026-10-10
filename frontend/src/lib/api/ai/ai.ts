import { request, getApiBaseUrl } from "../base";
import type {
  AIProvidersResponse,
  AllModelsResponse,
  LocalModelsResponse,
  AISystemInfo,
  AgentInfo,
  WarmupResponse,
  Tool,
  ToolResponse,
  CustomAgentsResponse,
  CustomAgent,
  AgentPresetsResponse,
  AgentPreset,
  SandboxTestRequest,
} from "./types";

// AI Providers
export async function getAIProviders() {
  return request<AIProvidersResponse>("/ai/providers");
}

export async function updateAIProvider(provider: string) {
  return request<{ message: string }>("/ai/provider", {
    method: "PUT",
    body: { provider },
  });
}

export async function getAllModels() {
  return request<AllModelsResponse>("/ai/models");
}

export async function getLocalModels() {
  return request<LocalModelsResponse>("/ai/models/local");
}

export async function pullModel(
  model: string,
): Promise<ReadableStream<Uint8Array> | null> {
  const response = await fetch(`${getApiBaseUrl()}/ai/models/pull`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify({ model }),
  });

  if (!response.ok) {
    const error = await response
      .json()
      .catch(() => ({ detail: "Pull model failed" }));
    throw new Error(
      error.detail || `Failed to pull model (HTTP ${response.status})`,
    );
  }

  return response.body;
}

export async function warmupModel(model: string): Promise<WarmupResponse> {
  return request("/ai/models/warmup", {
    method: "POST",
    body: { model },
  });
}

export async function getAISystemInfo() {
  return request<AISystemInfo>("/ai/system");
}

export async function saveAPIKey(provider: string, apiKey: string) {
  return request<{ message: string }>("/ai/api-key", {
    method: "POST",
    body: { provider, api_key: apiKey },
  });
}

export async function getAgentPrompts() {
  return request<{ agents: AgentInfo[] }>("/ai/agents");
}

export async function getAgentPrompt(id: string) {
  return request<{ id: string; prompt: string }>(`/ai/agents/${id}`);
}

export async function getTools() {
  return request<{ tools: Tool[] }>("/mcp/tools");
}

export async function executeTool(
  toolName: string,
  args: Record<string, unknown>,
) {
  return request<ToolResponse>("/mcp/execute", {
    method: "POST",
    body: { tool_name: toolName, arguments: args },
  });
}

// Custom Agents

/**
 * Get all custom agents for the current user
 * @param includeInactive - Include inactive agents in the response
 */
export async function getCustomAgents(includeInactive = false) {
  const params = includeInactive ? "?include_inactive=true" : "";
  return request<CustomAgentsResponse>(`/ai/custom-agents${params}`);
}

/**
 * Get a specific custom agent by ID
 * @param id - Agent ID
 */
export async function getCustomAgent(id: string) {
  const response = await request<{ agent: CustomAgent }>(
    `/ai/custom-agents/${id}`,
  );
  return response.agent;
}

/**
 * Create a new custom agent
 * @param data - Agent configuration data
 */
export async function createCustomAgent(data: Partial<CustomAgent>) {
  const response = await request<{ agent: CustomAgent }>("/ai/custom-agents", {
    method: "POST",
    body: data,
  });
  return response.agent;
}

/**
 * Update an existing custom agent
 * @param id - Agent ID
 * @param data - Partial agent data to update
 */
export async function updateCustomAgent(
  id: string,
  data: Partial<CustomAgent>,
) {
  const response = await request<{ agent: CustomAgent }>(
    `/ai/custom-agents/${id}`,
    {
      method: "PUT",
      body: data,
    },
  );
  return response.agent;
}

/**
 * Delete a custom agent
 * @param id - Agent ID
 */
export async function deleteCustomAgent(id: string) {
  return request<{ message: string }>(`/ai/custom-agents/${id}`, {
    method: "DELETE",
  });
}

/**
 * Get custom agents filtered by category
 * @param category - Category to filter by
 */
export async function getAgentsByCategory(category: string) {
  return request<CustomAgentsResponse>(
    `/ai/custom-agents/category/${encodeURIComponent(category)}`,
  );
}

/**
 * Get all available agent presets
 */
export async function getAgentPresets() {
  return request<AgentPresetsResponse>("/ai/agents/presets");
}

/**
 * Get a specific agent preset by ID
 * @param id - Preset ID
 */
export async function getAgentPreset(id: string) {
  return request<AgentPreset>(`/ai/agents/presets/${id}`);
}

/**
 * Create a new custom agent from a preset
 * @param presetId - ID of the preset to use
 * @param name - Name for the new agent (optional, defaults to preset name)
 */
export async function createFromPreset(presetId: string, name?: string) {
  const response = await request<{ agent: CustomAgent }>(
    `/ai/custom-agents/from-preset/${presetId}`,
    {
      method: "POST",
      body: { name },
    },
  );
  return response.agent;
}

/**
 * Test an existing custom agent with a message
 * @param id - Agent ID
 * @param message - Test message
 * @returns ReadableStream for SSE streaming or null if streaming is disabled
 */
export async function testAgent(
  id: string,
  message: string,
): Promise<ReadableStream<Uint8Array> | null> {
  const response = await fetch(
    `${getApiBaseUrl()}/ai/custom-agents/${id}/test`,
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ test_message: message }),
    },
  );

  if (!response.ok) {
    const error = await response
      .json()
      .catch(() => ({ detail: "Test agent failed" }));
    throw new Error(
      error.detail || `Failed to test agent (HTTP ${response.status})`,
    );
  }

  return response.body;
}

/**
 * Test an agent configuration in sandbox mode (without saving)
 * @param config - Agent configuration to test
 * @returns ReadableStream for SSE streaming
 */
export async function testSandbox(
  config: SandboxTestRequest,
): Promise<ReadableStream<Uint8Array> | null> {
  const response = await fetch(`${getApiBaseUrl()}/ai/custom-agents/sandbox`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "include",
    body: JSON.stringify(config),
  });

  if (!response.ok) {
    const error = await response
      .json()
      .catch(() => ({ detail: "Sandbox test failed" }));
    throw new Error(
      error.detail || `Failed to test in sandbox (HTTP ${response.status})`,
    );
  }

  return response.body;
}
