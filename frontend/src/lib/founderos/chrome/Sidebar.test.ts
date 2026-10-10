// The Sidebar's resize / hide / peek (FounderOS v1 components/Sidebar.tsx, pinned
// by tests/sidebar-layout.test.ts and tests/sidebar-collapse.test.ts). The
// grouped nav, the rail toggle and the live count are in chrome.test.ts.
import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it } from 'vitest';
import Sidebar from './Sidebar.svelte';
import { SIDEBAR, SIDEBAR_KEYS } from './sidebar-layout';

const aside = () => document.querySelector('aside') as HTMLElement;
const shellW = () => document.documentElement.style.getPropertyValue('--bn-sidebar-w');
const handle = () => screen.getByRole('separator', { name: /resize sidebar/i });

async function dragEdgeTo(x: number) {
	await fireEvent.mouseDown(handle(), { clientX: aside().offsetWidth });
	await fireEvent.mouseMove(window, { clientX: x });
	await fireEvent.mouseUp(window, { clientX: x });
	await tick();
}

beforeEach(() => {
	localStorage.clear();
	document.documentElement.style.removeProperty('--bn-sidebar-w');
});

describe('Sidebar: resize', () => {
	it('dragging the right edge docks it at that width, and the page follows', async () => {
		render(Sidebar, { pathname: '/os' });
		await tick();
		expect(shellW()).toBe(`${SIDEBAR.DEFAULT_W}px`);
		await dragEdgeTo(300);
		expect(aside().style.width).toBe('300px');
		expect(shellW()).toBe('300px');
		expect(localStorage.getItem(SIDEBAR_KEYS.width)).toBe('300');
	});

	it('the width is clamped and double-click resets it', async () => {
		render(Sidebar, { pathname: '/os' });
		await dragEdgeTo(2000);
		expect(aside().style.width).toBe(`${SIDEBAR.MAX_W}px`);
		await fireEvent.dblClick(handle());
		expect(aside().style.width).toBe(`${SIDEBAR.DEFAULT_W}px`);
	});

	it('restores a saved width', async () => {
		localStorage.setItem(SIDEBAR_KEYS.width, '310');
		render(Sidebar, { pathname: '/os' });
		await tick();
		expect(aside().style.width).toBe('310px');
		expect(shellW()).toBe('310px');
	});

	it('the rail has no resize handle', async () => {
		render(Sidebar, { pathname: '/os' });
		await fireEvent.click(screen.getByRole('button', { name: /collapse sidebar/i }));
		expect(screen.queryByRole('separator', { name: /resize sidebar/i })).toBeNull();
	});
});

describe('Sidebar: hide and peek', () => {
	it('dragging past the hide threshold hides it completely; the page takes the viewport', async () => {
		render(Sidebar, { pathname: '/os' });
		await dragEdgeTo(SIDEBAR.HIDE_AT - 20);
		expect(shellW()).toBe('0px');
		expect(aside().getAttribute('data-hidden')).toBe('true');
		expect(aside().style.visibility).toBe('hidden');
		expect(screen.getByRole('button', { name: /reveal sidebar/i })).toBeTruthy();
		expect(localStorage.getItem(SIDEBAR_KEYS.hidden)).toBe('1');
		// the width it passed through on the way is not kept
		expect(aside().style.width).toBe(`${SIDEBAR.DEFAULT_W}px`);
	});

	it('the far-left edge reveals it as an overlay that leaves when the mouse is clear', async () => {
		localStorage.setItem(SIDEBAR_KEYS.hidden, '1');
		render(Sidebar, { pathname: '/os' });
		await tick();
		expect(aside().style.visibility).toBe('hidden');
		await fireEvent.mouseMove(window, { clientX: 4 });
		expect(aside().style.visibility).toBe('');
		expect(aside().getAttribute('data-peek')).toBe('true');
		expect(shellW()).toBe('0px');
		await fireEvent.mouseMove(window, { clientX: SIDEBAR.DEFAULT_W + 10 });
		expect(aside().getAttribute('data-peek')).toBe('true');
		await fireEvent.mouseMove(window, { clientX: SIDEBAR.DEFAULT_W + SIDEBAR.PEEK_EXIT_SLOP + 5 });
		expect(aside().style.visibility).toBe('hidden');
	});

	it('touching the reveal strip peeks too', async () => {
		localStorage.setItem(SIDEBAR_KEYS.hidden, '1');
		render(Sidebar, { pathname: '/os' });
		await tick();
		await fireEvent.mouseEnter(screen.getByRole('button', { name: /reveal sidebar/i }));
		expect(aside().getAttribute('data-peek')).toBe('true');
	});

	it('the toggle on a peeking overlay docks it, expanded', async () => {
		localStorage.setItem(SIDEBAR_KEYS.hidden, '1');
		render(Sidebar, { pathname: '/os' });
		await tick();
		await fireEvent.mouseMove(window, { clientX: 2 });
		await fireEvent.click(screen.getByRole('button', { name: /collapse sidebar/i }));
		expect(aside().getAttribute('data-hidden')).toBe('false');
		expect(shellW()).toBe(`${SIDEBAR.DEFAULT_W}px`);
		expect(localStorage.getItem(SIDEBAR_KEYS.hidden)).toBe('0');
	});

	it('⌘\\ and Ctrl+\\ hide and dock it from the keyboard', async () => {
		render(Sidebar, { pathname: '/os' });
		await tick();
		await fireEvent.keyDown(window, { key: '\\', metaKey: true });
		expect(shellW()).toBe('0px');
		await fireEvent.keyDown(window, { key: '\\', ctrlKey: true });
		expect(shellW()).toBe(`${SIDEBAR.DEFAULT_W}px`);
	});

	it('dragging out from the hidden edge docks it again', async () => {
		localStorage.setItem(SIDEBAR_KEYS.hidden, '1');
		render(Sidebar, { pathname: '/os' });
		await tick();
		await fireEvent.mouseDown(screen.getByRole('button', { name: /reveal sidebar/i }), { clientX: 2 });
		await fireEvent.mouseMove(window, { clientX: 260 });
		await fireEvent.mouseUp(window, { clientX: 260 });
		expect(aside().getAttribute('data-hidden')).toBe('false');
		expect(shellW()).toBe('260px');
	});
});
