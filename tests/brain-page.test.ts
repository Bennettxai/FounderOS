import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, test } from 'vitest';

const read = (p: string) => readFileSync(join(process.cwd(), p), 'utf8');

/**
 * /brain layout contract. The Example-6 radial is the hero directly under the
 * title, with the knowledge graph as the next section.
 *
 * Capture moved OFF the header on 2026-08-26 . It now floats over the canvas as BrainDock,
 * which owns the open/collapsed states; the header carries nothing.
 */
describe('/brain header capture + graph placement', () => {
  test('capture floats over the canvas as the dock, not in the header slot', () => {
    const page = read('app/brain/page.tsx');
    expect(page).toMatch(/<BrainDock \/>/);
    // the header is bare — capture is no longer parked in its right slot
    expect(page).not.toMatch(/right=\{<BrainDump/);
    // and the dock lives INSIDE the radial section so it overlays the graph
    const sectionStart = page.indexOf('<section', page.indexOf('<PageHeader'));
    const sectionEnd = page.indexOf('</section>', sectionStart);
    expect(page.indexOf('<BrainDock />')).toBeGreaterThan(sectionStart);
    expect(page.indexOf('<BrainDock />')).toBeLessThan(sectionEnd);
  });

  test('retrieval notices render above the canvas, at title height, without pushing it', () => {
    const page = read('app/brain/page.tsx');
    // the landing slot lives in the header, level with the G-Brain title
    expect(page).toMatch(/id="gbrain-notice-slot"/);
    // h-0 is load-bearing: a slot with real height grows the header, which
    // pushes the canvas down and drops the capture card off the fold. Assert
    // the two classes that carry meaning, not their order — the slot also
    // needs a z-index, and requiring `relative h-0` adjacent broke the moment
    // z-40 landed between them.
    expect(page).toMatch(/id="gbrain-notice-slot"[^>]*className="[^"]*\brelative\b/);
    expect(page).toMatch(/id="gbrain-notice-slot"[^>]*className="[^"]*\bh-0\b/);

    const radial = read('components/GBrainRadial.tsx');
    // the radial portals its notice up into that slot rather than drawing it
    // inside its own frame
    expect(radial).toMatch(/createPortal\(/);
    expect(radial).toMatch(/gbrain-notice-slot/);
  });

  test('the bottom middle belongs to capture, not to the ask bar', () => {
    const radial = read('components/GBrainRadial.tsx');
    // the ask bar moved to the top-left of the frame
    expect(radial).not.toMatch(/ask bar[\s\S]{0,200}bottom-\[56px\][^\n]*left-1\/2/);
    const dock = read('components/BrainDock.tsx');
    // both dock states are anchored to the bottom centre
    expect(dock).toMatch(/bd-line[^"]*absolute bottom-\[52px\] left-1\/2/);
    expect(dock).toMatch(/bd-card (?:gl-glow-host )?absolute bottom-\[46px\] left-1\/2/);
    // and the card grows upward out of the line rather than floating mid-canvas
    expect(dock).toMatch(/transformOrigin: 'bottom center'/);
    expect(dock).not.toMatch(/top-1\/2/);
  });

  test('the dock folds down instead of discarding a half-typed note', () => {
    const dock = read('components/BrainDock.tsx');
    // BrainDump is rendered unconditionally — hiding it, never unmounting it,
    // is what keeps a draft alive across a collapse
    expect(dock).toMatch(/<BrainDump compact \/>/);
    expect(dock).not.toMatch(/open &&\s*<BrainDump/);
  });

  test('the dock is hover-driven, with no idle timer to shrink it mid-use', () => {
    const dock = read('components/BrainDock.tsx');
    // open is derived from pointing at it / using it, never from a countdown
    expect(dock).toMatch(/const open = hover \|\| pinned \|\| focused;/);
    expect(dock).not.toMatch(/IDLE_MS/);
    expect(dock).not.toMatch(/setOpen\(/);
    // hover handlers ride BOTH states, so crossing between them holds it open (in each style's tree:
    // Classic and GladOS, glados-ui S05 F05.12)
    expect(dock.match(/\{\.\.\.hoverProps\}/g)).toHaveLength(4);
    // a click inside pins it — the file dialog steals the pointer otherwise
    expect(dock).toMatch(/setPinned\(!!wrapRef\.current\?\.contains/);
  });

  test('the radial hero leads, the knowledge graph follows', () => {
    const page = read('app/brain/page.tsx');
    const header = page.indexOf('<PageHeader');
    const firstSection = page.indexOf('<section', header);
    // the first section on the page IS the live radial core
    expect(page.indexOf('GBrainRadialLazy', firstSection)).toBeLessThan(page.indexOf('</section>', firstSection));
    // and the knowledge graph is still on the page, right after
    expect(page.indexOf('BrainGraphView')).toBeGreaterThan(page.indexOf('GBrainRadialLazy'));
  });

  test('BrainDump has a compact mode with document drop that reads files as text', () => {
    const dump = read('components/BrainDump.tsx');
    expect(dump).toMatch(/compact/);
    expect(dump).toMatch(/onDrop/);
    expect(dump).toMatch(/\.text\(\)/);
    // dropped docs keep their filename as the note title
    expect(dump).toMatch(/name\.replace/);
  });
});
