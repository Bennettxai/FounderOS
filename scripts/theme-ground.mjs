// data-ground for a theme the audit scripts force. ground-check, theme-audit and screenshot-matrix switch
// <html data-theme> after load; the glass light (groove, drop, shade) reads data-ground, so each switch
// sets it too, the way the app does (lib/appearance.ts htmlAttributes and the head script). The table is
// read from lib/appearance.ts, so there is one source; tests/glass-audit-scripts.test.ts checks groundFor
// here against lib/appearance.ts groundFor for every theme, mode and style.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

/** GROUND_BY_THEME from lib/appearance.ts: theme id to black | grey | white. */
export const GROUND_BY_THEME = (() => {
  const src = fs.readFileSync(path.join(root, 'lib/appearance.ts'), 'utf8');
  const m = src.match(/export const GROUND_BY_THEME\b[^=]*=\s*\{([^}]*)\}/);
  if (!m) throw new Error('scripts/theme-ground.mjs: GROUND_BY_THEME not found in lib/appearance.ts');
  return /** @type {Record<string, string>} */ (Object.fromEntries([...m[1].matchAll(/'?([\w-]+)'?\s*:\s*'(\w+)'/g)].map((x) => [x[1], x[2]])));
})();

/**
 * lib/appearance.ts groundFor. Serialised into the page, so it closes over nothing: the table is passed in.
 * @param {string} theme
 * @param {'dark' | 'light'} mode
 * @param {string | null | undefined} ui
 * @param {Record<string, string>} table
 */
export function groundFor(theme, mode, ui, table) {
  if (ui === 'glados') return mode === 'light' ? 'white' : 'grey';
  return table[theme] ?? (mode === 'light' ? 'white' : 'black');
}

/**
 * A page expression that forces <html data-theme> and sets the matching data-ground, from the page's own
 * data-ui and data-mode. Evaluates to the ground it set.
 * @param {string} theme
 */
export function setThemeExpr(theme) {
  const t = JSON.stringify(theme);
  return `(() => {
  const d = document.documentElement;
  const groundFor = ${groundFor.toString()};
  d.setAttribute('data-theme', ${t});
  d.setAttribute('data-ground', groundFor(${t}, d.getAttribute('data-mode') === 'light' ? 'light' : 'dark', d.getAttribute('data-ui'), ${JSON.stringify(GROUND_BY_THEME)}));
  return d.getAttribute('data-ground');
})()`;
}
