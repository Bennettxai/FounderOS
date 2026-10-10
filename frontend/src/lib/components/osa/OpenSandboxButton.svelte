<script lang="ts">
	import { ExternalLink, Play, Square, Loader2, Monitor } from 'lucide-svelte';
	import type { SandboxContainer } from '$lib/types/sandbox';
	import { isSandboxActive } from '$lib/types/sandbox';

	interface Props {
		sandbox?: SandboxContainer;
		appId: string;
		variant?: 'primary' | 'secondary' | 'icon';
		onStart?: () => Promise<void>;
		onStop?: () => Promise<void>;
	}

	let { sandbox, appId, variant = 'primary', onStart, onStop }: Props = $props();

	let isLoading = $state(false);

	let canOpen = $derived(sandbox?.status === 'running' && sandbox?.url);
	let canStart = $derived(!sandbox || sandbox.status === 'stopped' || sandbox.status === 'failed');
	let canStop = $derived(isSandboxActive(sandbox?.status ?? 'pending'));

	async function handleStart() {
		if (!onStart) return;
		isLoading = true;
		try {
			await onStart();
		} finally {
			isLoading = false;
		}
	}

	async function handleStop() {
		if (!onStop) return;
		isLoading = true;
		try {
			await onStop();
		} finally {
			isLoading = false;
		}
	}

	function handleOpen() {
		if (sandbox?.url) {
			window.open(sandbox.url, '_blank', 'noopener,noreferrer');
		}
	}

	let buttonClasses = $derived({
		primary: 'px-4 py-2 text-sm font-medium rounded-lg transition-colors flex items-center gap-2',
		secondary: 'px-3 py-1.5 text-sm font-medium rounded-lg transition-colors flex items-center gap-2',
		icon: 'p-2 rounded-lg transition-colors'
	}[variant]);
</script>

<div class="flex items-center gap-2">
	{#if canOpen}
		<button
			onclick={handleOpen}
			class="{buttonClasses} text-white bg-green-600 hover:bg-green-700"
			title="Open in new tab"
		>
			{#if variant !== 'icon'}
				<ExternalLink class="w-4 h-4" />
				Open Preview
			{:else}
				<ExternalLink class="w-5 h-5" />
			{/if}
		</button>
	{/if}

	{#if canStart && onStart}
		<button
			onclick={handleStart}
			disabled={isLoading}
			class="{buttonClasses} text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
			title="Start sandbox"
		>
			{#if isLoading}
				<Loader2 class="w-4 h-4 animate-spin" />
				{#if variant !== 'icon'}Starting...{/if}
			{:else}
				<Play class="w-4 h-4" />
				{#if variant !== 'icon'}Start Sandbox{/if}
			{/if}
		</button>
	{/if}

	{#if canStop && onStop}
		<button
			onclick={handleStop}
			disabled={isLoading}
			class="{buttonClasses} text-white bg-red-600 hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
			title="Stop sandbox"
		>
			{#if isLoading}
				<Loader2 class="w-4 h-4 animate-spin" />
				{#if variant !== 'icon'}Stopping...{/if}
			{:else}
				<Square class="w-4 h-4" />
				{#if variant !== 'icon'}Stop{/if}
			{/if}
		</button>
	{/if}

	{#if sandbox?.status === 'deploying'}
		<span class="{buttonClasses} text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-800 cursor-wait">
			<Loader2 class="w-4 h-4 animate-spin" />
			{#if variant !== 'icon'}Deploying...{/if}
		</span>
	{/if}

	{#if !sandbox && !onStart}
		<span class="{buttonClasses} text-gray-500 bg-gray-100 dark:bg-gray-800 cursor-not-allowed">
			<Monitor class="w-4 h-4" />
			{#if variant !== 'icon'}No Sandbox{/if}
		</span>
	{/if}
</div>
