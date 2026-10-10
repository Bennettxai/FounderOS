<script lang="ts">
	/**
	 * StatusBadge - Colored status indicator pill
	 */

	interface StatusOption {
		value: string;
		label: string;
		color?: string;
	}

	interface Props {
		value: string | null | undefined;
		options: StatusOption[];
		editable?: boolean;
		onchange?: (value: string) => void;
	}

	let {
		value,
		options,
		editable = false,
		onchange
	}: Props = $props();

	let showDropdown = $state(false);

	const currentOption = $derived(options.find((o) => o.value === value));

	const statusColor = $derived(() => {
		if (!currentOption?.color) return 'var(--tpl-bg-tertiary)';
		return currentOption.color;
	});

	const textColor = $derived(() => {
		if (!currentOption?.color) return 'var(--tpl-text-secondary)';
		// Simple contrast calculation
		const color = currentOption.color;
		if (color.includes('bg') || isLightColor(color)) {
			return 'inherit';
		}
		return 'white';
	});

	function isLightColor(color: string): boolean {
		// Basic check for light colors
		if (color.startsWith('#')) {
			const hex = color.slice(1);
			const r = parseInt(hex.slice(0, 2), 16);
			const g = parseInt(hex.slice(2, 4), 16);
			const b = parseInt(hex.slice(4, 6), 16);
			return (r * 299 + g * 587 + b * 114) / 1000 > 128;
		}
		return true;
	}

	function toggleDropdown() {
		if (editable) {
			showDropdown = !showDropdown;
		}
	}

	function selectOption(option: StatusOption) {
		showDropdown = false;
		if (option.value !== value) {
			onchange?.(option.value);
		}
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.tpl-status-wrapper')) {
			showDropdown = false;
		}
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="tpl-status-wrapper">
	<button
		type="button"
		class="tpl-status-badge"
		class:tpl-status-editable={editable}
		class:tpl-status-empty={!currentOption}
		style="background-color: {statusColor()}; color: {textColor()}"
		onclick={toggleDropdown}
		disabled={!editable}
	>
		<span class="tpl-status-dot"></span>
		<span class="tpl-status-label">{currentOption?.label || value || '—'}</span>
		{#if editable}
			<svg class="tpl-status-chevron" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
			</svg>
		{/if}
	</button>

	{#if showDropdown}
		<div class="tpl-status-dropdown">
			{#each options as option}
				<button
					type="button"
					class="tpl-status-option"
					class:tpl-status-option-selected={option.value === value}
					onclick={() => selectOption(option)}
				>
					<span
						class="tpl-status-option-dot"
						style="background-color: {option.color || 'var(--tpl-text-muted)'}"
					></span>
					<span>{option.label}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.tpl-status-wrapper {
		position: relative;
		display: inline-flex;
	}

	.tpl-status-badge {
		display: inline-flex;
		align-items: center;
		gap: var(--tpl-space-2);
		height: 26px;
		padding: 0 var(--tpl-space-3);
		border: none;
		border-radius: var(--tpl-radius-full);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-xs);
		font-weight: var(--tpl-font-medium);
		cursor: default;
		transition: all var(--tpl-transition-fast);
	}

	.tpl-status-editable {
		cursor: pointer;
	}

	.tpl-status-editable:hover {
		filter: brightness(0.95);
	}

	.tpl-status-empty {
		background-color: var(--tpl-bg-tertiary) !important;
		color: var(--tpl-text-muted) !important;
	}

	.tpl-status-dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: currentColor;
		opacity: 0.8;
	}

	.tpl-status-chevron {
		width: 14px;
		height: 14px;
		margin-left: var(--tpl-space-1);
		opacity: 0.6;
	}

	.tpl-status-dropdown {
		position: absolute;
		top: 100%;
		left: 0;
		z-index: var(--tpl-z-dropdown);
		min-width: 160px;
		margin-top: var(--tpl-space-1);
		padding: var(--tpl-space-1);
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		box-shadow: var(--tpl-shadow-lg);
	}

	.tpl-status-option {
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
		transition: background var(--tpl-transition-fast);
	}

	.tpl-status-option:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-status-option-selected {
		background: var(--tpl-bg-selected);
	}

	.tpl-status-option-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}
</style>
