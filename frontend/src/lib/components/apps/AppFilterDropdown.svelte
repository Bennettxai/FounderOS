<script lang="ts">
	import type { AppStatus } from '$lib/types/apps';

	type FilterValue = AppStatus | 'all';

	interface Props {
		value: FilterValue;
		onChange: (value: FilterValue) => void;
	}

	let { value = 'all', onChange }: Props = $props();

	let isOpen = $state(false);

	const options: { value: FilterValue; label: string }[] = [
		{ value: 'all', label: 'All Apps' },
		{ value: 'active', label: 'Active' },
		{ value: 'draft', label: 'Draft' },
		{ value: 'generating', label: 'Generating' },
		{ value: 'archived', label: 'Archived' }
	];

	const selectedLabel = $derived(options.find((o) => o.value === value)?.label || 'All Apps');

	function handleSelect(newValue: FilterValue) {
		onChange(newValue);
		isOpen = false;
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.filter-dropdown')) {
			isOpen = false;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="relative filter-dropdown">
	<!-- Trigger -->
	<button
		onclick={() => (isOpen = !isOpen)}
		aria-label="Filter apps by status"
		aria-expanded={isOpen}
		aria-haspopup="listbox"
		class="inline-flex items-center gap-2 h-10 px-4 border rounded-xl text-sm font-medium transition-all duration-150
			{isOpen ? 'shadow-sm' : ''}"
	style="background: var(--dbg); border-color: var(--dbd); color: var(--dt2);"
	>
		{selectedLabel}
		<svg
			class="w-4 h-4 transition-transform duration-150 {isOpen ? 'rotate-180' : ''}"
		style="color: var(--dt2);"
			fill="none"
			stroke="currentColor"
			viewBox="0 0 24 24"
			aria-hidden="true"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	<!-- Dropdown -->
	{#if isOpen}
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="absolute right-0 mt-2 w-44 rounded-xl overflow-hidden z-50 animate-in fade-in-0 zoom-in-95 duration-150"
			style="background: var(--dbg); border: 1px solid var(--dbd); box-shadow: 0 8px 24px rgba(0,0,0,0.12);"
			role="listbox"
			aria-label="App status filter options"
		>
			{#each options as option}
				<button
					onclick={() => handleSelect(option.value)}
					role="option"
					aria-selected={value === option.value}
					class="w-full flex items-center gap-2 px-4 py-2.5 text-sm text-left transition-colors
						{value === option.value ? 'af-item--active' : 'af-item--idle'}"
				>
					{#if value === option.value}
						<svg class="w-4 h-4" style="color: var(--bos-primary-color);" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
						</svg>
					{:else}
						<span class="w-4"></span>
					{/if}
					{option.label}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	:global(.af-item--active) {
		background: var(--dbg2);
		color: var(--dt);
		font-weight: 500;
	}
	:global(.af-item--idle) {
		color: var(--dt2);
	}
	:global(.af-item--idle:hover) {
		background: var(--dbg2);
	}
</style>
