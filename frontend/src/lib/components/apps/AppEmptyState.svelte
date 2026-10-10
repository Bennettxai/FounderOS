<script lang="ts">
	import { APP_TEMPLATES } from '$lib/types/apps';
	import * as icons from 'lucide-svelte';

	interface Props {
		onCreateApp?: () => void;
		onSelectTemplate?: (templateId: string) => void;
	}

	let { onCreateApp, onSelectTemplate }: Props = $props();

	// Get icon component by name
	function getIcon(name: string) {
		return (icons as Record<string, any>)[name] || icons.Layers;
	}
</script>

<div class="flex flex-col items-center justify-center py-16 px-4">
	<!-- Illustration -->
	<div class="relative mb-8">
		<div
			class="w-24 h-24 rounded-2xl bg-gradient-to-br from-gray-100 to-gray-200 dark:from-gray-700 dark:to-gray-800 flex items-center justify-center"
		>
			<svg
				class="w-12 h-12 text-gray-400 dark:text-gray-500"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					stroke-width="1.25"
					d="M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
				/>
			</svg>
		</div>
		<!-- Decorative dots -->
		<div
			class="absolute -top-2 -right-2 w-4 h-4 rounded-full bg-blue-500/20 dark:bg-blue-400/20"
		></div>
		<div
			class="absolute -bottom-1 -left-3 w-3 h-3 rounded-full bg-purple-500/20 dark:bg-purple-400/20"
		></div>
	</div>

	<!-- Text -->
	<h2 class="text-xl font-semibold text-gray-900 dark:text-white mb-2">No modules yet</h2>
	<p class="text-gray-600 dark:text-gray-400 text-center max-w-sm mb-6">
		Create your first module to get started, or ask OSA to build one in BUILD mode.
	</p>

	<!-- Create Button -->
	<button
		onclick={onCreateApp}
		class="btn-pill btn-pill-primary btn-pill-sm inline-flex items-center gap-2"
	>
		<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
		</svg>
		Create Module
	</button>

	<!-- Templates Section -->
	<div class="mt-12 w-full max-w-2xl">
		<h3
			class="text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-4 text-center"
		>
			Start from a template
		</h3>

		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			{#each APP_TEMPLATES as template}
				{@const Icon = getIcon(template.icon)}
				<button
					onclick={() => onSelectTemplate?.(template.id)}
					class="group flex flex-col items-center p-4 rounded-xl border-2 border-dashed border-gray-200 dark:border-gray-700
						transition-all duration-150 hover:border-gray-400 dark:hover:border-gray-500 hover:bg-gray-50 dark:hover:bg-gray-800/50"
				>
					<div
						class="w-10 h-10 rounded-xl bg-gray-100 dark:bg-gray-700 flex items-center justify-center mb-2.5
						text-gray-500 dark:text-gray-400 group-hover:text-gray-700 dark:group-hover:text-gray-300 transition-colors"
					>
						<svelte:component this={Icon} class="w-5 h-5" strokeWidth={1.75} />
					</div>
					<span class="text-sm font-medium text-gray-900 dark:text-white mb-0.5">
						{template.name}
					</span>
					<span class="text-xs text-gray-500 dark:text-gray-400 text-center">
						{template.description}
					</span>
				</button>
			{/each}
		</div>
	</div>
</div>
