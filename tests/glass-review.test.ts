/**
 * The glass branch review (phase 4): focus, "on" glow, old surfaces and radii, on the glass canon
 * (the settings the app, founderos.sh and the content calendar share).
 *
 * Colours are resolved statically from app/globals.css for every appearance the app ships: GladOS
 * (four accents, dark and light), Classic (five themes) and Bennett Legacy (Monolith, Daylight), each
 * on the ground lib/appearance.ts groundFor gives it.
 */
import { describe, expect, test } from 'vitest';
import { groundFor } from '@/lib/appearance';
import { CSS_NO_COMMENTS, splitSelectors, topLevelRules } from './helpers/theme-css';
import { allRules, FACE_CLASSES } from './helpers/glass-lint';
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

export { LOOKS, colour, over, topLevelRules };
