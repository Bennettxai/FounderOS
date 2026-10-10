// The Conductor dock's pure half (FounderOS v1 ConductorPanel.tsx, lib/composer.ts,
// ContextBar.tsx), pinned the way tests/conductor-mock-1h.test.ts,
// tests/composer.test.ts and tests/context-bar.test.ts pin the TS.
import { describe, expect, it, vi } from 'vitest';
import { FounderosApiError } from '../api';
import {
	COMPOSER_MAX_PX,
	COMPOSER_MIN_PX,
	CONDUCTOR_DEFAULT_W,
	CONDUCTOR_MAX_W,
	CONDUCTOR_MIN_W,
	CONDUCTOR_OPEN_EVENT,
	FALLBACK_QUICK_ACTIONS,
	REFUSED_NOTICE,
	clampConductorW,
	composerHeight,
	dispatchTurn,
	failureMessage,
	mergeTurns,
	openConductor,
	threadToTurns,
	uiRequest,
	withScreen
} from './conductor';

describe('dock width', () => {
	it('defaults to 380px and clamps a drag to 300–760', () => {
		expect(CONDUCTOR_DEFAULT_W).toBe(380);
		expect(clampConductorW(120)).toBe(CONDUCTOR_MIN_W);
		expect(clampConductorW(5000)).toBe(CONDUCTOR_MAX_W);
		expect(clampConductorW(412.6)).toBe(413);
	});

	it('a stored width that is not a number falls back to the default', () => {
		expect(clampConductorW(Number.NaN)).toBe(CONDUCTOR_DEFAULT_W);
	});
});

describe('open signal', () => {
	it('openConductor fires the window event the dock listens for', () => {
		const on = vi.fn();
		window.addEventListener(CONDUCTOR_OPEN_EVENT, on);
		openConductor();
		expect(on).toHaveBeenCalledTimes(1);
		window.removeEventListener(CONDUCTOR_OPEN_EVENT, on);
	});
});

describe('thread → turns', () => {
	const messages = [
		{ id: 'a', body: 'hi', authorType: 'user', createdAt: '2026-09-30T10:00:00Z' },
		{ id: 'b', body: 'hello', authorType: 'agent', createdAt: '2026-09-30T10:00:05Z' }
	];

	it('agent comments are the assistant, routed from the board', () => {
		expect(threadToTurns(messages, 0)).toEqual([
			{ id: 'a', role: 'user', content: 'hi', createdAt: '2026-09-30T10:00:00Z', routedTo: undefined },
			{ id: 'b', role: 'assistant', content: 'hello', createdAt: '2026-09-30T10:00:05Z', routedTo: 'Conductor · board' }
		]);
	});

	it('a local clear drops everything at or before the clear, so the poll cannot refill it', () => {
		const clearedAt = Date.parse('2026-09-30T10:00:02Z');
		expect(threadToTurns(messages, clearedAt).map((t) => t.id)).toEqual(['b']);
	});
});

describe('mergeTurns', () => {
	it('keeps a local receipt in time order across a board refresh', () => {
		const board = [
			{ id: 'a', role: 'user' as const, content: 'x', createdAt: '2026-09-30T10:00:00Z' },
			{ id: 'c', role: 'assistant' as const, content: 'z', createdAt: '2026-09-30T10:00:09Z' }
		];
		const local = [dispatchTurn('r', { workspaceId: 'w', branch: 'b' }, Date.parse('2026-09-30T10:00:05Z'))];
		expect(mergeTurns(board, local).map((t) => t.id)).toEqual(['a', local[0].id, 'c']);
	});

	it('an untimed turn stays at the end, in insertion order', () => {
		const t = (id: string) => ({ id, role: 'user' as const, content: id });
		expect(mergeTurns([t('p'), t('q')], []).map((x) => x.id)).toEqual(['p', 'q']);
	});
});

describe('outgoing messages', () => {
	it('the screen rides along with the message', () => {
		expect(withScreen('why is this red?', { title: 'Trading' })).toBe('why is this red?\n\n(the operator is looking at: Trading)');
		expect(withScreen('plain', null)).toBe('plain');
	});

	it('/ui <request> is a dispatch, anything else is chat', () => {
		expect(uiRequest('/ui make the header taller')).toBe('make the header taller');
		expect(uiRequest('/UI  tidy up ')).toBe('tidy up');
		expect(uiRequest('/uimake')).toBeNull();
		expect(uiRequest('what is /ui')).toBeNull();
	});

	it('a dispatch files a receipt turn with the branch and the workspace', () => {
		const t = dispatchTurn('make it blue', { workspaceId: 'ws-1', branch: 'ui/make-it-blue' }, 7);
		expect(t.id).toBe('dispatch-7');
		expect(t.role).toBe('assistant');
		expect(t.content).toContain('"make it blue"');
		expect(t.content).toContain('Branch: ui/make-it-blue');
		expect(t.content).toContain('Workspace: ws-1');
		expect(t.routedTo).toBe('Superset · coding agent');
		expect(t.receipt).toEqual({ text: 'Workspace opened on an isolated branch', href: '/os/agents' });
	});
});

describe('failures read honestly', () => {
	it('a 409 refusal says writes are off', () => {
		const err = new FounderosApiError(409, 'bridge writes are disabled', { error: 'bridge writes are disabled', refused: true });
		expect(failureMessage(err)).toBe(REFUSED_NOTICE);
		expect(REFUSED_NOTICE).toMatch(/writes are off/i);
		expect(REFUSED_NOTICE).not.toMatch(/bridge/i);
	});

	it('a 409 that is not a writes refusal keeps its own reason (Paperclip not configured)', () => {
		const err = new FounderosApiError(409, 'Paperclip is not configured, nothing was sent.', {
			error: 'Paperclip is not configured, nothing was sent.',
			notConfigured: true
		});
		expect(failureMessage(err)).toBe('Paperclip is not configured, nothing was sent. (HTTP 409)');
	});

	it('any other API error keeps the server reason', () => {
		expect(failureMessage(new FounderosApiError(502, 'paperclip: dial tcp refused'))).toBe('paperclip: dial tcp refused (HTTP 502)');
		expect(failureMessage(new Error('offline'))).toBe('offline');
		expect(failureMessage('weird')).toBe('weird');
	});
});

describe('quick actions fallback', () => {
	it('three openers until the context lands, each a real prompt', () => {
		expect(FALLBACK_QUICK_ACTIONS).toHaveLength(3);
		for (const a of FALLBACK_QUICK_ACTIONS) expect(a.prompt.length).toBeGreaterThan(a.label.length);
	});
});

// FounderOS v1 tests/composer.test.ts
describe('composerHeight', () => {
	it('grows to fit what the browser laid out, up to the cap', () => {
		expect(composerHeight(64)).toBe(64);
		expect(composerHeight(9000)).toBe(COMPOSER_MAX_PX);
	});

	it('never collapses below one line, even on a junk measurement', () => {
		expect(composerHeight(0)).toBe(COMPOSER_MIN_PX);
		expect(composerHeight(5)).toBe(COMPOSER_MIN_PX);
		expect(composerHeight(Number.NaN)).toBe(COMPOSER_MIN_PX);
		expect(composerHeight(-40)).toBe(COMPOSER_MIN_PX);
	});

	it('the cap leaves room for a real paragraph', () => {
		expect(COMPOSER_MAX_PX / 20).toBeGreaterThanOrEqual(8);
	});
});
