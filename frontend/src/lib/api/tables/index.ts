/**
 * Tables Module - API Barrel Export
 */

// Export all types
export * from './types';

// Export all API functions
export * from './tables';

// Namespace export for cleaner imports
import * as tablesApi from './tables';

export const api = {
	// Tables
	getTables: tablesApi.getTables,
	getTable: tablesApi.getTable,
	createTable: tablesApi.createTable,
	updateTable: tablesApi.updateTable,
	deleteTable: tablesApi.deleteTable,
	duplicateTable: tablesApi.duplicateTable,
	toggleTableFavorite: tablesApi.toggleTableFavorite,

	// Columns
	addColumn: tablesApi.addColumn,
	updateColumn: tablesApi.updateColumn,
	deleteColumn: tablesApi.deleteColumn,
	reorderColumns: tablesApi.reorderColumns,
	duplicateColumn: tablesApi.duplicateColumn,

	// Views
	getViews: tablesApi.getViews,
	getView: tablesApi.getView,
	createView: tablesApi.createView,
	updateView: tablesApi.updateView,
	deleteView: tablesApi.deleteView,
	duplicateView: tablesApi.duplicateView,
	reorderViews: tablesApi.reorderViews,

	// Rows
	getRows: tablesApi.getRows,
	getRow: tablesApi.getRow,
	createRow: tablesApi.createRow,
	updateRow: tablesApi.updateRow,
	deleteRow: tablesApi.deleteRow,
	bulkDeleteRows: tablesApi.bulkDeleteRows,
	bulkUpdateRows: tablesApi.bulkUpdateRows,
	reorderRows: tablesApi.reorderRows,

	// Import/Export
	exportTable: tablesApi.exportTable,
	importData: tablesApi.importData,

	// Search & Aggregate
	searchTables: tablesApi.searchTables,
	getColumnAggregates: tablesApi.getColumnAggregates,

	// Linked Records
	getLinkedRecords: tablesApi.getLinkedRecords,
	linkRecords: tablesApi.linkRecords,
	unlinkRecords: tablesApi.unlinkRecords,

	// Attachments
	uploadAttachment: tablesApi.uploadAttachment,
	deleteAttachment: tablesApi.deleteAttachment
};

export default api;
