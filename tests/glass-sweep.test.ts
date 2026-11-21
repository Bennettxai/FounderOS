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
import { inlineAnimations, localStyleSources, tsxFiles } from './helpers/glass-lint';

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
  test('sits once, delimited, at the end of globals.css (only the later glass phase blocks follow), with no literal colour (the fo/chats tail check)', () => {
    expect(GLOBALS_CSS.split(PHASE_START).length).toBe(2);
    const end = GLOBALS_CSS.indexOf(PHASE_END);
    expect(end).toBeGreaterThan(GLOBALS_CSS.indexOf(PHASE_START));
    const tail = GLOBALS_CSS.slice(end + PHASE_END.length).trim();
    expect(tail.replace(/^\/\* ── glass phase 3 ── \*\/[\s\S]*\/\* ── end glass phase 3 ── \*\/$/, '')).toBe('');
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
    // Classic: what `bg-os-bg2/70 backdrop-blur` painted, now on the layer (review finding 2: the fill came back)
    expect(v('.os-topbar', '--gl-bar-bg')).toBe('color-mix(in srgb, var(--bg-2) 70%, transparent)');
    expect(v('.os-topbar', '--gl-bar-frost')).toBe('blur(8px)');
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

  test('the Classic top bar carries no blur or fill utility of its own: both sit on its layer', () => {
    const cls = /className="(os-topbar [^"]*)"/.exec(read('components/Topbar.tsx'))![1];
    expect(cls).not.toMatch(/backdrop-blur|bg-os-bg2\//);
  });

  test('reduced transparency: the bar layers drop the blur and take the solid face colour', () => {
    const rt = RULES.filter(
      (r) => r.at.some((a) => /prefers-reduced-transparency:\s*reduce/.test(a)) && r.selectors.some((s) => /\.os-topbar::before$/.test(s)) && r.selectors.some((s) => /\.os-sidebar::before$/.test(s)),
    );
    expect(rt.length).toBe(1);
    const p = props(rt[0].body);
    expect(p.get('backdrop-filter')).toBe('none');
    expect(p.get('-webkit-backdrop-filter')).toBe('none');
    expect(p.get('background')).toBe('var(--solid-bg)');
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

describe('no face enters with an opacity fade (brief §4): every fading keyframe is classified', () => {
  /** Face entrances (card, panel, sheet, toast, popover, tooltip, chip): transform and the light only. */
  const FACE = {
    'panel-in': 'Optimal Engine doctor/search pop-out panel (.c-pop), materialises',
    'fin-tip-in': "Finances bar value tip, a solid raised tooltip: rises 6px",
    'wizard-row-in': 'Wizard project cards and conversation rows',
    'bh-pop': 'Blueprint canvas node (holds .bh-card), positioned by --tx/--ty',
    'bh-enter': 'Blueprint canvas node on first layout',
    'ae-sheet-in': 'Creative Engine sheet, dialog and drawer, materialise (drawer: from the right)',
    'ae-tl-tip-in': 'Creative Engine timeline tooltip: rises 6px',
    'ae-bstat-in': 'Creative Engine status picker opening (pill track)',
    'ae-o-in': 'Octane window, palette, workflow sheet, menu, coach mark, gate dialog',
    'ae-o-rise': 'Octane window sections, HUD, canvas nodes, start tiles, popover',
    'ae-o-sheet': 'Octane bottom sheet',
    'ae-o-toast': 'Octane toast: rises, holds, melts',
    'lab-c-sheet-in': 'Lab Classic side sheet, slides in as a drawer',
    'lab2c-seal': 'Lab script tag (chip) stamping on seal',
    'trd-g-arrive': 'Trading hero card (GHeroCard), materialises',
    'ch-slide': 'Chats thread pane sliding over the list on narrow screens',
  };
  /** Not faces: scrims and grounds (siblings of the faces they dim), text, dots, lines, media leaves,
   *  skeletons, progress, and looping status lights. These may fade. */
  const NOT_FACE = {
    'overlay-in': 'Optimal Engine pop-out scrim, a sibling of the panel',
    'wiz-dot-in': 'Wizard launcher status dot',
    'os-label-in': 'sidebar labels (text)',
    'bh-draw': 'Blueprint edge lines (SVG)',
    'bh-rise': 'Blueprint inspector text sections and the ask answer text',
    'ae-toast': 'Creative Engine notice caption (a <p>, text only)',
    'ae-arrive': 'UGC video (a media leaf)',
    'ae-phase': 'phase label text',
    'ae-o-arrive': 'Octane media leaves, stage note text, preview row',
    'ae-o-rowin': 'Octane tree note rows',
    'db-zoom-in': 'Database treemap view (data viz)',
    'lab2-fade': 'Lab aurora ground and the hero number',
    'route-skel-in': 'route skeleton',
    'nav-progress-in': 'nav progress bar',
    'l-fade': 'Legacy chart dots and peak labels',
    'ch-arrive': 'new chat messages (text rows)',
    // loops and decoration, not entrances
    'led-blink': 'status LED', blink: 'caret', 'funnel-halo': 'funnel ring', 'funnel-hub': 'funnel hub glow',
    'conductor-halo': 'launcher ring', 'ae-o-pulse': 'busy dot', 'g-pulse': 'busy dot', 'g-breathe': 'progress breathe',
    'g-viz-breathe': 'viz skeleton', 'lab-pulse': 'Lab ring', 'lab-c-pulse': 'Lab ring', 'route-skel-pulse': 'skeleton',
    'l-led-blink': 'status LED', 'l-halo': 'launcher ring', 'h6-caret-blink': 'caret', 'gl-reflect': 'brief §2 reflection',
  };

  test.each(Object.keys(FACE))('%s (a face entrance) moves only transform and the light', (name) => {
    const k = KF.get(name);
    expect(k, name).toBeDefined();
    expect(k, name).not.toMatch(/opacity/);
  });

  test('every keyframe that touches opacity is classified as not a face', () => {
    const fading = [...KF].filter(([, body]) => /opacity/.test(body)).map(([name]) => name);
    const unclassified = fading.filter((n) => !(n in NOT_FACE));
    expect(unclassified).toEqual([]);
  });

  test('the Octane grounds and veils fade their fill, not their opacity (they hold faces)', () => {
    expect(KF.get('ae-o-fade')).toBe('from { background-color: transparent; }');
  });

  const anim = (cls: string) =>
    RULES.filter((r) => r.at.length === 0 && r.selectors.some((s) => /\[data-ground\]/.test(s) && subject(s) === cls))
      .map((r) => props(r.body).get('animation') ?? '')
      .join(' | ');

  test('faces that used to fade now materialise (drawers slide from the right)', () => {
    for (const cls of ['.panel-in', '.ae-sheet', '.trd-g-hero', '.ae-o-wf-palette', '.ae-o-wf-banner']) {
      expect(anim(cls), cls).toMatch(/^gl-materialise var\(--gl-dur-materialise\) var\(--gl-spring\)/);
    }
    expect(RULES.some((r) => r.selectors.some((s) => /\[data-ground\] \.ae-o-gate > div$/.test(s)) && /gl-materialise var/.test(props(r.body).get('animation') ?? ''))).toBe(true);
    for (const cls of ['.ae-sheet.ae-drawer', '.ae-o-wf-sheet', '.lab-c-sheet']) {
      expect(anim(cls), cls).toMatch(/^gl-materialise-drawer var\(--gl-dur-drawer\) var\(--gl-spring-drawer\)/);
    }
    // the pop-out panel keeps its 50ms beat after the scrim (the materialise list reads --gl-delay)
    const panel = RULES.filter((r) => r.at.length === 0 && r.selectors.some((s) => /\[data-ground\]/.test(s) && subject(s) === '.panel-in'));
    expect(panel.map((r) => props(r.body).get('--gl-delay')).filter(Boolean).pop()).toBe('50ms');
    expect(anim('.panel-in')).toMatch(/var\(--gl-delay, 0ms\) backwards/);
  });

  test('the centred Octane hint rises 6px without leaving its centre', () => {
    expect(KF.get('gl-rise-centred')).toBe('from { transform: translateX(-50%) translateY(6px); } to { transform: translateX(-50%); }');
    expect(anim('.ae-o-wf-hint')).toMatch(/^gl-rise-centred /);
  });

  test('tooltips rise 6px from their anchor, never fade', () => {
    expect(KF.get('fin-tip-in')).toBe('from { transform: translateX(-50%) translateY(6px); } to { transform: translateX(-50%); }');
    expect(KF.get('ae-tl-tip-in')).toBe('from { transform: translateX(-50%) translateY(6px); } to { transform: translateX(-50%); }');
  });

  test('reduced motion: the faces that now materialise appear at rest', () => {
    const rm = RULES.filter((r) => r.at.some((a) => /prefers-reduced-motion: reduce/.test(a)) && r.selectors.some((s) => /\[data-ground\]/.test(s)));
    const covered = rm.filter((r) => props(r.body).get('animation') === 'none').flatMap((r) => r.selectors.map(subject));
    for (const cls of ['.panel-in', '.ae-sheet', '.trd-g-hero', '.ae-o-wf-palette', '.ae-o-wf-banner', '.ae-o-wf-sheet', '.lab-c-sheet', '.ae-o-wf-hint', 'div']) {
      expect(covered, cls).toContain(cls);
    }
  });

  test('the pop-out scrim is a sibling of its panel, so the panel frosts the page, not the scrim', () => {
    const core = read('components/BrainCore.tsx');
    const scrim = /<div[^>]*className="overlay-in[^"]*"[^>]*\/>/.exec(core);
    expect(scrim, 'self-closing scrim').not.toBeNull();
    expect(scrim![0]).toMatch(/backdrop-blur/);
    // the panel's own container carries no blur, fade or fill
    const holder = /<div\s+className="([^"]*)"(?:[^>]|=>)*>\s*<div\s+className="panel-in/.exec(core);
    expect(holder, 'panel holder').not.toBeNull();
    expect(holder![1]).not.toMatch(/backdrop|overlay-in|opacity|bg-/);
  });

  test('the Finances bar tip is a solid raised face', () => {
    expect(read('components/BusinessIncomeChart.tsx')).toMatch(/className="fin-tip c-pop glass-solid /);
  });
});

describe('Classic ad-hoc cards are glass faces', () => {
  /** The recipe phase 1 put on .glass-face: the one source of the six layers. */
  const recipe = props(
    TOP.filter((r) => r.selectors.some((s) => /\[data-ground\] :is\(\s*\.glass-face,/.test(s)) && /box-shadow/.test(r.body))[0].body,
  );
  const SIX = layers(recipe.get('box-shadow')!);
  const MAPPED = ':root[data-ground] .bg-os-surface.rounded-lg-t:not(.border-dashed)';
  const at = (sel: string, media?: RegExp) =>
    RULES.filter((r) => r.selectors.includes(sel) && (media ? r.at.some((a) => media.test(a)) : r.at.length === 0));

  test('an os surface with the card radius is a card face: transparent, light frost, the six layers', () => {
    const r = at(MAPPED);
    expect(r.length).toBe(1);
    const p = props(r[0].body);
    expect(p.get('background')).toBe('transparent');
    expect(p.get('border-color')).toBe('transparent');
    expect(p.get('backdrop-filter')).toBe('var(--gl-blur)');
    expect(p.get('-webkit-backdrop-filter')).toBe('var(--gl-blur)');
    expect(layers(p.get('box-shadow')!)).toEqual(SIX);
  });

  test('the mapped cards and the wells carry the knobs at zero specificity', () => {
    const k = TOP.filter((r) => r.selectors.includes(':where(.bg-os-surface.rounded-lg-t, .glass-well)'));
    expect(k.length).toBe(1);
    const p = props(k[0].body);
    expect(p.get('--gl-tint')).toBe('white');
    expect(p.get('--gl-rim-a')).toBe('15%');
    expect(p.get('--gl-low-a')).toBe('30%');
    expect(p.get('--gl-pool-a')).toBe('9%');
  });

  test('a card inside a face is a well: rim light only, no second frost, radius 16', () => {
    const wells = TOP.filter((r) => r.selectors.includes(':root[data-ground] .glass-well'));
    expect(wells.length).toBe(1);
    const nested = wells[0].selectors.find((s) => /\) :not\(\.border-dashed\)\.bg-os-surface\.rounded-lg-t$/.test(s));
    expect(nested, 'nested mapped cards share the well rule').toBeDefined();
    for (const face of ['.glass-face', '.c-pop', '.l-card', '.g-card', '.h6-card', '.glass-panel', '.bg-os-surface.rounded-lg-t']) {
      expect(nested!, face).toContain(face);
    }
    const p = props(wells[0].body);
    expect(p.get('background')).toBe('transparent');
    expect(p.get('border-color')).toBe('transparent');
    expect(p.get('backdrop-filter')).toBe('none');
    expect(p.get('border-radius')).toBe('var(--gl-r-well)');
    expect(layers(p.get('box-shadow')!)).toEqual([SIX[2], SIX[3]]);
  });

  test('no blur where the viewer or the browser asks for none: the mapped card is the solid face', () => {
    for (const media of [/prefers-reduced-transparency: reduce/, /^@supports not/]) {
      const r = at(MAPPED, media);
      expect(r.length, String(media)).toBe(1);
      const p = props(r[0].body);
      expect(p.get('background')).toBe('var(--solid-bg)');
      if (/reduced/.test(String(media))) expect(p.get('backdrop-filter')).toBe('none');
    }
  });

  test('an accent border becomes a tinted rim; a hoverable card brightens its rim', () => {
    const accent = props(at(':root[data-ground] .bg-os-surface.rounded-lg-t.border-os-accent')[0].body);
    expect(accent.get('--gl-tint')).toBe('var(--accent)');
    expect(accent.get('--gl-rim-a')).toBe('45%');
    const hover = TOP.filter((r) => r.selectors.includes(':root[data-ground] :is(.glass-face, .bg-os-surface.rounded-lg-t).hoverable:hover'));
    expect(hover.length).toBe(1);
    expect(props(hover[0].body).get('--gl-rim-a')).toBe('28%');
  });

  /** Lines that are not cards (inputs, buttons, tracks, badges, calendar chips) and keep their surface. */
  const NOT_CARD: [string, string, string][] = [
    ['components/knowledge/KnowledgeVisualizer.tsx', 'flex flex-wrap gap-1 rounded-[11px]', 'section nav track (buttons)'],
    ['components/compute/ComputeConsole.tsx', 'w-full rounded-[10px] border p-3 text-left', 'selectable option button'],
    ['components/KnowledgeDetail.tsx', 'g-btn kg-detail-row', 'row button'],
    ['components/InstagramDmInbox.tsx', 'inline-flex gap-1 rounded-md-t', 'tab track'],
    ['components/tasks/Board.tsx', 'tasks-card', 'calendar event chips inside grid cells'],
    ['components/BrainQuery.tsx', 'border-b-2 bg-os-surface', 'keyboard hint badge'],
    ['app/org/page.tsx', 'mb-4 rounded-lg border bg-os-surface', 'venture banner, lit by its own inline venture colour'],
  ];
  const SKIP_DIRS = /^(founderos)\//;
  const PHASE_2A = new Set(['components/CommandPalette.tsx', 'components/legacy/Dialog.tsx', 'components/g/GSheet.tsx', 'components/g/GDrawer.tsx', 'components/g/GDialog.tsx', 'components/ui/Toast.tsx']);
  const walk = (dir: string): string[] =>
    require('node:fs')
      .readdirSync(join(process.cwd(), dir), { withFileTypes: true })
      .flatMap((d: { name: string; isDirectory: () => boolean }) =>
        d.isDirectory() ? walk(`${dir}/${d.name}`) : /\.tsx?$/.test(d.name) ? [`${dir}/${d.name}`] : [],
      );
  const FILES = [...walk('app'), ...walk('components')].filter((f) => !SKIP_DIRS.test(f) && !PHASE_2A.has(f));
  // a variant-prefixed surface (wide:bg-os-surface) is a card at that breakpoint, so it counts too
  const SURFACE = /(^|[\s'"`:])bg-os-surface(?![\w/-])/;
  const CARD_RADIUS = /\brounded-(?:lg|xl|2xl)(?![\w-])|\brounded-\[(?:1\d|2\d)px\]|\brounded-(?:md|sm)-t\b/;

  test('no os-surface card keeps an ad-hoc radius: cards take the card token, insets are wells, pop-ups are solid', () => {
    const left: string[] = [];
    for (const f of FILES) {
      read(f)
        .split('\n')
        .forEach((line, i) => {
          if (!SURFACE.test(line) || !CARD_RADIUS.test(line) || !/\bborder\b/.test(line)) return;
          if (/outline-none/.test(line)) return; // inputs
          if (/\bglass-(well|solid|face)\b|\bc-pop\b/.test(line)) return;
          if (NOT_CARD.some(([file, needle]) => file === f && line.includes(needle))) return;
          left.push(`${f}:${i + 1} ${line.trim().slice(0, 110)}`);
        });
    }
    expect(left).toEqual([]);
  });

  test('every allowlisted non-card still exists (the list cannot rot)', () => {
    for (const [file, needle] of NOT_CARD) expect(read(file), `${file}: ${needle}`).toContain(needle);
  });

  test('menus and hover tips over the task charts are solid raised faces that rise, never fade', () => {
    for (const f of ['components/tasks/TaskOverview.tsx', 'components/tasks/RangePill.tsx']) {
      const menus = read(f).split('\n').filter((l) => /absolute right-0 z-20 mt-1/.test(l));
      expect(menus.length, f).toBeGreaterThan(0);
      for (const l of menus) expect(l, f).toMatch(/\bglass-solid\b/);
    }
    for (const f of ['components/tasks/TaskOverview.tsx', 'components/tasks/ActivityGraph.tsx']) {
      const tips = read(f).split('\n').filter((l) => /pointer-events-none absolute -top-\d/.test(l));
      expect(tips.length, f).toBeGreaterThan(0);
      for (const l of tips) {
        expect(l, f).toMatch(/\bglass-solid\b/);
        expect(l, f).not.toMatch(/opacity-0|group-hover:opacity/);
        expect(l, f).toMatch(/\binvisible\b.*\btranslate-y-1\.5\b.*\bgroup-hover:visible\b.*\bgroup-hover:translate-y-0\b/);
      }
    }
  });

  test('the Brand Deals cards rise without fading', () => {
    const kf = /@keyframes bd-rise \{([^}]*\}[^}]*)\}/.exec(read('components/brand-deals/DealBoard.tsx'));
    expect(kf).not.toBeNull();
    expect(kf![1]).not.toMatch(/opacity/);
  });
});

describe('component-local keyframes: no face fades in (CSS files and <style> blocks, glass phase 3)', () => {
  const ROOT = process.cwd();
  const SOURCES = localStyleSources(ROOT, ['app', 'components', 'founderos', 'lib']);
  /** name -> [file, body without comments] for every local @keyframes. */
  const LOCAL = new Map<string, { file: string; body: string }[]>();
  for (const { file, css } of SOURCES) {
    const flat = css.replace(/\/\*[\s\S]*?\*\//g, '');
    for (const [name, body] of keyframeBlocks(flat)) LOCAL.set(name, [...(LOCAL.get(name) ?? []), { file, body }]);
  }
  /** Face entrances and exits (panels, sheets, drawers, dialogs, menus, chips, tooltips, slabs, cards, and the
   *  wrappers that hold faces): transform and the light only. */
  const LOCAL_FACE: Record<string, string> = {
    'gbr-view-in': 'G-Brain radial ring view: holds the legend card and the year chips',
    'gbr-view-out': 'G-Brain radial ring view leaving',
    'gbr-panel-in': 'G-Brain expanded view (holds its cards)',
    'gbr-panel-out': 'G-Brain expanded view leaving',
    'skh-in': 'Skill hierarchy root (holds the location panel)',
    'skh-panel-in': 'Skill hierarchy panel',
    'kg-panel-in': 'Knowledge graph detail card',
    'kv-panel-in': 'Knowledge focus panel',
    'kv-in': 'Knowledge canvas root (holds the canvas chips)',
    'pc-rise': 'Pipeline chart active-stage label pill',
    'bd-pop': 'Brand Deals peak label pill',
    'fc-slab-in': 'Funnel slab (holds the cards)',
    'fc-card-in': 'Funnel cards and secondary panels',
    'fc-tip-in': 'Funnel menu, tooltip and peak label (solid raised faces): rise',
    'fc-drawer-in': 'Funnel lead drawer',
    'bn-panel-in': 'Lab (Legacy) sheet',
    'bn-panel-out': 'Lab (Legacy) sheet leaving',
    'lab2-verse-in': 'Lab Creatorverse creator panel',
    'lab2-verse-up': 'Lab Creatorverse creator panel (phone)',
    'lab2-enter': 'Lab page blocks (glass panels and the wrappers of glass panels)',
    'lab2-pop-in': 'Lab menus and run pop-ups (solid raised faces)',
    'lab2-pop-out': 'Lab menus and run pop-ups leaving',
    'lab2-chip-in': 'Lab badges and action chips: pop from .9',
    'lab2-dialog-in': 'Lab dialogs',
    pop: 'Workflow canvas modal, menu and toast',
  };
  /** Not faces: SVG and canvas leaves, dots, carets, scrims (siblings of their faces), text, skeletons, loops. */
  const LOCAL_NOT_FACE: Record<string, string> = {
    'labv-pulse': 'Lab loading skeleton',
    'gbr-grow': 'G-Brain radial spokes (SVG)',
    'gbr-leaf-in': 'G-Brain radial ring (SVG)',
    'gbr-leaf-out': 'G-Brain radial ring (SVG) leaving',
    'kg-fade-in': 'Knowledge graph branches (SVG)',
    'kg-leaf-in': 'Knowledge graph leaves (SVG)',
    'kg-glow-in': 'Knowledge graph glow (SVG)',
    'kg-breathe': 'Knowledge graph memory layer breathing (SVG)',
    'kg-glow-breathe': 'Knowledge graph core glow (SVG)',
    'kg-mem-in': 'Knowledge graph deeper notes (SVG)',
    'kg-exit-fade': 'Knowledge graph tree skeleton leaving (SVG)',
    'kg-web-fade-in': 'Knowledge graph resting web (SVG)',
    'kv-leaf-in': 'Knowledge canvas graph (SVG)',
    'pc-fadein': 'Pipeline chart ramps and bar caps (SVG)',
    'pc-caret': 'Pipeline chart caret',
    'bd-fadein': 'Brand Deals peak dot (SVG) and heat dots',
    'fc-blink': 'Funnel caret',
    'fc-pop': 'Funnel chart dots',
    'fc-scrim-in': 'Funnel drawer scrim, a sibling of the drawer',
    'fc-dot-in': 'Funnel peak dot and the copied note (text)',
    'l-lab-blink': 'Lab (Legacy) caret',
    'bn-overlay-in': 'Lab (Legacy) sheet scrim, a sibling of the sheet',
    'bn-overlay-out': 'Lab (Legacy) sheet scrim leaving',
    'lab2-ripple': 'Lab search focus ring (a pseudo-element)',
    'lab2-hold': 'Lab count-up text',
    'lab2-scrim-in': 'Lab dialog scrim, a sibling of the dialog',
    rise: 'Workflow canvas node body content (fields and text)',
    sweep: 'Workflow canvas panel light band (a pseudo-element)',
  };

  test('the scan reads CSS files and <style> blocks', () => {
    const files = SOURCES.map((s) => s.file);
    for (const f of ['components/funnel/funnel.css', 'components/lab/v2/shared/lab-motion.css', 'components/GBrainRadial.tsx <style>', 'components/KnowledgeGraph.tsx <style>']) {
      expect(files, f).toContain(f);
    }
  });

  test.each(Object.keys(LOCAL_FACE))('%s (a face entrance or exit) moves only transform and the light', (name) => {
    const k = LOCAL.get(name);
    expect(k, name).toBeDefined();
    for (const { file, body } of k!) expect(body, `${file} ${name}`).not.toMatch(/opacity/);
  });

  test('every local keyframe that touches opacity is classified as not a face', () => {
    const fading = [...LOCAL].filter(([, v]) => v.some((x) => /opacity/.test(x.body))).map(([n, v]) => `${n} (${v[0].file})`);
    expect(fading.filter((n) => !(n.split(' ')[0] in LOCAL_NOT_FACE))).toEqual([]);
  });

  test('a face that plays a local entrance does not start transparent (no opacity: 0 waiting for the keyframe)', () => {
    const bad: string[] = [];
    for (const { file, css } of SOURCES) {
      const flat = css.replace(/\/\*[\s\S]*?\*\//g, '');
      for (const r of allRules(flat)) {
        const anim = /(?:^|[;\s])animation(?:-name)?\s*:([^;]+)/.exec(r.body)?.[1] ?? '';
        const names = anim.split(/[\s,]+/).filter((n) => n in LOCAL_FACE || /^gl-materialise/.test(n));
        if (names.length && /(?:^|[;\s])opacity\s*:\s*0\s*(;|$)/.test(r.body)) bad.push(`${file}: ${r.selectors.join(', ')}`);
      }
    }
    expect(bad).toEqual([]);
  });

  test('inline style animations name a classified keyframe (a face never plays a fading one)', () => {
    const fadingLocal = new Set([...LOCAL].filter(([, v]) => v.some((x) => /opacity/.test(x.body))).map(([n]) => n));
    const unclassified: string[] = [];
    for (const abs of tsxFiles(ROOT, ['app', 'components', 'founderos'])) {
      for (const n of inlineAnimations(readFileSync(abs, 'utf8'))) {
        if (fadingLocal.has(n) && !(n in LOCAL_NOT_FACE)) unclassified.push(`${abs.slice(ROOT.length + 1)}: ${n}`);
      }
    }
    expect(unclassified).toEqual([]);
  });

  test('the Pipeline chart label and the Brand Deals peak pill rise or pop, never fade', () => {
    const pc = read('components/PipelineChart.tsx');
    expect(pc).toMatch(/animation: `pc-rise /);
    expect(pc.match(/animation: `pc-fadein [^`]*`/g)?.length).toBe(2); // the SVG ramps and caps only
    const bd = read('components/brand-deals/DealBoard.tsx');
    expect(bd).toMatch(/animation: `bd-pop /);
  });

  test('the workflow canvas modal root frosts on a layer under the modal, not around it', () => {
    const css = read('components/workflow-canvas/canvas.css').replace(/\/\*[\s\S]*?\*\//g, '');
    const root = allRules(css).filter((r) => r.selectors.some((s) => /\.modal-root$/.test(s)));
    for (const r of root) expect(r.body, r.selectors.join()).not.toMatch(/backdrop-filter\s*:\s*(?!none)/);
    expect(allRules(css).some((r) => r.selectors.some((s) => /\.modal-root::before$/.test(s)) && /backdrop-filter/.test(r.body))).toBe(true);
  });

  test('the funnel drawer sits beside its scrim, not in it', () => {
    const src = read('components/funnel/FunnelDashboard.tsx');
    expect(src).toMatch(/<div className="fc-layer">\s*<div className="fc-scrim" aria-hidden \/>\s*<aside className="fc-drawer/);
  });
});

describe('hover moves the glass knobs only (glass phase 3)', () => {
  /** Every face: the pack's classes, the wells and solid faces, the Classic card. */
  const FACES = [
    '.glass-face', '.glass-face--strong', '.l-card', '.l-chip', '.l-dialog', '.g-card', '.g-card-raised', '.g-well', '.g-chip',
    '.g-sheet', '.g-drawer', '.g-dialog', '.g-palette', '.ui-toast', '.c-pop', '.c-chip', '.h6-card', '.glass-panel', '.glass-panel-violet',
    '.glass-chip', '.lab2-glass', '.wiz-ask', '.g-ask-box', '.glass-well', '.glass-solid', '.bg-os-surface.rounded-lg-t',
  ];
  const notList = (sel: string) => {
    const m = /\.hoverable:not\(([^)]*(?:\([^)]*\)[^)]*)*)\):hover$/.exec(sel);
    return m ? splitSelectors(m[1]).map(norm) : [];
  };

  test('the theme hover glows (red, white, frost, grey frost, daylight) never land on a face', () => {
    const glows = RULES.filter((r) => r.selectors.some((s) => /^:root\[data-theme='[\w-]+'\] \.hoverable/.test(s) && /:hover$/.test(s)));
    expect(glows.length).toBeGreaterThanOrEqual(4);
    for (const r of glows) {
      for (const s of r.selectors) {
        const excluded = notList(s);
        expect(FACES.filter((f) => !excluded.includes(f)), s).toEqual([]);
      }
    }
  });

  test('no other :hover rule replaces a face shadow or adds a fill at the face rule specificity or above', () => {
    const offenders = RULES.filter((r) => r.at.length === 0 && r.selectors.some((s) => {
      if (!/:hover/.test(s) || /\[data-ground\]/.test(s)) return false;
      const subj = subject(s);
      if (!FACES.some((f) => f.split('.').filter(Boolean).every((c) => hasClass(subj, c)))) return false;
      const p = props(r.body);
      if (!p.has('box-shadow') && !p.has('background') && !p.has('background-color')) return false;
      // the face rule is :root[data-ground] :is(...) = 3 class-level parts; count this selector's
      const parts = (s.replace(/:(?:where)\([^)]*\)/g, '').match(/\.[\w-]+|\[[^\]]+\]|:(?!:)[\w-]+/g) ?? []).length;
      return parts >= 3;
    }));
    expect(offenders.map((r) => r.selectors.join(', '))).toEqual([]);
  });

  test('a hovered face brightens its rim, bottom edge and pool; the glow themes tint the light with their accent', () => {
    const knob = TOP.filter((r) => r.selectors.some((s) => /^:root\[data-ground\] :is\(/.test(s) && /\)\.hoverable:hover$/.test(s)) && /--gl-pool-a/.test(r.body));
    expect(knob.length).toBe(1);
    const members = splitSelectors(/:is\(([\s\S]*)\)\.hoverable:hover$/.exec(knob[0].selectors[0])![1]).map(norm);
    expect(FACES.filter((f) => !members.includes(f))).toEqual([]);
    const p = props(knob[0].body);
    expect([...p.keys()].every((k) => k.startsWith('--gl-')), [...p.keys()].join()).toBe(true);
    expect(p.get('--gl-rim-a')).toBe('28%');
    expect(p.get('--gl-low-a')).toBe('40%');
    expect(p.get('--gl-pool-a')).toBe('14%');
    const tint = TOP.filter((r) => r.selectors.some((s) => /data-theme='frost'/.test(s) && /\.hoverable:hover$/.test(s) && /\[data-ground\]/.test(s)));
    expect(tint.length).toBe(1);
    const t = props(tint[0].body);
    expect([...t.keys()]).toEqual(['--gl-tint']);
    expect(t.get('--gl-tint')).toBe('var(--accent)');
    for (const theme of ['red', 'grey-frost', 'light', 'daylight']) expect(tint[0].selectors[0], theme).toContain(`[data-theme='${theme}']`);
  });
});

describe('cards that are cards only at a breakpoint, and the Creative Engine sheets (glass phase 3)', () => {
  test('the org connector is a glass card from the wide breakpoint up (no wide: surface utilities)', () => {
    const line = read('app/org/page.tsx').split('\n').find((l) => /org-connector/.test(l))!;
    expect(line).not.toMatch(/wide:(bg-os-surface|border|rounded)/);
    const wide = RULES.filter((r) => r.at.some((a) => /min-width: 1800px/.test(a) && !/reduced/.test(a)) && r.selectors.some((x) => /\[data-ground\] \.org-connector$/.test(x)));
    expect(wide.length).toBe(1);
    const p = props(wide[0].body);
    expect(p.get('background')).toBe('transparent');
    expect(p.get('backdrop-filter')).toBe('var(--gl-blur)');
    expect(p.get('border-radius')).toBe('var(--gl-r-card)');
    expect(layers(p.get('box-shadow')!)).toEqual(
      layers(props(TOP.filter((r) => r.selectors.some((x) => /\[data-ground\] :is\(\s*\.glass-face,/.test(x)) && /box-shadow/.test(r.body))[0].body).get('box-shadow')!),
    );
    for (const k of ['--gl-tint', '--gl-low-a']) expect(p.has(k) || TOP.some((r) => r.selectors.some((x) => /org-connector/.test(x)) && props(r.body).has(k)), k).toBe(true);
  });

  const classic = read('components/creative-engine/classic.tsx');
  const fn = (name: string) => {
    const i = classic.indexOf(`function ${name}(`);
    const j = classic.indexOf('\nexport function', i + 1);
    return classic.slice(i, j < 0 ? undefined : j);
  };
  test.each(['ClassicSheet', 'ClassicDrawer', 'ClassicConfirmDelete'])('%s: the scrim is the sheet\'s sibling and the sheet a strong glass face', (name) => {
    const src = fn(name);
    expect(src).toMatch(/<div className="[^"]*\bae-sheet-layer\b[^"]*">\s*<div className="ae-sheet-scrim gl-scrim" aria-hidden onClick=\{\w+\} \/>/);
    expect(src).toMatch(/className=\{?[`"][^`"]*\bae-sheet\b[^`"]*\bglass-face glass-face--strong\b/);
    expect(src).not.toMatch(/stopPropagation/);
    expect(src).not.toMatch(/bg-\[var\(--p-card-2\)\]|shadow-\[var\(--p-shadow\)\]/);
  });

  test('the sheet scrim is a viewport layer, not a wrapper; Legacy\'s ae hairline leaves glass faces alone', () => {
    const layer = RULES.filter((r) => r.selectors.some((x) => /\.ae-sheet-layer > \.ae-sheet-scrim$/.test(x)));
    expect(layer.length).toBe(1);
    expect(props(layer[0].body).get('position')).toBe('fixed');
    expect(props(layer[0].body).get('inset')).toBe('0');
    const legacy = RULES.filter((r) => r.selectors.some((x) => /^:root\[data-ui='legacy'\] \.ae-paper :is\(\.ae-card,/.test(x)));
    expect(legacy.length).toBeGreaterThan(0);
    for (const r of legacy) for (const x of r.selectors) expect(x).toMatch(/:not\(\.glass-face\)$/);
  });
});

describe('"on" is a glow behind and a tinted rim, never a fill: pills, chips and tabs across the app (glass phase 3)', () => {
  const ON = [
    ".ap-pill[aria-pressed='true']", '.bh-itab.is-on', '.ib-chip.is-on', '.ib-seg-btn.is-on', ".ae-chip[data-on='true']",
    ".ae-pill[data-active='true']", '.g-pill[data-active]', ".wiz-bc-chip[aria-pressed='true']", ".g-btn.kg-radial-row[aria-pressed='true']",
    ".lab2-seg-opt[aria-selected='true']", ".lab2c-fchip[aria-current='true']", ".ch-g-strip-btn[aria-pressed='true']",
    ".ch-c-strip-btn[aria-pressed='true']", ".fo-c-filter[aria-pressed='true']",
  ];
  const onRule = TOP.filter((r) => r.selectors.some((s) => /^:root\[data-ground\] :is\(/.test(s) && s.includes(".fo-c-filter[aria-pressed='true']")) && /box-shadow/.test(r.body));

  test('every listed on-state clears its fill and takes the tinted six layers', () => {
    expect(onRule.length).toBe(1);
    const members = splitSelectors(/:is\(([\s\S]*)\)$/.exec(onRule[0].selectors[0])![1]).map(norm);
    expect(ON.filter((s) => !members.includes(s))).toEqual([]);
    const p = props(onRule[0].body);
    expect(p.get('background')).toBe('transparent');
    expect(p.get('border-color')).toBe('transparent');
    expect(layers(p.get('box-shadow')!)).toHaveLength(6);
    expect(p.get('box-shadow')).toMatch(/var\(--gl-tint\) var\(--gl-rim-a\)/);
  });

  test('the glow sits behind (a blurred ::before at .55), the rim takes its colour, the inverted ones read in ink again', () => {
    const glow = TOP.filter((r) => r.selectors.some((s) => s.includes(".fo-c-filter[aria-pressed='true']") && /::before$/.test(s)));
    expect(glow.length).toBe(1);
    const g = props(glow[0].body);
    expect(g.get('z-index')).toBe('-1');
    expect(g.get('filter')).toBe('blur(40px)');
    expect(g.get('opacity')).toBe('0.55');
    const knobs = TOP.filter((r) => r.selectors.some((s) => /^:where\(/.test(s) && s.includes(".fo-c-filter[aria-pressed='true']")));
    expect(knobs.length).toBe(1);
    expect(props(knobs[0].body).get('--gl-tint')).toBe('var(--gl-on)');
    expect(props(knobs[0].body).get('isolation')).toBe('isolate');
    const ink = (sel: string) => TOP.filter((r) => r.selectors.some((s) => s.includes(sel) && /\[data-ground\]/.test(s)) && props(r.body).has('color')).map((r) => props(r.body).get('color'));
    expect(ink(".fo-c-filter[aria-pressed='true']")).toContain('var(--text)');
    expect(ink('.ib-chip.is-on')).toContain('var(--text)');
    expect(ink(".lab2-seg-opt[aria-selected='true']")).toContain('var(--lab-ink-hero)');
    expect(ink(".ae-chip[data-on='true']")).toContain('var(--p-ink)');
  });

  test('TSX pills, chips and tabs mark "on" with gl-on instead of an accent or ink fill', () => {
    const cases: [string, RegExp][] = [
      ['founderos/components/slab.tsx', /active \? 'glass-chip gl-on /],
      ['founderos/components/SlidingTabs.tsx', /active \? 'gl-on /],
      ['app/org/page.tsx', /!venture \? 'glass-chip gl-on /],
      ['components/content/Composer.tsx', /tab === t \? 'gl-on /],
      ['components/AgentWorkPanel.tsx', /cron\.enabled \? 'glass-chip gl-on /],
    ];
    for (const [f, re] of cases) expect(read(f), f).toMatch(re);
    for (const f of ['components/content/Composer.tsx', 'components/content/ParamFields.tsx']) {
      expect(read(f), f).not.toMatch(/'border-os-accent\/45 bg-os-accent\/12 text-os-accent'/);
    }
    expect(read('components/brand-deals/DealBoard.tsx')).not.toMatch(/\? 'bg-os-accent font-semibold text-os-ink'/);
  });
});

describe('every element that still frosts itself in TSX is accounted for (glass phase 3 sweep)', () => {
  /** [file, a needle within 9 lines above the blur, why it is not a glass face]. Faces use the pack's classes. */
  const ALLOWED: [string, string, string][] = [
    ['founderos/components/slab.tsx', 'rotate-[24deg] rounded-[36px] border', 'InsightCard glass shard: a decorative leaf inside the one gradient card'],
    ['components/brand-deals/DealBoard.tsx', 'rotate-[24deg]', 'insight card glass shard: a decorative leaf'],
    ['components/brand-deals/DealBoard.tsx', 'bg-white/15 px-3 py-1', 'insight card badge on the gradient card (not glass, ink on the gradient)'],
    ['components/brand-deals/DealBoard.tsx', 'fixed inset-0 z-[60] bg-black/45', 'drawer scrim, the sibling of the glass drawer'],
    ['components/BrainCore.tsx', 'overlay-in fixed inset-0', 'pop-out scrim, the sibling of its panel'],
    ['components/wizard/CreateAgentModal.tsx', 'aria-label="Close" onClick={onClose} className="absolute inset-0', 'modal scrim (a close button), the sibling of the dialog'],
    ['components/adpilot/AdLibrary.tsx', 'ap-ask-glass absolute inset-0', 'the ask bar frost layer, under its content (like .wiz-ask-glass)'],
    ['components/adpilot/AdLibrary.tsx', 'ap-save absolute right-2 top-2', 'save toggle over a media thumbnail: a control on a leaf'],
    ['components/GBrainRadial.tsx', "title={fs ? 'Exit full screen (esc)'", 'full-screen toggle: a control'],
    ['components/GBrainRadial.tsx', "phase === 'typing' || phase === 'scan'", 'the radial ask input pill: a control (its answer card is the glass face)'],
    ['components/knowledge/KnowledgeVisualizer.tsx', "title={networkFs ? 'Exit full screen (esc)'", 'full-screen toggle: a control'],
    ['components/BrainDock.tsx', 'bd-line group absolute', 'the collapsed dock line: a button (the open dock card is the glass face)'],
    ['components/ConductorPanel.tsx', 'wiz-launch group fixed', 'the Conductor launcher: a button'],
    ['components/KnowledgeGraphFullscreen.tsx', 'kg-paddle', 'pillar paddles: buttons'],
    ['components/KnowledgeGraph.tsx', 'kg-paddle', 'pillar paddles: buttons'],
    ['components/GraphDirectory.tsx', 'kg-dir-rail', 'directory rail: a button (the open directory is the glass face)'],
    ['components/PipelineChart.tsx', 'AI prompt bar melting out of the chart base', 'prompt bar frost melting out of the chart: holds an input, no face'],
  ];

  test('no unlisted backdrop blur: an element that frosts itself is a glass face of the pack, a scrim, a leaf or a control', () => {
    const walk = (d: string): string[] =>
      require('node:fs')
        .readdirSync(join(process.cwd(), d), { withFileTypes: true })
        .flatMap((e: { name: string; isDirectory: () => boolean }) => (e.isDirectory() ? walk(`${d}/${e.name}`) : e.name.endsWith('.tsx') ? [`${d}/${e.name}`] : []));
    const left: string[] = [];
    for (const f of [...walk('app'), ...walk('components'), ...walk('founderos')]) {
      const lines = read(f).split('\n');
      lines.forEach((l, i) => {
        if (!/backdrop-blur(?!-none)|backdropFilter|WebkitBackdropFilter/.test(l) || /^\s*(\/\/|\*|\/\*)/.test(l)) return;
        const window = lines.slice(Math.max(0, i - 9), i + 1).join('\n');
        if (ALLOWED.some(([file, needle]) => file === f && window.includes(needle))) return;
        left.push(`${f}:${i + 1} ${l.trim().slice(0, 100)}`);
      });
    }
    expect(left).toEqual([]);
  });

  test('every allowlisted line still exists (the list cannot rot)', () => {
    for (const [file, needle] of ALLOWED) expect(read(file), `${file}: ${needle}`).toContain(needle);
  });
});

describe('pop-ups over text take the strong frost (review finding 5)', () => {
  const STRONG = RULES.find(
    (r) => r.at.length === 0 && r.selectors.some((s) => /^:root\[data-ground\] :is\(\.glass-face--strong,/.test(s)) && props(r.body).get('backdrop-filter') === 'var(--gl-blur-strong)',
  )!;
  const strongMembers = () => /:is\(([^)]*)\)/.exec(STRONG.selectors[0])![1].split(/\s*,\s*/);

  test('the toast is a strong-frost pill', () => {
    expect(strongMembers()).toContain('.ui-toast');
    const radius = TOP.filter((r) => r.selectors.includes('.ui-toast')).map((r) => props(r.body).get('border-radius')).filter(Boolean).pop();
    expect(radius).toBe('var(--gl-r-chip)');
  });

  test('the toast drops from the top right (y -40 to 0 on S), holds, and melts', () => {
    const stack = props(TOP.filter((r) => r.selectors.includes('.ui-toasts')).map((r) => r.body).join(';'));
    expect(stack.get('top')).toBeDefined();
    expect(stack.get('right')).toBeDefined();
    expect(stack.get('bottom') ?? 'auto').toBe('auto');
    expect(stack.get('left') ?? 'auto').toBe('auto');
    expect(stack.get('align-items')).toBe('flex-end');
    expect(KF.get('gl-materialise-toast')).toMatch(/^from \{ transform: translateY\(-40px\);/);
    expect(KF.get('ui-toast-in')).toBe('from { transform: translateY(-40px); }');
    const face = RULES.filter((r) => r.at.length === 0 && r.selectors.includes(':root[data-ground] .ui-toast')).map((r) => props(r.body).get('animation')).filter(Boolean).pop();
    expect(face).toMatch(/^gl-materialise-toast var\(--gl-dur-materialise\) var\(--gl-spring\)/);
  });

  test('the detail panels over the graphs are strong glass faces', () => {
    const kg = read('components/KnowledgeGraph.tsx');
    const panel = /className=\{gc\('kg-panel kg-graph-panel[^']*', '([^']*)'\)\}/.exec(kg);
    expect(panel, 'KnowledgeGraph detail overlay').not.toBeNull();
    expect(panel![0]).toMatch(/'kg-panel kg-graph-panel[^']*glass-face--strong/);
    expect(panel![1]).toMatch(/glass-face--strong/);
    expect(read('components/SkillHierarchy.tsx')).toMatch(/className="skh-panel glass-face glass-face--strong /);
  });
});
