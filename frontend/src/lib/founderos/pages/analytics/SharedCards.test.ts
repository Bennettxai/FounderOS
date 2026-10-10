// The /os/analytics cards that FounderOS v1 shares with /funnel
// (components/FunnelChannelAnalytics.tsx, VslAnalytics.tsx, TrakyoLinkStudio.tsx),
// held to production's shape: rounded pills and boxes, lowercase count tags,
// the selected video row lit, and the link studio's two forms with their
// outward writes held by the bridge guard.
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ChannelAnalyticsBody, VslSnapshot } from '../funnel/types';

vi.mock('$lib/founderos/api', () => {
	class FounderosApiError extends Error {
		status: number;
		body: unknown;
		constructor(status: number, message: string, body?: unknown) {
			super(message);
			this.status = status;
			this.body = body;
		}
	}
	const isGuardRefusal = (e: unknown) => e instanceof FounderosApiError && (e.body as { guarded?: boolean })?.guarded === true;
	return { founderosFetch: vi.fn(), FounderosApiError, isGuardRefusal };
});

const { founderosFetch, FounderosApiError } = await import('$lib/founderos/api');
const fetchMock = vi.mocked(founderosFetch);
const FunnelChannelAnalytics = (await import('../funnel/FunnelChannelAnalytics.svelte')).default;
const VslAnalytics = (await import('../funnel/VslAnalytics.svelte')).default;
const TrakyoLinkStudio = (await import('../funnel/TrakyoLinkStudio.svelte')).default;

const snap = (id: string, plays: number): VslSnapshot => ({
	videoId: id,
	title: `Video ${id}`,
	duration: '07:23',
	dateFrom: '2025-08-10',
	dateTo: '2026-09-24',
	capturedAt: '2026-09-24T00:00:00Z',
	plays,
	uniqueViewers: plays - 1,
	impressions: plays * 2,
	playRate: 0.9,
	averageWatched: 0.1,
	unmuteRate: 0.3,
	bounceRate: 0.3,
	recordedConversions: 0,
	recordedRevenue: 0,
	audience: [
		{ second: 0, viewers: 10 },
		{ second: 60, viewers: 4 }
	],
	trend: [{ date: '2025-08-01', plays: 3 }],
	trendInterval: 'month',
	source: 'vidalytics-browser-export'
});

const row = { id: 'ci_1', type: 'custom', name: 'IG bio link', source: 'Instagram', views: null, published_at: null, clicks: 1, visits: 0, form_submissions: 0, bookings: 0, revenue: { first_touch: '0', last_touch: '0' } };
const analytics = (): ChannelAnalyticsBody => ({
	period: '30d',
	fetchedAt: '2026-09-30T12:00:00Z',
	content: { state: 'ready', rows: [row], message: null },
	funnel: { state: 'ready', data: { currency: 'USD', attribution: 'first_touch', range: { start: '2026-09-02T00:00:00Z', end: '2026-10-01T00:00:00Z', timezone: 'America/Chicago' }, counts: { clicks: 287, visits: 123, form_submissions: 0, bookings: 0, closes: 1 }, revenue: '970' }, message: null },
	channels: [{ id: 'instagram_bio', label: 'IG bio', items: [row], clicks: 1, visits: 0, forms: 0, bookings: 0, views: null, videosWithViews: 0, revenue: { first_touch: 0, last_touch: 0 } }]
});

const workspace = { state: 'ready', scopes: ['content:write', 'links:write'], messages: [], domains: [{ id: 'd1', hostname: 'go.example.com' }], links: [], hasMore: false };

function route(map: Record<string, unknown>) {
	fetchMock.mockImplementation(async (path: string) => {
		for (const [prefix, v] of Object.entries(map)) {
			if (path.startsWith(prefix)) {
				if (v instanceof Error) throw v;
				return v as never;
			}
		}
		throw new Error(`unmocked ${path}`);
	});
}

beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.clearAllMocks());

describe('Channel Funnels (prod shape)', () => {
	it('rounded period select and Refresh pill, rounded channel boxes, lowercase revenue tag', async () => {
		route({ '/pages/funnel/analytics': analytics() });
		render(FunnelChannelAnalytics, { showVsl: false });
		const box = await screen.findByRole('button', { name: /IG bio/ });
		expect(box.className).toContain('rounded-[10px]');
		expect(box.getAttribute('data-lens')).toBe('c');
		expect(screen.getByLabelText('Analytics period').className).toContain('rounded-full');
		expect(screen.getByRole('button', { name: /Refresh/ }).className).toContain('rounded-full');
		const tag = screen.getAllByText('$0.00').find((el) => el.tagName === 'SPAN')!;
		expect(tag.className).toContain('rounded-full');
		expect(tag.className).not.toContain('uppercase');
	});

	it('an empty hand-off reads "none yet", not unknown', async () => {
		route({ '/pages/funnel/analytics': analytics() });
		render(FunnelChannelAnalytics, { showVsl: false });
		const label = await screen.findByText('Forms → bookings (0/0)');
		expect(label.parentElement?.textContent).toContain('none yet');
		expect(label.parentElement?.textContent).not.toContain('unknown');
	});
});

describe('VSL Performance (prod shape)', () => {
	it('counts the videos in words, lights the selected row and keeps plays tags lowercase and round', async () => {
		render(VslAnalytics, { snapshots: [snap('a', 30), snap('b', 70), snap('c', 5), snap('d', 4), snap('e', 3)], igVideo: 'a' });
		expect(await screen.findByText('Five videos, with their watch behavior preserved from Vidalytics. Pick one to inspect.')).toBeTruthy();
		const on = screen.getByRole('button', { name: /Video a/, pressed: true });
		expect(on.getAttribute('data-on')).not.toBeNull();
		expect(screen.getByRole('button', { name: /Video b/ }).getAttribute('data-on')).toBeNull();
		const tag = within(on).getByText('30 plays');
		expect(tag.className).toContain('rounded-full');
		expect(tag.className).not.toContain('uppercase');
		expect(screen.getByLabelText('Saved video period').className).toContain('rounded-full');
	});

	it('the retention curve draws itself', async () => {
		const { container } = render(VslAnalytics, { snapshots: [snap('a', 30)], igVideo: null });
		await screen.findByText('VSL Performance');
		const line = container.querySelector('polyline')!;
		expect(line.getAttribute('pathLength')).toBe('1');
		expect(line.getAttribute('class')).toContain('bn-draw');
	});
});

describe('Create content & tracking links (prod shape, writes held)', () => {
	it('opens to the two prod forms, and a create is held by the bridge guard with an honest note', async () => {
		route({
			'/pages/trakyo/workspace': workspace,
			'/pages/trakyo/content': new FounderosApiError(409, 'refused', { error: 'FOUNDEROS_WRITES=0', refused: true, guarded: true })
		});
		const { container } = render(TrakyoLinkStudio, { content: [] });
		const details = container.querySelector('details')!;
		details.open = true;
		await fireEvent(details, new Event('toggle'));
		expect(await screen.findByText('1. Register the content')).toBeTruthy();
		expect(screen.getByText('2. Create its tracking link')).toBeTruthy();
		expect(screen.getByText(/does not publish a social post/)).toBeTruthy();
		await fireEvent.input(screen.getByLabelText('Content name'), { target: { value: 'IG Reel' } });
		await fireEvent.click(screen.getByRole('button', { name: 'Create content in Trakyo' }));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/trakyo/content', { method: 'POST', json: { source: 'Instagram', name: 'IG Reel' } }));
		expect((await screen.findByRole('alert')).textContent).toMatch(/writes are off/i);
		expect(screen.getByRole('button', { name: 'Create tracking link' })).toBeTruthy();
	});
});
