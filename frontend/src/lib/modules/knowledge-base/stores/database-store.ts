/**
 * Database Store - DataSource Layer
 *
 * Provides abstraction for database block operations:
 * - Column management (add, update, delete, reorder)
 * - Row management (add, delete, move)
 * - Cell CRUD operations
 * - View management (table, kanban, calendar, etc.)
 * - Filtering and sorting
 *
 * Works with the Yjs-backed block store for real-time collaboration.
 */

import { derived, writable, get, type Readable } from 'svelte/store';
import type {
	Block,
	BlockFlavour,
	DatabaseBlockProps,
	DatabaseRowBlockProps,
	ColumnSchema,
	ColumnType,
	CellValue,
	DatabaseView,
	DatabaseViewType,
	FilterGroup,
	SortConfig,
	SelectOption,
	TextDelta
} from '../entities/block';
import type { BlockStore } from './yjs-block-store';

// ============================================================================
// Types
// ============================================================================

export interface DatabaseState {
	/** The database block */
	database: Block<'bos:database'> | null;
	/** Row blocks (children of database) */
	rows: Block<'bos:database-row'>[];
	/** Active view ID */
	activeViewId: string | null;
	/** Loading state */
	loading: boolean;
	/** Error state */
	error: string | null;
}

export interface CellUpdate {
	rowId: string;
	columnId: string;
	value: CellValue;
}

export interface ColumnUpdate {
	id: string;
	name?: string;
	type?: ColumnType;
	width?: number;
	data?: ColumnSchema['data'];
}

// ============================================================================
// Database Store Factory
// ============================================================================

/**
 * Create a database store for a specific database block.
 */
export function createDatabaseStore(blockStore: BlockStore, databaseId: string) {
	const { subscribe, set, update } = writable<DatabaseState>({
		database: null,
		rows: [],
		activeViewId: null,
		loading: true,
		error: null
	});

	// Sync from block store
	function sync() {
		const block = blockStore.getBlock(databaseId);
		if (!block || block.flavour !== 'bos:database') {
			update((s) => ({ ...s, database: null, rows: [], loading: false }));
			return;
		}

		const database = block as Block<'bos:database'>;
		const rows = blockStore
			.getChildren(databaseId)
			.filter((b): b is Block<'bos:database-row'> => b.flavour === 'bos:database-row');

		update((s) => ({
			...s,
			database,
			rows,
			activeViewId: s.activeViewId ?? database.props.views[0]?.id ?? null,
			loading: false
		}));
	}

	// Subscribe to block store changes
	const unsubscribe = blockStore.subscribe(() => sync());

	// Initial sync
	sync();

	return {
		subscribe,

		// ========================================================================
		// Column Operations
		// ========================================================================

		/**
		 * Add a new column to the database
		 */
		addColumn(column: Omit<ColumnSchema, 'id'>) {
			const state = get({ subscribe });
			if (!state.database) return;

			const id = crypto.randomUUID();
			const newColumn: ColumnSchema = { id, ...column };

			blockStore.updateBlock(databaseId, {
				columns: [...state.database.props.columns, newColumn]
			} as Partial<DatabaseBlockProps>);

			// Add column to all views
			const views = state.database.props.views.map((view) => ({
				...view,
				columns: [...view.columns, id]
			}));
			blockStore.updateBlock(databaseId, { views } as Partial<DatabaseBlockProps>);

			return id;
		},

		/**
		 * Update a column
		 */
		updateColumn(columnId: string, updates: Partial<Omit<ColumnSchema, 'id'>>) {
			const state = get({ subscribe });
			if (!state.database) return;

			const columns = state.database.props.columns.map((col) =>
				col.id === columnId ? { ...col, ...updates } : col
			);

			blockStore.updateBlock(databaseId, { columns } as Partial<DatabaseBlockProps>);
		},

		/**
		 * Delete a column
		 */
		deleteColumn(columnId: string) {
			const state = get({ subscribe });
			if (!state.database) return;

			// Remove from columns
			const columns = state.database.props.columns.filter((col) => col.id !== columnId);

			// Remove from all views
			const views = state.database.props.views.map((view) => ({
				...view,
				columns: view.columns.filter((id) => id !== columnId)
			}));

			// Remove cells for this column
			const cells = { ...state.database.props.cells };
			for (const rowId of Object.keys(cells)) {
				const rowCells = { ...cells[rowId] };
				delete rowCells[columnId];
				cells[rowId] = rowCells;
			}

			blockStore.updateBlock(databaseId, { columns, views, cells } as Partial<DatabaseBlockProps>);
		},

		/**
		 * Reorder columns
		 */
		reorderColumns(columnIds: string[]) {
			const state = get({ subscribe });
			if (!state.database) return;

			// Reorder columns array
			const columnMap = new Map(state.database.props.columns.map((c) => [c.id, c]));
			const columns = columnIds.map((id) => columnMap.get(id)!).filter(Boolean);

			blockStore.updateBlock(databaseId, { columns } as Partial<DatabaseBlockProps>);
		},

		/**
		 * Get column by ID
		 */
		getColumn(columnId: string): ColumnSchema | null {
			const state = get({ subscribe });
			return state.database?.props.columns.find((c) => c.id === columnId) ?? null;
		},

		// ========================================================================
		// Row Operations
		// ========================================================================

		/**
		 * Add a new row to the database
		 */
		addRow(initialCells?: Record<string, CellValue>, index?: number) {
			const state = get({ subscribe });
			if (!state.database) return;

			// Create row block
			const rowBlock = blockStore.addBlock<'bos:database-row'>(
				'bos:database-row',
				{ databaseId } as DatabaseRowBlockProps,
				databaseId,
				index
			);

			// Set initial cells if provided
			if (initialCells) {
				const cells = { ...state.database.props.cells };
				cells[rowBlock.id] = initialCells;
				blockStore.updateBlock(databaseId, { cells } as Partial<DatabaseBlockProps>);
			}

			return rowBlock.id;
		},

		/**
		 * Delete a row
		 */
		deleteRow(rowId: string) {
			const state = get({ subscribe });
			if (!state.database) return;

			// Remove cells for this row
			const cells = { ...state.database.props.cells };
			delete cells[rowId];
			blockStore.updateBlock(databaseId, { cells } as Partial<DatabaseBlockProps>);

			// Delete the row block
			blockStore.deleteBlock(rowId);
		},

		/**
		 * Move a row to a new position
		 */
		moveRow(rowId: string, newIndex: number) {
			blockStore.moveBlock(rowId, databaseId, newIndex);
		},

		/**
		 * Duplicate a row
		 */
		duplicateRow(rowId: string) {
			const state = get({ subscribe });
			if (!state.database) return;

			const cells = state.database.props.cells[rowId];
			return this.addRow(cells ? { ...cells } : undefined);
		},

		// ========================================================================
		// Cell Operations
		// ========================================================================

		/**
		 * Get cell value
		 */
		getCell(rowId: string, columnId: string): CellValue | null {
			const state = get({ subscribe });
			return state.database?.props.cells[rowId]?.[columnId] ?? null;
		},

		/**
		 * Set cell value
		 */
		setCell(rowId: string, columnId: string, value: CellValue) {
			const state = get({ subscribe });
			if (!state.database) return;

			const cells = { ...state.database.props.cells };
			if (!cells[rowId]) {
				cells[rowId] = {};
			}
			cells[rowId] = { ...cells[rowId], [columnId]: value };

			blockStore.updateBlock(databaseId, { cells } as Partial<DatabaseBlockProps>);
		},

		/**
		 * Batch update cells
		 */
		setCells(updates: CellUpdate[]) {
			const state = get({ subscribe });
			if (!state.database) return;

			const cells = { ...state.database.props.cells };

			for (const { rowId, columnId, value } of updates) {
				if (!cells[rowId]) {
					cells[rowId] = {};
				}
				cells[rowId] = { ...cells[rowId], [columnId]: value };
			}

			blockStore.updateBlock(databaseId, { cells } as Partial<DatabaseBlockProps>);
		},

		/**
		 * Clear cell value
		 */
		clearCell(rowId: string, columnId: string) {
			const state = get({ subscribe });
			if (!state.database) return;

			const cells = { ...state.database.props.cells };
			if (cells[rowId]) {
				const rowCells = { ...cells[rowId] };
				delete rowCells[columnId];
				cells[rowId] = rowCells;
				blockStore.updateBlock(databaseId, { cells } as Partial<DatabaseBlockProps>);
			}
		},

		// ========================================================================
		// View Operations
		// ========================================================================

		/**
		 * Add a new view
		 */
		addView(view: Omit<DatabaseView, 'id'>) {
			const state = get({ subscribe });
			if (!state.database) return;

			const id = crypto.randomUUID();
			const newView: DatabaseView = {
				id,
				...view,
				columns: view.columns.length > 0 ? view.columns : state.database.props.columns.map((c) => c.id)
			};

			blockStore.updateBlock(databaseId, {
				views: [...state.database.props.views, newView]
			} as Partial<DatabaseBlockProps>);

			return id;
		},

		/**
		 * Update a view
		 */
		updateView(viewId: string, updates: Partial<Omit<DatabaseView, 'id'>>) {
			const state = get({ subscribe });
			if (!state.database) return;

			const views = state.database.props.views.map((view) =>
				view.id === viewId ? { ...view, ...updates } : view
			);

			blockStore.updateBlock(databaseId, { views } as Partial<DatabaseBlockProps>);
		},

		/**
		 * Delete a view
		 */
		deleteView(viewId: string) {
			const state = get({ subscribe });
			if (!state.database) return;

			// Don't delete last view
			if (state.database.props.views.length <= 1) return;

			const views = state.database.props.views.filter((v) => v.id !== viewId);
			blockStore.updateBlock(databaseId, { views } as Partial<DatabaseBlockProps>);

			// Switch active view if needed
			if (state.activeViewId === viewId) {
				update((s) => ({ ...s, activeViewId: views[0]?.id ?? null }));
			}
		},

		/**
		 * Set active view
		 */
		setActiveView(viewId: string) {
			update((s) => ({ ...s, activeViewId: viewId }));
		},

		/**
		 * Get active view
		 */
		getActiveView(): DatabaseView | null {
			const state = get({ subscribe });
			if (!state.database || !state.activeViewId) return null;
			return state.database.props.views.find((v) => v.id === state.activeViewId) ?? null;
		},

		/**
		 * Update view filter
		 */
		setViewFilter(viewId: string, filter: FilterGroup | undefined) {
			this.updateView(viewId, { filter });
		},

		/**
		 * Update view sorts
		 */
		setViewSorts(viewId: string, sorts: SortConfig[] | undefined) {
			this.updateView(viewId, { sorts });
		},

		/**
		 * Update view visible columns
		 */
		setViewColumns(viewId: string, columns: string[]) {
			this.updateView(viewId, { columns });
		},

		// ========================================================================
		// Select Options Operations
		// ========================================================================

		/**
		 * Add a select option to a column
		 */
		addSelectOption(columnId: string, option: Omit<SelectOption, 'id'>) {
			const column = this.getColumn(columnId);
			if (!column || (column.type !== 'select' && column.type !== 'multi-select')) return;

			const id = crypto.randomUUID();
			const data = column.data as { type: 'select' | 'multi-select'; options: SelectOption[] };
			const options = [...(data?.options ?? []), { id, ...option }];

			this.updateColumn(columnId, {
				data: { type: column.type, options }
			});

			return id;
		},

		/**
		 * Update a select option
		 */
		updateSelectOption(columnId: string, optionId: string, updates: Partial<Omit<SelectOption, 'id'>>) {
			const column = this.getColumn(columnId);
			if (!column || (column.type !== 'select' && column.type !== 'multi-select')) return;

			const data = column.data as { type: 'select' | 'multi-select'; options: SelectOption[] };
			const options = (data?.options ?? []).map((opt) =>
				opt.id === optionId ? { ...opt, ...updates } : opt
			);

			this.updateColumn(columnId, {
				data: { type: column.type, options }
			});
		},

		/**
		 * Delete a select option
		 */
		deleteSelectOption(columnId: string, optionId: string) {
			const column = this.getColumn(columnId);
			if (!column || (column.type !== 'select' && column.type !== 'multi-select')) return;

			const data = column.data as { type: 'select' | 'multi-select'; options: SelectOption[] };
			const options = (data?.options ?? []).filter((opt) => opt.id !== optionId);

			this.updateColumn(columnId, {
				data: { type: column.type, options }
			});

			// Clear cells that use this option
			const state = get({ subscribe });
			if (!state.database) return;

			const cells = { ...state.database.props.cells };
			for (const rowId of Object.keys(cells)) {
				const cell = cells[rowId][columnId];
				if (cell) {
					if (cell.type === 'select' && cell.value === optionId) {
						cells[rowId] = { ...cells[rowId], [columnId]: { type: 'select', value: null } };
					} else if (cell.type === 'multi-select') {
						const values = cell.value.filter((v: string) => v !== optionId);
						cells[rowId] = { ...cells[rowId], [columnId]: { type: 'multi-select', value: values } };
					}
				}
			}
			blockStore.updateBlock(databaseId, { cells } as Partial<DatabaseBlockProps>);
		},

		// ========================================================================
		// Query Operations
		// ========================================================================

		/**
		 * Get filtered and sorted rows for the active view
		 */
		getFilteredRows(): Block<'bos:database-row'>[] {
			const state = get({ subscribe });
			if (!state.database) return [];

			const view = this.getActiveView();
			if (!view) return state.rows;

			let rows = [...state.rows];

			// Apply filter
			if (view.filter) {
				rows = rows.filter((row) => this.matchesFilter(row.id, view.filter!));
			}

			// Apply sort
			if (view.sorts && view.sorts.length > 0) {
				rows = this.sortRows(rows, view.sorts);
			}

			return rows;
		},

		/**
		 * Check if a row matches a filter
		 */
		matchesFilter(rowId: string, filter: FilterGroup): boolean {
			const state = get({ subscribe });
			if (!state.database) return true;

			const cells = state.database.props.cells[rowId] ?? {};

			const results = filter.conditions.map((condition) => {
				if ('type' in condition && (condition.type === 'and' || condition.type === 'or')) {
					return this.matchesFilter(rowId, condition as FilterGroup);
				}

				const cond = condition as { columnId: string; operator: string; value: unknown };
				const cell = cells[cond.columnId];
				return this.evaluateCondition(cell, cond.operator, cond.value);
			});

			if (filter.type === 'and') {
				return results.every(Boolean);
			} else {
				return results.some(Boolean);
			}
		},

		/**
		 * Evaluate a filter condition
		 */
		evaluateCondition(cell: CellValue | undefined, operator: string, value: unknown): boolean {
			if (!cell) {
				return operator === 'is-empty';
			}

			const cellValue = 'value' in cell ? cell.value : null;

			switch (operator) {
				case 'equals':
					return cellValue === value;
				case 'not-equals':
					return cellValue !== value;
				case 'contains':
					return String(cellValue).toLowerCase().includes(String(value).toLowerCase());
				case 'not-contains':
					return !String(cellValue).toLowerCase().includes(String(value).toLowerCase());
				case 'starts-with':
					return String(cellValue).toLowerCase().startsWith(String(value).toLowerCase());
				case 'ends-with':
					return String(cellValue).toLowerCase().endsWith(String(value).toLowerCase());
				case 'is-empty':
					return (
						cellValue === null ||
						cellValue === undefined ||
						cellValue === '' ||
						(Array.isArray(cellValue) && cellValue.length === 0)
					);
				case 'is-not-empty':
					return !(
						cellValue === null ||
						cellValue === undefined ||
						cellValue === '' ||
						(Array.isArray(cellValue) && cellValue.length === 0)
					);
				case 'greater-than':
					return Number(cellValue) > Number(value);
				case 'less-than':
					return Number(cellValue) < Number(value);
				case 'greater-equal':
					return Number(cellValue) >= Number(value);
				case 'less-equal':
					return Number(cellValue) <= Number(value);
				default:
					return true;
			}
		},

		/**
		 * Sort rows by sort config
		 */
		sortRows(rows: Block<'bos:database-row'>[], sorts: SortConfig[]): Block<'bos:database-row'>[] {
			const state = get({ subscribe });
			if (!state.database) return rows;

			return rows.sort((a, b) => {
				for (const sort of sorts) {
					const cellA = state.database!.props.cells[a.id]?.[sort.columnId];
					const cellB = state.database!.props.cells[b.id]?.[sort.columnId];

					const valueA = cellA && 'value' in cellA ? cellA.value : null;
					const valueB = cellB && 'value' in cellB ? cellB.value : null;

					let comparison = 0;

					if (valueA === null && valueB === null) {
						comparison = 0;
					} else if (valueA === null) {
						comparison = 1;
					} else if (valueB === null) {
						comparison = -1;
					} else if (typeof valueA === 'string' && typeof valueB === 'string') {
						comparison = valueA.localeCompare(valueB);
					} else if (typeof valueA === 'number' && typeof valueB === 'number') {
						comparison = valueA - valueB;
					} else {
						comparison = String(valueA).localeCompare(String(valueB));
					}

					if (comparison !== 0) {
						return sort.direction === 'asc' ? comparison : -comparison;
					}
				}
				return 0;
			});
		},

		// ========================================================================
		// Cleanup
		// ========================================================================

		/**
		 * Destroy the store and clean up subscriptions
		 */
		destroy() {
			unsubscribe();
		}
	};
}

// ============================================================================
// Derived Stores
// ============================================================================

/**
 * Create a derived store for a specific column
 */
export function createColumnDerived(
	dbStore: ReturnType<typeof createDatabaseStore>,
	columnId: string
): Readable<ColumnSchema | null> {
	return derived(dbStore, ($db) => $db.database?.props.columns.find((c) => c.id === columnId) ?? null);
}

/**
 * Create a derived store for active view
 */
export function createActiveViewDerived(
	dbStore: ReturnType<typeof createDatabaseStore>
): Readable<DatabaseView | null> {
	return derived(dbStore, ($db) => {
		if (!$db.database || !$db.activeViewId) return null;
		return $db.database.props.views.find((v) => v.id === $db.activeViewId) ?? null;
	});
}

/**
 * Create a derived store for filtered rows
 */
export function createFilteredRowsDerived(
	dbStore: ReturnType<typeof createDatabaseStore>
): Readable<Block<'bos:database-row'>[]> {
	return derived(dbStore, () => dbStore.getFilteredRows());
}

/**
 * Create a derived store for a cell
 */
export function createCellDerived(
	dbStore: ReturnType<typeof createDatabaseStore>,
	rowId: string,
	columnId: string
): Readable<CellValue | null> {
	return derived(dbStore, ($db) => $db.database?.props.cells[rowId]?.[columnId] ?? null);
}

// ============================================================================
// Types Export
// ============================================================================

export type DatabaseStore = ReturnType<typeof createDatabaseStore>;
