import { describe, expect, it } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { CSS_NO_COMMENTS, GLOBALS_CSS, declarations, topLevelRules } from './helpers/theme-css';
import {
  G_SCOPE,
  HERO_ZONES,
  deltaE,
  gCascade,
  gRules,
  over,
  parseBackground,
  ratio,
  resolve,
  tokenColour,
  worstContrast,
  type RGBA,
} from './helpers/g-colour';

/**
 * GladOS token block (docs/projects/glados-ui/SPEC.md S00 §5.1, §5.2, X2, GP3;
 * AC00.10, AC00.11, AC00.12 at token level, ACGP3.6). The block lives in
 * app/globals.css between "G0 tokens (lane 1c)" and "end G0 tokens".
 */

/**
 * The accent registry. lib/appearance.ts (lane 1a) owns ACCENTS; when it
 * exists this test reads it, so adding a fifth accent there fails here until
 * its token block is complete. Until then the S00 FR list is used.
 */
function accents(): string[] {
  const p = path.join(process.cwd(), 'lib/appearance.ts');
  if (fs.existsSync(p)) {
    const m = fs.readFileSync(p, 'utf8').match(/ACCENTS\s*(?::[^=]+)?=\s*\[([^\]]*)\]/);
    if (m) return [...m[1].matchAll(/'([\w-]+)'|"([\w-]+)"/g)].map((x) => x[1] ?? x[2]);
  }
  return ['ice', 'lime', 'ember', 'mint'];
}
const ACCENTS = accents();
const MODES = ['dark', 'light'] as const;
const VARIANTS = ['a', 'b', 'c'] as const;

/** X2 canonical tokens (aliases are documentation and are not declared). */
const TOKENS = {
  surfaces: ['--g-bg', '--g-bg-lift', '--g-card', '--g-card-2', '--g-well'],
  ink: ['--g-ink', '--g-ink-inverse', '--g-muted', '--g-dim'],
  lines: ['--g-line', '--g-ring', '--g-track', '--g-track-soft'],
  elevation: ['--g-scrim', '--g-shadow', '--g-shadow-sm'],
  status: ['--g-ok', '--g-warn', '--g-err', '--g-ok-soft', '--g-warn-soft', '--g-err-soft'],
  accent: ['--g-accent', '--g-accent-rgb', '--g-accent-2', '--g-accent-soft', '--g-badge', '--g-badge-ink'],
  heroes: [
    '--g-hero-a', '--g-hero-b', '--g-hero-c',
    '--g-hero-a-ink', '--g-hero-b-ink', '--g-hero-c-ink',
    '--g-on-hero', '--g-on-hero-dim', '--g-on-hero-faint',
  ],
  series: [1, 2, 3, 4, 5, 6].map((n) => `--g-series-${n}`),
  cat: Array.from({ length: 11 }, (_, i) => `--g-cat-${i + 1}`),
  states: ['--g-state-ready', '--g-state-approved', '--g-state-draft', '--g-state-sent'],
  viz: ['--g-viz-stage', '--g-viz-ink', '--g-viz-hud'],
  diff: ['--g-diff-del-bg', '--g-diff-del-fg', '--g-diff-add-bg', '--g-diff-add-fg'],
  shape: ['--g-r-card', '--g-r-nested', '--g-r-well', '--g-r-input', '--g-r-pill'],
  type: ['--g-fs-display', '--g-fs-section', '--g-fs-title', '--g-fs-card', '--g-fs-body', '--g-fs-cap', '--g-fs-faint', '--g-fw-light', '--g-fw-body', '--g-fw-badge'],
  space: [1, 2, 3, 4, 5, 6, 7, 8].map((n) => `--g-space-${n}`),
  motion: ['--g-dur-colour', '--g-dur-lift', '--g-dur-sheet', '--g-dur-arrive'],
};
const ALL = Object.values(TOKENS).flat();

/** Fixed values X2 / S00 §5.1 give for the mode-independent tokens (card and well radii: the glass pack, card 24, well 16). */
const FIXED: Record<string, string> = {
  '--g-r-card': '24px', '--g-r-nested': '20px', '--g-r-well': '16px', '--g-r-input': '12px', '--g-r-pill': '999px',
  '--g-fs-display': '44px', '--g-fs-section': '26px', '--g-fs-title': '28px', '--g-fs-card': '15px',
  '--g-fs-body': '13.5px', '--g-fs-cap': '11.5px', '--g-fs-faint': '11px',
  '--g-fw-light': '300', '--g-fw-body': '400', '--g-fw-badge': '500',
  '--g-space-1': '4px', '--g-space-2': '8px', '--g-space-3': '12px', '--g-space-4': '16px',
  '--g-space-5': '20px', '--g-space-6': '24px', '--g-space-7': '36px', '--g-space-8': '48px',
  '--g-dur-colour': '150ms', '--g-dur-lift': '180ms', '--g-dur-sheet': '240ms', '--g-dur-arrive': '420ms',
};

const RULES = gRules();
const envs = new Map<string, Map<string, string>>(ACCENTS.flatMap((a) => MODES.map((m) => [`${a}/${m}`, gCascade(a, m, RULES)] as const)));
const env = (a: string, m: string) => envs.get(`${a}/${m}`)!;
const combos = ACCENTS.flatMap((a) => MODES.map((m) => [a, m] as const));
const opaque = (e: Map<string, string>, t: string): RGBA => {
  const c = tokenColour(e, t);
  expect(c[3], `${t} must be opaque`).toBe(1);
  return c;
};

describe('G0 token block: scope (Classic untouched)', () => {
  it('the block is delimited in app/globals.css', () => {
    expect(GLOBALS_CSS).toContain('/* ── G0 tokens (lane 1c)');
    expect(GLOBALS_CSS).toContain('/* ── end G0 tokens ── */');
  });

  it('every --g-* declaration in globals.css is scoped under data-ui=glados', () => {
    const stray: string[] = [];
    for (const r of topLevelRules(CSS_NO_COMMENTS)) {
      if (!declarations(r.body).some(([k]) => k.startsWith('--g-'))) continue;
      for (const s of r.selectors) if (!s.includes("[data-ui='glados']")) stray.push(s);
    }
    // @media / @supports blocks are skipped by topLevelRules; check them by text too.
    for (const m of CSS_NO_COMMENTS.matchAll(/@(media|supports)[^{]*\{([\s\S]*?\})\s*\}/g))
      for (const inner of m[2].matchAll(/([^{}]+)\{([^{}]*)\}/g))
        if (/--g-[\w-]+\s*:/.test(inner[2]) && !inner[1].includes("[data-ui='glados']")) stray.push(inner[1].trim());
    expect(stray).toEqual([]);
  });

  it('every rule of the block starts with the G scope', () => {
    const start = GLOBALS_CSS.indexOf('/* ── G0 tokens (lane 1c)');
    const end = GLOBALS_CSS.indexOf('/* ── end G0 tokens ── */');
    const block = GLOBALS_CSS.slice(start, end).replace(/\/\*[\s\S]*?\*\//g, '');
    const rules = topLevelRules(block);
    expect(rules.length).toBeGreaterThan(0);
    for (const r of rules) for (const s of r.selectors) expect(s.startsWith(G_SCOPE), s).toBe(true);
  });

  it('a palette exists for exactly the registered accents', () => {
    const inCss = new Set([...CSS_NO_COMMENTS.matchAll(/:root\[data-ui='glados'\]\[data-accent='([\w-]+)'\]/g)].map((m) => m[1]));
    expect([...inCss].sort()).toEqual([...ACCENTS].sort());
  });
});

describe('G0 token coverage (AC00.10, ACGP3.6)', () => {
  for (const [a, m] of combos) {
    it(`every X2 token resolves under ${a} × ${m}`, () => {
      const e = env(a, m);
      const missing = ALL.filter((t) => {
        try {
          return resolve(e, `var(${t})`).trim() === '';
        } catch {
          return true;
        }
      });
      expect(missing).toEqual([]);
    });
  }

  it('without data-accent the palette falls back to ice in both modes', () => {
    for (const m of MODES) {
      const fallback = gCascade(null, m, RULES);
      for (const t of ALL) expect(fallback.get(t), `${t} (${m})`).toBe(env('ice', m).get(t));
    }
  });

  it('shape, type, space and motion carry the X2 values in every accent × mode', () => {
    for (const [a, m] of combos) for (const [t, v] of Object.entries(FIXED)) expect(env(a, m).get(t), `${t} ${a}/${m}`).toBe(v);
  });

  it('the elevation shadows carry no top-edge highlight (glass is lit from the bottom; review finding 6b)', () => {
    for (const [a, m] of combos) for (const t of ['--g-shadow', '--g-shadow-sm']) expect(env(a, m).get(t), `${t} ${a}/${m}`).not.toMatch(/inset/);
  });

  it('neutrals are palette independent (same in every accent of a mode)', () => {
    const neutral = [...TOKENS.surfaces, ...TOKENS.ink, ...TOKENS.lines, ...TOKENS.elevation, '--g-ok', '--g-warn', '--g-err'];
    for (const m of MODES) for (const a of ACCENTS) for (const t of neutral) expect(env(a, m).get(t), `${t} ${a}/${m}`).toBe(env(ACCENTS[0], m).get(t));
  });

  it('dark and light set color-scheme', () => {
    for (const a of ACCENTS) {
      expect(env(a, 'dark').get('color-scheme') ?? gColorScheme('dark')).toBe('dark');
      expect(env(a, 'light').get('color-scheme') ?? gColorScheme('light')).toBe('light');
    }
  });

  it('--g-accent-rgb is the --g-accent colour', () => {
    for (const [a, m] of combos) {
      const e = env(a, m);
      const c = tokenColour(e, '--g-accent');
      expect(resolve(e, 'var(--g-accent-rgb)').split(',').map((s) => Number(s.trim())), `${a}/${m}`).toEqual(c.slice(0, 3));
    }
  });

  it('categorical 1 to 6 are the accent series; 7 to 11 are accent independent', () => {
    for (const [a, m] of combos) {
      const e = env(a, m);
      for (let n = 1; n <= 6; n++) expect(resolve(e, `var(--g-cat-${n})`)).toBe(resolve(e, `var(--g-series-${n})`));
      for (let n = 7; n <= 11; n++) expect(e.get(`--g-cat-${n}`)).toBe(env(ACCENTS[0], m).get(`--g-cat-${n}`));
    }
  });

  it('hero gradients keep the same hues in both modes (§5.6)', () => {
    for (const a of ACCENTS) for (const v of VARIANTS) expect(env(a, 'light').get(`--g-hero-${v}`)).toBe(env(a, 'dark').get(`--g-hero-${v}`));
  });
});

/** color-scheme is not a custom property; read it from the block's rules directly. */
function gColorScheme(mode: 'dark' | 'light'): string | undefined {
  const sel = mode === 'dark' ? G_SCOPE : `${G_SCOPE}[data-mode='light']`;
  const r = topLevelRules(CSS_NO_COMMENTS).find((x) => x.selectors.includes(sel) && /color-scheme\s*:/.test(x.body));
  return r?.body.match(/color-scheme\s*:\s*(\w+)/)?.[1];
}

describe('G0 contrast (AC00.11, §3.3.5)', () => {
  const grounds = ['--g-bg', '--g-card', '--g-card-2', '--g-well'];

  for (const [a, m] of combos) {
    const e = env(a, m);
    it(`${a} × ${m}: ink, muted and status text ≥ 4.5:1 on every ground`, () => {
      for (const fg of ['--g-ink', '--g-muted', '--g-ok', '--g-warn', '--g-err'])
        for (const bg of grounds) expect(ratio(opaque(e, fg), opaque(e, bg)), `${fg} on ${bg}`).toBeGreaterThanOrEqual(4.5);
    });

    it(`${a} × ${m}: ink buttons, badge and soft-tag text ≥ 4.5:1`, () => {
      expect(ratio(opaque(e, '--g-ink-inverse'), opaque(e, '--g-ink')), 'ink-inverse on ink').toBeGreaterThanOrEqual(4.5);
      expect(ratio(opaque(e, '--g-badge-ink'), opaque(e, '--g-badge')), 'badge-ink on badge').toBeGreaterThanOrEqual(4.5);
      for (const tone of ['ok', 'warn', 'err'])
        for (const bg of ['--g-card', '--g-card-2']) {
          const fill = over(tokenColour(e, `--g-${tone}-soft`), opaque(e, bg));
          expect(ratio(opaque(e, `--g-${tone}`), fill), `${tone} on ${tone}-soft over ${bg}`).toBeGreaterThanOrEqual(4.5);
        }
    });

    it(`${a} × ${m}: series, cat, accent and states ≥ 3:1 on the card (non-text)`, () => {
      const marks = [...TOKENS.series, ...TOKENS.cat, '--g-accent', '--g-accent-2', '--g-state-ready', '--g-state-approved'];
      for (const t of marks) expect(ratio(opaque(e, t), opaque(e, '--g-card')), `${t} on --g-card`).toBeGreaterThanOrEqual(3);
    });

    it(`${a} × ${m}: categorical neighbours are distinct (ΔE ≥ 20)`, () => {
      for (let n = 1; n < 11; n++) {
        const d = deltaE(opaque(e, `--g-cat-${n}`), opaque(e, `--g-cat-${n + 1}`));
        expect(d, `cat-${n} vs cat-${n + 1}`).toBeGreaterThanOrEqual(20);
      }
    });

    it(`${a} × ${m}: diff text ≥ 4.5:1 on its tint over the card and the well; viz ink ≥ 3:1`, () => {
      for (const kind of ['del', 'add'])
        for (const bg of ['--g-card', '--g-well']) {
          const fill = over(tokenColour(e, `--g-diff-${kind}-bg`), opaque(e, bg));
          expect(ratio(opaque(e, `--g-diff-${kind}-fg`), fill), `${kind} over ${bg}`).toBeGreaterThanOrEqual(4.5);
        }
      const stage = opaque(e, '--g-viz-stage');
      expect(ratio(over(tokenColour(e, '--g-viz-ink'), stage), stage)).toBeGreaterThanOrEqual(3);
    });
  }
});

describe('G0 hero contrast (FR-19, AC00.12 at token level)', () => {
  for (const [a, m] of combos) {
    for (const v of VARIANTS) {
      it(`${a} × ${m} hero ${v}: caption and status ≥ 4.5:1, numeral ≥ 3:1 on every sampled pixel`, () => {
        const e = env(a, m);
        const bg = resolve(e, `var(--g-hero-${v})`);
        expect(() => parseBackground(bg)).not.toThrow();
        const ink = tokenColour(e, `--g-hero-${v}-ink`);
        const num = tokenColour(e, '--g-on-hero');
        expect(worstContrast(bg, ink, HERO_ZONES.caption), 'caption').toBeGreaterThanOrEqual(4.5);
        expect(worstContrast(bg, ink, HERO_ZONES.status), 'status').toBeGreaterThanOrEqual(4.5);
        expect(worstContrast(bg, num, HERO_ZONES.numeral), 'numeral').toBeGreaterThanOrEqual(3);
      });
    }
  }

  it('the sampler reproduces the measured failure it fixes (white on the old Ice top is 1.11:1)', () => {
    const oldIce = 'radial-gradient(120% 90% at 20% 0%, #eaf4ff 0%, rgba(234, 244, 255, 0) 55%), linear-gradient(180deg, #9fc6ff 0%, #5f86c9 45%, #27416b 100%)';
    expect(worstContrast(oldIce, [255, 255, 255, 1], { x: [0.2, 0.2], y: [0, 0] })).toBeCloseTo(1.11, 1);
    expect(worstContrast(oldIce, [255, 255, 255, 1], HERO_ZONES.caption)).toBeLessThan(4.5);
  });
});
