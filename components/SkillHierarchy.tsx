'use client';

/**
 * The skill hierarchy — WHERE skills live and under what guidance boundary,
 * in the Optimal Engine radial's visual language: mint core, pearl domain nodes,
 * group nodes hanging off them, and a leaf dot for every real SKILL.md on
 * disk. Movable and zoomable (shared useSvgCamera: hover lean, pinch zoom,
 * scroll pan, double-click reset).
 *
 * Three levels (2026-08-26): the library outgrew a flat scope ring, so the
 * `Domain/Group` path a skill declares is rendered as an actual tree —
 * core → domain → group → skill. Click a domain to focus its branch, a group
 * to isolate it, a skill for its file panel.
 *
 * Honest by construction: nodes come 1:1 from the cards the server read off
 * disk — counts, names, and paths are real; a search query dims non-matches.
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import type { SkillCard } from '@/components/SkillsGrid';
import { useSvgCamera, type CameraTarget } from '@/components/useSvgCamera';
import { usePalette } from '@/lib/use-palette';
import { GButton } from '@/components/g';

// Console-only branch coloring, unchanged. Under red, domain/group/leaf are
// keyed by DEPTH instead of branch (see domainColor/groupColor/leafColor
// below) — core red fading through orange to amber/cream at the leaves.
const PEARLS_CONSOLE = ['#00ffab', '#7fe8ff', '#b49cff', '#ffd47f', '#ff9ecb', '#9fffe0', '#cddbd5', '#8a90a8', '#7fd3a8', '#c9b6ff'];
const VB = 1200; // square viewBox span

// Deep-zoom-into-cluster maths (clicking into a busy
// domain/group should zoom far enough to actually read the leaf labels).
// The <svg viewBox="-640 -370 1280 740"> is centered on (0,0), and the
// camera <g> carries `translate(tx ty) scale(s)`, so a scene point P lands
// on screen at s*P + (tx,ty). Framing a bbox centered at (cx,cy) means
// tx = -s*cx, ty = -s*cy.
const FRAME_W = 1280;
const FRAME_H = 740;
const CLUSTER_FILL = 0.7; // cluster should occupy ~70% of the frame
const CLUSTER_PAD = 55; // scene-unit padding so node labels aren't clipped
const CLUSTER_MAX_SCALE = 4.2;
const CLUSTER_MIN_SCALE = 1; // deep zoom never zooms OUT past the overview

/** Pure: bounding box of scene points -> a camera target framing them at
 * ~70% of the viewBox, centered, clamped. Exported for testing. */
export function computeClusterTarget(points: { x: number; y: number }[]): CameraTarget | null {
  if (points.length === 0) return null;
  let minX = Infinity;
  let maxX = -Infinity;
  let minY = Infinity;
  let maxY = -Infinity;
  for (const p of points) {
    if (p.x < minX) minX = p.x;
    if (p.x > maxX) maxX = p.x;
    if (p.y < minY) minY = p.y;
    if (p.y > maxY) maxY = p.y;
  }
  minX -= CLUSTER_PAD;
  maxX += CLUSTER_PAD;
  minY -= CLUSTER_PAD;
  maxY += CLUSTER_PAD;
  const w = Math.max(1, maxX - minX);
  const h = Math.max(1, maxY - minY);
  const s = Math.min(
    CLUSTER_MAX_SCALE,
    Math.max(CLUSTER_MIN_SCALE, Math.min((FRAME_W * CLUSTER_FILL) / w, (FRAME_H * CLUSTER_FILL) / h)),
  );
  const cx = (minX + maxX) / 2;
  const cy = (minY + maxY) / 2;
  return { x: -cx * s, y: -cy * s, s };
}

type LeafNode = { card: SkillCard; x: number; y: number };
type GroupNode = { name: string; x: number; y: number; ang: number; skills: LeafNode[] };
type DomainNode = {
  name: string;
  ci: number;
  ang: number;
  x: number;
  y: number;
  groups: GroupNode[];
  count: number;
};

/**
 * Round layout coordinates to 2 decimals. Math.cos/sin/atan2 may differ in
 * the last bits between the server's V8 and the browser's, which made SVG
 * attributes mismatch on hydration ("-146.42853012591993" vs
 * "-146.4285301259199"). Rounded, both sides print the same string.
 */
export const r2 = (n: number): number => Math.round(n * 100) / 100;

/** "Domain/Group" → the two levels; a bare label becomes its own domain. */
function split(group: string): [string, string] {
  const i = group.indexOf('/');
  return i === -1 ? [group, group] : [group.slice(0, i), group.slice(i + 1)];
}

/**
 * Fan `count` leaves around a group centre so neighbours never collide. Leaves
 * fill concentric rings from the inside out; each ring holds only as many as
 * its arc can space LEAF_GAP px apart, and rings sit RING_STEP px apart so the
 * gap holds radially too (after the 0.82 vertical squash). Pure, so the
 * geometry is testable without rendering.
 */
export function leafPositions(
  count: number,
  ga: number,
  gx: number,
  gy: number,
  mode: 'compact' | 'bloom' = 'compact',
): { x: number; y: number }[] {
  const LEAF_GAP = 36;
  const MAX_SPREAD = 2.4; // radians, centred on the group's outward angle
  const RING0 = 54;
  const RING_STEP = 42;
  const out: { x: number; y: number }[] = [];
  if (mode === 'bloom') {
    // An open group blooms onto ONE arc: every leaf gets its own spoke and the
    // label reads outward along it, so the arc only needs LABEL_H of room per
    // leaf no matter how long the names are.
    const LABEL_H = 16;
    const BLOOM_SPREAD = 2.8;
    const radius = Math.max(RING0 + 20, ((count - 1) * LABEL_H) / BLOOM_SPREAD);
    const spread = count === 1 ? 0 : Math.min(BLOOM_SPREAD, ((count - 1) * LABEL_H) / radius);
    for (let m = 0; m < count; m++) {
      const la = ga + (count === 1 ? 0 : (m / (count - 1) - 0.5) * spread);
      out.push({ x: r2(gx + Math.cos(la) * radius), y: r2(gy + Math.sin(la) * radius * 0.82) });
    }
    return out;
  }
  for (let r = 0; out.length < count; r++) {
    const radius = RING0 + r * RING_STEP;
    const k = Math.min(Math.floor((radius * MAX_SPREAD) / LEAF_GAP) + 1, count - out.length);
    const spread = k === 1 ? 0 : ((k - 1) * LEAF_GAP) / radius;
    for (let m = 0; m < k; m++) {
      const la = ga + (k === 1 ? 0 : (m / (k - 1) - 0.5) * spread);
      out.push({ x: r2(gx + Math.cos(la) * radius), y: r2(gy + Math.sin(la) * radius * 0.82) });
    }
  }
  return out;
}

function buildTree(cards: SkillCard[]): DomainNode[] {
  // domain -> group -> cards
  const domains = new Map<string, Map<string, SkillCard[]>>();
  for (const c of cards) {
    const [d, g] = split(c.group);
    const groups = domains.get(d) ?? new Map<string, SkillCard[]>();
    const list = groups.get(g) ?? [];
    list.push(c);
    groups.set(g, list);
    domains.set(d, groups);
  }

  const countOf = (m: Map<string, SkillCard[]>) => [...m.values()].reduce((s, l) => s + l.length, 0);
  const names = [...domains.keys()].sort(
    (a, b) => countOf(domains.get(b)!) - countOf(domains.get(a)!) || a.localeCompare(b),
  );

  return names.map((name, i) => {
    const groups = domains.get(name)!;
    const ang = (i / names.length) * Math.PI * 2 - Math.PI / 2;
    // Ellipse: the canvas is wider than it is tall, so widen x and squash y.
    const x = r2(Math.cos(ang) * 236);
    const y = r2(Math.sin(ang) * 150);

    const gNames = [...groups.keys()].sort(
      (a, b) => (groups.get(b)?.length ?? 0) - (groups.get(a)?.length ?? 0) || a.localeCompare(b),
    );
    // A domain with six groups needs a wider fan than one with two, but the fan
    // can never close on itself or the branches cross the core.
    const gSpread = Math.min(2.2, 0.46 * gNames.length);
    const gNodes: GroupNode[] = gNames.map((gn, k) => {
      // groups fan outward from the domain, away from the core
      const ga = ang + (gNames.length === 1 ? 0 : (k / (gNames.length - 1) - 0.5) * gSpread);
      const gr = 112 + (k % 2) * 22;
      const gx = r2(x + Math.cos(ga) * gr);
      const gy = r2(y + Math.sin(ga) * gr * 0.82);

      const list = groups.get(gn)!;
      const pos = leafPositions(list.length, ga, gx, gy);
      const skills: LeafNode[] = list.map((card, j) => ({ card, x: pos[j].x, y: pos[j].y }));
      return { name: gn, x: gx, y: gy, ang: ga, skills };
    });

    return { name, ci: i % PEARLS_CONSOLE.length, ang, x, y, groups: gNodes, count: countOf(groups) };
  });
}

export function SkillHierarchy({
  cards,
  query,
  onOpen,
  skin,
  focusDomain,
  onDomain,
}: {
  cards: SkillCard[];
  query: string;
  onOpen: (id: string) => void;
  /**
   * `g`: the GladOS style (glados-ui SPEC S04 F-S3): every colour a --g-*
   * token, flat nodes and core (no pearl sheen, no hover halo), sans labels,
   * and the file panel on kit controls. Omitted: the Classic look, unchanged.
   */
  skin?: 'g';
  /** GladOS only: the domain the grid's tab selected; the tree follows it. */
  focusDomain?: string | null;
  /** GladOS only: a domain click (or clear) here, so the grid's tab follows. */
  onDomain?: (domain: string | null) => void;
}) {
  const G = skin === 'g';
  const tree = useMemo(() => buildTree(cards), [cards]);
  const palette = usePalette();
  const byDepth = palette.name !== 'console';
  // Console: unchanged per-branch cycling through the 10-color set. Every other
  // theme renders domain/group/leaf by DEPTH (uniform across branches) from its
  // own pearls, so the tree reads as concentric rings in that theme's family.
  const domainColor = (ci: number) => (G ? 'var(--g-accent)' : byDepth ? palette.pearls[0] : PEARLS_CONSOLE[ci % PEARLS_CONSOLE.length]);
  const groupColor = (ci: number) => (G ? 'var(--g-ink)' : byDepth ? palette.pearls[3] : PEARLS_CONSOLE[ci % PEARLS_CONSOLE.length]);
  const leafColor = (ci: number) => (G ? 'var(--g-muted)' : byDepth ? palette.pearls[6] : PEARLS_CONSOLE[ci % PEARLS_CONSOLE.length]);
  // GladOS: flat fills from tokens; Classic: the pearl gradients.
  const pearl = (kind: 'd' | 'g' | 'l', ci: number) => (G ? (kind === 'd' ? domainColor(ci) : kind === 'g' ? groupColor(ci) : leafColor(ci)) : `url(#skh-pearl-${kind}-${ci})`);
  const [focus, setFocusState] = useState<string | null>(null); // domain name
  const [openGroup, setOpenGroup] = useState<string | null>(null); // "Domain/Group"
  const [sel, setSel] = useState<SkillCard | null>(null);
  const onDomainRef = useRef(onDomain);
  onDomainRef.current = onDomain;
  const setFocus = (next: string | null) => {
    setFocusState(next);
    if (G) onDomainRef.current?.(next);
  };
  useEffect(() => {
    if (!G || focusDomain === undefined) return;
    setFocusState(focusDomain);
    if (focusDomain === null) setOpenGroup(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focusDomain]);
  // wheel=false: hovering glides the scene, but scrolling scrolls the page —
  // wheel capture made it hard to get past the section .
  const { svgRef, camGRef, setTarget } = useSvgCamera(VB, true, 1.13, false);

  // Deep zoom: opening a group zooms tight to just that group's nodes;
  // focusing a domain (no group open yet) zooms to the whole domain so its
  // groups + skills are readable; clearing both resets to the overview.
  // Re-selecting a different domain/group retargets automatically since this
  // effect depends on focus/openGroup.
  useEffect(() => {
    if (openGroup) {
      for (const d of tree) {
        for (const g of d.groups) {
          if (`${d.name}/${g.name}` !== openGroup) continue;
          const bloom = leafPositions(g.skills.length, g.ang, g.x, g.y, 'bloom');
          // Labels extend ~90 units past each bloomed dot: frame those too.
          const pts = [{ x: g.x, y: g.y }, ...bloom.map((s) => ({ x: s.x + (s.x - g.x) * 0.9, y: s.y + (s.y - g.y) * 0.9 }))];
          setTarget(computeClusterTarget(pts));
          return;
        }
      }
    }
    if (focus) {
      const d = tree.find((dm) => dm.name === focus);
      if (d) {
        const pts = [{ x: d.x, y: d.y }];
        for (const g of d.groups) {
          pts.push({ x: g.x, y: g.y });
          for (const s of g.skills) pts.push({ x: s.x, y: s.y });
        }
        setTarget(computeClusterTarget(pts));
        return;
      }
    }
    setTarget(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [focus, openGroup, tree]);

  const q = query.trim().toLowerCase();
  const matches = (c: SkillCard) =>
    q === '' ||
    c.name.toLowerCase().includes(q) ||
    c.description.toLowerCase().includes(q) ||
    c.id.toLowerCase().includes(q) ||
    c.group.toLowerCase().includes(q);

  const selPath = sel ? split(sel.group) : null;
  const selDomain = selPath ? tree.find((d) => d.name === selPath[0]) : null;

  return (
    <div className="skh relative h-full w-full overflow-hidden" data-skin={G ? 'g' : undefined}>
      {/* mint atmosphere — same air as the Optimal Engine radial (ember under red); none under GladOS (no glows) */}
      {!G && (
        <div
          className="pointer-events-none absolute inset-0"
          style={{ background: 'var(--skh-glow-bg)' }}
        />
      )}
      <svg
        ref={svgRef}
        viewBox="-640 -370 1280 740"
        className="absolute inset-0 h-full w-full"
        style={{ overflow: 'hidden' }}
        onDoubleClick={() => {
          // useSvgCamera's own dblclick listener resets the camera; clear
          // the selection state too so nothing re-targets it back in.
          setFocus(null);
          setOpenGroup(null);
          setSel(null);
        }}
      >
        <defs>
          {PEARLS_CONSOLE.map((_, i) => (
            <g key={i}>
              <radialGradient id={`skh-pearl-d-${i}`} cx="0.35" cy="0.3" r="0.85">
                <stop offset="0%" stopColor="#fff" stopOpacity="0.95" />
                <stop offset="28%" stopColor={domainColor(i)} />
                <stop offset="100%" stopColor={domainColor(i)} stopOpacity="0.55" />
              </radialGradient>
              <radialGradient id={`skh-pearl-g-${i}`} cx="0.35" cy="0.3" r="0.85">
                <stop offset="0%" stopColor="#fff" stopOpacity="0.95" />
                <stop offset="28%" stopColor={groupColor(i)} />
                <stop offset="100%" stopColor={groupColor(i)} stopOpacity="0.55" />
              </radialGradient>
              <radialGradient id={`skh-pearl-l-${i}`} cx="0.35" cy="0.3" r="0.85">
                <stop offset="0%" stopColor="#fff" stopOpacity="0.95" />
                <stop offset="28%" stopColor={leafColor(i)} />
                <stop offset="100%" stopColor={leafColor(i)} stopOpacity="0.55" />
              </radialGradient>
            </g>
          ))}
          <radialGradient id="skh-core" cx="0.38" cy="0.32" r="0.9">
            <stop offset="0%" stopColor="var(--brain-core-light)" />
            <stop offset="45%" style={{ stopColor: 'var(--accent)' }} />
            <stop offset="100%" stopColor="var(--brain-core-dark)" />
          </radialGradient>
        </defs>
        <g ref={camGRef}>
          {tree.map((d) => {
            const domainHasMatch = d.groups.some((g) => g.skills.some((l) => matches(l.card)));
            const dimDomain = (focus !== null && focus !== d.name) || (q !== '' && !domainHasMatch);
            return (
              <g key={d.name} className="skh-fade" opacity={dimDomain ? 0.18 : 1}>
                {/* core → domain */}
                <path
                  d={`M 0 0 C ${d.x * 0.4} ${d.y * 0.1}, ${d.x * 0.65} ${d.y * 0.75}, ${d.x} ${d.y}`}
                  fill="none"
                  stroke={domainColor(d.ci)}
                  strokeOpacity="0.32"
                  strokeWidth="1.4"
                />

                {d.groups.map((g) => {
                  const path = `${d.name}/${g.name}`;
                  const groupHasMatch = g.skills.some((l) => matches(l.card));
                  const dimGroup =
                    (openGroup !== null && openGroup !== path && focus === d.name) || (q !== '' && !groupHasMatch);
                  // Leaf labels only appear once a branch is deep-zoomed —
                  // 79 labels at once is unreadable, and unreadable is
                  // useless. Focusing the domain (camera zooms to fit the
                  // whole domain) reveals all its groups' labels; opening one
                  // group (camera zooms tighter) is already covered by that.
                  const isOpen = openGroup === path;
                  const hoverLabels = !isOpen && focus === d.name;
                  const bloom = isOpen ? leafPositions(g.skills.length, g.ang, g.x, g.y, 'bloom') : null;
                  return (
                    <g key={path} className="skh-fade" opacity={dimGroup ? 0.2 : 1}>
                      {/* domain → group */}
                      <path
                        d={`M ${d.x} ${d.y} Q ${(d.x + g.x) / 2} ${(d.y + g.y) / 2 - 12}, ${g.x} ${g.y}`}
                        fill="none"
                        stroke={domainColor(d.ci)}
                        strokeOpacity="0.24"
                        strokeWidth="1"
                      />
                      {g.skills.map((leaf, li) => {
                        const dim = q !== '' && !matches(leaf.card);
                        const isSel = sel?.id === leaf.card.id;
                        const lx = bloom ? bloom[li].x : leaf.x;
                        const ly = bloom ? bloom[li].y : leaf.y;
                        // Spoke direction in scene space (the fan is squashed vertically).
                        const spoke = r2((Math.atan2(ly - g.y, lx - g.x) * 180) / Math.PI);
                        const flip = Math.cos((spoke * Math.PI) / 180) < 0;
                        return (
                          <g key={leaf.card.id} className="skh-fade" opacity={dim ? 0.16 : 1}>
                            <line
                              className="skh-move"
                              x1={g.x}
                              y1={g.y}
                              x2={lx}
                              y2={ly}
                              stroke={leafColor(d.ci)}
                              strokeOpacity="0.16"
                              strokeDasharray="2 4"
                            />
                            <g
                              className="skh-move"
                              transform={`translate(${lx},${ly})`}
                              onClick={() => {
                                setSel(leaf.card);
                                setFocus(d.name);
                                setOpenGroup(path);
                              }}
                              style={{ cursor: 'pointer' }}
                            >
                              <g className="skh-node">
                                {!G && <circle r="13" fill={leafColor(d.ci)} opacity={isSel ? 0.2 : 0} className="skh-glow" />}
                                <circle r={isSel ? 5.2 : 3.9} fill={isSel && G ? 'var(--g-accent)' : pearl('l', d.ci)} />
                                {isSel && <circle r="9" fill="none" stroke={leafColor(d.ci)} strokeOpacity="0.7" strokeDasharray="1 2.2" />}
                              </g>
                              {isOpen ? (
                                <text
                                  transform={`rotate(${flip ? spoke + 180 : spoke})`}
                                  x={flip ? -9 : 9}
                                  y="3"
                                  textAnchor={flip ? 'end' : 'start'}
                                  fontSize="8.5"
                                  fill={G ? (isSel ? 'var(--g-ink)' : 'var(--g-muted)') : isSel ? '#f2f6f4' : 'rgb(var(--mist-rgb) / 0.78)'}
                                  style={{ pointerEvents: 'none' }}
                                >
                                  {leaf.card.id.length > 28 ? `${leaf.card.id.slice(0, 27)}…` : leaf.card.id}
                                </text>
                              ) : (
                                (hoverLabels || isSel) && (
                                  <text
                                    className={isSel ? undefined : 'skh-hover-label'}
                                    y="-10"
                                    textAnchor="middle"
                                    fontSize="8.5"
                                    fill={G ? (isSel ? 'var(--g-ink)' : 'var(--g-muted)') : isSel ? '#f2f6f4' : 'rgb(var(--mist-rgb) / 0.9)'}
                                    style={{ pointerEvents: 'none' }}
                                  >
                                    {leaf.card.id.length > 18 ? `${leaf.card.id.slice(0, 17)}…` : leaf.card.id}
                                  </text>
                                )
                              )}
                            </g>
                          </g>
                        );
                      })}

                      {/* group node */}
                      <g
                        transform={`translate(${g.x},${g.y})`}
                        onClick={() => {
                          setOpenGroup(openGroup === path ? null : path);
                          setFocus(d.name);
                        }}
                        style={{ cursor: 'pointer' }}
                      >
                        <g className="skh-node">
                          {!G && <circle r="17" fill={groupColor(d.ci)} opacity="0" className="skh-glow" />}
                          <circle r="6.6" fill={pearl('g', d.ci)} opacity="0.9" />
                        </g>
                        {(focus === d.name || openGroup === path || q !== '') && (
                          <text y="-12" textAnchor="middle" fontSize="9.5" fill={G ? 'var(--g-ink)' : 'rgba(226,236,232,0.9)'} style={{ pointerEvents: 'none' }}>
                            {g.name}
                            <tspan fill={G ? 'var(--g-muted)' : groupColor(d.ci)} className={G ? undefined : 'font-mono'} fontSize="8.5">
                              {'  '}
                              {g.skills.length}
                            </tspan>
                          </text>
                        )}
                      </g>
                    </g>
                  );
                })}

                {/* domain pearl */}
                <g
                  transform={`translate(${d.x},${d.y})`}
                  onClick={() => {
                    const next = focus === d.name ? null : d.name;
                    setFocus(next);
                    setOpenGroup(null);
                    if (!next) setSel(null);
                  }}
                  style={{ cursor: 'pointer' }}
                >
                  <g className="skh-node">
                    {!G && <circle r="26" fill={domainColor(d.ci)} opacity="0" className="skh-glow" />}
                    <circle r="15" fill="none" stroke={domainColor(d.ci)} strokeOpacity="0.4" strokeDasharray="1 2.4" />
                    <circle r="10" fill={pearl('d', d.ci)} />
                  </g>
                  <g transform="translate(16,-16)">
                    <rect
                      width={d.name.length * 6.2 + 34}
                      height="17"
                      rx="8.5"
                      fill={G ? 'var(--g-card-2)' : 'color-mix(in srgb, var(--elevated) 90%, transparent)'}
                      stroke={G ? 'none' : 'rgb(var(--mist-rgb) / 0.14)'}
                    />
                    <text x="9" y="12.2" fontSize="10.5" fill={G ? 'var(--g-ink)' : 'var(--text)'}>
                      {d.name}
                    </text>
                    <text x={d.name.length * 6.2 + 25} y="12.2" fontSize="9.5" textAnchor="end" style={{ fill: G ? 'var(--g-muted)' : 'var(--accent)' }} className={G ? undefined : 'font-mono'}>
                      {d.count}
                    </text>
                  </g>
                </g>
              </g>
            );
          })}

          {/* core */}
          <g
            onClick={() => {
              setFocus(null);
              setOpenGroup(null);
              setSel(null);
            }}
            style={{ cursor: 'pointer' }}
          >
            {G ? (
              <circle r="17" fill="var(--g-accent)" />
            ) : (
              <>
                <circle r="34" fill="none" stroke="color-mix(in srgb, var(--accent) 14%, transparent)" strokeDasharray="1 3.5" />
                <circle r="24" fill="color-mix(in srgb, var(--accent) 8%, transparent)" />
                <circle r="17" fill="url(#skh-core)" />
              </>
            )}
            <text y="52" textAnchor="middle" fontSize="12.5" fill={G ? 'var(--g-ink)' : 'var(--text)'} fontWeight="500">
              Skills
            </text>
            <text y="67" textAnchor="middle" fontSize="9" fill={G ? 'var(--g-muted)' : '#7e8c88'} className={G ? undefined : 'font-mono'}>
              {cards.length} on disk · {tree.length} domains
            </text>
          </g>
        </g>
      </svg>

      {/* GladOS: the file panel on the kit */}
      {G && sel && selDomain && selPath && (
        <div className="wg-skh-panel" role="region" aria-label={`${sel.name} location`}>
          <div className="wg-skh-panel-top">
            <span className="wg-cap">{sel.group}</span>
            <GButton variant="icon" size="compact" aria-label="Close" onClick={() => setSel(null)}>
              ×
            </GButton>
          </div>
          <div className="wg-skh-panel-name">{sel.name}</div>
          <div className="wg-skh-path" data-mono-ok="">
            <div>skills/</div>
            <div style={{ paddingLeft: 12 }}>├ {selPath[0]}</div>
            <div style={{ paddingLeft: 24 }}>├ {selPath[1]}</div>
            <div style={{ paddingLeft: 36 }}>└ {sel.id}/</div>
            <div style={{ paddingLeft: 52 }}>└ SKILL.md</div>
          </div>
          <p className="wg-cap">{sel.description}</p>
          <GButton variant="ink" size="sm" onClick={() => onOpen(sel.id)}>
            Read SKILL.md
          </GButton>
        </div>
      )}

      {/* vertical file-hierarchy panel */}
      {!G && sel && selDomain && selPath && (
        <div
          className="skh-panel absolute bottom-4 right-4 top-4 w-[290px] overflow-y-auto rounded-[16px] border p-4"
          style={{ background: 'color-mix(in srgb, var(--elevated) 92%, transparent)', borderColor: 'rgb(var(--mist-rgb) / 0.12)', backdropFilter: 'blur(10px)' }}
        >
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full" style={{ background: domainColor(selDomain.ci) }} />
            <span className="truncate text-[11px] text-os-muted">{sel.group}</span>
            <button onClick={() => setSel(null)} className="ml-auto font-mono text-[12px] text-os-dim hover:text-os-text" aria-label="Close">
              ×
            </button>
          </div>
          <div className="mt-1.5 text-[16px] font-medium leading-snug text-os-text">{sel.name}</div>

          {/* the vertical hierarchy — where this skill lives */}
          <div className="glass-well mt-3 border border-os-border bg-os-surface2 p-3 font-mono text-[10.5px] leading-[1.9]">
            <div className="text-os-dim">skills/</div>
            <div className="pl-3 text-os-muted">
              <span style={{ color: domainColor(selDomain.ci) }}>├</span> {selPath[0]}
            </div>
            <div className="pl-6 text-os-muted">
              <span style={{ color: domainColor(selDomain.ci) }}>├</span> {selPath[1]}
            </div>
            <div className="pl-9 text-os-text">
              <span style={{ color: domainColor(selDomain.ci) }}>└</span> {sel.id}/
            </div>
            <div className="pl-[52px] text-os-accent">└ SKILL.md</div>
          </div>

          <p className="mt-3 text-[11.5px] leading-relaxed text-os-muted">{sel.description}</p>
          <div className="mt-2 truncate font-mono text-[9.5px] text-os-dim" title={sel.meta}>
            {sel.meta}
          </div>
          <button
            onClick={() => onOpen(sel.id)}
            className="mt-3 w-full rounded-[9px] border px-3 py-2 text-[12px] font-medium text-os-accent transition-colors hover:bg-[var(--accent-soft)]"
            style={{ borderColor: 'var(--accent-line)' }}
          >
            Read SKILL.md ›
          </button>
        </div>
      )}

      <style dangerouslySetInnerHTML={{ __html: `
        .skh { animation: skh-in 0.45s cubic-bezier(0.2, 0.7, 0.2, 1) both; }
        @keyframes skh-in { from { transform: translateY(8px); } } /* holds the glass panel: rises, never fades */
        .skh-fade { transition: opacity 0.35s ease; }
        .skh-node { transition: transform 0.25s cubic-bezier(0.2, 0.9, 0.3, 1.4); transform-box: fill-box; transform-origin: center; }
        g:hover > .skh-node { transform: scale(1.25); }
        .skh-glow { transition: opacity 0.3s ease; }
        .skh-move { transition: transform 0.45s cubic-bezier(0.2, 0.7, 0.2, 1), x2 0.45s ease, y2 0.45s ease; }
        .skh-hover-label { opacity: 0; transition: opacity 0.15s ease; }
        g:hover > .skh-hover-label { opacity: 1; }
        g:hover > .skh-node .skh-glow { opacity: 0.3; }
        .skh-panel { animation: skh-panel-in 0.3s cubic-bezier(0.2, 0.7, 0.2, 1) both; }
        @keyframes skh-panel-in { from { transform: translateX(12px); } }
        @media (prefers-reduced-motion: reduce) {
          .skh, .skh-panel { animation: none; }
          .skh-node, .skh-fade, .skh-glow, .skh-move, .skh-hover-label { transition: none; }
        }
      ` }} />
    </div>
  );
}
