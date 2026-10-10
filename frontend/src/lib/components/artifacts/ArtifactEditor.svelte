<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { editor, wordCount, type EditorBlock, createEmptyBlock } from '$lib/stores/editor';
	import BlockComponent from '$lib/components/editor/Block.svelte';
	import BlockMenu from '$lib/components/editor/BlockMenu.svelte';
	import { markdownToBlocks, blocksToMarkdown } from '$lib/utils/markdown-blocks';

	interface Props {
		artifact: {
			title: string;
			type: string;
			content: string;
		};
		onSave?: (content: string) => void;
		darkMode?: boolean;
	}

	let { artifact, onSave, darkMode = true }: Props = $props();

	let saveTimeout: ReturnType<typeof setTimeout> | null = null;

	// Initialize editor with artifact content converted to blocks
	onMount(() => {
		const blocks = markdownToBlocks(artifact.content || '');
		editor.initialize(blocks);
	});

	// Auto-save when editor becomes dirty
	$effect(() => {
		if ($editor.isDirty && onSave) {
			// Debounce save
			if (saveTimeout) clearTimeout(saveTimeout);
			saveTimeout = setTimeout(() => {
				const markdown = blocksToMarkdown($editor.blocks);
				onSave(markdown);
				editor.markSaved();
			}, 1500);
		}
	});

	// Cleanup on destroy
	onDestroy(() => {
		if (saveTimeout) clearTimeout(saveTimeout);
		editor.reset();
	});

	function addNewBlockAtEnd() {
		const lastBlock = $editor.blocks[$editor.blocks.length - 1];
		if (lastBlock) {
			editor.addBlockAfter(lastBlock.id, 'paragraph');
		}
	}

	function saveNow() {
		if (onSave) {
			const markdown = blocksToMarkdown($editor.blocks);
			onSave(markdown);
			editor.markSaved();
		}
	}
</script>

<div class="artifact-editor flex flex-col h-full" class:dark-mode={darkMode}>
	<!-- Editor Content -->
	<div class="flex-1 overflow-y-auto">
		<div class="max-w-none mx-auto px-4 py-6">
			<!-- Blocks -->
			<div class="blocks-container" role="textbox" tabindex="-1">
				{#each $editor.blocks as block, index (block.id)}
					<BlockComponent {block} {index} readonly={false} />
				{/each}
			</div>

			<!-- Click area to add new blocks -->
			<button
				onclick={addNewBlockAtEnd}
				class="btn-pill btn-pill-ghost w-full min-h-16 mt-4 text-left cursor-text group"
			>
				<span class="text-gray-500 opacity-0 group-hover:opacity-100 transition-opacity text-sm">
					Click to add a block, or press / for commands
				</span>
			</button>
		</div>
	</div>

	<!-- Status Bar -->
	<div class="px-4 py-2 border-t border-gray-700/50 flex items-center justify-between text-xs text-gray-400">
		<div class="flex items-center gap-4">
			<span>{$wordCount} words</span>
			<span>{$editor.blocks.length} blocks</span>
		</div>
		<button
			onclick={saveNow}
			class="btn-pill btn-pill-ghost"
			disabled={!$editor.isDirty}
		>
			{$editor.isSaving ? 'Saving...' : $editor.isDirty ? 'Save now' : 'Saved'}
		</button>
	</div>

	<!-- Slash Command Menu (with dark mode wrapper) -->
	{#if $editor.showSlashMenu && $editor.slashMenuPosition}
		<div class="block-menu-wrapper">
			<BlockMenu />
		</div>
	{/if}
</div>

<style>
	.artifact-editor {
		background-color: transparent;
	}

	/* Dark mode overrides for BlockMenu (slash commands) */
	.block-menu-wrapper :global(.bg-white) {
		background-color: #2c2c2e !important;
		border-color: #4b5563 !important;
	}

	.block-menu-wrapper :global(.border-gray-100) {
		border-color: #374151 !important;
	}

	.block-menu-wrapper :global(.text-gray-400) {
		color: #9ca3af !important;
	}

	.block-menu-wrapper :global(.text-gray-900) {
		color: #f5f5f7 !important;
	}

	.block-menu-wrapper :global(.text-gray-500) {
		color: #9ca3af !important;
	}

	.block-menu-wrapper :global(.text-gray-600) {
		color: #d1d5db !important;
	}

	.block-menu-wrapper :global(.bg-gray-100) {
		background-color: #374151 !important;
	}

	.block-menu-wrapper :global(.bg-gray-50) {
		background-color: #1c1c1e !important;
	}

	.block-menu-wrapper :global(.border-gray-200) {
		border-color: #4b5563 !important;
	}

	.block-menu-wrapper :global(.bg-gray-200) {
		background-color: #4b5563 !important;
	}

	.block-menu-wrapper :global(button:hover) {
		background-color: #374151 !important;
	}

	/* Dark mode overrides for blocks */
	.artifact-editor.dark-mode :global(.block-wrapper) {
		color: #f5f5f7;
	}

	.artifact-editor.dark-mode :global(.block-wrapper [contenteditable]) {
		color: #f5f5f7;
		caret-color: #f5f5f7;
	}

	.artifact-editor.dark-mode :global(.block-wrapper [contenteditable]:empty::before) {
		color: #6b7280;
	}

	/* Dark mode headings */
	.artifact-editor.dark-mode :global(.block-wrapper h1),
	.artifact-editor.dark-mode :global(.block-wrapper h2),
	.artifact-editor.dark-mode :global(.block-wrapper h3) {
		color: #ffffff;
	}

	/* Dark mode quote */
	.artifact-editor.dark-mode :global(.block-wrapper blockquote) {
		border-left-color: #4b5563;
		color: #9ca3af;
	}

	/* Dark mode code */
	.artifact-editor.dark-mode :global(.block-wrapper pre) {
		background-color: #1c1c1e;
		border-color: #374151;
	}

	.artifact-editor.dark-mode :global(.block-wrapper code) {
		color: #e5e7eb;
	}

	/* Dark mode divider */
	.artifact-editor.dark-mode :global(.block-wrapper hr) {
		border-color: #374151;
	}

	/* Dark mode drag handle */
	.artifact-editor.dark-mode :global(.block-wrapper .text-gray-400) {
		color: #6b7280;
	}

	.artifact-editor.dark-mode :global(.block-wrapper .hover\:bg-gray-100:hover) {
		background-color: #374151;
	}

	/* Dark mode todo checkbox */
	.artifact-editor.dark-mode :global(.block-wrapper input[type="checkbox"]) {
		background-color: #374151;
		border-color: #4b5563;
	}

	/* Dark mode list markers */
	.artifact-editor.dark-mode :global(.block-wrapper .text-gray-400) {
		color: #6b7280;
	}

	/* Dark mode page link */
	.artifact-editor.dark-mode :global(.block-wrapper a) {
		background-color: #374151;
		color: #e5e7eb;
	}

	.artifact-editor.dark-mode :global(.block-wrapper a:hover) {
		background-color: #4b5563;
	}
</style>
