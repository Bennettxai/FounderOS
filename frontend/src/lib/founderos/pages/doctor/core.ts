/**
 * Deterministic geometry and readings for the Doctor Core (FounderOS v1
 * lib/brain-viz.ts + components/BrainViz.tsx), on Optimal Engine. Prod's inner
 * ring is one node per brain-store page grouped by folder; here it is the
 * engines' knowledge tables (contexts, claims, edges…) summed across the
 * engines that answered, scaled to a node cap so 200k rows still draw.
 */
import type { DoctorCheck, EngineReading } from './types';

export type CoreCluster = { label: string; pages: number };
export type CoreNode = { x: number; y: number };
export type CoreLabel = { angle: number; label: string; pages: number };

export const CX = 260;
export const CY = 260;
const RING_R = 108;
const MAX_CLUSTERS = 6;
/** prod drew 382 page nodes; past this the ring is a solid band anyway */
export const NODE_CAP = 384;

/** Knowledge tables (telemetry like events and jobs is left out), with short labels. */
const KNOWLEDGE: [string, string][] = [
	['contexts', 'contexts'],
	['claims', 'claims'],
	['facts', 'facts'],
	['episodes', 'episodes'],
	['memory_objects', 'memories'],
	['edges', 'edges'],
	['nodes', 'nodes']
];

const up = (es: EngineReading[]) => es.filter((e) => e.reachable);

/** Sum one table across the engines that answered; null when none reported it. */
export function tableSum(es: EngineReading[], table: string): number | null {
	let total = 0;
	let seen = false;
	for (const e of up(es)) {
		for (const s of e.stores ?? []) {
			const n = s.tables?.[table];
			if (typeof n === 'number') {
				total += n;
				seen = true;
			}
		}
	}
	return seen ? total : null;
}

/** Engine knowledge tables → up to six clusters, biggest first; the tail folds into `misc`. */
export function knowledgeClusters(es: EngineReading[]): CoreCluster[] {
	const nonEmpty = KNOWLEDGE.map(([table, label]) => ({ label, pages: tableSum(es, table) ?? 0 }))
		.filter((c) => c.pages > 0)
		.sort((a, b) => b.pages - a.pages || a.label.localeCompare(b.label));
	if (nonEmpty.length <= MAX_CLUSTERS) return nonEmpty;
	const head = nonEmpty.slice(0, MAX_CLUSTERS - 1);
	const tail = nonEmpty.slice(MAX_CLUSTERS - 1);
	return [...head, { label: 'misc', pages: tail.reduce((n, c) => n + c.pages, 0) }];
}

/**
 * Clusters share 360° proportionally starting at -90°; per-node jitter
 * `((ci*7 + i*13) % 10) - 5` on the ring radius (prod layoutBrainNodes). Past
 * `cap` rows each cluster draws a proportional share, at least one node.
 */
export function layoutCoreNodes(clusters: CoreCluster[], cap = NODE_CAP): { nodes: CoreNode[]; labels: CoreLabel[] } {
	const total = clusters.reduce((n, c) => n + c.pages, 0);
	if (total === 0) return { nodes: [], labels: [] };
	const scale = total > cap ? cap / total : 1;
	const nodes: CoreNode[] = [];
	const labels: CoreLabel[] = [];
	let angle = -90;
	clusters.forEach((cluster, ci) => {
		const span = (cluster.pages / total) * 360;
		labels.push({ angle: angle + span / 2, label: cluster.label, pages: cluster.pages });
		const count = Math.max(1, Math.round(cluster.pages * scale));
		for (let i = 0; i < count; i++) {
			const a = ((angle + (span * (i + 0.5)) / count) * Math.PI) / 180;
			const r = RING_R + (((ci * 7 + i * 13) % 10) - 5);
			nodes.push({ x: Math.round((CX + r * Math.cos(a)) * 100) / 100, y: Math.round((CY + r * Math.sin(a)) * 100) / 100 });
		}
		angle += span;
	});
	return { nodes, labels };
}

export function polar(cx: number, cy: number, r: number, deg: number): [number, number] {
	const a = (deg * Math.PI) / 180;
	return [Math.round((cx + r * Math.cos(a)) * 100) / 100, Math.round((cy + r * Math.sin(a)) * 100) / 100];
}

/** The outer ring's sampled dots (prod: 42, jittered around r 206). */
export function outerDots(): CoreNode[] {
	return Array.from({ length: 42 }, (_, i) => {
		const [x, y] = polar(CX, CY, 206 + ((i * 11) % 9) - 4, (i / 42) * 360 + (i % 5) * 1.7);
		return { x, y };
	});
}

const fmt = (n: number | null) => (n == null ? '—' : n.toLocaleString('en-US'));

/** The three ring callouts: store (inner), vectors (middle), topology (outer). */
export function coreCallouts(es: EngineReading[]): { inner: string; middle: string; outer: string } {
	const live = up(es);
	const ws = new Set(live.flatMap((e) => e.workspaces ?? []));
	const down = es.length - live.length;
	const names = es.map((e) => e.name.toUpperCase()).join(' + ') || 'NO ENGINES';
	return {
		inner: `ENGINE STORE · ${fmt(tableSum(es, 'contexts'))} CONTEXTS`,
		middle: `VECTOR · ${fmt(tableSum(es, 'chunk_embeddings'))} CHUNKS · COSINE`,
		outer: `${names} · ${ws.size} WORKSPACES · ${down === 0 && es.length > 0 ? 'LIVE' : `${down} DOWN`}`
	};
}

/** An audit detail as words: JSON objects flatten to "k v · k v", long nested strings drop. */
export function checkMessage(raw: string): string {
	const msg = (raw ?? '').trim();
	if (msg.startsWith('{')) {
		try {
			const obj = JSON.parse(msg) as Record<string, unknown>;
			return Object.entries(obj)
				.filter(([, v]) => typeof v === 'number' || typeof v === 'boolean' || (typeof v === 'string' && v.length <= 24))
				.map(([k, v]) => `${k} ${v}`)
				.join(' · ');
		} catch {
			return msg;
		}
	}
	return msg.replace(/^:/, '');
}

export type CheckLine = { name: string; status: string; message: string; engines: string[] };

const SEVERITY = (s: string) => (s === 'ok' ? 0 : s === 'warn' ? 1 : 2);

/** One line per check name across engines: worst status wins, disagreeing readings stay per engine. */
export function groupChecks(checks: DoctorCheck[]): CheckLine[] {
	const by = new Map<string, DoctorCheck[]>();
	for (const c of checks) by.set(c.name, [...(by.get(c.name) ?? []), c]);
	return [...by.entries()].map(([name, cs]) => {
		const worst = cs.reduce((w, c) => (SEVERITY(c.status) > SEVERITY(w) ? c.status : w), 'ok');
		const msgs = cs.map((c) => checkMessage(c.message));
		const same = msgs.every((m) => m === msgs[0]);
		return {
			name,
			status: worst,
			message: same ? msgs[0] : cs.map((c, i) => `${c.engine} ${msgs[i]}`).join(' · '),
			engines: cs.map((c) => c.engine)
		};
	});
}
