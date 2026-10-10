import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { BrandDeal, BrandDealsBody, DealView } from '$lib/founderos/pages/brand-deals/view';

const fetchMock = vi.fn();
vi.mock('$lib/founderos/api', async (orig) => {
	const real = await orig<typeof import('$lib/founderos/api')>();
	return { ...real, founderosFetch: (...a: unknown[]) => fetchMock(...a) };
});

import Page from './+page.svelte';

const here = __dirname;
const libDir = resolve(here, '../../../../lib/founderos/pages/brand-deals');
const read = (p: string) => readFileSync(p, 'utf8');

function deal(over: Partial<BrandDeal> & { id: string; brand: string }): BrandDeal {
	return {
		status: 'New',
		tier: null,
		dealValueUsd: null,
		budgetUsd: null,
		amountAgreedUsd: null,
		suggestedRateUsd: null,
		paidInFull: false,
		deadline: null,
		followUpDate: null,
		contactName: null,
		contactEmail: null,
		mainChannel: null,
		videoType: null,
		source: null,
		icpFit: null,
		notionUrl: 'https://www.notion.so/x',
		lastEdited: '2026-09-16T09:00:00.000Z',
		seeded: false,
		...over
	};
}

const DEALS = [
	deal({ id: 'a', brand: 'Acme Notes', status: 'Negotiating', tier: 'S', dealValueUsd: 6000, followUpDate: '2026-09-18', contactName: 'Sam Rivera', notionUrl: 'https://www.notion.so/a' }),
	deal({ id: 'c', brand: 'Shopify', status: 'Filming', amountAgreedUsd: 8500, notionUrl: 'https://www.notion.so/c' }),
	deal({ id: 'd', brand: 'Riverside', status: 'Paid', amountAgreedUsd: 2500, paidInFull: true, notionUrl: 'https://www.notion.so/d' })
];

const VIEW: DealView = {
	volume: { openUsd: 14500, talksUsd: 6000, productionUsd: 8500, paidUsd: 2500, declinedUsd: 0, quotedDeals: 3, counts: { all: 3, talks: 1, production: 1, paid: 1, declined: 0, tierS: 1 } },
	stages: [
		{ label: 'All deals', value: 3, usd: 17000, note: 'every deal in the hub', filter: 'all' },
		{ label: 'In talks', value: 1, usd: 6000, note: 'inbound and negotiating', filter: 'talks' },
		{ label: 'In production', value: 1, usd: 8500, note: 'researching through invoiced', filter: 'production' },
		{ label: 'Paid', value: 1, usd: 2500, note: '0 declined · 0 paused', filter: 'paid' }
	],
	activity: Array.from({ length: 30 }, (_, i) => ({ label: `09/${String(i + 1).padStart(2, '0')}`, count: i === 28 ? 2 : 0 })),
	editsInWindow: 2,
	sizes: [
		{ label: '<1k', count: 0 },
		{ label: '1-3k', count: 1 },
		{ label: '3-5k', count: 0 },
		{ label: '5-10k', count: 2 },
		{ label: '10k+', count: 0 }
	],
	largestUsd: 8500,
	due: { followUps: ['a'], deadlines: [] },
	needsYou: 1,
	needsYouBrands: ['Acme Notes'],
	openCount: 2,
	needsYouFrac: 0.5,
	hubUrl: 'https://www.notion.so/a'
};

const LIVE: BrandDealsBody = { deals: DEALS, mode: 'live', detail: 'Brand Deals Hub · 3 deals', syncedAt: Date.now() - 5 * 60_000, view: VIEW };

function reducedMotion() {
	vi.spyOn(window, 'matchMedia').mockImplementation(
		(q: string) => ({ matches: q.includes('reduce'), media: q, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList
	);
}

beforeEach(() => {
	fetchMock.mockReset();
	reducedMotion();
});
afterEach(() => {
	vi.useRealTimers();
	vi.restoreAllMocks();
});

async function renderWith(body: BrandDealsBody) {
	fetchMock.mockResolvedValue(body);
	const r = render(Page);
	await screen.findByRole('heading', { level: 1, name: 'Brand Deals' });
	await waitFor(() => expect(r.container.querySelector('[data-part="loading"]')).toBeNull());
	return r;
}

describe('/os/brand-deals: live Notion board', () => {
	it('shows a loading state, then reads GET /pages/brand-deals', async () => {
		let resolve!: (b: BrandDealsBody) => void;
		fetchMock.mockReturnValue(new Promise((r) => (resolve = r)));
		const { container } = render(Page);
		expect(container.querySelector('[data-part="loading"]')).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledWith('/pages/brand-deals');
		resolve(LIVE);
		await screen.findByText('live · Notion');
	});

	it('keeps the honest live pill, the Notion row count, the sync age and the open pipeline', async () => {
		await renderWith(LIVE);
		expect(screen.getByText('live · Notion')).toBeTruthy();
		expect(screen.getByText(/\$14,500 open pipeline · 3 deals · 3 Notion rows · synced 5m ago/)).toBeTruthy();
		expect(screen.getByText('read-only · refreshes 60s')).toBeTruthy();
		const hub = screen.getAllByRole('link', { name: /Open in Notion/ })[0];
		expect(hub.getAttribute('href')).toBe('https://www.notion.so/a');
	});

	it('the header actions are production pills: rounded read-only pill, accent Open in Notion, round refresh', async () => {
		await renderWith(LIVE);
		expect(screen.getByText('read-only · refreshes 60s').className).toContain('rounded-full');
		const hub = screen.getAllByRole('link', { name: /Open in Notion/ })[0];
		expect(hub.className).toContain('bn-pill-accent');
		expect(hub.className).toContain('rounded-full');
		expect(screen.getByRole('button', { name: 'Refresh' }).className).toContain('rounded-full');
		// card menus and the prompt bar carry prod's rounded shape too
		expect(screen.getByRole('button', { name: 'Pipeline menu' }).className).toContain('rounded-full');
		expect(screen.getByLabelText('Filter deals').closest('label')!.className).toContain('rounded-[12px]');
	});

	it('a long meta line runs at full width, as in v1: the actions wrap under it rather than squeezing it', async () => {
		await renderWith({ ...LIVE, mode: 'seeded', detail: 'Showing examples. Plant NOTION_API_KEY (internal integration secret, shared with Brand Deals Hub) to go live.' });
		const meta = screen.getByText(/Showing examples\. Plant NOTION_API_KEY/);
		expect(meta.closest('[data-part="bd-meta"]')?.className ?? '').not.toMatch(/max-w-/);
	});

	it('renders the pipeline hero, the volume meters, the activity, the sizes and ONE insight card', async () => {
		const { container } = await renderWith(LIVE);
		expect(container.querySelectorAll('[data-stage]')).toHaveLength(4);
		expect(screen.getByText('In talks (1)')).toBeTruthy();
		expect(screen.getByText('$2,500 / $0')).toBeTruthy();
		expect(screen.getByText('Notion edits, last 30 days')).toBeTruthy();
		expect(container.querySelector('[data-part="sizes-stat"]')!.textContent).toContain('Largest: $8,500');
		expect(container.querySelectorAll('.bn-insight')).toHaveLength(1);
		expect(screen.getByText('1 follow-up due · 0 deadlines inside 7 days.')).toBeTruthy();
		expect(screen.getByText('Acme Notes', { selector: '.bn-insight-body *, .bn-insight-body' })).toBeTruthy();
	});

	it('lists every deal with its money, tier and status', async () => {
		const { container } = await renderWith(LIVE);
		const rows = container.querySelectorAll('[data-deal]');
		expect(rows).toHaveLength(3);
		expect(within(rows[0] as HTMLElement).getByText('$6,000')).toBeTruthy();
		expect(within(rows[0] as HTMLElement).getByText('S-tier')).toBeTruthy();
		expect(screen.getByText('3 of 3')).toBeTruthy();
	});

	it('the prompt bar is a live filter with slash tokens', async () => {
		const { container } = await renderWith(LIVE);
		const input = screen.getByLabelText('Filter deals') as HTMLInputElement;
		expect(input.placeholder).toContain('/talks /production /paid');
		await fireEvent.input(input, { target: { value: '/talks' } });
		expect(container.querySelector('[data-part="slash"]')?.textContent).toBe('/talks');
		expect([...container.querySelectorAll('[data-deal]')].map((r) => r.getAttribute('data-deal'))).toEqual(['a']);
		await fireEvent.input(input, { target: { value: 'zzz' } });
		expect(screen.getByText('Nothing matches that filter.')).toBeTruthy();
	});

	it('hero columns and chips filter the list', async () => {
		const { container } = await renderWith(LIVE);
		await fireEvent.click(container.querySelector('[data-stage="3"]')!);
		expect([...container.querySelectorAll('[data-deal]')].map((r) => r.getAttribute('data-deal'))).toEqual(['d']);
		await fireEvent.click(container.querySelector('[data-filter="production"]')!);
		expect(container.querySelector('[data-stage="2"]')!.getAttribute('aria-pressed')).toBe('true');
		expect([...container.querySelectorAll('[data-deal]')].map((r) => r.getAttribute('data-deal'))).toEqual(['c']);
	});

	it('a deal opens the drawer, which deep-links into Notion; Escape closes it', async () => {
		const { container } = await renderWith(LIVE);
		await fireEvent.click(container.querySelector('[data-deal="c"]')!);
		const drawer = container.querySelector('[data-part="drawer"]') as HTMLElement;
		expect(drawer).toBeTruthy();
		expect(within(drawer).getByRole('link', { name: /Open in Notion/ }).getAttribute('href')).toBe('https://www.notion.so/c');
		expect(within(drawer).getByText('$8,500')).toBeTruthy();
		expect(within(drawer).getAllByText('not set').length).toBeGreaterThan(0);
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(container.querySelector('[data-part="drawer"]')).toBeNull();
	});
});

describe('/os/brand-deals: production shapes and states (round 2)', () => {
	it('filter chips are production pills: rounded, pressable, the active one solid', async () => {
		const { container } = await renderWith(LIVE);
		const all = container.querySelector('[data-filter="all"]')!;
		const talks = container.querySelector('[data-filter="talks"]')!;
		for (const c of [all, talks]) {
			expect(c.className).toContain('rounded-full');
			expect(c.className).toContain('bn-pressable');
			expect(c.className).toContain('bn-filter-chip');
		}
		expect(all.className).toContain('is-on');
		expect(talks.className).not.toContain('is-on');
		expect(talks.className).toContain('border');
	});

	it('row money, S-tier and status pills are rounded, and each row is pressable', async () => {
		const { container } = await renderWith(LIVE);
		const row = container.querySelector('[data-deal="a"]') as HTMLElement;
		expect(row.className).toContain('bn-pressable');
		for (const t of ['$6,000', 'S-tier']) expect(within(row).getByText(t).className).toContain('rounded-full');
		expect(row.querySelector('[data-part="status"]')!.className).toContain('rounded-full');
	});

	it('the pipeline tooltip is a rounded pill with a drop shadow; caps and the active column are rounded; columns press', async () => {
		const { container } = await renderWith(LIVE);
		const tip = container.querySelector('[data-part="tooltip"]') as HTMLElement;
		expect(tip.className).toContain('rounded-full');
		expect(tip.getAttribute('style')).toContain('box-shadow');
		expect(container.querySelector('[data-bar="0"] rect')!.getAttribute('rx')).toBe('3');
		expect(container.querySelector('[data-bar="1"] rect')!.getAttribute('rx')).toBe('0');
		expect(container.querySelector('[data-cap="0"]')!.getAttribute('rx')).toBe('3');
		expect(container.querySelector('[data-stage="0"]')!.className).toContain('bn-pressable');
	});

	it('the solid column fades into the accent-tinted shade (prod --pc-shade), not bare background', () => {
		const chart = read(resolve(libDir, 'PipelineChart.svelte'));
		expect(chart).toContain('color-mix(in oklab, var(--bn-accent) 55%, color-mix(in oklab, var(--bn-accent) 25%, var(--bn-bg)))');
	});

	it('Deal Activity reads its count and caption on one line, like production', async () => {
		const { container } = await renderWith(LIVE);
		const stat = container.querySelector('[data-part="activity-stat"]') as HTMLElement;
		expect(stat).toBeTruthy();
		const cap = within(stat).getByText('Notion edits, last 30 days');
		expect(cap.tagName).toBe('SPAN');
		expect(cap.className).toContain('ml-2');
		expect(stat.textContent).toMatch(/^\s*2\s/);
	});

	it('Deal Sizes stacks a 30px count over "priced deals" and a rounded Largest chip', async () => {
		const { container } = await renderWith(LIVE);
		const stat = container.querySelector('[data-part="sizes-stat"]') as HTMLElement;
		expect(stat.querySelector('.text-\\[30px\\]')!.textContent!.trim()).toBe('3');
		expect(within(stat).getByText('priced deals').className).toContain('mt-1');
		const largest = within(stat).getByText(/Largest:/);
		expect(largest.className).toContain('rounded-full');
		expect(within(largest).getByText('$8,500').className).toContain('font-semibold');
	});

	it('a card kebab opens the production menu: 8px corners, 4px under the button, a pressable refresh row', async () => {
		await renderWith(LIVE);
		await fireEvent.click(screen.getByRole('button', { name: 'Pipeline menu' }));
		const refresh = screen.getByRole('button', { name: /Refresh from Notion/ });
		expect(refresh.className).toContain('bn-pressable');
		const menu = refresh.parentElement as HTMLElement;
		expect(menu.className).toContain('rounded-[8px]');
		expect(menu.className).toContain('top-9');
		expect(menu.getAttribute('style')).toContain('--bn-bg-2');
	});

	it('the drawer keeps production shapes: rounded Open in Notion and close buttons', async () => {
		const { container } = await renderWith(LIVE);
		await fireEvent.click(container.querySelector('[data-deal="c"]')!);
		const drawer = container.querySelector('[data-part="drawer"]') as HTMLElement;
		expect(within(drawer).getByRole('link', { name: /Open in Notion/ }).className).toContain('rounded-[9px]');
		expect(within(drawer).getByRole('button', { name: 'Close' }).className).toMatch(/rounded/);
		expect(drawer.getAttribute('style')).toContain('--bn-bg-2');
	});
});

describe('/os/brand-deals: honest modes', () => {
	it('seeded: warns, says why, and names no hub', async () => {
		const seeded: BrandDealsBody = {
			...LIVE,
			mode: 'seeded',
			syncedAt: null,
			detail: 'Showing examples. Plant NOTION_API_KEY to go live.',
			deals: DEALS.map((d) => ({ ...d, seeded: true })),
			view: { ...VIEW, hubUrl: null }
		};
		const { container } = await renderWith(seeded);
		expect(screen.getByText('seeded · connect Notion')).toBeTruthy();
		expect(screen.getByText(/Showing examples\. Plant NOTION_API_KEY/)).toBeTruthy();
		expect(screen.queryAllByRole('link', { name: /Open in Notion/ })).toHaveLength(0);
		await fireEvent.click(container.querySelector('[data-deal="a"]')!);
		expect(screen.getByText('example row')).toBeTruthy();
	});

	it('error: the figures read unknown, never zero, and the list says Notion was unreadable', async () => {
		const { container } = await renderWith({ deals: [], mode: 'error', detail: 'Key set but the fetch failed: 401.', syncedAt: null, view: null });
		expect(screen.getByText('Notion error')).toBeTruthy();
		expect(container.querySelector('[data-part="pipeline-unknown"]')).toBeTruthy();
		expect(container.querySelectorAll('[data-unknown]').length).toBeGreaterThanOrEqual(4);
		expect(screen.getByText('open pipeline unknown until Notion answers')).toBeTruthy();
		expect(screen.getByText('Notion could not be read, so no deals are shown.')).toBeTruthy();
		expect(screen.queryByText('$0')).toBeNull();
	});

	it('a failed first load says so and retries', async () => {
		fetchMock.mockRejectedValueOnce(new Error('backend down (HTTP 502)')).mockResolvedValueOnce(LIVE);
		const { container } = render(Page);
		await waitFor(() => expect(container.querySelector('[data-part="load-error"]')).toBeTruthy());
		expect(screen.getByText(/Could not load brand deals: backend down/)).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
		await screen.findByText('live · Notion');
	});

	it('refreshes every 60s and keeps the last good read when a refresh fails', async () => {
		vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
		fetchMock.mockResolvedValueOnce(LIVE).mockRejectedValueOnce(new Error('timeout (HTTP 504)'));
		render(Page);
		await screen.findByText('live · Notion');
		vi.advanceTimersByTime(60_000);
		await screen.findByText('refresh failed · showing last read');
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(screen.getByText('live · Notion')).toBeTruthy();
	});
});

// Ported from FounderOS v1 tests/brand-deals-slab.test.ts: the layout contract.
describe('/os/brand-deals slab contract', () => {
	const page = read(resolve(here, '+page.svelte'));
	const board = read(resolve(libDir, 'DealBoard.svelte'));
	const chart = read(resolve(libDir, 'PipelineChart.svelte'));

	it('the page hands Notion to the board, and the slab owns its title row (no PageHeader)', () => {
		expect(page).toContain("'/pages/brand-deals'");
		expect(page).toContain('<DealBoard');
		expect(page).not.toContain('PageHeader');
	});

	it('the board is built on the hatched PipelineChart hero, the view module and the kit', () => {
		expect(board).toMatch(/from '\.\/PipelineChart\.svelte'/);
		expect(board).toMatch(/from '\.\/view'/);
		expect(board).toMatch(/from '\$lib\/founderos\/kit'/);
		expect(chart).toContain('patternTransform="rotate(45)"');
	});

	it('refreshes from the GET route and nothing else: read-only, no send/draft/override', () => {
		for (const src of [page, board]) {
			expect(src).not.toMatch(/method:\s*'POST'/);
			expect(src).not.toMatch(/brand-deals\/(send|draft|override)/);
		}
	});

	it('every deal deep-links into Notion', () => {
		expect(board).toContain('notionUrl');
		expect(board).toContain('Open in Notion');
	});

	it('no em dashes, no emoji', () => {
		for (const src of [page, board, chart, read(resolve(libDir, 'CardMenu.svelte'))]) {
			expect(src).not.toContain('—');
			expect(src).not.toMatch(/\p{Extended_Pictographic}/u);
		}
	});
});
