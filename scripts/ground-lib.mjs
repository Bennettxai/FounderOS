// Pure parts of scripts/ground-check.mjs (no Chrome here, so tests import them).
//
// coversPoint is serialised into the page (its .toString() runs inside PROBE), so it closes over
// nothing: the caller passes the computed style and whether the element is a glass face.

/**
 * The glass pack's face classes (app/globals.css, "The face." rule; tests/glass-audit-scripts.test.ts
 * keeps this list equal to tests/helpers/glass-lint.ts FACE_CLASSES). A face is transparent by
 * design: its frost and its light (groove, drop, rim, pool) are what covers the ground.
 */
export const GLASS_FACE_SELECTOR = [
  '.glass-face', '.glass-face--strong', '.l-card', '.l-chip', '.l-dialog', '.g-card', '.g-card-raised', '.g-well',
  '.g-chip', '.g-sheet', '.g-drawer', '.g-dialog', '.g-palette', '.ui-toast', '.c-pop', '.c-chip', '.h6-card', '.glass-panel',
  '.glass-panel-violet', '.glass-chip', '.lab2-glass', '.wiz-ask', '.g-ask-box', '.bh-ask', '.ap-panel', '.ap-chip',
].join(', ');

/**
 * Does this element cover the sampled point (so a pixel that is not the ground colour is content,
 * not a ground that stops short)? Opaque backgrounds, background images and media cover it, and so
 * does a glass face or anything else that frosts (backdrop-filter): transparent, but not ground.
 * @param {{ backgroundColor: string, backgroundImage?: string, backdropFilter?: string, tagName: string, isFace?: boolean }} s
 */
export function coversPoint({ backgroundColor, backgroundImage, backdropFilter, tagName, isFace }) {
  const bg = String(backgroundColor || '').trim();
  const clear = bg === '' || bg === 'transparent' || bg === 'rgba(0, 0, 0, 0)' || /,\s*0\)$/.test(bg);
  if (!clear) return true;
  if (backgroundImage && backgroundImage !== 'none') return true;
  if (['IMG', 'VIDEO', 'CANVAS', 'svg'].includes(tagName)) return true;
  if (isFace) return true;
  return Boolean(backdropFilter && backdropFilter !== 'none');
}
