#!/usr/bin/env node
// GladOS UI audit (docs/projects/glados-ui/SPEC.md S00 §10 and step 8, §10.2, GP4).
//
// USAGE
//   Classic baseline (AC00.18): Classic style, the five lib/theme.ts themes, every audit route, 1440×900.
//     node scripts/g-audit.mjs --classic-baseline --capture <dir>
//     node scripts/g-audit.mjs --classic-baseline --compare <dir> [--out <dir>] [--threshold 1] [--tolerance 2]
//       capture writes <dir>/<theme>/<route>.png and <dir>/manifest.json; compare re-captures the same
//       themes × routes into --out (default <dir>-compare), diffs pixel by pixel, writes
//       <out>/diff/<theme>/<route>.png for every route over the threshold and <out>/report.json, prints a
//       line per route and exits 1 when any route differs by more than --threshold percent of its pixels
//       (a pixel differs when a channel moves by more than --tolerance). Animations and transitions are
//       frozen and reduced motion is emulated on both sides so the diff sees layout and colour only.
//     Masks: regions showing live third-party state are painted flat on BOTH sides before the diff
//       (the union of the baseline's recorded rects and the new ones). Defaults: /compute (live MIOSA
//       state: the header's right slot and everything below the header) and the /setup data-dir line
//       (differs per checkout). Add your own with
//       --mask <route>=<css selector> (repeatable); --no-default-masks drops the defaults. Capture
//       records each shot's mask rects in the manifest.
//     node scripts/g-audit.mjs --classic-baseline --capture <dir> --update --routes /setup [--themes ...]
//       --update re-captures only the given routes x themes into an existing baseline and merges them
//       into its manifest (other shots are kept; the merge is logged under "updates").
//
//   GladOS matrix (§10.2): style glados × accents × modes × widths, with automated checks per shot.
//     node scripts/g-audit.mjs --glados [--accents ice,ember] [--modes dark,light] [--widths 1440,1024,390] [--full]
//       shots and report.json go to --out (default docs/projects/glados-ui/shots/audit, gitignored) as
//       <accent>-<mode>/<width>/<route>.png. Checks: appearance applied (data-ui/-accent/-mode), horizontal
//       overflow, card border (AC00.14), accent-hued or outer-glow box-shadow (§3.3.8), unexpected mono
//       outside code/kbd/pre/samp/[data-copy]/[data-mono-ok] (C24), uppercase or letterspaced labels (AC04.5),
//       hero count ≤ 2 per view (C22), hero contrast sampled from the rendered gradient behind each text
//       (caption ≥ 4.5:1, numeral ≥ 3:1, AC00.12), computed text contrast inside kit components, and the
//       Classic-control guard (GP4) on routes in G_READY. Card, shadow, mono and contrast checks look at
//       kit elements ([data-g]) everywhere and at the whole document on converted routes.
//
//   Classic-control guard only (FR-GP4.3, ACGP4.1):
//     node scripts/g-audit.mjs --controls                          every G_READY route, glados · ice · dark, 1440
//     node scripts/g-audit.mjs --controls --fixture classic-button exits 1: one .c-btn under data-ui=glados
//     node scripts/g-audit.mjs --controls --fixture kit-button     exits 0: the same page built from GButton
//     node scripts/g-audit.mjs --controls --fixture <file.html>    any fixture page
//
//   Content Lab v2 styles (docs/projects/content-lab/v2/spec/02-components.md 15.3):
//     node scripts/g-audit.mjs --lab-styles [--styles glados-ember-dark,classic-console] [--widths 1440,1024,390] [--routes ...]
//       every /lab route with ?sample=1 in the 13 styles at 1440, 1024 and 390. Checks: the page checks above,
//       pixel contrast of every [data-lab-ink] (4.5:1, 3:1 at 24px and over) and [data-lab-mark] (3:1) sampled
//       from the rendered pixels behind it at each viewport down the page (lab-contrast), at most 12 blurred
//       surfaces in view at scroll 0, 1 and 2 viewports (lab-glass-budget), reduced transparency emulated:
//       every Glass opaque with no blur (lab-reduced-transparency, first width only), and under GladOS: no
//       Glass border, kit shadow rules, at most 2 heroes or narrative cards in view (hero-count), the control
//       guard. The lab.mjs interactions open the sheets and menus and the checks run again. Shots:
//       docs/projects/glados-ui/shots/audit/lab/<style>/<width>/<route>.png. Exit 1 on any failure.
//
// COMMON OPTIONS
//   --routes /,/inbox   comma list (default: AUDIT_ROUTES from scripts/audit-routes.mjs, or ROUTES env)
//   --themes a,b        Classic themes (default: every id in lib/theme.ts)
//   GLADOS_URL          server (default http://localhost:4001); WAIT ms after load (default 2500)
//   CDP_PORT            attach to a Chrome already started with --remote-debugging-port; without it the
//                       script launches headless Chrome itself (CHROME = binary path, default Google Chrome)
//
// HOW APPEARANCE IS SET: cookie glados-appearance=<ui>.<accent>.<mode> plus localStorage glados-ui,
// glados-accent, glados-mode and glados-theme (S00 FR-2), and prefers-color-scheme emulated to the mode.
// G_READY and the route → files registry are read from lib/appearance.ts (else lib/g-ready.ts) by
// scripts/g-audit/_lib.mjs; with neither the registry is empty and the guard has nothing to check. Until
// the appearance model ships, the server ignores the cookie: --glados then reports "appearance not
// applied" as a warning (an error once lib/appearance.ts exists).
//
// ROUTE PLUGINS (C42): every scripts/g-audit/<area>.mjs (not starting with "_") may export
//   export const interactions = { '/route': [{ name: 'open details sheet', run: async (ctx) => { await ctx.click('[data-g=GButton]'); } }] };
// ctx = { ev, send, sleep, click(selector), press(key) }. After each step the guard and the page checks
// run again, so sheets, menus and toasts a step opens are audited too. A plugin may also export
//   export const pixelContrast = { '/lab': ['[data-lab-ink]', '[data-lab-mark]'] }
// and every matching route gets the pixel-sampled contrast check (02 15.3 item 1) after load and each step.
//
// SAFETY: every non-GET /api request is failed (cdp() in scripts/audit-routes.mjs, SPEC §3.1.8).
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync, spawn } from 'node:child_process';
import { pathToFileURL } from 'node:url';
import { AUDIT_ROUTES, THEME_IDS, cdp } from './audit-routes.mjs';
import {
  ROOT, badShadows, composite, contrastRatio, decodePng, diffImages, encodePng, loadClassicControls, maskImage, parseColor,
  parseMasks, readGReady, readyEntryFor, routeInReady, routeSlug,
} from './g-audit/_lib.mjs';
import {
  LAB_GLASS_BUDGET, LAB_PIXEL, LAB_ROUTES, LAB_STYLES, glassScanInPage, labHeroesInPage, pixelTargetsInPage, reducedGlassInPage, sampleTargets,
} from './g-audit/_lab.mjs';

// ── args ─────────────────────────────────────────────────────────────────────────────────────
const argv = process.argv.slice(2);
const flag = (n) => argv.includes(n);
const opt = (n, d) => { const i = argv.indexOf(n); return i >= 0 && argv[i + 1] && !argv[i + 1].startsWith('--') ? argv[i + 1] : d; };
const list = (n, d) => { const v = opt(n); return v ? v.split(',').filter(Boolean) : d; };

const BASE = (process.env.GLADOS_URL ?? 'http://localhost:4001').replace(/\/$/, '');
const WAIT = Number(process.env.WAIT ?? 2500);
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const envRoutes = (process.env.ROUTES ?? '').split(' ').filter(Boolean);
const CLASSIC = loadClassicControls();
const REG = readGReady();
// G_READY entries with a :param (e.g. /agents/setup/:id) are not in AUDIT_ROUTES; the default list adds
// one concrete page for each (§10.2 "/agents/setup/<first id>") so the guard and checks reach it.
const PARAM_SAMPLES = {
  '/agents/setup/:id': async () => {
    const r = await fetch(`${BASE}/api/agents`).then((x) => x.json()).catch(() => null);
    const id = r?.agents?.[0]?.id;
    return id ? `/agents/setup/${encodeURIComponent(id)}` : null;
  },
  // any SocialPlatformSchema value renders the platform page; instagram is the one the demo fills
  '/social/:platform': async () => '/social/instagram',
};
async function paramRoutes() {
  if (opt('--routes') || envRoutes.length || flag('--classic-baseline')) return [];
  const out = [];
  for (const e of REG.routes.filter((r) => r.includes('/:'))) {
    const route = PARAM_SAMPLES[e] ? await PARAM_SAMPLES[e]() : null;
    if (route) out.push(route);
    else console.log(`${e}: no sample page to audit (add one to PARAM_SAMPLES)`);
  }
  return out;
}
const ROUTES = [...list('--routes', envRoutes.length ? envRoutes : AUDIT_ROUTES), ...(await paramRoutes())];

function usage(msg) {
  if (msg) console.error(msg);
  console.error(fs.readFileSync(new URL(import.meta.url), 'utf8').split('\n').filter((l) => l.startsWith('//')).slice(0, 60).map((l) => l.slice(3)).join('\n'));
  process.exit(2);
}

// ── browser ──────────────────────────────────────────────────────────────────────────────────
async function startBrowser() {
  if (process.env.CDP_PORT) return { stop: () => {} };
  const bin = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
  if (!fs.existsSync(bin)) usage(`No Chrome at ${bin}. Set CHROME=<binary> or start one and set CDP_PORT.`);
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'g-audit-'));
  const child = spawn(bin, [
    '--headless=new', '--remote-debugging-port=0', `--user-data-dir=${dir}`, '--hide-scrollbars', '--force-color-profile=srgb',
    '--no-first-run', '--no-default-browser-check', '--window-size=1440,900', 'about:blank',
  ], { stdio: 'ignore' });
  const portFile = path.join(dir, 'DevToolsActivePort');
  for (let i = 0; i < 150 && !fs.existsSync(portFile); i++) await sleep(100);
  if (!fs.existsSync(portFile)) { child.kill(); usage('Chrome did not start (no DevToolsActivePort).'); }
  process.env.CDP_PORT = fs.readFileSync(portFile, 'utf8').split('\n')[0].trim();
  return { stop: () => { child.kill(); setTimeout(() => fs.rmSync(dir, { recursive: true, force: true }), 500); } };
}

let S; // { send, ev, close }
let initScriptId = null;

async function setViewport(width) {
  await S.send('Emulation.setDeviceMetricsOverride', { width, height: width <= 480 ? 844 : 900, deviceScaleFactor: 1, mobile: false });
}

/** Appearance for the next loads: cookie + storage + emulated colour scheme (S00 FR-2). */
async function setAppearance({ ui, accent = 'ice', mode = 'dark', theme }) {
  await S.send('Network.setCookie', {
    name: 'glados-appearance', value: `${ui}.${accent}.${mode}`, url: BASE, path: '/',
    sameSite: 'Lax', expires: Math.floor(Date.now() / 1000) + 365 * 86400,
  });
  if (initScriptId) await S.send('Page.removeScriptToEvaluateOnNewDocument', { identifier: initScriptId }).catch(() => {});
  const store = { 'glados-ui': ui, 'glados-accent': accent, 'glados-mode': mode, ...(theme ? { 'glados-theme': theme } : {}) };
  ({ identifier: initScriptId } = await S.send('Page.addScriptToEvaluateOnNewDocument', {
    source: `try{var s=${JSON.stringify(store)};for(var k in s)localStorage.setItem(k,s[k]);}catch(e){}`,
  }));
  const features = [{ name: 'prefers-color-scheme', value: mode === 'light' ? 'light' : 'dark' }];
  if (FREEZE) features.push({ name: 'prefers-reduced-motion', value: 'reduce' });
  await S.send('Emulation.setEmulatedMedia', { features });
}

let FREEZE = false;
const FREEZE_CSS = '*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important}';

/** Navigate and wait for load, fonts and WAIT ms. Returns { status, error }. */
async function go(url) {
  await S.ev('window.__gAuditStale = 1').catch(() => {});
  const nav = await S.send('Page.navigate', { url });
  if (nav.errorText) return { status: 0, error: nav.errorText };
  const t0 = Date.now();
  while (Date.now() - t0 < 120000) {
    await sleep(200);
    if (await S.ev("!window.__gAuditStale && document.readyState === 'complete'").catch(() => false)) break;
  }
  await S.ev('document.fonts ? document.fonts.ready.then(() => 1) : 1').catch(() => {});
  if (FREEZE) await S.ev(`(() => { const s = document.createElement('style'); s.textContent = ${JSON.stringify(FREEZE_CSS)}; document.head.appendChild(s); return 1; })()`);
  await sleep(WAIT);
  const status = await S.ev("(performance.getEntriesByType('navigation')[0] || {}).responseStatus || 0").catch(() => 0);
  return { status };
}

async function shot(file, full = false) {
  let params = { format: 'png' };
  if (full) {
    const h = await S.ev('Math.max(document.documentElement.scrollHeight, document.body ? document.body.scrollHeight : 0)');
    const w = await S.ev('document.documentElement.clientWidth');
    params = { format: 'png', captureBeyondViewport: true, clip: { x: 0, y: 0, width: w, height: Math.min(h, 16000), scale: 1 } };
  }
  const r = await S.send('Page.captureScreenshot', params);
  const buf = Buffer.from(r.data, 'base64');
  if (file) { fs.mkdirSync(path.dirname(file), { recursive: true }); fs.writeFileSync(file, buf); }
  return buf;
}

const attrs = () => S.ev(`(() => { const h = document.documentElement; return { ui: h.getAttribute('data-ui'), accent: h.getAttribute('data-accent'), mode: h.getAttribute('data-mode'), modePref: h.getAttribute('data-mode-pref'), theme: h.getAttribute('data-theme') }; })()`);

// ── in-page scans (serialised; no closures) ──────────────────────────────────────────────────
function pageScan(cfg) {
  const sel = (el) => {
    const parts = [];
    for (let n = el, i = 0; n && n.nodeType === 1 && i < 4; n = n.parentElement, i++) {
      const c = typeof n.className === 'string' ? n.className.trim().split(/\s+/).filter(Boolean).slice(0, 3).join('.') : '';
      parts.unshift(n.tagName.toLowerCase() + (n.id ? '#' + n.id : '') + (n.getAttribute('data-g') ? `[data-g=${n.getAttribute('data-g')}]` : '') + (c ? '.' + c : ''));
    }
    return parts.join(' > ');
  };
  const visible = (el) => { const r = el.getBoundingClientRect(); if (r.width < 1 || r.height < 1) return false; const cs = getComputedStyle(el); return cs.visibility !== 'hidden' && cs.display !== 'none' && cs.opacity !== '0'; };
  const ownText = (el) => [...el.childNodes].filter((n) => n.nodeType === 3).map((n) => n.textContent).join('').trim();
  const inScope = (el) => cfg.converted || !!el.closest('[data-g]');
  const out = { cards: [], shadows: [], mono: [], upper: [], heroes: [], text: [] };
  const de = document.documentElement;
  out.overflow = { scrollWidth: de.scrollWidth, innerWidth: window.innerWidth };

  for (const el of document.querySelectorAll("[data-g^='GCard'],[data-g='GHeroCard']")) {
    if (!visible(el)) continue;
    const cs = getComputedStyle(el);
    const bw = ['Top', 'Right', 'Bottom', 'Left'].map((s) => (cs[`border${s}Style`] === 'none' ? 0 : parseFloat(cs[`border${s}Width`]) || 0));
    if (bw.some((w) => w > 0)) out.cards.push({ el: sel(el), border: bw.join(' ') });
  }
  for (const el of document.querySelectorAll('body *')) {
    if (!inScope(el) || el.closest("[data-g='GHeroCard'] *") || !visible(el)) continue;
    const bs = getComputedStyle(el).boxShadow;
    if (bs && bs !== 'none') out.shadows.push({ el: sel(el), boxShadow: bs, card: /^GCard|^GHeroCard$/.test(el.getAttribute('data-g') || '') });
  }
  const MONO_OK = 'code, kbd, pre, samp, [data-copy], [data-mono-ok]';
  for (const el of document.querySelectorAll('body *')) {
    const t = ownText(el);
    if (!t || !inScope(el) || !visible(el) || el.closest(MONO_OK)) continue;
    const cs = getComputedStyle(el);
    if (/mono/i.test(cs.fontFamily)) out.mono.push({ el: sel(el), text: t.slice(0, 40), font: cs.fontFamily.slice(0, 60) });
    const ls = parseFloat(cs.letterSpacing) || 0;
    const caps = /[A-Z].*[A-Z]/.test(t) && t === t.toUpperCase();
    if (cs.textTransform === 'uppercase' || (caps && ls >= 0.5)) out.upper.push({ el: sel(el), text: t.slice(0, 40), transform: cs.textTransform, letterSpacing: cs.letterSpacing });
  }
  for (const h of document.querySelectorAll("[data-g='GHeroCard']")) {
    if (visible(h)) out.heroes.push({ el: sel(h) });
  }
  // computed contrast for text inside kit components, outside heroes (heroes are sampled from pixels)
  for (const el of document.querySelectorAll('[data-g] *, [data-g]')) {
    const t = ownText(el);
    if (!t || !visible(el) || el.closest("[data-g='GHeroCard']")) continue;
    const cs = getComputedStyle(el);
    let bg = null; let unknown = false;
    for (let n = el; n; n = n.parentElement) {
      const ns = getComputedStyle(n);
      if (ns.backgroundImage && ns.backgroundImage !== 'none') { unknown = true; break; }
      const m = ns.backgroundColor.match(/rgba?\(([^)]+)\)/);
      if (m) { const p = m[1].split(/[\s,/]+/).filter(Boolean).map(parseFloat); if ((p[3] ?? 1) > 0.98) { bg = ns.backgroundColor; break; } }
    }
    if (unknown || !bg) continue;
    const size = parseFloat(cs.fontSize); const weight = parseInt(cs.fontWeight, 10) || 400;
    out.text.push({ el: sel(el), text: t.slice(0, 30), color: cs.color, bg, large: size >= 24 || (size >= 18.66 && weight >= 700) });
  }
  return out;
}

/** Make every text and dot numeral inside hero i transparent (or restore them), scroll it into view, return target rects. */
function heroTargets(i, hide) {
  const h = document.querySelectorAll("[data-g='GHeroCard']")[i];
  if (!h) return [];
  const ownText = (el) => [...el.childNodes].some((n) => n.nodeType === 3 && n.textContent.trim());
  const all = [h, ...h.querySelectorAll('*')];
  if (!hide) {
    for (const el of all) {
      if (!el.hasAttribute('data-g-audit-style')) continue;
      const s = el.getAttribute('data-g-audit-style');
      if (s) el.setAttribute('style', s); else el.removeAttribute('style');
      el.removeAttribute('data-g-audit-style');
    }
    return [];
  }
  h.scrollIntoView({ block: 'center' });
  const targets = [], touched = [];
  for (const el of all) {
    const dots = el.getAttribute('data-g') === 'GDotNumber';
    if (!dots && (!ownText(el) || el.closest("[data-g='GDotNumber']"))) continue;
    const r = el.getBoundingClientRect(); if (r.width < 2 || r.height < 2) continue;
    const cs = getComputedStyle(el);
    const size = parseFloat(cs.fontSize); const weight = parseInt(cs.fontWeight, 10) || 400;
    targets.push({ text: (el.textContent || '').trim().slice(0, 30), color: cs.color, numeral: dots || size >= 24 || (size >= 18.66 && weight >= 700), rect: { x: r.x, y: r.y, w: r.width, h: r.height } });
    touched.push([el, dots]);
  }
  // text goes transparent (the hero's own gradient stays); dot numerals are hidden outright
  for (const [el, dots] of touched) {
    el.setAttribute('data-g-audit-style', el.getAttribute('style') || '');
    if (dots && el !== h) el.style.setProperty('visibility', 'hidden', 'important');
    el.style.setProperty('color', 'transparent', 'important');
    el.style.setProperty('-webkit-text-fill-color', 'transparent', 'important');
    el.style.setProperty('text-shadow', 'none', 'important');
  }
  return targets;
}

/** FR-GP4.3: every control under data-ui=glados must be a kit control. */
function controlGuard(cfg) {
  const SEL = "button, a[href], input, select, textarea, [role=button], [role=tab], [role=radio], [role=switch], [role=checkbox], [role=menuitem], [role=option], [tabindex]:not([tabindex='-1'])";
  const sel = (el) => {
    const parts = [];
    for (let n = el, i = 0; n && n.nodeType === 1 && i < 4; n = n.parentElement, i++) {
      const c = typeof n.className === 'string' ? n.className.trim().split(/\s+/).filter(Boolean).slice(0, 3).join('.') : '';
      parts.unshift(n.tagName.toLowerCase() + (n.id ? '#' + n.id : '') + (c ? '.' + c : ''));
    }
    return parts.join(' > ');
  };
  if (document.documentElement.getAttribute('data-ui') !== 'glados') return { error: 'data-ui is not glados', failures: [] };
  const failures = []; let checked = 0, exempt = 0;
  const NATIVE = new Set(['BUTTON', 'A', 'INPUT', 'SELECT', 'TEXTAREA']);
  for (const el of document.querySelectorAll(SEL)) {
    if (el.tagName === 'INPUT' && el.type === 'hidden') continue;
    const ex = el.closest('[data-g-exempt]');
    if (ex) {
      if ((ex.getAttribute('data-g-exempt') || '').trim()) { exempt++; continue; }
      failures.push({ el: sel(el), why: 'data-g-exempt without a reason', classes: el.className && String(el.className) });
      continue;
    }
    if (el.tagName === 'A' && !el.getAttribute('class') && el.parentElement && el.parentElement.closest('p, li, td')) continue; // prose link
    checked++;
    const cl = [...el.classList];
    const classic = cl.filter((c) => cfg.classes.includes(c) || cfg.onControlOnly.includes(c) || (NATIVE.has(el.tagName) && cfg.utilityOnControl.includes(c)));
    if (classic.length) { failures.push({ el: sel(el), why: 'Classic control class', classes: classic.join(' ') }); continue; }
    let kit = false;
    for (let n = el, i = 0; n && i <= 2; n = n.parentElement, i++) {
      if (n.hasAttribute('data-g') || [...n.classList].some((c) => cfg.kitControlClasses.includes(c))) { kit = true; break; }
    }
    // A kit primitive's own inner controls (GHeatmap cells, GTable sort headers, GLegend items) carry a g-*
    // part class and sit inside the primitive's [data-g] root, sometimes more than 2 levels down.
    if (!kit && cl.some((c) => c.startsWith('g-')) && el.parentElement && el.parentElement.closest('[data-g]')) kit = true;
    if (!kit) failures.push({ el: sel(el), why: 'not a kit control (no data-g or .g-* within 2 levels)', classes: cl.join(' ') });
  }
  return { checked, exempt, failures };
}

const call = (fn, arg) => S.ev(`(${fn.toString()})(${JSON.stringify(arg)})`);

// ── judging ──────────────────────────────────────────────────────────────────────────────────
function judge(scan, { width }) {
  const fails = [];
  if (scan.overflow.scrollWidth > scan.overflow.innerWidth + 1) fails.push({ check: 'overflow', detail: `scrollWidth ${scan.overflow.scrollWidth} > ${scan.overflow.innerWidth} at ${width}` });
  for (const c of scan.cards) fails.push({ check: 'card-border', el: c.el, detail: c.border });
  for (const s of scan.shadows) for (const b of badShadows(s.boxShadow)) fails.push({ check: b.why === 'outer glow' ? 'outer-glow' : 'hued-shadow', el: s.el, detail: s.boxShadow.slice(0, 120) });
  for (const m of scan.mono) fails.push({ check: 'mono', el: m.el, detail: `"${m.text}" ${m.font}` });
  for (const u of scan.upper) fails.push({ check: 'uppercase', el: u.el, detail: `"${u.text}" ${u.transform} ${u.letterSpacing}` });
  if (scan.heroes.length > 2) fails.push({ check: 'hero-count', detail: `${scan.heroes.length} GHeroCard in view (max 2)` });
  for (const t of scan.text) {
    const fg = parseColor(t.color), bg = parseColor(t.bg);
    if (!fg || !bg) continue;
    const ratio = contrastRatio(composite(fg, bg), bg);
    const need = t.large ? 3 : 4.5;
    if (ratio < need) fails.push({ check: 'text-contrast', el: t.el, detail: `"${t.text}" ${ratio.toFixed(2)}:1 < ${need}` });
  }
  return fails;
}

/** AC00.12: hide each hero's text, screenshot, sample the gradient behind every text box. */
async function heroContrast(count) {
  const fails = [], samples = [];
  for (let i = 0; i < count; i++) {
    const targets = await S.ev(`(${heroTargets.toString()})(${i}, true)`);
    await sleep(150);
    const img = decodePng(await shot(null));
    await S.ev(`(${heroTargets.toString()})(${i}, false)`);
    for (const t of targets) {
      const ink = parseColor(t.color); if (!ink) continue;
      let worst = Infinity;
      for (let gy = 0; gy < 3; gy++) for (let gx = 0; gx < 6; gx++) {
        const x = Math.round(t.rect.x + 1 + ((t.rect.w - 2) * (gx + 0.5)) / 6), y = Math.round(t.rect.y + 1 + ((t.rect.h - 2) * (gy + 0.5)) / 3);
        if (x < 0 || y < 0 || x >= img.width || y >= img.height) continue;
        const o = (y * img.width + x) * 4; const bg = [img.data[o], img.data[o + 1], img.data[o + 2]];
        worst = Math.min(worst, contrastRatio(composite(ink, bg), bg));
      }
      if (worst === Infinity) continue;
      const need = t.numeral ? 3 : 4.5;
      samples.push({ hero: i, text: t.text, ratio: +worst.toFixed(2), need });
      if (worst < need) fails.push({ check: 'hero-contrast', detail: `hero ${i + 1} "${t.text}" ${worst.toFixed(2)}:1 < ${need}` });
    }
  }
  return { fails, samples };
}

// ── plugins ──────────────────────────────────────────────────────────────────────────────────
async function loadPlugins() {
  const dir = path.join(ROOT, 'scripts/g-audit');
  const map = {};
  for (const f of fs.existsSync(dir) ? fs.readdirSync(dir) : []) {
    if (!f.endsWith('.mjs') || f.startsWith('_')) continue;
    const mod = await import(pathToFileURL(path.join(dir, f)).href);
    for (const [route, steps] of Object.entries(mod.interactions ?? {})) (map[route] ??= []).push(...steps.map((s) => ({ ...s, plugin: f })));
    for (const [route, sels] of Object.entries(mod.pixelContrast ?? {})) (PIXEL[route] ??= []).push(...sels);
  }
  return map;
}

/** Plugin `pixelContrast` selectors by route (02 15.3 item 1). */
const PIXEL = {};
const pixelFor = (route) => PIXEL[route.split('?')[0]] ?? PIXEL[route] ?? null;

/**
 * 02 15.3 item 1: for each viewport down the page (at most 8), hide every in-view target's text (marks
 * hidden whole), screenshot, sample a 6 x 3 grid behind each target, restore. Selectors containing
 * "mark" are marks (3:1); the rest are ink (4.5:1, 3:1 when large).
 */
async function pixelContrast(sels, step) {
  const ink = sels.filter((x) => !/mark/.test(x)).join(', ') || LAB_PIXEL.ink;
  const mark = sels.filter((x) => /mark/.test(x)).join(', ') || LAB_PIXEL.mark;
  const fails = [], rows = [];
  const vh = await S.ev('window.innerHeight');
  const total = await S.ev('Math.max(document.documentElement.scrollHeight, document.body ? document.body.scrollHeight : 0)');
  const sheetOpen = await S.ev("!!document.querySelector('.lab2-sheet-layer')");
  const stops = sheetOpen ? [0] : Array.from({ length: Math.min(8, Math.ceil(total / vh)) }, (_, i) => i * vh);
  const seen = new Set();
  for (const y of stops) {
    if (!sheetOpen) { await S.ev(`window.scrollTo(0, ${y})`); await sleep(250); }
    const targets = await S.ev(`(${pixelTargetsInPage.toString()})(${JSON.stringify({ ink, mark, hide: true })})`);
    await sleep(120);
    const img = decodePng(await shot(null));
    await S.ev(`(${pixelTargetsInPage.toString()})({ hide: false })`);
    for (const t of sampleTargets(img, targets, { parseColor, composite, contrastRatio })) {
      const key = `${t.kind}|${t.text}|${t.ratio}`;
      if (seen.has(key)) continue;
      seen.add(key);
      rows.push(t);
      if (t.ratio < t.need) fails.push({ check: 'lab-contrast', step, detail: `${t.kind} "${t.text}" ${t.ratio}:1 < ${t.need}` });
    }
  }
  await S.ev('window.scrollTo(0, 0)');
  return { fails, rows };
}

/** 02 15.3 items 3, 4 and 6 at scroll 0, 1 and 2 viewports. */
async function labGlassChecks(step) {
  const fails = [];
  const vh = await S.ev('window.innerHeight');
  const glados = await S.ev("document.documentElement.getAttribute('data-ui') === 'glados'");
  for (let i = 0; i < 3; i++) {
    await S.ev(`window.scrollTo(0, ${i * vh})`);
    await sleep(1200); // the budget re-grants on the next frame after its IntersectionObserver fires
    const g = await S.ev(`(${glassScanInPage.toString()})()`);
    if (g.blurred > LAB_GLASS_BUDGET) fails.push({ check: 'lab-glass-budget', step, detail: `${g.blurred} blurred surfaces in view at scroll ${i} (max ${LAB_GLASS_BUDGET})` });
    for (const b of g.borders) fails.push({ check: 'card-border', step, el: b.el, detail: `Glass border ${b.border}` });
    for (const sh of g.shadows) for (const b of badShadows(sh.boxShadow)) fails.push({ check: b.why === 'outer glow' ? 'outer-glow' : 'hued-shadow', step, el: sh.el, detail: sh.boxShadow.slice(0, 120) });
    const heroes = await S.ev(`(${labHeroesInPage.toString()})()`);
    if (glados && heroes > 2) fails.push({ check: 'hero-count', step, detail: `${heroes} heroes or narrative cards in view at scroll ${i} (max 2)` });
  }
  await S.ev('window.scrollTo(0, 0)');
  return fails;
}

const ctx = () => ({
  ev: S.ev, send: S.send, sleep,
  click: (selector) => S.ev(`(() => { const el = document.querySelector(${JSON.stringify(selector)}); if (!el) throw new Error('no ' + ${JSON.stringify(selector)}); el.click(); return 1; })()`),
  press: async (key) => { await S.send('Input.dispatchKeyEvent', { type: 'keyDown', key }); await S.send('Input.dispatchKeyEvent', { type: 'keyUp', key }); },
});

// ── modes ────────────────────────────────────────────────────────────────────────────────────
async function maskRects(route, masks) {
  const rects = [];
  for (const expr of masks[route] ?? []) rects.push(...((await S.ev(expr).catch(() => [])) ?? []));
  return rects.map((r) => ({ x: +r.x.toFixed(1), y: +r.y.toFixed(1), w: +r.w.toFixed(1), h: +r.h.toFixed(1) }));
}

async function classicCapture(dir, themes, routes, masks = {}) {
  FREEZE = true;
  await setViewport(1440);
  const rows = [];
  for (const theme of themes) {
    await setAppearance({ ui: 'classic', accent: 'ice', mode: theme === 'light' ? 'light' : 'dark', theme });
    for (const route of routes) {
      const file = path.join(dir, theme, `${routeSlug(route)}.png`);
      const nav = await go(BASE + route);
      const a = await attrs().catch(() => ({}));
      const bad = nav.error || a.theme !== theme || (a.ui && a.ui !== 'classic');
      const mask = nav.error ? [] : await maskRects(route, masks);
      if (!nav.error) await shot(file);
      rows.push({ theme, route, file: path.relative(dir, file), status: nav.status, mask: mask.length ? mask : undefined, error: nav.error ?? (bad ? `data-theme=${a.theme} data-ui=${a.ui}` : undefined) });
      console.log(`${theme.padEnd(10)} ${route.padEnd(34)} ${nav.error ? 'ERROR ' + nav.error : bad ? `WRONG APPEARANCE theme=${a.theme} ui=${a.ui}` : `ok ${nav.status}`}`);
    }
  }
  return rows;
}

async function runClassic() {
  const capDir = opt('--capture'), cmpDir = opt('--compare');
  if (!capDir === !cmpDir) usage('--classic-baseline needs exactly one of --capture <dir> or --compare <dir>.');
  const masks = parseMasks(argv, { noDefaults: flag('--no-default-masks') });
  if (capDir) {
    const dir = path.resolve(capDir);
    const themes = list('--themes', THEME_IDS);
    const rows = await classicCapture(dir, themes, ROUTES, masks);
    let head = '';
    try { head = execFileSync('git', ['rev-parse', 'HEAD'], { cwd: ROOT, encoding: 'utf8' }); } catch { /* not a checkout */ }
    const manifestFile = path.join(dir, 'manifest.json');
    const now = new Date().toISOString();
    if (flag('--update')) {
      if (!fs.existsSync(manifestFile)) usage(`--update needs an existing baseline at ${manifestFile}.`);
      const m = JSON.parse(fs.readFileSync(manifestFile, 'utf8'));
      const key = (r) => `${r.theme} ${r.route}`;
      const fresh = new Map(rows.map((r) => [key(r), { ...r, url: BASE, captured: now }]));
      const kept = (m.rows ?? []).map((r) => fresh.get(key(r)) ?? r);
      for (const [k, r] of fresh) if (!kept.some((x) => key(x) === k)) kept.push(r);
      m.rows = kept;
      m.themes = [...new Set([...(m.themes ?? []), ...themes])];
      m.routes = [...new Set([...(m.routes ?? []), ...ROUTES])];
      (m.updates ??= []).push({ routes: ROUTES, themes, url: BASE, captured: now, git: head.trim() || undefined });
      fs.writeFileSync(manifestFile, JSON.stringify(m, null, 1));
    } else {
      fs.writeFileSync(manifestFile, JSON.stringify({ url: BASE, width: 1440, height: 900, themes, routes: ROUTES, captured: now, git: head.trim() || undefined, rows }, null, 1));
    }
    const errors = rows.filter((r) => r.error);
    console.log(`captured ${rows.length - errors.length}/${rows.length} shots into ${dir}`);
    return errors.length ? 1 : 0;
  }
  const base = path.resolve(cmpDir);
  const manifestFile = path.join(base, 'manifest.json');
  if (!fs.existsSync(manifestFile)) usage(`No baseline manifest at ${manifestFile}; capture first.`);
  const manifest = JSON.parse(fs.readFileSync(manifestFile, 'utf8'));
  if (manifest.url !== BASE) console.log(`note: baseline was captured from ${manifest.url}, comparing against ${BASE}`);
  const out = path.resolve(opt('--out', `${base}-compare`));
  const themes = list('--themes', manifest.themes);
  const routes = opt('--routes') || envRoutes.length ? ROUTES : manifest.routes;
  const threshold = Number(opt('--threshold', '1'));
  const tolerance = Number(opt('--tolerance', '2'));
  const rows = await classicCapture(out, themes, routes, masks);
  const baseMask = new Map((manifest.rows ?? []).map((r) => [`${r.theme} ${r.route}`, r.mask ?? []]));
  const report = [];
  for (const r of rows) {
    const was = path.join(base, r.file), now = path.join(out, r.file);
    if (r.error) { report.push({ ...r, pass: false }); continue; }
    if (!fs.existsSync(was)) { report.push({ ...r, pass: false, error: 'no baseline shot' }); console.log(`${r.theme.padEnd(10)} ${r.route.padEnd(34)} NO BASELINE`); continue; }
    const a = decodePng(fs.readFileSync(was)), b = decodePng(fs.readFileSync(now));
    const rects = [...(baseMask.get(`${r.theme} ${r.route}`) ?? []), ...(r.mask ?? [])];
    if (rects.length) { maskImage(a, rects); maskImage(b, rects); }
    const d = diffImages(a, b, { tolerance });
    const pct = +(d.ratio * 100).toFixed(3);
    const pass = pct <= threshold;
    if (!pass || d.sizeChanged) { const f = path.join(out, 'diff', r.file); fs.mkdirSync(path.dirname(f), { recursive: true }); fs.writeFileSync(f, encodePng(d.image)); }
    report.push({ theme: r.theme, route: r.route, diffPct: pct, sizeChanged: d.sizeChanged, masked: rects.length || undefined, pass });
    console.log(`${r.theme.padEnd(10)} ${r.route.padEnd(34)} ${String(pct).padStart(7)}%  ${pass ? 'PASS' : 'FAIL'}${d.sizeChanged ? ' (size changed)' : ''}${rects.length ? ` (masked ${rects.length})` : ''}`);
  }
  fs.writeFileSync(path.join(out, 'report.json'), JSON.stringify({ baseline: base, url: BASE, threshold, tolerance, compared: new Date().toISOString(), report }, null, 1));
  const failed = report.filter((r) => !r.pass);
  console.log(`${report.length - failed.length}/${report.length} within ${threshold}% · report ${path.join(out, 'report.json')}`);
  return failed.length ? 1 : 0;
}

async function auditRoute(route, { width, converted, plugins, full, file }) {
  const res = { route, width, fails: [], warnings: [] };
  const nav = await go(BASE + route);
  if (nav.error) { res.fails.push({ check: 'load', detail: nav.error }); return res; }
  res.status = nav.status;
  if (nav.status >= 400) res.fails.push({ check: 'load', detail: `HTTP ${nav.status}` });
  if (file) { await shot(file); if (full) await shot(file.replace(/\.png$/, '.full.png'), true); }
  const scan = await call(pageScan, { converted });
  res.heroes = scan.heroes.length;
  res.fails.push(...judge(scan, { width }));
  if (scan.heroes.length) {
    const hc = await heroContrast(Math.min(scan.heroes.length, 4));
    res.heroContrast = hc.samples; res.fails.push(...hc.fails);
  }
  if (converted) {
    const g = await call(controlGuard, CLASSIC);
    res.controls = { checked: g.checked, exempt: g.exempt };
    if (g.error) res.fails.push({ check: 'controls', detail: g.error });
    for (const f of g.failures) res.fails.push({ check: 'controls', el: f.el, detail: `${f.why}${f.classes ? ` [${f.classes}]` : ''}` });
  }
  const px = pixelFor(route);
  if (px) res.fails.push(...(await pixelContrast(px, 'load')).fails);
  for (const step of plugins[route.split('?')[0]] ?? plugins[route] ?? plugins[readyEntryFor(route, REG.routes)] ?? []) {
    try { await step.run(ctx()); await sleep(400); } catch (e) { res.fails.push({ check: 'interaction', detail: `${step.plugin} "${step.name}": ${e.message}` }); continue; }
    if (px) res.fails.push(...(await pixelContrast(px, step.name)).fails);
    const s2 = await call(pageScan, { converted });
    for (const f of judge(s2, { width })) res.fails.push({ ...f, step: step.name });
    if (converted) {
      const g = await call(controlGuard, CLASSIC);
      for (const f of g.failures) res.fails.push({ check: 'controls', el: f.el, detail: `${f.why}${f.classes ? ` [${f.classes}]` : ''}`, step: step.name });
    }
  }
  return res;
}

async function runGlados() {
  const accents = list('--accents', ['ice', 'ember']);
  const modes = list('--modes', ['dark', 'light']);
  const widths = list('--widths', ['1440', '1024', '390']).map(Number);
  const out = path.resolve(opt('--out', path.join(ROOT, 'docs/projects/glados-ui/shots/audit')));
  const full = flag('--full');
  const plugins = await loadPlugins();
  const modelBuilt = fs.existsSync(path.join(ROOT, 'lib/appearance.ts'));
  if (!REG.source) console.log('G_READY: no registry (lib/appearance.ts or lib/g-ready.ts); treated as empty, the control guard has nothing to check.');
  else console.log(`G_READY (${REG.source}): ${REG.routes.join(' ') || '(empty)'}`);
  const results = [];
  for (const accent of accents) for (const mode of modes) {
    await setAppearance({ ui: 'glados', accent, mode });
    for (const width of widths) {
      await setViewport(width);
      for (const route of ROUTES) {
        const converted = routeInReady(route, REG.routes);
        const file = path.join(out, `${accent}-${mode}`, String(width), `${routeSlug(route)}.png`);
        const r = await auditRoute(route, { width, converted, plugins, full, file });
        const a = await attrs().catch(() => ({}));
        if (a.ui !== 'glados' || a.accent !== accent || a.mode !== mode) {
          const msg = `appearance not applied: data-ui=${a.ui} data-accent=${a.accent} data-mode=${a.mode}`;
          (modelBuilt ? r.fails : r.warnings).push({ check: 'appearance', detail: msg });
        }
        Object.assign(r, { accent, mode, converted, shot: path.relative(out, file) });
        results.push(r);
        console.log(`${accent}-${mode} ${String(width).padStart(4)} ${route.padEnd(34)} ${converted ? 'G' : '-'} heroes ${r.heroes ?? '?'}  ${r.fails.length ? `FAIL ${r.fails.length}: ${[...new Set(r.fails.map((f) => f.check))].join(', ')}` : 'ok'}${r.warnings.length ? ' (warn: ' + r.warnings[0].detail + ')' : ''}`);
      }
    }
  }
  fs.mkdirSync(out, { recursive: true });
  fs.writeFileSync(path.join(out, 'report.json'), JSON.stringify({ url: BASE, at: new Date().toISOString(), gReady: REG, accents, modes, widths, results }, null, 1));
  const failed = results.filter((r) => r.fails.length);
  for (const r of failed) for (const f of r.fails.slice(0, 12)) console.log(`  ${r.accent}-${r.mode} ${r.width} ${r.route} · ${f.check}${f.step ? ` (after "${f.step}")` : ''} · ${f.el ?? ''} ${f.detail ?? ''}`);
  console.log(`${results.length - failed.length}/${results.length} shots clean · report ${path.join(out, 'report.json')}`);
  return failed.length ? 1 : 0;
}

// ── Content Lab v2 styles (02 15.3 item 7) ──────────────────────────────────────────────────
async function labRoute(route, { width, plugins, file, glados }) {
  const res = { route, width, fails: [], warnings: [] };
  const nav = await go(BASE + route);
  if (nav.error) { res.fails.push({ check: 'load', detail: nav.error }); return res; }
  if (nav.status >= 400) res.fails.push({ check: 'load', detail: `HTTP ${nav.status}` });
  if (file) await shot(file);
  const auroras = await S.ev("document.querySelectorAll('.lab2-aurora').length");
  if (auroras !== 1) res.fails.push({ check: 'lab-aurora', detail: `${auroras} .lab2-aurora on the page (want 1)` });
  const checks = async (step) => {
    const scan = await call(pageScan, { converted: glados });
    for (const f of judge(scan, { width })) if (glados || !['card-border', 'hero-count', 'mono', 'uppercase'].includes(f.check)) res.fails.push({ ...f, step });
    res.fails.push(...(await labGlassChecks(step)).filter((f) => glados || f.check === 'lab-glass-budget'));
    res.fails.push(...(await pixelContrast(pixelFor(route) ?? [LAB_PIXEL.ink, LAB_PIXEL.mark], step)).fails);
    if (glados) {
      const g = await call(controlGuard, CLASSIC);
      for (const f of g.failures) res.fails.push({ check: 'controls', el: f.el, detail: `${f.why}${f.classes ? ` [${f.classes}]` : ''}`, step });
    }
  };
  await checks('load');
  for (const step of plugins[route.split('?')[0]] ?? plugins[route] ?? []) {
    try { await step.run(ctx()); await sleep(500); } catch (e) { res.fails.push({ check: 'interaction', detail: `${step.plugin} "${step.name}": ${e.message}` }); continue; }
    await checks(step.name);
  }
  return res;
}

async function reducedTransparency(route) {
  const mode = await S.ev("document.documentElement.getAttribute('data-mode') === 'light' ? 'light' : 'dark'").catch(() => 'dark');
  await S.send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: mode }, { name: 'prefers-reduced-transparency', value: 'reduce' }] });
  await go(BASE + route);
  const r = await S.ev(`(${reducedGlassInPage.toString()})()`);
  await S.send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: mode }] });
  return r;
}

async function runLabStyles() {
  const pick = list('--styles', null);
  const styles = pick ? LAB_STYLES.filter((x) => pick.includes(x.id)) : LAB_STYLES;
  const widths = list('--widths', ['1440', '1024', '390']).map(Number);
  const routes = list('--routes', LAB_ROUTES).map((r) => (r.includes('sample=') ? r : `${r}${r.includes('?') ? '&' : '?'}sample=1`));
  const out = path.resolve(opt('--out', path.join(ROOT, 'docs/projects/glados-ui/shots/audit/lab')));
  const plugins = await loadPlugins();
  const results = [];
  let emulatedSaid = false;
  for (const st of styles) {
    await setAppearance({ ui: st.ui, accent: st.accent, mode: st.mode, theme: st.theme });
    const glados = st.ui === 'glados';
    for (const width of widths) {
      await setViewport(width);
      for (const route of routes) {
        const file = path.join(out, st.id, String(width), `${routeSlug(route.split('?')[0])}.png`);
        const r = await labRoute(route, { width, plugins, file, glados });
        if (width === widths[0]) {
          const rt = await reducedTransparency(route);
          if (!emulatedSaid) { console.log(`reduced-transparency emulated: ${rt.emulated ? 'yes' : 'no'}`); emulatedSaid = true; }
          if (!rt.emulated) r.warnings.push({ check: 'lab-reduced-transparency', detail: 'Chrome did not honour the emulated feature' });
          for (const f of rt.fails) r.fails.push({ check: 'lab-reduced-transparency', el: f.el, detail: `backdrop ${f.backdrop} alpha ${f.alpha}` });
        }
        Object.assign(r, { style: st.id, shot: path.relative(out, file) });
        results.push(r);
        console.log(`${st.id.padEnd(20)} ${String(width).padStart(4)} ${route.padEnd(26)} ${r.fails.length ? `FAIL ${r.fails.length}: ${[...new Set(r.fails.map((f) => f.check))].join(', ')}` : 'ok'}`);
      }
    }
  }
  fs.mkdirSync(out, { recursive: true });
  fs.writeFileSync(path.join(out, 'report.json'), JSON.stringify({ url: BASE, at: new Date().toISOString(), styles: styles.map((x) => x.id), widths, routes, results }, null, 1));
  const failed = results.filter((r) => r.fails.length);
  for (const r of failed) for (const f of r.fails.slice(0, 12)) console.log(`  ${r.style} ${r.width} ${r.route} · ${f.check}${f.step && f.step !== 'load' ? ` (after "${f.step}")` : ''} · ${f.el ?? ''} ${f.detail ?? ''}`);
  console.log(`${results.length - failed.length}/${results.length} shots clean · report ${path.join(out, 'report.json')}`);
  return failed.length ? 1 : 0;
}

const FIXTURES = {
  'classic-button': '<!doctype html><html data-ui="glados" data-accent="ice" data-mode="dark"><body><main><h1>Fixture</h1><p>One Classic button under GladOS.</p><button type="button" class="c-btn c-btn-primary">Save</button></main></body></html>',
  'kit-button': '<!doctype html><html data-ui="glados" data-accent="ice" data-mode="dark"><body><main><h1>Fixture</h1><p>The same page from the kit, with a <a href="#docs">prose link</a>.</p><button type="button" data-g="GButton" class="g-btn">Save</button></main></body></html>',
};

async function runControls() {
  const fx = opt('--fixture');
  if (fx) {
    const html = FIXTURES[fx] ?? (fs.existsSync(fx) ? fs.readFileSync(fx, 'utf8') : null);
    if (!html) usage(`Unknown fixture "${fx}" (built in: ${Object.keys(FIXTURES).join(', ')}, or a file path).`);
    await setViewport(1440);
    await S.send('Page.navigate', { url: `data:text/html;charset=utf-8,${encodeURIComponent(html)}` });
    await sleep(400);
    const g = await call(controlGuard, CLASSIC);
    if (g.error) { console.log(`fixture ${fx}: ERROR ${g.error}`); return 1; }
    for (const f of g.failures) console.log(`fixture ${fx} · ${f.el} · ${f.why}${f.classes ? ` [${f.classes}]` : ''}`);
    console.log(`fixture ${fx}: ${g.checked} controls checked, ${g.failures.length} Classic`);
    return g.failures.length ? 1 : 0;
  }
  const routes = ROUTES.filter((r) => routeInReady(r, REG.routes));
  console.log(REG.source ? `G_READY (${REG.source}): ${REG.routes.join(' ') || '(empty)'}` : 'G_READY: no registry (lib/appearance.ts or lib/g-ready.ts); treated as empty.');
  if (!routes.length) { console.log('No audited route is in G_READY: nothing to guard.'); return 0; }
  const plugins = await loadPlugins();
  await setAppearance({ ui: 'glados', accent: 'ice', mode: 'dark' });
  await setViewport(1440);
  let failures = 0;
  for (const route of routes) {
    const nav = await go(BASE + route);
    if (nav.error) { console.log(`${route}: ERROR ${nav.error}`); failures++; continue; }
    const runs = [{ name: 'load' }, ...(plugins[route.split('?')[0]] ?? plugins[readyEntryFor(route, REG.routes)] ?? [])];
    for (const step of runs) {
      if (step.run) { try { await step.run(ctx()); await sleep(400); } catch (e) { console.log(`${route} · ${step.plugin} "${step.name}": ${e.message}`); failures++; continue; } }
      const g = await call(controlGuard, CLASSIC);
      if (g.error) { console.log(`${route} (${step.name}): ERROR ${g.error}`); failures++; continue; }
      for (const f of g.failures) console.log(`${route} (${step.name}) · ${f.el} · ${f.why}${f.classes ? ` [${f.classes}]` : ''}`);
      failures += g.failures.length;
      console.log(`${route} (${step.name}): ${g.checked} controls, ${g.failures.length} Classic`);
    }
  }
  return failures ? 1 : 0;
}

// ── main ─────────────────────────────────────────────────────────────────────────────────────
const MODE = flag('--classic-baseline') ? 'classic' : flag('--glados') ? 'glados' : flag('--controls') ? 'controls' : flag('--lab-styles') ? 'lab' : null;
if (!MODE || flag('--help')) usage();
const browser = await startBrowser();
let code = 1;
try {
  S = await cdp();
  code = MODE === 'classic' ? await runClassic() : MODE === 'glados' ? await runGlados() : MODE === 'lab' ? await runLabStyles() : await runControls();
} catch (e) {
  console.error(e);
} finally {
  S?.close();
  browser.stop();
}
process.exit(code);
