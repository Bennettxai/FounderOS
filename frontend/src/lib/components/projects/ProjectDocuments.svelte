<script lang="ts">
	import type { ContextListItem, Context } from '$lib/api';
	import type { EditorBlock } from '$lib/stores/editor';
	import { editor, wordCount } from '$lib/stores/editor';
	import { contexts } from '$lib/stores/contexts';
	import BlockComponent from '$lib/components/editor/Block.svelte';
	import BlockMenu from '$lib/components/editor/BlockMenu.svelte';
	import { formatDate } from '$lib/utils/project';
	import { onDestroy } from 'svelte';

	type DocumentPanelMode = 'hidden' | 'side' | 'center' | 'full';

	interface Props {
		availableDocuments: ContextListItem[];
		loadingAvailable: boolean;
		embedSuffix: string;
	}

	let { availableDocuments, loadingAvailable, embedSuffix }: Props = $props();

	let documentPanelMode = $state<DocumentPanelMode>('hidden');
	let selectedDocument = $state<Context | null>(null);
	let selectedDocumentId = $state<string | null>(null);
	let loadingDocument = $state(false);
	let documentPanelWidth = $state(550);
	let isResizingPanel = $state(false);
	let documentTitle = $state('');
	let autoSaveTimer: ReturnType<typeof setTimeout>;

	onDestroy(() => {
		if (autoSaveTimer) clearTimeout(autoSaveTimer);
		editor.reset();
	});

	$effect(() => {
		if ($editor.isDirty && selectedDocument && documentPanelMode !== 'hidden') {
			if (autoSaveTimer) clearTimeout(autoSaveTimer);
			autoSaveTimer = setTimeout(async () => {
				await saveDocument();
			}, 1500);
		}
	});

	async function saveDocument() {
		if (!selectedDocument || $editor.isSaving) return;
		editor.setSaving(true);
		try {
			await contexts.updateBlocks(selectedDocument.id, $editor.blocks, $wordCount);
			editor.markSaved();
		} catch (e) {
			console.error('Failed to save:', e);
			editor.setSaving(false);
		}
	}

	async function updateDocumentTitle() {
		const doc = selectedDocument;
		if (!doc || documentTitle === doc.name) return;
		try {
			await contexts.updateContext(doc.id, { name: documentTitle });
		} catch (e) {
			console.error('Failed to update title:', e);
		}
	}

	async function openDocument(docId: string, mode: DocumentPanelMode = 'side') {
		if (selectedDocumentId === docId && documentPanelMode !== 'hidden') {
			documentPanelMode = mode;
			return;
		}

		loadingDocument = true;
		selectedDocumentId = docId;
		documentPanelMode = mode;

		try {
			const doc = await contexts.loadContext(docId);
			selectedDocument = doc;
			documentTitle = doc.name;
			editor.initialize(doc.blocks);
		} catch (e) {
			console.error('Failed to load document:', e);
			closeDocumentPanel();
		} finally {
			loadingDocument = false;
		}
	}

	function closeDocumentPanel() {
		documentPanelMode = 'hidden';
		selectedDocument = null;
		selectedDocumentId = null;
		editor.reset();
	}

	function handlePanelResize(e: MouseEvent) {
		if (!isResizingPanel) return;
		const newWidth = window.innerWidth - e.clientX;
		documentPanelWidth = Math.min(Math.max(newWidth, 400), 900);
	}

	function startPanelResize(e: MouseEvent) {
		e.preventDefault();
		isResizingPanel = true;
		document.addEventListener('mousemove', handlePanelResize);
		document.addEventListener('mouseup', stopPanelResize);
	}

	function stopPanelResize() {
		isResizingPanel = false;
		document.removeEventListener('mousemove', handlePanelResize);
		document.removeEventListener('mouseup', stopPanelResize);
	}

	function addNewBlockAtEnd() {
		const lastBlock = $editor.blocks[$editor.blocks.length - 1];
		if (lastBlock) {
			const newBlockId = editor.addBlockAfter(lastBlock.id);
			setTimeout(() => {
				const blockEl = document.querySelector(`[data-block-id="${newBlockId}"]`) as HTMLElement;
				blockEl?.focus();
			}, 10);
		}
	}
</script>

<!-- Documents Tab Content -->
<div class="prm-doc-card">
	<div class="p-6 prm-doc-card__header flex items-center justify-between">
		<div>
			<h2 class="prm-doc-title">Documents</h2>
			<p class="text-sm prm-doc-muted mt-0.5">Knowledge base documents for this project</p>
		</div>
		<div class="flex gap-2">
			<!-- View mode selector (shown when panel is open) -->
			{#if documentPanelMode !== 'hidden'}
				<div class="flex items-center prm-doc-mode-switcher mr-2">
					<button
						onclick={() => documentPanelMode = 'side'}
						class="btn-pill btn-pill-icon {documentPanelMode === 'side' ? 'btn-pill-primary' : 'btn-pill-ghost'}"
						aria-label="Side panel"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7" />
						</svg>
					</button>
					<button
						onclick={() => documentPanelMode = 'center'}
						class="btn-pill btn-pill-icon {documentPanelMode === 'center' ? 'btn-pill-primary' : 'btn-pill-ghost'}"
						aria-label="Center panel"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
						</svg>
					</button>
					<button
						onclick={() => documentPanelMode = 'full'}
						class="btn-pill btn-pill-icon {documentPanelMode === 'full' ? 'btn-pill-primary' : 'btn-pill-ghost'}"
						aria-label="Full screen"
					>
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
						</svg>
					</button>
				</div>
			{/if}
			<a href="/knowledge{embedSuffix}" class="btn-pill btn-pill-primary btn-pill-sm">
				<svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				New Document
			</a>
		</div>
	</div>

	{#if loadingAvailable}
		<div class="flex items-center justify-center py-16">
			<div class="animate-spin h-6 w-6 prm-doc-spinner rounded-full"></div>
		</div>
	{:else if availableDocuments.length === 0}
		<div class="text-center py-16">
			<div class="w-16 h-16 rounded-full prm-doc-empty-circle flex items-center justify-center mx-auto mb-4">
				<svg class="w-8 h-8 prm-doc-icon" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
				</svg>
			</div>
			<h3 class="prm-doc-title mb-1">No documents yet</h3>
			<p class="prm-doc-muted mb-4">Create documents in the Knowledge Base to link them here</p>
			<a href="/knowledge{embedSuffix}" class="btn-pill btn-pill-primary">Go to Knowledge Base</a>
		</div>
	{:else}
		<div class="p-4 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
			{#each availableDocuments as doc}
				<button
					onclick={() => openDocument(doc.id, 'side')}
					class="btn-pill btn-pill-secondary text-left p-4 border {selectedDocumentId === doc.id ? 'prm-doc-item--active' : 'prm-doc-item-border'}"
				>
					<div class="flex items-start gap-3">
						<span class="text-2xl">{doc.icon || '📄'}</span>
						<div class="flex-1 min-w-0">
							<h4 class="text-sm font-medium prm-doc-text truncate">{doc.name}</h4>
							<p class="text-xs prm-doc-meta mt-0.5">
								{Number(doc.word_count) > 0 ? `${Number(doc.word_count).toLocaleString()} words` : 'Empty'}
							</p>
							<p class="text-xs prm-doc-meta">Updated {formatDate(doc.updated_at)}</p>
						</div>
						{#if selectedDocumentId === doc.id}
							<span class="prm-doc-active-icon">
								<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							</span>
						{/if}
					</div>
				</button>
			{/each}
		</div>
	{/if}
</div>

<!-- Document Editor Panel - Side Mode -->
{#if documentPanelMode === 'side' && selectedDocument}
	<div
		class="fixed inset-y-0 right-0 prm-doc-panel shadow-xl z-40 flex flex-col"
		style="width: {documentPanelWidth}px"
	>
		<!-- Resize Handle -->
		<div
			onmousedown={startPanelResize}
			class="absolute left-0 top-0 bottom-0 w-1 cursor-ew-resize prm-doc-resize-hover transition-colors group"
		>
			<div class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-12 prm-doc-resize rounded-full opacity-0 group-hover:opacity-100 transition-opacity"></div>
		</div>

		<!-- Panel Header -->
		<div class="px-4 py-3 prm-doc-panel__header flex items-center justify-between">
			<div class="flex items-center gap-3 min-w-0 flex-1">
				<span class="text-xl flex-shrink-0">{selectedDocument.icon || '📄'}</span>
				<input
					type="text"
					bind:value={documentTitle}
					onblur={updateDocumentTitle}
					onkeydown={(e) => e.key === 'Enter' && updateDocumentTitle()}
					class="flex-1 min-w-0 font-medium prm-doc-input border-none outline-none prm-doc-input-focus rounded px-1"
				/>
			</div>
			<div class="flex items-center gap-1">
				<div class="text-xs prm-doc-meta mr-2">
					{#if $editor.isDirty}
						<span class="prm-doc-unsaved">Unsaved</span>
					{:else if $editor.isSaving}
						<span>Saving...</span>
					{:else if $editor.lastSavedAt}
						<span class="prm-doc-saved">Saved</span>
					{/if}
				</div>
				<!-- Mode switcher -->
				<div class="flex items-center prm-doc-mode-switcher">
					<button onclick={() => documentPanelMode = 'side'} class="btn-pill btn-pill-primary btn-pill-icon" aria-label="Side panel">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7" />
						</svg>
					</button>
					<button onclick={() => documentPanelMode = 'center'} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Center panel">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
						</svg>
					</button>
					<button onclick={() => documentPanelMode = 'full'} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Full screen">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
						</svg>
					</button>
				</div>
				<a href="/knowledge/{selectedDocument.id}{embedSuffix}" class="btn-pill btn-pill-ghost btn-pill-icon ml-1" aria-label="Open in full page">
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
					</svg>
				</a>
				<button onclick={closeDocumentPanel} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Close">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>

		<!-- Editor Content -->
		{#if loadingDocument}
			<div class="flex-1 flex items-center justify-center">
				<div class="animate-spin h-6 w-6 prm-doc-spinner rounded-full"></div>
			</div>
		{:else}
			<div class="flex-1 overflow-y-auto">
				<div class="max-w-none mx-auto px-6 py-8">
					<div class="blocks-container" role="textbox" tabindex="-1">
						{#each $editor.blocks as block, index (block.id)}
							<BlockComponent {block} {index} readonly={false} parentContextId={selectedDocument.id} />
						{/each}
					</div>
					<button onclick={addNewBlockAtEnd} class="btn-pill btn-pill-ghost w-full min-h-24 mt-4 text-left cursor-text group" aria-label="Add new block">
						<span class="prm-doc-hint opacity-0 group-hover:opacity-100 transition-opacity text-sm">
							Click to add a block, or press / for commands
						</span>
					</button>
				</div>
			</div>
			<div class="px-4 py-2 prm-doc-panel__footer">
				<div class="flex items-center gap-4">
					<span>{$wordCount} words</span>
					<span>{$editor.blocks.length} blocks</span>
				</div>
				<button onclick={saveDocument} class="btn-pill btn-pill-ghost" disabled={!$editor.isDirty}>Save now</button>
			</div>
		{/if}
		{#if $editor.showSlashMenu && $editor.slashMenuPosition}
			<BlockMenu />
		{/if}
	</div>
{/if}

<!-- Document Editor Panel - Center Mode -->
{#if documentPanelMode === 'center' && selectedDocument}
	<div
		class="fixed inset-0 bg-black/30 z-40 flex items-center justify-center p-8"
		onclick={(e) => { if (e.target === e.currentTarget) closeDocumentPanel(); }}
		role="dialog"
		aria-modal="true"
		aria-label="Document editor"
	>
		<div class="prm-doc-dialog shadow-2xl w-full max-w-4xl h-full max-h-[90vh] flex flex-col overflow-hidden">
			<div class="px-6 py-4 prm-doc-card__header flex items-center justify-between">
				<div class="flex items-center gap-3 min-w-0 flex-1">
					<span class="text-2xl flex-shrink-0">{selectedDocument.icon || '📄'}</span>
					<input
						type="text"
						bind:value={documentTitle}
						onblur={updateDocumentTitle}
						onkeydown={(e) => e.key === 'Enter' && updateDocumentTitle()}
						class="flex-1 min-w-0 text-lg font-semibold prm-doc-input border-none outline-none prm-doc-input-focus rounded px-1"
					/>
				</div>
				<div class="flex items-center gap-2">
					<div class="text-sm prm-doc-meta mr-2">
						{#if $editor.isDirty}
							<span class="prm-doc-unsaved">Unsaved</span>
						{:else if $editor.isSaving}
							<span>Saving...</span>
						{:else if $editor.lastSavedAt}
							<span class="prm-doc-saved">Saved</span>
						{/if}
					</div>
					<div class="flex items-center prm-doc-mode-switcher">
						<button onclick={() => documentPanelMode = 'side'} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Side panel">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7" />
							</svg>
						</button>
						<button onclick={() => documentPanelMode = 'center'} class="btn-pill btn-pill-primary btn-pill-icon" aria-label="Center panel">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
							</svg>
						</button>
						<button onclick={() => documentPanelMode = 'full'} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Full screen">
							<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
							</svg>
						</button>
					</div>
					<a href="/knowledge/{selectedDocument.id}{embedSuffix}" class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Open in full page">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
						</svg>
					</a>
					<button onclick={closeDocumentPanel} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Close">
						<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			</div>
			{#if loadingDocument}
				<div class="flex-1 flex items-center justify-center">
					<div class="animate-spin h-6 w-6 prm-doc-spinner rounded-full"></div>
				</div>
			{:else}
				<div class="flex-1 overflow-y-auto">
					<div class="max-w-3xl mx-auto px-8 py-12">
						<div class="blocks-container" role="textbox" tabindex="-1">
							{#each $editor.blocks as block, index (block.id)}
								<BlockComponent {block} {index} readonly={false} parentContextId={selectedDocument.id} />
							{/each}
						</div>
						<button onclick={addNewBlockAtEnd} class="btn-pill btn-pill-ghost w-full min-h-24 mt-4 text-left cursor-text group" aria-label="Add new block">
							<span class="prm-doc-hint opacity-0 group-hover:opacity-100 transition-opacity text-sm">
								Click to add a block, or press / for commands
							</span>
						</button>
					</div>
				</div>
				<div class="px-6 py-3 prm-doc-panel__footer">
					<div class="flex items-center gap-4">
						<span>{$wordCount} words</span>
						<span>{$editor.blocks.length} blocks</span>
					</div>
					<button onclick={saveDocument} class="btn-pill btn-pill-ghost" disabled={!$editor.isDirty}>Save now</button>
				</div>
			{/if}
			{#if $editor.showSlashMenu && $editor.slashMenuPosition}
				<BlockMenu />
			{/if}
		</div>
	</div>
{/if}

<!-- Document Editor Panel - Full Screen Mode -->
{#if documentPanelMode === 'full' && selectedDocument}
	<div class="fixed inset-0 prm-doc-full flex flex-col">
		<div class="px-6 py-4 prm-doc-full__header flex items-center justify-between">
			<div class="flex items-center gap-3">
				<button onclick={closeDocumentPanel} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Back to project">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
					</svg>
				</button>
				<span class="prm-doc-sep">|</span>
				<span class="text-2xl">{selectedDocument.icon || '📄'}</span>
				<input
					type="text"
					bind:value={documentTitle}
					onblur={updateDocumentTitle}
					onkeydown={(e) => e.key === 'Enter' && updateDocumentTitle()}
					class="text-xl font-semibold prm-doc-input border-none outline-none prm-doc-input-focus rounded px-1"
				/>
			</div>
			<div class="flex items-center gap-2">
				<div class="text-sm prm-doc-meta mr-4">
					{#if $editor.isDirty}
						<span class="prm-doc-unsaved">Unsaved changes</span>
					{:else if $editor.isSaving}
						<span>Saving...</span>
					{:else if $editor.lastSavedAt}
						<span class="prm-doc-saved">All changes saved</span>
					{/if}
				</div>
				<div class="flex items-center prm-doc-mode-switcher">
					<button onclick={() => documentPanelMode = 'side'} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Side panel">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 17V7m0 10a2 2 0 01-2 2H5a2 2 0 01-2-2V7a2 2 0 012-2h2a2 2 0 012 2m0 10a2 2 0 002 2h2a2 2 0 002-2M9 7a2 2 0 012-2h2a2 2 0 012 2m0 10V7" />
						</svg>
					</button>
					<button onclick={() => documentPanelMode = 'center'} class="btn-pill btn-pill-ghost btn-pill-icon" aria-label="Center panel">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z" />
						</svg>
					</button>
					<button onclick={() => documentPanelMode = 'full'} class="btn-pill btn-pill-primary btn-pill-icon" aria-label="Full screen">
						<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4" />
						</svg>
					</button>
				</div>
				<a href="/knowledge/{selectedDocument.id}{embedSuffix}" class="btn-pill btn-pill-secondary btn-pill-sm ml-2" aria-label="Open in Knowledge Base">
					Open in Knowledge Base
				</a>
				<button onclick={closeDocumentPanel} class="btn-pill btn-pill-ghost btn-pill-icon ml-2" aria-label="Exit full screen">
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		</div>
		{#if loadingDocument}
			<div class="flex-1 flex items-center justify-center">
				<div class="animate-spin h-8 w-8 prm-doc-spinner rounded-full"></div>
			</div>
		{:else}
			<div class="flex-1 overflow-y-auto prm-doc-full__content-bg">
				<div class="max-w-3xl mx-auto px-8 py-12 prm-doc-full__editor">
					<div class="blocks-container" role="textbox" tabindex="-1">
						{#each $editor.blocks as block, index (block.id)}
							<BlockComponent {block} {index} readonly={false} parentContextId={selectedDocument.id} />
						{/each}
					</div>
					<button onclick={addNewBlockAtEnd} class="btn-pill btn-pill-ghost w-full min-h-32 mt-4 text-left cursor-text group" aria-label="Add new block">
						<span class="prm-doc-hint opacity-0 group-hover:opacity-100 transition-opacity text-sm">
							Click to add a block, or press / for commands
						</span>
					</button>
				</div>
			</div>
			<div class="px-6 py-3 prm-doc-full__footer">
				<div class="flex items-center gap-6">
					<span>{$wordCount} words</span>
					<span>{$editor.blocks.length} blocks</span>
				</div>
				<button onclick={saveDocument} class="btn-pill btn-pill-ghost" disabled={!$editor.isDirty}>
					Save now
				</button>
			</div>
		{/if}
		{#if $editor.showSlashMenu && $editor.slashMenuPosition}
			<BlockMenu />
		{/if}
	</div>
{/if}

<style>
	.prm-doc-card {
		background: var(--dbg, #fff);
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.75rem;
	}
	.prm-doc-card__header {
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
	}
	.prm-doc-title {
		font-size: 1.125rem;
		font-weight: 500;
		color: var(--dt, #111);
	}
	.prm-doc-text {
		color: var(--dt, #111);
	}
	.prm-doc-muted {
		color: var(--dt2, #555);
	}
	.prm-doc-meta {
		color: var(--dt3, #888);
	}
	.prm-doc-icon {
		color: var(--dt3, #888);
	}
	.prm-doc-hint {
		color: var(--dt4, #bbb);
	}
	.prm-doc-sep {
		color: var(--dt4, #bbb);
	}
	.prm-doc-spinner {
		border: 2px solid var(--dt, #111);
		border-top-color: transparent;
	}
	.prm-doc-empty-circle {
		background: var(--dbg2, #f5f5f5);
	}
	.prm-doc-item-border {
		border-color: var(--dbd, #e0e0e0);
	}
	.prm-doc-item--active {
		box-shadow: 0 0 0 2px var(--dt, #111);
		border-color: var(--dt, #111);
	}
	.prm-doc-active-icon { color: var(--dt, #111); }
	.prm-doc-unsaved { color: var(--dt2, #555); font-style: italic; }
	.prm-doc-saved { color: var(--dt2, #555); }
	.prm-doc-input-focus:focus { outline: none; box-shadow: 0 0 0 2px var(--dt, #111); }
	.prm-doc-resize-hover:hover { background: var(--dt3, #888); }
	.prm-doc-mode-switcher {
		border: 1px solid var(--dbd, #e0e0e0);
		border-radius: 0.5rem;
		overflow: hidden;
	}
	.prm-doc-panel {
		background: var(--dbg, #fff);
		border-left: 1px solid var(--dbd, #e0e0e0);
	}
	.prm-doc-panel__header {
		border-bottom: 1px solid var(--dbd2, #f0f0f0);
		background: var(--dbg2, rgba(245,245,245,0.5));
	}
	.prm-doc-panel__footer {
		border-top: 1px solid var(--dbd2, #f0f0f0);
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 0.75rem;
		color: var(--dt3, #888);
		background: var(--dbg2, rgba(245,245,245,0.5));
	}
	.prm-doc-input {
		color: var(--dt, #111);
		background: transparent;
	}
	.prm-doc-resize {
		background: var(--dt4, #bbb);
	}
	.prm-doc-dialog {
		background: var(--dbg, #fff);
		border-radius: 1rem;
	}
	.prm-doc-full {
		background: var(--dbg, #fff);
		z-index: 50;
	}
	.prm-doc-full__header {
		border-bottom: 1px solid var(--dbd, #e0e0e0);
		background: var(--dbg, #fff);
	}
	.prm-doc-full__content-bg {
		background: var(--dbg2, rgba(245,245,245,0.5));
	}
	.prm-doc-full__editor {
		background: var(--dbg, #fff);
		min-height: 100%;
		box-shadow: 0 1px 2px rgba(0,0,0,0.05);
	}
	.prm-doc-full__footer {
		border-top: 1px solid var(--dbd, #e0e0e0);
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 0.875rem;
		color: var(--dt2, #555);
		background: var(--dbg, #fff);
	}
</style>
