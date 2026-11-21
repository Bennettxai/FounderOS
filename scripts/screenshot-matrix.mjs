// Screenshot matrix (SPEC §10.2, Phase 4): every route in scripts/audit-routes.mjs × every theme in
// lib/theme.ts × widths 1440, 1024, 390. One full-page screenshot per case (capped at 6000px tall) and a
// layout scan:
//   hscroll      the page scrolls sideways (document wider than the viewport), with the widest offenders
//   innerscroll  a region at least 70% of the viewport wide scrolls sideways on its own (tables, pre and
//                [data-matrix-scroll] are intended scrollers and skipped)
//   overlap      two leaf text boxes from different elements intersect by more than 4px each way
//                (text inside [data-matrix-skip], a pannable canvas under its own HUD, is not compared)
//   clipped      a leaf text element whose text is cut off by its own box (overflow hidden, no ellipsis)
// Exits 1 on any hscroll, innerscroll or overlap finding; clipped text is reported for review (ellipsis by design is skipped).
// Every non-GET /api request is failed at the network layer, so nothing is spent or written.
//
// Usage:
//   GLADOS_URL=http://127.0.0.1:4000 node scripts/screenshot-matrix.mjs <outDir> [themes,comma|all] [widths,comma] [routes,comma|all]
// Env:
//   GLADOS_URL  the instance to shoot (required; never a hard-coded host)
//   CHROME      Chrome or chrome-headless-shell binary (default: macOS Google Chrome), started headless here
//   CDP_URL     use an already running Chrome (e.g. http://127.0.0.1:9333) instead of starting one
//   WAIT        ms to settle after load (default 2500)
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import net from 'node:net';
import { spawn } from 'node:child_process';
import { THEME_IDS, routesFromEnv } from './audit-routes.mjs';
import { setThemeExpr } from './theme-ground.mjs';

const BASE = (process.env.GLADOS_URL ?? '').replace(/\/$/, '');
const [, , outArg, themesArg = 'all', widthsArg = '1440,1024,390', routesArg = 'all'] = process.argv;
if (!BASE || !outArg) { console.error('Usage: GLADOS_URL=http://127.0.0.1:4000 node scripts/screenshot-matrix.mjs <outDir> [themes] [widths] [routes]'); process.exit(2); }
const OUT = outArg;
fs.mkdirSync(OUT, { recursive: true });
const themes = themesArg === 'all' ? THEME_IDS : themesArg.split(',');
const widths = widthsArg.split(',').map(Number);
const routes = routesFromEnv(routesArg);
const WAIT = Number(process.env.WAIT ?? 2500);
const CHROME = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/* ── Chrome over CDP ── */
async function freePort() {
  return new Promise((res) => { const s = net.createServer(); s.listen(0, () => { const { port } = s.address(); s.close(() => res(port)); }); });
}
let chrome = null;
async function cdpBase() {
  if (process.env.CDP_URL) return process.env.CDP_URL.replace(/\/$/, '');
  const port = await freePort();
  const profile = fs.mkdtempSync(path.join(os.tmpdir(), 'matrix-profile-'));
  chrome = spawn(CHROME, ['--headless=new', `--remote-debugging-port=${port}`, `--user-data-dir=${profile}`, '--hide-scrollbars', '--no-first-run', '--no-default-browser-check', 'about:blank'], { stdio: 'ignore' });
  const base = `http://127.0.0.1:${port}`;
  for (let i = 0; i < 100; i += 1) { try { await (await fetch(`${base}/json/version`)).json(); return base; } catch { await sleep(100); } }
  throw new Error(`Chrome did not start (${CHROME})`);
}
const root = await cdpBase();
const blocked = [];

/** A fresh tab per case: a page that wedges its renderer (a busy canvas) cannot take the next case with it. */
async function openTab(theme, width) {
  const target = await (await fetch(`${root}/json/new?about:blank`, { method: 'PUT' })).json();
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  let id = 0;
  const pending = new Map();
  ws.onmessage = (m) => {
    const d = JSON.parse(m.data);
    if (d.id && pending.has(d.id)) { const p = pending.get(d.id); pending.delete(d.id); d.error ? p.rej(new Error(JSON.stringify(d.error))) : p.res(d.result); return; }
    // the safety rail: every non-GET /api request is failed before it leaves the browser (SPEC §3.1.7)
    if (d.method === 'Fetch.requestPaused') {
      const { requestId, request } = d.params;
      if (request.method === 'GET' || request.method === 'HEAD') send('Fetch.continueRequest', { requestId }).catch(() => {});
      else { blocked.push(`${request.method} ${new URL(request.url).pathname}`); send('Fetch.failRequest', { requestId, errorReason: 'BlockedByClient' }).catch(() => {}); }
    }
  };
  await new Promise((r, j) => { ws.onopen = r; ws.onerror = j; });
  // every call has a deadline, so one page that never settles cannot stall the matrix
  const send = (method, params = {}, ms = 45000) => new Promise((res, rej) => {
    const i = ++id;
    const t = setTimeout(() => { pending.delete(i); rej(new Error(`${method} timed out after ${ms}ms`)); }, ms);
    pending.set(i, { res: (v) => { clearTimeout(t); res(v); }, rej: (e) => { clearTimeout(t); rej(e); } });
    ws.send(JSON.stringify({ id: i, method, params }));
  });
  const ev = async (expression) => {
    const r = await send('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true });
    if (r.exceptionDetails) throw new Error(r.exceptionDetails.exception?.description ?? r.exceptionDetails.text);
    return r.result.value;
  };
  await send('Fetch.enable', { patterns: [{ urlPattern: '*/api/*', requestStage: 'Request' }] });
  await send('Page.enable');
  await send('Runtime.enable');
  // a background tab never paints and a capture of it hangs
  await send('Page.bringToFront');
  await send('Emulation.setDeviceMetricsOverride', { width, height: 900, deviceScaleFactor: 1, mobile: width < 600 });
  await send('Page.addScriptToEvaluateOnNewDocument', { source: `try{localStorage.setItem('glados-theme',${JSON.stringify(theme)})}catch(e){}` });
  const close = async () => { try { ws.close(); } catch {} await fetch(`${root}/json/close/${target.id}`).catch(() => {}); };
  return { send, ev, close };
}

/* ── the layout scan ── */
const SCAN = `(() => {
  const vw = document.documentElement.clientWidth;
  const sw = Math.max(document.documentElement.scrollWidth, document.body.scrollWidth);
  const tag = (el) => { const c = (el.className && typeof el.className === 'string') ? el.className.trim().split(/\\s+/).slice(0, 3).join('.') : ''; return el.tagName.toLowerCase() + (el.id ? '#' + el.id : '') + (c ? '.' + c : ''); };
  // painted at all: an ancestor at opacity 0 (a closed card waiting to fade in) hides it too
  const visible = (el, cs) => cs.visibility !== 'hidden' && cs.display !== 'none' && Number(cs.opacity) > 0.05 &&
    (typeof el.checkVisibility !== 'function' || el.checkVisibility({ opacityProperty: true, visibilityProperty: true }));
  // inside a horizontally scrolling or clipping ancestor: its own overflow is contained
  const contained = (el) => { for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) { const ox = getComputedStyle(p).overflowX; if (ox === 'auto' || ox === 'scroll' || ox === 'hidden' || ox === 'clip') return true; } return false; };
  const findings = [];
  if (sw > vw + 1) {
    const off = [];
    for (const el of document.querySelectorAll('body *')) {
      const r = el.getBoundingClientRect(); if (r.width < 1 || r.height < 1) continue;
      if (r.right > vw + 1 && !contained(el)) { const cs = getComputedStyle(el); if (!visible(el, cs) || cs.position === 'fixed') continue; off.push({ el: tag(el), right: Math.round(r.right), width: Math.round(r.width) }); }
    }
    off.sort((a, b) => a.width - b.width);
    findings.push({ kind: 'hscroll', by: sw - vw, offenders: off.slice(0, 6) });
  }
  // a page-wide region that scrolls sideways on its own (the shell's main, a console column grid)
  for (const el of document.querySelectorAll('body *')) {
    const cs = getComputedStyle(el); if (cs.overflowX !== 'auto' && cs.overflowX !== 'scroll') continue;
    if (el.clientWidth < vw * 0.7 || el.scrollWidth <= el.clientWidth + 1 || !visible(el, cs)) continue;
    if (el.matches('[data-matrix-scroll], table, pre, code') || el.querySelector(':scope > table')) continue;
    findings.push({ kind: 'innerscroll', el: tag(el), by: el.scrollWidth - el.clientWidth });
  }
  // the part of a text box that is actually painted: clipped by itself and every ancestor that hides overflow
  const visibleRect = (el, r) => {
    let l = r.left, t = r.top, rt = r.right, b = r.bottom;
    for (let p = el; p && p !== document.documentElement; p = p.parentElement) {
      const cs = getComputedStyle(p);
      if (cs.overflowX !== 'visible' || cs.overflowY !== 'visible') { const q = p.getBoundingClientRect(); l = Math.max(l, q.left); t = Math.max(t, q.top); rt = Math.min(rt, q.right); b = Math.min(b, q.bottom); }
      if (cs.position === 'fixed') break;
    }
    return rt - l > 0 && b - t > 0 ? { left: l, top: t, right: rt, bottom: b, width: rt - l, height: b - t } : null;
  };
  // the fixed layer a box paints in (sidebar, dock, a full-screen console): boxes in different layers are
  // stacked on purpose, so only boxes that share a layer can collide
  const layerOf = (el) => { for (let p = el; p && p !== document.body; p = p.parentElement) if (getComputedStyle(p).position === 'fixed') return p; return null; };
  // leaf text boxes
  const leaves = [];
  for (const el of document.querySelectorAll('body *')) {
    if (el.closest('svg, canvas, video, [aria-hidden="true"], .sr-only, [data-matrix-skip]')) continue;
    let text = ''; for (const n of el.childNodes) if (n.nodeType === 3) text += n.textContent; text = text.trim(); if (!text) continue;
    const cs = getComputedStyle(el); if (!visible(el, cs)) continue;
    const range = document.createRange(); range.selectNodeContents(el); const r0 = range.getBoundingClientRect();
    const r = visibleRect(el, r0);
    if (!r || r.width < 2 || r.height < 2) continue;
    // one box per line: a wrapped label's bounding box covers the whole paragraph, its lines do not
    const lines = [...range.getClientRects()].map((q) => visibleRect(el, q)).filter((q) => q && q.width >= 2 && q.height >= 2);
    leaves.push({ el, r, lines, text, cs, layer: layerOf(el) });
    // clipped: content wider than the box, hidden overflow, no ellipsis
    if ((cs.overflowX === 'hidden' || cs.overflowX === 'clip') && cs.textOverflow !== 'ellipsis' && el.scrollWidth > el.clientWidth + 2 && el.clientWidth > 0) findings.push({ kind: 'clipped', el: tag(el), text: text.slice(0, 40) });
  }
  // stacking: a leaf hidden behind an overlay (dialog, dock, drawer) is not an overlap between two leaves
  const topAt = (x, y) => document.elementFromPoint(x, y);
  for (let i = 0; i < leaves.length; i += 1) {
    for (let j = i + 1; j < leaves.length; j += 1) {
      const a = leaves[i], b = leaves[j];
      if (a.layer !== b.layer || a.el.contains(b.el) || b.el.contains(a.el)) continue;
      const meet = (p, q) => { const w = Math.min(p.right, q.right) - Math.max(p.left, q.left); const h = Math.min(p.bottom, q.bottom) - Math.max(p.top, q.top); return w > 4 && h > 4 ? { w, h, cx: Math.max(p.left, q.left) + w / 2, cy: Math.max(p.top, q.top) + h / 2 } : null; };
      if (!meet(a.r, b.r)) continue;
      let hit = null;
      for (const p of a.lines) { for (const q of b.lines) { hit = meet(p, q); if (hit) break; } if (hit) break; }
      if (!hit) continue;
      const { w, h, cx, cy } = hit;
      if (cy < 0 || cy > innerHeight) {
        // off screen: only report when neither box sits in a fixed layer (overlays are checked on screen)
        if (a.cs.position === 'fixed' || b.cs.position === 'fixed') continue;
      } else {
        // something else covers the spot (a dialog, the dock): not a collision between these two
        const t = topAt(cx, cy); if (!t || !(a.el.contains(t) || b.el.contains(t))) continue;
      }
      findings.push({ kind: 'overlap', a: tag(a.el), at: a.text.slice(0, 30), b: tag(b.el), bt: b.text.slice(0, 30), w: Math.round(w), h: Math.round(h) });
      if (findings.length > 40) return { vw, sw, height: document.documentElement.scrollHeight, findings };
    }
  }
  return { vw, sw, height: document.documentElement.scrollHeight, findings };
})()`;

const report = [];
let failures = 0;
for (const theme of themes) {
  for (const width of widths) {
    for (const route of routes) {
      const slug = route === '/' ? 'home' : route.replace(/^\//, '').replace(/[/?=&]/g, '_');
      const name = `${theme}/${slug}-${width}`;
      fs.mkdirSync(path.join(OUT, theme), { recursive: true });
      let tab = null;
      try {
        tab = await openTab(theme, width);
        const { send, ev } = tab;
        await send('Page.navigate', { url: BASE + route });
        await sleep(WAIT);
        await ev(setThemeExpr(theme)); // data-theme and the data-ground the glass light reads
        await sleep(300);
        const scan = await ev(SCAN);
        // full page beyond the viewport, so vh-sized sections keep their real size; a capture that
        // hangs is reported and the scan still counts
        const h = Math.min(6000, Math.max(900, scan.height ?? 900));
        let shotNote = '';
        try {
          const shot = await send('Page.captureScreenshot', { format: 'png', captureBeyondViewport: true, clip: { x: 0, y: 0, width, height: h, scale: 1 } }, 30000);
          fs.writeFileSync(path.join(OUT, `${name}.png`), Buffer.from(shot.data, 'base64'));
        } catch (e) {
          // a busy canvas can wedge the beyond-viewport capture: retry in a fresh tab, viewport only
          await tab.close();
          tab = await openTab(theme, width);
          try {
            await tab.send('Page.navigate', { url: BASE + route });
            await sleep(WAIT);
            await tab.ev(setThemeExpr(theme));
            await sleep(300);
            const shot = await tab.send('Page.captureScreenshot', { format: 'png' }, 20000);
            fs.writeFileSync(path.join(OUT, `${name}.png`), Buffer.from(shot.data, 'base64'));
            shotNote = ' (viewport screenshot only)';
          } catch (e2) { shotNote = ` (no screenshot: ${e2.message.slice(0, 60)})`; scan.shotError = e2.message; }
        }
        const hard = scan.findings.filter((f) => f.kind === 'hscroll' || f.kind === 'innerscroll' || f.kind === 'overlap');
        failures += hard.length;
        report.push({ theme, width, route, ...scan });
        const counts = ['hscroll', 'innerscroll', 'overlap', 'clipped'].map((k) => `${scan.findings.filter((f) => f.kind === k).length} ${k}`).join(', ');
        console.log(`${name}: ${counts}${scan.sw > scan.vw + 1 ? ` (+${scan.sw - scan.vw}px)` : ''}${shotNote}`);
      } catch (e) {
        failures += 1;
        report.push({ theme, width, route, error: e.message });
        console.log(`${name}: ERROR ${e.message.slice(0, 100)}`);
      } finally {
        if (tab) await tab.close();
      }
    }
  }
}
fs.writeFileSync(path.join(OUT, 'report.json'), JSON.stringify({ base: BASE, blocked, report }, null, 1));
console.log(`\n${report.length} cases, ${failures} hscroll/overlap findings, ${blocked.length} non-GET /api requests blocked. Report: ${path.join(OUT, 'report.json')}`);
if (chrome) chrome.kill();
process.exit(failures ? 1 : 0);
