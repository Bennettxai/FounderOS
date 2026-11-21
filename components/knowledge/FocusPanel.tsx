'use client';

/**
 * What you get when you click a record: what it is, what its numbers actually
 * mean, and every connection grouped by how we know about it — each one
 * carrying the evidence that produced it, so a claimed relationship can be
 * checked rather than believed.
 */
import { useMemo, useState } from 'react';
import type { KIEdge, KIEntity, EdgeKind } from '@/lib/knowledge-index';
import type { NodeMetrics } from '@/lib/knowledge-metrics';
import {
  CATEGORY_COLOR,
  EDGE_COLOR,
  EDGE_HINT,
  EDGE_LABEL,
  KIND_CATEGORY,
  KIND_LABEL,
  interpret,
} from '@/lib/knowledge-views';

const ORDER: EdgeKind[] = ['link', 'owns', 'mention', 'similar'];

export function FocusPanel({
  entity,
  metrics,
  edges,
  entities,
  sortedPr,
  medianDegree,
  onFocus,
  onClose,
}: {
  entity: KIEntity;
  metrics: Map<string, NodeMetrics>;
  edges: KIEdge[];
  entities: KIEntity[];
  sortedPr: number[];
  medianDegree: number;
  onFocus: (id: string) => void;
  onClose: () => void;
}) {
  const byId = useMemo(() => new Map(entities.map((e) => [e.id, e])), [entities]);
  const m = metrics.get(entity.id)!;
  // `null` means "not chosen yet" — it resolves to the biggest group below, so
  // an agent (which has no wikilinks) still opens showing something.
  const [openKind, setOpenKind] = useState<EdgeKind | null | undefined>(undefined);

  const connections = useMemo(() => {
    const out = new Map<EdgeKind, { other: KIEntity; edge: KIEdge }[]>();
    for (const e of edges) {
      if (e.a !== entity.id && e.b !== entity.id) continue;
      const otherId = e.a === entity.id ? e.b : e.a;
      const other = byId.get(otherId);
      if (!other) continue;
      (out.get(e.kind) ?? out.set(e.kind, []).get(e.kind)!).push({ other, edge: e });
    }
    for (const list of out.values()) list.sort((a, b) => b.edge.weight - a.edge.weight);
    return out;
  }, [edges, entity.id, byId]);

  const total = [...connections.values()].reduce((s, l) => s + l.length, 0);
  const biggest =
    ORDER.filter((k) => (connections.get(k)?.length ?? 0) > 0).sort(
      (a, b) => (connections.get(b)?.length ?? 0) - (connections.get(a)?.length ?? 0),
    )[0] ?? null;
  const effectiveOpen = openKind === undefined ? biggest : openKind;
  const color = CATEGORY_COLOR[KIND_CATEGORY[entity.kind] ?? 'system'];

  const stat = (label: string, value: string, hint: string) => (
    <div className="glass-well px-2.5 py-2" title={hint}>
      <div className="font-mono text-[14px] font-light text-os-text">{value}</div>
      <div className="mt-0.5 text-[10px] text-os-dim">{label}</div>
    </div>
  );

  return (
    <div
      className="kv-panel flex h-full w-[340px] shrink-0 flex-col overflow-hidden border-l"
      style={{ borderColor: 'var(--border)', background: 'color-mix(in srgb, var(--elevated) 96%, transparent)' }}
    >
      <div className="flex items-start gap-2 border-b border-os-border p-4">
        <span className="mt-[6px] h-2 w-2 shrink-0 rounded-full" style={{ background: color }} />
        <div className="min-w-0 flex-1">
          <div className="text-[10.5px] text-os-muted">
            {KIND_LABEL[entity.kind] ?? entity.kind} · {entity.cluster}
          </div>
          <div className="mt-0.5 break-words text-[15px] font-medium leading-snug text-os-text">{entity.label}</div>
        </div>
        <button onClick={onClose} className="font-mono text-[13px] text-os-dim hover:text-os-text" aria-label="Close">
          ×
        </button>
      </div>

      <div className="flex-1 overflow-y-auto">
        {entity.summary && <p className="px-4 pt-3 text-[11.5px] leading-relaxed text-os-muted">{entity.summary}</p>}

        {/* what the numbers mean, in words */}
        <div className="mx-4 mt-3 rounded-[11px] border px-3 py-2.5" style={{ borderColor: 'var(--accent-line)', background: 'var(--accent-soft)' }}>
          <p className="text-[11.5px] leading-relaxed text-os-text">{interpret(entity, m, { sortedPr, medianDegree })}</p>
        </div>

        <div className="grid grid-cols-2 gap-2 p-4">
          {stat('connections', String(m.degree), 'Total edges touching this record')}
          {stat('written', String(m.authoredDegree), 'Edges a person actually wrote, excluding inferred similarity')}
          {stat('authority', (m.pagerank * 1000).toFixed(1), 'PageRank ×1000 — how much of the graph flows through here')}
          {stat('brokerage', String(Math.round(m.betweenness)), 'Shortest paths between other records that pass through this one')}
          {entity.days !== null && stat('last edited', `${entity.days}d`, 'Days since the file was last modified')}
          {entity.words > 0 && stat('words', String(entity.words), 'Body length of the record')}
        </div>

        {entity.terms.length > 0 && (
          <div className="px-4 pb-3">
            <div className="mb-1.5 text-[10.5px] text-os-dim">What it is about (most distinctive terms)</div>
            <div className="flex flex-wrap gap-1">
              {entity.terms.map((t) => (
                <span key={t} className="rounded-[6px] border px-1.5 py-0.5 font-mono text-[10px] text-os-muted" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.12)' }}>
                  {t}
                </span>
              ))}
            </div>
          </div>
        )}

        {entity.source && (
          <div className="px-4 pb-3">
            <div className="mb-1 text-[10.5px] text-os-dim">Source</div>
            <div className="break-all font-mono text-[10px] text-os-muted">{entity.source}</div>
          </div>
        )}

        {/* connections */}
        <div className="border-t border-os-border">
          <div className="px-4 pt-3 text-[11px] text-os-muted">
            {total} connections, grouped by how we know
          </div>
          {ORDER.map((k) => {
            const list = connections.get(k);
            if (!list?.length) return null;
            const isOpen = effectiveOpen === k;
            return (
              <div key={k} className="border-b border-os-border/60">
                <button
                  className="flex w-full items-center gap-2 px-4 py-2.5 text-left hover:bg-[rgb(var(--cold-hover-rgb) / 0.06)]"
                  onClick={() => setOpenKind(isOpen ? null : k)}
                  title={EDGE_HINT[k]}
                >
                  <span className="h-1.5 w-4 rounded-full" style={{ background: EDGE_COLOR[k] }} />
                  <span className="flex-1 text-[11.5px] text-os-text">{EDGE_LABEL[k]}</span>
                  <span className="font-mono text-[11px] text-os-dim">{list.length}</span>
                  <span className="text-os-dim">{isOpen ? '−' : '+'}</span>
                </button>
                {isOpen && (
                  <div className="pb-2">
                    {list.slice(0, 24).map(({ other, edge }, i) => (
                      <button
                        key={`${other.id}-${i}`}
                        onClick={() => onFocus(other.id)}
                        className="block w-full px-4 py-1.5 text-left hover:bg-[rgb(var(--cold-hover-rgb) / 0.06)]"
                      >
                        <span className="flex items-center gap-1.5">
                          <span
                            className="h-1.5 w-1.5 shrink-0 rounded-full"
                            style={{ background: CATEGORY_COLOR[KIND_CATEGORY[other.kind] ?? 'system'] }}
                          />
                          <span className="truncate text-[11.5px] text-os-text">{other.label}</span>
                        </span>
                        <span className="mt-0.5 block pl-3 text-[10.5px] italic leading-snug text-os-dim">
                          {edge.evidence.length > 150 ? `${edge.evidence.slice(0, 149)}…` : edge.evidence}
                        </span>
                      </button>
                    ))}
                    {list.length > 24 && (
                      <div className="px-4 pt-1 font-mono text-[10px] text-os-dim">+{list.length - 24} more</div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>

      <style dangerouslySetInnerHTML={{ __html: `
        .kv-panel { animation: kv-panel-in var(--gl-dur-drawer) var(--gl-spring-drawer) both; }
        @keyframes kv-panel-in { from { transform: translateX(14px); } } /* a panel slides in; it never fades (glass brief §4) */
        @media (prefers-reduced-motion: reduce) { .kv-panel { animation: none; } }
      ` }} />
    </div>
  );
}
