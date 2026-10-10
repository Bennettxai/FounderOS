import { render, waitFor } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const fetchMock = vi.hoisted(() => vi.fn());
vi.mock('$lib/founderos/api', () => ({ founderosFetch: fetchMock }));
// The (founderos) layout load hands every /os page the signed-in session.
vi.mock('$app/state', () => ({
	page: { data: { user: { name: 'Alex Rivera', email: 'alex@founderos.local' } }, url: new URL('http://localhost/os/brain'), params: {} }
}));

import Page from './+page.svelte';
import { PAGE } from '$lib/founderos/pages/brain/fixtures';

const OVERVIEW = {
	store: { path: 'optimal-engine', totalFiles: 3, folders: [{ name: 'founderos', files: 2 }] },
	doctor: {
		connected: true,
		status: 'warn',
		healthScore: null,
		detail: '2/2 engines up · 7 workspaces · 3 pages',
		checks: [
			{ name: 'hub engine', status: 'ok', message: 'up' },
			{ name: 'hermes workspace', status: 'warn', message: 'home engine is not staged on this machine; its memory is not read' },
			{ name: 'reranker', status: 'warn', message: 'offline: results stay in engine order' }
		]
	}
};

function serve(overrides: Record<string, () => unknown> = {}) {
	fetchMock.mockImplementation(async (path: string) => {
		if (overrides[path]) return overrides[path]();
		if (path === '/pages/brain') return PAGE;
		if (path === '/pages/brain/graph') return { constellation: { nodes: [], edges: [] }, workspaces: [{ workspace: 'founderos', engine: 'hub', pages: 2, edges: 1 }], unstaged: ['hermes'] };
		if (path === '/pages/brain/overview') return OVERVIEW;
		if (path === '/pages/brain/satellites') return { mounted: true, statusLine: 'optimal-engine · connected', pages: 3, folders: 1, clusters: [], freshPct: 100, stalePct: 0 };
		return { results: [] };
	});
}

beforeEach(() => fetchMock.mockReset());

describe('/os/brain', () => {
	it('one no-scroll screen: title, the capture in the header right slot, the graph under it', async () => {
		serve();
		const { container } = render(Page);
		const header = container.querySelector('header')!;
		expect(header.querySelector('h1')!.textContent).toBe('Brain');
		expect(header.textContent).toContain('knowledge core');
		expect(header.querySelector('[data-part="right"] [data-part="brain-dump"]')).toBeTruthy();
		expect(container.querySelector('[data-part="brain-page"]')!.getAttribute('style')).toContain('100dvh');
		await waitFor(() => expect(container.querySelector('[data-part="knowledge-graph"]')).toBeTruthy());
		expect(container.querySelectorAll('[data-node]').length).toBe(PAGE.graph.nodes.length);
		expect(container.querySelector('[data-part="satellites"]')).toBeTruthy();
		for (const path of ['/pages/brain', '/pages/brain/graph', '/pages/brain/satellites']) {
			expect(fetchMock).toHaveBeenCalledWith(path);
		}
	});

	it('the core is the signed-in operator, by first name, as in v1 ("Alex")', async () => {
		serve();
		const { container } = render(Page);
		await waitFor(() => expect(container.querySelector('[data-node="self"]')).toBeTruthy());
		expect(container.querySelector('[data-node="self"] title')!.textContent).toBe('Alex');
		expect(container.textContent).not.toContain('the operator');
	});

	it('like production, health readouts live on Doctor: no warnings row over the graph', async () => {
		serve();
		const { container } = render(Page);
		await waitFor(() => expect(container.querySelector('[data-part="knowledge-graph"]')).toBeTruthy());
		expect(container.querySelector('[data-part="doctor-warnings"]')).toBeNull();
		expect(fetchMock).not.toHaveBeenCalledWith('/pages/brain/overview');
	});

	it('an unavailable org says why instead of drawing an empty graph', async () => {
		serve({
			'/pages/brain': () => {
				throw new Error('the founderos workspace is not bootstrapped: no org to draw (HTTP 503)');
			}
		});
		const { container } = render(Page);
		await waitFor(() => expect(container.querySelector('[data-part="graph-error"]')).toBeTruthy());
		expect(container.querySelector('[data-part="graph-error"]')!.textContent).toContain('not bootstrapped');
		expect(container.querySelector('[data-node]')).toBeNull();
	});

	it('an unreadable workspace reaches the memory core as an error', async () => {
		serve({
			'/pages/brain/graph': () => ({ constellation: { nodes: [], edges: [] }, workspaces: [{ workspace: 'personal', engine: 'macbook', pages: null, edges: null, error: 'engine macbook: HTTP 503' }], unstaged: [] })
		});
		const { container } = render(Page);
		await waitFor(() => expect(container.querySelector('[data-part="memory-error"]')).toBeTruthy());
		expect(container.querySelector('[data-part="memory-error"]')!.textContent).toContain('personal: engine macbook: HTTP 503');
	});
});
