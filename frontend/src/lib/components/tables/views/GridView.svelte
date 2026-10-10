<script lang="ts">
	/**
	 * GridView - Spreadsheet-like table view
	 * Features: Virtual scrolling, keyboard navigation, column resize, row selection
	 */
	import {
		Plus,
		Type,
		Hash,
		Calendar,
		CheckSquare,
		CircleDot,
		Link,
		Mail,
		Paperclip,
		User,
		DollarSign,
		Percent,
		Star,
		Timer,
		Phone,
		Calculator,
		Sigma,
		Link2,
		QrCode,
		Barcode,
		MousePointer,
		Braces,
		Clock,
		AlignLeft
	} from 'lucide-svelte';
	import { onMount } from 'svelte';
	import type { Column, Row, ColumnType } from '$lib/api/tables/types';
	import type { ComponentType, SvelteComponent } from 'svelte';
	import CellRenderer from '../cells/CellRenderer.svelte';

	type IconComponent = ComponentType<SvelteComponent>;

	// Column type icons
	const columnTypeIcons: Record<ColumnType, IconComponent> = {
		text: Type as unknown as IconComponent,
		long_text: AlignLeft as unknown as IconComponent,
		number: Hash as unknown as IconComponent,
		single_select: CircleDot as unknown as IconComponent,
		multi_select: CheckSquare as unknown as IconComponent,
		date: Calendar as unknown as IconComponent,
		datetime: Clock as unknown as IconComponent,
		checkbox: CheckSquare as unknown as IconComponent,
		url: Link as unknown as IconComponent,
		email: Mail as unknown as IconComponent,
		attachment: Paperclip as unknown as IconComponent,
		user: User as unknown as IconComponent,
		currency: DollarSign as unknown as IconComponent,
		percent: Percent as unknown as IconComponent,
		rating: Star as unknown as IconComponent,
		duration: Timer as unknown as IconComponent,
		phone: Phone as unknown as IconComponent,
		lookup: Calculator as unknown as IconComponent,
		rollup: Calculator as unknown as IconComponent,
		formula: Sigma as unknown as IconComponent,
		link_to_record: Link2 as unknown as IconComponent,
		qr_code: QrCode as unknown as IconComponent,
		barcode: Barcode as unknown as IconComponent,
		button: MousePointer as unknown as IconComponent,
		json: Braces as unknown as IconComponent
	};

	function getColumnIcon(type: ColumnType): IconComponent {
		return columnTypeIcons[type] || (Type as unknown as IconComponent);
	}

	interface Props {
		columns: Column[];
		rows: Row[];
		selectedRowIds: Set<string>;
		editingCell: { rowId: string; columnId: string } | null;
		columnWidths: Record<string, number>;
		onCellChange: (rowId: string, columnId: string, value: unknown) => void;
		onRowSelect: (rowId: string, shiftKey?: boolean) => void;
		onSelectAll: () => void;
		onCellEdit: (rowId: string, columnId: string) => void;
		onCellBlur: () => void;
		onAddRow: () => void;
		onAddColumn: () => void;
		onColumnResize?: (columnId: string, width: number) => void;
	}

	let {
		columns,
		rows,
		selectedRowIds,
		editingCell,
		columnWidths,
		onCellChange,
		onRowSelect,
		onSelectAll,
		onCellEdit,
		onCellBlur,
		onAddRow,
		onAddColumn,
		onColumnResize
	}: Props = $props();

	// Virtual scrolling state
	const ROW_HEIGHT = 36; // Height of each row in pixels
	const BUFFER_ROWS = 5; // Extra rows to render above/below viewport
	let containerRef = $state<HTMLDivElement | null>(null);
	let scrollTop = $state(0);
	let containerHeight = $state(600);

	// Column resize state
	let resizingColumn = $state<string | null>(null);
	let resizeStartX = $state(0);
	let resizeStartWidth = $state(0);

	// Focused cell for keyboard navigation
	let focusedCell = $state<{ rowIndex: number; colIndex: number } | null>(null);

	// Last selected row for shift-click range selection
	let lastSelectedRowIndex = $state<number | null>(null);

	// Calculate visible row range for virtual scrolling
	const virtualRowRange = $derived.by(() => {
		const start = Math.max(0, Math.floor(scrollTop / ROW_HEIGHT) - BUFFER_ROWS);
		const visibleRows = Math.ceil(containerHeight / ROW_HEIGHT) + BUFFER_ROWS * 2;
		const end = Math.min(rows.length, start + visibleRows);
		return { start, end };
	});

	// Get visible rows
	const visibleRows = $derived(rows.slice(virtualRowRange.start, virtualRowRange.end));

	// Total height of all rows (for scroll container)
	const totalHeight = $derived(rows.length * ROW_HEIGHT);

	// Offset for visible rows positioning
	const offsetY = $derived(virtualRowRange.start * ROW_HEIGHT);

	const allSelected = $derived(
		rows.length > 0 && rows.every((row) => selectedRowIds.has(row.id))
	);

	// Handle scroll for virtual scrolling
	function handleScroll(e: Event) {
		const target = e.target as HTMLDivElement;
		scrollTop = target.scrollTop;
	}

	// Update container height on mount and resize
	onMount(() => {
		if (containerRef) {
			containerHeight = containerRef.clientHeight;
			const resizeObserver = new ResizeObserver((entries) => {
				for (const entry of entries) {
					containerHeight = entry.contentRect.height;
				}
			});
			resizeObserver.observe(containerRef);
			return () => resizeObserver.disconnect();
		}
	});

	function getColumnWidth(column: Column): number {
		return columnWidths[column.id] || column.width || 150;
	}

	function handleResizeStart(e: MouseEvent, columnId: string) {
		e.preventDefault();
		resizingColumn = columnId;
		resizeStartX = e.clientX;
		resizeStartWidth = columnWidths[columnId] || 150;

		window.addEventListener('mousemove', handleResizeMove);
		window.addEventListener('mouseup', handleResizeEnd);
	}

	function handleResizeMove(e: MouseEvent) {
		if (!resizingColumn) return;

		const diff = e.clientX - resizeStartX;
		const newWidth = Math.max(80, resizeStartWidth + diff);

		if (onColumnResize) {
			onColumnResize(resizingColumn, newWidth);
		}
	}

	function handleResizeEnd() {
		resizingColumn = null;
		window.removeEventListener('mousemove', handleResizeMove);
		window.removeEventListener('mouseup', handleResizeEnd);
	}

	// Handle row selection with shift-click for range selection
	function handleRowSelect(rowIndex: number, shiftKey: boolean) {
		if (shiftKey && lastSelectedRowIndex !== null) {
			// Range selection: select all rows between last selected and current
			const start = Math.min(lastSelectedRowIndex, rowIndex);
			const end = Math.max(lastSelectedRowIndex, rowIndex);
			for (let i = start; i <= end; i++) {
				onRowSelect(rows[i].id, false);
			}
		} else {
			onRowSelect(rows[rowIndex].id, false);
			lastSelectedRowIndex = rowIndex;
		}
	}

	// Keyboard navigation handler
	function handleCellKeydown(e: KeyboardEvent, rowIndex: number, colIndex: number) {
		const actualRowIndex = virtualRowRange.start + rowIndex;

		switch (e.key) {
			case 'Tab':
				e.preventDefault();
				const nextCol = e.shiftKey ? colIndex - 1 : colIndex + 1;
				if (nextCol >= 0 && nextCol < columns.length) {
					onCellEdit(rows[actualRowIndex].id, columns[nextCol].id);
					focusedCell = { rowIndex: actualRowIndex, colIndex: nextCol };
				} else if (nextCol >= columns.length && actualRowIndex < rows.length - 1) {
					onCellEdit(rows[actualRowIndex + 1].id, columns[0].id);
					focusedCell = { rowIndex: actualRowIndex + 1, colIndex: 0 };
				} else if (nextCol < 0 && actualRowIndex > 0) {
					onCellEdit(rows[actualRowIndex - 1].id, columns[columns.length - 1].id);
					focusedCell = { rowIndex: actualRowIndex - 1, colIndex: columns.length - 1 };
				}
				break;

			case 'Enter':
				if (!e.shiftKey) {
					e.preventDefault();
					if (actualRowIndex < rows.length - 1) {
						onCellEdit(rows[actualRowIndex + 1].id, columns[colIndex].id);
						focusedCell = { rowIndex: actualRowIndex + 1, colIndex };
					}
				}
				break;

			case 'Escape':
				onCellBlur();
				break;

			case 'ArrowUp':
				e.preventDefault();
				if (actualRowIndex > 0) {
					onCellEdit(rows[actualRowIndex - 1].id, columns[colIndex].id);
					focusedCell = { rowIndex: actualRowIndex - 1, colIndex };
					scrollToRow(actualRowIndex - 1);
				}
				break;

			case 'ArrowDown':
				e.preventDefault();
				if (actualRowIndex < rows.length - 1) {
					onCellEdit(rows[actualRowIndex + 1].id, columns[colIndex].id);
					focusedCell = { rowIndex: actualRowIndex + 1, colIndex };
					scrollToRow(actualRowIndex + 1);
				}
				break;

			case 'ArrowLeft':
				if (!editingCell) {
					e.preventDefault();
					if (colIndex > 0) {
						onCellEdit(rows[actualRowIndex].id, columns[colIndex - 1].id);
						focusedCell = { rowIndex: actualRowIndex, colIndex: colIndex - 1 };
					}
				}
				break;

			case 'ArrowRight':
				if (!editingCell) {
					e.preventDefault();
					if (colIndex < columns.length - 1) {
						onCellEdit(rows[actualRowIndex].id, columns[colIndex + 1].id);
						focusedCell = { rowIndex: actualRowIndex, colIndex: colIndex + 1 };
					}
				}
				break;
		}
	}

	// Scroll to ensure a row is visible
	function scrollToRow(rowIndex: number) {
		if (!containerRef) return;
		const rowTop = rowIndex * ROW_HEIGHT;
		const rowBottom = rowTop + ROW_HEIGHT;
		const viewTop = scrollTop;
		const viewBottom = scrollTop + containerHeight - 40; // Account for header

		if (rowTop < viewTop) {
			containerRef.scrollTop = rowTop;
		} else if (rowBottom > viewBottom) {
			containerRef.scrollTop = rowBottom - containerHeight + 40;
		}
	}
</script>

<div
	bind:this={containerRef}
	class="flex-1 overflow-auto"
	onscroll={handleScroll}
>
	<table class="dt2-table w-full border-collapse">
		<!-- Header -->
		<thead class="dt2-thead sticky top-0 z-10">
			<tr>
				<!-- Checkbox column -->
				<th class="dt2-th w-10 px-2 py-2">
					<input
						type="checkbox"
						checked={allSelected}
						onchange={onSelectAll}
						class="h-4 w-4 rounded"
					/>
				</th>

				<!-- Row number column -->
				<th class="dt2-th w-12 px-2 py-2 text-center text-xs font-medium">
					#
				</th>

				<!-- Data columns -->
				{#each columns as column, colIndex}
					{@const ColumnIcon = getColumnIcon(column.type)}
					<th
						class="dt2-th relative px-3 py-2 text-left text-sm font-medium"
						style="width: {getColumnWidth(column)}px; min-width: {getColumnWidth(column)}px;"
					>
						<div class="flex items-center gap-2">
							<!-- Column type icon -->
							<svelte:component this={ColumnIcon} class="dt2-icon h-4 w-4 flex-shrink-0" />
							<span class="truncate">{column.name}</span>
							{#if column.is_primary}
								<span class="rounded bg-blue-100 px-1 py-0.5 text-[10px] font-medium text-blue-600">Primary</span>
							{/if}
							{#if column.is_required}
								<span class="text-red-500">*</span>
							{/if}
						</div>

						<!-- Resize handle -->
						{#if onColumnResize}
							<button
								type="button"
								class="btn-pill btn-pill-ghost absolute -right-1 top-0 h-full w-2 cursor-col-resize"
								onmousedown={(e) => handleResizeStart(e, column.id)}
							></button>
						{/if}
					</th>
				{/each}

				<!-- Add column button -->
				<th class="dt2-th w-10 px-2 py-2" style="border-right: none;">
					<button
						type="button"
						class="btn-pill btn-pill-ghost flex h-6 w-6 items-center justify-center"
						onclick={onAddColumn}
					>
						<Plus class="h-4 w-4" />
					</button>
				</th>
			</tr>
		</thead>

		<!-- Body with virtual scrolling -->
		<tbody>
			<!-- Spacer for virtual scroll positioning -->
			{#if virtualRowRange.start > 0}
				<tr style="height: {offsetY}px;">
					<td colspan={columns.length + 3}></td>
				</tr>
			{/if}

			<!-- Visible rows only -->
			{#each visibleRows as row, localRowIndex (row.id)}
				{@const actualRowIndex = virtualRowRange.start + localRowIndex}
				<tr
					class="dt2-row group"
					class:dt2-row--selected={selectedRowIds.has(row.id)}
					style="height: {ROW_HEIGHT}px;"
				>
					<!-- Checkbox -->
					<td class="dt2-td px-2 py-1">
						<input
							type="checkbox"
							checked={selectedRowIds.has(row.id)}
							onclick={(e) => handleRowSelect(actualRowIndex, e.shiftKey)}
							class="h-4 w-4 rounded"
						/>
					</td>

					<!-- Row number (actual row number, not visible index) -->
					<td class="dt2-td dt2-td--muted px-2 py-1 text-center text-xs">
						{actualRowIndex + 1}
					</td>

					<!-- Data cells -->
					{#each columns as column, colIndex}
						{@const isEditing =
							editingCell?.rowId === row.id && editingCell?.columnId === column.id}
						{@const isFocused =
							focusedCell?.rowIndex === actualRowIndex && focusedCell?.colIndex === colIndex}
						<td
							class="dt2-td p-0"
							style="width: {getColumnWidth(column)}px; min-width: {getColumnWidth(column)}px;"
						>
							<div
								class="cell-container h-full w-full cursor-text px-2 py-1 outline-none transition-shadow"
								class:cell-focused={isFocused || isEditing}
								class:cell-editing={isEditing}
								onclick={() => onCellEdit(row.id, column.id)}
								onkeydown={(e) => handleCellKeydown(e, localRowIndex, colIndex)}
								onfocus={() => focusedCell = { rowIndex: actualRowIndex, colIndex }}
								role="gridcell"
								tabindex="0"
							>
								<CellRenderer
									type={column.type}
									value={row.data[column.id]}
									options={column.options}
									editing={isEditing}
									onChange={(value) => onCellChange(row.id, column.id, value)}
									onBlur={onCellBlur}
								/>
							</div>
						</td>
					{/each}

					<!-- Empty cell for add column -->
					<td class="dt2-td" style="border-right: none;"></td>
				</tr>
			{/each}

			<!-- Bottom spacer for remaining rows -->
			{#if virtualRowRange.end < rows.length}
				<tr style="height: {(rows.length - virtualRowRange.end) * ROW_HEIGHT}px;">
					<td colspan={columns.length + 3}></td>
				</tr>
			{/if}

			<!-- Add row button -->
			<tr>
				<td colspan={columns.length + 3} class="dt2-td p-0" style="border-right: none;">
					<button
						type="button"
						class="dt2-add-row flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors"
						onclick={onAddRow}
					>
						<Plus class="h-4 w-4" />
						Add row
					</button>
				</td>
			</tr>
		</tbody>
	</table>
</div>

<style>
	/* dt2- Data Table Foundation tokens */
	:global(.dt2-thead) {
		background: var(--dbg2);
	}

	:global(.dt2-th) {
		background: var(--dbg2);
		border-bottom: 1px solid var(--dbd);
		border-right: 1px solid var(--dbd);
		color: var(--dt2);
	}

	:global(.dt2-icon) {
		color: var(--dt3);
	}

	:global(.dt2-td) {
		background: var(--dbg);
		border-bottom: 1px solid var(--dbd2);
		border-right: 1px solid var(--dbd2);
		color: var(--dt);
	}

	:global(.dt2-td--muted) {
		color: var(--dt3);
	}

	:global(.dt2-row:hover) {
		background: var(--dbg2);
	}

	:global(.dt2-row:hover .dt2-td) {
		background: var(--dbg2);
	}

	:global(.dt2-row--selected) {
		background: rgba(30, 150, 235, 0.08);
	}

	:global(.dt2-row--selected .dt2-td) {
		background: rgba(30, 150, 235, 0.08);
	}

	:global(.dt2-add-row) {
		color: var(--dt3);
		border-radius: 0;
	}

	:global(.dt2-add-row:hover) {
		background: var(--dbg3);
		color: var(--dt);
	}

	/* Cell focus ring - NocoDB-style */
	.cell-container:focus {
		outline: none;
	}

	.cell-focused {
		box-shadow: inset 0 0 0 2px var(--dt2-accent, #3b82f6);
		background-color: var(--dbg2);
	}

	.cell-editing {
		box-shadow: inset 0 0 0 2px var(--dt2-accent-strong, #2563eb);
		background-color: var(--dbg);
	}

	/* Custom scrollbar */
	div::-webkit-scrollbar {
		width: 8px;
		height: 8px;
	}

	div::-webkit-scrollbar-track {
		background: var(--dbg2);
	}

	div::-webkit-scrollbar-thumb {
		background: var(--dbd);
		border-radius: 4px;
	}

	div::-webkit-scrollbar-thumb:hover {
		background: var(--dt3);
	}
</style>
