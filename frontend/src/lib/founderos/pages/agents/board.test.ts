import { describe, expect, it } from 'vitest';
import {
	ago,
	boardDecisionFor,
	CORE_ISSUE_LANES,
	groupIssues,
	modelSummary,
	orderRuns,
	orderSeats,
	runDuration,
	runGlyph,
	runningSince,
	seatTone
} from './board';
import type { PaperclipAgent, PaperclipIssue, PaperclipRun } from './types';

// Ported from FounderOS v1 tests/board-live.test.ts (the presentational half).

const agent = (over: Partial<PaperclipAgent> = {}): PaperclipAgent => ({
	id: 'a',
	name: 'Seat',
	status: 'idle',
	adapterType: 'claude_local',
	model: 'glm-5.2',
	lastHeartbeatAt: null,
	...over
});
const issue = (status: string, id = status): PaperclipIssue => ({ id, identifier: 'FOS-1', title: id, status, assigneeName: null, updatedAt: null });
const run = (over: Partial<PaperclipRun> = {}): PaperclipRun => ({
	id: 'r',
	agentId: 'a',
	agentName: null,
	status: 'succeeded',
	startedAt: '2026-08-07T05:00:00Z',
	finishedAt: '2026-08-07T05:00:14Z',
	...over
});

describe('modelSummary', () => {
	it('groups by model, most common first', () => {
		expect(modelSummary([agent(), agent(), agent({ model: 'claude-local' })])).toBe('glm-5.2 x2 · claude-local x1');
	});
	it('a seat without a model shows its adapter type, never disappears', () => {
		expect(modelSummary([agent({ model: null, adapterType: 'codex' })])).toBe('codex x1');
		expect(modelSummary([agent({ model: null, adapterType: null })])).toBe('unconfigured x1');
	});
});

describe('groupIssues', () => {
	it('known lanes ride in order, unknown statuses append, cancelled hidden', () => {
		const lanes = groupIssues([issue('todo'), issue('triage'), issue('cancelled'), issue('in_progress'), issue('done')]);
		expect(lanes.map((l) => l.status)).toEqual(['in_progress', 'todo', 'done', 'triage']);
		expect(lanes.flatMap((l) => l.issues).find((i) => i.status === 'cancelled')).toBeUndefined();
	});
	it('the core stages always render, optional ones only with work', () => {
		expect(groupIssues([]).map((l) => l.status)).toEqual([...CORE_ISSUE_LANES]);
		expect(groupIssues([issue('in_review')]).map((l) => l.status)).toEqual(['in_progress', 'in_review', 'todo', 'done']);
	});
});

describe('seats and runs', () => {
	it('the Conductor is swapped into the centre slot and nothing else moves', () => {
		const seats = ['A', 'B', 'C', 'Conductor', 'E'].map((name, i) => agent({ id: String(i), name }));
		expect(orderSeats(seats).map((s) => s.name)).toEqual(['A', 'B', 'Conductor', 'C', 'E']);
		expect(orderSeats(seats.slice(0, 2)).map((s) => s.name)).toEqual(['A', 'B']);
	});
	it('newest first, unstamped runs last', () => {
		const ordered = orderRuns([run({ id: 'old', startedAt: '2026-08-06T00:00:00Z' }), run({ id: 'unstamped', startedAt: null }), run({ id: 'new', startedAt: '2026-08-07T00:00:00Z' })]);
		expect(ordered.map((r) => r.id)).toEqual(['new', 'old', 'unstamped']);
	});
	it('duration reads human, in-flight runs have none', () => {
		expect(runDuration(run())).toBe('14s');
		expect(runDuration(run({ finishedAt: '2026-08-07T05:03:12Z' }))).toBe('3m 12s');
		expect(runDuration(run({ finishedAt: null }))).toBeNull();
	});
	it('runningSince only answers while the board says the seat is mid-run', () => {
		expect(runningSince([run({ status: 'running' })], 'a')).toBe('2026-08-07T05:00:00Z');
		expect(runningSince([run()], 'a')).toBeNull();
	});
	it('glyphs and tones follow the status, not the department colour', () => {
		expect(runGlyph('running')).toBe('▸');
		expect(runGlyph('completed')).toBe('✓');
		expect(runGlyph('timed_out')).toBe('✕');
		expect(seatTone(agent({ status: 'error' }), false)).toBe('err');
		expect(seatTone(agent({ status: 'idle' }), true)).toBe('ok');
		expect(seatTone(agent({ status: 'paused' }), false)).toBe('warn');
		expect(seatTone(agent({ status: 'idle' }), false)).toBe('idle');
	});
	it('ago is relative and never negative', () => {
		const now = Date.parse('2026-08-07T06:00:00Z');
		expect(ago('2026-08-07T05:59:30Z', now)).toBe('30s ago');
		expect(ago('2026-08-07T03:00:00Z', now)).toBe('3h ago');
		expect(ago('2026-08-08T00:00:00Z', now)).toBe('now');
		expect(ago(null, now)).toBe('');
	});
});

describe('boardDecisionFor', () => {
	it('binds a board-task decision to the revision it was made against', () => {
		const i = { ...issue('todo', 'i1'), updatedAt: 'v2' };
		const d = { id: 'board:i1', decision: 'approved' as const, decidedAt: 'x', decidedRevision: 'v2', note: '' };
		expect(boardDecisionFor(i, [d])).toEqual(d);
		expect(boardDecisionFor({ ...i, updatedAt: 'v3' }, [d])).toBeUndefined();
		expect(boardDecisionFor(i, [{ ...d, decidedRevision: '' }])).toBeTruthy();
	});
});

describe('writeFailure', () => {
	it('any guard refusal reads as refused, not failed, whatever its status', async () => {
		const { writeFailure } = await import('./runerror');
		const { FounderosApiError } = await import('$lib/founderos/api');
		expect(writeFailure(new FounderosApiError(409, 'x', { error: 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)', refused: true, guarded: true }), 'send')).toBe('send refused · writes are off');
		expect(writeFailure(new FounderosApiError(502, 'x', { ok: false, detail: 'bridge: outbound writes are disabled (FOUNDEROS_WRITES=0)' }), 'reply')).toBe('reply refused · writes are off');
	});
});
