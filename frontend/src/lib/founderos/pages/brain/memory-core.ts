/**
 * The client half of FounderOS v1 lib/memory-core.ts: the Obsidian-style memory
 * core's rest tier, disc projection and cinematic camera. Pure, no DOM. The
 * distillation itself (distillMemoryGraph) runs in the bridge
 * (pages/brain/constellation.go) over the Optimal Engine graphs.
 */

/** A note in the memory constellation. The engines report no word or chunk
 *  counts, so those are optional here (absent reads as 0, never invented). */
export type MemoryNode = {
	id: string;
	type: 'folder' | 'page';
	label: string;
	folder: string;
	genre?: string;
	excerpt: string;
	wordCount?: number;
	chunks?: number;
	vx: number; // layout coords in the unit disc
	vy: number;
	/** link community (0 = largest) */
	cluster: number;
	/** degree among kept nodes; 0 = orphan on the rim halo */
	links: number;
};
export type MemoryEdge = { source: string; target: string; type: 'member' | 'wikilink' | 'similar' };
export type MemoryGraph = { nodes: MemoryNode[]; edges: MemoryEdge[] };

const words = (n: MemoryNode) => n.wordCount ?? 0;

/**
 * The mini-Obsidian subset shown while the core is collapsed: folder hubs
 * always, then linked pages sampled ROUND-ROBIN across 12 angular sectors
 * (best-linked first within each sector) so the mini graph covers the whole
 * disc instead of clumping where the big communities sit, plus every 3rd
 * orphan for the rim halo. The full graph renders once the core is clicked
 * open. Pure + deterministic.
 */
export function pickRestTier(nodes: MemoryNode[], maxPages = 96): MemoryNode[] {
  const hubs = nodes.filter((n) => n.type === 'folder');
  const linked = nodes.filter((n) => n.type === 'page' && n.links > 0);
  const SECTORS = 12;
  const bySector: MemoryNode[][] = Array.from({ length: SECTORS }, () => []);
  for (const n of linked) {
    const a = Math.atan2(n.vy, n.vx);
    bySector[Math.min(SECTORS - 1, Math.floor(((a + Math.PI) / (2 * Math.PI)) * SECTORS))].push(n);
  }
  for (const bucket of bySector) {
    bucket.sort((a, b) => b.links - a.links || words(b) - words(a) || a.id.localeCompare(b.id));
  }
  const budget = Math.min(maxPages, linked.length);
  const picked: MemoryNode[] = [];
  for (let round = 0; picked.length < budget; round++) {
    let took = false;
    for (const bucket of bySector) {
      if (picked.length >= budget) break;
      const n = bucket[round];
      if (n) {
        picked.push(n);
        took = true;
      }
    }
    if (!took) break;
  }
  const orphans = nodes.filter((n) => n.type === 'page' && n.links === 0).filter((_, i) => i % 3 === 0);
  const keep = new Set([...hubs, ...picked, ...orphans].map((n) => n.id));
  return nodes.filter((n) => keep.has(n.id)); // original order → stable layers
}

/** Where a memory node sits on screen: the core center plus its projection
 * coords scaled onto the constellation disc. */
export function memoryNodePos(
  n: { vx: number; vy: number },
  center: { x: number; y: number },
  radius: number,
): { x: number; y: number } {
  return { x: center.x + n.vx * radius, y: center.y + n.vy * radius };
}

// The constellation disc radius in graph units. The component renders the
// disc at R_CORE + 10 and the rest layout parks the pillar ring outside it —
// tests assert the clearance so they can never overlap.
export const R_CORE = 52;

// ── cinematic camera ─────────────────────────────────────────────────────────

export type Rect = { x: number; y: number; w: number; h: number };
export type ViewSize = { w: number; h: number };

export type CameraState = {
  /** a department tree is focused (frames the whole canvas — the tree fills it) */
  focusedTeam: boolean;
  /** the the operator memory core is expanded */
  coreExpanded: boolean;
  /** live position of the core (the self anchor moves in tree mode) */
  coreCenter: { x: number; y: number };
  /** live position of the selected org node (task / worker / tool), if any */
  selectedNodePos?: { x: number; y: number } | null;
  /** live position of the selected memory note inside the core, if any */
  memorySelectedPos?: { x: number; y: number } | null;
};

// zoom widths as a fraction of the canvas: org-node close-up, core dive,
// memory-note close-up (each keeps the canvas aspect ratio).
// The operator 2026-07-30: the node dive read "a tad too zoomed in when clicked" —
// breathed out a touch (0.55 → 0.62) so the neighbours stay in frame.
const ZOOM_NODE = 0.62;
// Inside a focused tree a node click should NOT crop the tree to a fragment,
// but the operator still wants "a slight zoom on what I'm looking at, in the center
// of the screen" — so a gentle zoom that recentres the clicked node on screen.
const ZOOM_NODE_SOFT = 0.8;
// Core dive. The operator 2026-07-30: "don't zoom in as far — 25% zoom reduction" on
// the middle Obsidian core, so the 2x dive (0.5) eased to 1.5x (0.667).
const ZOOM_CORE = 0.667;
const ZOOM_NOTE = 0.22;

const round2 = (n: number): number => {
  const v = Math.round(n * 100) / 100;
  return Object.is(v, -0) ? 0 : v;
};

/** A `frac`-of-the-canvas rect centered on `c`, clamped fully inside the canvas. */
function frameOn(view: ViewSize, c: { x: number; y: number }, frac: number): Rect {
  const w = view.w * frac;
  const h = w * (view.h / view.w); // keep canvas aspect
  const x = Math.max(0, Math.min(view.w - w, c.x - w / 2));
  const y = Math.max(0, Math.min(view.h - h, c.y - h / 2));
  return { x: round2(x), y: round2(y), w: round2(w), h: round2(h) };
}

/**
 * Where the camera should be for the current selection. Priority runs inside
 * out: a selected memory note beats the core dive, the core dive beats an org
 * selection, an org selection beats the resting/tree full frame.
 */
// the resting/tree frame breathes out a touch beyond the canvas (the operator,
// 2026-07-12: "a bit more zoomed out") — it also reveals the flank trees'
// off-canvas sweep, so the wheel reads bigger than the frame
const ZOOM_OUT_PAD = 0.06;

export function cameraRect(view: ViewSize, s: CameraState): Rect {
  if (s.coreExpanded) {
    if (s.memorySelectedPos) return frameOn(view, s.memorySelectedPos, ZOOM_NOTE);
    return frameOn(view, s.coreCenter, ZOOM_CORE);
  }
  // A node click zooms onto it. On the home wheel that's the full clamped dive.
  // Inside a focused tree, panning the camera onto the clicked node swung the
  // frame off-canvas near the tree edges — it "bugged out" (the operator 2026-07-30).
  // So a tree click gives a gentle zoom held on the SCREEN CENTER (no pan): the
  // tree stays put and just breathes in a touch, and the card + selection ring
  // carry the feedback.
  if (s.selectedNodePos) {
    return s.focusedTeam
      ? frameOn(view, { x: view.w / 2, y: view.h / 2 }, ZOOM_NODE_SOFT)
      : frameOn(view, s.selectedNodePos, ZOOM_NODE);
  }
  return {
    x: round2(-view.w * ZOOM_OUT_PAD),
    y: round2(-view.h * ZOOM_OUT_PAD),
    w: round2(view.w * (1 + 2 * ZOOM_OUT_PAD)),
    h: round2(view.h * (1 + 2 * ZOOM_OUT_PAD)),
  };
}

/**
 * One easing step toward the target rect. `t` is the per-frame catch-up factor
 * (≈0.1 feels like a camera operator; 1 snaps — the reduced-motion branch).
 * Returns the exact target when close enough so the camera settles instead of
 * asymptoting forever.
 */
export function lerpRect(cur: Rect, target: Rect, t: number): Rect {
  if (t >= 1) return target;
  const done =
    Math.abs(cur.x - target.x) < 0.05 &&
    Math.abs(cur.y - target.y) < 0.05 &&
    Math.abs(cur.w - target.w) < 0.05 &&
    Math.abs(cur.h - target.h) < 0.05;
  if (done) return target;
  return {
    x: cur.x + (target.x - cur.x) * t,
    y: cur.y + (target.y - cur.y) * t,
    w: cur.w + (target.w - cur.w) * t,
    h: cur.h + (target.h - cur.h) * t,
  };
}
