<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type * as Monaco from 'monaco-editor';
	import { detectLanguage } from './utils/language-detection';

	interface Props {
		value?: string;
		filename?: string;
		language?: string;
		readonly?: boolean;
		onSave?: (value: string) => void;
		onChange?: (value: string) => void;
	}

	let {
		value = $bindable(''),
		filename = '',
		language = '',
		readonly = true,
		onSave,
		onChange,
	}: Props = $props();

	let editorContainer: HTMLElement;
	let editor: Monaco.editor.IStandaloneCodeEditor | undefined;
	let monaco: typeof Monaco | undefined;
	let isReady = $state(false);
	let containerWidth = $state(0);

	// Resolve language from filename or explicit prop
	let resolvedLanguage = $derived(language || detectLanguage(filename));

	let resizeObserver: ResizeObserver | undefined;

	onMount(async () => {
		// Track container width for responsive minimap
		resizeObserver = new ResizeObserver((entries) => {
			for (const entry of entries) {
				containerWidth = entry.contentRect.width;
			}
		});
		resizeObserver.observe(editorContainer);

		const mod = await import('./monaco');
		monaco = mod.default;

		editor = monaco.editor.create(editorContainer, {
			value,
			language: resolvedLanguage,
			theme: 'businessos-dark',
			readOnly: readonly,
			automaticLayout: true,

			// Typography
			fontFamily: '"JetBrains Mono", "Fira Code", "SF Mono", "Cascadia Code", monospace',
			fontLigatures: true,
			fontSize: 13,
			lineHeight: 22,
			letterSpacing: 0.3,

			// Cursor & animation
			cursorBlinking: 'smooth',
			cursorSmoothCaretAnimation: 'on',
			smoothScrolling: true,

			// Layout
			padding: { top: 16, bottom: 16 },
			renderLineHighlight: 'all',
			renderLineHighlightOnlyWhenFocus: false,
			roundedSelection: true,

			// Brackets
			bracketPairColorization: { enabled: true, independentColorPoolPerBracketType: true },
			guides: {
				bracketPairs: true,
				bracketPairsHorizontal: 'active',
				highlightActiveBracketPair: true,
				indentation: true,
				highlightActiveIndentation: true,
			},
			matchBrackets: 'always',

			// Minimap
			minimap: {
				enabled: true,
				renderCharacters: false,
				showSlider: 'mouseover',
				side: 'right',
				scale: 1,
				maxColumn: 120,
			},

			// Sticky scroll
			stickyScroll: { enabled: true, maxLineCount: 3 },

			// Scrollbar
			scrollbar: {
				verticalScrollbarSize: 8,
				horizontalScrollbarSize: 8,
				useShadows: false,
			},

			// Folding
			folding: true,
			foldingHighlight: true,
			showFoldingControls: 'mouseover',

			// Misc
			overviewRulerBorder: false,
			renderWhitespace: 'selection',
			colorDecorators: true,
			linkedEditing: true,
			hover: { enabled: true, delay: 300 },
			glyphMargin: true,
		});

		// Sync editor changes → bound value
		editor.onDidChangeModelContent((e) => {
			if (!e.isFlush) {
				const newValue = editor!.getValue();
				value = newValue;
				onChange?.(newValue);
			}
		});

		// Ctrl+S / Cmd+S save shortcut
		editor.addAction({
			id: 'businessos-save',
			label: 'Save File',
			keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS],
			run: () => {
				onSave?.(editor!.getValue());
			},
		});

		isReady = true;
	});

	// Sync external value changes → editor
	$effect(() => {
		if (editor && !editor.hasTextFocus() && editor.getValue() !== value) {
			editor.setValue(value);
		}
	});

	// React to language changes
	$effect(() => {
		if (editor && monaco && resolvedLanguage) {
			const model = editor.getModel();
			if (model) {
				monaco.editor.setModelLanguage(model, resolvedLanguage);
			}
		}
	});

	// React to readonly changes + auto-focus cursor on edit mode
	$effect(() => {
		if (editor) {
			editor.updateOptions({ readOnly: readonly });
			if (!readonly) {
				// Focus and position cursor when entering edit mode
				editor.focus();
				const pos = editor.getPosition();
				if (!pos || (pos.lineNumber === 1 && pos.column === 1)) {
					editor.setPosition({ lineNumber: 1, column: 1 });
				}
			}
		}
	});

	// Responsive minimap — disable under 1200px
	$effect(() => {
		if (editor) {
			editor.updateOptions({
				minimap: {
					enabled: containerWidth >= 1200,
					renderCharacters: false,
					showSlider: 'mouseover',
					side: 'right',
					scale: 1,
					maxColumn: 120,
				},
			});
		}
	});

	onDestroy(() => {
		resizeObserver?.disconnect();
		editor?.getModel()?.dispose();
		editor?.dispose();
	});

	/**
	 * Expose editor instance for parent components
	 */
	export function getEditor(): Monaco.editor.IStandaloneCodeEditor | undefined {
		return editor;
	}

	export function focus() {
		editor?.focus();
	}
</script>

<div
	class="monaco-editor-container"
	class:is-ready={isReady}
	bind:this={editorContainer}
></div>

<style>
	.monaco-editor-container {
		width: 100%;
		height: 100%;
		opacity: 0;
		transition: opacity 200ms ease;
	}

	.monaco-editor-container.is-ready {
		opacity: 1;
	}

	/* Override Monaco's default styles for glass integration */
	:global(.monaco-editor .suggest-widget) {
		border-radius: 8px !important;
	}

	:global(.monaco-editor .editor-widget) {
		border-radius: 6px !important;
	}
</style>
