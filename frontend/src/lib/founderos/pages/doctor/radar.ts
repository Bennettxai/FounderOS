/** Radar geometry and hover-to-sift layer picking (FounderOS v1 lib/pillar-radar.ts). */
import type { PillarAxis } from './types';

export type PillarLayerKey = 'score' | 'roster' | 'freshness' | 'sop';
export const PILLAR_LAYER_KEYS: PillarLayerKey[] = ['score', 'roster', 'freshness', 'sop'];

/** Vertex for axis `i` of `count`, `radius` out from `center`. First axis at top. */
export function radarPoint(axisIndex: number, count: number, radius: number, center: number): [number, number] {
	const a = (axisIndex / Math.max(1, count)) * 2 * Math.PI - Math.PI / 2;
	return [center + radius * Math.cos(a), center + radius * Math.sin(a)];
}

function layerVertices(axes: PillarAxis[], key: PillarLayerKey, R: number, center: number): [number, number][] {
	return axes.map((a, i) => radarPoint(i, axes.length, (Math.max(5, a[key]) / 100) * R, center));
}

function distToSegment(p: { x: number; y: number }, a: [number, number], b: [number, number]): number {
	const [ax, ay] = a;
	const [bx, by] = b;
	const dx = bx - ax;
	const dy = by - ay;
	const len2 = dx * dx + dy * dy;
	const t = len2 === 0 ? 0 : Math.max(0, Math.min(1, ((p.x - ax) * dx + (p.y - ay) * dy) / len2));
	return Math.hypot(p.x - (ax + t * dx), p.y - (ay + t * dy));
}

/** The layer whose polygon the cursor is nearest; ties and no axes resolve to 'score'. */
export function nearestPillarLayer(cursor: { x: number; y: number }, axes: PillarAxis[], R: number, center: number): PillarLayerKey {
	if (axes.length === 0) return 'score';
	let best: PillarLayerKey = 'score';
	let bestD = Infinity;
	for (const key of PILLAR_LAYER_KEYS) {
		const v = layerVertices(axes, key, R, center);
		let d = Infinity;
		for (let i = 0; i < v.length; i++) d = Math.min(d, distToSegment(cursor, v[i], v[(i + 1) % v.length]));
		if (d < bestD) {
			bestD = d;
			best = key;
		}
	}
	return best;
}

export function relativeTime(iso: string, now = Date.now()): string {
	const ms = now - new Date(iso).getTime();
	if (!Number.isFinite(ms) || ms < 0) return iso;
	const m = Math.floor(ms / 60_000);
	if (m < 1) return 'just now';
	if (m < 60) return `${m}m ago`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h ago`;
	return `${Math.floor(h / 24)}d ago`;
}
