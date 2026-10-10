// Manage > Data API client — a read-only inventory of every workspace-scoped
// data entity and its record count. Goes through request<T>() which handles the
// API base, session cookie, CSRF, and the X-Workspace-ID header.
//
// Mounted under /manage-data on the backend to avoid colliding with any other
// /data route group.
import { request } from "./base";

export interface DataEntity {
  key: string;
  label: string;
  table: string;
  count: number;
}

export interface DataSummary {
  entities: DataEntity[];
  total: number;
  export_ready: boolean;
}

export async function getDataSummary(): Promise<DataSummary> {
  return request<DataSummary>(`/manage-data/summary`);
}
