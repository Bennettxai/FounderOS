/**
 * Dashboard Analytics Store
 * Manages the analytics sidepanel visibility, time-range selection, and
 * the seeded (mock) analytics data with simulated loading state.
 *
 * Also holds the static per-widget analytics definitions (widgetAnalytics).
 *
 * Singleton factory pattern — matches chatUIStore.svelte.ts conventions.
 */

import type {
  AnalyticsTimeRange,
  SeededAnalytics,
  WidgetAnalyticsEntry,
  WidgetType,
} from "./types";

// ── Static per-widget analytics definitions ───────────────────────────────────

export const widgetAnalytics: Record<WidgetType, WidgetAnalyticsEntry> = {
  focus: {
    title: "Focus Analytics",
    stats: [
      { label: "Completion Rate", value: "78%", trend: "+12%" },
      { label: "Avg Time per Item", value: "2.3 hrs" },
      { label: "Current Streak", value: "7 days" },
      { label: "Best Day", value: "Tuesday" },
    ],
  },
  "quick-actions": {
    title: "Quick Actions Analytics",
    stats: [
      { label: "Most Used", value: "New Task" },
      { label: "Actions Today", value: 12 },
      { label: "Time Saved", value: "~45 min" },
    ],
  },
  projects: {
    title: "Projects Analytics",
    stats: [
      { label: "Active Projects", value: 3 },
      { label: "Avg Progress", value: "67%" },
      { label: "On-time Rate", value: "85%", trend: "+5%" },
      { label: "At Risk", value: 1 },
    ],
  },
  tasks: {
    title: "Tasks Analytics",
    stats: [
      { label: "Completed This Week", value: 23, trend: "+18%" },
      { label: "Due Today", value: 5 },
      { label: "Overdue", value: 2 },
      { label: "Avg/Day", value: "4.6" },
    ],
  },
  activity: {
    title: "Activity Analytics",
    stats: [
      { label: "Total Actions", value: 47 },
      { label: "Most Active", value: "Wednesday" },
      { label: "Top Activity", value: "Completing Tasks" },
    ],
  },
  metric: {
    title: "Metric Analytics",
    stats: [
      { label: "Current Value", value: 8 },
      { label: "vs Yesterday", value: "+12%" },
    ],
  },
  signal: {
    title: "Signal Analytics",
    stats: [
      { label: "Status", value: "Healthy" },
      { label: "Metrics Passing", value: "6/6" },
      { label: "Feedback Loop", value: "Active" },
    ],
  },
  "analytics-overview": {
    title: "Analytics Overview",
    stats: [
      { label: "Tasks This Week", value: 23, trend: "+18%" },
      { label: "Focus Rate", value: "78%", trend: "+12%" },
      { label: "Active Projects", value: 3 },
      { label: "Day Streak", value: "7 days" },
    ],
  },
};

// ── Default seeded analytics data ─────────────────────────────────────────────

const DEFAULT_ANALYTICS: SeededAnalytics = {
  focus: {
    completionRate: 78,
    completedToday: 3,
    totalToday: 4,
    streak: 7,
    avgCompletionTime: "2.3 hrs",
    weeklyData: [4, 5, 6, 4, 3, 2, 0],
  },
  tasks: {
    completedThisWeek: 23,
    dueToday: 5,
    overdue: 2,
    completionRate: 82,
    byPriority: { critical: 2, high: 5, medium: 8, low: 3 },
    weeklyData: [5, 6, 8, 4, 0, 0, 0],
  },
  projects: {
    active: 3,
    completed: 2,
    atRisk: 1,
    onTimeRate: 85,
    avgProgress: 67,
  },
  activity: {
    totalActions: 47,
    mostActiveDay: "Wednesday",
    topActivityType: "task_completed",
    weeklyData: [12, 15, 18, 8, 0, 0, 0],
  },
};

// ── Store ─────────────────────────────────────────────────────────────────────

function createDashboardAnalyticsStore() {
  let showAnalyticsSidepanel = $state(false);
  let analyticsLoading = $state(false);
  let analyticsTimeRange = $state<AnalyticsTimeRange>("week");
  let seededAnalytics = $state<SeededAnalytics | null>({
    ...DEFAULT_ANALYTICS,
  });

  async function handleAnalyticsTimeRangeChange(
    range: AnalyticsTimeRange,
  ): Promise<void> {
    analyticsTimeRange = range;
    analyticsLoading = true;

    // Backend pending: Analytics API endpoint not yet implemented.
    // When ready, replace the mock below with a real fetch.
    await new Promise<void>((resolve) => setTimeout(resolve, 600));

    const multiplier =
      range === "today"
        ? 0.3
        : range === "week"
          ? 1
          : range === "month"
            ? 2.5
            : 4;

    seededAnalytics = {
      focus: {
        completionRate: Math.min(
          100,
          Math.round(78 * (range === "today" ? 0.8 : 1)),
        ),
        completedToday: range === "today" ? 2 : 3,
        totalToday: 4,
        streak: 7,
        avgCompletionTime: "2.3 hrs",
        weeklyData:
          range === "today" ? [0, 0, 0, 0, 0, 0, 2] : [4, 5, 6, 4, 3, 2, 0],
      },
      tasks: {
        completedThisWeek: Math.round(23 * multiplier),
        dueToday: 5,
        overdue: range === "today" ? 1 : 2,
        completionRate: Math.min(
          100,
          Math.round(82 * (0.9 + Math.random() * 0.2)),
        ),
        byPriority: { critical: 2, high: 5, medium: 8, low: 3 },
        weeklyData:
          range === "today"
            ? [0, 0, 0, 0, 0, 0, 3]
            : [5, 6, 8, 4, 0, 0, 0].map((v) => Math.round(v * multiplier)),
      },
      projects: {
        active: 3,
        completed: Math.round(2 * multiplier),
        atRisk: 1,
        onTimeRate: Math.min(100, Math.round(85 * (0.9 + Math.random() * 0.2))),
        avgProgress: Math.min(
          100,
          Math.round(
            67 + (range === "month" ? 10 : range === "30days" ? 15 : 0),
          ),
        ),
      },
      activity: {
        totalActions: Math.round(47 * multiplier),
        mostActiveDay: range === "today" ? "Today" : "Wednesday",
        topActivityType: "task_completed",
        weeklyData:
          range === "today"
            ? [0, 0, 0, 0, 0, 0, 8]
            : [12, 15, 18, 8, 0, 0, 0].map((v) => Math.round(v * multiplier)),
      },
    };

    analyticsLoading = false;
  }

  return {
    get showAnalyticsSidepanel() {
      return showAnalyticsSidepanel;
    },
    set showAnalyticsSidepanel(v: boolean) {
      showAnalyticsSidepanel = v;
    },

    get analyticsLoading() {
      return analyticsLoading;
    },
    set analyticsLoading(v: boolean) {
      analyticsLoading = v;
    },

    get analyticsTimeRange() {
      return analyticsTimeRange;
    },
    set analyticsTimeRange(v: AnalyticsTimeRange) {
      analyticsTimeRange = v;
    },

    get seededAnalytics() {
      return seededAnalytics;
    },
    set seededAnalytics(v: SeededAnalytics | null) {
      seededAnalytics = v;
    },

    handleAnalyticsTimeRangeChange,
  };
}

export const dashboardAnalyticsStore = createDashboardAnalyticsStore();
