/**
 * BusinessOS Semantic Color System
 *
 * All colors reference CSS custom properties from bos-variables.css.
 * These maps provide TypeScript-friendly access for dynamic styling.
 *
 * RULE: Never hardcode hex values in components. Use these maps
 * or reference CSS variables directly via var(--bos-*).
 */

// ============================================
// STATUS COLORS
// Used for: badges, indicators, alerts, pills
// ============================================

export const STATUS_COLORS = {
  success: "var(--bos-status-success)",
  warning: "var(--bos-status-warning)",
  error: "var(--bos-status-error)",
  info: "var(--bos-status-info)",
  neutral: "var(--bos-status-neutral)",
} as const;

export const STATUS_BG_COLORS = {
  success: "var(--bos-status-success-bg)",
  warning: "var(--bos-status-warning-bg)",
  error: "var(--bos-status-error-bg)",
  info: "var(--bos-status-info-bg)",
  neutral: "var(--bos-status-neutral-bg)",
} as const;

export const STATUS_TEXT_COLORS = {
  success: "var(--bos-status-success-text)",
  warning: "var(--bos-status-warning-text)",
  error: "var(--bos-status-error-text)",
  info: "var(--bos-status-info-text)",
  neutral: "var(--bos-status-neutral-text)",
} as const;

// ============================================
// PRIORITY COLORS
// Used for: task priority indicators, badges
// ============================================

export const PRIORITY_COLORS = {
  critical: "var(--bos-priority-critical)",
  high: "var(--bos-priority-high)",
  medium: "var(--bos-priority-medium)",
  low: "var(--bos-priority-low)",
} as const;

export const PRIORITY_BG_COLORS = {
  critical: "var(--bos-priority-critical-bg)",
  high: "var(--bos-priority-high-bg)",
  medium: "var(--bos-priority-medium-bg)",
  low: "var(--bos-priority-low-bg)",
} as const;

// ============================================
// CATEGORY COLORS
// Used for: module categories, icon backgrounds
// Replaces duplicated categoryHexColors maps
// ============================================

export const CATEGORY_COLORS: Record<string, string> = {
  ai: "var(--bos-category-ai)",
  automation: "var(--bos-category-automation)",
  integration: "var(--bos-category-integration)",
  analytics: "var(--bos-category-analytics)",
  communication: "var(--bos-category-communication)",
  security: "var(--bos-category-security)",
  productivity: "var(--bos-category-productivity)",
  other: "var(--bos-category-other)",
  // Module-specific categories
  finance: "var(--bos-status-success)",
  utilities: "#64748b",
  custom: "var(--bos-status-info)",
};

// ============================================
// PROJECT COLORS
// Used for: project indicators, color dots
// ============================================

export const PROJECT_COLORS = [
  "var(--bos-status-info)",
  "var(--bos-status-success)",
  "var(--bos-category-ai)",
  "var(--bos-status-warning)",
  "var(--bos-status-error)",
  "var(--bos-category-automation)",
] as const;

// ============================================
// TASK STATUS MAP
// Maps status strings to semantic colors
// ============================================

export const TASK_STATUS_COLORS: Record<string, string> = {
  pending: "var(--bos-status-neutral)",
  "in-progress": "var(--bos-status-info)",
  "in-review": "var(--bos-category-ai)",
  done: "var(--bos-status-success)",
  completed: "var(--bos-status-success)",
  blocked: "var(--bos-status-warning)",
  cancelled: "var(--bos-status-neutral)",
};

export const TASK_STATUS_BG_COLORS: Record<string, string> = {
  pending: "var(--bos-status-neutral-bg)",
  "in-progress": "var(--bos-status-info-bg)",
  "in-review": "var(--bos-category-ai)",
  done: "var(--bos-status-success-bg)",
  completed: "var(--bos-status-success-bg)",
  blocked: "var(--bos-status-warning-bg)",
  cancelled: "var(--bos-status-neutral-bg)",
};

// ============================================
// PRIORITY MAP
// Maps priority strings to semantic colors
// ============================================

export const TASK_PRIORITY_COLORS: Record<string, string> = {
  critical: "var(--bos-priority-critical)",
  high: "var(--bos-priority-high)",
  medium: "var(--bos-priority-medium)",
  low: "var(--bos-priority-low)",
  none: "var(--bos-status-neutral)",
};

// ============================================
// HELPER: Get category color with fallback
// ============================================

export function getCategoryColor(category: string): string {
  return CATEGORY_COLORS[category.toLowerCase()] ?? CATEGORY_COLORS.other;
}

export function getStatusColor(status: string): string {
  return TASK_STATUS_COLORS[status.toLowerCase()] ?? TASK_STATUS_COLORS.pending;
}

export function getStatusBgColor(status: string): string {
  return (
    TASK_STATUS_BG_COLORS[status.toLowerCase()] ?? TASK_STATUS_BG_COLORS.pending
  );
}

export function getPriorityColor(priority: string): string {
  return (
    TASK_PRIORITY_COLORS[priority.toLowerCase()] ?? TASK_PRIORITY_COLORS.none
  );
}
