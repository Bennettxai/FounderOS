import { describe, expect, it } from 'vitest';
import {
  badShadows,
  contrastRatio,
  decodePng,
  diffImages,
  encodePng,
  CLASSIC_MASKS,
  isHued,
  maskImage,
  parseColor,
  parseMasks,
  routeInReady,
  routeSlug,
  stripComments,
} from '../scripts/g-audit/_lib.mjs';

/** Pure helpers behind scripts/g-audit.mjs (SPEC S00 step 8): baseline diff, contrast, shadow rules. */

function image(w: number, h: number, rgba: [number, number, number, number]) {
  const data = Buffer.alloc(w * h * 4);
  for (let i = 0; i < w * h; i++) data.set(rgba, i * 4);
  return { width: w, height: h, data };
}

describe('g-audit Classic baseline masks (live third-party state)', () => {
  it('masks /compute by default; --mask adds selector masks; --no-default-masks drops the defaults', () => {
    expect(Object.keys(CLASSIC_MASKS).sort()).toEqual(['/compute', '/setup']);
    expect(parseMasks([])['/compute']).toHaveLength(1);
    for (const expr of Object.values(CLASSIC_MASKS)) expect(() => new Function(`return ${expr}`)).not.toThrow();
    const m = parseMasks(['--classic-baseline', '--mask', '/setup=.live-pill', '--mask', '/compute=main iframe']);
    expect(m['/setup']).toHaveLength(2);
    expect(m['/setup'][1]).toContain('".live-pill"');
    expect(m['/compute']).toHaveLength(2);
    expect(parseMasks([], { noDefaults: true })).toEqual({});
    expect(() => parseMasks(['--mask', 'setup'])).toThrow(/route>=<selector/);
  });

  it('a masked region differs on neither side, the rest still does', () => {
    const a = image(10, 10, [0, 0, 0, 255]);
    const b = image(10, 10, [0, 0, 0, 255]);
    for (let y = 0; y < 4; y++) for (let x = 0; x < 4; x++) b.data.set([255, 255, 255, 255], (y * 10 + x) * 4); // live state
    b.data.set([255, 255, 255, 255], (9 * 10 + 9) * 4); // a real change outside the mask
    const rects = [{ x: 0.4, y: 0, w: 3.2, h: 3.5 }]; // rounded outward to 0..4 x 0..4
    expect(maskImage(a, rects)).toBe(16);
    maskImage(b, rects);
    expect(diffImages(a, b).differing).toBe(1);
    expect(maskImage(image(2, 2, [0, 0, 0, 255]), [{ x: -5, y: -5, w: 100, h: 100 }])).toBe(4);
  });
});

describe('g-audit PNG and diff', () => {
  it('round-trips RGBA through encode and decode', () => {
    const img = image(7, 5, [10, 200, 30, 255]);
    img.data.set([1, 2, 3, 128], 4 * 12);
    const back = decodePng(encodePng(img));
    expect(back.width).toBe(7);
    expect(back.height).toBe(5);
    expect(Buffer.compare(back.data, img.data)).toBe(0);
  });

  it('counts differing pixels above the tolerance only', () => {
    const a = image(10, 10, [50, 50, 50, 255]);
    const b = image(10, 10, [50, 50, 50, 255]);
    b.data.set([52, 50, 50, 255], 0); // within tolerance 2
    b.data.set([90, 50, 50, 255], 4 * 55); // differs
    const d = diffImages(a, b, { tolerance: 2 });
    expect(d.differing).toBe(1);
    expect(d.ratio).toBeCloseTo(0.01);
    expect(d.sizeChanged).toBe(false);
  });

  it('treats a size change as differing pixels', () => {
    const d = diffImages(image(10, 10, [0, 0, 0, 255]), image(10, 12, [0, 0, 0, 255]));
    expect(d.sizeChanged).toBe(true);
    expect(d.differing).toBe(20);
  });
});

describe('g-audit colour rules', () => {
  it('parses computed colours', () => {
    expect(parseColor('rgb(1, 2, 3)')).toEqual([1, 2, 3, 1]);
    expect(parseColor('rgba(1, 2, 3, 0.5)')).toEqual([1, 2, 3, 0.5]);
    expect(parseColor('none')).toBeNull();
  });

  it('computes WCAG contrast', () => {
    expect(contrastRatio([0, 0, 0], [255, 255, 255])).toBeCloseTo(21, 0);
    expect(contrastRatio([118, 118, 118], [255, 255, 255])).toBeGreaterThan(4.5);
  });

  it('separates neutral shadows from hued shadows and outer glows', () => {
    expect(badShadows('rgba(0, 0, 0, 0.3) 0px 8px 24px 0px')).toEqual([]);
    expect(badShadows('rgba(255, 255, 255, 0.06) 0px 1px 0px 0px inset')).toEqual([]);
    expect(badShadows('rgba(0, 255, 171, 0.4) 0px 8px 24px 0px').map((s) => s.why)).toEqual(['hued shadow']);
    expect(badShadows('rgba(0, 0, 0, 0.5) 0px 0px 30px 0px').map((s) => s.why)).toEqual(['outer glow']);
    expect(badShadows('none')).toEqual([]);
    // the glass recipe (glass pack): a 1px groove hairline, a neutral drop and inset rim/low/pool/shade layers pass
    expect(badShadows('rgba(0, 0, 0, 0.55) 0px 0px 0px 1px')).toEqual([]);
    expect(badShadows('rgba(16, 16, 24, 0.1) 0px 0px 0px 1px')).toEqual([]);
    const recipe = [
      'rgba(0, 0, 0, 0.55) 0px 0px 0px 1px',
      'rgba(0, 0, 0, 0.6) 0px 8px 18px -8px',
      'rgba(255, 255, 255, 0.15) 0px 0px 0px 1px inset',
      'rgba(255, 255, 255, 0.3) 0px -1px 0px 0px inset',
      'rgba(255, 255, 255, 0.09) 0px -18px 26px -18px inset',
      'rgba(0, 0, 0, 0.5) 0px 12px 18px -14px inset',
    ].join(', ');
    expect(badShadows(recipe)).toEqual([]);
    // a wider ring, a blurred ring or a hued hairline is still a glow
    expect(badShadows('rgba(0, 0, 0, 0.55) 0px 0px 0px 2px').map((s) => s.why)).toEqual(['outer glow']);
    expect(badShadows('rgba(0, 0, 0, 0.55) 0px 0px 4px 1px').map((s) => s.why)).toEqual(['outer glow']);
    expect(badShadows('rgba(0, 255, 171, 0.5) 0px 0px 0px 1px').map((s) => s.why)).toEqual(['outer glow']);
    expect(isHued([20, 20, 22])).toBe(false);
    expect(isHued([159, 198, 255])).toBe(true);
  });
});

describe('g-audit routes and source helpers', () => {
  it('matches routes with a query against their pathname', () => {
    expect(routeInReady('/creative-engine?engine=console', ['/creative-engine'])).toBe(true);
    expect(routeInReady('/funnel', ['/creative-engine'])).toBe(false);
    // :param entries match one concrete segment (the guard reaches /agents/setup/<id>)
    expect(routeInReady('/agents/setup/drake', ['/agents/setup/:id'])).toBe(true);
    expect(routeInReady('/agents/setup', ['/agents/setup/:id'])).toBe(false);
    expect(routeInReady('/agents/setup/a/b', ['/agents/setup/:id'])).toBe(false);
    expect(routeSlug('/')).toBe('home');
    expect(routeSlug('/creative-engine?engine=console')).toBe('creative-engine_engine_console');
  });

  it('blanks comments and keeps strings and line numbers', () => {
    const src = 'a // ae-chip\n/* c-btn\n */ "ae-btn // not a comment"';
    const out = stripComments(src);
    expect(out.split('\n')).toHaveLength(3);
    expect(out).not.toContain('ae-chip');
    expect(out).not.toContain('c-btn\n');
    expect(out).toContain('"ae-btn // not a comment"');
  });
});
