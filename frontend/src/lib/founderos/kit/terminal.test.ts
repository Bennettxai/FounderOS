import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Badge from './Badge.svelte';
import Dot from './Dot.svelte';
import Kbd from './Kbd.svelte';
import Label from './Label.svelte';
import PageHeader from './PageHeader.svelte';
import SectionHead from './SectionHead.svelte';
import Spark from './Spark.svelte';
import { snip } from './test-utils';

const css = readFileSync(resolve(__dirname, 'kit.css'), 'utf8');

describe('Dot', () => {
	it('maps the state to an LED class', () => {
		const { container } = render(Dot, { state: 'connected' });
		const dot = container.querySelector('.bn-dot')!;
		expect(dot.classList.contains('ok')).toBe(true);
		expect(dot.getAttribute('data-state')).toBe('ok');
	});
	it('warn / err / off states', () => {
		for (const [state, cls] of [
			['idle', 'warn'],
			['error', 'err'],
			['not_configured', 'off'],
			['whatever', 'off']
		]) {
			const { container } = render(Dot, { state });
			expect(container.querySelector('.bn-dot')!.classList.contains(cls), state).toBe(true);
		}
	});
	it('only an ok dot blinks', () => {
		const ok = render(Dot, { state: 'ok', pulse: true }).container.querySelector('.bn-dot')!;
		expect(ok.classList.contains('pulse')).toBe(true);
		const err = render(Dot, { state: 'error', pulse: true }).container.querySelector('.bn-dot')!;
		expect(err.classList.contains('pulse')).toBe(false);
	});
	it('css: round LED (FounderOS v1 premium pass), blinks on steps(), stops under reduced motion', () => {
		expect(css).toMatch(/\.bn-dot \{[^}]*border-radius: 50%/);
		expect(css).toMatch(/\.bn-dot\.ok\.pulse \{[^}]*animation: bn-led-blink [^;]*steps\(1\)/);
		expect(css).toMatch(/@media \(prefers-reduced-motion: reduce\) \{[^@]*\.bn-dot\.ok\.pulse \{\s*animation: none/);
	});
});

describe('Badge', () => {
	it('renders children in a tone', () => {
		const { container, getByText } = render(Badge, { tone: 'ok', children: snip('live') });
		expect(getByText('live')).toBeTruthy();
		expect(container.querySelector('[data-tone="ok"]')).toBeTruthy();
	});
	it('is a pill, as in FounderOS v1 terminal.tsx', () => {
		const { container } = render(Badge, { children: snip('x') });
		expect(container.querySelector('[data-tone]')!.className).toContain('rounded-full');
	});
	it('defaults to the neutral tone; ghost is dashed', () => {
		const { container } = render(Badge, { ghost: true, children: snip('x') });
		const b = container.querySelector('[data-tone]')!;
		expect(b.getAttribute('data-tone')).toBe('default');
		expect(b.classList.contains('border-dashed')).toBe(true);
	});
});

describe('Label', () => {
	it('mono section label: 10px/700/.26em uppercase, optional count and hairline rule', () => {
		const { container, getByText } = render(Label, { count: 12, rule: true, children: snip('Agents') });
		const el = container.querySelector('[data-part="label"]')!;
		expect(el.className).toContain('text-[10px]');
		expect(el.className).toContain('font-bold');
		expect(el.className).toContain('tracking-[0.26em]');
		expect(el.className).toContain('uppercase');
		expect(getByText('Agents')).toBeTruthy();
		expect(getByText('12')).toBeTruthy();
		expect(container.querySelector('[data-part="rule"]')).toBeTruthy();
	});
	it('no count and no rule unless asked', () => {
		const { container } = render(Label, { children: snip('x') });
		expect(container.querySelector('[data-part="count"]')).toBeNull();
		expect(container.querySelector('[data-part="rule"]')).toBeNull();
	});
});

describe('SectionHead', () => {
	it('label with a rule, a link arrow when both link and href are given, right controls', () => {
		const { container, getByText } = render(SectionHead, {
			label: 'Connections',
			count: '17/24',
			link: 'all',
			href: '/os/integrations',
			right: snip('btn')
		});
		expect(getByText('Connections')).toBeTruthy();
		expect(getByText('17/24')).toBeTruthy();
		const a = container.querySelector('a')!;
		expect(a.getAttribute('href')).toBe('/os/integrations');
		expect(a.textContent).toBe('all →');
		expect(getByText('btn')).toBeTruthy();
		expect(container.querySelector('[data-part="rule"]')).toBeTruthy();
	});
	it('no link without an href', () => {
		const { container } = render(SectionHead, { label: 'x', link: 'all' });
		expect(container.querySelector('a')).toBeNull();
	});
});

describe('Kbd', () => {
	it('renders a kbd', () => {
		const { container } = render(Kbd, { children: snip('⌘K') });
		expect(container.querySelector('kbd')!.textContent).toBe('⌘K');
	});
	it('css: a 5px keycap with a 2px floor; the compact palette key is 16px tall at 4px', () => {
		expect(css).toMatch(/\.bn-kbd \{[^}]*border-radius: var\(--bn-r-chip\)/);
		expect(css).toMatch(/\.bn-kbd \{[^}]*border-bottom-width: 2px/);
		const small = render(Kbd, { size: 'sm', children: snip('esc') }).container.querySelector('kbd')!;
		expect(small.classList.contains('bn-kbd-sm')).toBe(true);
		expect(css).toMatch(/\.bn-kbd-sm \{[^}]*height: 16px[^}]*border-radius: 4px/);
	});
});

describe('Spark', () => {
	it('draws an accent polyline with a 10% fill over the series', () => {
		const { container } = render(Spark, { data: [1, 3, 2], w: 60, h: 20 });
		const svg = container.querySelector('svg')!;
		expect(svg.getAttribute('viewBox')).toBe('0 0 60 20');
		expect(container.querySelector('polyline')!.getAttribute('stroke')).toBe('var(--bn-accent)');
		expect(container.querySelector('polygon')!.getAttribute('opacity')).toBe('0.1');
	});
	it('under two samples it draws nothing', () => {
		expect(render(Spark, { data: [5] }).container.querySelector('svg')).toBeNull();
	});
});

describe('PageHeader', () => {
	it('title is 25px/700 uppercase with .06em tracking', () => {
		const { container } = render(PageHeader, { title: 'Operator Console' });
		const h1 = container.querySelector('h1')!;
		expect(h1.textContent).toBe('Operator Console');
		for (const c of ['text-[25px]', 'font-bold', 'uppercase', 'tracking-[0.06em]']) expect(h1.className).toContain(c);
	});
	it('eyebrow is 9.5px/.32em and carries the // prefix', () => {
		const { container } = render(PageHeader, { title: 'x', eyebrow: 'operate' });
		const eb = container.querySelector('.bn-eyebrow')!;
		expect(eb.textContent).toContain('operate');
		expect(eb.className).toContain('text-[9.5px]');
		expect(eb.className).toContain('tracking-[0.32em]');
		expect(css).toMatch(/\.bn-eyebrow::before \{[^}]*content: '\/\/'/);
	});
	it('no eyebrow element without an eyebrow', () => {
		expect(render(PageHeader, { title: 'x' }).container.querySelector('.bn-eyebrow')).toBeNull();
	});
	it('caret blinks after the title, and stops under reduced motion', () => {
		const { container } = render(PageHeader, { title: 'x', caret: true });
		expect(container.querySelector('h1')!.classList.contains('bn-caret')).toBe(true);
		expect(css).toMatch(/@media \(prefers-reduced-motion: reduce\) \{[^@]*\.bn-caret::after \{\s*animation: none/);
	});
	it('right slot: compact by default, stretched when rightWide', () => {
		const compact = render(PageHeader, { title: 'x', right: snip('R') }).container;
		expect(compact.querySelector('[data-part="right"]')!.className).toContain('shrink-0');
		expect(compact.querySelector('header')!.className).toContain('items-end');
		const wide = render(PageHeader, { title: 'x', right: snip('R'), rightWide: true }).container;
		expect(wide.querySelector('[data-part="right"]')!.className).toContain('flex-1');
		expect(wide.querySelector('header')!.className).toContain('items-start');
	});
});
