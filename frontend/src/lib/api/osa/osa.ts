// OSA-5 API Client — aligned with backend routes (2026-02-11)
// Route registry: handlers.go confirmed by Pedro

import { request } from "../base";
import type {
  OSAHealthResponse,
  GenerateAppRequest,
  GenerateAppResponse,
  QueueItemStatus,
  OSAWorkspacesResponse,
  OSAGenerationEvent,
} from "./types";

/**
 * Check OSA integration health and availability
 * GET /api/osa/health (public, no auth)
 */
export async function checkOSAHealth(): Promise<OSAHealthResponse> {
  return request<OSAHealthResponse>("/osa/health");
}

/**
 * Generate a new application via the queue-based pipeline
 * POST /api/workspaces/:workspace_id/apps/generate-osa (handlers.go:569)
 *
 * @param workspaceId - Workspace UUID
 * @param req - Generation request (app_name, description, complexity, etc.)
 * @returns queue_item_id for SSE tracking
 */
export async function generateApp(
  workspaceId: string,
  req: GenerateAppRequest,
): Promise<GenerateAppResponse> {
  return request<GenerateAppResponse>(
    `/workspaces/${workspaceId}/apps/generate-osa`,
    {
      method: "POST",
      body: req,
    },
  );
}

/**
 * Get queue item status (polling fallback for SSE)
 * GET /api/osa/module-instances/queue/:queue_item_id/status (handlers.go:1472)
 *
 * @param queueItemId - Queue item UUID returned from generateApp()
 */
export async function getQueueItemStatus(
  queueItemId: string,
): Promise<QueueItemStatus> {
  return request<QueueItemStatus>(
    `/osa/module-instances/queue/${queueItemId}/status`,
  );
}

/**
 * Get list of available OSA workspaces
 * GET /api/osa/workspaces (handlers.go:1407)
 */
export async function getWorkspaces(): Promise<OSAWorkspacesResponse> {
  return request<OSAWorkspacesResponse>("/osa/workspaces");
}

/**
 * Stream app generation progress using Server-Sent Events
 * GET /api/osa/module-instances/generate/:queue_item_id/stream (handlers.go:1478)
 *
 * @param queueItemId - Queue item ID returned from generation endpoint
 * @returns EventSource for listening to generation events
 */
export function streamAppGeneration(queueItemId: string): EventSource | null {
  try {
    const eventSource = new EventSource(
      `/api/osa/module-instances/generate/${queueItemId}/stream`,
    );
    return eventSource;
  } catch (error) {
    console.error("Failed to create EventSource for OSA generation:", error);
    return null;
  }
}

/**
 * Parse OSA generation event from SSE stream
 * @param event - MessageEvent from EventSource
 * @returns Parsed OSA generation event
 */
export function parseGenerationEvent(
  event: MessageEvent,
): OSAGenerationEvent | null {
  try {
    const data = JSON.parse(event.data);
    return data as OSAGenerationEvent;
  } catch (error) {
    console.error("Failed to parse OSA generation event:", error);
    return null;
  }
}

/**
 * Cancel an in-progress app generation
 * POST /api/osa/module-instances/generate/:queue_item_id/cancel (handlers.go:1475)
 *
 * @param queueItemId - Queue item UUID to cancel
 */
export async function cancelGeneration(queueItemId: string): Promise<void> {
  return request<void>(`/osa/module-instances/generate/${queueItemId}/cancel`, {
    method: "POST",
  });
}
