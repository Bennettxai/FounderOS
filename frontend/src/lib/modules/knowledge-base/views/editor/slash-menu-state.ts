export interface SlashMenuPosition {
	x: number;
	y: number;
}

export interface SlashMenuState {
	show: boolean;
	position: SlashMenuPosition;
	filter: string;
}

export interface SlashMenuPositionInput {
	selection?: Pick<Selection, 'rangeCount' | 'getRangeAt'> | null;
	targetRect?: Pick<DOMRect, 'left' | 'bottom'> | null;
	offset?: number;
}

export interface SlashMenuInputStateOptions extends SlashMenuPositionInput {
	content: string;
	state: SlashMenuState;
}

export const SLASH_MENU_OFFSET = 4;

const HIDDEN_SLASH_MENU_STATE: SlashMenuState = {
	show: false,
	position: { x: 0, y: 0 },
	filter: ''
};

export function createSlashMenuState(overrides: Partial<SlashMenuState> = {}): SlashMenuState {
	const { position, ...stateOverrides } = overrides;

	return {
		show: HIDDEN_SLASH_MENU_STATE.show,
		filter: HIDDEN_SLASH_MENU_STATE.filter,
		...stateOverrides,
		position: position ? { ...position } : { ...HIDDEN_SLASH_MENU_STATE.position }
	};
}

export function getSlashCommandFilter(content: string): string | null {
	const slashMatch = content.match(/^\/(\S*)$/);
	return slashMatch ? slashMatch[1] || '' : null;
}

export function isSlashCommandFilter(content: string): boolean {
	return getSlashCommandFilter(content) !== null;
}

export function getSlashMenuPosition({
	selection,
	targetRect,
	offset = SLASH_MENU_OFFSET
}: SlashMenuPositionInput): SlashMenuPosition {
	if (selection && selection.rangeCount > 0) {
		const range = selection.getRangeAt(0);
		const rect = range.getBoundingClientRect();
		return getSlashMenuPositionFromRect(rect, offset);
	}

	if (targetRect) {
		return getSlashMenuPositionFromRect(targetRect, offset);
	}

	return { ...HIDDEN_SLASH_MENU_STATE.position };
}

export function getSlashMenuPositionFromRect(
	rect: Pick<DOMRect, 'left' | 'bottom'>,
	offset = SLASH_MENU_OFFSET
): SlashMenuPosition {
	return {
		x: rect.left,
		y: rect.bottom + offset
	};
}

export function updateSlashMenuFilterWhileOpen(state: SlashMenuState, content: string): SlashMenuState {
	if (!state.show) {
		return { ...state, position: { ...state.position } };
	}

	const lastSlashIndex = content.lastIndexOf('/');
	if (lastSlashIndex < 0) {
		return resetSlashMenuState(state);
	}

	return {
		...state,
		position: { ...state.position },
		filter: content.slice(lastSlashIndex + 1)
	};
}

export function getSlashMenuStateForContentInput({
	content,
	state,
	selection,
	targetRect,
	offset
}: SlashMenuInputStateOptions): SlashMenuState {
	const filter = getSlashCommandFilter(content);
	if (filter !== null) {
		return {
			show: true,
			position: getSlashMenuPosition({ selection, targetRect, offset }),
			filter
		};
	}

	return updateSlashMenuFilterWhileOpen(state, content);
}

export function resetSlashMenuState(state: SlashMenuState = HIDDEN_SLASH_MENU_STATE): SlashMenuState {
	return {
		...state,
		show: false,
		position: { ...state.position },
		filter: ''
	};
}

export function closeSlashMenu(state: SlashMenuState): SlashMenuState {
	return resetSlashMenuState(state);
}

export function selectSlashMenuItem(state: SlashMenuState): SlashMenuState {
	return resetSlashMenuState(state);
}
