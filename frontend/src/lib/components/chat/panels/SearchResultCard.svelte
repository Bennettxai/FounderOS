<script lang="ts">
	import { FileText, Plus, Eye, TrendingUp } from 'lucide-svelte';
	import type { HybridSearchResult } from '$lib/api/rag';

	interface Props {
		result: HybridSearchResult;
		showScores?: boolean;
		compact?: boolean;
		onpreview?: (data: { result: HybridSearchResult }) => void;
		onaddToContext?: (data: { result: HybridSearchResult }) => void;
	}

	let { result, showScores = true, compact = false, onpreview, onaddToContext }: Props = $props();

	// Format score as percentage
	function formatScore(score: number): string {
		return `${Math.round(score * 100)}%`;
	}

	// Get score color based on value
	function getScoreColor(score: number): string {
		if (score >= 0.8) return 'text-green-600 dark:text-green-400';
		if (score >= 0.6) return 'text-blue-600 dark:text-blue-400';
		if (score >= 0.4) return 'text-yellow-600 dark:text-yellow-400';
		return 'text-gray-600 dark:text-gray-400';
	}

	// Get strategy badge color
	function getStrategyColor(strategy: string): string {
		switch (strategy) {
			case 'semantic':
				return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400';
			case 'keyword':
				return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400';
			case 'hybrid':
				return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400';
			default:
				return 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400';
		}
	}

	// Truncate content
	function truncateContent(content: string, maxLength: number = 200): string {
		if (content.length <= maxLength) return content;
		return content.substring(0, maxLength).trim() + '...';
	}
</script>

<div
	class={`group rounded-lg border dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-blue-300 dark:hover:border-blue-700 transition-all ${
		compact ? 'p-3' : 'p-4'
	}`}
>
	<!-- Header -->
	<div class="flex items-start justify-between gap-2 mb-2">
		<div class="flex items-start gap-2 flex-1 min-w-0">
			<FileText class="w-4 h-4 mt-0.5 text-gray-400 flex-shrink-0" />
			<div class="flex-1 min-w-0">
				<h3 class="font-medium text-sm truncate">{result.context_name}</h3>
				<p class="text-xs text-gray-500 dark:text-gray-400">{result.context_type}</p>
			</div>
		</div>

		<!-- Strategy Badge -->
		<span
			class={`px-2 py-0.5 rounded text-xs font-medium flex-shrink-0 ${getStrategyColor(result.search_strategy)}`}
		>
			{result.search_strategy}
		</span>
	</div>

	<!-- Content Preview -->
	<div class="mb-3">
		<p class="text-sm text-gray-700 dark:text-gray-300 leading-relaxed">
			{truncateContent(result.content, compact ? 150 : 200)}
		</p>
	</div>

	<!-- Scores (if enabled) -->
	{#if showScores}
		<div class="flex items-center gap-4 mb-3 text-xs">
			<div class="flex items-center gap-1">
				<TrendingUp class="w-3 h-3 text-gray-400" />
				<span class="text-gray-500 dark:text-gray-400">Hybrid:</span>
				<span class={`font-medium ${getScoreColor(result.hybrid_score)}`}>
					{formatScore(result.hybrid_score)}
				</span>
			</div>
			{#if result.semantic_score > 0}
				<div class="flex items-center gap-1">
					<span class="text-gray-500 dark:text-gray-400">Semantic:</span>
					<span class={`font-medium ${getScoreColor(result.semantic_score)}`}>
						{formatScore(result.semantic_score)}
					</span>
				</div>
			{/if}
			{#if result.keyword_score > 0}
				<div class="flex items-center gap-1">
					<span class="text-gray-500 dark:text-gray-400">Keyword:</span>
					<span class={`font-medium ${getScoreColor(result.keyword_score)}`}>
						{formatScore(result.keyword_score)}
					</span>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Actions -->
	<div class="flex items-center justify-between gap-2">
		<div class="text-xs text-gray-400">
			{#if result.created_at}
				{new Date(result.created_at).toLocaleDateString()}
			{/if}
		</div>

		<div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
			<button
				onclick={() => onpreview?.({ result })}
				class="px-2 py-1 text-xs rounded hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-1"
				title="Preview full document"
			>
				<Eye class="w-3 h-3" />
				Preview
			</button>
			<button
				onclick={() => onaddToContext?.({ result })}
				class="px-2 py-1 text-xs rounded bg-blue-500 text-white hover:bg-blue-600 flex items-center gap-1"
				title="Add to chat context"
			>
				<Plus class="w-3 h-3" />
				Add to Context
			</button>
		</div>
	</div>
</div>
