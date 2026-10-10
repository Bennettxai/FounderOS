<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
		class?: string;
	}

	let {
		children,
		class: className = ''
	}: Props = $props();

	let containerRef: HTMLDivElement | undefined = $state(undefined);

	export function scrollToBottom(behavior: ScrollBehavior = 'smooth') {
		if (containerRef) {
			containerRef.scrollTo({
				top: containerRef.scrollHeight,
				behavior
			});
		}
	}
</script>

<div bind:this={containerRef} class="ai-conversation {className}">
	<div class="ai-conversation__inner">
		{@render children()}
	</div>
</div>

<style>
	.ai-conversation {
		flex: 1;
		overflow-y: auto;
		overflow-x: hidden;
	}

	.ai-conversation__inner {
		max-width: 48rem;
		margin: 0 auto;
		padding: 1.5rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	/* Scrollbar styling */
	.ai-conversation::-webkit-scrollbar {
		width: 0.5rem;
	}

	.ai-conversation::-webkit-scrollbar-track {
		background: transparent;
	}

	.ai-conversation::-webkit-scrollbar-thumb {
		background-color: var(--border);
		border-radius: 0.25rem;
	}

	.ai-conversation::-webkit-scrollbar-thumb:hover {
		background-color: var(--muted-foreground);
	}
</style>
