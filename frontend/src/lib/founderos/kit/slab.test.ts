import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render, waitFor } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import BigStat from './BigStat.svelte';
import Chip from './Chip.svelte';
import DotMatrix from './DotMatrix.svelte';
import InsightCard from './InsightCard.svelte';
import MeterStack from './MeterStack.svelte';
import Slab from './Slab.svelte';
import SlabCard from './SlabCard.svelte';
import SlabTitle from './SlabTitle.svelte';
import StepLine from './StepLine.svelte';
import { snip } from './test-utils';

const kitDir = __dirname;
const css = readFileSync(resolve(kitDir, 'kit.css'), 'utf8');

function reducedMotion(on: boolean) {
	return vi.spyOn(window, 'matchMedia').mockImplementation(
		(q: string) => ({ matches: on && q.includes('reduce'), media: q, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList
	);
}

afterEach(() => vi.restoreAllMocks());

describe('Slab', () => {
	it('is the floating surface every page sits on', () => {
		const { container, getByText } = render(Slab, { children: snip('body') });
		expect(container.querySelector('.bn-slab')).toBeTruthy();
		expect(getByText('body')).toBeTruthy();
	});
	it('css: the floating 28px surface on bg-2 with its drop (FounderOS v1 .os-slab)', () => {
		const rule = css.slice(css.indexOf('.bn-slab {'), css.indexOf('}', css.indexOf('.bn-slab {')));
		expect(rule).toContain('border: 1px solid var(--bn-border)');
		expect(rule).toContain('border-radius: var(--bn-r-slab)');
		expect(rule).toContain('background: var(--bn-bg-2)');
		expect(rule).toContain('box-shadow: var(--bn-shadow-slab)');
		expect(rule).toContain('padding: clamp(16px, 2.2vw, 28px)');
	});
	it('css: inside the slab, data-part cards round to 12px and their heads read sentence-case', () => {
		expect(css).toMatch(/\.bn-slab \[data-part='card'\] \{[^}]*border-radius: 12px/);
		expect(css).toMatch(/\.bn-slab \[data-part='card-head'\] \[data-part='label'\] \{[^}]*font-size: 19px[^}]*text-transform: none/);
		expect(css).toMatch(/\.bn-slab \[data-part='card-head'\] :is\(a, button\) \{[^}]*border-radius: 999px/);
		expect(css).toMatch(/\.bn-slab \[data-part='tile'\] \{[^}]*border-radius: 12px/);
		expect(css).toMatch(/\.bn-slab \[data-part='tile'\] \[data-part='label'\] \{[^}]*font-size: 13px[^}]*text-transform: none/);
		expect(css).toMatch(/\.bn-slab \[data-part='input'\] \{[^}]*border-radius: 12px/);
		expect(css).toMatch(/\.bn-slab \[data-part='row'\] button \{[^}]*border-radius: 9px/);
	});
});

describe('SlabTitle', () => {
	it('eyebrow with //, a 46px h1, a mono meta line, actions on the right', () => {
		const { container, getByText } = render(SlabTitle, { eyebrow: 'money · stripe', title: 'Finances', meta: snip('$12 today'), right: snip('go') });
		expect(getByText('// money · stripe')).toBeTruthy();
		const h1 = container.querySelector('h1')!;
		expect(h1.textContent).toBe('Finances');
		expect(h1.className).toContain('text-[46px]');
		expect(getByText('$12 today')).toBeTruthy();
		expect(getByText('go')).toBeTruthy();
		expect(container.querySelector('.bn-rise')!.getAttribute('style')).toContain('--rise-i: 0');
	});
});

describe('SlabCard', () => {
	it('css: a 12px card (slab.tsx rounded-[12px])', () => {
		expect(css).toMatch(/\.bn-card \{[^}]*border-radius: var\(--bn-r-tile\)/);
	});
	it('rises on its stagger, lifts on hover, 19px title, sub and action', () => {
		const { container, getByText } = render(SlabCard, { title: 'Deal Volume', sub: snip('4 open'), i: 3, action: snip('open'), children: snip('body') });
		const card = container.querySelector('.bn-rise')!;
		expect(card.classList.contains('bn-rise-card')).toBe(true);
		expect(card.getAttribute('style')).toContain('--rise-i: 3');
		const h2 = container.querySelector('h2')!;
		expect(h2.textContent).toBe('Deal Volume');
		expect(h2.className).toContain('text-[19px]');
		for (const t of ['4 open', 'open', 'body']) expect(getByText(t)).toBeTruthy();
	});
	it('without a title there is no head row', () => {
		const { container } = render(SlabCard, { children: snip('x') });
		expect(container.querySelector('h2')).toBeNull();
		expect(container.querySelector('.bn-rise')!.getAttribute('style')).toContain('--rise-i: 1');
	});
	it('css: the rise backfills its first frame only and stops under reduced motion', () => {
		expect(css).toMatch(/\.bn-rise \{[^}]*animation: bn-rise 0\.6s var\(--bn-ease\) backwards/);
		expect(css).toMatch(/@media \(prefers-reduced-motion: reduce\) \{[^@]*\.bn-rise[^{]*\{\s*animation: none !important/);
	});
});

describe('Chip', () => {
	it('a tone adds a round status dot; no tone is a plain pill', () => {
		const toned = render(Chip, { tone: 'err', children: snip('2 failed') }).container;
		expect(toned.querySelector('[data-tone="err"]')!.className).toContain('rounded-full');
		expect(toned.querySelector('.bn-chip')!.className).toContain('rounded-full');
		const plain = render(Chip, { children: snip('read-only') }).container;
		expect(plain.querySelector('[data-tone]')).toBeNull();
	});
});

describe('BigStat', () => {
	// The number and its unit touch: only the unit's ml-1.5 separates them. A
	// stray text space at 50px is ~30px ("2,060   clicks", wider than prod).
	it('puts no whitespace between the headline and its unit', () => {
		const { container } = render(BigStat, { display: '2,060', unit: 'clicks' });
		const v = container.querySelector('[data-part="bigstat-value"]')!;
		expect(v.textContent).toBe('2,060clicks');
		const unknown = render(BigStat, { value: null, unit: 'live' }).container.querySelector('[data-part="bigstat-value"]')!;
		expect(unknown.textContent).toBe('unknownlive');
	});

	it('the 50px headline with dot chips in status colors and a caption', () => {
		reducedMotion(true);
		const { container, getByText } = render(BigStat, {
			value: 1200,
			kind: 'usd',
			chips: [
				{ tone: 'ok', text: '$800 paid' },
				{ tone: 'err', text: '2 failed' }
			],
			caption: 'open pipeline across 4 deals'
		});
		expect(container.querySelector('[data-part="bigstat-value"]')!.className).toContain('text-[50px]');
		expect(getByText('$1,200')).toBeTruthy();
		expect(container.querySelector('[data-tone="ok"]')).toBeTruthy();
		expect(container.querySelector('[data-tone="err"]')).toBeTruthy();
		expect(getByText('open pipeline across 4 deals')).toBeTruthy();
	});
	it('a preformatted display wins, with a unit', () => {
		const { getByText } = render(BigStat, { display: '4.2M', unit: 'tokens' });
		expect(getByText('4.2M')).toBeTruthy();
		expect(getByText('tokens')).toBeTruthy();
	});
	it('size 30 is the small headline', () => {
		reducedMotion(true);
		const { container } = render(BigStat, { value: 3, size: 30 });
		expect(container.querySelector('[data-part="bigstat-value"]')!.className).toContain('text-[30px]');
	});
	it('no value and no display reads unknown, never 0', () => {
		const { container } = render(BigStat, { value: null });
		const v = container.querySelector('[data-part="bigstat-value"]')!;
		expect(v.textContent).toContain('unknown');
		expect(v.textContent).not.toContain('0');
	});
	it('counts up from 0 and lands exactly on the target', async () => {
		reducedMotion(false);
		// Drive animation frames with fake timers: real rAF timing is not
		// guaranteed on a loaded CI runner.
		vi.useFakeTimers({ toFake: ['requestAnimationFrame', 'cancelAnimationFrame', 'performance', 'Date'] });
		try {
			const { container } = render(BigStat, { value: 47501 });
			const v = () => container.querySelector('[data-part="bigstat-value"]')!.textContent!.trim();
			expect(v()).toBe('0');
			await vi.advanceTimersByTimeAsync(5000);
			expect(v()).toBe('47,501');
		} finally {
			vi.useRealTimers();
		}
	});
	it('reduced motion lands instantly', () => {
		reducedMotion(true);
		const { container } = render(BigStat, { value: 47501 });
		expect(container.querySelector('[data-part="bigstat-value"]')!.textContent!.trim()).toBe('47,501');
	});
});

describe('MeterStack', () => {
	const meters = [
		{ label: 'A', frac: 0.5, display: '50%', hue: 'var(--bn-accent)' },
		{ label: 'B', frac: 0.2, display: '20%', hue: 'var(--bn-warn)' }
	];
	it('one sweeping meter per row, staggered 150ms from 500ms, with a foot note', () => {
		const { container, getByText } = render(MeterStack, { meters, foot: 'open = talks + production' });
		const fills = [...container.querySelectorAll('.bn-vol-fill')] as HTMLElement[];
		expect(fills).toHaveLength(2);
		expect(fills[0].getAttribute('style')).toContain('width: 50%');
		expect(fills[0].getAttribute('style')).toContain('bn-vol-meter-in');
		expect(fills[0].getAttribute('style')).toContain('500ms');
		expect(fills[1].getAttribute('style')).toContain('650ms');
		expect(getByText('open = talks + production')).toBeTruthy();
	});
	it('a null fraction reads "unknown" and draws no bar', () => {
		const { container, getByText } = render(MeterStack, {
			meters: [{ label: 'Stripe', frac: null, display: '$0', hue: 'var(--bn-accent)' }]
		});
		expect(getByText('unknown')).toBeTruthy();
		expect(container.textContent).not.toContain('$0');
		expect(container.querySelector('.bn-vol-fill')).toBeNull();
		expect(container.querySelector('[data-unknown]')).toBeTruthy();
	});
	it('NaN is unknown too; a real zero still draws a 2% sliver', () => {
		const { container, getAllByText } = render(MeterStack, {
			meters: [
				{ label: 'nan', frac: Number.NaN, display: 'x', hue: 'var(--bn-accent)' },
				{ label: 'zero', frac: 0, display: '0', hue: 'var(--bn-accent)' }
			]
		});
		expect(getAllByText('unknown')).toHaveLength(1);
		const fills = container.querySelectorAll('.bn-vol-fill');
		expect(fills).toHaveLength(1);
		expect(fills[0].getAttribute('style')).toContain('width: 2%');
	});
	it('dense: the tighter rhythm (slab.tsx MeterStack dense)', () => {
		const loose = render(MeterStack, { meters, foot: 'f' }).container;
		expect(loose.firstElementChild!.className).toContain('mt-5');
		expect(loose.querySelector('[data-part="meters"]')!.className).toContain('gap-6 pt-5');
		expect(loose.querySelector('[data-part="foot"]')!.className).toContain('mt-5 pt-3');
		const dense = render(MeterStack, { meters, foot: 'f', dense: true }).container;
		expect(dense.firstElementChild!.className).toContain('mt-4');
		expect(dense.querySelector('[data-part="meters"]')!.className).toContain('gap-2.5 pt-3.5');
		expect(dense.querySelector('[data-part="foot"]')!.className).toContain('mt-3 pt-2.5');
	});
	it('the track and the fill are round (VolumeMeter rounded-full)', () => {
		const { container } = render(MeterStack, { meters });
		expect(container.querySelector('.bn-vol-track')!.className).toContain('rounded-full');
		expect(container.querySelector('.bn-vol-fill')!.className).toContain('rounded-full');
	});
	it('nothing to show says so instead of drawing empty bars', () => {
		expect(render(MeterStack, { meters: [], empty: 'no data yet' }).getByText('no data yet')).toBeTruthy();
	});
	it('css: meters land full under reduced motion', () => {
		expect(css).toMatch(/@media \(prefers-reduced-motion: reduce\) \{\s*\.bn-vol-fill \{\s*animation: none !important;/);
	});
});

describe('InsightCard', () => {
	it('badge, the big number, headline, body and 8 ticks lit by the fraction', () => {
		reducedMotion(true);
		const { container, getByText } = render(InsightCard, { badge: 'Needs you this week', value: 3, headline: 'follow-ups due', body: 'Acme · Beta', frac: 0.5, i: 5 });
		expect(getByText('Needs you this week')).toBeTruthy();
		expect(getByText('follow-ups due')).toBeTruthy();
		expect(getByText('Acme · Beta')).toBeTruthy();
		expect(container.querySelector('[data-part="insight-value"]')!.textContent!.trim()).toBe('3');
		const ticks = container.querySelectorAll('[data-tick]');
		expect(ticks).toHaveLength(8);
		expect(container.querySelectorAll('[data-tick="on"]')).toHaveLength(4);
		const card = container.querySelector('.bn-rise')!;
		expect(card.getAttribute('style')).toContain('bn-drift');
		expect(card.getAttribute('style')).toContain('--rise-i: 5');
	});
	it('css: 12px card, 36px rotated shard, round pill and ticks, white ink, glow off the brain tokens', () => {
		reducedMotion(true);
		const { container } = render(InsightCard, { badge: 'b', value: 1, headline: 'h', frac: 1 });
		const card = container.querySelector('.bn-insight')!;
		expect(card.className).toContain('rounded-[12px]');
		const shard = container.querySelector('.bn-insight-shard')!;
		for (const c of ['rotate-[24deg]', 'rounded-[36px]', 'h-56', 'w-56']) expect(shard.className).toContain(c);
		expect(container.querySelector('.bn-insight-pill')!.className).toContain('rounded-full');
		expect(container.querySelector('[data-tick]')!.className).toContain('rounded-full');
		expect(container.querySelector('[data-part="insight-value"]')!.className).toContain('text-[64px]');
		expect(css).toMatch(/\.bn-insight \{[^}]*--bn-glow-a: var\(--bn-brain-1\)[^}]*--bn-glow-b: var\(--bn-brain-2\)[^}]*--bn-glow-c: var\(--bn-accent\)/);
		expect(css).toMatch(/\.bn-insight \{[^}]*color: var\(--bn-insight-ink\)/);
		expect(css).toMatch(/\.bn-insight \{[^}]*border: 1px solid transparent/);
	});
	it('compact: badge and a 32px number on one line, tighter padding, a 24px shard', () => {
		reducedMotion(true);
		const { container } = render(InsightCard, { badge: 'b', value: 7, headline: 'h', body: 'x', compact: true });
		expect(container.querySelector('[data-part="insight-value"]')!.className).toContain('text-[32px]');
		expect(container.querySelector('[data-part="insight-value"]')!.className).not.toContain('text-[64px]');
		const shard = container.querySelector('.bn-insight-shard')!;
		for (const c of ['rounded-[24px]', 'h-32', 'w-32']) expect(shard.className).toContain(c);
		expect(container.querySelector('[data-part="insight-inner"]')!.className).toContain('px-4 py-3');
		expect(container.querySelector('[data-part="insight-top"]')!.className).toContain('justify-between');
	});
	it('no value and no display reads unknown', () => {
		const { container } = render(InsightCard, { badge: 'b', headline: 'h' });
		expect(container.querySelector('[data-part="insight-value"]')!.textContent).toContain('unknown');
	});
	it('the gradient is built from theme tokens only', () => {
		const src = readFileSync(resolve(kitDir, 'InsightCard.svelte'), 'utf8');
		expect(src).not.toMatch(/#[0-9a-f]{3,8}\b|rgba?\(|\bwhite\b/i);
	});
});

describe('StepLine', () => {
	const series = [
		{ label: 'Sep 1', count: 1 },
		{ label: 'Sep 2', count: 4 },
		{ label: 'Sep 3', count: 0 }
	];
	it('draws itself, pins the peak, and labels both ends', () => {
		const { container, getByText } = render(StepLine, { series, hue: 'var(--bn-accent)', unit: ' runs' });
		const line = container.querySelector('path[data-part="line"]')!;
		expect(line.getAttribute('pathLength')).toBe('1');
		expect(line.getAttribute('style')).toContain('bn-draw');
		expect(container.querySelector('[data-part="peak"]')!.textContent).toMatch(/4 runs\s+on Sep 2/);
		expect(getByText('Sep 1')).toBeTruthy();
		expect(getByText('Sep 3')).toBeTruthy();
	});
	it('the peak callout is a pill', () => {
		const { container } = render(StepLine, { series, hue: 'red' });
		expect(container.querySelector('[data-part="peak"]')!.className).toContain('rounded-full');
	});
	it('height sets a flatter box and scales the headroom (slab-charts.tsx height)', () => {
		const tall = render(StepLine, { series, hue: 'red' }).container;
		expect(tall.querySelector('svg')!.getAttribute('viewBox')).toBe('0 0 600 150');
		const flat = render(StepLine, { series, hue: 'red', height: 56 }).container;
		expect(flat.querySelector('svg')!.getAttribute('viewBox')).toBe('0 0 600 56');
		// peak at count 4 of max 4: y = H - round(H*.09) - (H - round(H*.4)) = 56 - 5 - 34 = 17
		expect(flat.querySelector('circle')!.getAttribute('cy')).toBe('17');
	});
	it('each instance gets its own pattern ids', () => {
		const a = render(StepLine, { series, hue: 'red' }).container.querySelector('pattern')!.id;
		const b = render(StepLine, { series, hue: 'red' }).container.querySelector('pattern')!.id;
		expect(a).not.toBe(b);
	});
	it('no activity says so', () => {
		expect(render(StepLine, { series: [{ label: 'a', count: 0 }], hue: 'red', empty: 'quiet month' }).getByText('quiet month')).toBeTruthy();
		expect(render(StepLine, { series: [], hue: 'red' }).getByText('No activity in this window.')).toBeTruthy();
	});
});

describe('DotMatrix', () => {
	it('a column of dots per bucket, tallest (6) at the max', () => {
		const { container, getByText } = render(DotMatrix, {
			cols: [
				{ label: '<1k', count: 1 },
				{ label: '1k+', count: 3 }
			],
			hue: 'var(--bn-accent)'
		});
		expect(container.querySelectorAll('[data-dot]')).toHaveLength(2 + 6);
		expect(getByText('<1k')).toBeTruthy();
	});
	it('dots are round', () => {
		const { container } = render(DotMatrix, { cols: [{ label: 'a', count: 1 }], hue: 'red' });
		expect(container.querySelector('[data-dot]')!.className).toContain('rounded-full');
	});
	it('an empty bucket shows one ghost dot, not a lit one', () => {
		const { container } = render(DotMatrix, { cols: [{ label: 'a', count: 0 }, { label: 'b', count: 2 }], hue: 'red' });
		expect(container.querySelectorAll('[data-dot]')).toHaveLength(6);
		expect(container.querySelectorAll('[data-ghost]')).toHaveLength(1);
	});
});

describe('kit uses theme tokens only', () => {
	it('no hard-coded colors in any kit component or the kit css', () => {
		const files = ['kit.css', 'Dot.svelte', 'Badge.svelte', 'Label.svelte', 'SectionHead.svelte', 'Kbd.svelte', 'Spark.svelte', 'PageHeader.svelte', 'Slab.svelte', 'SlabTitle.svelte', 'SlabCard.svelte', 'Chip.svelte', 'BigStat.svelte', 'CountUp.svelte', 'MeterStack.svelte', 'VolumeMeter.svelte', 'InsightCard.svelte', 'StepLine.svelte', 'DotMatrix.svelte', 'Rise.svelte', 'Pressable.svelte', 'ToggleChip.svelte', 'Spotlight.svelte', 'PageSpotlight.svelte', 'slab-classes.ts', 'lens.ts'];
		for (const f of files) {
			const src = readFileSync(resolve(kitDir, f), 'utf8').replace(/\/\*[\s\S]*?\*\/|<!--[\s\S]*?-->/g, '');
			expect(src, f).not.toMatch(/#[0-9a-f]{3,8}\b(?![\w-])|rgba?\(|hsla?\(|\b(white|black)\b(?!-space)/i);
			expect(src, f).not.toMatch(/var\(--(?!bn-|rise-i\b)[\w-]+\)/);
		}
	});
});
