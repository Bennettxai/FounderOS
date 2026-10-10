<script lang="ts">
	/**
	 * RatingCell - Interactive star rating display
	 */
	import { Star } from 'lucide-svelte';
	import type { ColumnOptions } from '$lib/api/tables/types';

	interface Props {
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, options, editing, onChange, onBlur }: Props = $props();

	// Get max rating from options or default to 5
	const maxRating = $derived(options?.max_value ?? 5);
	const currentRating = $derived(Number(value) || 0);

	let hoverRating = $state<number | null>(null);

	function handleClick(rating: number) {
		// Toggle off if clicking same rating
		const newRating = rating === currentRating ? 0 : rating;
		onChange(newRating);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onBlur();
		} else if (e.key === 'ArrowLeft') {
			e.preventDefault();
			const newRating = Math.max(0, currentRating - 1);
			onChange(newRating);
		} else if (e.key === 'ArrowRight') {
			e.preventDefault();
			const newRating = Math.min(maxRating, currentRating + 1);
			onChange(newRating);
		}
	}

	// Get filled state for each star
	function getStarFilled(index: number): boolean {
		const rating = hoverRating ?? currentRating;
		return index < rating;
	}
</script>

<div
	class="flex items-center gap-0.5"
	role="slider"
	aria-valuenow={currentRating}
	aria-valuemin={0}
	aria-valuemax={maxRating}
	aria-label="Rating"
	tabindex="0"
	onkeydown={handleKeydown}
	onmouseleave={() => (hoverRating = null)}
>
	{#each Array(maxRating) as _, index}
		{@const filled = getStarFilled(index)}
		<button
			type="button"
			class="btn-pill btn-pill-ghost p-0"
			onclick={() => handleClick(index + 1)}
			onmouseenter={() => (hoverRating = index + 1)}
		>
			<Star
				class="h-4 w-4 transition-colors {filled
					? 'fill-yellow-400 text-yellow-400'
					: 'fill-transparent text-gray-300 hover:text-yellow-300'}"
			/>
		</button>
	{/each}

	{#if currentRating > 0}
		<span class="ml-1 text-xs text-gray-500">{currentRating}/{maxRating}</span>
	{/if}
</div>
