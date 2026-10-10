import { fireEvent, render, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// The real module (FounderosApiError, isGuardRefusal) with the fetch stubbed.
vi.mock('$lib/founderos/api', async (orig) => ({ ...(await orig<typeof import('$lib/founderos/api')>()), founderosFetch: vi.fn(), founderosUrl: (p: string) => `/api/founderos${p}` }));

import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
import AgentsPage from './AgentsPage.svelte';
import Toaster from '$lib/founderos/chrome/Toaster.svelte';
import { toasts } from '$lib/founderos/chrome/toast';
import type { AgentsView, BoardLive, DeliverablesBody } from './types';

const fetchMock = vi.mocked(founderosFetch);

const volume = (over: Partial<BoardLive['volume']> = {}): BoardLive['volume'] => ({
	headline: 2,
	counts: { running: 1, idle: 0, paused: 0, error: 1 },
	chips: [
		{ tone: 'ok', text: '1 running' },
		{ tone: 'err', text: '1 in error' }
	],
	caption: 'seats on the Paperclip board · 1 model',
	meters: [{ label: 'Seats running (1/2)', frac: 0.5, display: '50%', hue: 'var(--bn-ok)' }],
	foot: '1 open task · 1 run in 24h',
	openTasks: 1,
	series: [{ label: 'Sep 30', count: 1 }],
	window: '14d',
	runsInWindow: 1,
	failedInWindow: 0,
	bySeat: [{ label: 'Conductor', count: 1 }],
	lanes: [{ label: 'review', count: 1 }],
	insight: { value: 1, headline: '1 task waiting on review or unblocking.', body: '1 seat in error · 0 failed runs in 24h', frac: 1 },
	...over
});

const board = (): AgentsView['board'] => ({
	connected: true,
	agents: [
		{ id: 'a1', name: 'Conductor', status: 'running', adapterType: 'claude_local', model: 'claude-fable-5', lastHeartbeatAt: null },
		{ id: 'a2', name: 'Tech', status: 'error', adapterType: 'codex', model: null, lastHeartbeatAt: null }
	],
	issues: [
		{ id: 'i1', identifier: 'FOS-1', title: 'Review the launch copy', status: 'in_review', assigneeName: 'Tech', updatedAt: 'u1' },
		{ id: 'i2', identifier: 'FOS-2', title: 'Wire the funnel', status: 'done', assigneeName: 'Forge', updatedAt: '2026-09-29T08:00:00Z' }
	],
	runs: [{ id: 'r1', agentId: 'a1', agentName: null, status: 'succeeded', startedAt: '2026-09-30T08:00:00Z', finishedAt: '2026-09-30T08:00:14Z' }],
	checkedAt: '2026-09-30T09:00:00Z',
	decisions: []
});

const view = (over: Partial<AgentsView> = {}): AgentsView => ({
	boardUrl: 'http://board.example:3100',
	hermesUrl: 'https://hermes.example:9119',
	board: board(),
	volume: volume(),
	stats: { seats: 2, running: 1, openTasks: 1, runs24h: 1, heartbeats24h: 0 },
	...over
});

const deliverables: DeliverablesBody = {
	groups: [{ name: 'Agent files', items: [] }],
	decisions: [],
	dir: '/tmp/ws',
	available: true,
	reason: '',
	needsYou: {
		open: [
			{
				id: 'ws1/STAGED-reply.md',
				name: 'STAGED-reply.md',
				kind: 'file',
				url: null,
				meta: 'ws1',
				modifiedAt: '2026-09-30T08:00:00.000Z',
				sizeBytes: 10,
				accessCode: '',
				title: 'Reply to Sam',
				summary: 'Thanks.',
				revision: 'file|2026-09-30T08:00:00.000Z|10',
				ask: 'staged',
				needsYou: true,
				deadline: null,
				overdue: false,
				label: 'staged · send it',
				action: 'The agent already wrote this. Approve to send it. Dismiss to bin it.',
				person: true,
				glyph: '✉',
				glyphTone: 'ok',
				why: 'a reply is written and unsent'
			}
		],
		decided: []
	}
};

function route(overrides: Record<string, (init?: { method?: string; json?: unknown }) => unknown> = {}) {
	fetchMock.mockImplementation(async (path: string, init?: { method?: string; json?: unknown }) => {
		const key = `${(init?.method ?? 'GET').toUpperCase()} ${path.split('?')[0]}`;
		if (overrides[key]) return overrides[key](init);
		switch (key) {
			case 'GET /pages/agents/view':
				return view();
			case 'GET /pages/board/live':
				return { ...board(), volume: volume(), stats: view().stats };
			case 'GET /pages/board/deliverables':
				return deliverables;
			case 'GET /pages/conductor/chat':
				return { messages: [] };
		}
		return { ok: true };
	});
}

beforeEach(() => {
	// the /os layout mounts the OS-wide toast stack; these tests mount it too
	render(Toaster);
	fetchMock.mockReset();
	window.localStorage.clear();
	window.history.replaceState(null, '', '/os/agents');
});
afterEach(() => {
	vi.useRealTimers();
	toasts.set([]);
});

describe('/agents page', () => {
	it('the tabs and the Needs You badge do not wait for the board view (prod renders them at once)', async () => {
		// the board read can take seconds when Paperclip is slow; deliverables are local
		route({ 'GET /pages/agents/view': () => new Promise(() => {}) });
		const { findByRole, container } = render(AgentsPage);
		expect(await findByRole('tab', { name: /needs you/i })).toBeTruthy();
		await waitFor(() => expect(container.querySelector('[data-badge="warn"]')?.textContent).toBe(String(deliverables.needsYou.open.length)));
	});

	it('the volume rows lead and the cockpit follows them (v1 app/agents/page.tsx)', async () => {
		route();
		const { findByText, getByText, findByRole } = render(AgentsPage);
		expect(await findByText('Real Agents')).toBeTruthy();
		const tabs = await findByRole('tablist');
		const runActivity = await findByText('Run Activity');
		expect(runActivity.compareDocumentPosition(tabs) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
		expect(getByText('seats on the Paperclip board · 1 model')).toBeTruthy();
		expect(getByText('1 task waiting on review or unblocking.')).toBeTruthy();
		expect(getByText('Agent Volume')).toBeTruthy();
		expect(getByText('Runs by Seat')).toBeTruthy();
		expect(getByText('Task Lanes')).toBeTruthy();
		expect(getByText('Open board').closest('a')!.getAttribute('href')).toBe('http://board.example:3100');
		expect(getByText('Open board').closest('a')!.className).toContain('text-[13px]');
	});

	it('the board strip shows seats with models, the run feed and the lanes', async () => {
		route();
		const { findByText, getByText, getAllByText } = render(AgentsPage);
		expect(await findByText('2 seats · 1 running')).toBeTruthy();
		expect(getByText('claude-fable-5 x1 · codex x1')).toBeTruthy();
		expect(getByText('Review the launch copy')).toBeTruthy();
		expect(getAllByText('Conductor').length).toBeGreaterThan(0);
		expect(getByText('14s')).toBeTruthy();
		for (const label of ['Seats', 'Running now', 'Open tasks', 'Runs · 24h', 'Heartbeats · 24h']) expect(getByText(label)).toBeTruthy();
	});

	it('an unreachable board is an honest dead strip, never fake green', async () => {
		route({
			'GET /pages/agents/view': () => view({ board: { ...board(), connected: false, error: 'board unreachable (dial tcp: timeout)', agents: [], issues: [], runs: [] }, volume: volume({ chips: [{ tone: 'err', text: 'board unreachable' }], meters: [], headline: 0 }) })
		});
		const { findByText, queryByText } = render(AgentsPage);
		const dead = await findByText(/No fake data/);
		// production's copy only; the dial error rides in the hover title
		expect(queryByText(/dial tcp: timeout/)).toBeNull();
		expect(dead.getAttribute('title')).toContain('dial tcp: timeout');
		expect(queryByText('Running now')).toBeNull();
	});

	it('an unreachable board reads like prod below the cockpit: zeros with the chip, a square red LED', async () => {
		route({
			'GET /pages/agents/view': () =>
				view({
					board: { ...board(), connected: false, agents: [], issues: [], runs: [] },
					volume: volume({ chips: [{ tone: 'err', text: 'board unreachable' }], meters: [], headline: 0, runsInWindow: 0, openTasks: 0, bySeat: [], lanes: [], series: [], insight: { value: 0, headline: 'Board unreachable.', body: 'No numbers until Paperclip answers on the tailnet.', frac: 0 } })
				})
		});
		const { findByText, queryByText, container } = render(AgentsPage);
		expect(await findByText('Board unreachable.')).toBeTruthy();
		expect(queryByText('offline')).toBeNull();
		// Run Activity, Agent Volume, Task Lanes and the Needs you card all read 0
		const zeros = [...container.querySelectorAll('[data-part="bigstat-value"], [data-part="insight-value"]')].map((n) => n.textContent?.trim());
		expect(zeros.filter((t) => t === '0').length).toBeGreaterThanOrEqual(4);
		const led = container.querySelector('[data-part="board-live"] [data-part="led"]') as HTMLElement;
		expect(led.className).toContain('h-1.5');
		expect(led.className).not.toContain('rounded');
	});

	it('the tabs and the Hermes link wear the interaction layer (pressable, lens)', async () => {
		route();
		const { findAllByRole, findByRole, findByText } = render(AgentsPage);
		for (const t of await findAllByRole('tab')) {
			expect(t.className).toContain('bn-pressable');
			expect(t.getAttribute('data-lens')).toBe('c');
		}
		await fireEvent.click(await findByRole('tab', { name: /Hermes/ }));
		const link = (await findByText(/Open full dashboard/)).closest('a') as HTMLElement;
		expect(link.className).toContain('bn-pressable');
		expect(link.className).toContain('is-dark');
		expect(link.getAttribute('data-lens')).toBe('c');
	});

	it('a seat Run button fires a board heartbeat and says so when the bridge guard refuses it', async () => {
		route({
			'POST /pages/board/agents/a2/run': () => {
				throw new FounderosApiError(403, 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)', { guarded: true });
			}
		});
		const { findByTitle, findByText } = render(AgentsPage);
		await fireEvent.click(await findByTitle('Run Tech heartbeat on the board'));
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/board/agents/a2/run', { method: 'POST' }));
		expect(await findByText(/writes are off/)).toBeTruthy();
	});

	it('tabs are Roster, Needs You (round count), Deliverables, Hermes; Roster is open', async () => {
		route();
		const { findAllByRole, getByRole } = render(AgentsPage);
		const tabs = await findAllByRole('tab');
		expect(tabs.map((t) => t.textContent?.replace(/\d+/g, '').replace(/\s+/g, ' ').trim())).toEqual(['Roster', 'Needs You', 'Deliverables', 'Hermes']);
		expect(getByRole('tab', { name: 'Roster' }).getAttribute('aria-selected')).toBe('true');
		await waitFor(() => expect(getByRole('tab', { name: /Needs You/ }).textContent).toContain('1'));
		expect(getByRole('tab', { name: /Needs You/ }).querySelector('[data-badge="warn"]')?.getAttribute('title')).toBe('1 agent files are waiting on you');
	});

	it('the page is the live board only: no bridge runtime roster below it', async () => {
		route();
		const { findByText, queryByText } = render(AgentsPage);
		expect(await findByText('Run Activity')).toBeTruthy();
		expect(queryByText('CRM Pulse')).toBeNull();
		expect(fetchMock.mock.calls.some((c) => c[0] === '/agents')).toBe(false);
	});

	it('/os/tasks is its own page again (v1): an old ?tab=tasks link lands on the Roster, no task queues here', async () => {
		window.history.replaceState(null, '', '/os/agents?tab=tasks');
		route();
		const { findByText, getByRole, queryByText } = render(AgentsPage);
		expect(await findByText('Run Activity')).toBeTruthy();
		expect(getByRole('tab', { name: 'Roster' }).getAttribute('aria-selected')).toBe('true');
		expect(queryByText('Scheduled jobs')).toBeNull();
		expect(queryByText('Local kanban')).toBeNull();
		expect(fetchMock.mock.calls.some((c) => c[0] === '/pages/tasks')).toBe(false);
	});

	it('switching tabs mirrors ?tab=, the Roster being the bare URL', async () => {
		route();
		const { findByRole } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: 'Hermes' }));
		expect(window.location.search).toBe('?tab=hermes');
		await fireEvent.click(await findByRole('tab', { name: 'Roster' }));
		expect(window.location.search).toBe('');
	});

	it('the Hermes tab says what it embeds in v1 words: the host, no private machine names', async () => {
		route();
		const { findByRole, findByText, queryByText } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: 'Hermes' }));
		expect(await findByText(/Stock Hermes dashboard, embedded live from the host/)).toBeTruthy();
		expect(queryByText(/mini|tailscale/)).toBeNull();
	});

	it('Deliverables is one flat feed: files, proposals and finished board tasks', async () => {
		route();
		const { findByRole, container } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: 'Deliverables' }));
		await waitFor(() => expect(container.querySelector('[data-part="deliverables"]')).toBeTruthy());
		const { findByText, getByRole, getByText } = within(container.querySelector('[data-part="deliverables"]') as HTMLElement);
		expect(await findByText('Wire the funnel')).toBeTruthy();
		expect(getByText(/Board · done · FOS-2 · Forge/)).toBeTruthy();
		expect(getByText('1 delivered · newest first')).toBeTruthy();
		for (const chip of ['All 1', 'Files 0', 'Proposals 0', 'Done tasks 1']) expect(getByRole('button', { name: chip })).toBeTruthy();
		await fireEvent.click(getByRole('button', { name: 'Files 0' }));
		expect(await findByText('Nothing of that kind yet.')).toBeTruthy();
	});

	it('Hermes mounts its iframe only once the tab is opened', async () => {
		route();
		const { findByRole, container } = render(AgentsPage);
		expect(container.querySelector('iframe')).toBeNull();
		await fireEvent.click(await findByRole('tab', { name: /Hermes/ }));
		const frame = container.querySelector('iframe')!;
		expect(frame.getAttribute('src')).toBe('https://hermes.example:9119');
	});

	it('Needs You lists open agent work and records an approval against its revision', async () => {
		route();
		const { findByRole, findByText, getByRole } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		expect(await findByText('Reply to Sam')).toBeTruthy();
		await fireEvent.click(getByRole('button', { name: 'send it' }));
		await waitFor(() =>
			expect(fetchMock).toHaveBeenCalledWith('/pages/board/deliverables/decision', {
				method: 'POST',
				json: { id: 'ws1/STAGED-reply.md', decision: 'approved', decidedRevision: 'file|2026-09-30T08:00:00.000Z|10' }
			})
		);
	});

	it('bulk dismiss takes two clicks, and cancel sends nothing', async () => {
		route();
		const { findByRole, getByRole, queryByRole } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		const decisions = () => fetchMock.mock.calls.filter((c) => c[0] === '/pages/board/deliverables/decision');
		await fireEvent.click(await findByRole('button', { name: /clear all/ }));
		expect(decisions()).toHaveLength(0);
		await fireEvent.click(getByRole('button', { name: 'cancel' }));
		expect(queryByRole('button', { name: /dismiss all/ })).toBeNull();
		await fireEvent.click(getByRole('button', { name: /clear all/ }));
		await fireEvent.click(getByRole('button', { name: /dismiss all 1/ }));
		await waitFor(() => expect(decisions()).toHaveLength(1));
		expect(decisions()[0][1]).toMatchObject({ json: { decision: 'dismissed' } });
	});

	it('a decision in flight cannot be sent twice', async () => {
		route({ 'POST /pages/board/deliverables/decision': () => new Promise(() => {}) });
		const { findByRole, getByRole } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		const approve = await findByRole('button', { name: 'send it' });
		await fireEvent.click(approve);
		await fireEvent.click(getByRole('button', { name: 'sending' }));
		await waitFor(() => expect(fetchMock.mock.calls.filter((c) => c[0] === '/pages/board/deliverables/decision')).toHaveLength(1));
	});

	it('Needs You reads like production: the card head counts, the Conductor’s ranking line, the verbs', async () => {
		route();
		const { findByRole, container } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		await waitFor(() => expect(container.querySelector('[data-card="needsyou"]')).toBeTruthy());
		const { findByText, getByText, getByRole } = within(container.querySelector('[data-card="needsyou"]') as HTMLElement);
		expect(await findByText('Needs you')).toBeTruthy();
		expect(getByText(/1 of 0 agent files are waiting on you/)).toBeTruthy();
		expect(getByText('ranked by the Conductor · people first, then deadlines')).toBeTruthy();
		expect(getByText(/why here · a reply is written and unsent/)).toBeTruthy();
		for (const b of ['send it', 'dismiss', 'snooze']) expect(getByRole('button', { name: b })).toBeTruthy();
	});

	it('Needs You and Deliverables carry no machine-reason line (prod shows none)', async () => {
		route({ 'GET /pages/board/deliverables': () => ({ ...deliverables, available: false, reason: 'the board workspaces directory is not on this machine' }) });
		const { findByRole, findByText, queryByText } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		expect(await findByText('Reply to Sam')).toBeTruthy();
		expect(queryByText(/not on this machine/)).toBeNull();
		await fireEvent.click(await findByRole('tab', { name: /Deliverables/ }));
		expect(await findByText(/delivered · newest first/)).toBeTruthy();
		expect(queryByText(/not on this machine/)).toBeNull();
	});

	it('an outward call held by the bridge guard stays in the queue and says so', async () => {
		route({
			'POST /pages/board/deliverables/decision': () => {
				throw new FounderosApiError(409, 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)', { guarded: true });
			}
		});
		const { findByRole, findByText, getByText } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		await fireEvent.click(await findByRole('button', { name: 'send it' }));
		expect(await findByText('held: writes off')).toBeTruthy();
		expect(getByText('Reply to Sam')).toBeTruthy();
	});

	it('snooze is a two-hour hold in this browser only: nothing is sent, and undo brings it back', async () => {
		route();
		const { findByRole, findByText, getByRole, queryByText } = render(AgentsPage);
		await fireEvent.click(await findByRole('tab', { name: /Needs You/ }));
		await fireEvent.click(await findByRole('button', { name: 'snooze' }));
		expect(queryByText('Reply to Sam')).toBeNull();
		expect((await findByText(/snoozed 2h · Reply to Sam/)).closest('[data-toast]')).toBeTruthy();
		expect(await findByText(/Nothing needs you\./)).toBeTruthy();
		expect(JSON.parse(window.localStorage.getItem('founderos-os:deliverables-snoozed') ?? '{}')['ws1/STAGED-reply.md']).toBeGreaterThan(Date.now());
		expect(fetchMock.mock.calls.some((c) => c[0] === '/pages/board/deliverables/decision')).toBe(false);
		await fireEvent.click(getByRole('button', { name: 'undo' }));
		expect(await findByText('Reply to Sam')).toBeTruthy();
	});

	describe('Deliverables unread (prod useDeliverables: seen map for the tab, opened map for the row)', () => {
		const file = (id: string, revision: string) => ({ ...deliverables.needsYou.open[0], id, name: `${id}.md`, title: `Doc ${id}`, revision, modifiedAt: '2026-09-30T08:00:00.000Z' });
		const withFiles = () =>
			route({ 'GET /pages/board/deliverables': () => ({ ...deliverables, groups: [{ name: 'Agent files', items: [file('a', 'r2'), file('b', 'r1'), file('c', 'r1')] }] }) });

		it('the Deliverables tab carries a red changed badge and a green new badge; looking at the tab clears them', async () => {
			window.localStorage.setItem('founderos-os:deliverables-seen', JSON.stringify({ a: 'r1', b: 'r1' }));
			withFiles();
			const { findByRole, getByRole } = render(AgentsPage);
			const tab = await findByRole('tab', { name: /Deliverables/ });
			await waitFor(() => expect(tab.querySelector('[data-badge="err"]')?.textContent).toBe('1'));
			expect(tab.querySelector('[data-badge="err"]')?.getAttribute('title')).toBe('1 changed since you last looked');
			expect(tab.querySelector('[data-badge="ok"]')?.textContent).toBe('1');
			expect(tab.querySelector('[data-badge="ok"]')?.getAttribute('title')).toBe('1 new since you last looked');
			await fireEvent.click(tab);
			await waitFor(() => expect(getByRole('tab', { name: /Deliverables/ }).querySelector('[data-badge]')).toBeNull());
			expect(JSON.parse(window.localStorage.getItem('founderos-os:deliverables-seen') ?? '{}')).toEqual({ a: 'r2', b: 'r1', c: 'r1' });
		});

		it('a fresh browser badges nothing and adopts the board silently', async () => {
			withFiles();
			const { findByRole } = render(AgentsPage);
			const tab = await findByRole('tab', { name: /Deliverables/ });
			await waitFor(() => expect(window.localStorage.getItem('founderos-os:deliverables-seen')).not.toBeNull());
			expect(tab.querySelector('[data-badge]')).toBeNull();
			await fireEvent.click(tab);
			await waitFor(() => expect(document.querySelector('[data-part="deliverables"]')?.textContent).toContain('Doc a'));
			expect(document.querySelectorAll('[data-part="deliverables"] [aria-label="new"], [data-part="deliverables"] [aria-label="updated"]')).toHaveLength(0);
		});

		it('each row carries its own dot (new / updated) and opening it for review clears only that one', async () => {
			window.localStorage.setItem('founderos-os:deliverables-opened', JSON.stringify({ a: 'r1', b: 'r1' }));
			withFiles();
			const { findByRole, container } = render(AgentsPage);
			await fireEvent.click(await findByRole('tab', { name: /Deliverables/ }));
			await waitFor(() => expect(container.querySelector('[data-part="deliverables"]')?.textContent).toContain('Doc a'));
			const rowOf = (t: string) => [...container.querySelectorAll<HTMLElement>('[data-part="feed-row"]')].find((r) => r.textContent?.includes(t)) as HTMLElement;
			expect(rowOf('Doc a').querySelector('[aria-label="updated"]')?.getAttribute('title')).toBe('Rewritten since you read it');
			expect(rowOf('Doc c').querySelector('[aria-label="new"]')?.getAttribute('title')).toBe('New since you last looked');
			expect(rowOf('Doc b').querySelector('[aria-label="new"], [aria-label="updated"]')).toBeNull();
			await fireEvent.click(within(rowOf('Doc a')).getByRole('button', { name: 'review' }));
			await waitFor(() => expect(rowOf('Doc a').querySelector('[aria-label="updated"]')).toBeNull());
			expect(rowOf('Doc c').querySelector('[aria-label="new"]')).toBeTruthy();
			expect(JSON.parse(window.localStorage.getItem('founderos-os:deliverables-opened') ?? '{}').a).toBe('r2');
		});
	});

	it('the Conductor rail recovers from one failed poll instead of going dark', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		try {
			let fail = true;
			route({
				'GET /pages/conductor/chat': () => {
					if (fail) throw new Error('board hiccup');
					return { messages: [{ id: 'x1', body: 'back online', authorType: 'agent', createdAt: '2026-09-30T08:00:00Z' }] };
				}
			});
			const { findByText, findByPlaceholderText, queryByText } = render(AgentsPage);
			// prod's ConductorChat reads a failed poll as "nothing new": no error line
			expect(await findByPlaceholderText('Message the CEO…')).toBeTruthy();
			await vi.advanceTimersByTimeAsync(100);
			expect(queryByText(/unreadable|board hiccup/)).toBeNull();
			fail = false;
			await vi.advanceTimersByTimeAsync(13_000);
			expect(await findByText('back online')).toBeTruthy();
			expect(queryByText(/Cockpit thread unreadable/)).toBeNull();
		} finally {
			vi.useRealTimers();
		}
	});

	it('the Conductor rail is the CEO chat: emblem head, model pill, composer', async () => {
		route();
		const { findByText, getByText, getByPlaceholderText } = render(AgentsPage);
		expect(await findByText('CONDUCTOR')).toBeTruthy();
		expect(getByText('board · claude-fable-5')).toBeTruthy();
		expect(getByText('the real CEO on the company board · delegates, creates tasks, reads your data')).toBeTruthy();
		expect(getByPlaceholderText('Message the CEO…')).toBeTruthy();
	});

	it('a message to the CEO is held by the bridge guard and never shown as sent', async () => {
		route({
			'POST /pages/conductor/chat': () => {
				throw new FounderosApiError(409, 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)', { guarded: true });
			}
		});
		const { findByPlaceholderText, findByText, queryByText } = render(AgentsPage);
		const box = await findByPlaceholderText('Message the CEO…');
		await fireEvent.input(box, { target: { value: 'Ship the reel' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/chat', { method: 'POST', json: { message: 'Ship the reel' } }));
		expect(await findByText(/Message did not reach the board: writes are off/)).toBeTruthy();
		expect(queryByText('Ship the reel')).toBeNull();
	});

	it('a failed first load is an error, not an empty page', async () => {
		route({
			'GET /pages/agents/view': () => {
				throw new FounderosApiError(500, 'boom');
			}
		});
		const { findByText } = render(AgentsPage);
		expect(await findByText(/Could not load the board: boom/)).toBeTruthy();
	});
});
