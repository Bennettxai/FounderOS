/**
 * Content Lab v2 token block and static contrast (02 section 15.2): AC-02-1 to AC-02-10, AC-02-16, AC-02-17.
 */
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { badShadows } from '../scripts/g-audit/_lib.mjs';
import { GLOBALS_CSS } from './helpers/theme-css';
import { LAB_STYLES, evalColour, labCascade, labColour, labResolveText, over, ratio, styleName, toHex, type RGBA } from './helpers/lab-colour';

const V2_START = '/* ── Content Lab v2 · Aurora tokens';
const V2_END = '/* ── end Content Lab v2 · Aurora tokens ── */';
const block = GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(V2_START), GLOBALS_CSS.indexOf(V2_END));

/** Every token the 2.1 block declares. */
const DECLARED = [...new Set([...block.matchAll(/(--lab-[\w-]+)\s*:/g)].map((m) => m[1]))];
const COLOUR_TOKENS = [
  '--lab-aurora-hot', '--lab-aurora-warm', '--lab-aurora-deep', '--lab-aurora-cool', '--lab-aurora-anchor',
  '--lab-glass', '--lab-glass-strong', '--lab-glass-solid', '--lab-glass-tint',
  '--lab-ink-hero', '--lab-ink-2', '--lab-on-ink', '--lab-shade', '--lab-dot-on', '--lab-dot-off',
  '--lab-up', '--lab-down', '--lab-warn', '--lab-scrim', '--lab-focus', '--lab-narr-ink', '--lab-narr-paper', '--lab-narr-bed',
];
const pct = (env: Map<string, string>, t: string) => Number(String(labResolve(env, t)).replace('%', '')) / 100;
function labResolve(env: Map<string, string>, t: string): string {
  let v = env.get(t) ?? '';
  for (let i = 0; i < 6 && v.startsWith('var('); i++) v = env.get(/var\((--[\w-]+)/.exec(v)![1]) ?? '';
  return v;
}
const withAlpha = (c: RGBA, a: number): RGBA => [c[0], c[1], c[2], c[3] * a];

describe('AC-02-1 the v2 token block', () => {
  it('sits after the v1 token block and declares the shared contract names', () => {
    expect(GLOBALS_CSS.indexOf(V2_START)).toBeGreaterThan(GLOBALS_CSS.indexOf('/* ── end Content Lab tokens ── */'));
    for (const t of ['--lab-aurora-hot', '--lab-aurora-warm', '--lab-aurora-deep', '--lab-aurora-cool', '--lab-glass', '--lab-glass-strong', '--lab-glass-blur', '--lab-ink-hero', '--lab-dot-on', '--lab-dot-off']) {
      expect(DECLARED).toContain(t);
    }
    for (const t of ['--lab-s1', '--lab-s9', '--lab-r-tile', '--lab-r-narrative', '--lab-fs-hero', '--lab-narr-ink', '--lab-glass-edge', '--lab-glass-shadow']) expect(DECLARED).toContain(t);
  });
});

describe.each(LAB_STYLES.map((s) => [styleName(s), s] as const))('%s', (_name, style) => {
  const env = labCascade(style);
  const c = (t: string) => labColour(env, t);

  it('AC-02-2 resolves every colour token without undefined vars', () => {
    for (const t of COLOUR_TOKENS) expect(() => c(t), t).not.toThrow();
    for (const t of DECLARED) expect(env.has(t), t).toBe(true);
  });

  it('AC-02-4 ink hero >= 4.5:1 on deep, on the warm haze and at the glow edge', () => {
    const deep = c('--lab-aurora-deep');
    const ink = c('--lab-ink-hero');
    const haze = over(withAlpha(c('--lab-aurora-warm'), pct(env, '--lab-aurora-haze')), deep);
    const glow = over(withAlpha(c('--lab-aurora-hot'), pct(env, '--lab-aurora-glow') * 0.4), deep);
    expect(ratio(ink, deep)).toBeGreaterThanOrEqual(4.5);
    expect(ratio(ink, haze)).toBeGreaterThanOrEqual(4.5);
    expect(ratio(ink, glow)).toBeGreaterThanOrEqual(4.5);
  });

  it('AC-02-5 ink 2 and ink hero >= 4.5:1 on strong glass over cool and over deep', () => {
    for (const ground of ['--lab-aurora-cool', '--lab-aurora-deep']) {
      const bg = over(c('--lab-glass-strong'), c(ground));
      expect(ratio(c('--lab-ink-2'), bg), `ink-2 on strong over ${ground}`).toBeGreaterThanOrEqual(4.5);
      expect(ratio(c('--lab-ink-hero'), bg), `ink-hero on strong over ${ground}`).toBeGreaterThanOrEqual(4.5);
    }
    expect(ratio(c('--lab-on-ink'), c('--lab-ink-hero'))).toBeGreaterThanOrEqual(4.5);
  });

  it('AC-02-6 narrative ink >= 4.5:1 on its text beds', () => {
    if (style.ui === 'classic' && style.theme === 'console') {
      expect(ratio(c('--text'), c('--surface'))).toBeGreaterThanOrEqual(4.5);
      return;
    }
    const tint = pct(env, '--lab-narr-tint');
    const paper = c('--lab-narr-paper');
    const mix = (a: RGBA, p: number, b: RGBA): RGBA => evalColour(`color-mix(in srgb, ${toHex(a)} ${p * 100}%, ${toHex(b)})`);
    const nA = mix(c('--lab-aurora-warm'), tint, paper);
    const nB = mix(c('--lab-aurora-hot'), tint, paper);
    const bed = c('--lab-narr-bed');
    expect(ratio(c('--lab-narr-ink'), over(bed, nA))).toBeGreaterThanOrEqual(4.5);
    expect(ratio(c('--lab-narr-ink'), over(bed, nB))).toBeGreaterThanOrEqual(4.5);
  });

  it('AC-02-7 lit dots >= 3:1 on deep and on strong glass', () => {
    expect(ratio(c('--lab-dot-on'), c('--lab-aurora-deep'))).toBeGreaterThanOrEqual(3);
    expect(ratio(c('--lab-dot-on'), over(c('--lab-glass-strong'), c('--lab-aurora-cool')))).toBeGreaterThanOrEqual(3);
  });

  if (style.ui === 'classic' && ['red', 'frost', 'grey-frost'].includes(style.theme)) {
    it('AC-02-8 v2 colours are green-free', () => {
      for (const t of COLOUR_TOKENS.filter((x) => !['--lab-up', '--lab-down', '--lab-warn'].includes(x)).concat(['--lab-up'])) {
        const [r, g, b, a] = c(t);
        if (a < 0.05) continue;
        expect(g > r + 24 && g > b + 24, `${t} ${toHex([r, g, b, 1])}`).toBe(false);
      }
    });
  }

  if (style.ui === 'glados') {
    it('AC-02-9 glass has no border and a shadow badShadows() accepts', () => {
      expect(labResolveText(env, '--lab-glass-border')).toMatch(/^0\b/);
      expect(badShadows(labResolveText(env, '--lab-glass-shadow'))).toEqual([]);
    });
  }
});

describe('AC-02-3 resolved stops match table 2.2 within 2 per channel', () => {
  const TABLE: [string, Record<string, string>][] = [
    ['glados ember dark', { deep: '#5d262e', cool: '#43474e', warm: '#ff865c', hot: '#ff4557' }],
    ['glados ice dark', { cool: '#43474e', hot: '#9fc6ff' }],
    ['glados ice light', { hot: '#3a6fc4' }],
    ['classic console', { hot: '#00ffab' }],
  ];
  for (const [name, want] of TABLE) {
    it(name, () => {
      const env = labCascade(LAB_STYLES.find((s) => styleName(s) === name)!);
      for (const [k, hex] of Object.entries(want)) {
        const got = labColour(env, `--lab-aurora-${k}`);
        const exp = evalColour(hex);
        for (let i = 0; i < 3; i++) expect(Math.abs(got[i] - exp[i]), `${name} ${k} ${toHex(got)} vs ${hex}`).toBeLessThanOrEqual(2);
      }
    });
  }
});

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });
}
const V2_FILES = walk(join(process.cwd(), 'components/lab/v2'));

describe('AC-02-10, AC-02-16, AC-02-17 source scans of components/lab/v2', () => {
  it('has no raw colour and no emoji', () => {
    for (const f of V2_FILES) {
      const src = readFileSync(f, 'utf8');
      expect(src, f).not.toMatch(/#[0-9a-fA-F]{3,8}\b(?![\w-])/);
      expect(src, f).not.toMatch(/\b(rgba?|hsla?|oklch)\(/);
      expect(src, f).not.toMatch(/\p{Extended_Pictographic}/u);
      expect(src, f).not.toMatch(/[–—]/);
    }
  });
  it('declares no number formatter, no LabSheetKind and no second SheetKind', () => {
    for (const f of V2_FILES) {
      const src = readFileSync(f, 'utf8');
      expect(src, f).not.toMatch(/export (function|const) (fmt|format)\w*/);
      expect(src, f).not.toMatch(/\b(LabSheetKind|formatValue|formatDelta|nLabel)\b/);
      expect(src, f).not.toMatch(/type SheetKind\b/);
      expect(src, f).not.toMatch(/toFixed\(|toLocaleString\(|Intl\.NumberFormat/);
    }
  });
});

describe('glass phase 3: Lab glass frosts in every style (the brief: light 3px/150%, strong 28px/160%)', () => {
  const FROST = { '--lab-glass-blur': '3px', '--lab-glass-sat': '150%', '--lab-glass-blur-strong': '28px', '--lab-glass-sat-strong': '160%' };

  it.each(LAB_STYLES.map((s) => [styleName(s), s] as const))('%s: the light and strong frost tokens are the brief values', (_n, style) => {
    const env = labCascade(style);
    for (const [t, v] of Object.entries(FROST)) expect(labResolveText(env, t), t).toBe(v);
  });

  it('no style (Legacy and Console included) sets a Lab blur of 0 outside the reduced-transparency fallback', () => {
    const flat = GLOBALS_CSS.replace(/\/\*[\s\S]*?\*\//g, '');
    const outside = flat.replace(/@media \(prefers-reduced-transparency: reduce\) \{[\s\S]*?\}\s*\}/g, '').replace(/@supports not [^{]*\{[\s\S]*?\}\s*\}/g, '');
    const decls = [...outside.matchAll(/(--lab-glass-(?:blur|sat)(?:-strong)?)\s*:\s*([^;]+);/g)].map((m) => `${m[1]}: ${m[2].trim()}`);
    expect(decls.length).toBeGreaterThan(0);
    for (const d of decls) expect(Object.entries(FROST).map(([k, v]) => `${k}: ${v}`), d).toContain(d);
    // the Legacy Lab block re-keys colours only; it inherits the frost
    const legacy = [...flat.matchAll(/:root\[data-ui='legacy'\] \.lab \{([^}]*)\}/g)].map((m) => m[1]).join('');
    expect(legacy).not.toMatch(/--lab-glass-(blur|sat)/);
  });

  it('console blurs too: no rule switches Lab glass frost off by theme', () => {
    expect(GLOBALS_CSS.replace(/\/\*[\s\S]*?\*\//g, '')).not.toMatch(/\[data-theme='console'\] \.lab2-glass \{[^}]*backdrop-filter: none/);
  });

  it('frosted Lab glass uses the light frost; strong glass the strong frost', () => {
    const flat = GLOBALS_CSS.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ');
    expect(flat).toMatch(/\.lab2-glass\[data-blur='on'\] \{ backdrop-filter: blur\(var\(--lab-glass-blur\)\) saturate\(var\(--lab-glass-sat\)\);/);
    expect(flat).toMatch(/\.lab2-glass\[data-blur='on'\]\[data-tone='strong'\] \{ backdrop-filter: blur\(var\(--lab-glass-blur-strong\)\) saturate\(var\(--lab-glass-sat-strong\)\);/);
  });

  it('the solid fallback stays: reduced transparency and no backdrop-filter zero the blur; the cap stays 24', () => {
    const flat = GLOBALS_CSS.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\s+/g, ' ');
    expect(flat).toMatch(/@media \(prefers-reduced-transparency: reduce\) \{ \.lab \{ [^}]*--lab-glass-blur: 0px;/);
    expect(flat).toMatch(/@supports not \(\(backdrop-filter: blur\(1px\)\) or \(-webkit-backdrop-filter: blur\(1px\)\)\) \{ \.lab \{ [^}]*--lab-glass-blur: 0px;/);
    expect(readFileSync(join(process.cwd(), 'components/lab/v2/shared/tokens.ts'), 'utf8')).toMatch(/export const GLASS_BUDGET = 24;/);
  });
});
