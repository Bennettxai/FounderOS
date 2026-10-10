import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { beforeEach, describe, expect, it } from 'vitest';
import { FOUNDEROS_THEMES, FOUNDEROS_THEME_INIT_SCRIPT, DEFAULT_FOUNDEROS_THEME, STORAGE_KEY, THEME_META, applyFounderosTheme, isDarkTheme, readStoredTheme } from './theme';

describe('the operator theme selection', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-founderos-theme');
    document.documentElement.classList.remove('dark');
  });

  it('defaults to Monolith Signal, like FounderOS v1', () => {
    expect(DEFAULT_FOUNDEROS_THEME).toBe('mono');
    expect(readStoredTheme()).toBe('mono');
  });

  // FounderOS v1 lib/theme.ts THEMES, in its order. Its 'dark' (Terminal) keeps
  // the bridge's id 'terminal'; every other id is production's.
  it("offers every production skin, in production's order, and ignores unknown stored values", () => {
    expect(FOUNDEROS_THEMES).toEqual(['mono', 'mono-light', 'terminal', 'light', 'midnight', 'ember']);
    localStorage.setItem(STORAGE_KEY, 'neon');
    expect(readStoredTheme()).toBe('mono');
    for (const t of FOUNDEROS_THEMES) {
      localStorage.setItem(STORAGE_KEY, t);
      expect(readStoredTheme()).toBe(t);
    }
  });

  it("reads production's id for Terminal ('dark') as terminal", () => {
    localStorage.setItem(STORAGE_KEY, 'dark');
    expect(readStoredTheme()).toBe('terminal');
  });

  it('picker metadata matches FounderOS v1 THEME_META: name, one-line feel, [bg, accent, text] swatch', () => {
    expect(THEME_META).toEqual({
      terminal: { name: 'Terminal', blurb: 'phosphor green on near-black', swatch: ['#050807', '#3df08c', '#e4efe6'] },
      light: { name: 'Clay', blurb: 'warm paper with clay orange', swatch: ['#ece3d2', '#c96442', '#2b2722'] },
      midnight: { name: 'Midnight', blurb: 'deep navy, signal blue', swatch: ['#070d1f', '#5ec9f8', '#e8ecf9'] },
      ember: { name: 'Ember', blurb: 'coal dark, vault orange', swatch: ['#0c0806', '#e35c35', '#f2e9e2'] },
      mono: { name: 'Monolith', blurb: 'white on black, color = status only', swatch: ['#0a0a0a', '#f2f2f2', '#2fd36f'] },
      // G-Brain is retired on the bridge; the blurb names the colour, not the product
      'mono-light': { name: 'Daylight', blurb: 'brain blue on cool white', swatch: ['#f2f6f9', '#4db3de', '#16222c'] }
    });
  });

  it('applies a dark theme as a data attribute on a dark document and persists it', () => {
    applyFounderosTheme('terminal');
    expect(document.documentElement.getAttribute('data-founderos-theme')).toBe('terminal');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
    expect(localStorage.getItem(STORAGE_KEY)).toBe('terminal');
  });

  // BusinessOS components under /os follow `.dark`; a light skin must drop it
  // or they paint dark panels on Daylight's white canvas.
  it('light skins (Daylight, Clay) leave the document light', () => {
    expect(FOUNDEROS_THEMES.filter((t) => !isDarkTheme(t))).toEqual(['mono-light', 'light']);
    document.documentElement.classList.add('dark');
    applyFounderosTheme('mono-light');
    expect(document.documentElement.getAttribute('data-founderos-theme')).toBe('mono-light');
    expect(document.documentElement.classList.contains('dark')).toBe(false);
    applyFounderosTheme('midnight');
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });
});

// Token parity: the bridge must look like FounderOS v1, so the Monolith and
// Terminal values are pinned to app/globals.css in FounderOS v1 (2026-09-30).
const css = readFileSync(resolve(__dirname, 'monolith.css'), 'utf8');

function block(selector: string): Record<string, string> {
  const start = css.indexOf(selector);
  expect(start, `missing ${selector}`).toBeGreaterThanOrEqual(0);
  const body = css.slice(css.indexOf('{', start) + 1, css.indexOf('}', start));
  const out: Record<string, string> = {};
  for (const m of body.matchAll(/(--[a-z0-9-]+)\s*:\s*([^;]+);/g)) out[m[1]] = m[2].trim();
  return out;
}

describe('Monolith Signal tokens', () => {
  it('match FounderOS v1 mono exactly', () => {
    expect(block(":root[data-founderos-theme='mono']")).toMatchObject({
      '--bn-bg': '#0a0a0a',
      '--bn-surface': '#0a0a0a',
      '--bn-surface-2': '#141414',
      '--bn-surface-3': '#1c1c1c',
      '--bn-border': '#242424',
      '--bn-border-strong': '#3a3a3a',
      '--bn-text': '#f2f2f2',
      '--bn-text-2': '#9c9c9c',
      '--bn-text-3': '#5c5c5c',
      '--bn-accent': '#f2f2f2',
      '--bn-accent-ink': '#0a0a0a',
      '--bn-ok': '#2fd36f',
      '--bn-warn': '#ffb000',
      '--bn-err': '#ff2d3f',
      '--bn-hairline': '#1c1c1c',
    });
  });

  it('match FounderOS v1 Terminal exactly', () => {
    expect(block(":root[data-founderos-theme='terminal']")).toMatchObject({
      '--bn-bg': '#050807',
      '--bn-surface': '#0a0f0c',
      '--bn-border': '#18211b',
      '--bn-border-strong': '#243029',
      '--bn-text': '#e4efe6',
      '--bn-text-2': '#8fa295',
      '--bn-text-3': '#54665b',
      '--bn-accent': '#3df08c',
      '--bn-ok': '#3df08c',
      '--bn-warn': '#ffc53d',
      '--bn-err': '#ff6259',
    });
  });

  it('JetBrains Mono everywhere, mapped onto BusinessOS tokens', () => {
    const shared = block(':root[data-founderos-theme]');
    expect(shared['--bn-font']).toContain('JetBrains Mono');
    for (const bos of ['--bos-v2-layer-background-primary', '--bos-text-primary-color', '--bos-border-color', '--radius', '--background', '--foreground']) {
      expect(css, `${bos} must be mapped`).toMatch(new RegExp(`${bos}\\s*:\\s*var\\(--bn-`));
    }
  });

  // FounderOS v1 rounds its controls and panels (interaction rebrand, 2026-09-07:
  // tailwind.config.ts ctl 6 / panel 10 / tile 12, chips 5, cards 8, the
  // .os-slab 28). The bridge used to square every element; it must not.
  it('rounded like FounderOS v1: the radius scale, and no global square-corner override', () => {
    const shared = block(':root[data-founderos-theme]');
    expect(shared).toMatchObject({
      '--bn-r-chip': '5px',
      '--bn-r-ctl': '6px',
      '--bn-r-card': '8px',
      '--bn-r-panel': '10px',
      '--bn-r-tile': '12px',
      '--bn-r-slab': '28px',
      '--bn-radius': 'var(--bn-r-panel)',
    });
    expect(css).not.toMatch(/border-radius:\s*0\s*!important/);
  });

  it('carries the FounderOS v1 tokens the slab, insight card and lens need, per colorway', () => {
    expect(block(":root[data-founderos-theme='mono']")).toMatchObject({
      '--bn-bg-2': '#0a0a0a',
      '--bn-accent-2': '#d4d4d4',
      '--bn-accent-line': 'rgba(242, 242, 242, 0.25)',
      '--bn-brain-1': '#f2f2f2',
      '--bn-brain-2': '#9c9c9c',
      '--bn-brain-3': '#5c5c5c',
    });
    expect(block(":root[data-founderos-theme='terminal']")).toMatchObject({
      '--bn-bg-2': '#070b09',
      '--bn-accent-2': '#35d97e',
      '--bn-accent-line': 'rgba(61, 240, 140, 0.25)',
      '--bn-brain-1': '#8b7cff',
      '--bn-brain-2': '#5ec9f8',
      '--bn-brain-3': '#4ade96',
    });
    expect(block(':root[data-founderos-theme]')).toMatchObject({
      '--bn-insight-ink': '#fff',
      '--bn-shadow-slab': '0 1px 2px rgb(0 0 0 / 0.4), 0 24px 70px -18px rgb(0 0 0 / 0.6)',
      '--bn-shadow-lift': '0 8px 22px rgba(0, 0, 0, 0.6)',
      '--bn-shadow-pop': '0 24px 60px rgba(0, 0, 0, 0.75)',
    });
  });

  it('round scrollbar thumbs and accent selection, on /os only', () => {
    expect(css).toMatch(/:root\[data-founderos-theme\] ::-webkit-scrollbar-thumb \{[^}]*border-radius: 999px/);
    expect(css).toMatch(/:root\[data-founderos-theme\] ::selection \{[^}]*background: var\(--bn-accent\)/);
  });
});

// The four production skins the bridge lacked (FounderOS v1 app/globals.css
// :root[data-theme='mono-light' | 'light' | 'midnight' | 'ember'] plus the
// per-theme --hairline block), value for value.
const PROD_SKINS: Record<string, Record<string, string>> = {
  'mono-light': {
    '--bn-bg': '#f2f6f9', '--bn-bg-2': '#e9eff5', '--bn-surface': '#ffffff', '--bn-surface-2': '#e9eff4', '--bn-surface-3': '#dce6ee',
    '--bn-border': '#cfdce6', '--bn-border-strong': '#afc3d2', '--bn-text': '#16222c', '--bn-text-2': '#47596b', '--bn-text-3': '#70839a',
    '--bn-accent': '#4db3de', '--bn-accent-2': '#2f93c0', '--bn-accent-ink': '#16222c', '--bn-accent-soft': 'rgba(77, 179, 222, 0.1)',
    '--bn-accent-line': 'rgba(77, 179, 222, 0.32)', '--bn-ok': '#17924e', '--bn-warn': '#b7791f', '--bn-err': '#d92d20',
    '--bn-grid': 'transparent', '--bn-hairline': '#e3eaf1', '--bn-brain-1': '#6d5cf0', '--bn-brain-2': '#4db3de', '--bn-brain-3': '#2dbdaf'
  },
  light: {
    '--bn-bg': '#ece3d2', '--bn-bg-2': '#f1ead9', '--bn-surface': '#f6f0e4', '--bn-surface-2': '#efe7d6', '--bn-surface-3': '#e7decb',
    '--bn-border': '#ddd3bf', '--bn-border-strong': '#c9bda4', '--bn-text': '#2b2722', '--bn-text-2': '#6f675a', '--bn-text-3': '#9a9082',
    '--bn-accent': '#c96442', '--bn-accent-2': '#a84e30', '--bn-accent-ink': '#fdf7ee', '--bn-accent-soft': 'rgba(201, 100, 66, 0.12)',
    '--bn-accent-line': 'rgba(201, 100, 66, 0.32)', '--bn-ok': '#0b8a52', '--bn-warn': '#b7791f', '--bn-err': '#c0392b',
    '--bn-grid': 'rgba(43, 39, 34, 0.04)', '--bn-hairline': '#e4dbc9', '--bn-brain-1': '#7a5cf0', '--bn-brain-2': '#2b8fd8', '--bn-brain-3': '#c96442'
  },
  midnight: {
    '--bn-bg': '#070d1f', '--bn-bg-2': '#091126', '--bn-surface': '#0c142e', '--bn-surface-2': '#111a3a', '--bn-surface-3': '#172246',
    '--bn-border': '#1a2547', '--bn-border-strong': '#283765', '--bn-text': '#e8ecf9', '--bn-text-2': '#93a3cf', '--bn-text-3': '#5b6d9e',
    '--bn-accent': '#5ec9f8', '--bn-accent-2': '#48b4e6', '--bn-accent-ink': '#041019', '--bn-accent-soft': 'rgba(94, 201, 248, 0.09)',
    '--bn-accent-line': 'rgba(94, 201, 248, 0.28)', '--bn-ok': '#4ade96', '--bn-warn': '#fbbf24', '--bn-err': '#f87171',
    '--bn-grid': 'rgba(232, 236, 249, 0.02)', '--bn-hairline': '#131c3a', '--bn-brain-1': '#8b7cff', '--bn-brain-2': '#5ec9f8', '--bn-brain-3': '#4ade96'
  },
  ember: {
    '--bn-bg': '#0c0806', '--bn-bg-2': '#100a08', '--bn-surface': '#140d0a', '--bn-surface-2': '#1a120d', '--bn-surface-3': '#221711',
    '--bn-border': '#241812', '--bn-border-strong': '#38251b', '--bn-text': '#f2e9e2', '--bn-text-2': '#b39a8c', '--bn-text-3': '#755f52',
    '--bn-accent': '#e35c35', '--bn-accent-2': '#c94e2c', '--bn-accent-ink': '#180a05', '--bn-accent-soft': 'rgba(227, 92, 53, 0.1)',
    '--bn-accent-line': 'rgba(227, 92, 53, 0.3)', '--bn-ok': '#4ade96', '--bn-warn': '#fbbf24', '--bn-err': '#ff6259',
    '--bn-grid': 'rgba(242, 233, 226, 0.02)', '--bn-hairline': '#1d130e', '--bn-brain-1': '#e35c35', '--bn-brain-2': '#f0a05a', '--bn-brain-3': '#d98d62'
  }
};

describe('every production skin', () => {
  for (const [id, tokens] of Object.entries(PROD_SKINS)) {
    it(`${id} matches FounderOS v1 exactly`, () => {
      expect(block(`:root[data-founderos-theme='${id}']`)).toMatchObject(tokens);
    });
  }

  it('Terminal keeps its row hairline (#141b16) and grid', () => {
    expect(block(":root[data-founderos-theme='terminal']")).toMatchObject({ '--bn-hairline': '#141b16', '--bn-grid': 'rgba(228, 239, 230, 0.018)' });
  });

  it('light skins declare color-scheme light, dark skins dark', () => {
    for (const id of ['mono-light', 'light']) expect(css).toMatch(new RegExp(`:root\\[data-founderos-theme='${id}'\\] \\{\\s*color-scheme: light;`));
    for (const id of ['mono', 'terminal', 'midnight', 'ember']) expect(css).toMatch(new RegExp(`:root\\[data-founderos-theme='${id}'\\] \\{\\s*color-scheme: dark;`));
  });

  // The insight card's type is white; on light colorways its floor goes to deep ink.
  it('the insight card floor darkens on the light colorways', () => {
    const kit = readFileSync(resolve(__dirname, '../kit/kit.css'), 'utf8');
    expect(kit).toMatch(/:root\[data-founderos-theme='light'\] \.bn-insight,\s*:root\[data-founderos-theme='mono-light'\] \.bn-insight \{\s*--bn-insight-base: color-mix\(in oklab, var\(--bn-text\) 90%, var\(--bn-brain-1\)\);/);
  });

  // FounderOS v1 body: a 48px grid off --grid (transparent on Monolith/Daylight).
  it('the canvas carries the 48px grid texture from --bn-grid', () => {
    const layout = readFileSync(resolve(__dirname, '../../../routes/(founderos)/os/+layout.svelte'), 'utf8');
    expect(layout).toMatch(/repeating-linear-gradient\(0deg, var\(--bn-grid\) 0 1px, transparent 1px 48px\)/);
    expect(layout).toMatch(/repeating-linear-gradient\(90deg, var\(--bn-grid\) 0 1px, transparent 1px 48px\)/);
  });
});

// FounderOS v1 type: next/font JetBrains Mono (font-sans AND font-mono resolve to
// it), its generated Arial fallback face (that is what draws glyphs the font
// lacks, like "→", the long Arial arrow), and an antialiased body.
describe('lettering, as production draws it', () => {
  it('Tailwind font-mono and font-sans resolve to JetBrains Mono on /os', () => {
    const shared = block(':root[data-founderos-theme]');
    expect(shared['--bn-font']).toBe("'JetBrains Mono', 'the operator Mono Fallback', ui-monospace, SFMono-Regular, monospace");
    expect(shared['--font-mono']).toBe('var(--bn-font)');
    expect(shared['--font-sans']).toBe('var(--bn-font)');
  });

  it("the fallback face is next/font's: local Arial with its metric overrides", () => {
    expect(css).toMatch(
      /@font-face \{\s*font-family: 'the operator Mono Fallback';\s*src: local\('Arial'\);\s*ascent-override: 75\.79%;\s*descent-override: 22\.29%;\s*line-gap-override: 0%;\s*size-adjust: 134\.59%;\s*\}/
    );
  });

  it('the body is antialiased, as FounderOS v1 body @apply antialiased', () => {
    expect(css).toMatch(/:root\[data-founderos-theme\] body \{[^}]*-webkit-font-smoothing: antialiased;[^}]*-moz-osx-font-smoothing: grayscale;/);
  });
});

// BusinessOS ships two UNLAYERED universal rules that beat every Tailwind
// utility on /os: `* { border-color: hsl(var(--border)) }` (variables.css; and
// under the skin --border is a hex, so hsl() is invalid and every border fell
// to currentColor) and `* { scrollbar-color: … }` (app.css; once set, Chrome
// ignores the ::-webkit-scrollbar styling FounderOS v1 draws). Under the skin
// both roll back to the layered cascade, at element-type specificity, so any
// class rule (.scrollbar-hide, a component's own border) still wins.
describe('BusinessOS universal rules stand down under the skin', () => {
  it('border-color and the scrollbar properties revert to the layers, BusinessOS pages untouched', () => {
    expect(css).toMatch(
      /:where\(:root\[data-founderos-theme\]\) :is\(html \*\) \{\s*border-color: revert-layer;\s*scrollbar-width: revert-layer;\s*scrollbar-color: revert-layer;\s*\}/
    );
    expect(css).toMatch(/:root\[data-founderos-theme\] \{\s*scrollbar-width: auto;\s*scrollbar-color: auto;\s*\}/);
    const vars = readFileSync(resolve(__dirname, '../../modules/theme/styles/variables.css'), 'utf8');
    expect(vars).toMatch(/\*\s*\{\s*border-color: hsl\(var\(--border\)\);/);
  });
});

describe('first paint', () => {
  it('app.html runs the exact init script, before SvelteKit, and loads JetBrains Mono', () => {
    const html = readFileSync(resolve(__dirname, '../../../app.html'), 'utf8');
    expect(html).toContain(`<script>${FOUNDEROS_THEME_INIT_SCRIPT}</script>`);
    expect(html.indexOf(FOUNDEROS_THEME_INIT_SCRIPT)).toBeLessThan(html.indexOf('%sveltekit.head%'));
    expect(html).toMatch(/fonts\.googleapis\.com\/css2\?family=JetBrains\+Mono/);
  });

  // It used to pin BusinessOS's own 'theme' to dark on every page, which
  // flipped the whole desktop dark once FounderOS v1 had been opened.
  it("never writes BusinessOS's own theme", () => {
    expect(FOUNDEROS_THEME_INIT_SCRIPT).not.toContain(`setItem('theme'`);
  });

  it('skins only the /os pages', () => {
    localStorage.removeItem(STORAGE_KEY);
    const run = (path: string) => {
      window.history.pushState({}, '', path);
      const root = document.documentElement;
      root.removeAttribute('data-founderos-theme');
      root.classList.remove('dark');
      new Function(FOUNDEROS_THEME_INIT_SCRIPT)();
      return [root.getAttribute('data-founderos-theme'), root.classList.contains('dark')];
    };
    expect(run('/os/comms')).toEqual(['mono', true]);
    expect(run('/os')).toEqual(['mono', true]);
    expect(run('/window')).toEqual([null, false]);
    localStorage.setItem(STORAGE_KEY, 'mono-light');
    expect(run('/os/usage')).toEqual(['mono-light', false]);
    localStorage.setItem(STORAGE_KEY, 'dark');
    expect(run('/os/usage')).toEqual(['terminal', true]);
    localStorage.removeItem(STORAGE_KEY);
    expect(run('/osa')).toEqual([null, false]);
  });

  it('app.css imports the Monolith layer after the BusinessOS theme layers', () => {
    const appCss = readFileSync(resolve(__dirname, '../../../app.css'), 'utf8');
    const mono = appCss.indexOf("$lib/founderos/theme/monolith.css");
    expect(mono).toBeGreaterThan(appCss.indexOf('bos-variables.css'));
    expect(mono).toBeGreaterThan(appCss.indexOf('variables.css'));
  });
});
