import { describe, expect, it } from 'vitest';
import { ACTION_LENSES, ALL_LENSES, ENTITY_LENSES, FUNCTION_LENSES, lensNodeSet } from './graph-lens';

const nodes = [
	{ id: 'self', kind: 'self' },
	{ id: 'team:dept-sales', kind: 'team' },
	{ id: 'team:dept-tech', kind: 'team' },
	{ id: 'task:sop-l', kind: 'task' },
	{ id: 'emp:vantage-sales', kind: 'employee' },
	{ id: 'emp:data-agent', kind: 'employee' },
	{ id: 'person:len', kind: 'person' },
	{ id: 'tool:attio@dept-sales', kind: 'tool' }
];
const team: Record<string, string> = {
	'task:sop-l': 'team:dept-sales',
	'emp:vantage-sales': 'team:dept-sales',
	'person:len': 'team:dept-sales',
	'emp:data-agent': 'team:dept-tech',
	'tool:attio@dept-sales': 'team:dept-sales'
};
const ctx = { nodes, teamOf: (id: string) => team[id] ?? null };

describe('graph lenses (FounderOS v1 lib/graph-lens.ts)', () => {
	it('carries production’s three lens groups in order', () => {
		expect(ENTITY_LENSES.map((l) => l.label)).toEqual(['All people', 'Sub-agents', 'Tools', 'Workflows', 'SOPs', 'Projects', 'Teams', 'Departments']);
		expect(FUNCTION_LENSES.map((l) => l.label)).toEqual(['Core', 'Enabling', 'Vantage team', 'Launchpad Cohort team']);
		expect(ACTION_LENSES).toHaveLength(11);
		expect(ALL_LENSES).toHaveLength(23);
	});

	it('entity lenses light a node kind; unmodeled entities are an honest empty set', () => {
		expect([...lensNodeSet('ent-tools', ctx)]).toEqual(['tool:attio@dept-sales']);
		expect([...lensNodeSet('ent-people', ctx)]).toEqual(['person:len']);
		expect([...lensNodeSet('ent-departments', ctx)]).toEqual(['team:dept-sales', 'team:dept-tech']);
		expect(lensNodeSet('ent-workflows', ctx).size).toBe(0);
		expect(lensNodeSet('nope', ctx).size).toBe(0);
	});

	it('function lenses split core from enabling pillars by each node’s team', () => {
		const core = lensNodeSet('fn-core', ctx);
		expect(core.has('team:dept-sales') && core.has('person:len') && core.has('tool:attio@dept-sales')).toBe(true);
		expect(core.has('emp:data-agent')).toBe(false);
		expect([...lensNodeSet('fn-enabling', ctx)].sort()).toEqual(['emp:data-agent', 'team:dept-tech']);
		expect([...lensNodeSet('fn-vantage', ctx)]).toEqual(['emp:vantage-sales']);
	});

	it('action lenses light the agents that run them', () => {
		expect([...lensNodeSet('act-lead-generation', ctx)]).toEqual(['emp:vantage-sales']);
		expect([...lensNodeSet('act-channel-budget', ctx)]).toEqual(['emp:data-agent']);
	});
});
