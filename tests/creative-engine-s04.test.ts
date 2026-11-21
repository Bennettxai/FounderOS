import { readFileSync, readdirSync } from 'node:fs';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { afterEach, beforeEach, describe, expect, test } from 'vitest';
import { brandInitials } from '@/lib/creative-engine/format';
import { capError, parseCap, SESSION_CAP_MAX } from '@/lib/creative-engine/connect';
import { dedupeSignals } from '@/lib/foreplay/signals';
import { sampleSignals } from '@/lib/creative-engine/demo';

/**
 * SPEC S04 (visual fixes, accessibility, Runs tab) and G8 (Connect Arcads):
 * the pure rules, the setup route's guards, and source scans for the house
 * rules a unit test cannot reach (AC04.7, AC04.8).
 */
const root = path.join(__dirname, '..');
const read = (p: string) => readFileSync(path.join(root, p), 'utf8');
const engineDir = path.join(root, 'components/creative-engine');
const engineFiles = readdirSync(engineDir).filter((f) => f.endsWith('.tsx'));

describe('AC04.7: no raw colour in components/creative-engine', () => {
  test.each(engineFiles)('%s has no hex, rgb() or Tailwind white/black colour', (f) => {
    const lines = read(`components/creative-engine/${f}`).split('\n');
    const bad = lines
      .map((line, i) => ({ line, n: i + 1 }))
      .filter(({ line }) => !/^\s*(\/\/|\*|\/\*)/.test(line))
      .filter(({ line }) => {
        if (/#[0-9a-fA-F]{3,8}\b/.test(line)) return true;
        if (/rgba?\(/.test(line)) return true;
        // §3.2.1: a media letterbox (bg-black behind a <video>) is the one allowed literal.
        const literals = line.match(/\b(?:text|bg|border|ring|fill|stroke|from|to|via)-(?:white|black)(?:\/\d+)?\b/g) ?? [];
        return literals.some((l) => !(l === 'bg-black' && /<video\b/.test(line)));
      })
      .map(({ n, line }) => `${f}:${n}: ${line.trim().slice(0, 120)}`);
    expect(bad).toEqual([]);
  });
});

describe('AC04.8 / B37: sheets and drawers are dialogs on the shared hook (X1)', () => {
  test('the Drawer and the Sheet use useDialog, name themselves, and drop their own Escape listeners', () => {
    // The paper Drawer and Sheet live in classic.tsx since GP1 (one engine, two skins); under the kit the
    // engine opens GDrawer and GSheet, which run on the same hook.
    const classic = read('components/creative-engine/classic.tsx');
    const kit = read('components/g/Overlays.tsx');
    expect(read('components/creative-engine/skin-dash.tsx')).toMatch(/<GSheet[\s\S]*<GDrawer/);
    // `open` (not `true`): a closed sheet releases the hook at once while its face melts out (lib/use-melt)
    expect(kit).toMatch(/useDialog\(\{ open(?:: true)?, onClose, labelledBy: titleId/);
    for (const src of [classic.slice(classic.indexOf('function ClassicDrawer('), classic.indexOf('function ClassicConfirmDelete(')), classic.slice(classic.indexOf('function ClassicSheet('), classic.indexOf('function ClassicDrawer('))]) {
      // Escape (the hook's onClose) melts the face first; the parent's onClose runs once it has melted.
      expect(src).toContain('useDialog({ open: true, onClose: () => melt.close(), labelledBy: titleId })');
      expect(src).toContain('{...dlg.props}');
      expect(src).toContain('id={titleId}');
      expect(src).toContain('data-dialog-close');
      expect(src).not.toMatch(/addEventListener\('keydown'/);
    }
  });
  test('B42: Engine and Runs are a tablist with aria-selected and arrow keys; the tab is in the URL', () => {
    const dash = read('components/creative-engine/Dashboard.tsx');
    const classic = read('components/creative-engine/classic.tsx');
    // Classic draws the tabs (classic.tsx ClassicTitleTabs); the kit draws GTabs (tablist, arrows, Home/End).
    expect(dash).toContain('<TitleTabs');
    expect(classic).toContain('role="tablist"');
    expect(classic).toContain('role="tab"');
    expect(classic).toContain('aria-selected={value === t.value}');
    expect(classic).toMatch(/ArrowLeft/);
    expect(dash).toContain('role="tabpanel"');
    expect(dash).toContain("q.set('tab', 'runs')");
  });
  test('B38: one focus ring for every control on paper, in the S04 block', () => {
    const css = read('app/globals.css');
    const block = css.slice(css.indexOf('/* ── S04 visual fixes and accessibility'), css.indexOf('/* ── end S04 visual fixes and accessibility ── */'));
    expect(block.length).toBeGreaterThan(100);
    expect(block).toMatch(/\.ae-paper :where\(:focus-visible\) \{ outline: 2px solid var\(--p-ring\)/);
    expect(block).not.toMatch(/#[0-9a-fA-F]{3,8}\b|rgba?\(/);
  });
});

describe('glass review, findings 1 and 6: the Classic sheets render in a portal and melt out', () => {
  const classic = read('components/creative-engine/classic.tsx');
  const fn = (name: string) => {
    const i = classic.indexOf(`export function ${name}(`);
    const j = classic.indexOf('\nexport function', i + 1);
    return classic.slice(i, j < 0 ? undefined : j);
  };

  test.each(['ClassicSheet', 'ClassicDrawer', 'ClassicConfirmDelete'])('%s: close melts the face, then the parent unmounts it', (name) => {
    const src = fn(name);
    // lib/use-melt useMeltClose on the dialog's own face: close() starts the melt, the parent's callback runs on gl-melt's end
    expect(src).toMatch(/const melt = useMeltClose\((?:onClose|onCancel), dlg\.ref\);/);
    // (the delete confirm's Escape goes through its busy guard, which melts)
    expect(src).toMatch(/useDialog\(\{ open: true, onClose: \(\) => (?:melt\.close|cancel)\(\), labelledBy: titleId \}\)/);
    if (name === 'ClassicConfirmDelete') expect(src).toMatch(/const cancel = \(\) => \{\s*if \(!busy\) melt\.close\(\);\s*\};/);
    expect(src).toContain('{...melt.layer}');
    expect(src).toContain('onAnimationEnd={melt.onAnimationEnd}');
    expect(src).toContain("${melt.melting ? ' is-melting' : ''}");
    // nothing closes around the melt: the scrim, Close, Back-to and Cancel all go through it
    expect(src).not.toMatch(/onClick=\{(?:onClose|onCancel)\}/);
  });

  test.each(['ClassicSheet', 'ClassicDrawer', 'ClassicConfirmDelete'])('%s: the layer is portalled to <body> and carries the paper scope', (name) => {
    const src = fn(name);
    expect(src).toMatch(/return \(\s*<Layer>\s*<div className="ae-paper ae-sheet-layer /);
    expect(src).toMatch(/<\/div>\s*<\/Layer>\s*\);\s*\}/);
  });

  test('the melt plays on the paper sheet: it beats both entrances, and reduced motion skips it', () => {
    const css = read('app/globals.css').replace(/\/\*[\s\S]*?\*\//g, '');
    const sel = ':root[data-ground] .ae-sheet-layer > .ae-sheet.is-melting';
    const esc = sel.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    expect(css).toMatch(new RegExp(`${esc} \\{ animation: gl-melt var\\(--gl-dur-melt\\) linear forwards; pointer-events: none; \\}`));
    expect(css).toMatch(new RegExp(`@media \\(prefers-reduced-motion: reduce\\) \\{[^{}]*(?:\\{[^{}]*\\}[^{}]*)*${esc} \\{ animation: none; \\}`));
  });
});

describe('B27: one RunRow for the Runs tab and the Runs drawer (C24)', () => {
  test('both lists render RunRow with Abandon, and the tab has Check now', () => {
    const drawers = read('components/creative-engine/Drawers.tsx');
    const dash = read('components/creative-engine/Dashboard.tsx');
    expect(drawers).toMatch(/export function RunRow\(/);
    expect(drawers.slice(drawers.indexOf('export function RunsDrawer'))).toContain('<RunRow ');
    const tab = dash.slice(dash.indexOf('function RunsTab('));
    expect(tab).toContain('<RunRow ');
    expect(tab).toContain('<CheckNow ');
    expect(dash).not.toContain('function RunsTable(');
    // the error wraps under the title instead of truncating it to "nan…"
    expect(drawers).not.toMatch(/max-w-\[260px\] shrink-0 truncate/);
  });
});

describe('B26: Proof library rows link into AdPilot; repeated signals collapse (X5, G2)', () => {
  test('saved ads, watched brands and signals are links to /adpilot?ad= or ?brand=', () => {
    const drawers = read('components/creative-engine/Drawers.tsx');
    const proof = drawers.slice(drawers.indexOf('export function ProofDrawer'), drawers.indexOf('export function CreditsDrawer'));
    expect(proof).toContain('<a href={adpilotHref(link)}');
    expect(proof.match(/\{row\(/g)?.length).toBe(3);
    expect(proof).toContain('proofLibrarySections(saved, watchlist, signals)');
  });
  test('the 24 h window collapses the same message, and keeps different ones', () => {
    const at = '2026-10-08T10:00:00.000Z';
    const later = '2026-10-08T16:00:00.000Z';
    const a = { type: 'new_launch' as const, brandId: 'b1', brandName: 'Pinion', message: 'Pinion launched 2 new ads', at };
    const out = dedupeSignals([a, { ...a, at: later }, { ...a, message: 'Pinion: an ad passed 30 days live', at: later }], { windowMs: 24 * 3600_000 });
    expect(out.map((s) => s.count)).toEqual([2, 1]);
  });
  test('the sample signals stay fictional and each points at an ad or a brand', () => {
    for (const s of sampleSignals()) expect(s.adId || s.brandId).toBeTruthy();
  });
});

describe('B32: brand initials', () => {
  test('two words give two initials; one word gives two letters; never a word fragment', () => {
    expect(brandInitials('Northwind Tools')).toBe('NT');
    expect(brandInitials('Kestrel Systems')).toBe('KS');
    expect(brandInitials('Pinion')).toBe('PI');
    expect(brandInitials('orbit-labs')).toBe('OL');
    expect(brandInitials('  ')).toBe('?');
  });
});

describe('G8 A6: the session cap is a whole number from 1', () => {
  test('parseCap', () => {
    expect(parseCap('200')).toBe(200);
    expect(parseCap(' 1 ')).toBe(1);
    expect(parseCap('')).toBeNull();
    expect(parseCap('0')).toBeNull();
    expect(parseCap('12.5')).toBeNull();
    expect(parseCap(String(SESSION_CAP_MAX + 1))).toBeNull();
  });
  test('capError names the rule', () => {
    expect(capError('')).toBe('Enter a whole number from 1');
    expect(capError('0')).toBe('Enter a whole number from 1');
    expect(capError('100001')).toBe('Enter a whole number up to 100,000');
    expect(capError('50')).toBeNull();
  });
});

describe('G8: the setup route refuses a zero cap and, in Filming, product creation', () => {
  let dir: string;
  beforeEach(() => {
    dir = fs.mkdtempSync(path.join(os.tmpdir(), 'ce-g8-'));
    process.env.CREATIVE_ENGINE_STORE_DIR = dir;
    process.env.CREATIVE_ENGINE_ASSET_DIR = path.join(dir, 'assets');
  });
  afterEach(() => {
    delete process.env.CREATIVE_ENGINE_STORE_DIR;
    delete process.env.CREATIVE_ENGINE_ASSET_DIR;
    delete process.env.DEMO_MODE;
    fs.rmSync(dir, { recursive: true, force: true });
  });
  const post = async (body: unknown) => {
    const { POST } = await import('@/app/api/creative-engine/arcads/setup/route');
    return POST(new Request('http://x/api/creative-engine/arcads/setup', { method: 'POST', body: JSON.stringify(body) }));
  };
  test('sessionCreditCap 0 is a 400 and the stored cap is unchanged', async () => {
    const res = await post({ action: 'settings', sessionCreditCap: 0 });
    expect(res.status).toBe(400);
    const { readSettings } = await import('@/lib/creative-engine/store');
    expect(readSettings().sessionCreditCap).toBeGreaterThan(0);
  });
  test('a valid cap saves; sessionStartedAt (the Undo) round-trips', async () => {
    expect((await post({ action: 'settings', sessionCreditCap: 150 })).status).toBe(200);
    const when = '2026-10-01T09:00:00.000Z';
    expect((await post({ action: 'settings', sessionStartedAt: when })).status).toBe(200);
    const { readSettings } = await import('@/lib/creative-engine/store');
    expect(readSettings().sessionCreditCap).toBe(150);
    expect(readSettings().sessionStartedAt).toBe(when);
  });
  test('Q-ENG-14: in Filming, createProduct is refused with its reason', async () => {
    process.env.DEMO_MODE = '1';
    const res = await post({ action: 'createProduct', name: 'Lumen Desk lamp', description: '' });
    expect(res.status).toBe(409);
    expect((await res.json()).error).toBe('Filming mode: creating Arcads products is off');
  });
});

describe('G8: Connect Arcads asks before it spends or writes, and says why a control is off', () => {
  const src = read('components/creative-engine/ConnectArcads.tsx');
  test('Run the probe and Create open a confirm; only the confirm calls the API', () => {
    expect(src).toContain("onClick={() => setAsk('probe')}");
    expect(src).toContain('onClick={() => void firstRun()}');
    expect(src).toMatch(/Spends credits on \$\{probeModel\} \(one request\)\./);
    expect(src).toMatch(/Creates “\{newProduct\.trim\(\)\}” in your Arcads account\./);
  });
  test('product and path pills carry aria-pressed; Reset shows the count and offers Undo; load errors retry', () => {
    // product and path choices are PickPills: aria-pressed on the paper pill (classic.tsx) and on the kit's GChip
    expect(src.match(/<PickPill /g)?.length).toBeGreaterThanOrEqual(3);
    expect(read('components/creative-engine/classic.tsx')).toMatch(/function ClassicPickPill[\s\S]*?aria-pressed=\{on\}/);
    expect(src).toMatch(/Reset session count \(/);
    expect(src).toContain("label: 'Undo'");
    expect(src).toContain('Retry');
  });
  test('every disabled control in the sheet has a title', () => {
    // Each `disabled={…}` with its own tag: from the tag's `<` to the next `<`.
    const tags: string[] = [];
    for (let i = src.indexOf(' disabled={'); i >= 0; i = src.indexOf(' disabled={', i + 1)) {
      const start = src.lastIndexOf('<', i);
      const end = src.indexOf('<', i);
      tags.push(src.slice(start, end));
    }
    expect(tags.length).toBeGreaterThan(5);
    for (const tag of tags) expect(tag).toMatch(/title=\{/);
  });
});

describe('§3.1.7: live research asks before it spends Foreplay credits', () => {
  test('Generate arms first when not dry, names the source and count, and disarms on edit', () => {
    const gen = read('components/creative-engine/CreativeGenerator.tsx');
    expect(gen).toContain('if (!health.dry && !armed) { setArmed(true); return; }');
    expect(gen).toContain('`Spend up to ${DEPTH_BUDGET[depth].maxCalls} Foreplay credits. Run`');
    expect(gen).toContain('useEffect(() => { setArmed(false); }, [text, depth]);');
  });
});
