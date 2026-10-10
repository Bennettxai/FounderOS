<script lang="ts">
	import { fly, fade } from 'svelte/transition';
	import { onMount } from 'svelte';
	import { useSession } from '$lib/auth-client';

	const session = useSession();

	interface Props {
		onQuickAction?: (prompt: string) => void;
	}

	let { onQuickAction }: Props = $props();

	// Animated subtitle phrases
	const subtitles = [
		'Draft a business proposal',
		'Analyze your project data',
		'Plan your weekly tasks',
		'Debug your code',
		'Brainstorm new ideas',
		'Write marketing copy',
		'Create a project timeline',
		'Summarize meeting notes'
	];

	let currentSubtitleIndex = $state(0);
	let isVisible = $state(true);

	onMount(() => {
		const interval = setInterval(() => {
			isVisible = false;
			setTimeout(() => {
				currentSubtitleIndex = (currentSubtitleIndex + 1) % subtitles.length;
				isVisible = true;
			}, 300);
		}, 3000);
		return () => clearInterval(interval);
	});

	// Pill-shaped quick actions
	const quickActions = [
		{ label: 'Write a proposal', prompt: 'Help me write a business proposal for' },
		{ label: 'Analyze data', prompt: 'Analyze this data and provide insights:' },
		{ label: 'Plan my week', prompt: 'Help me plan and organize my tasks for this week' },
		{ label: 'Debug code', prompt: 'Help me debug this code:' },
		{ label: 'Brainstorm ideas', prompt: 'Help me brainstorm ideas for' }
	];
</script>

<!-- Centered greeting and suggestions - no input here, input is at bottom -->
<div class="flex-1 flex flex-col items-center justify-center px-6" in:fade={{ duration: 300 }}>
	<div class="max-w-2xl w-full text-center">
		<!-- Greeting -->
		<div class="mb-8" in:fly={{ y: -20, duration: 400, delay: 100 }}>
			<h1 class="text-3xl font-semibold text-gray-900 mb-3">
				Good {new Date().getHours() < 12 ? 'morning' : new Date().getHours() < 17 ? 'afternoon' : 'evening'}, {$session.data?.user?.name?.split(' ')[0] || 'there'}!
			</h1>
			<!-- Animated Subtitle -->
			<div class="h-6">
				{#if isVisible}
					<p 
						class="text-gray-400 transition-opacity duration-300"
						in:fade={{ duration: 300 }}
						out:fade={{ duration: 200 }}
					>
						{subtitles[currentSubtitleIndex]}
					</p>
				{/if}
			</div>
		</div>

		<!-- Pill-shaped Quick Actions -->
		<div class="flex flex-wrap justify-center gap-2" in:fly={{ y: 20, duration: 400, delay: 200 }}>
			{#each quickActions as action}
				<button
					onclick={() => onQuickAction?.(action.prompt)}
					class="btn-pill btn-pill-ghost btn-pill-sm"
				>
					{action.label}
				</button>
			{/each}
		</div>
	</div>
</div>
