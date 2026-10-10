/**
 * Tables Module - API Functions
 *
 * All API calls for tables, columns, views, and rows.
 */

import { request } from '../base';
import type {
	Table,
	TableListItem,
	Column,
	TableView,
	Row,
	RowsResponse,
	CreateTableData,
	UpdateTableData,
	CreateColumnData,
	UpdateColumnData,
	CreateViewData,
	UpdateViewData,
	CreateRowData,
	UpdateRowData,
	GetRowsParams,
	GetTablesParams,
	BulkDeleteRowsData,
	BulkUpdateRowsData,
	ExportOptions,
	ImportOptions
} from './types';

// ============================================================================
// Tables CRUD
// ============================================================================

/**
 * Get all tables with optional filters
 */
export async function getTables(
	params?: GetTablesParams,
	workspaceId?: string
): Promise<TableListItem[]> {
	const searchParams = new URLSearchParams();

	if (params?.source) searchParams.set('source', params.source);
	if (params?.source_integration) searchParams.set('source_integration', params.source_integration);
	if (params?.search) searchParams.set('search', params.search);
	if (params?.is_favorite !== undefined) searchParams.set('is_favorite', String(params.is_favorite));
	if (workspaceId) searchParams.set('workspace_id', workspaceId);

	const query = searchParams.toString();
	return request<TableListItem[]>(`/tables${query ? `?${query}` : ''}`, {
		headers: workspaceId ? { 'X-Workspace-ID': workspaceId } : undefined
	});
}

/**
 * Get a single table with all columns and views
 */
export async function getTable(id: string, workspaceId?: string): Promise<Table> {
	const headers = workspaceId ? { 'X-Workspace-ID': workspaceId } : undefined;
	const table = await request<Table>(`/tables/${id}`, { headers, skipCache: true });

	// Older desktop backends return table metadata without the nested fields and
	// views even though the detail contract includes them. Resolve those
	// resources explicitly so row data can always be translated from field IDs.
	const rawColumns = Array.isArray(table.columns)
		? table.columns
		: await request<Array<Record<string, unknown>>>(`/tables/${id}/columns`, {
			headers,
			skipCache: true
		});
	const views = Array.isArray(table.views)
		? table.views
		: await request<TableView[]>(`/tables/${id}/views`, { headers, skipCache: true });

	const columns = rawColumns.map((raw, index) => {
		const column = raw as unknown as Record<string, unknown>;
		return {
			...column,
			id: String(column.id),
			table_id: String(column.table_id ?? id),
			name: String(column.name ?? `Field ${index + 1}`),
			type: String(column.type ?? 'text'),
			order: Number(column.order ?? column.position ?? index),
			is_primary: Boolean(column.is_primary ?? index === 0),
			is_required: Boolean(column.is_required ?? column.required),
			is_unique: Boolean(column.is_unique ?? column.unique),
			is_hidden: Boolean(column.is_hidden ?? column.hidden),
			created_at: String(column.created_at ?? table.created_at),
			updated_at: String(column.updated_at ?? table.updated_at)
		} as Column;
	});

	return { ...table, columns, views };
}

/**
 * Create a new table
 */
export async function createTable(data: CreateTableData): Promise<Table> {
	return request<Table>('/tables', {
		method: 'POST',
		body: data
	});
}

/**
 * Update a table
 */
export async function updateTable(id: string, data: UpdateTableData): Promise<Table> {
	return request<Table>(`/tables/${id}`, {
		method: 'PUT',
		body: data
	});
}

/**
 * Delete a table
 */
export async function deleteTable(id: string): Promise<void> {
	await request(`/tables/${id}`, {
		method: 'DELETE'
	});
}

/**
 * Duplicate a table
 */
export async function duplicateTable(
	id: string,
	options?: { include_data?: boolean; new_name?: string }
): Promise<Table> {
	return request<Table>(`/tables/${id}/duplicate`, {
		method: 'POST',
		body: options
	});
}

/**
 * Toggle table favorite status
 */
export async function toggleTableFavorite(id: string): Promise<Table> {
	return request<Table>(`/tables/${id}/favorite`, {
		method: 'POST'
	});
}

// ============================================================================
// Columns CRUD
// ============================================================================

/**
 * Add a column to a table
 */
export async function addColumn(tableId: string, data: CreateColumnData): Promise<Column> {
	return request<Column>(`/tables/${tableId}/columns`, {
		method: 'POST',
		body: data
	});
}

/**
 * Update a column
 */
export async function updateColumn(
	tableId: string,
	columnId: string,
	data: UpdateColumnData
): Promise<Column> {
	return request<Column>(`/tables/${tableId}/columns/${columnId}`, {
		method: 'PUT',
		body: data
	});
}

/**
 * Delete a column
 */
export async function deleteColumn(tableId: string, columnId: string): Promise<void> {
	await request(`/tables/${tableId}/columns/${columnId}`, {
		method: 'DELETE'
	});
}

/**
 * Reorder columns
 */
export async function reorderColumns(tableId: string, columnIds: string[]): Promise<Column[]> {
	return request<Column[]>(`/tables/${tableId}/columns/reorder`, {
		method: 'POST',
		body: { column_ids: columnIds }
	});
}

/**
 * Duplicate a column
 */
export async function duplicateColumn(tableId: string, columnId: string): Promise<Column> {
	return request<Column>(`/tables/${tableId}/columns/${columnId}/duplicate`, {
		method: 'POST'
	});
}

// ============================================================================
// Views CRUD
// ============================================================================

/**
 * Get all views for a table
 */
export async function getViews(tableId: string): Promise<TableView[]> {
	return request<TableView[]>(`/tables/${tableId}/views`);
}

/**
 * Get a single view
 */
export async function getView(tableId: string, viewId: string): Promise<TableView> {
	return request<TableView>(`/tables/${tableId}/views/${viewId}`);
}

/**
 * Create a new view
 */
export async function createView(tableId: string, data: CreateViewData): Promise<TableView> {
	return request<TableView>(`/tables/${tableId}/views`, {
		method: 'POST',
		body: data
	});
}

/**
 * Update a view
 */
export async function updateView(
	tableId: string,
	viewId: string,
	data: UpdateViewData
): Promise<TableView> {
	return request<TableView>(`/tables/${tableId}/views/${viewId}`, {
		method: 'PUT',
		body: data
	});
}

/**
 * Delete a view
 */
export async function deleteView(tableId: string, viewId: string): Promise<void> {
	await request(`/tables/${tableId}/views/${viewId}`, {
		method: 'DELETE'
	});
}

/**
 * Duplicate a view
 */
export async function duplicateView(tableId: string, viewId: string): Promise<TableView> {
	return request<TableView>(`/tables/${tableId}/views/${viewId}/duplicate`, {
		method: 'POST'
	});
}

/**
 * Reorder views
 */
export async function reorderViews(tableId: string, viewIds: string[]): Promise<TableView[]> {
	return request<TableView[]>(`/tables/${tableId}/views/reorder`, {
		method: 'POST',
		body: { view_ids: viewIds }
	});
}

// ============================================================================
// Rows CRUD
// ============================================================================

/**
 * Get rows with pagination, filters, and sorts
 */
export async function getRows(
	tableId: string,
	params?: GetRowsParams,
	workspaceId?: string
): Promise<RowsResponse> {
	const searchParams = new URLSearchParams();

	if (params?.view_id) searchParams.set('view_id', params.view_id);
	if (params?.page !== undefined) searchParams.set('page', String(params.page));
	if (params?.page_size !== undefined) searchParams.set('page_size', String(params.page_size));
	if (params?.search) searchParams.set('search', params.search);
	if (params?.filters) searchParams.set('filters', JSON.stringify(params.filters));
	if (params?.sorts) searchParams.set('sorts', JSON.stringify(params.sorts));

	const query = searchParams.toString();
	return request<RowsResponse>(`/tables/${tableId}/rows${query ? `?${query}` : ''}`, {
		headers: workspaceId ? { 'X-Workspace-ID': workspaceId } : undefined,
		skipCache: true
	});
}

/**
 * Get a single row
 */
export async function getRow(tableId: string, rowId: string): Promise<Row> {
	return request<Row>(`/tables/${tableId}/rows/${rowId}`);
}

/**
 * Create a new row
 */
export async function createRow(tableId: string, data: CreateRowData): Promise<Row> {
	return request<Row>(`/tables/${tableId}/rows`, {
		method: 'POST',
		body: data
	});
}

/**
 * Update a row
 */
export async function updateRow(tableId: string, rowId: string, data: UpdateRowData): Promise<Row> {
	return request<Row>(`/tables/${tableId}/rows/${rowId}`, {
		method: 'PUT',
		body: data
	});
}

/**
 * Delete a row
 */
export async function deleteRow(tableId: string, rowId: string): Promise<void> {
	await request(`/tables/${tableId}/rows/${rowId}`, {
		method: 'DELETE'
	});
}

/**
 * Bulk delete rows
 */
export async function bulkDeleteRows(tableId: string, data: BulkDeleteRowsData): Promise<void> {
	await request(`/tables/${tableId}/rows/bulk-delete`, {
		method: 'POST',
		body: data
	});
}

/**
 * Bulk update rows
 */
export async function bulkUpdateRows(tableId: string, data: BulkUpdateRowsData): Promise<Row[]> {
	return request<Row[]>(`/tables/${tableId}/rows/bulk-update`, {
		method: 'POST',
		body: data
	});
}

/**
 * Reorder rows
 */
export async function reorderRows(
	tableId: string,
	rowId: string,
	newOrder: number
): Promise<void> {
	await request(`/tables/${tableId}/rows/${rowId}/reorder`, {
		method: 'POST',
		body: { order: newOrder }
	});
}

// ============================================================================
// Import / Export
// ============================================================================

/**
 * Export table data
 */
export async function exportTable(
	tableId: string,
	options: ExportOptions
): Promise<Blob> {
	const searchParams = new URLSearchParams();
	searchParams.set('format', options.format);
	if (options.view_id) searchParams.set('view_id', options.view_id);
	if (options.include_hidden_columns !== undefined) {
		searchParams.set('include_hidden_columns', String(options.include_hidden_columns));
	}
	if (options.include_row_ids !== undefined) {
		searchParams.set('include_row_ids', String(options.include_row_ids));
	}

	const query = searchParams.toString();
	const response = await fetch(`/api/tables/${tableId}/export?${query}`, {
		method: 'GET',
		credentials: 'include'
	});

	if (!response.ok) {
		throw new Error('Export failed');
	}

	return response.blob();
}

/**
 * Import data into a table (or create new table)
 */
export async function importData(
	file: File,
	options: ImportOptions,
	tableId?: string
): Promise<{ table_id: string; rows_imported: number }> {
	const formData = new FormData();
	formData.append('file', file);
	formData.append('options', JSON.stringify(options));

	const endpoint = tableId ? `/tables/${tableId}/import` : '/tables/import';

	const response = await fetch(`/api${endpoint}`, {
		method: 'POST',
		credentials: 'include',
		body: formData
	});

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Import failed');
	}

	return response.json();
}

// ============================================================================
// Search & Aggregate
// ============================================================================

/**
 * Search across all tables
 */
export async function searchTables(query: string): Promise<TableListItem[]> {
	return request<TableListItem[]>(`/tables/search?q=${encodeURIComponent(query)}`);
}

/**
 * Get aggregated values for a column
 */
export async function getColumnAggregates(
	tableId: string,
	columnId: string,
	viewId?: string
): Promise<{
	count: number;
	unique_count?: number;
	sum?: number;
	avg?: number;
	min?: unknown;
	max?: unknown;
}> {
	const params = viewId ? `?view_id=${viewId}` : '';
	return request(`/tables/${tableId}/columns/${columnId}/aggregates${params}`);
}

// ============================================================================
// Linked Records
// ============================================================================

/**
 * Get linked records for a link_to_record column
 */
export async function getLinkedRecords(
	tableId: string,
	rowId: string,
	columnId: string
): Promise<Row[]> {
	return request<Row[]>(`/tables/${tableId}/rows/${rowId}/links/${columnId}`);
}

/**
 * Link records
 */
export async function linkRecords(
	tableId: string,
	rowId: string,
	columnId: string,
	linkedRowIds: string[]
): Promise<void> {
	await request(`/tables/${tableId}/rows/${rowId}/links/${columnId}`, {
		method: 'POST',
		body: { linked_row_ids: linkedRowIds }
	});
}

/**
 * Unlink records
 */
export async function unlinkRecords(
	tableId: string,
	rowId: string,
	columnId: string,
	linkedRowIds: string[]
): Promise<void> {
	await request(`/tables/${tableId}/rows/${rowId}/links/${columnId}`, {
		method: 'DELETE',
		body: { linked_row_ids: linkedRowIds }
	});
}

// ============================================================================
// Attachments
// ============================================================================

/**
 * Upload attachment to a cell
 */
export async function uploadAttachment(
	tableId: string,
	rowId: string,
	columnId: string,
	file: File
): Promise<{ id: string; url: string; name: string; size: number; mime_type: string }> {
	const formData = new FormData();
	formData.append('file', file);

	const response = await fetch(
		`/api/tables/${tableId}/rows/${rowId}/columns/${columnId}/attachments`,
		{
			method: 'POST',
			credentials: 'include',
			body: formData
		}
	);

	if (!response.ok) {
		const error = await response.json();
		throw new Error(error.message || 'Upload failed');
	}

	return response.json();
}

/**
 * Delete attachment from a cell
 */
export async function deleteAttachment(
	tableId: string,
	rowId: string,
	columnId: string,
	attachmentId: string
): Promise<void> {
	await request(
		`/tables/${tableId}/rows/${rowId}/columns/${columnId}/attachments/${attachmentId}`,
		{ method: 'DELETE' }
	);
}
