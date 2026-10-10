import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import PersonasPage from '../../../../routes/(founderos)/os/personas/+page.svelte';
import PersonasViewer from './PersonasViewer.svelte';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { brainGraph, pad2, step } from './graph';
import type { Persona } from './types';

const persona = (i: number, over: Partial<Persona> = {}): Persona => ({
	id: `persona-${i}`,
	order: i,
	name: `Persona ${i}`,
	archetype: `Archetype ${i}`,
	tagline: `Tagline ${i}`,
	summary: `Summary ${i}`,
	accent: '#3df08c',
	northStar: `North ${i}`,
	pillars: [
		{ name: 'Sales', focus: 'pipeline', agents: ['Closer', 'Setter'] },
		{ name: 'Ops', focus: 'back office', agents: ['Runner'] }
	],
	connectors: ['stripe', 'gmail'],
	metrics: ['MRR', 'Churn'],
	brainUse: `Brain use ${i}`,
	signaturePlay: `Play ${i}`,
	...over
});

describe('graph geometry (PersonaBrainGraph)', () => {
	it('puts the first pillar straight up and fans agents out on the outer ring', () => {
		const g = brainGraph(persona(1));
		expect(g.pillars).toHaveLength(2);
		expect(g.pillars[0].angle).toBe(-90);
		expect(Math.round(g.pillars[0].x)).toBe(g.cx);
		expect(Math.round(g.pillars[0].y)).toBe(g.cy - 210);
		expect(g.pillars[0].agents).toHaveLength(2);
		expect(g.pillars[1].agents).toHaveLength(1);
		const a = g.pillars[1].agents[0];
		expect(Math.round(Math.hypot(a.x - g.cx, a.y - g.cy))).toBe(328);
	});
	it('the core dot field is deterministic per persona', () => {
		expect(brainGraph(persona(1)).coreDots).toEqual(brainGraph(persona(1)).coreDots);
		expect(brainGraph(persona(1)).coreDots).toHaveLength(46);
		expect(brainGraph(persona(1)).coreDots).not.toEqual(brainGraph(persona(2)).coreDots);
	});
	it('step wraps both ways and pad2 zero-pads', () => {
		expect(step(0, -1, 11)).toBe(10);
		expect(step(10, 1, 11)).toBe(0);
		expect(pad2(3)).toBe('03');
	});
});

describe('PersonasViewer', () => {
	const list = [persona(1), persona(2), persona(3)];

	it('shows one persona: header, north star, pillars with agents, connectors, tracks, signature play', () => {
		const { container } = render(PersonasViewer, { personas: list });
		expect(screen.getAllByText('Persona 1').length).toBeGreaterThan(0);
		expect(screen.getByText('Archetype 1')).toBeTruthy();
		expect(screen.getByText('North 1')).toBeTruthy();
		expect(screen.getByText('Pillars · 2')).toBeTruthy();
		expect(screen.getAllByText('Closer').length).toBe(2); // the card chip and the graph label
		expect(screen.getByText('stripe')).toBeTruthy();
		expect(screen.getByText('MRR')).toBeTruthy();
		expect(screen.getByText('Play 1')).toBeTruthy();
		expect(screen.getByText('2 depts · 3 agents')).toBeTruthy();
		// v1 heads the graph "G-Brain · knowledge graph"; G-Brain is the Brain in v2
		expect(screen.getByText('Brain · knowledge graph')).toBeTruthy();
		expect(container.textContent).not.toMatch(/G-Brain|Optimal Engine · knowledge graph/);
		const graph = container.querySelector('svg[aria-label="Persona 1 knowledge graph"]') as SVGElement;
		expect(graph).toBeTruthy();
		// app.css forces svg height:auto outside the utilities layer, which pins the
		// graph to the panel top; inline height keeps it centred like production
		expect(graph.getAttribute('style')).toContain('height: 100%');
		expect(container.textContent).toContain('01 / 03');
		// production's radius scale: panels 10, cards 8, chips 5, round rail dots
		expect(container.querySelector('article')!.className).toContain('rounded-[var(--bn-r-panel)]');
		expect(screen.getByRole('button', { name: 'Previous persona' }).className).toContain('rounded-[var(--bn-r-chip)]');
	});

	it('arrows and the jump rail step through every persona, wrapping around', async () => {
		render(PersonasViewer, { personas: list });
		await fireEvent.click(screen.getByLabelText('Previous persona'));
		expect(screen.getByText('North 3')).toBeTruthy();
		await fireEvent.click(screen.getByLabelText('Next persona'));
		expect(screen.getByText('North 1')).toBeTruthy();
		await fireEvent.click(screen.getByTitle('Persona 2'));
		expect(screen.getByText('North 2')).toBeTruthy();
	});

	it('the stepper and rail hover and light like production (scoped rules the kit colour classes cannot outrank)', async () => {
		render(PersonasViewer, { personas: list });
		const rail = screen.getByTitle('Persona 1');
		expect(rail.classList.contains('is-on')).toBe(true);
		expect(screen.getByTitle('Persona 2').classList.contains('is-on')).toBe(false);
		await fireEvent.click(screen.getByTitle('Persona 2'));
		expect(screen.getByTitle('Persona 2').classList.contains('is-on')).toBe(true);
		expect(screen.getByLabelText('Next persona').classList.contains('pv-step')).toBe(true);
		const src = readFileSync(resolve(__dirname, 'PersonasViewer.svelte'), 'utf8');
		const rule = (sel: string) => src.match(new RegExp(sel.replace(/[.:]/g, (c) => '\\' + c) + '\\s*\\{([^}]*)\\}'))?.[1] ?? '';
		expect(rule('.pv-step:hover')).toMatch(/border-color:\s*var\(--bn-border-strong\)/);
		expect(rule('.pv-step:hover')).toMatch(/color:\s*var\(--bn-text\)/);
		expect(rule('.pv-rail:hover')).toMatch(/color:\s*var\(--bn-text-2\)/);
		expect(rule('.pv-rail.is-on')).toMatch(/border-color:\s*var\(--bn-border-strong\)/);
		expect(rule('.pv-rail.is-on')).toMatch(/color:\s*var\(--bn-text\)/);
	});

	it('← / → keys step too, but not while typing', async () => {
		render(PersonasViewer, { personas: list });
		await fireEvent.keyDown(window, { key: 'ArrowRight' });
		expect(screen.getByText('North 2')).toBeTruthy();
		await fireEvent.keyDown(window, { key: 'ArrowLeft' });
		expect(screen.getByText('North 1')).toBeTruthy();
		const input = document.createElement('input');
		document.body.appendChild(input);
		await fireEvent.keyDown(input, { key: 'ArrowRight' });
		expect(screen.getByText('North 1')).toBeTruthy();
		input.remove();
	});
});

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

describe('/os/personas page', () => {
	let fetchMock: ReturnType<typeof vi.fn>;
	beforeEach(() => {
		fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
	});
	afterEach(() => vi.unstubAllGlobals());

	it('reads /pages/personas and badges the template count', async () => {
		fetchMock.mockResolvedValue(json({ personas: [persona(1), persona(2)] }));
		render(PersonasPage);
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Personas');
		await waitFor(() => expect(screen.getByText('2 templates')).toBeTruthy());
		expect(screen.getByText('North 1')).toBeTruthy();
		expect(fetchMock.mock.calls[0][0]).toBe('/api/founderos/pages/personas');
	});

	it('an unreachable source says so, never "0 templates"', async () => {
		fetchMock.mockResolvedValue(json({ error: 'personas unavailable: no Postgres' }, 503));
		const { container } = render(PersonasPage);
		await waitFor(() => expect(container.textContent).toContain('personas unavailable: no Postgres'));
		expect(container.textContent).not.toContain('0 templates');
	});

	it('an empty table reads empty', async () => {
		fetchMock.mockResolvedValue(json({ personas: [] }));
		const { container } = render(PersonasPage);
		await waitFor(() => expect(container.textContent).toContain('No personas yet'));
	});
});

describe('PersonaBrainGraph info-neuron', () => {
	it('runs the production synapse: 0.14 dash over 5s at 0.55, a faint still line under reduced motion', () => {
		const src = readFileSync(resolve(__dirname, 'PersonaBrainGraph.svelte'), 'utf8');
		const rule = src.match(/\.pbg-synapse\s*\{([^}]*)\}/)?.[1] ?? '';
		expect(rule).toMatch(/stroke-dasharray:\s*0\.14 0\.86/);
		expect(rule).toMatch(/5s linear infinite/);
		expect(rule).toMatch(/opacity:\s*0\.55/);
		const reduced = src.match(/prefers-reduced-motion: reduce\)\s*\{\s*\.pbg-synapse\s*\{([^}]*)\}/)?.[1] ?? '';
		expect(reduced).toMatch(/animation:\s*none/);
		expect(reduced).toMatch(/opacity:\s*0\.3/);
	});
});
