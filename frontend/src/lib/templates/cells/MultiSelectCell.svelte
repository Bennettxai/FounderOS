<script lang="ts">
	/**
	 * MultiSelectCell - Display multiple selected values as badges
	 */

	interface SelectOption {
		value: string;
		label: string;
		color?: string;
	}

	interface Props {
		value: string[] | null | undefined;
		options: SelectOption[];
		max?: number;
		editable?: boolean;
		onchange?: (value: string[]) => void;
	}

	let {
		value = [],
		options,
		max = 3,
		editable = false,
		onchange
	}: Props = $props();

	let showDropdown = $state(false);

	const selectedOptions = $derived(() => {
		if (!value || value.length === 0) return [];
		return value
			.map((v) => options.find((o) => o.value === v))
			.filter((o): o is SelectOption => o !== undefined);
	});

	const displayOptions = $derived(() => selectedOptions().slice(0, max));
	const remaining = $derived(() => Math.max(0, selectedOptions().length - max));

	function toggleDropdown() {
		if (editable) {
			showDropdown = !showDropdown;
		}
	}

	function toggleOption(option: SelectOption) {
		if (!editable) return;

		const currentValue = value ?? [];
		const isSelected = currentValue.includes(option.value);

		if (isSelected) {
			onchange?.(currentValue.filter((v) => v !== option.value));
		} else {
			onchange?.([...currentValue, option.value]);
		}
	}

	function handleClickOutside(e: MouseEvent) {
		const target = e.target as HTMLElement;
		if (!target.closest('.tpl-multiselect-wrapper')) {
			showDropdown = false;
		}
	}

	function getOptionStyle(opt: SelectOption): string {
		if (!opt.color) return '';
		return `background-color: ${opt.color}20; color: ${opt.color}; border-color: ${opt.color}40`;
	}
</script>

<svelte:window onclick={handleClickOutside} />

<div class="tpl-multiselect-wrapper">
	<button
		type="button"
		class="tpl-multiselect-cell"
		class:tpl-multiselect-editable={editable}
		onclick={toggleDropdown}
		disabled={!editable}
	>
		{#if displayOptions().length === 0}
			<span class="tpl-multiselect-empty">—</span>
		{:else}
			<div class="tpl-multiselect-badges">
				{#each displayOptions() as opt}
					<span class="tpl-multiselect-badge" style={getOptionStyle(opt)}>
						{opt.label}
					</span>
				{/each}
				{#if remaining() > 0}
					<span class="tpl-multiselect-more">+{remaining()}</span>
				{/if}
			</div>
		{/if}
		{#if editable}
			<svg class="tpl-multiselect-chevron" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
			</svg>
		{/if}
	</button>

	{#if showDropdown}
		<div class="tpl-multiselect-dropdown">
			{#each options as option}
				{@const isSelected = value?.includes(option.value)}
				<button
					type="button"
					class="tpl-multiselect-option"
					class:tpl-multiselect-option-selected={isSelected}
					onclick={() => toggleOption(option)}
				>
					<span class="tpl-multiselect-checkbox" class:checked={isSelected}>
						{#if isSelected}
							<svg viewBox="0 0 16 16" fill="currentColor">
								<path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
							</svg>
						{/if}
					</span>
					{#if option.color}
						<span class="tpl-multiselect-option-dot" style="background-color: {option.color}"></span>
					{/if}
					<span>{option.label}</span>
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	.tpl-multiselect-wrapper {
		position: relative;
	}

	.tpl-multiselect-cell {
		display: flex;
		align-items: center;
		justify-content: space-between;
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		cursor: default;
	}

	.tpl-multiselect-editable {
		cursor: pointer;
		border-radius: var(--tpl-radius-sm);
		transition: background var(--tpl-transition-fast);
	}

	.tpl-multiselect-editable:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-multiselect-empty {
		color: var(--tpl-text-muted);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
	}

	.tpl-multiselect-badges {
		display: flex;
		flex-wrap: wrap;
		gap: var(--tpl-space-1);
	}

	.tpl-multiselect-badge {
		display: inline-flex;
		align-items: center;
		height: 22px;
		padding: 0 var(--tpl-space-2);
		background: var(--tpl-bg-tertiary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-full);
		font-family: var(--tpl-font-sans);
		font-size: 11px;
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-secondary);
	}

	.tpl-multiselect-more {
		display: inline-flex;
		align-items: center;
		height: 22px;
		padding: 0 var(--tpl-space-2);
		background: var(--tpl-bg-tertiary);
		border-radius: var(--tpl-radius-full);
		font-family: var(--tpl-font-sans);
		font-size: 11px;
		font-weight: var(--tpl-font-medium);
		color: var(--tpl-text-muted);
	}

	.tpl-multiselect-chevron {
		width: 16px;
		height: 16px;
		color: var(--tpl-text-muted);
		flex-shrink: 0;
	}

	.tpl-multiselect-dropdown {
		position: absolute;
		top: 100%;
		left: 0;
		z-index: var(--tpl-z-dropdown);
		min-width: 200px;
		max-height: 240px;
		margin-top: var(--tpl-space-1);
		padding: var(--tpl-space-1);
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-lg);
		box-shadow: var(--tpl-shadow-lg);
		overflow-y: auto;
	}

	.tpl-multiselect-option {
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

	.tpl-multiselect-option:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-multiselect-checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 16px;
		height: 16px;
		border: 2px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-sm);
		flex-shrink: 0;
	}

	.tpl-multiselect-checkbox.checked {
		background: var(--tpl-accent-primary);
		border-color: var(--tpl-accent-primary);
		color: white;
	}

	.tpl-multiselect-checkbox svg {
		width: 12px;
		height: 12px;
	}

	.tpl-multiselect-option-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		flex-shrink: 0;
	}
</style>
