// The operator's own sidebar keeps his groups and adds what BusinessOS still owns:
// the workspace switcher on top and a BusinessOS group of primitives below.
import { render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { describe, expect, it } from 'vitest';
import Sidebar from './Sidebar.svelte';

const switcher = createRawSnippet(() => ({ render: () => '<div data-testid="ws-switcher">ws</div>' }));

describe('Sidebar: BusinessOS primitives', () => {
	it('renders the workspace switcher it is given', () => {
		render(Sidebar, { pathname: '/os', workspace: switcher });
		expect(screen.getByTestId('ws-switcher')).toBeTruthy();
	});

	it('lists the BusinessOS primitives after FounderOS v1 groups', () => {
		render(Sidebar, { pathname: '/os' });
		const titles = [...document.querySelectorAll('[data-part="group-title"]')].map((e) => e.textContent?.trim());
		expect(titles.at(-1)).toBe('BusinessOS');
		const links = ['Command', 'Knowledge', 'Inbox', 'Calendar', 'Communications', 'Projects', 'Desktop', 'Settings'];
		const hrefs = ['/dashboard', '/knowledge', '/inbox', '/calendar', '/communication', '/projects', '/window', '/settings'];
		links.forEach((label, i) => {
			expect(screen.getByRole('link', { name: label }).getAttribute('href')).toBe(hrefs[i]);
		});
	});
});

describe('Sidebar: macOS window controls in Electron', () => {
	it('reserves a draggable strip above the header for the traffic lights', () => {
		render(Sidebar, { pathname: '/os', trafficLights: true });
		const strip = document.querySelector('[data-part="traffic-lights"]') as HTMLElement;
		expect(strip).toBeTruthy();
		expect(strip.getAttribute('style')).toContain('-webkit-app-region: drag');
	});

	it('has no strip in a browser', () => {
		render(Sidebar, { pathname: '/os' });
		expect(document.querySelector('[data-part="traffic-lights"]')).toBeNull();
	});
});

describe('Sidebar: workspace switcher colours', () => {
	// WorkspaceSwitcher reads the BusinessOS shell's --dt/--dt2/--dt3 text
	// tokens, which /os does not define: unmapped they fall back to #111, black
	// on the operator's black sidebar.
	it('maps the switcher text tokens onto the operator text colours', async () => {
		const { readFileSync } = await import('node:fs');
		const { resolve } = await import('node:path');
		const src = readFileSync(resolve(__dirname, 'Sidebar.svelte'), 'utf8');
		const ws = src.match(/\.bn-ws \{([^}]*)\}/)?.[1] ?? '';
		expect(ws).toMatch(/--dt:\s*var\(--bn-text\)/);
		expect(ws).toMatch(/--dt2:\s*var\(--bn-text-2\)/);
		expect(ws).toMatch(/--dt3:\s*var\(--bn-text-3\)/);
	});
});

describe('Sidebar: follows the workspace', () => {
	it('shows only the the operator pages switched on for the workspace, and drops empty groups', () => {
		const { container } = render(Sidebar, { pathname: '/os', visibleHrefs: new Set(['/os', '/os/trading']) });
		const founderos = [...container.querySelectorAll('nav a')].map((a) => a.getAttribute('href')).filter((h) => h?.startsWith('/os'));
		expect(founderos).toEqual(['/os', '/os/trading']);
		const titles = [...container.querySelectorAll('[data-part="group-title"]')].map((e) => e.textContent?.trim());
		expect(titles).toEqual(['Operate', 'BusinessOS']);
	});
});
