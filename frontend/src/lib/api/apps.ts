import { request } from "./base";

export type AppType =
  | "web_app"
  | "mini_app"
  | "internal_app"
  | "client_app"
  | "embedded_tool";
export type AppLaunchMode = "iframe" | "browser" | "external";
export type AppStatus = "active" | "draft" | "archived";
export type AppProvider = string;
export type AppUrlClass =
  | "temporary_preview"
  | "always_on_preview"
  | "stable_sandbox_embed"
  | "durable_deployment"
  | "custom_domain";

export interface WorkspaceApp {
  id: string;
  source: "manual" | "generated";
  source_id: string;
  catalog_app_id: string | null;
  name: string;
  app_type: AppType;
  provider: AppProvider;
  url: string;
  launch_mode: AppLaunchMode;
  status: AppStatus;
  icon: string;
  logo_url: string;
  color: string;
  category: string;
  notes: string;
  show_on_desktop: boolean;
  show_in_dock: boolean;
  position_index: number;
  url_class: AppUrlClass;
  read_only: boolean;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface CatalogApp {
  id: string;
  slug: string;
  name: string;
  app_type: AppType;
  provider: AppProvider;
  url: string;
  launch_mode: AppLaunchMode;
  icon: string;
  logo_url: string;
  color: string;
  category: string;
  notes: string;
  url_class: AppUrlClass;
  status: AppStatus;
  is_featured: boolean;
  position_index: number;
  installed: boolean;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceAppInput {
  name: string;
  app_type: AppType;
  provider: AppProvider;
  url: string;
  launch_mode: AppLaunchMode;
  status: AppStatus;
  icon: string;
  logo_url: string;
  color: string;
  category: string;
  notes: string;
  show_on_desktop: boolean;
  show_in_dock: boolean;
  position_index: number;
  url_class: AppUrlClass;
}

export async function getApps(
  q?: string,
  desktopOnly = false,
  workspaceId?: string,
): Promise<{ apps: WorkspaceApp[]; count: number }> {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (desktopOnly) params.set("desktop", "true");
  const query = params.toString() ? `?${params.toString()}` : "";
  return request<{ apps: WorkspaceApp[]; count: number }>(`/apps${query}`, {
    headers: workspaceId ? { "X-Workspace-ID": workspaceId } : undefined,
    skipCache: true,
  });
}

export function createApp(body: WorkspaceAppInput): Promise<WorkspaceApp> {
  return request<WorkspaceApp>("/apps", { method: "POST", body });
}

export function updateApp(
  id: string,
  body: WorkspaceAppInput,
): Promise<WorkspaceApp> {
  return request<WorkspaceApp>(`/apps/${id}`, { method: "PUT", body });
}

export function deleteApp(id: string): Promise<unknown> {
  return request(`/apps/${id}`, { method: "DELETE" });
}

export async function getAppCatalog(
  q?: string,
  workspaceId?: string,
): Promise<{ apps: CatalogApp[]; count: number }> {
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  const query = params.toString() ? `?${params.toString()}` : "";
  return request<{ apps: CatalogApp[]; count: number }>(`/apps/catalog${query}`, {
    headers: workspaceId ? { "X-Workspace-ID": workspaceId } : undefined,
    skipCache: true,
  });
}

export function installCatalogApp(id: string): Promise<WorkspaceApp> {
  return request<WorkspaceApp>(`/apps/catalog/${id}/install`, {
    method: "POST",
  });
}
