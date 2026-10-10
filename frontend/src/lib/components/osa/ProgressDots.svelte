<!--
	ProgressDots.svelte
	iOS-style progress dots indicator for onboarding

	Usage:
	<ProgressDots total={4} current={1} />
-->
<script lang="ts">
	interface Props {
		total: number;
		current: number;
		class?: string;
	}

	let {
		total,
		current,
		class: className = ''
	}: Props = $props();

	const classes = `progress-dots ${className}`.trim();

	// Generate array of dots
	const dots = Array.from({ length: total }, (_, i) => i);
</script>

<div class={classes} role="progressbar" aria-valuenow={current} aria-valuemin={0} aria-valuemax={total}>
	{#each dots as index}
		<div
			class="progress-dot"
			class:active={index === current}
		></div>
	{/each}
</div>

<style>
	.progress-dots {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		justify-content: center;
	}

	.progress-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		background-color: rgba(0, 0, 0, 0.2);
		transition: all 0.3s ease;
	}

	.progress-dot.active {
		width: 24px;
		border-radius: 4px;
		background-color: #1A1A1A;
	}

	:global(.dark) .progress-dot {
		background-color: rgba(255, 255, 255, 0.3);
	}

	:global(.dark) .progress-dot.active {
		background-color: #FFFFFF;
	}
</style>
