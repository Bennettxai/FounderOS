'use client';

import { useEffect, useState } from 'react';

type Hit = { title: string; snippet: string };

/**
 * "In the brain" rail panel for the knowledge graph (2026-08-24 spice pass):
 * selecting a node runs its name through GET /api/brain — real hybrid
 * retrieval — and lists the store pages that actually mention it. Honest by
 * construction: results are live, and no matches says so plainly.
 */
export function GraphBrainContext({ label }: { label: string | null }) {
  const [state, setState] = useState<{ q: string; hits: Hit[] | null } | null>(null);

  useEffect(() => {
    if (!label) {
      setState(null);
      return;
    }
    let alive = true;
    setState({ q: label, hits: null });
    const t = setTimeout(() => {
      fetch(`/api/brain?q=${encodeURIComponent(label)}`)
        .then((r) => (r.ok ? r.json() : null))
        .then((j: { results?: Hit[] } | null) => {
          if (alive) setState({ q: label, hits: j?.results?.slice(0, 4) ?? [] });
        })
        .catch(() => alive && setState({ q: label, hits: [] }));
    }, 250);
    return () => {
      alive = false;
      clearTimeout(t);
    };
  }, [label]);

  if (!label || !state) return null;
  return (
    <div className="border-t border-os-border pt-3">
      <div className="mb-1.5 flex items-baseline justify-between">
        <span className="text-[12.5px] font-medium text-os-muted">In the brain</span>
        <span className="font-mono text-[10px] text-os-dim">live retrieval</span>
      </div>
      {state.hits == null ? (
        <div className="py-1 text-[12px] text-os-dim">searching for “{state.q}”…</div>
      ) : state.hits.length === 0 ? (
        <div className="py-1 text-[12px] text-os-dim">no brain-store pages mention “{state.q}” yet</div>
      ) : (
        <div className="flex flex-col gap-1.5">
          {state.hits.map((h, i) => (
            <div key={`${h.title}-${i}`} className="glass-well px-2.5 py-2">
              <div className="truncate font-mono text-[11px] text-os-text">{h.title.split('/').pop()}</div>
              <div className="mt-0.5 line-clamp-2 text-[11.5px] leading-snug text-os-dim">
                {h.snippet.replace(/^#+\s*/, '').slice(0, 140)}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
