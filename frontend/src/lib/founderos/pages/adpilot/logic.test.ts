import { describe, expect, it } from 'vitest';
import {
	ago,
	creditTicks,
	deliveryShade,
	forYou,
	groupSignals,
	leadWithImage,
	money,
	money2,
	pct,
	railOrder,
	relatedAds,
	times,
	windowDelta
} from './logic';
import type { Campaign, Signal, WallAd } from './types';

function wall(id: string, over: Partial<WallAd> = {}): WallAd {
	return {
		id,
		brand: 'Acme',
		brandId: 'br_1',
		thumbnail: null,
		video: null,
		image: null,
		format: 'video',
		live: true,
		daysRunning: 10,
		hook: null,
		hookSource: null,
		transcript: [],
		drivers: {},
		ctaType: null,
		linkUrl: null,
		...over
	};
}

describe('number formats: null reads "-", never 0', () => {
	it('formats', () => {
		expect(money(1234.6)).toBe('$1,235');
		expect(money2(null)).toBe('-');
		expect(money2(3.456)).toBe('$3.46');
		expect(times(null)).toBe('-');
		expect(times(2.5)).toBe('2.50x');
		expect(pct(null)).toBe('-');
		expect(pct(0.0213)).toBe('2.13%');
	});
});

describe('ago', () => {
	const now = Date.parse('2026-09-30T12:00:00Z');
	it('reads never, minutes, hours, days', () => {
		expect(ago(null, now)).toBe('never');
		expect(ago('2026-09-30T11:30:00Z', now)).toBe('30m ago');
		expect(ago('2026-09-30T02:00:00Z', now)).toBe('10h ago');
		expect(ago('2026-09-27T12:00:00Z', now)).toBe('3d ago');
	});
});

describe('windowDelta', () => {
	const days = (vals: number[]) => vals.map((v, i) => ({ date: `d${i}`, spend: v, leads: v, bookings: 0 }));
	it('needs 14 days and a nonzero prior week', () => {
		expect(windowDelta(days(Array(13).fill(1)), (d) => d.spend)).toBeNull();
		expect(windowDelta(days([...Array(7).fill(0), ...Array(7).fill(5)]), (d) => d.spend)).toBeNull();
		expect(windowDelta(days([...Array(7).fill(10), ...Array(7).fill(15)]), (d) => d.spend)).toBeCloseTo(0.5);
	});
});

describe('credit meter', () => {
	it('lights ticks by the remaining fraction; unknown lights none', () => {
		expect(creditTicks(null)).toBe(0);
		expect(creditTicks({ remaining: 5000, total: 10000 })).toBe(10);
		expect(creditTicks({ remaining: 1, total: 0 })).toBe(0);
	});
});

describe('delivery shade', () => {
	it('is a quiet grey near zero and scales red with the day share', () => {
		expect(deliveryShade(0)).toBe('rgba(255,255,255,0.07)');
		expect(deliveryShade(1)).toBe('rgba(255,69,87,0.98)');
	});
});

describe('railOrder', () => {
	it('puts active campaigns first and filters by name', () => {
		const cs = [
			{ id: 'a', name: 'Paused one', status: 'paused' },
			{ id: 'b', name: 'Live one', status: 'active' }
		] as Campaign[];
		expect(railOrder(cs, '').map((c) => c.id)).toEqual(['b', 'a']);
		expect(railOrder(cs, 'PAUSED').map((c) => c.id)).toEqual(['a']);
	});
});

describe('forYou', () => {
	it('drops saved ads and ranks niche + longevity + live', () => {
		const ads = [
			wall('plain', { hook: 'buy shoes today', daysRunning: 5, live: false }),
			wall('niche', { hook: 'AI agents for your agency clients', daysRunning: 90 }),
			wall('saved', { hook: 'AI automation' })
		];
		const ranked = forYou(ads, [ads[2]]);
		expect(ranked.map((r) => r.ad.id)).toEqual(['niche', 'plain']);
		expect(ranked[0].score).toBeCloseTo(3 * 1 + 3 * 1 + 0 + 1);
	});
	it('saved taste pulls similar driver profiles up', () => {
		const saved = [wall('s', { drivers: { urgency: 9 } })];
		const a = wall('a', { drivers: { urgency: 8 }, daysRunning: 0, live: false });
		const b = wall('b', { drivers: { trust: 8 }, daysRunning: 0, live: false });
		expect(forYou([b, a], saved)[0].ad.id).toBe('a');
	});
});

describe('relatedAds', () => {
	it('returns the ads an answer is about, best first, at most 3', () => {
		const ads = [
			wall('x', { hook: 'Hire founders fast', brand: 'Skool', daysRunning: 100 }),
			wall('y', { hook: 'Nothing related', brand: 'Other', live: false, daysRunning: 0 })
		];
		expect(relatedAds('Skool founders win', ads).map((a) => a.id)).toEqual(['x']);
	});
});

describe('groupSignals', () => {
	it('groups by day, newest day first', () => {
		const s = (at: string): Signal => ({ type: 'winner', brandId: 'b', brandName: 'B', message: at, at });
		const g = groupSignals([s('2026-09-28T01:00:00Z'), s('2026-09-29T01:00:00Z'), s('2026-09-28T05:00:00Z')]);
		expect(g.map(([d, rows]) => [d, rows.length])).toEqual([
			['2026-09-29', 1],
			['2026-09-28', 2]
		]);
	});
});

describe('leadWithImage', () => {
	it('promotes the first ad with a still into the 2x2 slot', () => {
		const ads = [wall('a'), wall('b', { thumbnail: 't.jpg' }), wall('c')];
		expect(leadWithImage(ads).map((a) => a.id)).toEqual(['b', 'a', 'c']);
		expect(leadWithImage([wall('a')]).map((a) => a.id)).toEqual(['a']);
	});
});
