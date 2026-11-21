import fs from 'node:fs';
import path from 'node:path';
import ts from 'typescript';
import { CSS_NO_COMMENTS, splitSelectors } from './theme-css';
import { allRules, classTokens, faceOf, killerUtility, tsxFiles } from './glass-lint';

/**
 * Fixed under frost (glass canon, amendment "never frost page-tall containers"). `backdrop-filter`
 * makes an element the containing block and the stacking context of its `position: fixed`
 * descendants: a modal, drawer or popout layer inside a frosted element is sized and centred on that
 * element instead of the viewport, clipped by it, stacked under whatever sits above it, and turned
 * solid by the glass-over-glass rule. So a fixed element never renders under a frosted one unless it
 * is portalled to <body> (createPortal, or the kit's Layer and the overlays built on it).
 *
 * The scan is static and follows components across files:
 * - TSX is parsed with the TypeScript compiler. An element is frosted when it is a glass face
 *   (faceOf), carries a Tailwind backdrop utility, sets backdropFilter in a literal style, or names a
 *   class whose rule in app/globals.css sets a backdrop-filter (any style, any media query). It is
 *   fixed when it carries Tailwind `fixed`, sets position: 'fixed' in a literal style, or names a
 *   class whose rule sets position: fixed.
 * - A component's `children` slot is frosted when `{children}` renders under a frosted element (so
 *   `<SlabCard>` or `<LegacySlab>` frost what they wrap) and portalled when it renders inside
 *   createPortal (so `<Layer>`, `<GSheet>`, `<GDrawer>`, `<GDialog>` release what they wrap).
 * - A fixed element with nothing frosted above it in its own file marks its component; every place
 *   that renders that component is checked the same way, up to the pages, and a page is mounted in
 *   the `children` slot of every layout and template above it (app/layout.tsx puts it in LegacySlab).
 */

export const ROOT = path.join(__dirname, '..', '..');
export const SCAN_DIRS = ['app', 'components', 'founderos', 'lib'];

type El = ts.JsxElement | ts.JsxSelfClosingElement;

export type FixedFinding = {
  /** Where the fixed element is. */
  file: string;
  line: number;
  fixed: string;
  /** The frosted element it renders under. */
  under: string;
  /** The component chain from the fixed element up to where the frost is (empty when it is in the same file). */
  via: string[];
};

/** The last compound of a selector (after the last combinator outside brackets and parens). */
function lastCompound(sel: string): string {
  let depth = 0;
  let cut = 0;
  for (let i = 0; i < sel.length; i++) {
    const c = sel[i];
    if (c === '(' || c === '[') depth++;
    else if (c === ')' || c === ']') depth--;
    else if (depth === 0 && /[\s>+~]/.test(c)) cut = i + 1;
  }
  return sel.slice(cut).trim();
}

/** The class lists a compound can match: its own classes, times each member of an :is()/:where() in it. */
function compoundAlternatives(compound: string): string[][] {
  const groups: string[] = [];
  let rest = '';
  for (let i = 0; i < compound.length; i++) {
    const m = /^:(is|where|not|has|nth-[\w-]+|host|global)\(/.exec(compound.slice(i));
    if (!m) {
      rest += compound[i];
      continue;
    }
    let depth = 0;
    let j = i + m[0].length - 1;
    for (; j < compound.length; j++) {
      if (compound[j] === '(') depth++;
      else if (compound[j] === ')' && --depth === 0) break;
    }
    if (m[1] === 'is' || m[1] === 'where') groups.push(compound.slice(i + m[0].length, j));
    i = j;
  }
  const own = [...rest.matchAll(/\.([\w-]+)/g)].map((x) => x[1]);
  if (!groups.length) return own.length ? [own] : [];
  let alts: string[][] = [own];
  for (const g of groups) {
    const members = splitSelectors(g).flatMap((s) => compoundAlternatives(lastCompound(s)));
    alts = alts.flatMap((a) => members.map((mm) => [...a, ...mm]));
  }
  return alts.filter((a) => a.length);
}

/** The subject class lists of every rule (media and supports blocks included) whose body passes `test`. */
export function cssSubjects(test: (body: string) => boolean, css: string = CSS_NO_COMMENTS): string[][] {
  const out: string[][] = [];
  const seen = new Set<string>();
  for (const r of allRules(css)) {
    if (!test(r.body)) continue;
    for (const sel of r.selectors) {
      if (/::?(?:before|after|backdrop|placeholder|marker|selection|-webkit-[\w-]+)\b/.test(sel)) continue;
      for (const alt of compoundAlternatives(lastCompound(sel))) {
        const key = [...new Set(alt)].sort().join('.');
        if (seen.has(key)) continue;
        seen.add(key);
        out.push(key.split('.'));
      }
    }
  }
  return out;
}

export const frostsBody = (b: string) => /(?:^|[;\s{])(?:-webkit-)?backdrop-filter\s*:\s*(?!none\b|initial\b|unset\b)[^;\s]/.test(b);
export const fixedBody = (b: string) => /(?:^|[;\s{])position\s*:\s*fixed\b/.test(b);

function tagName(n: El): string {
  return (ts.isJsxElement(n) ? n.openingElement : n).tagName.getText();
}

function styleProp(n: El, keys: string[]): string | null {
  const open = ts.isJsxElement(n) ? n.openingElement : n;
  const a = open.attributes.properties.find((p): p is ts.JsxAttribute => ts.isJsxAttribute(p) && p.name.getText() === 'style');
  const e = a?.initializer && ts.isJsxExpression(a.initializer) ? a.initializer.expression : undefined;
  if (!e || !ts.isObjectLiteralExpression(e)) return null;
  for (const p of e.properties) {
    if (!ts.isPropertyAssignment(p)) continue;
    const key = p.name.getText().replace(/^['"]|['"]$/g, '');
    if (!keys.includes(key)) continue;
    return ts.isStringLiteral(p.initializer) || ts.isNoSubstitutionTemplateLiteral(p.initializer) ? p.initializer.text : p.initializer.getText();
  }
  return null;
}

const utilityBase = (t: string) => t.replace(/^(?:[\w-]+:)+/, '').replace(/^!/, '');
const PSEUDO = /^(?:placeholder|before|after|file|marker|selection|backdrop):/;

type Key = string; // `${relFile}#${componentName}`
type Slot = { frost: string | null; portal: boolean };
type Usage = { target: Key; file: string; line: number; frost: string | null; portalled: boolean; owner: Key | null };
type Loose = { file: string; line: number; fixed: string };

type Mod = {
  rel: string;
  sf: ts.SourceFile;
  /** Local name -> component key it stands for (local definitions, imports, dynamic() aliases). */
  names: Map<string, Key>;
  defaultKey: Key | null;
};

/** Every fixed element that is not inside a portal, wherever it renders (file, line, label). */
export type Unportalled = { file: string; line: number; fixed: string; tokens: string[]; owner: string | null };
export type FixedScan = { findings: FixedFinding[]; slots: Map<Key, Slot>; unportalled: Unportalled[] };

export function scanFixedUnderFrost(opts: { root?: string; dirs?: string[]; css?: string; files?: { file: string; src: string }[] } = {}): FixedScan {
  const root = opts.root ?? ROOT;
  const css = opts.css ?? CSS_NO_COMMENTS;
  const frostCss = cssSubjects(frostsBody, css);
  const fixedCss = cssSubjects(fixedBody, css);
  const files =
    opts.files ??
    [...tsxFiles(root, opts.dirs ?? SCAN_DIRS), ...barrelFiles(root, opts.dirs ?? SCAN_DIRS)].map((abs) => ({ file: path.relative(root, abs), src: fs.readFileSync(abs, 'utf8') }));
  const known = new Set(files.map((f) => f.file));

  const resolve = (from: string, spec: string): string | null => {
    let base: string;
    if (spec.startsWith('@/')) base = spec.slice(2);
    else if (spec.startsWith('.')) base = path.join(path.dirname(from), spec);
    else return null;
    for (const ext of ['.tsx', '/index.tsx', '.ts', '/index.ts', '']) if (known.has(base + ext)) return base + ext;
    return null;
  };

  // 1. Modules: their components, imports and default export.
  const mods = new Map<string, Mod>();
  const reexports = new Map<Key, Key>();
  const stars = new Map<string, string[]>();
  const defs = new Set<Key>();
  for (const { file, src } of files) {
    const sf = ts.createSourceFile(file, src, ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    const names = new Map<string, Key>();
    let defaultKey: Key | null = null;
    const local = (n: string) => `${file}#${n}`;
    for (const st of sf.statements) {
      const isDefault = ts.canHaveModifiers(st) && (ts.getModifiers(st) ?? []).some((m) => m.kind === ts.SyntaxKind.DefaultKeyword);
      if (ts.isFunctionDeclaration(st)) {
        const n = st.name?.text ?? '(default)';
        names.set(n, local(n));
        defs.add(local(n));
        if (isDefault) defaultKey = local(n);
      } else if (ts.isVariableStatement(st)) {
        for (const d of st.declarationList.declarations) {
          if (!ts.isIdentifier(d.name) || !/^[A-Z]/.test(d.name.text)) continue;
          names.set(d.name.text, local(d.name.text));
          defs.add(local(d.name.text));
          // next/dynamic: const X = dynamic(() => import('…').then((m) => m.Y))
          const text = d.initializer?.getText() ?? '';
          const imp = /\bimport\(\s*['"]([^'"]+)['"]\s*\)/.exec(text);
          if (imp && /\bdynamic\(/.test(text)) {
            const target = resolve(file, imp[1]);
            const named = /\.then\(\s*\(?\s*(\w+)\s*\)?\s*=>\s*\(?\s*\1\.(\w+)/.exec(text);
            if (target) reexports.set(local(d.name.text), named ? `${target}#${named[2]}` : `${target}#default`);
          }
        }
      } else if (ts.isExportAssignment(st) && ts.isIdentifier(st.expression)) {
        defaultKey = local(st.expression.text);
      } else if (ts.isExportDeclaration(st) && st.moduleSpecifier && ts.isStringLiteral(st.moduleSpecifier) && st.exportClause && ts.isNamedExports(st.exportClause)) {
        const target = resolve(file, st.moduleSpecifier.text);
        if (target) for (const e of st.exportClause.elements) reexports.set(local(e.name.text), `${target}#${(e.propertyName ?? e.name).text}`);
      } else if (ts.isExportDeclaration(st) && st.moduleSpecifier && ts.isStringLiteral(st.moduleSpecifier) && !st.exportClause) {
        const target = resolve(file, st.moduleSpecifier.text);
        if (target) stars.set(file, [...(stars.get(file) ?? []), target]);
      }
    }
    for (const st of sf.statements) {
      if (!ts.isImportDeclaration(st) || !ts.isStringLiteral(st.moduleSpecifier) || !st.importClause) continue;
      const target = resolve(file, st.moduleSpecifier.text);
      if (!target) continue;
      const c = st.importClause;
      if (c.name) names.set(c.name.text, `${target}#default`);
      if (c.namedBindings && ts.isNamedImports(c.namedBindings))
        for (const e of c.namedBindings.elements) names.set(e.name.text, `${target}#${(e.propertyName ?? e.name).text}`);
    }
    mods.set(file, { rel: file, sf, names, defaultKey });
  }

  /** Follow default exports, re-exports and dynamic() aliases to the defining component. */
  const canon = (k: Key): Key => {
    for (let i = 0; i < 10; i++) {
      const [f, n] = k.split('#');
      if (n === 'default') {
        const d = mods.get(f)?.defaultKey;
        if (!d) return k;
        k = d;
        continue;
      }
      const r = reexports.get(k) ?? (defs.has(k) ? undefined : (stars.get(f) ?? []).map((t) => `${t}#${n}`).find((t) => defs.has(t) || reexports.has(t)));
      if (!r) return k;
      k = r;
    }
    return k;
  };

  /** The component a node is defined in (nearest capitalised function or const). */
  const ownerOf = (file: string, x: ts.Node): Key | null => {
    for (let p: ts.Node | undefined = x.parent; p; p = p.parent) {
      if (ts.isFunctionDeclaration(p) && p.parent && ts.isSourceFile(p.parent)) return canon(`${file}#${p.name?.text ?? '(default)'}`);
      if (ts.isVariableDeclaration(p) && ts.isIdentifier(p.name) && /^[A-Z]/.test(p.name.text) && p.parent?.parent?.parent && ts.isSourceFile(p.parent.parent.parent))
        return canon(`${file}#${p.name.text}`);
      if (ts.isExportAssignment(p)) return mods.get(file)?.defaultKey ?? null;
    }
    return null;
  };

  const frostOf = (n: El, tokens: string[]): string | null => {
    const face = faceOf(tokens);
    if (face) return face;
    const util = tokens.find((t) => killerUtility(t) === 'backdrop-filter');
    if (util) return util;
    const have = new Set(tokens);
    const c = frostCss.find((cls) => cls.every((k) => have.has(k)));
    if (c) return `.${c.join('.')}`;
    const s = styleProp(n, ['backdropFilter', 'WebkitBackdropFilter']);
    if (s && !/^\s*none\s*$/.test(s)) return `style.backdropFilter`;
    return null;
  };
  const isFixed = (n: El, tokens: string[]): boolean => {
    if (tokens.some((t) => !PSEUDO.test(t) && utilityBase(t) === 'fixed')) return true;
    const have = new Set(tokens);
    if (fixedCss.some((cls) => cls.every((k) => have.has(k)))) return true;
    return /^\s*fixed\s*$/.test(styleProp(n, ['position']) ?? '');
  };
  const label = (n: El, tokens: string[]) => `<${tagName(n)}${tokens.length ? ` className="${tokens.join(' ').slice(0, 70)}"` : ''}>`;
  const isChildrenRef = (e: ts.Node | undefined) => !!e && /^(?:props\.)?children$/.test(e.getText().trim());

  // 2. Walk every module: slots (iterated to a fixpoint, wrappers of wrappers), then usages and fixed elements.
  const slots = new Map<Key, Slot>();
  const walkAll = (collect: boolean) => {
    const usages: Usage[] = [];
    const loose = new Map<Key, Loose>();
    const unportalled: Unportalled[] = [];
    const direct: FixedFinding[] = [];
    const nextSlots = new Map<Key, Slot>();
    const noteSlot = (owner: Key | null, frost: string | null, portalled: boolean) => {
      if (!owner) return;
      const s = nextSlots.get(owner) ?? { frost: null, portal: true };
      nextSlots.set(owner, { frost: s.frost ?? frost, portal: s.portal && portalled });
    };
    for (const m of mods.values()) {
      // `const overlay = (<div …/>); return createPortal(overlay, …)`: the JSX held by a portalled name is portalled.
      const portalledNames = new Set<string>();
      const findPortals = (x: ts.Node) => {
        if (ts.isCallExpression(x) && /(?:^|\.)createPortal$/.test(x.expression.getText()) && x.arguments[0] && ts.isIdentifier(x.arguments[0])) portalledNames.add(x.arguments[0].text);
        ts.forEachChild(x, findPortals);
      };
      findPortals(m.sf);
      const walk = (x: ts.Node, frost: string | null, portalled: boolean) => {
        if (ts.isVariableDeclaration(x) && ts.isIdentifier(x.name) && portalledNames.has(x.name.text) && x.initializer) {
          walk(x.initializer, null, true);
          return;
        }
        if (ts.isCallExpression(x) && /(?:^|\.)createPortal$/.test(x.expression.getText())) {
          for (const a of x.arguments) {
            if (isChildrenRef(a)) noteSlot(ownerOf(m.rel, a), null, true);
            walk(a, null, true);
          }
          return;
        }
        if (ts.isJsxExpression(x) && isChildrenRef(x.expression)) noteSlot(ownerOf(m.rel, x), frost, portalled);
        if (ts.isJsxElement(x) || ts.isJsxSelfClosingElement(x)) {
          const tag = tagName(x);
          const tokens = classTokens(x);
          const line = m.sf.getLineAndCharacterOfPosition(x.getStart()).line + 1;
          if (collect && isFixed(x, tokens) && !portalled) {
            unportalled.push({ file: m.rel, line, fixed: label(x, tokens), tokens, owner: ownerOf(m.rel, x) });
            if (frost) direct.push({ file: m.rel, line, fixed: label(x, tokens), under: frost, via: [] });
            else {
              const owner = ownerOf(m.rel, x);
              if (owner && !loose.has(owner)) loose.set(owner, { file: m.rel, line, fixed: label(x, tokens) });
            }
          }
          let childFrost = frostOf(x, tokens) ? label(x, tokens) : frost;
          let childPortalled = portalled;
          if (/^[A-Z]/.test(tag) && !tag.includes('.')) {
            const target = m.names.get(tag) ? canon(m.names.get(tag)!) : null;
            if (target) {
              if (collect) usages.push({ target, file: m.rel, line, frost, portalled, owner: ownerOf(m.rel, x) });
              const s = slots.get(target);
              if (s?.portal) {
                childFrost = null;
                childPortalled = true;
              } else if (s?.frost && !childFrost) childFrost = `<${tag}> (wraps its children in ${s.frost})`;
            }
          }
          ts.forEachChild(x, (c) => walk(c, childFrost, childPortalled));
          return;
        }
        ts.forEachChild(x, (c) => walk(c, frost, portalled));
      };
      walk(m.sf, null, false);
    }
    return { usages, loose, direct, nextSlots, unportalled };
  };
  for (let i = 0; i < 6; i++) {
    const { nextSlots } = walkAll(false);
    const same = nextSlots.size === slots.size && [...nextSlots].every(([k, v]) => slots.get(k)?.frost === v.frost && slots.get(k)?.portal === v.portal);
    slots.clear();
    for (const [k, v] of nextSlots) slots.set(k, v);
    if (same) break;
  }
  const { usages, loose, direct, unportalled } = walkAll(true);

  // 3. Pages render in the children slot of every layout and template above them.
  for (const m of mods.values()) {
    if (!/^app\/(?:.*\/)?page\.tsx$/.test(m.rel) || !m.defaultKey) continue;
    const parts = path.dirname(m.rel).split('/');
    for (let i = 1; i <= parts.length; i++) {
      for (const shell of ['layout.tsx', 'template.tsx']) {
        const sm = mods.get([...parts.slice(0, i), shell].join('/'));
        if (sm?.defaultKey) usages.push({ target: canon(m.defaultKey), file: sm.rel, line: 0, frost: slots.get(canon(sm.defaultKey))?.frost ?? null, portalled: false, owner: null });
      }
    }
  }

  // 4. Up the component tree from every loose fixed element until something frosted wraps it.
  const findings = [...direct];
  const queue: { key: Key; origin: Loose; via: string[] }[] = [...loose].map(([key, origin]) => ({ key, origin, via: [key.split('#')[1]] }));
  const seen = new Set<string>();
  while (queue.length) {
    const { key, origin, via } = queue.shift()!;
    for (const u of usages) {
      if (u.target !== key || u.portalled) continue;
      const at = `${u.file}:${u.line}`;
      if (u.frost) {
        const id = `${origin.file}:${origin.line}<${at}`;
        if (!seen.has(id)) {
          seen.add(id);
          findings.push({ file: origin.file, line: origin.line, fixed: origin.fixed, under: `${u.frost} at ${at}`, via });
        }
      } else if (u.owner && !seen.has(`${origin.file}:${origin.line}>${u.owner}`)) {
        seen.add(`${origin.file}:${origin.line}>${u.owner}`);
        queue.push({ key: u.owner, origin, via: [...via, u.owner.split('#')[1]] });
      }
    }
  }
  // A fixed element whose component is only ever rendered inside a portal is portalled too (GSheet's panel).
  const portalledOwner = (k: string | null) => {
    if (!k) return false;
    const us = usages.filter((u) => u.target === k);
    return us.length > 0 && us.every((u) => u.portalled);
  };
  return { findings, slots, unportalled: unportalled.filter((u) => !portalledOwner(u.owner)) };
}

/** Non-JSX modules that re-export components (barrels such as components/legacy/index.ts). */
function barrelFiles(root: string, dirs: string[]): string[] {
  const out: string[] = [];
  const visit = (d: string) => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) visit(p);
      else if (e.name.endsWith('.ts') && !e.name.endsWith('.d.ts') && /^\s*export\s+(?:\*|\{[^}]*\})\s+from\b/m.test(fs.readFileSync(p, 'utf8'))) out.push(p);
    }
  };
  for (const d of dirs) visit(path.join(root, d));
  return out.sort();
}

export const fixedKey = (f: FixedFinding) => `${f.file}:${f.line} ${f.fixed}`;
