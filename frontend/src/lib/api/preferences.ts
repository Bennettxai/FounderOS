// Workspace preferences API client - the business-level settings primitive
// (time zone, date/time format, week start, working hours) that every module
// reads, plus a calendar-specific config block. Goes through request<T>()
// which handles the API base, session cookie, CSRF, and X-Workspace-ID header.
import { request } from "./base";

export interface WorkspacePreferences {
  workspace_id: string;
  timezone: string;
  date_format: string;
  time_format: "12h" | "24h";
  week_start: number; // 0=Sunday, 1=Monday
  working_hours_start: number;
  working_hours_end: number;
  default_event_minutes: number;
  language: string;
  calendar: Record<string, unknown>;
  updated_at: string;
}

export type WorkspacePreferencesInput = Partial<
  Omit<WorkspacePreferences, "workspace_id" | "updated_at">
>;

export function getPreferences(): Promise<WorkspacePreferences> {
  return request<WorkspacePreferences>(`/workspace/preferences`, {
    skipCache: true,
  });
}

export function updatePreferences(
  body: WorkspacePreferencesInput,
): Promise<WorkspacePreferences> {
  return request<WorkspacePreferences>(`/workspace/preferences`, {
    method: "PUT",
    body,
  });
}
