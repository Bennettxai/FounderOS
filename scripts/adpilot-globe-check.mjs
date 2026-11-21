// AdPilot globe check (SPEC S11 §9, §10): the pixel and interaction acceptance tests that need a browser.
//   AC11.2  Console: radial luminance profile has one maximum, at the rim (no second ring)
//   AC11.3  Frost and Grey Frost: no red pixel (saturation > 0.5) in the globe slot; rim is blue
//   AC11.4  White: no black disc (body luma > 0.85), no pink fog (saturation at 1.15R < 0.35)
//   AC11.5  live theme switch re-colours the same canvas element within a couple of frames
//   AC11.6  clicking a pin centres it (tip within 12px) and opens the City card; rows sum to the totals
//   AC11.7  a City card row selects that campaign (?campaign= updates)
//   AC11.8  no chip overlaps another chip or a [data-ap-floater] rect, at 1440 and 1024
//   AC11.9  reduced motion: no frames while idle, two captures 1s apart are identical
//   AC11.10 scrolled out of view: no frames
//   AC11.11 frame cost at 1440x900, DPR 2 (budget 4ms avg on Apple silicon)
//   AC11.12 keyboard: focus the globe, Left rotates, Enter opens the nearest city, Escape closes
//   AC11.13 touch: a vertical swipe over the globe scrolls the page
// Every non-GET /api request is failed (SPEC §3.1.7), so nothing is spent. Captures land in <outdir>.
// Start Chrome first (see theme-audit.mjs), then:
//   BASE=http://127.0.0.1:4000 node scripts/adpilot-globe-check.mjs [outdir]
import fs from 'node:fs';
import path from 'node:path';
import { cdp } from './audit-routes.mjs';
import { setThemeExpr } from './theme-ground.mjs';

const BASE = process.env.BASE ?? 'http://127.0.0.1:4000';
const OUT = process.argv[2] ?? `${process.env.S ?? '.'}/audit/globe`;
fs.mkdirSync(OUT, { recursive: true });
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const { send, ev, close } = await cdp();
const results = [];
const record = (id, pass, detail) => {
  results.push({ id, pass, detail });
  console.log(`${pass ? 'PASS' : 'FAIL'} ${id} ${detail}`);
};

async function open(theme, { w = 1440, h = 900, dpr = 1, reduced = false, query = '' } = {}) {
  await send('Emulation.setDeviceMetricsOverride', { width: w, height: h, deviceScaleFactor: dpr, mobile: false });
  await send('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-reduced-motion', value: reduced ? 'reduce' : 'no-preference' }] });
  const url = `${BASE}/adpilot?globeperf=1${query}`;
  await send('Page.navigate', { url });
  await sleep(400);
  await ev(`localStorage.setItem('glados-theme', ${JSON.stringify(theme)})`);
  await send('Page.navigate', { url });
  for (let i = 0; i < 120; i++) {
    await sleep(250);
    if (await ev(`!!(window.__apGlobePerf && window.__apGlobePerf.frames > 0 && document.querySelector('[data-globe] canvas'))`)) break;
  }
  // Let the page settle (fonts, late layout) so a sub-pixel shift is not read as globe motion.
  await sleep(1800);
}

const GEO = `(() => {
  const c = document.querySelector('[data-globe] canvas');
  const r = c.getBoundingClientRect();
  const p = window.__apGlobePerf || {};
  return { left: r.left, top: r.top, w: r.width, h: r.height, R: p.R, pins: p.pins || [], frames: p.frames || 0, avg: p.avg, max: p.max };
})()`;
const geo = () => ev(GEO);
const frames = async () => (await geo()).frames;

/** A capture of the page, decoded in the page itself (no PNG decoder needed here). */
async function capture(name) {
  const { data } = await send('Page.captureScreenshot', { format: 'png' });
  if (name) fs.writeFileSync(path.join(OUT, `${name}.png`), Buffer.from(data, 'base64'));
  return data;
}
/** Run `body(px, W, H)` in the page over a decoded capture; px(x, y) returns [r, g, b] at CSS px. */
async function pixels(data, body) {
  return ev(`(async () => {
    const img = new Image();
    img.src = 'data:image/png;base64,${data}';
    await img.decode();
    const k = img.width / window.innerWidth;
    const c = document.createElement('canvas');
    c.width = img.width; c.height = img.height;
    const x2 = c.getContext('2d');
    x2.drawImage(img, 0, 0);
    const d = x2.getImageData(0, 0, c.width, c.height).data;
    const px = (x, y) => { const i = (Math.round(y * k) * c.width + Math.round(x * k)) * 4; return [d[i], d[i + 1], d[i + 2]]; };
    const hsv = ([r, g, b]) => { const mx = Math.max(r, g, b), mn = Math.min(r, g, b), dd = mx - mn; let h = 0;
      if (dd) h = mx === r ? ((g - b) / dd) % 6 : mx === g ? (b - r) / dd + 2 : (r - g) / dd + 4; h = (h * 60 + 360) % 360;
      return { h, s: mx ? dd / mx : 0, v: mx / 255 }; };
    const luma = ([r, g, b]) => (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
    return (${body})(px, hsv, luma);
  })()`);
}

// ── AC11.2 Console: one maximum, at the rim ──
await open('console', { reduced: true });
{
  const g = await geo();
  const data = await capture('console-1440');
  const prof = await pixels(data, `(px, hsv, luma) => {
    const g = ${JSON.stringify(g)};
    const cx = g.left + g.w / 2, cy = g.top + g.h / 2, out = [];
    for (let f = 0.86; f <= 1.45; f += 0.005) {
      const r = g.R * f, vals = [];
      for (let a = 0; a < 360; a += 4) { const x = cx + Math.cos(a * Math.PI / 180) * r, y = cy + Math.sin(a * Math.PI / 180) * r;
        if (x >= 0 && y >= 0 && x < innerWidth && y < innerHeight) vals.push(luma(px(x, y))); }
      vals.sort((p, q) => p - q);
      out.push([+f.toFixed(3), vals[Math.floor(vals.length / 2)]]);
    }
    return out;
  }`);
  const sm = prof.map((p, i) => [p[0], (prof[Math.max(0, i - 2)][1] + prof[Math.max(0, i - 1)][1] + p[1] + prof[Math.min(prof.length - 1, i + 1)][1] + prof[Math.min(prof.length - 1, i + 2)][1]) / 5]);
  const peaks = [];
  for (let i = 1; i < sm.length - 1; i++) {
    if (sm[i][1] > sm[i - 1][1] && sm[i][1] >= sm[i + 1][1]) {
      const lo = Math.min(...sm.slice(Math.max(0, i - 12), i).map((p) => p[1]), ...sm.slice(i + 1, i + 13).map((p) => p[1]));
      if (sm[i][1] - lo > 0.01) peaks.push(sm[i][0]);
    }
  }
  const tail = sm[sm.length - 1][1];
  record('AC11.2', peaks.length === 1 && Math.abs(peaks[0] - 1) <= 0.06, `maxima at ${peaks.join(', ')}R; luma at 1.0R ${sm.find((p) => p[0] >= 1)[1].toFixed(3)}, 1.3R ${sm.find((p) => p[0] >= 1.3)[1].toFixed(3)}, 1.45R ${tail.toFixed(3)}`);
}

// ── AC11.3 Frost and Grey Frost: blue, no red ──
for (const theme of ['frost', 'grey-frost']) {
  await open(theme, { reduced: true });
  const g = await geo();
  const data = await capture(`${theme}-1440`);
  const r = await pixels(data, `(px, hsv) => {
    const g = ${JSON.stringify(g)};
    let red = 0, n = 0; const rim = [];
    for (let y = Math.max(0, g.top); y < Math.min(innerHeight, g.top + g.h); y += 2)
      for (let x = Math.max(0, g.left); x < Math.min(innerWidth, g.left + g.w); x += 2) {
        const c = hsv(px(x, y)); n++;
        if ((c.h < 20 || c.h > 340) && c.s > 0.5 && c.v > 0.15) red++;
      }
    const cx = g.left + g.w / 2, cy = g.top + g.h / 2;
    for (let a = 0; a < 360; a += 10) rim.push(hsv(px(cx + Math.cos(a * Math.PI / 180) * g.R * 0.995, cy + Math.sin(a * Math.PI / 180) * g.R * 0.995)).h);
    rim.sort((p, q) => p - q);
    return { red, n, rimHue: rim[Math.floor(rim.length / 2)] };
  }`);
  record('AC11.3', r.red === 0 && r.rimHue > 195 && r.rimHue < 235, `${theme}: ${r.red} red px of ${r.n} sampled; rim hue ${r.rimHue.toFixed(0)}°`);
}

// ── AC11.4 White: light body, no pink fog ──
await open('light', { reduced: true });
{
  const g = await geo();
  const data = await capture('light-1440');
  const r = await pixels(data, `(px, hsv, luma) => {
    const g = ${JSON.stringify(g)};
    const cx = g.left + g.w / 2, cy = g.top + g.h / 2; let sum = 0, n = 0; const sat = [];
    for (let y = cy - g.R * 0.7; y < cy + g.R * 0.7; y += 3) for (let x = cx - g.R * 0.7; x < cx + g.R * 0.7; x += 3) {
      if (Math.hypot(x - cx, y - cy) > g.R * 0.7 || y < 0 || y >= innerHeight) continue; sum += luma(px(x, y)); n++; }
    for (let a = 0; a < 360; a += 3) { const x = cx + Math.cos(a * Math.PI / 180) * g.R * 1.15, y = cy + Math.sin(a * Math.PI / 180) * g.R * 1.15;
      if (y >= 0 && y < innerHeight) sat.push(hsv(px(x, y)).s); }
    sat.sort((p, q) => p - q);
    return { body: sum / n, sat95: sat[Math.floor(sat.length * 0.95)], satMax: sat[sat.length - 1] };
  }`);
  record('AC11.4', r.body > 0.85 && r.sat95 < 0.35, `body luma ${r.body.toFixed(3)}; saturation at 1.15R p95 ${r.sat95.toFixed(3)} (max ${r.satMax.toFixed(3)})`);
}
for (const theme of ['red']) {
  await open(theme, { reduced: true });
  await capture(`${theme}-1440`);
}

// ── AC11.5 live theme switch, same canvas ──
await open('console');
{
  await ev(`window.__globeCanvas = document.querySelector('[data-globe] canvas'); true`);
  await ev(`${setThemeExpr('frost')}; new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(() => r(true))))`); // data-theme and its data-ground
  const g = await geo();
  const data = await capture();
  const hue = await pixels(data, `(px, hsv) => { const g = ${JSON.stringify(g)}; const cx = g.left + g.w / 2, cy = g.top + g.h / 2; const hs = [];
    for (let a = 0; a < 360; a += 10) hs.push(hsv(px(cx + Math.cos(a * Math.PI / 180) * g.R * 0.995, cy + Math.sin(a * Math.PI / 180) * g.R * 0.995)).h);
    hs.sort((p, q) => p - q); return hs[Math.floor(hs.length / 2)]; }`);
  const same = await ev(`window.__globeCanvas === document.querySelector('[data-globe] canvas') && window.__globeCanvas.isConnected`);
  record('AC11.5', same && hue > 195 && hue < 235, `same canvas: ${same}; rim hue two frames after the switch ${hue.toFixed(0)}°`);
}

// ── AC11.6 / AC11.7 click a pin, City card, row selects ──
const click = async (x, y) => {
  await send('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y });
  await send('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
  await send('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
};
await open('frost');
{
  // A pin off-centre whose head is not under a chip (a chip click is its own path to the same card).
  await ev(`document.querySelectorAll('[data-globe] .ap-globe-chip').forEach((c) => (c.style.visibility = 'hidden')); true`);
  const g = await geo();
  const pin = [...g.pins].sort((a, b) => Math.hypot(b.x - g.w / 2, b.y - g.h / 2) - Math.hypot(a.x - g.w / 2, a.y - g.h / 2)).find((p) => Math.hypot(p.x - g.w / 2, p.y - g.h / 2) < g.R * 0.8) ?? g.pins[0];
  const now = (await geo()).pins.find((p) => p.key === pin.key) ?? pin;
  await click(g.left + now.x, g.top + now.hy);
  await ev(`document.querySelectorAll('[data-globe] .ap-globe-chip').forEach((c) => (c.style.visibility = '')); true`);
  await sleep(900);
  const after = await geo();
  const opened = await ev(`document.querySelector('[role=region][aria-label^="City:"]')?.getAttribute('aria-label') ?? ''`);
  const p2 = after.pins.find((p) => `City: ${p.key.replace('|', ', ')}` === opened);
  const off = p2 ? Math.hypot(p2.x - after.w / 2, p2.y - after.h / 2) : Infinity;
  const card = await ev(`(() => { const c = document.querySelector('[role=region][aria-label^="City:"]'); if (!c) return null;
    const dd = [...c.querySelectorAll('dd')].map((d) => d.textContent);
    const rows = [...c.querySelectorAll('button[aria-pressed]')].map((b) => b.textContent);
    return { label: c.getAttribute('aria-label'), dd, rows }; })()`);
  const num = (s) => Number(String(s).replace(/[^\d.]/g, ''));
  const rowLeads = card ? card.rows.reduce((s, r) => s + num(r.match(/([\d,]+) leads/)?.[1] ?? 0), 0) : -1;
  await capture('frost-city-card');
  record('AC11.6', off <= 12 && !!card && rowLeads === num(card.dd[0]), `clicked ${pin.key}, opened ${opened}: tip ${off.toFixed(1)}px from centre; card ${card?.label}; facts ${card?.dd.join(' / ')}; rows sum ${rowLeads} leads`);
  const rowAt = await ev(`(() => { const b = document.querySelector('[role=region][aria-label^="City:"] button[aria-pressed]'); if (!b) return null; const r = b.getBoundingClientRect(); return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; })()`);
  if (rowAt) await click(rowAt.x, rowAt.y);
  await sleep(400);
  const search = await ev('location.search');
  const still = await ev(`!!document.querySelector('[role=region][aria-label^="City:"]')`);
  record('AC11.7', /campaign=/.test(search), `URL ${search}; card still open on the campaign that has the city (Q-AP-8): ${still}`);
}

// ── AC11.8 chips never overlap chips or floaters ──
for (const w of [1440, 1024]) {
  await open('frost', { w });
  let worst = '';
  let samples = 0;
  for (let i = 0; i < 25; i++) {
    const bad = await ev(`(() => {
      const chips = [...document.querySelectorAll('[data-globe] .ap-globe-chip[data-show="1"]')].map((e) => [e.textContent, e.getBoundingClientRect()]);
      const fl = [...document.querySelectorAll('[data-ap-floater]')].map((e) => e.getBoundingClientRect());
      const hit = (a, b) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom;
      for (let i = 0; i < chips.length; i++) {
        for (const f of fl) if (hit(chips[i][1], f)) return chips[i][0] + ' x floater';
        for (let j = i + 1; j < chips.length; j++) if (hit(chips[i][1], chips[j][1])) return chips[i][0] + ' x ' + chips[j][0];
      }
      return '';
    })()`);
    samples++;
    if (bad) worst = bad;
    await sleep(120);
  }
  await capture(`frost-${w}`);
  record('AC11.8', !worst, `${w}: ${samples} samples over 3s, ${worst || 'no overlap'}`);
}

// ── AC11.9 reduced motion: static, no frames ──
await open('console', { reduced: true });
{
  // Compare the globe slot only: other page furniture (dev indicators, hover states) is not the globe's.
  const g = await geo();
  const clip = { x: g.left, y: Math.max(0, g.top), width: g.w, height: g.h, scale: 1 };
  const shot = async () => (await send('Page.captureScreenshot', { format: 'png', clip })).data;
  const f0 = await frames();
  const a = await shot();
  await sleep(1000);
  const b = await shot();
  const f1 = await frames();
  if (a !== b) {
    fs.writeFileSync(path.join(OUT, 'reduced-a.png'), Buffer.from(a, 'base64'));
    fs.writeFileSync(path.join(OUT, 'reduced-b.png'), Buffer.from(b, 'base64'));
  }
  const where =
    a === b
      ? ''
      : await ev(`(async () => {
    const load = async (s) => { const i = new Image(); i.src = 'data:image/png;base64,' + s; await i.decode(); const c = document.createElement('canvas'); c.width = i.width; c.height = i.height; const x = c.getContext('2d'); x.drawImage(i, 0, 0); return x.getImageData(0, 0, i.width, i.height); };
    const A = await load(${JSON.stringify(a)}), B = await load(${JSON.stringify(b)});
    let n = 0, x0 = 1e9, y0 = 1e9, x1 = 0, y1 = 0;
    for (let i = 0; i < A.data.length; i += 4) {
      if (A.data[i] === B.data[i] && A.data[i + 1] === B.data[i + 1] && A.data[i + 2] === B.data[i + 2]) continue;
      const k = i / 4, x = k % A.width, y = (k - x) / A.width;
      n++; x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y);
    }
    return '; ' + n + ' px differ in [' + [x0, y0, x1, y1].join(',') + ']';
  })()`);
  record('AC11.9', f0 === f1 && a === b, `frames while idle ${f1 - f0}; captures 1s apart identical: ${a === b}${where}`);
}

// ── AC11.10 off-screen: no frames ──
await open('console');
{
  await ev(`window.scrollTo(0, document.documentElement.scrollHeight); true`);
  await sleep(400);
  const f0 = await frames();
  await sleep(1000);
  const f1 = await frames();
  await ev(`window.scrollTo(0, 0); true`);
  await sleep(400);
  const f2 = await frames();
  record('AC11.10', f1 === f0 && f2 > f1, `frames off-screen ${f1 - f0}; resumed on return: ${f2 > f1}`);
}

// ── AC11.11 frame budget ──
await open('console', { dpr: 2 });
{
  await sleep(3000);
  const g = await geo();
  record('AC11.11', g.avg <= 4, `1440x900 DPR 2: ${g.avg.toFixed(2)}ms avg, ${g.max.toFixed(2)}ms max over ${g.frames} frames`);
}

// ── AC11.12 keyboard ──
await open('frost');
{
  const key = async (k, code = k) => {
    await send('Input.dispatchKeyEvent', { type: 'rawKeyDown', key: k, code, windowsVirtualKeyCode: { ArrowLeft: 37, Enter: 13, Escape: 27 }[k] });
    await send('Input.dispatchKeyEvent', { type: 'keyUp', key: k, code, windowsVirtualKeyCode: { ArrowLeft: 37, Enter: 13, Escape: 27 }[k] });
  };
  await ev(`document.querySelector('[data-globe]').focus({ focusVisible: true }); document.activeElement === document.querySelector('[data-globe]')`);
  const before = await geo();
  await key('ArrowLeft');
  await sleep(500);
  const after = await geo();
  const k0 = before.pins[0]?.key;
  const a0 = before.pins.find((p) => p.key === k0);
  const a1 = after.pins.find((p) => p.key === k0);
  const moved = a0 && a1 ? Math.hypot(a1.x - a0.x, a1.y - a0.y) : 0;
  await key('Enter');
  await sleep(700);
  const opened = await ev(`document.querySelector('[role=region][aria-label^="City:"]')?.getAttribute('aria-label') ?? null`);
  await key('Escape');
  await sleep(300);
  const closed = await ev(`!document.querySelector('[role=region][aria-label^="City:"]')`);
  record('AC11.12', moved > 5 && !!opened && closed, `Left moved ${k0} ${moved.toFixed(0)}px; Enter opened ${opened}; Escape closed: ${closed}`);
}

// ── AC11.13 touch: vertical swipe scrolls ──
await open('frost');
{
  await send('Emulation.setTouchEmulationEnabled', { enabled: true, maxTouchPoints: 1 });
  const g = await geo();
  const y0 = await ev('window.scrollY + (document.querySelector("main")?.scrollTop ?? 0)');
  await send('Input.synthesizeScrollGesture', { x: Math.round(g.left + g.w / 2), y: Math.round(Math.min(800, g.top + g.h / 2)), yDistance: -300, gestureSourceType: 'touch', speed: 800 });
  await sleep(600);
  const y1 = await ev('window.scrollY + (document.querySelector("main")?.scrollTop ?? 0)');
  await send('Emulation.setTouchEmulationEnabled', { enabled: false });
  const ta = await ev(`getComputedStyle(document.querySelector('[data-globe] canvas')).touchAction`);
  record('AC11.13', y1 > y0, `scroll ${y0} -> ${y1}; canvas touch-action ${ta}`);
}

close();
fs.writeFileSync(path.join(OUT, 'report.json'), JSON.stringify(results, null, 2));
const fails = results.filter((r) => !r.pass).length;
console.log(`\n${results.length - fails}/${results.length} passed; captures in ${OUT}`);
if (fails) process.exitCode = 1;
