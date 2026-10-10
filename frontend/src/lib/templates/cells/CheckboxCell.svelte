<script lang="ts">
	/**
	 * CheckboxCell - Boolean toggle checkbox
	 */

	interface Props {
		value: boolean | null | undefined;
		editable?: boolean;
		label?: string;
		onchange?: (value: boolean) => void;
	}

	let {
		value = false,
		editable = false,
		label,
		onchange
	}: Props = $props();

	function toggle() {
		if (editable) {
			onchange?.(!value);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === ' ' || e.key === 'Enter') {
			e.preventDefault();
			toggle();
		}
	}
</script>

<button
	type="button"
	class="tpl-checkbox-cell"
	class:tpl-checkbox-editable={editable}
	onclick={toggle}
	onkeydown={handleKeydown}
	disabled={!editable}
	role="checkbox"
	aria-checked={value}
>
	<span class="tpl-checkbox" class:tpl-checkbox-checked={value}>
		{#if value}
			<svg viewBox="0 0 16 16" fill="currentColor">
				<path d="M12.207 4.793a1 1 0 010 1.414l-5 5a1 1 0 01-1.414 0l-2-2a1 1 0 011.414-1.414L6.5 9.086l4.293-4.293a1 1 0 011.414 0z" />
			</svg>
		{/if}
	</span>
	{#if label}
		<span class="tpl-checkbox-label">{label}</span>
	{/if}
</button>

<style>
	.tpl-checkbox-cell {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		cursor: default;
	}

	.tpl-checkbox-editable {
		cursor: pointer;
	}

	.tpl-checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border: 2px solid var(--tpl-border-default);
		border-radius: var(--tpl-radius-sm);
		background: var(--tpl-bg-primary);
		transition: all var(--tpl-transition-fast);
		flex-shrink: 0;
	}

	.tpl-checkbox-editable:hover .tpl-checkbox {
		border-color: var(--tpl-border-hover);
	}

	.tpl-checkbox-checked {
		background: var(--tpl-accent-primary);
		border-color: var(--tpl-accent-primary);
		color: white;
	}

	.tpl-checkbox svg {
		width: 12px;
		height: 12px;
	}

	.tpl-checkbox-label {
		color: var(--tpl-text-secondary);
	}

	.tpl-checkbox-cell:focus-visible .tpl-checkbox {
		box-shadow: var(--tpl-shadow-focus);
	}
</style>
