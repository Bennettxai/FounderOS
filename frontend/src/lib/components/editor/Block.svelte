<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { editor, type EditorBlock, type BlockType } from '$lib/stores/editor';
	import Block from './Block.svelte';
	import BlockCodeBlock from './BlockCodeBlock.svelte';
	import BlockToggle from './BlockToggle.svelte';
	import BlockTableOfContents from './BlockTableOfContents.svelte';
	import BlockCallout from './BlockCallout.svelte';
	import BlockBookmark from './BlockBookmark.svelte';
	import { selectBlockType as selectBlockTypeAction } from './blockPageActions';
	import { createDragHandlers } from './blockDragHandlers';
	import { handleBlockKeydown, handleDividerKeydown as handleDividerKeydownFn } from './blockKeyHandlers';

	interface Props {
		block: EditorBlock;
		index: number;
		readonly?: boolean;
		parentContextId?: string;
		onPageClick?: (pageId: string) => void;
	}

	let { block, index, readonly = false, parentContextId, onPageClick }: Props = $props();

	let blockElement: HTMLElement | null = $state(null);

	// Drag and drop state
	let isDragging = $state(false);
	let isDragOver = $state(false);
	let dragOverPosition = $state<'above' | 'below' | null>(null);

	// Drag handlers (logic extracted to blockDragHandlers.ts)
	const { handleDragStart, handleDragEnd, handleDragOver, handleDragLeave, handleDrop } =
		createDragHandlers(
			block.id,
			() => dragOverPosition,
			(v) => { isDragging = v; },
			(v) => { isDragOver = v; },
			(v) => { dragOverPosition = v; }
		);

	// Track if we're initializing to prevent immediate updates
	let initialized = false;

	onMount(() => {
		if (blockElement && block.content) {
			blockElement.innerText = block.content;
		}
		initialized = true;
	});

	// Update element content when block changes (only if not focused)
	$effect(() => {
		if (blockElement && initialized && document.activeElement !== blockElement) {
			blockElement.innerText = block.content || '';
		}
	});

	// Focus this block when it becomes the focused block in the store
	$effect(() => {
		if ($editor.focusedBlockId === block.id && blockElement && document.activeElement !== blockElement) {
			blockElement.focus();
			const range = document.createRange();
			const sel = window.getSelection();
			if (blockElement.childNodes.length > 0) {
				range.setStartAfter(blockElement.lastChild!);
			} else {
				range.setStart(blockElement, 0);
			}
			range.collapse(true);
			sel?.removeAllRanges();
			sel?.addRange(range);
		}
	});

	// Handle pending block type selection from BlockMenu
	$effect(() => {
		const pending = $editor.pendingBlockTypeSelection;
		if (pending && pending.blockId === block.id) {
			editor.clearPendingBlockTypeSelection();
			selectBlockType(pending.type, pending.properties);
		}
	});

	function handleFocus() {
		editor.setFocusedBlock(block.id);
	}

	function handleBlur(e: FocusEvent) {
		const relatedTarget = e.relatedTarget as HTMLElement;
		if (relatedTarget?.closest('[data-slash-menu]')) {
			return;
		}

		if ($editor.showSlashMenu) {
			if (blockElement) {
				const newContent = blockElement.innerText || '';
				if (newContent !== block.content) {
					editor.updateBlock(block.id, newContent);
				}
			}
			return;
		}

		if (blockElement) {
			const newContent = blockElement.innerText || '';
			if (newContent !== block.content) {
				editor.updateBlock(block.id, newContent);
			}
		}
	}

	function handleInput(e: Event) {
		if (!blockElement) return;
		const content = blockElement.innerText || '';

		if (content === '/' || content.startsWith('/')) {
			editor.setFocusedBlock(block.id);
			const rect = blockElement.getBoundingClientRect();
			editor.showSlashMenu({ x: rect.left, y: rect.bottom + 4 });
			editor.setSlashMenuQuery(content.slice(1));
		} else {
			editor.hideSlashMenu();
		}

		editor.updateBlock(block.id, content);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (!blockElement) return;
		handleBlockKeydown(e, block, index, blockElement, () => $editor.blocks, $editor.showSlashMenu);
	}

	async function selectBlockType(type: BlockType, properties?: Record<string, unknown>) {
		await selectBlockTypeAction(block.id, type, blockElement, parentContextId, properties);
	}

	function handlePageClick(pageId: string) {
		if (onPageClick) {
			onPageClick(pageId);
		} else {
			goto(`/knowledge/${pageId}`);
		}
	}

	function handleTodoToggle(e: Event) {
		e.stopPropagation();
		if (block.type === 'todo') {
			editor.toggleTodo(block.id);
		}
	}

	function getPlaceholder(): string {
		if (block.type === 'heading1') return 'Heading 1';
		if (block.type === 'heading2') return 'Heading 2';
		if (block.type === 'heading3') return 'Heading 3';
		if (block.type === 'quote') return 'Quote';
		if (block.type === 'code') return 'Code';
		if (block.type === 'todo') return 'To-do';
		if (block.type === 'bulletList') return 'List item';
		if (block.type === 'numberedList') return 'List item';
		if (block.type === 'toggle') return 'Toggle header';
		if (block.type === 'page') return 'Page title';
		return "Type '/' for commands...";
	}

	let isEmpty = $derived(!block.content || block.content === '');

	// Handle keydown on divider blocks (which aren't contenteditable)
	function handleDividerKeydown(e: KeyboardEvent) {
		handleDividerKeydownFn(e, block, index, () => $editor.blocks);
	}
</script>

<div
	class="block-wrapper group relative py-0.5"
	class:dragging={isDragging}
	data-block-index={index}
	ondragover={handleDragOver}
	ondragleave={handleDragLeave}
	ondrop={handleDrop}
>
	<!-- Block handle (drag/menu) -->
	{#if !readonly}
		<div class="absolute -left-8 top-1/2 -translate-y-1/2 opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-0.5">
			<div
				draggable="true"
				ondragstart={handleDragStart}
				ondragend={handleDragEnd}
				class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400 dark:text-gray-500 cursor-grab active:cursor-grabbing"
				title="Drag to move"
				role="button"
				tabindex="-1"
			>
				<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
					<circle cx="9" cy="6" r="1.5"/>
					<circle cx="15" cy="6" r="1.5"/>
					<circle cx="9" cy="12" r="1.5"/>
					<circle cx="15" cy="12" r="1.5"/>
					<circle cx="9" cy="18" r="1.5"/>
					<circle cx="15" cy="18" r="1.5"/>
				</svg>
			</div>
		</div>
	{/if}

	<!-- Block content based on type -->
	{#if block.type === 'divider'}
		<div
			tabindex={readonly ? -1 : 0}
			data-block-id={block.id}
			onfocus={handleFocus}
			onkeydown={handleDividerKeydown}
			class="py-2 outline-none group cursor-pointer"
		>
			<hr class="border-gray-300 dark:border-gray-600 group-focus:border-blue-400 transition-colors" />
		</div>
	{:else if block.type === 'page'}
		<button
			onclick={() => block.properties?.pageId && handlePageClick(block.properties.pageId as string)}
			class="page-link group inline-flex items-center gap-1.5 py-0.5 text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-800 rounded transition-colors text-left"
		>
			<svg class="w-[18px] h-[18px] text-gray-400 dark:text-gray-500 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
			</svg>
			<span class="underline decoration-gray-300 dark:decoration-gray-600 underline-offset-2">{block.content || 'New page'}</span>
		</button>
	{:else if block.type === 'heading1'}
		<h1
			bind:this={blockElement}
			contenteditable={!readonly}
			data-block-id={block.id}
			data-placeholder={getPlaceholder()}
			onfocus={handleFocus}
			onblur={handleBlur}
			oninput={handleInput}
			onkeydown={handleKeydown}
			class="text-3xl font-bold text-gray-900 dark:text-gray-100 outline-none min-h-[1.2em] block-editable"
			class:is-empty={isEmpty}
		></h1>
	{:else if block.type === 'heading2'}
		<h2
			bind:this={blockElement}
			contenteditable={!readonly}
			data-block-id={block.id}
			data-placeholder={getPlaceholder()}
			onfocus={handleFocus}
			onblur={handleBlur}
			oninput={handleInput}
			onkeydown={handleKeydown}
			class="text-2xl font-semibold text-gray-900 dark:text-gray-100 outline-none min-h-[1.2em] block-editable"
			class:is-empty={isEmpty}
		></h2>
	{:else if block.type === 'heading3'}
		<h3
			bind:this={blockElement}
			contenteditable={!readonly}
			data-block-id={block.id}
			data-placeholder={getPlaceholder()}
			onfocus={handleFocus}
			onblur={handleBlur}
			oninput={handleInput}
			onkeydown={handleKeydown}
			class="text-xl font-semibold text-gray-800 dark:text-gray-200 outline-none min-h-[1.2em] block-editable"
			class:is-empty={isEmpty}
		></h3>
	{:else if block.type === 'bulletList'}
		<div class="flex items-start gap-2">
			<span class="mt-2 w-1.5 h-1.5 rounded-full bg-gray-400 dark:bg-gray-500 flex-shrink-0"></span>
			<div
				bind:this={blockElement}
				contenteditable={!readonly}
				data-block-id={block.id}
				data-placeholder={getPlaceholder()}
				onfocus={handleFocus}
				onblur={handleBlur}
				oninput={handleInput}
				onkeydown={handleKeydown}
				class="flex-1 text-gray-700 dark:text-gray-300 outline-none min-h-[1.5em] block-editable"
				class:is-empty={isEmpty}
			></div>
		</div>
	{:else if block.type === 'numberedList'}
		<div class="flex items-start gap-2">
			<span class="w-5 h-5 rounded-full bg-blue-100 dark:bg-blue-900/50 text-blue-600 dark:text-blue-400 text-xs flex items-center justify-center flex-shrink-0">
				{index + 1}
			</span>
			<div
				bind:this={blockElement}
				contenteditable={!readonly}
				data-block-id={block.id}
				data-placeholder={getPlaceholder()}
				onfocus={handleFocus}
				onblur={handleBlur}
				oninput={handleInput}
				onkeydown={handleKeydown}
				class="flex-1 text-gray-700 dark:text-gray-300 outline-none min-h-[1.5em] block-editable"
				class:is-empty={isEmpty}
			></div>
		</div>
	{:else if block.type === 'todo'}
		<div class="flex items-start gap-2">
			<button
				onclick={handleTodoToggle}
				class="w-4 h-4 mt-1 rounded border flex items-center justify-center transition-colors flex-shrink-0
					{block.properties?.checked
						? 'bg-blue-500 border-blue-500'
						: 'border-gray-500 hover:border-blue-400'}"
				tabindex="-1"
			>
				{#if block.properties?.checked}
					<svg class="w-3 h-3 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
					</svg>
				{/if}
			</button>
			<div
				bind:this={blockElement}
				contenteditable={!readonly}
				data-block-id={block.id}
				data-placeholder={getPlaceholder()}
				onfocus={handleFocus}
				onblur={handleBlur}
				oninput={handleInput}
				onkeydown={handleKeydown}
				class="flex-1 outline-none min-h-[1.5em] block-editable
					{block.properties?.checked ? 'line-through text-gray-500' : 'text-gray-700 dark:text-gray-300'}"
				class:is-empty={isEmpty}
			></div>
		</div>
	{:else if block.type === 'quote'}
		<blockquote class="border-l-4 border-gray-300 dark:border-gray-600 pl-4 py-1">
			<div
				bind:this={blockElement}
				contenteditable={!readonly}
				data-block-id={block.id}
				data-placeholder={getPlaceholder()}
				onfocus={handleFocus}
				onblur={handleBlur}
				oninput={handleInput}
				onkeydown={handleKeydown}
				class="text-gray-600 dark:text-gray-400 italic outline-none min-h-[1.5em] block-editable"
				class:is-empty={isEmpty}
			></div>
		</blockquote>
	{:else if block.type === 'code'}
		<BlockCodeBlock
			{block}
			{readonly}
			{isEmpty}
			onFocus={handleFocus}
			onBlur={handleBlur}
			onInput={handleInput}
			onKeydown={handleKeydown}
			onBindElement={(el) => { blockElement = el; }}
		/>
	{:else if block.type === 'callout'}
		<BlockCallout
			{block}
			{readonly}
			{isEmpty}
			onFocus={handleFocus}
			onBlur={handleBlur}
			onInput={handleInput}
			onKeydown={handleKeydown}
			onBindElement={(el) => { blockElement = el; }}
		/>
	{:else if block.type === 'toggle'}
		<BlockToggle
			{block}
			{readonly}
			{isEmpty}
			{parentContextId}
			{onPageClick}
			onFocus={handleFocus}
			onBlur={handleBlur}
			onInput={handleInput}
			onKeydown={handleKeydown}
			onBindElement={(el) => { blockElement = el; }}
		/>
	{:else if block.type === 'tableOfContents'}
		<BlockTableOfContents />
	{:else if block.type === 'columns'}
		<div class="columns-block grid grid-cols-2 gap-4 p-2 rounded-lg border border-dashed border-gray-300 dark:border-gray-600">
			<div class="min-h-[100px] rounded bg-gray-50 dark:bg-[var(--bos-v2-layer-background-primary)] p-3 flex items-center justify-center">
				<span class="text-xs text-gray-400 dark:text-gray-500">Column 1 - Click to add blocks</span>
			</div>
			<div class="min-h-[100px] rounded bg-gray-50 dark:bg-[var(--bos-v2-layer-background-primary)] p-3 flex items-center justify-center">
				<span class="text-xs text-gray-400 dark:text-gray-500">Column 2 - Click to add blocks</span>
			</div>
		</div>
	{:else if block.type === 'bookmark'}
		<BlockBookmark {block} />
	{:else}
		<!-- Default paragraph -->
		<p
			bind:this={blockElement}
			contenteditable={!readonly}
			data-block-id={block.id}
			data-placeholder={getPlaceholder()}
			onfocus={handleFocus}
			onblur={handleBlur}
			oninput={handleInput}
			onkeydown={handleKeydown}
			class="text-gray-800 dark:text-gray-100 outline-none min-h-[1.5em] block-editable"
			class:is-empty={isEmpty}
		></p>
	{/if}

</div>

<style>
	.block-editable {
		position: relative;
	}

	.block-editable:focus {
		outline: none;
	}

	/* Only show placeholder on focus for paragraphs, or when empty for headings */
	.block-editable.is-empty:focus:before,
	h1.block-editable.is-empty:before,
	h2.block-editable.is-empty:before,
	h3.block-editable.is-empty:before {
		content: attr(data-placeholder);
		color: var(--bos-status-neutral);
		pointer-events: none;
		position: absolute;
		font-style: normal;
		font-weight: normal;
	}

	/* Dark mode text selection */
	.block-editable::selection {
		background-color: rgba(59, 130, 246, 0.4);
		color: var(--bos-surface-on-color);
	}

	/* For webkit browsers (Chrome, Safari) */
	.block-editable::-webkit-selection {
		background-color: rgba(59, 130, 246, 0.4);
		color: var(--bos-surface-on-color);
	}

	/* Ensure no weird background on focus */
	.block-editable:focus {
		background-color: transparent;
	}

	/* Drag and drop visual indicators */
	.block-wrapper {
		position: relative;
	}

	.block-wrapper.dragging {
		opacity: 0.4;
	}

	.block-wrapper.drag-over-above::before {
		content: '';
		position: absolute;
		top: -2px;
		left: 0;
		right: 0;
		height: 3px;
		background: linear-gradient(90deg, var(--bos-status-info), var(--bos-nav-active));
		border-radius: 2px;
		z-index: 10;
		box-shadow: 0 0 6px rgba(59, 130, 246, 0.5);
	}

	.block-wrapper.drag-over-below::after {
		content: '';
		position: absolute;
		bottom: -2px;
		left: 0;
		right: 0;
		height: 3px;
		background: linear-gradient(90deg, var(--bos-status-info), var(--bos-nav-active));
		border-radius: 2px;
		z-index: 10;
		box-shadow: 0 0 6px rgba(59, 130, 246, 0.5);
	}

	/* Add small dot indicators at the edges for extra visibility */
	.block-wrapper.drag-over-above::before,
	.block-wrapper.drag-over-below::after {
		animation: pulse-indicator 0.8s ease-in-out infinite alternate;
	}

	@keyframes pulse-indicator {
		from {
			opacity: 0.7;
		}
		to {
			opacity: 1;
		}
	}
</style>
