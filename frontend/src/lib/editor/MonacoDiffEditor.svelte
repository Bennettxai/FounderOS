<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type * as Monaco from 'monaco-editor';
	import { detectLanguage } from './utils/language-detection';

	interface Props {
		originalValue: string;
		modifiedValue: string;
		filename: string;
		language?: string;
		renderSideBySide?: boolean;
	}

	let {
		originalValue,
		modifiedValue,
		filename,
		language = '',
		renderSideBySide = true,
	}: Props = $props();

	let editorContainer: HTMLElement;
	let diffEditor: Monaco.editor.IStandaloneDiffEditor | undefined;
	let monaco: typeof Monaco | undefined;
	let isReady = $state(false);

	let langId = $derived(language || detectLanguage(filename));

	onMount(async () => {
		const mod = await import('./monaco');
		monaco = mod.default;

		diffEditor = monaco.editor.createDiffEditor(editorContainer, {
			theme: 'businessos-dark',
			readOnly: true,
			automaticLayout: true,
			renderSideBySide,
			enableSplitViewResizing: true,

			// Typography (match MonacoEditor.svelte)
			fontFamily: '"JetBrains Mono", "Fira Code", "SF Mono", "Cascadia Code", monospace',
			fontLigatures: true,
			fontSize: 13,
			lineHeight: 22,
			letterSpacing: 0.3,

			// Layout
			padding: { top: 12, bottom: 12 },
			renderLineHighlight: 'none',
			roundedSelection: true,

			// Diff-specific
			renderIndicators: true,
			ignoreTrimWhitespace: true,

			// Minimap off for diff
			minimap: { enabled: false },

			// Scrollbar
			scrollbar: {
				verticalScrollbarSize: 8,
				horizontalScrollbarSize: 8,
				useShadows: false,
			},

			// Misc
			overviewRulerBorder: false,
			glyphMargin: false,
			folding: true,
		});

		const originalModel = monaco.editor.createModel(originalValue, langId);
		const modifiedModel = monaco.editor.createModel(modifiedValue, langId);

		diffEditor.setModel({
			original: originalModel,
			modified: modifiedModel,
		});

		isReady = true;
	});

	// React to content changes
	$effect(() => {
		if (diffEditor && monaco) {
			const model = diffEditor.getModel();
			if (model?.original && model.original.getValue() !== originalValue) {
				model.original.setValue(originalValue);
			}
			if (model?.modified && model.modified.getValue() !== modifiedValue) {
				model.modified.setValue(modifiedValue);
			}
		}
	});

	// React to language changes
	$effect(() => {
		if (diffEditor && monaco && langId) {
			const model = diffEditor.getModel();
			if (model?.original) monaco.editor.setModelLanguage(model.original, langId);
			if (model?.modified) monaco.editor.setModelLanguage(model.modified, langId);
		}
	});

	// React to side-by-side toggle
	$effect(() => {
		if (diffEditor) {
			diffEditor.updateOptions({ renderSideBySide });
		}
	});

	onDestroy(() => {
		const model = diffEditor?.getModel();
		model?.original?.dispose();
		model?.modified?.dispose();
		diffEditor?.dispose();
	});
</script>

<div
	class="diff-editor-container"
	class:is-ready={isReady}
	bind:this={editorContainer}
></div>

<style>
	.diff-editor-container {
		width: 100%;
		height: 100%;
		opacity: 0;
		transition: opacity 200ms ease;
	}

	.diff-editor-container.is-ready {
		opacity: 1;
	}
</style>
