/**
 * The glass branch review (phase 4): focus, "on" glow, old surfaces and radii, on the glass canon
 * (the settings the app, founderos.sh and the content calendar share).
 *
 * Colours are resolved statically from app/globals.css for every appearance the app ships: GladOS
 * (four accents, dark and light), Classic (five themes) and Bennett Legacy (Monolith, Daylight), each
 * on the ground lib/appearance.ts groundFor gives it.
 */
import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, test } from 'vitest';
import { groundFor } from '@/lib/appearance';
import { CSS_NO_COMMENTS, splitSelectors, topLevelRules } from './helpers/theme-css';
import { allRules, FACE_CLASSES, faceElementClasses, faceOf, tsxFiles } from './helpers/glass-lint';
import { evalColour, over, ratio, rootCascade, substitute, type RGBA } from './helpers/lab-colour';

type Look = { name: string; attrs: Record<string, string>; ground: string };

const LOOKS: Look[] = [
  ...(['ice', 'lime', 'ember', 'mint'] as const).flatMap((accent) =>
    (['dark', 'light'] as const).map((mode) => ({
      name: `glados ${accent} ${mode}`,
      attrs: { 'data-ui': 'glados', 'data-accent': accent, 'data-mode': mode, 'data-theme': mode === 'light' ? 'light' : 'console', 'data-ground': groundFor('console', mode, 'glados') },
      ground: '--g-bg',
    })),
  ),
  ...(['console', 'red', 'light', 'frost', 'grey-frost'] as const).map((theme) => {
    const mode = theme === 'light' ? 'light' : 'dark';
    return { name: `classic ${theme}`, attrs: { 'data-ui': 'classic', 'data-accent': 'ice', 'data-mode': mode, 'data-theme': theme, 'data-ground': groundFor(theme, mode, 'classic') }, ground: '--bg' };
  }),
  ...(['monolith', 'daylight'] as const).map((theme) => {
    const mode = theme === 'daylight' ? 'light' : 'dark';
    return { name: `legacy ${theme}`, attrs: { 'data-ui': 'legacy', 'data-accent': 'ice', 'data-mode': mode, 'data-theme': theme, 'data-ground': groundFor(theme, mode, 'legacy') }, ground: '--l-bg' };
  }),
];

const colour = (env: Map<string, string>, token: string): RGBA => evalColour(substitute(`var(${token})`, (t) => env.get(t)));
const norm = (s: string) => s.replace(/\s+/g, ' ').trim();
const RULES = allRules();
const decl = (body: string, name: string) => {
  const m = new RegExp(`(?:^|[;{\\s])${name.replace(/[-]/g, '\\-')}\\s*:\\s*([^;]+)`).exec(body);
  return m ? norm(m[1]) : undefined;
};

/** Glass controls: the faces of the pack and every element the pack lights as a control or an "on" state. */
const CONTROLS = [
  ...FACE_CLASSES.map((c) => `.${c}`),
  '.glass-solid', '.glass-well', '.g-seg-opt', '.l-toggle-chip', '.os-nav-link', '.gl-on',
  '.ap-pill', '.bh-itab', '.ib-chip', '.ib-seg-btn', '.ae-chip', '.ae-pill', '.g-pill', '.wiz-bc-chip', '.kg-radial-row', '.lab2-seg-opt', '.lab2c-fchip',
  '.ch-g-strip-btn', '.ch-c-strip-btn', '.fo-c-filter', '.bg-os-surface.rounded-lg-t',
];

describe('focus: a ring in the accent, 2px, 2px out, at least 3:1 on the ground (review finding 4, glass canon)', () => {
  const ring = RULES.filter((r) => decl(r.body, 'outline') === '2px solid var(--gl-focus)' && decl(r.body, 'outline-offset') === '2px');

  test('every glass control shows the ring on :focus-visible; the ask bars when their field does', () => {
    expect(ring.length).toBe(1);
    const sels = ring[0].selectors.map(norm);
    const focusList = sels.find((s) => /^:root\[data-ground\] :is\(.*\):focus-visible$/.test(s));
    expect(focusList, 'a :root[data-ground] :is(...):focus-visible list').toBeDefined();
    const members = splitSelectors(/:is\((.*)\):focus-visible$/.exec(focusList!)![1]).map(norm);
    expect(CONTROLS.filter((c) => !members.includes(c))).toEqual([]);
    expect(sels).toEqual(expect.arrayContaining([':root[data-ground] :is(.wiz-ask, .g-ask-box):has(:focus-visible)']));
  });

  test('focus never leans on the rim: the GladOS ask bar keeps its own edge (no 18-24% ring tint)', () => {
    expect(CSS_NO_COMMENTS).not.toMatch(/\.g-ask-box:focus-within\s*\{[^}]*--gl-tint:\s*var\(--g-ring\)/);
    expect(CSS_NO_COMMENTS).not.toMatch(/\.wiz-ask:focus-within\s*\{[^}]*--gl-tint/);
  });

  test.each(LOOKS.map((l) => [l.name, l] as const))('%s: the ring is at least 3:1 against the page ground', (_, look) => {
    const env = rootCascade(look.attrs);
    const ground = colour(env, look.ground);
    expect(ground[3], 'an opaque ground').toBe(1);
    expect(ratio(colour(env, '--gl-focus'), ground)).toBeGreaterThanOrEqual(3);
  });
});

const read = (rel: string) => fs.readFileSync(path.join(__dirname, '..', rel), 'utf8');

/**
 * "On" on a large face. The glow is a sibling behind the face (a halo of the face's own shape: an outer shadow,
 * blur 40px, --gl-on at full strength, opacity .55), so nothing coloured sits directly under the text; what
 * reaches the text is what the face's frost pulls in from just outside its edge. Model (1D, across the nearest
 * edge): the halo's alpha at x px outside is .55 * (1 - PHI(x / 20)) (a 40px CSS shadow blur is a Gaussian of
 * sigma 20), and a frost of blur(s) is a Gaussian of sigma s, so the tint under text p px inside the edge is
 * the integral of the halo against that kernel. The text is then measured over the page ground tinted that much.
 */
const erf = (x: number) => {
  const t = 1 / (1 + 0.3275911 * Math.abs(x));
  const y = 1 - ((((1.061405429 * t - 1.453152027) * t + 1.421413741) * t - 0.284496736) * t + 0.254829592) * t * Math.exp(-x * x);
  return x >= 0 ? y : -y;
};
const PHI = (z: number) => 0.5 * (1 + erf(z / Math.SQRT2));
const pdf = (z: number) => Math.exp(-(z * z) / 2) / Math.sqrt(2 * Math.PI);
function haloTint(alpha: number, frostSigma: number, padding: number): number {
  let sum = 0;
  for (let x = 0; x < 400; x += 0.25) sum += alpha * (1 - PHI(x / 20)) * (pdf((x + padding) / frostSigma) / frostSigma) * 0.25;
  return sum;
}

/** The large "on" faces: where the glow lives, the face's frost and its tightest text padding. */
const LARGE_ON = [
  { file: 'components/BrainDock.tsx', face: /<div className="glass-face glass-face--strong gl-on-rim rounded-\[24px\] p-3">/, frost: 28, padding: 12 },
  { file: 'components/GBrainRadial.tsx', face: /className=\{`glass-face glass-face--strong \$\{showAnswer \? 'gl-on-rim' : ''\} rounded-\[24px\] px-3\.5 py-2`\}/, frost: 28, padding: 8 },
  { file: 'components/adpilot/AdPilotDeck.tsx', face: /<div className="ap-panel gl-on-rim h-full p-5">/, frost: 3, padding: 20 },
];

describe('"on" on a large face: a glow behind it, never a wash behind the text (review finding 3, canon amendment)', () => {
  const rule = (sel: string) => RULES.filter((r) => r.selectors.map(norm).includes(sel)).map((r) => r.body).join(';');

  test('the glow is a sibling: an isolating host, the glow first at z-index -1, a halo of the face at blur 40px and .55', () => {
    const host = rule('.gl-glow-host');
    expect(decl(host, 'position')).toBe('relative');
    expect(decl(host, 'isolation')).toBe('isolate');
    const glow = rule('.gl-glow');
    expect(decl(glow, 'position')).toBe('absolute');
    expect(decl(glow, 'inset')).toBe('0');
    expect(decl(glow, 'z-index')).toBe('-1');
    expect(decl(glow, 'border-radius')).toBe('inherit');
    expect(decl(glow, 'background')).toBe('none');
    expect(decl(glow, 'box-shadow')).toBe('0 0 40px var(--gl-on)');
    expect(decl(glow, 'opacity')).toBe('0.55');
    expect(decl(glow, 'pointer-events')).toBe('none');
    expect(decl(rule(':root[data-ground] .gl-on-rim'), '--gl-tint')).toBe('var(--gl-on)');
  });

  test.each(LARGE_ON.map((f) => [f.file, f] as const))('%s: the host, then the glow, then the face with the tinted rim (no ::before glow)', (file, f) => {
    const src = read(file);
    const host = src.indexOf('gl-glow-host');
    const glow = src.indexOf('className="gl-glow"', host);
    const face = src.search(f.face);
    expect(host, 'host').toBeGreaterThan(-1);
    expect(glow, 'glow after the host').toBeGreaterThan(host);
    expect(face, 'face after the glow').toBeGreaterThan(glow);
    expect(src).not.toMatch(/[\s'"`]gl-on[\s'"`]/);
  });

  test('TSX keeps the ::before glow (gl-on) for small faces only: chips, pills and tabs', () => {
    const SMALL_TABS: Record<string, string> = {
      'components/content/Composer.tsx': 'the composer tab buttons (a row of text tabs)',
      'founderos/components/SlidingTabs.tsx': 'the sliding tab buttons',
    };
    const offenders: string[] = [];
    for (const dir of ['app', 'components', 'founderos']) {
      const walk = (d: string) => {
        for (const e of fs.readdirSync(path.join(__dirname, '..', d), { withFileTypes: true })) {
          const rel = path.join(d, e.name);
          if (e.isDirectory()) walk(rel);
          else if (e.name.endsWith('.tsx')) {
            read(rel).split('\n').forEach((line, i) => {
              for (const m of line.matchAll(/['"`]([^'"`]*\bgl-on\b[^'"`]*)['"`]/g)) {
                if (/(^|\s)gl-on(\s|$)/.test(m[1]) && !/\bglass-chip\b/.test(m[1]) && !SMALL_TABS[rel]) offenders.push(`${rel}:${i + 1} ${m[1]}`);
              }
            });
          }
        }
      };
      walk(dir);
    }
    expect(offenders).toEqual([]);
  });

  test.each(LOOKS.flatMap((l) => LARGE_ON.map((f) => [`${l.name} · ${f.file}`, l, f] as const)))('%s: body and secondary text stay at 4.5:1 over the glow', (_, look, f) => {
    const env = rootCascade(look.attrs);
    const ground = colour(env, look.ground);
    const on = colour(env, '--gl-on');
    const tint = haloTint(0.55, f.frost, f.padding);
    expect(tint).toBeLessThan(0.06);
    const under = over([on[0], on[1], on[2], tint], ground);
    for (const t of ['--text', '--text-2']) expect(ratio(colour(env, t), under), t).toBeGreaterThanOrEqual(4.5);
  });

  test('the old model (the blob centred under the text at .55) would fail: the reason the glow moved out', () => {
    const fails = LOOKS.filter((l) => {
      const env = rootCascade(l.attrs);
      const on = colour(env, '--gl-on');
      return ratio(colour(env, '--text-2'), over([on[0], on[1], on[2], 0.55], colour(env, l.ground))) < 4.5;
    });
    expect(fails.length).toBe(LOOKS.length);
  });
});

describe('frost costs nothing at rest (review finding 10)', () => {
  const FACE_ELEMENTS = tsxFiles(path.join(__dirname, '..'), ['app', 'components', 'founderos']).flatMap((abs) => faceElementClasses(abs, fs.readFileSync(abs, 'utf8')));
  const sets = FACE_ELEMENTS.map((t) => new Set(t));
  /** Does a selector's subject land on a face (a face class, or classes some face element in TSX carries)? */
  const onFace = (sel: string) => {
    if (sel.includes('::')) return false;
    const last = sel.replace(/:(?:not|has|where)\([^)]*\)/g, '').split(/\s*[\s>+~]\s*/).pop() ?? '';
    const is = /:is\(([^)]*)\)/.exec(last);
    return (is ? is[1].split(',').map((x) => x.trim()) : [last]).some((a) => {
      const cls = [...a.matchAll(/\.([\w-]+)/g)].map((m) => m[1]);
      return cls.length > 0 && (faceOf(cls) !== undefined || sets.some((x) => cls.every((c) => x.has(c)) && faceOf([...x]) !== undefined));
    });
  };

  test('a face nested in a frosted face (no frost of its own) materialises without animating a frost', () => {
    const kf = /@keyframes gl-materialise-lit\s*\{([^{}]*\{[^{}]*\})\s*\}/.exec(CSS_NO_COMMENTS);
    expect(kf, '@keyframes gl-materialise-lit').not.toBeNull();
    expect(kf![1]).not.toMatch(/backdrop-filter|opacity/);
    expect(norm(kf![1])).toMatch(/transform: translateY\(18px\) scale\(0\.965\)/);
    // only where motion is allowed (reduced motion keeps animation: none), and with the entering names only
    const nested = RULES.filter((r) => decl(r.body, 'animation-name') === 'gl-materialise-lit, gl-rim-in, gl-pool-in');
    expect(nested.length).toBe(1);
    const blocks = [...CSS_NO_COMMENTS.matchAll(/@media \(prefers-reduced-motion: no-preference\) \{([\s\S]*?\})\s*\}/g)].map((m) => m[1]);
    expect(blocks.some((b) => b.includes('gl-materialise-lit, gl-rim-in, gl-pool-in'))).toBe(true);
    const sel = norm(nested[0].selectors.join(', '));
    for (const k of ['.gl-materialise', '.l-card.l-rise', '.panel-in']) expect(sel, k).toContain(k);
  });

  test('no face transitions its frost (no transition on all or backdrop-filter)', () => {
    const bad = RULES.filter((r) => /\ball\b|backdrop-filter/.test(decl(r.body, 'transition') ?? decl(r.body, 'transition-property') ?? '')).flatMap((r) => r.selectors.filter(onFace));
    expect(bad).toEqual([]);
  });

  test('no face loops an animation', () => {
    const bad = RULES.filter((r) => /\binfinite\b/.test(decl(r.body, 'animation') ?? '')).flatMap((r) => r.selectors.filter(onFace));
    expect(bad).toEqual([]);
  });

  test('the ask bars hold their orbit still at rest: it turns only while the bar has focus or is busy (it sits under a 24px frost)', () => {
    for (const [orbit, live] of [
      ['.bh-ask-orbit', ['.bh-ask:focus-within .bh-ask-orbit', '.bh-ask.is-busy .bh-ask-orbit']],
      ['.wiz-ask-orbit', ['.wiz-ask:focus-within .wiz-ask-orbit', '.wiz-ask.is-busy .wiz-ask-orbit']],
      ['.ap-orbit', [".ap-orbit[data-spin='1']"]],
    ] as const) {
      const rest = RULES.filter((r) => r.selectors.map(norm).includes(orbit)).map((r) => decl(r.body, 'animation-play-state')).filter(Boolean).pop();
      expect(rest, orbit).toBe('paused');
      for (const sel of live) expect(RULES.some((r) => r.selectors.map(norm).includes(sel) && decl(r.body, 'animation-play-state') === 'running'), sel).toBe(true);
    }
    // GladOS's bar has no animation at rest at all
    expect(RULES.filter((r) => r.selectors.map(norm).includes('.g-ask-orbit') && decl(r.body, 'animation'))).toEqual([]);
  });
});

describe('faces that became glass in phase 4 never fade (brief §4: faces appear and go by visibility, transform and light)', () => {
  const canvas = allRules(read('components/workflow-canvas/canvas.css').replace(/\/\*[\s\S]*?\*\//g, ''));
  const CASES: [typeof RULES, string[]][] = [
    [RULES, ['.bh-search-panel', '.bh-search.is-open .bh-search-panel']],
    [canvas, ['.wfc .panel', '.wfc .panel.is-open', '.wfc .dock-chat', '.wfc .dock-chat.is-hidden', '.wfc .dock-logs', '.wfc .dock-logs.is-hidden']],
  ];
  test.each(CASES.flatMap(([rules, sels]) => sels.map((sel) => [sel, rules] as const)))('%s: no opacity, no opacity transition', (sel, rules) => {
    const bodies = rules.filter((r) => r.selectors.map(norm).includes(sel)).map((r) => r.body);
    expect(bodies.length, sel).toBeGreaterThan(0);
    for (const b of bodies) {
      expect(decl(b, 'opacity'), sel).toBeUndefined();
      expect(decl(b, 'transition') ?? '', sel).not.toMatch(/\bopacity\b/);
    }
  });

  test('the founderos toast is a strong-frost glass pill, not a filled card', () => {
    const src = read('founderos/components/Toaster.tsx');
    expect(src).toMatch(/className=\{`pressable pointer-events-auto relative flex items-center gap-2 glass-face glass-face--strong rounded-full /);
    expect(src).not.toMatch(/rounded-panel|bg-os-bg px-2\.5|shadow-\[0_12px_30px/);
  });
});

export { LOOKS, colour };
