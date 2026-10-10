<script lang="ts">
	import type { App } from '$lib/types/apps';

	interface Props {
		app: App;
		onOpen?: () => void;
		onEdit?: () => void;
		onMore?: () => void;
	}

	let { app, onOpen, onEdit, onMore }: Props = $props();

	const isDisabled = $derived(app.status === 'generating' || app.status === 'archived');
</script>

<div
	class="flex items-center gap-1 px-3 py-2 border-t border-gray-100 dark:border-gray-700/50 bg-gray-50/80 dark:bg-gray-800/50"
>
	<button
		onclick={onOpen}
		disabled={isDisabled}
		class="flex items-center gap-1.5 px-3 py-1.5 text-[13px] font-medium rounded-lg transition-all duration-150
			{isDisabled
			? 'text-gray-400 dark:text-gray-600 cursor-not-allowed'
			: 'text-gray-700 dark:text-gray-300 hover:bg-gray-200/80 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-white'}"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="1.5"
				d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
			/>
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="1.5"
				d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
			/>
		</svg>
		Open
	</button>

	<button
		onclick={onEdit}
		disabled={isDisabled}
		class="flex items-center gap-1.5 px-3 py-1.5 text-[13px] font-medium rounded-lg transition-all duration-150
			{isDisabled
			? 'text-gray-400 dark:text-gray-600 cursor-not-allowed'
			: 'text-gray-700 dark:text-gray-300 hover:bg-gray-200/80 dark:hover:bg-gray-700 hover:text-gray-900 dark:hover:text-white'}"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="1.5"
				d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
			/>
		</svg>
		Edit
	</button>

	<div class="flex-1"></div>

	<button
		onclick={onMore}
		class="btn-pill btn-pill-ghost flex items-center justify-center w-8 h-8"
	>
		<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path
				stroke-linecap="round"
				stroke-linejoin="round"
				stroke-width="2"
				d="M5 12h.01M12 12h.01M19 12h.01M6 12a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0zm7 0a1 1 0 11-2 0 1 1 0 012 0z"
			/>
		</svg>
	</button>
</div>
