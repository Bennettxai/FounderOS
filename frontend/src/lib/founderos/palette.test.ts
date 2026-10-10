import { describe, expect, it } from 'vitest';
import { NAV_ORDER } from './nav';
import { buildPaletteCommands, digitTarget, filterPalette, type PaletteAgent } from './palette';

const cmds = buildPaletteCommands();

describe('palette commands', () => {
	it('has a Go-to command for every nav view, in nav order', () => {
		expect(cmds.filter((c) => c.kind === 'go').map((c) => c.href)).toEqual(NAV_ORDER);
	});

	it('carries the group as the sub line (Library for Personas, as in FounderOS v1)', () => {
		expect(cmds.find((c) => c.href === '/os/comms')?.sub).toBe('Operate');
		expect(cmds.find((c) => c.href === '/os/personas')?.sub).toBe('Library');
	});

	it('ids are unique', () => {
		const ids = cmds.map((c) => c.id);
		expect(new Set(ids).size).toBe(ids.length);
	});
});

describe('filterPalette', () => {
	it('empty query returns everything', () => {
		expect(filterPalette(cmds, '')).toEqual(cmds);
		expect(filterPalette(cmds, '   ')).toEqual(cmds);
	});

	it('matches labels case-insensitively', () => {
		expect(filterPalette(cmds, 'COMMS').map((c) => c.href)).toContain('/os/comms');
	});

	it('every word must match: "jump comms" finds the view through its kind words', () => {
		expect(filterPalette(cmds, 'jump comms').map((c) => c.href)).toEqual(['/os/comms']);
	});

	it('aliases match what the operator types: inbox → comms, robinhood → trading', () => {
		expect(filterPalette(cmds, 'inbox').map((c) => c.href)).toContain('/os/comms');
		expect(filterPalette(cmds, 'robinhood').map((c) => c.href)).toEqual(['/os/trading']);
	});

	it('v1 aliases: kanban and todo find Tasks, threads find Chats, deliverables still find Agents', () => {
		expect(filterPalette(cmds, 'tasks').map((c) => c.href)).toContain('/os/tasks');
		for (const q of ['kanban', 'todo']) expect(filterPalette(cmds, q).map((c) => c.href), q).toContain('/os/tasks');
		for (const q of ['threads', 'conversations']) expect(filterPalette(cmds, q).map((c) => c.href), q).toContain('/os/chats');
		expect(filterPalette(cmds, 'deliverables').map((c) => c.href)).toContain('/os/agents');
	});

	it('the knowledge view is findable as brain, memory and optimal engine', () => {
		for (const q of ['brain', 'memory', 'optimal engine']) {
			expect(filterPalette(cmds, q).map((c) => c.href), q).toContain('/os/brain');
		}
	});

	it('group names match too: "system" lists the System views', () => {
		expect(filterPalette(cmds, 'system').map((c) => c.href)).toEqual(['/os/integrations', '/os/usage', '/os/roadmap', '/os/analytics', '/os/reference']);
	});

	it('garbage matches nothing', () => {
		expect(filterPalette(cmds, 'zzzqqq')).toEqual([]);
	});
});

describe('digit jumps', () => {
	it('1–9 map to the first nine views; 0 and letters do nothing', () => {
		expect(digitTarget('1')).toBe('/os');
		expect(digitTarget('3')).toBe('/os/funnel');
		expect(digitTarget('9')).toBe('/os/trading');
		expect(digitTarget('0')).toBeNull();
		expect(digitTarget('a')).toBeNull();
		expect(digitTarget('12')).toBeNull();
	});
});

// FounderOS v1 tests/palette.test.ts: the Run and Ask groups (spec 6.20).
describe('palette Run and Ask groups', () => {
	const AGENTS: PaletteAgent[] = [
		{ id: 'inbox-triage', name: 'Inbox Triage', role: 'communications' },
		{ id: 'markets', name: 'Markets Agent', role: 'finances' }
	];
	const all = buildPaletteCommands(AGENTS);

	it('still has every Go-to view with no agents known', () => {
		expect(buildPaletteCommands([]).filter((c) => c.kind === 'go').map((c) => c.href)).toEqual(NAV_ORDER);
	});

	it('has one Run command per agent, carrying the agent id for the run POST', () => {
		const runs = all.filter((c) => c.kind === 'run');
		expect(runs).toHaveLength(AGENTS.length);
		expect(runs.map((r) => r.agentId).sort()).toEqual(['inbox-triage', 'markets']);
		expect(runs.every((r) => !r.href)).toBe(true);
	});

	it('has Conductor Ask prompts (3+) plus a Brain jump', () => {
		const asks = all.filter((c) => c.kind === 'ask');
		const prompts = asks.filter((a) => a.prompt);
		expect(prompts.length).toBeGreaterThanOrEqual(3);
		for (const p of prompts) expect(p.prompt!.length).toBeGreaterThan(10);
		expect(asks.some((a) => a.href === '/os/brain')).toBe(true);
	});

	it('command ids are unique with agents in', () => {
		const ids = all.map((c) => c.id);
		expect(new Set(ids).size).toBe(ids.length);
	});

	it('scope narrows to one kind; all is the default', () => {
		expect(filterPalette(all, '', 'all')).toEqual(all);
		expect(filterPalette(all, '')).toEqual(all);
		expect(filterPalette(all, '', 'run').every((c) => c.kind === 'run')).toBe(true);
		const asks = filterPalette(all, '', 'ask');
		expect(asks.length).toBeGreaterThan(0);
		expect(asks.every((c) => c.kind === 'ask')).toBe(true);
	});

	it('"run inbox" finds the agent via its kind + name', () => {
		const hits = filterPalette(all, 'run inbox', 'all');
		expect(hits.map((c) => c.agentId)).toContain('inbox-triage');
		expect(hits.every((c) => c.kind === 'run')).toBe(true);
	});

	it('the old view queries are unchanged by the new groups', () => {
		expect(filterPalette(all, 'jump comms').map((c) => c.href)).toEqual(['/os/comms']);
		expect(filterPalette(all, 'system').map((c) => c.href)).toEqual(['/os/integrations', '/os/usage', '/os/roadmap', '/os/analytics', '/os/reference']);
		expect(filterPalette(all, 'zzzqqq', 'all')).toEqual([]);
	});
});
