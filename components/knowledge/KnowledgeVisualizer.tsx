'use client';

/**
 * The Knowledge Visualizer: one long page, not a window-in-a-window. Five
 * sections flow down it — Network (the centerpiece), Matrix, Timeline,
 * Ranking, Findings — each auto-height, none of them nesting a scrollbar of
 * its own. The old build switched between five tabs inside one fixed-height
 * frame; the operator's read on it: "the network and the timeline and the matrix
 * exist in this window that you can scroll through... it makes scrolling the
 * entire page really annoying... the network should be the main centerpiece
 * and you shouldn't be able to scroll up and down on it." This is that.
 *
 * The network keeps a fixed, generous height and never captures the wheel —
 * the page scrolls straight through it, same contract as the Optimal Engine inline
 * embed. Hover still glides the camera; a click still opens a record's
 * neighbourhood; a "Full screen" button hands the wheel over for real pan and
 * zoom. The cluster matrix is the one view that grows quadratically with the
 * graph, so it gets the same full-screen affordance rather than a forced
 * inner scrollbar — summarized inline, complete behind the button.
 *
 * Optimal Engine is untouched — that page answers "what is in the store". This one
 * answers "how is it all connected, and which parts of it are load-bearing".
 */
import { useEffect, useMemo, useState } from 'react';
import { Maximize2, Minimize2 } from 'lucide-react';
import type { KIEdge, KIEntity, EdgeKind, KnowledgeIndex } from '@/lib/knowledge-index';
import type { GraphAnalysis, NodeMetrics } from '@/lib/knowledge-metrics';
import {
  AUTHORITY_BANDS,
  CATEGORY_COLOR,
  CATEGORY_HINT,
  CATEGORY_LABEL,
  EDGE_COLOR,
  EDGE_HINT,
  EDGE_LABEL,
  HEALTH_BANDS,
  RECENCY_BANDS,
  bandKey,
  type ColorDimension,
  type HealthLookup,
  type LegendEntry,
} from '@/lib/knowledge-views';
import { KnowledgeCanvas } from '@/components/knowledge/KnowledgeCanvas';
import { FindingsView, MatrixView, RankingView, TimelineView } from '@/components/knowledge/KnowledgePanels';
import { FocusPanel } from '@/components/knowledge/FocusPanel';
import { FullscreenOverlay } from '@/components/knowledge/FullscreenOverlay';
import { SectionHead } from '@/components/terminal';

const SECTIONS: { key: string; label: string; blurb: string }[] = [
  { key: 'network', label: 'Network', blurb: 'How everything connects' },
  { key: 'matrix', label: 'Matrix', blurb: 'Which clusters talk' },
  { key: 'timeline', label: 'Timeline', blurb: 'What has gone cold' },
  { key: 'ranking', label: 'Ranking', blurb: 'The numbers, sortable' },
  { key: 'findings', label: 'Findings', blurb: 'The structural audit' },
];

const DIMS: { key: ColorDimension; label: string; blurb: string }[] = [
  { key: 'kind', label: 'Kind', blurb: 'What sort of record it is' },
  { key: 'recency', label: 'Recency', blurb: 'How long since it was edited' },
  { key: 'authority', label: 'Authority', blurb: 'How much the graph leans on it' },
  { key: 'health', label: 'Health', blurb: 'What is structurally wrong with it' },
];

// Inline cap for the cluster matrix — see the comment on MatrixView's `limit`.
const MATRIX_INLINE_LIMIT = 12;

function scrollToSection(key: string) {
  document.getElementById(key)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}


export type KnowledgeAnalysis = {
  metrics: [string, NodeMetrics][];
  components: GraphAnalysis['components'];
  clusterMatrix: GraphAnalysis['clusterMatrix'];
  findings: GraphAnalysis['findings'];
  summary: GraphAnalysis['summary'];
};

/**
 * Colour dimension, legend filter, connection kinds, search and focus, shared
 * by both trees of /knowledge (GladOS: components/knowledge/KnowledgeG.tsx).
 */
export function useKnowledgeState(index: KnowledgeIndex, analysis: KnowledgeAnalysis) {
  const [dim, setDim] = useState<ColorDimension>('kind');
  const [hidden, setHidden] = useState<Set<string>>(new Set());
  const [activeKinds, setActiveKinds] = useState<Set<EdgeKind>>(new Set(['link', 'owns', 'mention']));
  const [query, setQuery] = useState('');
  const [focusId, setFocusId] = useState<string | null>(null);
  const metrics = useMemo(() => new Map(analysis.metrics), [analysis.metrics]);
  const byId = useMemo(() => new Map(index.entities.map((e) => [e.id, e])), [index.entities]);

  const sortedPr = useMemo(
    () => [...metrics.values()].map((m) => m.pagerank).sort((a, b) => b - a),
    [metrics],
  );
  const medianDegree = useMemo(() => {
    const d = [...metrics.values()].map((m) => m.degree).sort((a, b) => a - b);
    return d.length ? d[Math.floor(d.length / 2)] : 0;
  }, [metrics]);

  const health: HealthLookup = useMemo(() => {
    const f = (id: string) => new Set(analysis.findings.find((x) => x.id === id)?.targets ?? []);
    return { orphans: f('orphans'), staleHubs: f('stale-hubs'), bridges: f('single-points') };
  }, [analysis.findings]);

  const ctx = { sortedPr, health };
  const bandOf = (e: KIEntity, m: NodeMetrics) => bandKey(e, m, dim, ctx);

  // ── legend, which is also the filter ──────────────────────────────────────
  const legend: LegendEntry[] = useMemo(() => {
    const counts = new Map<string, number>();
    for (const e of index.entities) {
      const k = bandKey(e, metrics.get(e.id)!, dim, ctx);
      counts.set(k, (counts.get(k) ?? 0) + 1);
    }
    const shape = (keys: { key: string; label: string; color: string; hint?: string }[]) =>
      keys
        .map((k) => ({ ...k, count: counts.get(k.key) ?? 0 }))
        .filter((k) => k.count > 0);

    switch (dim) {
      case 'kind':
        return shape(
          Object.keys(CATEGORY_LABEL).map((k) => ({
            key: k,
            label: CATEGORY_LABEL[k],
            color: CATEGORY_COLOR[k],
            hint: CATEGORY_HINT[k],
          })),
        );
      case 'recency':
        return shape(RECENCY_BANDS.map((b) => ({ key: b.key, label: b.label, color: b.color })));
      case 'authority':
        return shape(AUTHORITY_BANDS.map((b) => ({ key: b.key, label: b.label, color: b.color })));
      case 'health':
        return shape(HEALTH_BANDS.map((b) => ({ key: b.key, label: b.label, color: b.color, hint: b.hint })));
    }
  }, [dim, index.entities, metrics, health, sortedPr]);

  const colorMap = useMemo(() => new Map(legend.map((l) => [l.key, l.color])), [legend]);
  const colorOf = (e: KIEntity, m: NodeMetrics) => colorMap.get(bandKey(e, m, dim, ctx)) ?? 'var(--text-3)';

  const focus = focusId ? byId.get(focusId) ?? null : null;

  const toggle = <T,>(set: Set<T>, v: T): Set<T> => {
    const next = new Set(set);
    if (next.has(v)) next.delete(v);
    else next.add(v);
    return next;
  };

  return { dim, setDim, hidden, setHidden, activeKinds, setActiveKinds, query, setQuery, focusId, setFocusId, metrics, byId, sortedPr, medianDegree, health, legend, colorOf, bandOf, focus, toggle };
}

export function KnowledgeVisualizer({
  index,
  analysis,
}: {
  index: KnowledgeIndex;
  analysis: {
    metrics: [string, NodeMetrics][];
    components: GraphAnalysis['components'];
    clusterMatrix: GraphAnalysis['clusterMatrix'];
    findings: GraphAnalysis['findings'];
    summary: GraphAnalysis['summary'];
  };
}) {
  const { dim, setDim, hidden, setHidden, activeKinds, setActiveKinds, query, setQuery, focusId, setFocusId, metrics, sortedPr, medianDegree, legend, colorOf, bandOf, focus, toggle } = useKnowledgeState(index, analysis);
  const [networkFs, setNetworkFs] = useState(false);
  const [matrixFs, setMatrixFs] = useState(false);

  // Esc + body-scroll-lock for the network's own full-screen mode — the
  // matrix's overlay handles its own via FullscreenOverlay.
  useEffect(() => {
    if (!networkFs) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setNetworkFs(false);
    window.addEventListener('keydown', onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = prev;
    };
  }, [networkFs]);

  const openRecord = (id: string) => {
    setFocusId(id);
    scrollToSection('network');
  };

  const s = analysis.summary;

  if (!index.mounted && index.entities.length === 0) {
    return (
      <div className="rounded-lg-t border border-os-border bg-os-surface p-8 text-center">
        <div className="text-[14px] text-os-text">Nothing to index</div>
        <p className="mx-auto mt-2 max-w-[52ch] text-[12px] leading-relaxed text-os-muted">
          The knowledge store is not mounted and the database has no records, so there is no graph to draw. This page
          shows an empty state rather than an invented one.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-10">
      {/* ── summary strip ───────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-6">
        {[
          { v: String(s.nodes), l: 'records indexed', h: 'Pages, skills, agents, accounts, workflows, tools and departments' },
          { v: String(s.edges), l: 'connections', h: 'Every edge, including inferred similarity' },
          { v: s.avgDegree.toFixed(1), l: 'avg connections', h: 'Mean degree across the graph' },
          {
            v: `${s.largestComponentPct.toFixed(0)}%`,
            l: 'in one network',
            h: `${s.components} disconnected groups, counting written connections only`,
          },
          { v: String(s.orphans), l: 'orphans', h: 'Records nothing an author wrote points to' },
          { v: s.medianDays === null ? '—' : `${s.medianDays}d`, l: 'median age', h: 'Median days since last edit, files only' },
        ].map((t) => (
          <div key={t.l} className="hoverable rounded-lg-t border border-os-border bg-os-surface px-3 py-2.5" title={t.h}>
            <div className="font-mono text-[17px] font-light tabular-nums text-os-text">{t.v}</div>
            <div className="mt-0.5 text-[10.5px] text-os-dim">{t.l}</div>
          </div>
        ))}
      </div>

      {/* ── in-page nav + search ────────────────────────────────────────── */}
      <div className="flex flex-wrap items-center gap-2">
        <div className="flex flex-wrap gap-1 rounded-[11px] border border-os-border bg-os-surface p-1">
          {SECTIONS.map((v) => (
            <button
              key={v.key}
              onClick={() => scrollToSection(v.key)}
              title={v.blurb}
              className="rounded-[8px] px-3 py-1.5 text-[12px] text-os-muted transition-colors hover:text-os-text"
            >
              {v.label}
            </button>
          ))}
        </div>

        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search records, clusters, summaries…"
          className="min-w-[220px] flex-1 rounded-[10px] border border-os-border bg-os-surface px-3 py-2 text-[12px] text-os-text outline-none placeholder:text-os-dim focus:border-[var(--accent-line)]"
        />

        {focusId && (
          <button
            onClick={() => setFocusId(null)}
            className="rounded-[10px] border px-3 py-2 text-[12px] text-os-accent transition-colors hover:bg-[var(--accent-soft)]"
            style={{ borderColor: 'var(--accent-line)' }}
          >
            Whole graph
          </button>
        )}
      </div>

      {/* ── network — the centerpiece ───────────────────────────────────── */}
      <section id="network" className="scroll-mt-20">
        <SectionHead label="Network" count={`${s.nodes} records`} />
        <div
          className={
            networkFs
              ? 'fixed inset-0 z-[130] flex overflow-hidden'
              : 'relative flex overflow-hidden rounded-lg-t border border-os-border bg-os-surface'
          }
          style={{ height: networkFs ? '100vh' : '72vh', minHeight: networkFs ? undefined : 600, background: networkFs ? 'var(--bg)' : undefined }}
        >
          {/* controls rail */}
          <div className="w-[212px] shrink-0 overflow-y-auto border-r border-os-border p-3">
            <div className="text-[10.5px] uppercase tracking-wide text-os-dim">Colour by</div>
            <div className="mt-1.5 grid gap-1">
              {DIMS.map((d) => (
                <button
                  key={d.key}
                  onClick={() => {
                    setDim(d.key);
                    setHidden(new Set());
                  }}
                  title={d.blurb}
                  className="rounded-[8px] px-2 py-1.5 text-left text-[11.5px] transition-colors"
                  style={
                    dim === d.key
                      ? { background: 'var(--accent-soft)', color: 'var(--accent)' }
                      : { color: 'var(--text-2)' }
                  }
                >
                  {d.label}
                </button>
              ))}
            </div>

            <div className="mt-4 flex items-baseline justify-between">
              <span className="text-[10.5px] uppercase tracking-wide text-os-dim">Legend</span>
              {hidden.size > 0 && (
                <button onClick={() => setHidden(new Set())} className="text-[10px] text-os-accent">
                  reset
                </button>
              )}
            </div>
            <p className="mt-1 text-[10px] leading-snug text-os-dim">Click a swatch to hide that group.</p>
            <div className="mt-1.5 grid gap-0.5">
              {legend.map((l) => {
                const off = hidden.has(l.key);
                return (
                  <button
                    key={l.key}
                    onClick={() => setHidden(toggle(hidden, l.key))}
                    title={l.hint}
                    className="flex items-center gap-2 rounded-[7px] px-1.5 py-1 text-left transition-colors hover:bg-[rgb(var(--cold-hover-rgb) / 0.08)]"
                    aria-pressed={!off}
                    data-kv-off={off || undefined}
                    style={{ opacity: off ? 0.35 : 1 }}
                  >
                    <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ background: l.color }} />
                    <span className="flex-1 truncate text-[11px] text-os-muted">{l.label}</span>
                    <span className="font-mono text-[10px] text-os-dim">{l.count}</span>
                  </button>
                );
              })}
            </div>

            <div className="mt-4 text-[10.5px] uppercase tracking-wide text-os-dim">Connection kinds</div>
            <div className="mt-1.5 grid gap-0.5">
              {(['link', 'owns', 'mention', 'similar'] as EdgeKind[]).map((k) => {
                const on = activeKinds.has(k);
                const count = index.byEdgeKind[k] ?? 0;
                if (!count) return null;
                return (
                  <button
                    key={k}
                    onClick={() => setActiveKinds(toggle(activeKinds, k))}
                    title={EDGE_HINT[k]}
                    className="flex items-center gap-2 rounded-[7px] px-1.5 py-1 text-left transition-colors hover:bg-[rgb(var(--cold-hover-rgb) / 0.08)]"
                    aria-pressed={on}
                    data-kv-off={!on || undefined}
                    style={{ opacity: on ? 1 : 0.35 }}
                  >
                    <span className="h-1.5 w-4 shrink-0 rounded-full" style={{ background: EDGE_COLOR[k] }} />
                    <span className="flex-1 truncate text-[11px] text-os-muted">{EDGE_LABEL[k]}</span>
                    <span className="font-mono text-[10px] text-os-dim">{count}</span>
                  </button>
                );
              })}
            </div>
            <p className="mt-2 text-[10px] leading-snug text-os-dim">
              In the whole-graph view these are summed into the ribbons between clusters. Open one record to see the
              individual connections and the evidence behind each. Similarity is inferred rather than observed, so it
              stays off by default.
            </p>
          </div>

          {/* canvas — the full-screen toggle is anchored here, not to the
              whole frame, so it hugs the canvas/focus-panel boundary and
              never sits on top of the panel once a record is open */}
          <div className="relative min-w-0 flex-1">
            <KnowledgeCanvas
              entities={index.entities}
              edges={index.edges}
              metrics={metrics}
              colorOf={colorOf}
              bandOf={bandOf}
              activeKinds={activeKinds}
              hidden={hidden}
              query={query}
              focusId={focusId}
              onFocus={setFocusId}
              dim={dim}
              fullscreen={networkFs}
            />

            {/* full-screen toggle — same styling as the Optimal Engine inline embed's.
                Left corner: the right corner already carries the canvas's own
                click/scroll hint, and the right edge is where the focus panel
                docks once a record is open. */}
            <button
              onClick={() => setNetworkFs((v) => !v)}
              title={networkFs ? 'Exit full screen (esc)' : 'Full screen — pan + zoom'}
              className="absolute left-4 top-4 z-30 flex items-center gap-2 rounded-full border px-3 py-1.5 text-[11px]"
              style={{
                background: 'color-mix(in srgb, var(--elevated) 82%, transparent)',
                borderColor: networkFs ? 'color-mix(in srgb, var(--accent) 45%, transparent)' : 'rgb(var(--mist-rgb) / 0.14)',
                color: networkFs ? 'var(--accent)' : 'var(--text-2)',
                backdropFilter: 'blur(10px)',
              }}
            >
              {networkFs ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
              {networkFs ? 'Exit' : 'Full screen'}
            </button>
          </div>

          {/* focus panel */}
          {focus && (
            <FocusPanel
              entity={focus}
              metrics={metrics}
              edges={index.edges}
              entities={index.entities}
              sortedPr={sortedPr}
              medianDegree={medianDegree}
              onFocus={setFocusId}
              onClose={() => setFocusId(null)}
            />
          )}
        </div>
      </section>

      {/* ── matrix ───────────────────────────────────────────────────────── */}
      <section id="matrix" className="scroll-mt-20">
        <div className="mb-4 flex items-baseline justify-between gap-4">
          <SectionHead label="Matrix" count={`${analysis.clusterMatrix.clusters.length} clusters`} />
          {analysis.clusterMatrix.clusters.length > MATRIX_INLINE_LIMIT && (
            <button
              onClick={() => setMatrixFs(true)}
              className="flex shrink-0 items-center gap-2 rounded-[10px] border px-3 py-2 text-[12px] text-os-accent transition-colors hover:bg-[var(--accent-soft)]"
              style={{ borderColor: 'var(--accent-line)' }}
            >
              <Maximize2 className="h-3.5 w-3.5" /> Full screen
            </button>
          )}
        </div>
        <div className="rounded-lg-t border border-os-border bg-os-surface">
          <MatrixView analysis={analysis as unknown as GraphAnalysis} onPickCluster={setQuery} limit={MATRIX_INLINE_LIMIT} />
        </div>
      </section>
      {matrixFs && (
        <FullscreenOverlay title="Cluster connection matrix — complete" onClose={() => setMatrixFs(false)}>
          <MatrixView analysis={analysis as unknown as GraphAnalysis} onPickCluster={setQuery} />
        </FullscreenOverlay>
      )}

      {/* ── timeline ─────────────────────────────────────────────────────── */}
      <section id="timeline" className="scroll-mt-20">
        <SectionHead label="Timeline" />
        <div className="rounded-lg-t border border-os-border bg-os-surface">
          <TimelineView entities={index.entities} />
        </div>
      </section>

      {/* ── ranking ──────────────────────────────────────────────────────── */}
      <section id="ranking" className="scroll-mt-20">
        <SectionHead label="Ranking" count={`${index.entities.length} records`} />
        <div className="rounded-lg-t border border-os-border bg-os-surface">
          <RankingView entities={index.entities} metrics={metrics} query={query} onFocus={openRecord} />
        </div>
      </section>

      {/* ── findings ─────────────────────────────────────────────────────── */}
      <section id="findings" className="scroll-mt-20">
        <SectionHead label="Findings" count={`${analysis.findings.length} findings`} />
        <div className="rounded-lg-t border border-os-border bg-os-surface">
          <FindingsView findings={analysis.findings} entities={index.entities} onFocus={openRecord} />
        </div>
      </section>
    </div>
  );
}
