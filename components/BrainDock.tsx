'use client';

/**
 * The capture dock — how you get something INTO the brain from the Optimal Engine page.
 *
 * It has two states and morphs between them:
 *
 *   open      a small card at the bottom middle, growing upward out of the
 *             line. Type, dictate, drag documents onto it, or hit Upload. It
 *             does NOT sit in the middle of the canvas — the graph owns that.
 *   collapsed a slim line at the bottom middle. This is the spot the ask bar
 *             used to hold; capture earned it because it is the thing you
 *             reach for. Point at it and the card comes back with whatever you
 *             had typed still in it.
 *
 * It opens on HOVER and stays open while you are actually using it — clicked
 * into, typing, or focused from the keyboard. There is no idle timer: a box
 * that shrinks out from under you while you are looking at it is worse than
 * one you have to point at. Small by default, big while your mouse is on it.
 *
 * A draft is never thrown away: BrainDump stays mounted the whole time and is
 * only hidden, so collapsing mid-sentence costs nothing — and the collapsed
 * line says so when one is waiting.
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { BrainCircuit } from 'lucide-react';
import { BrainDump } from '@/components/BrainDump';

/** grace period when the pointer crosses the gap between the line and the card */
const LEAVE_MS = 180;

/** `g`: the GladOS style (S05 F05.12): a kit pill line and a kit card, same hover, pin, drag and draft rules. */
export function BrainDock({ g = false }: { g?: boolean } = {}) {
  const [hover, setHover] = useState(false);
  /** clicked into — survives the pointer leaving, e.g. while a file dialog is up */
  const [pinned, setPinned] = useState(false);
  const [focused, setFocused] = useState(false);
  const [hasDraft, setHasDraft] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);
  const leaveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const open = hover || pinned || focused;

  // The line and the card are two separate elements that swap places, so the
  // pointer briefly touches neither. A short grace period on leave stops that
  // gap from flickering the card shut and straight back open.
  const enter = useCallback(() => {
    if (leaveTimer.current) clearTimeout(leaveTimer.current);
    setHover(true);
  }, []);
  const leave = useCallback(() => {
    if (leaveTimer.current) clearTimeout(leaveTimer.current);
    leaveTimer.current = setTimeout(() => setHover(false), LEAVE_MS);
  }, []);

  useEffect(
    () => () => {
      if (leaveTimer.current) clearTimeout(leaveTimer.current);
    },
    [],
  );

  // clicking inside pins it open; clicking anywhere else lets it go
  useEffect(() => {
    const onDown = (e: PointerEvent) => {
      setPinned(!!wrapRef.current?.contains(e.target as Node));
    };
    document.addEventListener('pointerdown', onDown);
    return () => document.removeEventListener('pointerdown', onDown);
  }, []);

  // a dragged file opens the dock, so you can drop onto the collapsed line
  useEffect(() => {
    const onDragOver = (e: DragEvent) => {
      if (e.dataTransfer?.types?.includes('Files')) setPinned(true);
    };
    window.addEventListener('dragover', onDragOver);
    return () => window.removeEventListener('dragover', onDragOver);
  }, []);

  // watch the textarea so the collapsed line can say a draft is waiting
  useEffect(() => {
    const el = wrapRef.current?.querySelector('textarea');
    if (!el) return;
    const onInput = () => setHasDraft(el.value.trim().length > 0);
    el.addEventListener('input', onInput);
    return () => el.removeEventListener('input', onInput);
  }, [open]);

  const hoverProps = { onMouseEnter: enter, onMouseLeave: leave };

  if (g) {
    return (
      <div ref={wrapRef} className="kg-dock" data-open={open || undefined}>
        <button
          type="button"
          {...hoverProps}
          onClick={() => setPinned(true)}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          aria-label="Open capture"
          aria-expanded={open}
          className="g-btn kg-dock-line"
          data-draft={hasDraft || undefined}
        >
          <BrainCircuit size={14} strokeWidth={1.7} aria-hidden="true" />
          {hasDraft ? 'Draft waiting: click to finish' : 'Dump into the brain'}
        </button>
        <div className="kg-dock-card glass-face glass-face--strong" {...hoverProps} onFocus={() => setFocused(true)} onBlur={() => setFocused(false)}>
          <BrainDump compact g />
        </div>
      </div>
    );
  }

  return (
    <div ref={wrapRef}>
      {/* ── collapsed: the line along the bottom ───────────────────────── */}
      <button
        {...hoverProps}
        onClick={() => setPinned(true)}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        aria-label="Open capture"
        aria-expanded={open}
        className="bd-line group absolute bottom-[52px] left-1/2 z-30 flex items-center gap-2.5 whitespace-nowrap rounded-full border px-5 py-2.5"
        style={{
          background: 'color-mix(in srgb, var(--elevated) 90%, transparent)',
          borderColor: hasDraft ? 'color-mix(in srgb, var(--accent) 55%, transparent)' : 'color-mix(in srgb, var(--accent) 28%, transparent)',
          boxShadow: hasDraft
            ? '0 0 26px -6px color-mix(in srgb, var(--accent) 50%, transparent), inset 0 0 0 1px color-mix(in srgb, var(--accent) 12%, transparent)'
            : '0 0 18px -8px color-mix(in srgb, var(--accent) 35%, transparent)',
          backdropFilter: 'blur(10px)',
          opacity: open ? 0 : 1,
          pointerEvents: open ? 'none' : 'auto',
          transform: `translateX(-50%) translateY(${open ? 10 : 0}px)`,
        }}
      >
        <BrainCircuit className="h-3.5 w-3.5" strokeWidth={1.7} style={{ color: 'var(--accent)' }} />
        <span className="text-[12px]" style={{ color: 'var(--text)' }}>
          {hasDraft ? 'Draft waiting — click to finish' : 'Dump into the brain'}
        </span>
        <span className="font-mono text-[10px] max-sm:hidden" style={{ color: 'var(--text-3)' }}>
          type · talk · drop
        </span>
      </button>

      {/* ── open: the card over the graph ──────────────────────────────── */}
      <div
        // the open dump is a strong glass face, lit: a tinted rim and a glow behind it. On a face this large the glow
        // is a sibling behind the face, moved out to its edges (glass canon amendment), so the face frosts it without
        // washing the text. The card opens by visibility and scale, never a fade (glass brief §4).
        className="bd-card gl-glow-host absolute bottom-[46px] left-1/2 z-30 w-[440px] max-w-[86vw] rounded-[24px]"
        style={{
          visibility: open ? 'visible' : 'hidden',
          pointerEvents: open ? 'auto' : 'none',
          // grows upward out of the collapsed line rather than floating in the
          // middle of the canvas — the graph owns the middle
          transformOrigin: 'bottom center',
          transform: `translateX(-50%) scale(${open ? 1 : 0.94}) translateY(${open ? 0 : 8}px)`,
        }}
        {...hoverProps}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
      >
        <span className="gl-glow" aria-hidden />
        <div className="glass-face glass-face--strong gl-on-rim rounded-[24px] p-3">
          <BrainDump compact />
        </div>
      </div>

      <style dangerouslySetInnerHTML={{ __html: `
        .bd-line { transition: opacity 0.32s ease, transform 0.32s cubic-bezier(0.2,0.7,0.2,1), box-shadow 0.3s ease, border-color 0.3s ease; }
        .bd-card { transition: visibility 0.32s, transform 0.32s cubic-bezier(0.2,0.7,0.2,1); } /* a glass face: never fades */
        .bd-line:hover { border-color: color-mix(in srgb, var(--accent) 60%, transparent); box-shadow: 0 0 28px -6px color-mix(in srgb, var(--accent) 55%, transparent); }
        @media (prefers-reduced-motion: reduce) { .bd-card, .bd-line { transition: none; } }
      ` }} />
    </div>
  );
}
