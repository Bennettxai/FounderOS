<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import { fly } from 'svelte/transition';
	import { editor, wordCount, type EditorBlock, createEmptyBlock } from '$lib/stores/editor';
	import BlockComponent from '$lib/components/editor/Block.svelte';
	import BlockMenu from '$lib/components/editor/BlockMenu.svelte';
	import { markdownToBlocks, blocksToMarkdown } from '$lib/utils/markdown-blocks';
	import { contexts } from '$lib/stores/contexts';
	import type { Context } from '$lib/api/contexts/types';

	interface Props {
		document: Context | null;
		isNew?: boolean;
		parentId?: string;
		onClose: () => void;
		onSaved?: (doc: Context) => void;
		embedSuffix?: string;
		width?: number;
		onResize?: (width: number) => void;
	}

	let { document: contextDoc, isNew = false, parentId, onClose, onSaved, embedSuffix = '', width = 560, onResize }: Props = $props();

	// Resize handling
	let isResizing = $state(false);

	function startResize(e: MouseEvent) {
		isResizing = true;
		e.preventDefault();
		document.addEventListener('mousemove', handleResize);
		document.addEventListener('mouseup', stopResize);
	}

	function handleResize(e: MouseEvent) {
		if (!isResizing) return;
		const newWidth = window.innerWidth - e.clientX;
		const clampedWidth = Math.max(400, Math.min(900, newWidth));
		if (onResize) onResize(clampedWidth);
	}

	function stopResize() {
		isResizing = false;
		document.removeEventListener('mousemove', handleResize);
		document.removeEventListener('mouseup', stopResize);
	}

	let title = $state(contextDoc?.name || '');
	let saveTimeout: ReturnType<typeof setTimeout> | null = null;
	let isSaving = $state(false);
	let lastSaved = $state<Date | null>(null);
	let hasUnsavedChanges = $state(false);
	let titleInput: HTMLInputElement | null = $state(null);

	// Initialize editor with document content
	onMount(() => {
		if (contextDoc?.blocks && contextDoc.blocks.length > 0) {
			editor.initialize(contextDoc.blocks as EditorBlock[]);
		} else if (contextDoc?.content) {
			const blocks = markdownToBlocks(contextDoc.content);
			editor.initialize(blocks);
		} else {
			editor.initialize([createEmptyBlock()]);
		}

		// Focus title for new documents
		if (isNew && titleInput) {
			titleInput.select();
		}
	});

	// Auto-save when editor becomes dirty
	$effect(() => {
		if ($editor.isDirty) {
			hasUnsavedChanges = true;
			// Debounce save
			if (saveTimeout) clearTimeout(saveTimeout);
			saveTimeout = setTimeout(() => {
				saveDocument();
			}, 1500);
		}
	});

	// Cleanup on destroy
	onDestroy(() => {
		if (saveTimeout) clearTimeout(saveTimeout);
		// Save before closing if there are unsaved changes
		if (hasUnsavedChanges && contextDoc) {
			saveDocumentSync();
		}
		editor.reset();
	});

	async function saveDocument() {
		if (isSaving) return;
		isSaving = true;

		try {
			const markdown = blocksToMarkdown($editor.blocks);
			const blocks = $editor.blocks;

			if (isNew && !contextDoc) {
				// Create new document
				const newDoc = await contexts.createContext({
					name: title,
					type: 'document',
					parent_id: parentId,
					content: markdown,
					blocks: blocks
				});
				contextDoc = newDoc;
				if (onSaved) onSaved(newDoc);
			} else if (contextDoc) {
				// Update title/metadata separately from blocks
				if (title !== contextDoc.name) {
					await contexts.updateContext(contextDoc.id, { name: title });
				}
				// Use the proper blocks endpoint for saving block content
				await contexts.updateBlocks(contextDoc.id, blocks, $wordCount);
				contextDoc = { ...contextDoc, name: title, content: markdown, blocks };
				if (onSaved) onSaved(contextDoc);
			}

			editor.markSaved();
			hasUnsavedChanges = false;
			lastSaved = new Date();
		} catch (error) {
			console.error('Failed to save document:', error);
		} finally {
			isSaving = false;
		}
	}

	function saveDocumentSync() {
		// Synchronous save for cleanup - use proper blocks endpoint
		const blocks = $editor.blocks;

		if (contextDoc) {
			// Save title if changed
			if (title !== contextDoc.name) {
				contexts.updateContext(contextDoc.id, { name: title }).catch(console.error);
			}
			// Save blocks using proper endpoint
			contexts.updateBlocks(contextDoc.id, blocks, $wordCount).catch(console.error);
		}
	}

	function handleTitleChange(e: Event) {
		title = (e.target as HTMLInputElement).value;
		hasUnsavedChanges = true;
		// Debounce save
		if (saveTimeout) clearTimeout(saveTimeout);
		saveTimeout = setTimeout(() => {
			saveDocument();
		}, 1500);
	}

	function handleTitleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			// Focus first block
			const firstBlock = document.querySelector('[data-block-id]') as HTMLElement;
			if (firstBlock) {
				firstBlock.focus();
			}
		}
	}

	function addNewBlockAtEnd() {
		const lastBlock = $editor.blocks[$editor.blocks.length - 1];
		if (lastBlock) {
			editor.addBlockAfter(lastBlock.id, 'paragraph');
		}
	}

	function openFullPage() {
		if (contextDoc) {
			goto(`/knowledge/${contextDoc.id}${embedSuffix}`);
		}
	}

	function handleClose() {
		// Save before closing if there are unsaved changes
		if (hasUnsavedChanges) {
			saveDocument().then(() => {
				onClose();
			});
		} else {
			onClose();
		}
	}

	function formatTime(date: Date) {
		return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' });
	}
</script>

<!-- Side Peek Panel (no backdrop - true split view like Notion) -->
<div
	class="document-peek relative h-full bg-white dark:bg-[var(--bos-background-primary-color)] flex flex-col border-l border-gray-200 dark:border-gray-700 flex-shrink-0"
	style="width: {width}px;"
>
	<!-- Resize Handle (on left edge for dragging) -->
	<div
		class="absolute left-0 top-0 bottom-0 w-1.5 cursor-ew-resize hover:bg-blue-500/50 active:bg-blue-500 z-10 transition-colors group"
		onmousedown={startResize}
		role="separator"
		aria-label="Resize panel"
	>
		<div class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 w-1 h-8 rounded-full bg-gray-300 dark:bg-gray-600 opacity-0 group-hover:opacity-100 transition-opacity"></div>
	</div>
	<!-- Header -->
	<div class="px-4 py-3 border-b border-gray-200 dark:border-gray-700/50 flex items-center gap-3">
		<button
			onclick={handleClose}
			class="p-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500 dark:text-gray-400 transition-colors"
			title="Close"
		>
			<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>

		<input
			bind:this={titleInput}
			type="text"
			value={title}
			oninput={handleTitleChange}
			onkeydown={handleTitleKeydown}
			class="flex-1 text-lg font-semibold text-gray-900 dark:text-gray-100 bg-transparent border-0 focus:ring-0 focus:outline-none p-0 placeholder:text-gray-400 dark:placeholder:text-gray-500"
			placeholder="New page"
		/>

		<div class="flex items-center gap-2">
			{#if contextDoc}
				<button
					onclick={openFullPage}
					class="btn-pill btn-pill-secondary btn-pill-xs flex items-center gap-1.5"
					title="Open full page"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
					</svg>
					Full Page
				</button>
			{/if}
			<button
				onclick={() => saveDocument()}
				disabled={isSaving || !hasUnsavedChanges}
				class="btn-pill btn-pill-primary btn-pill-xs disabled:opacity-50"
			>
				{isSaving ? 'Saving...' : 'Save'}
			</button>
		</div>
	</div>

	<!-- Editor Content -->
	<div class="flex-1 overflow-y-auto">
		<div class="max-w-none mx-auto px-6 py-6">
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
				<span class="text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity text-sm">
					Click to add a block, or press / for commands
				</span>
			</button>
		</div>
	</div>

	<!-- Status Bar -->
	<div class="px-4 py-2 border-t border-gray-200 dark:border-gray-700/50 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
		<div class="flex items-center gap-4">
			<span>{$wordCount} words</span>
			<span>{$editor.blocks.length} blocks</span>
		</div>
		<div class="flex items-center gap-3">
			{#if lastSaved}
				<span>Saved at {formatTime(lastSaved)}</span>
			{:else if hasUnsavedChanges}
				<span class="text-amber-500">Unsaved changes</span>
			{:else}
				<span>No changes</span>
			{/if}
		</div>
	</div>

	<!-- Slash Command Menu (with theme wrapper) -->
	{#if $editor.showSlashMenu && $editor.slashMenuPosition}
		<div class="block-menu-wrapper">
			<BlockMenu />
		</div>
	{/if}
</div>

<style>
	/* Ensure proper focus styles for contenteditable elements */
	.blocks-container :global([contenteditable]:focus) {
		outline: none;
	}

	/* Light mode block styling (default) */
	.blocks-container :global(.block-wrapper) {
		color: #1f2937;
	}

	.blocks-container :global(.block-wrapper [contenteditable]) {
		color: #1f2937;
		caret-color: #1f2937;
	}

	.blocks-container :global(.block-wrapper [contenteditable]:empty::before) {
		color: #9ca3af;
	}

	/* Headings */
	.blocks-container :global(.block-wrapper h1),
	.blocks-container :global(.block-wrapper h2),
	.blocks-container :global(.block-wrapper h3) {
		color: #111827;
	}

	/* Quote */
	.blocks-container :global(.block-wrapper blockquote) {
		border-left-color: #d1d5db;
		color: #6b7280;
	}

	/* Code */
	.blocks-container :global(.block-wrapper pre) {
		background-color: #f3f4f6;
		border-color: #e5e7eb;
	}

	.blocks-container :global(.block-wrapper code) {
		color: #374151;
	}

	/* Divider */
	.blocks-container :global(.block-wrapper hr) {
		border-color: #e5e7eb;
	}

	/* Drag handle */
	.blocks-container :global(.block-wrapper .text-gray-400) {
		color: #9ca3af;
	}

	.blocks-container :global(.block-wrapper .hover\:bg-gray-100:hover) {
		background-color: #f3f4f6;
	}

	/* Todo checkbox */
	.blocks-container :global(.block-wrapper input[type="checkbox"]) {
		background-color: #ffffff;
		border-color: #d1d5db;
	}

	/* Dark mode block styling */
	:global(.dark) .blocks-container :global(.block-wrapper) {
		color: #f5f5f7;
	}

	:global(.dark) .blocks-container :global(.block-wrapper [contenteditable]) {
		color: #f5f5f7;
		caret-color: #f5f5f7;
	}

	:global(.dark) .blocks-container :global(.block-wrapper [contenteditable]:empty::before) {
		color: #6b7280;
	}

	/* Dark mode headings */
	:global(.dark) .blocks-container :global(.block-wrapper h1),
	:global(.dark) .blocks-container :global(.block-wrapper h2),
	:global(.dark) .blocks-container :global(.block-wrapper h3) {
		color: #ffffff;
	}

	/* Dark mode quote */
	:global(.dark) .blocks-container :global(.block-wrapper blockquote) {
		border-left-color: #4b5563;
		color: #9ca3af;
	}

	/* Dark mode code */
	:global(.dark) .blocks-container :global(.block-wrapper pre) {
		background-color: #0d0d0d;
		border-color: #374151;
	}

	:global(.dark) .blocks-container :global(.block-wrapper code) {
		color: #e5e7eb;
	}

	/* Dark mode divider */
	:global(.dark) .blocks-container :global(.block-wrapper hr) {
		border-color: #374151;
	}

	/* Dark mode drag handle */
	:global(.dark) .blocks-container :global(.block-wrapper .text-gray-400) {
		color: #6b7280;
	}

	:global(.dark) .blocks-container :global(.block-wrapper .hover\:bg-gray-100:hover) {
		background-color: #374151;
	}

	/* Dark mode todo checkbox */
	:global(.dark) .blocks-container :global(.block-wrapper input[type="checkbox"]) {
		background-color: #374151;
		border-color: #4b5563;
	}

	/* Dark mode overrides for BlockMenu (slash commands) */
	:global(.dark) .block-menu-wrapper :global(.bg-white) {
		background-color: #2c2c2e !important;
		border-color: #4b5563 !important;
	}

	:global(.dark) .block-menu-wrapper :global(.border-gray-100) {
		border-color: #374151 !important;
	}

	:global(.dark) .block-menu-wrapper :global(.text-gray-400) {
		color: #9ca3af !important;
	}

	:global(.dark) .block-menu-wrapper :global(.text-gray-900) {
		color: #f5f5f7 !important;
	}

	:global(.dark) .block-menu-wrapper :global(.text-gray-500) {
		color: #9ca3af !important;
	}

	:global(.dark) .block-menu-wrapper :global(.text-gray-600) {
		color: #d1d5db !important;
	}

	:global(.dark) .block-menu-wrapper :global(.bg-gray-100) {
		background-color: #374151 !important;
	}

	:global(.dark) .block-menu-wrapper :global(.bg-gray-50) {
		background-color: var(--bos-background-primary-color) !important;
	}

	:global(.dark) .block-menu-wrapper :global(.border-gray-200) {
		border-color: #4b5563 !important;
	}

	:global(.dark) .block-menu-wrapper :global(.bg-gray-200) {
		background-color: #4b5563 !important;
	}

	:global(.dark) .block-menu-wrapper :global(button:hover) {
		background-color: #374151 !important;
	}
</style>
