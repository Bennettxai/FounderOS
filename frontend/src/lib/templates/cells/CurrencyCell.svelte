<script lang="ts">
	/**
	 * CurrencyCell - Display and edit currency values
	 */

	interface Props {
		value: number | null | undefined;
		editable?: boolean;
		currency?: string;
		locale?: string;
		onchange?: (value: number) => void;
	}

	let {
		value,
		editable = false,
		currency = 'USD',
		locale = 'en-US',
		onchange
	}: Props = $props();

	let editing = $state(false);
	let editValue = $state('');

	const formattedValue = $derived(() => {
		if (value == null) return '—';

		return new Intl.NumberFormat(locale, {
			style: 'currency',
			currency: currency,
			minimumFractionDigits: 2,
			maximumFractionDigits: 2
		}).format(value);
	});

	function startEdit() {
		if (editable) {
			editValue = value != null ? String(value) : '';
			editing = true;
		}
	}

	function finishEdit() {
		editing = false;
		const num = parseFloat(editValue);
		if (!isNaN(num) && num !== value) {
			onchange?.(num);
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			finishEdit();
		}
		if (e.key === 'Escape') {
			editValue = value != null ? String(value) : '';
			editing = false;
		}
	}
</script>

{#if editing}
	<input
		type="number"
		class="tpl-cell-edit tpl-cell-currency-edit"
		bind:value={editValue}
		step="0.01"
		onblur={finishEdit}
		onkeydown={handleKeydown}
	/>
{:else}
	<button
		type="button"
		class="tpl-cell tpl-cell-currency"
		class:tpl-cell-editable={editable}
		class:tpl-cell-empty={value == null}
		class:tpl-cell-negative={value != null && value < 0}
		ondblclick={startEdit}
		disabled={!editable}
	>
		{formattedValue()}
	</button>
{/if}

<style>
	.tpl-cell {
		display: block;
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		background: transparent;
		border: none;
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		text-align: right;
		cursor: default;
		font-variant-numeric: tabular-nums;
	}

	.tpl-cell-empty {
		color: var(--tpl-text-muted);
		text-align: left;
	}

	.tpl-cell-negative {
		color: var(--tpl-status-error);
	}

	.tpl-cell-editable {
		cursor: text;
		border-radius: var(--tpl-radius-sm);
		transition: background var(--tpl-transition-fast);
	}

	.tpl-cell-editable:hover {
		background: var(--tpl-bg-hover);
	}

	.tpl-cell-edit {
		width: 100%;
		padding: var(--tpl-space-2) var(--tpl-space-3);
		margin: -1px;
		background: var(--tpl-bg-primary);
		border: 1px solid var(--tpl-accent-primary);
		border-radius: var(--tpl-radius-sm);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
		color: var(--tpl-text-primary);
		text-align: right;
		outline: none;
		box-shadow: var(--tpl-shadow-focus);
	}

	.tpl-cell-currency-edit::-webkit-inner-spin-button,
	.tpl-cell-currency-edit::-webkit-outer-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}

	.tpl-cell-currency-edit {
		-moz-appearance: textfield;
	}
</style>
