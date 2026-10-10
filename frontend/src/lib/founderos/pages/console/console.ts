/**
 * GET /api/founderos/pages/console — the operator console (FounderOS v1 app/page.tsx),
 * composed server side by internal/founderos/pages/console. Null is unknown:
 * the source could not be read, and the page says so instead of drawing 0.
 */
import type { Meter, SeriesPoint } from '$lib/founderos/kit';

export type SegTone = 'ok' | 'warn' | 'err' | 'accent' | 'dim';
export type Segment = { text: string; tone: SegTone };

export type EngineReading = {
	name: string;
	state: 'connected' | 'error' | 'not_configured' | (string & {});
	up: boolean;
	detail: string;
	workspaces: number | null;
	homes: string[];
	health?: number | null;
	warnings?: boolean;
};

export type Brain = {
	connected: boolean;
	enginesUp: number;
	enginesTotal: number;
	workspaces: number | null;
	/** Where prod stated G-Brain's doctor score: the share of the engines' own
	    health checks passing, 0-100. Null when no engine answers. */
	health: number | null;
	/** The doctor's word: ok | warnings | offline | not configured. */
	status: string;
	engines: EngineReading[];
};

export type SourceState = { source: string; state: 'ok' | 'error' | 'not_configured' | 'stale' | (string & {}); detail?: string };
export type DoneItem = { key: string; head: string; tone: SegTone; body: string; at: string };

export type ConsoleView = {
	generatedAt: string;
	hero: Segment[];
	systems: { connected: number; total: number; bars: string[] };
	agents: { active: number | null; total: number | null; spark: number[] | null };
	comms: { inbound: number; spark: number[] };
	brain: Brain;
	volume: { runsToday: number; failedToday: number; agentsToday: number; meters: Meter[] };
	chargedTodayCents: number | null;
	activity: SeriesPoint[];
	activityTotal: number;
	mix: SeriesPoint[];
	feedCount: number;
	sources: SourceState[];
	attention: { count: number; headline: string; frac: number };
	doneCount: number;
	done: DoneItem[];
	connections: unknown[];
	errors: Record<string, string>;
};

/** Colour means status only (Monolith Signal). */
export const TONE_COLOR: Record<SegTone, string> = {
	ok: 'var(--bn-ok)',
	warn: 'var(--bn-warn)',
	err: 'var(--bn-err)',
	accent: 'var(--bn-accent)',
	dim: 'var(--bn-text-3)'
};

export { operatorName } from '$lib/founderos/operator';

export function greeting(now: Date = new Date()): string {
	const h = now.getHours();
	if (h < 5) return 'Late night';
	if (h < 12) return 'Good morning';
	if (h < 18) return 'Good afternoon';
	return 'Good evening';
}

export function relativeTime(iso: string, now: number = Date.now()): string {
	const ms = now - Date.parse(iso);
	if (!Number.isFinite(ms) || ms < 0) return 'just now';
	const m = Math.floor(ms / 60_000);
	if (m < 1) return 'just now';
	if (m < 60) return `${m}m`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h`;
	return `${Math.floor(h / 24)}d`;
}

/** FounderOS v1 SparkBars: a 3px floor plus 15px of travel, per cell. */
export function sparkBarHeights(data: number[]): number[] {
	const max = Math.max(...data, 1);
	return data.map((v) => 3 + 15 * (v / max));
}

/** Prod HealthMeter: ten fixed cells lit while i < score/10, graded at 70 and 40. */
export function healthCells(score: number | null): { lit: number; tone: 'accent' | 'warn' | 'err' | 'dim' } {
	if (score == null) return { lit: 0, tone: 'dim' };
	const s = Math.max(0, Math.min(100, score));
	return { lit: Math.round(s / 10), tone: s >= 70 ? 'accent' : s >= 40 ? 'warn' : 'err' };
}

/** The fourth pulse tile, prod's "G-Brain health · 90 / 100 · warnings". */
export function brainTile(b: Brain): { value: number | null; unit: string } {
	const status = b.enginesTotal === 0 ? 'not configured' : b.enginesUp === 0 ? 'offline' : b.status || 'unknown';
	return { value: b.enginesUp > 0 ? b.health : null, unit: `/ 100 · ${status}` };
}

export function usd(cents: number): string {
	return (cents / 100).toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });
}
