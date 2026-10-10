import { readFileSync, readdirSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AdpilotPayload, Campaign, WallAd } from './types';

const fetchMock = vi.fn();
vi.mock('$lib/founderos/api', () => ({ founderosFetch: (...a: unknown[]) => fetchMock(...a) }));

import AdPilotRoute from '../../../../routes/(founderos)/os/adpilot/+page.svelte';

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
		daysRunning: 42,
		hook: 'AI agents replace your front desk',
		hookSource: 'spoken',
		transcript: [{ t: 0, s: 'AI agents replace your front desk' }],
		drivers: { urgency: 8 },
		ctaType: 'LEARN_MORE',
		linkUrl: null,
		...over
	};
}

function emptyPayload(over: Partial<AdpilotPayload['library']> = {}): AdpilotPayload {
	return {
		deck: { campaigns: [], syncedAt: null, liveCount: 0, views: null },
		library: { wall: [], watchlist: [], saved: [], signals: [], credits: null, lastSyncAt: null, configured: false, ...over },
		errors: {}
	};
}

const metrics = (spend: number, leads: number) => ({
	spend,
	impressions: 10000,
	clicks: 200,
	leads,
	bookings: 4,
	purchases: 0,
	revenue: spend * 2,
	ctr: 0.02,
	cpl: spend / leads,
	costPerBooking: spend / 4,
	costPerResult: spend / 4,
	roas: 2
});

function campaignPayload(): AdpilotPayload {
	const c = (id: string, name: string, status: 'active' | 'paused', spend: number): Campaign => ({
		id,
		name,
		objective: 'bookings',
		status,
		platform: 'Meta',
		period: { from: '2026-09-01', to: '2026-09-30' },
		spend,
		impressions: 10000,
		clicks: 200,
		leads: 20,
		bookings: 4,
		purchases: 0,
		revenue: spend * 2,
		audience: { age: { '25-34': 60 }, gender: { male: 70 }, placements: { Reels: 100 } },
		geo: [{ city: 'Austin', country: 'US', lat: 30, lng: -97, leads: 20, bookings: 4 }]
	});
	const audience = { age: { '25-34': 60 }, gender: { male: 70 }, placements: { Reels: 100 } };
	const p = emptyPayload({ configured: true, credits: { remaining: 9640, total: 10000 }, lastSyncAt: new Date().toISOString(), wall: [wall('a1')] });
	p.deck = {
		campaigns: [c('c1', 'Vantage intake', 'active', 1200), c('c2', 'LC cohort', 'paused', 800)],
		syncedAt: '2026-09-30T00:00:00Z',
		liveCount: 1,
		views: {
			all: { metrics: metrics(2000, 40), geo: [], audience, daily: [] },
			c1: { metrics: metrics(1200, 20), geo: [], audience, daily: [] },
			c2: { metrics: metrics(800, 20), geo: [], audience, daily: [] }
		}
	};
	return p;
}

beforeEach(() => {
	fetchMock.mockReset();
	vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('no network in tests')));
});
afterEach(() => vi.unstubAllGlobals());

describe('/os/adpilot page', () => {
	it('shows loading, then an honest empty deck: no fake zeros, credits not connected', async () => {
		let resolve!: (v: AdpilotPayload) => void;
		fetchMock.mockReturnValueOnce(new Promise((r) => (resolve = r)));
		const { container } = render(AdPilotRoute);
		expect(container.querySelector('[data-part="loading"]')).toBeTruthy();
		resolve(emptyPayload());
		await screen.findAllByText('No ad account connected');
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock).toHaveBeenCalledWith('/pages/adpilot');
		// like production, the floaters sum the (empty) campaign set: 0, not unknown
		const floats = [...container.querySelectorAll('[data-part="stat-float"]')];
		expect(floats.map((f) => f.lastElementChild!.textContent!.trim())).toEqual(['0', '0']);
		expect(container.querySelectorAll('[data-part="stat-float"] [data-unknown]').length).toBe(0);
		// credits read "not connected" in the text colour, as production does
		expect(screen.getByText('not connected').closest('.bn-dim')).toBeNull();
		expect(screen.getByText('last synced never')).toBeTruthy();
		expect(screen.getByText('0 tracked ads')).toBeTruthy();
	});

	it('a sync that answered with zero campaigns reads 0 like production, not unknown', async () => {
		const p = emptyPayload();
		p.deck.syncedAt = '2026-09-30T00:00:00Z';
		fetchMock.mockResolvedValueOnce(p);
		const { container } = render(AdPilotRoute);
		await screen.findAllByText('No ad account connected');
		const floats = [...container.querySelectorAll('[data-part="stat-float"]')];
		expect(floats.length).toBe(2);
		expect(container.querySelectorAll('[data-part="stat-float"] [data-unknown]').length).toBe(0);
		expect(floats.map((f) => f.lastElementChild!.textContent!.trim())).toEqual(['0', '0']);
	});

	it('keyed but never synced reads credits unknown, not 0', async () => {
		fetchMock.mockResolvedValueOnce(emptyPayload({ configured: true }));
		render(AdPilotRoute);
		expect(await screen.findByText('unknown until first sync')).toBeTruthy();
	});

	it('renders the deck and switches the view with the campaign rail', async () => {
		fetchMock.mockResolvedValueOnce(campaignPayload());
		const { container } = render(AdPilotRoute);
		await screen.findByText('1 active campaign');
		const spend = () => container.querySelector('[data-part="spend"] .text-\\[36px\\]')!.textContent;
		expect(spend()).toBe('$2,000');
		expect(screen.getByText('9,640')).toBeTruthy();
		const rail = container.querySelector('[data-part="rail"]') as HTMLElement;
		await fireEvent.click(within(rail).getByText('LC cohort'));
		expect(spend()).toBe('$800');
		expect(within(container.querySelector('[data-part="roas"]') as HTMLElement).getByText('2.00x')).toBeTruthy();
		expect(container.querySelector('[data-part="delivery"]')!.textContent).toContain('Daily delivery arrives with the ad-account sync.');
	});

	it('shows an error with a retry when the payload fails', async () => {
		fetchMock.mockRejectedValueOnce(new Error('backend down (HTTP 502)'));
		render(AdPilotRoute);
		expect(await screen.findByText('backend down (HTTP 502)')).toBeTruthy();
		fetchMock.mockResolvedValueOnce(emptyPayload());
		await fireEvent.click(screen.getByText('Retry'));
		await screen.findAllByText('No ad account connected');
	});

	it('surfaces an unreadable store instead of reading it as empty', async () => {
		const p = emptyPayload();
		p.errors = { store: 'adscout store watchlist.json: bad json' };
		fetchMock.mockResolvedValueOnce(p);
		render(AdPilotRoute);
		expect(await screen.findByText(/Ad store unreadable: adscout store watchlist.json/)).toBeTruthy();
	});

	it('opens the library, saves an ad through the saved route', async () => {
		fetchMock.mockResolvedValueOnce(campaignPayload());
		render(AdPilotRoute);
		await fireEvent.click(await screen.findByText('Open ad library'));
		expect(screen.getByRole('tab', { name: 'For you' })).toBeTruthy();
		fetchMock.mockResolvedValueOnce({ ok: true, saved: [{ ad: wall('a1'), savedAt: 'x' }] });
		await fireEvent.click(screen.getByLabelText('save ad'));
		expect(fetchMock).toHaveBeenLastCalledWith('/pages/adscout/saved', { method: 'POST', json: wall('a1') });
		// a saved ad leaves the For-you feed (saves teach it) and lands in Saved
		await waitFor(() => expect(screen.queryByLabelText('save ad')).toBeNull());
		await fireEvent.click(screen.getByRole('tab', { name: /Saved/ }));
		expect(screen.getByLabelText('unsave ad')).toBeTruthy();
	});

	it('asks Adscout only on submit and shows the answer', async () => {
		fetchMock.mockResolvedValueOnce(campaignPayload());
		render(AdPilotRoute);
		const input = await screen.findByLabelText('question for Adscout');
		expect(fetchMock).toHaveBeenCalledTimes(1);
		await fireEvent.input(input, { target: { value: 'what works for agents?' } });
		fetchMock.mockResolvedValueOnce({ ok: true, answer: 'Acme runs AI agents hooks for 42 days.' });
		await fireEvent.click(screen.getByLabelText('ask Adscout'));
		expect(fetchMock).toHaveBeenLastCalledWith('/pages/adscout/ask', { method: 'POST', json: { question: 'what works for agents?' } });
		expect(await screen.findByText('Acme runs AI agents hooks for 42 days.')).toBeTruthy();
		expect(screen.getByText('From the library')).toBeTruthy();
	});

	it('refresh runs the sync, then reloads the page payload', async () => {
		fetchMock.mockResolvedValueOnce(emptyPayload({ configured: true }));
		render(AdPilotRoute);
		await screen.findByText('Refresh data');
		fetchMock.mockResolvedValueOnce({ ok: true, brands: 2, signals: [1, 2, 3] });
		fetchMock.mockResolvedValueOnce(emptyPayload({ configured: true, credits: { remaining: 10, total: 100 } }));
		await fireEvent.click(screen.getByText('Refresh data'));
		expect(fetchMock).toHaveBeenCalledWith('/pages/adscout/sync', { method: 'POST' });
		expect(await screen.findByText('10')).toBeTruthy();
	});
});

// Ported from FounderOS v1 tests/adpilot-page.test.ts: the layout/theme contract.
const dir = __dirname;
const read = (f: string) => readFileSync(resolve(dir, f), 'utf8');
const route = readFileSync(resolve(dir, '../../../../routes/(founderos)/os/adpilot/+page.svelte'), 'utf8');

describe('/os/adpilot contract', () => {
	it('the deck says "No ad account connected"; the library says "not connected"', () => {
		expect(read('AdPilotDeck.svelte')).toContain('No ad account connected');
		expect(read('AdLibrary.svelte')).toContain('not connected');
	});

	it('the route renders the real view, not the porting placeholder', () => {
		expect(route).toContain('AdPilotView');
		expect(route).not.toMatch(/PortingPlaceholder|Porting in progress|bridge:partial/);
	});

	it('the red accent is scoped to the page wrapper', () => {
		const view = read('AdPilotView.svelte');
		expect(view).toContain('data-adpilot');
		expect(view).toMatch(/'--bn-accent: #ff4557'/);
	});

	it('one face on the page: mono-classed text keeps JetBrains with normal tracking (production rule)', () => {
		const css = read('adpilot.css');
		expect(css).toMatch(/\[data-adpilot\] \.font-mono\s*\{[^}]*font-family: var\(--bn-font\) !important;[^}]*letter-spacing: normal;/);
	});

	it('Tailwind border colours apply on this page (the app-wide `* { border-color }` is reverted to the utility layer)', () => {
		expect(read('adpilot.css')).toMatch(/\[data-adpilot\] :where\(\[class\*='border-'\]\)\s*\{\s*border-color: revert-layer;/);
	});

	it("buttons read like production's (unstyled c-btn): no fill, no frame, no default padding; pressable lens on every control", () => {
		const css = read('adpilot.css');
		const btn = /\[data-adpilot\] \.ap-btn,\s*\[data-adpilot\] \.ap-btn-primary\s*\{([^}]*)\}/.exec(css)![1];
		expect(btn).toMatch(/border: 0;/);
		expect(btn).toMatch(/background: none;/);
		expect(btn).not.toMatch(/padding/);
		// no other rule re-dresses them (no red fill, no frame, no dimmed disabled state)
		expect(css.match(/\.ap-btn(-primary)?[^{,]*\{/g)).toHaveLength(1);
		expect(read('AdLibrary.svelte')).toMatch(/onclick=\{runSync\}[^>]*class="bn-pressable ap-btn ap-btn-primary flex items-center justify-center gap-2 py-2\.5 text-\[13px\]"/);
		expect(read('AdLibrary.svelte')).toMatch(/class="bn-pressable ap-bright /);
		expect(read('AskWidget.svelte')).toMatch(/aria-label="ask Adscout" class="bn-pressable ap-btn px-3 py-2"/);
		for (const f of ['SearchTab.svelte', 'CardGrid.svelte', 'ActivityTab.svelte', 'Dossier.svelte', 'WatchlistTab.svelte', 'AdPilotDeck.svelte']) {
			expect(read(f), f).toContain('bn-pressable');
		}
	});

	it('production copy for the running states', () => {
		expect(read('SearchTab.svelte')).toContain('expanding concept → probing discovery → filtering to proven ads…');
		expect(read('AskWidget.svelte')).toContain('Checking the library: hooks, longevity, drivers, signals…');
	});

	it('the ap- surface classes are defined, scoped under [data-adpilot]', () => {
		const css = read('adpilot.css');
		for (const cls of ['ap-panel', 'ap-chip', 'ap-hatch', 'ap-glow', 'ap-frost', 'ap-bright', 'ap-orbit']) {
			expect(css, cls).toMatch(new RegExp(`\\[data-adpilot\\] \\.${cls}\\s*\\{`));
		}
		expect(css).toContain('@keyframes ap-spin');
		expect(css).toContain('prefers-reduced-motion');
	});

	const files = [...readdirSync(dir).filter((f) => /\.(svelte|ts|css)$/.test(f) && !f.endsWith('.test.ts')), '+page.svelte'];
	it.each(files)('house rules: %s', (f) => {
		const src = f === '+page.svelte' ? route : read(f);
		expect(src).not.toContain('—');
		expect(src).not.toMatch(/transition-(colors|all)\b/);
		expect(src).not.toMatch(/\bglados\b/i);
		expect(src).not.toMatch(/\p{Extended_Pictographic}/u);
	});
});

describe('globe math', () => {
	it('projects lat/lng onto the unit sphere and keeps the top six chips', async () => {
		const { latLngToVec, rotate, globeChips } = await import('./globe');
		const v = latLngToVec(0, 0);
		expect(v.x).toBeCloseTo(1);
		expect(v.z).toBeCloseTo(0);
		const r = rotate(latLngToVec(0, 90), Math.PI / 2, 0);
		expect(r.x).toBeCloseTo(-1);
		const geo = Array.from({ length: 8 }, (_, i) => ({ city: `c${i}`, country: 'US', lat: 0, lng: 0, leads: i, bookings: 0 }));
		expect(globeChips(geo).map((g) => g.city)).toEqual(['c7', 'c6', 'c5', 'c4', 'c3', 'c2']);
	});
});

void join;

describe('AdPilot copy matches FounderOS v1', () => {
	it("the remake brief asks for the user's voice (v1 AdLibrary.tsx), not a rename artifact", () => {
		const dossier = readFileSync(resolve(__dirname, 'Dossier.svelte'), 'utf8');
		expect(dossier).toContain("in the user's voice");
		expect(dossier).not.toContain("operator's voice");
	});
});
