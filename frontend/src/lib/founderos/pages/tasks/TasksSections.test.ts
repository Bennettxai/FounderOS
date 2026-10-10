import { fireEvent, render, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// The real module (FounderosApiError, isGuardRefusal) with the fetch stubbed.
vi.mock('$lib/founderos/api', async (orig) => ({ ...(await orig<typeof import('$lib/founderos/api')>()), founderosFetch: vi.fn() }));

import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
import TasksSections from './TasksSections.svelte';
import type { BoardIssue, TasksView } from './types';

const fetchMock = vi.mocked(founderosFetch);

function view(over: Partial<TasksView> = {}): TasksView {
	const series = Array.from({ length: 14 }, (_, i) => ({ label: `Sep ${17 + i}`, count: i === 13 ? 3 : 0 }));
	return {
		tasks: [
			{ id: 't1', agentId: 'crm-pulse', title: 'Clean the CRM', status: 'open', createdAt: '', updatedAt: '' },
			{ id: 't2', agentId: 'sales-agent', title: 'Follow up leads', status: 'done', createdAt: '', updatedAt: '' }
		],
		agentNames: { 'crm-pulse': 'CRM Pulse', 'sales-agent': 'Sales Agent' },
		issues: [{ id: 'i1', identifier: 'FOS-7', title: 'Ship the reel', status: 'blocked', assigneeName: 'Forge', updatedAt: null }],
		board: { connected: true, url: 'http://board:3100' },
		crons: [],
		cronStats: {},
		jobs: [
			{ id: 'c1', description: 'Morning CRM pulse', agentId: 'crm-pulse', agentName: 'CRM Pulse', unknownAgent: false, schedule: '0 9 * * *', scheduleLabel: 'at 09:00, daily', enabled: true, runs: 2, ok: 1, lastRunAt: null, lastOk: false, nextRunAt: null, overdue: false, history: [], lastSummary: 'boom' },
			{ id: 'c2', description: 'Ghost poll', agentId: 'ghost', agentName: 'ghost', unknownAgent: true, schedule: '*/30 * * * *', scheduleLabel: 'every 30 min', enabled: false, runs: 0, ok: 0, lastRunAt: null, lastOk: null, nextRunAt: null, overdue: false, history: [], lastSummary: null }
		],
		volume: {
			headline: 3,
			counts: { open: 1, doing: 0, review: 0, done: 1 },
			board: { total: 1, inProgress: 0, blocked: 1, done: 0 },
			boardOnline: true,
			chips: [{ text: '1 to do' }, { tone: 'ok', text: '1 done' }],
			caption: '2 on the local kanban · 1 on the board',
			meters: [{ label: 'Kanban shipped (1/2)', frac: 0.5, display: '50%', hue: 'var(--bn-ok)' }],
			foot: '2 crons · 2 runs recorded · 2 agents on the kanban',
			series,
			touchesInWindow: 3,
			cron: { runsInWindow: 2, failedInWindow: 1, rhythm: [{ label: 'Mon', count: 2 }] },
			owners: [{ label: 'CRM', count: 1 }],
			busiestOwner: { name: 'CRM Pulse', count: 1 },
			insight: { value: 2, headline: '1 blocked · 1 cron late or failing.', body: 'Morning CRM pulse', frac: 0.5 }
		},
		...over
	};
}

const ISSUES: BoardIssue[] = [
	{ id: 'i1', identifier: 'FOS-7', title: 'Ship the reel', status: 'blocked', assigneeName: 'Forge', updatedAt: null },
	{ id: 'i2', identifier: 'FOS-3', title: 'Old finished work', status: 'done', assigneeName: null, updatedAt: null }
];
const props = (over: Partial<{ view: TasksView; issues: BoardIssue[]; boardUrl: string | null }> = {}) => ({
	view: view(),
	issues: ISSUES,
	boardUrl: 'http://board:3100',
	...over
});

// braces: a function returned from beforeEach is run as its teardown
beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.useRealTimers());

describe('Tasks tab sections (FounderOS v1 TaskCronStrip + BoardTasks + TaskBoard)', () => {
	it('is the cron strip, then the open board queue, then the local kanban, and no hero', () => {
		const { container, getByText, queryByText } = render(TasksSections, props());
		const heads = [...container.querySelectorAll('h2')].map((h) => h.textContent?.trim());
		expect(heads).toEqual(['Scheduled jobs', 'Board queue', 'Local kanban']);
		expect(getByText('full panel →').closest('a')?.getAttribute('href')).toBe('/os/workflows');
		expect(getByText(/open board/).closest('a')?.getAttribute('href')).toBe('http://board:3100');
		expect(queryByText('Task Volume')).toBeNull();
		expect(container.querySelector('h1')).toBeNull();
	});

	it('the board queue is the OPEN issues only: done work lives in Deliverables', () => {
		const { getByText, queryByText } = render(TasksSections, props());
		expect(getByText('1 open issues')).toBeTruthy();
		expect(getByText('Ship the reel')).toBeTruthy();
		expect(queryByText('Old finished work')).toBeNull();
	});

	it('the cron strip flags failing and missing-agent jobs', () => {
		const { getByText, container } = render(TasksSections, props());
		expect(getByText('1 of 2 enabled')).toBeTruthy();
		expect(container.querySelector('[data-job="c1"]')?.textContent).toContain('Morning CRM pulse');
		expect(container.querySelector('[data-job="c1"]')?.textContent).toContain('1/2');
		expect(getByText('last failed · never')).toBeTruthy();
		expect(getByText('agent missing from the runtime')).toBeTruthy();
	});

	it('an unreachable or empty board says so in production copy', () => {
		const { getByText } = render(TasksSections, props({ issues: [], boardUrl: null }));
		expect(getByText('0 open issues')).toBeTruthy();
		expect(getByText('Board unreachable or empty · the live queue shows here.')).toBeTruthy();
	});

	it('the composer shows the guard refusal instead of pretending it created the task', async () => {
		fetchMock.mockRejectedValueOnce(new FounderosApiError(403, 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)', { guarded: true }));
		const { getByPlaceholderText, getByText } = render(TasksSections, props());
		await fireEvent.input(getByPlaceholderText(/Hand the company work/), { target: { value: 'Ship it' } });
		await fireEvent.click(getByText('TECH'));
		await fireEvent.click(getByText('Create issue'));
		await waitFor(() => expect(getByText(/Writes are off on the bridge/)).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/board/tasks', { method: 'POST', json: { title: 'Ship it', description: 'Route to the TECH pillar.' } });
		expect(getByText('Ship the reel')).toBeTruthy();
	});

	it('a created issue lands at the top of the queue', async () => {
		fetchMock.mockResolvedValueOnce({ issue: { id: 'n', identifier: 'FOS-9', title: 'Ship it', status: 'todo', assigneeName: null, updatedAt: null } });
		const { getByPlaceholderText, getByText } = render(TasksSections, props());
		await fireEvent.input(getByPlaceholderText(/Hand the company work/), { target: { value: 'Ship it' } });
		await fireEvent.click(getByText('Create issue'));
		await waitFor(() => expect(getByText('FOS-9')).toBeTruthy());
	});

	it('a kanban move the server refused snaps back at once and says why', async () => {
		fetchMock.mockImplementation(async (_p: string, init?: { method?: string }) => {
			if (init?.method === 'PATCH') throw new FounderosApiError(502, 'Bad Gateway', { error: 'task store unreachable' });
			return { ok: true } as never;
		});
		const { getByTitle, container, findByRole } = render(TasksSections, props());
		await fireEvent.click(getByTitle('Advance to In progress'));
		expect((await findByRole('alert')).textContent).toContain('task store unreachable');
		expect(container.querySelector('[data-lane="open"]')!.textContent).toContain('Clean the CRM');
		expect(container.querySelector('[data-lane="doing"]')!.textContent).not.toContain('Clean the CRM');
	});

	it('a poll that started before a move cannot undo it', async () => {
		vi.useFakeTimers({ shouldAdvanceTime: true });
		try {
			let releasePoll!: () => void;
			const before = view().tasks;
			fetchMock.mockImplementation(async (_p: string, init?: { method?: string }) => {
				if (init?.method === 'PATCH') return { ok: true } as never;
				return new Promise((r) => (releasePoll = () => r({ tasks: before }))) as never;
			});
			const { getByTitle, container } = render(TasksSections, props());
			await vi.advanceTimersByTimeAsync(6000); // the poll is now in flight with the old board
			await fireEvent.click(getByTitle('Advance to In progress'));
			releasePoll();
			await vi.advanceTimersByTimeAsync(0);
			expect(container.querySelector('[data-lane="doing"]')!.textContent).toContain('Clean the CRM');
		} finally {
			vi.useRealTimers();
		}
	});

	it('advancing a kanban card PATCHes the agent work route', async () => {
		fetchMock.mockResolvedValue({ ok: true });
		const { getByTitle, container } = render(TasksSections, props());
		await fireEvent.click(getByTitle('Advance to In progress'));
		expect(fetchMock).toHaveBeenCalledWith('/pages/agents/work', { method: 'PATCH', json: { kind: 'task', id: 't1', status: 'doing' } });
		const doing = container.querySelector('[data-lane="doing"]')!;
		expect(doing.textContent).toContain('Clean the CRM');
	});
});
