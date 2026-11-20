/** TS constants for the shared Lab v2 components (02 section 1). Colours live only in app/globals.css. */

/**
 * How many Lab Glass surfaces may frost at once (Glass.tsx grants the slots by priority; the rest
 * paint the solid face). It was 12 when Lab glass blurred at 20px. Under the glass pack a page
 * widget takes the light frost, blur(3px) saturate(150%); only sheets and drawers keep the 28px
 * strong frost, and they are priority 0 and 1, one or two at a time.
 *
 * Why 24: a backdrop blur samples a kernel a few sigma wide each way, so a 3px frost reads a small
 * fraction of the pixels a 20px one does (the compositor downscales large radii first, which
 * narrows the gap: count a 3px surface at a third to a half of a 20px one, not a sixth). 24 light
 * surfaces therefore do about 8 to 12 old surfaces' worth of blur work, inside the old budget. What
 * does not shrink with the radius is the per-surface cost: each frosted element is its own
 * composited layer with a copy of what lies behind it. That is why the cap doubles instead of
 * growing sixfold. scripts/g-audit/_lab.mjs LAB_GLASS_BUDGET mirrors this number.
 */
export const GLASS_BUDGET = 24;

export const LAB_BREAKPOINTS = { phone: 600, tablet: 1100 } as const;

export const DOT = { size: 3, pitch: 8, groupGap: 16, rows: 5 } as const;

export const VIZ = {
  tile: { w: 88, h: 32 },
} as const;

/** Group emphasis opacity on lit dots (02 section 6). */
export const DOT_EMPHASIS = { current: 1, prev: 0.55, context: 0.28 } as const;

/** Narrative window colour step (02 section 7.1): 0..1 rounded to 0.05. */
export function stepWindowT(t: number | undefined | null): number {
  const v = typeof t === 'number' && Number.isFinite(t) ? Math.min(1, Math.max(0, t)) : 0;
  return Math.round(v * 20) / 20;
}
