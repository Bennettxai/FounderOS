import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { AnalyticsBody } from './types';

vi.mock('$lib/founderos/api', () => {
	class FounderosApiError extends Error {
		status = 500;
	}
	return { founderosFetch: vi.fn(), FounderosApiError };
});

const { founderosFetch } = await import('$lib/founderos/api');
const fetchMock = vi.mocked(founderosFetch);
const AnalyticsPage = (await import('./AnalyticsPage.svelte')).default;
const RunVolumeCard = (await import('./RunVolumeCard.svelte')).default;

function days(n: number) {
	return Array.from({ length: n }, (_, i) => ({ date: `2026-09-${String(i + 1).padStart(2, '0')}`, count: i === n - 1 ? 3 : i % 5 === 0 ? 1 : 0 }));
}

function body(over: Partial<AnalyticsBody> = {}): AnalyticsBody {
	return {
		today: '2026-09-30',
		runVolume: days(30),
		runs30d: 9,
		runsKnown: true,
		volume: {
			reach: 12000,
			headline: '12,000',
			chips: [{ tone: 'ok', text: '+2.5% 7d' }],
			caption: 'total reach across 2 channels',
			meters: [{ label: 'Instagram', frac: 0.75, display: '9,000 · 75%', hue: 'var(--ramp-1)' }],
			foot: 'reach = followers + email list · 2 of 8 metrics live',
			runs: { total: 9, ok: 8, failed: 1, okPct: 89, agents: 2, meters: [{ label: 'Scout (7)', frac: 0.78, display: '78%', hue: 'var(--ramp-1)' }] },
			rhythm: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((label, i) => ({ label, count: i === 2 ? 4 : 1 })),
			rhythmTotal: 10,
			insight: { value: 6, headline: '6 metrics waiting on credentials', body: 'Subscribers · Stripe Available · Unread', frac: 0.25 }
		},
		live: [
			{ id: 'audience', label: 'Audience', unit: 'followers', source: '7d · Zernio', value: 12000, delta: 2.5, deltaPct: true, live: true, spark: [1, 2, 3], sparkReal: true },
			{ id: 'leads', label: 'Leads · 30d', unit: 'leads', source: 'Typeform', value: 14, delta: 0, deltaPct: false, live: true, spark: [1, 1, 1, 1, 1, 1, 1], sparkReal: false }
		],
		pending: [{ id: 'stripe', label: 'Stripe Available', unit: 'usd', source: 'Stripe', value: 0, delta: 0, deltaPct: false, live: false }],
		audience: {
			known: true,
			totalFollowers: 12000,
			asOf: '2026-09-30',
			growth7d: 2.5,
			platforms: [
				{ platform: 'instagram', label: 'Instagram', handle: '@founderosx.ai', url: null, followers: 9000, share: 75, d7: 1.2, bars: [8000, 9000] },
				{ platform: 'tiktok', label: 'TikTok', handle: '@founderosx.ai', url: null, followers: null, share: null, d7: null, bars: null }
			]
		},
		errors: {},
		...over
	};
}

function route(page: unknown) {
	fetchMock.mockImplementation(async (path: string) => {
		if (path.startsWith('/pages/funnel/analytics'))
			return { period: '30d', fetchedAt: '2026-09-30T00:00:00Z', content: { state: 'not_configured', rows: [], message: null }, funnel: { state: 'not_configured', data: null, message: null }, channels: [] } as never;
		if (path === '/pages/analytics') {
			if (page instanceof Error) throw page;
			return page as never;
		}
		throw new Error(`unmocked ${path}`);
	});
}

beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.clearAllMocks());

describe('/os/analytics page', () => {
	it('loads, then shows every section from real reads', async () => {
		route(body());
		render(AnalyticsPage);
		const skel = screen.getByTestId('analytics-loading');
		expect([...skel.children].every((b) => b.className.includes('rounded-[12px]'))).toBe(true);
		expect(await screen.findByText('Reach Volume')).toBeTruthy();
		expect(screen.getByRole('heading', { level: 1, name: 'Analytics' })).toBeTruthy();
		expect(screen.getByText(/12,000 reach · 2 live · 1 pending · 9 runs in 30 days/)).toBeTruthy();
		for (const title of ['Agent Run Volume', 'Runs by Agent', 'Run Rhythm', 'Awaiting credentials', 'Operating Metrics', 'Audience']) {
			expect(screen.getAllByText(title).length).toBeGreaterThan(0);
		}
		expect(screen.getByText('Busiest:')).toBeTruthy();
		expect(screen.getAllByTestId('metric-tile')).toHaveLength(2);
		expect(screen.getByText('Stripe Available · Stripe')).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledWith('/pages/analytics');
	});

	it('v1 section order and nothing else: no channel funnels or VSL cards (private-build only)', async () => {
		route(body());
		const { container } = render(AnalyticsPage);
		await screen.findByText('Reach Volume');
		const titles = [...container.querySelectorAll('h2, h3')].map((h) => h.textContent?.trim()).filter(Boolean);
		expect(titles.some((t) => /Channel Funnels|VSL Performance/.test(t!))).toBe(false);
		expect(screen.queryByText('Channel Funnels')).toBeNull();
		expect(screen.queryByText('VSL Performance')).toBeNull();
		expect(fetchMock.mock.calls.map((c) => c[0])).toEqual(['/pages/analytics']);
		const order = ['Agent Run Volume', 'Reach Volume', 'Runs by Agent', 'Run Rhythm', 'Operating Metrics', 'Audience'];
		const text = container.textContent!;
		const at = order.map((t) => text.indexOf(t));
		expect(at.every((i) => i >= 0)).toBe(true);
		expect([...at].sort((a, b) => a - b)).toEqual(at);
	});

	it('an unmeasured platform reads unknown, never zero', async () => {
		route(body());
		render(AnalyticsPage);
		const audience = await screen.findByTestId('audience');
		const tiktok = within(audience).getByText('TikTok').closest('a')!;
		expect(tiktok.textContent).toContain('—');
		expect(tiktok.textContent).toContain('no history yet');
		expect(tiktok.textContent).toContain('7d —');
	});

	it('names what could not be read', async () => {
		route(body({ errors: { runs: 'founderos Postgres is not wired' }, runsKnown: false }));
		render(AnalyticsPage);
		expect(screen.queryByText(/bridge/)).toBeNull();
		expect((await screen.findByText(/^could not read:/)).textContent).toContain('runs (founderos Postgres is not wired)');
		expect(screen.getByText('the run log could not be read')).toBeTruthy();
	});

	it('an unreachable backend is an error state', async () => {
		route(new Error('HTTP 502'));
		render(AnalyticsPage);
		expect((await screen.findByRole('alert')).textContent).toContain('HTTP 502');
		expect(screen.queryByText('Reach Volume')).toBeNull();
	});
});

describe('/os/analytics in production\'s shape', () => {
	it('round header pills: the count, Open Social, the accent Wire connectors', async () => {
		route(body());
		render(AnalyticsPage);
		await screen.findByText('Reach Volume');
		expect(screen.getAllByText('2 live · 1 pending')[0].className).toContain('rounded-full');
		const social = screen.getAllByRole('link', { name: 'Open Social' });
		expect(social.every((a) => a.className.includes('rounded-full'))).toBe(true);
		expect(screen.getByRole('link', { name: 'Wire connectors' }).className).toContain('bn-pill-accent');
		expect(screen.getByRole('link', { name: 'wire connectors → flip to live' }).className).toContain('rounded-full');
		expect(screen.getByText('Busiest:').className).toContain('rounded-full');
		// prod's body is antialiased; without it every weight renders heavier
		expect(screen.getByTestId('analytics-root').className).toContain('antialiased');
	});

	it('metric tiles are rounded lens rows; a long source stays on one line', async () => {
		const brain = { id: 'brain', label: 'Brain Pages', unit: 'pages', source: 'Optimal Engine · 2 engines · source packages', value: 1423, delta: 0, deltaPct: false, live: true, spark: [1, 1], sparkReal: false };
		route(body({ live: [...body().live, brain] }));
		render(AnalyticsPage);
		const tiles = await screen.findAllByTestId('metric-tile');
		expect(tiles[0].className).toContain('rounded-[12px]');
		expect(tiles[0].getAttribute('data-lens')).toBe('r');
		const src = within(tiles[2]).getByText('Optimal Engine · 2 engines · source packages');
		expect(src.className).toContain('truncate');
		expect(src.getAttribute('title')).toBe('Optimal Engine · 2 engines · source packages');
	});

	it('audience rows: round icon, lowercase share tag, uppercase 7d tag', async () => {
		route(body());
		render(AnalyticsPage);
		const audience = await screen.findByTestId('audience');
		const ig = within(audience).getByText('Instagram').closest('a')!;
		expect(ig.getAttribute('data-lens')).toBe('r');
		const share = within(ig).getByText('75% of reach');
		expect(share.className).toContain('rounded-full');
		expect(share.className).not.toContain('uppercase');
		const d7 = within(ig).getByText('7d +1.20%');
		expect(d7.className).toContain('uppercase');
		expect(d7.className).toContain('rounded-full');
		expect(ig.querySelector('[data-part="platform-icon"]')?.className).toContain('rounded-full');
	});
});

describe('RunVolumeCard', () => {
	it('range pills are prod chips; bars are rounded, glow and grow in', async () => {
		const { container } = render(RunVolumeCard, { data: days(30) });
		expect(screen.getByRole('button', { name: '14d' }).className).toContain('is-on');
		expect(screen.getByRole('button', { name: '7d' }).className).toContain('rounded-full');
		const bars = [...container.querySelectorAll('[data-part="run-bar"]')] as HTMLElement[];
		expect(bars).toHaveLength(14);
		expect(bars[0].className).toContain('rounded-t-[4px]');
		expect(bars[0].className).toContain('bn-grow');
		const lit = bars.find((b) => b.style.boxShadow);
		expect(lit?.style.boxShadow).toContain('0 0 12px');
		expect(bars.filter((b) => b.style.boxShadow).length).toBeLessThan(14);
	});

	it('range chips re-window the real run log', async () => {
		render(RunVolumeCard, { data: days(30) });
		expect(screen.getByText('agent runs in the last 14 days, from the real run log')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: '30d' }));
		expect(await screen.findByText('agent runs in the last 30 days, from the real run log')).toBeTruthy();
		expect(screen.getByText(/of 30 days active/)).toBeTruthy();
	});
});
