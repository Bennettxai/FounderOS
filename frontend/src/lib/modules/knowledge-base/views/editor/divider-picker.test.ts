import { describe, expect, it } from 'vitest';
import type { Block } from '../../entities/types';
import {
	closeDividerPickerState,
	createDividerStyleBlock,
	isDividerPickerOutsideTarget,
	toggleDividerPickerState
} from './divider-picker';

describe('divider picker state helpers', () => {
	it('toggles picker visibility from the current state', () => {
		expect(toggleDividerPickerState(false)).toBe(true);
		expect(toggleDividerPickerState(true)).toBe(false);
	});

	it('closes the picker', () => {
		expect(closeDividerPickerState()).toBe(false);
	});
});

describe('divider picker outside target detection', () => {
	it('treats clicks inside a divider wrapper as inside targets', () => {
		const wrapper = document.createElement('div');
		const option = document.createElement('button');
		wrapper.className = 'divider-wrapper';
		wrapper.append(option);
		document.body.append(wrapper);

		expect(isDividerPickerOutsideTarget(wrapper)).toBe(false);
		expect(isDividerPickerOutsideTarget(option)).toBe(false);

		document.body.innerHTML = '';
	});

	it('treats clicks outside the divider wrapper or without an element target as outside targets', () => {
		const outside = document.createElement('div');
		document.body.append(outside);

		expect(isDividerPickerOutsideTarget(outside)).toBe(true);
		expect(isDividerPickerOutsideTarget(document.createTextNode('not an element'))).toBe(true);
		expect(isDividerPickerOutsideTarget(null)).toBe(true);

		document.body.innerHTML = '';
	});

	it('supports a custom wrapper selector', () => {
		const wrapper = document.createElement('div');
		const child = document.createElement('button');
		wrapper.dataset.dividerPicker = 'true';
		wrapper.append(child);
		document.body.append(wrapper);

		expect(isDividerPickerOutsideTarget(child, '[data-divider-picker="true"]')).toBe(false);

		document.body.innerHTML = '';
	});
});

describe('divider picker block updates', () => {
	it('creates an updated divider block without mutating the original block', () => {
		const block = createBlock({
			properties: {
				color: 'gray',
				divider_style: 'solid'
			}
		});

		const updatedBlock = createDividerStyleBlock(block, 'double');

		expect(updatedBlock).toEqual({
			...block,
			properties: {
				color: 'gray',
				divider_style: 'double'
			}
		});
		expect(updatedBlock).not.toBe(block);
		expect(updatedBlock.properties).not.toBe(block.properties);
		expect(block.properties.divider_style).toBe('solid');
	});

	it('adds divider style when the block has no existing properties', () => {
		expect(createDividerStyleBlock(createBlock(), 'gradient').properties).toEqual({
			divider_style: 'gradient'
		});
	});
});

function createBlock(overrides: Partial<Block> = {}): Block {
	return {
		id: 'block-1',
		type: 'divider',
		content: [],
		properties: {},
		children: [],
		created_at: '2026-05-02T00:00:00.000Z',
		updated_at: '2026-05-02T00:00:00.000Z',
		...overrides
	};
}
