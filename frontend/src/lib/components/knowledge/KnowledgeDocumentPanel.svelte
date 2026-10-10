<script lang="ts">
	import { tick } from 'svelte';
	import { onDestroy } from 'svelte';
	import { Popover } from 'bits-ui';
	import type { Memory } from '$lib/api/memory/types';
	import type { ContextListItem } from '$lib/api/contexts/types';
	import { updateMemory, deleteMemory } from '$lib/api/memory';
	import { updateContext, archiveContext } from '$lib/api/contexts';
	import { editor, wordCount, type EditorBlock, createEmptyBlock } from '$lib/stores/editor';
	import BlockComponent from '$lib/components/editor/Block.svelte';
	import BlockMenu from '$lib/components/editor/BlockMenu.svelte';
	import { markdownToBlocks, blocksToMarkdown } from '$lib/utils/markdown-blocks';

	// Menu state
	let showActionsMenu = $state(false);

	interface Props {
		selectedMemory?: Memory | null;
		selectedDocument?: ContextListItem | null;
		relatedMemories?: Memory[];
		onClose?: () => void;
		onEdit?: () => void;
		onAddToChat?: () => void;
		onSave?: (memory: Memory) => void;
		onDelete?: (id: string) => void;
		onSelectRelated?: (memory: Memory) => void;
		onExpand?: () => void;
	}

	let {
		selectedMemory = null,
		selectedDocument = null,
		relatedMemories = [],
		onClose,
		onEdit,
		onAddToChat,
		onSave,
		onDelete,
		onSelectRelated,
		onExpand
	}: Props = $props();

	// Editing states
	let editedTitle = $state('');
	let editedCoverImage = $state<string | null>(null);
	let titleInput: HTMLInputElement | null = $state(null);

	// File input reference
	let coverImageInput = $state<HTMLInputElement | null>(null);
	let showCoverInput = $state(false);
	let coverInputValue = $state('');

	// Saving state
	let isSaving = $state(false);
	let hasChanges = $state(false);
	let saveTimeout: ReturnType<typeof setTimeout> | null = null;
	let currentMemoryId = $state<string | null>(null);

	// Memory type icons
	const typeIcons: Record<string, string> = {
		'fact': '📋',
		'preference': '💜',
		'decision': '🔑',
		'event': '📅',
		'learning': '📖',
		'context': '📄',
		'relationship': '👥',
		'episode': '💬'
	};

	// Initialize edited values and blocks when memory changes
	$effect(() => {
		const newId = selectedMemory?.id || null;
		if (newId && newId !== currentMemoryId) {
			currentMemoryId = newId;
			editedTitle = selectedMemory?.title || '';
			editedCoverImage = selectedMemory?.cover_image || null;
			hasChanges = false;

			// Initialize block editor with content
			const content = selectedMemory?.content || '';
			if (content) {
				const blocks = markdownToBlocks(content);
				editor.initialize(blocks);
			} else {
				editor.initialize([createEmptyBlock()]);
			}

			// Focus title on load
			tick().then(() => {
				titleInput?.focus();
			});
		}
	});

	// Auto-save when editor becomes dirty
	$effect(() => {
		if ($editor.isDirty && currentMemoryId) {
			hasChanges = true;
			// Debounce save
			if (saveTimeout) clearTimeout(saveTimeout);
			saveTimeout = setTimeout(() => {
				saveChanges();
			}, 2000);
		}
	});

	// Cleanup on destroy
	onDestroy(() => {
		if (saveTimeout) clearTimeout(saveTimeout);
		editor.reset();
	});

	// Format date nicely
	function formatDate(dateString: string | null): string {
		if (!dateString) return '';
		const date = new Date(dateString);
		return date.toLocaleDateString('en-US', {
			month: 'long',
			day: 'numeric',
			year: 'numeric'
		});
	}

	// Get icon for memory type
	function getTypeIcon(type: string | null | undefined): string {
		return typeIcons[type || 'context'] || '📄';
	}

	// Handle title changes
	function handleTitleBlur() {
		if (editedTitle !== selectedMemory?.title) {
			hasChanges = true;
			// Trigger save
			if (saveTimeout) clearTimeout(saveTimeout);
			saveTimeout = setTimeout(() => saveChanges(), 500);
		}
	}

	function handleTitleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			// Focus first block
			const firstBlock = document.querySelector('[data-block-id]') as HTMLElement;
			firstBlock?.focus();
		}
	}

	// Cover image handling
	function handleCoverImageUpload(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		if (!file.type.startsWith('image/')) {
			alert('Please select an image file');
			return;
		}

		if (file.size > 5 * 1024 * 1024) {
			alert('Image must be less than 5MB');
			return;
		}

		const reader = new FileReader();
		reader.onload = (e) => {
			editedCoverImage = e.target?.result as string;
			hasChanges = true;
		};
		reader.readAsDataURL(file);
	}

	function addCoverFromUrl() {
		if (coverInputValue) {
			editedCoverImage = coverInputValue;
			hasChanges = true;
		}
		showCoverInput = false;
		coverInputValue = '';
	}

	function removeCoverImage() {
		editedCoverImage = null;
		hasChanges = true;
	}

	// Save all changes
	async function saveChanges() {
		if (!selectedMemory || (!hasChanges && !$editor.isDirty)) return;

		isSaving = true;
		try {
			// Convert blocks to markdown for storage
			const editedContent = blocksToMarkdown($editor.blocks);

			// Check if this is a context (source_type === 'context') or actual memory
			if (selectedMemory.source_type === 'context') {
				const updatedContext = await updateContext(selectedMemory.id, {
					name: editedTitle,
					content: editedContent,
					cover_image: editedCoverImage || undefined
				});

				const updatedMemory: Memory = {
					...selectedMemory,
					title: updatedContext.name || editedTitle,
					content: updatedContext.content || editedContent,
					cover_image: editedCoverImage || undefined
				};

				editor.markSaved();
				hasChanges = false;
				onSave?.(updatedMemory);
			} else {
				const updatedMemory = await updateMemory(selectedMemory.id, {
					title: editedTitle,
					content: editedContent,
					cover_image: editedCoverImage || undefined
				});

				editor.markSaved();
				hasChanges = false;
				onSave?.(updatedMemory);
			}
		} catch (err) {
			console.error('Failed to save changes:', err);
		} finally {
			isSaving = false;
		}
	}

	// Delete memory
	async function handleDelete() {
		if (!selectedMemory) return;
		if (!confirm('Are you sure you want to delete this?')) return;

		try {
			if (selectedMemory.source_type === 'context') {
				await archiveContext(selectedMemory.id);
			} else {
				await deleteMemory(selectedMemory.id);
			}
			onDelete?.(selectedMemory.id);
			onClose?.();
		} catch (err) {
			console.error('Failed to delete:', err);
		}
	}

	// Handle keydown for Space on empty block (AI panel)
	function handleEditorKeydown(e: KeyboardEvent) {
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
</script>

<div class="document-panel">
	{#if selectedMemory}
		<!-- Hidden file input -->
		<input
			type="file"
			accept="image/*"
			bind:this={coverImageInput}
			onchange={handleCoverImageUpload}
			class="hidden"
		/>

		<!-- Cover Image -->
		{#if editedCoverImage}
			<div class="cover-image group">
				<img src={editedCoverImage} alt="Cover" />
				<div class="cover-overlay">
					<button onclick={() => coverImageInput?.click()} class="cover-btn">
						Change cover
					</button>
					<button onclick={removeCoverImage} class="btn-pill btn-pill-ghost cover-btn">
						Remove
					</button>
				</div>
			</div>
		{/if}

		<!-- Scrollable Content -->
		<div class="content-scroll">
			<div class="content-container">
				<!-- Add cover button (shown on hover when no cover) -->
				{#if !editedCoverImage}
					<div class="add-cover-area">
						{#if showCoverInput}
							<div class="cover-input-row">
								<input
									type="text"
									bind:value={coverInputValue}
									placeholder="Paste image URL..."
									class="cover-url-input"
									onkeydown={(e) => e.key === 'Enter' && addCoverFromUrl()}
								/>
								<button onclick={addCoverFromUrl} class="btn-pill btn-pill-ghost cover-add-btn">Add</button>
								<button onclick={() => { showCoverInput = false; coverInputValue = ''; }} class="cover-cancel-btn">Cancel</button>
							</div>
						{:else}
							<button onclick={() => showCoverInput = true} class="add-cover-btn">
								+ Add cover
							</button>
						{/if}
					</div>
				{/if}

				<!-- Icon and Title Row -->
				<div class="title-row">
					<span class="type-icon">{getTypeIcon(selectedMemory.memory_type)}</span>
					<input
						bind:this={titleInput}
						bind:value={editedTitle}
						onblur={handleTitleBlur}
						onkeydown={handleTitleKeydown}
						placeholder="Untitled"
						class="title-input"
					/>
				</div>

				<!-- Meta info -->
				<div class="meta-row">
					<span class="meta-date">{formatDate(selectedMemory.created_at)}</span>
				</div>

				<!-- Slash command hint -->
				{#if $editor.blocks.length === 1 && $editor.blocks[0].content === ''}
					<p class="slash-hint">
						Press <kbd>/</kbd> for commands
					</p>
				{/if}

				<!-- Blocks Editor -->
				<div
					class="blocks-container"
					onkeydown={handleEditorKeydown}
					role="textbox"
					tabindex="-1"
				>
					{#each $editor.blocks as block, index (block.id)}
						<BlockComponent {block} {index} />
					{/each}
				</div>

				<!-- Related Memories -->
				{#if relatedMemories.length > 0}
					<div class="related-section">
						<h4 class="related-title">Related</h4>
						<div class="related-list">
							{#each relatedMemories as related}
								<button
									class="btn-pill btn-pill-ghost related-item"
									onclick={() => onSelectRelated?.(related)}
								>
									<span class="related-icon">{getTypeIcon(related.memory_type)}</span>
									<span class="related-name">{related.title || 'Untitled'}</span>
								</button>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- Status Bar -->
		<div class="status-bar">
			<div class="status-left">
				<span>{$wordCount} words</span>
				{#if $editor.isDirty || hasChanges}
					<span class="status-unsaved">Unsaved</span>
				{:else if isSaving}
					<span>Saving...</span>
				{:else}
					<span class="status-saved">Saved</span>
				{/if}
			</div>
			<div class="status-right">
				<!-- 3-dot menu -->
				<Popover.Root bind:open={showActionsMenu}>
					<Popover.Trigger class="menu-trigger">
						<svg class="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
							<circle cx="12" cy="5" r="2"/>
							<circle cx="12" cy="12" r="2"/>
							<circle cx="12" cy="19" r="2"/>
						</svg>
					</Popover.Trigger>
					<Popover.Content class="actions-menu" side="top" sideOffset={8} align="end">
						<div class="actions-row">
							<button class="btn-pill btn-pill-ghost action-pill" onclick={() => { onAddToChat?.(); showActionsMenu = false; }}>
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
								</svg>
								Chat
							</button>
							<button class="btn-pill btn-pill-ghost action-pill" onclick={() => { onExpand?.(); showActionsMenu = false; }}>
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
								</svg>
								Expand
							</button>
							<button class="btn-pill btn-pill-ghost action-pill delete" onclick={() => { handleDelete(); showActionsMenu = false; }}>
								<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
								</svg>
								Delete
							</button>
						</div>
					</Popover.Content>
				</Popover.Root>
				<!-- Close button -->
				<button class="btn-pill btn-pill-ghost close-btn" onclick={onClose} aria-label="Close panel">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>

		<!-- Slash Command Menu -->
		{#if $editor.showSlashMenu && $editor.slashMenuPosition}
			<BlockMenu />
		{/if}

	{:else if selectedDocument}
		<!-- Document View -->
		<div class="content-scroll">
			<div class="content-container">
				<div class="title-row">
					<span class="type-icon">📄</span>
					<h1 class="title-display">{selectedDocument.name}</h1>
				</div>
				<div class="meta-row">
					<span class="meta-date">{formatDate(selectedDocument.updated_at)}</span>
				</div>
				<div class="document-action">
					<button onclick={onEdit} class="btn-pill btn-pill-ghost open-btn">
						Open Document
					</button>
				</div>
			</div>
		</div>

	{:else}
		<!-- Empty State -->
		<div class="empty-state">
			<span class="empty-icon">📄</span>
			<h4>Select a Bubble</h4>
			<p>Click on a bubble in the knowledge graph to view and edit its contents.</p>
		</div>
	{/if}
</div>

<style>
	.document-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: white;
		border-radius: 16px;
		overflow: hidden;
		box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
	}

	.hidden {
		display: none;
	}

	/* Cover Image */
	.cover-image {
		position: relative;
		height: 180px;
		width: 100%;
		flex-shrink: 0;
	}

	.cover-image img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.cover-overlay {
		position: absolute;
		inset: 0;
		background: rgba(0, 0, 0, 0);
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		opacity: 0;
		transition: all 0.2s;
	}

	.cover-image:hover .cover-overlay {
		background: rgba(0, 0, 0, 0.3);
		opacity: 1;
	}

	.cover-btn {
		padding: 8px 16px;
		background: white;
		border: none;
		border-radius: 6px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s;
	}

	.cover-btn:hover {
		background: #f5f5f5;
	}

	/* Scrollable Content */
	.content-scroll {
		flex: 1;
		overflow-y: auto;
	}

	.content-container {
		max-width: 700px;
		margin: 0 auto;
		padding: 48px 56px;
	}

	/* Add Cover Area */
	.add-cover-area {
		margin-bottom: 16px;
		opacity: 0;
		transition: opacity 0.15s;
	}

	.content-container:hover .add-cover-area {
		opacity: 1;
	}

	.add-cover-btn {
		font-size: 13px;
		color: #9ca3af;
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px 0;
	}

	.add-cover-btn:hover {
		color: #6b7280;
	}

	.cover-input-row {
		display: flex;
		gap: 8px;
	}

	.cover-url-input {
		flex: 1;
		padding: 8px 12px;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		font-size: 13px;
		outline: none;
	}

	.cover-url-input:focus {
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.cover-add-btn {
		padding: 8px 16px;
		background: #111;
		color: white;
		border: none;
		border-radius: 6px;
		font-size: 13px;
		font-weight: 500;
		cursor: pointer;
	}

	.cover-cancel-btn {
		padding: 8px 16px;
		background: #f3f4f6;
		color: #374151;
		border: none;
		border-radius: 6px;
		font-size: 13px;
		cursor: pointer;
	}

	/* Title Row */
	.title-row {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		margin-bottom: 8px;
	}

	.type-icon {
		font-size: 40px;
		line-height: 1;
		flex-shrink: 0;
	}

	.title-input {
		flex: 1;
		font-size: 40px;
		font-weight: 700;
		color: #111827;
		border: none;
		outline: none;
		background: transparent;
		line-height: 1.1;
	}

	.title-input::placeholder {
		color: #d1d5db;
	}

	.title-display {
		flex: 1;
		font-size: 40px;
		font-weight: 700;
		color: #111827;
		margin: 0;
		line-height: 1.1;
	}

	/* Meta Row */
	.meta-row {
		margin-bottom: 24px;
		padding-left: 52px; /* Align with title after icon */
	}

	.meta-date {
		font-size: 14px;
		color: #9ca3af;
	}

	/* Slash Hint */
	.slash-hint {
		font-size: 14px;
		color: #9ca3af;
		margin-bottom: 16px;
		padding-left: 52px;
	}

	.slash-hint kbd {
		display: inline-block;
		padding: 2px 8px;
		font-size: 12px;
		font-family: ui-monospace, monospace;
		background: #f3f4f6;
		border: 1px solid #e5e7eb;
		border-radius: 4px;
		color: #6b7280;
	}

	/* Blocks Container */
	.blocks-container {
		min-height: 200px;
		padding-left: 52px; /* Align with title after icon */
	}

	.blocks-container :global(.block-wrapper) {
		color: #1f2937;
	}

	.blocks-container :global([contenteditable]:focus) {
		outline: none;
	}

	.blocks-container :global(.block-editable) {
		font-size: 16px;
		line-height: 1.6;
	}

	.blocks-container :global(h1.block-editable) {
		font-size: 30px;
		line-height: 1.2;
	}

	.blocks-container :global(h2.block-editable) {
		font-size: 24px;
		line-height: 1.3;
	}

	.blocks-container :global(h3.block-editable) {
		font-size: 20px;
		line-height: 1.4;
	}

	/* Related Section */
	.related-section {
		margin-top: 48px;
		padding-top: 24px;
		border-top: 1px solid #f3f4f6;
		padding-left: 52px;
	}

	.related-title {
		font-size: 12px;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: #9ca3af;
		margin: 0 0 12px;
	}

	.related-list {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.related-item {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background: #f9fafb;
		border: 1px solid #f3f4f6;
		border-radius: 6px;
		font-size: 13px;
		color: #374151;
		cursor: pointer;
		transition: all 0.15s;
	}

	.related-item:hover {
		background: #f3f4f6;
		border-color: #e5e7eb;
	}

	.related-icon {
		font-size: 14px;
	}

	.related-name {
		max-width: 150px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* Document Action */
	.document-action {
		padding-left: 52px;
		margin-top: 24px;
	}

	.open-btn {
		padding: 12px 24px;
		background: #111;
		color: white;
		border: none;
		border-radius: 8px;
		font-size: 14px;
		font-weight: 500;
		cursor: pointer;
		transition: background 0.15s;
	}

	.open-btn:hover {
		background: #374151;
	}

	/* Status Bar */
	.status-bar {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 10px 20px;
		border-top: 1px solid #f3f4f6;
		background: #fafafa;
		font-size: 12px;
		color: #9ca3af;
		flex-shrink: 0;
	}

	.status-left {
		display: flex;
		align-items: center;
		gap: 16px;
	}

	.status-unsaved {
		color: #f59e0b;
	}

	.status-right {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.status-saved {
		color: #10b981;
	}

	:global(.menu-trigger) {
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		border-radius: 6px;
		color: #9ca3af;
		cursor: pointer;
		transition: all 0.15s;
	}

	:global(.menu-trigger:hover) {
		background: #f3f4f6;
		color: #374151;
	}

	.close-btn {
		width: 28px;
		height: 28px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: transparent;
		border: none;
		border-radius: 6px;
		color: #9ca3af;
		cursor: pointer;
		transition: all 0.15s;
	}

	.close-btn:hover {
		background: #f3f4f6;
		color: #374151;
	}

	/* Actions Menu Popover */
	:global(.actions-menu) {
		background: white;
		border-radius: 12px;
		padding: 8px;
		box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
		border: 1px solid #e5e7eb;
	}

	.actions-row {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.action-pill {
		display: flex;
		align-items: center;
		gap: 5px;
		padding: 6px 12px;
		background: #f3f4f6;
		border: none;
		border-radius: 20px;
		font-size: 12px;
		font-weight: 500;
		color: #374151;
		cursor: pointer;
		transition: all 0.15s;
		white-space: nowrap;
	}

	.action-pill:hover {
		background: #e5e7eb;
		color: #111827;
	}

	.action-pill.delete {
		background: #fef2f2;
		color: #dc2626;
	}

	.action-pill.delete:hover {
		background: #fee2e2;
		color: #b91c1c;
	}

	/* Empty State */
	.empty-state {
		flex: 1;
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 48px;
		text-align: center;
	}

	.empty-icon {
		font-size: 48px;
		margin-bottom: 16px;
		opacity: 0.5;
	}

	.empty-state h4 {
		font-size: 18px;
		font-weight: 600;
		color: #374151;
		margin: 0 0 8px;
	}

	.empty-state p {
		font-size: 14px;
		color: #9ca3af;
		margin: 0;
		max-width: 240px;
	}
</style>
