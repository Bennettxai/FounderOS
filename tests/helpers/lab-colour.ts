/**
 * Static resolver for the Content Lab v2 tokens (02 section 15.2): builds the `.lab` cascade for one of
 * the 13 styles from app/globals.css and evaluates `color-mix(in srgb, A p%, B)` (nested, var() inside).
 */
import { CSS_NO_COMMENTS, declarations, topLevelRules, type Rule } from './theme-css';

export type RGBA = [number, number, number, number];
export type LabStyle =
  | { ui: 'glados'; accent: 'ice' | 'lime' | 'ember' | 'mint'; mode: 'dark' | 'light' }
  | { ui: 'classic'; theme: 'console' | 'red' | 'light' | 'frost' | 'grey-frost' };

export const LAB_STYLES: LabStyle[] = [
  ...(['ice', 'lime', 'ember', 'mint'] as const).flatMap((accent) => (['dark', 'light'] as const).map((mode) => ({ ui: 'glados' as const, accent, mode }))),
  ...(['console', 'red', 'light', 'frost', 'grey-frost'] as const).map((theme) => ({ ui: 'classic' as const, theme })),
];

export const styleName = (s: LabStyle) => (s.ui === 'glados' ? `glados ${s.accent} ${s.mode}` : `classic ${s.theme}`);

/** <html> attributes for a style. Under GladOS the derived Classic theme does not feed any .lab v2 token. */
function attrs(s: LabStyle): Record<string, string> {
  if (s.ui === 'glados') return { 'data-ui': 'glados', 'data-accent': s.accent, 'data-mode': s.mode, 'data-theme': s.mode === 'light' ? 'light' : 'console' };
  return { 'data-ui': 'classic', 'data-accent': 'ice', 'data-mode': s.theme === 'light' ? 'light' : 'dark', 'data-theme': s.theme };
}

type Target = 'root' | 'lab';

/** Does a compound root part like `:root[data-ui='glados']:not([data-mode='light'])` match these attributes? */
function rootMatches(part: string, a: Record<string, string>): { ok: boolean; spec: number } | null {
  let rest = part.trim();
  let spec = 0;
  if (rest.startsWith(':root')) { rest = rest.slice(5); spec += 10; }
  else if (rest.startsWith('html')) { rest = rest.slice(4); spec += 1; }
  else if (!rest.startsWith('[')) return null;
  let ok = true;
  while (rest.length) {
    let m = /^\[([\w-]+)(?:='([^']*)')?\]/.exec(rest);
    if (m) {
      spec += 10;
      if (m[2] === undefined ? !(m[1] in a) : a[m[1]] !== m[2]) ok = false;
      rest = rest.slice(m[0].length);
      continue;
    }
    m = /^:not\(\[([\w-]+)(?:='([^']*)')?\]\)/.exec(rest);
    if (m) {
      spec += 10;
      if (m[2] === undefined ? m[1] in a : a[m[1]] === m[2]) ok = false;
      rest = rest.slice(m[0].length);
      continue;
    }
    return null;
  }
  return { ok, spec };
}

function classify(sel: string, a: Record<string, string>): { target: Target; spec: number } | null {
  const s = sel.trim();
  if (s === '.lab') return { target: 'lab', spec: 10 };
  const m = /^(.*?)\s+\.lab$/.exec(s);
  if (m) {
    const r = rootMatches(m[1], a);
    return r && r.ok ? { target: 'lab', spec: r.spec + 10 } : null;
  }
  const r = rootMatches(s, a);
  return r && r.ok ? { target: 'root', spec: r.spec } : null;
}

const RULES: Rule[] = topLevelRules(CSS_NO_COMMENTS);

/** The custom properties the `.lab` element sees for one style (html cascade, then .lab rules). */
export function labCascade(style: LabStyle, rules: Rule[] = RULES): Map<string, string> {
  const a = attrs(style);
  const hits: { target: Target; spec: number; order: number; decls: [string, string][] }[] = [];
  rules.forEach((r, order) => {
    const decls = declarations(r.body);
    if (!decls.length) return;
    for (const sel of r.selectors) {
      const c = classify(sel, a);
      if (c) hits.push({ ...c, order, decls });
    }
  });
  const out = new Map<string, string>();
  for (const target of ['root', 'lab'] as Target[]) {
    for (const h of hits.filter((x) => x.target === target).sort((x, y) => x.spec - y.spec || x.order - y.order)) {
      for (const [k, v] of h.decls) out.set(k, v);
    }
  }
  return out;
}

/**
 * The custom properties <html> carries for any set of appearance attributes (data-ui, data-theme,
 * data-mode, data-accent, data-ground): the root-level rules of app/globals.css in cascade order.
 */
export function rootCascade(a: Record<string, string>, rules: Rule[] = RULES): Map<string, string> {
  const hits: { spec: number; order: number; decls: [string, string][] }[] = [];
  rules.forEach((r, order) => {
    const decls = declarations(r.body);
    if (!decls.length) return;
    for (const sel of r.selectors) {
      const c = rootMatches(sel.trim(), a);
      if (c?.ok) hits.push({ spec: c.spec, order, decls });
    }
  });
  const out = new Map<string, string>();
  for (const h of hits.sort((x, y) => x.spec - y.spec || x.order - y.order)) for (const [k, v] of h.decls) out.set(k, v);
  return out;
}

/** Split on commas at paren depth 0. */
function splitTop(s: string): string[] {
  const out: string[] = [];
  let d = 0, cur = '';
  for (const c of s) {
    if (c === '(') d++;
    if (c === ')') d--;
    if (c === ',' && d === 0) { out.push(cur.trim()); cur = ''; } else cur += c;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

/** Substitute var() (with fallbacks), innermost first. Throws on an unresolved name or a cycle. */
export function substitute(value: string, env: (t: string) => string | undefined, depth = 0): string {
  if (depth > 40) throw new Error(`var() depth exceeded in ${value}`);
  const i = value.lastIndexOf('var(');
  if (i < 0) return value;
  let d = 0, j = i + 3;
  for (; j < value.length; j++) { if (value[j] === '(') d++; else if (value[j] === ')' && --d === 0) break; }
  const inner = value.slice(i + 4, j);
  const [name, ...fb] = splitTop(inner);
  const sub = env(name.trim()) ?? (fb.length ? fb.join(',') : undefined);
  if (sub === undefined) throw new Error(`unresolved ${name}`);
  return substitute(value.slice(0, i) + sub + value.slice(j + 1), env, depth + 1);
}

function parseHex(s: string): RGBA | null {
  const v = s.trim().toLowerCase();
  if (v === 'transparent') return [0, 0, 0, 0];
  let m = /^#([0-9a-f]{3})$/.exec(v);
  if (m) return [...m[1]].map((c) => parseInt(c + c, 16)).concat(1) as RGBA;
  m = /^#([0-9a-f]{6})([0-9a-f]{2})?$/.exec(v);
  if (m) return [0, 2, 4].map((k) => parseInt(m![1].slice(k, k + 2), 16)).concat(m[2] ? parseInt(m[2], 16) / 255 : 1) as RGBA;
  m = /^rgba?\(\s*([\d.]+)[\s,]+([\d.]+)[\s,]+([\d.]+)(?:\s*[,/]\s*([\d.]+%?))?\s*\)$/.exec(v);
  if (m) {
    const al = m[4] === undefined ? 1 : m[4].endsWith('%') ? Number(m[4].slice(0, -1)) / 100 : Number(m[4]);
    return [Number(m[1]), Number(m[2]), Number(m[3]), al];
  }
  return null;
}

/** Evaluate a colour expression (hex, rgb(a), transparent, color-mix in srgb, nested). Premultiplied, as CSS does. */
export function evalColour(expr: string): RGBA {
  const v = expr.trim();
  const direct = parseHex(v);
  if (direct) return direct;
  const m = /^color-mix\(\s*in srgb\s*,(.*)\)$/s.exec(v);
  if (!m) throw new Error(`not a colour: ${v}`);
  const [a, b] = splitTop(m[1]);
  const part = (s: string): { c: RGBA; p: number | null } => {
    const pm = /^(.*\S)\s+([\d.]+)%$/s.exec(s.trim());
    return pm ? { c: evalColour(pm[1]), p: Number(pm[2]) / 100 } : { c: evalColour(s), p: null };
  };
  const A = part(a), B = part(b);
  let pa = A.p ?? (B.p !== null ? 1 - B.p : 0.5);
  let pb = B.p ?? 1 - pa;
  const sum = pa + pb;
  pa /= sum; pb /= sum;
  const alpha = A.c[3] * pa + B.c[3] * pb;
  if (alpha === 0) return [0, 0, 0, 0];
  const ch = (k: number) => (A.c[k] * A.c[3] * pa + B.c[k] * B.c[3] * pb) / alpha;
  return [ch(0), ch(1), ch(2), alpha];
}

/** Resolve a token in a style to a colour. */
export function labColour(env: Map<string, string>, token: string): RGBA {
  return evalColour(substitute(`var(${token})`, (t) => env.get(t)));
}

/** Resolve any string (box shadows, borders) with colours rewritten as rgba(). */
export function labResolveText(env: Map<string, string>, token: string): string {
  const raw = substitute(`var(${token})`, (t) => env.get(t));
  let out = raw;
  for (let guard = 0; guard < 20; guard++) {
    const i = out.indexOf('color-mix(');
    if (i < 0) break;
    let d = 0, j = i;
    for (; j < out.length; j++) { if (out[j] === '(') d++; else if (out[j] === ')' && --d === 0) break; }
    const c = evalColour(out.slice(i, j + 1));
    out = `${out.slice(0, i)}rgba(${c.slice(0, 3).map((x) => Math.round(x)).join(', ')}, ${+c[3].toFixed(3)})${out.slice(j + 1)}`;
  }
  return out;
}

/** a over b (b opaque). */
export function over(a: RGBA, b: RGBA): RGBA {
  return [0, 1, 2].map((k) => a[k] * a[3] + b[k] * (1 - a[3])).concat(1) as RGBA;
}
const lin = (c: number) => { const s = c / 255; return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4; };
const lum = (c: RGBA) => 0.2126 * lin(c[0]) + 0.7152 * lin(c[1]) + 0.0722 * lin(c[2]);
/** WCAG ratio of a (composited over b when translucent) against opaque b. */
export function ratio(a: RGBA, b: RGBA): number {
  const fg = a[3] < 1 ? over(a, b) : a;
  const [x, y] = [lum(fg), lum(b)].sort((p, q) => q - p);
  return (x + 0.05) / (y + 0.05);
}
export const toHex = (c: RGBA) => `#${c.slice(0, 3).map((x) => Math.round(x).toString(16).padStart(2, '0')).join('')}`;
