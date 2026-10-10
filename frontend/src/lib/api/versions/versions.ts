import { request } from '../base';
import type {
  BackendVersionInfo,
  BackendVersionDiffResult,
} from './types';

/**
 * List all versions for an app
 * GET /workspaces/:workspaceId/apps/:appId/versions
 */
export async function listAppVersions(
  workspaceId: string,
  appId: string,
): Promise<BackendVersionInfo[]> {
  const response = await request<{ versions: BackendVersionInfo[] }>(
    `/workspaces/${workspaceId}/apps/${appId}/versions`
  );
  return response.versions ?? [];
}

/**
 * Create a manual version snapshot
 * POST /workspaces/:workspaceId/apps/:appId/versions
 */
export async function createAppSnapshot(
  workspaceId: string,
  appId: string,
  changeSummary?: string,
): Promise<BackendVersionInfo> {
  return request<BackendVersionInfo>(
    `/workspaces/${workspaceId}/apps/${appId}/versions`,
    {
      method: 'POST',
      body: changeSummary ? { change_summary: changeSummary } : {},
    }
  );
}

/**
 * Restore app to a specific version
 * POST /workspaces/:workspaceId/apps/:appId/restore/:versionNumber
 */
export async function restoreAppVersion(
  workspaceId: string,
  appId: string,
  versionNumber: string,
): Promise<{ message: string; version_number: string }> {
  return request<{ message: string; version_number: string }>(
    `/workspaces/${workspaceId}/apps/${appId}/restore/${versionNumber}`,
    { method: 'POST' }
  );
}

/**
 * Compare two workspace versions (file-level diff)
 * GET /workspaces/:workspaceId/versions/:v1/diff/:v2
 */
export async function compareVersions(
  workspaceId: string,
  fromVersion: string,
  toVersion: string,
): Promise<BackendVersionDiffResult> {
  return request<BackendVersionDiffResult>(
    `/workspaces/${workspaceId}/versions/${fromVersion}/diff/${toVersion}`
  );
}
