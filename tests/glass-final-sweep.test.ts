/**
 * Glass phase 5, the final sweep (FounderOS Glass brief §1, §2, §5B; glass canon). No face in the app is still on
 * the old look: a fill (an opaque or semi-opaque colour, a surface or card token, a gradient), a top-edge highlight,
 * a coloured "on" fill or a radius off the pack's three (card 24, well 16, chip a pill).
 *
 * The sweep reads every stylesheet the app ships (app/globals.css, the component sheets, <style> blocks) and every
 * TSX element in app/, components/, founderos/ and lib/ (tests/helpers/glass-faces.ts says what a face is and how
 * the cascade is judged). A face that paints the old look fails unless the glass pack outranks it, or it is on the
 * allowlists below: elements that are not faces, each with its reason. A future filled card fails here.
 */
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import ts from 'typescript';
import { describe, expect, test } from 'vitest';
import { groundFor } from '@/lib/appearance';
import {
  carriedPackClasses,
  classVariants,
  clearers,
  cleared,
  cssFindings,
  cssRules,
  elementsWith,
  faceClassOf,
  oldLook,
  PACK_MARK,
  radiusOk,
  sheets,
  specificity,
  tsxFindings,
  type CssFinding,
  type TsxFinding,
} from './helpers/glass-faces';
import { tsxFiles } from './helpers/glass-lint';
import { evalColour, over, ratio, rootCascade, substitute, type RGBA } from './helpers/lab-colour';
import { GLOBALS_CSS, splitSelectors } from './helpers/theme-css';

const ROOT = process.cwd();
const read = (p: string) => readFileSync(join(ROOT, p), 'utf8');
const norm = (s: string) => s.replace(/\s+/g, ' ').trim();
const walk = (d: string): string[] =>
  readdirSync(join(ROOT, d), { withFileTypes: true }).flatMap((e) => (e.isDirectory() ? walk(`${d}/${e.name}`) : /\.tsx?$/.test(e.name) ? [`${d}/${e.name}`] : []));
const SOURCES = ['app', 'components', 'founderos', 'lib'].flatMap(walk);

/** Face classes always rendered with a pack class (their own rules lose to the pack's where it outranks them). */
const CARRIES = (() => {
  const faces = new Set<string>();
  for (const s of sheets(ROOT)) for (const r of cssRules(s.file, s.css)) for (const sel of r.selectors) {
    const f = faceClassOf(sel);
    if (f) faces.add(f);
  }
  return carriedPackClasses(ROOT, SOURCES, faces);
})();

const P5 = (() => {
  const s = GLOBALS_CSS.indexOf('/* ── glass phase 5 ── */');
  const e = GLOBALS_CSS.indexOf('/* ── end glass phase 5 ── */');
  return s > -1 && e > s ? GLOBALS_CSS.slice(s, e) : '';
})();
/** Every rule of the pack (the GLASS PACK block and the phase blocks after it): phase 5 joined some of its lists. */
const P5_RULES = cssRules('app/globals.css', GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(PACK_MARK)));
/** The declarations the last pack rule naming `sel` (as one of its selectors) gives it, outside any media query. */
const p5 = (sel: string, prop: string, media?: RegExp) => {
  const rs = P5_RULES.filter((r) => r.selectors.map(norm).includes(sel) && (media ? r.at.some((a) => media.test(a)) : r.at.length === 0));
  const vals = rs.map((r) => new RegExp(`(?:^|[;{\\s])${prop.replace(/-/g, '\\-')}\\s*:\\s*([^;]+)`).exec(r.body)?.[1]).filter(Boolean) as string[];
  return vals.length ? norm(vals[vals.length - 1]) : undefined;
};
/** The pack's six layers, as the GLASS PACK face rule writes them. */
const SIX = (() => {
  const m = /:root\[data-ground\] :is\(\s*\.glass-face,[\s\S]*?\{([^}]*)\}/.exec(GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(PACK_MARK)));
  return norm(/box-shadow:([^;]+);/.exec(m![1])![1]);
})();
const LIT = norm(
  'inset 0 0 0 1px color-mix(in srgb, var(--gl-tint) var(--gl-rim-a), transparent), inset 0 -1px 0 color-mix(in srgb, var(--gl-tint) var(--gl-low-a), transparent)',
);

describe('the phase 5 block', () => {
  test('sits once, last in globals.css, after phase 3, with no literal colour', () => {
    expect(GLOBALS_CSS.split('/* ── glass phase 5 ── */').length).toBe(2);
    expect(P5).not.toBe('');
    expect(GLOBALS_CSS.indexOf('/* ── glass phase 5 ── */')).toBeGreaterThan(GLOBALS_CSS.indexOf('/* ── end glass phase 3 ── */'));
    const END = '/* ── end glass phase 5 ── */';
    expect(GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(END) + END.length).trim()).toBe('');
    const bare = P5.replace(/\/\*[\s\S]*?\*\//g, '');
    expect(bare).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(bare).not.toMatch(/rgba?\(/);
  });
});

describe('task 1: the faces that were still filled', () => {
  const CLEAR = clearers(cssRules('app/globals.css', GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(PACK_MARK))));
  const own = (file: string, sel: string) => {
    const src = file === 'app/globals.css' ? GLOBALS_CSS.slice(0, GLOBALS_CSS.indexOf(PACK_MARK)) : read(file);
    const r = cssRules(file, src, file === 'app/globals.css' ? 0 : 1e6).find((x) => x.selectors.map(norm).includes(sel) && !x.at.length);
    expect(r, `${file} ${sel}`).toBeTruthy();
    return r!;
  };

  test('.bh-ask (the Blueprint ask bar) is the dock (brief §5 B14): a glass face, its inner layer frosted and lit, the canon ring', () => {
    for (const sel of [':root[data-ground] .bh-ask', ":root[data-ground][data-ui='glados'] .kg-blueprint .bh-ask"]) {
      expect(p5(sel, 'background'), sel).toBe('transparent');
      expect(p5(sel, 'border-color'), sel).toBe('transparent');
      expect(p5(sel, 'backdrop-filter'), sel).toBe('var(--gl-blur)');
      expect(p5(sel, 'box-shadow'), sel).toBe(SIX);
      expect(p5(sel, 'border-radius'), sel).toBe('var(--gl-r-card)');
    }
    // the inner layer carries the strong frost and the inner light over the orbit, in GladOS too (the kit hid it)
    expect(p5(':root[data-ground] .bh-ask-glass', 'backdrop-filter')).toBe('var(--gl-blur-strong)');
    expect(p5(':root[data-ground] .bh-ask-glass', 'background')).toBe('transparent');
    expect(p5(':root[data-ground] .bh-ask-glass', 'box-shadow')).toMatch(/^inset 0 0 0 1px .*inset 0 12px 18px -14px var\(--gl-shade\)$/);
    expect(p5(":root[data-ground][data-ui='glados'] .kg-blueprint .bh-ask-orbit", 'display')).toBe('block');
    // the old fill, border and drop lose to the pack in both styles
    for (const [file, sel] of [['app/globals.css', '.bh-ask'], ['app/globals.css', ":root[data-ui='glados'] .kg-blueprint .bh-ask"]] as const) {
      for (const prop of ['background', 'box-shadow', 'border-radius']) expect(cleared(own(file, sel), sel, prop, CLEAR), `${sel} ${prop}`).toBe(true);
    }
    expect(cleared(own('app/globals.css', '.bh-ask-glass'), '.bh-ask-glass', 'background', CLEAR)).toBe(true);
    // focus: the accent ring 2px out; focus and a scope strengthen the rim, never a coloured border
    expect(p5(':root[data-ground] .bh-ask:has(:focus-visible)', 'outline')).toBe('2px solid var(--gl-focus)');
    expect(p5(':root[data-ground] .bh-ask:has(:focus-visible)', 'outline-offset')).toBe('2px');
    // it is the pack's one ring rule (tests/glass-review.test.ts), the ask bars' :has() beside the controls
    expect(P5_RULES.filter((r) => /outline: 2px solid var\(--gl-focus\)/.test(r.body))).toHaveLength(1);
    expect(P5).toMatch(/\.bh-ask:focus-within \.bh-ask-glass,\s*:root\[data-ground\] \.bh-ask\.has-scope \.bh-ask-glass \{\s*--gl-rim-a: 45%;/);
    // no blur where none is wanted: the solid face
    const rt = P5_RULES.filter((r) => r.at.some((a) => /prefers-reduced-transparency: reduce/.test(a))).flatMap((r) => r.selectors.map(norm));
    expect(rt).toEqual(expect.arrayContaining([':root[data-ground] .bh-ask', ":root[data-ground][data-ui='glados'] .kg-blueprint .bh-ask", ':root[data-ground] .bh-ask-glass']));
  });

  test('the brand-deals hero strip sits on the hero glow where no ink reads through a clear face, so it is the solid raised face', async () => {
    const sel = ':root[data-ground] .bd-g-volume .g-hero-strip';
    expect(p5(sel, 'background')).toBe('var(--solid-bg)');
    expect(p5(sel, 'backdrop-filter')).toBe('none');
    expect(p5(sel, 'box-shadow')).toBe(SIX);
    expect(p5(sel, 'border-radius')).toBe('var(--gl-r-well)');
    expect(p5(sel, 'color')).toBe('var(--g-ink)');
    expect(cleared(own('app/globals.css', '.bd-g-volume .g-hero-strip'), '.bd-g-volume .g-hero-strip', 'background', CLEAR)).toBe(true);
    // why not clear: over the strip's zone of the hero (its lower third) neither ink reaches 4.5:1 in any accent or mode
    const { gCascade, gRules, resolve, tokenColour, worstContrast } = await import('./helpers/g-colour');
    const R = gRules();
    for (const a of ['ice', 'lime', 'ember', 'mint']) for (const m of ['dark', 'light'] as const) {
      const e = gCascade(a, m, R);
      const bg = resolve(e, 'var(--g-hero-a)');
      const zone = { x: [0.06, 0.94], y: [0.62, 0.96] } as const;
      const best = Math.max(worstContrast(bg, tokenColour(e, '--g-on-hero'), zone), worstContrast(bg, tokenColour(e, '--g-ink'), zone));
      expect(best, `${a} ${m}`).toBeLessThan(4.5);
    }
  });

  test('the knowledge stat tiles are glass faces: transparent, light frost, the six layers, the card radius', () => {
    const sel = ':root[data-ground] .kg-know-stats .g-stat';
    expect(p5(sel, 'background')).toBe('transparent');
    expect(p5(sel, 'backdrop-filter')).toBe('var(--gl-blur)');
    expect(p5(sel, 'box-shadow')).toBe(SIX);
    expect(p5(sel, 'border-radius')).toBe('var(--gl-r-card)');
    expect(cleared(own('app/globals.css', '.kg-know-stats .g-stat'), '.kg-know-stats .g-stat', 'background', CLEAR)).toBe(true);
  });

  test('.lib-card is a glass face in both canvas skins; its gradient, card fill and hover drop lose to the pack', () => {
    for (const sel of [':root[data-ground] .wfc .lib-card', ":root[data-ground][data-ui='glados'] .wfc[data-skin='g'] .lib-card"]) {
      expect(p5(sel, 'background'), sel).toBe('transparent');
      expect(p5(sel, 'box-shadow'), sel).toBe(SIX);
      expect(p5(sel, 'border-radius'), sel).toBe('var(--gl-r-card)');
    }
    expect(specificity(":root[data-ground][data-ui='glados'] .wfc[data-skin='g'] .lib-card")).toEqual(specificity(":root[data-ui='glados'] .wfc[data-skin='g'] .lib-card:hover"));
    expect(specificity(':root[data-ground] .wfc .lib-card')[1]).toBeGreaterThan(specificity('.wfc .lib-card:hover')[1]);
    expect(cleared(own('components/workflow-canvas/canvas.css', '.wfc .lib-card'), '.wfc .lib-card', 'background', CLEAR)).toBe(true);
    expect(cleared(own('app/globals.css', ":root[data-ui='glados'] .wfc[data-skin='g'] .lib-card"), ":root[data-ui='glados'] .wfc[data-skin='g'] .lib-card", 'background', CLEAR)).toBe(true);
  });

  describe('.table: the container is the face; rows and cells are unfilled; the open row is "on" as a tinted light', () => {
    test('the executions and tests tables are glass faces in both canvas skins', () => {
      for (const sel of [':root[data-ground] .wfc .table', ":root[data-ground][data-ui='glados'] .wfc[data-skin='g'] .table"]) {
        expect(p5(sel, 'background'), sel).toBe('transparent');
        expect(p5(sel, 'backdrop-filter'), sel).toBe('var(--gl-blur)');
        expect(p5(sel, 'box-shadow'), sel).toBe(SIX);
        expect(p5(sel, 'border-radius'), sel).toBe('var(--gl-r-card)');
      }
      expect(read('components/WorkflowCanvas.tsx').match(/className="table"/g)).toHaveLength(2);
    });

    test('rows and the head row paint no fill, hover or not; hairlines separate them', () => {
      for (const sel of [':root[data-ground] .wfc .row', ':root[data-ground] .wfc .row.head', ':root[data-ground] .wfc .row:not(.head):hover']) {
        expect(p5(sel, 'background'), sel).toBe('transparent');
      }
      // the head row's shade and the hover surface are out-specified
      expect(specificity(':root[data-ground] .wfc .row.head')[1]).toBeGreaterThan(specificity('.wfc .row.head')[1]);
      expect(specificity(':root[data-ground] .wfc .row:not(.head):hover')[1]).toBeGreaterThan(specificity('.wfc .row:not(.head):hover')[1]);
      expect(read('components/workflow-canvas/canvas.css')).toMatch(/\.wfc \.row \{[^}]*border-bottom: 1px solid var\(--hairline\)/);
      expect(p5(':root[data-ground] .wfc .row:not(.head):hover', 'box-shadow')).toMatch(/^inset 0 -18px 26px -18px color-mix/);
    });

    test('selection: the open row is "on", a tinted rim and pool in the "on" colour, never a fill; the engine marks it', () => {
      const sel = ':root[data-ground] .wfc .row.is-on';
      expect(p5(sel, 'background')).toBe('transparent');
      const light = p5(sel, 'box-shadow')!;
      // every layer is light in the "on" colour: a rim, a bottom edge, a pool; no fill anywhere
      expect(splitSelectors(light)).toEqual([
        'inset 0 0 0 1px color-mix(in srgb, var(--gl-on) 45%, transparent)',
        'inset 0 -1px 0 color-mix(in srgb, var(--gl-on) 60%, transparent)',
        expect.stringMatching(/^inset 0 -18px 26px -18px color-mix\(in srgb, var\(--gl-on\) \d+%, transparent\)$/),
      ]);
      const engine = read('components/workflow-canvas/engine.ts');
      expect(engine).toMatch(/r\.classList\.toggle\('is-on', !!tr\?\.classList\.contains\('is-open'\)\)/);
      expect(engine).toMatch(/r\.setAttribute\('aria-expanded'/);
    });

    test('the text of a selected row reads at least 4.5:1 over its light in every style and theme', () => {
      const colour = (env: Map<string, string>, token: string): RGBA => evalColour(substitute(`var(${token})`, (t) => env.get(t)));
      const looks = [
        ...(['ice', 'lime', 'ember', 'mint'] as const).flatMap((accent) =>
          (['dark', 'light'] as const).map((mode) => ({
            name: `glados ${accent} ${mode}`,
            attrs: { 'data-ui': 'glados', 'data-accent': accent, 'data-mode': mode, 'data-theme': mode === 'light' ? 'light' : 'console', 'data-ground': groundFor('console', mode, 'glados') },
            ground: '--g-bg',
            inks: ['--g-ink', '--g-muted'],
          })),
        ),
        ...(['console', 'red', 'light', 'frost', 'grey-frost'] as const).map((theme) => {
          const mode = theme === 'light' ? 'light' : 'dark';
          return { name: `classic ${theme}`, attrs: { 'data-ui': 'classic', 'data-accent': 'ice', 'data-mode': mode, 'data-theme': theme, 'data-ground': groundFor(theme, mode, 'classic') }, ground: '--bg', inks: ['--text', '--text-2'] };
        }),
        ...(['monolith', 'daylight'] as const).map((theme) => {
          const mode = theme === 'daylight' ? 'light' : 'dark';
          return { name: `legacy ${theme}`, attrs: { 'data-ui': 'legacy', 'data-accent': 'ice', 'data-mode': mode, 'data-theme': theme, 'data-ground': groundFor(theme, mode, 'legacy') }, ground: '--bg', inks: ['--text', '--text-2'] };
        }),
      ];
      for (const look of looks) {
        const env = rootCascade(look.attrs);
        const ground = colour(env, look.ground);
        const on = colour(env, '--gl-on');
        // the pool's peak under the text (its percentage read from the rule) over the ground the frost shows; the rim
        // and the bottom edge are 1px lines at the row's edges, never under the text. Hover's white pool too.
        const pool = Number(/var\(--gl-on\) (\d+)%, transparent\)$/.exec(p5(':root[data-ground] .wfc .row.is-on', 'box-shadow')!)![1]) / 100;
        const hover = Number(/(\d+)%, transparent\)$/.exec(p5(':root[data-ground] .wfc .row:not(.head):hover', 'box-shadow')!)![1]) / 100;
        const lit = over([on[0], on[1], on[2], pool], ground);
        const hovered = over([255, 255, 255, hover], ground);
        for (const ink of look.inks) {
          expect(ratio(colour(env, ink), lit), `${look.name} ${ink} selected`).toBeGreaterThanOrEqual(4.5);
          expect(ratio(colour(env, ink), hovered), `${look.name} ${ink} hovered`).toBeGreaterThanOrEqual(4.5);
        }
      }
    });
  });
});

/* ── The sweep ─────────────────────────────────────────────────────────────────────────────────────────────────── */

type Why = 'input' | 'button' | 'scrim' | 'page background' | 'chart mark' | 'avatar' | 'progress track' | 'skeleton' | 'gradient tile' | 'text' | 'row' | 'bar' | 'retired';

/**
 * CSS rules that paint a fill on a face-named class but are not faces: [sheet, selector, why, reason].
 * 'retired' classes are rendered by no element (the test proves it), so reviving one makes it a face again.
 */
const CSS_NOT_FACES: [string, string, Why, string][] = [
  ['app/globals.css', '.ae-o-chip', 'chart mark', 'the Octane usage meter: each segment grows to its count (a data mark, not a chip)'],
  ['app/globals.css', '.ae-o-chip.pink', 'chart mark', 'a usage meter segment coloured by its state'],
  ['app/globals.css', '.ae-o-chip.green', 'chart mark', 'a usage meter segment coloured by its state'],
  ['app/globals.css', ".ae-o-chips[data-cap='warn'] .ae-o-chip.green", 'chart mark', 'the meter near its cap'],
  ['app/globals.css', ".ae-o-chips[data-cap='cap'] .ae-o-chip.green", 'chart mark', 'the meter at its cap'],
  ['app/globals.css', ".ae-o-chips[data-cap='cap'] .ae-o-chip.pink", 'chart mark', 'the meter at its cap'],
  ['app/globals.css', 'select.ae-o-pill.center', 'input', 'the campaign picker, a <select> drawn as the centre pill (its caret is the background image)'],
  ['app/globals.css', '.route-skel-card', 'skeleton', 'the route skeleton placeholder: it pulses while the page loads'],
  ['app/globals.css', ":root[data-ui='glados'] .route-skel-card", 'skeleton', 'the route skeleton placeholder under GladOS'],
  ['app/globals.css', ":root[data-ui='legacy'] .l-frame", 'page background', 'the Legacy page slab (the page column; never a frosted face, glass canon amendment)'],
  ['app/lab/lab-pages.css', '.labv-pill-solid', 'button', "the Lab's primary action button (an action, not a chip)"],
  ['app/globals.css', '.ib-stat', 'retired', 'no element renders it'],
  ['app/globals.css', '.lab .lab-field-hud', 'retired', 'no element renders it'],
  ['app/globals.css', '.lab-c-panel', 'retired', 'no element renders it'],
  ['app/globals.css', '.lab-c-tip', 'retired', 'no element renders it'],
  ['app/globals.css', '.lab-c-sheet', 'retired', 'no element renders it'],
  ['app/lab/lab-pages.css', '.labv-banner', 'retired', 'no element renders it'],
];

/** TSX elements with a radius and a surface utility that are not faces: [file, a needle in the opening tag, count, why, reason]. */
const TSX_NOT_FACES: [string, string, number, Why, string][] = [
  ['app/analytics/page.tsx', 'mt-4 flex items-center justify-center gap-2.5 rounded-sm-t', 1, 'button', 'the Connect link, drawn as a full-width button'],
  ['app/analytics/page.tsx', 'flex h-8 w-8 items-center justify-center rounded-sm-t bg-os-surface2', 1, 'avatar', 'an icon badge (a glyph), lit on hover by its group'],
  ['app/analytics/page.tsx', 'mt-4 h-1 overflow-hidden rounded-sm-t bg-os-surface2', 1, 'progress track', 'a 4px progress track'],
  ['app/finances/FinancesClient.tsx', 'h-1.5 overflow-hidden rounded-sm-t bg-os-surface2', 1, 'progress track', 'a budget progress track'],
  ['app/finances/FinancesClient.tsx', 'mt-3 h-1 overflow-hidden rounded-sm-t bg-os-surface2', 1, 'progress track', 'a progress track'],
  ['app/social/page.tsx', 'mt-3 h-1 overflow-hidden rounded-sm-t bg-os-surface2', 2, 'progress track', 'platform share tracks'],
  ['components/AgentAvatar.tsx', 'data-agent-avatar=""', 1, 'avatar', 'the agent avatar'],
  ['components/AgentChat.tsx', 'onClick={send}', 1, 'button', 'the send button'],
  ['components/AgentPanel.tsx', 'aria-label="Close"', 1, 'button', 'a round close button'],
  ['components/AgentWorkPanel.tsx', 'onClick={addTask}', 1, 'button', 'the add-task button'],
  ['components/AgentWorkPanel.tsx', 'onClick={addCron}', 1, 'button', 'the add-schedule button'],
  ['components/BrainCore.tsx', 'sticky top-0 flex items-center justify-between rounded-t-lg-t', 1, 'bar', "the pop-out panel's sticky title bar: it covers the list scrolling under it (a bar of its panel, not a face)"],
  ['components/BrainCore.tsx', 'href="/roadmap"', 1, 'button', 'a link drawn as a button'],
  ['components/BrainDump.tsx', 'absolute right-0 top-0 flex h-6 w-6', 1, 'button', 'the dictation toggle'],
  ['components/BrainDump.tsx', 'title="Choose documents to upload"', 1, 'button', 'the upload button'],
  ['components/BrainDump.tsx', 'absolute right-2 top-2 flex h-8 w-8', 1, 'button', 'the dictation toggle'],
  ['components/BrainQuery.tsx', 'border-b-2 bg-os-surface', 1, 'text', 'a keyboard hint (<kbd>)'],
  ['components/ConductorChat.tsx', 'onClick={send}', 1, 'button', 'the send button'],
  ['components/ConductorPanel.tsx', 'wiz-launch group fixed', 1, 'button', 'the Conductor launcher'],
  ['components/ConductorPanel.tsx', 'aria-label="Widen the panel"', 1, 'button', 'a round icon button'],
  ['components/ConductorPanel.tsx', 'aria-label="Slide the panel away"', 1, 'button', 'a round icon button'],
  ['components/ConnectFlow.tsx', 'onClick={() => setOpen(true)}', 1, 'button', 'the connect button'],
  ['components/InstagramDmInbox.tsx', 'inline-flex gap-1 rounded-md-t', 1, 'input', 'the tab track of a segmented control'],
  ['components/KnowledgeDetail.tsx', 'g-btn kg-detail-row', 1, 'button', 'a row button'],
  ['components/KnowledgeGraphFullscreen.tsx', 'g-btn g-btn-sm absolute right-5 top-5', 1, 'button', 'the exit button'],
  ['components/PostComposer.tsx', 'flex flex-1 items-center gap-3 rounded-sm-t', 1, 'input', 'a field (a label around its input)'],
  ['components/PostComposer.tsx', 'flex items-center gap-3 rounded-sm-t', 1, 'input', 'a field (a label around its input)'],
  ['components/SetupFilming.tsx', 'onClick={toggle}', 1, 'button', 'the filming toggle button'],
  ['components/SetupManualCheck.tsx', 'onClick={toggle}', 1, 'button', 'the mark-done button'],
  ['components/Sidebar.tsx', 'os-side-toggle', 1, 'button', 'the sidebar toggle'],
  ['components/SkillsExplorer.tsx', 'focus-within:border-[var(--accent-line)]', 1, 'input', 'the search field'],
  ['components/SkillsGrid.tsx', '<code', 1, 'text', 'inline code in a sentence (text formatting)'],
  ['components/StatementUploader.tsx', 'border-dashed', 1, 'input', 'the statement drop zone (a dashed file target)'],
  ['components/Topbar.tsx', 'onClick={openPalette}', 1, 'button', 'the palette button'],
  ['components/Topbar.tsx', 'CONDUCTOR_TOGGLE_EVENT', 1, 'button', 'the wizard button'],
  ['components/brand-deals/DealBoard.tsx', 'rotate-[24deg]', 1, 'gradient tile', 'the insight card shard: a decorative leaf on the one sanctioned gradient card'],
  ['components/brand-deals/DealBoard.tsx', 'bg-white/15 px-3 py-1', 1, 'gradient tile', 'the insight card badge: ink on the sanctioned gradient card'],
  ['components/compute/ComputeConsole.tsx', 'w-full rounded-[10px] border p-3 text-left', 1, 'button', 'a selectable option button'],
  ['components/content/Composer.tsx', 'absolute right-0.5 top-0.5 grid h-4 w-4', 1, 'button', 'the remove-media button over a thumbnail'],
  ['components/content/Composer.tsx', 'role="switch"', 1, 'input', 'a switch (an on/off input)'],
  ['components/content/CreateGrid.tsx', 'grid h-9 w-9 place-items-center rounded-md-t', 1, 'avatar', 'an icon badge (a glyph)'],
  ['components/content/MediaDetail.tsx', 'aria-label="Close"', 1, 'button', 'a round close button'],
  ['components/content/MediaLibrary.tsx', 'border-dashed', 1, 'input', 'the upload drop slot (a dashed file target)'],
  ['components/content/ParamFields.tsx', 'role="switch"', 1, 'input', 'a switch (an on/off input)'],
  ['components/content/PipelineBoard.tsx', 'aria-label="Close"', 1, 'button', 'a round close button'],
  ['components/content/PipelineBuilder.tsx', 'aria-label="Close"', 1, 'button', 'a round close button'],
  ['components/database/DatabaseExplorer.tsx', 'flex h-[4px] overflow-hidden rounded-full', 1, 'progress track', 'a 4px share track'],
  ['components/database/DatabaseExplorer.tsx', 'style={{ height: HEIGHT }}', 1, 'chart mark', 'the plot area of the table-size chart'],
  ['components/knowledge/KnowledgeVisualizer.tsx', 'flex flex-wrap gap-1 rounded-[11px]', 1, 'input', 'the section nav track (a segmented control)'],
  ['components/tasks/Board.tsx', 'tasks-todo flex', 1, 'row', 'a to-do row (the linked one marked), a row in a list, not a face'],
  ['components/tasks/Board.tsx', 'tasks-card overflow-hidden', 1, 'chart mark', 'calendar events: marks placed in the grid cells by their time'],
  ['components/tasks/RangePill.tsx', 'onClick={() => setOpen((o) => !o)}', 1, 'button', 'the range dropdown button'],
  ['components/tasks/TaskOverview.tsx', 'relative flex rounded-full bg-os-surface2 p-0.5', 1, 'input', 'the segmented control track (its thumb is the "on" light)'],
  ['components/tasks/TaskOverview.tsx', 'onClick={() => setMenu((m) => !m)}', 1, 'button', 'the menu button'],
  ['components/tasks/WorkloadTable.tsx', 'flex items-center gap-2 rounded-md-t border border-os-border bg-os-surface2 px-2.5 py-1.5', 1, 'input', 'the search field'],
  ['components/tasks/WorkloadTable.tsx', 'aria-pressed={openOnly}', 1, 'button', 'a toggle button'],
  ['components/terminal.tsx', 'border-b-2 bg-os-surface2', 1, 'text', 'a keyboard hint (<kbd>)'],
  ['components/trading/TradingClassic.tsx', 'mx-auto mt-4 flex w-full min-w-0 max-w-[520px]', 1, 'input', 'the ticker field'],
  ['founderos/components/ConductorComposer.tsx', 'aria-label="Send"', 1, 'button', 'the round send button'],
  ['founderos/components/DeliverablesList.tsx', 'pressable is-dark flex shrink-0', 2, 'button', 'open and download links drawn as buttons'],
  ['founderos/components/InstagramDmInbox.tsx', 'inline-flex gap-1 rounded-md-t', 1, 'input', 'the tab track of a segmented control'],
  ['founderos/components/PostComposer.tsx', 'onDragOver=', 1, 'input', 'the media drop zone (a file target)'],
  ['founderos/components/PostComposer.tsx', 'flex flex-1 items-center gap-2 rounded-sm-t', 1, 'input', 'a field (a label around its input)'],
  ['founderos/components/PostComposer.tsx', 'flex items-center gap-2 rounded-sm-t', 1, 'input', 'a field (a label around its input)'],
  ['founderos/components/terminal.tsx', 'border-b-2 bg-os-surface', 1, 'text', 'a keyboard hint (<kbd>)'],
];

const CSS_FOUND = cssFindings(ROOT, CARRIES);
const TSX_FOUND = tsxFiles(ROOT, ['app', 'components', 'founderos', 'lib']).flatMap((abs) => {
  const f = abs.slice(ROOT.length + 1);
  return tsxFindings(f, readFileSync(abs, 'utf8'));
});
const cssKey = (f: CssFinding) => `${f.file} ${norm(f.selector)}`;
const tsxCovered = (f: TsxFinding) => TSX_NOT_FACES.filter(([file, needle]) => file === f.file && f.opening.includes(needle));

describe('the sweep: no face in the app paints the old look', () => {
  test('stylesheets: every face rule that fills, highlights its top edge or sits off the pack radii is outranked by the pack, or is not a face', () => {
    const listed = new Set(CSS_NOT_FACES.map(([f, s]) => `${f} ${norm(s)}`));
    const left = CSS_FOUND.filter((f) => !listed.has(cssKey(f))).map((f) => `${f.file}:${f.line} ${f.selector} (${f.issue}: ${f.detail})`);
    expect(left).toEqual([]);
  });

  test('TSX: every element with a radius and a surface utility is a face of the pack (in every branch of its className), or is not a face', () => {
    const left = TSX_FOUND.filter((f) => tsxCovered(f).length !== 1).map((f) => `${f.file}:${f.line} <${f.tag}> ${f.fill.join(' ')} :: ${f.opening.slice(0, 120)}`);
    expect(left).toEqual([]);
  });

  test('every CSS exception still paints (the list cannot rot), and a retired class is rendered by no element', () => {
    const found = new Set(CSS_FOUND.map(cssKey));
    for (const [file, sel, why] of CSS_NOT_FACES) {
      expect(found.has(`${file} ${norm(sel)}`), `${file} ${sel}`).toBe(true);
      if (why === 'retired') {
        const cls = /\.([\w-]+)$/.exec(sel)![1];
        expect(elementsWith(ROOT, SOURCES, cls), cls).toEqual([]);
        expect(SOURCES.filter((f) => new RegExp(`(?<![\\w-])${cls}(?![\\w-])`).test(read(f))), cls).toEqual([]);
      }
    }
  });

  test('every TSX exception names exactly as many elements as it says (the list cannot rot)', () => {
    for (const [file, needle, count] of TSX_NOT_FACES) {
      expect(TSX_FOUND.filter((f) => f.file === file && f.opening.includes(needle)).length, `${file}: ${needle}`).toBe(count);
    }
  });

  test('every exception has a reason, and only the non-face kinds are allowed', () => {
    for (const e of [...CSS_NOT_FACES.map(([a, b, w, r]) => [a, b, w, r] as const), ...TSX_NOT_FACES.map(([a, b, , w, r]) => [a, b, w, r] as const)]) {
      expect(e[3].length, `${e[0]} ${e[1]}`).toBeGreaterThan(8);
    }
  });
});

describe('the sweep bites', () => {
  const CLEAR = clearers(cssRules('app/globals.css', GLOBALS_CSS.slice(GLOBALS_CSS.indexOf(PACK_MARK))));
  const fake = (sel: string, body: string) => cssRules('components/new.css', `${sel} { ${body} }`, 1e6)[0];

  test('specificity follows Selectors 4', () => {
    expect(specificity(':root[data-ground] :is(.glass-face, .l-card)')).toEqual([0, 3, 0]);
    expect(specificity(":root[data-ui='glados'] .wfc[data-skin='g'] .lib-card")).toEqual([0, 5, 0]);
    expect(specificity('.a:where(.b) > p::before')).toEqual([0, 1, 2]);
    expect(specificity('.bh-card:not(.bh-glyph)')).toEqual([0, 2, 0]);
    expect(specificity('select.ae-o-pill.center')).toEqual([0, 2, 1]);
  });

  test('the old look is recognised: fills, gradients, top highlights, off-spec radii; the pack radii and tokens pass', () => {
    expect(oldLook('background: var(--surface);').map((x) => x.issue)).toEqual(['fill']);
    expect(oldLook('background: linear-gradient(red, blue);').map((x) => x.issue)).toEqual(['fill']);
    expect(oldLook('box-shadow: 0 2px 4px black, inset 0 1px 0 white;').map((x) => x.issue)).toEqual(['top-highlight']);
    expect(oldLook('border-radius: 14px;').map((x) => x.issue)).toEqual(['radius']);
    expect(oldLook('background: transparent; border-radius: var(--gl-r-card);')).toEqual([]);
    expect(oldLook('background: var(--solid-bg); border-radius: 999px;')).toEqual([]);
    for (const r of ['24px', '16px', 'var(--r-card)', 'var(--gl-r-card) var(--gl-r-card) 0 0', '50%']) expect(radiusOk(r), r).toBe(true);
    for (const r of ['12px', 'var(--r-input)', '18px 18px 0 0', 'var(--p-radius)']) expect(radiusOk(r), r).toBe(false);
  });

  test('a new filled card in a component sheet is not cleared by the pack; a pack face class is', () => {
    const card = fake('.new-card', 'background: var(--surface); border-radius: 14px;');
    expect(faceClassOf('.new-card')).toBe('new-card');
    expect(cleared(card, '.new-card', 'background', CLEAR)).toBe(false);
    const gcard = fake('.g-card', 'background: var(--g-card);');
    // a component sheet loads after globals.css: only a higher specificity wins, and the pack's (0,3,0) does
    expect(cleared(gcard, '.g-card', 'background', CLEAR)).toBe(true);
    const loud = fake(":root[data-ui='glados'] .x .g-card", 'background: var(--g-card);');
    expect(cleared(loud, ":root[data-ui='glados'] .x .g-card", 'background', CLEAR)).toBe(false);
    // a :not() on the pack rule's subject keeps what it leaves out (the Blueprint glyphs, the Octane <select>)
    expect(cleared(fake('.bh-card.bh-glyph', 'background: red;'), '.bh-card.bh-glyph', 'background', CLEAR)).toBe(false);
    expect(cleared(fake('select.ae-o-pill', 'background: red;'), 'select.ae-o-pill', 'background', CLEAR)).toBe(false);
  });

  test('a filled card in TSX is caught, in either branch of its className; a pack face is not', () => {
    const scan = (src: string) => tsxFindings('components/X.tsx', `export const X = (p: { on: boolean }) => (${src});`);
    expect(scan('<div className="rounded-lg-t border bg-os-surface2 p-4" />')).toHaveLength(1);
    expect(scan('<div className="glass-well rounded-lg-t bg-os-surface2 p-4" />')).toHaveLength(0);
    expect(scan("<div className={`rounded-full px-2 ${p.on ? 'glass-chip gl-on' : 'bg-os-surface'}`} />")).toHaveLength(1);
    expect(scan("<div className={`glass-chip rounded-full px-2 ${p.on ? 'gl-on' : 'text-os-muted'}`} />")).toHaveLength(0);
    expect(scan('<input className="rounded-lg-t bg-os-surface2" />')).toHaveLength(0);
    const sf = ts.createSourceFile('x.tsx', "<a className={cx('a', on ? 'b' : 'c')} />", ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    let variants: string[][] = [];
    const visit = (n: ts.Node) => {
      if (ts.isJsxSelfClosingElement(n)) variants = classVariants(n);
      ts.forEachChild(n, visit);
    };
    visit(sf);
    expect(variants).toEqual([['a', 'b'], ['a', 'c']]);
  });

  test('a class always rendered with a pack class is judged with it (the funnel tip, a Lab menu)', () => {
    expect(CARRIES['lab2-menu']).toEqual(['lab2-glass']);
    expect(CARRIES['c-pop']).toEqual(['c-pop']);
  });
});

describe('"on" and status on the swept faces: a tinted rim (and the glow behind on chips), never a fill', () => {
  test('selected and expanded workflow nodes, the mint modal, the selected Blueprint card and the operator\'s bubbles tint their rim', () => {
    for (const sel of [
      ':root[data-ground] .wfc .node:is(.is-selected, .is-expanded) .frame',
      ':root[data-ground] .wfc .modal.mint',
      ':root[data-ground] .bh-node.is-selected .bh-card:not(.bh-glyph)',
      ':root[data-ground] .ib-w-card.is-accent',
      ':root[data-ground] .wz-bubble-user',
    ]) {
      expect(p5(sel, '--gl-tint'), sel).toBe('var(--gl-on)');
      expect(p5(sel, '--gl-rim-a'), sel).toBe('45%');
    }
    // the expanded node and the mint modal read in the theme's ink, not the on-gradient ink
    expect(p5(':root[data-ground] .wfc .node.is-expanded .frame', '--wf-on-grad')).toBe('var(--text)');
    expect(p5(':root[data-ground] .wfc .node.is-expanded .frame', 'color')).toBe('var(--text)');
  });

  test('chips that were filled when on take the glow behind (a blurred ::before at .55) and read in ink again', () => {
    expect(P5).toMatch(/\.fc-chip\.on\n?\)::before \{[^}]*filter: blur\(40px\);[^}]*opacity: 0\.55;/);
    expect(p5(':root[data-ground] :is(.bh-pill.hot, .bh-pill.dyn, .wfc .pill-add.is-open, .wfc .pill-set.is-open, .ib-filter-chip, .trd-c-chip[aria-checked=\'true\'])', 'color')).toBe('var(--text)');
  });

  test('TSX "on" states light up instead of filling: the segmented thumbs, the venture chips, the media filter, chat bubbles', () => {
    expect(read('components/tasks/TaskOverview.tsx')).toMatch(/className="glass-chip gl-on absolute inset-y-0\.5 rounded-full/);
    expect(read('founderos/components/SlidingTabs.tsx')).toMatch(/pill \? 'gl-on /);
    expect(read('app/org/page.tsx')).toMatch(/active \? 'glass-chip gl-on text-os-text'/);
    expect(read('app/org/page.tsx')).not.toMatch(/background: ventureColor\(v\.color\), borderColor/);
    expect(read('components/BrainDump.tsx')).toMatch(/on \? 'glass-chip gl-on text-os-text'/);
    expect(read('components/content/MediaLibrary.tsx')).toMatch(/filter\.kind === k \? 'glass-chip gl-on text-os-text'/);
    for (const f of ['components/InstagramDmInbox.tsx', 'founderos/components/InstagramDmInbox.tsx']) {
      expect(read(f), f).toMatch(/out \? 'gl-on-rim text-os-text'/);
      expect(read(f), f).not.toMatch(/'bg-os-accent text-os-bg'/);
    }
  });

  test('the Blueprint start and router glyphs are graph marks, not cards: they carry bh-glyph and the card rule skips them', () => {
    expect(read('components/blueprint/HierarchyNode.tsx')).toMatch(/<div className="bh-card bh-glyph">/);
    expect(P5).toMatch(/:root\[data-ground\] \.bh-card:not\(\.bh-glyph\),/);
  });
});

describe('the Wizard dock (GladOS) is page-tall and fixed: its frost and light are a layer under its content', () => {
  test('the dock is not itself frosted (nothing inside it loses its frost); its ::before carries the strong frost and the six layers', () => {
    const dock = ":root[data-ground][data-ui='glados'] .g-dock";
    expect(p5(dock, 'background')).toBe('transparent');
    expect(p5(dock, 'backdrop-filter')).toBeUndefined();
    expect(p5(dock, 'box-shadow')).toBe('none');
    expect(p5(dock, 'isolation')).toBe('isolate');
    expect(p5(`${dock}::before`, 'backdrop-filter')).toBe('var(--gl-blur-strong)');
    expect(p5(`${dock}::before`, 'box-shadow')).toBe(SIX);
    expect(p5(`${dock}::before`, 'z-index')).toBe('-1');
    // the open dock's drop (var(--g-shadow), equal specificity, earlier) loses to it
    expect(specificity(dock)).toEqual(specificity(":root[data-ui='glados'] .g-dock[data-open]"));
  });
});

describe('a face nested in a face does not frost again, and the swept faces go solid without blur', () => {
  test('the swept faces drop their frost inside a frosting face', () => {
    expect(P5).toMatch(/\) :is\(\.ae-card-2, \.ib-card, \.fc-card, \.fc-panel, \.g-vizstage-hud, \.bh-hud, \.ae-o-hud, \.ae-o-strip, \.trd-c-stat, \.g-stat\) \{\s*-webkit-backdrop-filter: none;\s*backdrop-filter: none;/);
  });

  test('wells are lit, never frosted; chips are lit like a chip in a face', () => {
    expect(p5(':root[data-ground] .wfc .node .tile', 'box-shadow')).toBe(LIT);
    expect(p5(':root[data-ground] .wfc .node .tile', 'backdrop-filter')).toBe('none');
    expect(p5(':root[data-ground] .wfc .node .tile', 'border-radius')).toBe('var(--gl-r-well)');
    expect(p5(':root[data-ground] .wfc .pill', 'border-radius')).toBe('var(--gl-r-chip)');
    expect(p5(':root[data-ground] .wfc .pill', 'box-shadow')).toMatch(/^inset 0 0 0 1px .*inset 0 -18px 26px -18px color-mix\(in srgb, var\(--gl-tint\) var\(--gl-pool-a\), transparent\)$/);
  });

  test('reduced transparency and no backdrop-filter: every swept frosting face is the solid face', () => {
    for (const media of [/prefers-reduced-transparency: reduce/, /^@supports not/]) {
      const r = cssRules('app/globals.css', P5).filter((x) => x.at.some((a) => media.test(a)));
      expect(r.length, String(media)).toBeGreaterThan(0);
      for (const x of r) expect(norm(x.body), String(media)).toMatch(/background: var\(--solid-bg\)/);
      for (const cls of ['.bh-ask', '.wfc .table', '.wfc .lib-card', '.kg-know-stats .g-stat', '.fc-drawer']) expect(r.map((x) => x.selectors.join()).join(), `${media} ${cls}`).toContain(cls);
    }
  });
});
