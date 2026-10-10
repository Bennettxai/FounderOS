/**
 * /brand-deals on the bridge: the payload types of GET /api/founderos/pages/brand-deals
 * and the interactive half of FounderOS v1 lib/brand-deals-view.ts. The server
 * precomputes every figure the slab shows (internal/founderos/pages/branddeals);
 * what stays here runs on each keystroke or on the viewer's clock: the prompt
 * bar's slash filters, the chip filters, row money, and "5m ago".
 */
import type { Meter, SeriesPoint, Tone } from '$lib/founderos/kit/format';

export type BrandDeal = {
	id: string;
	brand: string;
	status: string;
	tier: string | null;
	dealValueUsd: number | null;
	budgetUsd: number | null;
	amountAgreedUsd: number | null;
	suggestedRateUsd: number | null;
	paidInFull: boolean;
	deadline: string | null;
	followUpDate: string | null;
	contactName: string | null;
	contactEmail: string | null;
	mainChannel: string | null;
	videoType: string | null;
	source: string | null;
	icpFit: string | null;
	notionUrl: string;
	lastEdited: string;
	seeded: boolean;
};

export type Filter = 'all' | 'talks' | 'production' | 'paid' | 'declined' | 'tier-s';

export type FunnelStage = { label: string; value: number; usd: number; note: string; filter: Filter };

export type DealVolume = {
	openUsd: number;
	talksUsd: number;
	productionUsd: number;
	paidUsd: number;
	declinedUsd: number;
	quotedDeals: number;
	counts: { all: number; talks: number; production: number; paid: number; declined: number; tierS: number };
};

export type DealView = {
	volume: DealVolume;
	stages: FunnelStage[];
	activity: SeriesPoint[];
	editsInWindow: number;
	sizes: SeriesPoint[];
	largestUsd: number;
	due: { followUps: string[]; deadlines: string[] };
	needsYou: number;
	needsYouBrands: string[];
	openCount: number;
	needsYouFrac: number;
	hubUrl: string | null;
};

export type BrandDealsMode = 'live' | 'seeded' | 'error';

/** The route's body: FounderOS v1's BrandDealsResult plus the slab's view.
 *  `view` is null when Notion could not be read (unknown, never zeroes). */
export type BrandDealsBody = {
	deals: BrandDeal[];
	mode: BrandDealsMode;
	detail: string;
	syncedAt: number | null;
	view: DealView | null;
};

export type DealBucket = 'inbound' | 'talking' | 'producing' | 'billing' | 'paid' | 'declined' | 'paused' | 'other';

const BUCKET: Record<string, DealBucket> = {
	New: 'inbound',
	Negotiating: 'talking',
	Aligned: 'talking',
	Researching: 'producing',
	Filming: 'producing',
	Editing: 'producing',
	Delivered: 'producing',
	Invoiced: 'billing',
	Approved: 'billing',
	Paid: 'paid',
	Declined: 'declined',
	Paused: 'paused'
};

/** Notion lane to journey bucket; unknown lanes are kept as `other`. */
export function bucketOf(status: string): DealBucket {
	return BUCKET[status] ?? 'other';
}

/** Agreed, else value, else budget, else nothing (never the suggested rate). */
export function dealUsd(d: BrandDeal): number {
	return d.amountAgreedUsd ?? d.dealValueUsd ?? d.budgetUsd ?? 0;
}

export function matchesFilter(d: BrandDeal, f: Filter): boolean {
	const b = bucketOf(d.status);
	switch (f) {
		case 'all':
			return true;
		case 'talks':
			return b === 'inbound' || b === 'talking';
		case 'production':
			return b === 'producing' || b === 'billing';
		case 'paid':
			return b === 'paid';
		case 'declined':
			return b === 'declined' || b === 'paused';
		case 'tier-s':
			return (d.tier ?? '').trim().toUpperCase() === 'S';
	}
}

const SLASH: Record<string, Filter> = {
	'/talks': 'talks',
	'/production': 'production',
	'/paid': 'paid',
	'/declined': 'declined',
	'/s': 'tier-s',
	'/tier-s': 'tier-s'
};

/** A leading slash token is a filter; whatever follows is free text. An
 *  unknown token is echoed back (so the pill shows it) but filters nothing. */
export function parseQuery(q: string): { slash: string | null; token: Filter | null; text: string } {
	const t = q.trim();
	if (t.startsWith('/')) {
		const slash = t.split(/\s+/)[0];
		return { slash, token: SLASH[slash.toLowerCase()] ?? null, text: t.slice(slash.length).trim().toLowerCase() };
	}
	return { slash: null, token: null, text: t.toLowerCase() };
}

export function filterDeals(deals: BrandDeal[], opts: { filter: Filter; query: string }): BrandDeal[] {
	const { token, text } = parseQuery(opts.query);
	return deals.filter((d) => {
		if (!matchesFilter(d, opts.filter)) return false;
		if (token && !matchesFilter(d, token)) return false;
		if (text) {
			const hay = [d.brand, d.contactName, d.contactEmail, d.status, d.mainChannel, d.videoType, d.source, d.tier]
				.filter(Boolean)
				.join(' ')
				.toLowerCase();
			if (!hay.includes(text)) return false;
		}
		return true;
	});
}

export function timeAgo(iso: string | null, now: Date = new Date()): string {
	if (!iso) return '';
	const then = Date.parse(iso);
	if (Number.isNaN(then)) return '';
	const mins = Math.max(0, Math.round((now.getTime() - then) / 60_000));
	if (mins < 60) return `${mins}m ago`;
	const hrs = Math.round(mins / 60);
	if (hrs < 48) return `${hrs}h ago`;
	return `${Math.round(hrs / 24)}d ago`;
}

export function fmtUsd(n: number): string {
	return `$${Math.round(n).toLocaleString('en-US')}`;
}

export function syncAge(syncedAt: number | null, now: number = Date.now()): string | null {
	if (syncedAt === null) return null;
	const m = Math.max(0, Math.floor((now - syncedAt) / 60_000));
	if (m < 1) return 'synced just now';
	if (m < 60) return `synced ${m}m ago`;
	return `synced ${Math.floor(m / 60)}h ago`;
}

/** Which hero column lights for a chip filter. */
export const FILTER_TO_STAGE_IDX: Record<Filter, number> = { all: 0, 'tier-s': 0, talks: 1, production: 2, paid: 3, declined: 3 };

/** One hue per data domain, riding the Monolith tokens (color means status). */
export const HUE = {
	accent: 'var(--bn-accent)',
	amber: 'var(--bn-warn)',
	cobalt: 'var(--bn-text-2)',
	violet: 'var(--bn-accent)',
	activity: 'var(--bn-text-2)'
};

/** Status pill tone for a deal's lane. */
export function statusTone(d: BrandDeal): Tone | null {
	const b = bucketOf(d.status);
	if (b === 'paid') return 'ok';
	if (b === 'declined' || b === 'paused') return 'err';
	if (b === 'inbound' || b === 'talking') return 'warn';
	if (b === 'producing' || b === 'billing') return 'accent';
	return null;
}

export function badgeTone(d: BrandDeal): 'ok' | 'warn' | 'err' | 'default' {
	const t = statusTone(d);
	return t === 'accent' || t === null ? 'default' : t;
}

/** The Deal Volume card's three meters; with no view every row reads unknown. */
export function volumeMeters(view: DealView | null): Meter[] {
	if (!view) {
		return [
			{ label: 'In talks', frac: null, display: '', hue: HUE.amber },
			{ label: 'In production', frac: null, display: '', hue: HUE.cobalt },
			{ label: 'Paid vs declined', frac: null, display: '', hue: HUE.accent }
		];
	}
	const v = view.volume;
	const closed = v.paidUsd + v.declinedUsd;
	return [
		{ label: `In talks (${v.counts.talks})`, frac: v.openUsd > 0 ? v.talksUsd / v.openUsd : 0, display: fmtUsd(v.talksUsd), hue: HUE.amber },
		{ label: `In production (${v.counts.production})`, frac: v.openUsd > 0 ? v.productionUsd / v.openUsd : 0, display: fmtUsd(v.productionUsd), hue: HUE.cobalt },
		{
			label: `Paid vs declined (${v.counts.paid}/${v.counts.declined})`,
			frac: closed > 0 ? v.paidUsd / closed : 0,
			display: `${fmtUsd(v.paidUsd)} / ${fmtUsd(v.declinedUsd)}`,
			hue: HUE.accent
		}
	];
}

/** The chip row under "Deals", with counts; null counts read "?" when unknown. */
export function filterChips(view: DealView | null): Array<[Filter, string]> {
	const c = view?.volume.counts;
	const n = (k: keyof DealVolume['counts']) => (c ? String(c[k]) : '?');
	return [
		['all', `All ${n('all')}`],
		['talks', `In talks ${n('talks')}`],
		['production', `In production ${n('production')}`],
		['paid', `Paid ${n('paid')}`],
		['declined', `Declined ${n('declined')}`],
		['tier-s', `S-tier ${n('tierS')}`]
	];
}
