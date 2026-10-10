/**
 * The Blueprint graph as GET /api/founderos/pages/blueprint returns it: the
 * system, compiled from the bridge's live registries on every request
 * (desktop/backend-go/internal/founderos/pages/blueprint). Port of FounderOS v1
 * lib/blueprint/graph.ts without zod: the Go compiler validates before it
 * answers, and validateGraph re-checks here.
 */

export const NODE_KINDS = [
	'operator',
	'wizard',
	'department',
	'agent',
	'person',
	'connector',
	'model',
	'skillpack',
	'store',
	'daemon',
	'host',
	'service',
	'container',
	'tool',
	'surface',
	'router'
] as const;
export type NodeKind = (typeof NODE_KINDS)[number];

/** live (wired + working) · configured (wired, unverified) · not-configured
 *  (missing its credential/host) · designed (specified, not wired). */
export type NodeStatus = 'live' | 'configured' | 'not-configured' | 'designed';

export interface BlueprintNode {
	id: string;
	kind: NodeKind;
	name: string;
	layer: number;
	status: NodeStatus;
	blurb: string;
	facts: Record<string, string>;
	icon: string;
}

export type EdgeKind = 'commands' | 'member-of' | 'uses' | 'runs-on' | 'reads' | 'writes' | 'delivers-to' | 'deploys-to';

export interface BlueprintEdge {
	from: string;
	to: string;
	kind: EdgeKind;
	via?: string;
}

export interface BlueprintGraph {
	compiledAt: string;
	nodes: BlueprintNode[];
	edges: BlueprintEdge[];
}

export type BlueprintBody = { ok: boolean; graph?: BlueprintGraph; error?: string };

/** Every edge endpoint must exist and ids are unique. */
export function validateGraph(graph: BlueprintGraph): string[] {
	const ids = new Set(graph.nodes.map((n) => n.id));
	const problems: string[] = [];
	for (const e of graph.edges) {
		if (!ids.has(e.from)) problems.push(`edge from missing node: ${e.from}`);
		if (!ids.has(e.to)) problems.push(`edge to missing node: ${e.to}`);
	}
	const seen = new Set<string>();
	for (const n of graph.nodes) {
		if (seen.has(n.id)) problems.push(`duplicate node id: ${n.id}`);
		seen.add(n.id);
	}
	return problems;
}

export function compiledAgo(iso: string, now: Date = new Date()): string {
	const seconds = Math.round((now.getTime() - new Date(iso).getTime()) / 1000);
	if (!Number.isFinite(seconds)) return 'at an unknown time';
	if (seconds < 5) return 'just now';
	if (seconds < 60) return `${seconds}s ago`;
	const minutes = Math.round(seconds / 60);
	if (minutes < 60) return `${minutes}m ago`;
	const hours = Math.round(minutes / 60);
	if (hours < 24) return `${hours}h ago`;
	return `${Math.round(hours / 24)}d ago`;
}

/** FounderOS v1 app/blueprint/page.tsx's subline: counts by kind, and when. */
export function blueprintSubline(graph: BlueprintGraph, now: Date = new Date()): string {
	const count = (kind: NodeKind) => graph.nodes.filter((n) => n.kind === kind).length;
	return `${graph.nodes.length} components · ${count('agent')} agents · ${count('daemon')} daemons · ${count('host')} machines · compiled ${compiledAgo(graph.compiledAt, now)}`;
}
