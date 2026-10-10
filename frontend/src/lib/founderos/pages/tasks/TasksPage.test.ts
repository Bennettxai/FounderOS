import { render } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', async (orig) => ({ ...(await orig<typeof import('$lib/founderos/api')>()), founderosFetch: vi.fn() }));

import { founderosFetch } from '$lib/founderos/api';
import TasksPage from './TasksPage.svelte';
import type { TasksView } from './types';

const fetchMock = vi.mocked(founderosFetch);

function view(over: Partial<TasksView> = {}): TasksView {
	const series = Array.from({ length: 14 }, (_, i) => ({ label: `Sep ${17 + i}`, count: i === 13 ? 3 : 0 }));
	return {
		tasks: [
			{ id: 't1', agentId: 'crm-pulse', title: 'Clean the CRM', status: 'open', createdAt: '', updatedAt: '' },
			{ id: 't2', agentId: 'sales-agent', title: 'Follow up leads', status: 'done', createdAt: '', updatedAt: '' }
		],
		agentNames: { 'crm-pulse': 'CRM Pulse', 'sales-agent': 'Sales Agent' },
		issues: [
			{ id: 'i1', identifier: 'FOS-7', title: 'Ship the reel', status: 'blocked', assigneeName: 'Forge', updatedAt: null },
			{ id: 'i2', identifier: 'FOS-3', title: 'Old finished work', status: 'done', assigneeName: null, updatedAt: null }
		],
		board: { connected: true, url: 'http://board:3100' },
		crons: [],
		cronStats: {},
		jobs: [
			{ id: 'c1', description: 'Morning CRM pulse', agentId: 'crm-pulse', agentName: 'CRM Pulse', unknownAgent: false, schedule: '0 9 * * *', scheduleLabel: 'at 09:00, daily', enabled: true, runs: 2, ok: 1, lastRunAt: null, lastOk: false, nextRunAt: null, overdue: false, history: [], lastSummary: 'boom' }
		],
		volume: {
			headline: 4,
			counts: { open: 1, doing: 0, review: 0, done: 1 },
			board: { total: 2, inProgress: 0, blocked: 1, done: 1 },
			boardOnline: true,
			chips: [{ text: '1 to do' }, { tone: 'ok', text: '1 done' }],
			caption: '2 on the local kanban · 2 on the board',
			meters: [{ label: 'Kanban shipped (1/2)', frac: 0.5, display: '50%', hue: 'var(--bn-ok)' }],
			foot: '1 cron · 2 runs recorded · 2 agents on the kanban',
			series,
			touchesInWindow: 3,
			cron: { runsInWindow: 2, failedInWindow: 1, rhythm: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((label) => ({ label, count: label === 'Tue' ? 2 : 0 })) },
			owners: [{ label: 'CRM', count: 1 }],
			busiestOwner: { name: 'CRM Pulse', count: 1 },
			insight: { value: 2, headline: '1 blocked · 1 cron late or failing.', body: 'Morning CRM pulse', frac: 0.5 }
		},
		...over
	};
}

beforeEach(() => {
	fetchMock.mockReset();
	fetchMock.mockResolvedValue({ tasks: view().tasks } as never);
});

describe('/os/tasks page (FounderOS v1 app/tasks/page.tsx)', () => {
	it('carries the slab title with the moved / cron / blocked meta and the board chip + pills', () => {
		const { getByText } = render(TasksPage, { view: view() });
		expect(getByText('Tasks', { selector: 'h1' })).toBeTruthy();
		expect(getByText('3 items moved in 14 days · 2 cron runs · 1 blocked')).toBeTruthy();
		expect(getByText('board live · paperclip')).toBeTruthy();
		expect(getByText('Workflows').closest('a')?.getAttribute('href')).toBe('/os/workflows');
		expect(getByText('Agents').closest('a')?.getAttribute('href')).toBe('/os/agents');
	});

	it('an offline board reads "board offline · paperclip"', () => {
		const { getByText } = render(TasksPage, { view: view({ board: { connected: false, url: null } }) });
		expect(getByText('board offline · paperclip')).toBeTruthy();
	});

	it('runs the v1 section order: hero, second row, then the three queues', () => {
		const { container } = render(TasksPage, { view: view() });
		const heads = [...container.querySelectorAll('h2')].map((h) => h.textContent?.replace(/\s+/g, ' ').trim());
		expect(heads.map((h) => h?.split(' ')[0])).toEqual(['Task', 'Task', 'Cron', 'Owners', 'Scheduled', 'Board', 'Local']);
		expect(container.textContent).toContain('Task Activity');
		expect(container.textContent).toContain('Task Volume');
		expect(container.textContent).toContain('Needs you');
	});

	it('hero numbers and captions come from the volume payload', () => {
		const { getByText } = render(TasksPage, { view: view() });
		expect(getByText('kanban tasks and board issues, on the day each last moved')).toBeTruthy();
		expect(getByText('2 on the local kanban · 2 on the board')).toBeTruthy();
		expect(getByText('Kanban shipped (1/2)')).toBeTruthy();
		expect(getByText('busiest Tue · 2 runs')).toBeTruthy();
		expect(getByText('most on CRM Pulse')).toBeTruthy();
		expect(getByText('1 blocked · 1 cron late or failing.')).toBeTruthy();
	});

	it('quiet windows read v1 copy', () => {
		const v = view();
		v.volume = { ...v.volume, cron: { runsInWindow: 0, failedInWindow: 0, rhythm: v.volume.cron.rhythm.map((d) => ({ ...d, count: 0 })) }, owners: [], busiestOwner: null };
		const { getByText } = render(TasksPage, { view: v });
		expect(getByText('no runs in 14 days')).toBeTruthy();
		expect(getByText('nothing open on the kanban')).toBeTruthy();
	});

	it('the board queue lists every board issue the page read, as v1 BoardTasks does', () => {
		const { getByText } = render(TasksPage, { view: view() });
		expect(getByText('2 open issues')).toBeTruthy();
		expect(getByText('Ship the reel')).toBeTruthy();
		expect(getByText('Old finished work')).toBeTruthy();
	});
});
