<script lang="ts">
	interface Props {
		icon?: string;
		label: string;
		selected?: boolean;
		onclick?: () => void;
	}

	let { icon, label, selected = false, onclick }: Props = $props();
</script>

<button
	type="button"
	{onclick}
	class="w-full flex items-center gap-4 p-4 rounded-xl border-2 transition-all duration-200 text-left h-[64px]
		{selected
			? 'border-gray-900 bg-gray-50'
			: 'border-gray-200 hover:border-gray-300 hover:shadow-sm'}"
>
	{#if icon}
		<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center flex-shrink-0">
			{@html icon}
		</div>
	{/if}
	<span class="flex-1 font-medium text-gray-900">{label}</span>
	{#if selected}
		<svg class="w-5 h-5 text-gray-900 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20">
			<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
		</svg>
	{/if}
</button>
