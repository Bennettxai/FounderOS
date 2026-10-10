// Shared project utility functions

export function getStatusColor(status: string): string {
  switch (status) {
    case "active":
      return "prm-status prm-status--active";
    case "paused":
      return "prm-status prm-status--paused";
    case "completed":
      return "prm-status prm-status--completed";
    case "archived":
      return "prm-status prm-status--archived";
    default:
      return "prm-status prm-status--archived";
  }
}

export function getStatusIcon(status: string): string {
  switch (status) {
    case "active":
      return "M13 10V3L4 14h7v7l9-11h-7z";
    case "paused":
      return "M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z";
    case "completed":
      return "M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z";
    default:
      return "M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4";
  }
}

export function getPriorityColor(priority: string): string {
  switch (priority) {
    case "critical":
      return "prm-priority prm-priority--critical";
    case "high":
      return "prm-priority prm-priority--high";
    case "medium":
      return "prm-priority prm-priority--medium";
    case "low":
      return "prm-priority prm-priority--low";
    default:
      return "prm-priority prm-priority--default";
  }
}

export function getTypeLabel(type: string): string {
  switch (type) {
    case "internal":
      return "Internal";
    case "client_work":
      return "Client Work";
    case "learning":
      return "Learning";
    default:
      return type;
  }
}

/** @deprecated Use Lucide icons (Building2, Users, BookOpen, FolderOpen) directly in templates */
export function getTypeIcon(_type: string): string {
  // Emojis removed per design system. Use Lucide Svelte icons in templates instead.
  return "";
}

export function getPriorityDotVar(priority: string): string {
  switch (priority) {
    case "critical":
      return "var(--bos-priority-critical)";
    case "high":
      return "var(--bos-priority-high)";
    case "medium":
      return "var(--bos-priority-medium)";
    case "low":
      return "var(--bos-priority-low)";
    default:
      return "var(--bos-status-neutral)";
  }
}

export function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString(undefined, {
    hour: "numeric",
    minute: "2-digit",
  });
}
