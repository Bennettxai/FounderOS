/**
 * The Chrome audit scripts on the glass pack (they drive Chrome over CDP, so they never run here;
 * their pure parts do).
 *
 * - scripts/ground-check.mjs: a glass face is transparent by design (its frost and its light cover
 *   the ground), so a pixel on one is "covered", not a cut-off ground.
 * - The Content Lab blur budget (components/lab/v2/shared/tokens.ts GLASS_BUDGET) rose from 12 now
 *   that page glass takes the 3px light frost; g-audit's lab-glass-budget check reads the same cap.
 * - ground-check, theme-audit and screenshot-matrix force <html data-theme> after load; the glass light
 *   reads data-ground, so each switch sets it too, from lib/appearance.ts groundFor's own table
 *   (scripts/theme-ground.mjs).
 */
import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, test } from 'vitest';
import { GLASS_FACE_SELECTOR, coversPoint } from '../scripts/ground-lib.mjs';
import { LAB_GLASS_BUDGET } from '../scripts/g-audit/_lab.mjs';
import { GLASS_BUDGET } from '@/components/lab/v2/shared/tokens';
import { FACE_CLASSES } from './helpers/glass-lint';
import { GROUND_BY_THEME as SCRIPT_GROUNDS, groundFor as scriptGroundFor, setThemeExpr } from '../scripts/theme-ground.mjs';
import { GROUND_BY_THEME, LEGACY_THEMES, UI_STYLES, groundFor, type Mode, type UiStyle } from '@/lib/appearance';
import { THEMES } from '@/lib/theme';

const read = (rel: string) => fs.readFileSync(path.join(__dirname, '..', rel), 'utf8');
const CLEAR = { backgroundColor: 'rgba(0, 0, 0, 0)', backgroundImage: 'none', backdropFilter: 'none', tagName: 'DIV', isFace: false };

describe('ground-check: what covers a sampled point', () => {
  test('an opaque background, a background image or media covers it', () => {
    expect(coversPoint({ ...CLEAR, backgroundColor: 'rgb(12, 12, 12)' })).toBe(true);
    expect(coversPoint({ ...CLEAR, backgroundImage: 'linear-gradient(red, blue)' })).toBe(true);
    for (const tagName of ['IMG', 'VIDEO', 'CANVAS', 'svg']) expect(coversPoint({ ...CLEAR, tagName }), tagName).toBe(true);
  });

  test('a transparent element is not a cover (any zero alpha)', () => {
    expect(coversPoint(CLEAR)).toBe(false);
    expect(coversPoint({ ...CLEAR, backgroundColor: 'rgba(12, 34, 56, 0)' })).toBe(false);
    expect(coversPoint({ ...CLEAR, backgroundColor: 'transparent' })).toBe(false);
  });

  test('a transparent glass face covers it: its frost and its light are the face', () => {
    expect(coversPoint({ ...CLEAR, isFace: true })).toBe(true);
    expect(coversPoint({ ...CLEAR, backdropFilter: 'blur(3px) saturate(1.5)' })).toBe(true);
  });

  test('the face selector names every glass face class', () => {
    const named = GLASS_FACE_SELECTOR.split(/\s*,\s*/).map((s: string) => s.replace(/^\./, ''));
    expect([...named].sort()).toEqual([...FACE_CLASSES].sort());
  });

  test('ground-check runs the same function in the page', () => {
    const src = read('scripts/ground-check.mjs');
    expect(src).toMatch(/import \{[^}]*coversPoint[^}]*\} from '\.\/ground-lib\.mjs'/);
    expect(src).toContain('${coversPoint.toString()}');
    expect(src).toContain('e.matches(${JSON.stringify(GLASS_FACE_SELECTOR)})');
    expect(src).not.toMatch(/const opaque = \(cs\.backgroundColor/);
  });

  test('coversPoint closes over nothing (it is serialised into the page)', () => {
    const body = coversPoint.toString();
    expect(body).toMatch(/^function coversPoint\(/);
    expect(body).not.toMatch(/GLASS_FACE_SELECTOR|import|require\(/);
  });
});

describe('the Lab blur budget follows the light frost', () => {
  test('raised from 12 now that page glass frosts at 3px, with the reasoning next to the number', () => {
    expect(GLASS_BUDGET).toBe(24);
    const src = read('components/lab/v2/shared/tokens.ts');
    const at = src.indexOf('export const GLASS_BUDGET');
    const doc = src.slice(Math.max(0, src.lastIndexOf('/**', at)), at);
    expect(doc).toMatch(/3px/);
    expect(doc).toMatch(/20px/);
    expect(doc).toMatch(/12/);
  });

  test("g-audit's lab-glass-budget check reads the same cap (no literal 12)", () => {
    expect(LAB_GLASS_BUDGET).toBe(GLASS_BUDGET);
    const src = read('scripts/g-audit.mjs');
    expect(src).toMatch(/g\.blurred > LAB_GLASS_BUDGET/);
    expect(src).toMatch(/\(max \$\{LAB_GLASS_BUDGET\}\)/);
    expect(src).not.toMatch(/blurred > 12|max 12\)/);
  });
});

describe('the audit scripts set data-ground with every data-theme they force', () => {
  const SCRIPTS = ['scripts/ground-check.mjs', 'scripts/theme-audit.mjs', 'scripts/screenshot-matrix.mjs'];
  const MODES: Mode[] = ['dark', 'light'];
  const STYLES: (UiStyle | undefined)[] = [...UI_STYLES, undefined];
  const ALL_THEMES = [...THEMES, ...LEGACY_THEMES, 'not-a-theme'];

  test('the ground table is the one in lib/appearance.ts', () => {
    expect(SCRIPT_GROUNDS).toEqual(GROUND_BY_THEME);
  });

  test('groundFor agrees with lib/appearance.ts for every theme, mode and style', () => {
    for (const theme of ALL_THEMES)
      for (const mode of MODES)
        for (const ui of STYLES) expect(scriptGroundFor(theme, mode, ui, SCRIPT_GROUNDS), `${theme} ${mode} ${ui}`).toBe(groundFor(theme, mode, ui));
  });

  test('groundFor closes over nothing (it is serialised into the page)', () => {
    const body = scriptGroundFor.toString();
    expect(body).toMatch(/^function groundFor\(/);
    expect(body).not.toMatch(/GROUND_BY_THEME|import|require\(/);
  });

  test("the page expression sets data-theme and the matching data-ground from the page's own style and mode", () => {
    for (const theme of ALL_THEMES)
      for (const mode of MODES)
        for (const ui of UI_STYLES) {
          const attrs = new Map<string, string>([['data-ui', ui], ['data-mode', mode], ['data-theme', 'console'], ['data-ground', 'black']]);
          const documentElement = { getAttribute: (n: string) => attrs.get(n) ?? null, setAttribute: (n: string, v: string) => void attrs.set(n, v) };
          const ret = new Function('document', `return ${setThemeExpr(theme)}`)({ documentElement });
          expect(attrs.get('data-theme')).toBe(theme);
          expect(attrs.get('data-ground'), `${theme} ${mode} ${ui}`).toBe(groundFor(theme, mode, ui));
          expect(ret).toBe(attrs.get('data-ground'));
        }
  });

  test.each(SCRIPTS)('%s: every theme switch goes through setThemeExpr (no bare data-theme write)', (file) => {
    const src = read(file);
    expect(src).toMatch(/import \{[^}]*setThemeExpr[^}]*\} from '\.\/theme-ground\.mjs'/);
    expect(src).not.toMatch(/setAttribute\(\s*\\?['"`]data-theme/);
    expect(src).not.toMatch(/dataset\.theme\s*=/);
    expect(src).toMatch(/\bev\(setThemeExpr\(theme\)\)/);
  });

  test('no other script writes data-theme on its own either (every .mjs under scripts/)', () => {
    const walk = (d: string): string[] =>
      fs.readdirSync(path.join(__dirname, '..', d), { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(`${d}/${e.name}`) : e.name.endsWith('.mjs') ? [`${d}/${e.name}`] : []));
    const bare = walk('scripts')
      .filter((f) => f !== 'scripts/theme-ground.mjs')
      .filter((f) => /setAttribute\(\s*\\?['"`]data-theme|dataset\.theme\s*=/.test(read(f)));
    expect(bare).toEqual([]);
  });
});
