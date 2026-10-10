// The interaction layer (FounderOS v1 app/globals.css "Interaction layer" +
// "Slab motion", lib/hooks/useLens.ts, components/motion.tsx, Pressable.tsx,
// Spotlight.tsx, OsMark.tsx). Every class is bn- prefixed: kit.css is global
// once /os has loaded, and BusinessOS must never pick up a stray `.fill`.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import OsMark from './OsMark.svelte';
import PageSpotlight from './PageSpotlight.svelte';
import Pressable from './Pressable.svelte';
import Rise from './Rise.svelte';
import Spotlight from './Spotlight.svelte';
import ToggleChip from './ToggleChip.svelte';
import { LENS_PULL, installLens, lensVars } from './lens';
import { PILL, PILL_ACCENT, chipClass } from './slab-classes';
import { snip } from './test-utils';

const css = readFileSync(resolve(__dirname, 'kit.css'), 'utf8');

/** The body of the first rule whose selector list is exactly `sel`. */
function rule(sel: string): string {
	const at = css.indexOf(`${sel} {`);
	expect(at, `missing rule ${sel}`).toBeGreaterThanOrEqual(0);
	return css.slice(css.indexOf('{', at) + 1, css.indexOf('}', at));
}

/** Every reduced-motion block, concatenated. */
const reduced = [...css.matchAll(/@media \(prefers-reduced-motion: reduce\) \{([\s\S]*?)\n\}/g)].map((m) => m[1]).join('\n');

function reducedMotion(on: boolean) {
	return vi.spyOn(window, 'matchMedia').mockImplementation(
		(q: string) => ({ matches: on && q.includes('reduce'), media: q, addEventListener() {}, removeEventListener() {} }) as unknown as MediaQueryList
	);
}

afterEach(() => {
	vi.restoreAllMocks();
	document.documentElement.style.removeProperty('--bn-px');
	document.documentElement.style.removeProperty('--bn-py');
	document.body.innerHTML = '';
});

describe('motion tokens (globals.css :root)', () => {
	it('the house ease, the lens clock at 1.75x the base, press 200ms, panel 420ms, the spring', () => {
		const root = rule(':root');
		expect(root).toContain('--bn-ease: cubic-bezier(0.32, 0.72, 0, 1)');
		expect(root).toContain('--bn-ease-lens: cubic-bezier(0.22, 0.61, 0.36, 1)');
		expect(root).toContain('--bn-ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1)');
		expect(root).toContain('--bn-dur-press: 200ms');
		expect(root).toContain('--bn-dur: 360ms');
		expect(root).toContain('--bn-dur-lens: calc(var(--bn-dur) * 1.75)');
		expect(root).toContain('--bn-dur-panel: 420ms');
	});
});

describe('.bn-pressable: hover lens, press sink, focus ring', () => {
	it('eases six properties on the lens clock, radius never animated', () => {
		const r = rule('.bn-pressable');
		for (const p of ['transform', 'background-color', 'border-color', 'color', 'box-shadow', 'opacity']) {
			expect(r).toMatch(new RegExp(`${p} var\\(--bn-dur-lens\\) var\\(--bn-ease-lens\\)`));
		}
		expect(r).not.toContain('border-radius');
	});
	it('hover magnifies 1.05 (rows 1.02) with the magnetic pull and tilt, and lifts a shadow', () => {
		const h = rule('.bn-pressable:hover');
		expect(h).toContain('box-shadow: var(--bn-shadow-lift)');
		expect(h).toMatch(/perspective\(700px\) translate\(var\(--bn-lx, 0px\), var\(--bn-ly, 0px\)\)\s*rotateX\(var\(--bn-rx, 0deg\)\) rotateY\(var\(--bn-ry, 0deg\)\) translateY\(-1px\) scale\(1\.05\)/);
		expect(rule(".bn-pressable[data-lens='r']:hover")).toContain('scale(1.02)');
	});
	it('press sinks 1px at .97 (rows .995) on the 200ms clock with a 3px ring', () => {
		const a = rule('.bn-pressable:active');
		expect(a).toContain('transition-duration: var(--bn-dur-press)');
		expect(a).toContain('transform: translateY(1px) scale(0.97)');
		expect(a).toContain('box-shadow: 0 0 0 3px color-mix(in oklab, var(--bn-text) 22%, transparent)');
		expect(rule(".bn-pressable[data-lens='r']:active")).toContain('transform: scale(0.995)');
	});
	it('tones: is-dark fills surface-2, is-row tints 3%, is-primary wipes accent-2 in from the left', () => {
		expect(rule('.bn-pressable.is-dark:hover')).toContain('background-color: var(--bn-surface-2)');
		expect(rule('.bn-pressable.is-row:hover')).toContain('color-mix(in oklab, var(--bn-text) 3%, var(--bn-bg))');
		expect(rule('.bn-pressable.is-primary')).toContain('linear-gradient(90deg, var(--bn-accent) 50%, var(--bn-accent-2) 50%)');
		expect(rule('.bn-pressable.is-primary:hover')).toContain('background-position: 0 0');
	});
	it('keeps an absolute or fixed placement', () => {
		expect(rule('.bn-pressable.absolute')).toContain('position: absolute');
		expect(rule('.bn-pressable.fixed')).toContain('position: fixed');
	});
	it('focus-visible draws the ring, not an outline', () => {
		const f = rule('.bn-pressable:focus-visible');
		expect(f).toContain('outline: none');
		expect(f).toContain('border-color: var(--bn-text)');
	});
	it('reduced motion: no transform and no shadow on any state', () => {
		expect(reduced).toMatch(/\.bn-pressable,\s*\.bn-pressable:hover,\s*\.bn-pressable:active \{\s*transform: none !important;\s*box-shadow: none !important;/);
	});
});

describe('slab motion utilities', () => {
	it('rise, draw, fill, grow and sweep, on the house ease and the --rise-i stagger', () => {
		expect(rule('.bn-rise')).toContain('animation: bn-rise 0.6s var(--bn-ease) backwards');
		expect(rule('.bn-draw')).toContain('animation: bn-draw 1.4s var(--bn-ease) 0.3s both');
		expect(rule('.bn-fill')).toContain('animation-delay: calc(var(--rise-i) * 90ms + 300ms)');
		expect(rule('.bn-grow')).toContain('animation-delay: calc(var(--rise-i) * 90ms + 200ms)');
		expect(rule('.bn-sweep')).toContain('stroke-dashoffset: 1');
		expect(css).toMatch(/@keyframes bn-fill \{\s*from \{ transform: scaleX\(0\); \}/);
		expect(css).toMatch(/@keyframes bn-grow \{\s*from \{ transform: scaleY\(0\); \}/);
	});
	it('the rise-card hover lift rides the lens clock', () => {
		expect(rule('.bn-rise-card')).toContain('transform var(--bn-dur-lens) var(--bn-ease-lens)');
	});
	it('enter, pop, view, overlay and panel entrances, the skeleton shimmer', () => {
		expect(rule('.bn-enter')).toContain('animation: bn-om-in 0.3s var(--bn-ease-lens) both');
		expect(rule('.bn-pop')).toContain('animation: bn-om-pop 0.32s var(--bn-ease-spring) both');
		expect(rule('.bn-view')).toContain('animation: bn-view-in 0.24s var(--bn-ease)');
		expect(rule('.bn-overlay-in')).toContain('animation: bn-overlay-in 0.22s ease-out both');
		expect(rule('.bn-panel-in')).toContain('animation: bn-panel-in 0.3s var(--bn-ease) 0.05s both');
		expect(rule('.bn-skeleton')).toContain('animation: bn-om-shimmer 1.4s linear infinite');
		expect(rule('.bn-blink')).toContain('animation: bn-om-blink 2.6s steps(1) infinite');
	});
	it('lens-child, linky and state-fade transitions', () => {
		expect(rule('.bn-lens-child')).toContain('color var(--bn-dur-lens) var(--bn-ease-lens)');
		expect(rule('.bn-linky:hover')).toContain('transform: translateX(1px)');
		expect(rule('.bn-state-fade')).toContain('background-color var(--bn-dur-press) var(--bn-ease-lens)');
	});
	it('agent liveness orbits the border; a live task breathes', () => {
		expect(rule('.bn-agent-live::before')).toContain('animation: bn-agent-orbit 2.4s linear infinite');
		expect(rule('.bn-task-live')).toContain('animation: bn-task-breathe 1.5s ease-in-out infinite');
		expect(css).toContain('@property --bn-agent-angle');
	});
	it('reduced motion stops every one of them', () => {
		for (const c of ['.bn-rise', '.bn-draw', '.bn-fill', '.bn-grow', '.bn-enter', '.bn-pop', '.bn-skeleton', '.bn-spotlight', '.bn-view', '.bn-overlay-in', '.bn-panel-in', '.bn-blink']) {
			expect(reduced, c).toMatch(new RegExp(`${c.replace('.', '\\.')}\\b[^{]*\\{[^}]*animation: none`));
		}
		expect(reduced).toMatch(/\.bn-sweep \{[^}]*stroke-dashoffset: 0/);
		expect(reduced).toMatch(/\.bn-linky:hover \{\s*transform: none !important/);
		expect(reduced).toMatch(/\.bn-agent-live::before \{ animation: none/);
	});
	it('reduced motion keeps the fades and the insight card: they end at full opacity (prod stops only rise/draw/fill/grow/sweep)', () => {
		for (const c of ['.bn-fade', '.bn-insight']) {
			expect(reduced, c).not.toMatch(new RegExp(`${c.replace('.', '\\.')}\\b[^{]*\\{[^}]*animation: none`));
		}
	});
});

describe('spotlight', () => {
	it('a card glow at --bn-sx/--bn-sy, and one fixed page glow moved by transform', () => {
		expect(rule('.bn-spotlight')).toContain('260px circle at var(--bn-sx, -999px) var(--bn-sy, -999px)');
		expect(rule('.bn-spotlight.is-page')).toContain('position: fixed');
		expect(rule('.bn-spotlight.is-page')).toContain('z-index: 35');
		expect(rule('.bn-spotlight.is-page::before')).toContain('transform: translate(var(--bn-px, -999px), var(--bn-py, -999px))');
	});
	it('components render the layers, hidden from assistive tech', () => {
		const card = render(Spotlight).container.querySelector('span')!;
		expect(card.className).toBe('bn-spotlight');
		expect(card.getAttribute('aria-hidden')).toBe('true');
		const page = render(PageSpotlight).container.querySelector('span')!;
		expect(page.className).toBe('bn-spotlight is-page');
	});
});

describe('lens (useLens.ts)', () => {
	const box = (el: Element, r: { left: number; top: number; width: number; height: number }) =>
		vi.spyOn(el, 'getBoundingClientRect').mockReturnValue({ ...r, right: r.left + r.width, bottom: r.top + r.height, x: r.left, y: r.top, toJSON() {} } as DOMRect);

	it('controls pull 4px, rows 2px, tilt at .75 of the pull', () => {
		expect(LENS_PULL).toEqual({ c: 4, r: 2 });
		const rect = { left: 0, top: 0, width: 100, height: 50 };
		// pointer at the right edge, vertical centre
		expect(lensVars(rect, 100, 25, 'c')).toEqual({ lx: '4.0px', ly: '0.0px', ry: '3.0deg', rx: '0.0deg' });
		expect(lensVars(rect, 0, 0, 'r')).toEqual({ lx: '-2.0px', ly: '-2.0px', ry: '-1.5deg', rx: '1.5deg' });
		// clamped outside the box
		expect(lensVars(rect, 500, 25, 'c').lx).toBe('4.0px');
	});

	it('writes the pull onto the hovered [data-lens], the glow onto [data-spot] and :root, and clears on leave', () => {
		reducedMotion(false);
		document.body.innerHTML = '<div data-spot id="card"><button data-lens="c" id="btn">x</button></div><p id="out">y</p>';
		const btn = document.getElementById('btn')!;
		const card = document.getElementById('card')!;
		box(btn, { left: 0, top: 0, width: 100, height: 50 });
		box(card, { left: 0, top: 0, width: 200, height: 100 });
		const stop = installLens(document);
		btn.dispatchEvent(new MouseEvent('pointermove', { bubbles: true, clientX: 100, clientY: 25 }));
		expect(btn.style.getPropertyValue('--bn-lx')).toBe('4.0px');
		expect(btn.style.getPropertyValue('--bn-ry')).toBe('3.0deg');
		expect(card.style.getPropertyValue('--bn-sx')).toBe('100px');
		expect(document.documentElement.style.getPropertyValue('--bn-px')).toBe('100px');
		// moving off it resets the element and dims the spot
		document.getElementById('out')!.dispatchEvent(new MouseEvent('pointermove', { bubbles: true, clientX: 300, clientY: 300 }));
		expect(btn.style.getPropertyValue('--bn-lx')).toBe('');
		expect(card.style.getPropertyValue('--bn-sx')).toBe('-999px');
		stop();
		expect(document.documentElement.style.getPropertyValue('--bn-px')).toBe('-999px');
		btn.dispatchEvent(new MouseEvent('pointermove', { bubbles: true, clientX: 50, clientY: 25 }));
		expect(btn.style.getPropertyValue('--bn-lx')).toBe('');
	});

	it('does nothing at all under reduced motion', () => {
		reducedMotion(true);
		document.body.innerHTML = '<button data-lens="c" id="btn">x</button>';
		const btn = document.getElementById('btn')!;
		box(btn, { left: 0, top: 0, width: 100, height: 50 });
		const stop = installLens(document);
		btn.dispatchEvent(new MouseEvent('pointermove', { bubbles: true, clientX: 100, clientY: 25 }));
		expect(btn.style.getPropertyValue('--bn-lx')).toBe('');
		stop();
	});
});

describe('Rise (components/motion.tsx)', () => {
	it('a div that rises on its index', () => {
		const { container, getByText } = render(Rise, { i: 3, class: 'rise-extra', children: snip('hi') });
		const el = container.firstElementChild as HTMLElement;
		expect(el.tagName).toBe('DIV');
		expect(el.className).toBe('bn-rise rise-extra');
		expect(el.getAttribute('style')).toContain('--rise-i: 3');
		expect(getByText('hi')).toBeTruthy();
	});
	it('as another tag, with its own style kept', () => {
		const { container } = render(Rise, { as: 'section', style: 'color: red', children: snip('x') });
		const el = container.firstElementChild as HTMLElement;
		expect(el.tagName).toBe('SECTION');
		expect(el.getAttribute('style')).toMatch(/color: red;\s*--rise-i: 0/);
	});
});

describe('Pressable (components/Pressable.tsx)', () => {
	it('a secondary control button by default: lens c, is-dark, 26px, 6px corners', () => {
		const { container } = render(Pressable, { children: snip('go') });
		const b = container.querySelector('button')!;
		expect(b.getAttribute('type')).toBe('button');
		expect(b.getAttribute('data-lens')).toBe('c');
		expect(b.getAttribute('data-tone')).toBe('secondary');
		for (const c of ['bn-pressable', 'is-dark', 'h-[26px]', 'rounded-[6px]']) expect(b.className).toContain(c);
	});
	it('an href renders a link; the row tone is a lens r block', () => {
		const { container } = render(Pressable, { href: '/os/comms', tone: 'row', children: snip('row') });
		const a = container.querySelector('a')!;
		expect(a.getAttribute('href')).toBe('/os/comms');
		expect(a.getAttribute('data-lens')).toBe('r');
		expect(a.className).toContain('is-row');
	});
	it('primary and ghost tones', () => {
		expect(render(Pressable, { tone: 'primary', children: snip('p') }).container.querySelector('button')!.className).toContain('is-primary');
		const ghost = render(Pressable, { tone: 'ghost', children: snip('g') }).container.querySelector('button')!;
		expect(ghost.className).toContain('w-[26px]');
	});
	it('passes clicks and attributes through', async () => {
		const onclick = vi.fn();
		const { container } = render(Pressable, { onclick, 'aria-label': 'Run', children: snip('r') });
		const b = container.querySelector('button')!;
		expect(b.getAttribute('aria-label')).toBe('Run');
		await fireEvent.click(b);
		expect(onclick).toHaveBeenCalledTimes(1);
	});
});

describe('ToggleChip (Pressable.tsx Chip)', () => {
	it('a 24px pill that goes solid when on', () => {
		const off = render(ToggleChip, { children: snip('All') }).container.querySelector('button')!;
		for (const c of ['bn-pressable', 'rounded-full', 'h-6', 'is-dark']) expect(off.className).toContain(c);
		expect(off.getAttribute('data-on')).toBe('false');
		const on = render(ToggleChip, { on: true, children: snip('All') }).container.querySelector('button')!;
		expect(on.getAttribute('data-on')).toBe('true');
		expect(on.className).not.toContain('is-dark');
		expect(rule(".bn-toggle-chip[data-on='true']")).toContain('background: var(--bn-text)');
	});
});

describe('slab pill classes (slab.tsx PILL / PILL_ACCENT / chipClass)', () => {
	it('pills are round pressables; the accent pill lifts; the active filter chip is solid', () => {
		for (const c of ['bn-pressable', 'bn-pill', 'rounded-full', 'px-4', 'py-2', 'text-[13px]']) expect(PILL).toContain(c);
		for (const c of ['bn-pill-accent', 'rounded-full']) expect(PILL_ACCENT).toContain(c);
		expect(chipClass(true)).toContain('is-on');
		expect(chipClass(false)).not.toContain('is-on');
		expect(chipClass(false)).toContain('rounded-full');
		expect(rule('.bn-pill-accent:hover')).toContain('transform: translateY(-1px)');
		expect(rule('.bn-filter-chip.is-on')).toContain('background: var(--bn-accent)');
	});
});

describe('OsMark (components/OsMark.tsx)', () => {
	it('the swirl emblem image at the asked size, never a boxed letter', () => {
		const img = render(OsMark, { size: 34 }).container.querySelector('img')!;
		expect(img.getAttribute('alt')).toBe('Founder OS');
		expect(img.getAttribute('src')).toMatch(/os-emblem\.png/);
		expect(img.getAttribute('width')).toBe('34');
		expect(img.getAttribute('style')).toContain('object-fit: contain');
	});
});
