<!--
  TypewriterText.svelte
  Single line character-by-character typing animation
-->
<script lang="ts">
	import { onMount } from 'svelte';

	interface Props {
		text: string;
		speed?: number;
		delay?: number;
		showCursor?: boolean;
		onComplete?: () => void;
		class?: string;
	}

	let {
		text,
		speed = 30,
		delay = 0,
		showCursor = true,
		onComplete,
		class: className = ''
	}: Props = $props();

	let displayedText = $state('');
	let isComplete = $state(false);
	let hasStarted = $state(false);

	onMount(() => {
		const delayTimer = setTimeout(() => {
			hasStarted = true;
			let index = 0;
			const interval = setInterval(() => {
				if (index < text.length) {
					displayedText = text.slice(0, index + 1);
					index++;
				} else {
					clearInterval(interval);
					isComplete = true;
					onComplete?.();
				}
			}, speed);

			return () => clearInterval(interval);
		}, delay);

		return () => clearTimeout(delayTimer);
	});
</script>

<span class="typewriter {className}">
	{displayedText}
	{#if showCursor && !isComplete}
		<span class="cursor">|</span>
	{/if}
</span>

<style>
	.typewriter {
		display: inline;
	}

	.cursor {
		animation: blink 1s step-end infinite;
		color: var(--foreground, currentColor);
	}

	@keyframes blink {
		0%, 49% {
			opacity: 1;
		}
		50%, 100% {
			opacity: 0;
		}
	}
</style>
