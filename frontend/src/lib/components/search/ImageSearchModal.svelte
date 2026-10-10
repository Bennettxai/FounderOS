<script lang="ts">
	import {
		multimodalSearch,
		searchSimilarImages,
		searchImagesByText,
		createImagePreview,
		type MultimodalSearchOptions,
		type MultimodalSearchResult
	} from '$lib/api/multimodal-search';
	import { Upload, Search, X, Image as ImageIcon, Sparkles } from 'lucide-svelte';

	interface Props {
		show: boolean;
		mode?: 'multimodal' | 'image_similarity' | 'text_to_image';
		onresults?: (results: MultimodalSearchResult[], query?: string) => void;
		onclose?: () => void;
	}

	let { show = $bindable(false), mode = $bindable('multimodal'), onresults, onclose }: Props = $props();


	// State
	let searchQuery = $state('');
	let selectedImage: File | null = $state(null);
	let imagePreview = $state<string | null>(null);
	let searching = $state(false);
	let error = $state<string | null>(null);

	// Search options
	let options = $state<MultimodalSearchOptions>({
		semantic_weight: 0.4,
		keyword_weight: 0.3,
		image_weight: 0.3,
		max_results: 20,
		include_text: true,
		include_images: true,
		rerank_enabled: true
	});

	// File input
	let fileInput: HTMLInputElement;

	// Handle image selection
	async function handleImageSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		const files = target.files;

		if (files && files.length > 0) {
			const file = files[0];

			// Validate image
			if (!file.type.startsWith('image/')) {
				error = 'Please select an image file';
				return;
			}

			if (file.size > 10 * 1024 * 1024) {
				// 10MB limit
				error = 'Image must be smaller than 10MB';
				return;
			}

			selectedImage = file;
			error = null;

			// Create preview
			try {
				const preview = await createImagePreview(file);
				imagePreview = preview.preview_url;
			} catch (err) {
				error = 'Failed to load image preview';
			}
		}
	}

	// Handle drag & drop
	let dragOver = $state(false);

	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		dragOver = true;
	}

	function handleDragLeave() {
		dragOver = false;
	}

	async function handleDrop(event: DragEvent) {
		event.preventDefault();
		dragOver = false;

		const files = event.dataTransfer?.files;
		if (files && files.length > 0) {
			const file = files[0];
			if (file.type.startsWith('image/')) {
				selectedImage = file;
				const preview = await createImagePreview(file);
				imagePreview = preview.preview_url;
			} else {
				error = 'Please drop an image file';
			}
		}
	}

	// Remove image
	function removeImage() {
		selectedImage = null;
		if (imagePreview) {
			URL.revokeObjectURL(imagePreview);
			imagePreview = null;
		}
		if (fileInput) {
			fileInput.value = '';
		}
	}

	// Perform search
	async function performSearch() {
		if (!selectedImage && !searchQuery) {
			error = 'Please provide an image or text query';
			return;
		}

		searching = true;
		error = null;

		try {
			let results: MultimodalSearchResult[] = [];

			if (mode === 'multimodal') {
				// Multimodal search (text + image)
				const response = await multimodalSearch({
					...options,
					query: searchQuery || undefined,
					image: selectedImage || undefined
				});
				results = response.results;
			} else if (mode === 'image_similarity' && selectedImage) {
				// Image similarity search
				const response = await searchSimilarImages({
					image: selectedImage,
					max_results: options.max_results
				});
				results = response.results.map((img) => ({
					id: img.id,
					type: 'image' as const,
					score: 1.0,
					similarity: 0.9,
					image_id: img.id,
					image_caption: img.caption,
					user_id: img.user_id,
					source: 'image',
					metadata: img.metadata
				}));
			} else if (mode === 'text_to_image' && searchQuery) {
				// Cross-modal: text → images
				const response = await searchImagesByText({
					query: searchQuery,
					max_results: options.max_results
				});
				results = response.results;
			}

			onresults?.(results, searchQuery);
			show = false;
		} catch (err: any) {
			error = err.message || 'Search failed';
		} finally {
			searching = false;
		}
	}

	// Close modal
	function closeModal() {
		show = false;
		onclose?.();
	}

	// Keyboard shortcuts
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			closeModal();
		} else if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
			performSearch();
		}
	}
</script>

{#if show}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
		onclick={closeModal}
		onkeydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		tabindex="-1"
	>
		<div
			class="relative w-full max-w-2xl max-h-[90vh] overflow-y-auto bg-white dark:bg-gray-800 rounded-lg shadow-xl"
			onclick={(e) => e.stopPropagation()}
			role="document"
		>
			<!-- Header -->
			<div class="sticky top-0 z-10 flex items-center justify-between p-4 border-b dark:border-gray-700 bg-white dark:bg-gray-800">
				<div class="flex items-center gap-2">
					{#if mode === 'multimodal'}
						<Sparkles class="w-5 h-5 text-purple-500" />
						<h2 class="text-lg font-semibold">Multimodal Search</h2>
					{:else if mode === 'image_similarity'}
						<ImageIcon class="w-5 h-5 text-blue-500" />
						<h2 class="text-lg font-semibold">Find Similar Images</h2>
					{:else}
						<Search class="w-5 h-5 text-green-500" />
						<h2 class="text-lg font-semibold">Search Images by Text</h2>
					{/if}
				</div>
				<button
					onclick={closeModal}
					class="btn-pill btn-pill-ghost btn-pill-icon"
					aria-label="Close"
				>
					<X class="w-5 h-5" />
				</button>
			</div>

			<!-- Content -->
			<div class="p-6 space-y-4">
				<!-- Mode selector -->
				<div class="flex gap-2">
					<button
						onclick={() => (mode = 'multimodal')}
						class={`px-3 py-1.5 text-sm rounded-lg ${
							mode === 'multimodal'
								? 'bg-purple-500 text-white'
								: 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'
						}`}
					>
						<Sparkles class="w-4 h-4 inline mr-1" />
						Multimodal
					</button>
					<button
						onclick={() => (mode = 'image_similarity')}
						class={`px-3 py-1.5 text-sm rounded-lg ${
							mode === 'image_similarity'
								? 'bg-blue-500 text-white'
								: 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'
						}`}
					>
						<ImageIcon class="w-4 h-4 inline mr-1" />
						Image Similarity
					</button>
					<button
						onclick={() => (mode = 'text_to_image')}
						class={`px-3 py-1.5 text-sm rounded-lg ${
							mode === 'text_to_image'
								? 'bg-green-500 text-white'
								: 'bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600'
						}`}
					>
						<Search class="w-4 h-4 inline mr-1" />
						Text → Images
					</button>
				</div>

				<!-- Text query (for multimodal and text_to_image modes) -->
				{#if mode === 'multimodal' || mode === 'text_to_image'}
					<div>
						<label class="block text-sm font-medium mb-2">Text Query</label>
						<input
							type="text"
							bind:value={searchQuery}
							placeholder="Describe what you're looking for..."
							class="w-full px-3 py-2 border rounded-lg dark:bg-gray-700 dark:border-gray-600"
						/>
					</div>
				{/if}

				<!-- Image upload (for multimodal and image_similarity modes) -->
				{#if mode === 'multimodal' || mode === 'image_similarity'}
					<div>
						<label class="block text-sm font-medium mb-2">
							{mode === 'multimodal' ? 'Image (Optional)' : 'Image'}
						</label>

						{#if !selectedImage}
							<div
								class={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition ${
									dragOver
										? 'border-blue-500 bg-blue-50 dark:bg-blue-900/20'
										: 'border-gray-300 dark:border-gray-600 hover:border-gray-400 dark:hover:border-gray-500'
								}`}
								ondragover={handleDragOver}
								ondragleave={handleDragLeave}
								ondrop={handleDrop}
								onclick={() => fileInput?.click()}
								role="button"
								tabindex="0"
							>
								<Upload class="w-12 h-12 mx-auto mb-4 text-gray-400" />
								<p class="text-sm text-gray-600 dark:text-gray-400">
									Drop an image here or click to upload
								</p>
								<p class="text-xs text-gray-500 mt-2">PNG, JPG, WEBP (max 10MB)</p>
							</div>
							<input
								bind:this={fileInput}
								type="file"
								accept="image/*"
								class="hidden"
								onchange={handleImageSelect}
							/>
						{:else}
							<div class="relative">
								<img
									src={imagePreview}
									alt="Selected"
									class="w-full max-h-64 object-contain rounded-lg border dark:border-gray-600"
								/>
								<button
									onclick={removeImage}
									class="btn-pill btn-pill-danger btn-pill-icon absolute top-2 right-2"
									aria-label="Remove image"
								>
									<X class="w-4 h-4" />
								</button>
							</div>
						{/if}
					</div>
				{/if}

				<!-- Search options (for multimodal mode) -->
				{#if mode === 'multimodal'}
					<details class="border rounded-lg dark:border-gray-600">
						<summary class="px-4 py-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700">
							Advanced Options
						</summary>
						<div class="p-4 space-y-3 border-t dark:border-gray-600">
							<div>
								<label class="block text-sm mb-1">
									Semantic Weight: {(options.semantic_weight ?? 0.4).toFixed(1)}
								</label>
								<input
									type="range"
									min="0"
									max="1"
									step="0.1"
									bind:value={options.semantic_weight}
									class="w-full"
								/>
							</div>
							<div>
								<label class="block text-sm mb-1">
									Keyword Weight: {(options.keyword_weight ?? 0.3).toFixed(1)}
								</label>
								<input
									type="range"
									min="0"
									max="1"
									step="0.1"
									bind:value={options.keyword_weight}
									class="w-full"
								/>
							</div>
							<div>
								<label class="block text-sm mb-1">
									Image Weight: {(options.image_weight ?? 0.3).toFixed(1)}
								</label>
								<input
									type="range"
									min="0"
									max="1"
									step="0.1"
									bind:value={options.image_weight}
									class="w-full"
								/>
							</div>
							<div class="flex items-center gap-2">
								<input
									type="checkbox"
									id="rerank"
									bind:checked={options.rerank_enabled}
									class="rounded"
								/>
								<label for="rerank" class="text-sm">Enable re-ranking</label>
							</div>
						</div>
					</details>
				{/if}

				<!-- Error message -->
				{#if error}
					<div class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
						<p class="text-sm text-red-600 dark:text-red-400">{error}</p>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="sticky bottom-0 flex items-center justify-between p-4 border-t dark:border-gray-700 bg-white dark:bg-gray-800">
				<p class="text-xs text-gray-500">
					{#if mode === 'multimodal'}
						Cmd+Enter to search
					{:else}
						Press Enter to search
					{/if}
				</p>
				<div class="flex gap-2">
					<button
						onclick={closeModal}
						class="btn-pill btn-pill-ghost btn-pill-sm"
					>
						Cancel
					</button>
					<button
						onclick={performSearch}
						disabled={searching || (!selectedImage && !searchQuery)}
						class="btn-pill btn-pill-primary btn-pill-sm flex items-center gap-2"
					>
						{#if searching}
							<div class="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
							Searching...
						{:else}
							<Search class="w-4 h-4" />
							Search
						{/if}
					</button>
				</div>
			</div>
		</div>
	</div>
{/if}
