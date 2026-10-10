import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const fetchMock = vi.hoisted(() => vi.fn());
vi.mock('$lib/founderos/api', async (importOriginal) => ({ ...(await importOriginal<typeof import('$lib/founderos/api')>()), founderosFetch: fetchMock }));

import BrainDump from './BrainDump.svelte';
import BrainGraphView from './BrainGraphView.svelte';
import BrainSatellites from './BrainSatellites.svelte';
import KnowledgeGraph from './KnowledgeGraph.svelte';
import { FounderosApiError } from '$lib/founderos/api';
import { GRAPH, PAGE } from './fixtures';

beforeEach(() => {
	fetchMock.mockReset();
	fetchMock.mockResolvedValue({ results: [] });
});
afterEach(() => vi.useRealTimers());

describe('BrainDump (compact capture)', () => {
	it('Save captures into the engine through /pages/brain/dump and reports the signal', async () => {
		fetchMock.mockResolvedValueOnce({ ok: true, title: 'call Dana', relPath: 'inbox/x.md', workspace: 'founderos', embedded: true, slug: 'sig-1' });
		const onSaved = vi.fn();
		render(BrainDump, { onSaved });
		const box = screen.getByLabelText('Dump into the brain') as HTMLTextAreaElement;
		await fireEvent.input(box, { target: { value: 'call Dana' } });
		await fireEvent.click(screen.getByText('Save'));
		await waitFor(() => expect(screen.getByText('✓ embedded · sig-1')).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/brain/dump', { method: 'POST', json: { text: 'call Dana', folder: 'inbox', tags: [] } });
		expect(onSaved).toHaveBeenCalled();
		expect(box.value).toBe('');
	});

	it('a failed capture says nothing was saved and keeps the text', async () => {
		fetchMock.mockRejectedValueOnce(new Error('engine hub unreachable (HTTP 502)'));
		render(BrainDump);
		const box = screen.getByLabelText('Dump into the brain') as HTMLTextAreaElement;
		await fireEvent.input(box, { target: { value: 'keep me' } });
		await fireEvent.click(screen.getByText('Save'));
		await waitFor(() => expect(screen.getByText(/✗ nothing saved: engine hub unreachable/)).toBeTruthy());
		expect(box.value).toBe('keep me');
	});

	it('dropped documents become one note each, titled by filename; non-text is refused', async () => {
		fetchMock.mockResolvedValue({ ok: true, title: 't', relPath: 'x', workspace: 'founderos', embedded: true, slug: 's' });
		const { container } = render(BrainDump);
		const zone = container.querySelector('[data-part="brain-dump"]')!;
		// jsdom's File has no text(); browsers do
		const md = Object.assign(new File(['# notes'], 'Call notes.md', { type: 'text/markdown' }), { text: async () => '# notes' });
		const png = new File(['x'], 'pic.png', { type: 'image/png' });
		await fireEvent.drop(zone, { dataTransfer: { files: [md, png] } });
		await waitFor(() => expect(screen.getByText(/1 doc · 1 skipped/)).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/brain/dump', { method: 'POST', json: { text: '# notes', title: 'Call notes', folder: 'inbox', tags: [] } });
		await fireEvent.drop(zone, { dataTransfer: { files: [png] } });
		await waitFor(() => expect(screen.getByText(/only text documents/)).toBeTruthy());
	});
});

describe('BrainSatellites', () => {
	it('shows engine counts and answers the ask bar from /pages/brain/query', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/brain/satellites') return { mounted: true, statusLine: 'optimal-engine · connected · 2/2 engines up', pages: 1421, folders: 5, clusters: [], freshPct: 100, stalePct: 0 };
			if (path.startsWith('/pages/brain/query')) return { query: 'hermes', provider: 'optimal-engine', ranked: 'provider', results: [{ title: 'Hermes worker pool', snippet: 'how the pool runs', source: 'optimal://a', workspace: 'founderos', engine: 'hub', uri: 'optimal://a', score: null }] };
			throw new Error(path);
		});
		const { container } = render(BrainSatellites);
		await waitFor(() => expect(container.querySelector('[data-part="pages"]')?.textContent).toContain('1,421'));
		expect(screen.getByText('connected')).toBeTruthy();
		await fireEvent.input(screen.getByLabelText('Ask the brain'), { target: { value: 'hermes' } });
		await fireEvent.submit(container.querySelector('form')!);
		await waitFor(() => expect(screen.getByText('retrieved from 1 notes')).toBeTruthy());
		expect(screen.getByText('how the pool runs')).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledWith('/pages/brain/query?q=hermes');
	});

	it('counts the workspaces holding pages in the singular when there is one', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/brain/satellites') return { mounted: true, statusLine: 'up', pages: 114, folders: 1, clusters: [], freshPct: 100, stalePct: 0 };
			throw new Error(path);
		});
		const { container } = render(BrainSatellites);
		await waitFor(() => expect(container.querySelector('[data-part="folders"]')?.textContent).toBe('1workspace'));
	});

	it('an unreachable engine reads unreachable, and a failed ask shows its error', async () => {
		fetchMock.mockRejectedValue(new Error('engine hub unreachable'));
		const { container } = render(BrainSatellites);
		await waitFor(() => expect(container.querySelector('[data-part="ask-status"]')?.textContent).toBe('unreachable'));
		expect(container.querySelector('[data-part="pages"]')).toBeNull();
		await fireEvent.input(screen.getByLabelText('Ask the brain'), { target: { value: 'hermes' } });
		await fireEvent.submit(container.querySelector('form')!);
		await waitFor(() => expect(screen.getByText('engine hub unreachable')).toBeTruthy());
	});

	it('a slower earlier ask never overwrites the newer answer', async () => {
		let releaseFirst!: () => void;
		const hit = (snippet: string) => ({ query: snippet, provider: 'optimal-engine', ranked: 'provider', results: [{ title: `note: ${snippet}`, snippet, source: 'optimal://a', workspace: 'founderos', engine: 'hub', uri: 'optimal://a', score: null }] });
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/brain/satellites') return { mounted: true, statusLine: 'up', pages: 1, folders: 1, clusters: [], freshPct: 100, stalePct: 0 };
			if (path === '/pages/brain/query?q=first') return new Promise((r) => (releaseFirst = () => r(hit('old answer'))));
			if (path === '/pages/brain/query?q=second') return hit('new answer');
			throw new Error(path);
		});
		const { container } = render(BrainSatellites);
		const box = screen.getByLabelText('Ask the brain');
		await fireEvent.input(box, { target: { value: 'first' } });
		await fireEvent.submit(container.querySelector('form')!);
		await fireEvent.input(box, { target: { value: 'second' } });
		await fireEvent.submit(container.querySelector('form')!);
		await waitFor(() => expect(screen.getByText('new answer')).toBeTruthy());
		releaseFirst();
		await new Promise((r) => setTimeout(r, 0));
		expect(screen.queryByText('old answer')).toBeNull();
		expect(screen.getByText('new answer')).toBeTruthy();
	});

	it('a partial answer names the workspaces it could not search', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/brain/satellites') return { mounted: true, statusLine: 'up', pages: 1, folders: 1, clusters: [], freshPct: 100, stalePct: 0 };
			if (path.startsWith('/pages/brain/query'))
				return { query: 'q', provider: 'optimal-engine', ranked: 'provider', results: [{ title: 'Hit', snippet: 'found on the hub', source: 'optimal://a', workspace: 'founderos', engine: 'hub', uri: 'optimal://a', score: null }], failedWorkspaces: ['personal'], degraded: 'engine macbook unreachable' };
			throw new Error(path);
		});
		const { container } = render(BrainSatellites);
		await fireEvent.input(screen.getByLabelText('Ask the brain'), { target: { value: 'hermes' } });
		await fireEvent.submit(container.querySelector('form')!);
		await waitFor(() => expect(screen.getByText('found on the hub')).toBeTruthy());
		expect(screen.getByText(/partial · personal not searched/)).toBeTruthy();
	});

	it('fades out while a node card is open', () => {
		const { container } = render(BrainSatellites, { quiet: true });
		fetchMock.mockResolvedValue(null);
		expect(container.querySelector('[data-part="satellites"]')!.getAttribute('data-quiet')).toBe('true');
	});
});

const MEMORY = {
	nodes: [
		{ id: 'folder:founderos', type: 'folder' as const, label: 'founderos', folder: 'founderos', excerpt: '', vx: 0, vy: 0, cluster: 0, links: 1 },
		{ id: 'founderos:a', type: 'page' as const, label: 'Hermes worker pool', folder: 'founderos', genre: 'sop', excerpt: 'how the pool runs', vx: 0.2, vy: 0.1, cluster: 0, links: 1 }
	],
	edges: [{ source: 'folder:founderos', target: 'founderos:a', type: 'member' as const }]
};
const node = (c: Element, id: string) => c.querySelector(`[data-node="${id}"]`)!;
const litOf = (c: Element, id: string) => node(c, id).getAttribute('data-lit');

describe('BrainGraphView (production parity)', () => {
	it('the Radial/Neural switch is a sliding pill and the pillar chips follow department order', async () => {
		const page = { ...PAGE, departments: [...PAGE.departments].reverse() };
		const { container } = render(BrainGraphView, { page });
		const tabs = container.querySelector('[role="tablist"]')!;
		expect(tabs.getAttribute('data-variant')).toBe('pill');
		const indicator = () => (container.querySelector('[data-part="tab-indicator"]') as HTMLElement).style.transform;
		expect(indicator()).toBe('translateX(0px)');
		const chips = [...container.querySelectorAll('[data-part="pillar-chips"] button')];
		expect(chips.map((c) => c.textContent!.trim())).toEqual(['Sales', 'TECH']);
		expect(chips.every((c) => c.getAttribute('data-on') === 'true')).toBe(true);
		await fireEvent.click(screen.getByRole('tab', { name: 'Neural' }));
		expect(indicator()).toBe('translateX(84px)');
		expect(container.querySelector('[data-part="pillar-chips"]')).toBeNull();
	});

	it('every pillar off empties the directory with a way back, like production', async () => {
		const { container } = render(BrainGraphView, { page: PAGE });
		await waitFor(() => expect(container.querySelector('[data-part="directory"]')).toBeTruthy());
		for (const c of [...container.querySelectorAll('[data-part="pillar-chips"] button')]) await fireEvent.click(c);
		const dir = container.querySelector('[data-part="directory"]')!;
		expect(dir.textContent).toContain('No pillars selected.');
		await fireEvent.click(screen.getByRole('button', { name: 'show all pillars' }));
		const chips = [...container.querySelectorAll('[data-part="pillar-chips"] button')];
		expect(chips.every((c) => c.getAttribute('data-on') === 'true')).toBe(true);
	});

	it('an unreachable board is not announced over the graph (production shows no such note)', async () => {
		const { container } = render(BrainGraphView, { page: { ...PAGE, board: { state: 'error' as const, detail: 'Paperclip unreachable' } } });
		await waitFor(() => expect(container.querySelector('[data-part="knowledge-graph"]')).toBeTruthy());
		expect(container.textContent).not.toContain('board ring unavailable');
	});

	it('Neural swaps in production’s neural network: labelled layers, the directory, a hover readout and the detail card', async () => {
		const { container } = render(BrainGraphView, { page: PAGE });
		await fireEvent.click(screen.getByRole('tab', { name: 'Neural' }));
		await waitFor(() => expect(container.querySelector('[data-part="neural-graph"]')).toBeTruthy());
		expect(container.querySelector('[data-part="knowledge-graph"]')).toBeNull();
		expect(container.querySelector('[data-part="layer-names"]')!.textContent).toContain('INPUT · TOOLS');
		expect(container.querySelector('[data-part="neural-graph"] [data-part="directory"]')).toBeTruthy();
		const neuron = container.querySelector('[data-neuron="emp:conductor"]')!;
		await fireEvent.mouseEnter(neuron);
		expect(container.querySelector('[data-part="neural-readout"]')!.textContent).toContain('Conductor');
		expect(container.querySelector('[data-part="neural-readout"]')!.textContent).toContain('AI agent');
		await fireEvent.click(neuron);
		expect(container.querySelector('[data-part="neural-detail"]')!.textContent).toContain('Super agent · AI agent');
	});

	it('loads each engine with a dynamic import behind a matching skeleton, never statically', () => {
		const src = readFileSync(resolve(__dirname, 'BrainGraphView.svelte'), 'utf8');
		expect(src).toContain("import('./KnowledgeGraph.svelte')");
		expect(src).toContain("import('./NeuralGraph.svelte')");
		expect(src).not.toMatch(/import\s+\w+\s+from\s+'\.\/(KnowledgeGraph|NeuralGraph)\.svelte'/);
		expect(src).toContain('data-part="graph-skeleton"');
		expect(src).toContain('aspect-ratio: 1200 / 640');
	});
});

describe('KnowledgeGraph (the FounderOS v1 radial wheel, ported)', () => {
	const props = { page: PAGE, memory: null };

	it('draws the resting web: one edge path per graph edge (prod paints 188 on the live brain)', async () => {
		const { container } = render(KnowledgeGraph, props);
		await tick();
		const edges = container.querySelectorAll('[data-part="edges"] path');
		expect(edges.length).toBeGreaterThan(0);
	});

	it('draws one node per graph node, the lens/legend aside and the directory', async () => {
		const { container } = render(KnowledgeGraph, props);
		expect(container.querySelectorAll('[data-node]').length).toBe(GRAPH.nodes.length);
		const aside = container.querySelector('[data-part="graph-aside"]')!;
		expect([...aside.querySelectorAll('select')].map((s) => s.getAttribute('aria-label'))).toEqual(['Entity lens', 'Function lens', 'Action lens']);
		const legend = [...aside.querySelectorAll('[data-part="legend"] [data-kind]')].map((r) => r.textContent!.replace(/\s+/g, ' ').trim());
		expect(legend).toEqual(['Pillars 2', 'Board agents 1', 'SOP tasks 2', 'Humans 1', 'AI agents 2', 'Tools 3']);
		expect(aside.querySelector('[data-part="directory"]')!.textContent).toContain('Openclaw');
		await fireEvent.click(screen.getByRole('button', { name: 'Hide directory' }));
		expect(screen.getByRole('button', { name: 'Show directory' })).toBeTruthy();
	});

	it('nodes wear their kind glyphs and colours; a pillar wears its life-area hue, label under the hub', () => {
		const { container } = render(KnowledgeGraph, props);
		const ring = (id: string) => node(container, id).querySelector('circle[data-part="ring"]')!;
		expect(ring('tool:openclaw').getAttribute('stroke')).toBe('var(--bn-kg-tool)');
		expect(ring('team:dept-sales').getAttribute('stroke')).toBe('var(--kg-c-ef4444, #ef4444)');
		for (const [id, kind] of [['task:sop-c', 'task'], ['person:len', 'person'], ['emp:conductor', 'employee'], ['board:p2', 'board'], ['tool:openclaw', 'tool'], ['team:dept-sales', 'team']]) {
			expect(container.querySelector(`[data-node="${id}"] [data-icon="${kind}"]`)).toBeTruthy();
		}
		const label = node(container, 'team:dept-sales').querySelector('text')!;
		expect(label.textContent).toBe('Sales');
		expect(Number(label.getAttribute('y'))).toBeGreaterThan(0);
	});

	it('a pillar switched off dims its nodes and drops its directory rows; a lens dims what it does not light', async () => {
		const { container, rerender } = render(KnowledgeGraph, { ...props, activePillars: ['dept-sales'] });
		expect(litOf(container, 'emp:conductor')).toBe('false');
		expect(litOf(container, 'person:len')).toBe('true');
		expect(container.querySelector('[data-part="directory"]')!.textContent).not.toContain('Conductor');
		await rerender({ ...props, activePillars: undefined });
		await fireEvent.change(screen.getByLabelText('Entity lens'), { target: { value: 'ent-tools' } });
		expect(litOf(container, 'tool:openclaw')).toBe('true');
		expect(litOf(container, 'emp:conductor')).toBe('false');
		await fireEvent.click(screen.getByRole('button', { name: 'clear · 3 lit' }));
		expect(litOf(container, 'emp:conductor')).toBe('true');
	});

	it('hovering a node lights its pillar chain and pins the node hover card', async () => {
		fetchMock.mockResolvedValue({ results: [{ title: 'Conductor runbook', snippet: '', source: 's', workspace: 'w', engine: 'hub', uri: 'u', score: null }] });
		const { container } = render(KnowledgeGraph, props);
		await fireEvent.mouseEnter(node(container, 'emp:conductor'));
		expect(litOf(container, 'task:sop-c')).toBe('true');
		expect(litOf(container, 'person:len')).toBe('false');
		const card = container.querySelector('[data-part="node-card"]')!;
		expect(card.textContent).toContain('node · hover card');
		expect(card.textContent).toContain('TECH · employee · 3 links');
		await fireEvent.click(card.querySelector('[data-part="node-query"]')!);
		await waitFor(() => expect(card.textContent).toContain('Conductor runbook'));
		expect(fetchMock).toHaveBeenCalledWith('/pages/brain/query?q=Conductor');
	});

	it('clicking a pillar grows its tree: title, Back, the pillar stepper and its department-head card', async () => {
		const onPanel = vi.fn();
		const { container } = render(KnowledgeGraph, { ...props, onPanel });
		await fireEvent.click(node(container, 'team:dept-sales'));
		expect(container.querySelector('[data-part="focus-tree"]')).toBeTruthy();
		expect(container.querySelector('[data-part="dept-title"]')!.textContent).toContain('Sales');
		expect(screen.getByRole('button', { name: 'Back to the home view' })).toBeTruthy();
		expect(container.querySelector('[data-part="pillar-stepper"]')!.textContent).toContain('Sales');
		const panel = container.querySelector('[data-part="detail-panel"]')!;
		expect(panel.textContent).toContain('CRO');
		expect(panel.textContent).toContain('Sales · department head');
		expect(panel.querySelector('[data-part="board-seat"]')!.textContent).toContain('Sales');
		expect(panel.textContent).toContain('presides over (1 skills)');
		expect(onPanel).toHaveBeenLastCalledWith(true);
		// → turns the wheel to the next pillar; Escape backs out of the card, then the pillar
		await fireEvent.click(screen.getByRole('button', { name: 'Turn to the next pillar' }));
		expect(container.querySelector('[data-part="dept-title"]')!.textContent).toContain('TECH');
		await fireEvent.keyDown(window, { key: 'ArrowLeft' });
		expect(container.querySelector('[data-part="dept-title"]')!.textContent).toContain('Sales');
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(container.querySelector('[data-part="focus-tree"]')).toBeNull();
		expect(onPanel).toHaveBeenLastCalledWith(false);
	});

	it('the head card’s Run is held by the bridge guard and says so', async () => {
		fetchMock.mockRejectedValueOnce(new FounderosApiError(403, 'writes are off', { guarded: true }));
		const { container } = render(KnowledgeGraph, props);
		await fireEvent.click(node(container, 'team:dept-sales'));
		await fireEvent.click(screen.getByRole('button', { name: 'Run heartbeat on the board' }));
		await waitFor(() => expect(container.querySelector('[data-part="run-msg"]')!.textContent).toContain('Run held: writes are off'));
		expect(fetchMock).toHaveBeenCalledWith('/pages/board/agents/p1/run', { method: 'POST' });
	});

	it('an agent opens its harness card; its tool opens the tool wiki; Back returns to the pillar', async () => {
		const { container } = render(KnowledgeGraph, props);
		await fireEvent.click(node(container, 'emp:conductor'));
		const panel = () => container.querySelector('[data-part="detail-panel"]')!;
		expect(panel().querySelector('[data-card="agent"]')).toBeTruthy();
		expect(panel().textContent).toContain('Runs the board');
		expect(panel().querySelector('[data-part="last-run"]')!.textContent).toContain('swept the board');
		expect(panel().textContent).toContain('Back · TECH');
		await fireEvent.click([...panel().querySelectorAll('button')].find((b) => b.textContent!.includes('Openclaw'))!);
		expect(panel().querySelector('[data-card="tool"]')).toBeTruthy();
		expect(panel().textContent).toContain('no page in the brain yet');
		expect(panel().textContent).toContain('used by (1)');
		await fireEvent.click(screen.getByRole('button', { name: 'Back to the TECH' }));
		expect(container.querySelector('[data-part="detail-panel"]')).toBeNull();
		expect(container.querySelector('[data-part="focus-tree"]')).toBeTruthy();
	});

	it('an SOP opens its full playbook card; the card expands into a wide column', async () => {
		const { container } = render(KnowledgeGraph, props);
		await fireEvent.click(node(container, 'task:sop-l'));
		const panel = container.querySelector('[data-part="detail-panel"]')!;
		expect(panel.querySelector('[data-card="sop"]')).toBeTruthy();
		expect(panel.textContent).toContain('HUMAN-LED');
		expect(panel.textContent).toContain('the SOP, written out');
		expect(panel.textContent).toContain('Len');
		await fireEvent.click(screen.getByRole('button', { name: 'Expand the detail card' }));
		expect(container.querySelector('[data-part="detail-wide"]')).toBeTruthy();
		expect(container.querySelector('[data-part="detail-panel"]')).toBeNull();
	});

	it('the memory core burns ember hexagons; clicking it dives in with the engine overview and a vault search', async () => {
		const { container } = render(KnowledgeGraph, { ...props, memory: MEMORY });
		const core = container.querySelector('[data-part="memory-core"]')!;
		expect(core.querySelectorAll('polygon').length).toBeGreaterThan(0);
		expect(core.querySelector('polygon')!.getAttribute('fill')).toContain('--bn-kg-mem');
		await fireEvent.click(core);
		expect(container.querySelector('[data-card="memory-core"]')!.textContent).toContain('Optimal Engine');
		expect(screen.getByLabelText('Search the vault')).toBeTruthy();
		await fireEvent.click(container.querySelector('[data-mem="founderos:a"]')!);
		expect(container.querySelector('[data-card="memory-note"]')!.textContent).toContain('how the pool runs');
	});

	it('an unreadable memory is said so, not drawn as an empty core', () => {
		const { container } = render(KnowledgeGraph, { ...props, memoryError: 'personal: engine macbook unreachable' });
		expect(container.querySelector('[data-part="memory-core"]')).toBeNull();
		expect(container.querySelector('[data-part="memory-error"]')!.textContent).toContain('engine macbook unreachable');
	});

	it('Fullscreen opens the department wheel on the first pillar; Escape closes it', async () => {
		render(KnowledgeGraph, props);
		await fireEvent.click(screen.getByRole('button', { name: /Fullscreen/ }));
		const fs = document.body.querySelector('[data-part="kg-fullscreen"]')!;
		expect(fs).toBeTruthy();
		expect(fs.querySelector('[data-part="fs-title"]')!.textContent).toContain('Sales');
		expect(fs.querySelector('[data-part="fs-picker"]')!.textContent).toContain('1 / 2');
		expect(fs.querySelector('[data-part="compact-legend"]')!.textContent).toContain('SOP task');
		expect(fs.querySelector('[data-part="fs-stepper"]')!.textContent).toContain('Sales');
		// the head card opened with the pillar; Escape backs out of it, then out of fullscreen
		await fireEvent.keyDown(window, { key: 'Escape' });
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(document.body.querySelector('[data-part="kg-fullscreen"]')).toBeNull();
	});

	it('at rest the camera breathes out a touch beyond the canvas, like production', async () => {
		const { container } = render(KnowledgeGraph, props);
		await waitFor(() => expect(Number(container.querySelector('svg[aria-label="Operating knowledge graph"]')!.getAttribute('viewBox')!.split(' ')[0])).toBeLessThan(0), { timeout: 3000 });
	});
});

describe('the capture and the satellites', () => {
	it('the capture’s mic and Upload wear their icons and Save reads as production', async () => {
		vi.stubGlobal('webkitSpeechRecognition', class {});
		render(BrainDump);
		await waitFor(() => expect(screen.getByRole('button', { name: 'Start dictation' })).toBeTruthy());
		expect(screen.getByRole('button', { name: 'Start dictation' }).querySelector('svg')).toBeTruthy();
		expect(screen.getByTitle('Choose documents to upload').querySelector('svg')).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Save' }).className).not.toContain('uppercase');
		vi.unstubAllGlobals();
	});

	it('the satellites leave the Fullscreen tab its corner and the footer reads live retrieval', () => {
		fetchMock.mockReturnValue(new Promise(() => {}));
		const { container } = render(BrainSatellites);
		expect(container.querySelector('[data-part="sat-chips"]')!.className).toContain('right-[124px]');
		expect(container.querySelector('[data-part="satellites"]')!.textContent).toContain('live retrieval');
	});
});
