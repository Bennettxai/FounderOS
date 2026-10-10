/**
 * Dashboard Shared Types
 * Centralised type definitions consumed by all dashboard stores and the page component.
 */

// ── Widget domain ─────────────────────────────────────────────────────────────

export type WidgetType =
  | "focus"
  | "quick-actions"
  | "projects"
  | "tasks"
  | "activity"
  | "metric"
  | "signal"
  | "analytics-overview";

export type WidgetSize = "small" | "medium" | "large";

export interface Widget {
  id: string;
  type: WidgetType;
  title: string;
  size: WidgetSize;
  config?: Record<string, unknown>;
  collapsed?: boolean;
  accentColor?: string;
  showAnalytics?: boolean;
}

export interface UndoEntry {
  widget: Widget;
  index: number;
  timestamp: number;
}

// ── Analytics domain ──────────────────────────────────────────────────────────

export type AnalyticsTimeRange = "today" | "week" | "month" | "30days";

export interface AnalyticsStat {
  label: string;
  value: string | number;
  trend?: string;
}

export interface WidgetAnalyticsEntry {
  title: string;
  stats: AnalyticsStat[];
}

export interface SeededAnalytics {
  focus: {
    completionRate: number;
    completedToday: number;
    totalToday: number;
    streak: number;
    avgCompletionTime: string;
    weeklyData: number[];
  };
  tasks: {
    completedThisWeek: number;
    dueToday: number;
    overdue: number;
    completionRate: number;
    byPriority: { critical: number; high: number; medium: number; low: number };
    weeklyData: number[];
  };
  projects: {
    active: number;
    completed: number;
    atRisk: number;
    onTimeRate: number;
    avgProgress: number;
  };
  activity: {
    totalActions: number;
    mostActiveDay: string;
    topActivityType: string;
    weeklyData: number[];
  };
}

// ── Data domain ───────────────────────────────────────────────────────────────

export interface FocusItem {
  id: string;
  text: string;
  completed: boolean;
}

export interface DashboardProjectRow {
  id: string;
  name: string;
  clientName?: string;
  projectType: string;
  dueDate?: string;
  progress: number;
  health: "healthy" | "at_risk" | "critical";
  teamCount: number;
}

export interface DashboardTaskRow {
  id: string;
  title: string;
  projectName?: string;
  dueDate?: string;
  priority: "critical" | "high" | "medium" | "low";
  completed: boolean;
}

export interface DashboardActivityRow {
  id: string;
  type:
    | "task_completed"
    | "task_started"
    | "project_created"
    | "project_updated"
    | "conversation"
    | "team"
    | "artifact";
  description: string;
  actorName?: string;
  actorAvatar?: string;
  targetId?: string;
  targetType?: string;
  createdAt: string;
}
