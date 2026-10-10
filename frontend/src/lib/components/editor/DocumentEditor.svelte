<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { editor, wordCount, type EditorBlock } from '$lib/stores/editor';
	import { contexts } from '$lib/stores/contexts';
	import Block from './Block.svelte';
	import BlockMenu from './BlockMenu.svelte';
	import type { Context } from '$lib/api';

	interface Props {
		context: Context;
		readonly?: boolean;
	}

	let { context, readonly = false }: Props = $props();

	let titleInput: HTMLInputElement | undefined = $state(undefined);
	let editorContainer: HTMLDivElement | undefined = $state(undefined);
	let autoSaveTimer: ReturnType<typeof setTimeout>;
	let title = $state(context.name);
	let coverImage = $state(context.cover_image);
	let icon = $state(context.icon);
	let showCoverInput = $state(false);
	let coverInputValue = $state('');

	// Initialize editor with context blocks
	onMount(() => {
		editor.initialize(context.blocks);
	});

	onDestroy(() => {
		if (autoSaveTimer) clearTimeout(autoSaveTimer);
	});

	// Auto-save with debounce
	$effect(() => {
		if ($editor.isDirty && !readonly) {
			if (autoSaveTimer) clearTimeout(autoSaveTimer);
			autoSaveTimer = setTimeout(async () => {
				await saveDocument();
			}, 2000);
		}
	});

	async function saveDocument() {
		if (readonly || $editor.isSaving) return;
		editor.setSaving(true);
		try {
			await contexts.updateBlocks(context.id, $editor.blocks, $wordCount);
			editor.markSaved();
		} catch (error) {
			console.error('Failed to save:', error);
			editor.setSaving(false);
		}
	}

	async function updateTitle() {
		if (readonly || title === context.name) return;
		try {
			await contexts.updateContext(context.id, { name: title });
		} catch (error) {
			console.error('Failed to update title:', error);
		}
	}

	async function updateCoverImage() {
		if (readonly) return;
		try {
			await contexts.updateContext(context.id, { cover_image: coverInputValue || undefined });
			coverImage = coverInputValue || null;
			showCoverInput = false;
			coverInputValue = '';
		} catch (error) {
			console.error('Failed to update cover:', error);
		}
	}

	async function updateIcon(newIcon: string) {
		if (readonly) return;
		try {
			await contexts.updateContext(context.id, { icon: newIcon || undefined });
			icon = newIcon || null;
		} catch (error) {
			console.error('Failed to update icon:', error);
		}
	}

	function handleTitleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			updateTitle();
			// Focus first block
			const firstBlock = document.querySelector('[data-block-id]') as HTMLElement;
			firstBlock?.focus();
		}
	}

	function handleEditorKeydown(e: KeyboardEvent) {
		// Handle Space key on empty line to open AI panel
		if (e.key === ' ' && !e.ctrlKey && !e.metaKey) {
			const target = e.target as HTMLElement;
			if (target.getAttribute('data-block-id')) {
				const blockId = target.getAttribute('data-block-id');
				const block = $editor.blocks.find((b) => b.id === blockId);
				if (block && block.content === '') {
					e.preventDefault();
					editor.showAIPanel();
				}
			}
		}
	}

	// Common emojis for icon picker
	const commonEmojis = ['📄', '📝', '📋', '📌', '📎', '📁', '💡', '⭐', '🎯', '✨', '🔥', '💎', '🚀', '📊', '🎨', '💻'];
</script>

<div class="document-editor h-full flex flex-col bg-white" bind:this={editorContainer}>
	<!-- Cover Image -->
	{#if coverImage}
		<div class="relative h-48 w-full group">
			<img src={coverImage} alt="Cover" class="w-full h-full object-cover" />
			{#if !readonly}
				<div class="absolute inset-0 bg-black/0 group-hover:bg-black/20 transition-colors flex items-center justify-center opacity-0 group-hover:opacity-100">
					<button
						onclick={() => { showCoverInput = true; coverInputValue = coverImage || ''; }}
						class="btn btn-secondary text-sm"
					>
						Change cover
					</button>
					<button
						onclick={() => updateCoverImage()}
						class="btn btn-secondary text-sm ml-2"
					>
						Remove
					</button>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Editor Content -->
	<div class="flex-1 overflow-y-auto">
		<div class="max-w-3xl mx-auto px-8 py-12">
			<!-- Add cover button -->
			{#if !coverImage && !readonly}
				<div class="mb-4 opacity-0 hover:opacity-100 transition-opacity">
					{#if showCoverInput}
						<div class="flex gap-2">
							<input
								type="text"
								bind:value={coverInputValue}
								placeholder="Paste image URL..."
								class="input input-square text-sm flex-1"
								onkeydown={(e) => e.key === 'Enter' && updateCoverImage()}
							/>
							<button onclick={() => updateCoverImage()} class="btn btn-primary text-sm">Add</button>
							<button onclick={() => { showCoverInput = false; coverInputValue = ''; }} class="btn btn-secondary text-sm">Cancel</button>
						</div>
					{:else}
						<button
							onclick={() => showCoverInput = true}
							class="text-sm text-gray-400 hover:text-gray-600"
						>
							+ Add cover
						</button>
					{/if}
				</div>
			{/if}

			<!-- Icon and Title -->
			<div class="flex items-start gap-4 mb-6">
				<!-- Icon Picker -->
				{#if !readonly}
					<div class="relative group">
						<button
							class="btn-pill btn-pill-ghost w-16 h-16 flex items-center justify-center text-4xl"
						>
							{icon || '📄'}
						</button>
						<div class="absolute top-full left-0 mt-1 bg-white rounded-lg shadow-xl border border-gray-200 p-2 grid grid-cols-4 gap-1 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all z-10">
							{#each commonEmojis as emoji}
								<button
									onclick={() => updateIcon(emoji)}
									class="w-8 h-8 flex items-center justify-center hover:bg-gray-100 rounded text-xl"
								>
									{emoji}
								</button>
							{/each}
						</div>
					</div>
				{:else if icon}
					<span class="text-4xl">{icon}</span>
				{/if}

				<!-- Title -->
				<div class="flex-1">
					{#if readonly}
						<h1 class="text-4xl font-bold text-gray-900">{title}</h1>
					{:else}
						<input
							bind:this={titleInput}
							bind:value={title}
							onblur={() => updateTitle()}
							onkeydown={handleTitleKeydown}
							placeholder="Untitled"
							class="w-full text-4xl font-bold text-gray-900 placeholder-gray-300 border-none outline-none bg-transparent"
						/>
					{/if}
				</div>
			</div>

			<!-- Placeholder hint -->
			{#if $editor.blocks.length === 1 && $editor.blocks[0].content === '' && !readonly}
				<p class="text-gray-400 text-sm mb-4">
					Press <kbd class="px-1.5 py-0.5 bg-gray-100 rounded text-xs">/</kbd> for commands
				</p>
			{/if}

			<!-- Blocks -->
			<div
				class="blocks-container"
				onkeydown={handleEditorKeydown}
				role="textbox"
				tabindex="-1"
			>
				{#each $editor.blocks as block, index (block.id)}
					<Block {block} {index} {readonly} />
				{/each}
			</div>
		</div>
	</div>

	<!-- Status Bar -->
	<div class="px-4 py-2 border-t border-gray-100 flex items-center justify-between text-xs text-gray-400">
		<div class="flex items-center gap-4">
			<span>{$wordCount} words</span>
			{#if $editor.isDirty}
				<span class="text-amber-500">Unsaved changes</span>
			{:else if $editor.isSaving}
				<span>Saving...</span>
			{:else if $editor.lastSavedAt}
				<span>Saved</span>
			{/if}
		</div>
		<div class="flex items-center gap-2">
			{#if !readonly}
				<button onclick={() => saveDocument()} class="hover:text-gray-600" disabled={!$editor.isDirty}>
					Save now
				</button>
			{/if}
		</div>
	</div>

	<!-- Slash Command Menu -->
	{#if $editor.showSlashMenu && $editor.slashMenuPosition}
		<BlockMenu />
	{/if}
</div>

<style>
	.document-editor :global(.ProseMirror) {
		outline: none;
	}
</style>
