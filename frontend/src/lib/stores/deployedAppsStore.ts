// Deployed Apps Store - Manages OSA-generated apps running on localhost
import { writable } from "svelte/store";
import { windowStore } from "./windowStore";
import { browser } from "$app/environment";
import { getApiBaseUrl } from "$lib/api/base";
import type { PaginationMeta } from "$lib/api/base";

export interface AppMetadata {
  name: string;
  description: string;
  category: string;
  icon: string;
  keywords: string[];
}

// Map categories to Lucide icon names for consistent display
export const categoryIconMap: Record<string, string> = {
  finance: "DollarSign",
  communication: "MessageSquare",
  productivity: "Calendar",
  analytics: "BarChart",
  ecommerce: "ShoppingCart",
  crm: "Users",
  hr: "UserCheck",
  inventory: "Package",
  marketing: "Megaphone",
  project: "FolderKanban",
  general: "AppWindow",
};

// Map categories to hex colors for consistent styling
export const categoryColorMap: Record<
  string,
  { fg: string; bg: string; text: string }
> = {
  finance: {
    fg: "#10b981",
    bg: "rgba(16, 185, 129, 0.1)",
    text: "text-green-400",
  },
  communication: {
    fg: "#3b82f6",
    bg: "rgba(59, 130, 246, 0.1)",
    text: "text-blue-400",
  },
  productivity: {
    fg: "#a855f7",
    bg: "rgba(168, 85, 247, 0.1)",
    text: "text-purple-400",
  },
  analytics: {
    fg: "#f97316",
    bg: "rgba(249, 115, 22, 0.1)",
    text: "text-orange-400",
  },
  ecommerce: {
    fg: "#ec4899",
    bg: "rgba(236, 72, 153, 0.1)",
    text: "text-pink-400",
  },
  crm: { fg: "#06b6d4", bg: "rgba(6, 182, 212, 0.1)", text: "text-cyan-400" },
  hr: { fg: "#6366f1", bg: "rgba(99, 102, 241, 0.1)", text: "text-indigo-400" },
  inventory: {
    fg: "#f59e0b",
    bg: "rgba(245, 158, 11, 0.1)",
    text: "text-amber-400",
  },
  marketing: {
    fg: "#f43f5e",
    bg: "rgba(244, 63, 94, 0.1)",
    text: "text-rose-400",
  },
  project: {
    fg: "#14b8a6",
    bg: "rgba(20, 184, 166, 0.1)",
    text: "text-teal-400",
  },
  general: {
    fg: "#6b7280",
    bg: "rgba(107, 114, 128, 0.1)",
    text: "text-gray-400",
  },
};

// Helper function to get icon name for a category
export function getCategoryIconName(category: string): string {
  return categoryIconMap[category?.toLowerCase()] || categoryIconMap.general;
}

// Helper function to get colors for a category
export function getCategoryColors(category: string): {
  fg: string;
  bg: string;
  text: string;
} {
  return categoryColorMap[category?.toLowerCase()] || categoryColorMap.general;
}

export interface DeployedApp {
  id: string;
  name: string;
  url: string;
  port: number;
  status: "running" | "stopped" | "crashed";
  deployedAt?: string;
  metadata?: AppMetadata;
}

export const DEPLOYED_APPS_PAGE_SIZE = 20;

interface DeployedAppsStore {
  apps: DeployedApp[];
  loading: boolean;
  error: string | null;
  pagination: PaginationMeta;
  currentPage: number;
  pageSize: number;
}

const initialPagination: PaginationMeta = {
  page: 1,
  page_size: DEPLOYED_APPS_PAGE_SIZE,
  total_items: 0,
  total_pages: 0,
  has_more: false,
};

const initialState: DeployedAppsStore = {
  apps: [],
  loading: false,
  error: null,
  pagination: initialPagination,
  currentPage: 1,
  pageSize: DEPLOYED_APPS_PAGE_SIZE,
};

function createDeployedAppsStore() {
  const { subscribe, set, update } = writable<DeployedAppsStore>(initialState);
  let pollInterval: NodeJS.Timeout | null = null;
  let isDiscovering = false;

  async function fetchDeployedApps(
    page: number = 1,
    pageSize: number = DEPLOYED_APPS_PAGE_SIZE,
  ): Promise<{ apps: DeployedApp[]; pagination: PaginationMeta }> {
    if (!browser) {
      return {
        apps: [],
        pagination: { ...initialPagination },
      };
    }

    try {
      const qs = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
      });
      // Use correct backend endpoint: /api/v1/osa/deployments (not /apps/deployed)
      const response = await fetch(
        `${getApiBaseUrl()}/osa/deployments?${qs.toString()}`,
        { credentials: "include" },
      );

      if (!response.ok) {
        throw new Error(
          `Failed to fetch deployed apps: ${response.statusText}`,
        );
      }

      const data = await response.json();

      // Handle both paginated and legacy response shapes
      if (data.pagination && Array.isArray(data.data)) {
        return {
          apps: data.data as DeployedApp[],
          pagination: data.pagination as PaginationMeta,
        };
      }

      // Legacy shape: { apps: [...] }
      const apps: DeployedApp[] = data.apps || [];
      return {
        apps,
        pagination: {
          page: 1,
          page_size: apps.length,
          total_items: apps.length,
          total_pages: 1,
          has_more: false,
        },
      };
    } catch (err) {
      console.error("Error fetching deployed apps:", err);
      throw err;
    }
  }

  async function refresh(page?: number, pageSize?: number) {
    update((state) => ({ ...state, loading: true, error: null }));

    try {
      // Snapshot current pagination if no explicit params
      let resolvedPage = page;
      let resolvedPageSize = pageSize;
      if (resolvedPage === undefined || resolvedPageSize === undefined) {
        const unsubscribe = subscribe((s) => {
          if (resolvedPage === undefined) resolvedPage = s.currentPage;
          if (resolvedPageSize === undefined) resolvedPageSize = s.pageSize;
        });
        unsubscribe();
      }

      const { apps, pagination } = await fetchDeployedApps(
        resolvedPage ?? 1,
        resolvedPageSize ?? DEPLOYED_APPS_PAGE_SIZE,
      );

      update((state) => {
        // Register new apps in windowStore with category-based icons
        for (const app of apps) {
          if (app.status === "running") {
            const category = app.metadata?.category || "general";
            const iconName =
              app.metadata?.icon || getCategoryIconName(category);

            const enhancedApp = {
              ...app,
              metadata: app.metadata
                ? { ...app.metadata, icon: iconName }
                : {
                    name: app.name,
                    description: "",
                    category: "general",
                    icon: iconName,
                    keywords: [],
                  },
            };

            windowStore.registerDeployedApp(enhancedApp);
          }
        }

        return {
          ...state,
          apps,
          pagination,
          currentPage: pagination.page,
          pageSize: pagination.page_size,
          loading: false,
          error: null,
        };
      });
    } catch (err) {
      update((state) => ({
        ...state,
        loading: false,
        error: err instanceof Error ? err.message : "Unknown error",
      }));
    }
  }

  return {
    subscribe,

    // Start polling for deployed apps
    startDiscovery: async () => {
      if (isDiscovering) {
        return;
      }

      isDiscovering = true;

      // Initial fetch
      await refresh();

      // Poll every 10 seconds
      pollInterval = setInterval(refresh, 10000);
    },

    // Stop polling
    stopDiscovery: () => {
      if (pollInterval) {
        clearInterval(pollInterval);
        pollInterval = null;
      }
      isDiscovering = false;
    },

    // Manual refresh
    refresh,

    // Paginated navigation
    goToPage: async (page: number) => {
      let currentPageSize = DEPLOYED_APPS_PAGE_SIZE;
      const unsubscribe = subscribe((s) => {
        currentPageSize = s.pageSize;
      });
      unsubscribe();
      await refresh(page, currentPageSize);
    },

    setPageSize: async (pageSize: number) => {
      // Reset to page 1 when page size changes
      await refresh(1, pageSize);
    },

    // Get app by ID
    getApp: (appId: string): DeployedApp | undefined => {
      let result: DeployedApp | undefined;
      const unsubscribe = subscribe((state) => {
        result = state.apps.find((app) => app.id === appId);
      });
      unsubscribe();
      return result;
    },
  };
}

export const deployedAppsStore = createDeployedAppsStore();
