<script lang="ts">
	/**
	 * RatingCell - Star rating display and edit
	 */

	interface Props {
		value: number | null | undefined;
		max?: number;
		editable?: boolean;
		icon?: 'star' | 'heart';
		onchange?: (value: number) => void;
	}

	let {
		value = 0,
		max = 5,
		editable = false,
		icon = 'star',
		onchange
	}: Props = $props();

	let hoverValue = $state<number | null>(null);

	const displayValue = $derived(hoverValue ?? value ?? 0);

	function handleClick(rating: number) {
		if (editable) {
			onchange?.(rating);
		}
	}

	function handleMouseEnter(rating: number) {
		if (editable) {
			hoverValue = rating;
		}
	}

	function handleMouseLeave() {
		hoverValue = null;
	}
</script>

<div
	class="tpl-rating"
	class:tpl-rating-editable={editable}
	onmouseleave={handleMouseLeave}
	role="slider"
	tabindex={editable ? 0 : -1}
	aria-valuemin={0}
	aria-valuemax={max}
	aria-valuenow={value ?? 0}
>
	{#each Array(max) as _, i}
		{@const filled = i < displayValue}
		<button
			type="button"
			class="tpl-rating-item"
			class:tpl-rating-filled={filled}
			onclick={() => handleClick(i + 1)}
			onmouseenter={() => handleMouseEnter(i + 1)}
			disabled={!editable}
		>
			{#if icon === 'star'}
				<svg viewBox="0 0 20 20" fill={filled ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="1.5">
					<path d="M9.049 2.927c.3-.921 1.603-.921 1.902 0l1.07 3.292a1 1 0 00.95.69h3.462c.969 0 1.371 1.24.588 1.81l-2.8 2.034a1 1 0 00-.364 1.118l1.07 3.292c.3.921-.755 1.688-1.54 1.118l-2.8-2.034a1 1 0 00-1.175 0l-2.8 2.034c-.784.57-1.838-.197-1.539-1.118l1.07-3.292a1 1 0 00-.364-1.118L2.98 8.72c-.783-.57-.38-1.81.588-1.81h3.461a1 1 0 00.951-.69l1.07-3.292z" />
				</svg>
			{:else}
				<svg viewBox="0 0 20 20" fill={filled ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="1.5">
					<path d="M3.172 5.172a4 4 0 015.656 0L10 6.343l1.172-1.171a4 4 0 115.656 5.656L10 17.657l-6.828-6.829a4 4 0 010-5.656z" />
				</svg>
			{/if}
		</button>
	{/each}
</div>

<style>
	.tpl-rating {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-1);
		padding: var(--tpl-space-2) var(--tpl-space-3);
	}

	.tpl-rating-item {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		background: transparent;
		border: none;
		cursor: default;
		color: var(--tpl-border-default);
		transition: all var(--tpl-transition-fast);
	}

	.tpl-rating-editable .tpl-rating-item {
		cursor: pointer;
	}

	.tpl-rating-editable .tpl-rating-item:hover {
		transform: scale(1.15);
	}

	.tpl-rating-item svg {
		width: 18px;
		height: 18px;
	}

	.tpl-rating-filled {
		color: var(--tpl-status-warning);
	}

	.tpl-rating:focus-visible {
		outline: none;
		border-radius: var(--tpl-radius-sm);
		box-shadow: var(--tpl-shadow-focus);
	}
</style>
