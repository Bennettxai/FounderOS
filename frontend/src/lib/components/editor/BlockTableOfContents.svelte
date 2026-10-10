<script lang="ts">
	import { editor } from '$lib/stores/editor';
</script>

<!-- Table of Contents - auto-generated from headings -->
<div class="toc-block p-4 rounded-lg bg-gray-50 dark:bg-[var(--bos-v2-layer-background-primary)] border border-gray-200 dark:border-[var(--bos-v2-layer-insideBorder-border)]">
	<div class="flex items-center gap-2 mb-3 text-gray-600 dark:text-gray-400">
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 10h16M4 14h16M4 18h16" />
		</svg>
		<span class="text-sm font-medium">Table of Contents</span>
	</div>
	<nav class="space-y-1">
		{#each $editor.blocks.filter(b => ['heading1', 'heading2', 'heading3'].includes(b.type)) as heading}
			<a
				href="#{heading.id}"
				onclick={(e) => {
					e.preventDefault();
					const el = document.querySelector(`[data-block-id="${heading.id}"]`);
					el?.scrollIntoView({ behavior: 'smooth', block: 'start' });
				}}
				class="block text-sm transition-colors {
					heading.type === 'heading1' ? 'text-gray-800 dark:text-gray-200 font-medium' :
					heading.type === 'heading2' ? 'pl-4 text-gray-700 dark:text-gray-300' :
					'pl-8 text-gray-600 dark:text-gray-400'
				} hover:text-blue-600 dark:hover:text-blue-400"
			>
				{heading.content || 'Untitled'}
			</a>
		{/each}
		{#if $editor.blocks.filter(b => ['heading1', 'heading2', 'heading3'].includes(b.type)).length === 0}
			<p class="text-sm text-gray-400 dark:text-gray-500 italic">No headings found. Add headings to see them here.</p>
		{/if}
	</nav>
</div>
