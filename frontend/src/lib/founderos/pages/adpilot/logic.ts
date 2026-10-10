/**
 * Pure logic behind the /adpilot view, ported from FounderOS v1
 * components/adpilot/{AdPilotDeck,AdLibrary}.tsx and lib/foreplay/personal.ts.
 * Selection-dependent aggregates come precomputed from the backend (Deck
 * views); what stays here is display math and the client-side For-you feed,
 * which mixes in the last search's results and the viewer's saves.
 */
import type { Campaign, DailyPoint, Signal, WallAd } from './types';

const nf = new Intl.NumberFormat('en-US');
export const num = (v: number) => nf.format(v);
export const money = (v: number) => `$${nf.format(Math.round(v))}`;
export const money2 = (v: number | null) => (v === null ? '-' : `$${v.toFixed(2)}`);
export const times = (v: number | null) => (v === null ? '-' : `${v.toFixed(2)}x`);
export const pct = (v: number | null) => (v === null ? '-' : `${(v * 100).toFixed(2)}%`);

export function ago(iso: string | null, now = Date.now()): string {
	if (!iso) return 'never';
	const mins = Math.max(0, Math.round((now - new Date(iso).getTime()) / 60000));
	if (mins < 60) return `${mins}m ago`;
	if (mins < 60 * 36) return `${Math.round(mins / 60)}h ago`;
	return `${Math.round(mins / 1440)}d ago`;
}

/** Last 7 days against the 7 before; null without 14 days or a prior base. */
export function windowDelta(daily: DailyPoint[], pick: (d: DailyPoint) => number): number | null {
	if (daily.length < 14) return null;
	const last7 = daily.slice(-7).reduce((s, d) => s + pick(d), 0);
	const prior7 = daily.slice(-14, -7).reduce((s, d) => s + pick(d), 0);
	if (prior7 <= 0) return null;
	return (last7 - prior7) / prior7;
}

/** Lit ticks (of 20) on the credit meter; unknown credits light none. */
export function creditTicks(credits: { remaining: number; total: number } | null): number {
	if (!credits || credits.total <= 0) return 0;
	return Math.round((credits.remaining / credits.total) * 20);
}

/** Delivery matrix dot: that day's share of the campaign's peak spend. */
export function deliveryShade(t: number): string {
	return t <= 0.02 ? 'rgba(255,255,255,0.07)' : `rgba(255,69,87,${(0.1 + Math.pow(t, 1.7) * 0.88).toFixed(2)})`;
}

/** Rail order: active first, then a case-insensitive name filter. */
export function railOrder(campaigns: Campaign[], filter: string): Campaign[] {
	const q = filter.trim().toLowerCase();
	return [...campaigns]
		.sort((a, b) => Number(b.status === 'active') - Number(a.status === 'active'))
		.filter((c) => !q || c.name.toLowerCase().includes(q));
}

// ── For you (lib/foreplay/personal.ts) ───────────────────────────────────

/** The operator's lanes: AI agents/automation, founder offers, creator education. */
export const NICHE_KEYWORDS = [
	'ai', 'agent', 'agents', 'automation', 'automate', 'founder', 'business', 'agency', 'marketing', 'course',
	'community', 'coaching', 'clients', 'leads', 'software', 'saas', 'app', 'system', 'income', 'revenue'
];

function keywordScore(ad: WallAd): number {
	const words = new Set(`${ad.hook ?? ''} ${ad.brand}`.toLowerCase().split(/[^a-z0-9$]+/));
	let hits = 0;
	for (const kw of NICHE_KEYWORDS) if (words.has(kw)) hits += 1;
	return Math.min(hits / 4, 1);
}

const longevityScore = (ad: WallAd) => Math.min(ad.daysRunning / 90, 1);

function driverSimilarity(ad: WallAd, mean: Record<string, number>): number {
	const axes = Object.keys(mean);
	if (axes.length === 0 || Object.keys(ad.drivers).length === 0) return 0;
	let dot = 0;
	let a2 = 0;
	let b2 = 0;
	for (const axis of axes) {
		const a = ad.drivers[axis] ?? 0;
		const b = mean[axis];
		dot += a * b;
		a2 += a * a;
		b2 += b * b;
	}
	return a2 > 0 && b2 > 0 ? dot / (Math.sqrt(a2) * Math.sqrt(b2)) : 0;
}

export function meanDrivers(ads: WallAd[]): Record<string, number> {
	const sums = new Map<string, { total: number; n: number }>();
	for (const ad of ads) {
		for (const [axis, score] of Object.entries(ad.drivers)) {
			const cur = sums.get(axis) ?? { total: 0, n: 0 };
			cur.total += score;
			cur.n += 1;
			sums.set(axis, cur);
		}
	}
	return Object.fromEntries([...sums.entries()].map(([axis, { total, n }]) => [axis, total / n]));
}

export type ScoredAd = { ad: WallAd; score: number };

/** Weights: niche 3, longevity 3, saved-taste similarity 3, live 1. */
export function forYou(candidates: WallAd[], savedAds: WallAd[]): ScoredAd[] {
	const taste = meanDrivers(savedAds);
	const savedIds = new Set(savedAds.map((a) => a.id));
	return candidates
		.filter((ad) => !savedIds.has(ad.id))
		.map((ad) => ({
			ad,
			score: keywordScore(ad) * 3 + longevityScore(ad) * 3 + driverSimilarity(ad, taste) * 3 + (ad.live ? 1 : 0)
		}))
		.sort((a, b) => b.score - a.score);
}

/** Keyword overlap: the ads an Ask answer is most plausibly about. */
export function relatedAds(text: string, wall: WallAd[]): WallAd[] {
	const words = new Set(
		text
			.toLowerCase()
			.split(/[^a-z0-9$]+/)
			.filter((w) => w.length > 3)
	);
	return wall
		.map((ad) => {
			const hay = `${ad.hook ?? ''} ${ad.brand}`.toLowerCase().split(/[^a-z0-9$]+/);
			let score = 0;
			for (const w of hay) if (words.has(w)) score += 1;
			return { ad, score: score + (ad.live ? 0.5 : 0) + Math.min(ad.daysRunning / 200, 0.5) };
		})
		.filter((r) => r.score >= 1.5)
		.sort((a, b) => b.score - a.score)
		.slice(0, 3)
		.map((r) => r.ad);
}

/** Activity tab: signals grouped by day, newest day first. */
export function groupSignals(signals: Signal[]): [string, Signal[]][] {
	const map = new Map<string, Signal[]>();
	for (const s of signals) {
		const day = s.at.slice(0, 10);
		map.set(day, [...(map.get(day) ?? []), s]);
	}
	return [...map.entries()].sort((a, b) => b[0].localeCompare(a[0]));
}

/** The card grid's 2x2 lead slot needs imagery: promote the first still. */
export function leadWithImage(ads: WallAd[]): WallAd[] {
	const i = ads.findIndex((a) => a.thumbnail || a.image);
	if (i <= 0) return ads;
	return [ads[i], ...ads.slice(0, i), ...ads.slice(i + 1)];
}

/** Per-brand live count and longest run, for the watchlist rows. */
export function brandCounts(wall: WallAd[]): Map<string, { live: number; top: number }> {
	const counts = new Map<string, { live: number; top: number }>();
	for (const ad of wall) {
		if (!ad.brandId) continue;
		const cur = counts.get(ad.brandId) ?? { live: 0, top: 0 };
		if (ad.live) cur.live += 1;
		cur.top = Math.max(cur.top, ad.daysRunning);
		counts.set(ad.brandId, cur);
	}
	return counts;
}

export const mmss = (t: number) => `${Math.floor(t / 60)}:${String(Math.floor(t % 60)).padStart(2, '0')}`;
