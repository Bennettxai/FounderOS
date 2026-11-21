'use client';

import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { ChevronRight, CornerLeftUp, FileText, Folder, Layers, Lock, RefreshCw, Search, X } from 'lucide-react';
import type { DbNode } from '@/lib/database/scan';

/**
 * Database: the brain folder as a block map, one level at a time. Every
 * block is sized by space used and coloured by what it holds. Items too
 * small to click fold into one "smaller items" block that opens into its
 * own map at its own scale, so everything is reachable and nothing is
 * resized dishonestly. Click a folder to open it, a file to read it; the
 * breadcrumb, Up or Escape climb back out.
 */

export type Tree = { root: string; tree: DbNode; scannedAt: string; truncated: boolean };
type Rect = { x: number; y: number; w: number; h: number };
type Block = Rect & { node: DbNode | null; restCount?: number; restSize?: number };
type Preview =
  | { ok: true; path: string; size: number; mtime: number; kind: 'text'; text: string; truncated: boolean }
  | { ok: true; path: string; size: number; mtime: number; kind: 'binary' | 'private' }
  | { ok: false; error: string };

/* ── content types: the colour code ───────────────────────────────────── */
export type Kind = 'docs' | 'video' | 'image' | 'audio' | 'code' | 'data' | 'system' | 'other';
export const KINDS: { id: Kind; label: string; colour: string }[] = [
  { id: 'docs', label: 'Notes & docs', colour: 'var(--ramp-1)' },
  { id: 'video', label: 'Video', colour: 'var(--err)' },
  { id: 'image', label: 'Images', colour: 'var(--warn)' },
  { id: 'audio', label: 'Audio', colour: 'var(--ramp-4)' },
  { id: 'code', label: 'Code & config', colour: 'var(--ok)' },
  { id: 'data', label: 'Data & archives', colour: 'var(--ramp-3)' },
  { id: 'system', label: 'Git & system', colour: 'var(--text-3)' },
  { id: 'other', label: 'Other', colour: 'var(--ramp-2)' },
];
const COLOUR = Object.fromEntries(KINDS.map((k) => [k.id, k.colour])) as Record<Kind, string>;
const EXT: [Kind, RegExp][] = [
  ['docs', /\.(md|markdown|txt|pdf|docx?|rtf|pages|key|pptx?|odt|epub)$/i],
  ['video', /\.(mp4|mov|m4v|avi|mkv|webm|mxf|braw|r3d)$/i],
  ['image', /\.(png|jpe?g|gif|webp|heic|svg|tiff?|psd|ai|raw|dng|cr2|arw|ico)$/i],
  ['audio', /\.(mp3|wav|aiff?|m4a|flac|ogg|aac|logicx)$/i],
  ['code', /\.(ts|tsx|js|jsx|mjs|cjs|py|sh|zsh|rb|go|rs|swift|json|ya?ml|toml|html|css|scss|xml|ini|conf|plist|lock|env\w*)$/i],
  ['data', /\.(csv|tsv|xlsx?|db|sqlite3?|parquet|zip|tar|gz|tgz|7z|rar|dmg|iso|pkg|bin|log)$/i],
];
export function kindOf(path: string, name: string): Kind {
  if (/(^|\/)\.git(\/|$)/.test(path) || name === '.DS_Store' || name.startsWith('.tmp')) return 'system';
  for (const [k, re] of EXT) if (re.test(name)) return k;
  return 'other';
}
export type Mix = Record<Kind, number>;
const emptyMix = (): Mix => ({ docs: 0, video: 0, image: 0, audio: 0, code: 0, data: 0, system: 0, other: 0 });
export const dominant = (m: Mix): Kind => (Object.entries(m) as [Kind, number][]).sort((a, b) => b[1] - a[1])[0][0];

/* ── geometry ─────────────────────────────────────────────────────────── */
const MIN_W = 76; // a block narrower or shorter than this is too small to click and read
const MIN_H = 44;
const HEIGHT = 600;

export function fmtBytes(b: number): string {
  for (const [u, d] of [['GB', 1e9], ['MB', 1e6], ['KB', 1e3]] as const) {
    if (b >= d) { const v = b / d; return `${v < 100 ? v.toFixed(1) : Math.round(v)} ${u}`; }
  }
  return `${b} B`;
}
export const pct = (part: number, whole: number) => (whole ? `${((100 * part) / whole).toFixed(1)}%` : '0%');

function worst(row: number[], w: number) {
  const s = row.reduce((a, b) => a + b, 0);
  const mx = Math.max(...row), mn = Math.min(...row);
  return Math.max((w * w * mx) / (s * s), (s * s) / (w * w * mn));
}
/** Squarified treemap (Bruls et al.): areas in, rects out, same order. */
export function squarify(areas: number[], r: Rect): Rect[] {
  const out: Rect[] = [];
  let { x, y, w, h } = r;
  let i = 0;
  while (i < areas.length) {
    const short = Math.min(w, h);
    const row = [areas[i]];
    let j = i + 1;
    while (j < areas.length && worst([...row, areas[j]], short) <= worst(row, short)) row.push(areas[j++]);
    const s = row.reduce((a, b) => a + b, 0);
    if (w >= h) {
      const cw = h ? s / h : 0; let cy = y;
      for (const a of row) { const ch = cw ? a / cw : 0; out.push({ x, y: cy, w: cw, h: ch }); cy += ch; }
      x += cw; w -= cw;
    } else {
      const rh = w ? s / w : 0; let cx = x;
      for (const a of row) { const rw = rh ? a / rh : 0; out.push({ x: cx, y, w: rw, h: rh }); cx += rw; }
      y += rh; h -= rh;
    }
    i = j;
  }
  return out;
}

/**
 * Lay out one level: the largest items that can each get a clickable block
 * (MIN_W × MIN_H), and one "smaller items" block for the rest. The rest block
 * gets at least a clickable area of its own; its label carries the true size.
 * Returns the blocks and how many items were shown individually.
 */
export function layoutLevel(items: DbNode[], r: Rect): { blocks: Block[]; shown: number } {
  const all = items.filter((c) => c.size > 0);
  if (!all.length || r.w < 1 || r.h < 1) return { blocks: [], shown: 0 };
  const area = r.w * r.h;
  // the rest block is a door, so it is never a sliver: at least a fifth of the map (its label carries the true size)
  const restFloor = area * 0.2;
  for (let k = all.length; k >= 1; k--) {
    const shown = all.slice(0, k);
    const restSize = all.slice(k).reduce((a, c) => a + c.size, 0);
    const hasRest = k < all.length;
    const shownTotal = shown.reduce((a, c) => a + c.size, 0);
    const restArea = hasRest ? Math.max(restFloor, (restSize / (shownTotal + restSize)) * area) : 0;
    const shownArea = area - restArea;
    const areas = shown.map((c) => (c.size / shownTotal) * shownArea);
    if (hasRest) areas.push(restArea);
    const rects = squarify(areas, r);
    const fits = (hasRest ? rects : rects.slice(0, k)).every((q) => q.w >= MIN_W && q.h >= MIN_H);
    if (fits || k === 1) {
      const blocks: Block[] = rects.slice(0, k).map((q, i) => ({ ...q, node: shown[i] }));
      if (hasRest) blocks.push({ ...rects[k], node: null, restCount: all.length - k, restSize });
      return { blocks, shown: k };
    }
  }
  return { blocks: [], shown: 0 };
}

export function DatabaseExplorer({ initial }: { initial: Tree }) {
  const {
    data, focusPath, stack, setStack, selected, setSelected, preview, query, setQuery, busy, hover, setHover, pinned, setPinned, peek, setPeek, hl, box, width, mix, focus, offset, total, viewItems, viewSize, blocks, restPaths, viewMix, restMix, go, up, open, openRest, rescan, matches, jump, crumbs, lit, listItems,
  } = useBrainFiles(initial);
  const strip = (k: Kind) => `color-mix(in srgb, ${COLOUR[k]} 58%, var(--surface))`;
  const body = (k: Kind, dir: boolean) => `color-mix(in srgb, ${COLOUR[k]} ${dir ? 20 : 13}%, var(--surface))`;

  const MixBar = ({ m, of, className = '' }: { m: Mix; of: number; className?: string }) => (
    <span className={`flex h-[4px] overflow-hidden rounded-full bg-[var(--surface-2)] ${className}`}>
      {KINDS.filter((k) => m[k.id] > 0).map((k) => (
        <span key={k.id} style={{ width: `${(100 * m[k.id]) / Math.max(1, of)}%`, background: k.colour, opacity: !hl || hl === k.id ? 1 : 0.25 }} />
      ))}
    </span>
  );

  return (
    <div className="space-y-4">
      {/* summary + controls */}
      <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-[12.5px] text-os-muted">
        <span><b className="text-os-text">{fmtBytes(total)}</b> in {data.root}</span>
        <span>{data.tree.files.toLocaleString()} files</span>
        {data.truncated && <span className="text-os-warn">Large folder: the map stops at 60,000 entries.</span>}
        <span className="ml-auto flex items-center gap-2">
          <span className="relative">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-os-dim" strokeWidth={2} />
            <input value={query} onChange={(e) => setQuery(e.target.value)} placeholder="Find a file or folder" aria-label="Find a file or folder"
              className="h-8 w-[240px] max-w-[46vw] rounded-[9px] border border-[var(--hairline)] bg-os-surface pl-8 pr-2 text-[12.5px] text-os-text outline-none placeholder:text-os-dim focus:border-[var(--accent-line)]" />
            {matches.length > 0 && (
              <span className="glass-solid absolute right-0 top-9 z-20 w-[360px] max-w-[86vw] overflow-hidden rounded-lg-t py-1">
                {matches.map((m) => (
                  <button key={m.path} type="button" onClick={() => jump(m)} className="flex w-full items-center gap-2 px-3 py-1.5 text-left text-[12px] hover:bg-[var(--accent-soft)]">
                    <i className="h-2 w-2 shrink-0 rounded-[2px]" style={{ background: COLOUR[dominant(mix.get(m.path)!)] }} />
                    <span className="min-w-0 flex-1 truncate text-os-text" title={m.path}>{m.path}{m.dir ? '/' : ''}</span>
                    <span className="shrink-0 tabular-nums text-os-muted">{fmtBytes(m.size)}</span>
                  </button>
                ))}
              </span>
            )}
          </span>
          <button type="button" onClick={rescan} disabled={busy} className="inline-flex h-8 items-center gap-1.5 rounded-[9px] border border-[var(--hairline)] px-3 text-[12px] text-os-text hover:bg-[var(--accent-soft)] disabled:opacity-60">
            <RefreshCw className={`h-3.5 w-3.5 ${busy ? 'animate-spin' : ''}`} strokeWidth={2} />Rescan
          </button>
        </span>
      </div>

      {/* breadcrumb */}
      <nav aria-label="Folder path" className="flex flex-wrap items-center gap-1 text-[12.5px]">
        <button type="button" onClick={() => go('')} className={`rounded-[6px] px-1.5 py-0.5 hover:bg-[var(--accent-soft)] ${focusPath || stack.length ? 'text-os-muted' : 'font-semibold text-os-text'}`}>{data.tree.name}/</button>
        {crumbs.map((c, i) => {
          const p = crumbs.slice(0, i + 1).join('/');
          const last = i === crumbs.length - 1 && !stack.length;
          return (
            <span key={p} className="flex items-center gap-1">
              <ChevronRight className="h-3 w-3 text-os-dim" />
              <button type="button" onClick={() => go(p)} className={`rounded-[6px] px-1.5 py-0.5 hover:bg-[var(--accent-soft)] ${last ? 'font-semibold text-os-text' : 'text-os-muted'}`}>{c}/</button>
            </span>
          );
        })}
        {stack.map((_, i) => (
          <span key={`rest-${i}`} className="flex items-center gap-1">
            <ChevronRight className="h-3 w-3 text-os-dim" />
            <button type="button" onClick={() => { setStack((s) => s.slice(0, i + 1)); setSelected(null); }} className={`rounded-[6px] px-1.5 py-0.5 hover:bg-[var(--accent-soft)] ${i === stack.length - 1 ? 'font-semibold text-os-text' : 'text-os-muted'}`}>smaller items</button>
          </span>
        ))}
        {(focusPath || stack.length > 0) && (
          <button type="button" onClick={up} title="Up one level (Esc)" className="ml-auto inline-flex items-center gap-1 rounded-[6px] px-2 py-0.5 text-os-muted hover:bg-[var(--accent-soft)] hover:text-os-text">
            <CornerLeftUp className="h-3.5 w-3.5" strokeWidth={2} />Up
          </button>
        )}
      </nav>

      {/* what this view holds */}
      <div>
        <p className="text-[12px] text-os-muted"><b className="text-os-text">{fmtBytes(viewSize)}</b> here · {pct(viewSize, total)} of the brain · {viewItems.length} item{viewItems.length === 1 ? '' : 's'} · {viewItems.reduce((a, c) => a + c.files, 0).toLocaleString()} files</p>
      </div>

      {/* kind filter for this view: hover a type to find it, click to keep it */}
      <div className="flex flex-wrap items-center gap-1.5" role="group" aria-label="Filter by file type" onMouseLeave={() => setPeek(null)}>
        {KINDS.filter((k) => viewMix[k.id] > 0 || k.id === pinned).sort((a, b) => viewMix[b.id] - viewMix[a.id]).map((k) => (
          <button key={k.id} type="button" aria-pressed={pinned === k.id}
            title={pinned === k.id ? `Showing only items with ${k.label.toLowerCase()} · click to clear` : `Show only items with ${k.label.toLowerCase()}`}
            onMouseEnter={() => setPeek(k.id)} onFocus={() => setPeek(k.id)} onBlur={() => setPeek(null)}
            onClick={() => { setPinned((h) => (h === k.id ? null : k.id)); setPeek(null); }}
            className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-[3px] text-[11.5px] transition-colors ${pinned === k.id ? 'border-[var(--accent-line)] bg-[var(--accent-soft)] text-os-text' : hl === k.id ? 'border-[var(--border-strong)] text-os-text' : 'border-[var(--hairline)] text-os-muted hover:text-os-text'}`}>
            <i className="h-2 w-2 rounded-[2px]" style={{ background: k.colour }} />
            {k.label}<span className="tabular-nums text-os-dim">{fmtBytes(viewMix[k.id])}</span>
          </button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
        {/* block map */}
        <div ref={box} className="relative overflow-hidden rounded-[4px] bg-[var(--surface)]" style={{ height: HEIGHT }}>
          <div key={`${focusPath}#${offset}`} className="db-zoom absolute inset-0">
            {blocks.map((b, i) => {
              const isRest = !b.node;
              const m = isRest ? restMix : mix.get(b.node!.path)!;
              const k = dominant(m);
              const size = isRest ? b.restSize! : b.node!.size;
              const key = isRest ? 'rest' : b.node!.path;
              const sel = selected && b.node && selected.path === b.node.path;
              const hot = hover === key || (hover !== null && isRest && restPaths.has(hover));
              const dim = !lit(m);
              const tip = isRest
                ? `${b.restCount} smaller items · ${fmtBytes(size)}\nClick to see them at their own scale`
                : `${b.node!.path}\n${fmtBytes(size)} · ${pct(size, total)} of the brain${b.node!.dir ? ` · ${b.node!.files.toLocaleString()} files\nMostly ${KINDS.find((x) => x.id === k)!.label.toLowerCase()} · click to open` : `\n${KINDS.find((x) => x.id === k)!.label} · click to read`}`;
              return (
                <button key={`${key}-${i}`} type="button" title={tip}
                  onClick={() => (isRest ? openRest() : open(b.node!))}
                  onMouseEnter={() => setHover(key)} onMouseLeave={() => setHover(null)}
                  className="db-block absolute flex flex-col items-stretch justify-start overflow-hidden text-left"
                  style={{
                    left: b.x, top: b.y, width: b.w, height: b.h,
                    background: isRest ? 'var(--surface-2)' : body(k, b.node!.dir),
                    outline: sel ? '2px solid var(--accent)' : hot ? '2px solid var(--border-strong)' : undefined, outlineOffset: -2,
                    opacity: dim ? 0.28 : 1,
                  }}>
                  {b.w > 30 && b.h > 14 && (!isRest && b.node!.dir ? (
                    <>
                      <span className="block truncate px-1.5 text-[10.5px] font-bold leading-[17px] text-os-text" style={{ background: strip(k) }}>
                        {b.node!.name}/ <span className="font-semibold opacity-75">{fmtBytes(size)} · {pct(size, total)}</span>
                      </span>
                      {b.h > 40 && <span className="block truncate px-1.5 pt-1 text-[10px] tabular-nums text-os-muted">{b.node!.files.toLocaleString()} file{b.node!.files === 1 ? '' : 's'} · mostly {KINDS.find((x) => x.id === k)!.label.toLowerCase()}</span>}
                    </>
                  ) : (
                    <span className="block px-1.5 py-1 leading-tight">
                      <span className={`block truncate text-[10.5px] font-bold ${isRest ? 'text-os-muted' : 'text-os-text'}`}>{isRest ? `${b.restCount} smaller items` : b.node!.name}</span>
                      {b.h > 26 && <span className="block truncate text-[10px] tabular-nums text-os-muted">{fmtBytes(size)} · {pct(size, total)}</span>}
                    </span>
                  ))}
                  {(isRest || b.node!.dir) && b.h > 52 && b.w > 60 && <MixBar m={m} of={size} className="absolute inset-x-2 bottom-2" />}
                </button>
              );
            })}
            {!blocks.length && width > 0 && <div className="grid h-full place-items-center text-[13px] text-os-muted">This folder is empty.</div>}
          </div>
        </div>

        {/* side panel: file preview, or this view's items */}
        <aside className="flex max-h-[600px] min-h-0 flex-col overflow-hidden">
          {selected ? (
            <>
              <div className="flex items-start gap-2 border-b border-[var(--hairline)] px-4 py-3">
                <FileText className="mt-0.5 h-4 w-4 shrink-0" style={{ color: COLOUR[kindOf(selected.path, selected.name)] }} />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-[13px] font-semibold text-os-text" title={selected.name}>{selected.name}</div>
                  <div className="truncate text-[11.5px] text-os-muted" title={selected.path}>{selected.path}</div>
                  <div className="mt-1 text-[11.5px] tabular-nums text-os-muted">{fmtBytes(selected.size)} · {pct(selected.size, total)} · modified {new Date(selected.mtime).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })}</div>
                </div>
                <button type="button" onClick={() => setSelected(null)} aria-label="Close file" className="rounded-[6px] p-1 text-os-muted hover:bg-[var(--accent-soft)] hover:text-os-text"><X className="h-4 w-4" /></button>
              </div>
              <div className="min-h-0 flex-1 overflow-auto px-4 py-3">
                {!preview ? <p className="text-[12px] text-os-muted">Reading…</p>
                  : !preview.ok ? <p className="text-[12px] text-os-err">{preview.error}</p>
                  : preview.kind === 'private' ? <p className="flex items-start gap-2 text-[12px] text-os-muted"><Lock className="mt-0.5 h-3.5 w-3.5 shrink-0" />Private file: the contents stay on disk and are never shown here.</p>
                  : preview.kind === 'binary' ? <p className="text-[12px] text-os-muted">Not a text file, so there is no preview. Open it from {data.root}/{preview.path}.</p>
                  : preview.kind === 'text' ? <>
                      <pre className="whitespace-pre-wrap break-words font-mono text-[11.5px] leading-relaxed text-os-text">{preview.text}</pre>
                      {preview.truncated && <p className="mt-2 text-[11.5px] text-os-muted">First 24 KB shown.</p>}
                    </> : null}
              </div>
            </>
          ) : (
            <>
              <div className="px-4 pb-2 pt-3">
                <div className="text-[14px] font-bold text-os-text">{stack.length ? 'Smaller items' : focusPath ? `${focus.name}/` : 'Top-level breakdown'}</div>
              </div>
              <div className="mx-4 grid grid-cols-[minmax(0,1fr)_62px_50px_50px] gap-x-2 border-b border-[var(--hairline)] pb-1.5 text-[9.5px] font-semibold uppercase tracking-[0.08em] text-os-dim">
                <span>{focusPath || stack.length ? 'Item' : 'Folder'}</span><span className="text-right">Size</span><span className="text-right">% here</span><span className="text-right">% brain</span>
              </div>
              {pinned && (
                <p className="mx-4 mt-1.5 flex items-center gap-2 text-[11.5px] text-os-muted">
                  <span className="min-w-0 flex-1 truncate">Showing {listItems.length} of {viewItems.length} with {KINDS.find((x) => x.id === pinned)!.label.toLowerCase()}</span>
                  <button type="button" onClick={() => setPinned(null)} className="shrink-0 rounded-[6px] px-1.5 py-0.5 text-os-text hover:bg-[var(--accent-soft)]">Clear</button>
                </p>
              )}
              <ul className="min-h-0 flex-1 overflow-auto px-4 pb-2" onMouseLeave={() => setHover(null)}>
                {listItems.map((c) => {
                  const m = mix.get(c.path)!;
                  const k = dominant(m);
                  const hot = hover === c.path || (hover === 'rest' && restPaths.has(c.path));
                  return (
                    <li key={c.path} className="border-b border-[var(--hairline)]" style={{ opacity: lit(m) ? 1 : 0.35 }}>
                      <button type="button" onClick={() => open(c)} onMouseEnter={() => setHover(c.path)}
                        className={`grid w-full grid-cols-[minmax(0,1fr)_62px_50px_50px] items-center gap-x-2 py-[7px] text-left text-[12px] hover:bg-[var(--accent-soft)] ${hot ? 'bg-[var(--accent-soft)]' : ''}`}>
                        <span className="flex min-w-0 items-center gap-2">
                          <i className="h-[9px] w-[9px] shrink-0 rounded-[2px]" style={{ background: strip(k) }} />
                          <span className="truncate text-os-text" title={c.path}>{c.name}{c.dir ? '/' : ''}</span>
                        </span>
                        <span className="text-right tabular-nums text-os-text">{fmtBytes(c.size)}</span>
                        <span className="text-right tabular-nums text-os-muted">{pct(c.size, viewSize)}</span>
                        <span className="text-right tabular-nums text-os-muted">{pct(c.size, total)}</span>
                      </button>
                    </li>
                  );
                })}
                {!viewItems.length && <li className="py-3 text-[12px] text-os-muted">Nothing in this folder.</li>}
                {viewItems.length > 0 && !listItems.length && <li className="py-3 text-[12px] text-os-muted">Nothing here of that type.</li>}
              </ul>
            </>
          )}
        </aside>
      </div>
    </div>
  );
}
/**
 * Navigation, kind filter, search, preview and rescan for the brain block map,
 * shared by both trees of /database (GladOS: components/database/DatabaseG.tsx).
 * `height` is the map height the level layout is computed for.
 */
export function useBrainFiles(initial: Tree, height = HEIGHT) {
  const [data, setData] = useState(initial);
  const [focusPath, setFocusPath] = useState('');
  const [stack, setStack] = useState<number[]>([]); // offsets into the folder: each "smaller items" step pushes one
  const [selected, setSelected] = useState<DbNode | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [query, setQuery] = useState('');
  const [busy, setBusy] = useState(false);
  const [width, setWidth] = useState(0);
  const [hover, setHover] = useState<string | null>(null); // a node path, or 'rest'
  // Kind filter (GladOS UI H38): a click pins it (aria-pressed), hover or focus
  // only previews another kind; leaving the chips falls back to the pinned one.
  const [pinned, setPinned] = useState<Kind | null>(null);
  const [peek, setPeek] = useState<Kind | null>(null);
  const hl = peek ?? pinned;
  const box = useRef<HTMLDivElement>(null);

  // index, and each node's byte mix by content type (folders sum their contents)
  const { byPath, mix } = useMemo(() => {
    const byPath = new Map<string, DbNode>();
    const mix = new Map<string, Mix>();
    const walk = (n: DbNode): Mix => {
      byPath.set(n.path, n);
      const m = emptyMix();
      if (n.dir) for (const c of n.children ?? []) { const cm = walk(c); for (const k of KINDS) m[k.id] += cm[k.id]; }
      else m[kindOf(n.path, n.name)] += n.size;
      mix.set(n.path, m);
      return m;
    };
    walk(data.tree);
    return { byPath, mix };
  }, [data]);

  const focus = byPath.get(focusPath) ?? data.tree;
  const offset = stack.length ? stack[stack.length - 1] : 0;
  const total = data.tree.size;
  const viewItems = useMemo(() => (focus.children ?? []).filter((c) => c.size > 0).slice(offset), [focus, offset]);
  const viewSize = viewItems.reduce((a, c) => a + c.size, 0);

  useLayoutEffect(() => {
    const el = box.current; if (!el) return;
    const ro = new ResizeObserver(() => setWidth(el.clientWidth));
    ro.observe(el); setWidth(el.clientWidth);
    return () => ro.disconnect();
  }, []);
  const { blocks, shown } = useMemo(() => (width ? layoutLevel(viewItems, { x: 0, y: 0, w: width, h: height }) : { blocks: [], shown: 0 }), [viewItems, width]);
  const restPaths = useMemo(() => new Set(viewItems.slice(shown).map((c) => c.path)), [viewItems, shown]);

  const viewMix = useMemo(() => {
    const m = emptyMix();
    for (const c of viewItems) { const cm = mix.get(c.path)!; for (const k of KINDS) m[k.id] += cm[k.id]; }
    return m;
  }, [viewItems, mix]);
  const restMix = useMemo(() => {
    const m = emptyMix();
    for (const c of viewItems.slice(shown)) { const cm = mix.get(c.path)!; for (const k of KINDS) m[k.id] += cm[k.id]; }
    return m;
  }, [viewItems, shown, mix]);

  const go = (path: string) => { setFocusPath(path); setStack([]); setSelected(null); setHover(null); };
  const up = useCallback(() => {
    if (selected) { setSelected(null); return; }
    if (stack.length) { setStack((s) => s.slice(0, -1)); return; }
    if (!focusPath) return;
    go(focusPath.includes('/') ? focusPath.slice(0, focusPath.lastIndexOf('/')) : '');
  }, [focusPath, selected, stack.length]);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === 'Escape' && !(e.target as HTMLElement)?.closest('input')) up(); };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [up]);

  useEffect(() => {
    if (!selected || selected.dir) { setPreview(null); return; }
    let live = true;
    setPreview(null);
    fetch(`/api/database/file?path=${encodeURIComponent(selected.path)}`, { cache: 'no-store' })
      .then((r) => r.json()).then((p: Preview) => { if (live) setPreview(p); })
      .catch(() => { if (live) setPreview({ ok: false, error: 'Could not read the file.' }); });
    return () => { live = false; };
  }, [selected]);

  const open = (n: DbNode) => { if (n.dir) go(n.path); else setSelected(n); };
  const openRest = () => { setStack((s) => [...s, offset + shown]); setSelected(null); setHover(null); };
  const rescan = async () => {
    setBusy(true);
    try { const r = await fetch('/api/database?fresh=1', { cache: 'no-store' }).then((x) => x.json()); if (r.configured) { setData(r); go(''); } } finally { setBusy(false); }
  };

  const matches = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (q.length < 2) return [];
    const out: DbNode[] = [];
    for (const n of byPath.values()) if (n.path && n.name.toLowerCase().includes(q)) out.push(n);
    return out.sort((a, b) => b.size - a.size).slice(0, 12);
  }, [query, byPath]);
  const jump = (n: DbNode) => {
    setQuery('');
    if (n.dir) go(n.path);
    else { go(n.path.includes('/') ? n.path.slice(0, n.path.lastIndexOf('/')) : ''); setSelected(n); }
  };

  const crumbs = focusPath ? focusPath.split('/') : [];
  const lit = (m: Mix | undefined) => !hl || !m || m[hl] > 0;
  // the breakdown list keeps only items holding the pinned kind
  const listItems = pinned ? viewItems.filter((c) => (mix.get(c.path)?.[pinned] ?? 0) > 0) : viewItems;
  return { data, focusPath, stack, setStack, selected, setSelected, preview, query, setQuery, busy, hover, setHover, pinned, setPinned, peek, setPeek, hl, box, width, byPath, mix, focus, offset, total, viewItems, viewSize, blocks, shown, restPaths, viewMix, restMix, go, up, open, openRest, rescan, matches, jump, crumbs, lit, listItems };
}
