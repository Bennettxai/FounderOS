// MIOSA Cloud sync API client.
// All cloud interactions go through the Go backend - the frontend never calls
// api.miosa.ai directly. The API key is stored server-side only.

import { request } from "../base";

export type SyncMode = "local" | "cloud";
export type MiosaCapacityProvider = "local" | "businessos" | "user";

export interface MIOSAConnectionStatus {
  mode: SyncMode;
  connected: boolean;
  api_key_set: boolean;
  last_sync?: string; // ISO 8601
  error?: string;
  capacity_provider?: MiosaCapacityProvider;
  businessos_tenant_available?: boolean;
  businessos_sandbox_enabled?: boolean;
  user_key_available?: boolean;
  workspace_quota?: {
    max_sandboxes?: number;
    max_computers?: number;
    max_desktops?: number;
  };
  usage?: {
    active_sandboxes?: number;
    active_computers?: number;
    active_desktops?: number;
  };
}

export interface SyncResult {
  success: boolean;
  synced_at: string; // ISO 8601
  manifest_id?: string;
  error?: string;
}

export interface MIOSASandboxSession {
  sandbox_id: string;
  miosa_workspace_id?: string;
  external_workspace_id: string;
  external_user_id: string;
  status: string;
  preview_url?: string;
  terminal_session_id?: string;
  expires_at?: number;
  ws_url?: string;
}

/**
 * Returns current OSA mode and MIOSA Cloud connection status.
 * Makes no external network call; safe to call on every settings page load.
 */
export async function getMIOSAStatus(): Promise<MIOSAConnectionStatus> {
  return request<MIOSAConnectionStatus>("/miosa/status");
}

/**
 * Validates the API key stored in the backend against MIOSA Cloud.
 * Call this after the user saves a new API key via Settings.
 */
export async function pingMIOSACloud(): Promise<{
  connected: boolean;
  error?: string;
}> {
  return request<{ connected: boolean; error?: string }>("/miosa/ping", {
    method: "POST",
  });
}

/**
 * Pushes a WorkspaceManifest (config only, no business data) to MIOSA Cloud.
 * In local mode this is a no-op on the server and returns success immediately.
 */
export async function syncToMIOSACloud(
  workspaceId: string,
): Promise<SyncResult> {
  return request<SyncResult>("/miosa/sync", {
    method: "POST",
    body: { workspace_id: workspaceId },
  });
}

export async function createMIOSASandboxSession(input: {
  workspace_id?: string;
  name?: string;
  cols?: number;
  rows?: number;
  shell?: string;
}): Promise<MIOSASandboxSession> {
  return request<MIOSASandboxSession>("/miosa/sandboxes", {
    method: "POST",
    body: input,
    timeout: 90_000,
  });
}
