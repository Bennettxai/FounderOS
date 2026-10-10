import { request, raw } from "../base";
import { withOAuthRedirect } from "$lib/utils/platform";
import type {
  GoogleAuthResponse,
  GoogleConnectionStatus,
  SlackAuthResponse,
  SlackConnectionStatus,
  SlackChannelsResponse,
  SlackNotificationsResponse,
  NotionAuthResponse,
  NotionConnectionStatus,
  NotionDatabasesResponse,
  NotionPagesResponse,
  NotionSyncResponse,
  HubSpotAuthResponse,
  HubSpotConnectionStatus,
  GoHighLevelAuthResponse,
  GoHighLevelConnectionStatus,
  LinearAuthResponse,
  LinearConnectionStatus,
  AsanaAuthResponse,
  AsanaConnectionStatus,
  GenericAuthResponse,
  GenericConnectionStatus,
  AllIntegrationsStatus,
  IntegrationSyncResponse,
  IntegrationProvider,
  FileImportResponse,
  ImportProgress,
} from "./types";

// ============================================
// All Integrations Status
// ============================================

/**
 * Get status of all integrations at once
 */
export async function getAllIntegrationsStatus() {
  return request<AllIntegrationsStatus>("/integrations/status");
}

/**
 * Get status of a specific integration
 */
export async function getIntegrationStatus(provider: IntegrationProvider) {
  return request<GenericConnectionStatus>(`/integrations/${provider}/status`);
}

/**
 * Generic initiate auth for any OAuth provider
 */
export async function initiateAuth(provider: IntegrationProvider) {
  return request<GenericAuthResponse>(`/integrations/${provider}/auth`);
}

/**
 * Generic disconnect for any provider
 */
export async function disconnectIntegration(provider: IntegrationProvider) {
  return request(`/integrations/${provider}`, { method: "DELETE" });
}

/**
 * Trigger sync for a provider
 */
export async function syncIntegration(provider: IntegrationProvider) {
  return request<IntegrationSyncResponse>(`/integrations/${provider}/sync`, {
    method: "POST",
  });
}

// ============================================
// Google OAuth Integration
// Uses new provider infrastructure
// ============================================
export async function initiateGoogleAuth() {
  return request<GoogleAuthResponse>(
    withOAuthRedirect("/integrations/google/auth"),
  );
}

export async function getGoogleConnectionStatus() {
  return request<GoogleConnectionStatus>("/integrations/google/status");
}

export async function disconnectGoogle() {
  return request("/integrations/google/disconnect", { method: "POST" });
}

export async function syncGoogleCalendar() {
  return request<IntegrationSyncResponse>(
    "/integrations/google/calendar/sync",
    { method: "POST" },
  );
}

export async function syncGoogleGmail() {
  return request<IntegrationSyncResponse>("/integrations/google/gmail/sync", {
    method: "POST",
  });
}

// ============================================
// Slack Integration
// Uses new provider infrastructure
// ============================================

/**
 * Initiate Slack OAuth flow
 * Returns URL to redirect user to for Slack authorization
 */
export async function initiateSlackAuth() {
  return request<SlackAuthResponse>("/integrations/slack/auth");
}

/**
 * Get current Slack connection status
 */
export async function getSlackConnectionStatus() {
  return request<SlackConnectionStatus>("/integrations/slack/status");
}

/**
 * Disconnect Slack integration
 */
export async function disconnectSlack() {
  return request("/integrations/slack/disconnect", { method: "POST" });
}

/**
 * Get list of Slack channels the user has access to
 */
export async function getSlackChannels() {
  return request<SlackChannelsResponse>("/integrations/slack/channels");
}

/**
 * Sync Slack channels
 */
export async function syncSlackChannels() {
  return request<IntegrationSyncResponse>("/integrations/slack/channels/sync", {
    method: "POST",
  });
}

/**
 * Get Slack messages for a channel
 */
export async function getSlackMessages(
  channelId: string,
  limit = 50,
  offset = 0,
) {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return request<{ messages: unknown[]; count: number }>(
    `/integrations/slack/messages/${channelId}?${params}`,
  );
}

/**
 * Send a Slack message
 */
export async function sendSlackMessage(channelId: string, content: string) {
  return request<{ success: boolean }>(
    `/integrations/slack/messages/${channelId}`,
    {
      method: "POST",
      body: { content },
    },
  );
}

/**
 * Sync Slack messages for a channel
 */
export async function syncSlackMessages(channelId: string) {
  return request<IntegrationSyncResponse>(
    `/integrations/slack/messages/${channelId}/sync`,
    { method: "POST" },
  );
}

/**
 * Get Slack notifications/messages (legacy, maps to messages)
 * @param limit - Number of notifications to fetch (default 50)
 * @param cursor - Pagination cursor for fetching more results
 */
export async function getSlackNotifications(limit = 50, cursor?: string) {
  const params = new URLSearchParams({ limit: String(limit) });
  if (cursor) params.append("cursor", cursor);
  return request<SlackNotificationsResponse>(
    `/integrations/slack/notifications?${params}`,
  );
}

// ============================================
// Notion Integration
// Uses new provider infrastructure
// ============================================

/**
 * Initiate Notion OAuth flow
 * Returns URL to redirect user to for Notion authorization
 */
export async function initiateNotionAuth() {
  return request<NotionAuthResponse>("/integrations/notion/auth");
}

/**
 * Get current Notion connection status
 */
export async function getNotionConnectionStatus() {
  return request<NotionConnectionStatus>("/integrations/notion/status");
}

/**
 * Disconnect Notion integration
 */
export async function disconnectNotion() {
  return request("/integrations/notion/disconnect", { method: "POST" });
}

/**
 * Get list of Notion databases the integration has access to
 */
export async function getNotionDatabases() {
  return request<NotionDatabasesResponse>("/integrations/notion/databases");
}

/**
 * Sync Notion databases
 */
export async function syncNotionDatabases() {
  return request<NotionSyncResponse>("/integrations/notion/databases/sync", {
    method: "POST",
  });
}

/**
 * Get Notion pages for a database
 * @param databaseId - The database ID to get pages for
 * @param limit - Number of pages to fetch
 * @param offset - Pagination offset
 */
export async function getNotionPages(
  databaseId: string,
  limit = 50,
  offset = 0,
) {
  const params = new URLSearchParams({
    limit: String(limit),
    offset: String(offset),
  });
  return request<NotionPagesResponse>(
    `/integrations/notion/pages/${databaseId}?${params}`,
  );
}

/**
 * Sync pages for a Notion database
 * @param databaseId - The Notion database ID to sync pages for
 */
export async function syncNotionPages(databaseId: string) {
  return request<NotionSyncResponse>(
    `/integrations/notion/pages/${databaseId}/sync`,
    { method: "POST" },
  );
}

/**
 * Sync a Notion database to BusinessOS (legacy, use syncNotionPages)
 * @param databaseId - The Notion database ID to sync
 */
export async function syncNotionDatabase(databaseId: string) {
  return syncNotionPages(databaseId);
}

// ============================================
// HubSpot Integration
// ============================================

export async function initiateHubSpotAuth() {
  return request<HubSpotAuthResponse>("/integrations/hubspot/auth");
}

export async function getHubSpotConnectionStatus() {
  return request<HubSpotConnectionStatus>("/integrations/hubspot/status");
}

export async function disconnectHubSpot() {
  return request("/integrations/hubspot", { method: "DELETE" });
}

export async function syncHubSpot() {
  return request<IntegrationSyncResponse>("/integrations/hubspot/sync", {
    method: "POST",
  });
}

// ============================================
// GoHighLevel Integration
// ============================================

export async function initiateGoHighLevelAuth() {
  return request<GoHighLevelAuthResponse>("/integrations/gohighlevel/auth");
}

export async function getGoHighLevelConnectionStatus() {
  return request<GoHighLevelConnectionStatus>(
    "/integrations/gohighlevel/status",
  );
}

export async function disconnectGoHighLevel() {
  return request("/integrations/gohighlevel", { method: "DELETE" });
}

export async function syncGoHighLevel() {
  return request<IntegrationSyncResponse>("/integrations/gohighlevel/sync", {
    method: "POST",
  });
}

// ============================================
// Linear Integration
// ============================================

export async function initiateLinearAuth() {
  return request<LinearAuthResponse>("/integrations/linear/auth");
}

export async function getLinearConnectionStatus() {
  return request<LinearConnectionStatus>("/integrations/linear/status");
}

export async function disconnectLinear() {
  return request("/integrations/linear", { method: "DELETE" });
}

export async function syncLinear() {
  return request<IntegrationSyncResponse>("/integrations/linear/sync", {
    method: "POST",
  });
}

// ============================================
// Asana Integration
// ============================================

export async function initiateAsanaAuth() {
  return request<AsanaAuthResponse>("/integrations/asana/auth");
}

export async function getAsanaConnectionStatus() {
  return request<AsanaConnectionStatus>("/integrations/asana/status");
}

export async function disconnectAsana() {
  return request("/integrations/asana", { method: "DELETE" });
}

export async function syncAsana() {
  return request<IntegrationSyncResponse>("/integrations/asana/sync", {
    method: "POST",
  });
}

// ============================================
// File Import (ChatGPT, Claude exports, etc.)
// ============================================

/**
 * Import data from a file export (ChatGPT conversations, Claude exports, etc.)
 * @param file - The file to import
 * @param source - The source of the export (chatgpt, claude, perplexity, etc.)
 */
export async function importFile(
  file: File,
  source: "chatgpt" | "claude" | "perplexity" | "gemini" | "other",
) {
  const formData = new FormData();
  formData.append("file", file);
  formData.append("source", source);

  const response = await raw.postFormData("/integrations/import", formData);
  if (!response.ok) {
    const error = await response
      .json()
      .catch(() => ({ detail: "Import failed" }));
    throw new Error(error.detail || "Import failed");
  }
  return response.json() as Promise<FileImportResponse>;
}

/**
 * Get progress of an ongoing import
 */
export async function getImportProgress(importId: string) {
  return request<ImportProgress>(`/integrations/import/${importId}/progress`);
}

// ============================================
// Custom MCP Connector
// ============================================

export interface MCPConnectorTool {
  name: string;
  description: string;
  input_schema: Record<string, unknown>;
}

export interface MCPConnector {
  id: string;
  name: string;
  description: string;
  server_url: string;
  auth_type: "none" | "api_key" | "bearer";
  has_auth: boolean;
  enabled: boolean;
  status: "connected" | "disconnected" | "error";
  transport: string;
  tools: MCPConnectorTool[];
  tool_count: number;
  last_connected_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateMCPConnectorData {
  name: string;
  description?: string;
  server_url: string;
  auth_type: "none" | "api_key" | "bearer";
  auth_token?: string;
  transport?: string;
  custom_headers?: Record<string, string>;
}

export interface TestMCPConnectorResponse {
  success: boolean;
  message: string;
  tools_count?: number;
  tools?: MCPConnectorTool[];
}

export async function getMCPConnectors() {
  return request<{ servers: MCPConnector[] }>("/integrations/mcp/connectors");
}

export async function createMCPConnector(data: CreateMCPConnectorData) {
  return request<MCPConnector>("/integrations/mcp/connectors", {
    method: "POST",
    body: data,
  });
}

export async function updateMCPConnector(
  id: string,
  data: Partial<CreateMCPConnectorData>,
) {
  return request<MCPConnector>(`/integrations/mcp/connectors/${id}`, {
    method: "PUT",
    body: data,
  });
}

export async function deleteMCPConnector(id: string) {
  return request(`/integrations/mcp/connectors/${id}`, { method: "DELETE" });
}

export async function testMCPConnector(id: string) {
  return request<TestMCPConnectorResponse>(
    `/integrations/mcp/connectors/${id}/test`,
    {
      method: "POST",
    },
  );
}

export async function discoverMCPConnectorTools(id: string) {
  return request<TestMCPConnectorResponse>(
    `/integrations/mcp/connectors/${id}/discover`,
    {
      method: "POST",
    },
  );
}

// ============================================
// Sorx Integration Module
// ============================================

import type {
  IntegrationProviderInfo,
  UserIntegration,
  ModuleIntegrations,
  AIModelPreferences,
  IntegrationSettings,
  PendingDecision,
  SkillExecution,
  IntegrationCategory,
} from "./types";

/**
 * Get all available integration providers
 */
export async function getProviders(options?: {
  category?: IntegrationCategory;
  module?: string;
  status?: string;
}) {
  const params = new URLSearchParams();
  if (options?.category) params.set("category", options.category);
  if (options?.module) params.set("module", options.module);
  if (options?.status) params.set("status", options.status);
  const query = params.toString();
  return request<{
    success: boolean;
    providers: IntegrationProviderInfo[];
    count: number;
  }>(`/integrations/providers${query ? `?${query}` : ""}`);
}

/**
 * Get a specific provider's details
 */
export async function getProvider(providerId: string) {
  return request<{ success: boolean; provider: IntegrationProviderInfo }>(
    `/integrations/providers/${providerId}`,
  );
}

/**
 * Get user's connected integrations
 */
export async function getConnectedIntegrations() {
  return request<{
    success: boolean;
    integrations: UserIntegration[];
    count: number;
  }>("/integrations/connected");
}

/**
 * Get details of a specific user integration
 */
export async function getUserIntegration(integrationId: string) {
  return request<{ success: boolean; integration: UserIntegration }>(
    `/integrations/${integrationId}`,
  );
}

/**
 * Update integration settings
 */
export async function updateIntegrationSettings(
  integrationId: string,
  settings: Partial<IntegrationSettings>,
) {
  return request<{ success: boolean; message: string }>(
    `/integrations/${integrationId}/settings`,
    {
      method: "PATCH",
      body: settings,
    },
  );
}

/**
 * Disconnect an integration
 */
export async function disconnectUserIntegration(integrationId: string) {
  return request<{ success: boolean; message: string }>(
    `/integrations/${integrationId}`,
    { method: "DELETE" },
  );
}

/**
 * Trigger manual sync for an integration
 */
export async function triggerIntegrationSync(
  integrationId: string,
  module?: string,
) {
  const params = module ? `?module=${module}` : "";
  return request<{ success: boolean; sync_log_id: string; message: string }>(
    `/integrations/${integrationId}/sync${params}`,
    { method: "POST" },
  );
}

/**
 * Get integrations available for a specific module
 */
export async function getModuleIntegrations(moduleId: string) {
  return request<{ success: boolean } & ModuleIntegrations>(
    `/modules/${moduleId}/integrations`,
  );
}

// ============================================
// AI Model Preferences
// ============================================

/**
 * Get user's AI model preferences
 */
export async function getAIModelPreferences() {
  return request<{ success: boolean; preferences: AIModelPreferences }>(
    "/integrations/ai/preferences",
  );
}

/**
 * Update user's AI model preferences
 */
export async function updateAIModelPreferences(
  preferences: Partial<AIModelPreferences>,
) {
  return request<{ success: boolean; message: string }>(
    "/integrations/ai/preferences",
    {
      method: "PUT",
      body: preferences,
    },
  );
}

// ============================================
// Sorx Decisions (Human-in-the-loop)
// ============================================

/**
 * Get pending decisions requiring human input
 */
export async function getPendingDecisions() {
  return request<{
    success: boolean;
    decisions: PendingDecision[];
    count: number;
  }>("/sorx/decisions");
}

/**
 * Get a specific pending decision
 */
export async function getPendingDecision(decisionId: string) {
  return request<{ success: boolean; decision: PendingDecision }>(
    `/sorx/decisions/${decisionId}`,
  );
}

/**
 * Respond to a pending decision
 */
export async function respondToDecision(
  decisionId: string,
  response: {
    decision: string;
    inputs?: Record<string, unknown>;
    comment?: string;
  },
) {
  return request<{ success: boolean; message: string }>(
    `/sorx/decisions/${decisionId}/respond`,
    {
      method: "POST",
      body: response,
    },
  );
}

// ============================================
// Sorx Skill Execution
// ============================================

/**
 * Trigger a skill execution
 */
export async function triggerSkill(
  skillId: string,
  params?: Record<string, unknown>,
) {
  return request<{
    success: boolean;
    execution_id: string;
    skill_id: string;
    status: string;
    message: string;
  }>("/sorx/execute", {
    method: "POST",
    body: { skill_id: skillId, params },
  });
}

/**
 * Get skill execution status
 */
export async function getSkillExecution(executionId: string) {
  return request<{ success: boolean; execution: SkillExecution }>(
    `/sorx/executions/${executionId}`,
  );
}
