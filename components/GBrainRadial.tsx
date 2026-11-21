'use client';

/**
 * Optimal Engine Radial — design variant 3, from UI Example 6 (Fuselab "Control AI
 * Policy Platform"), re-toned to the Venture A mint (on black,
 * 2026-08-23). The brain-store's folders are angular sectors ("context
 * pool"), its pages are outer pearls, each spoke carries a context-health
 * bar. Clicking a pearl or folder opens the expanded fan view.
 *
 * Camera: hovering leans the ring toward the cursor (eased micro-zoom);
 * trackpad pinch zooms at the cursor, two-finger scroll pans, double-click
 * resets. View changes (ring ↔ expanded) animate in and out.
 *
 * Honesty rules hold (docs/DESIGN-GBRAIN.md): every number is real — page
 * counts, chunk counts, word counts, file mtime freshness, wikilink degree.
 * The ask bar runs live queries through GET /api/brain; typed queries dim
 * the ring to the pages retrieval actually came from. No fake metrics.
 */

import { useEffect, useMemo, useRef, useState } from 'react';
import { Maximize2, Minimize2 } from 'lucide-react';
import { createPortal } from 'react-dom';
import type { ObservatoryData, ObservatoryPage } from '@/lib/gbrain-observatory';
import { useGBrainThoughts } from '@/components/gbrain-live';
import { SetupLink } from '@/components/SetupLink';
import { usePalette } from '@/lib/use-palette';

const MINT = 'var(--accent)';
const TEXT = 'var(--gbr-text)';
const MUTED = 'var(--text-3)';

const SHOW_MAX = 66; // outer-ring pearl budget — counter says "showing X of N"
const VB = 1180; // viewBox span (see the <svg> below)

function freshColor(days: number | null): string {
  if (days == null) return 'var(--gbr-nodate)';
  if (days <= 30) return MINT;
  if (days <= 90) return 'var(--gbr-amber)';
  return 'var(--err)';
}
/**
 * Context-health ramp. Green once a page is above half; below that it takes
 * the app's amber then red, so the health bar and the freshness chip beside
 * it read as one pair of signals rather than two colour languages.
 */
function healthColor(health: number): string {
  if (health >= 50) return MINT;
  if (health >= 25) return 'var(--gbr-amber)';
  return 'var(--err)';
}
function freshLabel(days: number | null): string {
  if (days == null) return 'no date';
  if (days === 0) return 'today';
  return `${days}d ago`;
}

type Spoke = {
  pi: number; // index into data.pages
  c: number;
  angle: number; // degrees, 0 = right, clockwise
  trackLen: number;
  page: ObservatoryPage;
};
type Sector = { c: number; a0: number; a1: number };

function buildRing(data: ObservatoryData): { sectors: Sector[]; spokes: Spoke[]; shownPages: number } {
  const sectors: Sector[] = [];
  const spokes: Spoke[] = [];
  if (data.clusters.length === 0) return { sectors, spokes, shownPages: 0 };

  // budget pages per folder proportionally (min 3, never more than it has)
  const per = data.clusters.map((c) => Math.min(c.pages, Math.max(3, Math.round((SHOW_MAX * c.pages) / data.totalPages))));
  let excess = per.reduce((s, n) => s + n, 0) - SHOW_MAX;
  for (let i = per.length - 1; excess > 0 && i >= 0; i--) {
    const cut = Math.min(excess, Math.max(0, per[i] - 3));
    per[i] -= cut;
    excess -= cut;
  }

  const shown = per.reduce((s, n) => s + n, 0);
  const GAP = 4; // degrees between sectors
  const usable = 360 - GAP * data.clusters.length;
  let a = -90;
  data.clusters.forEach((cl, ci) => {
    const span = (usable * per[ci]) / shown;
    const a0 = a;
    const a1 = a + span;
    sectors.push({ c: ci, a0, a1 });
    const members = data.pages
      .map((p, pi) => ({ p, pi }))
      .filter((x) => x.p.c === ci)
      .sort((x, y) => y.p.chunks - x.p.chunks)
      .slice(0, per[ci]);
    const maxWords = Math.max(1, ...data.pages.map((p) => p.words));
    const inset = Math.min(4, span / (members.length + 1));
    members.forEach((m, i) => {
      const t = members.length === 1 ? 0.5 : i / (members.length - 1);
      spokes.push({
        pi: m.pi,
        c: ci,
        angle: a0 + inset + (span - inset * 2) * t,
        trackLen: 70 + 150 * Math.sqrt(m.p.words / maxWords),
        page: m.p,
      });
    });
    a = a1 + GAP;
  });
  return { sectors, spokes, shownPages: shown };
}

const R_BASE = 300; // numbered dot ring
const R_PILL = 322; // metric pill start
const R_TRACK = 386; // capsule track start
const PEARL_GAP = 20; // track end → pearl
const ORNAMENTS = [0, 60, 120, 180, 240, 300];

export function GBrainRadial({
  data,
  g = false,
  fullscreen,
}: {
  data: ObservatoryData;
  /**
   * GladOS style (glados-ui S05 F05.10, F05.11): the page's GVizStage owns the
   * frame, title, counter and Full screen, so the radial drops its own corners,
   * identity block, counter, full-screen button, folder legend and footer, and
   * its controls render as kit controls. Colours come from tokens either way.
   */
  g?: boolean;
  /** Controlled full screen (the GVizStage's): the wheel pans and zooms only then. */
  fullscreen?: boolean;
}) {
  const { typed, answer, sources, phase, litRef, manual, inputProps, statusWord, showAnswer, ask } =
    useGBrainThoughts(data);
  const palette = usePalette();
  // console value == the old literal array, byte for byte; under GladOS the accent's series slots
  const PEARLS = g && palette.series.length ? palette.series : palette.pearls;
  const ring = useMemo(() => buildRing(data), [data]);
  const [hover, setHover] = useState<number | null>(null); // spoke index
  const [year, setYear] = useState<number | null>(null);
  // Inline, the page owns the scroll wheel and the ring simply bleeds past
  // the frame; full screen takes over the viewport and turns pan/zoom on.
  const [ownFs, setFs] = useState(false);
  const fs = fullscreen ?? ownFs;
  useEffect(() => {
    if (!fs || fullscreen !== undefined) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setFs(false);
    window.addEventListener('keydown', onKey);
    const prev = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    return () => {
      window.removeEventListener('keydown', onKey);
      document.body.style.overflow = prev;
    };
  }, [fs]);
  // Retrieval notices render OUTSIDE this frame, in the page header, so they
  // sit above the canvas window at title height instead of inside it. The
  // slot is a plain div on the page; we portal into it once it exists.
  const [noticeSlot, setNoticeSlot] = useState<HTMLElement | null>(null);
  useEffect(() => {
    setNoticeSlot(document.getElementById('gbrain-notice-slot'));
  }, []);

  const [sel, setSel] = useState<{ c: number; pi: number } | null>(null);
  const [exiting, setExiting] = useState<null | 'ring' | 'panel'>(null);
  const [filter, setFilter] = useState<'all' | 'fresh' | 'stale' | 'linked'>('all');

  // ——— camera: hover micro-zoom + trackpad pinch/pan, eased in a rAF loop ———
  const svgRef = useRef<SVGSVGElement>(null);
  const camGRef = useRef<SVGGElement>(null);
  const userCam = useRef({ s: 1, tx: 0, ty: 0 });
  const cam = useRef({ s: 1, tx: 0, ty: 0 });
  const cursor = useRef<{ x: number; y: number } | null>(null);
  const ringVisible = data.mounted && sel === null;

  useEffect(() => {
    if (!ringVisible) return;
    const svg = svgRef.current;
    if (!svg) return;
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    const toView = (e: { clientX: number; clientY: number }) => {
      const r = svg.getBoundingClientRect();
      const fit = Math.min(r.width, r.height) / VB;
      return { x: (e.clientX - r.left - r.width / 2) / fit, y: (e.clientY - r.top - r.height / 2) / fit };
    };
    const move = (e: MouseEvent) => {
      cursor.current = toView(e);
    };
    const leave = () => {
      cursor.current = null;
    };
    // the ring can never leave the screen entirely — panning/zooming keeps
    // at least ~30% of the viewBox overlapping the viewport (2026-08-24)
    const clampPan = (u: { s: number; tx: number; ty: number }) => {
      const max = ((1 + u.s) * VB) / 2 - VB * 0.3;
      u.tx = Math.max(-max, Math.min(max, u.tx));
      u.ty = Math.max(-max, Math.min(max, u.ty));
    };
    const wheel = (e: WheelEvent) => {
      e.preventDefault();
      const u = userCam.current;
      const p = toView(e);
      if (e.ctrlKey || e.metaKey) {
        // pinch → zoom anchored at the cursor
        const s2 = Math.min(3, Math.max(0.7, u.s * Math.exp(-e.deltaY * 0.012)));
        u.tx = p.x - (s2 / u.s) * (p.x - u.tx);
        u.ty = p.y - (s2 / u.s) * (p.y - u.ty);
        u.s = s2;
      } else {
        // two-finger scroll → pan
        const r = svg.getBoundingClientRect();
        const fit = Math.min(r.width, r.height) / VB;
        u.tx -= e.deltaX / fit;
        u.ty -= e.deltaY / fit;
      }
      clampPan(u);
    };
    const dbl = () => {
      userCam.current = { s: 1, tx: 0, ty: 0 };
    };
    svg.addEventListener('mousemove', move);
    svg.addEventListener('mouseleave', leave);
    // Wheel capture ONLY in full screen — inline, scrolling over the ring
    // must keep scrolling the page .
    if (fs) svg.addEventListener('wheel', wheel, { passive: false });
    svg.addEventListener('dblclick', dbl);

    let raf = 0;
    let last = performance.now();
    const frame = (now: number) => {
      const dt = Math.min(0.05, (now - last) / 1000);
      last = now;
      const u = userCam.current;
      const c = cursor.current;
      const micro = !reduced && c ? 1.13 : 1;
      const sT = u.s * micro;
      const txT = c ? micro * u.tx + (1 - micro) * c.x : u.tx;
      const tyT = c ? micro * u.ty + (1 - micro) * c.y : u.ty;
      const k = reduced ? 1 : 1 - Math.exp(-dt * 5);
      cam.current.s += (sT - cam.current.s) * k;
      cam.current.tx += (txT - cam.current.tx) * k;
      cam.current.ty += (tyT - cam.current.ty) * k;
      camGRef.current?.setAttribute(
        'transform',
        `translate(${cam.current.tx.toFixed(2)} ${cam.current.ty.toFixed(2)}) scale(${cam.current.s.toFixed(4)})`,
      );
      raf = requestAnimationFrame(frame);
    };
    raf = requestAnimationFrame(frame);
    return () => {
      cancelAnimationFrame(raf);
      svg.removeEventListener('mousemove', move);
      svg.removeEventListener('mouseleave', leave);
      if (fs) svg.removeEventListener('wheel', wheel);
      svg.removeEventListener('dblclick', dbl);
    };
  }, [ringVisible, fs]);

  const active = phase === 'retrieve' || phase === 'answer';
  const litPages = useMemo(() => {
    if (!active || sources.length === 0) return null;
    const bases = new Set(sources.map((s) => s.toLowerCase()));
    const hit = new Set<number>();
    data.pages.forEach((p, pi) => {
      if (bases.has(p.t.split('/').pop()!.toLowerCase())) hit.add(pi);
    });
    return hit;
  }, [active, sources, data.pages]);
  const litClusters = active ? litRef.current : new Set<number>();

  // animated view changes: fade the ring out, mount the panel in — and back
  const openSel = (next: { c: number; pi: number }) => {
    if (exiting) return;
    setExiting('ring');
    window.setTimeout(() => {
      setSel(next);
      setExiting(null);
    }, 200);
  };
  const closeSel = () => {
    if (exiting) return;
    setExiting('panel');
    window.setTimeout(() => {
      setSel(null);
      setExiting(null);
    }, 230);
  };
  const selectPage = (pi: number) => {
    setFilter('all');
    openSel({ c: data.pages[pi].c, pi });
  };
  const selectFolder = (c: number) => {
    const biggest = data.pages
      .map((p, pi) => ({ p, pi }))
      .filter((x) => x.p.c === c)
      .sort((x, y) => y.p.chunks - x.p.chunks)[0];
    setFilter('all');
    openSel({ c, pi: biggest?.pi ?? 0 });
  };

  const ringAnim = exiting === 'ring' ? 'gbr-vout' : 'gbr-vin';

  return (
    <div
      className={
        fs
          ? 'gbr-root fixed inset-0 z-[130] overflow-hidden'
          : // no overflow-hidden inline — the ring deliberately bleeds past the
            // frame's top edge instead of being cut off by the window
            'gbr-root relative h-full w-full'
      }
      // transparent inline: the frame shares the page's background so there is
      // no visible box — the corners alone mark the region. Fullscreen paints
      // the page token itself so content can't show through.
      style={{ background: fs ? 'var(--bg)' : 'transparent', color: TEXT }}
    >
      {/* mint atmosphere — fades to nothing well inside the frame, so it never
          draws an edge. The old vignette darkened the borders into a visible
          box against the page; it is gone. */}
      <div
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            'var(--gbr-glow-bg)',
        }}
      />

      {/* viewfinder corners — shared cue across the Optimal Engine variants */}
      {!g && (['lt', 'rt', 'lb', 'rb'] as const).map((c) => (
        <div key={c} className={`gbr-corner gbr-${c}`} aria-hidden />
      ))}

      {/* identity */}
      {!g && (
      <div className="pointer-events-none absolute left-7 top-6 z-20 max-sm:left-4 max-sm:right-4">
        <div className="flex items-center gap-2.5">
          <span className="h-2 w-2 rounded-full" style={{ background: data.mounted ? MINT : 'var(--gbr-amber)' }} />
          <span className="text-[15px] font-medium tracking-[-0.01em]" style={{ color: TEXT }}>
            Optimal Engine
          </span>
          <span className="text-[12px]" style={{ color: 'var(--text-3)' }}>
            knowledge core · radial
          </span>
        </div>
        <div className="mt-1.5 font-mono text-[10.5px] leading-relaxed" style={{ color: 'var(--gbr-status)' }}>
          {data.statusLine}
        </div>
      </div>
      )}

      {/* counters */}
      {!g && data.mounted && (
        <div className="pointer-events-none absolute right-7 top-6 z-20 text-right max-sm:hidden">
          <div className="text-[26px] font-normal tabular-nums leading-none" style={{ color: TEXT }}>
            {data.totalChunks.toLocaleString()}
            <span className="ml-1.5 text-[12px]" style={{ color: 'var(--text-3)' }}>
              chunks
            </span>
          </div>
        </div>
      )}

      {/* full-screen toggle — the only mode where the wheel pans/zooms */}
      {!g && (
      <button
        onClick={() => setFs((v) => !v)}
        title={fs ? 'Exit full screen (esc)' : 'Full screen — pan + zoom'}
        className="gbr-widget absolute right-7 top-[84px] z-30 max-sm:right-4 flex items-center gap-2 rounded-full border px-3 py-1.5 text-[11px]"
        style={{
          background: 'color-mix(in srgb, var(--bg) 82%, transparent)',
          borderColor: fs ? 'color-mix(in srgb, var(--accent) 45%, transparent)' : 'rgb(var(--mist-rgb) / 0.14)',
          color: fs ? MINT : 'var(--text-2)',
          backdropFilter: 'blur(10px)',
        }}
      >
        {fs ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize2 className="h-3.5 w-3.5" />}
        {fs ? 'Exit' : 'Full screen'}
      </button>
      )}

      {!g && !data.mounted && (
        <div className="absolute inset-0 z-10 flex items-center justify-center">
          <div className="text-center">
            <div className="text-[14px]" style={{ color: 'var(--text-2)' }}>
              brain-store not mounted
            </div>
            <div className="mt-3 flex justify-center">
              <SetupLink href="/setup" hint="expected at ~/knowledge/brain-store (or GBRAIN_STORE)" />
            </div>
          </div>
        </div>
      )}

      {/* timeline rail — real modification years */}
      {ringVisible && data.years.length > 0 && (
        // sits under the ask bar (top-[92px] + its height) — it used to be
        // vertically centered on the left edge, where it collided with the
        // legend's column
        <div className={g ? `kg-radial-years ${ringAnim}` : `absolute left-7 top-[150px] z-20 ${ringAnim}`} role={g ? 'group' : undefined} aria-label={g ? 'Filter by year modified' : undefined}>
          {data.years.map((y) => {
            const on = year === y.year;
            const max = Math.max(...data.years.map((v) => v.count));
            if (g) {
              return (
                <button key={y.year} type="button" className="g-chip" data-size="compact" aria-pressed={on} onClick={() => setYear(on ? null : y.year)} title={`${y.count} pages last modified in ${y.year}`}>
                  {y.year}
                </button>
              );
            }
            return (
              <button
                key={y.year}
                onClick={() => setYear(on ? null : y.year)}
                className="group flex w-full items-center gap-2 py-1.5 text-left"
              >
                <span
                  className="h-[3px] rounded-full transition-all"
                  style={{ width: 6 + (26 * y.count) / max, background: on ? MINT : 'rgb(var(--gbr-tint-a) / 0.3)' }}
                />
                <span
                  className="rounded-full border px-2 py-0.5 font-mono text-[10.5px] tabular-nums transition-colors"
                  style={on ? { borderColor: MINT, color: MINT } : { borderColor: 'transparent', color: 'var(--gbr-year)' }}
                >
                  {y.year}
                </span>
              </button>
            );
          })}
          {!g && year != null && (
            <div className="mt-1 font-mono text-[9.5px]" style={{ color: 'var(--text-3)' }}>
              modified in {year}
            </div>
          )}
        </div>
      )}

      {/* ——— THE RING ——— */}
      {ringVisible && (
        <svg
          ref={svgRef}
          viewBox={`-${VB / 2} -${VB / 2} ${VB} ${VB}`}
          className={`gbr-leaf absolute inset-0 h-full w-full ${ringAnim}`}
          style={{ overflow: 'visible' }}
        >
          <defs>
            {PEARLS.map((p, i) => (
              <radialGradient key={i} id={`gbr-pearl-${i}`} cx="0.35" cy="0.3" r="0.85">
                <stop offset="0%" stopColor="var(--gbr-white)" stopOpacity="0.95" />
                <stop offset="28%" stopColor={p} />
                <stop offset="100%" stopColor={p} stopOpacity="0.55" />
              </radialGradient>
            ))}
            {PEARLS.map((p, i) => (
              <radialGradient key={`g${i}`} id={`gbr-glow-${i}`} cx="0.5" cy="0.5" r="0.5">
                <stop offset="0%" stopColor={p} stopOpacity="0.55" />
                <stop offset="60%" stopColor={p} stopOpacity="0.16" />
                <stop offset="100%" stopColor={p} stopOpacity="0" />
              </radialGradient>
            ))}
            <radialGradient id="gbr-core" cx="0.38" cy="0.32" r="0.9">
              <stop offset="0%" stopColor="var(--brain-core-light)" />
              <stop offset="45%" style={{ stopColor: 'var(--accent)' }} />
              <stop offset="100%" stopColor="var(--brain-core-dark)" />
            </radialGradient>
            <path id="gbr-arc-folders" d="M -258 0 A 258 258 0 0 1 258 0" fill="none" />
            <path id="gbr-arc-pages" d="M -348 0 A 348 348 0 0 1 348 0" fill="none" />
          </defs>

          <g ref={camGRef}>
            {/* inner disc + ring hairlines */}
            <circle r="252" fill="var(--gbr-disc)" stroke="rgb(var(--mist-rgb) / 0.07)" />
            <circle r="140" fill="none" stroke="rgb(var(--mist-rgb) / 0.05)" />
            <circle r={R_BASE} fill="none" stroke="rgb(var(--mist-rgb) / 0.06)" />
            {/* tracked uppercase arc labels are Classic chrome; the GladOS style uses sentence case (§3.3.7) */}
            <text fontSize={g ? 10 : 9} letterSpacing={g ? undefined : '0.32em'} fill="rgb(var(--gbr-tint-b) / 0.3)">
              <textPath href="#gbr-arc-folders" startOffset="50%" textAnchor="middle">
                {g ? 'Folders' : 'CONTEXT POOL'}
              </textPath>
            </text>
            <text fontSize={g ? 10 : 9} letterSpacing={g ? undefined : '0.32em'} fill="rgb(var(--gbr-tint-b) / 0.24)">
              <textPath href="#gbr-arc-pages" startOffset="50%" textAnchor="middle">
                {g ? 'Pages' : 'PAGES'}
              </textPath>
            </text>

            {/* center ornaments — instrument chrome */}
            {ORNAMENTS.map((d) => (
              <g key={d} transform={`rotate(${d + 30}) translate(112,0) rotate(${-(d + 30)})`} opacity="0.3">
                <path d="M0 -5 L3 0 L0 5 L-3 0 Z M-7 0 h3 M7 0 h-3" strokeWidth="1" style={{ stroke: 'var(--accent)', fill: 'none' }} />
              </g>
            ))}

            {/* core */}
            <g>
              <circle r="56" fill="none" stroke="color-mix(in srgb, var(--accent) 10%, transparent)" />
              <circle r="46" fill="none" stroke="color-mix(in srgb, var(--accent) 16%, transparent)" />
              <circle r="34" fill="color-mix(in srgb, var(--accent) 8%, transparent)" />
              <circle r="27" fill="url(#gbr-core)" />
              <circle r="27" fill="none" stroke="rgb(var(--gbr-tint-c) / 0.35)" />
              <text y="102" textAnchor="middle" fontSize="15" fill={TEXT} fontWeight="500">
                Optimal Engine
              </text>
              <g transform="translate(0,118)">
                <rect x="-34" y="0" width="68" height="16" rx="8" fill="color-mix(in srgb, var(--accent) 9%, transparent)" stroke="color-mix(in srgb, var(--accent) 35%, transparent)" />
                <text y="11.5" textAnchor="middle" fontSize="9.5" style={{ fill: 'var(--accent)' }} className="font-mono">
                  {data.totalPages} pages
                </text>
              </g>
            </g>

            {/* context pool — folder pearls + dotted spines */}
            {data.clusters.map((cl, ci) => {
              const s = ring.sectors[ci];
              const mid = ((s.a0 + s.a1) / 2) * (Math.PI / 180);
              const r = 172 + (ci % 3) * 26;
              const x = Math.cos(mid) * r;
              const yy = Math.sin(mid) * r;
              const dim = litPages != null && !litClusters.has(ci) && manual;
              return (
                <g key={cl.key} opacity={dim ? 0.3 : 1} className="gbr-fade">
                  <line x1={Math.cos(mid) * 80} y1={Math.sin(mid) * 80} x2={x} y2={yy} stroke="rgb(var(--gbr-tint-b) / 0.14)" strokeDasharray="2 5" />
                  <g transform={`translate(${x},${yy})`} onClick={() => selectFolder(ci)} style={{ cursor: 'pointer' }}>
                    <g className="gbr-fnode">
                      <circle r="26" fill={`url(#gbr-glow-${ci % PEARLS.length})`} className="gbr-glowc" />
                      <circle r="16" fill="none" stroke={PEARLS[ci % PEARLS.length]} strokeOpacity="0.4" strokeDasharray="1 2.4" />
                      <circle r="11" fill={`url(#gbr-pearl-${ci % PEARLS.length})`} />
                    </g>
                    <g transform="translate(17,-17)">
                      <rect width={cl.name.length * 6 + 30} height="17" rx="8.5" fill="var(--gbr-label-bg)" stroke="rgb(var(--mist-rgb) / 0.14)" />
                      <text x="9" y="12.2" fontSize="10.5" fill={TEXT}>
                        {cl.name}
                      </text>
                      <text x={cl.name.length * 6 + 21} y="12.2" fontSize="9.5" fill={MINT} textAnchor="end" className="font-mono">
                        {cl.pages}
                      </text>
                    </g>
                  </g>
                </g>
              );
            })}

            {/* sector arcs */}
            {ring.sectors.map((s) => {
              const a0 = (s.a0 * Math.PI) / 180;
              const a1 = (s.a1 * Math.PI) / 180;
              const r = 344;
              const large = s.a1 - s.a0 > 180 ? 1 : 0;
              const lit = litClusters.has(s.c);
              return (
                <path
                  key={s.c}
                  d={`M ${Math.cos(a0) * r} ${Math.sin(a0) * r} A ${r} ${r} 0 ${large} 1 ${Math.cos(a1) * r} ${Math.sin(a1) * r}`}
                  fill="none"
                  stroke={PEARLS[s.c % PEARLS.length]}
                  strokeOpacity={litPages != null ? (lit ? 0.55 : manual ? 0.08 : 0.22) : 0.22}
                  strokeWidth="1.5"
                  className="gbr-fade"
                />
              );
            })}

            {/* spokes */}
            {ring.spokes.map((sp, i) => {
              const flip = ((sp.angle % 360) + 360) % 360 > 90 && ((sp.angle % 360) + 360) % 360 < 270;
              const pearlX = R_TRACK + sp.trackLen + PEARL_GAP;
              const color = PEARLS[sp.c % PEARLS.length];
              const fc = freshColor(sp.page.days);
              const yearDim = year != null && sp.page.year !== year;
              // auto queries only BRIGHTEN their matches — dimming the rest is
              // reserved for queries you typed yourself
              const litDim = litPages != null && !litPages.has(sp.pi) && manual;
              const isLit = litPages != null && litPages.has(sp.pi);
              const isHover = hover === i;
              const op = yearDim ? 0.14 : litDim ? 0.3 : 1;
              return (
                <g key={sp.pi} className="gbr-spoke gbr-fade" style={{ animationDelay: `${0.15 + i * 0.014}s`, opacity: op }}>
                  <g transform={`rotate(${sp.angle})`}>
                    {/* numbered base dot (real chunk count) */}
                    <circle cx={R_BASE} r="3.2" fillOpacity="0.8" style={{ fill: 'var(--accent)' }} />
                    <g transform={`translate(${R_BASE},-11) rotate(${-sp.angle})`}>
                      <text textAnchor="middle" fontSize="8.5" fill="rgb(var(--gbr-tint-d) / 0.72)" className="font-mono">
                        {sp.page.chunks}
                      </text>
                    </g>
                    {/* context-health bar — proportional fill, green above half */}
                    <g transform={`translate(${R_PILL},0) ${flip ? 'rotate(180) translate(-56,0)' : ''}`}>
                      <rect x="0" y="-7" width="56" height="14" rx="7" fill="color-mix(in srgb, var(--bg) 80%, transparent)" stroke="rgb(var(--mist-rgb) / 0.12)" />
                      <rect x="5" y="-3" width="22" height="6" rx="3" fill="rgb(var(--mist-rgb) / 0.14)" />
                      <rect
                        x="5"
                        y="-3"
                        width={Math.max(2, (22 * sp.page.health) / 100)}
                        height="6"
                        rx="3"
                        fill={healthColor(sp.page.health)}
                      />
                      <text x="51" y="3.2" textAnchor="end" fontSize="8.5" fill="var(--gbr-pill-text)" className="font-mono">
                        {sp.page.health}%
                      </text>
                    </g>
                    {/* capsule track (length = relative page size) */}
                    <rect
                      x={R_TRACK}
                      y="-5"
                      width={sp.trackLen}
                      height="10"
                      rx="5"
                      fill="color-mix(in srgb, var(--accent) 4%, transparent)"
                      stroke={isLit || isHover ? 'color-mix(in srgb, var(--accent) 60%, transparent)' : 'color-mix(in srgb, var(--accent) 18%, transparent)'}
                      className="gbr-fade"
                    />
                    <line x1={R_TRACK + sp.trackLen} y1="0" x2={pearlX - 10} y2="0" stroke="color-mix(in srgb, var(--accent) 30%, transparent)" />
                    {/* pearl */}
                    <g
                      transform={`translate(${pearlX},0)`}
                      onMouseEnter={() => setHover(i)}
                      onMouseLeave={() => setHover(null)}
                      onClick={() => selectPage(sp.pi)}
                      style={{ cursor: 'pointer' }}
                    >
                      <g className="gbr-node">
                        <circle r="24" fill={`url(#gbr-glow-${sp.c % PEARLS.length})`} className="gbr-glowc" />
                        <circle r="13" fill="none" stroke={color} strokeOpacity={isLit || isHover ? 0.85 : 0.35} strokeDasharray="1 2.2" className="gbr-fade" />
                        <circle r="10" fill="none" stroke={color} strokeOpacity={isLit || isHover ? 0.6 : 0.25} className="gbr-fade" />
                        <circle r={6.4} fill={`url(#gbr-pearl-${sp.c % PEARLS.length})`} />
                      </g>
                    </g>
                    {/* label + freshness chip */}
                    <g transform={`translate(${pearlX + 19},0) ${flip ? 'rotate(180)' : ''}`}>
                      <text
                        y="-1"
                        textAnchor={flip ? 'end' : 'start'}
                        fontSize="10"
                        fill={isLit || isHover ? 'var(--text)' : 'rgb(var(--mist-rgb) / 0.72)'}
                        className="gbr-fade"
                      >
                        {sp.page.t.length > 19 ? `${sp.page.t.slice(0, 18)}…` : sp.page.t}
                      </text>
                      <text y="11" textAnchor={flip ? 'end' : 'start'} fontSize="8" fill={fc} fillOpacity="0.9" className="font-mono">
                        {freshLabel(sp.page.days)}
                      </text>
                    </g>
                  </g>
                </g>
              );
            })}

            {/* hover tooltip — upright, near the pearl */}
            {hover != null && ring.spokes[hover] && (
              <g
                transform={(() => {
                  const sp = ring.spokes[hover];
                  const a = (sp.angle * Math.PI) / 180;
                  const r = R_TRACK + sp.trackLen + PEARL_GAP;
                  return `translate(${Math.cos(a) * r},${Math.sin(a) * r - 34})`;
                })()}
                pointerEvents="none"
              >
                <rect x="-78" y="-30" width="156" height="26" rx="8" fill="color-mix(in srgb, var(--elevated) 94%, transparent)" stroke="color-mix(in srgb, var(--accent) 40%, transparent)" />
                <text y="-13" textAnchor="middle" fontSize="9.5" fill={TEXT}>
                  {ring.spokes[hover].page.words.toLocaleString()} words · {ring.spokes[hover].page.chunks} chunks · click to expand
                </text>
              </g>
            )}
          </g>
        </svg>
      )}

      {/* ——— EXPANDED VIEW ——— */}
      {data.mounted && sel !== null && (
        <div className={`absolute inset-0 z-10 ${exiting === 'panel' ? 'gbr-pv-out' : 'gbr-pv'}`}>
          <ExpandedView
            data={data}
            c={sel.c}
            pi={sel.pi}
            filter={filter}
            setFilter={setFilter}
            onSelect={(pi) => setSel({ c: sel.c, pi })}
            onBack={closeSel}
            onAsk={(q) => ask(q)}
            g={g}
          />
        </div>
      )}

      {/* legend */}
      {!g && ringVisible && (
        <div
          className={`glass-face absolute bottom-7 left-7 z-20 w-[196px] rounded-[24px] p-4 max-sm:hidden ${ringAnim}`}
        >
          <div className="mb-2.5 text-[9.5px] tracking-[0.3em]" style={{ color: 'var(--text-2)' }}>
            LEGEND
          </div>
          {data.clusters.map((c, i) => (
            <button
              key={c.key}
              onClick={() => selectFolder(i)}
              className="gbr-widget flex w-full items-center gap-2.5 rounded-[7px] px-1 py-[4.5px] text-left hover:bg-white/[0.05]"
            >
              <span
                className="flex h-[17px] min-w-[30px] items-center justify-center gap-1 rounded-full border px-1.5"
                style={{ borderColor: 'rgb(var(--mist-rgb) / 0.16)' }}
              >
                <span className="h-2 w-2 rounded-full" style={{ background: PEARLS[i % PEARLS.length] }} />
                <span className="font-mono text-[8.5px] tabular-nums" style={{ color: 'var(--text-2)' }}>
                  {c.pages}
                </span>
              </span>
              <span className="text-[11.5px]" style={{ color: 'var(--gbr-legend-text)' }}>
                {c.name}
              </span>
            </button>
          ))}
          <div className="my-2.5 h-px" style={{ background: 'rgb(var(--mist-rgb) / 0.1)' }} />
          {data.freshPct != null && (
            <div className="flex items-center justify-between py-[3px]">
              <span className="text-[11px]" style={{ color: 'var(--text-2)' }}>
                Fresh ≤30d
              </span>
              <span className="rounded-[6px] border px-1.5 py-0.5 font-mono text-[9.5px]" style={{ borderColor: 'color-mix(in srgb, var(--accent) 40%, transparent)', color: MINT }}>
                {data.freshPct}%
              </span>
            </div>
          )}
          {data.stalePct != null && (
            <div className="flex items-center justify-between py-[3px]">
              <span className="text-[11px]" style={{ color: 'var(--text-2)' }}>
                Stale ≥90d
              </span>
              <span className="rounded-[6px] border px-1.5 py-0.5 font-mono text-[9.5px]" style={{ borderColor: 'var(--gbr-amber-line)', color: 'var(--gbr-amber)' }}>
                {data.stalePct}%
              </span>
            </div>
          )}
        </div>
      )}

      {/* Retrieval notices are portalled into the page header (see noticeSlot):
          above the canvas window, level with the Optimal Engine title. Inside the
          frame they competed with the graph for the same corner. */}
      {noticeSlot &&
        createPortal(
          <div
            // Glass over the ring: the radial bleeds up under this notice, so
            // it carries a real drop shadow + heavy blur to separate cleanly.
            // z-40 is load-bearing — the canvas is a LATER sibling of the header
            // slot this portals into, so without it the ring painted on top and
            // backdrop-filter had nothing behind it to frost.
            // a glass face (strong frost over the ring); it shows and hides by visibility and a 6px rise, never a
            // fade (glass brief §4), and an answer lights it: a glow behind and a tinted rim (gl-on), not a fill
            className={`glass-face glass-face--strong ${showAnswer ? 'gl-on' : ''} absolute right-0 top-0 z-40 w-[560px] max-w-full rounded-[24px] px-3.5 py-2 transition-[transform,visibility] duration-500`}
            style={{
              visibility: showAnswer ? 'visible' : 'hidden',
              transform: showAnswer ? 'translateY(0)' : 'translateY(-6px)',
            }}
          >
            <div className="flex items-baseline gap-2">
              <span className="h-1.5 w-1.5 shrink-0 translate-y-[-2px] rounded-full" style={{ background: MINT }} />
              <span className="shrink-0 font-mono text-[10px]" style={{ color: 'var(--text-2)' }}>
                {sources.length > 0 ? `retrieved from ${sources.length} notes` : 'retrieval'}
              </span>
              <span className="min-w-0 flex-1 truncate text-[12.5px]" style={{ color: 'var(--text)' }}>
                {answer}
              </span>
            </div>
            {sources.length > 0 && (
              <div className="mt-1 flex gap-1 overflow-hidden pl-[14px]">
                {sources.slice(0, 3).map((src) => (
                  <span
                    key={src}
                    className="shrink-0 truncate rounded-[5px] border px-1.5 py-0.5 font-mono text-[9px]"
                    style={{ borderColor: 'rgb(var(--mist-rgb) / 0.14)', color: 'var(--gbr-source)' }}
                  >
                    {src}
                  </span>
                ))}
                {sources.length > 3 && (
                  <span className="shrink-0 self-center font-mono text-[9px]" style={{ color: 'var(--gbr-source-dim)' }}>
                    +{sources.length - 3}
                  </span>
                )}
              </div>
            )}
          </div>,
          noticeSlot,
        )}

      {/* Ask bar, top-left under the frame's own title block. It used to own
          the bottom middle, but that is where capture belongs — the operator: "that
          should be what the text box has for when i need to dump stuff in
          there". Retrieval reads better next to the frame's identity anyway,
          and its answer now lands in the header notice above. */}
      {g ? (
        <div className="kg-radial-ask" data-phase={phase}>
          <input
            {...inputProps}
            disabled={!data.mounted}
            placeholder={data.mounted ? 'Ask the brain a question…' : 'brain-store not mounted'}
            aria-label="Ask the brain"
            spellCheck={false}
            className="g-input"
            data-size="md"
          />
          <span className="kg-cap">{statusWord}</span>
        </div>
      ) : (
      <div className="absolute left-5 top-[92px] z-20 w-[380px] max-w-[46vw]">
        <div
          className="flex items-center gap-2.5 rounded-full border px-4 py-2 transition-all duration-300"
          style={{
            background: 'var(--gbr-pill-bg)',
            borderColor: phase === 'typing' || phase === 'scan' ? 'color-mix(in srgb, var(--accent) 55%, transparent)' : 'var(--gbr-ask-line)',
            boxShadow: phase === 'typing' || phase === 'scan' ? '0 0 32px color-mix(in srgb, var(--accent) 14%, transparent)' : 'var(--gbr-ask-shadow)',
            backdropFilter: 'blur(10px)',
          }}
        >
          <span className="shrink-0 font-mono text-[11.5px]" style={{ color: MINT }}>
            optimal-engine ›
          </span>
          <input
            {...inputProps}
            disabled={!data.mounted}
            placeholder={data.mounted ? 'ask a question…' : 'brain-store not mounted'}
            spellCheck={false}
            className="min-h-[16px] w-full flex-1 border-0 bg-transparent font-mono text-[11.5px] outline-none"
            style={{ color: 'var(--text)', caretColor: MINT }}
          />
          <span className="shrink-0 font-mono text-[9.5px]" style={{ color: 'var(--text-3)' }}>
            {statusWord}
          </span>
        </div>
      </div>
      )}

      {!g && (
      <div className="pointer-events-none absolute bottom-2 right-7 z-20 font-mono text-[9px]" style={{ color: 'var(--gbr-footer)' }}>
        {manual ? 'live retrieval' : 'auto queries · live retrieval'}
      </div>
      )}

      <style>{`
        .gbr-corner { position: absolute; width: 22px; height: 22px; border-color: rgb(var(--gbr-tint-e) / 0.16); border-style: solid; border-width: 0; z-index: 20; }
        .gbr-lt { left: 14px; top: 14px; border-left-width: 1px; border-top-width: 1px; }
        .gbr-rt { right: 14px; top: 14px; border-right-width: 1px; border-top-width: 1px; }
        .gbr-lb { left: 14px; bottom: 14px; border-left-width: 1px; border-bottom-width: 1px; }
        .gbr-rb { right: 14px; bottom: 14px; border-right-width: 1px; border-bottom-width: 1px; }
        .gbr-spoke { animation: gbr-grow 0.65s cubic-bezier(0.2, 0.7, 0.2, 1) both; transform-box: view-box; transform-origin: center; }
        @keyframes gbr-grow { from { opacity: 0; transform: scale(0.72); } to { transform: scale(1); } }
        .gbr-fade { transition: opacity 0.45s ease, stroke-opacity 0.45s ease, fill 0.3s ease, stroke 0.3s ease; }
        .gbr-tick { animation: gbr-rotate 90s linear infinite; transform-box: view-box; transform-origin: center; }
        @keyframes gbr-rotate { to { transform: rotate(360deg); } }
        /* hover pops — nodes glow and lift */
        .gbr-node, .gbr-fnode { transition: transform 0.28s cubic-bezier(0.2, 0.9, 0.3, 1.4); transform-box: fill-box; transform-origin: center; }
        .gbr-node:hover { transform: scale(1.35); }
        .gbr-fnode:hover { transform: scale(1.18); }
        .gbr-glowc { opacity: 0; transition: opacity 0.3s ease; }
        .gbr-node:hover .gbr-glowc, .gbr-fnode:hover .gbr-glowc { opacity: 1; }
        .gbr-widget { transition: transform 0.2s ease, box-shadow 0.25s ease, background 0.2s ease, border-color 0.2s ease; }
        .gbr-widget:hover { transform: translateY(-2px); box-shadow: 0 0 22px color-mix(in srgb, var(--accent) 8%, transparent), 0 12px 28px var(--gbr-widget-shadow); }
        /* view transitions: the views hold glass (the legend card, the year chips, the expanded cards), so
           they move without fading (glass brief §3.2, §4); only the ring itself, an SVG leaf, fades. */
        .gbr-vin { animation: gbr-view-in 0.45s cubic-bezier(0.2, 0.7, 0.2, 1) both; }
        .gbr-vout { animation: gbr-view-out 0.2s linear forwards; }
        @keyframes gbr-view-in { from { transform: scale(0.985); } }
        @keyframes gbr-view-out { to { transform: scale(0.98); } }
        .gbr-leaf.gbr-vin { animation-name: gbr-leaf-in; }
        .gbr-leaf.gbr-vout { animation-name: gbr-leaf-out; }
        @keyframes gbr-leaf-in { from { opacity: 0; transform: scale(0.985); } }
        @keyframes gbr-leaf-out { to { opacity: 0; transform: scale(0.99); } }
        .gbr-pv { animation: gbr-panel-in var(--gl-dur-materialise) var(--gl-spring) both; }
        .gbr-pv-out { animation: gbr-panel-out 0.22s linear forwards; }
        @keyframes gbr-panel-in { from { transform: translateY(18px) scale(0.965); } }
        @keyframes gbr-panel-out { to { transform: scale(0.98); } }
        @media (prefers-reduced-motion: reduce) {
          .gbr-spoke, .gbr-tick, .gbr-vin, .gbr-vout, .gbr-leaf.gbr-vin, .gbr-leaf.gbr-vout, .gbr-pv, .gbr-pv-out { animation: none; }
          .gbr-node, .gbr-fnode, .gbr-widget { transition: none; }
        }
      `}</style>
    </div>
  );
}

/** The Example-6 expanded view: folder core → capsule filters → fan → detail. */
function ExpandedView({
  data,
  c,
  pi,
  filter,
  setFilter,
  onSelect,
  onBack,
  onAsk,
  g = false,
}: {
  g?: boolean;
  data: ObservatoryData;
  c: number;
  pi: number;
  filter: 'all' | 'fresh' | 'stale' | 'linked';
  setFilter: (f: 'all' | 'fresh' | 'stale' | 'linked') => void;
  onSelect: (pi: number) => void;
  onBack: () => void;
  onAsk: (q: string) => void;
}) {
  const palette = usePalette();
  const PEARLS = g && palette.series.length ? palette.series : palette.pearls;
  const cluster = data.clusters[c];
  const color = PEARLS[c % PEARLS.length];
  const folderPages = useMemo(
    () =>
      data.pages
        .map((p, i) => ({ p, i }))
        .filter((x) => x.p.c === c)
        .sort((x, y) => y.p.chunks - x.p.chunks),
    [data.pages, c],
  );
  const counts = {
    all: folderPages.length,
    fresh: folderPages.filter((x) => x.p.days != null && x.p.days <= 30).length,
    stale: folderPages.filter((x) => x.p.days != null && x.p.days >= 90).length,
    linked: folderPages.filter((x) => x.p.links > 0).length,
  };
  const filtered = folderPages.filter((x) => {
    if (filter === 'fresh') return x.p.days != null && x.p.days <= 30;
    if (filter === 'stale') return x.p.days != null && x.p.days >= 90;
    if (filter === 'linked') return x.p.links > 0;
    return true;
  });
  const ROWS = 14;
  const shown = filtered.slice(0, ROWS);
  const page = data.pages[pi];
  const folderChunks = folderPages.reduce((s, x) => s + x.p.chunks, 0) || 1;
  const maxChunkWords = Math.max(1, ...page.chunkWords);
  const modified =
    page.days != null ? new Date(Date.now() - page.days * 86_400_000).toLocaleDateString('en-US', { day: 'numeric', month: 'short', year: 'numeric' }) : null;

  const ROW_H = 46;
  const fanH = Math.max(shown.length, 1) * ROW_H;
  const CAPS: { key: typeof filter; label: string; n: number }[] = [
    { key: 'all', label: 'Pages', n: counts.all },
    { key: 'fresh', label: 'Fresh ≤30d', n: counts.fresh },
    { key: 'stale', label: 'Stale ≥90d', n: counts.stale },
    { key: 'linked', label: 'Wikilinked', n: counts.linked },
  ];

  return (
    <div className="flex h-full items-center gap-5 px-8 pb-28 pt-16">
      {/* left — folder core + capsule filters */}
      <div className="flex shrink-0 items-center gap-5">
        <div className="relative flex flex-col items-center">
          {g ? (
            <button type="button" onClick={onBack} className="g-btn kg-radial-back" data-variant="default" data-size="sm">
              Back to the ring
            </button>
          ) : (
          <button onClick={onBack} className="absolute -top-10 left-0 text-[11.5px] transition-colors hover:text-white" style={{ color: 'var(--text-2)' }}>
            ← ring
          </button>
          )}
          <div
            className="flex h-[150px] w-[150px] items-center justify-center rounded-full"
            style={{ background: 'var(--gbr-circle-bg)', border: '1px solid rgb(var(--mist-rgb) / 0.08)' }}
          >
            <div
              className="h-[54px] w-[54px] rounded-full"
              style={{
                background: `radial-gradient(circle at 35% 30%, var(--gbr-white) 0%, ${color} 30%, color-mix(in srgb, ${color} 33%, transparent) 100%)`,
                boxShadow: g ? undefined : `0 0 42px color-mix(in srgb, ${color} 40%, transparent)`,
              }}
            />
          </div>
          <div className="mt-4 text-center">
            <div className="text-[15px] font-medium">{cluster.name}</div>
            <div className="mt-1 font-mono text-[10.5px]" style={{ color: 'var(--text-3)' }}>
              {counts.all} pages · {folderChunks} chunks
            </div>
          </div>
        </div>
        <div className="flex flex-col gap-3">
          {CAPS.map((cap) => {
            const on = filter === cap.key;
            return (
              <button
                key={cap.key}
                onClick={() => setFilter(cap.key)}
                aria-pressed={g ? on : undefined}
                className={g ? 'g-chip kg-radial-cap' : 'gbr-widget w-[148px] rounded-[13px] border px-3.5 py-2.5 text-left'}
                style={
                  g
                    ? undefined
                    : on
                    ? { background: 'color-mix(in srgb, var(--accent) 8%, transparent)', borderColor: 'color-mix(in srgb, var(--accent) 55%, transparent)', boxShadow: '0 0 24px color-mix(in srgb, var(--accent) 10%, transparent)' }
                    : { background: 'color-mix(in srgb, var(--bg) 80%, transparent)', borderColor: 'rgb(var(--mist-rgb) / 0.1)' }
                }
              >
                {g ? (
                  <>
                    <span className="kg-radial-cap-n">{cap.n}</span>
                    <span>{cap.label}</span>
                  </>
                ) : (
                  <>
                <div className="text-[17px] font-medium tabular-nums" style={{ color: on ? 'var(--gbr-mint-bright)' : TEXT }}>
                  {cap.n}
                </div>
                <div className="mt-0.5 text-[10.5px]" style={{ color: on ? 'var(--gbr-mint-mid)' : 'var(--text-3)' }}>
                  {cap.label}
                </div>
                  </>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* fan + rows */}
      <div className="relative flex min-w-0 flex-1 items-center">
        <svg width="90" height={fanH} viewBox={`0 0 90 ${fanH}`} className="shrink-0" style={{ overflow: 'visible' }}>
          {shown.map((x, i) => {
            const y = i * ROW_H + ROW_H / 2;
            const oy = fanH / 2;
            return (
              <path
                key={x.i}
                d={`M 0 ${oy} C 36 ${oy}, 56 ${y}, 88 ${y}`}
                fill="none"
                stroke={x.i === pi ? 'color-mix(in srgb, var(--accent) 75%, transparent)' : 'color-mix(in srgb, var(--accent) 22%, transparent)'}
                strokeWidth={x.i === pi ? 1.5 : 1}
              />
            );
          })}
        </svg>
        <div className="min-w-0 flex-1">
          {shown.map((x) => {
            const on = x.i === pi;
            return (
              <button
                key={x.i}
                onClick={() => onSelect(x.i)}
                aria-pressed={g ? on : undefined}
                className={g ? 'g-btn kg-radial-row' : 'gbr-widget flex w-full items-center gap-2.5 border-b py-2 pl-2 pr-3 text-left'}
                style={g ? { height: ROW_H } : { height: ROW_H, borderColor: 'rgb(var(--mist-rgb) / 0.05)', background: on ? 'color-mix(in srgb, var(--accent) 5%, transparent)' : 'transparent' }}
              >
                <span className="h-[7px] w-[7px] shrink-0 rounded-full" style={{ background: on ? MINT : `color-mix(in srgb, ${color} 60%, transparent)` }} />
                <span className="min-w-0 flex-1 truncate text-[12px]" style={{ color: on ? 'var(--text)' : 'var(--gbr-row-text)' }}>
                  {x.p.t}
                </span>
                <span className="rounded-[5px] border px-1.5 py-px font-mono text-[8.5px]" style={{ borderColor: 'color-mix(in srgb, var(--accent) 35%, transparent)', color: 'var(--h6-mint)' }}>
                  {x.p.chunks} ch
                </span>
                <span className="font-mono text-[8.5px]" style={{ color: freshColor(x.p.days) }}>
                  {freshLabel(x.p.days)}
                </span>
              </button>
            );
          })}
          {filtered.length > ROWS && (
            <div className="py-2 pl-2 font-mono text-[9.5px]" style={{ color: 'var(--text-3)' }}>
              +{filtered.length - ROWS} more by chunk count
            </div>
          )}
          {filtered.length === 0 && (
            <div className="py-2 pl-2 text-[11.5px]" style={{ color: 'var(--text-3)' }}>
              no pages match this filter
            </div>
          )}
        </div>
      </div>

      {/* detail panel */}
      <div
        className="glass-face flex h-full w-[330px] shrink-0 flex-col overflow-y-auto rounded-[24px] p-5"
      >
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: color }} />
          <span className="text-[11px]" style={{ color: 'var(--text-2)' }}>
            {cluster.name}
          </span>
        </div>
        <div className="mt-2 text-[19px] font-medium leading-snug" style={{ color: 'var(--text)' }}>
          {page.title}
        </div>
        <div className="mt-3 flex items-center gap-2.5 rounded-[11px] border p-2.5" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}>
          <span
            className="gbr-state-pill rounded-full border px-2 py-0.5 text-[9.5px]"
            data-fresh={page.days != null && page.days <= 30 ? '' : undefined}
            style={{ borderColor: `color-mix(in srgb, ${freshColor(page.days)} 40%, transparent)`, color: freshColor(page.days) }}
          >
            {page.days != null && page.days <= 30 ? 'Fresh' : page.days != null && page.days >= 90 ? 'Stale' : 'Tracked'}
          </span>
          <span className="flex-1 text-[10.5px] leading-snug" style={{ color: 'var(--text-2)' }}>
            {modified ? `last modified ${modified}` : 'no modification date on file'}
          </span>
          <button
            onClick={() => onAsk(page.t.replace(/\.md$/, '').split('/').pop() ?? page.t)}
            className={g ? 'g-btn' : 'gbr-widget rounded-[8px] border px-2.5 py-1.5 text-[10.5px] hover:bg-white/[0.06]'}
            data-size={g ? 'sm' : undefined}
            style={g ? undefined : { borderColor: 'color-mix(in srgb, var(--accent) 45%, transparent)', color: 'var(--h6-mint)' }}
          >
            Ask the brain ›
          </button>
        </div>

        {/* size card — real chunk-size distribution */}
        <div className="gbr-widget mt-3 rounded-[13px] border p-3.5" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)', background: 'var(--gbr-size-bg)' }}>
          <div className="flex items-baseline justify-between">
            <span className="text-[11px]" style={{ color: 'var(--text-2)' }}>
              Page size
            </span>
            <span className="rounded-full border px-2 py-0.5 text-[9px]" style={{ borderColor: 'color-mix(in srgb, var(--accent) 40%, transparent)', color: MINT }}>
              {page.chunks} chunks
            </span>
          </div>
          <div className="mt-1 text-[30px] font-normal tabular-nums leading-none" style={{ color: 'var(--text)' }}>
            {page.words.toLocaleString()}
            <span className="ml-1.5 text-[11px]" style={{ color: 'var(--text-3)' }}>
              words
            </span>
          </div>
          <div className="mt-3 flex h-[34px] items-end gap-[3px]">
            {page.chunkWords.map((wds, i) => (
              <div
                key={i}
                className="w-[7px] rounded-t-[2px]"
                style={{ height: `${Math.max(12, (wds / maxChunkWords) * 100)}%`, background: `color-mix(in srgb, var(--accent) ${i % 2 ? 45 : 75}%, transparent)` }}
                title={`chunk ${i + 1}: ${wds} words`}
              />
            ))}
          </div>
        </div>

        {/* stat tiles */}
        <div className="mt-3 grid grid-cols-2 gap-2.5">
          {[
            { n: page.chunks, l: 'Embedded chunks' },
            { n: page.links, l: 'Wikilinks' },
            { n: page.days != null ? `${page.days}d` : '—', l: 'Since last edit' },
            { n: `${Math.max(1, Math.round((page.chunks / folderChunks) * 100))}%`, l: 'Of folder chunks' },
          ].map((tile) => (
            <div key={tile.l} className="gbr-widget rounded-[12px] border p-3" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}>
              <div className="text-[19px] font-normal tabular-nums leading-none" style={{ color: 'var(--gbr-tile-text)' }}>
                {tile.n}
              </div>
              <div className="mt-1.5 text-[10px] leading-snug" style={{ color: 'var(--text-3)' }}>
                {tile.l}
              </div>
            </div>
          ))}
        </div>

        {/* excerpt */}
        {page.excerpt && (
          <div className="gbr-widget mt-3 rounded-[13px] border p-3.5" style={{ borderColor: 'rgb(var(--mist-rgb) / 0.09)' }}>
            <div className="mb-1.5 flex items-center gap-1.5 text-[11px]" style={{ color: 'var(--text-2)' }}>
              From the page
            </div>
            <p className="text-[11.5px] leading-relaxed" style={{ color: 'var(--text-2)' }}>
              {page.excerpt}
              {page.excerpt.length >= 219 ? '…' : ''}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
