// The Conductor dock (FounderOS v1 components/ConductorPanel.tsx), driven through
// a mocked founderosFetch. Mirrors what tests/conductor-mock-1h.test.ts and
// tests/conductor-model-label.test.ts pin in the TS: 380px dock that pushes the
// page, per-screen quick actions in their own band, clear, honest model chip,
// receipt card on a dispatch, the synthesizing shimmer, and the 150s notice.
import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('../api', async (orig) => ({ ...(await orig<typeof import('../api')>()), founderosFetch: vi.fn() }));

import { FounderosApiError, founderosFetch } from '../api';
import ConductorPanel from './ConductorPanel.svelte';
import { CONDUCTOR_OPEN_EVENT, CONDUCTOR_WIDTH_KEY, REFUSED_NOTICE, SLOW_MESSAGE } from './conductor';

const fetchMock = vi.mocked(founderosFetch);

type Msg = { id: string; body: string; authorType: string; createdAt: string };
let thread: Msg[];
let ctxBody: unknown;
let post: (path: string, json: unknown) => unknown;

const CTX = {
	title: 'Trading',
	context: 'Robinhood + Phantom board.\nsecond line',
	quickActions: [
		{ label: 'Explain the sleeve', prompt: 'Explain the agentic sleeve on this screen.' },
		{ label: 'Open orders?', prompt: 'What orders are open right now and why?' }
	],
	model: 'claude-haiku-4-5'
};

beforeEach(() => {
	localStorage.clear();
	document.documentElement.style.removeProperty('--bn-conductor-w');
	thread = [];
	ctxBody = CTX;
	post = () => ({ comment: { id: 'c1' } });
	fetchMock.mockReset();
	fetchMock.mockImplementation(async (path: string, init?: { method?: string; json?: unknown }) => {
		if (init?.method === 'POST') return post(path, init.json) as never;
		if (path.startsWith('/pages/conductor/context')) {
			if (ctxBody instanceof Error) throw ctxBody;
			return ctxBody as never;
		}
		if (path === '/pages/conductor/chat') return { messages: thread } as never;
		throw new Error(`unexpected ${path}`);
	});
});

afterEach(() => vi.useRealTimers());

async function openDock() {
	window.dispatchEvent(new CustomEvent(CONDUCTOR_OPEN_EVENT));
	await tick();
	await waitFor(() => expect(screen.getByText(/seeing: Trading/i)).toBeTruthy());
}

const dock = () => document.querySelector('aside[data-part="conductor"]') as HTMLElement;

describe('ConductorPanel: open, close, push', () => {
	it('wears the Conductor emblem, alive only while a reply is pending', async () => {
		let release!: () => void;
		post = () => new Promise((r) => (release = () => r({ comment: { id: 'c1' } })));
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		const emblem = () => dock().querySelector('header .conductor-emblem') as HTMLElement;
		expect(emblem()).toBeTruthy();
		expect(emblem().classList.contains('thinking')).toBe(false);
		await fireEvent.input(screen.getByRole('textbox'), { target: { value: 'hi' } });
		await fireEvent.keyDown(screen.getByRole('textbox'), { key: 'Enter' });
		await waitFor(() => expect(emblem().classList.contains('thinking')).toBe(true));
		release();
		// production: the header emblem thinks while SENDING; the wait for the
		// CEO's reply is the small emblem + synthesizing line in the transcript
		await waitFor(() => expect(screen.getByText(/synthesizing/)).toBeTruthy());
		expect(emblem().classList.contains('thinking')).toBe(false);
		expect(dock().querySelector('[data-thinking] .conductor-emblem')!.classList.contains('thinking')).toBe(true);
	});

	it('a slow context answer for the page he left never replaces the page he is on', async () => {
		let releaseTrading!: () => void;
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/conductor/context?path=%2Fos%2Ftrading')
				return new Promise((r) => (releaseTrading = () => r({ ...CTX, title: 'Trading' }))) as never;
			if (path.startsWith('/pages/conductor/context')) return { ...CTX, title: 'Funnel' } as never;
			if (path === '/pages/conductor/chat') return { messages: [] } as never;
			throw new Error(`unexpected ${path}`);
		});
		const r = render(ConductorPanel, { pathname: '/os/trading' });
		window.dispatchEvent(new CustomEvent(CONDUCTOR_OPEN_EVENT));
		await tick();
		await r.rerender({ pathname: '/os/funnel' });
		await waitFor(() => expect(screen.getByText(/seeing: Funnel/i)).toBeTruthy());
		releaseTrading();
		await new Promise((res) => setTimeout(res, 0));
		await tick();
		expect(screen.queryByText(/seeing: Trading/i)).toBeNull();
		expect(screen.getByText(/seeing: Funnel/i)).toBeTruthy();
	});

	it('starts closed with the corner button, and fetches nothing', () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		expect(dock().getAttribute('aria-hidden')).toBe('true');
		expect(screen.getByRole('button', { name: /open the conductor agent panel/i })).toBeTruthy();
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('the open event opens it and loads this screen’s context and the board thread', async () => {
		thread = [
			{ id: 'm1', body: 'how is trading?', authorType: 'user', createdAt: '2026-09-30T10:00:00Z' },
			{ id: 'm2', body: 'Flat on the day.', authorType: 'agent', createdAt: '2026-09-30T10:00:05Z' }
		];
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		expect(dock().getAttribute('aria-hidden')).toBe('false');
		expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/context?path=%2Fos%2Ftrading');
		expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/chat');
		await waitFor(() => expect(screen.getByText('Flat on the day.')).toBeTruthy());
		expect(screen.getByText('how is trading?')).toBeTruthy();
		expect(screen.getByText(/→ Conductor · board/)).toBeTruthy();
		expect(screen.getByText('Robinhood + Phantom board.')).toBeTruthy();
		expect(screen.queryByRole('button', { name: /open the conductor agent panel/i })).toBeNull();
	});

	it('the corner button opens it too', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await fireEvent.click(screen.getByRole('button', { name: /open the conductor agent panel/i }));
		await waitFor(() => expect(dock().getAttribute('aria-hidden')).toBe('false'));
	});

	it('pushes the page by publishing its width, and gives it back on close', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		const w = () => document.documentElement.style.getPropertyValue('--bn-conductor-w');
		expect(w()).toBe('0px');
		await openDock();
		expect(w()).toBe('380px');
		await fireEvent.keyDown(document, { key: 'Escape' });
		expect(dock().getAttribute('aria-hidden')).toBe('true');
		expect(w()).toBe('0px');
	});

	it('restores a stored width, clamped', async () => {
		localStorage.setItem(CONDUCTOR_WIDTH_KEY, '9999');
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		expect(dock().style.width).toBe('760px');
		expect(document.documentElement.style.getPropertyValue('--bn-conductor-w')).toBe('760px');
	});

	it('drag the left edge to resize; the width is remembered', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		const handle = screen.getByTitle('Drag to resize');
		await fireEvent.pointerDown(handle, { clientX: 1000, pointerId: 1 });
		await fireEvent.pointerMove(handle, { clientX: 900, pointerId: 1 });
		await fireEvent.pointerUp(handle, { clientX: 900, pointerId: 1 });
		expect(dock().style.width).toBe('480px');
		expect(localStorage.getItem(CONDUCTOR_WIDTH_KEY)).toBe('480');
	});

	it('the close and slide-away controls both close it', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await fireEvent.click(screen.getByRole('button', { name: /close conductor/i }));
		expect(dock().getAttribute('aria-hidden')).toBe('true');
		await openDock();
		await fireEvent.click(screen.getByRole('button', { name: /slide the panel away/i }));
		expect(dock().getAttribute('aria-hidden')).toBe('true');
	});
});

describe('ConductorPanel: header and quick actions', () => {
	it('the model chip reads the live seat', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		// the header chip and the composer chip both carry it
		expect(screen.getAllByText('claude-haiku-4-5')).toHaveLength(2);
		expect(screen.queryByText('model unknown')).toBeNull();
	});

	it('says "model unknown" when the board reports none', async () => {
		ctxBody = { ...CTX, model: null };
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		expect(screen.getAllByText('model unknown').length).toBeGreaterThan(0);
	});

	it('the quick-action band comes from the context', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		expect(screen.getByText(/quick actions · this screen/i)).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Explain the sleeve' })).toBeTruthy();
		expect(screen.queryByRole('button', { name: 'Board status' })).toBeNull();
	});

	it('without context it falls back to the three openers and the empty-state hint', async () => {
		ctxBody = new FounderosApiError(502, 'down');
		render(ConductorPanel, { pathname: '/os/trading' });
		window.dispatchEvent(new CustomEvent(CONDUCTOR_OPEN_EVENT));
		await waitFor(() => expect(screen.getByRole('button', { name: 'Board status' })).toBeTruthy());
		expect(screen.getByText(/seeing: …/)).toBeTruthy();
		expect(screen.getByText(/ask about this screen, or pick a quick action above/i)).toBeTruthy();
		// production's header has no context status bar (live prod, 2026-10-01)
		expect(document.querySelector('[data-context-bar]')).toBeNull();
	});

	it('a quick action sends its prompt with the screen attached', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await fireEvent.click(screen.getByRole('button', { name: 'Explain the sleeve' }));
		await waitFor(() =>
			expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/chat', {
				method: 'POST',
				json: { message: 'Explain the agentic sleeve on this screen.\n\n(the operator is looking at: Trading)' }
			})
		);
		// the bubble shows the label, not the long prompt
		expect(screen.getAllByText('Explain the sleeve')).toHaveLength(2);
	});
});

describe('ConductorPanel: sending', () => {
	async function type(text: string) {
		const box = screen.getByRole('textbox', { name: /message the conductor/i });
		await fireEvent.input(box, { target: { value: text } });
		await fireEvent.keyDown(box, { key: 'Enter' });
	}

	it('Enter sends: the user bubble shows at once, then the synthesizing wait', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await type('why is AAPL red?');
		expect(screen.getByText('why is AAPL red?')).toBeTruthy();
		expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/chat', {
			method: 'POST',
			json: { message: 'why is AAPL red?\n\n(the operator is looking at: Trading)' }
		});
		await waitFor(() => expect(screen.getByText(/synthesizing/)).toBeTruthy());
		// production's synth-shimmer sweep on the word
		expect(screen.getByText(/synthesizing/).classList.contains('bn-synth-shimmer')).toBe(true);
	});

	it('Shift+Enter does not send', async () => {
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		const box = screen.getByRole('textbox', { name: /message the conductor/i });
		await fireEvent.input(box, { target: { value: 'two' } });
		await fireEvent.keyDown(box, { key: 'Enter', shiftKey: true });
		expect(fetchMock).not.toHaveBeenCalledWith('/pages/conductor/chat', expect.objectContaining({ method: 'POST' }));
	});

	it('a 409 refusal says writes are off, and stops waiting', async () => {
		post = () => {
			throw new FounderosApiError(409, 'bridge writes are disabled', { error: 'bridge writes are disabled', refused: true });
		};
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await type('hello');
		await waitFor(() => expect(screen.getByText(new RegExp(REFUSED_NOTICE.replace(/[()]/g, '\\$&')))).toBeTruthy());
		expect(screen.queryByText(/synthesizing/)).toBeNull();
	});

	it('an upstream failure shows the server reason', async () => {
		post = () => {
			throw new FounderosApiError(502, 'paperclip unreachable');
		};
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await type('hello');
		await waitFor(() => expect(screen.getByText(/paperclip unreachable \(HTTP 502\)/)).toBeTruthy());
	});

	it('/ui <request> dispatches a coding agent and files a receipt turn', async () => {
		post = (path) => {
			if (path === '/pages/conductor/dispatch') return { workspaceId: 'ws-9', branch: 'ui/taller-header' };
			throw new Error('chat should not be called');
		};
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await type('/ui make the header taller');
		await waitFor(() => expect(screen.getByText(/Workspace opened on an isolated branch/)).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/conductor/dispatch', { method: 'POST', json: { request: 'make the header taller' } });
		expect(screen.getByText(/Branch: ui\/taller-header/)).toBeTruthy();
		expect(screen.getByText(/→ Superset · coding agent/)).toBeTruthy();
		expect(screen.getByRole('link', { name: /open →/ }).getAttribute('href')).toBe('/os/agents');
		expect(screen.getByText('✓')).toBeTruthy();
	});

	it('a refused dispatch says writes are off too', async () => {
		post = () => {
			throw new FounderosApiError(409, 'refused', { refused: true });
		};
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await type('/ui tidy');
		await waitFor(() => expect(screen.getByText(/writes are off/i)).toBeTruthy());
	});
});

describe('ConductorPanel: the poll', () => {
	it('picks up the reply while waiting (4s) and stops the wait', async () => {
		vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });
		vi.setSystemTime(new Date('2026-09-30T10:00:00Z'));
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		const box = screen.getByRole('textbox', { name: /message the conductor/i });
		await fireEvent.input(box, { target: { value: 'status?' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(screen.getByText(/synthesizing/)).toBeTruthy());
		thread = [
			{ id: 'u', body: 'status?', authorType: 'user', createdAt: '2026-09-30T10:00:01Z' },
			{ id: 'r', body: 'All green.', authorType: 'agent', createdAt: '2026-09-30T10:00:03Z' }
		];
		await vi.advanceTimersByTimeAsync(4000);
		await waitFor(() => expect(screen.getByText('All green.')).toBeTruthy());
		expect(screen.queryByText(/synthesizing/)).toBeNull();
	});

	it('at idle it only polls every third tick (~12s)', async () => {
		vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		const chatGets = () => fetchMock.mock.calls.filter(([p, i]) => p === '/pages/conductor/chat' && !i).length;
		const base = chatGets();
		await vi.advanceTimersByTimeAsync(8000);
		expect(chatGets()).toBe(base);
		await vi.advanceTimersByTimeAsync(4000);
		expect(chatGets()).toBe(base + 1);
	});

	it('after 150s without a reply it says the run is taking a while', async () => {
		vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });
		vi.setSystemTime(new Date('2026-09-30T10:00:00Z'));
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		const box = screen.getByRole('textbox', { name: /message the conductor/i });
		await fireEvent.input(box, { target: { value: 'slow one' } });
		await fireEvent.keyDown(box, { key: 'Enter' });
		await waitFor(() => expect(screen.getByText(/synthesizing/)).toBeTruthy());
		await vi.advanceTimersByTimeAsync(152_000);
		await waitFor(() => expect(screen.getByText(new RegExp(SLOW_MESSAGE.slice(0, 30)))).toBeTruthy());
		expect(screen.queryByText(/synthesizing/)).toBeNull();
	});

	it('clear is local and the poll does not put the old transcript back', async () => {
		vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval', 'Date'] });
		vi.setSystemTime(new Date('2026-09-30T11:00:00Z'));
		thread = [{ id: 'old', body: 'an old reply', authorType: 'agent', createdAt: '2026-09-30T10:00:00Z' }];
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await waitFor(() => expect(screen.getByText('an old reply')).toBeTruthy());
		await fireEvent.click(screen.getByRole('button', { name: /clear the transcript/i }));
		expect(screen.queryByText('an old reply')).toBeNull();
		await vi.advanceTimersByTimeAsync(12_000);
		expect(screen.queryByText('an old reply')).toBeNull();
		expect(fetchMock.mock.calls.some(([p, i]) => p === '/pages/conductor/chat' && (i as { method?: string })?.method === 'DELETE')).toBe(false);
	});

	it('a thread that cannot load says so instead of reading empty', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path.startsWith('/pages/conductor/context')) return CTX as never;
			throw new FounderosApiError(502, 'paperclip unreachable');
		});
		render(ConductorPanel, { pathname: '/os/trading' });
		await openDock();
		await waitFor(() => expect(screen.getByText(/thread unavailable: paperclip unreachable/i)).toBeTruthy());
	});
});
