/**
 * Fixed under frost (glass canon: "never frost page-tall containers"; review of the glass branch,
 * finding 1). backdrop-filter makes an element the containing block and the stacking context of its
 * position: fixed descendants, so a modal, drawer or popout inside a frosted element is sized and
 * centred on that element, clipped by it, trapped under the sticky topbar and unable to dim the
 * chrome. Every such layer is portalled to <body>, and no page-tall container frosts.
 *
 * The scan (tests/helpers/glass-fixed.ts) follows components across files: a fixed element is
 * reported when anything frosted renders above it, in its own file or wherever its component (or a
 * component above that, up to the pages and the layouts that mount them) is rendered, unless a
 * createPortal (Layer, GSheet, GDrawer, GDialog, LDialog) sits between.
 */
import { describe, expect, test } from 'vitest';
import { cssSubjects, fixedBody, fixedKey, frostsBody, scanFixedUnderFrost } from './helpers/glass-fixed';
import { CSS_NO_COMMENTS } from './helpers/theme-css';

const SCAN = scanFixedUnderFrost();
const show = (fs: ReturnType<typeof scanFixedUnderFrost>['findings']) =>
  fs.map((f) => `${fixedKey(f)}\n    under ${f.under}${f.via.length ? `\n    via ${f.via.join(' < ')}` : ''}`);

describe('the lint itself', () => {
  const css = `
    .frosty { backdrop-filter: blur(4px); }
    .solid { backdrop-filter: none; }
    .m-layer { position: fixed; inset: 0; }
    :root[data-ground] :is(.panel-a, .panel-b).hot { -webkit-backdrop-filter: blur(2px); }
    .bar::before { backdrop-filter: blur(8px); }
  `;
  const scan = (files: Record<string, string>) =>
    scanFixedUnderFrost({ css, files: Object.entries(files).map(([file, src]) => ({ file, src })) }).findings.map((f) => `${f.file}:${f.line} under ${f.under}${f.via.length ? ` via ${f.via.join('<')}` : ''}`);

  test('CSS: frost and fixed subjects, :is() members, pseudo-elements and none aside', () => {
    expect(cssSubjects(frostsBody, css)).toEqual([['frosty'], ['hot', 'panel-a'], ['hot', 'panel-b']]);
    expect(cssSubjects(fixedBody, css)).toEqual([['m-layer']]);
  });

  test('a fixed element inside a frosted one in the same file', () => {
    const files = {
      'components/A.tsx': `export function A() { return <div className="glass-face p-4"><div className="fixed inset-0" /></div>; }`,
      'components/B.tsx': `export function B() { return <div className="frosty"><div style={{ position: 'fixed' }} /></div>; }`,
      'components/C.tsx': `export function C() { return <div className="backdrop-blur-sm"><div className="m-layer" /></div>; }`,
      'components/D.tsx': `export function D() { return <div className="solid bar"><div className="fixed" /></div>; }`,
    };
    expect(scan(files)).toEqual([
      'components/A.tsx:1 under <div className="glass-face p-4">',
      'components/B.tsx:1 under <div className="frosty">',
      'components/C.tsx:1 under <div className="backdrop-blur-sm">',
    ]);
  });

  test('portalled layers are fine, through createPortal or a component that portals its children', () => {
    const files = {
      'components/Layer.tsx': `import { createPortal } from 'react-dom'; export function Layer({ children }) { return createPortal(children, document.body); }`,
      'components/A.tsx': `import { createPortal } from 'react-dom'; export function A() { return <div className="glass-face">{createPortal(<div className="fixed inset-0" />, document.body)}</div>; }`,
      'components/B.tsx': `import { Layer } from './Layer'; export function B() { return <div className="glass-face"><Layer><div className="fixed inset-0" /></Layer></div>; }`,
    };
    expect(scan(files)).toEqual([]);
  });

  test('across files: the component is rendered under frost, or wrapped by a frosting component', () => {
    const files = {
      'components/Modal.tsx': `export function Modal() { return <div className="fixed inset-0 grid" />; }`,
      'components/Card.tsx': `export function Card({ children }) { return <section className="g-card">{children}</section>; }`,
      'components/Mid.tsx': `import { Modal } from './Modal'; export default function Mid() { return <div><Modal /></div>; }`,
      'components/Page.tsx': `import { Card } from './Card'; import Mid from '@/components/Mid'; export function Page() { return <Card><Mid /></Card>; }`,
      'components/Free.tsx': `import Mid from './Mid'; export function Free() { return <main><Mid /></main>; }`,
    };
    expect(scan(files)).toEqual(['components/Modal.tsx:1 under <Card> (wraps its children in <section className="g-card">) at components/Page.tsx:1 via Modal<Mid']);
  });

  test('pages render in the children slot of the layouts above them', () => {
    const files = {
      'components/Slab.tsx': `export function Slab({ children }) { return <div className="l-card">{children}</div>; }`,
      'app/layout.tsx': `import { Slab } from '@/components/Slab'; export default function Root({ children }) { return <body><Slab>{children}</Slab></body>; }`,
      'app/x/page.tsx': `export default function XPage() { return <div className="fixed inset-0" />; }`,
    };
    expect(scan(files)).toEqual([expect.stringMatching(/^app\/x\/page\.tsx:1 under <Slab> \(wraps its children in <div className="l-card">\) at app\/layout\.tsx:0 via XPage$/)]);
  });
});

describe('no fixed layer renders under a frosted element', () => {
  test('every modal, drawer and popout is portalled to <body> (or nothing frosted is above it)', () => {
    expect(show(SCAN.findings)).toEqual([]);
  });
});

/**
 * Everything fixed that may render in place (not portalled), and why. Overlays are not on this list: every
 * modal, drawer and popout layer is portalled to <body> (review finding 1), mounted or not, so a layer
 * that is safe today stays safe when someone mounts it inside a card.
 */
const MAY_STAY_IN_PLACE: { file?: string; has?: string; why: string }[] = [
  { file: 'components/Sidebar.tsx', why: 'chrome: app/layout.tsx mounts it outside the page column (its frost is a ::before layer)' },
  { file: 'components/NavTooltip.tsx', why: 'chrome: the sidebar tooltip; the sidebar frosts only its ::before' },
  { file: 'components/NavProgress.tsx', why: 'chrome: app/layout.tsx mounts it beside the page column' },
  { file: 'components/CommandPalette.tsx', why: 'chrome: app/layout.tsx mounts it beside the page column' },
  { file: 'components/ConductorPanel.tsx', why: 'chrome: app/layout.tsx mounts it beside the page column' },
  { file: 'components/ui/Toast.tsx', has: 'ui-toasts', why: 'the toast stack: its ToastProvider sits in app/layout.tsx beside the page column' },
  { has: 'page-ground', why: 'a page ground: fixed behind the page, never an overlay' },
  { has: 'lab2-aurora', why: 'the Lab ground, behind the page' },
  { has: 'bh-page', why: 'the Blueprint workspace: a full-height page (lib/l-ready.ts, no slab), not an overlay' },
  { has: 'bh-skeleton', why: "the Blueprint workspace's loading frame, the page itself" },
  { has: 'ib-page', why: 'the Inbox workspace: a full-height page (lib/l-ready.ts, no slab), not an overlay' },
  { has: 'g-vizstage', why: 'GVizStage: the stage itself switches to a fixed layer in full screen (portalling it would remount the graph)' },
  { file: 'components/knowledge/KnowledgeVisualizer.tsx', has: 'z-[130]', why: 'the graph itself switches to a fixed layer in full screen (portalling it would remount the canvas)' },
  { file: 'components/GBrainRadial.tsx', has: 'gbr-root', why: 'the radial itself switches to a fixed layer when expanded (portalling it would remount it)' },
  { has: 'lab2-sheet-layer', why: 'Lab sheets render inside the Lab root (.lab, never frosted); their CSS is scoped to it' },
];
const mayStay = (u: { file: string; tokens: string[] }) =>
  MAY_STAY_IN_PLACE.find((a) => (!a.file || a.file === u.file) && (!a.has || u.tokens.includes(a.has)));

describe('every overlay layer is portalled', () => {
  test('nothing fixed renders in place except chrome, grounds and full-screen modes (each with a reason)', () => {
    expect(SCAN.unportalled.filter((u) => !mayStay(u)).map((u) => `${u.file}:${u.line} ${u.fixed}`)).toEqual([]);
  });

  test('every allowance still matches something (stale ones go)', () => {
    for (const a of MAY_STAY_IN_PLACE) expect(SCAN.unportalled.some((u) => mayStay(u) === a), a.why).toBe(true);
  });
});

describe('page-tall containers never frost', () => {
  test('the Legacy slab (the page column every page renders in) is not a frosted face', () => {
    const slab = SCAN.slots.get('components/legacy/LegacySlab.tsx#LegacySlab');
    expect(slab?.frost ?? null).toBeNull();
    expect(cssSubjects(frostsBody, CSS_NO_COMMENTS).filter((c) => c.includes('l-slab') || c.includes('l-frame'))).toEqual([]);
  });

  test('neither are the layouts nor the template', () => {
    for (const k of ['app/layout.tsx#RootLayout', 'app/template.tsx#Template', 'app/lab/layout.tsx#LabLayout']) expect(SCAN.slots.get(k)?.frost ?? null, k).toBeNull();
  });
});
