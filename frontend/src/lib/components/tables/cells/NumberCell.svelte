<script lang="ts">
	/**
	 * NumberCell - Number input cell (also handles currency, percent, rating, duration)
	 */
	import type { ColumnType, ColumnOptions } from '$lib/api/tables/types';
	import { Star } from 'lucide-svelte';

	interface Props {
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		type: ColumnType;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, options, editing, type, onChange, onBlur }: Props = $props();

	let inputRef = $state<HTMLInputElement | null>(null);

	const numValue = $derived(value != null ? Number(value) : null);
	const isCurrency = $derived(type === 'currency');
	const isPercent = $derived(type === 'percent');
	const isRating = $derived(type === 'rating');
	const isDuration = $derived(type === 'duration');

	const precision = $derived(options?.precision ?? 2);
	const currencyCode = $derived(options?.currency_code ?? 'USD');
	const ratingMax = $derived(options?.rating_max ?? 5);

	$effect(() => {
		if (editing && inputRef) {
			inputRef.focus();
			inputRef.select();
		}
	});

	function formatNumber(val: number | null): string {
		if (val == null) return '';

		if (isCurrency) {
			return new Intl.NumberFormat('en-US', {
				style: 'currency',
				currency: currencyCode
			}).format(val);
		}

		if (isPercent) {
			return `${val.toFixed(precision)}%`;
		}

		if (isDuration) {
			const hours = Math.floor(val / 60);
			const mins = val % 60;
			return `${hours}:${mins.toString().padStart(2, '0')}`;
		}

		return val.toFixed(precision);
	}

	function handleChange(e: Event) {
		const target = e.target as HTMLInputElement;
		const newValue = target.value ? parseFloat(target.value) : null;
		onChange(newValue);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			onBlur();
		}
	}

	function handleRatingClick(rating: number) {
		onChange(numValue === rating ? rating - 1 : rating);
	}
</script>

{#if isRating && !editing}
	<!-- Rating display/edit mode -->
	<div class="flex items-center gap-0.5">
		{#each Array(ratingMax) as _, i}
			<button
				type="button"
				class="btn-pill btn-pill-ghost"
				onclick={(e) => {
					e.stopPropagation();
					handleRatingClick(i + 1);
				}}
			>
				<Star
					class="h-4 w-4 {numValue != null && i < numValue ? 'fill-current' : ''}"
				/>
			</button>
		{/each}
	</div>
{:else if editing}
	<input
		bind:this={inputRef}
		type="number"
		value={numValue ?? ''}
		step={isPercent || isCurrency ? Math.pow(10, -precision) : 1}
		oninput={handleChange}
		onblur={onBlur}
		onkeydown={handleKeydown}
		class="h-full w-full bg-transparent text-sm text-right outline-none"
	/>
{:else}
	<div class="text-sm text-right">
		{#if numValue != null}
			<span class="text-gray-900">{formatNumber(numValue)}</span>
		{:else}
			<span class="text-gray-300">-</span>
		{/if}
	</div>
{/if}
