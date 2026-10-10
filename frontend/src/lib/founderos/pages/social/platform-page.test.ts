import { render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import PlatformPage from '../../../../routes/(founderos)/os/social/[platform]/+page.svelte';
import { load } from '../../../../routes/(founderos)/os/social/[platform]/+page';
import type { PlatformPage as Model } from './types';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const model: Model = {
	account: { platform: 'instagram', handle: '@founderosx.ai', url: 'https://instagram.com/founderosx.ai', order: 1 },
	followers: 1250,
	growth: { d7: 1.234, d30: -2.5, d60: null, allTime: 10 },
	snapshots: [
		{ platform: 'instagram', capturedAt: '2026-09-23', followers: 1050, source: 'zernio-config' },
		{ platform: 'instagram', capturedAt: '2026-09-24', followers: 1250, source: 'zernio-live' }
	],
	label: 'Instagram',
	today: '2026-09-24',
	volume: {
		headline: 1250,
		chips: [{ tone: 'ok', text: '+1.23% 7d' }],
		caption: 'Instagram followers · 2 snapshots since Sep 23',
		meters: [{ label: 'Days gained (1/1)', frac: 1, display: '1 of 1', hue: 'var(--bn-ok)' }],
		foot: '2 snapshots · latest Sep 24 · zernio-live',
		series: [{ label: 'Sep 24', count: 200 }],
		intervals: 1,
		gainedDays: 1,
		dippedDays: 0,
		trackedDays: 2,
		net: 200,
		insight: { value: 200, display: '+200', headline: 'Gained 200 followers in the last 30 days.', body: 'up on 1 of 1 tracked days', frac: 1 }
	}
};

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

describe('/os/social/[platform]', () => {
	it('load passes the route param through', async () => {
		expect(await (load as (e: unknown) => unknown)({ params: { platform: 'tiktok' } })).toEqual({ platform: 'tiktok' });
	});

	it('renders the history bars, growth badges, windows and the 30-day card', async () => {
		fetchMock.mockResolvedValue(json(model));
		const { container } = render(PlatformPage, { data: { platform: 'instagram' } });
		await waitFor(() => expect(container.querySelector('[data-part="follower-bars"]')).toBeTruthy());
		expect(String(fetchMock.mock.calls[0][0])).toContain('/pages/social/instagram');
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Instagram');
		expect(container.querySelectorAll('[data-part="follower-bars"] rect').length).toBe(4);
		expect(container.querySelector('[data-growth="60d"]')?.textContent).toContain('—');
		expect(container.querySelector('[data-part="windows"]')?.textContent).toContain('60 days · not enough history');
		expect(screen.getByText('Gained 200 followers in the last 30 days.')).toBeTruthy();
		expect(screen.getByText('Open profile')).toBeTruthy();
	});

	it('an untracked platform says so; a down backend says so', async () => {
		fetchMock.mockResolvedValue(json({ error: 'unknown platform: myspace' }, 404));
		const a = render(PlatformPage, { data: { platform: 'myspace' } });
		await waitFor(() => expect(a.container.querySelector('[data-part="not-found"]')).toBeTruthy());
		a.unmount();
		fetchMock.mockResolvedValue(json({ error: 'db down' }, 500));
		const b = render(PlatformPage, { data: { platform: 'instagram' } });
		await waitFor(() => expect(b.container.querySelector('[data-part="error"]')?.textContent).toContain('db down'));
	});
});

describe('/os/social/[platform] shape (prod PILL, rounded badges and swatches)', () => {
	it('pills are rounded, growth badges 6px, legend swatches round', async () => {
		fetchMock.mockResolvedValue(json(model));
		const { container } = render(PlatformPage, { data: { platform: 'instagram' } });
		await waitFor(() => expect(container.querySelector('[data-growth="7d"]')).toBeTruthy());
		expect(screen.getByText(/All platforms/).closest('a')?.className).toContain('rounded-full');
		expect(container.querySelector('[data-growth="7d"]')?.className).toContain('rounded-[6px]');
		const swatch = screen.getByText(/gained vs prev/).querySelector('span') as HTMLElement;
		expect(swatch.className).toContain('rounded-full');
	});
});
