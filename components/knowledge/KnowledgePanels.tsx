'use client';

/**
 * The non-graph views. Each one answers a question the network picture cannot:
 *
 *   Matrix    which clusters actually talk to each other (where the silos are)
 *   Timeline  when the knowledge was last touched, and what has gone cold
 *   Ranking   the same numbers as a sortable table — the least glamorous and
 *             most used view, and the one that makes the graph checkable
 *   Findings  the structural audit, each item naming the number behind it
 */
import { useMemo, useState } from 'react';
import type { KIEntity } from '@/lib/knowledge-index';
import type { Finding, GraphAnalysis, NodeMetrics } from '@/lib/knowledge-metrics';
import {
  CATEGORICAL,
  CATEGORY_COLOR,
  CATEGORY_LABEL,
  KIND_CATEGORY,
  KIND_LABEL,
  SEQ_MINT,
  rankRows,
  timelineBuckets,
  type RankRow,
} from '@/lib/knowledge-views';

// ─────────────────────────────────────────────────────────────────────────────
// Matrix
// ─────────────────────────────────────────────────────────────────────────────

export function MatrixView({
  analysis,
  onPickCluster,
  limit,
}: {
  analysis: GraphAnalysis;
  onPickCluster: (c: string) => void;
  /** Inline-embed cap: show only the `limit` most-connected clusters so a
   * quadratically-growing matrix never forces a section to nest a scrollbar.
   * Omit for the full-screen overlay, which draws every cluster. */
  limit?: number;
}) {
  const full = analysis.clusterMatrix;
  const capped = limit != null && full.clusters.length > limit;
  const { clusters, counts, internal } = useMemo(() => {
    if (!capped) return full;
    const totals = full.clusters.map((_, i) => full.internal[i] + full.counts[i].reduce((s, v) => s + v, 0));
    const kept = full.clusters
      .map((_, i) => i)
      .sort((a, b) => totals[b] - totals[a])
      .slice(0, limit)
      .sort((a, b) => a - b); // back to the lib's alphabetical order for a readable grid
    return {
      clusters: kept.map((i) => full.clusters[i]),
      counts: kept.map((i) => kept.map((j) => full.counts[i][j])),
      internal: kept.map((i) => full.internal[i]),
    };
  }, [full, capped, limit]);
  const [hover, setHover] = useState<{ i: number; j: number } | null>(null);
  const max = Math.max(1, ...counts.flat());

  // Sequential: one hue, light to dark. On this surface more = brighter.
  const shade = (v: number) => {
    if (v === 0) return 'rgb(var(--mist-rgb) / 0.035)';
    const t = Math.sqrt(v / max); // sqrt so the long tail is visible at all
    const idx = Math.min(SEQ_MINT.length - 1, Math.floor((1 - t) * SEQ_MINT.length));
    return SEQ_MINT[idx];
  };

  const cell = 26;
  const pad = 126;

  return (
    <div className={limit != null ? 'overflow-x-auto p-5' : 'h-full overflow-auto p-5'}>
      <div className="mb-3">
        <div className="text-[13px] text-os-text">Cluster connection matrix</div>
        <p className="mt-1 max-w-[70ch] text-[11.5px] leading-relaxed text-os-muted">
          Every edge between two clusters, counted. A bright cell means those two bodies of knowledge reference each
          other; a dark row means a cluster that mostly talks to itself. The diagonal is deliberately left out — a
          cluster is always densest internally, and including it would wash out the comparison that matters.
        </p>
        {capped && (
          <p className="mt-1.5 text-[10.5px] text-os-dim">
            Showing the {limit} most-connected of {full.clusters.length} clusters — open Full screen for the complete
            matrix.
          </p>
        )}
      </div>

      <div className="relative inline-block">
        <svg width={pad + clusters.length * cell + 20} height={pad + clusters.length * cell + 20}>
          {/* column labels */}
          {clusters.map((c, j) => (
            <text
              key={`col-${c}`}
              x={pad + j * cell + cell / 2}
              y={pad - 8}
              fontSize="10"
              fill={hover?.j === j ? 'var(--text)' : 'var(--text-2)'}
              textAnchor="start"
              transform={`rotate(-52 ${pad + j * cell + cell / 2} ${pad - 8})`}
            >
              {c}
            </text>
          ))}
          {/* rows */}
          {clusters.map((c, i) => (
            <g key={`row-${c}`}>
              <text
                x={pad - 10}
                y={pad + i * cell + cell / 2 + 3.5}
                fontSize="10.5"
                fill={hover?.i === i ? 'var(--text)' : 'var(--text-2)'}
                textAnchor="end"
                style={{ cursor: 'pointer' }}
                onClick={() => onPickCluster(c)}
              >
                {c}
              </text>
              {clusters.map((d, j) => {
                if (i === j) {
                  return (
                    <rect
                      key={d}
                      x={pad + j * cell + 1}
                      y={pad + i * cell + 1}
                      width={cell - 2}
                      height={cell - 2}
                      rx="3"
                      fill="rgb(var(--mist-rgb) / 0.05)"
                    />
                  );
                }
                const v = counts[i][j];
                return (
                  <rect
                    key={d}
                    x={pad + j * cell + 1}
                    y={pad + i * cell + 1}
                    width={cell - 2}
                    height={cell - 2}
                    rx="3"
                    fill={shade(v)}
                    fillOpacity={v === 0 ? 1 : 0.92}
                    stroke={hover?.i === i && hover?.j === j ? 'var(--text)' : 'none'}
                    strokeWidth="1"
                    onMouseEnter={() => setHover({ i, j })}
                    onMouseLeave={() => setHover(null)}
                  />
                );
              })}
            </g>
          ))}
        </svg>

        {hover && (
          <div
            className="pointer-events-none absolute z-10 rounded-[10px] border px-3 py-2"
            style={{
              left: pad + hover.j * cell + 30,
              top: pad + hover.i * cell - 6,
              background: 'color-mix(in srgb, var(--elevated) 96%, transparent)',
              borderColor: 'rgb(var(--mist-rgb) / 0.16)',
            }}
          >
            <div className="text-[11px] text-os-text">
              {clusters[hover.i]} ↔ {clusters[hover.j]}
            </div>
            <div className="font-mono text-[13px] text-os-accent">{counts[hover.i][hover.j]} edges</div>
          </div>
        )}
      </div>

      {/* scale legend */}
      <div className="mt-4 flex items-center gap-2">
        <span className="text-[10.5px] text-os-dim">fewer</span>
        {[...SEQ_MINT].reverse().map((c) => (
          <span key={c} className="h-3 w-8 rounded-[2px]" style={{ background: c }} />
        ))}
        <span className="text-[10.5px] text-os-dim">more · up to {max}</span>
      </div>

      <div className="mt-5 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {clusters
          .map((c, i) => ({
            c,
            out: counts[i].reduce((s, v) => s + v, 0),
            inside: internal[i],
          }))
          .sort((a, b) => a.out / Math.max(1, a.out + a.inside) - b.out / Math.max(1, b.out + b.inside))
          .slice(0, 6)
          .map(({ c, out, inside }) => {
            const pct = (out / Math.max(1, out + inside)) * 100;
            return (
              <div key={c} className="glass-well hoverable border border-os-border bg-os-surface p-3">
                <div className="flex items-baseline justify-between">
                  <span className="text-[12px] text-os-text">{c}</span>
                  <span className="font-mono text-[13px] font-light text-os-accent">{pct.toFixed(0)}%</span>
                </div>
                <div className="mt-1 text-[10.5px] text-os-dim">
                  of its {out + inside} connections leave the cluster
                </div>
              </div>
            );
          })}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Timeline
// ─────────────────────────────────────────────────────────────────────────────

export function TimelineView({ entities }: { entities: KIEntity[] }) {
  const buckets = useMemo(() => timelineBuckets(entities), [entities]);
  const [hover, setHover] = useState<number | null>(null);

  // Stack by the five validated categories, in fixed order.
  const series = ['page', 'skill', 'agent', 'client', 'system'];
  const rows = useMemo(
    () =>
      buckets.map((b) => {
        const counts: Record<string, number> = { page: 0, skill: 0, agent: 0, client: 0, system: 0 };
        return { ...b, counts };
      }),
    [buckets],
  );

  // recompute per-category counts (timelineBuckets groups by cluster, so redo it here by kind)
  const byMonth = useMemo(() => {
    const m = new Map<string, Record<string, number>>();
    const now = Date.now();
    for (const e of entities) {
      if (e.days === null) continue;
      const d = new Date(now - e.days * 86_400_000);
      const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
      const rec = m.get(key) ?? { page: 0, skill: 0, agent: 0, client: 0, system: 0 };
      const cat = KIND_CATEGORY[e.kind] ?? 'system';
      rec[cat] = (rec[cat] ?? 0) + 1;
      m.set(key, rec);
    }
    return m;
  }, [entities]);

  const data = rows.map((r) => ({ ...r, counts: byMonth.get(r.key) ?? r.counts }));
  const max = Math.max(1, ...data.map((d) => Object.values(d.counts).reduce((s, v) => s + v, 0)));

  const W = 980;
  const H = 320;
  const PAD_L = 34;
  const PAD_B = 30;
  const plotW = W - PAD_L - 10;
  const plotH = H - PAD_B - 12;
  const bw = plotW / Math.max(1, data.length);
  const bar = Math.max(3, bw * 0.66);

  const step = max <= 8 ? 2 : Math.ceil(max / 4 / 5) * 5;
  const grid = Array.from({ length: Math.floor(max / step) + 1 }, (_, i) => i * step);

  return (
    <div className="p-5">
      <div className="mb-1 text-[13px] text-os-text">When this knowledge was last touched</div>
      <p className="mb-4 max-w-[70ch] text-[11.5px] leading-relaxed text-os-muted">
        Each bar is one month, counting records whose file was last modified then. Records with no date on disk —
        database rows rather than files — are not plotted, because they would be a guess. A long tail on the left with
        nothing above it is knowledge that was written once and never revisited.
      </p>

      <div className="relative">
        <svg viewBox={`0 0 ${W} ${H}`} className="w-full" onMouseLeave={() => setHover(null)}>
          {grid.map((v) => {
            const y = 12 + plotH - (v / max) * plotH;
            return (
              <g key={v}>
                <line x1={PAD_L} x2={W - 10} y1={y} y2={y} stroke="rgb(var(--mist-rgb) / 0.06)" />
                <text x={PAD_L - 8} y={y + 3} textAnchor="end" fontSize="9.5" fill="var(--text-3)">
                  {v}
                </text>
              </g>
            );
          })}
          {data.map((d, i) => {
            let acc = 0;
            const x = PAD_L + i * bw + (bw - bar) / 2;
            const total = Object.values(d.counts).reduce((s, v) => s + v, 0);
            return (
              <g key={d.key} onMouseEnter={() => setHover(i)}>
                <rect x={PAD_L + i * bw} y={12} width={bw} height={plotH} fill={hover === i ? 'rgb(var(--mist-rgb) / 0.05)' : 'transparent'} />
                {series.map((s) => {
                  const v = d.counts[s] ?? 0;
                  if (v === 0) return null;
                  const h = (v / max) * plotH;
                  const y = 12 + plotH - ((acc + v) / max) * plotH;
                  acc += v;
                  // 2px surface gap between stacked segments
                  return (
                    <rect
                      key={s}
                      x={x}
                      y={y}
                      width={bar}
                      height={Math.max(1.5, h - 2)}
                      rx="2"
                      fill={CATEGORY_COLOR[s]}
                      opacity={hover === null || hover === i ? 1 : 0.5}
                    />
                  );
                })}
                {total === 0 && null}
              </g>
            );
          })}
          {data.map((d, i) =>
            i % Math.ceil(data.length / 12) === 0 ? (
              <text key={`t-${d.key}`} x={PAD_L + i * bw} y={H - 8} fontSize="9.5" fill="var(--text-3)">
                {d.label}
              </text>
            ) : null,
          )}
        </svg>

        {hover !== null && (
          <div
            className="pointer-events-none absolute z-10 rounded-[11px] border p-3"
            style={{
              left: `${Math.min(78, Math.max(3, (hover / data.length) * 100))}%`,
              top: 4,
              background: 'color-mix(in srgb, var(--elevated) 96%, transparent)',
              borderColor: 'rgb(var(--mist-rgb) / 0.16)',
            }}
          >
            <div className="mb-1.5 text-[10.5px] text-os-muted">{data[hover].label}</div>
            {series
              .filter((s) => (data[hover].counts[s] ?? 0) > 0)
              .map((s) => (
                <div key={s} className="flex items-center gap-2 text-[11px] text-os-text">
                  <span className="h-2 w-2 rounded-full" style={{ background: CATEGORY_COLOR[s] }} />
                  <span className="font-mono">{data[hover].counts[s]}</span>
                  <span className="text-os-muted">{CATEGORY_LABEL[s]}</span>
                </div>
              ))}
          </div>
        )}
      </div>

      <div className="mt-3 flex flex-wrap gap-3">
        {series
          .filter((s) => data.some((d) => (d.counts[s] ?? 0) > 0))
          .map((s) => (
            <span key={s} className="flex items-center gap-1.5 text-[11px] text-os-muted">
              <span className="h-2 w-2 rounded-full" style={{ background: CATEGORY_COLOR[s] }} />
              {CATEGORY_LABEL[s]}
            </span>
          ))}
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Ranking
// ─────────────────────────────────────────────────────────────────────────────

type SortKey = 'label' | 'kind' | 'cluster' | 'degree' | 'authored' | 'authority' | 'brokerage' | 'days' | 'words';

export function RankingView({
  entities,
  metrics,
  query,
  onFocus,
}: {
  entities: KIEntity[];
  metrics: Map<string, NodeMetrics>;
  query: string;
  onFocus: (id: string) => void;
}) {
  const [sort, setSort] = useState<SortKey>('authority');
  const [asc, setAsc] = useState(false);

  const rows = useMemo(() => rankRows(entities, metrics), [entities, metrics]);
  const q = query.trim().toLowerCase();
  const filtered = rows.filter(
    (r) => q === '' || r.label.toLowerCase().includes(q) || r.cluster.toLowerCase().includes(q) || r.kind.includes(q),
  );
  const sorted = [...filtered].sort((a, b) => {
    const av = a[sort];
    const bv = b[sort];
    let c: number;
    if (typeof av === 'string' && typeof bv === 'string') c = av.localeCompare(bv);
    else c = (Number(av ?? -1) || 0) - (Number(bv ?? -1) || 0);
    return asc ? c : -c;
  });

  const head = (k: SortKey, label: string, hint?: string) => (
    <th
      className="cursor-pointer select-none px-3 py-2 text-left font-normal text-os-muted hover:text-os-text"
      onClick={() => {
        if (sort === k) setAsc(!asc);
        else {
          setSort(k);
          setAsc(false);
        }
      }}
      title={hint}
    >
      {label}
      {sort === k && <span className="ml-1 text-os-accent">{asc ? '↑' : '↓'}</span>}
    </th>
  );

  const maxAuth = Math.max(...rows.map((r) => r.authority), 1e-9);
  const maxBrok = Math.max(...rows.map((r) => r.brokerage), 1);

  return (
    <div>
      <div className="px-5 pb-2 pt-5">
        <div className="text-[13px] text-os-text">Every record, ranked</div>
        <p className="mt-1 max-w-[76ch] text-[11.5px] leading-relaxed text-os-muted">
          The same numbers the graph is drawn from, sortable. <span className="text-os-text">Authority</span> is
          PageRank — how much of the graph eventually flows through a record.{' '}
          <span className="text-os-text">Brokerage</span> is betweenness — how often it sits on the shortest path
          between two others. High brokerage with low connections is the signature of a single point of failure. Both
          are computed on <span className="text-os-text">written</span> connections only: inferred vocabulary
          similarity is a way to find things, not evidence that anyone relies on them.
        </p>
      </div>
      <div className="overflow-x-auto">
      <table className="w-full min-w-[720px] text-[11.5px]">
        <thead className="bg-os-bg2/60">
          <tr className="border-b border-os-border">
            {head('label', 'Record')}
            {head('kind', 'Kind')}
            {head('cluster', 'Cluster')}
            {head('degree', 'Links', 'Total connections, inferred included')}
            {head('authored', 'Written', 'Connections a person actually wrote')}
            {head('authority', 'Authority', 'PageRank')}
            {head('brokerage', 'Brokerage', 'Betweenness centrality')}
            {head('days', 'Age', 'Days since last edit')}
          </tr>
        </thead>
        <tbody>
          {sorted.slice(0, 300).map((r) => (
            <tr
              key={r.id}
              className="cursor-pointer border-b border-os-border/50 hover:bg-[rgb(var(--cold-hover-rgb) / 0.06)]"
              onClick={() => onFocus(r.id)}
            >
              <td className="px-3 py-1.5 text-os-text">
                <span className="mr-2 inline-block h-1.5 w-1.5 rounded-full align-middle" style={{ background: CATEGORY_COLOR[KIND_CATEGORY[r.kind] ?? 'system'] }} />
                {r.label.length > 46 ? `${r.label.slice(0, 45)}…` : r.label}
              </td>
              <td className="px-3 py-1.5 text-os-muted">{KIND_LABEL[r.kind] ?? r.kind}</td>
              <td className="px-3 py-1.5 text-os-muted">{r.cluster}</td>
              <td className="px-3 py-1.5 font-mono text-os-text">{r.degree}</td>
              <td className="px-3 py-1.5 font-mono text-os-muted">{r.authored}</td>
              <td className="px-3 py-1.5">
                <span className="flex items-center gap-2">
                  <span className="h-1 w-14 overflow-hidden rounded-full bg-[rgb(var(--mist-rgb) / 0.08)]">
                    <span className="block h-full rounded-full" style={{ width: `${(r.authority / maxAuth) * 100}%`, background: SEQ_MINT[1] }} />
                  </span>
                  <span className="font-mono text-os-dim">{(r.authority * 1000).toFixed(1)}</span>
                </span>
              </td>
              <td className="px-3 py-1.5">
                <span className="flex items-center gap-2">
                  <span className="h-1 w-14 overflow-hidden rounded-full bg-[rgb(var(--mist-rgb) / 0.08)]">
                    <span className="block h-full rounded-full" style={{ width: `${(r.brokerage / maxBrok) * 100}%`, background: CATEGORICAL[2] }} />
                  </span>
                  <span className="font-mono text-os-dim">{Math.round(r.brokerage)}</span>
                </span>
              </td>
              <td className="px-3 py-1.5 font-mono text-os-muted">{r.days === null ? '—' : `${r.days}d`}</td>
            </tr>
          ))}
        </tbody>
      </table>
      </div>
      {sorted.length > 300 && (
        <div className="px-5 py-3 text-[11px] text-os-dim">
          Showing the first 300 of {sorted.length} — narrow with the search box.
        </div>
      )}
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// Findings
// ─────────────────────────────────────────────────────────────────────────────

const SEV_COLOR: Record<Finding['severity'], string> = {
  high: 'var(--err)',
  medium: 'var(--warn)',
  low: 'var(--ramp-1-ink, var(--ramp-1))',
};

export function FindingsView({
  findings,
  entities,
  onFocus,
}: {
  findings: Finding[];
  entities: KIEntity[];
  onFocus: (id: string) => void;
}) {
  const byId = useMemo(() => new Map(entities.map((e) => [e.id, e])), [entities]);
  const [open, setOpen] = useState<string | null>(findings[0]?.id ?? null);

  return (
    <div className="p-5">
      <div className="mb-1 text-[13px] text-os-text">Structural audit</div>
      <p className="mb-4 max-w-[74ch] text-[11.5px] leading-relaxed text-os-muted">
        Everything below is derived from the graph, not asserted. Each finding names the measurement that produced it,
        so you can disagree with the threshold rather than the conclusion. Open one to see the records it is about.
      </p>

      <div className="grid gap-3">
        {findings.map((f) => {
          const isOpen = open === f.id;
          const targets = [...new Set(f.targets)];
          return (
            <div key={f.id} className="glass-well hoverable overflow-hidden border border-os-border bg-os-surface">
              <button className="flex w-full items-start gap-3 p-4 text-left" onClick={() => setOpen(isOpen ? null : f.id)}>
                <span className="mt-[5px] h-2 w-2 shrink-0 rounded-full" style={{ background: SEV_COLOR[f.severity] }} />
                <span className="min-w-0 flex-1">
                  <span className="block text-[13px] text-os-text">{f.title}</span>
                  <span className="mt-1 block text-[11.5px] leading-relaxed text-os-muted">{f.detail}</span>
                </span>
                <span className="shrink-0 font-mono text-[10px] uppercase tracking-wide" style={{ color: SEV_COLOR[f.severity] }}>
                  {f.severity}
                </span>
              </button>
              {isOpen && targets.length > 0 && (
                <div className="border-t border-os-border px-4 py-3">
                  <div className="mb-2 font-mono text-[10px] text-os-dim">
                    {targets.length < f.total
                      ? `showing ${Math.min(60, targets.length)} of ${f.total}`
                      : `${targets.length} records`}
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {targets.slice(0, 60).map((id) => {
                      const e = byId.get(id);
                      if (!e) return null;
                      return (
                        <button
                          key={id}
                          onClick={() => onFocus(id)}
                          className="rounded-[7px] border px-2 py-1 text-[10.5px] text-os-muted transition-colors hover:bg-[rgb(var(--cold-hover-rgb) / 0.1)] hover:text-os-text"
                          style={{ borderColor: 'rgb(var(--mist-rgb) / 0.12)' }}
                        >
                          <span
                            className="mr-1.5 inline-block h-1.5 w-1.5 rounded-full align-middle"
                            style={{ background: CATEGORY_COLOR[KIND_CATEGORY[e.kind] ?? 'system'] }}
                          />
                          {e.label.length > 34 ? `${e.label.slice(0, 33)}…` : e.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
