'use client';

/**
 * The graph canvas — overview and ego mode.
 *
 * Overview: angle = cluster, radius = authority (PageRank), size = degree.
 * Nothing is placed decoratively, so "near the middle" always means "the rest
 * of the graph depends on this".
 *
 * Ego mode: click any record and the canvas re-lays out around it in hop
 * rings, showing only what actually reaches it. This is the view that answers
 * "what is connected to this thing, and how do I know".
 */
import { useMemo } from 'react';
import type { KIEdge, KIEntity, EdgeKind } from '@/lib/knowledge-index';
import type { NodeMetrics } from '@/lib/knowledge-metrics';
import { EDGE_COLOR, egoNetwork, radialLayout, type ColorDimension, type Placed } from '@/lib/knowledge-views';
import { useSvgCamera } from '@/components/useSvgCamera';

const VB = 1240;

export type CanvasProps = {
  entities: KIEntity[];
  edges: KIEdge[];
  metrics: Map<string, NodeMetrics>;
  colorOf: (e: KIEntity, m: NodeMetrics) => string;
  bandOf: (e: KIEntity, m: NodeMetrics) => string;
  activeKinds: Set<EdgeKind>;
  /** legend keys switched off; empty = show everything */
  hidden: Set<string>;
  query: string;
  focusId: string | null;
  onFocus: (id: string | null) => void;
  dim: ColorDimension;
  /** Full screen is the only mode where the wheel pans/zooms the scene —
   * inline, the page owns the wheel so scrolling past the centerpiece keeps
   * scrolling the page . Hover glide and click-to-open stay on either way. */
  fullscreen?: boolean;
};

export function KnowledgeCanvas({
  entities,
  edges,
  metrics,
  colorOf,
  bandOf,
  activeKinds,
  hidden,
  query,
  focusId,
  onFocus,
  fullscreen = false,
}: CanvasProps) {
  const { svgRef, camGRef } = useSvgCamera(VB, true, 1.13, fullscreen);

  const view = useMemo(() => {
    if (focusId) {
      const ego = egoNetwork(focusId, entities, edges, metrics, { maxHops: 2, kinds: activeKinds });
      return {
        placed: ego.placed,
        edges: ego.edges,
        hops: ego.hops,
        labeled: ego.labeled,
        arcs: [] as { cluster: string; from: number; to: number; mid: number; count: number }[],
        ego: true,
      };
    }
    const { placed, arcs } = radialLayout(entities, metrics);
    return { placed, edges, hops: null as Map<string, number> | null, labeled: null as Set<string> | null, arcs, ego: false };
  }, [focusId, entities, edges, metrics, activeKinds]);

  const pos = useMemo(() => new Map(view.placed.map((p) => [p.id, p])), [view.placed]);

  const q = query.trim().toLowerCase();
  const matches = (e: KIEntity) =>
    q === '' || e.label.toLowerCase().includes(q) || e.cluster.toLowerCase().includes(q) || e.summary.toLowerCase().includes(q);

  const visible = (p: Placed) => !hidden.has(bandOf(p, p.m));

  // At overview density, 1,200 individual edges is a hairball that answers
  // nothing. Overview therefore draws CLUSTER RIBBONS — one curve per cluster
  // pair, thickness by how many edges run between them — which is the same
  // information the matrix view holds, drawn on the same rim the nodes sit on.
  // Individual edges, with their evidence, appear once you open one record.
  const ribbons = useMemo(() => {
    if (view.ego) return [];
    const arcAt = new Map(view.arcs.map((a) => [a.cluster, a]));
    const clusterOf = new Map(entities.map((e) => [e.id, e.cluster]));
    const pairs = new Map<string, { a: string; b: string; n: number }>();
    for (const e of edges) {
      if (!activeKinds.has(e.kind)) continue;
      const ca = clusterOf.get(e.a);
      const cb = clusterOf.get(e.b);
      if (!ca || !cb || ca === cb) continue;
      const key = ca < cb ? `${ca}|${cb}` : `${cb}|${ca}`;
      const rec = pairs.get(key) ?? { a: ca < cb ? ca : cb, b: ca < cb ? cb : ca, n: 0 };
      rec.n++;
      pairs.set(key, rec);
    }
    const R = 470;
    const out = [...pairs.values()]
      .map((p) => {
        const aa = arcAt.get(p.a);
        const ab = arcAt.get(p.b);
        if (!aa || !ab) return null;
        return {
          key: `${p.a}|${p.b}`,
          n: p.n,
          x1: Math.cos(aa.mid) * R,
          y1: Math.sin(aa.mid) * R * 0.72,
          x2: Math.cos(ab.mid) * R,
          y2: Math.sin(ab.mid) * R * 0.72,
        };
      })
      .filter((v): v is NonNullable<typeof v> => v !== null);
    return out.sort((x, y) => x.n - y.n);
  }, [view, edges, activeKinds, entities]);

  const maxRibbon = useMemo(() => Math.max(1, ...ribbons.map((r) => r.n)), [ribbons]);

  // Ego mode keeps every real edge — at one record's scale they are readable
  // and each one is the thing being explained.
  const drawnEdges = useMemo(() => {
    if (!view.ego) return [];
    const out: { e: KIEdge; a: Placed; b: Placed }[] = [];
    for (const e of view.edges) {
      if (!activeKinds.has(e.kind)) continue;
      const a = pos.get(e.a);
      const b = pos.get(e.b);
      if (!a || !b) continue;
      if (hidden.has(bandOf(a, a.m)) || hidden.has(bandOf(b, b.m))) continue;
      out.push({ e, a, b });
    }
    const order: Record<EdgeKind, number> = { similar: 0, mention: 1, owns: 2, link: 3 };
    return out.sort((x, y) => order[x.e.kind] - order[y.e.kind]);
  }, [view, pos, activeKinds, hidden, bandOf]);

  const hops = view.hops;

  return (
    <div className="kv-canvas relative h-full w-full overflow-hidden" style={{ background: 'var(--bg-2)' }}>
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            'var(--kc-glow-bg)',
        }}
      />
      <svg
        ref={svgRef}
        viewBox="-660 -350 1320 700"
        className="absolute inset-0 h-full w-full"
        style={{ overflow: 'hidden' }}
        onDoubleClick={() => onFocus(null)}
      >
        <g ref={camGRef}>
          {!view.ego &&
            view.arcs.map((a, i) => {
              const R = 505;
              const x1 = Math.cos(a.from) * R;
              const y1 = Math.sin(a.from) * R * 0.72;
              const x2 = Math.cos(a.to) * R;
              const y2 = Math.sin(a.to) * R * 0.72;
              const large = a.to - a.from > Math.PI ? 1 : 0;
              // stagger alternate labels outward so neighbouring arcs on a
              // crowded stretch of rim do not print on top of each other
              const push = R + 26 + (i % 2) * 15;
              const lx = Math.cos(a.mid) * push;
              const ly = Math.sin(a.mid) * push * 0.72;
              const anchor = Math.cos(a.mid) > 0.15 ? 'start' : Math.cos(a.mid) < -0.15 ? 'end' : 'middle';
              return (
                <g key={a.cluster}>
                  <path
                    d={`M ${x1} ${y1} A ${R} ${R * 0.72} 0 ${large} 1 ${x2} ${y2}`}
                    fill="none"
                    stroke="rgb(var(--kc-tint-rgb) / 0.16)"
                    strokeWidth="1"
                  />
                  <text x={lx} y={ly} fontSize="10.5" fill="rgb(var(--mist-rgb) / 0.7)" textAnchor={anchor}>
                    {a.cluster}
                    <tspan fill="var(--text-3)" className="font-mono" fontSize="9">
                      {'  '}
                      {a.count}
                    </tspan>
                  </text>
                </g>
              );
            })}

          {view.ego &&
            (() => {
              const rs = [...new Set(view.placed.filter((p) => (view.hops?.get(p.id) ?? 0) > 0).map((p) => Math.round(Math.hypot(p.x, p.y / 0.72) / 10) * 10))];
              const r1 = Math.min(...rs.filter((r) => r > 0), 190);
              const r2 = Math.max(...rs, r1 + 130);
              return [r1, r2];
            })().map((r, i) => (
              <g key={r}>
                <ellipse cx="0" cy="0" rx={r} ry={r * 0.72} fill="none" stroke="rgb(var(--kc-tint-rgb) / 0.13)" strokeDasharray="2 5" />
                <text x={r + 8} y="-4" fontSize="9.5" fill="var(--text-3)" className="font-mono">
                  {i + 1} hop{i ? 's' : ''}
                </text>
              </g>
            ))}

          {/* cluster ribbons — overview only */}
          <g>
            {ribbons.map((r) => {
              const t = r.n / maxRibbon;
              return (
                <path
                  key={r.key}
                  d={`M ${r.x1} ${r.y1} Q 0 0, ${r.x2} ${r.y2}`}
                  fill="none"
                  stroke="var(--kc-accent)"
                  strokeOpacity={0.08 + Math.sqrt(t) * 0.3}
                  strokeWidth={0.5 + Math.sqrt(t) * 5}
                  strokeLinecap="round"
                />
              );
            })}
          </g>

          <g>
            {drawnEdges.map(({ e, a, b }, i) => {
              const faded =
                q !== '' && !matches(a) && !matches(b) ? 0.05 : e.kind === 'similar' ? 0.22 : e.kind === 'mention' ? 0.3 : 0.45;
              const mx = (a.x + b.x) / 2;
              const my = (a.y + b.y) / 2;
              const k = view.ego ? 0.86 : 0.62;
              return (
                <path
                  key={`${e.a}|${e.b}|${e.kind}|${i}`}
                  d={`M ${a.x} ${a.y} Q ${mx * k} ${my * k}, ${b.x} ${b.y}`}
                  fill="none"
                  stroke={EDGE_COLOR[e.kind]}
                  strokeOpacity={faded}
                  strokeWidth={e.kind === 'link' ? 0.9 : 0.6}
                />
              );
            })}
          </g>

          <g>
            {view.placed.map((p) => {
              const shown = visible(p);
              const hit = matches(p);
              const hop = hops?.get(p.id) ?? 0;
              const isFocus = p.id === focusId;
              const opacity = !shown ? 0.06 : q !== '' && !hit ? 0.12 : view.ego && hop === 2 ? 0.72 : 1;
              const color = colorOf(p, p.m);
              return (
                <g
                  key={p.id}
                  transform={`translate(${p.x},${p.y})`}
                  opacity={opacity}
                  className="kv-fade"
                  onClick={(ev) => {
                    ev.stopPropagation();
                    onFocus(p.id);
                  }}
                  style={{ cursor: shown ? 'pointer' : 'default' }}
                >
                  <g className="kv-node">
                    <circle r={p.r * 2.6} fill={color} opacity="0" className="kv-glow" />
                    {isFocus && <circle r={p.r + 9} fill="none" stroke={color} strokeOpacity="0.65" strokeDasharray="1.5 3" />}
                    <circle r={p.r} fill={color} fillOpacity={p.m.authoredDegree === 0 ? 0.35 : 0.92} />
                    {p.m.articulation && !view.ego && (
                      <circle r={p.r + 3.2} fill="none" stroke={color} strokeOpacity="0.5" strokeWidth="0.8" />
                    )}
                  </g>
                  {(isFocus ||
                    (view.ego && (view.labeled?.has(p.id) ?? hop <= 1)) ||
                    (!view.ego && p.r > 6.4) ||
                    (q !== '' && hit)) && (
                    <text
                      y={-p.r - 6}
                      textAnchor="middle"
                      fontSize={isFocus ? 12 : 9}
                      fill={isFocus ? 'var(--text)' : 'color-mix(in srgb, var(--text) 82%, transparent)'}
                      style={{ pointerEvents: 'none' }}
                    >
                      {p.label.length > 26 ? `${p.label.slice(0, 25)}…` : p.label}
                    </text>
                  )}
                </g>
              );
            })}
          </g>
        </g>
      </svg>

      {/* Hidden once a record is focused: the panel's own close button and the
          "Whole graph" control above already cover getting back, and at a
          narrow width + open panel the canvas is too tight for this text to
          share a corner with the full-screen toggle. */}
      {!focusId && (
        <div className="pointer-events-none absolute right-4 top-3 max-w-[230px] text-right font-mono text-[9.5px] leading-relaxed text-os-dim">
          {view.ego ? 'double-click to return to the whole graph' : 'click any node to open its network'}
          {fullscreen ? ' · pinch zoom · scroll pan · esc to exit' : ' · scroll to keep reading the page'}
        </div>
      )}

      <style dangerouslySetInnerHTML={{ __html: `
        .kv-canvas { animation: kv-in 0.4s cubic-bezier(0.2,0.7,0.2,1) both; }
        @keyframes kv-in { from { transform: translateY(6px); } } /* holds glass chips: rises, never fades */
        .kv-canvas > svg { animation: kv-leaf-in 0.4s cubic-bezier(0.2,0.7,0.2,1) both; }
        @keyframes kv-leaf-in { from { opacity: 0; } } /* the graph itself is a leaf */
        .kv-fade { transition: opacity 0.35s ease; }
        .kv-node { transition: transform 0.22s cubic-bezier(0.2,0.9,0.3,1.4); transform-box: fill-box; transform-origin: center; }
        g:hover > .kv-node { transform: scale(1.4); }
        .kv-glow { transition: opacity 0.25s ease; }
        g:hover > .kv-node .kv-glow { opacity: 0.28; }
        @media (prefers-reduced-motion: reduce) {
          .kv-canvas, .kv-canvas > svg { animation: none; }
          .kv-node, .kv-fade, .kv-glow { transition: none; }
        }
      ` }} />
    </div>
  );
}
