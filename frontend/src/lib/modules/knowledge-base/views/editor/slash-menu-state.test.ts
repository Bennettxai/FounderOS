import { describe, expect, it } from 'vitest';
import {
	closeSlashMenu,
	createSlashMenuState,
	getSlashCommandFilter,
	getSlashMenuPosition,
	getSlashMenuPositionFromRect,
	getSlashMenuStateForContentInput,
	isSlashCommandFilter,
	resetSlashMenuState,
	selectSlashMenuItem,
	updateSlashMenuFilterWhileOpen,
	type SlashMenuState
} from './slash-menu-state';

describe('slash command filter detection', () => {
	it.each([
		['/', ''],
		['/h', 'h'],
		['/heading', 'heading'],
		['/heading-1', 'heading-1']
	])('detects %s as filter %s', (content, filter) => {
		expect(getSlashCommandFilter(content)).toBe(filter);
		expect(isSlashCommandFilter(content)).toBe(true);
	});

	it.each(['', 'plain /heading', '/heading one', ' /heading'])('ignores non-command content: %s', (content) => {
		expect(getSlashCommandFilter(content)).toBeNull();
		expect(isSlashCommandFilter(content)).toBe(false);
	});
});

describe('slash menu position helpers', () => {
	it('computes a menu position from a rect with the default offset', () => {
		expect(getSlashMenuPositionFromRect({ left: 12, bottom: 34 })).toEqual({ x: 12, y: 38 });
	});

	it('uses the active selection range before the target rect', () => {
		const selection = createSelection({ left: 20, bottom: 50 });

		expect(
			getSlashMenuPosition({
				selection,
				targetRect: { left: 200, bottom: 500 }
			})
		).toEqual({ x: 20, y: 54 });
	});

	it('falls back to the target rect when no range is selected', () => {
		expect(
			getSlashMenuPosition({
				selection: { rangeCount: 0, getRangeAt: () => createRange({ left: 0, bottom: 0 }) },
				targetRect: { left: 9, bottom: 14 },
				offset: 8
			})
		).toEqual({ x: 9, y: 22 });
	});

	it('returns the hidden position when no geometry is available', () => {
		expect(getSlashMenuPosition({ selection: null, targetRect: null })).toEqual({ x: 0, y: 0 });
	});
});

describe('slash menu state updates', () => {
	it('creates hidden state with optional overrides without sharing position references', () => {
		const position = { x: 10, y: 20 };
		const state = createSlashMenuState({ show: true, position, filter: 'h' });
		position.x = 99;

		expect(state).toEqual({ show: true, position: { x: 10, y: 20 }, filter: 'h' });
	});

	it('opens the menu for slash command content and computes its position', () => {
		const state = createSlashMenuState();

		expect(
			getSlashMenuStateForContentInput({
				content: '/todo',
				state,
				selection: createSelection({ left: 16, bottom: 24 })
			})
		).toEqual({
			show: true,
			position: { x: 16, y: 28 },
			filter: 'todo'
		});
	});

	it('updates the filter while the menu is already open', () => {
		const state = createSlashMenuState({ show: true, position: { x: 1, y: 2 }, filter: 'h' });

		expect(updateSlashMenuFilterWhileOpen(state, '/heading')).toEqual({
			show: true,
			position: { x: 1, y: 2 },
			filter: 'heading'
		});
		expect(getSlashMenuStateForContentInput({ content: 'typed /quote', state })).toEqual({
			show: true,
			position: { x: 1, y: 2 },
			filter: 'quote'
		});
	});

	it('closes and clears the filter when open content no longer contains a slash', () => {
		const state = createSlashMenuState({ show: true, position: { x: 5, y: 6 }, filter: 'h' });

		expect(updateSlashMenuFilterWhileOpen(state, 'heading')).toEqual({
			show: false,
			position: { x: 5, y: 6 },
			filter: ''
		});
	});

	it('leaves closed state closed for non-command input', () => {
		const state = createSlashMenuState({ position: { x: 3, y: 4 } });

		expect(getSlashMenuStateForContentInput({ content: 'paragraph', state })).toEqual({
			show: false,
			position: { x: 3, y: 4 },
			filter: ''
		});
	});

	it.each([
		['reset', resetSlashMenuState],
		['close', closeSlashMenu],
		['select', selectSlashMenuItem]
	] as const)('%s clears open/select state without moving the menu position', (_label, reset) => {
		const state: SlashMenuState = { show: true, position: { x: 7, y: 8 }, filter: 'code' };

		expect(reset(state)).toEqual({
			show: false,
			position: { x: 7, y: 8 },
			filter: ''
		});
	});
});

function createSelection(rect: Pick<DOMRect, 'left' | 'bottom'>): Pick<Selection, 'rangeCount' | 'getRangeAt'> {
	return {
		rangeCount: 1,
		getRangeAt: () => createRange(rect)
	};
}

function createRange(rect: Pick<DOMRect, 'left' | 'bottom'>): Range {
	return {
		getBoundingClientRect: () => ({
			x: rect.left,
			y: rect.bottom,
			left: rect.left,
			top: rect.bottom,
			right: rect.left,
			bottom: rect.bottom,
			width: 0,
			height: 0,
			toJSON: () => ({})
		})
	} as Range;
}
