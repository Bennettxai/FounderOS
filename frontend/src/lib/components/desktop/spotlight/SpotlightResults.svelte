<script lang="ts">
	import SpotlightResultItem from './SpotlightResultItem.svelte';
	import type { SearchItem } from './spotlightSearch.ts';

	interface Props {
		items: SearchItem[];
		selectedIndex: number;
		onSelect: (item: SearchItem) => void;
		onHoverIndex: (index: number) => void;
	}

	let { items, selectedIndex, onSelect, onHoverIndex }: Props = $props();
</script>

{#if items.length > 0}
	<div class="search-results" role="listbox" aria-label="Search results">
		{#each items as item, index (item.id)}
			<SpotlightResultItem
				{item}
				selected={index === selectedIndex}
				onSelect={onSelect}
				onHover={() => onHoverIndex(index)}
			/>
		{/each}
	</div>
{/if}

<style>
	.search-results {
		background: white;
		border-radius: 16px;
		margin-top: 8px;
		padding: 8px;
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
		border: 1px solid rgba(0, 0, 0, 0.06);
		max-height: 320px;
		overflow-y: auto;
		scrollbar-width: none;
		-ms-overflow-style: none;
	}

	.search-results::-webkit-scrollbar {
		display: none;
	}

	/* Dark mode */
	:global(.dark) .search-results {
		background: #1c1c1e;
		border-color: rgba(255, 255, 255, 0.12);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.4);
	}
</style>
