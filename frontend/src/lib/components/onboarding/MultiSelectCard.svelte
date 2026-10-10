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
	class="w-full flex flex-col items-center justify-center gap-2 p-4 rounded-xl border-2 transition-all duration-200 text-center h-[120px]
		{selected
			? 'border-gray-900 bg-gray-50'
			: 'border-dashed border-gray-300 hover:border-gray-400 hover:bg-gray-50'}"
>
	{#if icon}
		<div class="w-10 h-10 rounded-lg bg-gray-100 flex items-center justify-center">
			{@html icon}
		</div>
	{/if}
	<span class="text-sm font-medium text-gray-900">{label}</span>
	<div class="w-5 h-5 rounded border-2 flex items-center justify-center transition-all
		{selected ? 'border-gray-900 bg-gray-900' : 'border-gray-300'}">
		{#if selected}
			<svg class="w-3 h-3 text-white" fill="currentColor" viewBox="0 0 20 20">
				<path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
			</svg>
		{/if}
	</div>
</button>
