<script lang="ts">
	interface Props {
		variant?: 'dots' | 'spinner' | 'shimmer';
		size?: 'sm' | 'md' | 'lg';
		class?: string;
	}

	let {
		variant = 'dots',
		size = 'md',
		class: className = ''
	}: Props = $props();

	const sizeMap = {
		sm: { dot: 'w-1.5 h-1.5', spinner: 'w-4 h-4', shimmer: 'h-4' },
		md: { dot: 'w-2 h-2', spinner: 'w-5 h-5', shimmer: 'h-5' },
		lg: { dot: 'w-2.5 h-2.5', spinner: 'w-6 h-6', shimmer: 'h-6' }
	};
</script>

{#if variant === 'dots'}
	<div class="ai-loader ai-loader--dots {className}">
		<div class="ai-loader__dot {sizeMap[size].dot}" style="animation-delay: 0ms"></div>
		<div class="ai-loader__dot {sizeMap[size].dot}" style="animation-delay: 150ms"></div>
		<div class="ai-loader__dot {sizeMap[size].dot}" style="animation-delay: 300ms"></div>
	</div>
{:else if variant === 'spinner'}
	<div class="ai-loader ai-loader--spinner {className}">
		<svg class="ai-loader__spinner {sizeMap[size].spinner}" viewBox="0 0 24 24" fill="none">
			<circle class="ai-loader__track" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" />
			<path class="ai-loader__arc" d="M12 2a10 10 0 0 1 10 10" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
		</svg>
	</div>
{:else if variant === 'shimmer'}
	<div class="ai-loader ai-loader--shimmer {sizeMap[size].shimmer} {className}">
		<div class="ai-loader__shimmer-bar"></div>
	</div>
{/if}

<style>
	.ai-loader {
		display: inline-flex;
		align-items: center;
	}

	/* Dots variant */
	.ai-loader--dots {
		gap: 0.375rem;
	}

	.ai-loader__dot {
		border-radius: 50%;
		background-color: var(--muted-foreground);
		animation: dot-bounce 1.4s ease-in-out infinite;
	}

	@keyframes dot-bounce {
		0%, 80%, 100% {
			transform: translateY(0);
			opacity: 0.5;
		}
		40% {
			transform: translateY(-0.375rem);
			opacity: 1;
		}
	}

	/* Spinner variant */
	.ai-loader--spinner {
		display: inline-flex;
	}

	.ai-loader__spinner {
		animation: spin 1s linear infinite;
		color: var(--muted-foreground);
	}

	.ai-loader__track {
		opacity: 0.25;
	}

	.ai-loader__arc {
		opacity: 1;
	}

	@keyframes spin {
		from { transform: rotate(0deg); }
		to { transform: rotate(360deg); }
	}

	/* Shimmer variant */
	.ai-loader--shimmer {
		width: 100%;
		max-width: 200px;
		border-radius: 0.375rem;
		background-color: var(--muted);
		overflow: hidden;
		position: relative;
	}

	.ai-loader__shimmer-bar {
		position: absolute;
		inset: 0;
		background: linear-gradient(
			90deg,
			transparent 0%,
			var(--muted-foreground) 50%,
			transparent 100%
		);
		opacity: 0.1;
		animation: shimmer 1.5s ease-in-out infinite;
	}

	@keyframes shimmer {
		0% { transform: translateX(-100%); }
		100% { transform: translateX(100%); }
	}
</style>
