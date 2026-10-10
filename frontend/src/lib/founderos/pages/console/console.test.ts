import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import OsHome from '../../../../routes/(founderos)/os/+page.svelte';

// The (founderos) layout load hands every /os page the signed-in session.
vi.mock('$app/state', () => ({
	page: { data: { user: { name: 'Alex Rivera', email: 'alex@founderos.local' } }, url: new URL('http://localhost/os'), params: {} }
}));
import Interject from './Interject.svelte';
import { brainTile, greeting, healthCells, operatorName, relativeTime, sparkBarHeights, type ConsoleView } from './console';

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

const NOW = Date.now();
const iso = (msAgo: number) => new Date(NOW - msAgo).toISOString();

function view(over: Partial<ConsoleView> = {}): ConsoleView {
	return {
		generatedAt: iso(0),
		hero: [
			{ text: '1 run failed', tone: 'err' },
			{ text: '3 agents live', tone: 'ok' },
			{ text: '2 idle', tone: 'dim' }
		],
		systems: { connected: 19, total: 24, bars: ['connected', 'error', 'not_configured'] },
		agents: { active: 3, total: 5, spark: [0, 1, 0, 2, 0, 1, 4] },
		comms: { inbound: 7, spark: [1, 0, 0, 0, 2, 0, 5] },
		brain: {
			connected: true,
			enginesUp: 2,
			enginesTotal: 2,
			workspaces: 7,
			health: 90,
			status: 'warnings',
			engines: [
				{ name: 'hub', state: 'connected', up: true, detail: '5 workspaces', workspaces: 5, homes: ['launchpad-cohort', 'founderos'] },
				{ name: 'macbook', state: 'connected', up: true, detail: '2 workspaces', workspaces: 2, homes: ['personal'] },
				{ name: 'mini', state: 'not_configured', up: false, detail: 'not staged on this machine', workspaces: null, homes: ['hermes'] }
			]
		},
		volume: {
			runsToday: 12,
			failedToday: 1,
			agentsToday: 4,
			meters: [
				{ label: 'Systems connected (19/24)', frac: 19 / 24, display: '79%', hue: 'var(--bn-accent)' },
				{ label: 'Agents live (3/5)', frac: 0.6, display: '60%', hue: 'var(--bn-text-2)' },
				{ label: 'Runs OK today (11/12)', frac: 11 / 12, display: '11 ok · 1 failed', hue: 'var(--bn-warn)' },
				{ label: 'Optimal Engine health', frac: 0.9, display: '90 / 100', hue: 'var(--bn-text)' }
			]
		},
		chargedTodayCents: 600000,
		activity: [
			{ label: 'Sep 29', count: 3 },
			{ label: 'Sep 30', count: 9 }
		],
		activityTotal: 12,
		mix: [
			{ label: 'email', count: 4 },
			{ label: 'whatsapp', count: 2 },
			{ label: 'slack', count: 1 }
		],
		feedCount: 31,
		sources: [
			{ source: 'email', state: 'ok' },
			{ source: 'whatsapp', state: 'ok' },
			{ source: 'slack', state: 'error', detail: 'HTTP 500' }
		],
		attention: { count: 9, headline: '7 inbound · 1 failed run · 1 connector down', frac: 0.4 },
		doneCount: 12,
		done: [
			{ key: 'run-1', head: '✓', tone: 'ok', body: 'plaud · 20 recordings ×3', at: iso(5 * 60_000) },
			{ key: 'charge-1', head: '$', tone: 'accent', body: '$6,000 · August retainer', at: iso(2 * 3600_000) }
		],
		connections: [],
		errors: {},
		...over
	};
}

let fetchMock: ReturnType<typeof vi.fn>;
let interjectReply: Response = json({ ok: true, route: 'note' });
function route(consoleBody: unknown, consoleStatus = 200, deliverables: unknown = { groups: [], decisions: {} }, delStatus = 200) {
	fetchMock.mockImplementation((url: string) => {
		if (url.endsWith('/pages/console')) return Promise.resolve(json(consoleBody, consoleStatus));
		if (url.includes('/pages/board/deliverables')) return Promise.resolve(json(deliverables, delStatus));
		if (url.endsWith('/pages/interject')) return Promise.resolve(interjectReply);
		return Promise.resolve(json({})); // the CSRF bootstrap
	});
}

beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
	vi.spyOn(window, 'matchMedia').mockImplementation(
		(q: string) => ({ matches: q.includes('reduce'), media: q, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList
	);
});
afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
});

describe('console helpers', () => {
	it('greets by the hour', () => {
		expect(greeting(new Date('2026-09-30T03:00:00'))).toBe('Late night');
		expect(greeting(new Date('2026-09-30T09:00:00'))).toBe('Good morning');
		expect(greeting(new Date('2026-09-30T13:00:00'))).toBe('Good afternoon');
		expect(greeting(new Date('2026-09-30T20:00:00'))).toBe('Good evening');
	});
	it('says how long ago, compactly', () => {
		const now = Date.parse('2026-09-30T12:00:00Z');
		expect(relativeTime('2026-09-30T11:59:40Z', now)).toBe('just now');
		expect(relativeTime('2026-09-30T11:15:00Z', now)).toBe('45m');
		expect(relativeTime('2026-09-30T07:00:00Z', now)).toBe('5h');
		expect(relativeTime('2026-09-27T12:00:00Z', now)).toBe('3d');
	});
	it('SparkBars is the artboard formula: 3px floor, 15px of travel', () => {
		expect(sparkBarHeights([0, 5, 10])).toEqual([3, 10.5, 18]);
		expect(sparkBarHeights([0, 0])).toEqual([3, 3]);
	});
	it('the health meter is prod HealthMeter: ten cells lit to score/10, graded 70/40; unknown lights none', () => {
		expect(healthCells(90)).toEqual({ lit: 9, tone: 'accent' });
		expect(healthCells(100)).toEqual({ lit: 10, tone: 'accent' });
		expect(healthCells(55)).toEqual({ lit: 6, tone: 'warn' });
		expect(healthCells(20)).toEqual({ lit: 2, tone: 'err' });
		expect(healthCells(null)).toEqual({ lit: 0, tone: 'dim' });
	});
	it('the brain tile reads like prod G-Brain health: score / 100 · status', () => {
		const b = { connected: true, enginesUp: 2, enginesTotal: 2, workspaces: 6, health: 90, status: 'warnings', engines: [] };
		expect(brainTile(b)).toEqual({ value: 90, unit: '/ 100 · warnings' });
		expect(brainTile({ ...b, health: 100, status: 'ok' })).toEqual({ value: 100, unit: '/ 100 · ok' });
		expect(brainTile({ ...b, enginesUp: 0, health: null, status: 'offline' })).toEqual({ value: null, unit: '/ 100 · offline' });
		expect(brainTile({ ...b, enginesUp: 0, enginesTotal: 0, health: null, status: 'not configured' })).toEqual({ value: null, unit: '/ 100 · not configured' });
	});
});

describe('/os operator console', () => {
	it('renders the state-of-the-world line, the pulse row and the volume card from /pages/console', async () => {
		route(view());
		const { container } = render(OsHome);
		await waitFor(() => expect(screen.getByText('1 run failed')).toBeTruthy());
		expect(screen.getByRole('heading', { level: 1 }).textContent).toMatch(/, Alex$/);
		expect(fetchMock.mock.calls.some((c) => c[0] === '/api/founderos/pages/console')).toBe(true);

		const tiles = container.querySelectorAll('[data-part="tile"]');
		expect(tiles.length).toBe(4);
		expect([...tiles].map((t) => t.getAttribute('href'))).toEqual(['/os/integrations', '/os/agents', '/os/comms', '/os/brain']);
		expect(tiles[0].textContent).toContain('/ 24 connected');
		// prod tiles read in title case: Systems · Agents live · Communications · G-Brain health
		expect([...tiles].map((t) => t.querySelector('[data-part="label"]')?.textContent?.trim())).toEqual([
			'Systems',
			'Agents live',
			'Communications',
			'Optimal Engine health'
		]);
		expect(tiles[3].textContent).toContain('90');
		expect(tiles[3].textContent).toContain('/ 100 · warnings');
		expect(tiles[3].querySelector('[data-lit]')?.getAttribute('data-lit')).toBe('9');
		// the systems foot is one bar per connector, coloured by state
		expect(tiles[0].querySelectorAll('rect').length).toBe(3);

		expect(screen.getByText('Operating volume')).toBeTruthy();
		expect(screen.getByText('1 failed')).toBeTruthy();
		expect(screen.getByText('$6,000 charged')).toBeTruthy();
		expect(screen.getByText('agent runs today across 4 agents')).toBeTruthy();
		expect(screen.getAllByText('Optimal Engine health').length).toBe(2); // the tile and the volume meter
		expect(screen.getByText('90 / 100')).toBeTruthy();
		expect(screen.getByText('19 systems · 3 agents · brain 90/100')).toBeTruthy();
	});

	it('renders the second row: activity line, inbound mix and the attention card', async () => {
		route(view());
		render(OsHome);
		await waitFor(() => expect(screen.getByText('Agent activity')).toBeTruthy());
		expect(screen.getByText('agent runs, last 14 days')).toBeTruthy();
		expect(screen.getByText('Inbound mix')).toBeTruthy();
		expect(screen.getByText('latest messages in the feed')).toBeTruthy();
		expect(screen.getByText('slack unreachable: HTTP 500')).toBeTruthy();
		expect(screen.getByText('Needs you now')).toBeTruthy();
		expect(screen.getByText('7 inbound · 1 failed run · 1 connector down')).toBeTruthy();
		expect(screen.getByText('12 things already done today.')).toBeTruthy();
	});

	it('the done ledger: a finished run is a ✓, a payment is a $', async () => {
		route(view());
		render(OsHome);
		await waitFor(() => expect(screen.getByText('Done today')).toBeTruthy());
		expect(screen.getByText('12 since midnight')).toBeTruthy();
		expect(screen.getByText('plaud · 20 recordings ×3')).toBeTruthy();
		expect(screen.getByText('$6,000 · August retainer')).toBeTruthy();
		expect(screen.getByText('✓')).toBeTruthy();
		expect(screen.getByText('$')).toBeTruthy();
	});

	it('ends where prod ends: talk to the OS and done today, with no brain core below', async () => {
		route(view());
		const { container } = render(OsHome);
		await waitFor(() => expect(screen.getByText('Done today')).toBeTruthy());
		expect(container.querySelector('[data-part="brain-core"]')).toBeNull();
		// the slab restyles every card head as prod's .os-slab does: 19px title-case labels
		const heads = [...container.querySelectorAll('[data-part="card-head"]')].map((h) => h.querySelector('[data-part="label"], h2')?.textContent?.trim());
		expect(heads).toEqual(expect.arrayContaining(['Needs you', 'Interject', 'Done today']));
		expect(screen.queryByText('Brain core')).toBeNull();
	});

	it('unknowns read unknown: no roster, no Stripe, no engines', async () => {
		route(
			view({
				agents: { active: null, total: null, spark: null },
				chargedTodayCents: null,
				brain: { connected: false, enginesUp: 0, enginesTotal: 0, workspaces: null, health: null, status: 'not configured', engines: [] },
				errors: { roster: 'founderos workspace missing', charges: 'stripe: no key' }
			})
		);
		const { container } = render(OsHome);
		await waitFor(() => expect(container.querySelectorAll('[data-part="tile"]').length).toBe(4));
		const tiles = container.querySelectorAll('[data-part="tile"]');
		expect(tiles[1].textContent).toContain('—');
		expect(tiles[1].textContent).toContain('roster unknown');
		expect(tiles[3].textContent).toContain('—');
		expect(tiles[3].textContent).toContain('/ 100 · not configured');
		expect(container.textContent).not.toContain('$0 charged');
		expect(screen.getByText(/founderos workspace missing/)).toBeTruthy();
	});

	it('an unreachable backend says so instead of drawing zeros', async () => {
		route({ error: 'backend down' }, 503);
		const { container } = render(OsHome);
		await waitFor(() => expect(screen.getByText(/backend down/)).toBeTruthy());
		expect(container.querySelectorAll('[data-part="tile"]').length).toBe(0);
	});

	it('pulse tiles are pressable rows with the lens; the charts take prod ramp hues', async () => {
		route(view());
		const { container } = render(OsHome);
		await waitFor(() => expect(container.querySelectorAll('[data-part="tile"]').length).toBe(4));
		for (const t of container.querySelectorAll('[data-part="tile"]')) {
			expect(t.className).toContain('bn-pressable');
			expect(t.className).toContain('is-row');
			expect(t.getAttribute('data-lens')).toBe('r');
			// FounderOS v1 .os-slab .rounded-tile: 22px padding, the slab's 10px gap
			// and 34px figure (no private-build 6px / 28px / 14px overrides)
			expect(t.className).toContain('p-[22px]');
			expect(t.className).not.toMatch(/gap-1\.5!|py-3\.5!|px-\[18px\]!/);
			const figure = t.children[1] as HTMLElement;
			expect(figure.className).not.toContain('text-[28px]');
			expect(figure.className).not.toContain('leading-none');
		}
		expect(container.innerHTML).toContain('color-mix(in oklab, var(--bn-brain-2) 70%, var(--bn-text))');
	});

	it('the hero row is Needs you beside Operating volume, the queue filling the volume card\'s height', async () => {
		route(view(), 200, { groups: [], decisions: [], available: true, reason: '', needsYou: { open: [], decided: [] } });
		const { container } = render(OsHome);
		await waitFor(() => expect(screen.getByText('Nothing needs you.')).toBeTruthy());
		const slot = container.querySelector('[data-part="needs-you-slot"]') as HTMLElement;
		expect(slot.className).toContain('relative');
		expect(slot.nextElementSibling?.textContent).toContain('Operating volume');
		// v1 spacing: the pulse row sits 24px above the hero row, rows 24px apart
		const pulse = container.querySelector('[data-part="tile"]')!.parentElement!;
		expect(pulse.className).toContain('mb-6');
		const rows = [...container.querySelectorAll('.grid')].filter((g) => /\bmt-/.test(g.className));
		expect(rows.length).toBeGreaterThanOrEqual(2);
		for (const r of rows) expect(r.className).toContain('mt-6');
		expect(screen.getByText('ranked by the Conductor · people first, then deadlines')).toBeTruthy();
	});
});

describe('interject composer', () => {
	it('sends to /pages/interject with the pinned route and shows the honest receipt', async () => {
		route(view());
		render(OsHome);
		await waitFor(() => expect(screen.getByText('Interject')).toBeTruthy());
		interjectReply = json({ ok: true, route: 'task', ref: 'FOS-9' });
		const box = screen.getByPlaceholderText(/Enter sends/) as HTMLTextAreaElement;
		await fireEvent.input(box, { target: { value: 'call Nick back' } });
		await fireEvent.click(screen.getByText('→ Board'));
		expect(screen.getByText('route pinned')).toBeTruthy();
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(screen.getByText(/FOS-9 · on the board/)).toBeTruthy());
		const call = fetchMock.mock.calls.find((c) => String(c[0]).endsWith('/pages/interject'));
		expect(call).toBeTruthy();
		expect(JSON.parse(call![1].body)).toEqual({ text: 'call Nick back', route: 'task' });
	});

	it('the receipt timer is cleared when the composer goes away', async () => {
		route(view());
		interjectReply = json({ ok: true, route: 'note' }); // a Response body reads once
		const r = render(Interject);
		const box = screen.getByPlaceholderText(/Enter sends/) as HTMLTextAreaElement;
		await fireEvent.input(box, { target: { value: 'note this' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(screen.getByText(/captured to the Optimal Engine/)).toBeTruthy());
		const clear = vi.spyOn(globalThis, 'clearTimeout');
		r.unmount();
		expect(clear).toHaveBeenCalled();
		clear.mockRestore();
	});

	it('a refused interject shows the failure, not a fake receipt', async () => {
		route(view());
		render(OsHome);
		await waitFor(() => expect(screen.getByText('Interject')).toBeTruthy());
		interjectReply = json({ ok: false, route: 'task', error: 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)' }, 502);
		const box = screen.getByPlaceholderText(/Enter sends/) as HTMLTextAreaElement;
		await fireEvent.input(box, { target: { value: 'todo: x' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(screen.getByText(/failed to land: bridge: outbound writes are disabled/)).toBeTruthy());
	});

	it('looks like prod InterjectComposer: toggle-chip routes, a rounded input, the white send disc', async () => {
		route(view());
		const { container } = render(OsHome);
		await waitFor(() => expect(screen.getByText('Interject')).toBeTruthy());
		const chip = screen.getByText('→ Board').closest('button') as HTMLElement;
		expect(chip.className).toContain('bn-toggle-chip');
		expect(chip.getAttribute('data-on')).toBe('false');
		await fireEvent.click(chip);
		expect(chip.getAttribute('data-on')).toBe('true');
		const box = container.querySelector('textarea[data-part="input"]') as HTMLElement;
		expect(box.className).toContain('bn-interject-input');
		const send = screen.getByRole('button', { name: 'send' });
		expect(send.getAttribute('data-tone')).toBe('primary');
		expect(send.className).toContain('rounded-full');
	});

	it('the chips read as routes and there is no auto chip', async () => {
		route(view());
		render(OsHome);
		await waitFor(() => expect(screen.getByText('Interject')).toBeTruthy());
		expect(screen.getByText('→ Optimal Engine')).toBeTruthy();
		expect(screen.getByText('→ Board')).toBeTruthy();
		expect(screen.getByText('→ Agent')).toBeTruthy();
		expect(screen.queryByText('auto')).toBeNull();
	});
});

describe('operatorName', () => {
	it('greets the signed-in operator by first name', () => {
		expect(operatorName({ name: 'Alex Rivera', email: 'alex@founderos.local' })).toBe('Alex');
	});
	it('falls back to the email local part, then to a neutral word', () => {
		expect(operatorName({ name: '  ', email: 'sam@vantage.example' })).toBe('sam');
		expect(operatorName({})).toBe('there');
		expect(operatorName(undefined)).toBe('there');
	});
});
