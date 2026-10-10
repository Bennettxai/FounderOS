import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import BeehiivPage from '../../../../routes/(founderos)/os/social/beehiiv/+page.svelte';
import type { BeehiivPage as Model } from './types';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const issue = { id: 'p1', title: 'Ship the system', publishedAt: '2026-09-01T15:00:00.000Z', webUrl: 'https://news/p1', recipients: 1000, delivered: 980, deliveryRate: 98, opens: 294, openRate: 30, clicks: 30, clickRate: 3, unsubscribes: 5, unsubscribeRate: 0.5, spamReports: 1, webViews: 4 };

const model = (over: Partial<Model> = {}): Model => ({
	newsletters: [issue],
	seeded: false,
	subscribers: 24814,
	live: true,
	fresh: true,
	volume: {
		headline: 24814,
		chips: [{ tone: 'accent', text: '1 issue' }],
		caption: 'subscribers, live via Beehiiv',
		meters: [{ label: 'Delivered', frac: 0.98, display: '98.0% of sends', hue: 'var(--bn-ok)' }],
		foot: '1 issue · 1,000 sends',
		sends: [{ label: 'Sep 1', count: 1000 }],
		openRates: [{ label: 'Sep 1', count: 30 }],
		avgOpenRate: 30,
		totalRecipients: 1000,
		totalClicks: 30,
		unsubscribes: 5,
		spamReports: 1,
		insight: { display: '30.0%', headline: 'Best open rate: Ship the system.', body: '30.0% average across 1 issue · Sep 1', frac: 0.3 }
	},
	...over
});

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

describe('/os/social/beehiiv', () => {
	it('renders sends, volume, open rate, engagement, best open, and unfolds an issue', async () => {
		fetchMock.mockResolvedValue(json(model()));
		const { container } = render(BeehiivPage);
		await waitFor(() => expect(container.querySelector('[data-part="newsletters"]')).toBeTruthy());
		expect(String(fetchMock.mock.calls[0][0])).toContain('/pages/social/beehiiv');
		expect(screen.getByText('Newsletter Volume')).toBeTruthy();
		expect(screen.getByText('Best open rate: Ship the system.')).toBeTruthy();
		expect(screen.getByText('beehiiv live')).toBeTruthy();
		expect(screen.getByText('1 spam reports')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: /Ship the system/ }));
		expect(container.querySelector('[data-part="analytics"]')?.textContent).toContain('Web views');
		expect(screen.getByText('Read the issue')).toBeTruthy();
	});

	it('labels seeded issues and an absent subscriber count honestly', async () => {
		fetchMock.mockResolvedValue(json(model({ seeded: true, live: false, subscribers: null, volume: { ...model().volume, headline: null, caption: 'no live subscriber count · add BEEHIIV_API_KEY' } })));
		const { container } = render(BeehiivPage);
		await waitFor(() => expect(screen.getByText('seeded')).toBeTruthy());
		expect(container.textContent).toContain('seeded preview · add BEEHIIV_API_KEY for live');
		expect(container.textContent).toContain('no live subscriber count · add BEEHIIV_API_KEY');
	});

	it('says so when the backend is down', async () => {
		fetchMock.mockResolvedValue(json({ error: 'boom' }, 500));
		const { container } = render(BeehiivPage);
		await waitFor(() => expect(container.querySelector('[data-part="error"]')?.textContent).toContain('boom'));
	});
});

describe('/os/social/beehiiv header (prod copy and pills)', () => {
	it('a live, fresh read says what prod says; pills are rounded', async () => {
		fetchMock.mockResolvedValue(json(model()));
		const { container } = render(BeehiivPage);
		await waitFor(() => expect(container.querySelector('[data-part="newsletters"]')).toBeTruthy());
		expect(screen.getByText('live via Beehiiv API')).toBeTruthy();
		expect(screen.getByText(/All platforms/).closest('a')?.className).toContain('rounded-full');
		expect(screen.getByText(/Open Beehiiv/).closest('a')?.className).toContain('bn-pill-accent');
	});

	it('a stale read and a missing key say so in prod style', async () => {
		fetchMock.mockResolvedValue(json(model({ fresh: false })));
		const a = render(BeehiivPage);
		await waitFor(() => expect(screen.getByText('last good Beehiiv reading · API failing')).toBeTruthy());
		a.unmount();
		fetchMock.mockResolvedValue(json(model({ seeded: true, live: false, subscribers: null })));
		render(BeehiivPage);
		await waitFor(() => expect(screen.getByText('seeded preview · add BEEHIIV_API_KEY for live')).toBeTruthy());
	});
});
