<!--
  PurpleOrb.svelte
  Animated gradient orb with pulse and shimmer effects
  Used as visual branding element during onboarding
-->
<script lang="ts">
	interface Props {
		size?: 'sm' | 'md' | 'lg';
		isPulsing?: boolean;
		isThinking?: boolean;
		class?: string;
	}

	let { size = 'md', isPulsing = true, isThinking = false, class: className = '' }: Props = $props();

	const sizeMap = {
		sm: 'w-16 h-16',
		md: 'w-24 h-24',
		lg: 'w-32 h-32'
	};
</script>

<div
	class="orb-container {sizeMap[size]} {className}"
	class:is-pulsing={isPulsing}
	class:is-thinking={isThinking}
>
	<div class="orb">
		<div class="orb-shimmer"></div>
	</div>
</div>

<style>
	.orb-container {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.orb {
		width: 100%;
		height: 100%;
		border-radius: 50%;
		background: linear-gradient(
			135deg,
			var(--orb-gradient-1, #e0e7ff) 0%,
			var(--orb-gradient-2, #a5b4fc) 25%,
			var(--orb-gradient-3, #818cf8) 50%,
			var(--orb-gradient-4, #6366f1) 75%,
			var(--orb-gradient-5, #4f46e5) 100%
		);
		box-shadow:
			0 0 40px var(--orb-glow, rgba(99, 102, 241, 0.4)),
			inset 0 0 20px var(--orb-inner-glow, rgba(255, 255, 255, 0.3));
		position: relative;
		overflow: hidden;
	}

	.orb-container.is-pulsing .orb {
		animation: pulse-orb 2s ease-in-out infinite;
	}

	.orb-shimmer {
		position: absolute;
		inset: 0;
		background: linear-gradient(
			90deg,
			transparent,
			rgba(255, 255, 255, 0.4),
			transparent
		);
		background-size: 1000px 100%;
		animation: shimmer 3s linear infinite;
	}

	@keyframes pulse-orb {
		0%, 100% {
			transform: scale(1);
			filter: brightness(1);
		}
		50% {
			transform: scale(1.05);
			filter: brightness(1.2);
		}
	}

	@keyframes shimmer {
		0% {
			background-position: -1000px 0;
		}
		100% {
			background-position: 1000px 0;
		}
	}

	/* Thinking state - more intense animation */
	.orb-container.is-thinking .orb {
		animation: thinking-pulse 1s ease-in-out infinite;
		box-shadow:
			0 0 60px var(--orb-glow, rgba(99, 102, 241, 0.6)),
			0 0 100px var(--orb-glow, rgba(99, 102, 241, 0.3)),
			inset 0 0 30px var(--orb-inner-glow, rgba(255, 255, 255, 0.4));
	}

	.orb-container.is-thinking .orb-shimmer {
		animation: shimmer 1.5s linear infinite;
	}

	@keyframes thinking-pulse {
		0%, 100% {
			transform: scale(1);
			filter: brightness(1.1);
		}
		50% {
			transform: scale(1.08);
			filter: brightness(1.3);
		}
	}

	/* Dark mode adjustments */
	:global(.dark) .orb {
		--orb-gradient-1: #312e81;
		--orb-gradient-2: #3730a3;
		--orb-gradient-3: #4338ca;
		--orb-gradient-4: #4f46e5;
		--orb-gradient-5: #6366f1;
		--orb-glow: rgba(99, 102, 241, 0.6);
	}
</style>
