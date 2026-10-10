import { describe, expect, it } from 'vitest';
import {
	BLOCK_DRAG_ID_MIME,
	BLOCK_DRAG_INDEX_MIME,
	allowBlockDrop,
	clearBlockDragOver,
	clearBlockDragState,
	createBlockDragState,
	getBlockDragOverPosition,
	getBlockDropMove,
	parseBlockDragIndex,
	readBlockDragIndex,
	setBlockDragOver,
	startBlockDrag,
	writeBlockDragData
} from './block-drag';

describe('block drag state helpers', () => {
	it('creates idle state with optional overrides', () => {
		expect(createBlockDragState()).toEqual({
			isDragging: false,
			isDragOver: false,
			dragOverPosition: null
		});

		expect(createBlockDragState({ isDragging: true })).toEqual({
			isDragging: true,
			isDragOver: false,
			dragOverPosition: null
		});
	});

	it('marks drag lifecycle states without mutating the input', () => {
		const initial = createBlockDragState();
		const dragging = startBlockDrag(initial);
		const over = setBlockDragOver(dragging, 'after');

		expect(initial).toEqual({
			isDragging: false,
			isDragOver: false,
			dragOverPosition: null
		});
		expect(dragging).toEqual({
			isDragging: true,
			isDragOver: false,
			dragOverPosition: null
		});
		expect(over).toEqual({
			isDragging: true,
			isDragOver: true,
			dragOverPosition: 'after'
		});
		expect(clearBlockDragOver(over)).toEqual({
			isDragging: true,
			isDragOver: false,
			dragOverPosition: null
		});
		expect(clearBlockDragState(over)).toEqual({
			isDragging: false,
			isDragOver: false,
			dragOverPosition: null
		});
	});
});

describe('block drag position and move math', () => {
	it('chooses before above the midpoint and after at or below it', () => {
		const rect = { top: 100, height: 40 };

		expect(getBlockDragOverPosition(119, rect)).toBe('before');
		expect(getBlockDragOverPosition(120, rect)).toBe('after');
		expect(getBlockDragOverPosition(121, rect)).toBe('after');
	});

	it.each([
		[{ fromIndex: 3, targetIndex: 1, position: 'before' as const }, { fromIndex: 3, toIndex: 1 }],
		[{ fromIndex: 3, targetIndex: 1, position: 'after' as const }, { fromIndex: 3, toIndex: 2 }],
		[{ fromIndex: 1, targetIndex: 4, position: 'before' as const }, { fromIndex: 1, toIndex: 3 }],
		[{ fromIndex: 1, targetIndex: 4, position: 'after' as const }, { fromIndex: 1, toIndex: 4 }]
	])('maps drop input %o to move %o', (input, move) => {
		expect(getBlockDropMove(input)).toEqual(move);
	});

	it.each([
		{ fromIndex: 2, targetIndex: 2, position: 'before' as const },
		{ fromIndex: 2, targetIndex: 2, position: 'after' as const },
		{ fromIndex: 2, targetIndex: 3, position: 'before' as const },
		{ fromIndex: 2.5, targetIndex: 3, position: 'before' as const },
		{ fromIndex: 2, targetIndex: Number.NaN, position: 'before' as const }
	])('returns null for non-moves and invalid indexes: %o', (input) => {
		expect(getBlockDropMove(input)).toBeNull();
	});
});

describe('block drag dataTransfer helpers', () => {
	it('writes drag payload data and move effect', () => {
		const data = new Map<string, string>();
		const dataTransfer = {
			effectAllowed: 'none' as DataTransfer['effectAllowed'],
			setData: (format: string, value: string) => data.set(format, value)
		};

		writeBlockDragData(dataTransfer, { index: 7, blockId: 'block-7' });

		expect(dataTransfer.effectAllowed).toBe('move');
		expect(data.get(BLOCK_DRAG_INDEX_MIME)).toBe('7');
		expect(data.get(BLOCK_DRAG_ID_MIME)).toBe('block-7');
	});

	it('parses dragged indexes from text/plain payloads', () => {
		expect(parseBlockDragIndex('12')).toBe(12);
		expect(parseBlockDragIndex('12px')).toBe(12);
		expect(parseBlockDragIndex('not-a-number')).toBeNull();

		expect(readBlockDragIndex({ getData: () => '4' })).toBe(4);
		expect(readBlockDragIndex({ getData: () => '' })).toBeNull();
	});

	it('sets move as the allowed drop effect', () => {
		const dataTransfer = { dropEffect: 'none' as DataTransfer['dropEffect'] };

		allowBlockDrop(dataTransfer);

		expect(dataTransfer.dropEffect).toBe('move');
	});
});
