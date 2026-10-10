import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ContentView from './ContentView.svelte';
import ContentRoute from '../../../../routes/(founderos)/os/content/+page.svelte';
import type { ContentPage, LeadMagnet } from './types';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const meter = (label: string, frac: number | null, display: string) => ({ label, frac, display, hue: 'var(--bn-ok)' });
const series = (n: number, hit = -1) => Array.from({ length: n }, (_, i) => ({ label: `D${i}`, count: i === hit ? 2 : 0 }));

const magnet = (over: Partial<LeadMagnet> = {}): LeadMagnet => ({
	id: 'agent-stack',
	name: 'The Agent Stack',
	offer: 'Every tool behind the company',
	url: 'https://founderos-agent-stack.vercel.app',
	status: 'live',
	captures: 'email',
	destination: 'Beehiiv · newsletter',
	source: 'IG carousel',
	launchedAt: '2026-08-12',
	notes: '',
	origin: 'seed',
	...over
});

const contentPage = (over: Partial<ContentPage> = {}): ContentPage => ({
	today: '2026-09-24',
	windowDays: 30,
	crew: [
		{ id: 'social-agent', name: 'Social Agent', role: 'Lead', status: 'active', tier: 'lead', description: 'Runs social', model: 'sonnet', tools: ['zernio-mcp'], parentId: null, departmentId: 'dept-marketing-growth', runnable: true },
		{ id: 'postly-publisher', name: 'Zernio Publisher', role: 'Publisher', status: 'idle', tier: 'worker', description: 'Posts', model: 'haiku', tools: [], parentId: 'social-agent', departmentId: 'dept-marketing-growth', runnable: false }
	],
	leadMagnets: [magnet()],
	posts: [{ platform: 'instagram', caption: 'Three agents that run my business', url: 'https://instagram.com/p/1', publishedAt: '2026-09-22T10:00:00Z', status: 'published' }],
	postsError: null,
	postsKnown: true,
	pipelineActiveDays: 4,
	volume: {
		headline: 3,
		chips: [{ tone: 'accent', text: '2 active days' }],
		caption: 'posts out through Zernio, last 30 days',
		meters: [meter('Lead magnets live (1/1)', 1, '1 of 1')],
		foot: '2 agents · 1 lead magnet · 1 recent post pulled',
		series: series(30, 28),
		postsInWindow: 3,
		activeDays: 2,
		crewRuns: [{ label: 'Social', count: 2 }, { label: 'Zernio', count: 0 }],
		runsInWindow: 2,
		magnets: { total: 1, live: 1, draft: 0, paused: 0, archived: 0 },
		insight: { value: 2, headline: '2 days since the last post went out.', body: '2 active days in the last 30', frac: 0.07 }
	},
	...over
});

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	document.cookie = 'csrf_token=t';
});
afterEach(() => vi.unstubAllGlobals());

const calls = () => fetchMock.mock.calls.map(([u, init]) => [String(u), (init as RequestInit | undefined)?.method ?? 'GET'] as const);

describe('/os/content', () => {
	it('the route renders the content view', async () => {
		fetchMock.mockResolvedValue(json(contentPage()));
		render(ContentRoute);
		await waitFor(() => expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Content Creation'));
	});

	it('reads /pages/content; the volume slab leads, then intelligence, agents and the pipeline (v1)', async () => {
		fetchMock.mockResolvedValue(json(contentPage()));
		const { container } = render(ContentView);
		await waitFor(() => expect(screen.getByText('Content intelligence')).toBeTruthy());
		expect(calls()[0][0]).toMatch(/\/founderos\/pages\/content$/);
		expect(screen.getByText('2 agents · 1 lead magnets · 2 active days in 30')).toBeTruthy();
		const titles = [...container.querySelectorAll('h2')].map((h) => h.textContent?.trim());
		expect(titles).toEqual(['Posting Activity', 'Content Volume', 'Crew Runs', 'Lead Magnets', 'Content intelligence', 'Content agents', 'Zernio content pipeline']);
		// three backlinks: the in-OS index, Vantage Intel, My Analytics
		const intel = container.querySelector('[data-part="intel"]')!;
		expect(intel.querySelectorAll('a')).toHaveLength(3);
		expect(within(intel as HTMLElement).getByText('Vantage Intel').closest('a')!.getAttribute('href')).toBe('https://intel.vantage.example');
		expect(within(intel as HTMLElement).getByText('Lead Magnets').closest('a')!.getAttribute('href')).toBe('/os/content/lead-magnets');
		expect(within(intel as HTMLElement).getByText('1 page · 1 live')).toBeTruthy();
		// the Vantage surface carries the real brand mark, not a letter
		expect(within(intel as HTMLElement).getByText('Vantage Intel').closest('a')!.querySelector('img')).toBeTruthy();
		// the pipeline row links the live post
		expect(screen.getByText('Three agents that run my business')).toBeTruthy();
		expect(screen.getByText(/Published across six platforms via Zernio · 4 active days tracked/)).toBeTruthy();
	});

	it('the volume slab: posting line, Content Volume meters, crew runs, lead magnets, since last post (v1)', async () => {
		fetchMock.mockResolvedValue(
			json(contentPage({ volume: { ...contentPage().volume, magnets: { total: 3, live: 2, draft: 0, paused: 1, archived: 0 }, crewRuns: [{ label: 'Social', count: 2 }, { label: 'Adsmith', count: 5 }], runsInWindow: 7 } }))
		);
		const { container } = render(ContentView);
		await waitFor(() => expect(screen.getByText('Content Volume')).toBeTruthy());
		const slab = container.querySelector('[data-part="volume"]') as HTMLElement;
		expect(within(slab).getByText('posts out through Zernio, one per cross-post')).toBeTruthy();
		expect(within(slab).getByText('posts out through Zernio, last 30 days')).toBeTruthy();
		expect(within(slab).getByText('Lead magnets live (1/1)')).toBeTruthy();
		expect(within(slab).getByText('2 agents · 1 lead magnet · 1 recent post pulled')).toBeTruthy();
		expect(within(slab).getByText('busiest · Adsmith with 5')).toBeTruthy();
		expect(within(slab).getByText('of 3 landing pages shipped')).toBeTruthy();
		expect(within(slab).getByText('1 paused')).toBeTruthy();
		expect(within(slab).getByText('Open').closest('a')!.getAttribute('href')).toBe('/os/content/lead-magnets');
		expect(within(slab).getByText('Since last post')).toBeTruthy();
		expect(within(slab).getByText('2 days since the last post went out.')).toBeTruthy();
	});

	it('lead card first, tool chips, and Run posts to the agent runtime', async () => {
		fetchMock.mockImplementation((u: string) =>
			Promise.resolve(String(u).endsWith('/run') ? json({ ok: true, summary: 'done' }) : json(contentPage()))
		);
		const { container } = render(ContentView);
		await waitFor(() => expect(screen.getByText('Social Agent')).toBeTruthy());
		const cards = container.querySelectorAll('[data-part="agent-card"]');
		expect(cards[0].textContent).toContain('Social Agent');
		expect(cards[0].textContent).toContain('lead');
		expect(cards[0].textContent).toContain('Zernio Mcp');
		await fireEvent.click(within(cards[0] as HTMLElement).getByRole('button', { name: /run/ }));
		await waitFor(() => expect((cards[0].querySelector('button[data-tone]') as HTMLElement).textContent).toMatch(/✓\s*ok/));
		expect(calls().some(([u, m]) => u.endsWith('/founderos/agents/social-agent/run') && m === 'POST')).toBe(true);
		// an agent the bridge cannot run yet says so instead of failing blind
		const worker = within(cards[1] as HTMLElement).getByRole('button', { name: /run/ }) as HTMLButtonElement;
		expect(worker.disabled).toBe(true);
		expect(worker.title).toMatch(/not on the bridge/);
	});

	it('a failed run surfaces the error', async () => {
		fetchMock.mockImplementation((u: string) =>
			Promise.resolve(String(u).endsWith('/run') ? json({ error: 'agent runtime unavailable' }, 503) : json(contentPage()))
		);
		const { container } = render(ContentView);
		await waitFor(() => expect(screen.getByText('Social Agent')).toBeTruthy());
		const card = container.querySelector('[data-part="agent-card"]') as HTMLElement;
		await fireEvent.click(within(card).getByRole('button', { name: /run/ }));
		await waitFor(() => expect(within(card).getByText(/agent runtime unavailable/)).toBeTruthy());
	});

	it('Zernio down: the posting line and since-last-post read unavailable; meta counts active days like v1', async () => {
		const down = contentPage({ posts: [], postsError: 'HTTP 503', postsKnown: false, pipelineActiveDays: null });
		down.volume = { ...down.volume, headline: 0, series: series(30), postsInWindow: 0, activeDays: 0, chips: [], caption: 'posting history unavailable · Zernio not answering', insight: { value: 0, display: '—', headline: 'Posting history unavailable.', body: 'Zernio did not answer, so days since the last post are unknown.', frac: 0 } };
		fetchMock.mockResolvedValue(json(down));
		render(ContentView);
		await waitFor(() => expect(screen.getByText('2 agents · 1 lead magnets · 0 active days in 30')).toBeTruthy());
		expect(screen.getByText('Zernio not answering · posting history unknown')).toBeTruthy();
		expect(screen.getByText('Posting history unavailable right now.')).toBeTruthy();
		expect(screen.getByText('Posting history unavailable.')).toBeTruthy();
		expect(screen.getByText(/No live Zernio pull right now/)).toBeTruthy();
		expect(screen.getByText(/HTTP 503/)).toBeTruthy();
		expect(screen.getByText(/Published across six platforms via Zernio · 0 active days tracked/)).toBeTruthy();
	});

	it('an unreachable backend is an error state, not an empty page', async () => {
		fetchMock.mockResolvedValue(json({ error: 'workspace not bootstrapped on the bridge' }, 503));
		render(ContentView);
		await waitFor(() => expect(screen.getByText(/workspace not bootstrapped/)).toBeTruthy());
		expect(screen.queryByText('Content intelligence')).toBeNull();
	});
});


describe('/os/content: production shape and motion (round 2)', () => {
	it('header and card pills are the kit rounded pills with the control lens', async () => {
		fetchMock.mockResolvedValue(json(contentPage()));
		render(ContentView);
		await waitFor(() => expect(screen.getByText('Content intelligence')).toBeTruthy());
		for (const name of ['Lead magnets', 'Social', 'Agents']) {
			const pills = screen.getAllByRole('link', { name: new RegExp(`^${name}`) }).filter((a) => a.className.includes('px-4'));
			expect(pills.length).toBeGreaterThan(0);
			for (const a of pills) {
				expect(a.className).toContain('rounded-full');
				expect(a.className).toContain('bn-pressable');
				expect(a.getAttribute('data-lens')).toBe('c');
			}
		}
	});

	it('backlinks and agent cards are rounded lens rows; lead tag and tool chips are rounded pills', async () => {
		fetchMock.mockResolvedValue(json(contentPage()));
		const { container } = render(ContentView);
		await waitFor(() => expect(screen.getByText('Social Agent')).toBeTruthy());
		for (const a of container.querySelectorAll('[data-part="intel"] a')) {
			expect(a.className).toContain('rounded-[10px]');
			expect(a.className).toContain('bn-pressable');
			expect(a.getAttribute('data-lens')).toBe('r');
		}
		const card = container.querySelector('[data-part="agent-card"]')!;
		expect(card.className).toContain('rounded-[10px]');
		expect(card.className).toContain('bn-pressable');
		expect(card.getAttribute('data-lens')).toBe('r');
		expect(within(card as HTMLElement).getByText('lead').className).toContain('rounded-full');
		const chip = within(card as HTMLElement).getByText('Zernio Mcp');
		expect(chip.className).toContain('rounded-full');
		expect(chip.getAttribute('data-lens')).toBe('c');
		const run = within(card as HTMLElement).getByRole('button', { name: /run/ });
		expect(run.getAttribute('data-tone')).toBe('secondary');
	});

	it('run walks busy (spinner + elapsed) → ✓ ok → back to ▸ run, like AsyncButton', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		try {
			let finish: (r: Response) => void = () => {};
			fetchMock.mockImplementation((u: string) =>
				String(u).endsWith('/run') ? new Promise<Response>((r) => (finish = r)) : Promise.resolve(json(contentPage()))
			);
			const { container } = render(ContentView);
			await waitFor(() => expect(screen.getByText('Social Agent')).toBeTruthy());
			const card = container.querySelector('[data-part="agent-card"]') as HTMLElement;
			await fireEvent.click(within(card).getByRole('button', { name: /run/ }));
			const btn = () => card.querySelector('button[data-tone]') as HTMLButtonElement;
			await waitFor(() => expect(btn().getAttribute('aria-busy')).toBe('true'));
			expect(btn().textContent).toMatch(/running\s*0s/);
			finish(json({ ok: true }));
			await waitFor(() => expect(btn().textContent).toMatch(/✓\s*ok/));
			await vi.advanceTimersByTimeAsync(1500);
			await waitFor(() => expect(btn().textContent?.trim()).toBe('▸ run'));
		} finally {
			vi.useRealTimers();
		}
	});

	it('a failed run reads ✗ failed in the button, never a false ✓', async () => {
		fetchMock.mockImplementation((u: string) =>
			Promise.resolve(String(u).endsWith('/run') ? json({ error: 'held by FOUNDEROS_WRITES' }, 403) : json(contentPage()))
		);
		const { container } = render(ContentView);
		await waitFor(() => expect(screen.getByText('Social Agent')).toBeTruthy());
		const card = container.querySelector('[data-part="agent-card"]') as HTMLElement;
		await fireEvent.click(within(card).getByRole('button', { name: /run/ }));
		await waitFor(() => expect((card.querySelector('button[data-tone]') as HTMLElement).textContent).toMatch(/✗\s*failed/));
		expect(within(card).getByText(/held by FOUNDEROS_WRITES/)).toBeTruthy();
	});

	it('pipeline rows are lens rows with rounded status pills and a rounded open control', async () => {
		fetchMock.mockResolvedValue(json(contentPage()));
		render(ContentView);
		await waitFor(() => expect(screen.getByText('Three agents that run my business')).toBeTruthy());
		const li = screen.getByText('Three agents that run my business').closest('li')!;
		expect(li.className).toContain('bn-pressable');
		expect(li.getAttribute('data-lens')).toBe('r');
		expect(within(li).getByText('published').className).toContain('rounded-full');
		const open = within(li).getByLabelText('open post');
		expect(open.className).toContain('rounded-full');
		expect(open.getAttribute('data-lens')).toBe('c');
	});

	it('the Zernio-down empty state is the rounded dashed box', async () => {
		fetchMock.mockResolvedValue(json(contentPage({ posts: [], postsError: null, postsKnown: false, pipelineActiveDays: null })));
		render(ContentView);
		await waitFor(() => expect(screen.getByText(/No live Zernio pull right now/)).toBeTruthy());
		expect(screen.getByText(/No live Zernio pull right now/).className).toContain('rounded-[12px]');
	});
});
