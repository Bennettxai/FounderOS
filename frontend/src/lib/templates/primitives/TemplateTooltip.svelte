<script lang="ts">
	/**
	 * TemplateTooltip - Tooltip component for app templates
	 * Shows helpful text on hover
	 */

	type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

	interface Props {
		text: string;
		position?: TooltipPosition;
		delay?: number;
	}

	let {
		text,
		position = 'top',
		delay = 200,
		children
	}: Props & { children?: any } = $props();

	let visible = $state(false);
	let timeoutId: ReturnType<typeof setTimeout> | null = null;

	function showTooltip() {
		timeoutId = setTimeout(() => {
			visible = true;
		}, delay);
	}

	function hideTooltip() {
		if (timeoutId) {
			clearTimeout(timeoutId);
			timeoutId = null;
		}
		visible = false;
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="tpl-tooltip-wrapper"
	role="group"
	onmouseenter={showTooltip}
	onmouseleave={hideTooltip}
	onfocusin={showTooltip}
	onfocusout={hideTooltip}
>
	{@render children?.()}
	{#if visible && text}
		<div class="tpl-tooltip tpl-tooltip-{position}" role="tooltip">
			{text}
			<span class="tpl-tooltip-arrow"></span>
		</div>
	{/if}
</div>

<style>
	.tpl-tooltip-wrapper {
		position: relative;
		display: inline-flex;
	}

	.tpl-tooltip {
		position: absolute;
		z-index: var(--tpl-z-tooltip);
		padding: var(--tpl-space-1-5) var(--tpl-space-2-5);
		background: var(--tpl-text-primary);
		color: var(--tpl-bg-primary);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		line-height: var(--tpl-leading-snug);
		border-radius: var(--tpl-radius-sm);
		white-space: nowrap;
		pointer-events: none;
		animation: tpl-tooltip-fade 0.15s ease-out;
		box-shadow: var(--tpl-shadow-md);
	}

	@keyframes tpl-tooltip-fade {
		from {
			opacity: 0;
			transform: scale(0.95);
		}
		to {
			opacity: 1;
			transform: scale(1);
		}
	}

	.tpl-tooltip-arrow {
		position: absolute;
		width: 8px;
		height: 8px;
		background: var(--tpl-text-primary);
		transform: rotate(45deg);
	}

	/* Positions */
	.tpl-tooltip-top {
		bottom: 100%;
		left: 50%;
		transform: translateX(-50%);
		margin-bottom: 8px;
	}

	.tpl-tooltip-top .tpl-tooltip-arrow {
		bottom: -4px;
		left: 50%;
		margin-left: -4px;
	}

	.tpl-tooltip-bottom {
		top: 100%;
		left: 50%;
		transform: translateX(-50%);
		margin-top: 8px;
	}

	.tpl-tooltip-bottom .tpl-tooltip-arrow {
		top: -4px;
		left: 50%;
		margin-left: -4px;
	}

	.tpl-tooltip-left {
		right: 100%;
		top: 50%;
		transform: translateY(-50%);
		margin-right: 8px;
	}

	.tpl-tooltip-left .tpl-tooltip-arrow {
		right: -4px;
		top: 50%;
		margin-top: -4px;
	}

	.tpl-tooltip-right {
		left: 100%;
		top: 50%;
		transform: translateY(-50%);
		margin-left: 8px;
	}

	.tpl-tooltip-right .tpl-tooltip-arrow {
		left: -4px;
		top: 50%;
		margin-top: -4px;
	}
</style>
