<script lang="ts">
	/**
	 * SelectCell - Single/Multi select cell
	 */
	import type { ColumnType, ColumnOptions, SelectChoice } from '$lib/api/tables/types';
	import { ChevronDown, X, Check } from 'lucide-svelte';

	interface Props {
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		type: ColumnType;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, options, editing, type, onChange, onBlur }: Props = $props();

	const isMulti = $derived(type === 'multi_select');
	const choices = $derived(options?.choices ?? []);

	// Parse value
	const selectedIds = $derived.by(() => {
		if (value == null) return [];
		if (Array.isArray(value)) return value.map((v) => (typeof v === 'object' ? v.id : v));
		if (typeof value === 'object' && 'id' in (value as object)) return [(value as SelectChoice).id];
		return [String(value)];
	});

	const selectedChoices = $derived(
		choices.filter((c) => selectedIds.includes(c.id))
	);

	let showDropdown = $state(false);

	function getChoiceColor(color: string): string {
		const colorMap: Record<string, string> = {
			red: 'bg-red-100 text-red-700',
			orange: 'bg-orange-100 text-orange-700',
			yellow: 'bg-yellow-100 text-yellow-700',
			green: 'bg-green-100 text-green-700',
			blue: 'bg-blue-100 text-blue-700',
			purple: 'bg-purple-100 text-purple-700',
			pink: 'bg-pink-100 text-pink-700',
			gray: 'bg-gray-100 text-gray-700'
		};
		return colorMap[color] || colorMap.gray;
	}

	function handleSelect(choice: SelectChoice) {
		if (isMulti) {
			const newIds = selectedIds.includes(choice.id)
				? selectedIds.filter((id) => id !== choice.id)
				: [...selectedIds, choice.id];
			onChange(choices.filter((c) => newIds.includes(c.id)));
		} else {
			onChange(choice);
			showDropdown = false;
			onBlur();
		}
	}

	function handleRemove(e: Event, choiceId: string) {
		e.stopPropagation();
		if (isMulti) {
			const newIds = selectedIds.filter((id) => id !== choiceId);
			onChange(choices.filter((c) => newIds.includes(c.id)));
		} else {
			onChange(null);
		}
	}

	function handleClickOutside() {
		if (showDropdown) {
			showDropdown = false;
			onBlur();
		}
	}

	$effect(() => {
		if (editing && !showDropdown) {
			showDropdown = true;
		}
	});
</script>

<svelte:window onclick={handleClickOutside} />

<div class="relative">
	<!-- Selected values -->
	<div
		class="flex flex-wrap items-center gap-1 min-h-[24px] cursor-pointer"
		onclick={(e) => {
			e.stopPropagation();
			showDropdown = !showDropdown;
		}}
		role="button"
		tabindex="0"
		onkeydown={(e) => {
			if (e.key === 'Enter' || e.key === ' ') {
				e.stopPropagation();
				showDropdown = !showDropdown;
			}
		}}
	>
		{#if selectedChoices.length > 0}
			{#each selectedChoices as choice}
				<span
					class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium {getChoiceColor(
						choice.color
					)}"
				>
					{choice.label}
					{#if isMulti}
						<button
							type="button"
							class="btn-pill btn-pill-ghost"
							onclick={(e) => handleRemove(e, choice.id)}
						>
							<X class="h-3 w-3" />
						</button>
					{/if}
				</span>
			{/each}
		{:else}
			<span class="text-sm text-gray-300">Select...</span>
		{/if}
	</div>

	<!-- Dropdown -->
	{#if showDropdown && choices.length > 0}
		<div
			class="dt2-dropdown absolute left-0 top-full mt-1 w-48 rounded-lg py-1"
			onclick={(e) => e.stopPropagation()}
			role="listbox"
		>
			{#each choices as choice}
				{@const isSelected = selectedIds.includes(choice.id)}
				<button
					type="button"
					class="dt2-dropdown__item flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
					onclick={() => handleSelect(choice)}
					role="option"
					aria-selected={isSelected}
				>
					{#if isMulti}
						<div
							class="flex h-4 w-4 items-center justify-center rounded border {isSelected
								? 'border-blue-600 bg-blue-600 text-white'
								: 'border-gray-300'}"
						>
							{#if isSelected}
								<Check class="h-3 w-3" />
							{/if}
						</div>
					{/if}
					<span
						class="rounded-full px-2 py-0.5 text-xs font-medium {getChoiceColor(choice.color)}"
					>
						{choice.label}
					</span>
					{#if !isMulti && isSelected}
						<Check class="ml-auto h-4 w-4 text-blue-600" />
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.dt2-dropdown {
		z-index: var(--bos-z-index-popover, 1001);
		background: var(--dbg);
		border: 1px solid var(--dbd);
		box-shadow: var(--shadow-lg);
	}

	.dt2-dropdown__item {
		color: var(--dt);
		border-radius: 6px;
	}

	.dt2-dropdown__item:hover {
		background: var(--dbg3);
	}
</style>
