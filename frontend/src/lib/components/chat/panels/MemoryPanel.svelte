<script lang="ts">
	import { api, type MemoryListItem, type MemoryType } from '$lib/api';
	import MemoryCard from '../shared/MemoryCard.svelte';
	import MemoryStats from '../shared/MemoryStats.svelte';
	import MemoryFilters from '../shared/MemoryFilters.svelte';
	import MemoryDetailModal from '../modals/MemoryDetailModal.svelte';

	interface Props {
		conversationId?: string;
		projectId?: string;
		nodeId?: string;
		onMemoryClick?: (memory: MemoryListItem) => void;
		selectedMemoryIds?: string[];
	}

	let { conversationId, projectId, nodeId, onMemoryClick, selectedMemoryIds = [] }: Props = $props();

	let memories = $state<MemoryListItem[]>([]);
	let loading = $state(false);
	let searchQuery = $state('');
	let selectedType = $state<MemoryType | 'all'>('all');
	let showPinnedOnly = $state(false);
	let importanceMin = $state(0);
	let dateRange = $state<{ start: string; end: string } | null>(null);
	let selectedMemory = $state<MemoryListItem | null>(null);

	const memoryTypes: { value: MemoryType | 'all'; label: string }[] = [
		{ value: 'all', label: 'All Types' },
		{ value: 'fact', label: 'Facts' },
		{ value: 'preference', label: 'Preferences' },
		{ value: 'decision', label: 'Decisions' },
		{ value: 'event', label: 'Events' },
		{ value: 'learning', label: 'Learnings' },
		{ value: 'context', label: 'Context' },
		{ value: 'relationship', label: 'Relationships' }
	];

	async function loadMemories() {
		loading = true;
		try {
			if (searchQuery.trim()) {
				const results = await api.searchMemories({
					query: searchQuery,
					memory_type: selectedType !== 'all' ? selectedType : undefined,
					project_id: projectId,
					node_id: nodeId,
					limit: 50
				});
				memories = results;
			} else {
				const results = await api.getMemories({
					memory_type: selectedType !== 'all' ? selectedType : undefined,
					project_id: projectId,
					node_id: nodeId,
					is_pinned: showPinnedOnly || undefined,
					limit: 50
				});
				memories = results;
			}
		} catch (error) {
			console.error('Failed to load memories:', error);
			memories = [];
		} finally {
			loading = false;
		}
	}

	function handleSearch(e: Event) {
		e.preventDefault();
		loadMemories();
	}

	function handleTypeChange(e: Event) {
		const target = e.target as HTMLSelectElement;
		selectedType = target.value as MemoryType | 'all';
		loadMemories();
	}

	function togglePinnedOnly() {
		showPinnedOnly = !showPinnedOnly;
		loadMemories();
	}

	function handleMemoryPin(memory: MemoryListItem) {
		memories = memories.map(m =>
			m.id === memory.id ? { ...m, is_pinned: !m.is_pinned } : m
		);
		// Re-sort if showing pinned only
		if (!showPinnedOnly) {
			memories = [...memories].sort((a, b) => {
				if (a.is_pinned && !b.is_pinned) return -1;
				if (!a.is_pinned && b.is_pinned) return 1;
				return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
			});
		}
	}

	async function handleMemoryDelete(memory: MemoryListItem) {
		if (!confirm('Are you sure you want to delete this memory?')) return;

		try {
			await api.deleteMemory(memory.id);
			memories = memories.filter(m => m.id !== memory.id);
		} catch (error) {
			console.error('Failed to delete memory:', error);
		}
	}

	// Sort and filter memories: pinned first, then by date
	let sortedMemories = $derived.by(() => {
		let filtered = [...memories];

		// Apply importance filter
		if (importanceMin > 0) {
			filtered = filtered.filter(m => m.importance_score >= importanceMin / 100);
		}

		// Apply date range filter
		if (dateRange) {
			const startDate = new Date(dateRange.start);
			const endDate = new Date(dateRange.end);
			endDate.setHours(23, 59, 59, 999); // Include full end date

			filtered = filtered.filter(m => {
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

	// Load memories on mount
	$effect(() => {
		loadMemories();
	});
</script>

<div class="memory-panel">
	<div class="panel-header">
		<h3 class="panel-title">Memories</h3>
		<button
			class="pin-filter-btn"
			class:active={showPinnedOnly}
			onclick={togglePinnedOnly}
			aria-label={showPinnedOnly ? 'Show all memories' : 'Show pinned only'}
		>
			<svg xmlns="http://www.w3.org/2000/svg" fill={showPinnedOnly ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 3.75V16.5L12 14.25 7.5 16.5V3.75m9 0H18A2.25 2.25 0 0 1 20.25 6v12A2.25 2.25 0 0 1 18 20.25H6A2.25 2.25 0 0 1 3.75 18V6A2.25 2.25 0 0 1 6 3.75h1.5m9 0h-9" />
			</svg>
		</button>
	</div>

	<div class="search-section">
		<form class="search-form" onsubmit={handleSearch}>
			<div class="search-input-wrapper">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="search-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
				</svg>
				<input
					type="text"
					bind:value={searchQuery}
					placeholder="Search memories..."
					class="search-input"
				/>
				{#if searchQuery}
					<button
						type="button"
						class="btn-pill btn-pill-ghost clear-btn"
						onclick={() => { searchQuery = ''; loadMemories(); }}
						aria-label="Clear search"
					>
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
							<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
						</svg>
					</button>
				{/if}
			</div>
		</form>
	</div>

	<MemoryFilters
		{selectedType}
		{showPinnedOnly}
		{importanceMin}
		{dateRange}
		onTypeChange={(type) => { selectedType = type; loadMemories(); }}
		onPinnedToggle={togglePinnedOnly}
		onImportanceChange={(min) => importanceMin = min}
		onDateRangeChange={(range) => dateRange = range}
	/>

	{#if !loading && memories.length > 0}
		<MemoryStats memories={sortedMemories} />
	{/if}

	<div class="panel-content">
		{#if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading memories...</p>
			</div>
		{:else if sortedMemories.length === 0}
			<div class="empty-state">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="empty-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
				</svg>
				{#if searchQuery}
					<p class="empty-text">No memories found for "{searchQuery}"</p>
					<button class="btn-pill btn-pill-ghost clear-search-btn" onclick={() => { searchQuery = ''; loadMemories(); }}>
						Clear search
					</button>
				{:else}
					<p class="empty-text">No memories yet</p>
					<p class="empty-hint">Memories will be extracted from your conversations</p>
				{/if}
			</div>
		{:else}
			<div class="memory-list">
				{#each sortedMemories as memory (memory.id)}
					<MemoryCard
						{memory}
						onClick={() => onMemoryClick?.(memory)}
						onPin={handleMemoryPin}
						onDelete={handleMemoryDelete}
						isSelected={selectedMemoryIds.includes(memory.id)}
					/>
				{/each}
			</div>
		{/if}
	</div>

	<div class="panel-footer">
		<span class="memory-count">{sortedMemories.length} memor{sortedMemories.length === 1 ? 'y' : 'ies'}</span>
		<button class="btn-pill btn-pill-ghost refresh-btn" onclick={loadMemories} disabled={loading}>
			<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.023 9.348h4.992v-.001M2.985 19.644v-4.992m0 0h4.992m-4.993 0 3.181 3.183a8.25 8.25 0 0 0 13.803-3.7M4.031 9.865a8.25 8.25 0 0 1 13.803-3.7l3.181 3.182m0-4.991v4.99" />
			</svg>
			Refresh
		</button>
	</div>
</div>

<MemoryDetailModal
	memory={selectedMemory}
	onClose={() => selectedMemory = null}
	onPin={handleMemoryPin}
	onDelete={handleMemoryDelete}
/>

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
		display: flex;
		gap: 8px;
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

	.type-filter {
		padding: 8px 12px;
		font-size: 13px;
		border: none;
		background: var(--color-bg-secondary);
		color: var(--color-text);
		border-radius: 8px;
		cursor: pointer;
		outline: none;
	}

	:global(.dark) .type-filter {
		background: #2c2c2e;
		color: #f5f5f7;
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
