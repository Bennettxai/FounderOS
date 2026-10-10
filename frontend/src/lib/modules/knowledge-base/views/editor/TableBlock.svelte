<script lang="ts">
	/**
	 * TableBlock - Editable table component for block editor
	 * Supports adding/removing rows and columns, cell editing
	 */
	import { Plus, Trash2, GripVertical } from 'lucide-svelte';
	import type { Block, RichText } from '../../entities/types';

	interface Props {
		block: Block;
		readOnly?: boolean;
		onBlockChange?: (block: Block) => void;
	}

	let { block, readOnly = false, onBlockChange }: Props = $props();

	// Table data structure stored in block properties
	// Format: { rows: string[][], headerRow: boolean }
	type TableData = { rows: string[][]; headerRow: boolean };

	let tableData = $derived(
		(block.properties.tableData as unknown as TableData) || {
			rows: [
				['', '', ''],
				['', '', '']
			],
			headerRow: true
		}
	);

	let rows = $derived(tableData.rows);
	let hasHeader = $derived(tableData.headerRow);

	function updateTable(newRows: string[][], headerRow = hasHeader) {
		onBlockChange?.({
			...block,
			properties: {
				...block.properties,
				// Store table data as a special format - cast through unknown to satisfy type checker
				tableData: { rows: newRows, headerRow } as unknown as unknown[][]
			}
		});
	}

	function handleCellChange(rowIndex: number, colIndex: number, value: string) {
		const newRows = rows.map((row, ri) =>
			ri === rowIndex ? row.map((cell, ci) => (ci === colIndex ? value : cell)) : [...row]
		);
		updateTable(newRows);
	}

	function addRow(atIndex?: number) {
		const colCount = rows[0]?.length || 3;
		const newRow = Array(colCount).fill('');
		const index = atIndex !== undefined ? atIndex : rows.length;
		const newRows = [...rows.slice(0, index), newRow, ...rows.slice(index)];
		updateTable(newRows);
	}

	function removeRow(index: number) {
		if (rows.length <= 1) return;
		const newRows = rows.filter((_, i) => i !== index);
		updateTable(newRows);
	}

	function addColumn(atIndex?: number) {
		const index = atIndex !== undefined ? atIndex : (rows[0]?.length || 0);
		const newRows = rows.map((row) => [...row.slice(0, index), '', ...row.slice(index)]);
		updateTable(newRows);
	}

	function removeColumn(index: number) {
		if ((rows[0]?.length || 0) <= 1) return;
		const newRows = rows.map((row) => row.filter((_, i) => i !== index));
		updateTable(newRows);
	}

	function toggleHeader() {
		updateTable(rows, !hasHeader);
	}

	// Handle keyboard navigation between cells
	function handleCellKeydown(e: KeyboardEvent, rowIndex: number, colIndex: number) {
		if (e.key === 'Tab') {
			e.preventDefault();
			const nextCol = e.shiftKey ? colIndex - 1 : colIndex + 1;
			const colCount = rows[0]?.length || 0;

			if (nextCol >= 0 && nextCol < colCount) {
				focusCell(rowIndex, nextCol);
			} else if (nextCol >= colCount && rowIndex < rows.length - 1) {
				focusCell(rowIndex + 1, 0);
			} else if (nextCol < 0 && rowIndex > 0) {
				focusCell(rowIndex - 1, colCount - 1);
			}
		} else if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			if (rowIndex < rows.length - 1) {
				focusCell(rowIndex + 1, colIndex);
			} else {
				// Add new row if at last row
				addRow();
				setTimeout(() => focusCell(rowIndex + 1, colIndex), 10);
			}
		}
	}

	function focusCell(rowIndex: number, colIndex: number) {
		const cell = document.querySelector(
			`[data-table-cell="${block.id}-${rowIndex}-${colIndex}"]`
		) as HTMLElement;
		cell?.focus();
	}
</script>

<div class="table-block" class:table-block--readonly={readOnly}>
	<div class="table-block__wrapper">
		<table class="table-block__table">
			{#if hasHeader && rows.length > 0}
				<thead>
					<tr>
						{#each rows[0] as cell, colIndex}
							<th class="table-block__cell table-block__cell--header">
								{#if !readOnly}
									<input
										type="text"
										class="table-block__input table-block__input--header"
										value={cell}
										oninput={(e) => handleCellChange(0, colIndex, (e.target as HTMLInputElement).value)}
										onkeydown={(e) => handleCellKeydown(e, 0, colIndex)}
										data-table-cell="{block.id}-0-{colIndex}"
										placeholder="Header"
									/>
								{:else}
									<span>{cell || 'Header'}</span>
								{/if}
							</th>
						{/each}
						{#if !readOnly}
							<th class="table-block__cell table-block__cell--actions">
								<button class="btn-pill btn-pill-ghost table-block__action" onclick={() => addColumn()} title="Add column">
									<Plus class="h-3 w-3" />
								</button>
							</th>
						{/if}
					</tr>
				</thead>
			{/if}
			<tbody>
				{#each hasHeader ? rows.slice(1) : rows as row, rowOffset}
					{@const rowIndex = hasHeader ? rowOffset + 1 : rowOffset}
					<tr class="table-block__row">
						{#each row as cell, colIndex}
							<td class="table-block__cell">
								{#if !readOnly}
									<input
										type="text"
										class="table-block__input"
										value={cell}
										oninput={(e) => handleCellChange(rowIndex, colIndex, (e.target as HTMLInputElement).value)}
										onkeydown={(e) => handleCellKeydown(e, rowIndex, colIndex)}
										data-table-cell="{block.id}-{rowIndex}-{colIndex}"
										placeholder=""
									/>
								{:else}
									<span>{cell}</span>
								{/if}
							</td>
						{/each}
						{#if !readOnly}
							<td class="table-block__cell table-block__cell--actions">
								<button
									class="btn-pill btn-pill-ghost table-block__action table-block__action--delete"
									onclick={() => removeRow(rowIndex)}
									title="Delete row"
									disabled={rows.length <= 1}
								>
									<Trash2 class="h-3 w-3" />
								</button>
							</td>
						{/if}
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if !readOnly}
		<div class="table-block__controls">
			<button class="btn-pill btn-pill-ghost table-block__control" onclick={() => addRow()}>
				<Plus class="h-3.5 w-3.5" />
				Add row
			</button>
			<button class="btn-pill btn-pill-ghost table-block__control" onclick={toggleHeader}>
				{hasHeader ? 'Remove header' : 'Add header'}
			</button>
		</div>
	{/if}
</div>

<style>
	.table-block {
		margin: 0.5rem 0;
		user-select: none;
	}

	.table-block__wrapper {
		overflow-x: auto;
		border: 1px solid var(--dbd);
		border-radius: 0.375rem;
	}

	.table-block__table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	.table-block__cell {
		border: 1px solid var(--dbd);
		min-width: 100px;
		padding: 0;
		vertical-align: top;
	}

	.table-block__cell--header {
		background-color: var(--dbg2);
		font-weight: 500;
	}

	.table-block__cell--actions {
		width: 32px;
		min-width: 32px;
		border: none;
		padding: 0;
	}

	.table-block__input {
		width: 100%;
		padding: 0.5rem 0.75rem;
		background: transparent;
		border: none;
		outline: none;
		font-size: 0.875rem;
		color: inherit;
	}

	.table-block__input--header {
		font-weight: 500;
	}

	.table-block__input:focus {
		background-color: var(--dbg3);
	}

	.table-block__input::placeholder {
		color: var(--dt4);
	}

	.table-block__action {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		margin: 4px;
		background: transparent;
		border: none;
		border-radius: 0.25rem;
		color: var(--dt3);
		cursor: pointer;
		opacity: 0;
		transition: opacity 0.1s, background-color 0.1s;
	}

	.table-block__row:hover .table-block__action,
	thead:hover .table-block__action {
		opacity: 1;
	}

	.table-block__action:hover {
		background-color: var(--dbg2);
	}

	.table-block__action--delete:hover {
		background-color: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}

	.table-block__action:disabled {
		opacity: 0.3;
		cursor: not-allowed;
	}

	.table-block__controls {
		display: flex;
		gap: 0.5rem;
		margin-top: 0.5rem;
	}

	.table-block__control {
		display: inline-flex;
		align-items: center;
		gap: 0.25rem;
		padding: 0.375rem 0.625rem;
		background: transparent;
		border: 1px dashed var(--dbd);
		border-radius: 0.375rem;
		color: var(--dt3);
		font-size: 0.75rem;
		cursor: pointer;
		transition: background-color 0.1s, border-color 0.1s;
	}

	.table-block__control:hover {
		background-color: var(--dbg2);
		border-color: var(--dt4);
	}

	.table-block--readonly .table-block__cell {
		padding: 0.5rem 0.75rem;
	}
</style>
