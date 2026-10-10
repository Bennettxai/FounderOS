/**
 * Backend API response types for workspace/app versioning.
 * Matches Go structs in workspace_version_service.go and app_version_service.go
 */

export interface BackendVersionInfo {
  id: string;
  app_id?: string;
  version_number: string;
  snapshot_data?: Record<string, unknown>;
  snapshot_metadata?: Record<string, unknown>;
  change_summary?: string | null;
  created_by?: string | null;
  created_at: string;
}

export interface BackendVersionDiffResult {
  from_version: string;
  to_version: string;
  summary: BackendDiffSummary;
  files: BackendFileDiff[];
}

export interface BackendDiffSummary {
  files_added: number;
  files_removed: number;
  files_modified: number;
  files_unchanged: number;
  total_lines_added: number;
  total_lines_removed: number;
  apps_added: number;
  apps_removed: number;
}

export interface BackendFileDiff {
  file_path: string;
  change_type: 'added' | 'removed' | 'modified' | 'unchanged';
  language?: string;
  file_type?: string;
  old_content?: string;
  new_content?: string;
  unified_diff?: string;
  lines_added: number;
  lines_removed: number;
}
