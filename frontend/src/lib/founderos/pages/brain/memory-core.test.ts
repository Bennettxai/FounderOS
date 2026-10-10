// Ported from FounderOS v1 tests/memory-core.test.ts: the client half (rest
// tier, disc projection, camera). distillMemoryGraph runs server side in the
// port (pages/brain/constellation.go) and is tested there.
import { describe, expect, test } from 'vitest';
import { cameraRect, lerpRect, memoryNodePos, pickRestTier, R_CORE, type CameraState, type MemoryNode, type Rect } from './memory-core';
import { responsiveRingR } from './tree-layout';

const page = (id: string, over: Partial<MemoryNode> = {}): MemoryNode => ({
  id,
  type: 'page',
  label: id,
  folder: 'ideas',
  excerpt: `${id} excerpt`,
  wordCount: 100,
  vx: 0.1,
  vy: -0.2,
  cluster: 0,
  links: 0,
  ...over,
});
const hub = (folder: string): MemoryNode =>
  page(`folder:${folder}`, { type: 'folder', label: folder, folder, excerpt: '', wordCount: 0 });

describe('rest tier', () => {
  test('pickRestTier: a calm subset for the collapsed view — hubs, best-linked pages, sparse orphans', () => {
    const nodes = [
      hub('ideas'),
      ...Array.from({ length: 30 }, (_, i) => ({ ...page(`p${String(i).padStart(2, '0')}`), cluster: 0, links: 30 - i }) as MemoryNode),
      ...Array.from({ length: 12 }, (_, i) => ({ ...page(`o${String(i).padStart(2, '0')}`), cluster: 0, links: 0 }) as MemoryNode),
    ].map((n) => ({ ...n, cluster: (n as MemoryNode).cluster ?? 0, links: (n as MemoryNode).links ?? 0 }) as MemoryNode);
    const tier = pickRestTier(nodes, 10);
    const ids = new Set(tier.map((n) => n.id));
    expect(ids.has('folder:ideas')).toBe(true); // hubs always stay
    expect(ids.has('p00')).toBe(true); // best-linked pages stay
    expect(ids.has('p29')).toBe(false); // weakly-linked pages rest
    expect(tier.filter((n) => n.links === 0 && n.type === 'page').length).toBeLessThan(12); // orphans thinned, not gone
    expect(tier.filter((n) => n.links === 0 && n.type === 'page').length).toBeGreaterThan(0);
    // deterministic
    expect(pickRestTier(nodes, 10)).toEqual(tier);
  });

  test('pickRestTier spreads the mini graph around the whole disc, not just the dense blob', () => {
    // 48 heavily-linked pages piled into one sector plus 48 lightly-linked
    // pages evenly around the rim: link-rank alone would show one clump. The
    // mini Obsidian view must cover (nearly) every angular sector instead.
    const heavy = Array.from({ length: 48 }, (_, i) =>
      ({ ...page(`h${String(i).padStart(2, '0')}`, { vx: 0.5 + (i % 7) * 0.012, vy: 0.02 + Math.floor(i / 7) * 0.012 }), cluster: 0, links: 100 - i }) as MemoryNode);
    const light = Array.from({ length: 48 }, (_, i) => {
      const a = (i / 48) * 2 * Math.PI;
      return { ...page(`l${String(i).padStart(2, '0')}`, { vx: 0.7 * Math.cos(a), vy: 0.7 * Math.sin(a) }), cluster: 0, links: 1 } as MemoryNode;
    });
    const tier = pickRestTier([...heavy, ...light], 24);
    const sectors = new Set(
      tier.filter((n) => n.type === 'page').map((n) => Math.min(11, Math.floor(((Math.atan2(n.vy, n.vx) + Math.PI) / (2 * Math.PI)) * 12))),
    );
    expect(sectors.size, `pages cover only ${sectors.size}/12 sectors`).toBeGreaterThanOrEqual(10);
    // determinism holds under the spatial sampling too
    expect(pickRestTier([...heavy, ...light], 24)).toEqual(tier);
  });

  test('pickRestTier default budget stays mini: at most 96 linked pages', () => {
    const nodes = Array.from({ length: 300 }, (_, i) => {
      const a = (i / 300) * 2 * Math.PI;
      return { ...page(`n${String(i).padStart(3, '0')}`, { vx: 0.6 * Math.cos(a), vy: 0.6 * Math.sin(a) }), cluster: 0, links: 1 + (i % 9) } as MemoryNode;
    });
    const tier = pickRestTier(nodes);
    expect(tier.filter((n) => n.type === 'page' && n.links > 0).length).toBeLessThanOrEqual(96);
  });

  test('spacing contract: the resting disc clears the pillar ring by ≥ 10u', () => {
    // disc edge = R_CORE + 10 (component backdrop); pillar icon inner edge =
    // ring1 − 14 (team node radius). They must never overlap at rest.
    const ring1 = responsiveRingR(880, 600)[1];
    expect(ring1 - 14 - (R_CORE + 10)).toBeGreaterThanOrEqual(10);
  });
});

const VIEW = { w: 880, h: 600 };
const CORE = { x: 440, y: 300 };

const state = (over: Partial<CameraState> = {}): CameraState => ({
  focusedTeam: false,
  coreExpanded: false,
  coreCenter: CORE,
  selectedNodePos: null,
  memorySelectedPos: null,
  ...over,
});

describe('cameraRect', () => {
  test('rest → the full frame, breathed out a touch (the operator: a bit more zoomed out)', () => {
    const r = cameraRect(VIEW, state());
    expect(r.x).toBeLessThan(0);
    expect(r.y).toBeLessThan(0);
    expect(r.w).toBeGreaterThan(880);
    expect(r.w).toBeLessThan(880 * 1.2); // just a little, not a satellite view
    expect(r.w / r.h).toBeCloseTo(880 / 600, 5);
    // centered: equal margins on both sides
    expect(r.x + r.w / 2).toBeCloseTo(440, 5);
    expect(r.y + r.h / 2).toBeCloseTo(300, 5);
  });

  test('dept focus without a selection → the same breathed-out frame (the wheel fills it)', () => {
    expect(cameraRect(VIEW, state({ focusedTeam: true }))).toEqual(cameraRect(VIEW, state()));
  });

  test('selected org node on the home wheel → tighter frame centered on the node, canvas aspect kept', () => {
    const r = cameraRect(VIEW, state({ selectedNodePos: { x: 440, y: 300 } }));
    expect(r.w).toBeLessThan(880 * 0.7);
    expect(r.w / r.h).toBeCloseTo(880 / 600, 5);
    expect(r.x + r.w / 2).toBeCloseTo(440, 5);
    expect(r.y + r.h / 2).toBeCloseTo(300, 5);
  });

  // The operator 2026-07-30 (revised): inside a focused tree a full dive cropped the
  // tree, but he still wants "a slight zoom on what I'm looking at, in the
  // center of the screen". So a node click gives a GENTLE zoom that recentres
  // the clicked node exactly on screen center (un-clamped), not the full dive.
  test('selected node INSIDE a focused tree → gentle zoom held on screen center (no pan)', () => {
    const resting = cameraRect(VIEW, state({ focusedTeam: true }));
    const r = cameraRect(VIEW, state({ focusedTeam: true, selectedNodePos: { x: 300, y: 120 } }));
    // a real but SLIGHT zoom-in relative to the resting frame...
    expect(r.w).toBeLessThan(resting.w);
    expect(r.w).toBeCloseTo(880 * 0.8, 5);
    // ...gentler than the home-wheel dive (0.62)...
    expect(r.w).toBeGreaterThan(880 * 0.62);
    // ...and held on the SCREEN CENTER — it does NOT pan to the node (panning
    // swung the frame off-canvas near tree edges and glitched, the operator)
    expect(r.x + r.w / 2).toBeCloseTo(440, 5);
    expect(r.y + r.h / 2).toBeCloseTo(300, 5);
  });

  test('selected node near a corner → rect clamps inside the canvas', () => {
    const r = cameraRect(VIEW, state({ selectedNodePos: { x: 6, y: 4 } }));
    expect(r.x).toBeGreaterThanOrEqual(0);
    expect(r.y).toBeGreaterThanOrEqual(0);
    expect(r.x + r.w).toBeLessThanOrEqual(880);
    expect(r.y + r.h).toBeLessThanOrEqual(600);
  });

  test('expanded core → eased dive with breathing room (the operator: 25% less zoom)', () => {
    const r = cameraRect(VIEW, state({ coreExpanded: true }));
    expect(r.w).toBeCloseTo(880 * 0.667, 5);
    expect(r.x + r.w / 2).toBeCloseTo(CORE.x, 5);
    expect(r.y + r.h / 2).toBeCloseTo(CORE.y, 5);
  });

  test('selected memory note zooms closer than the expanded core', () => {
    const core = cameraRect(VIEW, state({ coreExpanded: true }));
    const note = cameraRect(VIEW, state({ coreExpanded: true, memorySelectedPos: { x: 430, y: 310 } }));
    expect(note.w).toBeLessThan(core.w);
    expect(note.x + note.w / 2).toBeCloseTo(430, 5);
  });
});

describe('memoryNodePos', () => {
  test('maps projection coords onto a disc around the core center', () => {
    const p = memoryNodePos({ vx: 0.5, vy: -1 }, { x: 440, y: 300 }, 70);
    expect(p).toEqual({ x: 440 + 0.5 * 70, y: 300 - 70 });
  });

  test('center coords land exactly on the core center', () => {
    expect(memoryNodePos({ vx: 0, vy: 0 }, { x: 100, y: 50 }, 70)).toEqual({ x: 100, y: 50 });
  });
});

describe('lerpRect', () => {
  const a: Rect = { x: 0, y: 0, w: 880, h: 600 };
  const b: Rect = { x: 200, y: 150, w: 300, h: 204.545 };

  test('t=1 snaps exactly to the target (reduced-motion branch)', () => {
    expect(lerpRect(a, b, 1)).toEqual(b);
  });

  test('converges onto the target under repeated small steps', () => {
    let cur = a;
    for (let i = 0; i < 120; i++) cur = lerpRect(cur, b, 0.12);
    expect(cur.x).toBeCloseTo(b.x, 1);
    expect(cur.y).toBeCloseTo(b.y, 1);
    expect(cur.w).toBeCloseTo(b.w, 1);
    expect(cur.h).toBeCloseTo(b.h, 1);
  });

  test('is stable once at the target', () => {
    expect(lerpRect(b, b, 0.12)).toEqual(b);
  });

  test('moves monotonically toward the target', () => {
    const one = lerpRect(a, b, 0.12);
    const two = lerpRect(one, b, 0.12);
    expect(Math.abs(two.x - b.x)).toBeLessThan(Math.abs(one.x - b.x));
    expect(Math.abs(two.w - b.w)).toBeLessThan(Math.abs(one.w - b.w));
  });
});
