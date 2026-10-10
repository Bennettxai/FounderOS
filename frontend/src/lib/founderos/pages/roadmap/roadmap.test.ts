import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import RoadmapPage from '../../../../routes/(founderos)/os/roadmap/+page.svelte';
import RoadmapBoard from './RoadmapBoard.svelte';
import { groupRoadmapByQuarter, pctOf, quarterLabel, visibleRows } from './roadmap';
import type { PhaseProgress, RoadmapBody, RoadmapItem } from './types';

const item = (id: string, over: Partial<RoadmapItem> = {}): RoadmapItem => ({
	id,
	title: `Title ${id}`,
	quarter: '2026-Q2',
	status: 'done',
	departmentId: 'dept-tech',
	description: `Description ${id}`,
	phaseId: 'phase-1',
	...over
});

const items = [
	item('a', { title: 'Connect Slack workspace', departmentId: 'dept-comms' }),
	item('b', { title: 'Statement ingestion', status: 'now', quarter: '2026-Q3', departmentId: 'dept-finance' }),
	item('c', { title: 'Interaction rebrand', status: 'now', quarter: '2026-Q3', phaseId: 'phase-2' }),
	item('d', { title: 'Auth + remote access', status: 'next', quarter: '2026-Q4', phaseId: 'phase-2' }),
	item('e', { title: 'Board fully inside the OS', status: 'later', quarter: '2026-Q4', phaseId: null, departmentId: null })
];

const phases: PhaseProgress[] = [
	{ phase: { id: 'phase-1', number: 1, title: 'Real Connections', items: [] }, items: [items[0], items[1]], done: 1, total: 2, pct: 50 },
	{ phase: { id: 'phase-2', number: 2, title: 'Real Agents', items: [] }, items: [items[2], items[3]], done: 0, total: 2, pct: 0 }
];

const departments = { 'dept-tech': 'TECH', 'dept-comms': 'Communications', 'dept-finance': 'Finances' };

const body: RoadmapBody = { phases, items, departments, shipped: 1, total: 5 };

const json = (b: unknown, status = 200) =>
	new Response(JSON.stringify(b), { status, headers: { 'content-type': 'application/json' } });

describe('roadmap logic (v1 lib/roadmap.ts)', () => {
	it('groups by quarter, chronologically', () => {
		const g = groupRoadmapByQuarter([items[3], items[0], items[1]]);
		expect(g.map((q) => q.quarter)).toEqual(['2026-Q2', '2026-Q3', '2026-Q4']);
		expect(quarterLabel('2026-Q3')).toBe('2026 · Q3');
	});
	it('a phase reads done/total of the rows it owns; none reads 0', () => {
		expect(pctOf(items, 'phase-1')).toBe(50);
		expect(pctOf(items, 'phase-9')).toBe(0);
	});
	it('the phase narrows, then the status filter applies on top', () => {
		expect(visibleRows(items, 'phase-2', 'All').map((i) => i.id)).toEqual(['c', 'd']);
		expect(visibleRows(items, 'phase-2', 'Next').map((i) => i.id)).toEqual(['d']);
		expect(visibleRows(items, null, 'Now').map((i) => i.id)).toEqual(['b', 'c']);
	});
});

describe('RoadmapBoard', () => {
	afterEach(() => vi.unstubAllGlobals());

	it('renders v1: phase cards with done/total + bar, quarter columns with badges and departments', () => {
		const { container } = render(RoadmapBoard, { phases, items, departments });
		expect(screen.getByText('Phases')).toBeTruthy();
		expect(screen.getByText('PHASE 01')).toBeTruthy();
		expect(screen.getByText('Real Connections')).toBeTruthy();
		expect(screen.getByText('50% · 1/2')).toBeTruthy();
		expect(screen.getByText('0% · 0/2')).toBeTruthy();
		expect(screen.getByText('Quarter by quarter')).toBeTruthy();
		for (const q of ['2026 · Q2', '2026 · Q3', '2026 · Q4']) expect(screen.getByText(q)).toBeTruthy();
		expect(screen.getByText('1/1 done')).toBeTruthy();
		expect(screen.getAllByText('0/2 done')).toHaveLength(2);
		const bars = container.querySelectorAll<HTMLElement>('[data-part="phase-bar"]');
		expect(bars[0].style.width).toBe('50%');
		// badges: Done (ok), Now (accent), Next (warn), Later (ghost)
		const later = screen.getByText('Later', { selector: '.bn-badge' });
		expect(later.className).toContain('border-dashed');
		expect(screen.getByText('Next', { selector: '.bn-badge' }).getAttribute('data-tone')).toBe('warn');
		expect(screen.getAllByText('Done', { selector: '.bn-badge' })[0].getAttribute('data-tone')).toBe('ok');
		expect(screen.getByText('Communications')).toBeTruthy();
		expect(screen.getByText('Finances')).toBeTruthy();
		// a done row is struck through and dimmed; the card list strikes it too
		const doneCard = container.querySelector('[data-item="a"]') as HTMLElement;
		expect(doneCard.className).toContain('opacity-[0.62]');
		expect(within(doneCard).getByText('Connect Slack workspace').className).toContain('line-through');
		// no department, no department line
		expect((container.querySelector('[data-item="e"]') as HTMLElement).textContent).not.toContain('TECH');
	});

	it('selecting a phase narrows the quarters; chips filter on top', async () => {
		const { container } = render(RoadmapBoard, { phases, items, departments });
		await fireEvent.click(screen.getByText('Real Agents'));
		expect(container.querySelectorAll('[data-item]').length).toBe(2);
		expect(screen.queryByText('2026 · Q2')).toBeNull();
		await fireEvent.click(screen.getByRole('button', { name: 'Next' }));
		expect([...container.querySelectorAll('[data-item]')].map((e) => e.getAttribute('data-item'))).toEqual(['d']);
		await fireEvent.click(screen.getByRole('button', { name: 'All' }));
		await fireEvent.click(screen.getByText('Real Agents'));
		expect(container.querySelectorAll('[data-item]').length).toBe(5);
	});

	it('an item opens into its description; mark done PATCHes and the bar moves', async () => {
		const fetchMock = vi.fn().mockResolvedValue(
			json({ item: { ...items[1], status: 'done' }, board: { ...body, items: items.map((i) => (i.id === 'b' ? { ...i, status: 'done' } : i)) } })
		);
		vi.stubGlobal('fetch', fetchMock);
		document.cookie = 'csrf_token=t';
		const { container } = render(RoadmapBoard, { phases, items, departments });
		expect(screen.queryByText('Description b')).toBeNull();
		await fireEvent.click(container.querySelector('[data-item="b"]') as HTMLElement);
		expect(screen.getByText('Description b')).toBeTruthy();
		await fireEvent.click(screen.getByRole('button', { name: 'mark done' }));
		await waitFor(() => expect(fetchMock).toHaveBeenCalled());
		const [url, init] = fetchMock.mock.calls[0];
		expect(url).toBe('/api/founderos/pages/roadmap');
		expect(init.method).toBe('PATCH');
		expect(new Headers(init.headers).get('X-CSRF-Token')).toBe('t');
		expect(JSON.parse(init.body)).toEqual({ id: 'b', status: 'done' });
		await waitFor(() => expect(screen.getByText('100% · 2/2')).toBeTruthy());
		expect((container.querySelectorAll<HTMLElement>('[data-part="phase-bar"]')[0]).style.width).toBe('100%');
	});
});

describe('/os/roadmap page', () => {
	let fetchMock: ReturnType<typeof vi.fn>;
	beforeEach(() => {
		fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
	});
	afterEach(() => vi.unstubAllGlobals());

	it('reads /pages/roadmap: build plan eyebrow, Roadmap title, shipped count', async () => {
		fetchMock.mockResolvedValue(json(body));
		render(RoadmapPage);
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Roadmap');
		expect(screen.getByText('build plan')).toBeTruthy();
		await waitFor(() => expect(screen.getByText('1/5 shipped')).toBeTruthy());
		expect(screen.getByText('Real Connections')).toBeTruthy();
		expect(fetchMock.mock.calls[0][0]).toBe('/api/founderos/pages/roadmap');
	});

	it('an unreachable source says so, never "0/0 shipped"', async () => {
		fetchMock.mockResolvedValue(json({ error: 'roadmap unavailable: no Postgres' }, 503));
		const { container } = render(RoadmapPage);
		await waitFor(() => expect(container.textContent).toContain('roadmap unavailable: no Postgres'));
		expect(container.textContent).not.toContain('0/0 shipped');
	});

	it('an empty roadmap reads empty', async () => {
		fetchMock.mockResolvedValue(json({ phases: [], items: [], departments: {}, shipped: 0, total: 0 }));
		const { container } = render(RoadmapPage);
		await waitFor(() => expect(container.textContent).toContain('No roadmap yet'));
	});
});
