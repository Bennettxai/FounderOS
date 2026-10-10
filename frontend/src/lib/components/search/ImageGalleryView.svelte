<script lang="ts">
	import type { MultimodalSearchResult } from '$lib/api/multimodal-search';
	import { getImageDataUrl } from '$lib/api/multimodal-search';
	import { getApiBaseUrl } from '$lib/api/base';
	import { Image as ImageIcon, X, ExternalLink, Info } from 'lucide-svelte';

	interface Props {
		results: MultimodalSearchResult[];
		loading?: boolean;
		onselect?: (data: { result: MultimodalSearchResult }) => void;
		onclose?: () => void;
	}

	let { results = [], loading = false, onselect, onclose }: Props = $props();

	// Selected image for preview
	let selectedImage = $state<MultimodalSearchResult | null>(null);

	// Get image URL for result
	function getImageUrl(result: MultimodalSearchResult): string {
		if (result.image_data) {
			return `data:image/png;base64,${result.image_data}`;
		}
		if (result.image_id) {
			return getImageDataUrl(result.image_id, getApiBaseUrl());
		}
		return '';
	}

	// Handle image click
	function handleImageClick(result: MultimodalSearchResult) {
		selectedImage = result;
		onselect?.({ result });
	}

	// Close preview
	function closePreview() {
		selectedImage = null;
	}

	// Get result type badge color
	function getBadgeColor(type: string): string {
		switch (type) {
			case 'image':
				return 'bg-blue-500';
			case 'text':
				return 'bg-green-500';
			case 'hybrid':
				return 'bg-purple-500';
			default:
				return 'bg-gray-500';
		}
	}
</script>

{#if loading}
	<div class="flex items-center justify-center p-12">
		<div class="w-12 h-12 border-4 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
	</div>
{:else if results.length === 0}
	<div class="flex flex-col items-center justify-center p-12 text-gray-500 dark:text-gray-400">
		<ImageIcon class="w-16 h-16 mb-4 opacity-50" />
		<p class="text-lg font-medium">No images found</p>
		<p class="text-sm">Try adjusting your search or upload a different image</p>
	</div>
{:else}
	<!-- Image Grid -->
	<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4 p-4">
		{#each results as result (result.id)}
			<div
				class="group relative aspect-square rounded-lg overflow-hidden border dark:border-gray-700 cursor-pointer hover:ring-2 hover:ring-blue-500 transition"
				onclick={() => handleImageClick(result)}
				role="button"
				tabindex="0"
			>
				<!-- Image -->
				<img
					src={getImageUrl(result)}
					alt={result.image_caption || 'Search result'}
					class="w-full h-full object-cover"
					loading="lazy"
				/>

				<!-- Overlay with info on hover -->
				<div
					class="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity flex flex-col justify-between p-3"
				>
					<!-- Top: Type badge and score -->
					<div class="flex items-start justify-between">
						<span
							class={`${getBadgeColor(result.type)} text-white text-xs px-2 py-1 rounded`}
						>
							{result.type}
						</span>
						<span class="text-white text-xs bg-black/50 px-2 py-1 rounded">
							{(result.similarity * 100).toFixed(0)}%
						</span>
					</div>

					<!-- Bottom: Caption -->
					<div class="text-white text-sm line-clamp-2">
						{result.image_caption || result.title || 'No caption'}
					</div>
				</div>
			</div>
		{/each}
	</div>
{/if}

<!-- Image Preview Modal -->
{#if selectedImage}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
		onclick={closePreview}
		onkeydown={(e) => e.key === 'Escape' && closePreview()}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div
			class="relative max-w-4xl max-h-[90vh] w-full mx-4"
			onclick={(e) => e.stopPropagation()}
			role="document"
		>
			<!-- Close button -->
			<button
				onclick={closePreview}
				class="btn-pill btn-pill-primary btn-pill-icon absolute top-4 right-4 z-10"
				aria-label="Close preview"
			>
				<X class="w-6 h-6" />
			</button>

			<!-- Image -->
			<img
				src={getImageUrl(selectedImage)}
				alt={selectedImage.image_caption || 'Preview'}
				class="w-full h-auto max-h-[70vh] object-contain rounded-lg"
			/>

			<!-- Info panel -->
			<div class="mt-4 p-6 bg-white dark:bg-gray-800 rounded-lg">
				<div class="flex items-start justify-between mb-4">
					<div class="flex items-center gap-2">
						<Info class="w-5 h-5 text-gray-500" />
						<h3 class="text-lg font-semibold">Image Details</h3>
					</div>
					<span class={`${getBadgeColor(selectedImage.type)} text-white text-sm px-3 py-1 rounded`}>
						{selectedImage.type}
					</span>
				</div>

				<div class="space-y-3 text-sm">
					<!-- Caption -->
					{#if selectedImage.image_caption}
						<div>
							<span class="font-medium text-gray-700 dark:text-gray-300">Caption:</span>
							<p class="text-gray-600 dark:text-gray-400 mt-1">
								{selectedImage.image_caption}
							</p>
						</div>
					{/if}

					<!-- Similarity Score -->
					<div>
						<span class="font-medium text-gray-700 dark:text-gray-300">Similarity:</span>
						<div class="flex items-center gap-2 mt-1">
							<div class="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
								<div
									class="h-full bg-blue-500 transition-all"
									style={`width: ${selectedImage.similarity * 100}%`}
								></div>
							</div>
							<span class="text-gray-600 dark:text-gray-400">
								{(selectedImage.similarity * 100).toFixed(1)}%
							</span>
						</div>
					</div>

					<!-- Source -->
					<div>
						<span class="font-medium text-gray-700 dark:text-gray-300">Source:</span>
						<span class="text-gray-600 dark:text-gray-400 ml-2">
							{selectedImage.source}
						</span>
					</div>

					<!-- Metadata -->
					{#if selectedImage.metadata && Object.keys(selectedImage.metadata).length > 0}
						<div>
							<span class="font-medium text-gray-700 dark:text-gray-300">Metadata:</span>
							<pre
								class="mt-1 p-2 bg-gray-100 dark:bg-gray-900 rounded text-xs overflow-x-auto">
								{JSON.stringify(selectedImage.metadata, null, 2)}
							</pre>
						</div>
					{/if}

					<!-- Action buttons -->
					<div class="flex gap-2 pt-4 border-t dark:border-gray-700">
						{#if selectedImage.context_id}
							<button
								class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
							>
								<ExternalLink class="w-4 h-4" />
								View Context
							</button>
						{/if}
						<button
							onclick={closePreview}
							class="btn-pill btn-pill-ghost btn-pill-sm flex items-center gap-2"
						>
							Close
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}
