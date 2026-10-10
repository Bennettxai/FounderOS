<script lang="ts">
	import { X, FileText, Plus, Copy, Check } from 'lucide-svelte';
	import type { HybridSearchResult } from '$lib/api/rag';

	interface Props {
		show: boolean;
		result: HybridSearchResult | null;
		fullContent?: string;
		onclose?: () => void;
		onaddToContext?: (data: { result: HybridSearchResult }) => void;
	}

	let { show = $bindable(false), result, fullContent, onclose, onaddToContext }: Props = $props();

	let copied = $state(false);

	// Close modal
	function closeModal() {
		show = false;
		onclose?.();
	}

	// Copy content to clipboard
	async function copyContent() {
		if (!result) return;
		const content = fullContent || result.content;
		try {
			await navigator.clipboard.writeText(content);
			copied = true;
			setTimeout(() => (copied = false), 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	// Add to context and close
	function handleAddToContext() {
		if (result) {
			onaddToContext?.({ result });
			closeModal();
		}
	}

	// Keyboard shortcuts
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			closeModal();
		} else if (event.key === 'c' && (event.metaKey || event.ctrlKey)) {
			event.preventDefault();
			copyContent();
		}
	}

	// Highlight query matches in content
	function highlightContent(content: string, query?: string): string {
		if (!query) return content;

		// Simple highlighting - escape HTML first
		let highlighted = content
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;');

		// Highlight query words (case insensitive)
		const words = query.split(/\s+/).filter(w => w.length > 2);
		words.forEach(word => {
			const regex = new RegExp(`(${word})`, 'gi');
			highlighted = highlighted.replace(regex, '<mark class="bg-yellow-200 dark:bg-yellow-700">$1</mark>');
		});

		return highlighted;
	}

	// Get display content
	let displayContent = $derived.by(() => {
		if (!result) return '';
		return fullContent || result.content;
	});
</script>

{#if show && result}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
		onclick={closeModal}
		onkeydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div
			class="relative w-full max-w-4xl max-h-[90vh] overflow-hidden bg-white dark:bg-gray-800 rounded-lg shadow-xl flex flex-col"
			onclick={(e) => e.stopPropagation()}
			role="document"
		>
			<!-- Header -->
			<div
				class="flex items-center justify-between p-4 border-b dark:border-gray-700 bg-white dark:bg-gray-800 flex-shrink-0"
			>
				<div class="flex items-center gap-3 flex-1 min-w-0">
					<FileText class="w-5 h-5 text-blue-500 flex-shrink-0" />
					<div class="flex-1 min-w-0">
						<h2 class="text-lg font-semibold truncate">{result.context_name}</h2>
						<p class="text-sm text-gray-500 dark:text-gray-400">
							{result.context_type} • Block: {result.block_id}
						</p>
					</div>
				</div>
				<button
					onclick={closeModal}
					class="btn-pill btn-pill-ghost btn-pill-icon flex-shrink-0"
					aria-label="Close"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Metadata Bar -->
			<div class="px-4 py-2 bg-gray-50 dark:bg-gray-900 border-b dark:border-gray-700 flex items-center gap-4 text-sm flex-shrink-0">
				<div class="flex items-center gap-2">
					<span class="text-gray-500 dark:text-gray-400">Block Type:</span>
					<span class="font-medium">{result.block_type}</span>
				</div>
				<div class="flex items-center gap-2">
					<span class="text-gray-500 dark:text-gray-400">Strategy:</span>
					<span class="font-medium capitalize">{result.search_strategy}</span>
				</div>
				<div class="flex items-center gap-2">
					<span class="text-gray-500 dark:text-gray-400">Score:</span>
					<span class="font-medium text-blue-600 dark:text-blue-400">
						{Math.round(result.hybrid_score * 100)}%
					</span>
				</div>
			</div>

			<!-- Content Area -->
			<div class="flex-1 overflow-y-auto p-6">
				<div class="prose dark:prose-invert max-w-none">
					{@html highlightContent(displayContent)}
				</div>
			</div>

			<!-- Footer Actions -->
			<div
				class="flex items-center justify-between p-4 border-t dark:border-gray-700 bg-gray-50 dark:bg-gray-900 flex-shrink-0"
			>
				<p class="text-xs text-gray-500">
					Press <kbd class="px-1 py-0.5 bg-white dark:bg-gray-800 border rounded">Esc</kbd> to close,
					<kbd class="px-1 py-0.5 bg-white dark:bg-gray-800 border rounded">Cmd+C</kbd> to copy
				</p>
				<div class="flex gap-2">
					<button
						onclick={copyContent}
						class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-2"
					>
						{#if copied}
							<Check class="w-4 h-4 text-green-500" />
							Copied!
						{:else}
							<Copy class="w-4 h-4" />
							Copy
						{/if}
					</button>
					<button
						onclick={handleAddToContext}
						class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
					>
						<Plus class="w-4 h-4" />
						Add to Context
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	/* Styling for highlighted content */
	:global(mark) {
		padding: 0.125rem 0.25rem;
		border-radius: 0.25rem;
	}

	/* Prose styling for content */
	.prose {
		color: inherit;
		font-size: 0.875rem;
		line-height: 1.75;
	}

	.prose :global(p) {
		margin-bottom: 1rem;
	}

	.prose :global(h1),
	.prose :global(h2),
	.prose :global(h3) {
		margin-top: 1.5rem;
		margin-bottom: 0.75rem;
		font-weight: 600;
	}

	.prose :global(code) {
		padding: 0.125rem 0.25rem;
		background-color: rgba(0, 0, 0, 0.05);
		border-radius: 0.25rem;
		font-size: 0.875em;
	}

	:global(.dark) .prose :global(code) {
		background-color: rgba(255, 255, 255, 0.1);
	}

	.prose :global(pre) {
		padding: 1rem;
		background-color: rgba(0, 0, 0, 0.05);
		border-radius: 0.375rem;
		overflow-x: auto;
		margin: 1rem 0;
	}

	:global(.dark) .prose :global(pre) {
		background-color: rgba(255, 255, 255, 0.05);
	}
</style>
