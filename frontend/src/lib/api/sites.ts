// Sites API client - the per-workspace registry of the business's web
// properties (name + url + status). Goes through request<T>() which handles the
// API base, session cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

export type SiteStatus = "live" | "draft" | "building" | "archived";
// What kind of web property this is (funnels/pages are the common case).
export type SiteKind = "funnel" | "page" | "form" | "site" | "app";

export interface Site {
  id: string;
  name: string;
  kind: SiteKind;
  url: string;
  status: string;
  cta: string;
  notes: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface SiteInput {
  name: string;
  kind: SiteKind;
  url: string;
  status: string;
  cta: string;
  notes: string;
}

export async function getSites(
  q?: string,
): Promise<{ sites: Site[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ sites: Site[]; count: number }>(`/sites${query}`, {
    skipCache: true,
  });
}

export function createSite(body: SiteInput): Promise<Site> {
  return request<Site>(`/sites`, { method: "POST", body });
}

export function updateSite(id: string, body: SiteInput): Promise<Site> {
  return request<Site>(`/sites/${id}`, { method: "PUT", body });
}

export function deleteSite(id: string): Promise<unknown> {
  return request(`/sites/${id}`, { method: "DELETE" });
}
