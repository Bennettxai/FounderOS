<script lang="ts">
	import { tick } from 'svelte';
	import { ScrollArea } from '$lib/ui';
	import { activeDocument, activeDocumentStore } from '../../stores/documents';
	import { kbSettings, fontSizePx, pageMaxWidth } from '../../stores/settings';
	import { updateDocument, toggleFavorite, deleteDocument, enableSharing, disableSharing, exportDocumentAsMarkdown, exportDocumentAsJSON } from '../../services/documents.service';
	import type { Document, Block } from '../../entities/types';
	import EditorHeader from './EditorHeader.svelte';
	import BlockRenderer from './BlockRenderer.svelte';
	import EditorToolbar from './EditorToolbar.svelte';
	import ShareModal from './ShareModal.svelte';
	import ExportMenu from './ExportMenu.svelte';
	import DocumentProperties from './DocumentProperties.svelte';
	import { debounce, formatRelativeTime } from '$lib/utils';

	interface Props {
		documentId?: string;
		readOnly?: boolean;
		onClose?: () => void;
	}

	let { documentId, readOnly = false, onClose }: Props = $props();

	let doc = $derived($activeDocument);
	let isLoading = $derived($activeDocumentStore.loading);
	let isSaving = $derived($activeDocumentStore.saving);
	let lastSaved = $derived($activeDocumentStore.lastSaved);
	let loadError = $derived($activeDocumentStore.error);

	// Local state for editing
	let localTitle = $state('');
	let localContent = $state<Block[]>([]);
	let hasChanges = $state(false);

	// Cover hover state - passed to EditorHeader to show controls
	let isHoveringCover = $state(false);

	// Cover reposition state
	let isRepositioning = $state(false);
	let coverPositionY = $state(50); // 0-100 percentage, default center
	let dragStartY = $state(0);
	let dragStartPositionY = $state(50);
	let coverElement: HTMLDivElement | null = $state(null);

	// Share & Export modals
	let showShareModal = $state(false);
	let showExportMenu = $state(false);

	// Focus management - track which block should receive focus
	let focusBlockId = $state<string | null>(null);

	// Track which document we're editing to detect document switches
	let currentDocId = $state<string | null>(null);

	// Word count from content blocks
	const wordCount = $derived(
		localContent.reduce((count, block) => {
			const text = block.content
				? (typeof block.content === 'string'
					? block.content
					: block.content.map(rt => rt.plain_text || '').join(''))
				: '';
			return count + (text.trim() ? text.trim().split(/\s+/).length : 0);
		}, 0)
	);

	// Sync local state ONLY when switching to a different document
	// This prevents the save response from overwriting user's in-progress edits
	$effect(() => {
		if (doc && doc.id !== currentDocId) {
			// Document switched - load the new document's content
			currentDocId = doc.id;
			localTitle = doc.title;
			localContent = [...doc.content];
			hasChanges = false;
		}
	});

	// Reset when document is closed/unloaded
	$effect(() => {
		if (!doc && currentDocId) {
			currentDocId = null;
			localTitle = '';
			localContent = [];
			hasChanges = false;
		}
	});

	// Auto-save with debounce — delay sourced from settings store
	let debouncedSave = debounce(async () => {
		if (!doc || !hasChanges) return;

		try {
			await updateDocument(doc.id, {
				title: localTitle,
				content: localContent
			});
			hasChanges = false;
		} catch (error) {
			console.error('Failed to save document:', error);
		}
	}, $kbSettings.autoSaveDelay);

	// Rebuild debounce when delay setting changes
	$effect(() => {
		const delay = $kbSettings.autoSaveDelay;
		debouncedSave = debounce(async () => {
			if (!doc || !hasChanges) return;
			try {
				await updateDocument(doc.id, {
					title: localTitle,
					content: localContent
				});
				hasChanges = false;
			} catch (error) {
				console.error('Failed to save document:', error);
			}
		}, delay);
	});

	function handleTitleChange(newTitle: string) {
		localTitle = newTitle;
		hasChanges = true;
		debouncedSave();
	}

	function handleBlockChange(index: number, updatedBlock: Block) {
		localContent = localContent.map((block, i) =>
			i === index ? { ...updatedBlock, updated_at: new Date().toISOString() } : block
		);
		hasChanges = true;
		debouncedSave();
	}

	function handleBlockDelete(index: number) {
		if (localContent.length <= 1) return; // Keep at least one block
		localContent = localContent.filter((_, i) => i !== index);
		hasChanges = true;
		debouncedSave();
	}

	async function handleBlockAdd(index: number, newBlock: Block) {
		localContent = [
			...localContent.slice(0, index + 1),
			newBlock,
			...localContent.slice(index + 1)
		];
		hasChanges = true;
		debouncedSave();

		// Focus the new block after it renders
		await tick();
		focusBlockId = newBlock.id;
	}

	function handleBlockFocused() {
		// Clear focus target after block has been focused
		focusBlockId = null;
	}

	function handleBlockMove(fromIndex: number, toIndex: number) {
		if (fromIndex === toIndex) return;

		const newContent = [...localContent];
		const [movedBlock] = newContent.splice(fromIndex, 1);
		newContent.splice(toIndex, 0, movedBlock);

		localContent = newContent;
		hasChanges = true;
		debouncedSave();
	}

	// Handle click on the editor content area (outside blocks)
	async function handleContentAreaClick(e: MouseEvent) {
		if (readOnly) return;

		// Check if click was on empty space (not on a block or interactive element)
		const target = e.target as HTMLElement;
		const clickedOnBlock = target.closest('.block, .block-wrapper, .editor-header, button');

		if (!clickedOnBlock) {
			// Click was on empty space - focus or create the last block
			if (localContent.length > 0) {
				// Focus the last block
				const lastBlock = localContent[localContent.length - 1];
				focusBlockId = lastBlock.id;
			} else {
				// No blocks, create one
				const newBlock: Block = {
					id: crypto.randomUUID(),
					type: 'paragraph',
					content: [],
					properties: {},
					children: [],
					created_at: new Date().toISOString(),
					updated_at: new Date().toISOString()
				};
				localContent = [newBlock];
				hasChanges = true;
				debouncedSave();
				await tick();
				focusBlockId = newBlock.id;
			}
		}
	}

	function handleIconChange(icon: string | null) {
		if (!doc) return;
		updateDocument(doc.id, { icon });
	}

	function handleCoverChange(cover: string | null) {
		if (!doc) return;
		updateDocument(doc.id, { cover });
	}

	// Cover reposition handlers
	function handleStartReposition() {
		if (isRepositioning) {
			// Clicking "Done" - exit reposition mode
			isRepositioning = false;
		} else {
			// Start repositioning
			isRepositioning = true;
		}
	}

	function handleCoverMouseDown(e: MouseEvent) {
		if (!isRepositioning) return;
		e.preventDefault();
		dragStartY = e.clientY;
		dragStartPositionY = coverPositionY;

		// Add global listeners
		window.addEventListener('mousemove', handleCoverMouseMove);
		window.addEventListener('mouseup', handleCoverMouseUp);
	}

	function handleCoverMouseMove(e: MouseEvent) {
		if (!isRepositioning || !coverElement) return;

		const coverHeight = coverElement.offsetHeight;
		const deltaY = e.clientY - dragStartY;
		// Convert pixel movement to percentage (inverted: drag down = lower position value)
		const deltaPercent = (deltaY / coverHeight) * 100;
		const newPositionY = Math.max(0, Math.min(100, dragStartPositionY - deltaPercent));

		coverPositionY = newPositionY;
	}

	function handleCoverMouseUp() {
		window.removeEventListener('mousemove', handleCoverMouseMove);
		window.removeEventListener('mouseup', handleCoverMouseUp);
		// TODO: Persist coverPositionY to document when backend supports it
	}

	async function handleToggleFavorite() {
		if (!doc) return;
		try {
			await toggleFavorite(doc.id);
		} catch (error) {
			console.error('Failed to toggle favorite:', error);
		}
	}

	function handleShare() {
		showShareModal = true;
	}

	function handleExport() {
		showExportMenu = true;
	}

	async function handleDelete() {
		if (!doc) return;
		if (!confirm('Are you sure you want to delete this document?')) return;

		try {
			await deleteDocument(doc.id);
			onClose?.();
		} catch (error) {
			console.error('Failed to delete document:', error);
		}
	}
</script>

<div class="document-editor">
	{#if isLoading}
		<div class="document-editor__loading">
			<div class="document-editor__spinner"></div>
			<p>Loading document...</p>
		</div>
	{:else if loadError}
		<div class="document-editor__error">
			<p>Failed to load document</p>
			<p class="document-editor__error-detail">{loadError}</p>
		</div>
	{:else if doc}
		<EditorToolbar
			{isSaving}
			{lastSaved}
			{hasChanges}
			{readOnly}
			isFavorite={doc.is_favorite}
			{onClose}
			onToggleFavorite={handleToggleFavorite}
			onShare={handleShare}
			onExport={handleExport}
			onDelete={handleDelete}
		/>

		<ScrollArea class="document-editor__content">
			<!-- Cover rendered outside container for full-width -->
			{#if doc.cover}
				<!-- svelte-ignore a11y_no_static_element_interactions -->
				<div
					class="document-editor__cover"
					class:document-editor__cover--repositioning={isRepositioning}
					bind:this={coverElement}
					onmouseenter={() => isHoveringCover = true}
					onmouseleave={() => isHoveringCover = false}
					onmousedown={handleCoverMouseDown}
				>
					{#if doc.cover.startsWith('linear-gradient') || (doc.cover.startsWith('#') && doc.cover.length === 7)}
						<div class="document-editor__cover-bg" style="background: {doc.cover}"></div>
					{:else}
						<img
							src={doc.cover}
							alt="Cover"
							class="document-editor__cover-img"
							style="object-position: center {coverPositionY}%"
							draggable="false"
						/>
					{/if}
					{#if isRepositioning}
						<div class="document-editor__reposition-hint">
							<span>Drag to reposition</span>
						</div>
					{/if}
				</div>
			{/if}
			<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
			<div
				class="document-editor__container"
				style="max-width: {$pageMaxWidth}; font-size: {$fontSizePx}"
				onclick={handleContentAreaClick}
			>
				<EditorHeader
					title={localTitle}
					icon={doc.icon}
					cover={doc.cover}
					{readOnly}
					{isHoveringCover}
					{isRepositioning}
					onTitleChange={handleTitleChange}
					onIconChange={handleIconChange}
					onCoverChange={handleCoverChange}
					onStartReposition={handleStartReposition}
				/>

				<!-- Document properties panel (Obsidian-style, collapsible) -->
				<DocumentProperties {doc} content={localContent} />

				<!-- Page metadata bar -->
				<div class="document-editor__meta">
					<span class="document-editor__meta-item">
						{wordCount} {wordCount === 1 ? 'word' : 'words'}
					</span>
					<span class="document-editor__meta-sep">·</span>
					<span class="document-editor__meta-item">
						Edited {formatRelativeTime(doc.updated_at)}
					</span>
					{#if doc.is_public}
						<span class="document-editor__meta-sep">·</span>
						<span class="document-editor__meta-item document-editor__meta-item--public">Published</span>
					{/if}
				</div>

				<div class="document-editor__blocks">
					{#each localContent as block, index (block.id)}
						{@const blockText = block.content.map(rt => rt.plain_text || '').join('')}
						{@const isOnlyEmptyBlock = localContent.length === 1 && blockText === ''}
						<BlockRenderer
							{block}
							{index}
							{readOnly}
							shouldFocus={focusBlockId === block.id}
							isFirstBlock={index === 0}
							{isOnlyEmptyBlock}
							totalBlocks={localContent.length}
							onBlockChange={(updated) => handleBlockChange(index, updated)}
							onBlockDelete={() => handleBlockDelete(index)}
							onBlockAdd={(newBlock) => handleBlockAdd(index, newBlock)}
							onFocused={handleBlockFocused}
							onBlockMove={handleBlockMove}
						/>
					{/each}
				</div>

					<!-- Empty space acts as click target for adding blocks at the end -->
				<div class="document-editor__footer"></div>
			</div>
		</ScrollArea>
	{:else}
		<div class="document-editor__empty">
			<p>Select a document to start editing</p>
		</div>
	{/if}

	{#if doc}
		<ShareModal
			bind:open={showShareModal}
			documentId={doc.id}
			documentTitle={doc.title}
			isPublic={doc.is_public}
			shareId={doc.share_id}
		/>

		<ExportMenu
			bind:open={showExportMenu}
			document={doc}
		/>
	{/if}
</div>

<style>
	.document-editor {
		display: flex;
		flex-direction: column;
		height: 100%;
		background-color: var(--dbg);
	}

	/* Content area - applied via class prop on ScrollArea */
	:global(.document-editor__content) {
		flex: 1;
		overflow: hidden;
	}

	/* Full-width cover outside container */
	.document-editor__cover {
		position: relative;
		width: 100%;
		height: 200px;
		overflow: hidden;
	}

	.document-editor__cover-img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}

	.document-editor__cover-bg {
		width: 100%;
		height: 100%;
	}

	/* Reposition mode */
	.document-editor__cover--repositioning {
		cursor: ns-resize;
		user-select: none;
	}

	.document-editor__cover--repositioning::after {
		content: '';
		position: absolute;
		inset: 0;
		background: rgba(0, 0, 0, 0.1);
		pointer-events: none;
	}

	.document-editor__reposition-hint {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		padding: 0.5rem 1rem;
		background: var(--dbg);
		border: 1px solid var(--dbd);
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--dt);
		pointer-events: none;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
	}

	.document-editor__container {
		max-width: 900px; /* overridden by inline style from settings */
		margin: 0 auto;
		padding: 0 4rem 8rem;
		min-height: 100%;
		cursor: text;
		transition: max-width 0.3s ease;
	}

	.document-editor__meta {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 0 16px;
		font-size: 12px;
		color: var(--dt4);
	}

	.document-editor__meta-item {
		white-space: nowrap;
	}

	.document-editor__meta-item--public {
		color: #22c55e;
	}

	.document-editor__meta-sep {
		opacity: 0.5;
	}

	.document-editor__blocks {
		display: flex;
		flex-direction: column;
		min-height: 200px;
	}

	.document-editor__footer {
		min-height: 300px;
		cursor: text;
	}

	.document-editor__empty {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 100%;
		color: var(--dt3);
	}

	.document-editor__loading {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 1rem;
		color: var(--dt3);
	}

	.document-editor__spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--dbd);
		border-top-color: #1e96eb;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.document-editor__error {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		height: 100%;
		gap: 0.5rem;
		color: #ef4444;
	}

	.document-editor__error-detail {
		font-size: 0.875rem;
		color: var(--dt3);
	}
</style>
