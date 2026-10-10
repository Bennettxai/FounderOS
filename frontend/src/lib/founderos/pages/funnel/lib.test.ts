import { describe, expect, it } from 'vitest';
import { decayedColor, decayedOpacity, easeInOut, orbitSpread, rnd, smoothK, usd } from './viz';
import { trakyoVolume, vslForPeriod, vslPeriods, vslRateMeters, vslVolume } from './volume';
import type { TrakyoFunnel, VslSnapshot } from './types';

// Ported from FounderOS v1 tests/funnel.test.ts (orbitSpread), tests/channel-volume.test.ts
// and tests/vsl-analytics.test.ts. An empty hand-off reads "none yet" over an empty
// meter; an unexported rate is frac null (the kit draws no bar), never a fake 0.

describe('funnel viz math', () => {
	it('rnd is deterministic and quantized to 4 decimals', () => {
		expect(rnd(3, 1)).toBe(rnd(3, 1));
		expect(String(rnd(7, 2)).split('.')[1]?.length ?? 0).toBeLessThanOrEqual(4);
		expect(rnd(1, 1)).toBeGreaterThanOrEqual(0);
		expect(rnd(1, 1)).toBeLessThan(1);
	});
	it('a dozen leads keep the tight constellation, a live pipeline spreads', () => {
		expect(orbitSpread(12)).toBe(1);
		expect(orbitSpread(3)).toBe(1);
		expect(orbitSpread(48)).toBeCloseTo(2);
		expect(orbitSpread(1000)).toBe(2.4);
	});
	it('easing, smoothing and the fade-to-red', () => {
		expect(easeInOut(0)).toBe(0);
		expect(easeInOut(1)).toBe(1);
		expect(smoothK(0)).toBe(0);
		expect(smoothK(1000)).toBeCloseTo(smoothK(64));
		expect(decayedColor('var(--fn-s1)', 0, true)).toBe('var(--bn-ok)');
		expect(decayedColor('var(--fn-s1)', 0, false)).toBe('var(--fn-s1)');
		expect(decayedColor('var(--fn-s1)', 1, false)).toBe('color-mix(in oklab, var(--bn-err) 85%, var(--fn-s1))');
		expect(decayedOpacity(1)).toBeCloseTo(0.5);
		expect(usd(1234.4)).toBe('$1,234');
	});
});

const funnel = (counts: Partial<TrakyoFunnel['counts']> = {}, revenue = '4500.5'): TrakyoFunnel => ({
	currency: 'USD',
	attribution: 'first_touch',
	range: { start: '2026-08-25T00:00:00Z', end: '2026-09-23T23:59:59Z', timezone: 'America/Chicago' },
	counts: { clicks: 1200, visits: 800, form_submissions: 40, bookings: 10, closes: 3, ...counts },
	revenue
});

describe('trakyoVolume', () => {
	const v = trakyoVolume(funnel());
	it('first-touch revenue is the headline; closes and bookings ride as chips', () => {
		expect(v.revenue).toBeCloseTo(4500.5);
		expect(v.headline).toBe('$4,500.50');
		expect(v.chips).toEqual([{ tone: 'ok', text: '3 closes' }, { text: '10 bookings' }]);
		expect(v.caption).toBe('first-touch revenue · 1,200 clicks · 800 visits');
		expect(v.foot).toBe('2026-08-25 to 2026-09-23 · America/Chicago · events, not unique people');
	});
	it('three hand-off meters, each a real fraction', () => {
		expect(v.meters.map((m) => m.label)).toEqual(['Visits → forms (40/800)', 'Forms → bookings (10/40)', 'Bookings → closes (3/10)']);
		expect(v.meters.map((m) => m.frac)).toEqual([0.05, 0.25, 0.3]);
		expect(v.meters.map((m) => m.display)).toEqual(['5%', '25%', '30%']);
	});
	it('events are not people: past 100% shows its truth but the bar stops full', () => {
		expect(trakyoVolume(funnel({ bookings: 12, form_submissions: 10 })).meters[1]).toMatchObject({ frac: 1, display: '120%' });
	});
	// Prod tests/channel-volume.test.ts: an empty hand-off is an empty meter
	// that says "none yet" (frac 0, the white label), never a fake rate.
	it('nothing tracked yet reads as empty meters, never a fake rate', () => {
		const e = trakyoVolume(funnel({ clicks: 0, visits: 0, form_submissions: 0, bookings: 0, closes: 0 }, '0'));
		expect(e.meters.every((m) => m.frac === 0)).toBe(true);
		expect(e.meters.map((m) => m.display)).toEqual(['none yet', 'none yet', 'none yet']);
		expect(e.headline).toBe('$0.00');
	});
});

const video = (id: string, plays: number, rates: Partial<VslSnapshot> = {}): VslSnapshot => ({
	videoId: id,
	title: `Video ${id}`,
	duration: '10:00',
	dateFrom: '2026-08-26',
	dateTo: '2026-09-24',
	capturedAt: '2026-09-24T10:00:00.000Z',
	plays,
	uniqueViewers: Math.round(plays / 2),
	impressions: plays * 3,
	playRate: 0.3,
	averageWatched: 0.42,
	unmuteRate: 0.6,
	bounceRate: 0.1,
	recordedConversions: 0,
	recordedRevenue: 0,
	audience: [],
	trend: [],
	trendInterval: 'day',
	source: 'vidalytics-browser-export',
	...rates
});

describe('vslVolume', () => {
	it('plays are the headline; one meter per video, biggest first, as a share of plays', () => {
		const v = vslVolume([video('a', 20), video('b', 60), video('c', 20)]);
		expect(v.plays).toBe(100);
		expect(v.chips).toEqual([{ text: '3 videos' }, { text: '50 unique viewers' }]);
		expect(v.caption).toBe('plays in this saved period · captured 2026-09-24');
		expect(v.meters.map((m) => m.label)).toEqual(['Video b', 'Video a', 'Video c']);
		expect(v.meters[0]).toMatchObject({ frac: 0.6, display: '60 · 60%' });
	});
	it('more than four videos fold the tail into Other', () => {
		const v = vslVolume([1, 2, 3, 4, 5, 6].map((n) => video(`v${n}`, n * 10)));
		expect(v.meters.map((m) => m.label)).toEqual(['Video v6', 'Video v5', 'Video v4', 'Other videos (3)']);
		expect(v.meters[3].frac).toBeCloseTo(60 / 210);
	});
	it('an empty import is empty meters', () => {
		const e = vslVolume([]);
		expect(e.plays).toBe(0);
		expect(e.meters).toEqual([]);
		expect(e.caption).toBe('no video snapshots imported yet');
	});
	it('periods newest window first; rows filter by window', () => {
		const rows = [video('a', 1), video('b', 2, { dateFrom: '2025-08-10' })];
		expect(vslPeriods(rows)).toEqual(['2026-08-26/2026-09-24', '2025-08-10/2026-09-24']);
		expect(vslForPeriod(rows, '2025-08-10/2026-09-24').map((r) => r.videoId)).toEqual(['b']);
	});
});

describe('vslRateMeters', () => {
	it('four rate meters; a rate Vidalytics did not export reads N/A with no bar', () => {
		const m = vslRateMeters(video('a', 10, { unmuteRate: null }));
		expect(m.map((x) => x.label)).toEqual(['Average watched', 'Play rate', 'Unmute rate', 'Bounce rate']);
		expect(m[0]).toMatchObject({ frac: 0.42, display: '42.0%' });
		expect(m[2]).toMatchObject({ frac: null, display: 'N/A' });
		expect(m[3].hue).toBe('var(--bn-warn)');
	});
});

import { funnelHref, funnelQuery, parseFunnelView } from './url';

describe('funnel view ↔ URL', () => {
	it('round-trips every control and drops unknown values', () => {
		const v = parseFunnelView('?venture=vantage&view=archive&stage=engaged&layout=radial&lead=typeform-r1');
		expect(v).toEqual({ venture: 'vantage', view: 'archive', stage: 'engaged', layout: 'radial', lead: 'typeform-r1' });
		expect(funnelHref(v)).toBe('/os/funnel?venture=vantage&view=archive&stage=engaged&layout=radial&lead=typeform-r1');
		expect(parseFunnelView('?venture=acme&stage=won&layout=grid')).toEqual({ venture: null, view: 'live', stage: null, layout: 'flow', lead: null });
		expect(funnelHref(parseFunnelView(''))).toBe('/os/funnel');
	});
	it('only venture and stage reach the backend', () => {
		expect(funnelQuery(parseFunnelView('?venture=vantage&layout=radial&stage=converted'))).toBe('/pages/funnel?venture=vantage&stage=converted');
		expect(funnelQuery(parseFunnelView('?layout=radial'))).toBe('/pages/funnel');
	});
});
