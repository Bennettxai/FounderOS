import { fireEvent, render, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// The (founderos) layout load hands every /os page the signed-in session.
vi.mock('$app/state', () => ({
	page: { data: { user: { name: 'Alex Rivera', email: 'alex@founderos.local' } }, url: new URL('http://localhost/os/org'), params: {} }
}));

vi.mock('$lib/founderos/api', () => {
	class FounderosApiError extends Error {
		status: number;
		constructor(status: number, message: string) {
			super(`${message} (HTTP ${status})`);
			this.status = status;
		}
	}
	return { founderosFetch: vi.fn(), FounderosApiError };
});

import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
import OrgBoard from './OrgBoard.svelte';
import type { OrgView } from './types';

const fetchMock = vi.mocked(founderosFetch);

const agent = (id: string, departmentId: string, tier: string, extra: Record<string, unknown> = {}) => ({
	id,
	departmentId,
	name: id.replace(/-/g, ' '),
	role: `${id} role`,
	status: 'active',
	tier,
	description: 'd',
	model: 'm',
	tools: [] as string[],
	parentId: null,
	instance: 'builtin',
	...extra
});

function view(over: Partial<OrgView> = {}): OrgView {
	const sales = { id: 'dept-sales', name: 'Sales', slug: 'sales', tagline: '', color: '#ffd166', order: 1 };
	const clients = { id: 'dept-clients', name: 'Clients', slug: 'clients', tagline: '', color: '#14b8a6', order: 2 };
	const lead = agent('sales-agent', 'dept-sales', 'lead', { tools: ['typeform'], instance: 'openclaw' });
	const worker = agent('crm-pulse', 'dept-sales', 'worker', { parentId: 'sales-agent' });
	const wa = agent('whatsapp-worker', 'dept-sales', 'worker');
	return {
		departments: [sales, clients],
		agents: [lead, worker, wa],
		agentNames: { 'sales-agent': 'Sales Agent', 'crm-pulse': 'CRM Pulse', 'whatsapp-worker': 'WA' },
		conductor: agent('conductor', 'dept-tech', 'lead', { tools: ['slack'], instance: 'claude-code' }),
		tree: { totalAgents: 3, activeAgents: 3 },
		crews: [
			{
				department: sales,
				area: { id: 'sales', label: 'Sales', color: '#ef4444', detail: '', agents: [], departmentIds: ['dept-sales'] },
				live: { id: 'b2', name: 'Sales', status: 'idle', adapterType: null, model: 'glm-5', lastHeartbeatAt: null },
				leads: [lead],
				pills: [
					{ agent: worker, children: [] },
					{ agent: wa, children: [] }
				],
				tools: ['typeform'],
				empty: false
			},
			{ department: clients, area: null, live: null, leads: [], pills: [], tools: [], empty: true }
		],
		live: {
			connected: true,
			conductor: { id: 'b1', name: 'Conductor', status: 'running', adapterType: null, model: 'claude-fable-5', lastHeartbeatAt: null },
			byDepartment: {},
			extras: [{ id: 'b3', name: 'Forge', status: 'error', adapterType: null, model: null, lastHeartbeatAt: null }]
		},
		ventures: [
			{ id: 'vantage', label: 'Vantage', kind: 'AI agency', color: '#00ffaa', detail: 'agency arm', brainTag: 'vantage', focus: ['Ship builds'], areaAgents: {}, agentIds: ['conductor', 'sales-agent', 'crm-pulse'] }
		],
		lifeAreas: [{ id: 'sales', label: 'Sales', color: '#ef4444', detail: '', agents: [], departmentIds: [] }],
		lastBroadcast: null,
		...over
	};
}

// braces: a function returned from beforeEach is run as its teardown
beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.restoreAllMocks());

describe('OrgBoard', () => {
	it('hangs each department crew off the rail with a short connector tick, as prod .org-connector', async () => {
		const { container } = render(OrgBoard, { view: view(), ventureId: null });
		const crews = [...container.querySelectorAll('[data-part="crew"]')];
		expect(crews.length).toBeGreaterThan(0);
		for (const c of crews) expect(c.classList.contains('bn-org-connector')).toBe(true);
		const { readFileSync } = await import('node:fs');
		const { resolve } = await import('node:path');
		const src = readFileSync(resolve(__dirname, 'OrgBoard.svelte'), 'utf8');
		const rule = src.match(/\.bn-org-connector::before \{([^}]*)\}/)?.[1] ?? '';
		expect(rule).toMatch(/top:\s*-16px/);
		expect(rule).toMatch(/height:\s*16px/);
		expect(rule).toMatch(/width:\s*1px/);
	});

	it('renders operator, the conductor head row and each department crew', () => {
		const { getByText, getAllByText, container } = render(OrgBoard, { view: view(), ventureId: null });
		expect(getByText('Alex')).toBeTruthy();
		expect(getByText('CONDUCTOR')).toBeTruthy();
		expect(getByText('Optimal Engine')).toBeTruthy();
		expect(getByText('Comms Feed')).toBeTruthy();
		expect(getByText('Sales crew')).toBeTruthy();
		expect(getByText('sales agent')).toBeTruthy(); // instance lead card
		expect(getByText('crm pulse')).toBeTruthy(); // worker pill
		expect(getByText('Agents land here as this department goes live')).toBeTruthy();
		expect(getAllByText('typeform').length).toBeGreaterThan(0);
		expect(container.querySelectorAll('[data-part="crew"]').length).toBe(2);
	});

	it('the AI Head card wears the 62px Conductor emblem, quiet until a message is sent', () => {
		const { container } = render(OrgBoard, { view: view(), ventureId: null });
		const emblem = container.querySelector('.conductor-emblem') as HTMLElement;
		expect(emblem).toBeTruthy();
		expect(emblem.style.width).toBe('62px');
		expect(emblem.classList.contains('thinking')).toBe(false);
	});

	it('the AI Head caption reads like v1: runtime until the dedicated host lands', () => {
		const { container } = render(OrgBoard, { view: view(), ventureId: null });
		expect(container.textContent).toMatch(/runtime until the dedicated host lands/);
		expect(container.textContent).not.toMatch(/Mac mini/);
	});

	it('shows the board-live strip with extras and live chips when the board answers', () => {
		const { getByText, container } = render(OrgBoard, { view: view(), ventureId: null });
		expect(getByText('Board live')).toBeTruthy();
		expect(getByText('Forge')).toBeTruthy();
		expect(container.querySelectorAll('[data-part="live-chip"]').length).toBe(2); // conductor + Sales
	});

	it('a board that is down goes inert like prod: no strip, no chips, no raw error or URL', () => {
		const err = 'board unreachable (Get "http://mini.example.ts.net:3100/api/companies/x/agents": context deadline exceeded)';
		const v = view({ live: { connected: false, error: err, conductor: null, byDepartment: {}, extras: [] } });
		v.crews[0].live = null;
		const { queryByText, container } = render(OrgBoard, { view: v, ventureId: null });
		expect(queryByText('Board live')).toBeNull();
		expect(container.querySelectorAll('[data-part="live-chip"]').length).toBe(0);
		expect(container.querySelector('[data-part="board-down"]')).toBeNull();
		expect(container.textContent).not.toMatch(/unreachable|deadline|http:\/\//);
	});

	it('the AI Head card keeps the last broadcast footer: «message» then ok count and caret', () => {
		const replies = Array.from({ length: 30 }, (_, i) => ({ agentId: `a${i}`, ok: i < 21, reply: 'r', finishedAt: '' }));
		const { getByText } = render(OrgBoard, {
			view: view({ lastBroadcast: { id: 'bc', message: 'execute with maximum visible parallelism', createdAt: '', replies } }),
			ventureId: null
		});
		expect(getByText('«execute with maximum visible parallelism»')).toBeTruthy();
		const count = getByText(/21\/30 ok/);
		expect(count.textContent?.replace(/\s+/g, ' ').trim()).toBe('21/30 ok ▸');
	});

	it('wears production\'s rounded shapes: tile head card, panel system cards, pill chat and capabilities', async () => {
		const { container, getByPlaceholderText, getByText } = render(OrgBoard, { view: view(), ventureId: null });
		const head = container.querySelector('[data-part="conductor-card"]')!;
		expect(head.className).toContain('rounded-[12px]');
		const sys = container.querySelectorAll('[data-part="system-card"]');
		expect(sys.length).toBe(2);
		sys.forEach((c) => expect(c.className).toContain('rounded-[10px]'));
		expect(getByPlaceholderText('Chat with Conductor — reaches every agent').className).toContain('rounded-full');
		expect(getByText('Send').className).toContain('rounded-full');
		expect(getByText('Broadcast').className).toContain('rounded-full');
		expect(getByText('All ventures').className).toContain('rounded-[6px]');
		const legend = getByText('Life areas').parentElement!;
		expect(legend.className).toContain('rounded-[10px]');
		const pill = container.querySelector('[data-agent="crm-pulse"]')!;
		expect(pill.className).toContain('rounded-full');
		await fireEvent.click(pill);
		expect(pill.className).toContain('rounded-[8px]');
		expect(container.querySelector('[data-agent="sales-agent"]')!.className).toContain('rounded-[6px]');
	});

	it('roster dots read like prod STATUS_DOT: active white, planned hollow; board idle is muted', () => {
		const v = view();
		v.crews[0].pills[1] = { agent: { ...v.crews[0].pills[1].agent, status: 'planned' }, children: [] };
		const { container } = render(OrgBoard, { view: v, ventureId: null });
		const dotOf = (sel: string) => container.querySelector(`${sel} [data-part="org-dot"]`) as HTMLElement;
		const lead = dotOf('[data-agent="sales-agent"]');
		expect(lead.className).toContain('rounded-full');
		expect(lead.style.background).toBe('var(--bn-text)');
		expect(dotOf('[data-agent="crm-pulse"]').style.background).toBe('var(--bn-text)');
		const planned = dotOf('[data-agent="whatsapp-worker"]');
		expect(planned.className).toContain('border');
		expect(planned.style.borderColor).toBe('var(--bn-text-3)');
		// the Sales crew's board chip is idle: muted, not amber
		const chips = container.querySelectorAll('[data-part="live-chip"] [data-part="org-dot"]');
		const idle = [...chips].find((d) => d.parentElement!.textContent!.includes('idle')) as HTMLElement;
		expect(idle.style.background).toBe('var(--bn-text-2)');
	});

	it('the Conductor tool chips sit on one centred row like prod (no wrap)', () => {
		const { container } = render(OrgBoard, { view: view(), ventureId: null });
		const head = container.querySelector('[data-part="conductor-card"]')!;
		const row = head.querySelector('[data-part="conductor-tools"]')!;
		expect(row.className).not.toContain('flex-wrap');
		expect(row.className).toContain('justify-center');
	});

	it('a venture lens dims agents outside its crew and shows the executive focus', () => {
		const { container, getByText } = render(OrgBoard, { view: view(), ventureId: 'vantage' });
		expect(getByText('Vantage — executive focus')).toBeTruthy();
		expect(getByText('Ship builds')).toBeTruthy();
		expect(container.textContent).toMatch(/Brain tag: #vantage · 3 agents on this venture/);
		expect(container.textContent).not.toMatch(/G-Brain/);
		const wa = container.querySelector('[data-agent="whatsapp-worker"]')!;
		const crm = container.querySelector('[data-agent="crm-pulse"]')!;
		expect(wa.getAttribute('data-dim')).toBe('true');
		expect(crm.getAttribute('data-dim')).toBe('false');
	});

	it('broadcast composer posts to /agents/broadcast and shows the replies', async () => {
		fetchMock.mockResolvedValueOnce({
			id: 'b1',
			message: 'status?',
			createdAt: '2026-09-30T00:00:00Z',
			replies: [
				{ agentId: 'crm-pulse', ok: true, reply: 'all green', finishedAt: '' },
				{ agentId: 'sales-agent', ok: false, reply: 'board down', finishedAt: '' }
			]
		});
		const { getByPlaceholderText, getByText } = render(OrgBoard, { view: view(), ventureId: null });
		const input = getByPlaceholderText('Chat with Conductor — reaches every agent') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: 'status?' } });
		await fireEvent.click(getByText('Send'));
		await waitFor(() => expect(getByText('1/2 ok')).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/agents/broadcast', { method: 'POST', json: { message: 'status?' } });
		expect(getByText(/all green/)).toBeTruthy();
		expect(getByText('CRM Pulse')).toBeTruthy();
	});

	it('a worker pill run surfaces a 404 inline instead of pretending', async () => {
		fetchMock.mockRejectedValueOnce(new FounderosApiError(404, 'unknown agent: crm-pulse'));
		const { container, getByText } = render(OrgBoard, { view: view(), ventureId: null });
		await fireEvent.click(container.querySelector('[data-agent="crm-pulse"]')!);
		await fireEvent.click(getByText('▸ run'));
		await waitFor(() => expect(getByText(/unknown agent: crm-pulse/)).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/agents/crm-pulse/run', { method: 'POST' });
	});

	it('the worker run control is prod AsyncButton secondary: 26px, spinner + running + elapsed, then ✓ ok', async () => {
		let resolve!: (v: unknown) => void;
		fetchMock.mockReturnValueOnce(new Promise((r) => (resolve = r)) as never);
		const { container, getByText, queryByText } = render(OrgBoard, { view: view(), ventureId: null });
		await fireEvent.click(container.querySelector('[data-agent="crm-pulse"]')!);
		const btn = getByText('▸ run').closest('button')!;
		expect(btn.className).toContain('h-[26px]');
		expect(btn.getAttribute('data-tone')).toBe('secondary');
		await fireEvent.click(btn);
		await waitFor(() => expect(btn.textContent).toContain('running'));
		expect(btn.getAttribute('aria-busy')).toBe('true');
		resolve({ ok: true, summary: 'done' });
		await waitFor(() => expect(btn.textContent?.replace(/\s+/g, '')).toBe('✓ok'));
		expect(queryByText('▸ run')).toBeNull();
	});

	it('Send is prod AsyncButton primary in a full pill: a failed broadcast reads ✗, not ✓', async () => {
		fetchMock.mockRejectedValueOnce(new FounderosApiError(503, 'bridge down'));
		const { getByPlaceholderText, getByText, container } = render(OrgBoard, { view: view(), ventureId: null });
		const send = getByText('Send').closest('button')!;
		expect(send.getAttribute('data-tone')).toBe('primary');
		expect(send.className).toContain('rounded-full');
		await fireEvent.input(getByPlaceholderText('Chat with Conductor — reaches every agent'), { target: { value: 'hi' } });
		await fireEvent.click(send);
		await waitFor(() => expect(send.textContent).toContain('✗'));
		expect(container.textContent).toContain('⚠ bridge down');
	});
});
