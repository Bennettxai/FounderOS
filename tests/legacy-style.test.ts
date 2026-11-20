import fs from 'node:fs';
import path from 'node:path';
import { createElement } from 'react';
import { describe, expect, test } from 'vitest';
import { PathnameContext } from 'next/dist/shared/lib/hooks-client-context.shared-runtime';
import {
  APPEARANCE_COOKIE,
  APPEARANCE_INIT_SCRIPT,
  G_READY,
  L_READY,
  LEGACY_THEMES,
  STORAGE_KEYS,
  STYLE_LABELS,
  appearanceCookie,
  htmlAttributes,
  isLReady,
  parseAppearanceCookie,
  resolveAppearance,
  serializeAppearance,
  switchStyle,
  treeFor,
  withStoredChoices,
  type Appearance,
} from '@/lib/appearance';
import { L_NO_SLAB, isLegacyWorkspace } from '@/lib/l-ready';
import { LegacySlab, legacyEyebrow } from '@/components/legacy/LegacySlab';
import { LegacyKitSpecimen } from '@/app/dev/l-kit/LegacyKitSpecimen';
import { dotColumns, dotState, insightTicks, meterWidth, stepGeometry } from '@/components/legacy/format';
import { renderWithRouter } from './helpers/render-client';
import { readLReady } from '../scripts/l-audit/_lib.mjs';

/**
 * Bennett Legacy, the third UI style (docs/projects/founderos-ui/SPEC.md §2.5,
 * SCAFFOLD §3.2-3.3, pages/setup.md): tree picker, cookie round trip (AC-5),
 * the --l-* tokens, the font, the kit and the slab frame.
 */

const root = path.resolve(__dirname, '..');
const read = (f: string) => fs.readFileSync(path.join(root, f), 'utf8');
const CSS = read('app/globals.css');
const L0 = CSS.slice(CSS.indexOf('L0 BENNETT LEGACY'), CSS.indexOf('end L0 BENNETT LEGACY'));
const stripComments = (s: string) => s.replace(/\/\*[\s\S]*?\*\//g, '');

describe('treeFor (X5 generalised)', () => {
  test('glados → G tree on G_READY routes, legacy → Legacy tree on L_READY routes, else classic', () => {
    expect(treeFor({ ui: 'glados' }, '/funnel', { g: ['/funnel'], l: [] })).toBe('glados');
    expect(treeFor({ ui: 'glados' }, '/trading', { g: ['/funnel'], l: ['/trading'] })).toBe('classic');
    expect(treeFor({ ui: 'legacy' }, '/trading', { g: ['/funnel'], l: ['/trading'] })).toBe('legacy');
    expect(treeFor({ ui: 'legacy' }, '/funnel', { g: ['/funnel'], l: [] })).toBe('classic'); // Classic tree under the Legacy skin (O1)
    expect(treeFor({ ui: 'classic' }, '/funnel', { g: ['/funnel'], l: ['/funnel'] })).toBe('classic');
    expect(treeFor({ ui: 'legacy' }, '/agents/setup/drake', { l: ['/agents/setup/:id'] })).toBe('legacy');
    expect(treeFor({ ui: 'legacy' }, '/trading?x=1', { l: ['/trading'] })).toBe('legacy');
  });

  test('default registries: L_READY holds the kit specimen and the five FounderOS pages and the GladOS-only pages with their own Legacy tree; every G_READY route still forks to GladOS', () => {
    expect(L_READY).toEqual(expect.arrayContaining(['/dev/l-kit', '/trading', '/usage', '/clients', '/lead-magnets', '/chats', '/wizard', '/login']));
    expect(isLReady('/dev/l-kit')).toBe(true);
    for (const r of G_READY.filter((x) => !x.includes(':'))) {
      expect(treeFor({ ui: 'glados' }, r), r).toBe('glados');
      expect(treeFor({ ui: 'legacy' }, r), r).toBe(isLReady(r) ? 'legacy' : 'classic');
    }
    expect(treeFor({ ui: 'legacy' }, '/dev/l-kit')).toBe('legacy');
  });

  test('scripts/l-audit reads the same L_READY literal', () => {
    expect(readLReady(root).routes).toEqual(L_READY);
  });

  test('every app page.tsx that forked on usesGTree now forks on treeFor (the Lab stays untouched)', () => {
    const pages: string[] = [];
    const walk = (d: string) => {
      for (const e of fs.readdirSync(path.join(root, d), { withFileTypes: true })) {
        const rel = `${d}/${e.name}`;
        if (e.isDirectory()) walk(rel);
        else if (/\.tsx$/.test(e.name)) pages.push(rel);
      }
    };
    walk('app');
    const forks = pages.filter((p) => !p.startsWith('app/lab/') && /usesGTree|treeFor\(/.test(read(p)));
    expect(forks.length).toBeGreaterThanOrEqual(28);
    for (const p of forks) expect(read(p), p).not.toMatch(/usesGTree/);
  });
});

describe('Legacy appearance: themes, cookie, AC-5 round trip', () => {
  test('mono skin wears Monolith in dark and Daylight in light; system follows the OS; ?mode= forces', () => {
    const a: Appearance = { ui: 'legacy', accent: 'ice', mode: 'dark', skin: 'mono' };
    expect(htmlAttributes(resolveAppearance(a))).toEqual({ 'data-ui': 'legacy', 'data-accent': 'ice', 'data-mode-pref': 'dark', 'data-mode': 'dark', 'data-theme': 'monolith', 'data-ground': 'black' });
    expect(resolveAppearance({ ...a, mode: 'light' }).theme).toBe('daylight');
    expect(resolveAppearance({ ...a, mode: 'system' }, { prefersLight: true })).toMatchObject({ modePref: 'system', mode: 'light', theme: 'daylight' });
    expect(resolveAppearance({ ...a, mode: 'system' })).toMatchObject({ mode: 'dark', theme: 'monolith' }); // server: system renders dark
    expect(resolveAppearance(a, { forced: 'light' }).theme).toBe('daylight');
    expect(LEGACY_THEMES).toEqual(['monolith', 'daylight']);
  });

  test('labels: GladOS is the product name, Legacy is "Bennett Legacy" (T1)', () => {
    expect(STYLE_LABELS.legacy).toBe('Bennett Legacy');
    expect(STYLE_LABELS.classic).toBe('Classic');
  });

  test('AC-5: GladOS → Legacy → GladOS keeps the accent; Legacy → Classic → Legacy keeps the skin', () => {
    const start = withStoredChoices(parseAppearanceCookie('glados.ember.light'), { skin: null });
    expect(start).toEqual({ ui: 'glados', accent: 'ember', mode: 'light', skin: 'mono' });
    const legacy = switchStyle(start, 'legacy');
    expect(serializeAppearance(legacy)).toBe('legacy.mono.light');
    // the cookie alone loses the accent; storage (glados-accent) brings it back
    const reread = withStoredChoices(parseAppearanceCookie(serializeAppearance(legacy)), { accent: 'ember' });
    expect(reread.accent).toBe('ember');
    expect(serializeAppearance(switchStyle(reread, 'glados'))).toBe('glados.ember.light');
    const classic = switchStyle(reread, 'classic');
    expect(serializeAppearance(classic)).toBe('classic.ember.light');
    const back = withStoredChoices(parseAppearanceCookie(serializeAppearance(classic)), { skin: 'mono' });
    expect(serializeAppearance(switchStyle(back, 'legacy'))).toBe('legacy.mono.light');
  });

  function runScript(cookie: string | null, store: Record<string, string> = {}) {
    const attrs: Record<string, string> = {};
    const storage = new Map(Object.entries(store));
    const writes: string[] = [];
    const localStorage = { getItem: (k: string) => storage.get(k) ?? null, setItem: (k: string, v: string) => storage.set(k, v), removeItem: (k: string) => storage.delete(k) };
    const document = {
      get cookie() {
        return cookie == null ? '' : `${APPEARANCE_COOKIE}=${cookie}`;
      },
      set cookie(v: string) {
        writes.push(v);
      },
      documentElement: { setAttribute: (k: string, v: string) => (attrs[k] = v), getAttribute: (k: string) => attrs[k] ?? null },
    };
    let listener: (() => void) | null = null;
    const mq = { matches: false, addEventListener: (_: string, l: () => void) => (listener = l) };
    new Function('localStorage', 'document', 'window', 'location', APPEARANCE_INIT_SCRIPT)(localStorage, document, { matchMedia: () => mq }, { search: '' });
    return { attrs, storage, writes, fire: (light: boolean) => ((mq.matches = light), listener?.()) };
  }

  test('pre-paint script: a Legacy cookie mirrors the skin and leaves the stored GladOS accent alone', () => {
    const f = runScript('legacy.mono.dark', { [STORAGE_KEYS.accent]: 'mint' });
    expect(f.attrs['data-ui']).toBe('legacy');
    expect(f.attrs['data-theme']).toBe('monolith');
    expect(f.storage.get(STORAGE_KEYS.skin)).toBe('mono');
    expect(f.storage.get(STORAGE_KEYS.accent)).toBe('mint');
    expect(f.writes).toEqual([]);
  });

  test('pre-paint script: a Legacy cookie with an accent in the middle is rewritten with the skin', () => {
    const f = runScript('legacy.ember.light');
    expect(f.writes).toEqual([appearanceCookie({ ui: 'legacy', accent: 'ice', mode: 'light', skin: 'mono' })]);
    expect(f.attrs['data-theme']).toBe('daylight');
  });

  test('pre-paint script: Legacy on System follows the OS live (Monolith ↔ Daylight)', () => {
    const f = runScript('legacy.mono.system');
    expect(f.attrs['data-theme']).toBe('monolith');
    f.fire(true);
    expect(f.attrs).toMatchObject({ 'data-mode': 'light', 'data-theme': 'daylight' });
    f.fire(false);
    expect(f.attrs).toMatchObject({ 'data-mode': 'dark', 'data-theme': 'monolith' });
  });
});

describe('L0 tokens (app/globals.css)', () => {
  const block = (sel: string) => {
    const i = L0.indexOf(`${sel} {`);
    return i < 0 ? '' : L0.slice(i, L0.indexOf('}', i));
  };
  const names = (body: string) => [...body.matchAll(/(--l-[\w-]+)\s*:/g)].map((m) => m[1]).sort();

  test('Monolith and Daylight define the same --l-* set, with the FounderOS values for the ground and ink', () => {
    const mono = block(":root[data-theme='monolith']");
    const day = block(":root[data-theme='daylight']");
    expect(names(mono).length).toBeGreaterThan(25);
    expect(names(mono)).toEqual(names(day));
    expect(mono).toMatch(/--l-bg:\s*#0a0a0a/);
    expect(mono).toMatch(/--l-text:\s*#f2f2f2/);
    expect(mono).toMatch(/--l-text-2:\s*#9c9c9c/);
    expect(day).toMatch(/--l-bg:\s*#f2f6f9/);
    expect(day).toMatch(/--l-text:\s*#16222c/);
  });

  test('Classic tokens are remapped under both skins (O1)', () => {
    const remap = L0.slice(L0.lastIndexOf('L0.3 Classic tokens remapped'));
    for (const t of ['--bg', '--surface', '--border', '--text', '--text-2', '--accent', '--ok', '--warn', '--err']) {
      expect(remap, t).toMatch(new RegExp(`\\n\\s*${t}:\\s*var\\(--l-`));
    }
  });

  test('no raw hex outside the token blocks (L0.1, the generated Daylight copies L0.2, L0.3)', () => {
    const furniture = stripComments(L0.slice(L0.lastIndexOf('L0.4 Legacy furniture')));
    expect(furniture.match(/#[0-9a-fA-F]{3,8}\b/g) ?? []).toEqual([]);
  });

  test('no raw hex in the kit, the frame or the specimen', () => {
    const files = [
      ...fs.readdirSync(path.join(root, 'components/legacy')).map((f) => `components/legacy/${f}`),
      'app/dev/l-kit/page.tsx',
      'app/dev/l-kit/LegacyKitSpecimen.tsx',
    ];
    for (const f of files) expect(stripComments(read(f)).match(/['"`(:\s]#[0-9a-fA-F]{3,8}\b/g) ?? [], f).toEqual([]);
  });

  test('the Legacy kit and furniture never use GladOS kit classes, and the GladOS kit never uses .l-*', () => {
    const kit = fs.readdirSync(path.join(root, 'components/legacy')).map((f) => read(`components/legacy/${f}`)).join('\n');
    expect(kit).not.toMatch(/className=[^>]*\bg-(card|btn|seg|chip|badge|page)\b/);
    const g = fs.readdirSync(path.join(root, 'components/g')).map((f) => read(`components/g/${f}`)).join('\n');
    expect(g).not.toMatch(/['" ]l-[a-z]/);
  });
});

describe('JetBrains Mono, bundled (O6)', () => {
  test('the font file and its OFL licence are in the repo; the layout loads it with next/font/local, no Google request', () => {
    expect(fs.statSync(path.join(root, 'app/fonts/JetBrainsMono[wght].woff2')).size).toBeGreaterThan(20_000);
    expect(read('app/fonts/OFL-JetBrainsMono.txt')).toMatch(/SIL OPEN FONT LICENSE Version 1\.1/);
    const layout = read('app/layout.tsx');
    expect(layout).toMatch(/src: '\.\/fonts\/JetBrainsMono\[wght\]\.woff2'/);
    expect(layout).toMatch(/variable: '--l-font-jb'/);
    expect(layout).not.toMatch(/next\/font\/google/);
    expect(L0).toMatch(/--l-font: var\(--l-font-jb/);
    expect(L0).toMatch(/:root\[data-ui='legacy'\] \{\s*[^}]*--font-sans: var\(--l-font\)/);
  });
});

describe('the slab frame (app/layout.tsx, components/legacy/LegacySlab.tsx)', () => {
  const render = (ui: 'legacy' | 'glados' | 'classic', pathname: string) =>
    renderWithRouter(createElement(PathnameContext.Provider, { value: pathname }, createElement(LegacySlab, { ui }, createElement('p', null, 'body'))));

  test('wraps the page in one slab only under Legacy, and never on full-height workspaces', () => {
    expect(render('legacy', '/funnel')).toMatch(/^<div data-l="Slab" data-l-frame="" class="l-slab l-frame"/);
    expect(render('glados', '/funnel')).toBe('<p>body</p>');
    expect(render('classic', '/funnel')).toBe('<p>body</p>');
    for (const w of L_NO_SLAB) expect(render('legacy', w), w).toBe('<p>body</p>');
    expect(isLegacyWorkspace('/workflows/abc')).toBe(true);
    expect(isLegacyWorkspace('/funnel')).toBe(false);
  });

  test('the eyebrow comes from the nav group and the brand, never a hard-coded product name', () => {
    expect(legacyEyebrow('/funnel')).toMatch(/^\/\/ REVENUE · /);
    expect(legacyEyebrow('/funnel')).not.toMatch(/FOUNDER OS/);
    expect(render('legacy', '/funnel')).toContain('--l-eyebrow');
  });

  test('the layout mounts the frame around the page', () => {
    expect(read('app/layout.tsx')).toMatch(/<LegacySlab ui=\{ui\}>\s*<ModuleGate off=\{off\} ui=\{ui\}>\s*\{children\}\s*<\/ModuleGate>\s*<\/LegacySlab>/);
  });
});

describe('kit render (/dev/l-kit specimen)', () => {
  const html = renderWithRouter(createElement(LegacyKitSpecimen));
  const KIT = [
    'SlabTitle', 'SlabCard', 'Chip', 'Badge', 'Dot', 'BigStat', 'MeterStack', 'VolumeMeter', 'StepLine', 'DotMatrix',
    'InsightCard', 'Pressable', 'ToggleChip', 'Kbd', 'ConductorEmblem', 'OsMark', 'Label',
    'Empty', 'Unconfigured', 'Error', 'Skeleton', 'SampleChip', 'SetupLink',
  ];
  test.each(KIT)('renders %s with its data-l name', (name) => {
    expect(html).toContain(`data-l="${name}"`);
  });

  test('states are honest: unknown reads unknown, unconfigured links to setup, sample is marked', () => {
    expect(html).toContain('unknown');
    expect(html).toMatch(/data-l="SetupLink"[^>]*href="\/integrations[^"]*"/);
    expect(html).toContain('Not configured');
    expect(html).toContain('data-sample-badge');
    expect(html).not.toMatch(/GladOS|FOUNDER OS/);
  });

  test('no GladOS kit class anywhere in the Legacy specimen', () => {
    expect(html).not.toMatch(/class="[^"]*\bg-(card|btn|seg|chip|badge|page|swatch)\b/);
    expect(html).not.toMatch(/data-g=/);
  });
});

describe('kit logic (components/legacy/format.ts, from FounderOS kit/format.ts)', () => {
  test('dots, ticks, meters', () => {
    expect(dotState('connected')).toBe('ok');
    expect(dotState('not_configured')).toBe('off');
    expect(dotState('whatever')).toBe('off');
    expect(insightTicks(0.6)).toBe(5);
    expect(insightTicks(null)).toBe(0);
    expect(meterWidth(0)).toBe(0.02);
    expect(meterWidth(null)).toBeNull();
    expect(meterWidth(2)).toBe(1);
  });

  test('StepLine geometry pins the peak; empty series draws nothing', () => {
    expect(stepGeometry([])).toBeNull();
    expect(stepGeometry([{ label: 'a', count: 0 }])).toBeNull();
    const g = stepGeometry([{ label: 'a', count: 1 }, { label: 'b', count: 5 }, { label: 'c', count: 2 }])!;
    expect(g.peakIdx).toBe(1);
    expect(g.path.startsWith('M 0 ')).toBe(true);
  });

  test('DotMatrix: the tallest column is six dots, a zero column none', () => {
    const c = dotColumns([{ label: 'a', count: 0 }, { label: 'b', count: 4 }, { label: 'c', count: 8 }]);
    expect(c.map((x) => x.dots)).toEqual([0, 3, 6]);
  });
});
