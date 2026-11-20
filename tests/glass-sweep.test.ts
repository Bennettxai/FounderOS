/**
 * Glass pack, phase 2b: the sweep after the recipe (FounderOS Glass brief §3.2 lint rule, §4, §5B).
 *
 * - Lint rule: a glass face never sits inside an element with filter, opacity < 1, mask or
 *   backdrop-filter. Bars frost on a layer behind their content; the segmented control's edge fade
 *   is a layer over the options, not a mask around them; disabled faces dim their text, not the face.
 * - Tooltips are solid raised faces that rise 6px on enter, never fade.
 * - No face enters with an opacity fade: every keyframe that fades from transparent is classified,
 *   and the face ones move only transform and the light.
 */
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';
import { CSS_NO_COMMENTS, GLOBALS_CSS, splitSelectors, topLevelRules, type Rule } from './helpers/theme-css';

const read = (p: string) => readFileSync(join(process.cwd(), p), 'utf8');
const norm = (s: string) => s.replace(/\s+/g, ' ').trim();

/** Every declaration of a rule body (custom properties included), last one wins. */
function props(body: string): Map<string, string> {
  const out = new Map<string, string>();
  let depth = 0;
  let cur = '';
  for (const c of body + ';') {
    if (c === '(') depth++;
    if (c === ')') depth--;
    if (c === ';' && depth === 0) {
      const i = cur.indexOf(':');
      if (i > 0) out.set(cur.slice(0, i).trim(), norm(cur.slice(i + 1)));
      cur = '';
    } else cur += c;
  }
  return out;
}

/** Split a value on top-level commas (box-shadow layers). */
function layers(v: string): string[] {
  return splitSelectors(v).map(norm);
}

/** Every rule, top level and inside @media/@supports, with the at-rule heads it sits in. */
function allRules(src = CSS_NO_COMMENTS): (Rule & { at: string[] })[] {
  const out: (Rule & { at: string[] })[] = [];
  const walk = (s: string, at: string[]) => {
    let depth = 0;
    let start = 0;
    let open = -1;
    for (let i = 0; i < s.length; i++) {
      const c = s[i];
      if (c === '{') {
        if (depth === 0) open = i;
        depth++;
      } else if (c === '}') {
        depth--;
        if (depth === 0) {
          const head = norm(s.slice(start, open));
          const body = s.slice(open + 1, i);
          if (/^@(media|supports|container)/.test(head)) walk(body, [...at, head]);
          else if (!head.startsWith('@')) out.push({ selectors: splitSelectors(head), body, at });
          start = i + 1;
        }
      } else if (c === ';' && depth === 0) start = i + 1;
    }
  };
  walk(src, []);
  return out;
}

/** Bodies of every @keyframes block by name (the last one wins in the cascade). */
function keyframeBlocks(src = CSS_NO_COMMENTS): Map<string, string> {
  const out = new Map<string, string>();
  const re = /@keyframes\s+([\w-]+)\s*\{/g;
  let m: RegExpExecArray | null;
  while ((m = re.exec(src))) {
    let depth = 1;
    let i = re.lastIndex;
    while (depth && i < src.length) {
      if (src[i] === '{') depth++;
      else if (src[i] === '}') depth--;
      i++;
    }
    out.set(m[1], norm(src.slice(re.lastIndex, i - 1)));
  }
  return out;
}

/** The subject compound of a selector: its last compound, with :is()/:where()/:has()/:not() arguments dropped. */
function subject(sel: string): string {
  let flat = '';
  let depth = 0;
  for (const c of sel) {
    if (c === '(') {
      if (depth === 0) flat += '(';
      depth++;
    } else if (c === ')') {
      depth--;
      if (depth === 0) flat += ')';
    } else if (depth === 0) flat += c;
  }
  return flat.trim().split(/\s*[\s>+~]\s*/).pop()!;
}
const hasClass = (compound: string, cls: string) => new RegExp(`\\.${cls.replace(/[-]/g, '\\-')}(?![\\w-])`).test(compound);

const RULES = allRules();
const TOP = topLevelRules();
const KF = keyframeBlocks();

/** Rules whose subject carries one of `classes` (no pseudo-element), optionally filtered. */
function rulesOn(classes: string[], pred: (compound: string) => boolean = () => true) {
  return RULES.filter((r) =>
    r.selectors.some((s) => {
      const c = subject(s);
      return !c.includes('::') && classes.some((k) => hasClass(c, k)) && pred(c);
    }),
  );
}

const PHASE_START = '/* ── glass phase 2b ── */';
const PHASE_END = '/* ── end glass phase 2b ── */';

describe('the phase 2b block', () => {
  test('sits once, delimited, at the very end of globals.css, with no literal colour (the fo/chats tail check)', () => {
    expect(GLOBALS_CSS.split(PHASE_START).length).toBe(2);
    const end = GLOBALS_CSS.indexOf(PHASE_END);
    expect(end).toBeGreaterThan(GLOBALS_CSS.indexOf(PHASE_START));
    expect(GLOBALS_CSS.slice(end + PHASE_END.length).trim()).toBe('');
    const block = GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(PHASE_START), end);
    expect(block).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(block).not.toMatch(/rgba?\(\s*\d/);
  });
});

describe('lint rule: bars frost on a layer behind their content (brief §3.2)', () => {
  const BARS = ['os-topbar', 'g-topbar', 'os-sidebar'];

  test('no bar is itself a backdrop-filter ancestor of its buttons or the active pill', () => {
    const offenders = rulesOn(BARS).filter((r) => {
      const p = props(r.body);
      return [p.get('backdrop-filter'), p.get('-webkit-backdrop-filter')].some((v) => v && v !== 'none');
    });
    expect(offenders.map((r) => r.selectors.join(', '))).toEqual([]);
  });

  test('the frost lives on a ::before layer under the content', () => {
    const layer = RULES.filter((r) => r.at.length === 0 && r.selectors.some((s) => /\.os-topbar::before$/.test(s)) && r.selectors.some((s) => /\.os-sidebar::before$/.test(s)));
    expect(layer.length).toBe(1);
    const p = props(layer[0].body);
    expect(p.get('content')).toBe("''");
    expect(p.get('position')).toBe('absolute');
    expect(p.get('inset')).toBe('0');
    expect(p.get('z-index')).toBe('-1');
    expect(p.get('pointer-events')).toBe('none');
    expect(p.get('background')).toBe('var(--gl-bar-bg, transparent)');
    expect(p.get('backdrop-filter')).toBe('var(--gl-bar-frost, none)');
    expect(p.get('-webkit-backdrop-filter')).toBe('var(--gl-bar-frost, none)');
    // the bar isolates, so the layer stays inside it, above the page and below the bar's content
    const iso = RULES.filter((r) => r.at.length === 0 && r.selectors.includes('.os-topbar') && r.selectors.includes('.os-sidebar'));
    expect(iso.some((r) => props(r.body).get('isolation') === 'isolate')).toBe(true);
  });

  test('each style keeps its own bar frost, now as layer tokens', () => {
    const v = (sel: string, name: string) =>
      TOP.filter((r) => r.selectors.includes(sel)).map((r) => props(r.body).get(name)).filter(Boolean).pop();
    expect(v('.os-topbar', '--gl-bar-frost')).toBe('blur(8px)'); // Classic: what Tailwind backdrop-blur painted
    expect(v(":root[data-ui='glados'] .g-topbar", '--gl-bar-bg')).toBe('color-mix(in srgb, var(--g-bg) 72%, transparent)');
    expect(v(":root[data-ui='glados'] .g-topbar", '--gl-bar-frost')).toBe('blur(16px)');
    expect(v(":root[data-ui='legacy'] .os-topbar", '--gl-bar-bg')).toBe('color-mix(in oklab, var(--l-bg-2) 70%, transparent)');
    expect(v(":root[data-ui='legacy'] .os-topbar", '--gl-bar-frost')).toBe('blur(8px)');
    expect(v('html:has(.lab[data-lab-root]) .os-sidebar', '--gl-bar-bg')).toBe('color-mix(in srgb, var(--g-bg, var(--bg)) 42%, transparent)');
    expect(v('html:has(.lab[data-lab-root]) .os-sidebar', '--gl-bar-frost')).toBe('blur(20px) saturate(1.2)');
    expect(v('html:has(.lab[data-lab-root]) .os-topbar', '--gl-bar-bg')).toBe('color-mix(in srgb, var(--g-bg, var(--bg)) 32%, transparent)');
    expect(v('html:has(.lab[data-lab-root]) .os-topbar', '--gl-bar-frost')).toBe('blur(20px) saturate(1.2)');
  });

  test('the Lab bars clear their own fill so the page reads through the frost layer', () => {
    for (const sel of ['html:has(.lab[data-lab-root]) .os-sidebar', 'html:has(.lab[data-lab-root]) .os-topbar']) {
      const bg = TOP.filter((r) => r.selectors.includes(sel)).map((r) => props(r.body).get('background')).filter(Boolean).pop();
      expect(bg, sel).toBe('transparent !important');
    }
  });

  test('the Classic top bar carries no blur utility of its own', () => {
    const cls = /className="(os-topbar [^"]*)"/.exec(read('components/Topbar.tsx'))![1];
    expect(cls).not.toMatch(/backdrop-blur|bg-os-bg2\//);
  });
});

describe('lint rule: the segmented control masks nothing around its active option', () => {
  test('no .g-seg rule masks the track', () => {
    const offenders = rulesOn(['g-seg']).filter((r) => /(^|;|\s)(-webkit-)?mask(-image)?\s*:/.test(r.body));
    expect(offenders.map((r) => r.selectors.join(', '))).toEqual([]);
  });

  test('the edge fade is a sticky layer over the options, painted in the track colour', () => {
    const pick = (re: RegExp) => RULES.filter((r) => r.selectors.some((s) => re.test(s)));
    const both = pick(/^\.g-seg\[data-fade-start\]::before$/).filter((r) => r.selectors.some((s) => /^\.g-seg\[data-fade-end\]::after$/.test(s)));
    expect(both.length).toBe(1);
    const p = props(both[0].body);
    expect(p.get('content')).toBe("''");
    expect(p.get('position')).toBe('sticky');
    expect(p.get('pointer-events')).toBe('none');
    expect(p.get('width')).toBe('20px');
    const start = props(pick(/^\.g-seg\[data-fade-start\]::before$/).filter((r) => r.selectors.length === 1).pop()!.body);
    const end = props(pick(/^\.g-seg\[data-fade-end\]::after$/).filter((r) => r.selectors.length === 1).pop()!.body);
    expect(start.get('left')).toBe('0');
    expect(start.get('background')).toBe('linear-gradient(to right, var(--gl-seg-ground), transparent)');
    expect(end.get('right')).toBe('0');
    expect(end.get('background')).toBe('linear-gradient(to left, var(--gl-seg-ground), transparent)');
    // the layer takes no room: its width is given back (20px + the 2px gap)
    expect(start.get('margin-right')).toBe('-22px');
    expect(end.get('margin-left')).toBe('-22px');
  });
});

describe('lint rule: disabled faces dim their text and icons, not the face', () => {
  const DISABLED_FACES = ['g-btn', 'g-chip', 'g-seg-opt', 'l-pressable'];

  test('no disabled face drops its own opacity', () => {
    const offenders = rulesOn(DISABLED_FACES, (c) => /:disabled/.test(c)).filter((r) => {
      const o = props(r.body).get('opacity');
      return o !== undefined && Number(o) < 1;
    });
    expect(offenders.map((r) => r.selectors.join(', '))).toEqual([]);
  });

  test('the text fades toward transparent and only the icon leaves take opacity', () => {
    const text = RULES.filter((r) => r.selectors.some((s) => /^:is\(\.g-btn, \.g-chip, \.g-seg-opt, \.l-pressable\):disabled/.test(s) && !/svg|img/.test(s)));
    expect(text.length).toBe(1);
    expect(props(text[0].body).get('-webkit-text-fill-color')).toBe('color-mix(in srgb, currentColor 45%, transparent)');
    const icons = RULES.filter((r) => r.selectors.some((s) => /^:is\(\.g-btn, \.g-chip, \.g-seg-opt, \.l-pressable\):disabled.* :is\(svg, img\)$/.test(s)));
    expect(icons.length).toBe(1);
    expect(props(icons[0].body).get('opacity')).toBe('0.45');
  });
});

describe('tooltips are solid raised faces that rise 6px, never fade (brief §5 B13)', () => {
  const tip = props(TOP.filter((r) => r.selectors.includes('.g-tip')).pop()!.body);
  const shown = props(TOP.filter((r) => r.selectors.includes('.g-tip[data-show]')).pop()!.body);

  test('hidden: invisible and 6px low, no opacity', () => {
    expect(tip.has('opacity')).toBe(false);
    expect(tip.get('visibility')).toBe('hidden');
    expect(tip.get('transform')).toBe('translate(-50%, 6px)');
    expect(tip.get('transition') ?? '').not.toMatch(/opacity/);
  });

  test('shown: visible, risen into place on a transform transition', () => {
    expect(shown.has('opacity')).toBe(false);
    expect(shown.get('visibility')).toBe('visible');
    expect(shown.get('transform')).toBe('translate(-50%, 0)');
    expect(shown.get('transition')).toMatch(/^transform var\(--g-dur-colour\) var\(--ease\)$/);
  });
});
