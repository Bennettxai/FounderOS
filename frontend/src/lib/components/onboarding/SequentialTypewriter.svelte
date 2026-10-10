<!--
  SequentialTypewriter.svelte
  Sequentially types multiple lines of text with typewriter effect
-->
<script lang="ts">
	import { onMount } from 'svelte';

	interface Props {
		lines: string[];
		speed?: number;
		lineDelay?: number;
		showCursor?: boolean;
		onComplete?: () => void;
		class?: string;
	}

	let {
		lines,
		speed = 30,
		lineDelay = 300,
		showCursor = true,
		onComplete,
		class: className = ''
	}: Props = $props();

	let displayedLines = $state<string[]>([]);
	let currentLineIndex = $state(0);
	let currentCharIndex = $state(0);
	let isComplete = $state(false);

	onMount(() => {
		if (lines.length === 0) {
			isComplete = true;
			onComplete?.();
			return;
		}

		displayedLines = [''];

		const typeNextChar = () => {
			if (currentLineIndex >= lines.length) {
				isComplete = true;
				onComplete?.();
				return;
			}

			const currentLine = lines[currentLineIndex];

			if (currentCharIndex < currentLine.length) {
				displayedLines[currentLineIndex] = currentLine.slice(0, currentCharIndex + 1);
				currentCharIndex++;
				setTimeout(typeNextChar, speed);
			} else {
				// Move to next line
				currentLineIndex++;
				currentCharIndex = 0;
				if (currentLineIndex < lines.length) {
					displayedLines = [...displayedLines, ''];
					setTimeout(typeNextChar, lineDelay);
				} else {
					isComplete = true;
					onComplete?.();
				}
			}
		};

		setTimeout(typeNextChar, 100);
	});
</script>

<div class="sequential-typewriter {className}">
	{#each displayedLines as line, i}
		<p class="line">
			{line}
			{#if showCursor && i === currentLineIndex && !isComplete}
				<span class="cursor">|</span>
			{/if}
		</p>
	{/each}
</div>

<style>
	.sequential-typewriter {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.line {
		margin: 0;
		min-height: 1.5em;
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
