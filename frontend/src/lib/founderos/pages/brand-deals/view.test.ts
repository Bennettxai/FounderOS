import { describe, expect, test } from 'vitest';
import {
	bucketOf,
	dealUsd,
	filterChips,
	filterDeals,
	matchesFilter,
	parseQuery,
	syncAge,
	timeAgo,
	volumeMeters,
	type BrandDeal,
	type DealView
} from './view';

/**
 * Ported from FounderOS v1 tests/brand-deals-view.test.ts: the interactive half
 * (buckets, money, filters, the prompt bar, timeAgo). The figures half runs
 * in Go (internal/founderos/pages/branddeals/view_test.go).
 */
const NOW = new Date('2026-09-17T12:00:00.000Z');

function deal(over: Partial<BrandDeal> & { id: string; brand: string }): BrandDeal {
	return {
		status: 'New',
		tier: null,
		dealValueUsd: null,
		budgetUsd: null,
		amountAgreedUsd: null,
		suggestedRateUsd: null,
		paidInFull: false,
		deadline: null,
		followUpDate: null,
		contactName: null,
		contactEmail: null,
		mainChannel: null,
		videoType: null,
		source: null,
		icpFit: null,
		notionUrl: 'https://www.notion.so/x',
		lastEdited: '2026-09-16T09:00:00.000Z',
		seeded: false,
		...over
	};
}

const DEALS: BrandDeal[] = [
	deal({ id: 'a', brand: 'Notion', status: 'Negotiating', tier: 'S', dealValueUsd: 6000, followUpDate: '2026-09-18', contactName: 'Sam Rivera', mainChannel: 'Instagram' }),
	deal({ id: 'b', brand: 'Framer', status: 'New', tier: 'A', suggestedRateUsd: 4000, followUpDate: '2026-09-10' }),
	deal({ id: 'c', brand: 'Shopify', status: 'Filming', tier: 'S', dealValueUsd: 9000, amountAgreedUsd: 8500, deadline: '2026-09-20' }),
	deal({ id: 'd', brand: 'Riverside', status: 'Paid', tier: 'B', amountAgreedUsd: 2500, paidInFull: true }),
	deal({ id: 'e', brand: 'Loom', status: 'Declined', budgetUsd: 1200 }),
	deal({ id: 'f', brand: 'Descript', status: 'Invoiced', amountAgreedUsd: 12000, deadline: '2026-10-30' }),
	deal({ id: 'g', brand: 'Mystery', status: 'Vibing' })
];

describe('buckets and dollars', () => {
	test('every Notion lane lands in a journey bucket; unknown lanes are kept, not dropped', () => {
		expect(bucketOf('New')).toBe('inbound');
		expect(bucketOf('Negotiating')).toBe('talking');
		expect(bucketOf('Aligned')).toBe('talking');
		expect(bucketOf('Researching')).toBe('producing');
		expect(bucketOf('Filming')).toBe('producing');
		expect(bucketOf('Editing')).toBe('producing');
		expect(bucketOf('Delivered')).toBe('producing');
		expect(bucketOf('Invoiced')).toBe('billing');
		expect(bucketOf('Approved')).toBe('billing');
		expect(bucketOf('Paid')).toBe('paid');
		expect(bucketOf('Declined')).toBe('declined');
		expect(bucketOf('Paused')).toBe('paused');
		expect(bucketOf('Vibing')).toBe('other');
	});

	test('a deal is worth what was agreed, else its value, else its budget, else nothing (never the suggested rate)', () => {
		expect(dealUsd(DEALS[2])).toBe(8500);
		expect(dealUsd(DEALS[0])).toBe(6000);
		expect(dealUsd(DEALS[4])).toBe(1200);
		expect(dealUsd(DEALS[1])).toBe(0);
	});
});

describe('filters and the prompt bar', () => {
	test('filters group buckets the way the chips read', () => {
		expect(matchesFilter(DEALS[1], 'talks')).toBe(true);
		expect(matchesFilter(DEALS[0], 'talks')).toBe(true);
		expect(matchesFilter(DEALS[2], 'production')).toBe(true);
		expect(matchesFilter(DEALS[5], 'production')).toBe(true);
		expect(matchesFilter(DEALS[3], 'paid')).toBe(true);
		expect(matchesFilter(DEALS[4], 'declined')).toBe(true);
		expect(matchesFilter(DEALS[0], 'tier-s')).toBe(true);
		expect(matchesFilter(DEALS[1], 'tier-s')).toBe(false);
		expect(matchesFilter(DEALS[6], 'all')).toBe(true);
	});

	test('a leading slash token becomes a filter, the rest is free text; unknown tokens filter nothing', () => {
		expect(parseQuery('/talks notion')).toEqual({ slash: '/talks', token: 'talks', text: 'notion' });
		expect(parseQuery('/s')).toEqual({ slash: '/s', token: 'tier-s', text: '' });
		expect(parseQuery('/nope shopify')).toEqual({ slash: '/nope', token: null, text: 'shopify' });
		expect(parseQuery('  Framer ')).toEqual({ slash: null, token: null, text: 'framer' });
	});

	test('free text searches brand, contact, status, channel, format, source and tier', () => {
		expect(filterDeals(DEALS, { filter: 'all', query: 'rivera' }).map((d) => d.id)).toEqual(['a']);
		expect(filterDeals(DEALS, { filter: 'all', query: 'instagram' }).map((d) => d.id)).toEqual(['a']);
		expect(filterDeals(DEALS, { filter: 'all', query: 'filming' }).map((d) => d.id)).toEqual(['c']);
		expect(filterDeals(DEALS, { filter: 'talks', query: '/s' }).map((d) => d.id)).toEqual(['a']);
		expect(filterDeals(DEALS, { filter: 'all', query: '' })).toHaveLength(7);
	});
});

describe('clock-relative labels', () => {
	test('timeAgo reads in minutes, hours, then days', () => {
		expect(timeAgo('2026-09-17T11:30:00.000Z', NOW)).toBe('30m ago');
		expect(timeAgo('2026-09-16T12:00:00.000Z', NOW)).toBe('24h ago');
		expect(timeAgo('2026-09-10T12:00:00.000Z', NOW)).toBe('7d ago');
		expect(timeAgo(null, NOW)).toBe('');
	});

	test('syncAge: none when nothing synced, then just now, minutes, hours', () => {
		const t = NOW.getTime();
		expect(syncAge(null, t)).toBeNull();
		expect(syncAge(t - 20_000, t)).toBe('synced just now');
		expect(syncAge(t - 5 * 60_000, t)).toBe('synced 5m ago');
		expect(syncAge(t - 130 * 60_000, t)).toBe('synced 2h ago');
	});
});

describe('honest meters and chips', () => {
	test('no view (Notion unreadable): every meter is unknown and chip counts read ?', () => {
		expect(volumeMeters(null).every((m) => m.frac === null)).toBe(true);
		expect(filterChips(null)[0]).toEqual(['all', 'All ?']);
	});

	test('with a view the meters split the open pipeline and closed money', () => {
		const view = {
			volume: { openUsd: 26500, talksUsd: 6000, productionUsd: 20500, paidUsd: 2500, declinedUsd: 1200, quotedDeals: 5, counts: { all: 7, talks: 2, production: 2, paid: 1, declined: 1, tierS: 2 } }
		} as DealView;
		const m = volumeMeters(view);
		expect(m.map((x) => x.label)).toEqual(['In talks (2)', 'In production (2)', 'Paid vs declined (1/1)']);
		expect(m[0].frac).toBeCloseTo(6000 / 26500);
		expect(m[2].display).toBe('$2,500 / $1,200');
		expect(filterChips(view).map(([, l]) => l)).toEqual(['All 7', 'In talks 2', 'In production 2', 'Paid 1', 'Declined 1', 'S-tier 2']);
	});
});
