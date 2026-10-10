import { describe, expect, it } from 'vitest';
import { checkMessage, coreCallouts, groupChecks, knowledgeClusters, layoutCoreNodes, outerDots, polar } from './core';
import type { EngineReading } from './types';

function engine(over: Partial<EngineReading> = {}): EngineReading {
	return {
		name: 'hub',
		url: 'http://127.0.0.1:4211',
		reachable: true,
		health: 'up',
		workspaces: ['launchpad-cohort', 'founderos'],
		checks: [],
		stores: [
			{ id: 'relational', status: 'available', technology: 'SQLite', rowCount: 1, tables: { contexts: 700, claims: 650, episodes: 280, facts: 0, events: 9999 } },
			{ id: 'vector', status: 'available', technology: 'SQLite BLOB', rowCount: 1, tables: { chunk_embeddings: 39000, vectors: 700 } },
			{ id: 'graph', status: 'available', technology: 'SQLite', rowCount: 1, tables: { edges: 2000 } }
		],
		searches: 0,
		uptimeMs: 1,
		...over
	};
}

describe('knowledgeClusters (prod foldersToClusters, on the engine store)', () => {
	it('sums the knowledge tables across reachable engines, biggest first, drops empties and telemetry', () => {
		const down = engine({ name: 'macbook', reachable: false, stores: null, workspaces: null });
		const c = knowledgeClusters([engine(), engine({ name: 'macbook' }), down]);
		expect(c).toEqual([
			{ label: 'edges', pages: 4000 },
			{ label: 'contexts', pages: 1400 },
			{ label: 'claims', pages: 1300 },
			{ label: 'episodes', pages: 560 }
		]);
	});
	it('folds the tail into misc past six clusters', () => {
		const e = engine({
			stores: [{ id: 'r', status: 'a', technology: 't', rowCount: 1, tables: { contexts: 9, claims: 8, facts: 7, episodes: 6, memory_objects: 5, edges: 4, nodes: 3 } }]
		});
		const c = knowledgeClusters([e]);
		expect(c).toHaveLength(6);
		expect(c[5]).toEqual({ label: 'misc', pages: 7 });
		expect(c.map((x) => x.label)).toContain('memories');
	});
	it('is empty when no engine reported stores', () => {
		expect(knowledgeClusters([engine({ stores: null })])).toEqual([]);
	});
});

describe('layoutCoreNodes (prod layoutBrainNodes, capped)', () => {
	it('lays one node per row on the 108 ring, starting at -90°', () => {
		const { nodes, labels } = layoutCoreNodes([{ label: 'a', pages: 4 }]);
		expect(nodes).toHaveLength(4);
		for (const n of nodes) {
			const r = Math.hypot(n.x - 260, n.y - 260);
			expect(r).toBeGreaterThanOrEqual(102.9);
			expect(r).toBeLessThanOrEqual(113.1);
		}
		expect(labels).toEqual([{ angle: 90, label: 'a', pages: 4 }]);
	});
	it('scales a huge store down to the cap but keeps the real counts on the labels', () => {
		const { nodes, labels } = layoutCoreNodes(
			[
				{ label: 'edges', pages: 4000 },
				{ label: 'contexts', pages: 1400 }
			],
			380
		);
		expect(nodes.length).toBeLessThanOrEqual(381);
		expect(nodes.length).toBeGreaterThan(300);
		expect(labels.map((l) => l.pages)).toEqual([4000, 1400]);
	});
	it('gives even a tiny cluster one node', () => {
		const { nodes } = layoutCoreNodes([{ label: 'big', pages: 100000 }, { label: 'tiny', pages: 1 }], 100);
		expect(nodes.length).toBe(101);
	});
	it('draws nothing for an empty store', () => {
		expect(layoutCoreNodes([])).toEqual({ nodes: [], labels: [] });
	});
});

describe('coreCallouts', () => {
	it('names the store, the vectors and the topology like prod names brain-store, Ollama and Supabase', () => {
		const c = coreCallouts([engine(), engine({ name: 'macbook', workspaces: ['personal'] })]);
		expect(c.inner).toBe('ENGINE STORE · 1,400 CONTEXTS');
		expect(c.middle).toBe('VECTOR · 78,000 CHUNKS · COSINE');
		expect(c.outer).toBe('HUB + MACBOOK · 3 WORKSPACES · LIVE');
	});
	it('reads unknown as —, and a down engine as down', () => {
		const c = coreCallouts([engine({ stores: [] }), engine({ name: 'macbook', reachable: false, stores: null, workspaces: null })]);
		expect(c.inner).toBe('ENGINE STORE · — CONTEXTS');
		expect(c.middle).toBe('VECTOR · — CHUNKS · COSINE');
		expect(c.outer).toBe('HUB + MACBOOK · 2 WORKSPACES · 1 DOWN');
	});
});

describe('checkMessage', () => {
	it('flattens audit JSON into key value pairs', () => {
		expect(checkMessage('{"applied":52,"expected":52}')).toBe('applied 52 · expected 52');
		expect(checkMessage('{"contexts":777,"indexed":777}')).toBe('contexts 777 · indexed 777');
	});
	it('drops long nested strings so a stack trace never floods the list', () => {
		expect(checkMessage('{"available":false,"optional":true,"reason":"{:rlm_health_exit, 1, \\"dspy is not installed; install the optional Optimal RLM environment\\"}"}')).toBe(
			'available false · optional true'
		);
	});
	it('reads atoms and plain text as words', () => {
		expect(checkMessage(':no_verified_backup')).toBe('no_verified_backup');
		expect(checkMessage('no violations')).toBe('no violations');
		expect(checkMessage('')).toBe('');
	});
});

describe('groupChecks', () => {
	it('folds the same check across engines into one line, worst status wins', () => {
		const g = groupChecks([
			{ engine: 'hub', name: 'sqlite_integrity', status: 'ok', message: 'ok' },
			{ engine: 'hub', name: 'verified_backup', status: 'error', message: ':no_verified_backup' },
			{ engine: 'macbook', name: 'sqlite_integrity', status: 'ok', message: 'ok' },
			{ engine: 'macbook', name: 'verified_backup', status: 'warn', message: ':no_verified_backup' }
		]);
		expect(g).toEqual([
			{ name: 'sqlite_integrity', status: 'ok', message: 'ok', engines: ['hub', 'macbook'] },
			{ name: 'verified_backup', status: 'error', message: 'no_verified_backup', engines: ['hub', 'macbook'] }
		]);
	});
	it('keeps per-engine readings when the engines disagree', () => {
		const g = groupChecks([
			{ engine: 'hub', name: 'fts_parity', status: 'ok', message: '{"contexts":777,"indexed":777}' },
			{ engine: 'macbook', name: 'fts_parity', status: 'ok', message: '{"contexts":674,"indexed":674}' }
		]);
		expect(g[0].message).toBe('hub contexts 777 · indexed 777 · macbook contexts 674 · indexed 674');
	});
});

describe('geometry', () => {
	it('polar rounds to 2 decimals, 0° at 3 o’clock', () => {
		expect(polar(260, 260, 100, 0)).toEqual([360, 260]);
		expect(polar(260, 260, 100, 90)).toEqual([260, 360]);
	});
	it('samples 42 outer dots around r≈206', () => {
		const d = outerDots();
		expect(d).toHaveLength(42);
		for (const p of d) expect(Math.hypot(p.x - 260, p.y - 260)).toBeGreaterThan(200);
	});
});
