// The small pure helpers FounderOS v1's knowledge graph leans on
// (lib/knowledge-graph, lib/agent-wiki, lib/personnel, lib/kg-colors,
// lib/memory-search, lib/neural-layout, lib/sop-playbooks), ported as is.
import { describe, expect, test } from 'vitest';
import { GRAPH, PAGE } from './fixtures';
import { GRAPH_DEPT_ORDER, SELF_ID, buildToolWiki, headForDepartment, orderGraphDepartments, prettifySlug, themedNodeColor, toolSlugOf } from './kg';
import { searchMemoryNotes } from './memory-search';
import { NEURAL_H, NEURAL_W, neuralLayout } from './neural-layout';
import { AUTONOMY_LEVELS, PLAYBOOK_STATUSES, autonomyLabel, playbookFor, statusLabel } from './sop-playbooks';
import type { MemoryNode } from './memory-core';
import type { KnowledgeGraph } from './types';

describe('knowledge-graph helpers', () => {
	test('the core is the self node', () => expect(SELF_ID).toBe('self'));

	test('tool node ids map back to their slug, per-department copies included', () => {
		expect(toolSlugOf('tool:attio@dept-sales')).toBe('attio');
		expect(toolSlugOf('tool:openclaw')).toBe('openclaw');
	});

	test('graph order puts Finances beside Sales; unknown pillars go last', () => {
		expect(GRAPH_DEPT_ORDER[0]).toBe('dept-sales');
		expect(GRAPH_DEPT_ORDER[1]).toBe('dept-finance');
		const ids = ['dept-comms', 'dept-x', 'dept-finance', 'dept-sales'];
		expect(orderGraphDepartments(ids, (d) => d)).toEqual(['dept-sales', 'dept-finance', 'dept-comms', 'dept-x']);
	});

	test('a life-area hex becomes a theme-remappable var; anything else passes through', () => {
		expect(themedNodeColor('#EF4444')).toBe('var(--kg-c-ef4444, #EF4444)');
		expect(themedNodeColor('var(--accent)')).toBe('var(--accent)');
	});

	test('only Marketing/Growth has a human head (Sales has none since Len left)', () => {
		expect(headForDepartment('dept-marketing-growth')?.name).toBe('Syed');
		expect(headForDepartment('dept-sales')).toBeNull();
	});
});

describe('tool wiki', () => {
	test('humanizes a slug', () => expect(prettifySlug('comms-feed')).toBe('Comms Feed'));

	test('flags MCP-backed tools and admits there is no page without an index', () => {
		const w = buildToolWiki('slack', ['Comms Agent']);
		expect(w).toMatchObject({ slug: 'slack', name: 'Slack', mcp: true, kind: 'MCP server', hasPage: false, path: null, usedBy: ['Comms Agent'] });
		expect(buildToolWiki('stripe').kind).toBe('integration');
	});
});

describe('vault search', () => {
	const note = (id: string, label: string, over: Partial<MemoryNode> = {}): MemoryNode => ({
		id,
		type: 'page',
		label,
		folder: 'ideas',
		excerpt: '',
		vx: 0,
		vy: 0,
		cluster: 0,
		links: 1,
		...over
	});
	test('ranks label prefix over substring over folder over excerpt; under 2 chars finds nothing', () => {
		const nodes = [
			note('a', 'Pricing model'),
			note('b', 'Old pricing'),
			note('c', 'Notes', { folder: 'pricing' }),
			note('d', 'Misc', { excerpt: 'talks pricing' }),
			note('f', 'pricing hub', { type: 'folder' })
		];
		expect(searchMemoryNotes(nodes, 'pricing').map((n) => n.id)).toEqual(['a', 'b', 'c', 'd']);
		expect(searchMemoryNotes(nodes, 'p')).toEqual([]);
	});
});

describe('neural layout (the feedforward view of the same graph)', () => {
	const graph: KnowledgeGraph = {
		nodes: [...GRAPH.nodes, { id: 'emp:data', kind: 'employee', label: 'Data', ring: 3 }],
		edges: [...GRAPH.edges, { source: 'emp:data', target: 'emp:conductor', kind: 'reports' }]
	};

	test('five layers left to right: tools, workers, tasks, pillars, self; board seats sit it out', () => {
		const { layers, pos } = neuralLayout(graph);
		expect(layers.map((l) => l.kind)).toEqual(['tool', 'worker', 'task', 'team', 'self']);
		expect(layers.map((l) => l.name)).toEqual(['INPUT · TOOLS', 'HL 1 · WORKERS', 'HL 2 · SOP TASKS', 'HL 3 · PILLARS', 'OUTPUT · ENGINE']);
		expect(layers[4].nodeIds).toEqual(['self']);
		expect(pos.has('board:p2')).toBe(false);
		for (const p of pos.values()) {
			expect(p.x).toBeGreaterThanOrEqual(0);
			expect(p.x).toBeLessThanOrEqual(NEURAL_W);
			expect(p.y).toBeGreaterThanOrEqual(0);
			expect(p.y).toBeLessThanOrEqual(NEURAL_H);
		}
	});

	test('strands join adjacent layers only, oriented left to right, with signed weights; reports come back apart', () => {
		const { strands, layers, reports } = neuralLayout(graph);
		const layerOf = new Map<string, number>();
		layers.forEach((l, i) => l.nodeIds.forEach((id) => layerOf.set(id, i)));
		expect(strands.length).toBeGreaterThan(0);
		for (const s of strands) {
			expect(layerOf.get(s.target)! - layerOf.get(s.source)!).toBe(1);
			expect(Math.abs(s.weight)).toBeGreaterThan(0);
			expect(Math.abs(s.weight)).toBeLessThanOrEqual(1);
		}
		expect(neuralLayout(graph).strands).toEqual(strands);
		expect(reports).toEqual([{ source: 'emp:data', target: 'emp:conductor' }]);
	});
});

describe('SOP playbooks', () => {
	test('an authored SOP resolves to its playbook with a runnable skill file', () => {
		const pb = playbookFor({ ...PAGE.tasks[0], id: 'sop-conductor' });
		expect(pb.categoryPath).toBe('TECH · Fleet Orchestration');
		expect(pb.autonomy).toBe('fully-autonomous');
		expect(pb.skillMarkdown).toContain('---');
		expect(pb.skillMarkdown).toContain(pb.skill.name);
	});

	test('an unknown SOP still gets a full card: human-owned reads human-led', () => {
		const pb = playbookFor(PAGE.tasks[1]);
		expect(pb.autonomy).toBe('human-led');
		expect(pb.skill.name).toBe('Close deals Skill');
		expect(pb.skill.slug).toMatch(/^[a-z0-9-]+$/);
		expect(AUTONOMY_LEVELS).toContain(pb.autonomy);
		expect(PLAYBOOK_STATUSES).toContain(pb.status);
	});

	test('labels read like production', () => {
		expect(autonomyLabel('fully-autonomous')).toBe('FULLY AUTONOMOUS');
		expect(statusLabel('ready-to-run')).toBe('READY TO RUN');
	});
});
