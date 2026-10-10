<script lang="ts">
	import type { SearchItem } from './spotlightSearch.ts';

	interface Props {
		item: SearchItem;
		selected: boolean;
		onSelect: (item: SearchItem) => void;
		onHover: () => void;
	}

	let { item, selected, onSelect, onHover }: Props = $props();
</script>

<button
	class="search-item"
	class:selected
	onclick={() => onSelect(item)}
	onmouseenter={onHover}
	aria-label="Open {item.name}"
>
	<div class="item-icon" style="background: {item.color}15;">
		<svg viewBox="0 0 24 24" fill="none" stroke={item.color} stroke-width="1.5">
			<path d={item.icon} />
		</svg>
	</div>
	<div class="item-info">
		<span class="item-name">{item.name}</span>
		<span class="item-desc">{item.description}</span>
	</div>
</button>

<style>
	.search-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 10px 12px;
		border: none;
		background: none;
		border-radius: 12px;
		cursor: pointer;
		text-align: left;
		transition: all 0.15s;
	}

	.search-item:hover,
	.search-item.selected {
		background: #f5f5f5;
	}

	.search-item:active {
		transform: scale(0.98);
	}

	.item-icon {
		width: 40px;
		height: 40px;
		border-radius: 12px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		transition: transform 0.15s;
	}

	.search-item:hover .item-icon {
		transform: scale(1.05);
	}

	.item-icon svg {
		width: 20px;
		height: 20px;
	}

	.item-info {
		flex: 1;
		min-width: 0;
	}

	.item-name {
		display: block;
		font-size: 14px;
		font-weight: 500;
		color: #111;
	}

	.item-desc {
		display: block;
		font-size: 12px;
		color: #666;
		margin-top: 2px;
	}

	/* Dark mode */
	:global(.dark) .search-item:hover,
	:global(.dark) .search-item.selected {
		background: #2c2c2e;
	}

	:global(.dark) .item-name {
		color: #f5f5f7;
	}

	:global(.dark) .item-desc {
		color: #6e6e73;
	}
</style>
