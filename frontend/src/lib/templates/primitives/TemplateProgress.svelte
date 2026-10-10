<script lang="ts">
	/**
	 * TemplateProgress - Progress bar/circle component for app templates
	 */

	type ProgressVariant = 'bar' | 'circle';
	type ProgressSize = 'sm' | 'md' | 'lg';
	type ProgressColor = 'primary' | 'success' | 'warning' | 'error' | 'info';

	interface Props {
		value?: number;
		max?: number;
		variant?: ProgressVariant;
		size?: ProgressSize;
		color?: ProgressColor;
		showValue?: boolean;
		indeterminate?: boolean;
		label?: string;
		thickness?: number;
	}

	let {
		value = 0,
		max = 100,
		variant = 'bar',
		size = 'md',
		color = 'primary',
		showValue = false,
		indeterminate = false,
		label,
		thickness
	}: Props = $props();

	const percent = $derived(Math.min(Math.max((value / max) * 100, 0), 100));

	const circleSizes = {
		sm: 40,
		md: 64,
		lg: 96
	};

	const circleSize = $derived(circleSizes[size]);
	const strokeWidth = $derived(thickness || (size === 'sm' ? 4 : size === 'md' ? 6 : 8));
	const radius = $derived((circleSize - strokeWidth) / 2);
	const circumference = $derived(2 * Math.PI * radius);
	const offset = $derived(circumference - (percent / 100) * circumference);
</script>

{#if variant === 'bar'}
	<div class="tpl-progress-bar tpl-progress-bar-{size} tpl-progress-{color}">
		{#if label}
			<div class="tpl-progress-bar-header">
				<span class="tpl-progress-bar-label">{label}</span>
				{#if showValue}
					<span class="tpl-progress-bar-value">{Math.round(percent)}%</span>
				{/if}
			</div>
		{/if}
		<div class="tpl-progress-bar-track" role="progressbar" aria-valuenow={value} aria-valuemin={0} aria-valuemax={max}>
			<div
				class="tpl-progress-bar-fill"
				class:tpl-progress-indeterminate={indeterminate}
				style:width={indeterminate ? '100%' : `${percent}%`}
			></div>
		</div>
		{#if showValue && !label}
			<div class="tpl-progress-bar-footer">
				<span class="tpl-progress-bar-value">{Math.round(percent)}%</span>
			</div>
		{/if}
	</div>
{:else}
	<div class="tpl-progress-circle tpl-progress-{color}" style:width="{circleSize}px" style:height="{circleSize}px">
		<svg viewBox="0 0 {circleSize} {circleSize}">
			<circle
				class="tpl-progress-circle-track"
				cx={circleSize / 2}
				cy={circleSize / 2}
				r={radius}
				stroke-width={strokeWidth}
			/>
			<circle
				class="tpl-progress-circle-fill"
				class:tpl-progress-indeterminate={indeterminate}
				cx={circleSize / 2}
				cy={circleSize / 2}
				r={radius}
				stroke-width={strokeWidth}
				stroke-dasharray={circumference}
				stroke-dashoffset={indeterminate ? circumference * 0.75 : offset}
				transform="rotate(-90 {circleSize / 2} {circleSize / 2})"
			/>
		</svg>
		{#if showValue}
			<div class="tpl-progress-circle-content">
				<span class="tpl-progress-circle-value tpl-progress-circle-value-{size}">{Math.round(percent)}%</span>
			</div>
		{/if}
	</div>
{/if}

<style>
	/* ─────────────────────────────────────────────────────────────────────────
	   BAR VARIANT
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-progress-bar {
		width: 100%;
	}

	.tpl-progress-bar-header,
	.tpl-progress-bar-footer {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-family: var(--tpl-font-sans);
	}

	.tpl-progress-bar-header {
		margin-bottom: var(--tpl-space-1-5);
	}

	.tpl-progress-bar-footer {
		margin-top: var(--tpl-space-1);
		justify-content: flex-end;
	}

	.tpl-progress-bar-label {
		font-size: var(--tpl-text-sm);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-primary);
	}

	.tpl-progress-bar-value {
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-muted);
		font-variant-numeric: tabular-nums;
	}

	.tpl-progress-bar-track {
		width: 100%;
		background: var(--tpl-bg-tertiary);
		border-radius: var(--tpl-radius-full);
		overflow: hidden;
	}

	.tpl-progress-bar-sm .tpl-progress-bar-track {
		height: 4px;
	}

	.tpl-progress-bar-md .tpl-progress-bar-track {
		height: 8px;
	}

	.tpl-progress-bar-lg .tpl-progress-bar-track {
		height: 12px;
	}

	.tpl-progress-bar-fill {
		height: 100%;
		border-radius: var(--tpl-radius-full);
		transition: width var(--tpl-transition-slow);
	}

	/* Colors */
	.tpl-progress-primary .tpl-progress-bar-fill {
		background: var(--tpl-accent-primary);
	}

	.tpl-progress-success .tpl-progress-bar-fill {
		background: var(--tpl-status-success);
	}

	.tpl-progress-warning .tpl-progress-bar-fill {
		background: var(--tpl-status-warning);
	}

	.tpl-progress-error .tpl-progress-bar-fill {
		background: var(--tpl-status-error);
	}

	.tpl-progress-info .tpl-progress-bar-fill {
		background: var(--tpl-status-info);
	}

	/* Indeterminate animation */
	.tpl-progress-bar-fill.tpl-progress-indeterminate {
		width: 30% !important;
		animation: tpl-progress-indeterminate 1.5s ease-in-out infinite;
	}

	@keyframes tpl-progress-indeterminate {
		0% {
			transform: translateX(-100%);
		}
		100% {
			transform: translateX(400%);
		}
	}

	/* ─────────────────────────────────────────────────────────────────────────
	   CIRCLE VARIANT
	   ───────────────────────────────────────────────────────────────────────── */
	.tpl-progress-circle {
		position: relative;
		display: inline-flex;
		align-items: center;
		justify-content: center;
	}

	.tpl-progress-circle svg {
		width: 100%;
		height: 100%;
	}

	.tpl-progress-circle-track {
		fill: none;
		stroke: var(--tpl-bg-tertiary);
	}

	.tpl-progress-circle-fill {
		fill: none;
		stroke-linecap: round;
		transition: stroke-dashoffset var(--tpl-transition-slow);
	}

	.tpl-progress-primary .tpl-progress-circle-fill {
		stroke: var(--tpl-accent-primary);
	}

	.tpl-progress-success .tpl-progress-circle-fill {
		stroke: var(--tpl-status-success);
	}

	.tpl-progress-warning .tpl-progress-circle-fill {
		stroke: var(--tpl-status-warning);
	}

	.tpl-progress-error .tpl-progress-circle-fill {
		stroke: var(--tpl-status-error);
	}

	.tpl-progress-info .tpl-progress-circle-fill {
		stroke: var(--tpl-status-info);
	}

	.tpl-progress-circle-fill.tpl-progress-indeterminate {
		animation: tpl-spin 1s linear infinite;
	}

	.tpl-progress-circle-content {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.tpl-progress-circle-value {
		font-family: var(--tpl-font-sans);
		font-weight: var(--tpl-font-semibold);
		color: var(--tpl-text-primary);
		font-variant-numeric: tabular-nums;
	}

	.tpl-progress-circle-value-sm {
		font-size: var(--tpl-text-xs);
	}

	.tpl-progress-circle-value-md {
		font-size: var(--tpl-text-sm);
	}

	.tpl-progress-circle-value-lg {
		font-size: var(--tpl-text-base);
	}
</style>
