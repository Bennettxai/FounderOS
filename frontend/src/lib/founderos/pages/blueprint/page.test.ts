import { readFileSync, readdirSync } from 'node:fs';
import { join, resolve } from 'node:path';
import { fireEvent, render, waitFor } from '@testing-library/svelte';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import type { BlueprintGraph } from './graph';

const fetchMock = vi.fn();
vi.mock('$lib/founderos/api', () => ({
	founderosFetch: (...args: unknown[]) => fetchMock(...args)
}));

import Page from '../../../../routes/(founderos)/os/blueprint/+page.svelte';

function graph(): BlueprintGraph {
	return {
		compiledAt: new Date(Date.now() - 3 * 60_000).toISOString(),
		nodes: [
			{ id: 'operator', kind: 'operator', name: 'the operator', layer: 0, status: 'live', blurb: 'The operator.', facts: {}, icon: 'user-round' },
			{ id: 'agent-conductor', kind: 'wizard', name: 'Conductor', layer: 1, status: 'live', blurb: 'Routes everything.', facts: {}, icon: 'wand-2' },
			{ id: 'dept-sales', kind: 'department', name: 'Sales', layer: 2, status: 'live', blurb: 'pipeline', facts: {}, icon: 'folder' },
			{ id: 'agent-inbox', kind: 'agent', name: 'Inbox Agent', layer: 2, status: 'live', blurb: '', facts: { role: 'Reads mail' }, icon: 'bot' },
			{ id: 'connector-slack', kind: 'connector', name: 'Slack', layer: 3, status: 'live', blurb: 'ok', facts: { kind: 'slack' }, icon: 'plug' },
			{ id: 'host-macbook', kind: 'host', name: 'MacBook Pro', layer: 4, status: 'live', blurb: 'Dev machine.', facts: { chip: 'M5 Pro' }, icon: 'laptop' },
			{ id: 'service-bridge-backend', kind: 'service', name: 'Bridge backend', layer: 4, status: 'configured', blurb: 'Go backend', facts: { port: '8801' }, icon: 'activity' },
			{ id: 'cloud-github', kind: 'service', name: 'GitHub', layer: 4, status: 'configured', blurb: 'origin/main', facts: {}, icon: 'git-branch' }
		],
		edges: [
			{ from: 'operator', to: 'agent-conductor', kind: 'commands' },
			{ from: 'agent-conductor', to: 'agent-inbox', kind: 'commands' },
			{ from: 'agent-inbox', to: 'dept-sales', kind: 'member-of' },
			{ from: 'agent-inbox', to: 'connector-slack', kind: 'uses' },
			{ from: 'service-bridge-backend', to: 'host-macbook', kind: 'runs-on' }
		]
	};
}

// The page lazy-loads the workspace; warm the module so a cold transform
// never races the 1s findBy timeout.
beforeAll(async () => {
	await import('./HierarchyWorkspace.svelte');
}, 30_000);
// braces matter: a function returned from beforeEach is run as its teardown,
// and mockReset returns the mock itself
beforeEach(() => {
	fetchMock.mockReset();
});

describe('/os/blueprint', () => {
	it('fetches the compiled graph from the bridge route', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		render(Page);
		await waitFor(() => expect(fetchMock).toHaveBeenCalledWith('/pages/blueprint'));
	});

	it('shows a dimension-matched skeleton while the graph compiles', async () => {
		let answer: (v: unknown) => void = () => {};
		fetchMock.mockReturnValue(new Promise((r) => (answer = r)));
		const { container } = render(Page);
		expect(container.querySelector('.bh-skeleton')).toBeTruthy();
		answer({ ok: true, graph: graph() });
		await waitFor(() => expect(container.querySelector('.bh-skeleton')).toBeNull());
	});

	it('draws the workspace: title, counted subline, containers, levels, legend', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, getByText, container } = render(Page);
		expect(await findByText('Blueprint')).toBeTruthy();
		expect(getByText(/8 components · 1 agents · 0 daemons · 1 machines · compiled 3m ago/)).toBeTruthy();
		for (const l of ['Overview', 'Systems', 'Everything', 'Fit', 'Trace the chain']) expect(getByText(l)).toBeTruthy();
		// containers: the operator, the running OS, the machine, the cloud
		expect(container.querySelector('[data-id="k-operator"]')).toBeTruthy();
		expect(container.querySelector('[data-id="k-founderos"]')).toBeTruthy();
		expect(container.querySelector('[data-id="k-host-macbook"]')).toBeTruthy();
		expect(container.querySelector('[data-id="k-cloud"]')).toBeTruthy();
		// v1's legend plus the stack's services: no container or dev-tool kinds
		expect(container.querySelectorAll('.bh-kind').length).toBe(13);
		const legend = [...container.querySelectorAll('.bh-kind')].map((k) => k.textContent?.trim());
		expect(legend).toContain('service');
		expect(legend).not.toContain('container');
		expect(legend).not.toContain('dev tool');
		// v1 names the running OS and the ask bar "Founder OS"
		expect(container.textContent).toContain('Ask Founder OS');
		expect(container.textContent).not.toMatch(/FounderOS v1/);
		expect(container.querySelector('[data-id="k-founderos"]')?.textContent).toContain('Founder OS');
	});

	it('Everything opens every group so the agents are drawn', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, container } = render(Page);
		await findByText('Blueprint');
		expect(container.querySelector('.bh-node[data-id="agent-inbox"]')).toBeNull();
		await fireEvent.click(await findByText('Everything'));
		await waitFor(() => expect(container.querySelector('.bh-node[data-id="agent-inbox"]')).toBeTruthy());
	});

	it('selecting a card opens the inspector with its facts and connections', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, container } = render(Page);
		await findByText('Blueprint');
		await fireEvent.click(await findByText('Everything'));
		const card = await waitFor(() => {
			const el = container.querySelector('.bh-node[data-id="agent-inbox"]');
			if (!el) throw new Error('not drawn');
			return el as HTMLElement;
		});
		await fireEvent.click(card.querySelector('.bh-card') as HTMLElement);
		await waitFor(() => expect(container.querySelector('.bh-inspector .bh-insp-head')?.textContent).toContain('Inbox Agent'));
		const insp = container.querySelector('.bh-inspector') as HTMLElement;
		expect(insp.textContent).toContain('Reads mail');
		expect(insp.textContent).toContain('Slack');
		expect(container.querySelector('.bh-crumb.is-on')?.textContent).toContain('Sales');
	});

	it('search finds a thing by name and reports its path', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, container } = render(Page);
		await findByText('Blueprint');
		await fireEvent.click(await findByText('Search the system'));
		const input = container.querySelector('.bh-search-panel input') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: 'slack' } });
		await waitFor(() => expect(container.querySelector('.bh-sr .t')?.textContent).toBe('Slack'));
		expect(container.querySelector('.bh-sr .k')?.textContent).toBe('Founder OS › Connectors');
	});

	it('the ask bar looks like production (live input, chips, no standing banner) and answers a short honest line', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, container } = render(Page);
		await findByText('Blueprint');
		const input = container.querySelector('.bh-ask input') as HTMLInputElement;
		expect(input.disabled).toBe(false);
		expect(input.placeholder).toBe('ask anything about the system…');
		expect(container.querySelector('.bh-ask-answer')).toBeNull();
		const chip = container.querySelector('.bh-ask-chip') as HTMLButtonElement;
		expect(chip.disabled).toBe(false);
		await fireEvent.click(chip);
		const answer = await waitFor(() => {
			const el = container.querySelector('.bh-ask-answer.is-error');
			if (!el) throw new Error('no answer');
			return el as HTMLElement;
		});
		expect(answer.textContent!.length).toBeLessThan(90);
		expect(answer.textContent).not.toMatch(/\/api\//);
		// no analyst route exists, so nothing is fetched beyond the graph
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('every control presses like production (pressable, no magnetic lens)', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, container } = render(Page);
		await findByText('Blueprint');
		for (const sel of ['.bh-lvl', '.bh-btn', '.bh-kind', '.bh-search-pill', '.bh-hud-btn', '.bh-ask-scope', '.bh-ask-send', '.bh-ask-chip']) {
			const els = [...container.querySelectorAll(sel)];
			expect(els.length, sel).toBeGreaterThan(0);
			for (const el of els) {
				expect(el.classList.contains('bn-pressable'), sel).toBe(true);
				expect(el.hasAttribute('data-lens'), sel).toBe(false);
			}
		}
		// selecting a card opens the inspector; its tabs and actions press too
		await fireEvent.click(await findByText('Everything'));
		const card = await waitFor(() => {
			const el = container.querySelector('.bh-node[data-id="agent-inbox"] .bh-card');
			if (!el) throw new Error('not drawn');
			return el as HTMLElement;
		});
		await fireEvent.click(card);
		await waitFor(() => expect(container.querySelector('.bh-itab')).toBeTruthy());
		for (const sel of ['.bh-insp-handle', '.bh-itab', '.bh-insp-min', '.bh-insp-actions .bh-btn']) {
			for (const el of container.querySelectorAll(sel)) expect(el.classList.contains('bn-pressable'), sel).toBe(true);
		}
	});

	it('an ask thinks first (scoped copy, busy bar), then lands its honest line', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: graph() });
		const { findByText, container } = render(Page);
		await findByText('Blueprint');
		await fireEvent.click(container.querySelector('.bh-ask-chip') as HTMLButtonElement);
		const thinking = container.querySelector('.bh-ask-answer.is-thinking');
		expect(thinking?.textContent).toBe('Looking at the whole map…');
		expect(container.querySelector('.bh-ask.is-busy')).toBeTruthy();
		await waitFor(() => expect(container.querySelector('.bh-ask-answer.is-error')).toBeTruthy());
		expect(container.querySelector('.bh-ask.is-busy')).toBeNull();
	});

	it('an unreachable roster reads as an error, never an empty map', async () => {
		fetchMock.mockRejectedValue(new Error('the roster is unreachable: no founderos workspace (HTTP 503)'));
		const { findByText, container } = render(Page);
		expect(await findByText(/roster is unreachable/)).toBeTruthy();
		expect(container.querySelector('.bh-canvas')).toBeNull();
	});

	it('an empty graph says so', async () => {
		fetchMock.mockResolvedValue({ ok: true, graph: { compiledAt: new Date().toISOString(), nodes: [], edges: [] } });
		const { findByText } = render(Page);
		expect(await findByText(/nothing compiled/i)).toBeTruthy();
	});
});

describe('/os/blueprint: house rules on every ported file', () => {
	const dir = resolve(__dirname);
	const route = resolve(__dirname, '../../../../routes/(founderos)/os/blueprint/+page.svelte');
	const files = [route, ...readdirSync(dir).filter((f) => !f.endsWith('.test.ts')).map((f) => join(dir, f))];
	it.each(files)('%s', (file) => {
		const src = readFileSync(file, 'utf8');
		expect(src).not.toContain('—');
		expect(src).not.toMatch(/glados/i);
		expect(src).not.toMatch(/PortingPlaceholder|bridge:partial/);
	});
	it('the edge layer escapes the global svg max-width so the connecting lines draw', () => {
		const css = readFileSync(join(dir, 'blueprint.css'), 'utf8');
		const rule = css.match(/\.bh-edges\s*\{([^}]*)\}/)?.[1] ?? '';
		expect(rule).toMatch(/max-width:\s*none/);
	});
	it('icons keep their size when the header squeezes (no global svg max-width cap)', () => {
		const css = readFileSync(join(dir, 'blueprint.css'), 'utf8');
		const rule = css.match(/\.bh-page :where\(svg\)\s*\{([^}]*)\}/)?.[1] ?? '';
		expect(rule).toMatch(/max-width:\s*none/);
	});
	it('the canvas yields to the Conductor dock like production (right: the dock width)', () => {
		const css = readFileSync(join(dir, 'blueprint.css'), 'utf8');
		expect(css).toMatch(/right:\s*var\(--bn-conductor-w, 0px\)/);
	});
	it('the canvas sits under the 52px top bar beside the resizable sidebar', () => {
		const css = readFileSync(join(dir, 'blueprint.css'), 'utf8');
		expect(css).toMatch(/top:\s*52px/);
		expect(css).toMatch(/left:\s*var\(--bn-sidebar-w, 232px\)/);
		for (const k of ['operator', 'command', 'app', 'router', 'agent', 'department', 'person', 'skill', 'connector', 'model', 'store', 'surface', 'daemon', 'service', 'docker', 'tool', 'machine', 'cloud', 'group']) {
			expect(css, k).toMatch(new RegExp(`--bh-k-${k}\\s*:`));
		}
	});
});
