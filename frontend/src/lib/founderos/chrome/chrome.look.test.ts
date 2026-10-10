// The chrome's look, pinned to FounderOS v1 (components/Sidebar.tsx, Topbar.tsx,
// ThemeToggle.tsx, CommandPalette.tsx, ConductorPanel.tsx, ConductorComposer.tsx):
// the swirl emblem, "v3 · Operator Mode", rounded pressable controls on the
// hover lens, round pills and bubbles. Behaviour lives in chrome.test.ts,
// Sidebar.test.ts and ConductorPanel.test.ts.
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api', async (orig) => ({
	...(await orig<typeof import('../api')>()),
	founderosFetch: vi.fn(async (path: string) => {
		if (path.startsWith('/pages/conductor/context')) return { title: 'Home', context: 'c', quickActions: [{ label: 'Explain', prompt: 'p' }], model: 'm' };
		if (path === '/pages/conductor/chat') return { messages: [] };
		return { agents: [] };
	})
}));

import CommandPalette from './CommandPalette.svelte';
import ConductorPanel from './ConductorPanel.svelte';
import Sidebar from './Sidebar.svelte';
import Topbar from './Topbar.svelte';
import { CONDUCTOR_OPEN_EVENT } from './conductor';

const cls = (el: Element | null) => (el as HTMLElement).className;

beforeEach(() => {
	localStorage.clear();
	document.documentElement.setAttribute('data-founderos-theme', 'mono');
});

describe('Sidebar look', () => {
	it('the swirl emblem and the v3 · Operator Mode line, never a boxed letter', () => {
		const { container } = render(Sidebar, { pathname: '/os' });
		const mark = container.querySelector('aside img[alt="Founder OS"]')!;
		expect(mark.getAttribute('src')).toMatch(/os-emblem\.png/);
		expect(mark.getAttribute('width')).toBe('34');
		expect(container.querySelector('.bn-mark')).toBeNull();
		expect(screen.getByText('v3 · Operator Mode')).toBeTruthy();
		expect(screen.queryByText(/bridge · Operator Mode/i)).toBeNull();
	});
	it('collapsed, the emblem alone carries the identity at 30px', async () => {
		const { container } = render(Sidebar, { pathname: '/os' });
		await fireEvent.click(screen.getByRole('button', { name: /collapse sidebar/i }));
		expect(container.querySelector('aside img[alt="Founder OS"]')!.getAttribute('width')).toBe('30');
	});
	it('nav rows are 6px row lenses; the active one wears the accent line', () => {
		const { container } = render(Sidebar, { pathname: '/os/comms' });
		const row = container.querySelector('a[href="/os/comms"]')!;
		expect(row.getAttribute('data-lens')).toBe('r');
		for (const c of ['bn-pressable', 'is-dark', 'rounded-[6px]', 'border']) expect(cls(row)).toContain(c);
		expect(row.getAttribute('data-active')).toBe('true');
	});
	it('the collapse toggle is a 28px control lens with 6px corners', () => {
		render(Sidebar, { pathname: '/os' });
		const t = screen.getByRole('button', { name: /collapse sidebar/i });
		expect(t.getAttribute('data-lens')).toBe('c');
		for (const c of ['bn-pressable', 'is-dark', 'h-7', 'w-7', 'rounded-[6px]']) expect(cls(t)).toContain(c);
	});
	it('the live dot is a round LED', () => {
		const { container } = render(Sidebar, { pathname: '/os', live: { up: 3, total: 4 } });
		expect(container.querySelector('.bn-dot.ok.pulse')).toBeTruthy();
	});
	it('keeps the bridge extras: traffic-light strip, workspace block, BusinessOS group, Brain', () => {
		const { container } = render(Sidebar, { pathname: '/os', trafficLights: true });
		expect(container.querySelector('[data-part="traffic-lights"]')).toBeTruthy();
		expect(screen.getByText('BusinessOS')).toBeTruthy();
		expect(screen.getByText('Brain')).toBeTruthy();
	});
});

describe('Topbar look', () => {
	it('theme, search and Conductor are 30px icon buttons on the control lens, then the emblem', () => {
		const { container } = render(Topbar, { pathname: '/os' });
		const theme = screen.getByRole('button', { name: 'Choose a theme' });
		const search = screen.getByRole('button', { name: /command palette/i });
		const agent = screen.getByRole('button', { name: /open the conductor/i });
		// ThemeToggle.tsx is a plain pressable (no data-lens), the other two ride the control lens
		for (const b of [search, agent]) expect(b.getAttribute('data-lens')).toBe('c');
		for (const b of [theme, search, agent]) {
			for (const c of ['bn-pressable', 'h-[30px]', 'w-[30px]', 'border']) expect(cls(b)).toContain(c);
		}
		expect(cls(theme)).toContain('rounded-[5px]');
		expect(cls(search)).toContain('rounded-[6px]');
		expect(cls(agent)).toContain('rounded-[6px]');
		expect(theme.textContent!.trim()).toBe('');
		const mark = container.querySelector('img[alt="Founder OS"]')!;
		expect(mark.getAttribute('width')).toBe('26');
		expect(agent.compareDocumentPosition(mark) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});
});

describe('CommandPalette look', () => {
	async function openPalette() {
		render(CommandPalette, { navigate: vi.fn() });
		await fireEvent.keyDown(window, { key: 'k', metaKey: true });
		await tick();
		return screen.getByRole('dialog');
	}
	it('a 12px floating panel that glides in over a fading scrim', async () => {
		const dlg = await openPalette();
		expect(cls(dlg)).toContain('rounded-[12px]');
		expect(cls(dlg)).toContain('bn-palette');
		expect(cls(dlg.parentElement)).toContain('bn-enter');
	});
	it('scopes are round toggle chips; the chosen one is solid', async () => {
		await openPalette();
		const tabs = screen.getAllByRole('tab');
		expect(tabs.map((t) => t.textContent!.trim())).toEqual(['All', 'Go to', 'Run', 'Ask']);
		for (const t of tabs) expect(cls(t)).toContain('rounded-full');
		expect(tabs[0].getAttribute('data-on')).toBe('true');
		expect(tabs[1].getAttribute('data-on')).toBe('false');
	});
	it('rows are 6px row lenses that enter; glyph tiles have 5px corners', async () => {
		await openPalette();
		const row = screen.getAllByRole('option')[0];
		expect(row.getAttribute('data-lens')).toBe('r');
		for (const c of ['bn-pressable', 'is-row', 'bn-enter', 'rounded-[6px]']) expect(cls(row)).toContain(c);
		expect(cls(row.querySelector('.bn-glyph'))).toContain('rounded-[5px]');
	});
	it('the footer says the Conductor is listening, on a blinking LED', async () => {
		await openPalette();
		expect(screen.getByText('Conductor listening')).toBeTruthy();
		expect(document.querySelector('[data-part="listening"] .bn-blink')).toBeTruthy();
		expect(document.querySelectorAll('.bn-kbd-sm').length).toBeGreaterThan(0);
	});
});

describe('ConductorPanel look', () => {
	it('the corner agent is a round pill with the Vantage spark', () => {
		render(ConductorPanel, { pathname: '/os' });
		const corner = screen.getByRole('button', { name: /open the conductor agent panel/i });
		for (const c of ['bn-pressable', 'rounded-full']) expect(cls(corner)).toContain(c);
		expect(corner.querySelector('[aria-label="Vantage"]')).toBeTruthy();
	});
	it('the edge control is round; header controls, quick actions and the model chip are 6px; the composer 10px', async () => {
		render(ConductorPanel, { pathname: '/os' });
		window.dispatchEvent(new CustomEvent(CONDUCTOR_OPEN_EVENT));
		await tick();
		await waitFor(() => expect(screen.getByText('Explain')).toBeTruthy());
		const edge = screen.getByRole('button', { name: /slide the panel away/i });
		for (const c of ['bn-pressable', 'is-dark', 'rounded-full']) expect(cls(edge)).toContain(c);
		for (const name of [/clear the transcript/i, /close conductor/i]) {
			const b = screen.getByRole('button', { name });
			for (const c of ['bn-pressable', 'is-dark', 'rounded-[6px]']) expect(cls(b)).toContain(c);
		}
		const qa = screen.getByRole('button', { name: 'Explain' });
		for (const c of ['bn-pressable', 'is-dark', 'rounded-[6px]']) expect(cls(qa)).toContain(c);
		expect(cls(document.querySelector('[data-part="model-chip"]'))).toContain('rounded-[6px]');
		expect(cls(document.querySelector('.bn-composer'))).toContain('rounded-[10px]');
		expect(cls(screen.getByRole('button', { name: 'Send' }))).toContain('rounded-full');
	});
});
