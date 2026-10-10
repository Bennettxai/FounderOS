import { fireEvent, render, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$lib/founderos/api', () => ({ founderosFetch: vi.fn(), FounderosApiError: class extends Error {} }));

import { founderosFetch } from '$lib/founderos/api';
import WorkflowsPage from './WorkflowsPage.svelte';
import type { JobRow, Workflow, WorkflowsView } from './types';

const fetchMock = vi.mocked(founderosFetch);

const job = (o: Partial<JobRow>): JobRow => ({
	id: 'c1', description: 'Morning CRM pulse', agentId: 'crm-pulse', agentName: 'CRM Pulse', unknownAgent: false, schedule: '0 9 * * *',
	scheduleLabel: 'at 09:00, daily', enabled: true, runs: 2, ok: 1, lastRunAt: '2026-09-29T09:00:00Z', lastOk: false, nextRunAt: null,
	overdue: false, history: [true, false], lastSummary: 'boom', ...o
});

const wf: Workflow = {
	id: 'wf-vantage-sales', name: 'Vantage sales machine', subtitle: '', revenueUsd: 0, order: 0,
	steps: [
		{ id: 's1', title: 'Lead in', detail: 'Typeform lead lands', ownerKind: 'agent', owner: 'Sales Agent', hoursPerWeek: 2, tools: ['typeform'], edgeLabel: null, leakUsd: null, automation: { title: 'auto-tag', state: 'live', recoveredUsd: 300 }, branch: null },
		{ id: 's2', title: 'Call', detail: 'Alex calls', ownerKind: 'human', owner: 'Alex', hoursPerWeek: 5, tools: ['zoom'], edgeLabel: null, leakUsd: 1200, automation: null, branch: null },
		{ id: 's3', title: 'Lost path', detail: 'Nurture', ownerKind: 'agent', owner: 'Sales Agent', hoursPerWeek: 1, tools: [], edgeLabel: null, leakUsd: null, automation: null, branch: { from: 's1', condition: 'went quiet' } }
	]
};

function view(over: Partial<WorkflowsView> = {}): WorkflowsView {
	return {
		workflows: [wf],
		jobs: [
			job({ id: 'late', description: 'Late heartbeat', overdue: true, lastOk: true, history: [] }),
			job({ id: 'ghost', description: 'Ghost poll', agentId: 'ghost', agentName: 'ghost', unknownAgent: true, enabled: true })
		],
		agents: [{ id: 'sales-agent', name: 'Sales Agent' }, { id: 'crm-pulse', name: 'CRM Pulse' }],
		agentPresence: { 'Sales Agent': 'active', 'CRM Pulse': 'inactive' },
		runsByOwner: { 'Sales Agent': [{ id: 'r1', agentId: 'sales-agent', startedAt: '2026-09-30T00:00:00Z', finishedAt: '2026-09-30T00:00:00Z', ok: false, summary: 'failed to send' }], Alex: [] },
		toolIds: ['typeform', 'zoom'],
		volume: {
			headline: 2, counts: { healthy: 0, overdue: 1, failing: 1, paused: 0, enabled: 2 },
			chips: [{ tone: 'warn', text: '1 overdue' }, { tone: 'err', text: '1 failing' }], caption: '2 enabled · 4 runs recorded',
			meters: [{ label: 'Crons healthy (0/2)', frac: 0, display: '0%', hue: 'var(--bn-warn)' }], foot: '1 workflows · 3 steps · 2 tools',
			series: Array.from({ length: 14 }, (_, i) => ({ label: `d${i}`, count: i === 13 ? 2 : 0 })), runsInWindow: 2, failedInWindow: 1,
			rhythm: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map((label) => ({ label, count: label === 'Tue' ? 2 : 0 })),
			load: { manualHours: 5, agentHours: 3, perWorkflow: [{ label: 'Vantage', count: 5 }] },
			insight: { value: 2, headline: '1 overdue · 1 failing.', body: 'Late heartbeat · Ghost poll', frac: 1 }
		},
		...over
	};
}

// braces: a function returned from beforeEach is run as its teardown
beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.restoreAllMocks());

describe('WorkflowsPage slab', () => {
	it('renders the title, volume cards and the one insight from the payload', () => {
		const { getByText, getAllByText } = render(WorkflowsPage, { view: view() });
		expect(getByText('Workflows')).toBeTruthy();
		expect(getByText('Cron Volume')).toBeTruthy();
		expect(getByText('2 enabled · 4 runs recorded')).toBeTruthy();
		expect(getByText('Crons healthy (0/2)')).toBeTruthy();
		expect(getByText('1 overdue · 1 failing.')).toBeTruthy();
		expect(getByText('busiest day · 2 runs')).toBeTruthy();
		expect(getAllByText(/by hand · 3h carried by agents/).length).toBe(1);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('the meta line counts workflows, cron runs and failures in the window (v1)', () => {
		const { getByText } = render(WorkflowsPage, { view: view() });
		expect(getByText('1 workflows · 2 cron runs in 14 days · 1 failed')).toBeTruthy();
	});

	it('Tasks links to /os/tasks and Agents to /os/agents (v1 /tasks, /agents)', () => {
		const { getByText } = render(WorkflowsPage, { view: view() });
		expect(getByText('Tasks').closest('a')!.getAttribute('href')).toBe('/os/tasks');
		expect(getByText('Agents').closest('a')!.getAttribute('href')).toBe('/os/agents');
	});

	it('Run Activity: the run total under the title with its caption, then the step line', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const hero = container.querySelector('[data-part="hero-row"]')!;
		const card = hero.children[0] as HTMLElement;
		expect(card.textContent).toContain('Run Activity');
		expect(card.textContent).toContain('last 14 days');
		expect(card.textContent).not.toContain('real cron_runs ·');
		expect(card.textContent).toContain('real cron runs, straight off cron_runs');
		expect(card.textContent).toContain('1 failed');
		expect(card.querySelector('[data-part="line"]')).toBeTruthy();
		// the folded strip of the private build is gone
		expect(container.querySelector('[data-part="run-fold"]')).toBeNull();
	});

	it('second row: Run Rhythm, Process Load and Needs you as three cards (v1)', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const row = container.querySelector('[data-part="second-row"]')!;
		const blocks = Array.from(row.children).map((c) => c.textContent ?? '');
		expect(blocks).toHaveLength(3);
		expect(blocks[0]).toContain('Run Rhythm');
		expect(blocks[0]).toContain('by weekday');
		expect(blocks[0]).toContain('Tue');
		expect(blocks[1]).toContain('Process Load');
		expect(blocks[1]).toContain('hours per week');
		expect(blocks[2]).toContain('Needs you');
		expect(blocks[2]).toContain('Late heartbeat · Ghost poll');
	});

	it('hero row is two cards, then the second row, then scheduled tasks, then the process map', () => {
		const { container, getByText } = render(WorkflowsPage, { view: view() });
		const hero = container.querySelector('[data-part="hero-row"]')!;
		expect(hero.children).toHaveLength(2);
		expect(hero.children[1].textContent).toContain('Cron Volume');
		const second = container.querySelector('[data-part="second-row"]')!;
		const scheduled = container.querySelector('[data-part="scheduled"]')!;
		expect(hero.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
		expect(second.compareDocumentPosition(scheduled) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
		expect(scheduled.compareDocumentPosition(getByText('Process map')) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});
});

describe('WorkflowsPage uses the kit as v1 does', () => {
	it('Tasks and Agents are the rounded slab pills', () => {
		const { getByText } = render(WorkflowsPage, { view: view() });
		for (const t of ['Tasks', 'Agents']) expect(getByText(t).closest('a')!.className).toContain('rounded-full');
	});

	it('Needs you is the full-size kit InsightCard, its body unclamped', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const row = container.querySelector('[data-part="second-row"]')!;
		const inner = row.querySelector('[data-part="insight-inner"]')!;
		expect(inner.className).toContain('px-6 py-5');
		expect(inner.parentElement!.className).not.toContain('line-clamp-2');
	});

	it('Cron Volume carries the standard meter rhythm', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const hero = container.querySelector('[data-part="hero-row"]')!;
		expect(hero.children[1].querySelector('[data-part="meters"]')!.className).toContain('gap-6');
	});
});

describe('ScheduledTasks', () => {
	it('flags overdue rows and crons naming an agent the runtime does not have', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		expect(container.querySelector('[data-job="late"]')!.textContent).toContain('overdue');
		expect(container.querySelector('[data-job="ghost"]')!.textContent).toContain('no such agent');
		expect(container.querySelector('[data-part="scheduled"]')!.textContent).toContain('1 overdue');
	});

	it('run now posts the cron to /pages/cron/run and shows what ran', async () => {
		const onchange = vi.fn();
		fetchMock.mockResolvedValueOnce({ ok: true, summary: 'pulsed 3 deals' });
		const { container, getByText } = render(WorkflowsPage, { view: view(), onchange });
		await fireEvent.click(container.querySelector('[data-job="late"] button')!);
		expect(fetchMock).toHaveBeenCalledWith('/pages/cron/run', { method: 'POST', json: { cronId: 'late' } });
		await waitFor(() => expect(getByText('ran: pulsed 3 deals')).toBeTruthy());
		expect(onchange).toHaveBeenCalled();
	});

	it('pause and delete go through /pages/agents/work; errors show inline', async () => {
		fetchMock.mockResolvedValueOnce({ ok: true }).mockRejectedValueOnce(new Error('cron not found'));
		const { container, getByText } = render(WorkflowsPage, { view: view() });
		await fireEvent.click(container.querySelector('[data-job="late"] button[title="Pause"]')!);
		expect(fetchMock).toHaveBeenCalledWith('/pages/agents/work', { method: 'PATCH', json: { kind: 'cron', id: 'late', enabled: false } });
		await fireEvent.click(container.querySelector('[data-job="late"] button[title="Delete"]')!);
		expect(fetchMock).toHaveBeenLastCalledWith('/pages/agents/work', { method: 'DELETE', json: { kind: 'cron', id: 'late' } });
		await waitFor(() => expect(getByText(/cron not found/)).toBeTruthy());
	});

	it('row controls read like prod: run now, on, and a trash icon for delete', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const row = container.querySelector('[data-job="late"]')!;
		expect(row.textContent).toContain('▸ run now');
		const del = row.querySelector('button[title="Delete"]')!;
		expect(del.querySelector('svg.lucide-trash-2, svg.lucide-trash2')).toBeTruthy();
		expect(del.textContent!.trim()).toBe('');
	});

	it('header and row controls carry prod\'s 5px chip radius; the list is an 8px card', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const btn = container.querySelector('[data-part="new-task"]')!;
		expect(btn.className).toContain('rounded-[5px]');
		expect(btn.className).toContain('py-1');
		expect(container.querySelector('[data-part="job-list"]')!.className).toContain('rounded-lg');
		expect(container.querySelector('[data-job="late"] button')!.className).toContain('rounded-[5px]');
	});

	it('new task refuses an empty description before calling anything', async () => {
		const { getByText, container } = render(WorkflowsPage, { view: view() });
		await fireEvent.click(getByText('new task'));
		await fireEvent.click(getByText('add'));
		expect(container.querySelector('[data-part="error"]')!.textContent).toContain('pick an agent and describe the task');
		expect(fetchMock).not.toHaveBeenCalled();
	});
});

describe('Process map', () => {
	it('a card expands into the real tree with branches and a step drawer', async () => {
		const { container, getByText, getByLabelText } = render(WorkflowsPage, { view: view() });
		expect(container.querySelector('[data-part="expanded"]')).toBeNull();
		await fireEvent.click(container.querySelector('[data-workflow-card]')!);
		expect(container.querySelector('[data-part="expanded"]')).toBeTruthy();
		expect(getByText('if went quiet')).toBeTruthy();
		await fireEvent.click(getByLabelText('Lead in: view step detail'));
		const drawer = container.querySelector('[data-part="step-detail"]')!;
		expect(drawer.textContent).toContain('failed to send');
		expect(drawer.textContent).toContain('live');
	});

	it('tree nodes and the drawer read like prod: round lucide glyphs, a bot avatar by the owner, logo tool chips', async () => {
		const { container, getByLabelText } = render(WorkflowsPage, { view: view() });
		await fireEvent.click(container.querySelector('[data-workflow-card]')!);
		const node = getByLabelText('Lead in: view step detail');
		expect(node.className).toContain('wft-node');
		expect(node.querySelector('svg.lucide-zap')).toBeTruthy();
		expect(node.querySelector('[data-part="avatar"] svg.lucide-bot')).toBeTruthy();
		expect(node.querySelector('[data-mark]')).toBeTruthy();
		expect(node.textContent).toContain('Typeform');
		expect(getByLabelText('Call: view step detail').querySelector('svg.lucide-user')).toBeTruthy();
		await fireEvent.click(node);
		const drawer = container.querySelector('[data-part="step-detail"]')!;
		expect(drawer.querySelector('[data-part="avatar"]')).toBeTruthy();
		expect(drawer.querySelector('[aria-label="Close step detail"] svg.lucide-x')).toBeTruthy();
	});

	it('a click outside the open card collapses it (prod captures the click)', async () => {
		const { container } = render(WorkflowsPage, { view: view() });
		await fireEvent.click(container.querySelector('[data-workflow-card]')!);
		expect(container.querySelector('[data-part="expanded"]')).toBeTruthy();
		await fireEvent.click(container.querySelector('[data-part="hero-row"]')!);
		expect(container.querySelector('[data-part="expanded"]')).toBeNull();
	});

	it('collapsed cards read like prod: a pressable row with no frame, round step dots, brand-logo tool chips, edit on hover', () => {
		const { container } = render(WorkflowsPage, { view: view() });
		const card = container.querySelector('[data-workflow-card]') as HTMLElement;
		expect(card.className).toContain('bn-pressable');
		expect(card.className).toContain('is-row');
		expect(card.className).not.toMatch(/(^|\s)border(\s|$)/);
		const dots = card.querySelectorAll('[data-part="mini-dot"]');
		expect(dots).toHaveLength(3);
		for (const d of dots) expect(d.className).toContain('rounded-full');
		const chips = card.querySelectorAll('[data-part="tool-chip"]');
		expect(Array.from(chips).map((c) => c.getAttribute('title'))).toEqual(['Typeform', 'zoom']);
		for (const c of chips) expect(c.querySelector('[data-mark]')).toBeTruthy();
		expect((card.querySelector('[aria-label="Edit Vantage sales machine"]') as HTMLElement).className).toContain('opacity-0');
	});

	it('New workflow is prod\'s bare button: a plus icon over the words, no frame or fill', () => {
		const { getByText } = render(WorkflowsPage, { view: view() });
		const btn = getByText('New workflow').closest('button')!;
		expect(btn.querySelector('svg.lucide-plus')).toBeTruthy();
		expect(btn.getAttribute('style') ?? '').not.toContain('background');
		expect(btn.className).not.toMatch(/(^|\s)border(\s|$)/);
	});

	it('the builder validates before saving and only drafts on an explicit click', async () => {
		const { getByText, container, getByPlaceholderText } = render(WorkflowsPage, { view: view() });
		await fireEvent.click(getByText('New workflow'));
		expect(fetchMock).not.toHaveBeenCalled();
		await fireEvent.click(getByText('Save'));
		expect(container.querySelector('[data-part="builder-error"]')!.textContent).toContain('Give the workflow a name.');
		expect(fetchMock).not.toHaveBeenCalled();

		fetchMock.mockResolvedValueOnce({ ok: false, unavailable: true, error: 'drafting assistant unavailable: build manually' });
		await fireEvent.input(getByPlaceholderText("Describe the workflow and I'll draft it…"), { target: { value: 'weekly review' } });
		await fireEvent.click(getByText('✦ Draft'));
		expect(fetchMock).toHaveBeenCalledWith('/pages/workflows/draft', { method: 'POST', json: { prompt: 'weekly review' } });
		await waitFor(() => expect(container.querySelector('[data-part="draft-error"]')!.textContent).toContain('Drafting assistant unavailable: build manually.'));
	});

	it('a draft fills the form, then Save posts the builder payload', async () => {
		const onchange = vi.fn();
		fetchMock.mockResolvedValueOnce({
			ok: true,
			draft: { name: 'Weekly review', subtitle: 'Fridays', steps: [{ title: 'Pull', detail: 'Numbers', ownerKind: 'agent', owner: 'crm pulse', hoursPerWeek: 1, tools: ['stripe'], automation: null, branchFromIndex: null, branchCondition: null }] }
		});
		const { getByText, getByPlaceholderText, getByDisplayValue } = render(WorkflowsPage, { view: view(), onchange });
		await fireEvent.click(getByText('New workflow'));
		await fireEvent.input(getByPlaceholderText("Describe the workflow and I'll draft it…"), { target: { value: 'review' } });
		await fireEvent.click(getByText('✦ Draft'));
		await waitFor(() => expect(getByDisplayValue('Weekly review')).toBeTruthy());
		fetchMock.mockResolvedValueOnce({ workflow: {} });
		await fireEvent.click(getByText('Save'));
		await waitFor(() => expect(onchange).toHaveBeenCalled());
		const [path, init] = fetchMock.mock.calls.at(-1)!;
		expect(path).toBe('/pages/workflows');
		expect((init as { json: { steps: { owner: string }[] } }).json.steps[0].owner).toBe('CRM Pulse');
	});
});
