<script lang="ts">
	/**
	 * CheckboxCell - Boolean checkbox cell
	 */
	import type { ColumnType, ColumnOptions } from '$lib/api/tables/types';
	import { Check } from 'lucide-svelte';

	interface Props {
		value: unknown;
		options?: ColumnOptions;
		editing: boolean;
		type: ColumnType;
		onChange: (value: unknown) => void;
		onBlur: () => void;
	}

	let { value, options, editing, type, onChange, onBlur }: Props = $props();

	const checked = $derived(Boolean(value));

	function handleToggle(e: Event) {
		e.stopPropagation();
		onChange(!checked);
	}
</script>

<div class="flex items-center justify-center">
	<button
		type="button"
		class="flex h-5 w-5 items-center justify-center rounded border-2 transition-colors {checked
			? 'border-blue-600 bg-blue-600 text-white'
			: 'border-gray-300 bg-white hover:border-gray-400'}"
		onclick={handleToggle}
	>
		{#if checked}
			<Check class="h-3 w-3" />
		{/if}
	</button>
</div>
