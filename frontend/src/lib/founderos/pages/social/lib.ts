/**
 * Client-side logic of the /os/social pages: FounderOS v1's formatters
 * (components/SocialStats.tsx), the posting-activity bucketing
 * (lib/posting-activity.ts), the stat strip's range math and the chart
 * geometry (lib/social-chart.ts). The page numbers themselves are computed
 * by the backend (internal/founderos/pages/social); these only draw.
 */
import type { PostDay, SeriesValue } from './types';

const LABELS: Record<string, string> = {
	instagram: 'Instagram',
	tiktok: 'TikTok',
	twitter: 'X',
	youtube: 'YouTube',
	linkedin: 'LinkedIn',
	facebook: 'Facebook'
};

export const PLATFORM_COLORS: Record<string, string> = {
	instagram: '#e1306c',
	tiktok: '#25f4ee',
	twitter: '#f2f2f2',
	youtube: '#ff4d4d',
	linkedin: '#0a85c2',
	facebook: '#1877f2'
};

export const PLATFORM_ORDER = ['instagram', 'tiktok', 'twitter', 'youtube', 'linkedin', 'facebook'];

export function platformLabel(p: string): string {
	return LABELS[p] ?? p.charAt(0).toUpperCase() + p.slice(1);
}

export const colorFor = (p: string): string => PLATFORM_COLORS[p] ?? 'var(--bn-accent)';

export function formatFollowers(n: number | null | undefined): string {
	return n == null ? '—' : n.toLocaleString('en-US');
}

export function formatPct(n: number | null | undefined): string {
	if (n == null || Number.isNaN(n)) return '—';
	const r = Math.abs(n) < 10 ? n.toFixed(2) : n.toFixed(1);
	return `${n >= 0 ? '+' : ''}${r}%`;
}

export const pctTone = (n: number | null | undefined): string =>
	n == null ? 'bn-dim' : n >= 0 ? 'soc-ok' : 'soc-err';

// ── posting activity ────────────────────────────────────────────────────

export type PostingActivityDay = { date: string; counts: Record<string, number>; total: number };

const DAY_MS = 86_400_000;

export function dateAxis(start: string, end: string): string[] {
	const s = Date.parse(`${start}T00:00:00Z`);
	const e = Date.parse(`${end}T00:00:00Z`);
	if (!Number.isFinite(s) || !Number.isFinite(e) || s > e) return [];
	const out: string[] = [];
	for (let t = s; t <= e; t += DAY_MS) out.push(new Date(t).toISOString().slice(0, 10));
	return out;
}

export function bucketPostingByDay(posts: PostDay[], axis: string[]): PostingActivityDay[] {
	const byDate = new Map<string, Record<string, number>>(axis.map((d) => [d, {}]));
	for (const post of posts) {
		const counts = byDate.get(post.date);
		if (!counts) continue;
		for (const p of post.platforms) counts[p] = (counts[p] ?? 0) + 1;
	}
	return axis.map((date) => {
		const counts = byDate.get(date)!;
		return { date, counts, total: Object.values(counts).reduce((a, b) => a + b, 0) };
	});
}

export function forwardFill(points: SeriesValue[], axis: string[]): (number | null)[] {
	const byDate = new Map(points.map((p) => [p.date, p.value]));
	let last: number | null = null;
	return axis.map((date) => {
		if (byDate.has(date)) last = byDate.get(date)!;
		return last;
	});
}

export function platformsPresent(posts: PostDay[], preferred: string[]): string[] {
	const seen = new Set(posts.flatMap((p) => p.platforms));
	const ordered = preferred.filter((p) => seen.has(p));
	const extras = [...seen].filter((p) => !preferred.includes(p)).sort();
	return [...ordered, ...extras];
}

// ── stat strip ranges ───────────────────────────────────────────────────

export type Range = 7 | 30 | 60 | 'all';
export const RANGES: Range[] = [7, 30, 60, 'all'];
export const RANGE_LABEL: Record<string, string> = { '7': '7d', '30': '30d', '60': '60d', all: 'All' };
export const RANGE_KEY = { '7': 'd7', '30': 'd30', '60': 'd60', all: 'allTime' } as const;

function dateMinusDays(day: string, days: number): string {
	const d = new Date(`${day}T00:00:00Z`);
	d.setUTCDate(d.getUTCDate() - days);
	return d.toISOString().slice(0, 10);
}

export function inRange(points: SeriesValue[], range: Range): SeriesValue[] {
	if (range === 'all' || points.length === 0) return points;
	const start = dateMinusDays(points[points.length - 1].date, range);
	return points.filter((p) => p.date >= start);
}

export function growthOf(points: SeriesValue[]): number | null {
	if (points.length < 2 || points[0].value === 0) return null;
	return ((points[points.length - 1].value - points[0].value) / points[0].value) * 100;
}

// ── charts ──────────────────────────────────────────────────────────────

export type PieItem = { key: string; label: string; value: number | null };
export type PieSlice = { key: string; label: string; value: number; share: number; startAngle: number; endAngle: number };

export function pieSlices(items: PieItem[]): PieSlice[] {
	const live = items.filter((i): i is PieItem & { value: number } => i.value != null && i.value > 0);
	const total = live.reduce((s, i) => s + i.value, 0);
	if (total <= 0) return [];
	let angle = -90;
	return live.map((i) => {
		const share = i.value / total;
		const startAngle = angle;
		angle += share * 360;
		return { key: i.key, label: i.label, value: i.value, share, startAngle, endAngle: angle };
	});
}

export type FollowerPoint = { date: string; followers: number };
export type FollowerBar = FollowerPoint & { h: number; delta: number | null; xLabel: string | null };

function niceStep(range: number, ticks: number): number {
	const raw = range / Math.max(1, ticks);
	const mag = 10 ** Math.floor(Math.log10(Math.max(1, raw)));
	for (const m of [1, 2, 5, 10]) if (raw <= m * mag) return m * mag;
	return 10 * mag;
}

export function followerBarModel(series: FollowerPoint[]): { bars: FollowerBar[]; yTicks: number[]; floor: number; ceil: number } {
	if (series.length === 0) return { bars: [], yTicks: [], floor: 0, ceil: 0 };
	const values = series.map((p) => p.followers);
	const min = Math.min(...values);
	const max = Math.max(...values);
	const pad = Math.max((max - min) * 0.15, max * 0.02, 1);
	const floor = Math.max(0, min - pad);
	const ceil = max;
	const span = Math.max(1, ceil - floor);
	const labelEvery = Math.max(1, Math.ceil(series.length / 8));
	const bars = series.map((p, i) => ({
		date: p.date,
		followers: p.followers,
		h: Math.min(1, Math.max(0.02, (p.followers - floor) / span)),
		delta: i === 0 ? null : p.followers - series[i - 1].followers,
		xLabel: i % labelEvery === 0 || i === series.length - 1 ? p.date.slice(5) : null
	}));
	const step = niceStep(span, 4);
	const yTicks: number[] = [];
	for (let v = Math.floor(floor / step) * step; v <= ceil + step; v += step) {
		if (v >= 0) yTicks.push(v);
		if (yTicks.length > 8) break;
	}
	return { bars, yTicks, floor, ceil };
}

/** Donut arc path between two angles (SharePie). */
export function arcPath(cx: number, cy: number, r: number, a0: number, a1: number): string {
	const rad = (d: number) => (d * Math.PI) / 180;
	const sweep = Math.min(a1 - a0, 359.98);
	const x0 = cx + r * Math.cos(rad(a0));
	const y0 = cy + r * Math.sin(rad(a0));
	const x1 = cx + r * Math.cos(rad(a0 + sweep));
	const y1 = cy + r * Math.sin(rad(a0 + sweep));
	return `M ${x0.toFixed(2)} ${y0.toFixed(2)} A ${r} ${r} 0 ${sweep > 180 ? 1 : 0} 1 ${x1.toFixed(2)} ${y1.toFixed(2)}`;
}

/** Compact relative time for DMs: now · 5m · 3h · 2d (rounded). */
export function relativeTime(iso: string, nowMs: number): string {
	const diff = nowMs - new Date(iso).getTime();
	if (Number.isNaN(diff)) return '';
	const m = Math.round(diff / 60000);
	if (m < 1) return 'now';
	if (m < 60) return `${m}m`;
	const h = Math.round(m / 60);
	if (h < 24) return `${h}h`;
	return `${Math.round(h / 24)}d`;
}

export function initials(name: string): string {
	return name
		.split(/\s+/)
		.slice(0, 2)
		.map((p) => p[0]?.toUpperCase() ?? '')
		.join('');
}

// The Go pagekit hues, for the charts the page draws itself.
export const HUE_SEND = 'color-mix(in oklab, var(--bn-text-2) 70%, var(--bn-text))';
export const HUE_RAMP1 = 'var(--bn-text-2)';
export const HUE_RAMP3 = 'var(--bn-text)';
export const HUE_OK = 'var(--bn-ok)';

// ── engagement (lib/engagement.ts) ──────────────────────────────────────

/** Like-to-view ratio as a percentage; null when views isn't a positive number. */
export function likeToViewRatio(likes: number, views: number): number | null {
	if (!Number.isFinite(likes) || !Number.isFinite(views) || views <= 0) return null;
	return (likes / views) * 100;
}

/** Mean like-to-view ratio across the posts that have a usable view count. */
export function averageLikeToView(posts: { likes: number; views: number }[]): number | null {
	const ratios = posts.map((p) => likeToViewRatio(p.likes, p.views)).filter((r): r is number => r != null);
	return ratios.length === 0 ? null : ratios.reduce((s, r) => s + r, 0) / ratios.length;
}

/** "8.9%", or an em dash when there's nothing to show. */
export const formatRatioPct = (ratio: number | null): string => (ratio == null ? '—' : `${ratio.toFixed(1)}%`);

/** app/social/page.tsx RECENT_POSTS: shown (labelled "sample") only when
 *  Zernio's live history is empty or unreachable. */
export const SAMPLE_POSTS = [
	{ tag: 'Instagram · Reel', ago: '2h', caption: '3 agents that run my business while I sleep', kind: 'views', views: 12400, likes: 1104 },
	{ tag: 'TikTok · Video', ago: '6h', caption: 'POV: your operating system has a command palette', kind: 'views', views: 8100, likes: 640 },
	{ tag: 'X · Thread', ago: '1d', caption: 'How I wired 7 real connectors into one OS', kind: 'impressions', views: 1200, likes: 74 },
	{ tag: 'YouTube · Long', ago: '2d', caption: 'Founder OS walkthrough - building in public #4', kind: 'views', views: 940, likes: 88 },
	{ tag: 'Instagram · Carousel', ago: '3d', caption: 'The larp-first, real-ready architecture', kind: 'reach', views: 6700, likes: 717 }
];

/** Relative "2h"/"3d" from a published-at stamp (app/social/page.tsx agoFrom). */
export function agoFrom(iso: string | null, nowMs: number): string {
	if (!iso) return '';
	const ms = nowMs - new Date(iso).getTime();
	if (!Number.isFinite(ms) || ms < 0) return '';
	const mins = Math.round(ms / 60_000);
	if (mins < 60) return `${Math.max(1, mins)}m`;
	const hrs = Math.round(mins / 60);
	if (hrs < 48) return `${hrs}h`;
	return `${Math.round(hrs / 24)}d`;
}

// ── the newsletter header (app/social/page.tsx) ─────────────────────────

/** 7d / 30d growth chips from real snapshots only; a window with no honest
 *  baseline gets no chip. */
export function emailChips(growth: { d7: number | null; d30: number | null }): { tone: 'ok' | 'err'; text: string }[] {
	const out: { tone: 'ok' | 'err'; text: string }[] = [];
	if (growth.d7 != null) out.push({ tone: growth.d7 >= 0 ? 'ok' : 'err', text: `${formatPct(growth.d7)} 7d` });
	if (growth.d30 != null) out.push({ tone: growth.d30 >= 0 ? 'ok' : 'err', text: `${formatPct(growth.d30)} 30d` });
	return out;
}

export const emailCaption = (asOf: string | null): string =>
	asOf ? `subscribers · Beehiiv reading ${asOf.slice(0, 10)}` : 'no Beehiiv reading yet · add BEEHIIV_API_KEY';

// ── open-rate matrix labels ─────────────────────────────────────────────
/**
 * The DotMatrix columns as prod prints them: a date that repeats reads the
 * same each time ("Sep 9", "Sep 9"). The backend numbers repeats ("Sep 9 2")
 * so keyed lists stay unique; this strips that number and keeps the keys
 * unique with zero-width spaces, which render as nothing.
 */
export function matrixCols<T extends { label: string; count: number }>(points: T[]): T[] {
	const seen = new Map<string, number>();
	return points.map((p) => {
		const base = /^([A-Z][a-z]{2} \d{1,2}) \d+$/.exec(p.label)?.[1] ?? p.label;
		const n = seen.get(base) ?? 0;
		seen.set(base, n + 1);
		return { ...p, label: base + '\u200b'.repeat(n) };
	});
}
