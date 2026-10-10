<script lang="ts">
	interface Props {
		value: string;
		onInput: (value: string) => void;
		placeholder?: string;
	}

	let { value = '', onInput, placeholder = 'Search apps...' }: Props = $props();

	let inputRef: HTMLInputElement;
	let isFocused = $state(false);
</script>

<div
	class="relative flex items-center w-60 transition-all duration-200
		{isFocused ? 'w-72' : ''}"
>
	<!-- Search Icon -->
	<svg
		class="absolute left-3 w-4 h-4 text-gray-400 dark:text-gray-500 pointer-events-none transition-colors
			{isFocused ? 'text-gray-600 dark:text-gray-300' : ''}"
		fill="none"
		stroke="currentColor"
		viewBox="0 0 24 24"
	>
		<path
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
			d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
		/>
	</svg>

	<!-- Input -->
	<input
		bind:this={inputRef}
		type="text"
		{value}
		oninput={(e) => onInput(e.currentTarget.value)}
		onfocus={() => (isFocused = true)}
		onblur={() => (isFocused = false)}
		{placeholder}
		aria-label="Search apps"
		class="w-full h-10 pl-10 pr-4 bg-gray-100 dark:bg-gray-800 border border-transparent
			rounded-xl text-sm text-gray-900 dark:text-gray-100 placeholder-gray-500 dark:placeholder-gray-500
			transition-all duration-200
			focus:outline-none focus:border-gray-300 dark:focus:border-gray-600 focus:bg-white dark:focus:bg-gray-800 focus:shadow-sm"
	/>

	<!-- Clear Button -->
	{#if value}
		<button
			onclick={() => {
				onInput('');
				inputRef?.focus();
			}}
			aria-label="Clear search"
			class="absolute right-3 w-5 h-5 flex items-center justify-center rounded-full
				bg-gray-300 dark:bg-gray-600 text-gray-600 dark:text-gray-300
				hover:bg-gray-400 dark:hover:bg-gray-500 transition-colors"
		>
			<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12" />
			</svg>
		</button>
	{/if}
</div>
