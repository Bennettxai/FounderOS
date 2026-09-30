/**
 * Dot-matrix column labels for the slab pages: the first token of each name
 * (cut to `max` characters), numbered only when two would collide, so two
 * columns never share a label or a React key. Shared so no view-model can
 * forget the de-dupe (it did on /agents, review 2026-09-24).
 */
export function shortLabels(names: string[], split: RegExp = /\s+/, max = Infinity, fallback = 'Agent'): string[] {
  const seen = new Map<string, number>();
  return names.map((name) => {
    const base = (name.trim().split(split)[0] || name.trim() || fallback).slice(0, max) || fallback;
    const n = (seen.get(base) ?? 0) + 1;
    seen.set(base, n);
    return n === 1 ? base : `${base} ${n}`;
  });
}
