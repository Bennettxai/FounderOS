/**
 * Keyboard Shortcuts Manager for App Templates
 */

export interface KeyboardShortcut {
	key: string;
	ctrl?: boolean;
	shift?: boolean;
	alt?: boolean;
	meta?: boolean;
	action: () => void;
	description?: string;
}

export interface KeyboardShortcutGroup {
	name: string;
	shortcuts: KeyboardShortcut[];
}

/**
 * Default shortcuts for data views
 */
export const defaultViewShortcuts: KeyboardShortcutGroup[] = [
	{
		name: 'Navigation',
		shortcuts: [
			{ key: 'j', action: () => {}, description: 'Move to next row' },
			{ key: 'k', action: () => {}, description: 'Move to previous row' },
			{ key: 'ArrowDown', action: () => {}, description: 'Move to next row' },
			{ key: 'ArrowUp', action: () => {}, description: 'Move to previous row' },
			{ key: 'Home', ctrl: true, action: () => {}, description: 'Go to first row' },
			{ key: 'End', ctrl: true, action: () => {}, description: 'Go to last row' },
			{ key: 'PageDown', action: () => {}, description: 'Next page' },
			{ key: 'PageUp', action: () => {}, description: 'Previous page' },
		]
	},
	{
		name: 'Selection',
		shortcuts: [
			{ key: ' ', action: () => {}, description: 'Toggle row selection' },
			{ key: 'a', ctrl: true, action: () => {}, description: 'Select all' },
			{ key: 'Escape', action: () => {}, description: 'Clear selection' },
			{ key: 'j', shift: true, action: () => {}, description: 'Extend selection down' },
			{ key: 'k', shift: true, action: () => {}, description: 'Extend selection up' },
		]
	},
	{
		name: 'Actions',
		shortcuts: [
			{ key: 'Enter', action: () => {}, description: 'Open record' },
			{ key: 'e', action: () => {}, description: 'Edit record' },
			{ key: 'n', ctrl: true, action: () => {}, description: 'New record' },
			{ key: 'Delete', action: () => {}, description: 'Delete selected' },
			{ key: 'c', ctrl: true, action: () => {}, description: 'Copy' },
			{ key: 'v', ctrl: true, action: () => {}, description: 'Paste' },
		]
	},
	{
		name: 'View',
		shortcuts: [
			{ key: '/', action: () => {}, description: 'Focus search' },
			{ key: 'f', ctrl: true, action: () => {}, description: 'Open filter' },
			{ key: 's', ctrl: true, action: () => {}, description: 'Save' },
			{ key: '?', action: () => {}, description: 'Show shortcuts' },
		]
	}
];

/**
 * Create a keyboard event handler
 */
export function createKeyboardHandler(shortcuts: KeyboardShortcut[]) {
	return function handleKeydown(event: KeyboardEvent) {
		// Don't handle if in input/textarea
		if (
			event.target instanceof HTMLInputElement ||
			event.target instanceof HTMLTextAreaElement ||
			(event.target as HTMLElement).isContentEditable
		) {
			return;
		}

		for (const shortcut of shortcuts) {
			const keyMatches = event.key === shortcut.key || event.key.toLowerCase() === shortcut.key.toLowerCase();
			const ctrlMatches = !!shortcut.ctrl === (event.ctrlKey || event.metaKey);
			const shiftMatches = !!shortcut.shift === event.shiftKey;
			const altMatches = !!shortcut.alt === event.altKey;

			if (keyMatches && ctrlMatches && shiftMatches && altMatches) {
				event.preventDefault();
				shortcut.action();
				return;
			}
		}
	};
}

/**
 * Format shortcut for display
 */
export function formatShortcut(shortcut: KeyboardShortcut): string {
	const parts: string[] = [];

	if (shortcut.ctrl || shortcut.meta) {
		parts.push(navigator.platform.includes('Mac') ? '⌘' : 'Ctrl');
	}
	if (shortcut.alt) {
		parts.push(navigator.platform.includes('Mac') ? '⌥' : 'Alt');
	}
	if (shortcut.shift) {
		parts.push('⇧');
	}

	// Format special keys
	const keyMap: Record<string, string> = {
		' ': 'Space',
		'ArrowUp': '↑',
		'ArrowDown': '↓',
		'ArrowLeft': '←',
		'ArrowRight': '→',
		'Enter': '↵',
		'Escape': 'Esc',
		'Delete': 'Del',
		'Backspace': '⌫',
	};

	parts.push(keyMap[shortcut.key] || shortcut.key.toUpperCase());

	return parts.join(' + ');
}

/**
 * Hook for using keyboard shortcuts in Svelte components
 */
export function useKeyboardShortcuts(
	shortcuts: KeyboardShortcut[],
	enabled: boolean = true
): { destroy: () => void } {
	const handler = createKeyboardHandler(shortcuts);

	if (enabled) {
		document.addEventListener('keydown', handler);
	}

	return {
		destroy() {
			document.removeEventListener('keydown', handler);
		}
	};
}
