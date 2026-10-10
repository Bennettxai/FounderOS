<script lang="ts">
	import type { HTMLInputAttributes } from 'svelte/elements';

	interface Props {
		id: string;
		label: string;
		type?: 'text' | 'email';
		value: string;
		placeholder?: string;
		error?: string;
		autocomplete?: HTMLInputAttributes['autocomplete'];
		required?: boolean;
		maxlength?: number;
	}

	let {
		id,
		label,
		type = 'text',
		value = $bindable(),
		placeholder = '',
		error = '',
		autocomplete = '',
		required = false,
		maxlength = undefined
	}: Props = $props();
</script>

<div class="space-y-1.5">
	<label for={id} class="block text-sm font-medium text-gray-700">
		{label}
	</label>
	<input
		{id}
		name={id}
		{type}
		bind:value
		{placeholder}
		{autocomplete}
		{required}
		maxlength={maxlength}
		class="input input-square w-full {error ? 'border-red-500 focus:ring-red-500' : ''}"
	/>
	{#if error}
		<p class="text-sm text-red-600 flex items-center gap-1">
			<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			{error}
		</p>
	{/if}
</div>
