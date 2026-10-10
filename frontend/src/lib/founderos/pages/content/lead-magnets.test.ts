import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import LeadMagnetsView from './LeadMagnetsView.svelte';
import type { LeadMagnet, LeadMagnetsPage } from './types';

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

const lmPage = (over: Partial<LeadMagnetsPage> = {}): LeadMagnetsPage => ({
	today: '2026-09-24',
	weeks: 12,
	filter: 'all',
	filters: ['all', 'live', 'draft', 'paused', 'archived'],
	rows: [magnet(), magnet({ id: 'offer-doc', name: 'Offer Doc', status: 'draft', captures: 'none', url: 'https://example.com/offer' })],
	volume: {
		headline: 1,
		total: 2,
		counts: { live: 1, draft: 1, paused: 0, archived: 0 },
		chips: [{ tone: 'warn', text: '1 draft' }],
		caption: 'of 2 landing pages shipped',
		meters: [meter('Live (1/2)', 0.5, '1 of 2')],
		foot: '1 destination · 1 campaign',
		series: series(12, 11),
		shippedInWindow: 1,
		captures: [{ label: 'Email', count: 1 }, { label: 'Booking', count: 0 }, { label: 'None', count: 1 }],
		destinations: [{ label: 'Beehiiv', count: 2 }],
		insight: { value: 4, headline: '4 days since Offer Doc shipped.', body: '1 of 2 live', frac: 0.5 }
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

describe('/os/content/lead-magnets', () => {
	it('reads the filtered rows and draws the slab over every row', async () => {
		fetchMock.mockResolvedValue(json(lmPage()));
		render(LeadMagnetsView, { status: null });
		await waitFor(() => expect(screen.getByText('Magnet Volume')).toBeTruthy());
		expect(calls()[0][0]).toMatch(/\/founderos\/pages\/content\/lead-magnets$/);
		for (const t of ['Pages Shipped', 'Captures', 'Where Leads Land', 'All pages']) expect(screen.getAllByText(t).length).toBeGreaterThan(0);
		expect(screen.getByText('4 days since Offer Doc shipped.')).toBeTruthy();
		expect(screen.getByText('2 pages · 1 live · 1 destination · 1 campaign')).toBeTruthy();
		// the Notion-style table: property columns, every row opens the real page
		for (const col of ['Name', 'Status', 'Captures', 'Leads to', 'Source', 'Live', 'Link', 'Manage'])
			expect(screen.getByRole('columnheader', { name: col })).toBeTruthy();
		const link = screen.getByText('The Agent Stack').closest('a')!;
		expect(link.getAttribute('href')).toBe('https://founderos-agent-stack.vercel.app');
		expect(link.getAttribute('target')).toBe('_blank');
		expect(screen.getByText('founderos-agent-stack.vercel.app')).toBeTruthy();
	});

	it('filter pills are links carrying ?status and the active one is marked', async () => {
		fetchMock.mockResolvedValue(json(lmPage({ filter: 'live', rows: [magnet()] })));
		render(LeadMagnetsView, { status: 'live' });
		await waitFor(() => expect(screen.getByText('Magnet Volume')).toBeTruthy());
		expect(calls()[0][0]).toMatch(/lead-magnets\?status=live$/);
		const live = screen.getByText('Live 1').closest('a')!;
		expect(live.getAttribute('href')).toBe('/os/content/lead-magnets?status=live');
		expect(live.getAttribute('aria-current')).toBe('page');
		expect(screen.getByText('All 2').closest('a')!.getAttribute('href')).toBe('/os/content/lead-magnets');
		expect(screen.getAllByText('1 of 2').length).toBeGreaterThan(0);
	});

	it('the form posts to /pages/content/lead-magnets and refetches', async () => {
		fetchMock.mockImplementation((u: string, init?: RequestInit) =>
			Promise.resolve(init?.method === 'POST' ? json({ leadMagnet: magnet({ id: 'new', name: 'New Page' }) }, 201) : json(lmPage()))
		);
		render(LeadMagnetsView, { status: null });
		await waitFor(() => expect(screen.getByText('Magnet Volume')).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: /New lead magnet/ }));
		await fireEvent.input(screen.getByLabelText('Name'), { target: { value: 'New Page' } });
		await fireEvent.input(screen.getByLabelText('Live URL'), { target: { value: 'https://example.com/new' } });
		await fireEvent.submit(screen.getByLabelText('Name').closest('form')!);
		await waitFor(() => expect(screen.getByText('Added New Page to the register.')).toBeTruthy());
		const post = fetchMock.mock.calls.find(([, init]) => (init as RequestInit | undefined)?.method === 'POST')!;
		expect(String(post[0])).toMatch(/\/founderos\/pages\/content\/lead-magnets$/);
		expect(JSON.parse((post[1] as RequestInit).body as string)).toMatchObject({ name: 'New Page', url: 'https://example.com/new', status: 'live', captures: 'email' });
		expect(calls().filter(([, m]) => m === 'GET')).toHaveLength(2);
	});

	it('a rejected create says why and keeps the form open', async () => {
		fetchMock.mockImplementation((_u: string, init?: RequestInit) =>
			Promise.resolve(init?.method === 'POST' ? json({ error: 'url must be a URL' }, 400) : json(lmPage()))
		);
		render(LeadMagnetsView, { status: null });
		await waitFor(() => expect(screen.getByText('Magnet Volume')).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: /New lead magnet/ }));
		await fireEvent.input(screen.getByLabelText('Name'), { target: { value: 'X' } });
		await fireEvent.input(screen.getByLabelText('Live URL'), { target: { value: 'https://x.y' } });
		await fireEvent.submit(screen.getByLabelText('Name').closest('form')!);
		await waitFor(() => expect(screen.getByText(/url must be a URL/)).toBeTruthy());
		expect(screen.getByLabelText('Name')).toBeTruthy();
	});

	it('row actions: status select PATCHes, delete confirms then DELETEs, both refetch', async () => {
		fetchMock.mockImplementation((_u: string, init?: RequestInit) =>
			Promise.resolve(init?.method && init.method !== 'GET' ? json({ ok: true }) : json(lmPage()))
		);
		render(LeadMagnetsView, { status: null });
		await waitFor(() => expect(screen.getByText('Magnet Volume')).toBeTruthy());
		const select = screen.getAllByLabelText('Status')[0] as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'archived' } });
		await waitFor(() => expect(calls().some(([u, m]) => u.endsWith('/founderos/pages/lead-magnets/agent-stack') && m === 'PATCH')).toBe(true));
		const patch = fetchMock.mock.calls.find(([, init]) => (init as RequestInit | undefined)?.method === 'PATCH')!;
		expect(JSON.parse((patch[1] as RequestInit).body as string)).toEqual({ status: 'archived' });

		await fireEvent.click(screen.getAllByLabelText('Delete lead magnet')[1]);
		expect(calls().some(([, m]) => m === 'DELETE')).toBe(false);
		await fireEvent.click(screen.getByRole('button', { name: 'sure?' }));
		await waitFor(() => expect(calls().some(([u, m]) => u.endsWith('/founderos/pages/lead-magnets/offer-doc') && m === 'DELETE')).toBe(true));
		await waitFor(() => expect(calls().filter(([, m]) => m === 'GET').length).toBe(3));
	});

	it('nothing shipped reads honest copy, a filter with no match says so', async () => {
		fetchMock.mockResolvedValue(json(lmPage({ filter: 'paused', rows: [] })));
		render(LeadMagnetsView, { status: 'paused' });
		await waitFor(() => expect(screen.getByText('Nothing matches that filter.')).toBeTruthy());
	});
});

describe('/os/content/lead-magnets: production shape (round 2)', () => {
	it('pills, chips and the table controls take production rounding', async () => {
		fetchMock.mockResolvedValue(json(lmPage()));
		const { container } = render(LeadMagnetsView);
		await waitFor(() => expect(screen.getByText('The Agent Stack')).toBeTruthy());
		const back = screen.getByRole('link', { name: /Content/ });
		expect(back.className).toContain('rounded-full');
		expect(back.getAttribute('data-lens')).toBe('c');
		for (const f of ['All 2', 'Live 1', 'Draft 1']) expect(screen.getByRole('link', { name: f }).className).toContain('rounded-full');
		const add = screen.getByRole('button', { name: /New lead magnet/ });
		expect(add.className).toContain('rounded-full');
		expect(add.className).toContain('bn-pressable');
		const row = screen.getByText('The Agent Stack').closest('tr')!;
		expect(within(row).getByText('live', { selector: 'span' }).className).toContain('rounded-full');
		expect(within(row).getByText('Email').closest('span')!.className).toContain('rounded-full');
		expect(within(row).getByRole('button', { name: /copy/ }).className).toContain('rounded-[5px]');
		expect(within(row).getByLabelText('Status').className).toContain('rounded-[5px]');
		// the trash can only shows on row hover, like production
		expect(within(row).getByLabelText('Delete lead magnet').className).toContain('opacity-0');
		expect(container.querySelector('table')).toBeTruthy();
	});

	it('the register form is the rounded panel with rounded fields', async () => {
		fetchMock.mockResolvedValue(json(lmPage()));
		render(LeadMagnetsView);
		await waitFor(() => expect(screen.getByText('The Agent Stack')).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: /New lead magnet/ }));
		const form = screen.getByLabelText('Name').closest('form')!;
		expect(form.className).toContain('rounded-[10px]');
		expect(screen.getByLabelText('Name').className).toContain('rounded-[8px]');
		expect(screen.getByRole('button', { name: 'add to the register' }).className).toContain('rounded-full');
	});
});
