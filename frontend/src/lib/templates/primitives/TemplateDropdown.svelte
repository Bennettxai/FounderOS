<script lang="ts">
	/**
	 * TemplateDropdown - Action dropdown menu for app templates
	 */

	import type { Snippet } from 'svelte';

	type DropdownAlign = 'start' | 'end';
	type DropdownSide = 'top' | 'bottom';

	interface DropdownItem {
		id: string;
		label: string;
		icon?: string;
		disabled?: boolean;
		danger?: boolean;
		divider?: boolean;
	}

	interface Props {
		items?: DropdownItem[];
		align?: DropdownAlign;
		side?: DropdownSide;
		disabled?: boolean;
		trigger: Snippet;
		children?: Snippet;
		onselect?: (item: DropdownItem) => void;
	}

	let {
		items = [],
		align = 'start',
		side = 'bottom',
		disabled = false,
		trigger,
		children,
		onselect
	}: Props = $props();

	let open = $state(false);
	let triggerEl: HTMLElement;
	let menuEl: HTMLElement;

	function handleTriggerClick() {
		if (!disabled) {
			open = !open;
		}
	}

	function handleItemClick(item: DropdownItem) {
		if (!item.disabled && !item.divider) {
			onselect?.(item);
			open = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			open = false;
			triggerEl?.focus();
		}
	}

	function handleClickOutside(e: MouseEvent) {
		if (open && !triggerEl?.contains(e.target as Node) && !menuEl?.contains(e.target as Node)) {
			open = false;
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

<div class="tpl-dropdown">
	<div
		bind:this={triggerEl}
		class="tpl-dropdown-trigger"
		onclick={handleTriggerClick}
		onkeydown={(e) => e.key === 'Enter' && handleTriggerClick()}
		role="button"
		tabindex={disabled ? -1 : 0}
		aria-haspopup="true"
		aria-expanded={open}
	>
		{@render trigger()}
	</div>

	{#if open}
		<div
			bind:this={menuEl}
			class="tpl-dropdown-menu tpl-dropdown-align-{align} tpl-dropdown-side-{side}"
			role="menu"
		>
			{#if children}
				{@render children()}
			{:else}
				{#each items as item}
					{#if item.divider}
						<div class="tpl-dropdown-divider"></div>
					{:else}
						<button
							type="button"
							class="tpl-dropdown-item"
							class:tpl-dropdown-item-danger={item.danger}
							class:tpl-dropdown-item-disabled={item.disabled}
							disabled={item.disabled}
							role="menuitem"
							onclick={() => handleItemClick(item)}
						>
							{#if item.icon}
								<span class="tpl-dropdown-item-icon">{item.icon}</span>
							{/if}
							<span class="tpl-dropdown-item-label">{item.label}</span>
						</button>
					{/if}
				{/each}
			{/if}
		</div>
	{/if}
</div>

<style>
	.tpl-dropdown {
		position: relative;
		display: inline-flex;
	}

	.tpl-dropdown-trigger {
		cursor: pointer;
	}

	.tpl-dropdown-menu {
		position: absolute;
		z-index: var(--tpl-z-dropdown);
		min-width: 180px;
		max-width: 280px;
		padding: var(--tpl-space-1);
		background: var(--tpl-bg-elevated);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		box-shadow: var(--tpl-shadow-lg);
		animation: tpl-scale-in var(--tpl-transition-fast) ease-out;
	}

	/* Alignment */
	.tpl-dropdown-align-start {
		left: 0;
	}

	.tpl-dropdown-align-end {
		right: 0;
	}

	/* Side */
	.tpl-dropdown-side-bottom {
		top: 100%;
		margin-top: var(--tpl-space-1);
	}

	.tpl-dropdown-side-top {
		bottom: 100%;
		margin-bottom: var(--tpl-space-1);
	}

	/* Items */
	.tpl-dropdown-item {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		border-radius: var(--tpl-radius-md);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		text-align: left;
		cursor: pointer;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-dropdown-item:hover:not(:disabled) {
		background: var(--tpl-bg-hover);
	}

	.tpl-dropdown-item:focus-visible {
		outline: none;
		background: var(--tpl-bg-hover);
		box-shadow: inset 0 0 0 2px var(--tpl-accent-primary);
	}

	.tpl-dropdown-item-danger {
		color: var(--tpl-status-error);
	}

	.tpl-dropdown-item-danger:hover:not(:disabled) {
		background: var(--tpl-status-error-bg);
	}

	.tpl-dropdown-item-disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.tpl-dropdown-item-icon {
		flex-shrink: 0;
		width: var(--tpl-icon-md);
		height: var(--tpl-icon-md);
		color: var(--tpl-text-muted);
	}

	.tpl-dropdown-item-danger .tpl-dropdown-item-icon {
		color: var(--tpl-status-error);
	}

	.tpl-dropdown-item-label {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	/* Divider */
	.tpl-dropdown-divider {
		height: 1px;
		margin: var(--tpl-space-1) 0;
		background: var(--tpl-border-subtle);
	}
</style>
