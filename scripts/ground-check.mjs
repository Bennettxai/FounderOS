// Ground check (SPEC S06 §10, AC06.1 to AC06.5, AC06.7): no page ground stops short of the scroll end.
// For every route × theme × width it loads the page, scrolls to the end and samples three pixels 4px
// above the viewport bottom (sidebar + 18, centre, right - 8; right = layout viewport, scrollbar excluded). Each must equal the page ground: the
// computed background of <html>, or the page-scoped ground layer (.ae-ground, .ap-ground, and the Content Lab v2 .lab2-aurora, AC-02-41). A pixel
// covered by an opaque content element, a glass face (transparent by design: its frost and light cover the ground; scripts/ground-lib.mjs)
// or a drop shadow is reported as covered, not as a cut-off. Under Grey Frost it
// also looks for near-black regions larger than 200x200 (holes left by literal blacks), ignoring media
// and dialogs. Screenshots land in <outdir>/<theme>/<route>-<width>-bottom.png. PNGs are decoded with
// zlib, no extra dependency. Same Chrome as theme-audit.mjs (CDP on 9333 or CDP_PORT).
// Usage: BASE=http://127.0.0.1:4302 node scripts/ground-check.mjs [themes,comma|all] [routes,comma|all] [outdir] [widths=1440,2200]
import fs from 'node:fs';
import zlib from 'node:zlib';
import { THEME_IDS, cdp, routesFromEnv } from './audit-routes.mjs';
import { GLASS_FACE_SELECTOR, coversPoint } from './ground-lib.mjs';
import { setThemeExpr } from './theme-ground.mjs';

const [,, themesArg = 'all', routesArg = 'all', outArg, widthsArg = '1440,2200'] = process.argv;
const BASE = process.env.BASE ?? 'http://127.0.0.1:4302';
const OUT = outArg ?? `${process.env.S ?? '.'}/audit/ground`;
const themes = themesArg === 'all' ? THEME_IDS : themesArg.split(',');
const routes = routesFromEnv(routesArg);
const widths = widthsArg.split(',').map(Number);
const HEIGHT = 900;
const TOLERANCE = 3;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

/** Minimal PNG decoder: 8-bit greyscale/RGB/RGBA, non-interlaced (what Chrome writes). */
export function decodePng(buf) {
  let pos = 8;
  let width = 0, height = 0, colorType = 0;
  const idat = [];
  while (pos < buf.length) {
    const len = buf.readUInt32BE(pos);
    const type = buf.toString('ascii', pos + 4, pos + 8);
    const data = buf.subarray(pos + 8, pos + 8 + len);
    if (type === 'IHDR') { width = data.readUInt32BE(0); height = data.readUInt32BE(4); colorType = data[9]; if (data[8] !== 8 || data[12] !== 0) throw new Error('unsupported PNG'); }
    else if (type === 'IDAT') idat.push(data);
    else if (type === 'IEND') break;
    pos += 12 + len;
  }
  const bpp = { 0: 1, 2: 3, 4: 2, 6: 4 }[colorType];
  const raw = zlib.inflateSync(Buffer.concat(idat));
  const stride = width * bpp;
  const px = Buffer.alloc(height * stride);
  for (let y = 0; y < height; y++) {
    const f = raw[y * (stride + 1)];
    const line = raw.subarray(y * (stride + 1) + 1, (y + 1) * (stride + 1));
    for (let x = 0; x < stride; x++) {
      const a = x >= bpp ? px[y * stride + x - bpp] : 0;
      const b = y > 0 ? px[(y - 1) * stride + x] : 0;
      const c = x >= bpp && y > 0 ? px[(y - 1) * stride + x - bpp] : 0;
      let v = line[x];
      if (f === 1) v += a;
      else if (f === 2) v += b;
      else if (f === 3) v += (a + b) >> 1;
      else if (f === 4) { const p = a + b - c; const pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c); v += pa <= pb && pa <= pc ? a : pb <= pc ? b : c; }
      px[y * stride + x] = v & 255;
    }
  }
  const at = (x, y) => { const i = y * stride + x * bpp; return bpp >= 3 ? [px[i], px[i + 1], px[i + 2]] : [px[i], px[i], px[i]]; };
  return { width, height, at };
}

const lum = ([r, g, b]) => { const f = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; }; return 0.2126 * f(r) + 0.7152 * f(g) + 0.0722 * f(b); };
/** rgb()/rgba() in 0..255, or `color(srgb r g b)` (what a color-mix() ground such as the Lab Aurora computes to) in 0..1. */
const rgbOf = (s) => {
  const v = (s.replace(/^color\(srgb/, '').match(/\d+(\.\d+)?/g) ?? []).slice(0, 3).map(Number);
  return s.startsWith('color(srgb') ? v.map((c) => Math.round(c * 255)) : v;
};

/** What sits on a point: 'ground' when only page-level layers are there, else the covering element. */
const PROBE = (x, y) => `(()=>{
  const covers = ${coversPoint.toString()};
  const stack = document.elementsFromPoint(${x}, ${y});
  const ground = (e) => e === document.documentElement || e === document.body || e.matches('.page-ground, .ae-ground, .ap-ground, .lab2-aurora, [data-lab-root], main, .os-shell, [data-creative-engine], [data-adpilot]') || (e.parentElement === document.querySelector('main') && getComputedStyle(e).backgroundColor === 'rgba(0, 0, 0, 0)');
  for (const e of stack) {
    if (ground(e)) continue;
    const cs = getComputedStyle(e);
    const opaque = covers({ backgroundColor: cs.backgroundColor, backgroundImage: cs.backgroundImage, backdropFilter: cs.backdropFilter || cs.webkitBackdropFilter, tagName: e.tagName, isFace: e.matches(${JSON.stringify(GLASS_FACE_SELECTOR)}) });
    if (opaque) return (e.tagName.toLowerCase() + '.' + String(e.className && e.className.baseVal !== undefined ? e.className.baseVal : e.className).split(' ').slice(0, 2).join('.')).slice(0, 60);
  }
  // a drop shadow cast by a card that ends just above the point is content, not a cut-off
  for (const e of document.querySelectorAll('main *')) {
    const r = e.getBoundingClientRect();
    if (r.bottom <= ${y} && r.bottom > ${y} - 80 && r.left <= ${x} && r.right >= ${x} && getComputedStyle(e).boxShadow !== 'none') return 'shadow of ' + e.tagName.toLowerCase();
  }
  return 'ground';
})()`;

const report = [];
let fails = 0;
const { send, ev, close } = await cdp();
for (const theme of themes) {
  await send('Page.addScriptToEvaluateOnNewDocument', { source: `try{localStorage.setItem('glados-theme',${JSON.stringify(theme)})}catch(e){}` });
  fs.mkdirSync(`${OUT}/${theme}`, { recursive: true });
  for (const width of widths) {
    await send('Emulation.setDeviceMetricsOverride', { width, height: HEIGHT, deviceScaleFactor: 1, mobile: false });
    for (const route of routes) {
      const name = route === '/' ? 'home' : route.replace(/^\//, '').replace(/[/?=&]/g, '_');
      const row = { theme, width, route, samples: [], blackRegions: [] };
      try {
        await send('Page.navigate', { url: BASE + route }); await sleep(Number(process.env.WAIT ?? 4000));
        await ev(setThemeExpr(theme)); // data-theme and the data-ground the glass light reads
        await ev(`(async()=>{for(let i=0;i<6;i++){window.scrollTo(0, document.documentElement.scrollHeight);document.querySelectorAll('main, main *').forEach(e=>{if(e.scrollHeight>e.clientHeight+4&&/(auto|scroll)/.test(getComputedStyle(e).overflowY))e.scrollTop=e.scrollHeight});await new Promise(r=>setTimeout(r,150))}})()`);
        await sleep(500);
        const expect = await ev(`(()=>{const g=document.querySelector('.ae-ground, .ap-ground, .lab2-aurora');return getComputedStyle(g||document.documentElement).backgroundColor})()`);
        const sidebar = await ev(`(()=>{const s=document.querySelector('.os-sidebar');return s?Math.round(s.getBoundingClientRect().right):0})()`);
        const shot = Buffer.from((await send('Page.captureScreenshot', { format: 'png' })).data, 'base64');
        fs.writeFileSync(`${OUT}/${theme}/${name}-${width}-bottom.png`, shot);
        const img = decodePng(shot);
        const want = rgbOf(expect);
        const labAurora = await ev(`!!document.querySelector('.lab2-aurora')`);
        // right edge of the layout viewport: a classic scrollbar is browser chrome, not page ground
        const inner = await ev(`document.documentElement.clientWidth`);
        for (const x of [sidebar + 18, Math.round((sidebar + inner) / 2), inner - 8]) {
          const y = HEIGHT - 4;
          // The Classic Console Aurora carries a 3px scanline texture (02 2.1): a sample that lands on a
          // scanline row is read against the two rows above it, so the texture is not taken for a cut-off.
          const rows = labAurora && theme === 'console' ? [y, y - 1, y - 2] : [y];
          const reads = rows.map((yy) => img.at(Math.min(x, img.width - 1), Math.min(yy, img.height - 1)));
          const deltaOf = (g) => Math.max(...g.map((c, i) => Math.abs(c - want[i])));
          const got = reads.reduce((a, b) => (deltaOf(b) < deltaOf(a) ? b : a));
          const delta = deltaOf(got);
          const on = delta > TOLERANCE ? await ev(PROBE(x, y)) : 'ground';
          const status = delta <= TOLERANCE ? 'ok' : on === 'ground' ? 'CUT' : 'covered';
          if (status === 'CUT') fails++;
          row.samples.push({ x, y, got: got.join(','), want: want.join(','), status, on });
        }
        if (theme === 'grey-frost') {
          // 20px cells; a black cell is darker than any Grey Frost surface (L < 0.004, about #0e0e0e)
          const C = 20, cols = Math.floor(img.width / C), rows = Math.floor(img.height / C);
          const black = Array.from({ length: rows }, (_, r) => Array.from({ length: cols }, (_, c) => lum(img.at(c * C + 10, r * C + 10)) < 0.004));
          const seen = new Set();
          for (let r = 0; r <= rows - 10; r++) for (let c = 0; c <= cols - 10; c++) {
            let all = true;
            for (let i = 0; i < 10 && all; i++) for (let j = 0; j < 10 && all; j++) all = black[r + i][c + j];
            if (!all || seen.has(`${Math.floor(r / 10)},${Math.floor(c / 10)}`)) continue;
            seen.add(`${Math.floor(r / 10)},${Math.floor(c / 10)}`);
            const x = c * C + 100, y = r * C + 100;
            const what = await ev(`(()=>{const e=document.elementFromPoint(${x},${y});if(!e)return 'none';if(e.closest('[role=dialog], dialog'))return 'dialog';if(e.closest('img, video, canvas'))return 'media';return e.tagName.toLowerCase()+'.'+String(e.className&&e.className.baseVal!==undefined?e.className.baseVal:e.className).split(' ').slice(0,2).join('.')})()`);
            const okRegion = what === 'dialog' || what === 'media' || what.startsWith('canvas');
            row.blackRegions.push({ x, y, on: what, status: okRegion ? 'media' : 'HOLE' });
            if (!okRegion) fails++;
          }
        }
      } catch (e) { row.error = e.message.slice(0, 120); fails++; }
      report.push(row);
      const s = row.samples.map((p) => p.status === 'ok' ? 'ok' : `${p.status}(${p.got} vs ${p.want} on ${p.on})`).join(' ');
      const holes = row.blackRegions.filter((b) => b.status === 'HOLE').map((b) => `${b.x},${b.y} ${b.on}`);
      console.log(`${theme} ${width} ${route}: ${row.error ?? s}${holes.length ? ` | black holes: ${holes.join('; ')}` : ''}`);
    }
  }
}
fs.mkdirSync(OUT, { recursive: true });
fs.writeFileSync(`${OUT}/ground-report.json`, JSON.stringify(report, null, 1));
close();
console.log(fails ? `FAIL ${fails}` : 'PASS');
if (fails) process.exitCode = 1;
