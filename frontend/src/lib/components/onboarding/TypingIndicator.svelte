<!--
  TypingIndicator.svelte
  Three bouncing dots indicating AI is thinking
-->
<script lang="ts">
	interface Props {
		class?: string;
	}

	let { class: className = '' }: Props = $props();
</script>

<div class="typing-indicator {className}">
	<span class="dot"></span>
	<span class="dot"></span>
	<span class="dot"></span>
</div>

<style>
	.typing-indicator {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 8px 12px;
	}

	.dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background-color: var(--muted-foreground, #6b7280);
		animation: bounce 1.4s ease-in-out infinite;
	}

	.dot:nth-child(1) {
		animation-delay: 0ms;
	}

	.dot:nth-child(2) {
		animation-delay: 160ms;
	}

	.dot:nth-child(3) {
		animation-delay: 320ms;
	}

	@keyframes bounce {
		0%, 60%, 100% {
			transform: translateY(0);
		}
		30% {
			transform: translateY(-6px);
		}
	}
</style>
