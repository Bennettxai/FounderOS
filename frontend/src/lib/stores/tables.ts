/**
 * Tables Store
 *
 * Svelte store for managing tables state following the Clients module pattern.
 */

import { writable, derived, get } from 'svelte/store';
import { api } from '$lib/api/tables';
import type {
	Table,
	TableListItem,
	TableView,
	Row,
	RowsResponse,
	Column,
	Filter,
	Sort,
	CreateTableData,
	UpdateTableData,
	CreateColumnData,
	UpdateColumnData,
	CreateViewData,
	UpdateViewData,
	GetRowsParams,
	GetTablesParams,
	TableSource,
	ViewType
} from '$lib/api/tables/types';

// ============================================================================
// Types
// ============================================================================

export type TableViewMode = 'list' | 'grid' | 'card';

interface TableFilters {
	source: TableSource | null;
	source_integration: string | null;
	search: string;
	is_favorite: boolean | null;
}

interface TablesState {
	// Data
	tables: TableListItem[];
	currentTable: Table | null;
	currentView: TableView | null;
	rows: Row[];
	rowsTotal: number;
	rowsPage: number;
	rowsPageSize: number;
	rowsHasMore: boolean;

	// UI State
	loading: boolean;
	loadingRows: boolean;
	saving: boolean;
	error: string | null;
	filters: TableFilters;
	viewMode: TableViewMode;

	// Selection
	selectedRowIds: Set<string>;
	editingCell: { rowId: string; columnId: string } | null;
}

const initialState: TablesState = {
	tables: [],
	currentTable: null,
	currentView: null,
	rows: [],
	rowsTotal: 0,
	rowsPage: 1,
	rowsPageSize: 50,
	rowsHasMore: false,

	loading: false,
	loadingRows: false,
	saving: false,
	error: null,
	filters: {
		source: null,
		source_integration: null,
		search: '',
		is_favorite: null
	},
	viewMode: 'list',

	selectedRowIds: new Set(),
	editingCell: null
};

// ============================================================================
// Store Factory
// ============================================================================

function createTablesStore() {
	const { subscribe, set, update } = writable<TablesState>(initialState);

	return {
		subscribe,

		// ========================================================================
		// Tables
		// ========================================================================

		/**
		 * Load all tables
		 */
		async loadTables(params?: GetTablesParams) {
			update((s) => ({ ...s, loading: true, error: null }));

			try {
				const currentState = get({ subscribe });
				const tables = await api.getTables({
					source: params?.source ?? currentState.filters.source ?? undefined,
					source_integration:
						params?.source_integration ?? currentState.filters.source_integration ?? undefined,
					search: params?.search ?? currentState.filters.search ?? undefined,
					is_favorite: params?.is_favorite ?? currentState.filters.is_favorite ?? undefined
				});

				update((s) => ({ ...s, tables, loading: false }));
				return tables;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load tables';
				update((s) => ({ ...s, loading: false, error: message }));
				console.error('Failed to load tables:', error);
				return [];
			}
		},

		/**
		 * Load a single table with columns and views
		 */
		async loadTable(id: string) {
			update((s) => ({ ...s, loading: true, error: null }));

			try {
				const table = await api.getTable(id);

				// Ensure columns and views are arrays
				const safeTable = {
					...table,
					columns: table.columns ?? [],
					views: table.views ?? []
				};

				// Set default view if none selected
				const defaultView = safeTable.views.find((v) => v.is_default) || safeTable.views[0] || null;

				update((s) => ({
					...s,
					currentTable: safeTable,
					currentView: defaultView,
					loading: false
				}));

				return safeTable;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load table';
				update((s) => ({ ...s, loading: false, error: message }));
				console.error('Failed to load table:', error);
				return null;
			}
		},

		/**
		 * Create a new table
		 */
		async createTable(data: CreateTableData) {
			update((s) => ({ ...s, saving: true, error: null }));

			try {
				const table = await api.createTable(data);

				update((s) => ({
					...s,
					tables: [
						{
							id: table.id,
							name: table.name,
							description: table.description,
							icon: table.icon,
							source: table.source,
							source_integration: table.source_integration,
							row_count: table.row_count ?? 0,
							column_count: table.columns?.length ?? 0,
							is_favorite: table.is_favorite ?? false,
							updated_at: table.updated_at
						},
						...s.tables
					],
					saving: false
				}));

				return table;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to create table';
				update((s) => ({ ...s, saving: false, error: message }));
				console.error('Failed to create table:', error);
				throw error;
			}
		},

		/**
		 * Update a table
		 */
		async updateTable(id: string, data: UpdateTableData) {
			update((s) => ({ ...s, saving: true, error: null }));

			try {
				const table = await api.updateTable(id, data);

				update((s) => ({
					...s,
					tables: s.tables.map((t) =>
						t.id === id
							? {
									...t,
									name: table.name,
									description: table.description,
									icon: table.icon,
									is_favorite: table.is_favorite,
									updated_at: table.updated_at
								}
							: t
					),
					currentTable: s.currentTable?.id === id ? table : s.currentTable,
					saving: false
				}));

				return table;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to update table';
				update((s) => ({ ...s, saving: false, error: message }));
				console.error('Failed to update table:', error);
				throw error;
			}
		},

		/**
		 * Delete a table
		 */
		async deleteTable(id: string) {
			try {
				await api.deleteTable(id);

				update((s) => ({
					...s,
					tables: s.tables.filter((t) => t.id !== id),
					currentTable: s.currentTable?.id === id ? null : s.currentTable
				}));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to delete table';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to delete table:', error);
				throw error;
			}
		},

		/**
		 * Toggle favorite status
		 */
		async toggleFavorite(id: string) {
			try {
				const table = await api.toggleTableFavorite(id);

				update((s) => ({
					...s,
					tables: s.tables.map((t) =>
						t.id === id ? { ...t, is_favorite: table.is_favorite } : t
					),
					currentTable:
						s.currentTable?.id === id
							? { ...s.currentTable, is_favorite: table.is_favorite }
							: s.currentTable
				}));
			} catch (error) {
				console.error('Failed to toggle favorite:', error);
			}
		},

		// ========================================================================
		// Columns
		// ========================================================================

		/**
		 * Add a column to the current table
		 */
		async addColumn(data: CreateColumnData) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return null;

			try {
				const column = await api.addColumn(currentState.currentTable.id, data);

				update((s) => ({
					...s,
					currentTable: s.currentTable
						? {
								...s.currentTable,
								columns: [...s.currentTable.columns, column]
							}
						: null
				}));

				return column;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to add column';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to add column:', error);
				throw error;
			}
		},

		/**
		 * Update a column
		 */
		async updateColumn(columnId: string, data: UpdateColumnData) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return null;

			try {
				const column = await api.updateColumn(currentState.currentTable.id, columnId, data);

				update((s) => ({
					...s,
					currentTable: s.currentTable
						? {
								...s.currentTable,
								columns: s.currentTable.columns.map((c) => (c.id === columnId ? column : c))
							}
						: null
				}));

				return column;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to update column';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to update column:', error);
				throw error;
			}
		},

		/**
		 * Delete a column
		 */
		async deleteColumn(columnId: string) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return;

			try {
				await api.deleteColumn(currentState.currentTable.id, columnId);

				update((s) => ({
					...s,
					currentTable: s.currentTable
						? {
								...s.currentTable,
								columns: s.currentTable.columns.filter((c) => c.id !== columnId)
							}
						: null
				}));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to delete column';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to delete column:', error);
				throw error;
			}
		},

		/**
		 * Reorder columns
		 */
		async reorderColumns(columnIds: string[]) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return;

			try {
				const columns = await api.reorderColumns(currentState.currentTable.id, columnIds);

				update((s) => ({
					...s,
					currentTable: s.currentTable
						? {
								...s.currentTable,
								columns
							}
						: null
				}));
			} catch (error) {
				console.error('Failed to reorder columns:', error);
			}
		},

		// ========================================================================
		// Views
		// ========================================================================

		/**
		 * Set the current view
		 */
		setCurrentView(viewId: string) {
			update((s) => {
				const view = s.currentTable?.views.find((v) => v.id === viewId) || null;
				return { ...s, currentView: view };
			});
		},

		/**
		 * Create a new view
		 */
		async createView(data: CreateViewData) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return null;

			try {
				const view = await api.createView(currentState.currentTable.id, data);

				update((s) => ({
					...s,
					currentTable: s.currentTable
						? {
								...s.currentTable,
								views: [...s.currentTable.views, view]
							}
						: null,
					currentView: view
				}));

				return view;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to create view';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to create view:', error);
				throw error;
			}
		},

		/**
		 * Update a view
		 */
		async updateView(viewId: string, data: UpdateViewData) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return null;

			try {
				const view = await api.updateView(currentState.currentTable.id, viewId, data);

				update((s) => ({
					...s,
					currentTable: s.currentTable
						? {
								...s.currentTable,
								views: s.currentTable.views.map((v) => (v.id === viewId ? view : v))
							}
						: null,
					currentView: s.currentView?.id === viewId ? view : s.currentView
				}));

				return view;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to update view';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to update view:', error);
				throw error;
			}
		},

		/**
		 * Delete a view
		 */
		async deleteView(viewId: string) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return;

			try {
				await api.deleteView(currentState.currentTable.id, viewId);

				update((s) => {
					const views = s.currentTable?.views.filter((v) => v.id !== viewId) || [];
					const newCurrentView =
						s.currentView?.id === viewId ? views.find((v) => v.is_default) || views[0] || null : s.currentView;

					return {
						...s,
						currentTable: s.currentTable
							? {
									...s.currentTable,
									views
								}
							: null,
						currentView: newCurrentView
					};
				});
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to delete view';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to delete view:', error);
				throw error;
			}
		},

		// ========================================================================
		// Rows
		// ========================================================================

		/**
		 * Load rows for the current table/view
		 */
		async loadRows(params?: GetRowsParams) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return;

			update((s) => ({ ...s, loadingRows: true }));

			try {
				const response = await api.getRows(currentState.currentTable.id, {
					view_id: params?.view_id ?? currentState.currentView?.id,
					page: params?.page ?? currentState.rowsPage,
					page_size: params?.page_size ?? currentState.rowsPageSize,
					filters: params?.filters ?? currentState.currentView?.filters,
					sorts: params?.sorts ?? currentState.currentView?.sorts,
					search: params?.search
				});

				update((s) => ({
					...s,
					rows: response.rows,
					rowsTotal: response.total,
					rowsPage: response.page,
					rowsHasMore: response.has_more,
					loadingRows: false
				}));

				return response;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to load rows';
				update((s) => ({ ...s, loadingRows: false, error: message }));
				console.error('Failed to load rows:', error);
				return null;
			}
		},

		/**
		 * Load more rows (pagination)
		 */
		async loadMoreRows() {
			const currentState = get({ subscribe });
			if (!currentState.currentTable || !currentState.rowsHasMore || currentState.loadingRows)
				return;

			update((s) => ({ ...s, loadingRows: true }));

			try {
				const response = await api.getRows(currentState.currentTable.id, {
					view_id: currentState.currentView?.id,
					page: currentState.rowsPage + 1,
					page_size: currentState.rowsPageSize,
					filters: currentState.currentView?.filters,
					sorts: currentState.currentView?.sorts
				});

				update((s) => ({
					...s,
					rows: [...s.rows, ...response.rows],
					rowsPage: response.page,
					rowsHasMore: response.has_more,
					loadingRows: false
				}));
			} catch (error) {
				update((s) => ({ ...s, loadingRows: false }));
				console.error('Failed to load more rows:', error);
			}
		},

		/**
		 * Create a new row
		 */
		async createRow(data: Record<string, unknown>) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return null;

			try {
				const row = await api.createRow(currentState.currentTable.id, { data });

				update((s) => ({
					...s,
					rows: [...s.rows, row],
					rowsTotal: s.rowsTotal + 1,
					tables: s.tables.map((t) =>
						t.id === currentState.currentTable?.id ? { ...t, row_count: t.row_count + 1 } : t
					),
					currentTable: s.currentTable
						? { ...s.currentTable, row_count: s.currentTable.row_count + 1 }
						: null
				}));

				return row;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to create row';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to create row:', error);
				throw error;
			}
		},

		/**
		 * Update a row
		 */
		async updateRow(rowId: string, data: Record<string, unknown>) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return null;

			try {
				const row = await api.updateRow(currentState.currentTable.id, rowId, { data });

				update((s) => ({
					...s,
					rows: s.rows.map((r) => (r.id === rowId ? row : r))
				}));

				return row;
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to update row';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to update row:', error);
				throw error;
			}
		},

		/**
		 * Update a single cell value
		 */
		async updateCell(rowId: string, columnId: string, value: unknown) {
			const currentState = get({ subscribe });
			const row = currentState.rows.find((r) => r.id === rowId);
			if (!row) return;

			// Optimistic update
			update((s) => ({
				...s,
				rows: s.rows.map((r) =>
					r.id === rowId ? { ...r, data: { ...r.data, [columnId]: value } } : r
				)
			}));

			try {
				await this.updateRow(rowId, { ...row.data, [columnId]: value });
			} catch (error) {
				// Revert on failure
				update((s) => ({
					...s,
					rows: s.rows.map((r) => (r.id === rowId ? row : r))
				}));
				throw error;
			}
		},

		/**
		 * Delete a row
		 */
		async deleteRow(rowId: string) {
			const currentState = get({ subscribe });
			if (!currentState.currentTable) return;

			try {
				await api.deleteRow(currentState.currentTable.id, rowId);

				update((s) => ({
					...s,
					rows: s.rows.filter((r) => r.id !== rowId),
					rowsTotal: s.rowsTotal - 1,
					selectedRowIds: new Set([...s.selectedRowIds].filter((id) => id !== rowId)),
					tables: s.tables.map((t) =>
						t.id === currentState.currentTable?.id ? { ...t, row_count: t.row_count - 1 } : t
					),
					currentTable: s.currentTable
						? { ...s.currentTable, row_count: s.currentTable.row_count - 1 }
						: null
				}));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to delete row';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to delete row:', error);
				throw error;
			}
		},

		/**
		 * Bulk delete selected rows
		 */
		async deleteSelectedRows() {
			const currentState = get({ subscribe });
			if (!currentState.currentTable || currentState.selectedRowIds.size === 0) return;

			const rowIds = [...currentState.selectedRowIds];

			try {
				await api.bulkDeleteRows(currentState.currentTable.id, { row_ids: rowIds });

				update((s) => ({
					...s,
					rows: s.rows.filter((r) => !rowIds.includes(r.id)),
					rowsTotal: s.rowsTotal - rowIds.length,
					selectedRowIds: new Set(),
					tables: s.tables.map((t) =>
						t.id === currentState.currentTable?.id
							? { ...t, row_count: t.row_count - rowIds.length }
							: t
					),
					currentTable: s.currentTable
						? { ...s.currentTable, row_count: s.currentTable.row_count - rowIds.length }
						: null
				}));
			} catch (error) {
				const message = error instanceof Error ? error.message : 'Failed to delete rows';
				update((s) => ({ ...s, error: message }));
				console.error('Failed to delete rows:', error);
				throw error;
			}
		},

		// ========================================================================
		// Selection & UI
		// ========================================================================

		/**
		 * Toggle row selection
		 */
		toggleRowSelection(rowId: string) {
			update((s) => {
				const newSelected = new Set(s.selectedRowIds);
				if (newSelected.has(rowId)) {
					newSelected.delete(rowId);
				} else {
					newSelected.add(rowId);
				}
				return { ...s, selectedRowIds: newSelected };
			});
		},

		/**
		 * Select all rows
		 */
		selectAllRows() {
			update((s) => ({
				...s,
				selectedRowIds: new Set(s.rows.map((r) => r.id))
			}));
		},

		/**
		 * Clear row selection
		 */
		clearSelection() {
			update((s) => ({ ...s, selectedRowIds: new Set() }));
		},

		/**
		 * Set editing cell
		 */
		setEditingCell(rowId: string | null, columnId: string | null) {
			update((s) => ({
				...s,
				editingCell: rowId && columnId ? { rowId, columnId } : null
			}));
		},

		/**
		 * Set view mode
		 */
		setViewMode(mode: TableViewMode) {
			update((s) => ({ ...s, viewMode: mode }));
		},

		/**
		 * Set filters
		 */
		setFilters(filters: Partial<TableFilters>) {
			update((s) => ({
				...s,
				filters: { ...s.filters, ...filters }
			}));
		},

		/**
		 * Clear error
		 */
		clearError() {
			update((s) => ({ ...s, error: null }));
		},

		/**
		 * Reset store to initial state
		 */
		reset() {
			set(initialState);
		}
	};
}

// ============================================================================
// Export Store Instance
// ============================================================================

export const tables = createTablesStore();

// ============================================================================
// Derived Stores
// ============================================================================

/**
 * Filtered tables based on search and filters
 */
export const filteredTables = derived(tables, ($tables) => {
	let result = $tables.tables;

	if ($tables.filters.search) {
		const query = $tables.filters.search.toLowerCase();
		result = result.filter(
			(t) =>
				t.name.toLowerCase().includes(query) ||
				t.description?.toLowerCase().includes(query)
		);
	}

	return result;
});

/**
 * Favorite tables
 */
export const favoriteTables = derived(tables, ($tables) =>
	$tables.tables.filter((t) => t.is_favorite)
);

/**
 * Columns for current table
 */
export const currentColumns = derived(tables, ($tables) => $tables.currentTable?.columns || []);

/**
 * Views for current table
 */
export const currentViews = derived(tables, ($tables) => $tables.currentTable?.views || []);

/**
 * Visible columns based on current view settings
 */
export const visibleColumns = derived(tables, ($tables) => {
	const tableColumns = $tables.currentTable?.columns ?? [];

	if (!$tables.currentTable || !$tables.currentView) {
		return tableColumns;
	}

	const hiddenColumns = $tables.currentView.hidden_columns ?? [];
	const columnOrder = $tables.currentView.column_order ?? [];
	const hiddenSet = new Set(hiddenColumns);

	let columns = tableColumns.filter((c) => !hiddenSet.has(c.id));

	if (columnOrder.length > 0) {
		const orderMap = new Map(columnOrder.map((id, idx) => [id, idx]));
		columns = columns.sort((a, b) => {
			const aOrder = orderMap.get(a.id) ?? a.order;
			const bOrder = orderMap.get(b.id) ?? b.order;
			return aOrder - bOrder;
		});
	}

	return columns;
});

/**
 * Selected row count
 */
export const selectedRowCount = derived(tables, ($tables) => $tables.selectedRowIds.size);
