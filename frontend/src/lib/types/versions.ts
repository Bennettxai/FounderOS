/**
 * Version System Types
 *
 * Types for app versioning, history tracking, and restore functionality.
 * Based on the versioning system spec in docs/work/VERSIONING_SYSTEM.md
 */

// What triggered the version creation
export type VersionTrigger =
  | 'ai_generation'    // AI created/modified the app
  | 'user_edit'        // User changed settings
  | 'manual_snapshot'  // User clicked "Save Version"
  | 'auto_snapshot'    // System saved before risky change
  | 'restore';         // Created when restoring older version

// Version status
export type VersionStatus = 'current' | 'previous' | 'restored';

/**
 * Full version record from database
 */
export interface Version {
  id: string;
  appId: string;
  versionNumber: number;

  // Metadata
  label?: string;              // User-provided or auto-generated label
  createdAt: Date;
  createdBy?: string;          // User ID who created this version
  createdByName?: string;      // User display name

  // What triggered this version
  trigger: VersionTrigger;
  prompt?: string;             // The prompt/action that created this version

  // Snapshots
  configSnapshot: Record<string, unknown>;  // Full app config at this point
  codeSnapshotRef?: string;    // Git commit SHA or storage ref (OSA-5 only)

  // Backend reference
  backendVersion?: string;     // Raw backend version string (e.g., "0.0.1") for API calls

  // Lineage
  parentVersionId?: string;    // Previous version this was based on

  // Status
  isCurrent: boolean;
}

/**
 * Lightweight version for dropdowns and lists
 */
export interface VersionSummary {
  id: string;
  versionNumber: number;
  label?: string;
  createdAt: Date;
  trigger: VersionTrigger;
  isCurrent: boolean;
}

/**
 * Version comparison diff item
 */
export interface VersionDiffItem {
  path: string;           // e.g., "fields.priority" or "views.kanban.columns"
  type: 'added' | 'removed' | 'changed';
  oldValue?: unknown;
  newValue?: unknown;
}

/**
 * Version comparison result
 */
export interface VersionComparison {
  fromVersion: VersionSummary;
  toVersion: VersionSummary;
  diffs: VersionDiffItem[];
  summary: {
    added: number;
    removed: number;
    changed: number;
  };
}

/**
 * Props for version components
 */
export interface VersionBadgeProps {
  currentVersion: number;
  hasUnsavedChanges?: boolean;
  isPreviewingOldVersion?: boolean;
  previewingVersion?: number;
  onclick?: () => void;
}

export interface VersionDropdownProps {
  appId: string;
  currentVersion: number;
  versions: VersionSummary[];
  isLoading?: boolean;
  onVersionSelect: (version: VersionSummary) => void;
  onViewAll: () => void;
  onSaveVersion: () => void;
}

export interface VersionTimelinePanelProps {
  appId: string;
  versions: Version[];
  isOpen: boolean;
  isLoading?: boolean;
  onClose: () => void;
  onPreview: (version: Version) => void;
  onRestore: (version: Version) => void;
  onCompare?: () => void;
}

export interface RestoreConfirmDialogProps {
  version: Version;
  currentVersion: number;
  isOpen: boolean;
  isRestoring?: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

export interface SaveVersionModalProps {
  appId: string;
  currentVersion: number;
  isOpen: boolean;
  isSaving?: boolean;
  onClose: () => void;
  onSave: (label?: string) => void;
}

export interface VersionPreviewModalProps {
  version: Version;
  isOpen: boolean;
  onClose: () => void;
  onRestore: (version: Version) => void;
}

/**
 * Helper to convert Version to VersionSummary
 */
export function toVersionSummary(version: Version): VersionSummary {
  return {
    id: version.id,
    versionNumber: version.versionNumber,
    label: version.label,
    createdAt: version.createdAt,
    trigger: version.trigger,
    isCurrent: version.isCurrent,
  };
}

/**
 * Helper to get trigger display text
 */
export function getTriggerLabel(trigger: VersionTrigger): string {
  const labels: Record<VersionTrigger, string> = {
    ai_generation: 'AI Generation',
    user_edit: 'User Edit',
    manual_snapshot: 'Manual Save',
    auto_snapshot: 'Auto Save',
    restore: 'Restored',
  };
  return labels[trigger];
}

/**
 * Helper to format relative time
 */
export function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffMins < 1) return 'Just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays === 1) return 'Yesterday';
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}
