import { describe, expect, test } from 'vitest';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { createElement as h } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { Slab, SlabTitle, SlabCard, BigStat, Chip, MeterStack, InsightCard, insightTicks } from '@/components/slab';
import { StepLine, DotMatrix } from '@/components/slab-charts';

/**
 * The Brand Deals slab as a kit (Alex, 2026-09-24: "rebuild the entire OS
 * in that light"). Every page wears the same floating slab, the same 19px
 * card titles, the same 50px count-up headline with dot chips, the same
 * sweeping hatched meters, and at most ONE gradient insight card.
 */
const read = (rel: string) => readFileSync(path.join(process.cwd(), rel), 'utf8');
const html = (el: ReturnType<typeof h>) => renderToStaticMarkup(el);

describe('slab kit', () => {
  test('Slab is the floating surface every page sits on', () => {
    const out = html(h(Slab, null, 'x'));
    expect(out).toContain('class="os-slab');
  });

  test('SlabTitle: eyebrow, the 46px mixed-case title, a mono meta line, actions on the right', () => {
    const out = html(h(SlabTitle, { eyebrow: 'money · stripe', title: 'Finances', meta: '$12 today', right: h('button', null, 'go') }));
    expect(out).toContain('// money · stripe');
    expect(out).toMatch(/<h1[^>]*text-\[46px\][^>]*>Finances<\/h1>/);
    expect(out).toContain('$12 today');
    expect(out).toContain('<button>go</button>');
  });

  test('SlabCard: rises on its stagger, lifts on hover, 19px title, optional action', () => {
    const out = html(h(SlabCard, { title: 'Deal Volume', i: 3, action: h('a', { href: '/x' }, 'open') }, 'body'));
    expect(out).toContain('rise');
    expect(out).toContain('rise-card');
    expect(out).toContain('--rise-i:3');
    expect(out).toMatch(/<h2[^>]*text-\[19px\][^>]*>Deal Volume<\/h2>/);
    expect(out).toContain('href="/x"');
    expect(out).toContain('body');
  });

  test('BigStat: the 50px headline, dot chips in status colors, a caption', () => {
    const out = html(
      h(BigStat, {
        value: 1200,
        kind: 'usd',
        chips: [
          { tone: 'ok', text: '$800 paid' },
          { tone: 'err', text: '2 failed' },
        ],
        caption: 'open pipeline across 4 deals',
      }),
    );
    expect(out).toContain('text-[50px]');
    expect(out).toContain('bg-os-ok');
    expect(out).toContain('bg-os-err');
    expect(out).toContain('$800 paid');
    expect(out).toContain('open pipeline across 4 deals');
  });

  test('BigStat accepts a preformatted display for values CountUp cannot format', () => {
    const out = html(h(BigStat, { display: '4.2M', caption: 'tokens' }));
    expect(out).toContain('4.2M');
  });

  test('Chip without a tone is a plain pill', () => {
    expect(html(h(Chip, null, 'read-only'))).toContain('rounded-full');
  });

  test('MeterStack: one sweeping meter per row, staggered 150ms, with a foot note', () => {
    const out = html(
      h(MeterStack, {
        meters: [
          { label: 'A', frac: 0.5, display: '50%', hue: 'var(--accent)' },
          { label: 'B', frac: 0.2, display: '20%', hue: 'var(--warn)' },
        ],
        foot: 'open = talks + production',
      }),
    );
    expect((out.match(/vol-meter-in/g) ?? []).length).toBe(2);
    expect(out).toContain('500ms');
    expect(out).toContain('650ms');
    expect(out).toContain('open = talks + production');
  });

  test('MeterStack with nothing to show says so instead of drawing empty bars', () => {
    expect(html(h(MeterStack, { meters: [], empty: 'no data yet' }))).toContain('no data yet');
  });

  test('InsightCard: the gradient, the grain, the big number, 8 progress ticks', () => {
    const out = html(h(InsightCard, { badge: 'Needs you this week', value: 3, headline: '3 follow-ups due', body: 'Acme · Beta', frac: 0.5, i: 5 }));
    expect(out).toContain('--tile-glow-a');
    expect(out).toContain('os-drift');
    expect(out).toContain('Needs you this week');
    expect(out).toContain('3 follow-ups due');
    expect(out).toContain('text-[64px]');
    expect((out.match(/data-tick/g) ?? []).length).toBe(8);
  });

  test('insightTicks lights a real fraction of 8 and never overflows', () => {
    expect(insightTicks(0)).toBe(0);
    expect(insightTicks(0.5)).toBe(4);
    expect(insightTicks(3)).toBe(8);
    expect(insightTicks(Number.NaN)).toBe(0);
  });
});

describe('slab charts', () => {
  test('StepLine draws itself, marks the peak, and labels both ends', () => {
    const out = html(h(StepLine, { series: [{ label: 'Sep 1', count: 1 }, { label: 'Sep 2', count: 4 }, { label: 'Sep 3', count: 0 }], hue: 'var(--ramp-1)' }));
    expect(out).toContain('os-draw');
    expect(out).toContain('pathLength="1"');
    expect(out).toContain('on Sep 2');
    expect(out).toContain('Sep 1');
    expect(out).toContain('Sep 3');
  });

  test('StepLine with no activity says so', () => {
    expect(html(h(StepLine, { series: [{ label: 'a', count: 0 }], hue: 'red', empty: 'quiet month' }))).toContain('quiet month');
  });

  test('DotMatrix: a column of dots per bucket, tallest at the max', () => {
    const out = html(h(DotMatrix, { cols: [{ label: '<1k', count: 1 }, { label: '1k+', count: 3 }], hue: 'var(--ramp-1)' }));
    expect((out.match(/data-dot/g) ?? []).length).toBe(2 + 6);
    expect(out).toContain('&lt;1k');
  });

  test('Brand Deals draws the same charts it lent the kit', () => {
    const board = read('components/brand-deals/DealBoard.tsx');
    expect(board).toContain("from '@/components/slab-charts'");
    expect(board).not.toMatch(/^function StepLine/m);
    expect(board).not.toMatch(/^function DotMatrix/m);
  });
});

describe('slab css', () => {
  const css = read('app/globals.css');
  test('the slab is one shared class, not a Home-only one', () => {
    expect(css).toMatch(/\.os-slab \{/);
    expect(css).not.toContain('.home-slab');
    expect(read('app/page.tsx')).toContain('os-slab');
  });
  test('the drift and fade the insight card and charts use are global keyframes', () => {
    expect(css).toMatch(/@keyframes os-drift/);
    expect(css).toMatch(/@keyframes os-fade/);
  });
});

describe('count-up formats for the slab headlines', () => {
  test('tokens and percents count up too, landing on the same text the pages printed', async () => {
    const { formatCount } = await import('@/components/CountUp');
    expect(formatCount('tokens', 68_219_000)).toBe('68.2M');
    expect(formatCount('tokens', 1_500)).toBe('1.5k');
    expect(formatCount('tokens', 2_100_000_000)).toBe('2.1B');
    expect(formatCount('tokens', 12)).toBe('12');
    expect(formatCount('pct', 56)).toBe('56%');
    // plain counts group their thousands like every other slab numeral
    expect(formatCount('int', 47501)).toBe('47,501');
    expect(formatCount('int', 12)).toBe('12');
  });

  test('BigStat and InsightCard take a count kind', () => {
    const out = renderToStaticMarkup(h(InsightCard, { badge: 'b', value: 56, kind: 'pct', headline: 'h' }));
    expect(out).toMatch(/>0%<\/div>/);
  });
});

describe('the insight card stays legible on the light colorways', () => {
  const css = read('app/globals.css');
  test('its base is a token, dark by default and dark again on light themes', () => {
    const out = renderToStaticMarkup(h(InsightCard, { badge: 'b', value: 1, headline: 'h' }));
    expect(out).toContain('var(--insight-base)');
    expect(css).toMatch(/--insight-base:\s*var\(--surface\)/);
    const light = css.slice(css.indexOf("/* Insight card on light colorways"));
    expect(light).toContain(":root[data-theme='light']");
    expect(light).toContain(":root[data-theme='mono-light']");
    expect(light).toMatch(/--insight-base:\s*color-mix\(in oklab, var\(--text\)/);
  });
  test('Brand Deals wears the same base', () => {
    expect(read('components/brand-deals/DealBoard.tsx')).toContain('var(--insight-base)');
  });
});

describe('count-ups on polled boards move from the last value, not from zero (review 2026-09-24)', () => {
  test('countFrame eases from where the number already is to the new target', async () => {
    const { countFrame } = await import('@/components/CountUp');
    expect(countFrame(1000, 1100, 0)).toBe(1000);
    expect(countFrame(1000, 1100, 1)).toBe(1100);
    const mid = countFrame(1000, 1100, 0.5);
    expect(mid).toBeGreaterThan(1000);
    expect(mid).toBeLessThan(1100);
    // counting down works too
    expect(countFrame(50, 20, 1)).toBe(20);
  });
  test('the hook remembers the value on screen and starts the next count there', () => {
    const src = read('components/CountUp.tsx');
    const hook = src.slice(src.indexOf('export function useCountUp'), src.indexOf('export type CountKind'));
    expect(hook).toMatch(/useRef/);
    expect(hook).toContain('countFrame(');
  });
});

describe('cards can lift once they have risen (review 2026-09-24)', () => {
  // `both` holds the last keyframe (transform: none) after the rise, and an
  // animation outranks the :hover rule, so the 1px lift never showed.
  test('the rise only backfills its first frame; it does not hold its last', () => {
    const css = read('app/globals.css');
    const rise = css.slice(css.indexOf('.rise {'), css.indexOf('}', css.indexOf('.rise {')));
    expect(rise).toMatch(/os-rise 0\.6s var\(--ease\) backwards/);
    expect(renderToStaticMarkup(h(InsightCard, { badge: 'b', value: 1, headline: 'h' }))).toContain('os-rise .6s var(--ease) calc(var(--rise-i) * 90ms) backwards');
    const board = read('components/brand-deals/DealBoard.tsx');
    expect(board).not.toMatch(/bd-rise [^`]*both/);
  });
});

describe('seen live on the mini (2026-09-24)', () => {
  test('meters land full under reduced motion instead of waiting on a sweep', () => {
    const out = renderToStaticMarkup(h(MeterStack, { meters: [{ label: 'A', frac: 0.5, display: '50%', hue: 'red' }] }));
    expect(out).toContain('vol-fill');
    const css = read('app/globals.css');
    expect(css).toMatch(/@media \(prefers-reduced-motion: reduce\) \{\s*\.vol-fill \{\s*animation: none !important;/);
  });
  test("Home's volume card keeps its own height when the Needs you list runs long", () => {
    const page = read('app/page.tsx');
    const card = page.slice(page.indexOf('title="Operating volume"'), page.indexOf('title="Operating volume"') + 120);
    expect(card).toContain('self-start');
    expect(card).not.toContain('self-stretch');
  });
});
