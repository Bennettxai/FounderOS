<script lang="ts">
	/**
	 * Database Table View
	 * Full table view for database blocks with sorting, filtering, and editing
	 */
	import { Plus, MoreHorizontal, Trash2, Copy, GripVertical } from 'lucide-svelte';
	import { Menu, MenuItem, MenuSeparator, Tooltip } from '$lib/ui';
	import TableHeader from './TableHeader.svelte';
	import TableCell from './TableCell.svelte';
	import ColumnTypeIcon from './ColumnTypeIcon.svelte';
	import type { DatabaseStore } from '../../stores/database-store';
	import type {
		Block,
		ColumnSchema,
		ColumnType,
		CellValue,
		SortConfig
	} from '../../entities/block';

	interface Props {
		store: DatabaseStore;
	}

	let { store }: Props = $props();

	// Local state
	let editingCell: { rowId: string; columnId: string } | null = $state(null);
	let sorts = $state<SortConfig[]>([]);
	let columnWidths = $state<Record<string, number>>({});
	let hoveredRowId = $state<string | null>(null);

	// Derived from store
	let database = $derived($store.database);
	let columns = $derived(database?.props.columns ?? []);
	let rows = $derived(store.getFilteredRows());
	let activeView = $derived(store.getActiveView());

	// Get visible columns based on view
	let visibleColumns = $derived(() => {
		if (!activeView) return columns;
		return activeView.columns
			.map((id) => columns.find((c) => c.id === id))
			.filter((c): c is ColumnSchema => c !== undefined);
	});

	// Column operations
	function handleAddColumn() {
		const name = `Column ${columns.length + 1}`;
		store.addColumn({ name, type: 'text' });
	}

	function handleColumnRename(columnId: string, name: string) {
		store.updateColumn(columnId, { name });
	}

	function handleColumnTypeChange(columnId: string, type: ColumnType) {
		store.updateColumn(columnId, { type });
	}

	function handleColumnDelete(columnId: string) {
		store.deleteColumn(columnId);
	}

	function handleColumnHide(columnId: string) {
		if (!activeView) return;
		const newColumns = activeView.columns.filter((id) => id !== columnId);
		store.setViewColumns(activeView.id, newColumns);
	}

	function handleColumnResize(columnId: string, width: number) {
		columnWidths = { ...columnWidths, [columnId]: width };
		store.updateColumn(columnId, { width });
	}

	function handleColumnSort(columnId: string) {
		const existing = sorts.find((s) => s.columnId === columnId);
		if (!existing) {
			sorts = [{ columnId, direction: 'asc' }];
		} else if (existing.direction === 'asc') {
			sorts = [{ columnId, direction: 'desc' }];
		} else {
			sorts = [];
		}

		if (activeView) {
			store.setViewSorts(activeView.id, sorts.length > 0 ? sorts : undefined);
		}
	}

	function getSortDirection(columnId: string): 'asc' | 'desc' | null {
		const sort = sorts.find((s) => s.columnId === columnId);
		return sort?.direction ?? null;
	}

	// Row operations
	function handleAddRow() {
		store.addRow();
	}

	function handleDeleteRow(rowId: string) {
		store.deleteRow(rowId);
	}

	function handleDuplicateRow(rowId: string) {
		store.duplicateRow(rowId);
	}

	// Cell operations
	function handleCellUpdate(rowId: string, columnId: string, value: CellValue) {
		store.setCell(rowId, columnId, value);
	}

	function handleStartEdit(rowId: string, columnId: string) {
		editingCell = { rowId, columnId };
	}

	function handleEndEdit() {
		editingCell = null;
	}

	function getColumnWidth(column: ColumnSchema): number {
		return columnWidths[column.id] ?? column.width ?? 180;
	}

	// Initialize sorts from active view
	$effect(() => {
		if (activeView?.sorts) {
			sorts = activeView.sorts;
		}
	});
</script>

<div class="bos-database-table">
	<div class="bos-database-table__wrapper">
		<table class="bos-database-table__table" role="grid">
			<thead>
				<tr>
					<!-- Row handle column -->
					<th class="bos-database-table__row-handle-header"></th>

					<!-- Data columns -->
					{#each visibleColumns() as column (column.id)}
						<TableHeader
							{column}
							width={getColumnWidth(column)}
							sortDirection={getSortDirection(column.id)}
							onSort={() => handleColumnSort(column.id)}
							onHide={() => handleColumnHide(column.id)}
							onDelete={() => handleColumnDelete(column.id)}
							onRename={(name) => handleColumnRename(column.id, name)}
							onChangeType={(type) => handleColumnTypeChange(column.id, type)}
							onResize={(width) => handleColumnResize(column.id, width)}
						/>
					{/each}

					<!-- Add column button -->
					<th class="bos-database-table__add-column">
						<Tooltip content="Add column" side="top">
							<button
								class="btn-pill btn-pill-ghost bos-database-table__add-column-btn"
								onclick={handleAddColumn}
							>
								<Plus />
							</button>
						</Tooltip>
					</th>
				</tr>
			</thead>

			<tbody>
				{#each rows as row (row.id)}
					{@const isHovered = hoveredRowId === row.id}
					<tr
						class="bos-database-table__row"
						class:bos-database-table__row--hovered={isHovered}
						onmouseenter={() => (hoveredRowId = row.id)}
						onmouseleave={() => (hoveredRowId = null)}
					>
						<!-- Row handle -->
						<td class="bos-database-table__row-handle">
							<div class="bos-database-table__row-actions">
								{#if isHovered}
									<button class="btn-pill btn-pill-ghost bos-database-table__row-grip">
										<GripVertical />
									</button>
									<Menu>
										{#snippet trigger()}
											<button class="btn-pill btn-pill-ghost bos-database-table__row-menu">
												<MoreHorizontal />
											</button>
										{/snippet}

										<MenuItem onSelect={() => handleDuplicateRow(row.id)}>
											{#snippet prefix()}<Copy />{/snippet}
											Duplicate row
										</MenuItem>
										<MenuSeparator />
										<MenuItem destructive onSelect={() => handleDeleteRow(row.id)}>
											{#snippet prefix()}<Trash2 />{/snippet}
											Delete row
										</MenuItem>
									</Menu>
								{/if}
							</div>
						</td>

						<!-- Data cells -->
						{#each visibleColumns() as column (column.id)}
							{@const cell = store.getCell(row.id, column.id)}
							{@const isEditing = editingCell?.rowId === row.id && editingCell?.columnId === column.id}
							<td
								class="bos-database-table__cell"
								style:width="{getColumnWidth(column)}px"
								style:min-width="{getColumnWidth(column)}px"
							>
								<TableCell
									{column}
									value={cell}
									{isEditing}
									onUpdate={(value) => handleCellUpdate(row.id, column.id, value)}
									onStartEdit={() => handleStartEdit(row.id, column.id)}
									onEndEdit={handleEndEdit}
								/>
							</td>
						{/each}

						<!-- Empty cell for add column -->
						<td class="bos-database-table__cell bos-database-table__cell--empty"></td>
					</tr>
				{/each}

				<!-- Add row button -->
				<tr class="bos-database-table__add-row">
					<td></td>
					<td colspan={visibleColumns().length + 1}>
						<button
							class="btn-pill btn-pill-ghost bos-database-table__add-row-btn"
							onclick={handleAddRow}
						>
							<Plus />
							<span>New row</span>
						</button>
					</td>
				</tr>
			</tbody>
		</table>
	</div>

	<!-- Status bar -->
	<div class="bos-database-table__status">
		<span class="bos-database-table__count">
			{rows.length} {rows.length === 1 ? 'row' : 'rows'}
		</span>
	</div>
</div>

<style>
	.bos-database-table {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--dbg);
	}

	.bos-database-table__wrapper {
		flex: 1;
		overflow: auto;
	}

	.bos-database-table__table {
		width: 100%;
		border-collapse: collapse;
		table-layout: fixed;
	}

	.bos-database-table__row-handle-header {
		width: 32px;
		min-width: 32px;
		background: var(--dbg2);
		border-bottom: 1px solid var(--dbd);
	}

	.bos-database-table__add-column {
		width: 40px;
		min-width: 40px;
		background: var(--dbg2);
		border-bottom: 1px solid var(--dbd);
		text-align: center;
	}

	.bos-database-table__add-column-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 24px;
		height: 24px;
		margin: 4px auto;
		border: none;
		background: transparent;
		border-radius: 4px;
		color: var(--dt4);
		cursor: pointer;
		transition: all 0.15s;
	}

	.bos-database-table__add-column-btn:hover {
		background: var(--dbg3);
		color: var(--dt3);
	}

	.bos-database-table__add-column-btn :global(svg) {
		width: 16px;
		height: 16px;
	}

	.bos-database-table__row {
		border-bottom: 1px solid var(--dbd);
	}

	.bos-database-table__row:nth-child(even) {
		background: var(--dbg2);
	}

	.bos-database-table__row--hovered {
		background: var(--dbg3);
	}

	.bos-database-table__row-handle {
		width: 32px;
		min-width: 32px;
		padding: 0;
		vertical-align: middle;
	}

	.bos-database-table__row-actions {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 2px;
		padding: 0 4px;
	}

	.bos-database-table__row-grip,
	.bos-database-table__row-menu {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border: none;
		background: transparent;
		border-radius: 4px;
		color: var(--dt4);
		cursor: pointer;
	}

	.bos-database-table__row-grip:hover,
	.bos-database-table__row-menu:hover {
		background: var(--dbg3);
		color: var(--dt3);
	}

	.bos-database-table__row-grip {
		cursor: grab;
	}

	.bos-database-table__row-grip :global(svg),
	.bos-database-table__row-menu :global(svg) {
		width: 14px;
		height: 14px;
	}

	.bos-database-table__cell {
		padding: 0;
		border-right: 1px solid var(--dbd);
		vertical-align: middle;
	}

	.bos-database-table__cell--empty {
		border-right: none;
	}

	.bos-database-table__add-row {
		border-bottom: none;
	}

	.bos-database-table__add-row-btn {
		display: inline-flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		border: none;
		background: transparent;
		border-radius: 4px;
		font-size: 14px;
		color: var(--dt2);
		cursor: pointer;
		transition: all 0.15s;
	}

	.bos-database-table__add-row-btn:hover {
		background: var(--dbg3);
		color: var(--dt);
	}

	.bos-database-table__add-row-btn :global(svg) {
		width: 14px;
		height: 14px;
	}

	.bos-database-table__status {
		display: flex;
		align-items: center;
		height: 28px;
		padding: 0 12px;
		border-top: 1px solid var(--dbd);
		background: var(--dbg2);
	}

	.bos-database-table__count {
		font-size: 12px;
		color: var(--dt2);
	}
</style>
