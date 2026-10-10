// The Conductor emblem (FounderOS v1 components/ConductorEmblem.tsx +
// tests/conductor-emblem.test.ts): a round core in a hairline ring that comes
// alive only while an agent is being chatted with, and never under reduced
// motion.
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import ConductorEmblem from './ConductorEmblem.svelte';
import { conductorEmblemClasses } from './emblem';

describe('conductorEmblemClasses', () => {
	it('is a plain conductor-emblem when idle', () => {
		expect(conductorEmblemClasses(false)).toBe('conductor-emblem');
	});
	it('adds thinking while chatting, keeps the base class first, passes extras, never an empty token', () => {
		expect(conductorEmblemClasses(true).split(' ')).toEqual(['conductor-emblem', 'thinking']);
		expect(conductorEmblemClasses(false, 'shrink-0')).toBe('conductor-emblem shrink-0');
		expect(conductorEmblemClasses(true, '')).toBe('conductor-emblem thinking');
	});
});

describe('ConductorEmblem', () => {
	it('draws halo, sweep, ring and core at the given size', () => {
		const { container } = render(ConductorEmblem, { size: 32 });
		const el = container.querySelector('.conductor-emblem') as HTMLElement;
		expect(el.style.width).toBe('32px');
		for (const part of ['conductor-halo', 'conductor-sweep', 'conductor-ring', 'conductor-core']) expect(el.querySelector(`.${part}`), part).toBeTruthy();
		expect(el.dataset.thinking).toBeUndefined();
	});
	it('thinking turns the motion on', () => {
		const { container } = render(ConductorEmblem, { thinking: true });
		const el = container.querySelector('.conductor-emblem') as HTMLElement;
		expect(el.classList.contains('thinking')).toBe(true);
		expect(el.dataset.thinking).toBe('true');
	});
	it('its motion is scoped to .thinking and off under prefers-reduced-motion, on theme tokens only', () => {
		const src = readFileSync(resolve(__dirname, 'ConductorEmblem.svelte'), 'utf8');
		expect(src).toMatch(/prefers-reduced-motion: reduce[\s\S]*animation: none/);
		expect(src).not.toMatch(/#[0-9a-f]{3,6}\b/i);
		expect(src).toMatch(/\.thinking[^{]*\.conductor-sweep/);
	});
});
