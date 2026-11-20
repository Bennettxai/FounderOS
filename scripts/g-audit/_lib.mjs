// Pure helpers for scripts/g-audit.mjs and tests/g-controls.test.ts (SPEC S00 step 8, GP4).
// No browser and no dependencies: PNG decode/encode on node:zlib, pixel diff, WCAG contrast,
// box-shadow parsing, the G_READY reader and the Classic-control source scanner.
// Files in scripts/g-audit/ whose name starts with "_" are helpers, not route plugins.
import fs from 'node:fs';
import path from 'node:path';
import zlib from 'node:zlib';
import { fileURLToPath } from 'node:url';

export const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..');

/**
 * @typedef {{ classes: string[], onControlOnly: string[], utilityOnControl: string[], kitControlClasses: string[] }} ClassicControls
 * @typedef {{ routes: string[], files: Record<string, string[]> }} GReady
 * @typedef {{ line: number, kind: 'classic-class' | 'raw-control' | 'classic-on-control', token: string, text: string }} SourceHit
 */

/** scripts/g-audit/classic-controls.json (FR-GP4.2). @returns {ClassicControls} */
export function loadClassicControls(root = ROOT) {
  return JSON.parse(fs.readFileSync(path.join(root, 'scripts/g-audit/classic-controls.json'), 'utf8'));
}

// ── G_READY ──────────────────────────────────────────────────────────────────────────────────
// The appearance model (SPEC S00 FR-14, Phase 1 lane 1a) owns the registry. Contract this reader
// expects, in lib/appearance.ts (or lib/g-ready.ts until the model lands):
//   export const G_READY: string[] = ['/creative-engine', ...];            // routes on the GladOS tree
//   export const G_TREE_FILES: Record<string, string[]> = {                 // route → its GladOS-tree files
//     '/creative-engine': ['components/creative-engine/DashboardG.tsx', ...],
//   };
// Both are read as source text (string literals only), so the .mjs audit can use them without a
// TypeScript loader. No file, or no G_READY in it, means an empty registry.
export const G_READY_SOURCES = ['lib/appearance.ts', 'lib/g-ready.ts'];

const strings = (s) => [...s.matchAll(/'([^'\n]*)'|"([^"\n]*)"/g)].map((m) => m[1] ?? m[2]);

/** Parse G_READY and G_TREE_FILES out of one source text. @param {string} src @returns {GReady} */
export function parseGReady(src) {
  const ready = src.match(/export\s+const\s+G_READY\b[^=]*=\s*\[([\s\S]*?)\]/);
  const routes = ready ? strings(ready[1]) : [];
  /** @type {Record<string, string[]>} */
  const files = {};
  const reg = src.match(/export\s+const\s+G_TREE_FILES\b[^=]*=\s*\{([\s\S]*?)\n\}/);
  if (reg) {
    for (const m of reg[1].matchAll(/(?:'([^']+)'|"([^"]+)")\s*:\s*\[([\s\S]*?)\]/g)) files[m[1] ?? m[2]] = strings(m[3]);
  }
  return { routes, files };
}

/** { routes, files, source } from the first registry file that exists, else empty. @returns {GReady & { source: string | null }} */
export function readGReady(root = ROOT) {
  for (const rel of G_READY_SOURCES) {
    const p = path.join(root, rel);
    if (!fs.existsSync(p)) continue;
    const src = fs.readFileSync(p, 'utf8');
    // A re-export (`export { G_READY } from './g-ready'`) is not the registry: fall through to the file that declares it.
    if (!/export\s+const\s+G_READY\b/.test(src)) continue;
    return { ...parseGReady(src), source: rel };
  }
  return { routes: [], files: {}, source: null };
}

/** A route (may carry a query) is converted when its pathname is in G_READY. */
/** The registry entry a route falls under: exact first, then `:param` patterns (lib/g-ready.ts matchesReadyRoute). */
export function readyEntryFor(route, routes) {
  if (routes.includes(route)) return route;
  const p = route.split('?')[0];
  if (routes.includes(p)) return p;
  const b = p.split('/');
  return routes.find((e) => {
    if (!e.includes('/:')) return false;
    const a = e.split('/');
    return a.length === b.length && a.every((seg, i) => (seg.startsWith(':') ? b[i].length > 0 : seg === b[i]));
  }) ?? null;
}

export function routeInReady(route, routes) {
  return readyEntryFor(route, routes) !== null;
}

export const routeSlug = (route) => (route === '/' ? 'home' : route.replace(/^\//, '').replace(/[/?=&]/g, '_'));

// ── PNG ──────────────────────────────────────────────────────────────────────────────────────
const SIG = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);

/** Decode an 8-bit, non-interlaced PNG (what Chrome's captureScreenshot writes) to RGBA. */
export function decodePng(buf) {
  if (!buf.subarray(0, 8).equals(SIG)) throw new Error('not a PNG');
  let off = 8, width = 0, height = 0, depth = 0, type = 0, interlace = 0;
  const idat = [];
  while (off < buf.length) {
    const len = buf.readUInt32BE(off);
    const kind = buf.toString('latin1', off + 4, off + 8);
    const body = buf.subarray(off + 8, off + 8 + len);
    if (kind === 'IHDR') { width = body.readUInt32BE(0); height = body.readUInt32BE(4); depth = body[8]; type = body[9]; interlace = body[12]; }
    else if (kind === 'IDAT') idat.push(body);
    else if (kind === 'IEND') break;
    off += 12 + len;
  }
  const channels = { 0: 1, 2: 3, 4: 2, 6: 4 }[type];
  if (depth !== 8 || !channels || interlace) throw new Error(`unsupported PNG (depth ${depth}, type ${type}, interlace ${interlace})`);
  const raw = zlib.inflateSync(Buffer.concat(idat));
  const stride = width * channels;
  const px = Buffer.alloc(stride * height);
  for (let y = 0; y < height; y++) {
    const f = raw[y * (stride + 1)];
    const src = y * (stride + 1) + 1;
    const dst = y * stride;
    for (let x = 0; x < stride; x++) {
      const a = x >= channels ? px[dst + x - channels] : 0;
      const b = y ? px[dst - stride + x] : 0;
      const c = x >= channels && y ? px[dst - stride + x - channels] : 0;
      const v = raw[src + x];
      let out;
      if (f === 0) out = v;
      else if (f === 1) out = v + a;
      else if (f === 2) out = v + b;
      else if (f === 3) out = v + ((a + b) >> 1);
      else { const p = a + b - c, pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c); out = v + (pa <= pb && pa <= pc ? a : pb <= pc ? b : c); }
      px[dst + x] = out & 255;
    }
  }
  const data = Buffer.alloc(width * height * 4);
  for (let i = 0, j = 0; i < width * height; i++, j += channels) {
    const g = channels <= 2;
    data[i * 4] = px[j]; data[i * 4 + 1] = g ? px[j] : px[j + 1]; data[i * 4 + 2] = g ? px[j] : px[j + 2];
    data[i * 4 + 3] = channels === 4 ? px[j + 3] : channels === 2 ? px[j + 1] : 255;
  }
  return { width, height, data };
}

function chunk(kind, body) {
  const len = Buffer.alloc(4); len.writeUInt32BE(body.length);
  const kb = Buffer.from(kind, 'latin1');
  const crc = Buffer.alloc(4); crc.writeUInt32BE(zlib.crc32(Buffer.concat([kb, body])) >>> 0);
  return Buffer.concat([len, kb, body, crc]);
}

/** Encode RGBA to PNG (filter 0). Used for diff images only. */
export function encodePng({ width, height, data }) {
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(width, 0); ihdr.writeUInt32BE(height, 4); ihdr[8] = 8; ihdr[9] = 6;
  const raw = Buffer.alloc((width * 4 + 1) * height);
  for (let y = 0; y < height; y++) data.copy(raw, y * (width * 4 + 1) + 1, y * width * 4, (y + 1) * width * 4);
  return Buffer.concat([SIG, chunk('IHDR', ihdr), chunk('IDAT', zlib.deflateSync(raw)), chunk('IEND', Buffer.alloc(0))]);
}

/**
 * Pixel diff of two RGBA images. A pixel differs when any channel moves by more than
 * `tolerance`. Size changes compare the overlap and count the rest as differing.
 * Returns { ratio (0..1), differing, total, sizeChanged, image } where image marks
 * differing pixels in full red over a dimmed copy of `b`.
 */
export function diffImages(a, b, { tolerance = 2 } = {}) {
  const width = Math.max(a.width, b.width), height = Math.max(a.height, b.height);
  const out = Buffer.alloc(width * height * 4);
  let differing = 0;
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const o = (y * width + x) * 4;
      const inA = x < a.width && y < a.height, inB = x < b.width && y < b.height;
      let diff = !(inA && inB);
      if (!diff) {
        const ia = (y * a.width + x) * 4, ib = (y * b.width + x) * 4;
        for (let k = 0; k < 4; k++) if (Math.abs(a.data[ia + k] - b.data[ib + k]) > tolerance) { diff = true; break; }
      }
      if (diff) { differing++; out[o] = 255; out[o + 1] = 0; out[o + 2] = 0; out[o + 3] = 255; }
      else {
        const ib = (y * b.width + x) * 4;
        const l = (b.data[ib] * 0.3 + b.data[ib + 1] * 0.59 + b.data[ib + 2] * 0.11) * 0.35;
        out[o] = out[o + 1] = out[o + 2] = l; out[o + 3] = 255;
      }
    }
  }
  const total = width * height;
  return { ratio: differing / total, differing, total, sizeChanged: a.width !== b.width || a.height !== b.height, image: { width, height, data: out } };
}

// ── Classic baseline masks ───────────────────────────────────────────────────────────────────
/**
 * Regions of a Classic route that show live third-party state, blanked on BOTH sides before the
 * baseline diff so a change in that state is not read as a regression. Each entry is a browser
 * expression returning [{ x, y, w, h }] in CSS pixels (viewport at deviceScaleFactor 1).
 *
 * /compute reads MIOSA on every visit (configured or not, credits, the computer list), so everything
 * of its content except the page title and subtitle is live: the header's right slot (credits pill)
 * and the content below the header, down to the viewport bottom so a longer or shorter body is
 * covered either way. The shell around it is still compared.
 *
 * /setup prints the data directory the server reads (Filming: <checkout>/demo-data), which differs per
 * checkout and worktree; only that one mono line is masked.
 */
export const CLASSIC_MASKS = {
  '/setup': `[...document.querySelectorAll('main p.font-mono')].filter((e) => e.textContent.trim().startsWith('/')).map((e) => e.getBoundingClientRect()).map((r) => ({ x: r.left, y: r.top, w: r.width, h: r.height }))`,
  '/compute': `(() => {
    const h = document.querySelector('main header'); if (!h) return [];
    const r = h.getBoundingClientRect(), t = (h.firstElementChild || h).getBoundingClientRect(), c = (h.parentElement || h).getBoundingClientRect();
    return [{ x: t.right, y: r.top, w: r.right - t.right, h: r.height }, { x: c.left, y: r.bottom, w: c.width, h: innerHeight - r.bottom }];
  })()`,
};

/** A browser expression for the rects of every element matching a CSS selector (the --mask option). */
export const selectorMaskExpr = (selector) =>
  `[...document.querySelectorAll(${JSON.stringify(selector)})].map((e) => e.getBoundingClientRect()).filter((r) => r.width && r.height).map((r) => ({ x: r.left, y: r.top, w: r.width, h: r.height }))`;

/**
 * Masks for the run: the defaults (unless `noDefaults`) plus every `--mask <route>=<selector>`
 * (repeatable). Returns { route: [expr, ...] }.
 *
 * @returns {Record<string, string[]>}
 */
export function parseMasks(argv, { noDefaults = false } = {}) {
  /** @type {Record<string, string[]>} */
  const out = {};
  if (!noDefaults) for (const [route, expr] of Object.entries(CLASSIC_MASKS)) (out[route] ??= []).push(expr);
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] !== '--mask') continue;
    const v = argv[i + 1] ?? '';
    const eq = v.indexOf('=');
    if (eq <= 0 || !v.startsWith('/')) throw new Error(`--mask needs <route>=<selector>, got "${v}"`);
    (out[v.slice(0, eq)] ??= []).push(selectorMaskExpr(v.slice(eq + 1)));
  }
  return out;
}

/** Paint `rects` (CSS px, clipped, rounded outward) in one flat colour, in place. Returns the pixels painted. */
export function maskImage(img, rects, rgba = [255, 0, 255, 255]) {
  let n = 0;
  for (const r of rects ?? []) {
    const x0 = Math.max(0, Math.floor(r.x)), y0 = Math.max(0, Math.floor(r.y));
    const x1 = Math.min(img.width, Math.ceil(r.x + r.w)), y1 = Math.min(img.height, Math.ceil(r.y + r.h));
    for (let y = y0; y < y1; y++) for (let x = x0; x < x1; x++) {
      const o = (y * img.width + x) * 4;
      img.data[o] = rgba[0]; img.data[o + 1] = rgba[1]; img.data[o + 2] = rgba[2]; img.data[o + 3] = rgba[3];
      n++;
    }
  }
  return n;
}

// ── Colour ───────────────────────────────────────────────────────────────────────────────────
/** 'rgb(1, 2, 3)' / 'rgba(1, 2, 3, 0.5)' / 'rgb(1 2 3 / 50%)' → [r, g, b, a] or null. */
export function parseColor(s) {
  const m = s && s.match(/rgba?\(([^)]+)\)/);
  if (!m) return null;
  const parts = m[1].split(/[\s,/]+/).filter(Boolean).map((x) => (x.endsWith('%') ? parseFloat(x) / 100 : parseFloat(x)));
  if (parts.length < 3 || parts.some(Number.isNaN)) return null;
  return [parts[0], parts[1], parts[2], parts[3] ?? 1];
}

const lin = (c) => { c /= 255; return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4; };
export const luminance = ([r, g, b]) => 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);

/** WCAG contrast ratio of two opaque colours. */
export function contrastRatio(a, b) {
  const la = luminance(a), lb = luminance(b);
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05);
}

/** Source-over composite of [r,g,b,a] on an opaque [r,g,b]. */
export const composite = (fg, bg) => { const a = fg[3] ?? 1; return [0, 1, 2].map((i) => fg[i] * a + bg[i] * (1 - a)); };

/** HSL saturation and chroma; a colour is "hued" (accent-tinted) above both floors. */
export function isHued([r, g, b], { minSat = 0.2, minChroma = 24 } = {}) {
  const mx = Math.max(r, g, b), mn = Math.min(r, g, b), d = mx - mn;
  if (d < minChroma) return false;
  const l = (mx + mn) / 510;
  const s = d / 255 / (1 - Math.abs(2 * l - 1) || 1);
  return s >= minSat;
}

/** Computed box-shadow → [{ color, x, y, blur, spread, inset }]. */
export function parseShadows(v) {
  if (!v || v === 'none') return [];
  const layers = [];
  let depth = 0, cur = '';
  for (const ch of v) {
    if (ch === '(') depth++;
    if (ch === ')') depth--;
    if (ch === ',' && depth === 0) { layers.push(cur.trim()); cur = ''; } else cur += ch;
  }
  if (cur.trim()) layers.push(cur.trim());
  return layers.map((l) => {
    const color = parseColor(l);
    const rest = l.replace(/rgba?\([^)]+\)/, '');
    const nums = [...rest.matchAll(/(-?[\d.]+)px/g)].map((m) => parseFloat(m[1]));
    return { color, x: nums[0] ?? 0, y: nums[1] ?? 0, blur: nums[2] ?? 0, spread: nums[3] ?? 0, inset: /\binset\b/.test(rest) };
  });
}

/**
 * Shadow layers that break S00 §3.3.8: an accent-hued colour, or an outer glow (outset, no
 * offset, some blur or a spread wider than a hairline). Transparent layers and inset strokes are
 * ignored, and so is the glass groove (glass pack): a neutral, unblurred 1px ring that only draws
 * the face's edge.
 */
export function badShadows(v) {
  return parseShadows(v).filter((s) => {
    if (s.inset || !s.color || s.color[3] < 0.04) return false;
    const glow = s.x === 0 && s.y === 0 && (s.blur > 0 || s.spread > 1);
    return glow || isHued(s.color);
  }).map((s) => ({ ...s, why: s.x === 0 && s.y === 0 ? 'outer glow' : 'hued shadow' }));
}

// ── Source guard (FR-GP4.4) ──────────────────────────────────────────────────────────────────
/** Blank out // and /* *\/ comments, keeping line numbers. Strings are left alone. */
export function stripComments(src) {
  let out = '', i = 0, q = null;
  while (i < src.length) {
    const c = src[i], n = src[i + 1];
    if (q) { out += c; if (c === '\\') { out += n ?? ''; i += 2; continue; } if (c === q) q = null; i++; continue; }
    if (c === '"' || c === "'" || c === '`') { q = c; out += c; i++; continue; }
    if (c === '/' && n === '/') { while (i < src.length && src[i] !== '\n') { out += ' '; i++; } continue; }
    if (c === '/' && n === '*') { out += '  '; i += 2; while (i < src.length && !(src[i] === '*' && src[i + 1] === '/')) { out += src[i] === '\n' ? '\n' : ' '; i++; } out += '  '; i += 2; continue; }
    out += c; i++;
  }
  return out;
}

const esc = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');

/**
 * Scan one GladOS-tree source file for Classic controls: any Classic control class name
 * (as a whole class token), a glass-* class on a control tag, a Classic utility on a control
 * tag, or a raw <button>/<input>/<select>/<textarea> written without a kit component.
 * Returns [{ line, kind, token, text }].
 * @param {string} src @param {ClassicControls} [cfg] @returns {SourceHit[]}
 */
export function scanSourceForClassicControls(src, cfg = loadClassicControls()) {
  const clean = stripComments(src);
  const lines = clean.split('\n');
  const raw = src.split('\n');
  const hits = [];
  const cls = new RegExp(`(?<![\\w-])(${cfg.classes.map(esc).join('|')})(?![\\w-])`, 'g');
  lines.forEach((line, i) => {
    for (const m of line.matchAll(cls)) hits.push({ line: i + 1, kind: 'classic-class', token: m[1], text: raw[i].trim() });
    for (const m of line.matchAll(/<(button|input|select|textarea)\b/g)) hits.push({ line: i + 1, kind: 'raw-control', token: `<${m[1]}>`, text: raw[i].trim() });
  });
  // Tag-level checks: glass-* and Classic utilities on control tags (tags may span lines).
  const ctl = /<(button|input|select|textarea|a|Link)\b([^>]*)>/g;
  const onCtl = new RegExp(`(?<![\\w-])(${[...cfg.onControlOnly, ...cfg.utilityOnControl].map(esc).join('|')})(?![\\w-])`, 'g');
  for (const m of clean.matchAll(ctl)) {
    for (const u of m[2].matchAll(onCtl)) {
      const line = clean.slice(0, m.index + m[0].indexOf(u[0], m[1].length + 1)).split('\n').length;
      hits.push({ line, kind: 'classic-on-control', token: `${u[1]} on <${m[1]}>`, text: raw[line - 1].trim() });
    }
  }
  return hits.sort((a, b) => a.line - b.line);
}
