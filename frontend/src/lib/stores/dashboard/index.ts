/**
 * Dashboard Stores — Barrel Export
 * Three domain stores extracted from dashboard/+page.svelte
 */

export { dashboardLayoutStore } from "./dashboardLayoutStore.svelte";
export {
  accentColors,
  accentColorClasses,
  availableWidgets,
  uniqueWidgetTypes,
  configurableWidgetTypes,
  getAccentColorClass,
  getWidgetGridClass,
  getAccentBorderClass,
} from "./dashboardLayoutStore.svelte";

export {
  dashboardAnalyticsStore,
  widgetAnalytics,
} from "./dashboardAnalyticsStore.svelte";

export { dashboardDataStore } from "./dashboardDataStore.svelte";

// Re-export all types
export type {
  WidgetType,
  WidgetSize,
  Widget,
  UndoEntry,
  AnalyticsTimeRange,
  AnalyticsStat,
  WidgetAnalyticsEntry,
  SeededAnalytics,
  FocusItem,
  DashboardProjectRow,
  DashboardTaskRow,
  DashboardActivityRow,
} from "./types";
