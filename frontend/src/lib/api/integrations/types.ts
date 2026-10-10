// Integrations API Types

// Generic integration types
export type IntegrationProvider =
  | 'google' | 'slack' | 'notion' | 'hubspot' | 'gohighlevel'
  | 'linear' | 'asana' | 'monday' | 'trello' | 'jira' | 'clickup'
  | 'zoom' | 'loom' | 'fireflies' | 'fathom' | 'tldv' | 'granola'
  | 'dropbox' | 'discord' | 'teams' | 'salesforce' | 'pipedrive'
  | 'instagram' | 'tiktok' | 'youtube'
  | 'chatgpt' | 'claude' | 'perplexity' | 'gemini'
  | 'evernote' | 'obsidian' | 'roam' | 'apple_notes';

export interface GenericAuthResponse {
  auth_url: string;
}

export interface GenericConnectionStatus {
  connected: boolean;
  provider: IntegrationProvider;
  account_name?: string;
  account_id?: string;
  account_email?: string;
  connected_at?: string;
  last_synced_at?: string;
  sync_enabled?: boolean;
  error?: string;
}

export interface AllIntegrationsStatus {
  integrations: Record<IntegrationProvider, GenericConnectionStatus>;
}

export interface IntegrationSyncResponse {
  success: boolean;
  synced_count: number;
  message?: string;
}

// Google
export interface GoogleAuthResponse {
  auth_url: string;
}

export interface GoogleConnectionStatus {
  connected: boolean;
  email?: string;
  connected_at?: string;
}

// Slack
export interface SlackAuthResponse {
  auth_url: string;
}

export interface SlackConnectionStatus {
  connected: boolean;
  workspace_name?: string;
  workspace_id?: string;
  user_name?: string;
  user_id?: string;
  connected_at?: string;
}

// SlackChannel + SlackChannelsResponse live in $lib/api/slack/types now —
// re-exported here for backwards-compat with callers that still import
// from $lib/api/integrations. Single source of truth = slack/types.ts.
export type { SlackChannel, SlackChannelsResponse } from "$lib/api/slack/types";

export interface SlackNotification {
  id: string;
  channel_id: string;
  channel_name: string;
  text: string;
  user_id: string;
  user_name: string;
  timestamp: string;
  thread_ts?: string;
}

export interface SlackNotificationsResponse {
  notifications: SlackNotification[];
  has_more: boolean;
}

// Notion
export interface NotionAuthResponse {
  auth_url: string;
}

export interface NotionConnectionStatus {
  connected: boolean;
  workspace_name?: string;
  workspace_id?: string;
  workspace_icon?: string;
  connected_at?: string;
}

export interface NotionDatabase {
  id: string;
  title: string;
  description?: string;
  icon?: string;
  url: string;
  created_time: string;
  last_edited_time: string;
}

export interface NotionPage {
  id: string;
  title: string;
  icon?: string;
  url: string;
  parent_type: 'database' | 'page' | 'workspace';
  parent_id?: string;
  created_time: string;
  last_edited_time: string;
}

export interface NotionDatabasesResponse {
  databases: NotionDatabase[];
}

export interface NotionPagesResponse {
  pages: NotionPage[];
  has_more: boolean;
  next_cursor?: string;
}

export interface NotionSyncResponse {
  success: boolean;
  synced_count: number;
  message?: string;
}

// HubSpot
export interface HubSpotAuthResponse {
  auth_url: string;
}

export interface HubSpotConnectionStatus {
  connected: boolean;
  portal_id?: string;
  portal_name?: string;
  user_email?: string;
  connected_at?: string;
}

// GoHighLevel
export interface GoHighLevelAuthResponse {
  auth_url: string;
}

export interface GoHighLevelConnectionStatus {
  connected: boolean;
  location_id?: string;
  location_name?: string;
  user_email?: string;
  connected_at?: string;
}

// Linear
export interface LinearAuthResponse {
  auth_url: string;
}

export interface LinearConnectionStatus {
  connected: boolean;
  workspace_id?: string;
  workspace_name?: string;
  user_name?: string;
  connected_at?: string;
}

// Asana
export interface AsanaAuthResponse {
  auth_url: string;
}

export interface AsanaConnectionStatus {
  connected: boolean;
  workspace_id?: string;
  workspace_name?: string;
  user_name?: string;
  user_email?: string;
  connected_at?: string;
}

// File Import (for ChatGPT, Claude exports, etc.)
export interface FileImportResponse {
  success: boolean;
  imported_count: number;
  message?: string;
  import_id?: string;
}

export interface ImportProgress {
  import_id: string;
  status: 'pending' | 'processing' | 'completed' | 'failed';
  progress_percent: number;
  imported_count: number;
  total_count: number;
  error?: string;
}

// ============================================================================
// Sorx Integration Module Types
// ============================================================================

export type IntegrationCategory =
  | 'communication'
  | 'crm'
  | 'tasks'
  | 'calendar'
  | 'storage'
  | 'meetings'
  | 'finance'
  | 'code'
  | 'ai'
  | 'custom'
  | 'productivity';

export type IntegrationStatus = 'available' | 'coming_soon' | 'beta' | 'deprecated';
export type UserIntegrationStatus = 'connected' | 'disconnected' | 'expired' | 'error';

export interface IntegrationProviderInfo {
  id: string;
  name: string;
  description: string | null;
  category: IntegrationCategory;
  icon_url: string | null;
  oauth_config?: Record<string, unknown>;
  oauth_provider?: string; // Maps to actual OAuth endpoint (e.g., google_calendar -> google)
  modules: string[];
  skills: string[];
  status: IntegrationStatus;
  // Rich UI display fields
  auto_live_sync?: boolean;
  est_nodes?: string;
  initial_sync?: string;
  tooltip?: string;
}

export interface UserIntegration {
  id: string;
  provider_id: string;
  oauth_provider?: string;
  provider_name: string;
  category: IntegrationCategory;
  icon_url: string | null;
  status: UserIntegrationStatus;
  connected_at: string;
  last_used_at: string | null;
  external_account_id?: string;
  external_account_name?: string;
  external_workspace_id?: string;
  external_workspace_name?: string;
  scopes: string[];
  settings: IntegrationSettings;
  skills: string[];
  modules: string[];
  metadata?: Record<string, unknown>;
  // New fields from enhanced API
  sync_stats?: IntegrationSyncStats;
  sync_history?: SyncHistoryEntry[];
  available_permissions?: AvailablePermission[];
}

export interface IntegrationSettings {
  enabledSkills: string[];
  notifications: boolean;
  syncSettings?: Record<string, unknown>;
}

// Sync Stats returned by backend
export interface IntegrationSyncStats {
  total_items: number;
  items_by_type: Record<string, number>;
  date_range: {
    from: string | null;
    to: string | null;
  } | null;
  last_sync: string | null;
  last_sync_status: string | null;
  sync_count: number;
}

// Sync History entry
export interface SyncHistoryEntry {
  id: string;
  sync_type: string;
  direction: string;
  status: string;
  started_at: string | null;
  completed_at: string | null;
  records_synced: number | null;
  error_message: string | null;
}

// Available Permission for a provider
export interface AvailablePermission {
  scope: string;
  name: string;
  description: string;
  granted: boolean;
}

export interface ModuleIntegrations {
  module: string;
  available_providers: IntegrationProviderInfo[];
  connected_integrations: UserIntegration[];
}

// AI Model Preferences
export interface ModelSelection {
  model_id: string;
  provider: string;
}

export interface AIModelPreferences {
  tier_2_model: ModelSelection;
  tier_3_model: ModelSelection;
  tier_4_model: ModelSelection;
  tier_2_fallbacks: ModelSelection[];
  tier_3_fallbacks: ModelSelection[];
  tier_4_fallbacks: ModelSelection[];
  skill_overrides: Record<string, ModelSelection>;
  allow_model_upgrade_on_failure: boolean;
  max_latency_ms: number;
  prefer_local: boolean;
}

// Pending Decisions (Human-in-the-loop)
export interface PendingDecision {
  id: string;
  execution_id: string;
  skill_id: string;
  step_id: string;
  user_id: string;
  question: string;
  description?: string;
  options: string[];
  input_fields?: Record<string, unknown>;
  context?: Record<string, unknown>;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  status: 'pending' | 'decided' | 'expired' | 'cancelled';
  decision?: string;
  decision_inputs?: Record<string, unknown>;
  decided_by?: string;
  decided_at?: string;
  created_at: string;
  expires_at?: string;
}

// Skill Execution
export interface SkillExecution {
  id: string;
  skill_id: string;
  user_id: string;
  status: 'pending' | 'running' | 'waiting_callback' | 'complete' | 'failed' | 'cancelled';
  current_step: number;
  params: Record<string, unknown>;
  result?: Record<string, unknown>;
  error?: string;
  context: Record<string, unknown>;
  step_results: Record<string, unknown>;
  metrics: Record<string, unknown>;
  started_at: string;
  completed_at?: string;
}

// Sync Log
export interface IntegrationSyncLog {
  id: string;
  user_integration_id: string;
  module_id?: string;
  sync_type: string;
  direction: 'import' | 'export' | 'bidirectional';
  status: 'pending' | 'running' | 'success' | 'failed';
  records_processed: number;
  records_created: number;
  records_updated: number;
  records_failed: number;
  error_message?: string;
  error_details?: Record<string, unknown>;
  started_at: string;
  completed_at?: string;
}
