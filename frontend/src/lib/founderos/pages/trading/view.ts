/**
 * The interactive half of FounderOS v1 lib/trading-view.ts and
 * lib/trading-chart.ts: what the board recomputes on a click (the account
 * graph, the trade-log filter). Everything fixed per payload (freshness, the
 * Accounts card, position sizes, the agent summary) is computed by the
 * backend and arrives in payload.view.
 */
import type { TradeActivity, TradingAccountSnapshot } from './types';

const round = (n: number) => Math.round(n * 100) / 100;

export const usd = (n: number) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 });
export const usd0 = (n: number) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 });

/** Signed money with a +/- and a status tone. */
export function pnl(n: number): { text: string; tone: 'ok' | 'err' | 'muted' } {
	const sign = n > 0 ? '+' : n < 0 ? '-' : '';
	return { text: `${sign}${usd(Math.abs(n))}`, tone: n > 0 ? 'ok' : n < 0 ? 'err' : 'muted' };
}

export function timeAgo(iso: string, now: number): string {
	const m = Math.max(0, Math.round((now - new Date(iso).getTime()) / 60000));
	if (m < 60) return `${m}m`;
	const h = Math.round(m / 60);
	return h < 24 ? `${h}h` : `${Math.round(h / 24)}d`;
}

export const clock = (iso: string) => new Date(iso).toLocaleString('en-US', { month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit' });

export type ActivityFilter = 'all' | 'agent' | 'you' | 'rejected';

const isAgent = (a: TradeActivity) => /agent/i.test(a.agent) && !/founderos/i.test(a.agent);
export const placedByAgent = isAgent;

/** Who did it and how it ended: "you" is anything a human placed, "agent" is
 *  the Markets Agent whatever the outcome, "rejected" never reached a fill. */
export function filterActivity(rows: TradeActivity[], filter: ActivityFilter): TradeActivity[] {
	switch (filter) {
		case 'agent':
			return rows.filter(isAgent);
		case 'you':
			return rows.filter((a) => !isAgent(a));
		case 'rejected':
			return rows.filter((a) => a.status === 'rejected');
		default:
			return rows;
	}
}

export function activityCounts(rows: TradeActivity[]): Record<ActivityFilter, number> {
	return {
		all: rows.length,
		agent: filterActivity(rows, 'agent').length,
		you: filterActivity(rows, 'you').length,
		rejected: filterActivity(rows, 'rejected').length
	};
}

export function statusTone(status: TradeActivity['status']): 'ok' | 'warn' | 'err' | 'default' {
	if (status === 'filled') return 'ok';
	if (status === 'pending') return 'warn';
	if (status === 'rejected' || status === 'cancelled') return 'err';
	return 'default';
}

/** The chip order the strategy thinks in; anything else lands after these,
 *  alphabetically, rather than being hidden. */
const VERDICT_ORDER = ['signal', 'watch', 'dropped'];

export function sortVerdicts(verdicts: string[]): string[] {
	return [...new Set(verdicts)].sort((a, b) => {
		const ia = VERDICT_ORDER.indexOf(a);
		const ib = VERDICT_ORDER.indexOf(b);
		if (ia !== -1 || ib !== -1) return (ia === -1 ? 99 : ia) - (ib === -1 ? 99 : ib);
		return a.localeCompare(b);
	});
}

export type ChartPoint = { x: number; y: number; at: string; valueUsd: number };
export type ChartMarker = { x: number; y: number; trade: TradeActivity };
export type ChartGeometry = {
	points: ChartPoint[];
	markers: ChartMarker[];
	line: string;
	area: string;
	firstUsd: number;
	lastUsd: number;
	changeUsd: number;
	changePct: number;
	minUsd: number;
	maxUsd: number;
};

/**
 * One account's value over time with the agent's trades marked on the line.
 * x is spread evenly across samples (the feed pushes on its own cadence).
 */
export function chartGeometry(
	history: TradingAccountSnapshot[],
	trades: TradeActivity[],
	{ w, h, pad = 4 }: { w: number; h: number; pad?: number }
): ChartGeometry | null {
	if (history.length < 2) return null;
	const series = [...history].sort((a, b) => a.capturedAt.localeCompare(b.capturedAt));
	const values = series.map((s) => s.accountValueUsd);
	const minUsd = Math.min(...values);
	const maxUsd = Math.max(...values);
	const span = maxUsd - minUsd;
	const usable = Math.max(0, h - pad * 2);
	const points: ChartPoint[] = series.map((s, i) => ({
		x: round((i / (series.length - 1)) * w),
		// a flat series has no range: park it on the centre line
		y: round(span === 0 ? h / 2 : pad + (1 - (s.accountValueUsd - minUsd) / span) * usable),
		at: s.capturedAt,
		valueUsd: s.accountValueUsd
	}));
	const firstUsd = values[0];
	const lastUsd = values[values.length - 1];
	const changeUsd = round(lastUsd - firstUsd);
	const from = series[0].capturedAt;
	const to = series[series.length - 1].capturedAt;
	const markers: ChartMarker[] = trades
		.filter((t) => t.at >= from && t.at <= to)
		.map((t) => {
			const nearest = points.reduce((best, p) =>
				Math.abs(Date.parse(p.at) - Date.parse(t.at)) < Math.abs(Date.parse(best.at) - Date.parse(t.at)) ? p : best
			);
			return { x: nearest.x, y: nearest.y, trade: t };
		});
	const line = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ');
	return {
		points,
		markers,
		line,
		area: `${line} L${w},${h} L0,${h} Z`,
		firstUsd,
		lastUsd,
		changeUsd,
		changePct: firstUsd === 0 ? 0 : round((changeUsd / firstUsd) * 100),
		minUsd,
		maxUsd
	};
}
