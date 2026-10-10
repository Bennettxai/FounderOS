/**
 * Dashboard Data Store
 * Owns all API-fetched data for the dashboard: loading state, error state,
 * focus items, projects, tasks, activities, and energy level.
 * Provides mutation handlers that keep local state in sync after API calls.
 *
 * Singleton factory pattern — matches chatUIStore.svelte.ts conventions.
 */

import { api } from "$lib/api";
import type {
  FocusItem,
  DashboardProjectRow,
  DashboardTaskRow,
  DashboardActivityRow,
} from "./types";

function createDashboardDataStore() {
  // ── Loading / error ──────────────────────────────────────────────────────────
  let isLoading = $state(true);
  let error = $state<string | null>(null);

  // ── API data ─────────────────────────────────────────────────────────────────
  let energyLevel = $state<number | null>(null);
  let focusItems = $state<FocusItem[]>([]);
  let projects = $state<DashboardProjectRow[]>([]);
  let tasks = $state<DashboardTaskRow[]>([]);
  let activities = $state<DashboardActivityRow[]>([]);

  // ── Load ─────────────────────────────────────────────────────────────────────

  async function loadDashboard(): Promise<void> {
    try {
      isLoading = true;
      error = null;

      const summary = await api.getDashboardSummary();

      focusItems = summary.focus_items.map((item) => ({
        id: item.id,
        text: item.text,
        completed: item.completed,
      }));

      projects = summary.projects.map((p) => ({
        id: p.id,
        name: p.name,
        clientName: p.client_name ?? undefined,
        projectType: p.project_type,
        dueDate: p.due_date ?? undefined,
        progress: p.progress,
        health: p.health,
        teamCount: p.team_count,
      }));

      tasks = summary.tasks.map((t) => ({
        id: t.id,
        title: t.title,
        projectName: t.project_name ?? undefined,
        dueDate: t.due_date ?? undefined,
        priority: t.priority,
        completed: t.completed,
      }));

      activities = summary.activities.map((a) => ({
        id: a.id,
        type: a.type,
        description: a.description,
        actorName: a.actor_name ?? undefined,
        actorAvatar: a.actor_avatar ?? undefined,
        targetId: a.target_id ?? undefined,
        targetType: a.target_type ?? undefined,
        createdAt: a.created_at,
      }));

      energyLevel = summary.energy_level;
    } catch (err) {
      console.error("Failed to load dashboard:", err);
      error = err instanceof Error ? err.message : "Failed to load dashboard";
    } finally {
      isLoading = false;
    }
  }

  // ── Focus item handlers ───────────────────────────────────────────────────────

  async function handleFocusToggle(id: string): Promise<void> {
    const item = focusItems.find((i) => i.id === id);
    if (!item) return;
    try {
      await api.updateFocusItem(id, { completed: !item.completed });
      focusItems = focusItems.map((i) =>
        i.id === id ? { ...i, completed: !i.completed } : i,
      );
    } catch (err) {
      console.error("Failed to toggle focus item:", err);
    }
  }

  async function handleFocusAdd(text: string): Promise<void> {
    try {
      const newItem = await api.createFocusItem(text);
      focusItems = [
        ...focusItems,
        { id: newItem.id, text: newItem.text, completed: newItem.completed },
      ];
    } catch (err) {
      console.error("Failed to add focus item:", err);
    }
  }

  async function handleFocusRemove(id: string): Promise<void> {
    try {
      await api.deleteFocusItem(id);
      focusItems = focusItems.filter((item) => item.id !== id);
    } catch (err) {
      console.error("Failed to remove focus item:", err);
    }
  }

  function handleFocusEdit(): void {
    // TODO: Implement focus edit mode
    if (import.meta.env.DEV) console.log("Edit focus items");
  }

  // ── Task handlers ─────────────────────────────────────────────────────────────

  async function handleTaskToggle(id: string): Promise<void> {
    try {
      await api.toggleTask(id);
      tasks = tasks.map((task) =>
        task.id === id ? { ...task, completed: !task.completed } : task,
      );
    } catch (err) {
      console.error("Failed to toggle task:", err);
    }
  }

  // ── Energy ────────────────────────────────────────────────────────────────────

  function handleEnergySet(level: number): void {
    energyLevel = level;
    // TODO: Save to backend
  }

  // ── Public API ────────────────────────────────────────────────────────────────

  return {
    get isLoading() {
      return isLoading;
    },
    set isLoading(v: boolean) {
      isLoading = v;
    },

    get error() {
      return error;
    },
    set error(v: string | null) {
      error = v;
    },

    get energyLevel() {
      return energyLevel;
    },
    set energyLevel(v: number | null) {
      energyLevel = v;
    },

    get focusItems() {
      return focusItems;
    },
    set focusItems(v: FocusItem[]) {
      focusItems = v;
    },

    get projects() {
      return projects;
    },
    set projects(v: DashboardProjectRow[]) {
      projects = v;
    },

    get tasks() {
      return tasks;
    },
    set tasks(v: DashboardTaskRow[]) {
      tasks = v;
    },

    get activities() {
      return activities;
    },
    set activities(v: DashboardActivityRow[]) {
      activities = v;
    },

    loadDashboard,
    handleFocusToggle,
    handleFocusAdd,
    handleFocusRemove,
    handleFocusEdit,
    handleTaskToggle,
    handleEnergySet,
  };
}

export const dashboardDataStore = createDashboardDataStore();
