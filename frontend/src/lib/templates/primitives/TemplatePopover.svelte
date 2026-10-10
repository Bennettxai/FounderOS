<script lang="ts">
	/**
	 * TemplatePopover - Positioned popover container for app templates
	 */

	import type { Snippet } from 'svelte';

	type PopoverAlign = 'start' | 'center' | 'end';
	type PopoverSide = 'top' | 'bottom' | 'left' | 'right';

	interface Props {
		open?: boolean;
		align?: PopoverAlign;
		side?: PopoverSide;
		width?: string;
		closeOnClickOutside?: boolean;
		showArrow?: boolean;
		trigger: Snippet;
		children: Snippet;
		onopen?: () => void;
		onclose?: () => void;
	}

	let {
		open = $bindable(false),
		align = 'center',
		side = 'bottom',
		width = 'auto',
		closeOnClickOutside = true,
		showArrow = false,
		trigger,
		children,
		onopen,
		onclose
	}: Props = $props();

	let triggerEl: HTMLElement;
	let popoverEl: HTMLElement;

	function handleTriggerClick() {
		open = !open;
		if (open) {
			onopen?.();
		} else {
			onclose?.();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && open) {
			open = false;
			onclose?.();
			triggerEl?.focus();
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (closeOnClickOutside && open && !triggerEl?.contains(e.target as Node) && !popoverEl?.contains(e.target as Node)) {
			open = false;
			onclose?.();
		}
	}

	$effect(() => {
		if (open) {
			document.addEventListener('click', handleClickOutside);
			document.addEventListener('keydown', handleKeydown);
			return () => {
				document.removeEventListener('click', handleClickOutside);
				document.removeEventListener('keydown', handleKeydown);
			};
		}
	});
</script>

<div class="tpl-popover-wrapper">
	<div
		bind:this={triggerEl}
		class="tpl-popover-trigger"
		onclick={handleTriggerClick}
		onkeydown={(e) => e.key === 'Enter' && handleTriggerClick()}
		role="button"
		tabindex="0"
		aria-haspopup="dialog"
		aria-expanded={open}
	>
		{@render trigger()}
	</div>

	{#if open}
		<div
			bind:this={popoverEl}
			class="tpl-popover tpl-popover-side-{side} tpl-popover-align-{align}"
			class:tpl-popover-has-arrow={showArrow}
			style:width={width}
			role="dialog"
		>
			{#if showArrow}
				<div class="tpl-popover-arrow"></div>
			{/if}
			<div class="tpl-popover-content">
				{@render children()}
			</div>
		</div>
	{/if}
</div>

<style>
	.tpl-popover-wrapper {
		position: relative;
		display: inline-flex;
	}

	.tpl-popover-trigger {
		cursor: pointer;
	}

	.tpl-popover {
		position: absolute;
		z-index: var(--tpl-z-popover);
		background: var(--tpl-bg-elevated);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		box-shadow: var(--tpl-shadow-xl);
		animation: tpl-scale-in var(--tpl-transition-fast) ease-out;
	}

	.tpl-popover-content {
		padding: var(--tpl-space-4);
	}

	/* Side positioning */
	.tpl-popover-side-bottom {
		top: 100%;
		margin-top: var(--tpl-space-2);
	}

	.tpl-popover-side-top {
		bottom: 100%;
		margin-bottom: var(--tpl-space-2);
	}

	.tpl-popover-side-left {
		right: 100%;
		margin-right: var(--tpl-space-2);
	}

	.tpl-popover-side-right {
		left: 100%;
		margin-left: var(--tpl-space-2);
	}

	/* Alignment */
	.tpl-popover-side-bottom.tpl-popover-align-start,
	.tpl-popover-side-top.tpl-popover-align-start {
		left: 0;
	}

	.tpl-popover-side-bottom.tpl-popover-align-center,
	.tpl-popover-side-top.tpl-popover-align-center {
		left: 50%;
		transform: translateX(-50%);
	}

	.tpl-popover-side-bottom.tpl-popover-align-end,
	.tpl-popover-side-top.tpl-popover-align-end {
		right: 0;
	}

	.tpl-popover-side-left.tpl-popover-align-start,
	.tpl-popover-side-right.tpl-popover-align-start {
		top: 0;
	}

	.tpl-popover-side-left.tpl-popover-align-center,
	.tpl-popover-side-right.tpl-popover-align-center {
		top: 50%;
		transform: translateY(-50%);
	}

	.tpl-popover-side-left.tpl-popover-align-end,
	.tpl-popover-side-right.tpl-popover-align-end {
		bottom: 0;
	}

	/* Arrow */
	.tpl-popover-arrow {
		position: absolute;
		width: 12px;
		height: 12px;
		background: var(--tpl-bg-elevated);
		border: 1px solid var(--tpl-border-default);
		transform: rotate(45deg);
	}

	.tpl-popover-side-bottom .tpl-popover-arrow {
		top: -7px;
		border-right: none;
		border-bottom: none;
	}

	.tpl-popover-side-top .tpl-popover-arrow {
		bottom: -7px;
		border-left: none;
		border-top: none;
	}

	.tpl-popover-side-left .tpl-popover-arrow {
		right: -7px;
		border-left: none;
		border-bottom: none;
	}

	.tpl-popover-side-right .tpl-popover-arrow {
		left: -7px;
		border-right: none;
		border-top: none;
	}

	.tpl-popover-align-start .tpl-popover-arrow {
		left: var(--tpl-space-4);
	}

	.tpl-popover-side-bottom.tpl-popover-align-center .tpl-popover-arrow,
	.tpl-popover-side-top.tpl-popover-align-center .tpl-popover-arrow {
		left: 50%;
		margin-left: -6px;
	}

	.tpl-popover-align-end .tpl-popover-arrow {
		right: var(--tpl-space-4);
	}

	.tpl-popover-side-left.tpl-popover-align-center .tpl-popover-arrow,
	.tpl-popover-side-right.tpl-popover-align-center .tpl-popover-arrow {
		top: 50%;
		margin-top: -6px;
	}
</style>
