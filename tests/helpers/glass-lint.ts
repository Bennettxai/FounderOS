import fs from 'node:fs';
import path from 'node:path';
import ts from 'typescript';
import { CSS_NO_COMMENTS, splitSelectors, topLevelRules, type Rule } from './theme-css';

/**
 * Static glass lint (FounderOS Glass brief §3.2): a glass face must never sit inside an element
 * with `filter`, `opacity < 1`, `mask` or `backdrop-filter`. Each of those makes the element a
 * backdrop root, so the face's frost stops there and the face reads as see-through.
 *
 * The scan parses TSX (the TypeScript compiler, no regex over markup) and walks every JSX tree
 * with the stack of its ancestors. An ancestor kills the frost when its className names a
 * Tailwind utility that sets one of those properties, or a class whose rule in app/globals.css
 * does (passed in as `killers`), or its style object sets one. A face is an element whose
 * className names a face class, or a kit component that renders one in place (not a portal).
 *
 * Faces inside faces are not reported: glass over glass is the brief's own pattern (the lower
 * face frosts the page once; the upper one is lit, not frosted again). app/globals.css holds that
 * exemption up (glass phase 3: every face nested in a frosting face drops its own frost, chips are
 * lit like wells, pop-ups are solid raised faces; tests/glass-lint.test.ts checks the rule covers
 * every face). A face that frosts itself again inside a face (a backdrop utility or style of its
 * own) is reported. The Classic card, an os surface with the card radius (glass phase 2b), is a
 * face like the classes are (FACE_COMPOUNDS).
 */

/** The face classes of the glass pack (app/globals.css, "The face." rule). */
export const FACE_CLASSES = [
  'glass-face', 'glass-face--strong', 'l-card', 'l-slab', 'l-chip', 'l-dialog', 'g-card', 'g-card-raised', 'g-well', 'g-chip',
  'g-sheet', 'g-drawer', 'g-dialog', 'g-palette', 'ui-toast', 'c-pop', 'c-chip', 'h6-card', 'glass-panel', 'glass-panel-violet',
  'glass-chip', 'lab2-glass', 'wiz-ask', 'g-ask-box',
] as const;

/**
 * Faces named by a compound: an element that carries every class (and none of `unless`) is a face.
 * The Classic card: bg-os-surface + rounded-lg-t (glass phase 2b maps it), a dashed drop zone excepted.
 */
export const FACE_COMPOUNDS: readonly { label: string; classes: readonly string[]; unless: readonly string[] }[] = [
  { label: 'bg-os-surface.rounded-lg-t', classes: ['bg-os-surface', 'rounded-lg-t'], unless: ['border-dashed'] },
];

/** The face an element's class tokens make it, if any: a face class, else a face compound's label. */
export function faceOf(tokens: readonly string[]): string | undefined {
  const faces = new Set<string>(FACE_CLASSES);
  const hit = tokens.find((t) => faces.has(t));
  if (hit) return hit;
  const have = new Set(tokens);
  return FACE_COMPOUNDS.find((c) => c.classes.every((k) => have.has(k)) && !c.unless.some((k) => have.has(k)))?.label;
}

/** Kit components that render a face in place. GSheet, GDrawer and GDialog portal to <body>, so nothing wraps them. */
export const FACE_COMPONENTS: Record<string, string> = {
  GCard: 'g-card',
  GCardRaised: 'g-card-raised',
  GWell: 'g-well',
  GChip: 'g-chip',
  GAsk: 'g-ask-box',
  LDialog: 'l-dialog',
  SlabCard: 'l-card',
  Glass: 'lab2-glass',
};

/** Pseudo-element variants style something other than the element itself. */
const PSEUDO_VARIANT = /^(?:placeholder|before|after|file|marker|selection|first-letter|first-line|backdrop):/;

/** The utility without its variants (`md:hover:` …); colons inside [arbitrary] brackets are not variant separators. */
function utilityBase(token: string): string {
  let depth = 0;
  let cut = 0;
  for (let i = 0; i < token.length; i++) {
    const c = token[i];
    if (c === '[') depth++;
    else if (c === ']') depth--;
    else if (c === ':' && depth === 0) cut = i + 1;
  }
  return token.slice(cut).replace(/^!/, '');
}

/** Tailwind utilities that set filter, backdrop-filter, opacity < 1 or mask on the element. */
export function killerUtility(token: string): string | null {
  if (PSEUDO_VARIANT.test(token)) return null;
  const base = utilityBase(token);
  if (/^backdrop-(?:blur|brightness|contrast|grayscale|hue-rotate|invert|opacity|saturate|sepia)(?:-|$)/.test(base) && !/-none$/.test(base)) return 'backdrop-filter';
  if (base === 'backdrop-filter') return 'backdrop-filter';
  if (/^(?:blur|brightness|contrast|grayscale|hue-rotate|invert|saturate|sepia|drop-shadow)(?:-|$)/.test(base) && !/-none$/.test(base) && !/^(?:grayscale|invert|sepia)-0$/.test(base)) return 'filter';
  if (base === 'filter') return 'filter';
  if (/^opacity-(\d+)$/.test(base)) return Number(base.slice(8)) < 100 ? 'opacity' : null;
  if (/^opacity-\[/.test(base)) return 'opacity';
  if (/^mask(?:-|$)/.test(base) || /^\[(?:-webkit-)?mask/.test(base)) return 'mask';
  if (/^\[(?:backdrop-)?filter:/.test(base)) return /none\]$/.test(base) ? null : 'filter';
  if (/^\[opacity:/.test(base)) return 'opacity';
  return null;
}

const STYLE_KILLERS: Record<string, string> = {
  filter: 'filter',
  backdropFilter: 'backdrop-filter',
  WebkitBackdropFilter: 'backdrop-filter',
  mask: 'mask',
  maskImage: 'mask',
  WebkitMask: 'mask',
  WebkitMaskImage: 'mask',
  opacity: 'opacity',
};

type Node = ts.JsxElement | ts.JsxSelfClosingElement;

function tagName(n: Node): string {
  const open = ts.isJsxElement(n) ? n.openingElement : n;
  return open.tagName.getText();
}

function attr(n: Node, name: string): ts.JsxAttribute | undefined {
  const open = ts.isJsxElement(n) ? n.openingElement : n;
  return open.attributes.properties.find((p): p is ts.JsxAttribute => ts.isJsxAttribute(p) && p.name.getText() === name);
}

/** Every literal class token a className expression can produce (strings, templates, cx() and ternary arms). */
export function classTokens(n: Node): string[] {
  const a = attr(n, 'className');
  if (!a?.initializer) return [];
  const out: string[] = [];
  const visit = (x: ts.Node) => {
    if (ts.isStringLiteral(x) || ts.isNoSubstitutionTemplateLiteral(x)) out.push(...x.text.split(/\s+/));
    else if (ts.isTemplateExpression(x)) {
      out.push(...x.head.text.split(/\s+/));
      for (const s of x.templateSpans) {
        visit(s.expression);
        out.push(...s.literal.text.split(/\s+/));
      }
      return;
    }
    ts.forEachChild(x, visit);
  };
  visit(a.initializer);
  return out.filter(Boolean);
}

/** Frost killers set by a literal style object. */
function styleKillers(n: Node): string[] {
  const a = attr(n, 'style');
  const e = a?.initializer && ts.isJsxExpression(a.initializer) ? a.initializer.expression : undefined;
  if (!e || !ts.isObjectLiteralExpression(e)) return [];
  const out: string[] = [];
  for (const p of e.properties) {
    if (!ts.isPropertyAssignment(p)) continue;
    const key = p.name.getText().replace(/^['"]|['"]$/g, '');
    const kind = STYLE_KILLERS[key];
    if (!kind) continue;
    const v = p.initializer;
    if (ts.isStringLiteral(v) && /^\s*none\s*$/.test(v.text)) continue;
    if (kind === 'opacity' && ts.isNumericLiteral(v) && Number(v.text) >= 1) continue;
    out.push(`style.${key}`);
  }
  return out;
}

export type Finding = { file: string; line: number; face: string; ancestor: string; why: string };

/** A CSS compound (every class it names) and the frost-killing property its rule sets. */
export type Killer = { classes: string[]; why: string };

/** Keyframes that animate opacity or filter (a parent playing one fades or filters its faces). */
export function fadingKeyframes(css: string = CSS_NO_COMMENTS): Set<string> {
  return new Set([...keyframeBodies(css)].filter(([, b]) => /(^|[;{\s])(opacity|filter)\s*:/.test(b)).map(([n]) => n));
}

/**
 * Classes whose rule in app/globals.css sets backdrop-filter, filter, opacity < 1 or mask, or plays
 * a fading keyframe: the last compound of each selector (`.bh-ask.has-chips .bh-ask-chips` gives
 * ['bh-ask-chips']; `.g-btn.is-busy` gives both, and an element must carry both). Face classes are
 * left out (glass over glass). Top-level rules only.
 */
export function cssKillers(css: string = CSS_NO_COMMENTS): Killer[] {
  const fading = fadingKeyframes(css);
  const faces = new Set<string>(FACE_CLASSES);
  const out: Killer[] = [];
  const seen = new Set<string>();
  for (const r of topLevelRules(css)) {
    const b = r.body;
    let why: string | null = null;
    if (/(^|[;\s])(-webkit-)?backdrop-filter\s*:\s*(?!none)/.test(b)) why = 'backdrop-filter';
    else if (/(^|[;\s])filter\s*:\s*(?!none)/.test(b)) why = 'filter';
    else if (/(^|[;\s])opacity\s*:\s*(0?\.\d+|0)\s*(;|$)/.test(b)) why = 'opacity';
    else if (/(^|[;\s])(-webkit-)?mask(-image)?\s*:\s*(?!none)/.test(b)) why = 'mask';
    else {
      const a = /(?:^|[;\s])animation(?:-name)?\s*:\s*([^;]+)/.exec(b);
      const names = a ? a[1].split(/[\s,]+/).filter((n) => fading.has(n)) : [];
      if (names.length) why = `animation ${names.join(',')}`;
    }
    if (!why) continue;
    for (const sel of r.selectors) {
      const last = sel.replace(/:(?:not|is|where|has)\([^)]*\)/g, '').split(/\s*[\s>+~]\s*/).pop() ?? '';
      const classes = [...last.matchAll(/\.([\w-]+)/g)].map((m) => m[1]);
      if (!classes.length || classes.some((c) => faces.has(c))) continue;
      if (FACE_COMPOUNDS.some((f) => f.classes.every((k) => classes.includes(k)))) continue;
      const key = [...classes].sort().join('.');
      if (seen.has(key)) continue;
      seen.add(key);
      out.push({ classes, why });
    }
  }
  return out;
}

/** Scan one TSX source against the CSS-derived killers (and the Tailwind utilities and style objects). */
export function lintSource(file: string, src: string, killers: readonly Killer[] = []): Finding[] {
  const sf = ts.createSourceFile(file, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const findings: Finding[] = [];
  type Frame = { label: string; why: string };
  const labelOf = (tag: string, tokens: string[]) => `<${tag} className="${tokens.join(' ').slice(0, 80)}">`;
  /** `stack`: the frost killers above; `inFace`: the nearest face above with no killer between. */
  const walk = (x: ts.Node, stack: Frame[], inFace: string | null) => {
    if (ts.isJsxElement(x) || ts.isJsxSelfClosingElement(x)) {
      const tag = tagName(x);
      const tokens = classTokens(x);
      const face = faceOf(tokens) ?? FACE_COMPONENTS[tag];
      const line = () => sf.getLineAndCharacterOfPosition(x.getStart()).line + 1;
      const have = new Set(tokens);
      const own = [
        ...tokens.flatMap((t) => {
          const u = killerUtility(t);
          return u ? [`${t} (${u})`] : [];
        }),
        ...killers.filter((k) => k.classes.every((c) => have.has(c))).map((k) => `.${k.classes.join('.')} (${k.why})`),
        ...styleKillers(x),
      ];
      if (face && stack.length) {
        const k = stack[stack.length - 1];
        findings.push({ file, line: line(), face, ancestor: k.label, why: k.why });
      } else if (face && inFace) {
        // glass over glass: the lower face frosts once; this one must not frost again
        const again = own.filter((w) => /backdrop-filter|[bB]ackdropFilter/.test(w));
        if (again.length) findings.push({ file, line: line(), face, ancestor: inFace, why: `${again.join(', ')} frosts again inside ${inFace}` });
      }
      const next = own.length ? [...stack, { label: labelOf(tag, tokens), why: own.join(', ') }] : stack;
      const nextFace = own.length ? null : face ? labelOf(tag, tokens) : inFace;
      ts.forEachChild(x, (c) => walk(c, next, nextFace));
      return;
    }
    ts.forEachChild(x, (c) => walk(c, stack, inFace));
  };
  walk(sf, [], null);
  return findings;
}

/** The class tokens of every face element in a TSX source (what CSS can match on a face). */
export function faceElementClasses(file: string, src: string): string[][] {
  const sf = ts.createSourceFile(file, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const out: string[][] = [];
  const walk = (x: ts.Node) => {
    if (ts.isJsxElement(x) || ts.isJsxSelfClosingElement(x)) {
      const tokens = classTokens(x);
      const comp = FACE_COMPONENTS[tagName(x)];
      if (comp) out.push([comp, ...tokens]);
      else if (faceOf(tokens)) out.push(tokens);
    }
    ts.forEachChild(x, walk);
  };
  walk(sf);
  return out;
}

/** Every style rule, including the ones inside @media and @supports blocks. */
export function allRules(src: string = CSS_NO_COMMENTS): Rule[] {
  const out: Rule[] = [];
  const walk = (s: string) => {
    let depth = 0;
    let start = 0;
    let open = -1;
    for (let i = 0; i < s.length; i++) {
      const c = s[i];
      if (c === '{') {
        if (depth === 0) open = i;
        depth++;
      } else if (c === '}') {
        depth--;
        if (depth === 0) {
          const head = s.slice(start, open).trim();
          const body = s.slice(open + 1, i);
          if (/^@(media|supports|layer)\b/.test(head)) walk(body);
          else if (!head.startsWith('@')) out.push({ selectors: splitSelectors(head), body });
          start = i + 1;
        }
      } else if (c === ';' && depth === 0) start = i + 1;
    }
  };
  walk(src);
  return out;
}

/** Keyframe bodies by name (a name declared twice keeps both bodies). */
export function keyframeBodies(css: string = CSS_NO_COMMENTS): Map<string, string> {
  const out = new Map<string, string>();
  for (const m of css.matchAll(/@keyframes\s+([\w-]+)\s*\{((?:[^{}]*\{[^{}]*\})*[^{}]*)\}/g)) out.set(m[1], (out.get(m[1]) ?? '') + m[2]);
  return out;
}

/**
 * The keyframes a face plays: every rule whose subject (the last compound, or each member of a
 * trailing :is()) names only classes some face element carries, and that sets an animation.
 * Returns keyframe name -> the selectors that play it on a face.
 */
export function faceKeyframes(faceElements: readonly string[][], css: string = CSS_NO_COMMENTS): Map<string, string[]> {
  const kf = keyframeBodies(css);
  const faces = new Set<string>(FACE_CLASSES);
  const sets = [...faceElements.map((t) => new Set(t)), ...FACE_CLASSES.map((c) => new Set([c]))];
  const out = new Map<string, string[]>();
  for (const r of allRules(css)) {
    const decls = [...r.body.matchAll(/(?:^|[;\s])animation(?:-name)?\s*:\s*([^;]+)/g)].map((m) => m[1]);
    if (!decls.length) continue;
    const names = decls.flatMap((d) => d.split(/[\s,]+/)).filter((n) => kf.has(n));
    if (!names.length) continue;
    for (const sel of r.selectors) {
      const is = /:is\(([\s\S]*)\)\s*$/.exec(sel);
      const subjects = is ? splitSelectors(is[1]) : [sel.split(/\s*[\s>+~]\s*/).pop() ?? ''];
      const onFace = subjects.some((sub) => {
        const classes = [...sub.replace(/:(?:not|has)\([^)]*\)/g, '').matchAll(/\.([\w-]+)/g)].map((m) => m[1]);
        if (!classes.length) return false;
        if (classes.some((c) => faces.has(c))) return true;
        return sets.some((s) => classes.every((c) => s.has(c)) && faceOf([...s]) !== undefined);
      });
      if (!onFace) continue;
      for (const n of names) out.set(n, [...(out.get(n) ?? []), sel]);
    }
  }
  return out;
}

export function tsxFiles(root: string, dirs: string[]): string[] {
  const out: string[] = [];
  const visit = (d: string) => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) visit(p);
      else if (e.name.endsWith('.tsx')) out.push(p);
    }
  };
  for (const d of dirs) visit(path.join(root, d));
  return out.sort();
}
