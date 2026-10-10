/** GET /api/founderos/pages/clients (internal/founderos/api/page_clients.go). */
import type { Meter, SeriesPoint, StatChip, Tone } from "$lib/founderos/kit";

export type WorkStatus = "saved" | "launching" | "launched" | "needs_attention";

export type ClientProject = {
  id: string;
  name: string;
  service: string;
  projectName: string;
  projectId: string;
  context: string;
  sources: string[];
  retainer: string;
};

export type ClientWork = {
  id: string;
  clientId: string;
  brief: string;
  status: WorkStatus;
  createdAt: string;
  workspaceId: string | null;
  detail: string | null;
};

export type StatusStep = {
  key: WorkStatus;
  meter: string;
  chip: string;
  tone: Tone;
  hue: string;
};

export type ClientsVolume = {
  headline: number;
  counts: Record<WorkStatus, number>;
  chips: StatChip[];
  caption: string;
  meters: Meter[];
  foot: string;
  series: SeriesPoint[];
  requestsInWindow: number;
  rhythm: SeriesPoint[];
  perClient: Array<{ id: string; count: number }>;
  insight: { value: number; headline: string; body: string; frac: number };
};

export type ClientsPayload = {
  clients: ClientProject[];
  work: ClientWork[];
  volume: ClientsVolume;
  windowDays: number;
  statusOrder: StatusStep[];
  slackBridge: string;
  launchEnabled: boolean;
};

/** Hue for the request line and rhythm dots (FounderOS v1 --send-activity). */
export const ACTIVITY_HUE =
  "color-mix(in oklab, var(--bn-accent) 70%, var(--bn-text))";

/** The busiest weekday of the rhythm, first on a tie (page.tsx reduce). */
export function busiestDay(rhythm: SeriesPoint[]): SeriesPoint | null {
  if (rhythm.length === 0) return null;
  return rhythm.reduce(
    (best, d) => (d.count > best.count ? d : best),
    rhythm[0],
  );
}

/** A status pill's colors: color means status only. */
export function statusStyle(order: StatusStep[], status: WorkStatus): string {
  const hue = order.find((s) => s.key === status)?.hue ?? "var(--bn-text-2)";
  return `background: color-mix(in oklab, ${hue} 16%, transparent); color: ${hue}`;
}
