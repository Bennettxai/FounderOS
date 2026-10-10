import type { Version, VersionTrigger } from '$lib/types/versions';
import type { BackendVersionInfo } from './types';

/**
 * Extract display number from backend version string.
 * Backend uses "0.0.X" pattern where X increments sequentially.
 */
export function extractDisplayNumber(versionNumber: string): number {
  const parts = versionNumber.split('.');
  const patch = parseInt(parts[parts.length - 1], 10);
  return isNaN(patch) ? 1 : patch;
}

/**
 * Infer a VersionTrigger from backend data.
 */
function inferTrigger(info: BackendVersionInfo): VersionTrigger {
  const summary = (info.change_summary ?? '').toLowerCase();
  if (summary.includes('restore')) return 'restore';
  if (summary.includes('auto')) return 'auto_snapshot';
  if (summary.includes('generation') || summary.includes('generated')) return 'ai_generation';
  if (info.created_by) return 'manual_snapshot';
  return 'ai_generation';
}

/**
 * Map a BackendVersionInfo to the frontend Version type.
 */
export function mapBackendToVersion(
  info: BackendVersionInfo,
  isCurrent: boolean,
): Version {
  return {
    id: info.id,
    appId: info.app_id ?? '',
    versionNumber: extractDisplayNumber(info.version_number),
    backendVersion: info.version_number,
    label: info.change_summary ?? undefined,
    createdAt: new Date(info.created_at),
    createdBy: info.created_by ?? undefined,
    trigger: inferTrigger(info),
    configSnapshot: info.snapshot_data ?? {},
    isCurrent,
  };
}

/**
 * Map a list of backend versions, marking the first one (latest) as current.
 * Backend returns versions ordered by created_at DESC.
 */
export function mapBackendVersionsList(
  infos: BackendVersionInfo[],
): Version[] {
  return infos.map((info, index) =>
    mapBackendToVersion(info, index === 0)
  );
}
