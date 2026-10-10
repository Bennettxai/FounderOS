export type SelectionFormat = 'bold' | 'italic' | 'underline' | 'strikethrough' | 'code' | 'link';

export interface SelectionToolbarPosition {
	x: number;
	y: number;
}

export interface SelectionSnapshot {
	start: number;
	end: number;
	text: string;
}

export interface SelectionToolbarState {
	show: boolean;
	position: SelectionToolbarPosition;
	selection: SelectionSnapshot | null;
}

export interface SelectionToolbarStateOptions {
	readOnly?: boolean;
	blockElement: HTMLElement | null;
	selection?: Selection | null;
	getSelection?: () => Selection | null;
}

export interface ApplySelectionFormatOptions {
	currentSelection: SelectionSnapshot | null;
	blockElement: HTMLElement | null;
	document?: Document;
	getSelection?: () => Selection | null;
	promptForUrl?: () => string | null;
}

const HIDDEN_TOOLBAR_STATE: SelectionToolbarState = {
	show: false,
	position: { x: 0, y: 0 },
	selection: null
};

const EXEC_COMMAND_BY_FORMAT: Partial<Record<SelectionFormat, string>> = {
	bold: 'bold',
	italic: 'italic',
	underline: 'underline',
	strikethrough: 'strikeThrough'
};

export function getHiddenSelectionToolbarState(): SelectionToolbarState {
	return {
		show: HIDDEN_TOOLBAR_STATE.show,
		position: { ...HIDDEN_TOOLBAR_STATE.position },
		selection: HIDDEN_TOOLBAR_STATE.selection
	};
}

export function getSelectionToolbarState({
	readOnly = false,
	blockElement,
	selection,
	getSelection = getWindowSelection
}: SelectionToolbarStateOptions): SelectionToolbarState {
	const activeSelection = selection === undefined ? getSelection() : selection;

	if (readOnly || !blockElement || !activeSelection || activeSelection.isCollapsed || activeSelection.rangeCount === 0) {
		return getHiddenSelectionToolbarState();
	}

	const range = activeSelection.getRangeAt(0);
	if (!blockElement.contains(range.commonAncestorContainer)) {
		return getHiddenSelectionToolbarState();
	}

	const selectedText = activeSelection.toString();
	if (selectedText.length === 0) {
		return getHiddenSelectionToolbarState();
	}

	const rect = range.getBoundingClientRect();
	const fullText = blockElement.textContent || '';
	const startOffset = fullText.indexOf(selectedText);
	const start = startOffset >= 0 ? startOffset : 0;

	return {
		show: true,
		position: {
			x: rect.left + rect.width / 2,
			y: rect.top
		},
		selection: {
			start,
			end: start + selectedText.length,
			text: selectedText
		}
	};
}

export function getExecCommandForSelectionFormat(format: SelectionFormat): string | null {
	return EXEC_COMMAND_BY_FORMAT[format] ?? null;
}

export function applySelectionFormat(
	format: SelectionFormat,
	{
		currentSelection,
		blockElement,
		document: activeDocument = getWindowDocument(),
		getSelection = getWindowSelection,
		promptForUrl = getPromptedUrl
	}: ApplySelectionFormatOptions
): boolean {
	if (!currentSelection || !blockElement || !activeDocument) {
		return false;
	}

	const execCommand = getExecCommandForSelectionFormat(format);
	if (execCommand) {
		activeDocument.execCommand(execCommand, false);
		return true;
	}

	if (format === 'code') {
		return surroundSelectionWithInlineCode(activeDocument, getSelection());
	}

	if (format === 'link') {
		const url = promptForUrl();
		if (!url) {
			return false;
		}

		activeDocument.execCommand('createLink', false, url);
		return true;
	}

	return false;
}

export function surroundSelectionWithInlineCode(activeDocument: Document, selection: Selection | null): boolean {
	if (!selection || selection.rangeCount === 0 || selection.isCollapsed) {
		return false;
	}

	const range = selection.getRangeAt(0);
	const codeElement = activeDocument.createElement('code');
	codeElement.className = 'inline-code';

	try {
		range.surroundContents(codeElement);
	} catch {
		codeElement.append(range.extractContents());
		range.insertNode(codeElement);
	}

	selection.removeAllRanges();
	const nextRange = activeDocument.createRange();
	nextRange.selectNodeContents(codeElement);
	selection.addRange(nextRange);
	return true;
}

function getWindowSelection(): Selection | null {
	if (typeof window === 'undefined') {
		return null;
	}

	return window.getSelection();
}

function getWindowDocument(): Document | undefined {
	if (typeof document === 'undefined') {
		return undefined;
	}

	return document;
}

function getPromptedUrl(): string | null {
	if (typeof prompt === 'undefined') {
		return null;
	}

	return prompt('Enter URL:');
}
