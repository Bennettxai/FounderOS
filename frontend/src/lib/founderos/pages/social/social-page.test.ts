import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import SocialPage from '../../../../routes/(founderos)/os/social/+page.svelte';
import type { BeehiivPage, SocialPage as Model } from './types';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const growth = (d7: number | null) => ({ d7, d30: null, d60: null, allTime: null });
const day = new Date().toISOString().slice(0, 10);
const series = (n: number) => Array.from({ length: n }, (_, i) => ({ label: `D${i}`, count: i === n - 1 ? 2 : 0 }));

export function socialModel(over: Partial<Model> = {}): Model {
	const msg = { id: 'm1', platform: 'instagram', subscriberId: 's1', name: 'Ava Lee', handle: 'ava', text: 'price?', direction: 'in' as const, tag: null, ts: new Date(Date.now() - 3600e3).toISOString(), source: 'seed' };
	return {
		totalFollowers: 1600,
		asOf: day,
		platforms: [
			{ platform: 'instagram', handle: '@founderosx.ai', url: null, followers: 1100, growth: growth(1.234), series: [] },
			{ platform: 'linkedin', handle: 'the operator', url: null, followers: null, growth: growth(null), series: [] }
		],
		emailList: { subscribers: 24814, asOf: `${day}T09:00:00.000Z`, growth: { d7: -0.14, d30: 21.8, d60: null, allTime: null }, series: [] },
		totalDms: 1600,
		audienceTotal: 25914,
		audienceGrowth: { d7: 1.5, d30: 3, d60: null, allTime: 10 },
		dmGrowth: growth(2),
		today: day,
		growthLeader: 'Instagram',
		dmThreads: [{ subscriberId: 's1', name: 'Ava Lee', handle: 'ava', messages: [msg], last: msg, unreplied: true }],
		audiencePoints: [{ date: day, value: 25914 }],
		postDays: [{ date: day, platforms: ['instagram', 'tiktok'] }],
		postsKnown: true,
		recentPosts: [{ platform: 'instagram', caption: '3 agents that run my business\nmore', url: 'https://ig/p/1', publishedAt: new Date().toISOString(), status: 'success' }],
		posts: [],
		queued: 1,
		sync: { source: 'zernio-live', recorded: 1 },
		volume: {
			headline: 25914,
			chips: [{ tone: 'ok', text: '+1.50% 7d' }, { tone: 'accent', text: '1 queued' }],
			caption: 'across 2 live channels · Instagram leads 7-day growth',
			meters: [
				{ label: 'Email list', frac: 0.95, display: '24,814 · 96%', hue: 'var(--bn-text-2)' },
				{ label: 'LinkedIn', frac: null, display: 'offline', hue: 'var(--bn-text)' }
			],
			foot: 'share of total reach',
			series: series(30),
			postsInWindow: 1,
			mix: [
				{ label: 'IG', count: 1 },
				{ label: 'TT', count: 1 },
				{ label: 'X', count: 0 },
				{ label: 'YT', count: 0 },
				{ label: 'LI', count: 0 }
			],
			insight: { value: 1, headline: '1 of 1 Instagram threads need a reply.', body: 'Ava Lee', frac: 1 }
		},
		...over
	};
}

const issue = { id: 'p1', title: 'Ship the system', publishedAt: '2026-09-01T15:00:00.000Z', webUrl: 'https://news/p1', recipients: 1000, delivered: 980, deliveryRate: 98, opens: 294, openRate: 30, clicks: 30, clickRate: 3, unsubscribes: 5, unsubscribeRate: 0.5, spamReports: 1, webViews: 4 };

export function beehiivModel(over: Partial<BeehiivPage> = {}): BeehiivPage {
	return {
		newsletters: [issue, { ...issue, id: 'p2', title: 'Agents at work', publishedAt: '2026-09-08T15:00:00.000Z', openRate: 20 }],
		seeded: false,
		subscribers: 24900,
		live: true,
		fresh: true,
		volume: {
			headline: 24900,
			chips: [{ tone: 'accent', text: '2 issues' }],
			caption: 'subscribers, live via Beehiiv',
			meters: [
				{ label: 'Delivered', frac: 0.98, display: '98.0% of sends', hue: 'var(--bn-ok)' },
				{ label: 'Opened', frac: 0.3, display: '30.0% of delivered', hue: 'var(--bn-text-2)' },
				{ label: 'Clicked', frac: 0.1, display: '10.2% of opens', hue: 'var(--bn-text)' },
				{ label: 'Unsubscribed', frac: 0.005, display: '0.51% of delivered', hue: 'var(--bn-warn)' }
			],
			foot: '2 issues · 2,000 sends',
			sends: [{ label: 'Sep 1', count: 1000 }, { label: 'Sep 8', count: 1000 }],
			openRates: [{ label: 'Sep 1', count: 30 }, { label: 'Sep 8', count: 20 }],
			avgOpenRate: 25,
			totalRecipients: 2000,
			totalClicks: 60,
			unsubscribes: 10,
			spamReports: 0,
			insight: { display: '30.0%', headline: 'Best open rate: Ship the system.', body: '25.0% average across 2 issues · Sep 1', frac: 0.3 }
		},
		...over
	};
}

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

// Most specific path first: /pages/social is a prefix of its sub-routes.
const route = (routes: Record<string, () => Response>) =>
	fetchMock.mockImplementation(async (url: string) => {
		for (const [k, r] of Object.entries(routes)) if (String(url).includes(k)) return r();
		return json({ error: 'unexpected ' + url }, 500);
	});
const social = (model = socialModel()) => route({ '/pages/social': () => json(model) });

const cardTitles = (c: HTMLElement) => [...c.querySelectorAll('h2')].map((h) => h.textContent?.trim());

describe('/os/social (FounderOS v1 app/social/page.tsx)', () => {
	it('meta is reach, posts in 30 days, queue and DM threads; the zernio chip is honest', async () => {
		social();
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="accounts"]')).toBeTruthy());
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Social');
		expect(screen.getByText('25,914 reach · 1 posts in 30 days · 1 queued · 1 DM threads')).toBeTruthy();
		expect(screen.getByText('zernio live')).toBeTruthy();
		for (const name of ['Beehiiv', 'Social agent']) expect(screen.getAllByText(name)[0].closest('a')?.className).toContain('rounded-full');
	});

	it('unknown posting history reads ? in the meta, never 0', async () => {
		const zero = series(30).map((p) => ({ ...p, count: 0 }));
		social(socialModel({ postDays: null, postsKnown: false, sync: { source: 'none', recorded: 0 }, volume: { ...socialModel().volume, series: zero, postsInWindow: 0 } }));
		render(SocialPage);
		await waitFor(() => expect(screen.getByText('25,914 reach · ? posts in 30 days · 1 queued · 1 DM threads')).toBeTruthy());
		expect(screen.getByText('zernio offline')).toBeTruthy();
		expect(screen.getByText('Zernio not answering · posting history unknown')).toBeTruthy();
		expect(screen.getByText('Posting history unavailable right now.')).toBeTruthy();
	});

	it('card order matches v1: Accounts, Audience Volume, Posting Activity, Platform Mix, Recent posts, Publish', async () => {
		social();
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="accounts"]')).toBeTruthy());
		expect(cardTitles(container)).toEqual(['Accounts', 'Audience Volume', 'Posting Activity', 'Platform Mix', 'Recent posts', 'Publish']);
		// the private build's newsletter surfaces are not on v1's /social
		for (const gone of ['Newsletter', 'Open Rate', 'Engagement', 'Latest issues']) expect(cardTitles(container)).not.toContain(gone);
		expect(fetchMock.mock.calls.some(([u]) => String(u).includes('/pages/social/beehiiv'))).toBe(false);
	});

	it('accounts sit in the Accounts card, three across, every channel plus the email list', async () => {
		social();
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="accounts"]')).toBeTruthy());
		const accounts = container.querySelector('[data-part="accounts"]')!;
		expect(accounts.className).toContain('sm:grid-cols-3');
		expect(accounts.closest('.bn-card')).toBeTruthy();
		expect(screen.getByText('25,914 total')).toBeTruthy();
		expect(accounts.querySelectorAll('a')).toHaveLength(3); // 2 platforms + email list
		expect(container.querySelector('[data-platform="instagram"]')?.getAttribute('href')).toBe('/os/social/instagram');
		expect(container.querySelector('[data-platform="email"]')?.getAttribute('href')).toBe('/os/social/beehiiv');
		expect(container.querySelector('[data-platform="email"]')?.textContent).toContain('Beehiiv · ');
		expect(container.querySelector('[data-platform="email"]')?.textContent).not.toContain('the operator');
		// no follower reading reads unknown, never 0
		expect(container.querySelector('[data-platform="linkedin"] [data-unknown]')).toBeTruthy();
		for (const tile of accounts.querySelectorAll('a')) {
			expect(tile.getAttribute('data-lens')).toBe('r');
			expect(tile.className).toContain('bn-pressable');
			expect(tile.className).toContain('rounded-[10px]');
		}
	});

	it('audience volume: the chips and the share meters', async () => {
		social(socialModel({ volume: { ...socialModel().volume, chips: [{ tone: 'ok', text: '+1.50% 7d' }, { tone: 'warn', text: '1 need reply' }, { tone: 'accent', text: '1 queued' }] } }));
		const { container } = render(SocialPage);
		await waitFor(() => expect(screen.getByText('+1.50% 7d')).toBeTruthy());
		expect(screen.getByText('1 need reply')).toBeTruthy();
		expect(screen.getByText('across 2 live channels · Instagram leads 7-day growth')).toBeTruthy();
		expect(screen.getByText('share of total reach')).toBeTruthy();
		expect(screen.getByText('24,814 · 96%')).toBeTruthy();
	});

	it('posting activity, platform mix and the Needs reply card', async () => {
		social();
		render(SocialPage);
		await waitFor(() => expect(screen.getByText('posts out through Zernio')).toBeTruthy());
		expect(screen.getByText('most posted · 1 posts')).toBeTruthy();
		expect(screen.getAllByText('Needs reply')).toHaveLength(2); // the insight card + the strip tile
		expect(screen.getByText('1 of 1 Instagram threads need a reply.')).toBeTruthy();
	});

	it('the stat strip: total reach, audience growth, total DMs and the DM inbox tile', async () => {
		social();
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="stat-strip"]')).toBeTruthy());
		const strip = container.querySelector('[data-part="stat-strip"]') as HTMLElement;
		for (const label of ['Total reach', 'Audience growth', 'Total DMs', 'Needs reply']) expect(within(strip).getByText(label)).toBeTruthy();
		expect(strip.textContent).toContain('2 platforms + email');
		expect(strip.textContent).toContain('Instagram leads · 7d');
		expect(strip.textContent).toContain('1 to reply');
		expect(strip.textContent).not.toContain('demo');
		await fireEvent.click(container.querySelector('[data-part="tile-inbox"]')!);
		await waitFor(() => expect(container.querySelector('[data-part="dm-inbox"]')).toBeTruthy());
		expect(container.querySelector('[data-part="dm-inbox"]')?.textContent).toContain('price?');
	});

	it('the audience + posting consistency chart rides with the share donut', async () => {
		social();
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="audience-chart"]')).toBeTruthy());
		const chart = container.querySelector('[data-part="audience-chart"]') as HTMLElement;
		expect(chart.textContent).toContain('Posting consistency');
		const pie = chart.querySelector('[data-part="audience-pie"]') as HTMLElement;
		expect(pie.textContent).toContain('total reach');
		const legend = [...pie.querySelectorAll('[data-part="legend-row"]')].map((r) => r.textContent);
		expect(legend[0]).toContain('Instagram'); // v1 keeps channel order: platforms, then email
		expect(legend.at(-1)).toContain('Email list');
	});

	it('recent posts are a row of boxes with recency dots: live Zernio history first', async () => {
		social();
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="recent-posts"]')).toBeTruthy());
		const recent = container.querySelector('[data-part="recent-posts"]') as HTMLElement;
		expect(recent.className).toContain('xl:grid-cols-5');
		expect(recent.textContent).toContain('3 agents that run my business');
		expect(recent.textContent).not.toContain('more');
		expect(screen.getByText('1 live · zernio')).toBeTruthy();
		expect(screen.getByText('view →').closest('a')?.getAttribute('href')).toBe('https://ig/p/1');
	});

	it('no live history falls back to the labelled sample posts with their L/V ratio', async () => {
		social(socialModel({ recentPosts: null, recentError: 'zernio down', sync: { source: 'none', recorded: 0 } }));
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="recent-posts"]')).toBeTruthy());
		expect(screen.getByText('8.6% avg L/V · sample')).toBeTruthy();
		const recent = container.querySelector('[data-part="recent-posts"]') as HTMLElement;
		expect(recent.textContent).toContain('3 agents that run my business while I sleep');
		expect(recent.textContent).toContain('Founder OS walkthrough');
		expect(recent.textContent).not.toContain('FounderOS v1');
		expect(recent.querySelectorAll('[data-part="recency"]')).toHaveLength(5);
	});

	it('the Publish card carries the composer and its queue', async () => {
		const post = { id: 'p1', caption: 'New case study', mediaUrl: null, platforms: ['instagram'], status: 'queued' as const, scheduledFor: null, createdAt: day };
		social(socialModel({ posts: [post] }));
		const { container } = render(SocialPage);
		await waitFor(() => expect(container.querySelector('[data-part="composer"]')).toBeTruthy());
		expect(screen.getByText('Publish').closest('.bn-card')?.textContent).toContain('1 queued');
		expect(container.querySelector('[data-part="composer"]')?.textContent).toContain('New case study');
		expect(container.querySelector('[data-part="composer"]')?.textContent).toContain('1 pending');
	});

	it('says so when the backend is down', async () => {
		route({ '/pages/social': () => json({ error: 'workspace not bootstrapped' }, 503) });
		render(SocialPage);
		await waitFor(() => expect(screen.getByText(/Social is unavailable: workspace not bootstrapped/)).toBeTruthy());
	});
});
