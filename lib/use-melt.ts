'use client';

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type AnimationEvent, type RefObject } from 'react';

/**
 * Melt, the glass exit (FounderOS Glass brief §4): on close a face plays `.is-melting`
 * (app/globals.css: rim, pool and frost to 0, scale to .98, linear, 8 frames) and unmounts on
 * its own animationend. The close itself is immediate: the parent's `open` goes false, the
 * dialog behaviour (focus return, scroll lock, Escape) releases at once, and only the pixels
 * linger for the melt. Under reduced motion the melt rule is `animation: none`, so animationend
 * never fires: the face goes at once (and a timer just past the melt covers anything else that
 * stops the animation, such as a hidden tab).
 *
 *   const melt = useMeltPhase(open);               // in the component that decides mounting
 *   if (!melt.mounted) return null;
 *   const onMeltEnd = useMeltEnd(faceRef, melt.melting, melt.end);
 *   <div ref={faceRef} className={cx('g-sheet', melt.melting && 'is-melting')} onAnimationEnd={onMeltEnd}>
 */

/** 8 frames at 24fps (--gl-dur-melt). */
export const MELT_MS = 333;
/** Unmount anyway this long after the melt starts. */
export const MELT_FALLBACK_MS = MELT_MS + 250;

export type MeltPhase = 'closed' | 'open' | 'melting';
export type MeltEvent = 'open' | 'close' | 'end';

/** Pure: the next phase. Reopening mid-melt keeps the face; a stray end or close changes nothing. */
export function meltNext(phase: MeltPhase, ev: MeltEvent): MeltPhase {
  if (ev === 'open') return 'open';
  if (ev === 'close') return phase === 'open' ? 'melting' : phase;
  return phase === 'melting' ? 'closed' : phase;
}

/** Pure: true when animationend will never come (reduced motion, or the face is not playing gl-melt). */
export function skipMelt({ reducedMotion, animationName }: { reducedMotion: boolean; animationName: string | null | undefined }): boolean {
  if (reducedMotion) return true;
  return !(animationName ?? '').split(/\s*,\s*/).includes('gl-melt');
}

/** Pure: is this animationend the face's own melt (not a child's animation bubbling up)? */
export function isOwnMeltEnd(e: { target: unknown; currentTarget: unknown; animationName: string }): boolean {
  return e.target === e.currentTarget && e.animationName === 'gl-melt';
}

export type Leaving<T> = { item: T; leaving: boolean };

/**
 * Pure: a list that keeps removed items in their slot, marked leaving, until they melt. Present
 * items take their newest version; new ones are appended. Returns `prev` itself when nothing
 * changed, so it is safe to call on every render.
 */
export function mergeLeaving<T extends { id: number | string }>(prev: readonly Leaving<T>[], next: readonly T[]): Leaving<T>[] {
  const byId = new Map(next.map((t) => [t.id, t]));
  const seen = new Set<T['id']>();
  let changed = false;
  const out: Leaving<T>[] = prev.map((p) => {
    const now = byId.get(p.item.id);
    if (now) {
      seen.add(now.id);
      if (now === p.item && !p.leaving) return p;
      changed = true;
      return { item: now, leaving: false };
    }
    if (p.leaving) return p;
    changed = true;
    return { item: p.item, leaving: true };
  });
  for (const t of next) {
    if (seen.has(t.id)) continue;
    changed = true;
    out.push({ item: t, leaving: false });
  }
  return changed ? out : (prev as Leaving<T>[]);
}

/** Mount state for a face that melts out: `mounted` stays true through the melt; call `end` when it is done. */
export function useMeltPhase(open: boolean): { mounted: boolean; melting: boolean; end: () => void } {
  const [phase, setPhase] = useState<MeltPhase>(open ? 'open' : 'closed');
  const [was, setWas] = useState(open);
  let current = phase;
  if (open !== was) {
    // React's derive-from-props pattern: update during render, no extra commit with a stale phase.
    current = meltNext(phase, open ? 'open' : 'close');
    setWas(open);
    setPhase(current);
  }
  const end = useCallback(() => setPhase((p) => meltNext(p, 'end')), []);
  return { mounted: current !== 'closed', melting: current === 'melting', end };
}

const useIsoLayoutEffect = typeof window === 'undefined' ? useEffect : useLayoutEffect;

function prefersReducedMotion(): boolean {
  try {
    return Boolean(window.matchMedia?.('(prefers-reduced-motion: reduce)').matches);
  } catch {
    return false;
  }
}

/**
 * While `melting`, wait for the face's own melt to end, then call `onGone`. Calls it at once when
 * no melt will play (reduced motion, `animation: none`, no face), and after MELT_FALLBACK_MS at
 * the latest. Returns the face's onAnimationEnd handler.
 */
export function useMeltEnd(ref: RefObject<HTMLElement | null>, melting: boolean, onGone: () => void): (e: AnimationEvent<HTMLElement>) => void {
  const gone = useRef(onGone);
  gone.current = onGone;
  useIsoLayoutEffect(() => {
    if (!melting) return;
    const el = ref.current;
    const name = el ? getComputedStyle(el).animationName : null;
    if (!el || skipMelt({ reducedMotion: prefersReducedMotion(), animationName: name })) {
      gone.current();
      return;
    }
    const t = setTimeout(() => gone.current(), MELT_FALLBACK_MS);
    return () => clearTimeout(t);
  }, [melting, ref]);
  return useCallback(
    (e: AnimationEvent<HTMLElement>) => {
      if (melting && isOwnMeltEnd(e)) gone.current();
    },
    [melting],
  );
}

/**
 * Melt for a face its parent unmounts on close (an ad-hoc modal with no `open` prop): `close` starts
 * the melt and the parent's `onClose` runs once the face has melted (at once under reduced motion,
 * MELT_FALLBACK_MS at the latest). Put `layer` on the element that holds the scrim and the face (the
 * scrim clears with the face and stops taking clicks), `faceRef` and `onAnimationEnd` on the face,
 * and add `is-melting` to the face while `melting`. Pass the face's own ref when it has one (useDialog).
 *
 *   const m = useMeltClose(onClose);
 *   <div className="fixed inset-0 z-[100] grid place-items-center" {...m.layer}>
 *     <div className="gl-scrim" aria-hidden onClick={m.close} />
 *     <div ref={m.faceRef} onAnimationEnd={m.onAnimationEnd} className={cx('c-pop glass-face--strong gl-materialise relative', m.melting && 'is-melting')} />
 */
export function useMeltClose<T extends HTMLElement = HTMLDivElement>(onClose: () => void, ref?: RefObject<T>) {
  const own = useRef<T>(null);
  const faceRef = ref ?? own;
  const [melting, setMelting] = useState(false);
  const close = useCallback(() => setMelting(true), []);
  const onAnimationEnd = useMeltEnd(faceRef, melting, onClose);
  return { melting, close, faceRef, onAnimationEnd, layer: { 'data-melting': melting ? '' : undefined } as const };
}
