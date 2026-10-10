<script lang="ts">
	import { currentWorkspaceId } from '$lib/stores/workspaces';
	import type { MemoryType } from '$lib/api/memory';
	import type {
		WorkspaceMemoryListItem,
		MemoryVisibility,
		WorkspaceMemoryFilters
	} from '$lib/api/workspaces/memory';
	import {
		listAccessibleMemories,
		deleteWorkspaceMemory,
		pinWorkspaceMemory
	} from '$lib/api/workspaces/memory';
	import MemoryCard from '../chat/shared/MemoryCard.svelte';
	import MemoryDetailModal from '../chat/modals/MemoryDetailModal.svelte';
	import MemoryFilters from '../chat/shared/MemoryFilters.svelte';
	import MemoryStats from '../chat/shared/MemoryStats.svelte';
	import MemoryVisibilitySelector from './MemoryVisibilitySelector.svelte';
	import MemorySharingModal from './MemorySharingModal.svelte';

	interface Props {
		onMemoryClick?: (memory: WorkspaceMemoryListItem) => void;
	}

	let { onMemoryClick }: Props = $props();

	let memories = $state<WorkspaceMemoryListItem[]>([]);
	let loading = $state(false);
	let searchQuery = $state('');
	let selectedType = $state<MemoryType | 'all'>('all');
	let selectedVisibility = $state<MemoryVisibility | 'all'>('all');
	let showPinnedOnly = $state(false);
	let importanceMin = $state(0);
	let dateRange = $state<{ start: string; end: string } | null>(null);
	let selectedMemory = $state<WorkspaceMemoryListItem | null>(null);
	let sharingMemory = $state<WorkspaceMemoryListItem | null>(null);

	async function loadMemories() {
		if (!$currentWorkspaceId) return;

		loading = true;
		try {
			const filters: WorkspaceMemoryFilters = {
				memory_type: selectedType !== 'all' ? selectedType : undefined,
				visibility: selectedVisibility !== 'all' ? selectedVisibility : undefined,
				is_pinned: showPinnedOnly || undefined,
				limit: 50
			};

			const results = await listAccessibleMemories($currentWorkspaceId, filters);
			memories = results;
		} catch (error) {
			console.error('Failed to load workspace memories:', error);
			memories = [];
		} finally {
			loading = false;
		}
	}

	function handleSearch(e: Event) {
		e.preventDefault();
		loadMemories();
	}

	function handleTypeChange(type: MemoryType | 'all') {
		selectedType = type;
		loadMemories();
	}

	function handleVisibilityChange(visibility: MemoryVisibility | 'all') {
		selectedVisibility = visibility;
		loadMemories();
	}

	function togglePinnedOnly() {
		showPinnedOnly = !showPinnedOnly;
		loadMemories();
	}

	async function handleMemoryPin(memory: WorkspaceMemoryListItem) {
		if (!$currentWorkspaceId) return;

		try {
			await pinWorkspaceMemory($currentWorkspaceId, memory.id, !memory.is_pinned);
			memories = memories.map((m) =>
				m.id === memory.id ? { ...m, is_pinned: !m.is_pinned } : m
			);
			// Re-sort
			sortMemories();
		} catch (error) {
			console.error('Failed to pin memory:', error);
		}
	}

	async function handleMemoryDelete(memory: WorkspaceMemoryListItem) {
		if (!$currentWorkspaceId) return;
		if (!confirm('Are you sure you want to delete this memory?')) return;

		try {
			await deleteWorkspaceMemory($currentWorkspaceId, memory.id);
			memories = memories.filter((m) => m.id !== memory.id);
		} catch (error) {
			console.error('Failed to delete memory:', error);
		}
	}

	function handleShare(memory: WorkspaceMemoryListItem) {
		sharingMemory = memory;
	}

	function handleShareComplete() {
		sharingMemory = null;
		loadMemories(); // Refresh to get updated shared_with_user_ids
	}

	function sortMemories() {
		memories = [...memories].sort((a, b) => {
			if (a.is_pinned && !b.is_pinned) return -1;
			if (!a.is_pinned && b.is_pinned) return 1;
			return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
		});
	}

	// Sort and filter memories
	let sortedMemories = $derived(() => {
		let filtered = [...memories];

		// Apply search filter
		if (searchQuery.trim()) {
			const query = searchQuery.toLowerCase();
			filtered = filtered.filter(
				(m) =>
					m.title.toLowerCase().includes(query) ||
					m.summary.toLowerCase().includes(query) ||
					m.tags.some((tag) => tag.toLowerCase().includes(query))
			);
		}

		// Apply importance filter
		if (importanceMin > 0) {
			filtered = filtered.filter((m) => m.importance_score >= importanceMin / 100);
		}

		// Apply date range filter
		if (dateRange) {
			const startDate = new Date(dateRange.start);
			const endDate = new Date(dateRange.end);
			endDate.setHours(23, 59, 59, 999);

			filtered = filtered.filter((m) => {
				const memoryDate = new Date(m.created_at);
				return memoryDate >= startDate && memoryDate <= endDate;
			});
		}

		// Sort: pinned first, then by date
		return filtered.sort((a, b) => {
			if (a.is_pinned && !b.is_pinned) return -1;
			if (!a.is_pinned && b.is_pinned) return 1;
			return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
		});
	});

	// Load memories when workspace changes
	$effect(() => {
		if ($currentWorkspaceId) {
			loadMemories();
		}
	});
</script>

<div class="memory-panel">
	<div class="panel-header">
		<h3 class="panel-title">Workspace Memories</h3>
		<button
			class="pin-filter-btn"
			class:active={showPinnedOnly}
			onclick={togglePinnedOnly}
			aria-label={showPinnedOnly ? 'Show all memories' : 'Show pinned only'}
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				fill={showPinnedOnly ? 'currentColor' : 'none'}
				viewBox="0 0 24 24"
				stroke-width="1.5"
				stroke="currentColor"
				width="16"
				height="16"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9"
				/>
			</svg>
		</button>
	</div>

	<div class="search-section">
		<form class="search-form" onsubmit={handleSearch}>
			<div class="search-input-wrapper">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					class="search-icon"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z"
					/>
				</svg>
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Search workspace memories..."
					class="search-input"
				/>
				{#if searchQuery}
					<button
						type="button"
						class="btn-pill btn-pill-ghost clear-btn"
						onclick={() => {
							searchQuery = '';
						}}
						aria-label="Clear search"
					>
						<svg
							xmlns="http://www.w3.org/2000/svg"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="1.5"
							stroke="currentColor"
							width="14"
							height="14"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
						</svg>
					</button>
				{/if}
			</div>
		</form>
	</div>

	<div class="filters-section">
		<MemoryFilters
			{selectedType}
			{showPinnedOnly}
			{importanceMin}
			{dateRange}
			onTypeChange={handleTypeChange}
			onPinnedToggle={togglePinnedOnly}
			onImportanceChange={(min) => (importanceMin = min)}
			onDateRangeChange={(range) => (dateRange = range)}
		/>

		<MemoryVisibilitySelector
			selected={selectedVisibility}
			onChange={handleVisibilityChange}
		/>
	</div>

	{#if !loading && memories.length > 0}
		<MemoryStats memories={sortedMemories()} />
	{/if}

	<div class="panel-content">
		{#if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading workspace memories...</p>
			</div>
		{:else if sortedMemories().length === 0}
			<div class="empty-state">
				<svg
					xmlns="http://www.w3.org/2000/svg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="1.5"
					stroke="currentColor"
					class="empty-icon"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125"
					/>
				</svg>
				{#if searchQuery}
					<p class="empty-text">No memories found for "{searchQuery}"</p>
					<button
						class="btn-pill btn-pill-ghost clear-search-btn"
						onclick={() => {
							searchQuery = '';
						}}
					>
						Clear search
					</button>
				{:else}
					<p class="empty-text">No workspace memories yet</p>
					<p class="empty-hint">Memories will be created and shared within your workspace</p>
				{/if}
			</div>
		{:else}
			<div class="memory-list">
				{#each sortedMemories() as memory (memory.id)}
					<div class="memory-card-wrapper">
						<MemoryCard
							{memory}
							onClick={() => selectedMemory = memory}
							onPin={handleMemoryPin}
							onDelete={handleMemoryDelete}
						/>
						<div class="memory-card-actions">
							<span class="visibility-badge" data-visibility={memory.visibility}>
								{#if memory.visibility === 'workspace'}
									<svg
										xmlns="http://www.w3.org/2000/svg"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
										width="12"
										height="12"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M18 18.72a9.094 9.094 0 0 0 3.741-.479 3 3 0 0 0-4.682-2.72m.94 3.198.001.031c0 .225-.012.447-.037.666A11.944 11.944 0 0 1 12 21c-2.17 0-4.207-.576-5.963-1.584A6.062 6.062 0 0 1 6 18.719m12 0a5.971 5.971 0 0 0-.941-3.197m0 0A5.995 5.995 0 0 0 12 12.75a5.995 5.995 0 0 0-5.058 2.772m0 0a3 3 0 0 0-4.681 2.72 8.986 8.986 0 0 0 3.74.477m.94-3.197a5.971 5.971 0 0 0-.94 3.197M15 6.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm6 3a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Zm-13.5 0a2.25 2.25 0 1 1-4.5 0 2.25 2.25 0 0 1 4.5 0Z"
										/>
									</svg>
									Workspace
								{:else if memory.visibility === 'private'}
									<svg
										xmlns="http://www.w3.org/2000/svg"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
										width="12"
										height="12"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"
										/>
									</svg>
									Private
								{:else}
									<svg
										xmlns="http://www.w3.org/2000/svg"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
										width="12"
										height="12"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M7.217 10.907a2.25 2.25 0 1 0 0 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186 9.566-5.314m-9.566 7.5 9.566 5.314m0 0a2.25 2.25 0 1 0 3.935 2.186 2.25 2.25 0 0 0-3.935-2.186Zm0-12.814a2.25 2.25 0 1 0 3.933-2.185 2.25 2.25 0 0 0-3.933 2.185Z"
										/>
									</svg>
									Shared
								{/if}
							</span>
							{#if memory.visibility === 'private' || memory.visibility === 'shared'}
								<button class="btn-pill btn-pill-ghost share-btn" onclick={() => handleShare(memory)}>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										fill="none"
										viewBox="0 0 24 24"
										stroke-width="1.5"
										stroke="currentColor"
										width="14"
										height="14"
									>
										<path
											stroke-linecap="round"
											stroke-linejoin="round"
											d="M7.217 10.907a2.25 2.25 0 1 0 0 2.186m0-2.186c.18.324.283.696.283 1.093s-.103.77-.283 1.093m0-2.186 9.566-5.314m-9.566 7.5 9.566 5.314m0 0a2.25 2.25 0 1 0 3.935 2.186 2.25 2.25 0 0 0-3.935-2.186Zm0-12.814a2.25 2.25 0 1 0 3.933-2.185 2.25 2.25 0 0 0-3.933 2.185Z"
										/>
									</svg>
									Share
								</button>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="panel-footer">
		<span class="memory-count"
			>{sortedMemories().length} memor{sortedMemories().length === 1 ? 'y' : 'ies'}</span
		>
		<button class="btn-pill btn-pill-ghost refresh-btn" onclick={loadMemories} disabled={loading}>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				fill="none"
				viewBox="0 0 24 24"
				stroke-width="1.5"
				stroke="currentColor"
				width="14"
				height="14"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99"
				/>
			</svg>
			Refresh
		</button>
	</div>
</div>

<MemoryDetailModal
	memory={selectedMemory}
	onClose={() => (selectedMemory = null)}
	onPin={handleMemoryPin}
	onDelete={handleMemoryDelete}
/>

{#if sharingMemory}
	<MemorySharingModal
		memory={sharingMemory}
		onClose={() => (sharingMemory = null)}
		onComplete={handleShareComplete}
	/>
{/if}

<style>
	.memory-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--color-border);
	}

	:global(.dark) .panel-header {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	.panel-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0;
	}

	:global(.dark) .panel-title {
		color: #f5f5f7;
	}

	.pin-filter-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
		transition: all 0.15s ease;
	}

	.pin-filter-btn:hover {
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	.pin-filter-btn.active {
		color: #3b82f6;
		background: rgba(59, 130, 246, 0.1);
	}

	:global(.dark) .pin-filter-btn {
		color: #6e6e73;
	}

	:global(.dark) .pin-filter-btn:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	.search-section {
		padding: 12px 16px;
		border-bottom: 1px solid var(--color-border);
	}

	:global(.dark) .search-section {
		border-bottom-color: rgba(255, 255, 255, 0.06);
	}

	.search-form {
		flex: 1;
	}

	.search-input-wrapper {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: var(--color-bg-secondary);
		border-radius: 8px;
	}

	:global(.dark) .search-input-wrapper {
		background: #2c2c2e;
	}

	.search-icon {
		width: 16px;
		height: 16px;
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	:global(.dark) .search-icon {
		color: #6e6e73;
	}

	.search-input {
		flex: 1;
		border: none;
		background: transparent;
		font-size: 13px;
		color: var(--color-text);
		outline: none;
	}

	.search-input::placeholder {
		color: var(--color-text-muted);
	}

	:global(.dark) .search-input {
		color: #f5f5f7;
	}

	:global(.dark) .search-input::placeholder {
		color: #6e6e73;
	}

	.clear-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 2px;
		border: none;
		background: transparent;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 4px;
	}

	.clear-btn:hover {
		color: var(--color-text);
	}

	.filters-section {
		padding: 12px 16px;
		border-bottom: 1px solid var(--color-border);
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	:global(.dark) .filters-section {
		border-bottom-color: rgba(255, 255, 255, 0.06);
	}

	.panel-content {
		flex: 1;
		overflow-y: auto;
		padding: 12px;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 40px 16px;
		gap: 12px;
		color: var(--color-text-muted);
		font-size: 13px;
	}

	.spinner {
		width: 24px;
		height: 24px;
		border: 2px solid var(--color-border);
		border-top-color: var(--color-text);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 40px 16px;
		text-align: center;
	}

	.empty-icon {
		width: 48px;
		height: 48px;
		color: var(--color-text-muted);
		margin-bottom: 12px;
	}

	:global(.dark) .empty-icon {
		color: #6e6e73;
	}

	.empty-text {
		font-size: 13px;
		color: var(--color-text-muted);
		margin: 0;
	}

	:global(.dark) .empty-text {
		color: #6e6e73;
	}

	.empty-hint {
		font-size: 12px;
		color: var(--color-text-muted);
		margin-top: 4px;
		opacity: 0.7;
	}

	.clear-search-btn {
		margin-top: 12px;
		padding: 6px 12px;
		font-size: 12px;
		font-weight: 500;
		color: #3b82f6;
		background: transparent;
		border: 1px solid #3b82f6;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.clear-search-btn:hover {
		background: rgba(59, 130, 246, 0.1);
	}

	.memory-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.memory-card-wrapper {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.memory-card-actions {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 0 12px;
	}

	.visibility-badge {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 2px 8px;
		font-size: 10px;
		font-weight: 600;
		text-transform: uppercase;
		border-radius: 4px;
		letter-spacing: 0.5px;
	}

	.visibility-badge[data-visibility='workspace'] {
		background: rgba(34, 197, 94, 0.1);
		color: #22c55e;
	}

	.visibility-badge[data-visibility='private'] {
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}

	.visibility-badge[data-visibility='shared'] {
		background: rgba(59, 130, 246, 0.1);
		color: #3b82f6;
	}

	:global(.dark) .visibility-badge[data-visibility='workspace'] {
		background: rgba(34, 197, 94, 0.15);
	}

	:global(.dark) .visibility-badge[data-visibility='private'] {
		background: rgba(239, 68, 68, 0.15);
	}

	:global(.dark) .visibility-badge[data-visibility='shared'] {
		background: rgba(59, 130, 246, 0.15);
	}

	.share-btn {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 4px 8px;
		font-size: 11px;
		font-weight: 500;
		color: var(--color-text-muted);
		background: transparent;
		border: 1px solid var(--color-border);
		border-radius: 4px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.share-btn:hover {
		color: var(--color-text);
		background: var(--color-bg-secondary);
	}

	:global(.dark) .share-btn {
		color: #a1a1a6;
		border-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .share-btn:hover {
		color: #f5f5f7;
		background: #3a3a3c;
	}

	.panel-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-top: 1px solid var(--color-border);
	}

	:global(.dark) .panel-footer {
		border-top-color: rgba(255, 255, 255, 0.1);
	}

	.memory-count {
		font-size: 12px;
		color: var(--color-text-muted);
	}

	:global(.dark) .memory-count {
		color: #6e6e73;
	}

	.refresh-btn {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text);
		background: var(--color-bg-secondary);
		border: none;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.refresh-btn:hover:not(:disabled) {
		background: var(--color-bg-tertiary);
	}

	.refresh-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	:global(.dark) .refresh-btn {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	:global(.dark) .refresh-btn:hover:not(:disabled) {
		background: #4a4a4c;
	}
</style>
