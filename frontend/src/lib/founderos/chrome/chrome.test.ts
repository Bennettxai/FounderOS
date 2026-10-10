import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

// The palette loads the agent roster on open; no network in these tests.
vi.mock('../api', async (orig) => ({
	...(await orig<typeof import('../api')>()),
	founderosFetch: vi.fn(async () => ({ agents: [] }))
}));

import CommandPalette from './CommandPalette.svelte';
import Sidebar from './Sidebar.svelte';
import Topbar from './Topbar.svelte';
import { breadcrumbFor, isOperatorView, PALETTE_EVENT, RAIL_KEY } from './chrome';
import { CONDUCTOR_OPEN_EVENT } from './conductor';
import { STORAGE_KEY as THEME_KEY } from '../theme/theme';

beforeEach(() => localStorage.clear());

describe('Sidebar', () => {
	it('renders the five FounderOS group headings, then BusinessOS, every item a link in nav order', () => {
		const { container } = render(Sidebar, { pathname: '/os' });
		const headings = [...container.querySelectorAll('[data-part="group-title"]')].map((h) => h.textContent?.trim());
		expect(headings).toEqual(['Operate', 'Agents', 'Intelligence', 'System', 'Variants', 'BusinessOS']);
		const links = [...container.querySelectorAll('nav a')].map((a) => a.getAttribute('href'));
		expect(links[0]).toBe('/os');
		expect(links).toContain('/os/brain');
		expect(links).toHaveLength(24 + 8);
		expect(screen.getByText('Brain')).toBeTruthy();
		expect(screen.queryByText('G-Brain')).toBeNull();
	});

	it('marks the active view: Home only on /os, others on their sub-routes too', () => {
		const { container } = render(Sidebar, { pathname: '/os/social/instagram' });
		const active = [...container.querySelectorAll('nav a[aria-current="page"]')];
		expect(active.map((a) => a.getAttribute('href'))).toEqual(['/os/social']);
		expect(active[0].getAttribute('data-active')).toBe('true');
	});

	it('shows the live connection count, and an honest dash while unknown', () => {
		render(Sidebar, { pathname: '/os', live: { up: 17, total: 24 } });
		expect(screen.getByText(/17\/24 systems live/)).toBeTruthy();
	});

	it('unknown live count reads —/—, never 0/0', () => {
		render(Sidebar, { pathname: '/os', live: null });
		expect(screen.getByText(/—\/— systems live/)).toBeTruthy();
	});

	it('collapses to an icon rail and remembers it', async () => {
		render(Sidebar, { pathname: '/os' });
		const toggle = screen.getByRole('button', { name: /collapse sidebar/i });
		expect(toggle.getAttribute('aria-expanded')).toBe('true');
		await fireEvent.click(toggle);
		expect(screen.getByRole('button', { name: /expand sidebar/i }).getAttribute('aria-expanded')).toBe('false');
		expect(screen.queryByText('Comms')).toBeNull();
		expect(localStorage.getItem(RAIL_KEY)).toBe('1');
	});

	// FounderOS v1 Sidebar.tsx: the nav scrolls (and clips), so a collapsed
	// item's label floats as a FIXED chip beside the rail, not a native title.
	it('on the rail, hovering an item floats its label beside the rail; leaving or expanding clears it', async () => {
		const { container } = render(Sidebar, { pathname: '/os' });
		await fireEvent.click(screen.getByRole('button', { name: /collapse sidebar/i }));
		const link = container.querySelector('a[href="/os/funnel"]') as HTMLElement;
		expect(link.getAttribute('title')).toBeNull();
		link.getBoundingClientRect = () => ({ top: 193, left: 0, right: 0, bottom: 0, width: 0, height: 0, x: 0, y: 193, toJSON() {} }) as DOMRect;
		await fireEvent.mouseEnter(link);
		const tip = container.querySelector('[data-part="rail-tip"]') as HTMLElement;
		expect(tip.textContent).toBe('Funnel');
		expect(tip.className).toMatch(/pointer-events-none fixed z-50 -translate-y-1\/2 whitespace-nowrap/);
		expect(tip.className).toMatch(/uppercase tracking-\[0\.14em\]/);
		expect(tip.style.left).toBe('64px');
		expect(tip.style.top).toBe('208px');
		await fireEvent.mouseLeave(link);
		expect(container.querySelector('[data-part="rail-tip"]')).toBeNull();
		await fireEvent.mouseEnter(link);
		await fireEvent.click(screen.getByRole('button', { name: /expand sidebar/i }));
		expect(container.querySelector('[data-part="rail-tip"]')).toBeNull();
	});

	it('expanded, hovering an item floats nothing', async () => {
		const { container } = render(Sidebar, { pathname: '/os' });
		await fireEvent.mouseEnter(container.querySelector('a[href="/os/funnel"]')!);
		expect(container.querySelector('[data-part="rail-tip"]')).toBeNull();
	});

	it('restores a saved rail', async () => {
		localStorage.setItem(RAIL_KEY, '1');
		render(Sidebar, { pathname: '/os' });
		await tick();
		expect(screen.getByRole('button', { name: /expand sidebar/i })).toBeTruthy();
	});
});

describe('Topbar', () => {
	it('breadcrumb: founder-os / <view>', () => {
		const { container } = render(Topbar, { pathname: '/os/integrations' });
		expect(container.querySelector('[data-part="crumb"]')!.textContent).toMatch(/founder-os\s*\/\s*connections/);
	});

	it('breadcrumb labels follow FounderOS v1, with Brain for the knowledge view', () => {
		expect(breadcrumbFor('/os')).toBe('home');
		expect(breadcrumbFor('/os/org')).toBe('org-chart');
		expect(breadcrumbFor('/os/brain')).toBe('brain');
		expect(breadcrumbFor('/os/integrations')).toBe('connections');
		expect(breadcrumbFor('/os/brand-deals/acme')).toBe('brand-deals');
	});

	it('the search button carries the ⌘K hint in its title (icon-only, as in FounderOS v1) and opens the palette', async () => {
		render(Topbar, { pathname: '/os' });
		const onOpen = vi.fn();
		window.addEventListener(PALETTE_EVENT, onOpen);
		const btn = screen.getByRole('button', { name: /command palette/i });
		expect(btn.getAttribute('title')).toBe('Command palette (⌘K)');
		expect(btn.textContent!.trim()).toBe('');
		await fireEvent.click(btn);
		expect(onOpen).toHaveBeenCalledTimes(1);
		window.removeEventListener(PALETTE_EVENT, onOpen);
	});
});

describe('Topbar: Conductor button and theme toggle (spec 6.20)', () => {
	beforeEach(() => document.documentElement.setAttribute('data-founderos-theme', 'mono'));

	it('the agent button opens the Conductor dock', async () => {
		render(Topbar, { pathname: '/os' });
		const onOpen = vi.fn();
		window.addEventListener(CONDUCTOR_OPEN_EVENT, onOpen);
		await fireEvent.click(screen.getByRole('button', { name: /open the conductor/i }));
		expect(onOpen).toHaveBeenCalledTimes(1);
		window.removeEventListener(CONDUCTOR_OPEN_EVENT, onOpen);
	});

	it('the Conductor button sits to the right of ⌘K, as in FounderOS v1', () => {
		const { container } = render(Topbar, { pathname: '/os' });
		const labels = [...container.querySelectorAll('button')].map((b) => b.getAttribute('aria-label') ?? '');
		expect(labels.findIndex((l) => /conductor/i.test(l))).toBeGreaterThan(labels.findIndex((l) => /command palette/i.test(l)));
	});

	// FounderOS v1 components/ThemeToggle.tsx: a palette chip that opens the
	// theme menu, every skin with its swatch trio, name and one-line feel.
	it('the palette chip opens the theme menu listing every skin, the active one checked', async () => {
		render(Topbar, { pathname: '/os' });
		const chip = screen.getByRole('button', { name: 'Choose a theme' });
		expect(chip.getAttribute('title')).toBe('Choose a theme');
		expect(chip.getAttribute('aria-expanded')).toBe('false');
		expect(screen.queryByRole('menu')).toBeNull();
		await fireEvent.click(chip);
		expect(chip.getAttribute('aria-expanded')).toBe('true');
		const menu = screen.getByRole('menu');
		expect(menu.querySelector('[data-part="menu-title"]')!.textContent).toBe('Theme');
		const items = screen.getAllByRole('menuitemradio');
		expect(items.map((i) => i.querySelector('[data-part="name"]')!.textContent)).toEqual(['Monolith', 'Daylight', 'Terminal', 'Clay', 'Midnight', 'Ember']);
		expect(items[0].getAttribute('aria-checked')).toBe('true');
		expect(items[0].querySelectorAll('[data-part="swatch"] > span')).toHaveLength(3);
		expect(items[3].textContent).toContain('warm paper with clay orange');
	});

	it('picking a skin applies it, remembers it and closes the menu', async () => {
		render(Topbar, { pathname: '/os' });
		await fireEvent.click(screen.getByRole('button', { name: 'Choose a theme' }));
		await fireEvent.click(screen.getByRole('menuitemradio', { name: /Midnight/ }));
		expect(document.documentElement.getAttribute('data-founderos-theme')).toBe('midnight');
		expect(localStorage.getItem(THEME_KEY)).toBe('midnight');
		expect(screen.queryByRole('menu')).toBeNull();
		await fireEvent.click(screen.getByRole('button', { name: 'Choose a theme' }));
		expect(screen.getByRole('menuitemradio', { name: /Midnight/ }).getAttribute('aria-checked')).toBe('true');
	});

	it('Escape or a click outside closes the menu', async () => {
		render(Topbar, { pathname: '/os' });
		const chip = screen.getByRole('button', { name: 'Choose a theme' });
		await fireEvent.click(chip);
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(screen.queryByRole('menu')).toBeNull();
		await fireEvent.click(chip);
		await fireEvent.pointerDown(document.body);
		expect(screen.queryByRole('menu')).toBeNull();
	});

	it('starts from the theme already on the page', async () => {
		document.documentElement.setAttribute('data-founderos-theme', 'terminal');
		render(Topbar, { pathname: '/os' });
		await tick();
		await fireEvent.click(screen.getByRole('button', { name: 'Choose a theme' }));
		expect(screen.getByRole('menuitemradio', { name: /Terminal/ }).getAttribute('aria-checked')).toBe('true');
	});
});

describe('CommandPalette', () => {
	const open = async () => {
		await fireEvent.keyDown(window, { key: 'k', metaKey: true });
		await tick();
	};

	it('is closed until ⌘K, then lists every view', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		expect(screen.queryByRole('dialog')).toBeNull();
		await open();
		expect(screen.getByRole('dialog')).toBeTruthy();
		await fireEvent.click(screen.getByRole('tab', { name: 'Go to' }));
		expect(screen.getAllByRole('option')).toHaveLength(24);
		expect(screen.getByText('24 results')).toBeTruthy();
	});

	it('Ctrl+K works too, and ⌘K again closes it', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await fireEvent.keyDown(window, { key: 'K', ctrlKey: true });
		expect(screen.getByRole('dialog')).toBeTruthy();
		await fireEvent.keyDown(window, { key: 'k', metaKey: true });
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('the Topbar event opens it', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		window.dispatchEvent(new CustomEvent(PALETTE_EVENT));
		await tick();
		expect(screen.getByRole('dialog')).toBeTruthy();
	});

	it('filters as you type and Enter navigates to the selection, then closes', async () => {
		const navigate = vi.fn();
		render(CommandPalette, { navigate });
		await open();
		const input = screen.getByRole('combobox');
		await fireEvent.input(input, { target: { value: 'robinhood' } });
		expect(screen.getAllByRole('option').map((o) => o.textContent)).toEqual([expect.stringContaining('Trading')]);
		await fireEvent.keyDown(input, { key: 'Enter' });
		expect(navigate).toHaveBeenCalledWith('/os/trading');
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('arrow keys move the selection', async () => {
		const navigate = vi.fn();
		render(CommandPalette, { navigate });
		await open();
		const input = screen.getByRole('combobox');
		await fireEvent.keyDown(input, { key: 'ArrowDown' });
		await fireEvent.keyDown(input, { key: 'ArrowDown' });
		await fireEvent.keyDown(input, { key: 'ArrowUp' });
		expect(screen.getAllByRole('option')[1].getAttribute('aria-selected')).toBe('true');
		await fireEvent.keyDown(input, { key: 'Enter' });
		expect(navigate).toHaveBeenCalledWith('/os/comms');
	});

	it('clicking a row navigates', async () => {
		const navigate = vi.fn();
		render(CommandPalette, { navigate });
		await open();
		await fireEvent.click(screen.getByText('Connections'));
		expect(navigate).toHaveBeenCalledWith('/os/integrations');
	});

	it('no match says so', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.input(screen.getByRole('combobox'), { target: { value: 'zzzqqq' } });
		expect(screen.getByText(/No match for «zzzqqq»/)).toBeTruthy();
	});

	it('Escape closes', async () => {
		render(CommandPalette, { navigate: vi.fn() });
		await open();
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('digits 1–9 jump views while closed', async () => {
		const navigate = vi.fn();
		render(CommandPalette, { navigate });
		await fireEvent.keyDown(window, { key: '1' });
		await fireEvent.keyDown(window, { key: '9' });
		expect(navigate.mock.calls).toEqual([['/os'], ['/os/trading']]);
	});

	it('digits do nothing with a modifier, while typing, or while the palette is open', async () => {
		const navigate = vi.fn();
		render(CommandPalette, { navigate });
		await fireEvent.keyDown(window, { key: '2', metaKey: true });
		const field = document.createElement('input');
		document.body.appendChild(field);
		field.focus();
		await fireEvent.keyDown(field, { key: '2' });
		field.remove();
		await open();
		await fireEvent.keyDown(window, { key: '2' });
		expect(navigate).not.toHaveBeenCalled();
	});
});

describe('isOperatorView', () => {
	it('is every /os view and nothing else', () => {
		expect(isOperatorView('/os')).toBe(true);
		expect(isOperatorView('/os/comms')).toBe(true);
		expect(isOperatorView('/login')).toBe(false);
		expect(isOperatorView('/osaka')).toBe(false);
	});
});
