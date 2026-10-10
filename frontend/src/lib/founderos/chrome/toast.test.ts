// The OS-wide toast stack (FounderOS v1 components/Toaster.tsx): results that
// live somewhere else (archived, delegated, created, failed). One line, verb
// first; ok/warn auto-dismiss at 2.6s, err stays; hover pauses; destructive
// actions carry undo; bottom-right, newest on top, max 4.
import { fireEvent, render, screen } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { tick } from 'svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import Toaster from './Toaster.svelte';
import { TOAST_TTL, hold, sweep, toast, toasts } from './toast';

beforeEach(() => toasts.set([]));
afterEach(() => vi.useRealTimers());

describe('toast store', () => {
	it('push returns an id; ok/warn/busy live 2.6s, err stays until closed', () => {
		expect(TOAST_TTL).toBe(2600);
		const a = toast.ok('Archived thread');
		toast.warn('Partial sync');
		toast.busy('Running…');
		toast.err('Send failed');
		const list = get(toasts);
		expect(list.map((t) => [t.kind, t.ttl])).toEqual([
			['ok', 2600],
			['warn', 2600],
			['busy', 2600],
			['err', Infinity]
		]);
		expect(list[0].id).toBe(a);
	});

	it('keeps at most four, dropping the oldest', () => {
		for (let i = 0; i < 6; i++) toast.ok(`t${i}`);
		expect(get(toasts).map((t) => t.text)).toEqual(['t2', 't3', 't4', 't5']);
	});

	it('update swaps kind and text and restarts the clock; close removes', () => {
		const id = toast.busy('Asking');
		toast.update(id, 'err', 'Conductor unreachable');
		expect(get(toasts)[0]).toMatchObject({ kind: 'err', text: 'Conductor unreachable', ttl: Infinity });
		toast.close(id);
		expect(get(toasts)).toEqual([]);
	});

	it('sweep drops expired toasts, but never a held one or an err', () => {
		toast.ok('done');
		const held = toast.ok('held');
		toast.err('broken');
		hold(held, true);
		sweep(Date.now() + 3000);
		expect(get(toasts).map((t) => t.text)).toEqual(['held', 'broken']);
		// releasing restarts its clock
		hold(held, false);
		sweep(Date.now() + 1000);
		expect(get(toasts).map((t) => t.text)).toEqual(['held', 'broken']);
		sweep(Date.now() + 3000);
		expect(get(toasts).map((t) => t.text)).toEqual(['broken']);
	});
});

describe('Toaster', () => {
	it('renders the stack bottom-right, newest on top, with status glyphs', async () => {
		const { container } = render(Toaster);
		toast.ok('Created task');
		toast.err('Send failed');
		await tick();
		const stack = container.querySelector('[data-part="toasts"]')!;
		expect(stack.className).toMatch(/fixed bottom-4 right-4 z-\[60\] flex w-\[300px\] flex-col-reverse gap-1\.5/);
		const rows = [...stack.querySelectorAll('[data-toast]')];
		expect(rows.map((r) => r.getAttribute('data-kind'))).toEqual(['ok', 'err']);
		expect(rows[0].textContent).toContain('✓');
		expect(rows[1].textContent).toContain('✕');
	});

	it('undo runs the callback and closes; × closes', async () => {
		render(Toaster);
		const undo = vi.fn();
		toast.ok('Archived', undo);
		await tick();
		await fireEvent.click(screen.getByRole('button', { name: 'undo' }));
		expect(undo).toHaveBeenCalledTimes(1);
		expect(get(toasts)).toEqual([]);
		toast.warn('Heads up');
		await tick();
		await fireEvent.click(screen.getByRole('button', { name: 'Dismiss' }));
		expect(get(toasts)).toEqual([]);
	});

	it('hover holds a toast past its ttl', async () => {
		vi.useFakeTimers();
		const { container } = render(Toaster);
		toast.ok('Hold me');
		await tick();
		await fireEvent.mouseEnter(container.querySelector('[data-toast]')!);
		vi.advanceTimersByTime(4000);
		await tick();
		expect(get(toasts)).toHaveLength(1);
		await fireEvent.mouseLeave(container.querySelector('[data-toast]')!);
		vi.advanceTimersByTime(3000);
		await tick();
		expect(get(toasts)).toHaveLength(0);
	});

	it('only a timed toast shows the progress hairline', async () => {
		const { container } = render(Toaster);
		toast.ok('timed');
		toast.err('sticky');
		await tick();
		const rows = [...container.querySelectorAll('[data-toast]')];
		expect(rows[0].querySelector('[data-part="ttl"]')).not.toBeNull();
		expect(rows[1].querySelector('[data-part="ttl"]')).toBeNull();
	});
});
