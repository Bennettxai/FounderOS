/**
 * Dashboard Layout Store
 * Owns all widget-editor state: the widget list, edit mode, drag-and-drop,
 * widget picker visibility, undo stack, and keyboard selection.
 *
 * Singleton factory pattern — matches chatUIStore.svelte.ts conventions.
 */

import type { Widget, WidgetType, WidgetSize, UndoEntry } from "./types";

// ── Constants (static, shared between store and page) ────────────────────────

export const accentColors: { name: string; value: string }[] = [
  { name: "Default", value: "" },
  { name: "Blue", value: "blue" },
  { name: "Green", value: "green" },
  { name: "Purple", value: "purple" },
  { name: "Orange", value: "orange" },
  { name: "Pink", value: "pink" },
];

export const accentColorClasses: Record<string, string> = {
  "": "bg-gray-300",
  blue: "bg-blue-500",
  green: "bg-green-500",
  purple: "bg-purple-500",
  orange: "bg-orange-500",
  pink: "bg-pink-500",
};

export const availableWidgets: {
  type: WidgetType;
  title: string;
  description: string;
  icon: string;
}[] = [
  {
    type: "focus",
    title: "Today's Focus",
    description: "Track your daily priorities",
    icon: "M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 14c-2.21 0-4-1.79-4-4s1.79-4 4-4 4 1.79 4 4-1.79 4-4 4zm0-6c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2z",
  },
  {
    type: "quick-actions",
    title: "Quick Actions",
    description: "Common shortcuts",
    icon: "M13 10V3L4 14h7v7l9-11h-7z",
  },
  {
    type: "projects",
    title: "Active Projects",
    description: "Project progress overview",
    icon: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z",
  },
  {
    type: "tasks",
    title: "My Tasks",
    description: "Tasks due soon",
    icon: "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4",
  },
  {
    type: "activity",
    title: "Recent Activity",
    description: "Latest workspace activity",
    icon: "M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z",
  },
  {
    type: "metric",
    title: "Metric Card",
    description: "Single KPI display",
    icon: "M16 8v8m-4-5v5m-4-2v2m-2 4h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z",
  },
  {
    type: "signal",
    title: "Signal Health",
    description: "Signal theory system status",
    icon: "M13 7h8m0 0v8m0-8l-8 8-4-4-6 6",
  },
  {
    type: "analytics-overview",
    title: "Analytics Overview",
    description: "Key metrics at a glance",
    icon: "M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z",
  },
];

/** Widget types that may only exist once on the dashboard at a time. */
export const uniqueWidgetTypes: WidgetType[] = [
  "focus",
  "quick-actions",
  "activity",
  "signal",
];

/** Widget types that can have multiple instances (differ by config). */
export const configurableWidgetTypes: WidgetType[] = [
  "metric",
  "projects",
  "tasks",
];

// ── Utility helpers (pure, no store dependency) ───────────────────────────────

export function getAccentColorClass(colorValue: string): string {
  return accentColorClasses[colorValue] ?? "bg-gray-300";
}

export function getWidgetGridClass(size: WidgetSize): string {
  switch (size) {
    case "small":
      return "dw-widget-sm";
    case "medium":
      return "dw-widget-md";
    case "large":
      return "dw-widget-lg";
  }
}

export function getAccentBorderClass(color?: string): string {
  switch (color) {
    case "blue":
      return "border-l-4 border-l-blue-500";
    case "green":
      return "border-l-4 border-l-green-500";
    case "purple":
      return "border-l-4 border-l-purple-500";
    case "orange":
      return "border-l-4 border-l-orange-500";
    case "pink":
      return "border-l-4 border-l-pink-500";
    default:
      return "";
  }
}

// ── Store ─────────────────────────────────────────────────────────────────────

function createDashboardLayoutStore() {
  // ── Widget list ──────────────────────────────────────────────────────────────
  let widgets = $state<Widget[]>([
    { id: "w1", type: "focus", title: "Today's Focus", size: "small" },
    { id: "w2", type: "quick-actions", title: "Quick Actions", size: "small" },
    { id: "w3", type: "projects", title: "Active Projects", size: "small" },
    { id: "w4", type: "tasks", title: "My Tasks", size: "small" },
    { id: "w5", type: "activity", title: "Recent Activity", size: "medium" },
  ]);

  // ── Edit mode ────────────────────────────────────────────────────────────────
  let isEditMode = $state(false);
  let draggedWidget = $state<string | null>(null);
  let dragOverWidget = $state<string | null>(null);

  // ── Widget picker ────────────────────────────────────────────────────────────
  let showWidgetPicker = $state(false);
  let selectedWidgetIndex = $state(-1);
  let pickerSelectedSize = $state<WidgetSize>("medium");

  // ── Right panel ─────────────────────────────────────────────────────────────
  let showRightPanel = $state(false);

  // ── Undo ─────────────────────────────────────────────────────────────────────
  let undoStack = $state<UndoEntry[]>([]);
  let showUndoToast = $state(false);
  let undoTimeoutId: ReturnType<typeof setTimeout> | null = null;

  // ── Derived ──────────────────────────────────────────────────────────────────
  const addedUniqueTypes = $derived(
    new Set(
      widgets
        .filter((w) => uniqueWidgetTypes.includes(w.type))
        .map((w) => w.type),
    ),
  );

  // ── Edit mode ────────────────────────────────────────────────────────────────

  async function toggleEditMode(): Promise<void> {
    isEditMode = !isEditMode;
    if (!isEditMode) {
      showWidgetPicker = false;
      try {
        // Backend pending: saveUserPreferences API not yet implemented.
        // When the endpoint is ready, replace this with the real call:
        // await api.saveUserPreferences({ dashboard_layout: { widgets: ... } });
        if (import.meta.env.DEV) {
          console.log(
            "[dashboard] layout persisted locally (backend endpoint pending)",
            widgets.map((w) => ({
              id: w.id,
              type: w.type,
              title: w.title,
              size: w.size,
            })),
          );
        }
      } catch (err) {
        console.error("Failed to save dashboard layout:", err);
      }
    }
  }

  // ── Widget picker ────────────────────────────────────────────────────────────

  function canAddWidget(type: WidgetType): boolean {
    if (uniqueWidgetTypes.includes(type)) {
      return !addedUniqueTypes.has(type);
    }
    return true;
  }

  function addWidget(type: WidgetType): void {
    if (!canAddWidget(type)) return;

    const template = availableWidgets.find((w) => w.type === type);
    if (!template) return;

    const existingOfType = widgets.filter((w) => w.type === type).length;
    const title =
      configurableWidgetTypes.includes(type) && existingOfType > 0
        ? `${template.title} ${existingOfType + 1}`
        : template.title;

    const newWidget: Widget = {
      id: `w${Date.now()}`,
      type,
      title,
      size: pickerSelectedSize,
      collapsed: false,
      accentColor: "",
    };

    widgets = [...widgets, newWidget];
    showWidgetPicker = false;
    pickerSelectedSize = "medium";
  }

  // ── Widget mutations ─────────────────────────────────────────────────────────

  function removeWidget(id: string): void {
    const index = widgets.findIndex((w) => w.id === id);
    if (index === -1) return;

    const removedWidget = widgets[index];
    undoStack = [
      ...undoStack,
      { widget: removedWidget, index, timestamp: Date.now() },
    ];
    widgets = widgets.filter((w) => w.id !== id);
    showUndoToast = true;

    if (undoTimeoutId) clearTimeout(undoTimeoutId);
    undoTimeoutId = setTimeout(() => {
      showUndoToast = false;
      undoStack = undoStack.filter(
        (item) => Date.now() - item.timestamp < 5000,
      );
    }, 5000);
  }

  function undoRemove(): void {
    if (undoStack.length === 0) return;

    const lastRemoved = undoStack[undoStack.length - 1];
    undoStack = undoStack.slice(0, -1);

    const newWidgets = [...widgets];
    newWidgets.splice(lastRemoved.index, 0, lastRemoved.widget);
    widgets = newWidgets;

    if (undoStack.length === 0) {
      showUndoToast = false;
      if (undoTimeoutId) clearTimeout(undoTimeoutId);
    }
  }

  function toggleWidgetCollapse(id: string): void {
    widgets = widgets.map((w) =>
      w.id === id ? { ...w, collapsed: !w.collapsed } : w,
    );
  }

  function toggleWidgetAnalytics(id: string): void {
    widgets = widgets.map((w) =>
      w.id === id ? { ...w, showAnalytics: !w.showAnalytics } : w,
    );
  }

  function setWidgetSize(id: string, size: WidgetSize): void {
    widgets = widgets.map((w) => (w.id === id ? { ...w, size } : w));
  }

  function setWidgetAccentColor(id: string, color: string): void {
    widgets = widgets.map((w) =>
      w.id === id ? { ...w, accentColor: color } : w,
    );
  }

  function moveWidget(fromIndex: number, toIndex: number): void {
    const newWidgets = [...widgets];
    const [moved] = newWidgets.splice(fromIndex, 1);
    newWidgets.splice(toIndex, 0, moved);
    widgets = newWidgets;
  }

  // ── Drag-and-drop ────────────────────────────────────────────────────────────

  function handleDragStart(e: DragEvent, widgetId: string): void {
    draggedWidget = widgetId;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = "move";
    }
  }

  function handleDragOver(e: DragEvent): void {
    e.preventDefault();
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = "move";
    }
  }

  function handleDragEnter(e: DragEvent, targetId: string): void {
    e.preventDefault();
    if (draggedWidget && draggedWidget !== targetId) {
      dragOverWidget = targetId;
    }
  }

  function handleDragLeave(e: DragEvent): void {
    const relatedTarget = e.relatedTarget as Node | null;
    const currentTarget = e.currentTarget as Node | null;
    if (currentTarget && relatedTarget && currentTarget.contains(relatedTarget))
      return;
    dragOverWidget = null;
  }

  function handleDrop(e: DragEvent, targetId: string): void {
    e.preventDefault();
    dragOverWidget = null;
    if (!draggedWidget || draggedWidget === targetId) return;

    const fromIndex = widgets.findIndex((w) => w.id === draggedWidget);
    const toIndex = widgets.findIndex((w) => w.id === targetId);
    if (fromIndex !== -1 && toIndex !== -1) {
      moveWidget(fromIndex, toIndex);
    }
    draggedWidget = null;
  }

  function handleDragEnd(): void {
    draggedWidget = null;
    dragOverWidget = null;
  }

  // ── Keyboard ─────────────────────────────────────────────────────────────────

  function handleKeydown(e: KeyboardEvent): void {
    if (
      e.target instanceof HTMLInputElement ||
      e.target instanceof HTMLTextAreaElement
    )
      return;

    if (e.key === "e" || e.key === "E") {
      e.preventDefault();
      toggleEditMode();
    }

    if (e.key === "Escape" && isEditMode) {
      e.preventDefault();
      isEditMode = false;
      showWidgetPicker = false;
      selectedWidgetIndex = -1;
    }

    if (isEditMode && widgets.length > 0) {
      if (e.key === "ArrowRight" || e.key === "ArrowDown") {
        e.preventDefault();
        selectedWidgetIndex = (selectedWidgetIndex + 1) % widgets.length;
      }
      if (e.key === "ArrowLeft" || e.key === "ArrowUp") {
        e.preventDefault();
        selectedWidgetIndex =
          selectedWidgetIndex <= 0
            ? widgets.length - 1
            : selectedWidgetIndex - 1;
      }
      if (e.key === "Enter" && selectedWidgetIndex >= 0) {
        e.preventDefault();
        toggleWidgetCollapse(widgets[selectedWidgetIndex].id);
      }
      if (e.key === "Delete" || e.key === "Backspace") {
        if (selectedWidgetIndex >= 0) {
          e.preventDefault();
          removeWidget(widgets[selectedWidgetIndex].id);
          selectedWidgetIndex = Math.min(
            selectedWidgetIndex,
            widgets.length - 1,
          );
        }
      }
    }

    if ((e.ctrlKey || e.metaKey) && e.key === "z" && undoStack.length > 0) {
      e.preventDefault();
      undoRemove();
    }
  }

  // ── Public API ────────────────────────────────────────────────────────────────

  return {
    // State getters/setters
    get widgets() {
      return widgets;
    },
    set widgets(v: Widget[]) {
      widgets = v;
    },

    get isEditMode() {
      return isEditMode;
    },
    set isEditMode(v: boolean) {
      isEditMode = v;
    },

    get draggedWidget() {
      return draggedWidget;
    },
    set draggedWidget(v: string | null) {
      draggedWidget = v;
    },

    get dragOverWidget() {
      return dragOverWidget;
    },
    set dragOverWidget(v: string | null) {
      dragOverWidget = v;
    },

    get showWidgetPicker() {
      return showWidgetPicker;
    },
    set showWidgetPicker(v: boolean) {
      showWidgetPicker = v;
    },

    get selectedWidgetIndex() {
      return selectedWidgetIndex;
    },
    set selectedWidgetIndex(v: number) {
      selectedWidgetIndex = v;
    },

    get pickerSelectedSize() {
      return pickerSelectedSize;
    },
    set pickerSelectedSize(v: WidgetSize) {
      pickerSelectedSize = v;
    },

    get undoStack() {
      return undoStack;
    },
    set undoStack(v: UndoEntry[]) {
      undoStack = v;
    },

    get showUndoToast() {
      return showUndoToast;
    },
    set showUndoToast(v: boolean) {
      showUndoToast = v;
    },

    get showRightPanel() {
      return showRightPanel;
    },
    set showRightPanel(v: boolean) {
      showRightPanel = v;
    },

    // Derived
    get addedUniqueTypes() {
      return addedUniqueTypes;
    },

    // Methods
    toggleEditMode,
    canAddWidget,
    addWidget,
    removeWidget,
    undoRemove,
    toggleWidgetCollapse,
    toggleWidgetAnalytics,
    setWidgetSize,
    setWidgetAccentColor,
    moveWidget,
    handleDragStart,
    handleDragOver,
    handleDragEnter,
    handleDragLeave,
    handleDrop,
    handleDragEnd,
    handleKeydown,
  };
}

export const dashboardLayoutStore = createDashboardLayoutStore();
