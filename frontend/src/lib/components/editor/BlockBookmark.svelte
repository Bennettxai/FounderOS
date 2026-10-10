<script lang="ts">
	import { editor, type EditorBlock } from '$lib/stores/editor';

	interface Props {
		block: EditorBlock;
	}

	let { block }: Props = $props();
</script>

<div class="bookmark-block rounded-lg border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)] overflow-hidden hover:border-gray-300 dark:hover:border-gray-500 transition-colors">
	{#if block.properties?.url}
		<a
			href={block.properties.url as string}
			target="_blank"
			rel="noopener noreferrer"
			class="flex items-stretch"
		>
			<div class="flex-1 p-4">
				<h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-1 line-clamp-1">
					{block.properties.title || block.properties.url}
				</h4>
				{#if block.properties.description}
					<p class="text-xs text-gray-500 dark:text-gray-400 line-clamp-2 mb-2">
						{block.properties.description}
					</p>
				{/if}
				<div class="flex items-center gap-2 text-xs text-gray-400 dark:text-gray-500">
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101" />
					</svg>
					<span class="truncate">{new URL(block.properties.url as string).hostname}</span>
				</div>
			</div>
			{#if block.properties.image}
				<div class="w-32 bg-gray-100 dark:bg-[var(--bos-v2-layer-background-secondary)]">
					<img src={block.properties.image as string} alt="" class="w-full h-full object-cover" />
				</div>
			{/if}
		</a>
	{:else}
		<div class="p-4">
			<div class="flex items-center gap-2 text-gray-500 dark:text-gray-400">
				<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101" />
				</svg>
				<input
					type="url"
					placeholder="Paste a link..."
					class="flex-1 bg-transparent border-none text-sm text-gray-700 dark:text-gray-200 placeholder:text-gray-400 focus:outline-none"
					onkeydown={(e) => {
						if (e.key === 'Enter') {
							const input = e.target as HTMLInputElement;
							if (input.value) {
								editor.updateBlock(block.id, block.content, { ...block.properties, url: input.value, title: input.value });
							}
						}
					}}
				/>
			</div>
		</div>
	{/if}
</div>
