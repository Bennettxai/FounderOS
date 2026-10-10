import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	applySelectionFormat,
	getExecCommandForSelectionFormat,
	getHiddenSelectionToolbarState,
	getSelectionToolbarState,
	surroundSelectionWithInlineCode,
	type SelectionSnapshot
} from './selection-formatting';

describe('selection formatting toolbar helpers', () => {
	afterEach(() => {
		window.getSelection()?.removeAllRanges();
		document.body.innerHTML = '';
	});

	it('returns hidden toolbar state when selection cannot be formatted', () => {
		const blockElement = document.createElement('div');
		blockElement.textContent = 'Plain text';
		document.body.append(blockElement);

		expect(getSelectionToolbarState({ readOnly: true, blockElement, selection: selectText(blockElement, 0, 5) })).toEqual(
			getHiddenSelectionToolbarState()
		);
		expect(getSelectionToolbarState({ blockElement, selection: null })).toEqual(getHiddenSelectionToolbarState());
		expect(getSelectionToolbarState({ blockElement: null, selection: selectText(blockElement, 0, 5) })).toEqual(
			getHiddenSelectionToolbarState()
		);
	});

	it('returns toolbar position and selected text offsets for an in-block selection', () => {
		const blockElement = document.createElement('div');
		blockElement.textContent = 'Hello toolbar';
		document.body.append(blockElement);
		const selection = selectText(blockElement, 6, 13, { left: 10, top: 5, width: 40 });

		expect(getSelectionToolbarState({ blockElement, selection })).toEqual({
			show: true,
			position: { x: 30, y: 5 },
			selection: { start: 6, end: 13, text: 'toolbar' }
		});
	});

	it('hides the toolbar for selections outside the active block', () => {
		const blockElement = document.createElement('div');
		blockElement.textContent = 'Inside';
		const outsideElement = document.createElement('div');
		outsideElement.textContent = 'Outside';
		document.body.append(blockElement, outsideElement);

		expect(getSelectionToolbarState({ blockElement, selection: selectText(outsideElement, 0, 7) })).toEqual(
			getHiddenSelectionToolbarState()
		);
	});
});

describe('selection format command helpers', () => {
	let execCommand: ReturnType<typeof vi.fn>;
	const currentSelection: SelectionSnapshot = { start: 0, end: 5, text: 'Hello' };

	beforeEach(() => {
		execCommand = vi.fn(() => true);
		Object.defineProperty(document, 'execCommand', {
			configurable: true,
			value: execCommand
		});
	});

	afterEach(() => {
		vi.restoreAllMocks();
		window.getSelection()?.removeAllRanges();
		document.body.innerHTML = '';
	});

	it.each([
		['bold', 'bold'],
		['italic', 'italic'],
		['underline', 'underline'],
		['strikethrough', 'strikeThrough']
	] as const)('maps %s to execCommand %s', (format, command) => {
		expect(getExecCommandForSelectionFormat(format)).toBe(command);

		expect(applySelectionFormat(format, { currentSelection, blockElement: document.createElement('div') })).toBe(true);
		expect(execCommand).toHaveBeenCalledWith(command, false);
	});

	it('wraps the active range in inline code', () => {
		const blockElement = document.createElement('div');
		blockElement.textContent = 'Hello';
		document.body.append(blockElement);
		const selection = selectText(blockElement, 0, 5);

		expect(applySelectionFormat('code', { currentSelection, blockElement, getSelection: () => selection })).toBe(true);
		expect(blockElement.innerHTML).toBe('<code class="inline-code">Hello</code>');
	});

	it('creates links only when a URL is provided', () => {
		const blockElement = document.createElement('div');

		expect(
			applySelectionFormat('link', {
				currentSelection,
				blockElement,
				promptForUrl: () => 'https://example.com'
			})
		).toBe(true);
		expect(execCommand).toHaveBeenCalledWith('createLink', false, 'https://example.com');

		execCommand.mockClear();
		expect(applySelectionFormat('link', { currentSelection, blockElement, promptForUrl: () => null })).toBe(false);
		expect(execCommand).not.toHaveBeenCalled();
	});

	it('does not apply commands without a current selection or block element', () => {
		expect(applySelectionFormat('bold', { currentSelection: null, blockElement: document.createElement('div') })).toBe(
			false
		);
		expect(applySelectionFormat('bold', { currentSelection, blockElement: null })).toBe(false);
		expect(execCommand).not.toHaveBeenCalled();
	});

	it('does not wrap code when the browser selection is collapsed', () => {
		const blockElement = document.createElement('div');
		blockElement.textContent = 'Hello';
		document.body.append(blockElement);
		const selection = selectText(blockElement, 0, 0);

		expect(surroundSelectionWithInlineCode(document, selection)).toBe(false);
		expect(blockElement.innerHTML).toBe('Hello');
	});
});

function selectText(
	element: HTMLElement,
	start: number,
	end: number,
	rect: Partial<DOMRect> = {}
): Selection {
	const textNode = element.firstChild;
	if (!textNode) {
		throw new Error('Expected element to contain a text node');
	}

	const range = document.createRange();
	range.setStart(textNode, start);
	range.setEnd(textNode, end);
	Object.defineProperty(range, 'getBoundingClientRect', {
		configurable: true,
		value: () => ({
			x: rect.x ?? rect.left ?? 0,
			y: rect.y ?? rect.top ?? 0,
			left: rect.left ?? 0,
			top: rect.top ?? 0,
			right: rect.right ?? 0,
			bottom: rect.bottom ?? 0,
			width: rect.width ?? 0,
			height: rect.height ?? 0,
			toJSON: () => ({})
		})
	});

	const selection = window.getSelection();
	if (!selection) {
		throw new Error('Expected window selection to be available');
	}

	selection.removeAllRanges();
	selection.addRange(range);
	return selection;
}
