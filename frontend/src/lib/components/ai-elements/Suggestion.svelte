<script lang="ts">
	import { fly } from 'svelte/transition';

	interface Props {
		suggestions: string[];
		onSelect?: (suggestion: string) => void;
		class?: string;
	}

	let {
		suggestions,
		onSelect,
		class: className = ''
	}: Props = $props();
</script>

<div class="ai-suggestions {className}" in:fly={{ y: 10, duration: 200 }}>
	{#each suggestions as suggestion, i}
		<button
			type="button"
			onclick={() => onSelect?.(suggestion)}
			class="ai-suggestion"
			style="animation-delay: {i * 50}ms"
		>
			{suggestion}
		</button>
	{/each}
</div>

<style>
	.ai-suggestions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		justify-content: center;
	}

	.ai-suggestion {
		padding: 0.625rem 1rem;
		font-size: 0.875rem;
		font-weight: 400;
		color: var(--foreground);
		background-color: var(--card);
		border: 1px solid var(--border);
		border-radius: 9999px;
		cursor: pointer;
		transition: all 0.2s ease;
		animation: fade-up 0.3s ease forwards;
		opacity: 0;
	}

	.ai-suggestion:hover {
		background-color: var(--accent);
		border-color: var(--border);
	}

	@keyframes fade-up {
		from {
			opacity: 0;
			transform: translateY(0.5rem);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	/* Dark mode */
	:global(.dark) .ai-suggestion {
		background-color: var(--card);
		border-color: var(--border);
		color: var(--foreground);
	}

	:global(.dark) .ai-suggestion:hover {
		background-color: var(--accent);
	}
</style>
