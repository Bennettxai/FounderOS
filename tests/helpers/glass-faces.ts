import fs from 'node:fs';
import path from 'node:path';
import ts from 'typescript';
import { splitSelectors } from './theme-css';
import { classTokens, FACE_COMPONENTS, faceOf, localStyleSources } from './glass-lint';

/**
 * The final glass sweep (glass phase 5): what a face is, what the old look is, and whether the glass pack
 * (the GLASS PACK block and the phase blocks after it, the tail of app/globals.css) wins over a face's own
 * rule. tests/glass-final-sweep.test.ts reads every stylesheet and every TSX element through these.
 *
 * A face is a card, panel, tile, sheet, popover, menu, toast, dock, chip or pill (brief §5B; glass canon).
 * In CSS that is a rule whose subject names a face class (FACE_NAME); in TSX an element with a radius
 * that paints a surface utility (TSX_FILL). The old look is a fill (an opaque or semi-opaque colour, a
 * surface or card token, a gradient), a top-edge highlight, or a radius off the pack's three (card 24,
 * well 16, chip a pill). A fill is gone when a pack rule clears it at a higher specificity (or equal and
 * later in the same sheet): the cascade, computed here on the selectors, not by reading the bodies alone.
 */

export type Spec = [number, number, number];

/** CSS selector specificity (Selectors 4: :is/:not/:has take their most specific argument, :where is 0). */
export function specificity(sel: string): Spec {
  const s: Spec = [0, 0, 0];
  let i = 0;
  const add = (o: Spec) => {
    s[0] += o[0];
    s[1] += o[1];
    s[2] += o[2];
  };
  const args = (from: number): [string, number] => {
    let depth = 1;
    let j = from;
    while (j < sel.length && depth) {
      if (sel[j] === '(') depth++;
      else if (sel[j] === ')') depth--;
      j++;
    }
    return [sel.slice(from, j - 1), j];
  };
  while (i < sel.length) {
    const c = sel[i];
    if (c === '#') {
      s[0]++;
      i++;
      while (i < sel.length && /[\w-]/.test(sel[i])) i++;
    } else if (c === '.') {
      s[1]++;
      i++;
      while (i < sel.length && /[\w-]/.test(sel[i])) i++;
    } else if (c === '[') {
      s[1]++;
      while (i < sel.length && sel[i] !== ']') i++;
      i++;
    } else if (c === ':') {
      if (sel[i + 1] === ':') {
        s[2]++;
        i += 2;
        while (i < sel.length && /[\w-]/.test(sel[i])) i++;
        if (sel[i] === '(') i = args(i + 1)[1];
        continue;
      }
      i++;
      let name = '';
      while (i < sel.length && /[\w-]/.test(sel[i])) name += sel[i++];
      if (/^(before|after|first-line|first-letter|placeholder|selection|marker)$/.test(name)) {
        s[2]++;
        continue;
      }
      if (sel[i] === '(') {
        const [inner, next] = args(i + 1);
        i = next;
        if (name === 'where') continue;
        if (name === 'is' || name === 'not' || name === 'has' || name === 'matches') {
          const max = splitSelectors(inner)
            .map(specificity)
            .reduce<Spec>((m, x) => (cmp(x, m) > 0 ? x : m), [0, 0, 0]);
          add(max);
          continue;
        }
        s[1]++;
        continue;
      }
      s[1]++;
    } else if (/[a-zA-Z]/.test(c)) {
      s[2]++;
      while (i < sel.length && /[\w-]/.test(sel[i])) i++;
    } else i++;
  }
  return s;
}

export function cmp(a: Spec, b: Spec): number {
  return a[0] - b[0] || a[1] - b[1] || a[2] - b[2];
}

/** A selector as [ancestor compounds…, subject compound] on its top-level combinators. */
export function compounds(sel: string): string[] {
  const out: string[] = [];
  let depth = 0;
  let cur = '';
  for (const c of sel.trim()) {
    if (c === '(' || c === '[') depth++;
    if (c === ')' || c === ']') depth--;
    if (depth === 0 && /[\s>+~]/.test(c)) {
      if (cur) out.push(cur);
      cur = '';
    } else cur += c;
  }
  if (cur) out.push(cur);
  return out;
}

/** The alternatives a compound stands for: `.a:is(.b, .c).d` gives `.a.b.d` and `.a.c.d` (a leading :is() too). */
export function alternatives(compound: string): string[] {
  const m = /:is\(/.exec(compound);
  if (!m) return [compound];
  let depth = 1;
  let j = m.index + 4;
  while (j < compound.length && depth) {
    if (compound[j] === '(') depth++;
    else if (compound[j] === ')') depth--;
    j++;
  }
  const before = compound.slice(0, m.index);
  const after = compound.slice(j);
  return splitSelectors(compound.slice(m.index + 4, j - 1)).flatMap((alt) =>
    alt.includes(' ') ? [] : alternatives(`${before}${alt}${after}`),
  );
}

/** The class, attribute and state tokens a compound requires (:not() and :where() asides dropped). */
export function requires(compound: string): string[] {
  const bare = compound.replace(/:(?:not|where|has)\((?:[^()]|\([^()]*\))*\)/g, '');
  return [...bare.matchAll(/\.[\w-]+|\[[^\]]+\]|:(?!:)[\w-]+(?:\([^)]*\))?/g)].map((m) => m[0].replace(/\s+/g, '')).filter((t) => t !== ':root');
}

export type CssRule = { file: string; order: number; line: number; selectors: string[]; body: string; at: string[] };

/** Every style rule of a sheet with its at-rule context (keyframes left out), in source order. */
export function cssRules(file: string, raw: string, order0 = 0): CssRule[] {
  const src = raw.replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, ' '));
  const out: CssRule[] = [];
  const heads: { head: string; open: number }[] = [];
  let start = 0;
  for (let i = 0; i < src.length; i++) {
    const c = src[i];
    if (c === '{') {
      heads.push({ head: src.slice(start, i).trim(), open: i });
      start = i + 1;
    } else if (c === '}') {
      const h = heads.pop();
      if (h && !h.head.startsWith('@') && !heads.some((x) => /^@keyframes/.test(x.head))) {
        const body = src.slice(h.open + 1, i);
        if (!body.includes('{'))
          out.push({
            file,
            order: order0 + out.length,
            line: src.slice(0, h.open).split('\n').length,
            selectors: splitSelectors(h.head.replace(/\s+/g, ' ')),
            body,
            at: heads.map((x) => x.head.replace(/\s+/g, ' ')),
          });
      }
      start = i + 1;
    } else if (c === ';' && heads.length === 0) start = i + 1;
  }
  return out;
}

/** The value a rule body gives a property (the last declaration), !important noted. */
export function decl(body: string, prop: string): { value: string; important: boolean } | null {
  const re = new RegExp(`(?:^|[;{\\s])${prop.replace(/-/g, '\\-')}\\s*:\\s*([^;]+)`, 'g');
  let last: string | null = null;
  for (const m of body.matchAll(re)) last = m[1].trim();
  if (last == null) return null;
  return { value: last.replace(/\s*!important$/, ''), important: /!important$/.test(last) };
}

/** Face words: a class whose last word is one of these names a face (a part such as -dot, -item or -close does not). */
export const FACE_NAME =
  /(?:^|-)(?:card|panel|tile|sheet|pop|popover|popup|menu|toast|dock|chip|pill|strip|modal|dialog|drawer|tip|tooltip|bubble|banner|stat|hud|table|frame)(?:-(?:raised|violet|accent|user|strong|ghost|solid|in|lg|sm|2|3))?$/;

/** No fill: what a face's background may be. */
const NO_FILL = /^(?:transparent|none|inherit|initial|unset|var\(--solid-bg\))$/;

/** The pack's radii, and the style tokens that resolve to them (tests/glass-sweep.test.ts holds the tokens). */
const RADIUS_OK =
  /^(?:0|0px|24px|16px|999px|9999px|50%|inherit|var\(--gl-r-(?:card|well|chip)\)|var\(--r-card(?:,\s*0px)?\)|var\(--g-r-(?:card|well|pill)\)|var\(--l-r-(?:tile|chip)\)|var\(--lab-r-(?:narrative|tile|sheet|well|chip|pill)\))$/;

export function radiusOk(v: string): boolean {
  const parts = splitSelectors(v.replace(/\s+/g, ' ').trim()).join(' ').split(/\s+(?![^(]*\))/);
  return parts.every((p) => RADIUS_OK.test(p.trim()));
}

export type Issue = 'fill' | 'top-highlight' | 'radius';

/** What a face rule paints of the old look. */
export function oldLook(body: string): { issue: Issue; detail: string }[] {
  const out: { issue: Issue; detail: string }[] = [];
  for (const p of ['background', 'background-color', 'background-image']) {
    const d = decl(body, p);
    if (d && !NO_FILL.test(d.value)) out.push({ issue: 'fill', detail: `${p}: ${d.value.slice(0, 70)}` });
  }
  const sh = decl(body, 'box-shadow');
  if (sh && /(?:^|,)\s*inset\s+0(?:px)?\s+1px\s+0/.test(sh.value)) out.push({ issue: 'top-highlight', detail: `box-shadow: ${sh.value.slice(0, 70)}` });
  const r = decl(body, 'border-radius');
  if (r && !radiusOk(r.value)) out.push({ issue: 'radius', detail: `border-radius: ${r.value}` });
  return out;
}

const PROP_OF: Record<Issue, string[]> = {
  fill: ['background', 'background-color', 'background-image'],
  'top-highlight': ['box-shadow'],
  radius: ['border-radius'],
};

/** The tail of app/globals.css from the GLASS PACK face rules on: the pack, and every phase block after it. */
export const PACK_MARK = '/* ════ GLASS PACK (FounderOS Glass brief';

/**
 * subjects: the subject's alternatives (each the tokens it requires); excludes: per alternative, the simple selectors
 * its :not() leaves out (a class, or an element name); context: per ancestor compound, its alternatives.
 */
export type Clearer = { rule: CssRule; selector: string; spec: Spec; subjects: string[][]; excludes: string[][]; context: string[][][]; props: Set<string> };

/** The simple arguments of a compound's :not()s (`.a:not(.b, select)` gives ['.b', 'select']). */
export function nots(compound: string): string[] {
  return [...compound.matchAll(/:not\(((?:[^()]|\([^()]*\))*)\)/g)].flatMap((m) => splitSelectors(m[1])).filter((a) => !/\s/.test(a));
}

/**
 * Pack rules that clear the old look: a background that is no fill (transparent, none, the solid face),
 * a box-shadow without a top highlight, a radius on the pack's three. Rules under a media query that does
 * not always hold (reduced transparency, a breakpoint) clear nothing in general and are left out.
 */
export function clearers(pack: CssRule[]): Clearer[] {
  const out: Clearer[] = [];
  for (const r of pack) {
    if (r.at.length) continue;
    const props = new Set<string>();
    for (const p of PROP_OF.fill) {
      const d = decl(r.body, p);
      if (d && NO_FILL.test(d.value)) props.add(p);
    }
    const bg = decl(r.body, 'background');
    if (bg && NO_FILL.test(bg.value)) PROP_OF.fill.forEach((p) => props.add(p));
    const sh = decl(r.body, 'box-shadow');
    if (sh && !/(?:^|,)\s*inset\s+0(?:px)?\s+1px\s+0/.test(sh.value)) props.add('box-shadow');
    const rad = decl(r.body, 'border-radius');
    if (rad && radiusOk(rad.value)) props.add('border-radius');
    if (!props.size) continue;
    for (const sel of r.selectors) {
      if (/::/.test(sel)) continue;
      const cs = compounds(sel);
      const subj = cs[cs.length - 1];
      out.push({
        rule: r,
        selector: sel,
        spec: specificity(sel),
        subjects: alternatives(subj).map(requires),
        excludes: alternatives(subj).map(nots),
        context: cs.slice(0, -1).map((c) => alternatives(c).map((a) => requires(a).filter((t) => t !== '[data-ground]'))),
        props,
      });
    }
  }
  return out;
}

/**
 * Whether the pack wins over a face rule's property: a clearer whose subject the face's subject satisfies,
 * whose context the face's selector names, at a higher specificity (or equal and later in the same sheet).
 * `carries` adds the pack class a face class is always rendered with (CARRIES in the test).
 */
export function cleared(face: CssRule, sel: string, prop: string, all: Clearer[], carries: Record<string, string[]> = {}): boolean {
  const cs = compounds(sel);
  const subj = cs[cs.length - 1];
  const base = requires(subj);
  // a face class always rendered with a pack class: every pack class it comes with must be cleared
  const extra = base.flatMap((t) => (t.startsWith('.') ? (carries[t.slice(1)] ?? []) : []));
  if (extra.length) return extra.every((k) => clearedFor(face, sel, prop, all, new Set([...base, ...k.split('.').map((c) => `.${c}`)])));
  return clearedFor(face, sel, prop, all, new Set(base));
}

/** Whether a :not() argument matches the face's subject: a class it requires, or its element name. */
function excluded(arg: string, subj: string, own: Set<string>): boolean {
  if (/^[a-z][\w-]*$/i.test(arg)) return new RegExp(`^${arg}(?![\\w-])`, 'i').test(subj);
  const toks = requires(arg);
  return toks.length > 0 && toks.every((t) => own.has(t));
}

function clearedFor(face: CssRule, sel: string, prop: string, all: Clearer[], own: Set<string>): boolean {
  const cs = compounds(sel);
  const subj = cs[cs.length - 1];
  const ctx = new Set(cs.slice(0, -1).flatMap((c) => requires(c)));
  const sp = specificity(sel);
  return all.some((c) => {
    if (!c.props.has(prop)) return false;
    if (!c.subjects.some((alt, i) => alt.every((t) => own.has(t) || t === '[data-ground]') && !c.excludes[i].some((x) => excluded(x, subj, own)))) return false;
    if (!c.context.every((alts) => alts.some((a) => a.every((t) => ctx.has(t))))) return false;
    const d = cmp(c.spec, sp);
    return d > 0 || (d === 0 && c.rule.file === face.file && c.rule.order > face.order);
  });
}

/** The face selectors a selector stands for: one per subject alternative that names a face class. */
export function faceSelectors(sel: string): { face: string; selector: string }[] {
  const cs = compounds(sel);
  const subj = cs[cs.length - 1];
  if (/::/.test(subj)) return [];
  const ctx = cs.slice(0, -1).join(' ');
  const out: { face: string; selector: string }[] = [];
  for (const alt of alternatives(subj)) {
    const hit = requires(alt)
      .filter((t) => t.startsWith('.'))
      .map((t) => t.slice(1))
      .find((c) => FACE_NAME.test(c));
    if (hit) out.push({ face: hit, selector: ctx ? `${ctx} ${alt}` : alt });
  }
  return out;
}

/** The face class a selector's subject names, if any. */
export function faceClassOf(sel: string): string | undefined {
  return faceSelectors(sel)[0]?.face;
}

export type CssFinding = { key: string; file: string; line: number; face: string; selector: string; issue: Issue; detail: string };

/** Every stylesheet the app ships: app/globals.css first, then component sheets and <style> blocks (bundled after it). */
export function sheets(root: string): { file: string; css: string }[] {
  const globals = fs.readFileSync(path.join(root, 'app/globals.css'), 'utf8');
  return [{ file: 'app/globals.css', css: globals }, ...localStyleSources(root, ['app', 'components', 'founderos', 'lib'])];
}

/** Every face rule in any sheet that still paints the old look and that no pack rule clears. */
export function cssFindings(root: string, carries: Record<string, string[]> = {}): CssFinding[] {
  const all = sheets(root);
  const globals = all[0].css;
  const cut = globals.indexOf(PACK_MARK);
  const head = cssRules('app/globals.css', globals.slice(0, cut));
  const pack = cssRules('app/globals.css', globals.slice(cut), head.length).map((r) => ({ ...r, line: r.line + globals.slice(0, cut).split('\n').length - 1 }));
  const clear = clearers(pack);
  const rest = all.slice(1).flatMap((s, i) => cssRules(s.file, s.css, (i + 1) * 1e6));
  const out: CssFinding[] = [];
  for (const r of [...head, ...rest]) {
    for (const { face, selector: sel } of r.selectors.flatMap(faceSelectors)) {
      for (const { issue, detail } of oldLook(r.body)) {
        const props = issue === 'fill' ? PROP_OF.fill.filter((p) => decl(r.body, p) && !NO_FILL.test(decl(r.body, p)!.value)) : PROP_OF[issue];
        if (props.every((p) => cleared(r, sel, p, clear, carries))) continue;
        out.push({ key: `${r.file} ${sel}`, file: r.file, line: r.line, face, selector: sel, issue, detail });
      }
    }
  }
  return out;
}

/* ── TSX ─────────────────────────────────────────────────────────────────────────────────────────── */

/** Surface utilities: the os surfaces, white or black washes, card and surface tokens, gradients. Base or breakpoint only. */
const TSX_FILL =
  /^(?:(?:sm|md|lg|xl|2xl|wide|max-[\w-]+|min-[\w-]+):)*(?:bg-os-surface\d?(?:\/\d+)?|bg-os-bg2(?:\/\d+)?|bg-white\/\d+|bg-black\/\d+|bg-\[(?:var\(--(?:surface|g-card|g-well|l-surface|elevated|glass)[^\]]*|(?:linear|radial|conic)-gradient[^\]]*)\]|bg-gradient-to-\w+)$/;
/** A radius makes a box a shape (rounded-none does not). */
const TSX_RADIUS = /^(?:(?:sm|md|lg|xl|2xl|wide):)*rounded(?:-(?!none)[\w[\]().%-]+)?$/;
/** Inputs, media and SVG parts are not faces. */
const NOT_A_BOX = /^(?:input|textarea|select|option|img|video|svg|path|rect|circle|line|g|text|canvas|iframe)$/;

export type TsxFinding = { file: string; line: number; tag: string; fill: string[]; opening: string };

const MAX_VARIANTS = 64;

/**
 * The class lists a className can render, one per branch: `a ? 'x' : 'y'` gives both arms, `c && 'x'` gives
 * with and without, and a template multiplies its spans. Calls (cx(), gc()) contribute every literal they hold.
 */
export function classVariants(open: ts.JsxOpeningElement | ts.JsxSelfClosingElement): string[][] {
  const a = open.attributes.properties.find((p): p is ts.JsxAttribute => ts.isJsxAttribute(p) && p.name.getText() === 'className');
  if (!a?.initializer) return [[]];
  const words = (t: string) => t.split(/\s+/).filter(Boolean);
  const cross = (xs: string[][], ys: string[][]) => xs.flatMap((x) => ys.map((y) => [...x, ...y])).slice(0, MAX_VARIANTS);
  const of = (x: ts.Node): string[][] => {
    if (ts.isJsxExpression(x)) return x.expression ? of(x.expression) : [[]];
    if (ts.isParenthesizedExpression(x)) return of(x.expression);
    if (ts.isStringLiteral(x) || ts.isNoSubstitutionTemplateLiteral(x)) return [words(x.text)];
    if (ts.isTemplateExpression(x)) {
      let acc: string[][] = [words(x.head.text)];
      for (const sp of x.templateSpans) acc = cross(cross(acc, of(sp.expression)), [words(sp.literal.text)]);
      return acc;
    }
    if (ts.isConditionalExpression(x)) return [...of(x.whenTrue), ...of(x.whenFalse)].slice(0, MAX_VARIANTS);
    if (ts.isBinaryExpression(x) && x.operatorToken.kind === ts.SyntaxKind.AmpersandAmpersandToken) return [[], ...of(x.right)];
    if (ts.isBinaryExpression(x) && (x.operatorToken.kind === ts.SyntaxKind.BarBarToken || x.operatorToken.kind === ts.SyntaxKind.QuestionQuestionToken)) return [...of(x.left), ...of(x.right)];
    if (ts.isBinaryExpression(x) && x.operatorToken.kind === ts.SyntaxKind.PlusToken) return cross(of(x.left), of(x.right));
    if (ts.isCallExpression(x)) return x.arguments.reduce<string[][]>((acc, arg) => cross(acc, of(arg)), [[]]);
    // anything else (an identifier, a property): its literals, if any, in one variant
    const lits: string[] = [];
    const visit = (y: ts.Node) => {
      if (ts.isStringLiteral(y) || ts.isNoSubstitutionTemplateLiteral(y)) lits.push(...words(y.text));
      ts.forEachChild(y, visit);
    };
    visit(x);
    return [lits];
  };
  return of(a.initializer);
}

/** Every JSX element with a radius that paints a surface utility and is not a face of the pack. */
export function tsxFindings(file: string, src: string): TsxFinding[] {
  const sf = ts.createSourceFile(file, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
  const out: TsxFinding[] = [];
  const visit = (x: ts.Node) => {
    if (ts.isJsxElement(x) || ts.isJsxSelfClosingElement(x)) {
      const open = ts.isJsxElement(x) ? x.openingElement : x;
      const tag = open.tagName.getText();
      // each branch of the className on its own: an "on" arm that is glass does not excuse a filled "off" arm
      const bad = classVariants(open).filter((tokens) => {
        if (NOT_A_BOX.test(tag) || !tokens.some((t) => TSX_FILL.test(t)) || !tokens.some((t) => TSX_RADIUS.test(t))) return false;
        return !(faceOf(tokens) || tokens.some((t) => /^glass-(?:well|solid|face)$/.test(t)));
      });
      if (bad.length) {
        const fill = [...new Set(bad.flatMap((t) => t.filter((u) => TSX_FILL.test(u))))];
        out.push({ file, line: sf.getLineAndCharacterOfPosition(open.getStart()).line + 1, tag, fill, opening: open.getText().replace(/\s+/g, ' ') });
      }
    }
    ts.forEachChild(x, visit);
  };
  visit(sf);
  return out;
}

/** Class lists of every element that carries `cls`, in TSX and in HTML built as strings (class="…"). */
export function elementsWith(root: string, files: string[], cls: string): string[][] {
  const out: string[][] = [];
  for (const f of files) {
    const src = fs.readFileSync(path.join(root, f), 'utf8');
    if (!src.includes(cls)) continue;
    if (f.endsWith('.tsx')) {
      const sf = ts.createSourceFile(f, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
      const visit = (x: ts.Node) => {
        if (ts.isJsxElement(x) || ts.isJsxSelfClosingElement(x)) {
          const t = classTokens(x);
          const comp = FACE_COMPONENTS[(ts.isJsxElement(x) ? x.openingElement : x).tagName.getText()];
          if (t.includes(cls)) out.push(comp ? [comp, ...t] : t);
        }
        ts.forEachChild(x, visit);
      };
      visit(sf);
    }
    for (const m of src.matchAll(/class="([^"]*)"/g)) {
      const t = m[1].split(/\s+/).filter(Boolean);
      if (t.includes(cls)) out.push(t);
    }
  }
  return out;
}

/** The pack class an element's tokens carry: a face class, the mapped Classic card, or a glass-* class. */
export function packClassOf(tokens: readonly string[]): string | undefined {
  return faceOf(tokens) ?? tokens.find((t) => /^glass-(?:well|solid|face)$/.test(t));
}

/**
 * Face classes whose every element (TSX or HTML string) carries a pack class: the pack class(es) each comes with.
 * A class with no element found carries nothing (its rule is judged on its own).
 */
export function carriedPackClasses(root: string, files: string[], classes: Iterable<string>): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const c of classes) {
    const els = elementsWith(root, files, c);
    if (!els.length) continue;
    const packs = els.map(packClassOf);
    if (packs.every(Boolean)) out[c] = [...new Set(packs as string[])];
  }
  return out;
}
