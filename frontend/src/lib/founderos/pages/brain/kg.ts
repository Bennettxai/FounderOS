/**
 * The small pure helpers FounderOS v1's /brain knowledge graph uses, gathered
 * from lib/knowledge-graph.ts, lib/agent-wiki.ts, lib/personnel.ts and
 * lib/kg-colors.ts. The graph, directory and exec titles themselves come from
 * the bridge (internal/founderos/api/page_brain.go).
 */

export const SELF_ID = 'self';

/** Graph display order: Finances rides next to Sales. */
export const GRAPH_DEPT_ORDER = ['dept-sales', 'dept-finance', 'dept-clients', 'dept-marketing-growth', 'dept-tech', 'dept-comms'] as const;

/** Rank a department id for graph layout; unknown ids sort after the known six. */
export function graphDeptRank(deptId: string): number {
	const i = (GRAPH_DEPT_ORDER as readonly string[]).indexOf(deptId);
	return i < 0 ? GRAPH_DEPT_ORDER.length + 1 : i;
}

/** Order any dept-keyed list by the graph display order. */
export function orderGraphDepartments<T>(items: T[], deptIdOf: (item: T) => string): T[] {
	return [...items].sort((a, b) => graphDeptRank(deptIdOf(a)) - graphDeptRank(deptIdOf(b)));
}

/** Tool node id → slug (`tool:fathom@dept-sales` → `fathom`). */
export function toolSlugOf(nodeId: string): string {
	return nodeId.replace(/^tool:/, '').split('@')[0];
}

/** 'comms-feed' → 'Comms Feed' */
export function prettifySlug(slug: string): string {
	return slug
		.split(/[-_]/)
		.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
		.join(' ');
}

const HEX6 = /^#([0-9a-fA-F]{6})$/;
/** A life-area hex wrapped in a per-hex CSS var so a theme can re-skin it. */
export function themedNodeColor(color: string): string {
	const m = HEX6.exec(color);
	return m ? `var(--kg-c-${m[1].toLowerCase()}, ${color})` : color;
}

/** Human personnel that run departments (panel-only data, never graph nodes). */
export type Personnel = { id: string; name: string; role: string; departmentId: string };
const DEPARTMENT_HEADS: Record<string, { name: string; role: string }> = {
	// Sales has no head: Len is gone (the operator, 2026-08-18) and nobody replaced him.
	'dept-marketing-growth': { name: 'Syed', role: 'Head of Growth & Marketing' }
};
export function headForDepartment(departmentId: string): Personnel | null {
	const h = DEPARTMENT_HEADS[departmentId];
	return h ? { id: `head:${departmentId}`, name: h.name, role: h.role, departmentId } : null;
}

// Tools backed by an MCP server: a hand-kept fact (lib/agent-wiki.ts MCP_SLUGS).
const MCP_SLUGS = new Set([
	'notion', 'slack', 'gbrain', 'obsidian', 'miro', 'playwright', 'figma',
	'serena', 'context7', 'zernio', 'arcads', 'wispr', 'higgsfield', 'canva', 'gmail',
	'google-calendar', 'calendar', 'vercel', 'plaud'
]);

export type WikiRef = { target: string; slug: string | null; title: string };
export type ToolWiki = {
	slug: string;
	name: string;
	mcp: boolean;
	kind: string;
	hasPage: boolean;
	path: string | null;
	summary: string;
	fields: Record<string, string>;
	/** who is wired to it right now, from live OS state */
	usedBy: string[];
	links: WikiRef[];
	backlinks: WikiRef[];
};

/**
 * The tool card's facts. FounderOS v1 read a brain-store page per tool; the
 * port has no wiki index (G-Brain is retired), so every tool honestly reports
 * no page rather than a made-up one.
 */
export function buildToolWiki(slug: string, usedBy: string[] = []): ToolWiki {
	const mcp = MCP_SLUGS.has(slug);
	return {
		slug,
		name: prettifySlug(slug),
		mcp,
		kind: mcp ? 'MCP server' : 'integration',
		hasPage: false,
		path: null,
		summary: '',
		fields: {},
		usedBy,
		links: [],
		backlinks: []
	};
}
