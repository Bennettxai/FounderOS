import type * as Monaco from 'monaco-editor';

export const businessOSDark: Monaco.editor.IStandaloneThemeData = {
	base: 'vs-dark',
	inherit: true,
	rules: [
		// Comments — muted, italic
		{ token: 'comment', foreground: '71717a', fontStyle: 'italic' },
		{ token: 'comment.doc', foreground: '71717a', fontStyle: 'italic' },

		// Keywords — violet accent
		{ token: 'keyword', foreground: 'a78bfa', fontStyle: 'bold' },
		{ token: 'keyword.control', foreground: 'a78bfa' },
		{ token: 'keyword.operator', foreground: 'a1a1aa' },

		// Strings — warm emerald
		{ token: 'string', foreground: '34d399' },
		{ token: 'string.escape', foreground: '6ee7b7' },
		{ token: 'string.key.json', foreground: '818cf8' },

		// Numbers — soft blue
		{ token: 'number', foreground: '60a5fa' },
		{ token: 'number.float', foreground: '60a5fa' },
		{ token: 'number.hex', foreground: '60a5fa' },

		// Types — indigo primary
		{ token: 'type', foreground: '6366f1' },
		{ token: 'type.identifier', foreground: '818cf8' },

		// Functions — warm amber
		{ token: 'function', foreground: 'fbbf24' },
		{ token: 'function.declaration', foreground: 'fbbf24' },

		// Variables — clean white
		{ token: 'variable', foreground: 'f4f4f5' },
		{ token: 'variable.predefined', foreground: 'c4b5fd' },
		{ token: 'variable.parameter', foreground: 'e2e8f0' },

		// Constants — bright cyan
		{ token: 'constant', foreground: '22d3ee' },

		// Operators — muted
		{ token: 'operator', foreground: 'a1a1aa' },
		{ token: 'delimiter', foreground: 'a1a1aa' },
		{ token: 'delimiter.bracket', foreground: 'a1a1aa' },

		// Tags (HTML/JSX/Svelte)
		{ token: 'tag', foreground: '6366f1' },
		{ token: 'metatag', foreground: '818cf8' },
		{ token: 'metatag.content.html', foreground: 'f4f4f5' },
		{ token: 'attribute.name', foreground: 'a78bfa' },
		{ token: 'attribute.value', foreground: '34d399' },
		{ token: 'attribute.value.html', foreground: '34d399' },

		// Regex
		{ token: 'regexp', foreground: 'f87171' },

		// Annotations/Decorators
		{ token: 'annotation', foreground: 'fbbf24' },
		{ token: 'tag.decorator', foreground: 'fbbf24' },

		// Go specific
		{ token: 'keyword.go', foreground: 'a78bfa' },
		{ token: 'predefined.go', foreground: '22d3ee' },

		// SQL specific
		{ token: 'keyword.sql', foreground: 'a78bfa' },
		{ token: 'operator.sql', foreground: 'a1a1aa' },
		{ token: 'predefined.sql', foreground: '22d3ee' },
	],
	colors: {
		// Editor core
		'editor.background': '#0a0a0b',
		'editor.foreground': '#f4f4f5',
		'editor.lineHighlightBackground': '#1f1f2340',
		'editor.lineHighlightBorder': '#00000000',
		'editor.selectionBackground': '#6366f133',
		'editor.inactiveSelectionBackground': '#6366f11a',
		'editor.selectionHighlightBackground': '#6366f11a',
		'editor.wordHighlightBackground': '#6366f11a',
		'editor.findMatchBackground': '#fbbf2444',
		'editor.findMatchHighlightBackground': '#fbbf2422',

		// Cursor — branded indigo
		'editorCursor.foreground': '#6366f1',
		'editorCursor.background': '#0a0a0b',

		// Line numbers
		'editorLineNumber.foreground': '#3f3f46',
		'editorLineNumber.activeForeground': '#a1a1aa',

		// Gutter
		'editorGutter.background': '#0a0a0b',
		'editorGutter.addedBackground': '#34d39966',
		'editorGutter.modifiedBackground': '#6366f166',
		'editorGutter.deletedBackground': '#f8717166',

		// Bracket matching — indigo glow
		'editorBracketMatch.background': '#6366f133',
		'editorBracketMatch.border': '#6366f1',

		// Bracket pair colorization — brand cascade
		'editorBracketHighlight.foreground1': '#6366f1',
		'editorBracketHighlight.foreground2': '#a78bfa',
		'editorBracketHighlight.foreground3': '#22d3ee',
		'editorBracketHighlight.foreground4': '#34d399',
		'editorBracketHighlight.foreground5': '#fbbf24',
		'editorBracketHighlight.foreground6': '#f87171',
		'editorBracketHighlight.unexpectedBracket.foreground': '#f87171',

		// Indent guides
		'editorIndentGuide.background': '#1f1f23',
		'editorIndentGuide.activeBackground': '#3f3f46',

		// Minimap
		'minimap.background': '#0a0a0b88',
		'minimapSlider.background': '#6366f122',
		'minimapSlider.hoverBackground': '#6366f133',
		'minimapSlider.activeBackground': '#6366f144',

		// Scrollbar — thin, subtle
		'scrollbar.shadow': '#00000000',
		'scrollbarSlider.background': '#3f3f4633',
		'scrollbarSlider.hoverBackground': '#3f3f4666',
		'scrollbarSlider.activeBackground': '#6366f144',

		// Widgets (autocomplete, hover, etc.)
		'editorWidget.background': '#0f0f10',
		'editorWidget.border': '#ffffff14',
		'editorSuggestWidget.background': '#0f0f10',
		'editorSuggestWidget.border': '#ffffff14',
		'editorSuggestWidget.selectedBackground': '#6366f133',
		'editorSuggestWidget.highlightForeground': '#6366f1',
		'editorSuggestWidget.focusHighlightForeground': '#818cf8',
		'editorHoverWidget.background': '#0f0f10',
		'editorHoverWidget.border': '#ffffff14',

		// Diff editor
		'diffEditor.insertedTextBackground': '#34d39922',
		'diffEditor.removedTextBackground': '#f8717122',
		'diffEditor.insertedLineBackground': '#34d39911',
		'diffEditor.removedLineBackground': '#f8717111',

		// Overview ruler
		'editorOverviewRuler.border': '#00000000',
		'editorOverviewRuler.errorForeground': '#f87171',
		'editorOverviewRuler.warningForeground': '#fbbf24',
		'editorOverviewRuler.infoForeground': '#60a5fa',

		// Sticky scroll
		'editorStickyScroll.background': '#0a0a0b',
		'editorStickyScrollHover.background': '#1f1f23',

		// Whitespace
		'editorWhitespace.foreground': '#1f1f23',
	},
};
