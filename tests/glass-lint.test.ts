/**
 * The glass lint (FounderOS Glass brief §3.2, §4, §5 B11).
 *
 * 1. A glass face never sits inside an element with filter, opacity < 1, mask or backdrop-filter:
 *    each makes that element a backdrop root, the face's frost stops there and the face reads
 *    as see-through. Scrims are siblings of their faces, never parents.
 * 2. No face enters with an opacity fade: the keyframes a face plays move transform and the
 *    light only.
 * 3. A scrim frosts the page from light to strong over 10 frames and dims it 20%; the face sits
 *    above it as its own strong-frost element.
 *
 * The scan is static (tests/helpers/glass-lint.ts): TSX parsed with the TypeScript compiler,
 * the frost killers read from Tailwind utilities, style objects and app/globals.css.
 */
import fs from 'node:fs';
import path from 'node:path';
import { createElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, test } from 'vitest';
import { LDialog } from '@/components/legacy/Dialog';
import { CSS_NO_COMMENTS, GLOBALS_CSS, splitSelectors, topLevelRules } from './helpers/theme-css';
import {
  FACE_CLASSES,
  allRules,
  cssKillers,
  faceElementClasses,
  faceKeyframes,
  keyframeBodies,
  killerUtility,
  lintSource,
  tsxFiles,
} from './helpers/glass-lint';

const ROOT = path.join(__dirname, '..');
const SCAN_DIRS = ['app', 'components', 'founderos'];
const FILES = tsxFiles(ROOT, SCAN_DIRS).map((abs) => ({ file: path.relative(ROOT, abs), src: fs.readFileSync(abs, 'utf8') }));
const KILLERS = cssKillers();
const FINDINGS = FILES.flatMap(({ file, src }) => lintSource(file, src, KILLERS));
const keyOf = (f: { file: string; face: string; why: string }) => `${f.file}: ${f.face} in ${f.why}`;

/**
 * PENDING_NESTING_OUTSIDE_PHASE_2A: faces that still sit inside a frost killer. Phase 3 moved every
 * ad-hoc modal's scrim beside its face and dimmed leaves instead of parents, so the list is empty
 * and must stay empty: any new violation fails.
 */
const PENDING_NESTING_OUTSIDE_PHASE_2A: string[] = [];

/**
 * PENDING_FACE_FADE_KEYFRAMES_PHASE_2B: face entrance keyframes that still animate opacity.
 * Phase 2b is converting the remaining fade-ins (panel-in, fin-tip-in, ...) to Materialise;
 * it empties this list as it goes. Stale names fail too, so the list only shrinks.
 */
const PENDING_FACE_FADE_KEYFRAMES_PHASE_2B: string[] = [];

/** Files phase 2a owns: none of them may sit on a pending list. */
const OWNED = ['components/legacy/Dialog.tsx', 'components/CommandPalette.tsx', 'components/g/Overlays.tsx', 'components/ui/Toast.tsx'];

describe('the lint itself', () => {
  const lint = (src: string) => lintSource('fixture.tsx', src, cssKillers('.fade-wrap { animation: fx 1s; } @keyframes fx { from { opacity: 0; } } .dim.is-off { opacity: .5; }'));

  test('a face inside a blurring, fading, filtered or masked wrapper is reported', () => {
    expect(lint(`<div className="fixed inset-0 backdrop-blur-lg"><div className="c-pop" /></div>`)).toHaveLength(1);
    expect(lint(`<div className="opacity-60"><span><i className="g-chip" /></span></div>`)).toHaveLength(1);
    expect(lint(`<div className="hover:drop-shadow-lg"><GCard /></div>`)).toHaveLength(1);
    expect(lint(`<div className="[mask-image:linear-gradient(red,blue)]"><Glass /></div>`)).toHaveLength(1);
    expect(lint(`<div style={{ backdropFilter: 'blur(3px)' }}><div className="l-card" /></div>`)).toHaveLength(1);
    expect(lint(`<div style={{ opacity: 0.4 }}><div className="g-well" /></div>`)).toHaveLength(1);
    expect(lint(`<section className="fade-wrap"><div className="glass-face" /></section>`)[0].why).toMatch(/animation fx/);
    expect(lint(`<div className={cx('dim', off && 'is-off')}><div className="glass-face" /></div>`)).toHaveLength(1);
  });

  test('a scrim beside its face, a leaf that fades, or a face inside a face is fine', () => {
    expect(lint(`<div className="layer"><div className="gl-scrim backdrop-blur" onClick={close} /><div className="c-pop" /></div>`)).toEqual([]);
    expect(lint(`<div className="glass-face"><img className="opacity-50" /></div>`)).toEqual([]);
    expect(lint(`<div className="opacity-100 placeholder:opacity-50 blur-none"><div className="g-card" /></div>`)).toEqual([]);
    expect(lint(`<div className="dim"><div className="glass-face" /></div>`)).toEqual([]);
    expect(lint(`<div className="l-slab"><div className="l-card"><span className="l-chip" /></div></div>`)).toEqual([]);
    expect(lint(`<div style={{ opacity: 1, filter: 'none' }}><GChip /></div>`)).toEqual([]);
  });

  test('glass over glass: the Classic card (os surface + card radius) is a face, so what it holds is not inside a killer', () => {
    const card = ':root[data-ground] .bg-os-surface.rounded-lg-t:not(.border-dashed) { backdrop-filter: var(--gl-blur); }';
    expect(cssKillers(card), 'the mapped card is a face, not a frost killer').toEqual([]);
    const k = cssKillers(card);
    expect(lintSource('f.tsx', `<div className="rounded-lg-t border bg-os-surface p-6"><span className="c-chip" /><span className="c-pop glass-solid" /></div>`, k)).toEqual([]);
    // the mapped card is itself checked like any face; a dashed drop zone is not a card
    expect(lint(`<div className="fixed inset-0 backdrop-blur-sm"><div className="rounded-lg-t bg-os-surface" /></div>`)).toHaveLength(1);
    expect(lint(`<div className="fixed inset-0 backdrop-blur-sm"><div className="rounded-lg-t border-dashed bg-os-surface" /></div>`)).toEqual([]);
    expect(lint(`<div className="fixed inset-0 backdrop-blur-sm"><div className="rounded-lg-t bg-os-surface2" /></div>`)).toEqual([]);
  });

  test('glass over glass stays strict: a face that frosts itself again inside a face is reported', () => {
    const again = lint(`<div className="g-card"><span className="glass-chip backdrop-blur-md" /></div>`);
    expect(again).toHaveLength(1);
    expect(again[0].why).toMatch(/frosts again inside <div className="g-card">/);
    expect(lint(`<div className="l-slab"><div style={{ backdropFilter: 'blur(4px)' }} className="l-card" /></div>`)).toHaveLength(1);
    // a face's own frost is fine where nothing frosts under it
    expect(lint(`<div><span className="glass-chip backdrop-blur-md" /></div>`)).toEqual([]);
  });

  test('utilities: only the ones that set the property on the element itself', () => {
    expect(killerUtility('backdrop-blur-lg')).toBe('backdrop-filter');
    expect(killerUtility('md:blur-sm')).toBe('filter');
    expect(killerUtility('opacity-0')).toBe('opacity');
    expect(killerUtility('opacity-100')).toBeNull();
    expect(killerUtility('before:opacity-50')).toBeNull();
    expect(killerUtility('bg-os-bg/60')).toBeNull();
    expect(killerUtility('backdrop-blur-none')).toBeNull();
  });

  test('the face list is the glass pack face rule', () => {
    const face = topLevelRules().find((r) => r.selectors.some((s) => /^:root\[data-ground\] :is\(/.test(s) && /\.glass-face,/.test(s)) && /box-shadow/.test(r.body));
    expect(face, 'the glass pack face rule').toBeTruthy();
    const listed = [...face!.selectors[0].matchAll(/\.([\w-]+)/g)].map((m) => m[1]);
    expect([...FACE_CLASSES].filter((c) => c !== 'glass-face--strong').sort()).toEqual([...listed].sort());
  });
});

describe('no glass face inside a filter, opacity < 1, mask or backdrop-filter (brief §3.2)', () => {
  test('every violation in the repo is on the pending list, and the list has no stale lines', () => {
    const found = [...new Set(FINDINGS.map(keyOf))].sort();
    const unexpected = found.filter((k) => !PENDING_NESTING_OUTSIDE_PHASE_2A.includes(k));
    expect(unexpected, FINDINGS.filter((f) => unexpected.includes(keyOf(f))).map((f) => `${f.file}:${f.line} ${f.face} inside ${f.ancestor}: ${f.why}`).join('\n')).toEqual([]);
    expect(PENDING_NESTING_OUTSIDE_PHASE_2A.filter((k) => !found.includes(k)), 'stale pending lines: delete them').toEqual([]);
  });

  test('the overlays phase 2a owns are clean (LDialog, both palettes, sheets, drawers, dialogs, toasts)', () => {
    for (const file of OWNED) {
      expect(FINDINGS.filter((f) => f.file === file), file).toEqual([]);
      expect(PENDING_NESTING_OUTSIDE_PHASE_2A.filter((k) => k.startsWith(`${file}:`)), file).toEqual([]);
    }
    expect(FINDINGS.filter((f) => f.file.startsWith('founderos/'))).toEqual([]);
  });

  test('the overlay layers carry no frost killer of their own; only their scrims frost', () => {
    const killerClasses = new Set(KILLERS.flatMap((k) => (k.classes.length === 1 ? k.classes : [])));
    for (const layer of ['l-layer', 'g-layer', 'g-palette-layer']) expect(killerClasses.has(layer), layer).toBe(false);
    for (const scrim of ['gl-scrim', 'g-scrim', 'l-scrim']) expect(killerClasses.has(scrim), scrim).toBe(true);
  });
});

describe('glass over glass (brief §3.2): a face inside a frosted face is lit, never frosted again', () => {
  const P3 = (() => {
    const s = GLOBALS_CSS.indexOf('/* ── glass phase 3 ── */');
    const e = GLOBALS_CSS.indexOf('/* ── end glass phase 3 ── */');
    return s > -1 && e > s ? GLOBALS_CSS.slice(s, e).replace(/\/\*[\s\S]*?\*\//g, '') : '';
  })();
  const norm = (s: string) => s.replace(/\s+/g, ' ').trim();
  /** `:root[data-ground] :is(OUTER) :is(INNER)` -> [outer members, inner members]. */
  const nested = (sel: string): [string[], string[]] | null => {
    const m = /^:root\[data-ground\] :is\(([^)]*(?:\([^)]*\)[^)]*)*)\) :is\(([^)]*(?:\([^)]*\)[^)]*)*)\)$/.exec(norm(sel));
    return m ? [m[1].split(',').map((x) => x.trim()), m[2].split(',').map((x) => x.trim())] : null;
  };
  const rules = topLevelRules(P3).filter((r) => r.selectors.some((s) => nested(s)));
  const face = topLevelRules().find((r) => r.selectors.some((s) => /^:root\[data-ground\] :is\(\s*\.glass-face,/.test(s)) && /box-shadow/.test(r.body))!;
  const shadowOf = (body: string) => {
    const v = /box-shadow:([^;]+);/.exec(body);
    return v ? splitSelectors(norm(v[1])).map(norm) : [];
  };
  const six = shadowOf(face.body);
  const FROSTING = [...FACE_CLASSES.map((c) => `.${c}`), '.bg-os-surface.rounded-lg-t:not(.border-dashed)'];

  test('the phase 3 block sits at the end of globals.css with no literal colour', () => {
    expect(P3, 'glass phase 3 block').not.toBe('');
    expect(P3).not.toMatch(/#[0-9a-fA-F]{3,8}\b|rgba?\(\s*\d/);
  });

  test('every face nested in any frosting face drops its own frost (the lint exemption rests on this rule)', () => {
    const unfrost = rules.find((r) => /(^|;)\s*backdrop-filter:\s*none/.test(r.body) && /-webkit-backdrop-filter:\s*none/.test(r.body) && !/box-shadow/.test(r.body));
    expect(unfrost, 'the glass-over-glass rule').toBeTruthy();
    const [outer, inner] = nested(unfrost!.selectors[0])!;
    expect([...FROSTING].filter((f) => !outer.includes(f)), 'outer faces missing').toEqual([]);
    expect([...FROSTING].filter((f) => !inner.includes(f.replace(':not(.border-dashed)', '')) && !inner.includes(f)), 'inner faces missing').toEqual([]);
  });

  test('a chip inside a frosted face is lit like a well: rim, bottom edge and pool, no groove, drop or shade', () => {
    const chips = rules.find((r) => {
      const n = nested(r.selectors[0]);
      return n && ['.c-chip', '.g-chip', '.l-chip', '.glass-chip'].every((c) => n[1].includes(c)) && /box-shadow/.test(r.body);
    });
    expect(chips, 'the nested chip rule').toBeTruthy();
    expect(nested(chips!.selectors[0])![0]).toEqual(nested(rules[0].selectors[0])![0]);
    expect(shadowOf(chips!.body)).toEqual([six[2], six[3], six[4]]);
    expect(chips!.body).not.toMatch(/background/);
  });

  test('a pop-up inside a frosted face is a solid raised face (brief: small pop-ups over content never frost)', () => {
    const pops = rules.find((r) => {
      const n = nested(r.selectors[0]);
      return n && ['.c-pop', '.glass-face--strong'].every((c) => n[1].includes(c)) && !n[1].includes('.l-card');
    });
    expect(pops, 'the nested pop-up rule').toBeTruthy();
    expect(norm(pops!.body)).toMatch(/(^|; )background: var\(--solid-bg\)/);
    expect(norm(pops!.body)).toMatch(/(^|; )backdrop-filter: none/);
    expect(shadowOf(pops!.body)).toEqual(six);
  });

  test('the three Classic cards that hold a chip or a pop-up are clean', () => {
    for (const f of ['components/BusinessIncomeChart.tsx', 'components/IntegrationCategory.tsx', 'components/PostComposer.tsx']) {
      expect(FINDINGS.filter((x) => x.file === f), f).toEqual([]);
    }
  });
});

describe('scrims are siblings: the face sits above its scrim, not in it', () => {
  test('LDialog: a layer holding the scrim, then the strong-frost face with its dialog roles', () => {
    const html = renderToStaticMarkup(createElement(LDialog, { open: true, onClose: () => {}, title: 'Rename', sub: 'one line', children: 'body', footer: 'foot' }));
    const m = /^<div data-l="DialogLayer" class="l-layer">(<div[^>]*><\/div>)(<div[^>]*>)/.exec(html);
    expect(m, html.slice(0, 300)).toBeTruthy();
    const [, scrim, face] = m!;
    expect(scrim).toMatch(/class="l-scrim gl-scrim"/);
    expect(scrim).toMatch(/aria-hidden="true"/);
    expect(scrim).toMatch(/data-l="DialogScrim"/);
    expect(face).toMatch(/role="dialog"/);
    expect(face).toMatch(/aria-modal="true"/);
    expect(face).toMatch(/class="l-dialog glass-face--strong"/);
    const labelled = /aria-labelledby="([^"]+)"/.exec(face)?.[1];
    expect(labelled).toBeTruthy();
    expect(html).toContain(`<h2 id="${labelled}" class="l-dialog-title">Rename</h2>`);
    expect(html).toContain('aria-label="Close"');
  });

  test('LDialog closes on the scrim, never on a press inside the face', () => {
    const src = fs.readFileSync(path.join(ROOT, 'components/legacy/Dialog.tsx'), 'utf8');
    expect(src).toMatch(/className="l-scrim gl-scrim"[^>]*onClick=\{onClose\}/);
    expect(src).not.toMatch(/stopPropagation/);
  });

  test('both palettes: a scrim that closes, then the face (role dialog, labelled)', () => {
    const src = fs.readFileSync(path.join(ROOT, 'components/CommandPalette.tsx'), 'utf8');
    expect(src).toMatch(/className="g-palette-layer"[^>]*>\s*<div className="gl-scrim" aria-hidden onClick=\{close\} \/>\s*<div/);
    expect(src.match(/<div className="gl-scrim" aria-hidden onClick=\{close\} \/>/g)?.length).toBe(2);
    expect(src).toMatch(/className=\{cx\('g-palette glass-face--strong'/);
    expect(src).toMatch(/className=\{cx\('c-pop glass-face--strong relative /);
    expect(src.match(/role="dialog"/g)?.length).toBe(2);
    expect(src.match(/aria-modal="true"/g)?.length).toBe(2);
    expect(src.match(/aria-label="Command palette"/g)?.length).toBe(2);
    expect(src).not.toMatch(/stopPropagation/);
    expect(src).not.toMatch(/backdrop-blur/);
  });

  test('sheets, drawers and dialogs: the scrim frosts as a sibling too', () => {
    const src = fs.readFileSync(path.join(ROOT, 'components/g/Overlays.tsx'), 'utf8');
    expect(src.match(/<div className="g-scrim gl-scrim" aria-hidden/g)?.length).toBe(3);
  });
});

describe('ad-hoc modals: the scrim beside the face, the face materialises and melts (phase 3)', () => {
  /** Modals that used to blur their own wrapper (or a scrim around the face). */
  const MODALS = [
    'components/AgentPanel.tsx',
    'components/AudienceConsistency.tsx',
    'components/SocialStatStrip.tsx',
    'founderos/components/SocialStatStrip.tsx',
    'components/content/Composer.tsx',
    'components/content/MediaDetail.tsx',
    'components/content/MediaLibrary.tsx',
    'components/content/PipelineBoard.tsx',
    'components/content/PipelineBuilder.tsx',
    'components/trading/TradingClassic.tsx',
  ];

  test.each(MODALS)('%s: a gl-scrim closes it, nothing frosts around the face, the face melts', (file) => {
    const src = fs.readFileSync(path.join(ROOT, file), 'utf8');
    expect(src).toMatch(/<div className="gl-scrim" aria-hidden onClick=\{[\w.]+\} \/>/);
    // the scrim is the only closer: the layer takes no click, the face stops nothing
    for (const m of src.matchAll(/<div className="gl-scrim"/g)) {
      const before = src.slice(0, m.index);
      expect(before.slice(before.lastIndexOf('<div')), 'the layer holding the scrim').not.toMatch(/onClick/);
    }
    expect(src).not.toMatch(/glass-face--strong[^<]*?stopPropagation/);
    expect(src).not.toMatch(/backdropFilter|backdrop-blur/);
    expect(src).toMatch(/useMeltClose\(|useMeltPhase\(/);
    expect(src).toMatch(/glass-face--strong/);
    expect(src).toMatch(/gl-materialise/);
    expect(src).toMatch(/is-melting/);
    expect(FINDINGS.filter((f) => f.file === file)).toEqual([]);
  });

  test('useMeltClose: close starts the melt; the parent unmounts the face when it has melted', () => {
    const src = fs.readFileSync(path.join(ROOT, 'lib/use-melt.ts'), 'utf8');
    expect(src).toMatch(/export function useMeltClose/);
    expect(src).toMatch(/useMeltEnd\(faceRef, melting, onClose\)/);
  });
});

describe('the scrim: light to strong frost over 10 frames, 20% dim (brief §5 B11)', () => {
  const start = GLOBALS_CSS.indexOf('/* ── glass phase 2a ── */');
  const end = GLOBALS_CSS.indexOf('/* ── end glass phase 2a ── */');
  const block = start > -1 && end > start ? GLOBALS_CSS.slice(start, end).replace(/\/\*[\s\S]*?\*\//g, '') : '';

  test('the phase 2a CSS sits in one delimited block', () => {
    expect(start, 'phase 2a block').toBeGreaterThan(-1);
    expect(end).toBeGreaterThan(start);
  });
  const norm = (s: string) => s.replace(/\s+/g, ' ').trim();
  const rule = (pred: (sel: string) => boolean, src = block) => topLevelRules(src).filter((r) => r.selectors.some(pred));

  test('dim token: the page dims 20% on every ground; 10 frames is 417ms; no literal colour (the block sits after /chats)', () => {
    expect(norm(block)).toMatch(/:root \{[^}]*--gl-scrim: color-mix\(in srgb, black 20%, transparent\);/);
    expect(norm(block)).toMatch(/--gl-dur-scrim: 417ms;/);
    expect(block).not.toMatch(/#[0-9a-fA-F]{3,8}\b|rgba?\(\s*\d/);
  });

  test('the scrim rests at strong frost and the dim, and arrives from the light frost', () => {
    const r = rule((s) => s === ':root[data-ground] .gl-scrim');
    expect(r).toHaveLength(1);
    const body = norm(r[0].body);
    expect(body).toMatch(/background: var\(--gl-scrim\)/);
    expect(body).toMatch(/(^|; )backdrop-filter: var\(--gl-blur-strong\)/);
    expect(body).toMatch(/animation: gl-scrim-in var\(--gl-dur-scrim\) linear backwards/);
    const k = norm(keyframeBodies().get('gl-scrim-in') ?? '');
    expect(k).toMatch(/from \{[^}]*background-color: transparent/);
    expect(k).toMatch(/from \{[^}]*(^|; |\{ )backdrop-filter: var\(--gl-blur\)/);
    expect(k).not.toMatch(/opacity/);
  });

  test('when its face melts the scrim clears with it, and stops taking clicks', () => {
    const r = rule((s) => /\[data-melting\] > \.gl-scrim$/.test(s));
    expect(r).toHaveLength(1);
    expect(norm(r[0].body)).toMatch(/animation: gl-scrim-out var\(--gl-dur-melt\) linear forwards/);
    expect(norm(r[0].body)).toMatch(/pointer-events: none/);
    const k = norm(keyframeBodies().get('gl-scrim-out') ?? '');
    expect(k).toMatch(/to \{[^}]*background-color: transparent/);
    expect(k).toMatch(/to \{[^}]*backdrop-filter: blur\(0px\) saturate\(100%\)/);
    expect(k).not.toMatch(/opacity/);
  });

  test('reduced motion: the scrim is simply there; reduced transparency: dim only, no blur', () => {
    const motion = [...block.matchAll(/@media \(prefers-reduced-motion: reduce\) \{([\s\S]*?\})\s*\}/g)].map((m) => norm(m[1])).join(' ');
    expect(motion).toMatch(/\.gl-scrim[^{]*\{[^}]*animation: none/);
    const clear = [...block.matchAll(/@media \(prefers-reduced-transparency: reduce\) \{([\s\S]*?\})\s*\}/g)].map((m) => norm(m[1])).join(' ');
    expect(clear).toMatch(/\.gl-scrim[^{]*\{[^}]*backdrop-filter: none/);
  });

  test('scrims cover their layer; faces stack above them', () => {
    const geo = allRules(block).filter((r) => r.selectors.includes('.gl-scrim'));
    expect(geo.some((r) => /position: absolute/.test(norm(r.body)) && /inset: 0/.test(norm(r.body))), '.gl-scrim geometry needs no ground').toBe(true);
    for (const face of ['.l-dialog', '.g-palette']) {
      expect(rule((s) => s.split(/\s*,\s*/).includes(face) || new RegExp(`(^|[ ,])\\${face}($|[ ,])`).test(s)).some((r) => /position: relative/.test(norm(r.body))), face).toBe(true);
    }
  });
});

describe('no face enters with an opacity fade (brief §4)', () => {
  const kf = keyframeBodies();
  const fades = (name: string) => /(^|[;{\s])opacity\s*:/.test(kf.get(name) ?? '');
  const played = faceKeyframes(FILES.flatMap(({ file, src }) => faceElementClasses(file, src)));

  test('the scan finds the face keyframes (the materialise pair and the overlays)', () => {
    for (const name of ['gl-materialise', 'gl-materialise-drawer', 'g-sheet-in', 'ui-toast-in']) expect(played.has(name), name).toBe(true);
  });

  test('every face keyframe that still fades is on the pending list, and the list has no stale names', () => {
    const fading = [...played.keys()].filter(fades).sort();
    expect(fading.filter((n) => !PENDING_FACE_FADE_KEYFRAMES_PHASE_2B.includes(n)), 'new face fades').toEqual([]);
    expect(PENDING_FACE_FADE_KEYFRAMES_PHASE_2B.filter((n) => !fading.includes(n)), 'stale pending names: delete them').toEqual([]);
  });

  test('the pending list is only ever entrances phase 2b owns', () => {
    for (const n of PENDING_FACE_FADE_KEYFRAMES_PHASE_2B) expect(CSS_NO_COMMENTS).toContain(`@keyframes ${n}`);
  });
});
