// Rhythm API client - the per-workspace operating rhythm (daily/weekly/monthly
// focuses, blockers, priorities, notes). Goes through request<T>() which handles
// the API base, session cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

export type RhythmPeriod = "daily" | "weekly" | "monthly";
export type RhythmKind = "focus" | "blocker" | "priority" | "note";

export interface RhythmEntry {
  id: string;
  period: RhythmPeriod;
  kind: RhythmKind;
  content: string;
  owner: string;
  entry_date: string | null;
  position: number;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface RhythmEntryInput {
  period: RhythmPeriod;
  kind: RhythmKind;
  content: string;
  owner: string;
  entry_date?: string | null;
  position?: number;
}

export async function getRhythm(
  period?: RhythmPeriod,
): Promise<{ entries: RhythmEntry[]; count: number }> {
  const query = period ? `?period=${encodeURIComponent(period)}` : "";
  return request<{ entries: RhythmEntry[]; count: number }>(`/rhythm${query}`, {
    skipCache: true,
  });
}

export function createRhythmEntry(
  body: RhythmEntryInput,
): Promise<RhythmEntry> {
  return request<RhythmEntry>(`/rhythm`, { method: "POST", body });
}

export function updateRhythmEntry(
  id: string,
  body: RhythmEntryInput,
): Promise<RhythmEntry> {
  return request<RhythmEntry>(`/rhythm/${id}`, { method: "PUT", body });
}

export function deleteRhythmEntry(id: string): Promise<unknown> {
  return request(`/rhythm/${id}`, { method: "DELETE" });
}
