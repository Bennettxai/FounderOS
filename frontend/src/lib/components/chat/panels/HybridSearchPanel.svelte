<script lang="ts">
	import { Search, X, Settings2, Sparkles, TrendingUp, Loader2, Info } from 'lucide-svelte';
	import { hybridSearch, agenticRAG, type HybridSearchResult, type AgenticRAGResult } from '$lib/api/rag';
	import SearchResultCard from './SearchResultCard.svelte';
	import DocumentPreview from './DocumentPreview.svelte';

	interface Props {
		show: boolean;
		workspaceId?: string;
		projectId?: string;
		onclose?: () => void;
		onaddToContext?: (data: { result: HybridSearchResult; query: string }) => void;
	}

	let { show = $bindable(false), workspaceId, projectId, onclose, onaddToContext }: Props = $props();

	// Search state
	let searchQuery = $state('');
	let searching = $state(false);
	let error = $state<string | null>(null);
	let results = $state<HybridSearchResult[]>([]);
	let useAgenticRAG = $state(true); // Use intelligent search by default
	let showAdvancedOptions = $state(false);

	// Search options
	let semanticWeight = $state(0.7);
	let keywordWeight = $state(0.3);
	let maxResults = $state(10);
	let minSimilarity = $state(0.3);
	let usePersonalization = $state(false);
	let enableReranking = $state(true);

	// Agentic RAG metadata
	let queryIntent = $state<string | null>(null);
	let strategyUsed = $state<string | null>(null);
	let strategyReasoning = $state<string | null>(null);
	let qualityScore = $state<number | null>(null);
	let processingTime = $state<number | null>(null);

	// Document preview
	let showPreview = $state(false);
	let previewResult = $state<HybridSearchResult | null>(null);

	// Perform search
	async function performSearch() {
		if (!searchQuery.trim()) {
			error = 'Please enter a search query';
			return;
		}

		searching = true;
		error = null;
		results = [];
		queryIntent = null;
		strategyUsed = null;
		strategyReasoning = null;
		qualityScore = null;
		processingTime = null;

		try {
			if (useAgenticRAG) {
				// Agentic RAG with intelligent retrieval
				const response = await agenticRAG({
					query: searchQuery,
					max_results: maxResults,
					min_quality_score: minSimilarity,
					project_id: projectId,
					use_personalization: usePersonalization,
					workspace_id: workspaceId
				});

				results = response.results as HybridSearchResult[];
				queryIntent = response.query_intent;
				strategyUsed = response.strategy_used;
				strategyReasoning = response.strategy_reasoning;
				qualityScore = response.quality_score;
				processingTime = response.processing_time_ms;
			} else {
				// Standard hybrid search
				const response = await hybridSearch({
					query: searchQuery,
					semantic_weight: semanticWeight,
					keyword_weight: keywordWeight,
					max_results: maxResults,
					min_similarity: minSimilarity,
					project_id: projectId,
					workspace_id: workspaceId
				});

				results = response.results;
			}
		} catch (err: any) {
			error = err.message || 'Search failed';
			console.error('Search error:', err);
		} finally {
			searching = false;
		}
	}

	// Handle keyboard shortcuts
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			closePanel();
		} else if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
			performSearch();
		}
	}

	// Close panel
	function closePanel() {
		show = false;
		onclose?.();
	}

	// Handle result preview
	function handlePreview(data: { result: HybridSearchResult }) {
		previewResult = data.result;
		showPreview = true;
	}

	// Handle add to context
	function handleAddToContext(data: { result: HybridSearchResult }) {
		onaddToContext?.({
			result: data.result,
			query: searchQuery
		});
	}

	// Adjust weights to sum to 1.0
	function normalizeWeights() {
		const sum = semanticWeight + keywordWeight;
		if (sum > 0) {
			semanticWeight = semanticWeight / sum;
			keywordWeight = keywordWeight / sum;
		}
	}

	// Reset to defaults
	function resetOptions() {
		semanticWeight = 0.7;
		keywordWeight = 0.3;
		maxResults = 10;
		minSimilarity = 0.3;
		usePersonalization = false;
		enableReranking = true;
	}
</script>

{#if show}
	<div
		class="fixed inset-0 z-40 flex items-center justify-center bg-black/50 backdrop-blur-sm"
		onclick={closePanel}
		onkeydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div
			class="relative w-full max-w-5xl max-h-[90vh] overflow-hidden bg-white dark:bg-gray-800 rounded-lg shadow-xl flex flex-col"
			onclick={(e) => e.stopPropagation()}
			role="document"
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between p-4 border-b dark:border-gray-700 bg-white dark:bg-gray-800 flex-shrink-0"
			>
				<div class="flex items-center gap-2">
					<Search class="w-5 h-5 text-blue-500" />
					<h2 class="text-lg font-semibold">Knowledge Search</h2>
					{#if useAgenticRAG}
						<span
							class="px-2 py-0.5 text-xs rounded bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400 flex items-center gap-1"
						>
							<Sparkles class="w-3 h-3" />
							Intelligent
						</span>
					{/if}
				</div>
				<button
					onclick={closePanel}
					class="p-1 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
					aria-label="Close"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Search Bar -->
			<div class="p-4 border-b dark:border-gray-700 flex-shrink-0">
				<div class="flex gap-2">
					<div class="flex-1 relative">
						<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
						<input
							type="text"
							bind:value={searchQuery}
							placeholder="Search your knowledge base..."
							class="w-full pl-10 pr-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
							onkeydown={(e) => {
								if (e.key === 'Enter' && !e.metaKey && !e.ctrlKey) {
									performSearch();
								}
							}}
						/>
					</div>
					<button
						onclick={performSearch}
						disabled={searching || !searchQuery.trim()}
						class="btn-pill btn-pill-primary flex items-center gap-2"
					>
						{#if searching}
							<Loader2 class="w-4 h-4 animate-spin" />
							Searching...
						{:else}
							<Search class="w-4 h-4" />
							Search
						{/if}
					</button>
					<button
						onclick={() => (showAdvancedOptions = !showAdvancedOptions)}
						class={`px-3 py-2 border rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${showAdvancedOptions ? 'bg-gray-100 dark:bg-gray-700' : ''}`}
						title="Advanced options"
					>
						<Settings2 class="w-4 h-4" />
					</button>
				</div>

				<!-- Advanced Options -->
				{#if showAdvancedOptions}
					<div class="mt-4 p-4 bg-gray-50 dark:bg-gray-900 rounded-lg space-y-4">
						<!-- Search Mode -->
						<div class="flex items-center gap-4">
							<label class="flex items-center gap-2 cursor-pointer">
								<input
									type="checkbox"
									bind:checked={useAgenticRAG}
									class="rounded"
								/>
								<span class="text-sm font-medium">Use Intelligent Search (Agentic RAG)</span>
							</label>
							{#if usePersonalization}
								<label class="flex items-center gap-2 cursor-pointer">
									<input
										type="checkbox"
										bind:checked={usePersonalization}
										class="rounded"
									/>
									<span class="text-sm">Personalization</span>
								</label>
							{/if}
						</div>

						<!-- Weights (only for standard hybrid search) -->
						{#if !useAgenticRAG}
							<div class="grid grid-cols-2 gap-4">
								<div>
									<label class="block text-sm mb-1">
										Semantic Weight: {semanticWeight.toFixed(2)}
									</label>
									<input
										type="range"
										min="0"
										max="1"
										step="0.1"
										bind:value={semanticWeight}
										onchange={normalizeWeights}
										class="w-full"
									/>
								</div>
								<div>
									<label class="block text-sm mb-1">
										Keyword Weight: {keywordWeight.toFixed(2)}
									</label>
									<input
										type="range"
										min="0"
										max="1"
										step="0.1"
										bind:value={keywordWeight}
										onchange={normalizeWeights}
										class="w-full"
									/>
								</div>
							</div>
						{/if}

						<!-- Other Options -->
						<div class="grid grid-cols-2 gap-4">
							<div>
								<label class="block text-sm mb-1">Max Results: {maxResults}</label>
								<input
									type="range"
									min="5"
									max="50"
									step="5"
									bind:value={maxResults}
									class="w-full"
								/>
							</div>
							<div>
								<label class="block text-sm mb-1">
									Min Similarity: {minSimilarity.toFixed(2)}
								</label>
								<input
									type="range"
									min="0.1"
									max="0.9"
									step="0.1"
									bind:value={minSimilarity}
									class="w-full"
								/>
							</div>
						</div>

						<button
							onclick={resetOptions}
							class="text-sm text-blue-500 hover:text-blue-600"
						>
							Reset to defaults
						</button>
					</div>
				{/if}
			</div>

			<!-- Metadata Bar (for Agentic RAG) -->
			{#if queryIntent && strategyUsed}
				<div class="px-4 py-2 bg-blue-50 dark:bg-blue-900/20 border-b dark:border-gray-700 flex items-center gap-4 text-sm flex-shrink-0">
					<div class="flex items-center gap-2">
						<Info class="w-4 h-4 text-blue-500" />
						<span class="text-gray-600 dark:text-gray-400">Intent:</span>
						<span class="font-medium capitalize">{queryIntent.replace('_', ' ')}</span>
					</div>
					<div class="flex items-center gap-2">
						<TrendingUp class="w-4 h-4 text-blue-500" />
						<span class="text-gray-600 dark:text-gray-400">Strategy:</span>
						<span class="font-medium capitalize">{strategyUsed.replace('_', ' ')}</span>
					</div>
					{#if qualityScore !== null}
						<div class="flex items-center gap-2">
							<span class="text-gray-600 dark:text-gray-400">Quality:</span>
							<span class="font-medium text-blue-600 dark:text-blue-400">
								{Math.round(qualityScore * 100)}%
							</span>
						</div>
					{/if}
					{#if processingTime !== null}
						<div class="flex items-center gap-2">
							<span class="text-gray-600 dark:text-gray-400">Time:</span>
							<span class="font-medium">{processingTime}ms</span>
						</div>
					{/if}
				</div>
				{#if strategyReasoning}
					<div class="px-4 py-2 bg-blue-50 dark:bg-blue-900/20 border-b dark:border-gray-700 text-sm flex-shrink-0">
						<p class="text-gray-700 dark:text-gray-300">{strategyReasoning}</p>
					</div>
				{/if}
			{/if}

			<!-- Results Area -->
			<div class="flex-1 overflow-y-auto p-4">
				{#if error}
					<div class="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
						<p class="text-sm text-red-600 dark:text-red-400">{error}</p>
					</div>
				{:else if searching}
					<div class="flex items-center justify-center py-12">
						<div class="text-center">
							<Loader2 class="w-8 h-8 animate-spin mx-auto mb-2 text-blue-500" />
							<p class="text-sm text-gray-500">Searching knowledge base...</p>
						</div>
					</div>
				{:else if results.length === 0 && searchQuery}
					<div class="flex items-center justify-center py-12">
						<div class="text-center">
							<Search class="w-12 h-12 mx-auto mb-2 text-gray-300 dark:text-gray-600" />
							<p class="text-sm text-gray-500">No results found</p>
							<p class="text-xs text-gray-400 mt-1">Try adjusting your search query or filters</p>
						</div>
					</div>
				{:else if results.length > 0}
					<div class="space-y-3">
						<div class="flex items-center justify-between mb-2">
							<p class="text-sm text-gray-600 dark:text-gray-400">
								Found {results.length} {results.length === 1 ? 'result' : 'results'}
							</p>
						</div>
						{#each results as result (result.context_id + result.block_id)}
							<SearchResultCard
								{result}
								showScores={true}
								onpreview={handlePreview}
								onaddToContext={handleAddToContext}
							/>
						{/each}
					</div>
				{:else}
					<div class="flex items-center justify-center py-12">
						<div class="text-center">
							<Search class="w-12 h-12 mx-auto mb-2 text-gray-300 dark:text-gray-600" />
							<p class="text-sm text-gray-500">Search your knowledge base</p>
							<p class="text-xs text-gray-400 mt-1">Enter a query and press Enter or click Search</p>
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="p-4 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-900 flex items-center justify-between text-xs text-gray-500 flex-shrink-0">
				<p>
					Press <kbd class="px-1 py-0.5 bg-white dark:bg-gray-800 border rounded">Esc</kbd> to close,
					<kbd class="px-1 py-0.5 bg-white dark:bg-gray-800 border rounded">Enter</kbd> to search
				</p>
				{#if results.length > 0}
					<p>Click "Add to Context" to include results in your chat</p>
				{/if}
			</div>
		</div>
	</div>
{/if}

<!-- Document Preview Modal -->
<DocumentPreview
	bind:show={showPreview}
	result={previewResult}
	onclose={() => (showPreview = false)}
	onaddToContext={handleAddToContext}
/>
