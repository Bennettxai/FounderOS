// Content Lab v2 checks for g-audit (docs/projects/content-lab/v2/spec/02-components.md 15.3, lane U1).
// Not a route plugin (leading underscore): scripts/g-audit.mjs imports it. Every function whose name ends
// in "InPage" is serialised and run inside the page, so it closes over nothing.
//
//   pixelTargetsInPage  02 15.3 item 1: the generalised heroTargets. Hides each target's own text (or the
//                       whole mark) so a screenshot shows what is behind it, and returns the target boxes.
//   glassScanInPage     items 4 and 6: elements with a backdrop filter in view, and Glass borders/shadows.
//   reducedGlassInPage  item 5: every [data-lab='Glass'] opaque with no backdrop filter.
//   labHeroesInPage     item 3: GHeroCard and LabNarrative in view (max 2).

/**
 * The lab-glass-budget cap: at most this many blurred surfaces in view in the Lab. Mirrors
 * components/lab/v2/shared/tokens.ts GLASS_BUDGET (24 since page glass takes the 3px light frost;
 * the reasoning is there). tests/glass-audit-scripts.test.ts keeps the two equal.
 */
export const LAB_GLASS_BUDGET = 24;

/** The 13 styles of 02 section 14. */
export const LAB_STYLES = [
  ...['ice', 'lime', 'ember', 'mint'].flatMap((accent) => ['dark', 'light'].map((mode) => ({ id: `glados-${accent}-${mode}`, ui: 'glados', accent, mode }))),
  { id: 'classic-console', ui: 'classic', accent: 'ice', mode: 'dark', theme: 'console' },
  { id: 'classic-red', ui: 'classic', accent: 'ice', mode: 'dark', theme: 'red' },
  { id: 'classic-white', ui: 'classic', accent: 'ice', mode: 'light', theme: 'light' },
  { id: 'classic-frost', ui: 'classic', accent: 'ice', mode: 'dark', theme: 'frost' },
  { id: 'classic-grey-frost', ui: 'classic', accent: 'ice', mode: 'dark', theme: 'grey-frost' },
];

/** The six Lab routes, each on the fixture world. */
export const LAB_ROUTES = ['/lab', '/lab/radar', '/lab/breakouts', '/lab/playbook', '/lab/studio', '/lab/ledger', '/lab/creatorverse'].map((r) => `${r}?sample=1`);

/** Default pixel-contrast selectors (plugins may add their own through `export const pixelContrast`). */
export const LAB_PIXEL = { ink: '[data-lab-ink]', mark: '[data-lab-mark]' };

/**
 * arg = { ink, mark, hide }. The same target rules as scripts/lab-frames.mjs (the frames gate), limited to
 * what is in the viewport: disabled, dimmed and scroll-clipped targets are exempt; a mark's colour is its
 * lit dot (DotMatrix), its line or fill stroke (mini viz, ring), a progress fill's background, else its
 * text colour (stroke icons use currentColor); a DotMatrix with nothing lit is decorative. hide true hides
 * the ink with one style element and returns the targets; hide false removes it.
 */
export function pixelTargetsInPage(arg) {
  const STYLE_ID = 'lab-audit-hide';
  if (!arg.hide) {
    // Restore with transitions off for one style recalc: a tab or link with a colour transition would
    // otherwise fade back in from transparent, and the page scan that follows would read the mid-fade ink.
    const still = document.createElement('style');
    still.textContent = '*, *::before, *::after { transition: none !important; }';
    document.head.appendChild(still);
    document.getElementById(STYLE_ID)?.remove();
    void document.body.offsetHeight;
    for (const el of document.querySelectorAll('[data-lab-ink], [data-lab-ink] *')) void getComputedStyle(el).color;
    still.remove();
    return [];
  }
  const cv = document.createElement('canvas'); cv.width = cv.height = 1;
  const c2 = cv.getContext('2d', { willReadFrequently: true });
  const rgba = (col, op = 1) => {
    c2.clearRect(0, 0, 1, 1); c2.fillStyle = 'rgba(0,0,0,0)'; c2.fillStyle = col; c2.fillRect(0, 0, 1, 1);
    const d = c2.getImageData(0, 0, 1, 1).data;
    return `rgba(${d[0]}, ${d[1]}, ${d[2]}, ${((d[3] / 255) * op).toFixed(3)})`;
  };
  const vw = window.innerWidth, vh = window.innerHeight;
  const clipped = (el) => { const sc = el.closest('.lab2-row-scroll, .lab2-tabs, .lab2g-tablewrap'); if (!sc) return false; const a = el.getBoundingClientRect(), b = sc.getBoundingClientRect(); return a.left < b.left - 1 || a.right > b.right + 1; };
  // Not rendered: the body of a closed <details> (content-visibility: hidden) still reports a box, and
  // sampling it read whatever was painted there instead (/lab/radar's watching rows, "4h" 3.48:1 over the
  // aurora gap). Such text is judged when a step opens it.
  const unrendered = (el) => typeof el.checkVisibility === 'function' && !el.checkVisibility();
  const exempt = (el) => !!el.closest("[aria-disabled='true'], :disabled, [data-dim], [data-lab-skip-contrast], .lab2-sheet-ghost, [inert]") || clipped(el) || unrendered(el);
  const sheet = document.querySelector('.lab2-sheet-layer');
  const underSheet = (el) => !!sheet && !sheet.contains(el);
  const inView = (r) => r.width >= 2 && r.height >= 2 && r.bottom > 0 && r.right > 0 && r.top < vh && r.left < vw;
  // Under something else (the shell's top bar, a menu): what is on top at the target's centre is neither
  // the target, inside it, nor one of its ancestors. Hit testing ignores pointer-events: none layers.
  // Tested at the top, centre and bottom of the box, at its left edge, centre and right edge: text half
  // under the shell's fixed top bar, or with a corner under the fixed Wizard launcher (/lab "Make this" at
  // 1024, scroll 0), is judged on the scroll step that shows it clear, as text cut by the viewport edge is.
  const covered = (el, r) => {
    const xs = [r.left + 2, r.left + r.width / 2, r.right - 2].map((x) => Math.min(vw - 1, Math.max(0, x)));
    return xs.some((x) => [r.top + 1, r.top + r.height / 2, r.bottom - 1].some((py) => {
      const top = document.elementFromPoint(x, Math.min(vh - 1, Math.max(0, py)));
      return !!top && !el.contains(top) && !top.contains(el);
    }));
  };
  const clip = (r) => { const x = Math.max(0, r.left), y = Math.max(0, r.top); return { x, y, w: Math.min(vw, r.right) - x, h: Math.min(vh, r.bottom) - y }; };
  const out = [];
  for (const el of document.querySelectorAll(arg.ink)) {
    if (exempt(el) || underSheet(el)) continue;
    const text = (el.textContent || el.getAttribute('placeholder') || '').trim();
    if (!text) continue;
    const r = el.getBoundingClientRect();
    // text cut by the viewport edge is judged on the scroll step that shows it whole: a sliver of glyphs
    // against whatever sits beside it is not a reading the eye makes
    if (!inView(r) || r.top < 0 || r.bottom > vh || covered(el, r)) continue;
    const cs = getComputedStyle(el);
    if (cs.visibility === 'hidden' || +cs.opacity === 0) continue;
    const size = parseFloat(cs.fontSize), weight = parseInt(cs.fontWeight, 10) || 400;
    out.push({ kind: 'text', text: text.slice(0, 40), color: rgba(cs.color), need: size >= 24 || (size >= 18.66 && weight >= 700) ? 3 : 4.5, rect: clip(r) });
  }
  for (const el of document.querySelectorAll(arg.mark)) {
    if (exempt(el) || underSheet(el)) continue;
    const r = el.getBoundingClientRect();
    if (!inView(r) || covered(el, r)) continue;
    const lit = el.querySelector("[data-emphasis='current'] [data-on='1']") || el.querySelector("[data-on='1']") || (el.getAttribute('data-on') === '1' ? el : null);
    const stroke = el.matches('.lab2-viz-line, .lab2-viz-fill') ? el : el.querySelector('.lab2-viz-line, .lab2-viz-fill, .lab2-ring-fill');
    if (el.matches("[data-lab='DotMatrix']") && !lit) continue;
    let color;
    if (lit) { const op = parseFloat(lit.getAttribute('fill-opacity') ?? '1'); color = rgba(getComputedStyle(lit).fill, op); }
    else if (el.classList.contains('lab2-prow-fill')) color = rgba(getComputedStyle(el).backgroundColor);
    else if (stroke) color = rgba(getComputedStyle(stroke).stroke);
    else if (!(el instanceof SVGElement) && !el.querySelector('svg') && getComputedStyle(el).backgroundColor !== 'rgba(0, 0, 0, 0)') color = rgba(getComputedStyle(el).backgroundColor);
    else if (el instanceof SVGElement && el.tagName !== 'svg') { const cs = getComputedStyle(el); color = rgba(cs.stroke && cs.stroke !== 'none' ? cs.stroke : cs.fill); }
    else color = rgba(getComputedStyle(el).color);
    const name = el.getAttribute('data-lab') || el.getAttribute('aria-label') || (el.className && el.className.baseVal !== undefined ? el.className.baseVal : el.className) || 'mark';
    out.push({ kind: 'mark', text: String(name).slice(0, 40), color, need: 3, rect: clip(r) });
  }
  const st = document.createElement('style');
  st.id = STYLE_ID;
  st.textContent = "[data-lab-ink], [data-lab-ink] *, [data-lab-ink]::placeholder { color: transparent !important; -webkit-text-fill-color: transparent !important; text-shadow: none !important; transition: none !important; } svg[data-lab-mark], [data-lab-mark] svg, svg:has([data-lab-mark]), [data-lab-mark] .lab2-prow-fill, span[data-lab-mark]:not(:has(svg)), [data-lab-overlay] { visibility: hidden !important; } .lab2-prow-fill[data-lab-mark], .lab2-prow-track { background: transparent !important; }";
  document.head.appendChild(st);
  return out;
}

/** Items 4 and 6: blur surfaces in view; under GladOS, Glass with a border. Returns { blurred, borders, shadows }. */
export function glassScanInPage() {
  const vh = window.innerHeight, vw = window.innerWidth;
  const inView = (el) => { const r = el.getBoundingClientRect(); return r.width > 1 && r.height > 1 && r.bottom > 0 && r.top < vh && r.right > 0 && r.left < vw; };
  // The budget is the Lab's (02 4.1): surfaces inside the Lab root; the shell's own bar is not counted.
  let blurred = 0;
  for (const el of document.querySelectorAll('.lab[data-lab-root] *')) {
    const cs = getComputedStyle(el);
    const bf = cs.backdropFilter || cs.webkitBackdropFilter;
    if (bf && bf !== 'none' && inView(el) && cs.visibility !== 'hidden' && cs.display !== 'none') blurred++;
  }
  const glados = document.documentElement.getAttribute('data-ui') === 'glados';
  const borders = [], shadows = [];
  if (glados) {
    for (const el of document.querySelectorAll("[data-lab='Glass']")) {
      if (!inView(el)) continue;
      const cs = getComputedStyle(el);
      const bw = ['Top', 'Right', 'Bottom', 'Left'].map((s) => (cs[`border${s}Style`] === 'none' ? 0 : parseFloat(cs[`border${s}Width`]) || 0));
      if (bw.some((w) => w > 0)) borders.push({ el: (el.className || '').toString().slice(0, 60), border: bw.join(' ') });
      if (cs.boxShadow && cs.boxShadow !== 'none') shadows.push({ el: (el.className || '').toString().slice(0, 60), boxShadow: cs.boxShadow });
    }
  }
  return { blurred, borders, shadows };
}

/** Item 5: with reduced transparency emulated, every Glass is opaque with no backdrop filter. */
export function reducedGlassInPage() {
  const cv = document.createElement('canvas'); cv.width = cv.height = 1;
  const c2 = cv.getContext('2d', { willReadFrequently: true });
  const alpha = (col) => { c2.clearRect(0, 0, 1, 1); c2.fillStyle = 'rgba(0,0,0,0)'; c2.fillStyle = col; c2.fillRect(0, 0, 1, 1); return c2.getImageData(0, 0, 1, 1).data[3] / 255; };
  const emulated = matchMedia('(prefers-reduced-transparency: reduce)').matches;
  const fails = [];
  for (const el of document.querySelectorAll("[data-lab='Glass']")) {
    const cs = getComputedStyle(el);
    const bf = cs.backdropFilter || cs.webkitBackdropFilter;
    const a = alpha(cs.backgroundColor);
    if ((bf && bf !== 'none') || a < 0.98) fails.push({ el: (el.className || '').toString().slice(0, 60), backdrop: bf, alpha: +a.toFixed(3) });
  }
  return { emulated, checked: document.querySelectorAll("[data-lab='Glass']").length, fails };
}

/** Item 3: kit heroes and Lab narrative cards in view. */
export function labHeroesInPage() {
  const vh = window.innerHeight;
  let n = 0;
  for (const el of document.querySelectorAll("[data-g='GHeroCard'], [data-g='LabNarrative']")) {
    const r = el.getBoundingClientRect();
    if (r.width > 1 && r.height > 1 && r.bottom > 0 && r.top < vh) n++;
  }
  return n;
}

/**
 * Sample the screenshot behind each target on a 6 x 3 grid and return the worst ratio per target.
 * `img` is a decoded PNG ({ width, height, data }), `scale` the device pixel ratio.
 */
export function sampleTargets(img, targets, { parseColor, composite, contrastRatio }, scale = 1) {
  const rows = [];
  for (const t of targets) {
    const ink = parseColor(t.color); if (!ink) continue;
    let worst = Infinity;
    for (let gy = 0; gy < 3; gy++) for (let gx = 0; gx < 6; gx++) {
      const x = Math.round((t.rect.x + 1 + ((t.rect.w - 2) * (gx + 0.5)) / 6) * scale), y = Math.round((t.rect.y + 1 + ((t.rect.h - 2) * (gy + 0.5)) / 3) * scale);
      if (x < 0 || y < 0 || x >= img.width || y >= img.height) continue;
      const o = (y * img.width + x) * 4; const bg = [img.data[o], img.data[o + 1], img.data[o + 2]];
      worst = Math.min(worst, contrastRatio(composite(ink, bg), bg));
    }
    if (worst === Infinity) continue;
    rows.push({ kind: t.kind, text: t.text, ratio: +worst.toFixed(2), need: t.need });
  }
  return rows;
}
