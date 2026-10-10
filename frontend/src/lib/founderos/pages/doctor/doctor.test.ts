import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import DoctorPage from '../../../../routes/(founderos)/os/doctor/+page.svelte';
import { nearestPillarLayer, radarPoint, relativeTime } from './radar';
import type { DoctorBody, PillarAxis } from './types';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const axes: PillarAxis[] = [
	{ id: 'dept-sales', label: 'Sales', color: '#fff', score: 90, roster: 100, freshness: 70, sop: 100 },
	{ id: 'dept-finance', label: 'Finances', color: '#fff', score: 40, roster: 50, freshness: 15, sop: 30 },
	{ id: 'dept-comms', label: 'Communications', color: '#fff', score: 15, roster: 0, freshness: 15, sop: 0 }
];

function body(over: Partial<DoctorBody> = {}): DoctorBody {
	return {
		generatedAt: '2026-09-30T15:00:00Z',
		windowDays: 14,
		engines: [
			{
				name: 'hub',
				url: 'http://127.0.0.1:4211',
				reachable: true,
				health: 'up',
				workspaces: ['launchpad-cohort', 'founderos', 'vantage', 'personal-brand'],
				checks: [],
				stores: [],
				searches: 619,
				uptimeMs: 1000
			},
			{
				name: 'macbook',
				url: 'http://127.0.0.1:4210',
				reachable: false,
				error: 'health: dial tcp 127.0.0.1:4210: connection refused',
				workspaces: null,
				checks: null,
				stores: null,
				searches: null,
				uptimeMs: null
			}
		],
		checks: [
			{ engine: 'hub', name: 'sqlite_integrity', status: 'ok', message: 'ok' },
			{ engine: 'hub', name: 'verified_backup', status: 'error', message: ':no_verified_backup' }
		],
		score: 25,
		volume: {
			headline: 25,
			counts: { ok: 1, warn: 0, fail: 1, total: 2 },
			chips: [
				{ tone: 'ok', text: '1 passing' },
				{ tone: 'err', text: '1 failing' }
			],
			caption: '2 checks · 1/2 engines up · 747 contexts',
			meters: [
				{ label: 'Engine health (25/100)', frac: 0.25, display: '25%', hue: 'var(--bn-accent)' },
				{ label: 'Checks passing (1/2)', frac: 0.5, display: '1 ok · 1 flagged', hue: 'var(--bn-warn)' },
				{ label: 'Storage layers live (3/3)', frac: 0.67, display: '67%', hue: 'var(--bn-text-2)' },
				{ label: 'Pillar health (3 pillars)', frac: 0.48, display: '48/100', hue: 'var(--bn-text)' }
			],
			foot: '4 workspaces · 3 pillars · 2/3 layers live',
			series: [
				{ label: 'Sep 29', count: 1 },
				{ label: 'Sep 30', count: 2 }
			],
			runsInWindow: 3,
			failedInWindow: 1,
			store: {
				cols: [
					{ label: 'chunks', count: 37559 },
					{ label: 'contexts', count: 747 },
					{ label: 'claims', count: 721 }
				],
				total: 747,
				top: { name: 'chunks', count: 37559 }
			},
			insight: { value: 1, headline: '0 warn · 1 failing.', body: 'hub · verified_backup', frac: 0.5 },
			enginesUp: 1,
			enginesTotal: 2,
			workspaces: 4,
			claims: 721,
			facts: 0,
			contexts: 747
		},
		layers: [
			{ name: 'hub engine', sub: 'http://127.0.0.1:4211 · 4 workspaces · health up', val: 'LIVE', state: 'connected' },
			{ name: 'macbook engine', sub: 'http://127.0.0.1:4210 · refused', val: 'UNREACHABLE', state: 'error' },
			{ name: 'relational', sub: 'contexts, claims · on 1 of 2 engines', val: '40,266 rows', state: 'connected' }
		],
		axes,
		relational: { ok: true },
		brain: { agents: ['data-agent'], last: { agentId: 'data-agent', finishedAt: new Date(Date.now() - 3 * 3600_000).toISOString(), ok: true } },
		...over
	};
}

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

describe('radar geometry', () => {
	it('puts the first axis at the top, then clockwise', () => {
		const [x, y] = radarPoint(0, 4, 100, 200);
		expect(x).toBeCloseTo(200);
		expect(y).toBeCloseTo(100);
		const [x1, y1] = radarPoint(1, 4, 100, 200);
		expect(x1).toBeCloseTo(300);
		expect(y1).toBeCloseTo(200);
	});
	it('sifts to whichever layer ring the cursor is on, and never throws on no axes', () => {
		const flat: PillarAxis[] = [0, 1, 2, 3].map((i) => ({ id: `d${i}`, label: `D${i}`, color: '', score: 100, roster: 50, freshness: 20, sop: 80 }));
		// on the score ring (radius 172 at the top axis) → score
		expect(nearestPillarLayer({ x: 260, y: 260 - 172 }, flat, 172, 260)).toBe('score');
		// halfway up the top spoke → roster (50)
		expect(nearestPillarLayer({ x: 260, y: 260 - 86 }, flat, 172, 260)).toBe('roster');
		expect(nearestPillarLayer({ x: 0, y: 0 }, [], 172, 260)).toBe('score');
	});
	it('relativeTime reads minutes, hours and days', () => {
		const now = Date.parse('2026-09-30T15:00:00Z');
		expect(relativeTime('2026-09-30T14:58:00Z', now)).toBe('2m ago');
		expect(relativeTime('2026-09-30T12:00:00Z', now)).toBe('3h ago');
		expect(relativeTime('2026-09-27T15:00:00Z', now)).toBe('3d ago');
		expect(relativeTime('2026-09-30T14:59:50Z', now)).toBe('just now');
	});
});

describe('/os/doctor', () => {
	it('reads like production: no retirement blockquote, a prod-shaped subtitle', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Doctor');
		await waitFor(() => expect(screen.getByText('Health Volume')).toBeTruthy());
		// prod has no explainer under the title; the engine data speaks for itself
		expect(screen.queryByText(/Doctor was GBrain health/)).toBeNull();
		// v1: "<store> · N pages · last run 8d ago · data-agent"
		expect(screen.getByText('hub + macbook · 747 pages · last run 3h ago · data-agent')).toBeTruthy();
		expect(fetchMock.mock.calls[0][0]).toContain('/founderos/pages/doctor');
	});

	it('the radar keeps production size: capped at 560px wide', async () => {
		fetchMock.mockResolvedValue(json(body()));
		const { container } = render(DoctorPage);
		await waitFor(() => expect(container.querySelector('svg[aria-label^="Pillar health radar"]')).toBeTruthy());
		// app.css caps every svg at max-width:100% outside the utilities layer, so
		// the cap has to be inline or the radar grows with the panel
		const svg = container.querySelector('svg[aria-label^="Pillar health radar"]') as SVGElement;
		expect(svg.getAttribute('style')).toContain('max-width: 560px');
	});

	it('Health Volume carries production meters: engine health, checks, storage layers, pillars', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('Storage layers live (3/3)')).toBeTruthy());
		expect(screen.queryByText(/Engines up/)).toBeNull();
		expect(screen.queryByText(/Facts promoted/)).toBeNull();
	});

	it('renders the slab sections from the payload', async () => {
		fetchMock.mockResolvedValue(json(body()));
		const { container } = render(DoctorPage);
		await waitFor(() => expect(screen.getByText('Pillar Health')).toBeTruthy());
		for (const t of ['Health Volume', 'Brain Runs', 'Brain Store', 'Doctor Core', 'Storage layers', 'Pipeline', 'Query path']) {
			expect(screen.getAllByText(t).length).toBeGreaterThan(0);
		}
		// v1's "N pages on disk": engine pages, singular engine reads singular
		expect(screen.getByText('747 pages across 1 engine')).toBeTruthy();
		// the status chip counts the flagged checks
		expect(screen.getAllByText('1 warning').length).toBeGreaterThan(0);
		// the radar draws one rim label per pillar
		expect(container.querySelector('svg[aria-label^="Pillar health radar"]')).toBeTruthy();
		expect(screen.getByText('COMMUNICATIONS')).toBeTruthy();
		// every storage layer row, with its honest pill
		expect(screen.getByText('macbook engine')).toBeTruthy();
		expect(screen.getAllByText('UNREACHABLE').length).toBeGreaterThan(0);
		// the check list names the failing audit check and its engine
		expect(screen.getAllByText(/verified_backup/).length).toBeGreaterThan(0);
		// claims vs facts is shown in the pipeline
		expect(screen.getByText('claims · rule-extracted')).toBeTruthy();
		// exactly one gradient insight card
		expect(container.querySelectorAll('.bn-insight').length).toBe(1);
		// pipeline stages carry prod's rounded-panel shape
		expect(screen.getByText('Store audit').closest('section')!.className).toContain('rounded-[var(--bn-r-panel)]');
	});

	it('shows an unreachable engine with its error, never as an empty brain', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		await waitFor(() => expect(screen.getAllByText(/connection refused/).length).toBeGreaterThan(0));
	});

	it('the Doctor Core is prod BrainViz on the engines: ring callouts, unknown reads —, never 0', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('ENGINE STORE · — CONTEXTS')).toBeTruthy());
		expect(screen.getByText('VECTOR · — CHUNKS · COSINE')).toBeTruthy();
		expect(screen.getByText('HUB + MACBOOK · 4 WORKSPACES · 1 DOWN')).toBeTruthy();
		expect(screen.getByText('OPTIMAL ENGINE')).toBeTruthy();
		expect(screen.getByText('HEALTH / 100')).toBeTruthy();
	});

	it('an engine that reports its stores draws its tables as inner-ring clusters', async () => {
		const b = body();
		b.engines[0] = {
			...b.engines[0],
			stores: [{ id: 'relational', status: 'available', technology: 'sqlite', rowCount: 747, tables: { contexts: 747, claims: 721 } }]
		};
		fetchMock.mockResolvedValue(json(b));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('ENGINE STORE · 747 CONTEXTS')).toBeTruthy());
		expect(screen.getByText('CONTEXTS · 747')).toBeTruthy();
		expect(screen.getByText('CLAIMS · 721')).toBeTruthy();
	});

	it('the core gauge opens the Doctor · Search pop-out: flagged checks and a read-only brain query', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		const hot = await screen.findByRole('button', { name: /open brain doctor and search/i });
		await fireEvent.click(hot);
		const dialog = await screen.findByRole('dialog');
		expect(dialog.textContent).toContain('Doctor · Search');
		expect(dialog.textContent).toContain('1 of 2 checks');
		expect(dialog.textContent).toContain('verified_backup');
		fetchMock.mockResolvedValueOnce(json({ query: 'x', provider: 'oe', results: [{ title: 'Hit one', snippet: 'snip', source: 'founderos', workspace: 'founderos', engine: 'hub', uri: 'u', score: 1 }] }));
		const input = dialog.querySelector('input') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: 'backups' } });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await waitFor(() => expect(dialog.textContent).toContain('Hit one'));
		expect(fetchMock.mock.calls.at(-1)![0]).toContain('/pages/brain/query?q=backups');
		await fireEvent.keyDown(window, { key: 'Escape' });
		await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
	});

	it('the store audit folds each check across engines into one prod-style line, then the command chips', async () => {
		const b = body({
			checks: [
				{ engine: 'hub', name: 'migrations', status: 'ok', message: '{"applied":52,"expected":52}' },
				{ engine: 'macbook', name: 'migrations', status: 'ok', message: '{"applied":52,"expected":52}' },
				{ engine: 'hub', name: 'verified_backup', status: 'error', message: ':no_verified_backup' }
			]
		});
		fetchMock.mockResolvedValue(json(b));
		const { container } = render(DoctorPage);
		await waitFor(() => expect(screen.getByText('Store audit')).toBeTruthy());
		const stage = screen.getByText('Store audit').closest('section')!;
		expect(stage.querySelectorAll('li').length).toBe(2);
		expect(stage.textContent).toContain('migrations · applied 52 · expected 52');
		expect(stage.textContent).not.toContain('hub ·');
		for (const cmd of ['capture', 'retrieve', 'search', 'audit']) expect(stage.textContent).toContain(cmd);
		// prod's Arrow: a vertical label between the stages
		expect(container.querySelector('[data-part="arrow"]')!.textContent).toContain('ingest');
	});

	it('cards read like prod: Brain Runs and Brain Store titles, store matrix capped at six columns', async () => {
		const b = body();
		b.volume.store.cols = ['a', 'b', 'c', 'd', 'e', 'f', 'g'].map((label, i) => ({ label, count: 10 - i }));
		fetchMock.mockResolvedValue(json(b));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('Brain Runs')).toBeTruthy());
		// v1: "data-agent · last 14 days"
		expect(screen.getByText('data-agent · last 14 days')).toBeTruthy();
		expect(screen.getByText('Brain Store')).toBeTruthy();
		expect(screen.getByText('7 tables')).toBeTruthy();
		const card = screen.getByText('Brain Store').closest('.bn-card') as HTMLElement;
		expect(card.textContent).toContain('f');
		expect([...card.querySelectorAll('span')].some((el) => el.textContent === 'g')).toBe(false);
		expect(screen.getByText('Doctor Core')).toBeTruthy();
	});

	it('query path stacks keyword and vector search like prod, with labelled arrows', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('Keyword search')).toBeTruthy());
		expect(screen.getByText('Vector search')).toBeTruthy();
		for (const l of ['route', 'fan out', 'merge', 'answer']) expect(screen.getAllByText(l).length).toBeGreaterThan(0);
	});

	it('radar vertices are prod circles and the line reads "/ 100 health"', async () => {
		fetchMock.mockResolvedValue(json(body()));
		const { container } = render(DoctorPage);
		await waitFor(() => expect(container.querySelector('svg[aria-label^="Pillar health radar"]')).toBeTruthy());
		const svg = container.querySelector('svg[aria-label^="Pillar health radar"]')!;
		expect(svg.querySelectorAll('circle').length).toBe(axes.length * 4);
		expect(svg.querySelectorAll('rect').length).toBe(0);
		expect(screen.getAllByText('/ 100 health').length).toBeGreaterThan(0);
	});

	it('says so when no engine answers', async () => {
		const b = body({
			engines: [{ ...body().engines[1] }],
			checks: [],
			score: null
		});
		b.volume = { ...b.volume, headline: null, chips: [{ tone: 'err', text: 'unreachable' }], insight: { value: null, headline: 'No engine answered.', body: 'memory is unreachable, not empty.', frac: 0 } };
		fetchMock.mockResolvedValue(json(b));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('No engine answered.')).toBeTruthy());
		expect(screen.getAllByText('unreachable').length).toBeGreaterThan(0);
	});

	it('says pillars are unknown when Postgres could not be read', async () => {
		fetchMock.mockResolvedValue(json(body({ axes: null, relational: { ok: false, error: 'Postgres is not connected' } })));
		render(DoctorPage);
		await waitFor(() => expect(screen.getAllByText(/Postgres is not connected/).length).toBeGreaterThan(0));
	});

	it('surfaces a failed read instead of a blank page', async () => {
		fetchMock.mockResolvedValue(json({ error: 'boom' }, 500));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText(/doctor unreachable: boom/)).toBeTruthy());
	});

	it('the primary button names the engine doctor, like prod names gbrain doctor --fast, and re-reads', async () => {
		fetchMock.mockResolvedValue(json(body()));
		render(DoctorPage);
		await waitFor(() => expect(screen.getByText('Health Volume')).toBeTruthy());
		const btn = screen.getByRole('button', { name: /engine doctor/i });
		expect(btn.textContent).toContain('▸ engine doctor');
		expect(btn.className).toContain('is-primary');
		let release: (r: Response) => void = () => {};
		fetchMock.mockReturnValueOnce(new Promise<Response>((r) => (release = r)));
		await fireEvent.click(btn);
		await waitFor(() => expect(btn.textContent).toContain('checking'));
		release(json(body()));
		await waitFor(() => expect(btn.textContent).toContain('25/100'));
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});
});
