/**
 * Versioning Components
 *
 * UI components for the app versioning system.
 * See docs/work/VERSIONING_SYSTEM.md for full specification.
 */

export { default as VersionBadge } from './VersionBadge.svelte';
export { default as VersionDropdown } from './VersionDropdown.svelte';
export { default as VersionTimelinePanel } from './VersionTimelinePanel.svelte';
export { default as RestoreConfirmDialog } from './RestoreConfirmDialog.svelte';
export { default as SaveVersionModal } from './SaveVersionModal.svelte';
export { default as VersionPreviewModal } from './VersionPreviewModal.svelte';
export { default as VersionDiffModal } from './VersionDiffModal.svelte';

// Re-export types for convenience
export type {
  Version,
  VersionSummary,
  VersionTrigger,
  VersionBadgeProps,
  VersionDropdownProps,
  VersionTimelinePanelProps,
  RestoreConfirmDialogProps,
  SaveVersionModalProps,
  VersionPreviewModalProps,
} from '$lib/types/versions';

export {
  toVersionSummary,
  getTriggerLabel,
  formatRelativeTime,
} from '$lib/types/versions';
