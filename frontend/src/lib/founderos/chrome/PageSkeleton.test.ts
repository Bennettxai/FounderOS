// FounderOS v1 components/PageSkeleton.tsx: the instant-paint shell a click lands
// on while the next view loads. Generic on purpose, in the Brand Deals slab
// shape: title row, a 2fr/1fr hero with a volume card's three meter tracks,
// a row of three, then one wide block.
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import PageSkeleton from './PageSkeleton.svelte';

describe('PageSkeleton', () => {
	it('is the slab shape: title row, 2fr/1fr hero, a row of three, one wide block', () => {
		const { container } = render(PageSkeleton);
		const slab = container.querySelector('[data-part="page-skeleton"]')!;
		expect(slab.classList.contains('bn-slab')).toBe(true);
		expect(slab.getAttribute('aria-busy')).toBe('true');
		const [title, hero, three, wide] = [...slab.children];
		expect(title.className).toMatch(/mb-7 flex items-end justify-between gap-4/);
		expect(title.querySelectorAll('[data-skel]')).toHaveLength(4);
		expect(hero.className).toMatch(/grid grid-cols-\[2fr_1fr\] gap-6 max-\[1200px\]:grid-cols-1/);
		expect(three.className).toMatch(/mt-6 grid grid-cols-3 gap-6 max-\[1200px\]:grid-cols-1/);
		expect(three.children).toHaveLength(3);
		expect(wide.className).toMatch(/mt-6 h-48/);
	});

	it('the volume card outline carries three meter tracks at 70 / 45 / 85%', () => {
		const { container } = render(PageSkeleton);
		const meters = [...container.querySelectorAll('[data-skel-meter]')];
		expect(meters).toHaveLength(3);
		expect(meters.map((m) => (m.querySelector('[data-part="fill"]') as HTMLElement).style.width)).toEqual(['70%', '45%', '85%']);
	});

	it('every block pulses on a hairline border at the tile radius', () => {
		const { container } = render(PageSkeleton);
		for (const b of container.querySelectorAll('[data-skel]')) expect(b.className).toMatch(/bn-skel-block animate-pulse/);
	});
});
