import { describe, expect, it } from 'vitest';
import { blueprintSubline, compiledAgo, validateGraph, type BlueprintGraph } from './graph';

const now = new Date('2026-09-30T12:00:00Z');

function graph(): BlueprintGraph {
	return {
		compiledAt: '2026-09-30T11:57:00Z',
		nodes: [
			{ id: 'operator', kind: 'operator', name: 'the operator', layer: 0, status: 'live', blurb: '', facts: {}, icon: 'user-round' },
			{ id: 'agent-a', kind: 'agent', name: 'A', layer: 2, status: 'live', blurb: '', facts: {}, icon: 'bot' },
			{ id: 'agent-b', kind: 'agent', name: 'B', layer: 2, status: 'designed', blurb: '', facts: {}, icon: 'bot' },
			{ id: 'host-mini', kind: 'host', name: 'Mini', layer: 4, status: 'live', blurb: '', facts: {}, icon: 'server' },
			{ id: 'service-x', kind: 'service', name: 'X', layer: 4, status: 'live', blurb: '', facts: {}, icon: 'activity' },
			{ id: 'container-y', kind: 'container', name: 'Y', layer: 4, status: 'configured', blurb: '', facts: {}, icon: 'container' },
			{ id: 'tool-z', kind: 'tool', name: 'Z', layer: 4, status: 'live', blurb: '', facts: {}, icon: 'code' },
			{ id: 'daemon-d', kind: 'daemon', name: 'D', layer: 4, status: 'configured', blurb: '', facts: {}, icon: 'play' }
		],
		edges: [{ from: 'agent-a', to: 'host-mini', kind: 'runs-on' }]
	};
}

describe('blueprint graph helpers', () => {
	it('the subline counts the compiled graph by kind, like FounderOS v1 app/blueprint/page.tsx', () => {
		expect(blueprintSubline(graph(), now)).toBe(
			'8 components · 2 agents · 1 daemons · 1 machines · compiled 3m ago'
		);
	});

	it('compiled-ago reads seconds, minutes, hours, days', () => {
		expect(compiledAgo('2026-09-30T11:59:58Z', now)).toBe('just now');
		expect(compiledAgo('2026-09-30T11:59:30Z', now)).toBe('30s ago');
		expect(compiledAgo('2026-09-30T09:00:00Z', now)).toBe('3h ago');
		expect(compiledAgo('2026-09-28T12:00:00Z', now)).toBe('2d ago');
	});

	it('validateGraph flags dangling edges and duplicate ids', () => {
		expect(validateGraph(graph())).toEqual([]);
		const bad = graph();
		bad.edges.push({ from: 'ghost', to: 'agent-a', kind: 'uses' });
		bad.nodes.push({ ...bad.nodes[1] });
		expect(validateGraph(bad)).toEqual(['edge from missing node: ghost', 'duplicate node id: agent-a']);
	});
});
