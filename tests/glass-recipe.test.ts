/**
 * The glass UI pack (FounderOS Glass brief, sections 1, 2 and 4) in app/globals.css.
 *
 * Glass is shaped by light, never by fill: every face is transparent with a groove and a
 * drop outside, a shade under the top edge, a rim brightest along the bottom and light
 * pooling at the bottom inside, frosting whatever is behind it. "On" is a glow behind the
 * face plus a tinted rim. Faces never fade in: they materialise and melt.
 */
import { describe, expect, test } from 'vitest';
import { CSS_NO_COMMENTS, splitSelectors, topLevelRules } from './helpers/theme-css';

const norm = (s: string) => s.replace(/\s+/g, ' ').trim();

/** Every declaration of a rule body (custom properties included), last one wins. */
function props(body: string): Map<string, string> {
  const out = new Map<string, string>();
  let depth = 0;
  let cur = '';
  for (const c of body) {
    if (c === '(') depth++;
    if (c === ')') depth--;
    if (c === ';' && depth === 0) {
      const i = cur.indexOf(':');
      if (i > 0) out.set(cur.slice(0, i).trim(), norm(cur.slice(i + 1)));
      cur = '';
    } else cur += c;
  }
  const i = cur.indexOf(':');
  if (i > 0 && cur.trim()) out.set(cur.slice(0, i).trim(), norm(cur.slice(i + 1)));
  return out;
}

/** Split a value on top-level commas (box-shadow layers, animation lists). */
function layers(v: string): string[] {
  const out: string[] = [];
  let depth = 0;
  let cur = '';
  for (const c of v) {
    if (c === '(') depth++;
    if (c === ')') depth--;
    if (c === ',' && depth === 0) {
      out.push(norm(cur));
      cur = '';
    } else cur += c;
  }
  if (cur.trim()) out.push(norm(cur));
  return out;
}

/** Bodies of every at-rule block whose head matches, e.g. /^@keyframes gl-orbit$/. */
function atBlocks(head: RegExp, src = CSS_NO_COMMENTS): string[] {
  const out: string[] = [];
  let depth = 0;
  let start = 0;
  let open = -1;
  for (let i = 0; i < src.length; i++) {
    const c = src[i];
    if (c === '{') {
      if (depth === 0) open = i;
      depth++;
    } else if (c === '}') {
      depth--;
      if (depth === 0) {
        const h = norm(src.slice(start, open));
        if (head.test(h)) out.push(src.slice(open + 1, i));
        start = i + 1;
      }
    } else if (c === ';' && depth === 0) start = i + 1;
  }
  return out;
}

const keyframes = (name: string) => {
  const b = atBlocks(new RegExp(`^@keyframes ${name}$`));
  expect(b.length, `@keyframes ${name}`).toBeGreaterThan(0);
  return norm(b[b.length - 1]);
};

/**
 * The subject's members: a trailing :is(...) group's members, or the last compound
 * (`:root[data-ground] .g-drawer` → `.g-drawer`). In `:root[data-ground] :is(A) :is(B)` the subject is B.
 */
function members(sel: string): string[] {
  const t = sel.trim();
  if (t.endsWith(')')) {
    let depth = 0;
    for (let i = t.length - 1; i >= 0; i--) {
      if (t[i] === ')') depth++;
      else if (t[i] === '(' && --depth === 0) {
        if (t.slice(i - 3, i) === ':is') return splitSelectors(t.slice(i + 1, -1));
        break;
      }
    }
  }
  return [t.split(/\s+/).pop()!];
}

const RULES = topLevelRules();
const ruleWith = (pred: (sel: string) => boolean) => RULES.filter((r) => r.selectors.some(pred));

/** The one rule that carries the recipe: the brief's .glass-face block. */
const RECIPE = (() => {
  const r = ruleWith((s) => members(s).includes('.glass-face') && /\[data-ground\]/.test(s)).filter((x) => props(x.body).has('box-shadow'));
  expect(r.length, 'one recipe rule holds .glass-face').toBe(1);
  return r[0];
})();
const RECIPE_MEMBERS = RECIPE.selectors.flatMap(members);
const RECIPE_PROPS = props(RECIPE.body);

const SIX = [
  '0 0 0 1px var(--gl-groove)',
  '0 8px 18px -8px var(--gl-drop)',
  'inset 0 0 0 1px color-mix(in srgb, var(--gl-tint) var(--gl-rim-a), transparent)',
  'inset 0 -1px 0 color-mix(in srgb, var(--gl-tint) var(--gl-low-a), transparent)',
  'inset 0 -18px 26px -18px color-mix(in srgb, var(--gl-tint) var(--gl-pool-a), transparent)',
  'inset 0 12px 18px -14px var(--gl-shade)',
];

describe('recipe tokens (brief §2)', () => {
  test(':root carries the exact groove, drop, shade and frost values', () => {
    const root = RULES.filter((r) => r.selectors.length === 1 && r.selectors[0] === ':root' && props(r.body).has('--gl-groove'));
    expect(root.length).toBe(1);
    const p = props(root[0].body);
    expect(p.get('--gl-groove')).toBe('color-mix(in srgb, #000 55%, transparent)');
    expect(p.get('--gl-drop')).toBe('color-mix(in srgb, #000 60%, transparent)');
    expect(p.get('--gl-shade')).toBe('color-mix(in srgb, #000 50%, transparent)');
    expect(p.get('--gl-blur')).toBe('blur(3px) saturate(150%)');
    expect(p.get('--gl-blur-strong')).toBe('blur(28px) saturate(160%)');
  });

  test("a white ground turns the groove into a hairline and lets the drop carry the shape", () => {
    const white = RULES.filter((r) => r.selectors.includes("[data-ground='white']"));
    expect(white.length).toBe(1);
    const p = props(white[0].body);
    expect(p.get('--gl-groove')).toBe('rgba(16, 16, 24, 0.10)');
    expect(p.get('--gl-drop')).toBe('rgba(16, 16, 24, 0.22)');
    expect(p.get('--gl-shade')).toBe('rgba(16, 16, 24, 0.06)');
    // same specificity as :root, so it must come later in the file
    const rootAt = RULES.findIndex((r) => r.selectors[0] === ':root' && props(r.body).has('--gl-groove'));
    expect(RULES.indexOf(white[0])).toBeGreaterThan(rootAt);
  });

  test('radii: card 24, well 16, chip a pill', () => {
    const root = RULES.find((r) => r.selectors[0] === ':root' && props(r.body).has('--gl-groove'))!;
    const p = props(root.body);
    expect(p.get('--gl-r-card')).toBe('24px');
    expect(p.get('--gl-r-well')).toBe('16px');
    expect(p.get('--gl-r-chip')).toBe('999px');
  });

  test('every theme scope names an opaque solid fallback (--solid-bg) from its own surface token', () => {
    const solid = (sel: string) => RULES.filter((r) => r.selectors.includes(sel)).map((r) => props(r.body).get('--solid-bg')).filter(Boolean).pop();
    expect(solid(':root')).toBe('var(--surface)');
    expect(solid(":root[data-ui='glados']")).toBe('var(--g-card)');
    expect(solid(":root[data-ui='legacy']")).toBe('var(--l-surface)');
    expect(solid('.lab')).toBe('var(--lab-glass-solid)');
  });
});

describe('the face (brief §2)', () => {
  test('.glass-face is transparent, frosted with the light blur and lit by the six layers in order', () => {
    expect(RECIPE_PROPS.get('background')).toBe('transparent');
    expect(RECIPE_PROPS.get('border-color')).toBe('transparent');
    expect(RECIPE_PROPS.get('backdrop-filter')).toBe('var(--gl-blur)');
    expect(RECIPE_PROPS.get('-webkit-backdrop-filter')).toBe('var(--gl-blur)');
    expect(layers(RECIPE_PROPS.get('box-shadow')!)).toEqual(SIX);
  });

  test('the knobs default to the brief (tint white, rim 15%, low 30%, pool 9%) and stay overridable', () => {
    const knobs = ruleWith((s) => /^:where\(/.test(s) && splitSelectors(s.slice(7, -1)).includes('.glass-face'));
    expect(knobs.length).toBe(1);
    const p = props(knobs[0].body);
    expect(p.get('--gl-tint')).toBe('#fff');
    expect(p.get('--gl-rim-a')).toBe('15%');
    expect(p.get('--gl-low-a')).toBe('30%');
    expect(p.get('--gl-pool-a')).toBe('9%');
  });

  test('the light comes from the bottom: rim and pool are bottom insets, the top only gets the shade', () => {
    const sh = layers(RECIPE_PROPS.get('box-shadow')!);
    expect(sh.filter((l) => /^inset 0 -/.test(l))).toHaveLength(2);
    // no tinted layer is offset downward from the top edge (the old top highlights)
    expect(sh.filter((l) => /^inset 0 [1-9]/.test(l) && /tint/.test(l))).toEqual([]);
  });

  const FACES = [
    '.glass-face', '.l-card', '.l-chip', '.l-dialog', '.g-card', '.g-card-raised', '.g-well', '.g-chip',
    '.g-sheet', '.g-drawer', '.g-dialog', '.g-palette', '.ui-toast', '.c-pop', '.c-chip', '.h6-card', '.glass-panel',
    '.glass-panel-violet', '.glass-chip', '.lab2-glass', '.wiz-ask', '.g-ask-box',
  ];
  test.each(FACES)('%s resolves to the recipe', (cls) => {
    expect(RECIPE_MEMBERS).toContain(cls);
  });

  test('the ask bars keep their frosted inner layer, lit by the same knobs', () => {
    for (const cls of ['.wiz-ask-glass', '.g-ask-glass']) {
      const r = ruleWith((s) => members(s).includes(cls) && /\[data-ground\]/.test(s));
      expect(r.length, cls).toBeGreaterThan(0);
      const p = props(r[r.length - 1].body);
      expect(p.get('background'), cls).toBe('transparent');
      expect(p.get('backdrop-filter'), cls).toBe('var(--gl-blur-strong)');
      expect(p.get('box-shadow'), cls).toContain('var(--gl-rim-a)');
      expect(p.get('box-shadow'), cls).toContain('var(--gl-pool-a)');
    }
  });

  test('pop-ups, sheets, drawers, dialogs and the palette take the strong frost', () => {
    // the glass-over-glass rules (glass phase 3: `:is(faces) :is(...)`) unfrost nested faces; this is the frost itself
    const strong = ruleWith((s) => members(s).includes('.glass-face--strong') && /\[data-ground\]/.test(s) && !/\) :is\(/.test(s));
    expect(strong.length).toBe(1);
    const p = props(strong[0].body);
    expect(p.get('backdrop-filter')).toBe('var(--gl-blur-strong)');
    const m = strong[0].selectors.flatMap(members);
    for (const cls of ['.g-sheet', '.g-drawer', '.g-dialog', '.g-palette', '.c-pop', '.l-dialog']) expect(m, cls).toContain(cls);
  });

  test('tooltips and menus are solid raised faces with the same light, never frost', () => {
    const solid = ruleWith((s) => members(s).includes('.glass-solid') && /\[data-ground\]/.test(s));
    expect(solid.length).toBe(1);
    const p = props(solid[0].body);
    expect(p.get('background')).toBe('var(--solid-bg)');
    expect(p.get('backdrop-filter')).toBe('none');
    expect(layers(p.get('box-shadow')!)).toEqual(SIX);
    const m = solid[0].selectors.flatMap(members);
    for (const cls of ['.os-nav-tip', '.g-pop', '.g-tip', '.g-chart-tip']) expect(m, cls).toContain(cls);
  });

  test('reduced transparency: no blur, the same light on a solid face', () => {
    const blocks = atBlocks(/^@media \(prefers-reduced-transparency: reduce\)$/);
    const inner = blocks.flatMap((b) => topLevelRules(b));
    const r = inner.filter((x) => x.selectors.some((s) => members(s).includes('.glass-face')));
    expect(r.length).toBeGreaterThan(0);
    const p = props(r[r.length - 1].body);
    expect(p.get('backdrop-filter')).toBe('none');
    expect(p.get('background')).toBe('var(--solid-bg)');
    expect(r[r.length - 1].selectors.flatMap(members)).toEqual(expect.arrayContaining(RECIPE_MEMBERS));
  });

  test('the Legacy kill switch is gone: no glass panel is forced flat', () => {
    for (const r of ruleWith((s) => /glass-panel/.test(s))) {
      expect(r.body, r.selectors.join(', ')).not.toMatch(/backdrop-filter:\s*none\s*!important/);
    }
    expect(CSS_NO_COMMENTS).not.toMatch(/no fills, no glows/);
  });
});

describe('"on" is a glow behind the face and a tinted rim, never a fill (brief §5 B8)', () => {
  const ON = [
    ".g-chip[aria-pressed='true']",
    '.g-chip.is-on',
    ".g-seg-opt[aria-checked='true']",
    ".l-toggle-chip[data-on='true']",
    ".os-sidebar a.os-nav-link[aria-current='page']",
  ];
  const on = ruleWith((s) => /\[data-ground\]/.test(s) && members(s).includes('.gl-on'));
  const glow = ruleWith((s) => /\[data-ground\]/.test(s) && /::before$/.test(s) && members(s.replace(/::before$/, '')).includes('.gl-on'));

  test('one on-state rule tints the rim and drops the fill', () => {
    expect(on.length).toBe(1);
    const p = props(on[0].body);
    expect(p.get('--gl-tint')).toBe('var(--gl-on)');
    expect(p.get('background')).toBe('transparent');
    const m = on[0].selectors.flatMap(members);
    for (const s of ON) expect(m, s).toContain(s);
  });

  test('the glow sits behind: 0.6x the face, blur 40px, opacity .55, below the content', () => {
    expect(glow.length).toBe(1);
    const p = props(glow[0].body);
    expect(p.get('content')).toBe("''");
    expect(p.get('inset')).toBe('20%');
    expect(p.get('filter')).toBe('blur(40px)');
    expect(p.get('opacity')).toBe('0.55');
    expect(p.get('z-index')).toBe('-1');
    expect(p.get('background')).toBe('var(--gl-on)');
  });

  test('status green for status, the theme accent otherwise', () => {
    const root = RULES.find((r) => r.selectors[0] === ':root' && props(r.body).has('--gl-groove'))!;
    expect(props(root.body).get('--gl-on')).toBe('var(--accent)');
    const ok = ruleWith((s) => /\[data-ground\]/.test(s) && /\[data-glow='ok'\]/.test(s));
    expect(ok.length).toBeGreaterThan(0);
    expect(props(ok[0].body).get('--gl-on')).toBe('var(--ok)');
  });
});

describe('motion (brief §2 keyframes, §4 Materialise and Melt)', () => {
  test('the four loop keyframes carry the exact values', () => {
    expect(keyframes('gl-breathe')).toBe('50% { filter: brightness(1.2); }');
    expect(keyframes('gl-sweep')).toBe('from { background-position: -60% 0; } to { background-position: 160% 0; }');
    expect(keyframes('gl-reflect')).toBe('50% { opacity: 0.42; }');
    expect(keyframes('gl-orbit')).toBe('to { transform: rotate(360deg); }');
    // g-orbit is an alias of gl-orbit
    expect(keyframes('g-orbit')).toBe(keyframes('gl-orbit'));
  });

  test('the knobs are registered, so Materialise and Melt can move them (glass canon)', () => {
    for (const [name, initial] of [['--gl-rim-a', '15%'], ['--gl-low-a', '30%'], ['--gl-pool-a', '9%']]) {
      const b = atBlocks(new RegExp(`^@property ${name}$`));
      expect(b.length, name).toBe(1);
      const p = props(b[0]);
      expect(p.get('syntax'), name).toBe("'<percentage>'");
      expect(p.get('inherits'), name).toBe('false');
      expect(p.get('initial-value'), name).toBe(initial);
    }
    // the outer groove and drop (and the inner shade) come in with the frost, so they are colours that interpolate
    for (const name of ['--gl-groove', '--gl-drop', '--gl-shade']) {
      const b = atBlocks(new RegExp(`^@property ${name}$`));
      expect(b.length, name).toBe(1);
      const p = props(b[0]);
      expect(p.get('syntax'), name).toBe("'<color>'");
      expect(p.get('inherits'), name).toBe('true');
      expect(p.get('initial-value'), name).toBe('transparent');
    }
  });

  test('Materialise (12f, 500ms): frost, groove, drop and shade from 0, scale .965 and y +18 on the spring; never opacity', () => {
    const k = keyframes('gl-materialise');
    expect(k).not.toMatch(/opacity/);
    expect(k).toMatch(/^(from|0%) \{[^}]*\}$/);
    expect(k).toMatch(/transform: translateY\(18px\) scale\(0\.965\)/);
    expect(k).toMatch(/[^-]backdrop-filter: blur\(0px\) saturate\(100%\)/);
    for (const v of ['--gl-groove', '--gl-drop', '--gl-shade']) expect(k, v).toMatch(new RegExp(`${v}: transparent`));
    // the light is not on the spring: the rim (and bottom edge) draw in over 10f, then the pool rises over 8f
    expect(keyframes('gl-rim-in')).toBe('from { --gl-rim-a: 0%; --gl-low-a: 0%; }');
    expect(keyframes('gl-pool-in')).toBe('from { --gl-pool-a: 0%; }');
  });

  test('every materialising face plays the frost on the spring, the rim over 417ms, then the pool 167ms later over 333ms', () => {
    const LIST = /^gl-materialise(-drawer|-toast)? var\(--gl-dur-(materialise|drawer)\) var\(--gl-spring(-drawer)?\) var\(--gl-delay, 0ms\) backwards, gl-rim-in var\(--gl-dur-rim\) linear var\(--gl-delay, 0ms\) backwards, gl-pool-in var\(--gl-dur-pool\) linear calc\(var\(--gl-delay, 0ms\) \+ var\(--gl-pool-after\)\) backwards$/;
    const plays = RULES.filter((r) => /(^|[;\s])animation\s*:\s*gl-materialise/.test(r.body));
    expect(plays.length).toBeGreaterThan(5);
    for (const r of plays) expect(props(r.body).get('animation'), r.selectors.join(', ')).toMatch(LIST);
    const root = RULES.find((r) => r.selectors[0] === ':root' && props(r.body).has('--gl-groove'))!;
    const p = props(root.body);
    expect(p.get('--gl-dur-rim')).toBe('417ms');
    expect(p.get('--gl-dur-pool')).toBe('333ms');
    expect(p.get('--gl-pool-after')).toBe('167ms');
  });

  test('the springs are linear() samplings of S {170, 20, .8} and the drawer spring {210, 26, 1}, both over 500ms', () => {
    const root = RULES.find((r) => r.selectors[0] === ':root' && props(r.body).has('--gl-groove'))!;
    const p = props(root.body);
    const spring = (k: number, c: number, m: number) => (t: number) => {
      const w0 = Math.sqrt(k / m);
      const z = c / (2 * Math.sqrt(k * m));
      const wd = w0 * Math.sqrt(1 - z * z);
      return 1 - Math.exp(-z * w0 * t) * (Math.cos(wd * t) + ((z * w0) / wd) * Math.sin(wd * t));
    };
    const check = (curve: string | undefined, f: (t: number) => number) => {
      expect(curve).toMatch(/^linear\(0, .*, 1\)$/);
      const stops = curve!.slice(7, -1).split(',').slice(1, -1).map((x) => x.trim().split(/\s+/));
      expect(stops.length).toBeGreaterThanOrEqual(19);
      for (const [v, at] of stops) expect(Math.abs(Number(v) - f((0.5 * parseFloat(at)) / 100)), `${v} at ${at}`).toBeLessThan(0.002);
    };
    check(p.get('--gl-spring'), spring(170, 20, 0.8));
    check(p.get('--gl-spring-drawer'), spring(210, 26, 1));
    expect(p.get('--gl-dur-materialise')).toBe('500ms');
    expect(p.get('--gl-dur-drawer')).toBe('500ms');
    expect(p.get('--gl-dur-melt')).toBe('333ms');
  });

  test('Melt: rim, bottom edge, pool, groove, drop, shade and frost to 0, scale to .98, linear; .is-melting plays it', () => {
    const k = keyframes('gl-melt');
    expect(k).not.toMatch(/opacity/);
    for (const v of ['--gl-rim-a', '--gl-low-a', '--gl-pool-a']) expect(k, v).toMatch(new RegExp(`to \\{[^}]*${v}: 0%`));
    for (const v of ['--gl-groove', '--gl-drop', '--gl-shade']) expect(k, v).toMatch(new RegExp(`to \\{[^}]*${v}: transparent`));
    expect(k).toMatch(/to \{[^}]*backdrop-filter: blur\(0px\) saturate\(100%\)/);
    expect(k).toMatch(/to \{[^}]*transform: scale\(0\.98\)/);
    const r = ruleWith((s) => /\.is-melting$/.test(s));
    expect(r.length).toBeGreaterThan(0);
    expect(props(r[0].body).get('animation')).toBe('gl-melt var(--gl-dur-melt) linear forwards');
  });

  test('staggers: cards 4f (167ms) apart, chips 3f (125ms)', () => {
    for (const sel of [':root[data-ground] .l-card.l-rise', ':root[data-ground] .gl-materialise.l-rise']) {
      const r = RULES.filter((x) => x.selectors.includes(sel));
      expect(r.length, sel).toBeGreaterThan(0);
      expect(r.map((x) => props(x.body).get('--gl-delay')).filter(Boolean).pop(), sel).toBe('calc(var(--l-rise-i, 0) * 167ms)');
    }
    expect(CSS_NO_COMMENTS).toMatch(/\.bh-ask-chip:nth-child\(2\) \{ animation-delay: 125ms; \}/);
  });

  test('no face entrance fades: every face keyframe moves only transform and the light', () => {
    for (const name of ['l-rise', 'g-sheet-in', 'g-drawer-in', 'g-pop-in', 'g-tip-in', 'ui-toast-in', 'lab2-rise', 'lab2-pop', 'g-fade', 'os-tip-in']) {
      expect(keyframes(name), name).not.toMatch(/opacity/);
    }
    expect(CSS_NO_COMMENTS).not.toMatch(/@keyframes lab-view-in/);
  });

  test('faces that used to fade in now materialise', () => {
    const anim = (cls: string) => {
      const r = ruleWith((s) => members(s).includes(cls) && /\[data-ground\]/.test(s)).filter((x) => /animation/.test(x.body));
      return r.map((x) => props(x.body).get('animation') ?? props(x.body).get('animation-name') ?? '').join(' | ');
    };
    for (const cls of ['.g-sheet', '.g-dialog', '.g-drawer', '.g-palette', '.ui-toast', '.l-dialog']) expect(anim(cls), cls).toMatch(/gl-materialise/);
  });

  test('reduced motion: faces appear at rest', () => {
    const blocks = atBlocks(/^@media \(prefers-reduced-motion: reduce\)$/);
    const inner = blocks.flatMap((b) => topLevelRules(b));
    const r = inner.filter((x) => x.selectors.some((s) => members(s).includes('.glass-face')));
    expect(r.length).toBeGreaterThan(0);
    expect(props(r[r.length - 1].body).get('animation')).toBe('none');
    expect(r[r.length - 1].selectors.flatMap(members)).toEqual(expect.arrayContaining(['.is-melting', '.g-sheet', '.ui-toast']));
  });
});

describe('radii on the existing token families', () => {
  const decl = (sel: string, name: string) =>
    RULES.filter((r) => r.selectors.includes(sel)).map((r) => props(r.body).get(name)).filter(Boolean).pop();
  test('GladOS, Legacy and Classic cards are 24, wells 16, chips pills', () => {
    expect(decl(":root[data-ui='glados']", '--g-r-card')).toBe('24px');
    expect(decl(":root[data-ui='glados']", '--g-r-well')).toBe('16px');
    expect(decl(":root[data-theme='monolith']", '--l-r-tile') ?? decl(':root', '--l-r-tile')).toBe('24px');
    expect(decl(":root[data-theme='monolith']", '--l-r-chip') ?? decl(':root', '--l-r-chip')).toBe('999px');
    expect(decl(':root', '--r-card')).toBe('24px');
    expect(decl('.h6-card', 'border-radius')).toBe('var(--gl-r-card)');
    expect(decl('.c-chip', 'border-radius')).toBe('var(--gl-r-chip)');
  });
});
