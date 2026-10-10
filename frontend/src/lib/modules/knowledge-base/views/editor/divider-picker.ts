import type { Block, DividerStyle } from '../../entities/types';

export const DIVIDER_PICKER_WRAPPER_SELECTOR = '.divider-wrapper';

export type DividerPickerState = boolean;

export function toggleDividerPickerState(isOpen: DividerPickerState): DividerPickerState {
	return !isOpen;
}

export function closeDividerPickerState(): DividerPickerState {
	return false;
}

export function isDividerPickerOutsideTarget(
	target: EventTarget | null,
	wrapperSelector = DIVIDER_PICKER_WRAPPER_SELECTOR
): boolean {
	if (typeof Element === 'undefined') {
		return true;
	}

	return !(target instanceof Element) || !target.closest(wrapperSelector);
}

export function createDividerStyleBlock(block: Block, style: DividerStyle): Block {
	return {
		...block,
		properties: {
			...block.properties,
			divider_style: style
		}
	};
}
