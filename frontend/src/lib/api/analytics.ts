// Manage > Analytics API client — read-only workspace counts + simple trends
// computed live from the operator's real data. Goes through request<T>() which
// handles the API base, session cookie, CSRF, and the X-Workspace-ID header.
//
// Mounted under /manage-analytics on the backend to avoid colliding with any
// other /analytics route group.
import { request } from "./base";

export interface StatCount {
  label: string;
  count: number;
}

export interface AnalyticsTotals {
  tasks: number;
  projects: number;
  clients: number;
  offers: number;
  content: number;
  campaigns: number;
}

export interface AnalyticsTrends {
  tasks_completed_30d: number;
  tasks_created_30d: number;
  tasks_open: number;
  tasks_overdue: number;
  projects_active: number;
  clients_active: number;
}

export interface AnalyticsSummary {
  totals: AnalyticsTotals;
  tasks_by_status: StatCount[];
  projects_by_status: StatCount[];
  clients_by_status: StatCount[];
  trends: AnalyticsTrends;
}

export async function getAnalyticsSummary(): Promise<AnalyticsSummary> {
  return request<AnalyticsSummary>(`/manage-analytics/summary`);
}
