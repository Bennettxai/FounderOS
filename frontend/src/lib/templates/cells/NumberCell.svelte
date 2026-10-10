<script lang="ts">
	/**
	 * NumberCell - Display and edit numeric values with formatting
	 */

	interface Props {
		value: number | null | undefined;
		editable?: boolean;
		precision?: number;
		format?: 'decimal' | 'integer' | 'percent';
		prefix?: string;
		suffix?: string;
		locale?: string;
		onchange?: (value: number) => void;
	}

	let {
		value,
		editable = false,
		precision = 2,
		format = 'decimal',
		prefix = '',
		suffix = '',
		locale = 'en-US',
		onchange
	}: Props = $props();

	let editing = $state(false);
	let editValue = $state('');

	const formattedValue = $derived(() => {
		if (value == null) return '—';

		let formatted: string;

		if (format === 'percent') {
			formatted = new Intl.NumberFormat(locale, {
				style: 'percent',
				minimumFractionDigits: precision,
				maximumFractionDigits: precision
			}).format(value / 100);
		} else if (format === 'integer') {
			formatted = new Intl.NumberFormat(locale, {
				style: 'decimal',
				minimumFractionDigits: 0,
				maximumFractionDigits: 0
			}).format(value);
		} else {
			formatted = new Intl.NumberFormat(locale, {
				style: 'decimal',
				minimumFractionDigits: precision,
				maximumFractionDigits: precision
			}).format(value);
		}

		return `${prefix}${formatted}${suffix}`;
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
		class="tpl-cell-edit tpl-cell-number-edit"
		bind:value={editValue}
		step={format === 'integer' ? 1 : Math.pow(10, -precision)}
		onblur={finishEdit}
		onkeydown={handleKeydown}
	/>
{:else}
	<button
		type="button"
		class="tpl-cell tpl-cell-number"
		class:tpl-cell-editable={editable}
		class:tpl-cell-empty={value == null}
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

	.tpl-cell-number-edit::-webkit-inner-spin-button,
	.tpl-cell-number-edit::-webkit-outer-spin-button {
		-webkit-appearance: none;
		margin: 0;
	}

	.tpl-cell-number-edit {
		-moz-appearance: textfield;
	}
</style>
