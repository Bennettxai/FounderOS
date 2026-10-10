/**
 * SVG path data for each desktop module icon.
 * Each entry provides the SVG path, stroke color, and background color.
 */
export interface IconData {
  path: string;
  color: string;
  bgColor: string;
}

export const iconPaths: Record<string, IconData> = {
  platform: {
    path: "M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5",
    color: "#333333",
    bgColor: "#F5F5F5",
  },
  terminal: {
    path: "M4 17l6-6-6-6M12 19h8",
    color: "#1E1E1E",
    bgColor: "#2D2D2D",
  },
  dashboard: {
    path: "M4 5a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1H5a1 1 0 01-1-1V5zm10 0a1 1 0 011-1h4a1 1 0 011 1v2a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zm0 6a1 1 0 011-1h4a1 1 0 011 1v5a1 1 0 01-1 1h-4a1 1 0 01-1-1v-5zm-10 1a1 1 0 011-1h4a1 1 0 011 1v3a1 1 0 01-1 1H5a1 1 0 01-1-1v-3z",
    color: "#1E88E5",
    bgColor: "#E3F2FD",
  },
  agents: {
    path: "M12 3l2.4 4.86L20 8.68l-4 3.9.94 5.52L12 15.5l-4.94 2.6L8 12.58l-4-3.9 5.6-.82L12 3z",
    color: "#7C3AED",
    bgColor: "#F5F3FF",
  },
  knowledge: {
    path: "M4 19.5A2.5 2.5 0 016.5 17H20M4 4.5A2.5 2.5 0 016.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15zM8 6h8M8 10h8M8 14h5",
    color: "#2563EB",
    bgColor: "#EFF6FF",
  },
  intelligence: {
    path: "M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z",
    color: "#9333EA",
    bgColor: "#FAF5FF",
  },
  inbox: {
    path: "M3 13h4l2 3h6l2-3h4M5 19h14a2 2 0 002-2v-4L18 5H6l-3 8v4a2 2 0 002 2z",
    color: "#0F766E",
    bgColor: "#F0FDFA",
  },
  chat: {
    path: "M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z",
    color: "#43A047",
    bgColor: "#E8F5E9",
  },
  tasks: {
    path: "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4",
    color: "#FB8C00",
    bgColor: "#FFF3E0",
  },
  projects: {
    path: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z",
    color: "#8E24AA",
    bgColor: "#F3E5F5",
  },
  team: {
    path: "M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z",
    color: "#00ACC1",
    bgColor: "#E0F7FA",
  },
  clients: {
    path: "M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4",
    color: "#7B1FA2",
    bgColor: "#F3E5F5",
  },
  relationships: {
    path: "M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z",
    color: "#0891B2",
    bgColor: "#ECFEFF",
  },
  rhythm: {
    path: "M4 12h4l2-6 4 12 2-6h4",
    color: "#DB2777",
    bgColor: "#FDF2F8",
  },
  pipelines: {
    path: "M4 6h5v5H4V6zm11 0h5v5h-5V6zM4 15h5v5H4v-5zm11 0h5v5h-5v-5zM9 8.5h6M9 17.5h6M6.5 11v4M17.5 11v4",
    color: "#CA8A04",
    bgColor: "#FEFCE8",
  },
  offers: {
    path: "M20 12v7a2 2 0 01-2 2H6a2 2 0 01-2-2v-7M12 21V8M12 8H8.5a2.5 2.5 0 110-5C12 3 12 8 12 8zm0 0h3.5a2.5 2.5 0 100-5C12 3 12 8 12 8zM2 8h20v4H2V8z",
    color: "#EA580C",
    bgColor: "#FFF7ED",
  },
  campaigns: {
    path: "M3 11l18-5v12L3 13v-2zm4 3v4a2 2 0 002 2h2",
    color: "#DC2626",
    bgColor: "#FEF2F2",
  },
  sites: {
    path: "M12 21a9 9 0 100-18 9 9 0 000 18zM3.6 9h16.8M3.6 15h16.8M12 3a15 15 0 010 18M12 3a15 15 0 000 18",
    color: "#0284C7",
    bgColor: "#F0F9FF",
  },
  personas: {
    path: "M12 12a4 4 0 100-8 4 4 0 000 8zm-7 9a7 7 0 0114 0",
    color: "#C026D3",
    bgColor: "#FDF4FF",
  },
  content: {
    path: "M4 4h16v16H4V4zm4 4h8M8 12h8M8 16h5",
    color: "#7C2D12",
    bgColor: "#FFF7ED",
  },
  apps: {
    path: "M4 4h6v6H4V4zm10 0h6v6h-6V4zM4 14h6v6H4v-6zm10 0h6v6h-6v-6z",
    color: "#4F46E5",
    bgColor: "#EEF2FF",
  },
  assets: {
    path: "M4 7a2 2 0 012-2h4l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H6a2 2 0 01-2-2V7zm4 8l2.5-3 2 2.5L15 11l3 4H8z",
    color: "#16A34A",
    bgColor: "#F0FDF4",
  },
  deliverables: {
    path: "M9 12l2 2 4-4M5 4h14v16H5V4zm4 4h6",
    color: "#059669",
    bgColor: "#ECFDF5",
  },
  engines: {
    path: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
    color: "#475569",
    bgColor: "#F8FAFC",
  },
  builders: {
    path: "M14.7 6.3a1 1 0 000 1.4l1.6 1.6a1 1 0 001.4 0l2.1-2.1a4 4 0 01-5.1 5.1L8 19l-4 1 1-4 6.7-6.7a4 4 0 015.1-5.1l-2.1 2.1z",
    color: "#0D9488",
    bgColor: "#F0FDFA",
  },
  drive: {
    path: "M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z",
    color: "#2563EB",
    bgColor: "#EFF6FF",
  },
  finance: {
    path: "M12 2v20M17 5H9.5a3.5 3.5 0 000 7H14a3.5 3.5 0 010 7H6",
    color: "#16A34A",
    bgColor: "#F0FDF4",
  },
  analytics: {
    path: "M4 19V5M8 19v-6M12 19V9M16 19v-9M20 19V7",
    color: "#7C3AED",
    bgColor: "#F5F3FF",
  },
  data: {
    path: "M12 3c4.418 0 8 1.343 8 3s-3.582 3-8 3-8-1.343-8-3 3.582-3 8-3zm8 6c0 1.657-3.582 3-8 3s-8-1.343-8-3m16 3c0 1.657-3.582 3-8 3s-8-1.343-8-3m16 3c0 1.657-3.582 3-8 3s-8-1.343-8-3",
    color: "#0F766E",
    bgColor: "#F0FDFA",
  },
  contexts: {
    path: "M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10",
    color: "#5E35B1",
    bgColor: "#EDE7F6",
  },
  nodes: {
    path: "M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z",
    color: "#E53935",
    bgColor: "#FFEBEE",
  },
  daily: {
    path: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    color: "#039BE5",
    bgColor: "#E1F5FE",
  },
  settings: {
    path: "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z",
    color: "#546E7A",
    bgColor: "#ECEFF1",
  },
  calendar: {
    path: "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z",
    color: "#E53935",
    bgColor: "#FFEBEE",
  },
  "ai-settings": {
    path: "M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z",
    color: "#9C27B0",
    bgColor: "#F3E5F5",
  },
  tables: {
    path: "M3 5a2 2 0 012-2h14a2 2 0 012 2v14a2 2 0 01-2 2H5a2 2 0 01-2-2V5zm0 4h18M3 13h18M9 3v18M15 3v18",
    color: "#0D9488",
    bgColor: "#F0FDFA",
  },
  communication: {
    path: "M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z",
    color: "#2563EB",
    bgColor: "#EFF6FF",
  },
  files: {
    path: "M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8l-6-6zm-1 1v5h5M9 13h6M9 17h6M9 9h1",
    color: "#64748B",
    bgColor: "#F1F5F9",
  },
  pages: {
    path: "M4 4h16v16H4V4zm2 4h12M6 12h8M6 16h10",
    color: "#7C3AED",
    bgColor: "#F5F3FF",
  },
  integrations: {
    path: "M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1",
    color: "#D97706",
    bgColor: "#FFFBEB",
  },
  connectors: {
    path: "M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1",
    color: "#D97706",
    bgColor: "#FFFBEB",
  },
  resources: {
    path: "M4 19.5A2.5 2.5 0 016.5 17H20M4 4.5A2.5 2.5 0 016.5 2H20v20H6.5A2.5 2.5 0 014 19.5v-15zM8 7h8M8 11h8M8 15h5",
    color: "#64748B",
    bgColor: "#F8FAFC",
  },
  notifications: {
    path: "M15 17h5l-1.4-1.4A2 2 0 0118 14.2V11a6 6 0 10-12 0v3.2a2 2 0 01-.6 1.4L4 17h5m6 0a3 3 0 01-6 0",
    color: "#2563EB",
    bgColor: "#EFF6FF",
  },
  admin: {
    path: "M12 3l7 4v5c0 4.5-3 7.5-7 9-4-1.5-7-4.5-7-9V7l7-4zm0 5v5l3 2",
    color: "#334155",
    bgColor: "#F1F5F9",
  },
  crm: {
    path: "M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z",
    color: "#059669",
    bgColor: "#ECFDF5",
  },
  trash: {
    path: "M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16",
    color: "#78909C",
    bgColor: "#ECEFF1",
  },
  folder: {
    path: "M3 7V17C3 18.1046 3.89543 19 5 19H19C20.1046 19 21 18.1046 21 17V9C21 7.89543 20.1046 7 19 7H12L10 5H5C3.89543 5 3 5.89543 3 7Z",
    color: "#3B82F6",
    bgColor: "#EFF6FF",
  },
};
