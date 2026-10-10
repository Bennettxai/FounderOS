/**
 * Geometry for a persona's knowledge graph (FounderOS v1 PersonaBrainGraph):
 * the core in the middle, the persona's pillars ringed around it, each
 * pillar's agents fanning out on an outer ring. Pure and deterministic.
 */
import type { Persona, PersonaPillar } from './types';

export const W = 900;
export const H = 760;
const CX = W / 2;
const CY = H / 2;
export const R_PILLAR = 210;
export const R_AGENT = 328;

export const trunc = (s: string, n: number) => (s.length > n ? `${s.slice(0, n - 1)}…` : s);
export const pad2 = (n: number) => String(n).padStart(2, '0');
/** The viewer's stepper: move by dir, wrapping around n. */
export const step = (i: number, dir: number, n: number) => (n ? (i + dir + n) % n : 0);

// deterministic pseudo-random in [0,1) from a string, for the core dot field
function rand(seed: string, i: number): number {
	let h = 2166136261 ^ i;
	for (let k = 0; k < seed.length; k++) {
		h ^= seed.charCodeAt(k);
		h = Math.imul(h, 16777619);
	}
	return ((h >>> 0) % 100000) / 100000;
}

const rad = (deg: number) => (deg * Math.PI) / 180;
export const pt = (cx: number, cy: number, r: number, deg: number) => ({
	x: cx + r * Math.cos(rad(deg)),
	y: cy + r * Math.sin(rad(deg))
});

export type Point = { x: number; y: number };
export type PillarNode = Point & { pillar: PersonaPillar; angle: number; agents: (Point & { name: string })[]; spokeStart: Point };

export function brainGraph(persona: Persona) {
	const pillars = persona.pillars;
	const n = pillars.length || 1;
	const sliceDeg = 360 / n;
	const nodes: PillarNode[] = pillars.map((p, i) => {
		const angle = -90 + i * sliceDeg;
		const c = pt(CX, CY, R_PILLAR, angle);
		const k = p.agents.length;
		const spread = Math.min(sliceDeg * 0.78, 62);
		const agents = p.agents.map((name, j) => {
			const a = k === 1 ? angle : angle - spread / 2 + (j * spread) / (k - 1);
			return { name, ...pt(CX, CY, R_AGENT, a) };
		});
		return { pillar: p, angle, ...c, agents, spokeStart: pt(CX, CY, 48, angle) };
	});
	const coreDots = Array.from({ length: 46 }, (_, i) => {
		const a = rand(persona.id, i * 2) * 360;
		const r = 8 + rand(persona.id, i * 2 + 1) * 34;
		return { ...pt(CX, CY, r, a), s: 1 + rand(persona.id, i * 7) * 2.2 };
	});
	return { cx: CX, cy: CY, pillars: nodes, coreDots };
}
