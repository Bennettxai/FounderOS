<script lang="ts">
	import { api } from '$lib/api';
	import type { TreeSearchResult, EntityType } from '$lib/api/context-tree/types';
	import { ImageSearchModal, ImageGalleryView } from '$lib/components/search';
	import type { MultimodalSearchResult } from '$lib/api/multimodal-search';

	interface Props {
		projectId?: string;
		nodeId?: string;
		onItemSelect?: (item: TreeSearchResult) => void;
	}

	let { projectId, nodeId, onItemSelect }: Props = $props();

	let searchQuery = $state('');
	let searchType = $state<'title' | 'content' | 'semantic' | 'browse'>('semantic');
	let entityType = $state<EntityType | 'all'>('all');
	let results = $state<TreeSearchResult[]>([]);
	let loading = $state(false);
	let error = $state<string | null>(null);

	// Multimodal search state
	let showImageSearch = $state(false);
	let imageResults = $state<MultimodalSearchResult[]>([]);
	let showImageResults = $state(false);

	const searchTypes = [
		{ value: 'semantic', label: 'Semantic Search' },
		{ value: 'title', label: 'Search by Title' },
		{ value: 'content', label: 'Search by Content' },
		{ value: 'browse', label: 'Browse Tree' }
	] as const;

	const entityTypes: { value: EntityType | 'all'; label: string }[] = [
		{ value: 'all', label: 'All Types' },
		{ value: 'memories', label: 'Memories' },
		{ value: 'contexts', label: 'Contexts' },
		{ value: 'artifacts', label: 'Artifacts' },
		{ value: 'documents', label: 'Documents' },
		{ value: 'voice_notes', label: 'Voice Notes' }
	];

	async function handleSearch(e?: Event) {
		e?.preventDefault();

		if (!searchQuery.trim() && searchType !== 'browse') {
			results = [];
			return;
		}

		loading = true;
		error = null;

		try {
			const searchResults = await api.searchContextTree({
				query: searchQuery,
				search_type: searchType,
				entity_type: entityType !== 'all' ? entityType : undefined,
				project_id: projectId,
				node_id: nodeId,
				limit: 50
			});
			results = searchResults;
		} catch (err) {
			console.error('Failed to search tree:', err);
			error = err instanceof Error ? err.message : 'Failed to search tree';
			results = [];
		} finally {
			loading = false;
		}
	}

	function getEntityIcon(type: string): string {
		switch (type) {
			case 'memories':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />`;
			case 'contexts':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />`;
			case 'artifacts':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9" />`;
			case 'documents':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
			case 'voice_notes':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M12 18.75a6 6 0 0 0 6-6v-1.5m-6 7.5a6 6 0 0 1-6-6v-1.5m6 7.5v3.75m-3.75 0h7.5M12 15.75a3 3 0 0 1-3-3V4.5a3 3 0 1 1 6 0v8.25a3 3 0 0 1-3 3Z" />`;
			default:
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M9.568 3H5.25A2.25 2.25 0 0 0 3 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 0 0 5.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 0 0 9.568 3Z" />`;
		}
	}

	function getEntityColor(type: string): string {
		switch (type) {
			case 'memories': return '#3b82f6';
			case 'contexts': return '#22c55e';
			case 'artifacts': return '#8b5cf6';
			case 'documents': return '#f59e0b';
			case 'voice_notes': return '#ec4899';
			default: return '#6b7280';
		}
	}

	function formatDate(dateStr: string): string {
		const date = new Date(dateStr);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		const diffHours = Math.floor(diffMs / 3600000);
		const diffDays = Math.floor(diffMs / 86400000);

		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffHours < 24) return `${diffHours}h ago`;
		if (diffDays < 7) return `${diffDays}d ago`;
		return date.toLocaleDateString();
	}

	// Auto-search when search type changes to browse
	$effect(() => {
		if (searchType === 'browse') {
			handleSearch();
		}
	});

	// Handle image search results
	function handleImageSearchResults(results: MultimodalSearchResult[], query?: string) {
		imageResults = results;
		showImageResults = true;
		showImageSearch = false;
	}

	// Close image search modal
	function closeImageSearch() {
		showImageSearch = false;
	}

	// Go back to text search results
	function backToTextResults() {
		showImageResults = false;
		imageResults = [];
	}
</script>

<div class="tree-search-panel">
	<div class="panel-header">
		<h3 class="panel-title">Tree Search</h3>
		<div class="header-hint">Search across all your knowledge base</div>
	</div>

	<div class="search-section">
		<form class="search-form" onsubmit={handleSearch}>
			<div class="search-controls">
				<div class="search-input-wrapper">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="search-icon">
						<path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
					</svg>
					<input
						type="text"
						bind:value={searchQuery}
						placeholder={searchType === 'browse' ? 'Browse all items...' : `Search by ${searchType}...`}
						class="search-input"
						disabled={searchType === 'browse'}
					/>
					{#if searchQuery && searchType !== 'browse'}
						<button
							type="button"
							class="btn-pill btn-pill-ghost clear-btn"
							onclick={() => { searchQuery = ''; results = []; }}
							aria-label="Clear search"
						>
							<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
								<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</div>
				<button type="submit" class="btn-pill btn-pill-ghost search-btn" disabled={loading || (!searchQuery && searchType !== 'browse')}>
					{loading ? 'Searching...' : 'Search'}
				</button>
				<button
					type="button"
					class="btn-pill btn-pill-ghost image-search-btn"
					onclick={() => showImageSearch = true}
					title="Multimodal Image Search"
				>
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="18" height="18">
						<path stroke-linecap="round" stroke-linejoin="round" d="m2.25 15.75 5.159-5.159a2.25 2.25 0 0 1 3.182 0l5.159 5.159m-1.5-1.5 1.409-1.409a2.25 2.25 0 0 1 3.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 0 0 1.5-1.5V6a1.5 1.5 0 0 0-1.5-1.5H3.75A1.5 1.5 0 0 0 2.25 6v12a1.5 1.5 0 0 0 1.5 1.5Zm10.5-11.25h.008v.008h-.008V8.25Zm.375 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Z" />
					</svg>
				</button>
			</div>

			<div class="filter-row">
				<div class="filter-group">
					<label for="search-type" class="filter-label">Search Type</label>
					<select id="search-type" bind:value={searchType} class="filter-select" onchange={handleSearch}>
						{#each searchTypes as type}
							<option value={type.value}>{type.label}</option>
						{/each}
					</select>
				</div>

				<div class="filter-group">
					<label for="entity-type" class="filter-label">Entity Type</label>
					<select id="entity-type" bind:value={entityType} class="filter-select" onchange={handleSearch}>
						{#each entityTypes as type}
							<option value={type.value}>{type.label}</option>
						{/each}
					</select>
				</div>
			</div>
		</form>
	</div>

	<div class="results-section">
		{#if error}
			<div class="error-state">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="error-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
				</svg>
				<p class="error-text">{error}</p>
				<button class="btn-pill btn-pill-ghost retry-btn" onclick={handleSearch}>Try Again</button>
			</div>
		{:else if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Searching...</p>
			</div>
		{:else if results.length === 0}
			<div class="empty-state">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="empty-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
				</svg>
				{#if searchQuery || searchType === 'browse'}
					<p class="empty-text">No results found</p>
					<p class="empty-hint">Try a different search term or filter</p>
				{:else}
					<p class="empty-text">Enter a search query to begin</p>
					<p class="empty-hint">Search across memories, contexts, documents, and more</p>
				{/if}
			</div>
		{:else}
			<div class="results-header">
				<span class="results-count">{results.length} result{results.length !== 1 ? 's' : ''}</span>
			</div>
			<div class="results-list">
				{#each results as result (result.id)}
					<button
						class="btn-pill btn-pill-ghost result-item"
						onclick={() => onItemSelect?.(result)}
					>
						<div class="result-icon" style="color: {getEntityColor(result.entity_type ?? result.type ?? 'all')}">
							<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="20" height="20">
								{@html getEntityIcon(result.entity_type ?? result.type ?? 'all')}
							</svg>
						</div>
						<div class="result-content">
							<div class="result-header-row">
								<span class="result-title">{result.title}</span>
								<span class="result-type-badge" style="background: {getEntityColor(result.entity_type ?? result.type ?? 'all')}15; color: {getEntityColor(result.entity_type ?? result.type ?? 'all')}">
									{result.entity_type ?? result.type ?? 'unknown'}
								</span>
							</div>
							{#if result.snippet}
								<p class="result-snippet">{result.snippet}</p>
							{/if}
							<div class="result-meta">
								{#if result.relevance_score !== undefined}
									<span class="meta-item">
										<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
											<path stroke-linecap="round" stroke-linejoin="round" d="M11.48 3.499a.562.562 0 0 1 1.04 0l2.125 5.111a.563.563 0 0 0 .475.345l5.518.442c.499.04.701.663.321.988l-4.204 3.602a.563.563 0 0 0-.182.557l1.285 5.385a.562.562 0 0 1-.84.61l-4.725-2.885a.562.562 0 0 0-.586 0L6.982 20.54a.562.562 0 0 1-.84-.61l1.285-5.386a.562.562 0 0 0-.182-.557l-4.204-3.602a.562.562 0 0 1 .321-.988l5.518-.442a.563.563 0 0 0 .475-.345L11.48 3.5Z" />
										</svg>
										{Math.round(result.relevance_score * 100)}% match
									</span>
								{/if}
								{#if result.token_count !== undefined}
									<span class="meta-item">
										<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
											<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
										</svg>
										{result.token_count} tokens
									</span>
								{/if}
								{#if result.created_at}
									<span class="meta-item">
										<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
											<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
										</svg>
										{formatDate(result.created_at)}
									</span>
								{/if}
							</div>
						</div>
					</button>
				{/each}
			</div>
		{/if}
	</div>
</div>

<!-- Image Search Modal -->
<ImageSearchModal
	bind:show={showImageSearch}
	onresults={handleImageSearchResults}
	onclose={closeImageSearch}
/>

<!-- Image Results View (replaces text results when active) -->
{#if showImageResults}
	<div class="fixed inset-0 z-40 bg-white dark:bg-gray-900 overflow-y-auto">
		<div class="max-w-7xl mx-auto p-4">
			<!-- Header -->
			<div class="flex items-center justify-between mb-6">
				<div class="flex items-center gap-3">
					<button
						onclick={backToTextResults}
						class="btn-pill btn-pill-ghost btn-pill-icon"
						title="Back to text search"
					>
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-5 h-5">
							<path stroke-linecap="round" stroke-linejoin="round" d="M10.5 19.5 3 12m0 0 7.5-7.5M3 12h18" />
						</svg>
					</button>
					<div>
						<h2 class="text-xl font-semibold">Image Search Results</h2>
						<p class="text-sm text-gray-500">{imageResults.length} result{imageResults.length !== 1 ? 's' : ''}</p>
					</div>
				</div>
				<button
					onclick={backToTextResults}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					Close
				</button>
			</div>

			<!-- Image Gallery -->
			<ImageGalleryView results={imageResults} />
		</div>
	</div>
{/if}

<style>
	.tree-search-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--color-bg);
	}

	.panel-header {
		padding: 20px 24px;
		border-bottom: 1px solid var(--color-border);
	}

	.panel-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0 0 4px 0;
	}

	.header-hint {
		font-size: 13px;
		color: var(--color-text-muted);
	}

	.search-section {
		padding: 16px 24px;
		border-bottom: 1px solid var(--color-border);
	}

	.search-form {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.search-controls {
		display: flex;
		gap: 8px;
	}

	.search-input-wrapper {
		flex: 1;
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 14px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		transition: all 0.15s ease;
	}

	.search-input-wrapper:focus-within {
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.search-icon {
		width: 18px;
		height: 18px;
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.search-input {
		flex: 1;
		border: none;
		background: transparent;
		font-size: 14px;
		color: var(--color-text);
		outline: none;
	}

	.search-input::placeholder {
		color: var(--color-text-muted);
	}

	.search-input:disabled {
		opacity: 0.5;
		cursor: not-allowed;
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
		transition: all 0.15s ease;
	}

	.clear-btn:hover {
		color: var(--color-text);
		background: var(--color-bg-tertiary);
	}

	.search-btn {
		padding: 10px 20px;
		font-size: 14px;
		font-weight: 500;
		color: white;
		background: #3b82f6;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		white-space: nowrap;
	}

	.search-btn:hover:not(:disabled) {
		background: #2563eb;
	}

	.search-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.image-search-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 10px 12px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		color: var(--color-text);
	}

	.image-search-btn:hover {
		background: var(--color-bg-tertiary);
		border-color: #3b82f6;
		color: #3b82f6;
	}

	.filter-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 12px;
	}

	.filter-group {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.filter-label {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.5px;
	}

	.filter-select {
		padding: 8px 12px;
		font-size: 13px;
		color: var(--color-text);
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		cursor: pointer;
		outline: none;
		transition: all 0.15s ease;
	}

	.filter-select:focus {
		border-color: #3b82f6;
	}

	.results-section {
		flex: 1;
		overflow-y: auto;
		padding: 16px 24px;
	}

	.results-header {
		margin-bottom: 12px;
		padding-bottom: 8px;
		border-bottom: 1px solid var(--color-border);
	}

	.results-count {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	.results-list {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.result-item {
		display: flex;
		gap: 12px;
		padding: 16px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: left;
		width: 100%;
	}

	.result-item:hover {
		background: var(--color-bg-tertiary);
		border-color: #3b82f6;
		transform: translateY(-1px);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
	}

	.result-icon {
		flex-shrink: 0;
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: currentColor;
		background: color-mix(in srgb, currentColor 10%, transparent);
		border-radius: 8px;
	}

	.result-content {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.result-header-row {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.result-title {
		font-size: 14px;
		font-weight: 600;
		color: var(--color-text);
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.result-type-badge {
		padding: 2px 8px;
		font-size: 11px;
		font-weight: 500;
		border-radius: 4px;
		text-transform: uppercase;
		letter-spacing: 0.3px;
		flex-shrink: 0;
	}

	.result-snippet {
		font-size: 13px;
		color: var(--color-text-muted);
		line-height: 1.5;
		margin: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
	}

	.result-meta {
		display: flex;
		align-items: center;
		gap: 12px;
		flex-wrap: wrap;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 12px;
		color: var(--color-text-muted);
	}

	.meta-item svg {
		flex-shrink: 0;
	}

	.loading-state,
	.empty-state,
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 60px 24px;
		text-align: center;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--color-border);
		border-top-color: #3b82f6;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		margin-bottom: 16px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.loading-state p,
	.empty-state p,
	.error-state p {
		margin: 0;
	}

	.empty-icon,
	.error-icon {
		width: 48px;
		height: 48px;
		color: var(--color-text-muted);
		margin-bottom: 16px;
	}

	.error-icon {
		color: #ef4444;
	}

	.empty-text,
	.error-text {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
		margin-bottom: 4px;
	}

	.empty-hint {
		font-size: 13px;
		color: var(--color-text-muted);
	}

	.retry-btn {
		margin-top: 16px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		color: white;
		background: #3b82f6;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.retry-btn:hover {
		background: #2563eb;
	}
</style>
