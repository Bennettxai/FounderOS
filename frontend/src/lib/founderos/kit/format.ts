/**
 * Pure logic behind the the operator kit, ported from FounderOS v1 components
 * (terminal.tsx, CountUp.tsx, slab.tsx, VolumeMeter.tsx). Kept out of the
 * .svelte files so it is unit-tested directly.
 */

export type DotState = 'ok' | 'warn' | 'err' | 'off';

const DOT_FOR: Record<string, DotState> = {
	connected: 'ok',
	active: 'ok',
	ok: 'ok',
	available: 'warn',
	warn: 'warn',
	training: 'warn',
	idle: 'warn',
	error: 'err',
	fail: 'err',
	not_configured: 'off',
	planned: 'off',
	off: 'off'
};

export function dotState(state: string): DotState {
	return DOT_FOR[state] ?? 'off';
}

/** One frame of a count: ease-out cubic from `from` to `to` at progress t (0..1).
 *  Intermediate frames are whole numbers; the last lands exactly. */
export function countFrame(from: number, to: number, t: number): number {
	if (t >= 1) return to;
	if (t <= 0) return from;
	const eased = 1 - Math.pow(1 - t, 3);
	return Math.round(from + (to - from) * eased);
}

export type CountKind = 'int' | 'usd' | 'usdCents' | 'followers' | 'tokens' | 'pct';

const FORMAT: Record<CountKind, (n: number) => string> = {
	int: (n) => Math.round(n).toLocaleString('en-US'),
	usd: (n) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 }),
	usdCents: (n) => n.toLocaleString('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 }),
	followers: (n) => Math.round(n).toLocaleString('en-US'),
	tokens: (n) =>
		n >= 1e9 ? `${(n / 1e9).toFixed(1)}B` : n >= 1e6 ? `${(n / 1e6).toFixed(1)}M` : n >= 1e3 ? `${(n / 1e3).toFixed(1)}k` : String(Math.round(n)),
	pct: (n) => `${Math.round(n)}%`
};

export function formatCount(kind: CountKind, n: number): string {
	return FORMAT[kind](n);
}

export function prefersReducedMotion(): boolean {
	return typeof window !== 'undefined' && typeof window.matchMedia === 'function' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/** A value is known when it is a finite number. */
export function isKnown(n: number | null | undefined): n is number {
	return typeof n === 'number' && Number.isFinite(n);
}

/** How many of the insight card's 8 ticks a fraction lights. */
export function insightTicks(frac: number | null | undefined): number {
	if (!isKnown(frac) || frac <= 0) return 0;
	return Math.min(8, Math.round(frac * 8));
}

/** The meter's fill fraction: known values clamp to [0.02, 1] (a real zero is
 *  still a sliver, as in FounderOS v1); unknown is null and draws no bar. */
export function meterWidth(frac: number | null | undefined): number | null {
	if (!isKnown(frac)) return null;
	return frac > 0 ? Math.max(0.02, Math.min(1, frac)) : 0.02;
}

/** Spark polyline points (terminal.tsx Spark), or null under two samples. */
export function sparkPoints(data: number[], w: number, h: number): string[] | null {
	if (data.length < 2) return null;
	const min = Math.min(...data);
	const max = Math.max(...data);
	const range = max - min || 1;
	return data.map((v, i) => `${((i / (data.length - 1)) * w).toFixed(1)},${(h - 2 - ((v - min) / range) * (h - 5)).toFixed(1)}`);
}

// ── shared prop types ──────────────────────────────────────────────────
export type BadgeTone = 'default' | 'accent' | 'ok' | 'warn' | 'err';
export type Tone = 'ok' | 'warn' | 'err' | 'accent';
/** One meter row. `frac: null` (or NaN) means unknown: the row says so and draws no bar. */
export type Meter = { label: string; frac: number | null; display: string; hue: string };
export type SeriesPoint = { label: string; count: number };
export type StatChip = { tone?: Tone; text: string };
