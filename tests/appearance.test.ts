import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, test } from 'vitest';
import {
  ACCENTS,
  ACCENT_IDS,
  APPEARANCE_COOKIE,
  APPEARANCE_INIT_SCRIPT,
  DEFAULT_APPEARANCE,
  G_READY,
  G_TREE_FILES,
  MODE_PREFS,
  OBSERVED_ATTRIBUTES,
  UI_STYLES,
  appearanceCookie,
  defaultAppearanceFor,
  classicMode,
  createAppearanceObserver,
  deriveClassicTheme,
  groundFor,
  htmlAttributes,
  isGReady,
  migrateLegacy,
  modeOverride,
  parseAppearanceCookie,
  readAppliedAppearance,
  resolveAppearance,
  resolveMode,
  serializeAppearance,
  usesGTree,
  LEGACY_THEMES,
  type AppearanceEnv,
  type Appearance,
  type Mode,
} from '@/lib/appearance';
import { THEMES, THEME_STORAGE_KEY } from '@/lib/theme';
import { readGReady } from '../scripts/g-audit/_lib.mjs';

const root = path.resolve(__dirname, '..');
const read = (f: string) => fs.readFileSync(path.join(root, f), 'utf8');

describe('model: cookie, defaults, derive, migrate (SPEC S00 FR-1 to FR-7)', () => {
  test('three axes with the registered ids', () => {
    expect(UI_STYLES).toEqual(['classic', 'glados', 'legacy']);
    expect(ACCENT_IDS).toEqual(['ice', 'lime', 'ember', 'mint']);
    expect(MODE_PREFS).toEqual(['dark', 'light', 'system']);
    expect(Object.keys(ACCENTS)).toEqual([...ACCENT_IDS]);
  });

  test('fresh-install default per build (FR-7, Q-SYS-1, Q-SYS-5; FounderOS ui merge T1)', () => {
    expect(defaultAppearanceFor('glados')).toEqual({ ui: 'glados', accent: 'ice', mode: 'dark' });
    expect(defaultAppearanceFor('founderos')).toEqual({ ui: 'legacy', accent: 'ice', mode: 'dark', skin: 'mono' });
    // tests run as the client build (GLADOS_BRAND unset)
    expect(DEFAULT_APPEARANCE).toEqual(defaultAppearanceFor('founderos'));
    expect(serializeAppearance(DEFAULT_APPEARANCE)).toBe('legacy.mono.dark');
  });

  test('cookie round trip for every combination', () => {
    for (const ui of UI_STYLES.filter((u) => u !== 'legacy'))
      for (const accent of ACCENT_IDS)
        for (const mode of MODE_PREFS) {
          const a: Appearance = { ui, accent, mode };
          expect(parseAppearanceCookie(serializeAppearance(a))).toEqual(a);
        }
    // under legacy the middle field is the skin; the accent waits in storage (withStoredChoices)
    for (const mode of MODE_PREFS) {
      const a: Appearance = { ui: 'legacy', accent: 'ice', mode, skin: 'mono' };
      expect(serializeAppearance(a)).toBe(`legacy.mono.${mode}`);
      expect(parseAppearanceCookie(serializeAppearance(a))).toEqual(a);
    }
    expect(serializeAppearance({ ui: 'glados', accent: 'ice', mode: 'dark' })).toBe('glados.ice.dark');
  });

  test('malformed fields fall back one by one (edge 2)', () => {
    expect(parseAppearanceCookie(null)).toEqual(DEFAULT_APPEARANCE);
    expect(parseAppearanceCookie('')).toEqual(DEFAULT_APPEARANCE);
    expect(parseAppearanceCookie('glados.pink.light')).toEqual({ ui: 'glados', accent: 'ice', mode: 'light' });
    expect(parseAppearanceCookie('classic.ember.auto')).toEqual({ ui: 'classic', accent: 'ember', mode: 'dark' });
    expect(parseAppearanceCookie('glados')).toEqual({ ui: 'glados', accent: 'ice', mode: 'dark' });
    // an unknown style takes this build's default style (Legacy on the client build), its middle field read for that style
    expect(parseAppearanceCookie('neon.ember.auto')).toEqual({ ui: 'legacy', accent: 'ice', mode: 'dark', skin: 'mono' });
    // Legacy: an accent or unknown skin in the middle falls back to mono
    expect(parseAppearanceCookie('legacy.ember.light')).toEqual({ ui: 'legacy', accent: 'ice', mode: 'light', skin: 'mono' });
    expect(parseAppearanceCookie('legacy.terminal.system')).toEqual({ ui: 'legacy', accent: 'ice', mode: 'system', skin: 'mono' });
  });

  test('cookie string: one year, Lax, root path, not HttpOnly', () => {
    const c = appearanceCookie({ ui: 'glados', accent: 'mint', mode: 'system' });
    expect(c).toBe(`${APPEARANCE_COOKIE}=glados.mint.system; Path=/; Max-Age=31536000; SameSite=Lax`);
    expect(c).not.toMatch(/HttpOnly/i);
  });

  test('deriveClassicTheme is the FR-3 table', () => {
    expect(deriveClassicTheme('ice', 'light')).toBe('light');
    expect(deriveClassicTheme('ember', 'light')).toBe('light');
    expect(deriveClassicTheme('lime', 'light')).toBe('light');
    expect(deriveClassicTheme('mint', 'light')).toBe('light');
    expect(deriveClassicTheme('ice', 'dark')).toBe('grey-frost');
    expect(deriveClassicTheme('ember', 'dark')).toBe('red');
    expect(deriveClassicTheme('lime', 'dark')).toBe('console');
    expect(deriveClassicTheme('mint', 'dark')).toBe('console');
  });

  test('AC00.3: migrateLegacy is the FR-6 table, missing and unknown take the default', () => {
    expect(migrateLegacy('console')).toEqual({ accent: 'lime', mode: 'dark' });
    expect(migrateLegacy('red')).toEqual({ accent: 'ember', mode: 'dark' });
    expect(migrateLegacy('light')).toEqual({ accent: 'lime', mode: 'light' });
    expect(migrateLegacy('frost')).toEqual({ accent: 'ice', mode: 'dark' });
    expect(migrateLegacy('grey-frost')).toEqual({ accent: 'ice', mode: 'dark' });
    expect(migrateLegacy(null)).toEqual({ accent: 'ice', mode: 'dark' });
    expect(migrateLegacy('glados-glass')).toEqual({ accent: 'ice', mode: 'dark' });
  });

  test('resolveMode, classicMode and ?mode= override', () => {
    expect(resolveMode('dark', true)).toBe('dark');
    expect(resolveMode('light', false)).toBe('light');
    expect(resolveMode('system', true)).toBe('light');
    expect(resolveMode('system', false)).toBe('dark');
    expect(classicMode('light')).toBe('light');
    for (const t of THEMES.filter((t) => t !== 'light')) expect(classicMode(t)).toBe('dark');
    expect(modeOverride('?mode=light')).toBe('light');
    expect(modeOverride('?x=1&mode=dark')).toBe('dark');
    expect(modeOverride('?mode=auto')).toBeNull();
    expect(modeOverride('')).toBeNull();
  });

  test('AC00.1: the server renders glados.ice.dark as the full attribute set', () => {
    expect(htmlAttributes(resolveAppearance(parseAppearanceCookie('glados.ice.dark')))).toEqual({
      'data-ui': 'glados',
      'data-accent': 'ice',
      'data-mode-pref': 'dark',
      'data-mode': 'dark',
      'data-theme': 'grey-frost',
      'data-ground': 'grey',
    });
  });

  test('groundFor: the ground every theme paints, which the glass light reads (glass brief §2)', () => {
    // Classic and Legacy follow the theme's own ground
    expect(groundFor('monolith', 'dark')).toBe('black');
    expect(groundFor('console', 'dark')).toBe('black');
    expect(groundFor('red', 'dark')).toBe('black');
    expect(groundFor('frost', 'dark')).toBe('black');
    expect(groundFor('grey-frost', 'dark')).toBe('grey');
    expect(groundFor('daylight', 'light')).toBe('white');
    expect(groundFor('light', 'light')).toBe('white');
    // a forced ?mode= does not repaint a Classic theme's ground
    expect(groundFor('red', 'light', 'classic')).toBe('black');
    // GladOS paints its own ground: grey in dark for every accent, white in light
    for (const accent of ACCENT_IDS) {
      expect(groundFor(deriveClassicTheme(accent, 'dark'), 'dark', 'glados'), accent).toBe('grey');
      expect(groundFor(deriveClassicTheme(accent, 'light'), 'light', 'glados'), accent).toBe('white');
    }
    // every theme the code defines has a ground
    for (const t of [...THEMES, ...LEGACY_THEMES]) expect(['black', 'grey', 'white'], t).toContain(groundFor(t, 'dark'));
  });

  test('htmlAttributes carries data-ground for every style', () => {
    expect(htmlAttributes(resolveAppearance(parseAppearanceCookie('legacy.mono.dark')))['data-ground']).toBe('black');
    expect(htmlAttributes(resolveAppearance(parseAppearanceCookie('legacy.mono.light')))['data-ground']).toBe('white');
    expect(htmlAttributes(resolveAppearance(parseAppearanceCookie('glados.ember.dark')))['data-ground']).toBe('grey');
    expect(htmlAttributes(resolveAppearance(parseAppearanceCookie('glados.lime.light')))['data-ground']).toBe('white');
    expect(htmlAttributes(resolveAppearance({ ui: 'classic', accent: 'ice', mode: 'dark' }, { classicTheme: 'grey-frost' }))['data-ground']).toBe('grey');
    expect(htmlAttributes(resolveAppearance({ ui: 'classic', accent: 'ice', mode: 'dark' }, { classicTheme: 'light' }))['data-ground']).toBe('white');
  });

  test('server: system renders dark (edge 4); Classic renders the default theme until the script reads storage', () => {
    expect(resolveAppearance({ ui: 'glados', accent: 'ember', mode: 'system' })).toMatchObject({ modePref: 'system', mode: 'dark', theme: 'red' });
    expect(resolveAppearance({ ui: 'classic', accent: 'ice', mode: 'light' })).toMatchObject({ modePref: 'dark', mode: 'dark', theme: 'console' });
    expect(resolveAppearance({ ui: 'classic', accent: 'ice', mode: 'dark' }, { classicTheme: 'light' })).toMatchObject({ mode: 'light', theme: 'light' });
  });

  test('readAppliedAppearance tolerates missing or junk attributes', () => {
    const el = (attrs: Record<string, string>) => ({ getAttribute: (k: string) => attrs[k] ?? null });
    expect(readAppliedAppearance(el({}))).toEqual({ ui: DEFAULT_APPEARANCE.ui, accent: 'ice', modePref: 'dark', mode: 'dark', theme: 'console' });
    expect(readAppliedAppearance(el({ 'data-ui': 'legacy', 'data-theme': 'daylight', 'data-mode': 'light' }))).toMatchObject({ ui: 'legacy', theme: 'daylight', mode: 'light' });
    expect(readAppliedAppearance(el({ 'data-ui': 'glados', 'data-accent': 'mint', 'data-mode-pref': 'system', 'data-mode': 'light', 'data-theme': 'light' }))).toEqual({
      ui: 'glados', accent: 'mint', modePref: 'system', mode: 'light', theme: 'light',
    });
  });

  test('the user-facing accent names carry no brand literal (§3.6)', () => {
    for (const a of Object.values(ACCENTS)) expect(`${a.name} ${a.blurb}`).not.toMatch(/glados/i);
  });
});

/* ── The head script, run against a fake browser ────────────────────────── */

type Fake = {
  attrs: Record<string, string>;
  cookieWrites: string[];
  storage: Map<string, string>;
  fireScheme: (light: boolean) => void;
};

function runScript(opts: {
  cookie?: string | null;
  storage?: Record<string, string>;
  prefersLight?: boolean;
  search?: string;
  storageThrows?: boolean;
  attrs?: Record<string, string>;
}): Fake {
  const attrs: Record<string, string> = { ...(opts.attrs ?? {}) };
  const storage = new Map(Object.entries(opts.storage ?? {}));
  const cookieWrites: string[] = [];
  const guard = () => {
    if (opts.storageThrows) throw new Error('blocked');
  };
  const localStorage = {
    getItem: (k: string) => (guard(), storage.has(k) ? storage.get(k)! : null),
    setItem: (k: string, v: string) => (guard(), storage.set(k, v)),
    removeItem: (k: string) => (guard(), storage.delete(k)),
  };
  const document = {
    get cookie() {
      return opts.cookie == null ? 'other=1' : `other=1; ${APPEARANCE_COOKIE}=${opts.cookie}; x=2`;
    },
    set cookie(v: string) {
      cookieWrites.push(v);
    },
    documentElement: {
      setAttribute: (k: string, v: string) => (attrs[k] = v),
      getAttribute: (k: string) => attrs[k] ?? null,
    },
  };
  let listener: (() => void) | null = null;
  const mq = {
    matches: !!opts.prefersLight,
    addEventListener: (_: string, l: () => void) => (listener = l),
  };
  const window = { matchMedia: () => mq };
  new Function('localStorage', 'document', 'window', 'location', APPEARANCE_INIT_SCRIPT)(localStorage, document, window, { search: opts.search ?? '' });
  return {
    attrs,
    cookieWrites,
    storage,
    fireScheme: (light) => {
      mq.matches = light;
      listener?.();
    },
  };
}

const attrsOf = (f: Fake) => ({
  'data-ui': f.attrs['data-ui'],
  'data-accent': f.attrs['data-accent'],
  'data-mode-pref': f.attrs['data-mode-pref'],
  'data-mode': f.attrs['data-mode'],
  'data-theme': f.attrs['data-theme'],
  'data-ground': f.attrs['data-ground'],
});

describe('APPEARANCE_INIT_SCRIPT (pre-paint)', () => {
  test('agrees with resolveAppearance for every ui × accent × mode × Classic theme × OS scheme × ?mode=', () => {
    let n = 0;
    for (const ui of UI_STYLES)
      for (const accent of ACCENT_IDS)
        for (const mode of MODE_PREFS)
          for (const theme of [...THEMES, null])
            for (const prefersLight of [false, true])
              for (const forced of [null, 'dark', 'light'] as (Mode | null)[]) {
                const a = { ui, accent, mode };
                const f = runScript({
                  cookie: serializeAppearance(a),
                  storage: theme ? { [THEME_STORAGE_KEY]: theme } : {},
                  prefersLight,
                  search: forced ? `?mode=${forced}` : '',
                });
                // under legacy the cookie carries the skin, not the accent: compare with what the cookie holds
                const want = htmlAttributes(resolveAppearance(parseAppearanceCookie(serializeAppearance(a)), { classicTheme: theme, prefersLight, forced }));
                expect(attrsOf(f), JSON.stringify({ a, theme, prefersLight, forced })).toEqual(want);
                n++;
              }
    expect(n).toBe(3 * 4 * 3 * 6 * 2 * 3);
  });

  test('a valid cookie is left alone and mirrored into storage', () => {
    const f = runScript({ cookie: 'glados.ember.light' });
    expect(f.cookieWrites).toEqual([]);
    expect(f.storage.get('glados-ui')).toBe('glados');
    expect(f.storage.get('glados-accent')).toBe('ember');
    expect(f.storage.get('glados-mode')).toBe('light');
  });

  test('a malformed cookie is rewritten with per-field fallbacks (edge 2)', () => {
    const f = runScript({ cookie: 'glados.pink.dusk' });
    expect(f.cookieWrites).toEqual([appearanceCookie({ ui: 'glados', accent: 'ice', mode: 'dark' })]);
    expect(f.attrs['data-accent']).toBe('ice');
  });

  test('no cookie: stored axes win once and the cookie is written (edge 3)', () => {
    const f = runScript({ storage: { 'glados-ui': 'glados', 'glados-accent': 'mint', 'glados-mode': 'light', [THEME_STORAGE_KEY]: 'red' } });
    expect(f.cookieWrites).toEqual([appearanceCookie({ ui: 'glados', accent: 'mint', mode: 'light' })]);
    expect(attrsOf(f)).toMatchObject({ 'data-ui': 'glados', 'data-accent': 'mint', 'data-mode': 'light', 'data-theme': 'light' });
  });

  test('migration (FR-6): an existing install keeps Classic, and its theme seeds the new-style accent and mode', () => {
    for (const theme of THEMES) {
      const f = runScript({ storage: { [THEME_STORAGE_KEY]: theme } });
      const m = migrateLegacy(theme);
      expect(f.cookieWrites, theme).toEqual([appearanceCookie({ ui: 'classic', accent: m.accent, mode: m.mode })]);
      expect(attrsOf(f), theme).toEqual({ 'data-ui': 'classic', 'data-accent': m.accent, 'data-mode-pref': classicMode(theme), 'data-mode': classicMode(theme), 'data-theme': theme, 'data-ground': groundFor(theme, classicMode(theme)) });
      expect(f.storage.get(THEME_STORAGE_KEY), 'Classic keeps its own key').toBe(theme);
    }
  });

  test("fresh install (no cookie, no storage): this build's default, cookie written with it (edge 1, FR-7, T1)", () => {
    const f = runScript({});
    expect(f.cookieWrites).toEqual([appearanceCookie(DEFAULT_APPEARANCE)]);
    // the client build opens on Bennett Legacy · Monolith · Dark
    expect(attrsOf(f)).toEqual({ 'data-ui': 'legacy', 'data-accent': 'ice', 'data-mode-pref': 'dark', 'data-mode': 'dark', 'data-theme': 'monolith', 'data-ground': 'black' });
    expect(f.storage.get('glados-legacy-skin')).toBe('mono');
  });

  test('blocked storage: attributes still apply from the cookie (edge 1)', () => {
    const f = runScript({ cookie: 'glados.lime.light', storageThrows: true });
    expect(attrsOf(f)).toEqual({ 'data-ui': 'glados', 'data-accent': 'lime', 'data-mode-pref': 'light', 'data-mode': 'light', 'data-theme': 'light', 'data-ground': 'white' });
  });

  test('ae-mode is deleted and never promoted (FR-10, Q-SYS-4)', () => {
    const f = runScript({ cookie: 'glados.ice.dark', storage: { 'ae-mode': 'light' } });
    expect(f.storage.has('ae-mode')).toBe(false);
    expect(f.attrs['data-mode']).toBe('dark');
  });

  test('AC00.20: ?mode= applies for the load and is never stored', () => {
    const f = runScript({ cookie: 'glados.ice.dark', search: '?mode=light' });
    expect(attrsOf(f)).toMatchObject({ 'data-mode-pref': 'dark', 'data-mode': 'light', 'data-theme': 'light' });
    expect(f.cookieWrites).toEqual([]);
    expect(f.storage.get('glados-mode')).toBe('dark');
    const c = runScript({ cookie: 'classic.ice.dark', storage: { [THEME_STORAGE_KEY]: 'red' }, search: '?mode=light' });
    expect(attrsOf(c)).toMatchObject({ 'data-ui': 'classic', 'data-mode': 'light', 'data-theme': 'red' });
  });

  test('FR-5: System follows the OS live, and stops once the preference is not System', () => {
    const f = runScript({ cookie: 'glados.ember.system', prefersLight: false });
    expect(attrsOf(f)).toMatchObject({ 'data-mode-pref': 'system', 'data-mode': 'dark', 'data-theme': 'red', 'data-ground': 'grey' });
    f.fireScheme(true);
    expect(attrsOf(f)).toMatchObject({ 'data-mode': 'light', 'data-theme': 'light', 'data-ground': 'white' });
    f.fireScheme(false);
    expect(attrsOf(f)).toMatchObject({ 'data-mode': 'dark', 'data-theme': 'red', 'data-ground': 'grey' });
    const l = runScript({ cookie: 'legacy.mono.system', prefersLight: false });
    expect(attrsOf(l)).toMatchObject({ 'data-theme': 'monolith', 'data-ground': 'black' });
    l.fireScheme(true);
    expect(attrsOf(l)).toMatchObject({ 'data-theme': 'daylight', 'data-ground': 'white' });
    f.attrs['data-mode-pref'] = 'dark'; // Settings switched to Dark later
    f.fireScheme(true);
    expect(f.attrs['data-mode']).toBe('dark');
  });
});

/* ── X3: one observer ─────────────────────────────────────────────────────── */

describe('onAppearanceChange (X3)', () => {
  function fakeEnv() {
    const attrs: Record<string, string> = { 'data-ui': 'glados', 'data-accent': 'ice', 'data-mode-pref': 'system', 'data-mode': 'dark', 'data-theme': 'grey-frost' };
    const observers: { cb: () => void; opts?: MutationObserverInit; live: boolean }[] = [];
    const mqListeners = new Set<() => void>();
    class MO {
      rec: { cb: () => void; opts?: MutationObserverInit; live: boolean };
      constructor(cb: () => void) {
        this.rec = { cb, live: false };
        observers.push(this.rec);
      }
      observe(_: unknown, opts: MutationObserverInit) {
        this.rec.opts = opts;
        this.rec.live = true;
      }
      disconnect() {
        this.rec.live = false;
      }
    }
    const env: AppearanceEnv = {
      root: { getAttribute: (k) => attrs[k] ?? null },
      MutationObserver: MO,
      matchMedia: () => ({ matches: false, addEventListener: (_t, l) => mqListeners.add(l), removeEventListener: (_t, l) => mqListeners.delete(l) }),
    };
    const set = (k: string, v: string) => {
      attrs[k] = v;
      for (const o of observers) if (o.live && o.opts?.attributeFilter?.includes(k)) o.cb();
    };
    return { env, observers, mqListeners, set };
  }

  test('one MutationObserver for many subscribers, on the four attributes, plus the scheme query', () => {
    const f = fakeEnv();
    const obs = createAppearanceObserver(() => f.env);
    const seenA: string[] = [];
    const seenB: string[] = [];
    const offA = obs.subscribe((a) => seenA.push(`${a.accent}/${a.mode}`));
    const offB = obs.subscribe((a) => seenB.push(a.theme));
    expect(f.observers).toHaveLength(1);
    expect(f.observers[0].opts?.attributeFilter).toEqual([...OBSERVED_ATTRIBUTES]);
    expect(OBSERVED_ATTRIBUTES).toEqual(['data-ui', 'data-accent', 'data-mode', 'data-theme']);
    expect(f.mqListeners.size).toBe(1);

    f.set('data-accent', 'ember');
    f.set('data-theme', 'red');
    expect(seenA).toEqual(['ember/dark', 'ember/dark']);
    expect(seenB).toEqual(['grey-frost', 'red']);

    for (const l of f.mqListeners) l();
    expect(seenA).toHaveLength(3);

    offA();
    expect(f.observers[0].live).toBe(true);
    offB();
    expect(f.observers[0].live).toBe(false);
    expect(f.mqListeners.size).toBe(0);
    expect(obs.size()).toBe(0);

    obs.subscribe(() => {});
    expect(f.observers).toHaveLength(2);
  });

  test('on the server (no env) subscribing is a no-op', () => {
    const obs = createAppearanceObserver(() => null);
    const off = obs.subscribe(() => {});
    expect(typeof off).toBe('function');
    off();
  });
});

/* ── Rollout registry and wiring ─────────────────────────────────────────── */

describe('G_READY registry (FR-14)', () => {
  test('every converted route registers its files, and the audit reads the same registry from lib/g-ready.ts', () => {
    // Lane 2c (GP1) shipped the Creative Engine; page lanes add theirs at lane exit (C37).
    expect(G_READY).toContain('/creative-engine');
    for (const r of G_READY) expect(G_TREE_FILES[r]?.length ?? 0, r).toBeGreaterThan(0);
    const reg = readGReady(root);
    expect(reg.source).toBe('lib/g-ready.ts');
    expect(reg.routes).toEqual(G_READY);
    expect(reg.files).toEqual(G_TREE_FILES);
  });

  test('isGReady and usesGTree', () => {
    expect(isGReady('/funnel?x=1', ['/funnel'])).toBe(true);
    expect(isGReady('/funnel#a', ['/funnel'])).toBe(true);
    expect(isGReady('/funnel/x', ['/funnel'])).toBe(false);
    expect(isGReady('/agents/setup/drake', ['/agents/setup/:id'])).toBe(true);
    expect(isGReady('/agents/setup/:id', ['/agents/setup/:id'])).toBe(true);
    expect(isGReady('/agents/setup', ['/agents/setup/:id'])).toBe(false);
    expect(usesGTree({ ui: 'glados' }, '/funnel', ['/funnel'])).toBe(true);
    expect(usesGTree({ ui: 'classic' }, '/funnel', ['/funnel'])).toBe(false);
    expect(usesGTree({ ui: 'glados' }, '/no-such-route')).toBe(false); // the default registry (lane R put /funnel in it)
  });
});

describe('layout wiring (FR-2)', () => {
  const layout = read('app/layout.tsx');

  test('the server renders the attributes from the cookie and injects one appearance script', () => {
    // read once, shared with the shell pieces (lane 2b passes `ui` to them)
    expect(layout).toMatch(/const appearance = getAppearance\(\);/);
    expect(layout).toMatch(/\{\.\.\.htmlAttributes\(appearance\)\}/);
    expect(layout).toContain('__html: APPEARANCE_INIT_SCRIPT');
    expect(layout).not.toContain('THEME_INIT_SCRIPT');
    expect(read('lib/theme.ts')).not.toMatch(/export const THEME_INIT_SCRIPT/);
  });

  test('the ground rides along wherever the theme is written by hand (glass brief §2)', () => {
    // htmlAttributes carries it for the server render, AppearanceSync and writeAppearance; the Classic picker sets data-theme itself
    expect(read('components/AppearanceSync.tsx')).toContain('htmlAttributes(r)');
    expect(read('components/ThemePicker.tsx')).toMatch(/setAttribute\('data-ground', groundFor\(/);
  });

  test('next/headers stays in the server module, out of anything a client component imports', () => {
    expect(read('lib/appearance-server.ts')).toContain("from 'next/headers'");
    for (const f of ['lib/appearance.ts', 'lib/use-appearance.ts', 'lib/use-palette.ts', 'lib/g-ready.ts']) expect(read(f), f).not.toMatch(/from ['"]next\/headers['"]/);
  });
});
