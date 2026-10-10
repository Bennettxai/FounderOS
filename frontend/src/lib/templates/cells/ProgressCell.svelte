<script lang="ts">
	/**
	 * ProgressCell - Progress bar display
	 */

	interface Props {
		value: number | null | undefined;
		max?: number;
		showLabel?: boolean;
		color?: string;
		size?: 'sm' | 'md' | 'lg';
	}

	let {
		value = 0,
		max = 100,
		showLabel = true,
		color,
		size = 'md'
	}: Props = $props();

	const percentage = $derived(Math.min(100, Math.max(0, ((value ?? 0) / max) * 100)));

	const barColor = $derived(() => {
		if (color) return color;
		if (percentage >= 100) return 'var(--tpl-status-success)';
		if (percentage >= 70) return 'var(--tpl-accent-primary)';
		if (percentage >= 30) return 'var(--tpl-status-warning)';
		return 'var(--tpl-status-error)';
	});

	const heights: Record<string, string> = {
		sm: '4px',
		md: '8px',
		lg: '12px'
	};
</script>

<div class="tpl-progress" style="--bar-height: {heights[size]}">
	<div class="tpl-progress-track">
		<div
			class="tpl-progress-bar"
			style="width: {percentage}%; background-color: {barColor()}"
			role="progressbar"
			aria-valuenow={value ?? 0}
			aria-valuemin={0}
			aria-valuemax={max}
		></div>
	</div>
	{#if showLabel}
		<span class="tpl-progress-label">{Math.round(percentage)}%</span>
	{/if}
</div>

<style>
	.tpl-progress {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-3);
		padding: var(--tpl-space-2) var(--tpl-space-3);
		width: 100%;
	}

	.tpl-progress-track {
		flex: 1;
		height: var(--bar-height);
		background: var(--tpl-bg-tertiary);
		border-radius: var(--tpl-radius-full);
		overflow: hidden;
	}

	.tpl-progress-bar {
		height: 100%;
		border-radius: var(--tpl-radius-full);
		transition: width var(--tpl-transition-slow);
	}

	.tpl-progress-label {
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
		min-width: 36px;
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
</style>
