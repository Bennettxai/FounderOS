/**
 * Sandbox API Client
 *
 * Routes to backend: /api/v1/sandbox/*
 * This client wraps the Docker sandbox deployment endpoints.
 *
 * Backend route reference (handlers/sandbox.go - commit cd8e4b49):
 *   POST   /api/v1/sandbox/deploy         → DeploySandbox
 *   GET    /api/v1/sandbox/:app_id        → GetSandboxInfo
 *   POST   /api/v1/sandbox/:app_id/stop   → StopSandbox
 *   POST   /api/v1/sandbox/:app_id/restart → RestartSandbox
 *   DELETE /api/v1/sandbox/:app_id        → RemoveSandbox
 *   GET    /api/v1/sandbox/:app_id/logs   → GetSandboxLogs (SSE)
 *   GET    /api/v1/sandbox                → ListUserSandboxes
 *   GET    /api/v1/sandbox/stats          → GetSandboxStats
 */
import { request, getApiBaseUrl } from "$lib/api/base";
import type { SandboxStatus, SandboxInfo } from "$lib/types/sandbox";

// Re-export types for backward compatibility
export type { SandboxStatus, SandboxInfo };

// ============================================================================
// Types (matching backend SandboxHandler responses)
// ============================================================================

export interface DeployRequest {
  app_id: string;
  app_name?: string;
  workspace_path?: string;
}

export interface DeployResponse {
  container_id: string;
  port: number;
  url: string;
  status: SandboxStatus;
}

export interface ListSandboxesResponse {
  sandboxes: SandboxInfo[];
  total: number;
}

export interface SandboxStats {
  total_sandboxes: number;
  running: number;
  stopped: number;
  deploying: number;
  failed: number;
}

// ============================================================================
// API Functions
// ============================================================================

/**
 * Deploy a generated app to a Docker sandbox
 * POST /api/v1/sandbox/deploy
 */
export async function deploySandbox(
  appId: string,
  appName?: string,
): Promise<DeployResponse> {
  return request<DeployResponse>("/sandbox/deploy", {
    method: "POST",
    body: { app_id: appId, app_name: appName },
  });
}

/**
 * Get sandbox info for an app
 * GET /api/v1/sandbox/:app_id
 */
export async function getSandboxInfo(appId: string): Promise<SandboxInfo> {
  return request<SandboxInfo>(`/sandbox/${appId}`);
}

/**
 * Stop a running sandbox
 * POST /api/v1/sandbox/:app_id/stop
 */
export async function stopSandbox(appId: string): Promise<{ message: string }> {
  return request<{ message: string }>(`/sandbox/${appId}/stop`, {
    method: "POST",
  });
}

/**
 * Restart a sandbox (single endpoint, not stop+deploy)
 * POST /api/v1/sandbox/:app_id/restart
 */
export async function restartSandbox(appId: string): Promise<DeployResponse> {
  return request<DeployResponse>(`/sandbox/${appId}/restart`, {
    method: "POST",
  });
}

/**
 * Remove a sandbox completely (stops container, releases port)
 * DELETE /api/v1/sandbox/:app_id
 */
export async function removeSandbox(
  appId: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(`/sandbox/${appId}`, {
    method: "DELETE",
  });
}

/**
 * List all sandboxes for current user
 * GET /api/v1/sandbox
 */
export async function listUserSandboxes(): Promise<ListSandboxesResponse> {
  return request<ListSandboxesResponse>("/sandbox");
}

/**
 * Get sandbox statistics
 * GET /api/v1/sandbox/stats
 */
export async function getSandboxStats(): Promise<SandboxStats> {
  return request<SandboxStats>("/sandbox/stats");
}

/**
 * Stream sandbox container logs via SSE
 * GET /api/v1/sandbox/:app_id/logs
 *
 * Returns an EventSource for real-time log streaming.
 * Caller is responsible for closing the connection.
 *
 * @example
 * const source = streamSandboxLogs(appId);
 * source.onmessage = (event) => console.log(event.data);
 * source.onerror = () => source.close();
 */
export function streamSandboxLogs(appId: string): EventSource {
  const baseUrl = getApiBaseUrl();
  const url = `${baseUrl}/sandbox/${appId}/logs`;

  // EventSource doesn't support custom headers, so we use credentials
  // The backend should accept session cookies for auth
  return new EventSource(url, { withCredentials: true });
}
