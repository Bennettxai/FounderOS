/**
 * Tables Module - Type Definitions
 *
 * NocoDB-inspired data modeling with metadata-driven column types.
 * Supports custom tables, imported data, and integration syncs.
 */

// ============================================================================
// Column Types
// ============================================================================

/**
 * All supported column types organized by implementation phase
 */
export type ColumnType =
	// Phase 1: Essential Types
	| 'text'
	| 'long_text'
	| 'number'
	| 'single_select'
	| 'multi_select'
	| 'date'
	| 'datetime'
	| 'checkbox'
	| 'url'
	| 'email'
	| 'attachment'
	| 'user'
	// Phase 2: Advanced Types
	| 'currency'
	| 'percent'
	| 'rating'
	| 'duration'
	| 'phone'
	| 'lookup'
	| 'rollup'
	| 'formula'
	| 'link_to_record'
	// Phase 3: Special Types
	| 'qr_code'
	| 'barcode'
	| 'button'
	| 'json';

/**
 * Column type metadata for UI rendering
 */
export interface ColumnTypeMeta {
	type: ColumnType;
	label: string;
	icon: string;
	description: string;
	category: 'basic' | 'advanced' | 'computed' | 'special';
}

/**
 * Column type definitions with metadata
 */
export const COLUMN_TYPES: ColumnTypeMeta[] = [
	// Basic Types
	{ type: 'text', label: 'Text', icon: 'Type', description: 'Single line text', category: 'basic' },
	{
		type: 'long_text',
		label: 'Long Text',
		icon: 'AlignLeft',
		description: 'Multi-line text',
		category: 'basic'
	},
	{
		type: 'number',
		label: 'Number',
		icon: 'Hash',
		description: 'Integer or decimal',
		category: 'basic'
	},
	{
		type: 'single_select',
		label: 'Single Select',
		icon: 'CircleDot',
		description: 'Choose one option',
		category: 'basic'
	},
	{
		type: 'multi_select',
		label: 'Multi Select',
		icon: 'CheckSquare',
		description: 'Choose multiple options',
		category: 'basic'
	},
	{ type: 'date', label: 'Date', icon: 'Calendar', description: 'Date only', category: 'basic' },
	{
		type: 'datetime',
		label: 'Date & Time',
		icon: 'Clock',
		description: 'Date with time',
		category: 'basic'
	},
	{
		type: 'checkbox',
		label: 'Checkbox',
		icon: 'CheckSquare',
		description: 'True or false',
		category: 'basic'
	},
	{ type: 'url', label: 'URL', icon: 'Link', description: 'Web link', category: 'basic' },
	{ type: 'email', label: 'Email', icon: 'Mail', description: 'Email address', category: 'basic' },
	{
		type: 'attachment',
		label: 'Attachment',
		icon: 'Paperclip',
		description: 'File uploads',
		category: 'basic'
	},
	{
		type: 'user',
		label: 'User',
		icon: 'User',
		description: 'Team member reference',
		category: 'basic'
	},

	// Advanced Types
	{
		type: 'currency',
		label: 'Currency',
		icon: 'DollarSign',
		description: 'Money values',
		category: 'advanced'
	},
	{
		type: 'percent',
		label: 'Percent',
		icon: 'Percent',
		description: 'Percentage values',
		category: 'advanced'
	},
	{
		type: 'rating',
		label: 'Rating',
		icon: 'Star',
		description: 'Star rating',
		category: 'advanced'
	},
	{
		type: 'duration',
		label: 'Duration',
		icon: 'Timer',
		description: 'Time duration',
		category: 'advanced'
	},
	{
		type: 'phone',
		label: 'Phone',
		icon: 'Phone',
		description: 'Phone number',
		category: 'advanced'
	},

	// Computed Types
	{
		type: 'lookup',
		label: 'Lookup',
		icon: 'Search',
		description: 'Value from linked record',
		category: 'computed'
	},
	{
		type: 'rollup',
		label: 'Rollup',
		icon: 'Calculator',
		description: 'Aggregate linked values',
		category: 'computed'
	},
	{
		type: 'formula',
		label: 'Formula',
		icon: 'Sigma',
		description: 'Calculated field',
		category: 'computed'
	},
	{
		type: 'link_to_record',
		label: 'Link to Record',
		icon: 'Link2',
		description: 'Reference another table',
		category: 'computed'
	},

	// Special Types
	{
		type: 'qr_code',
		label: 'QR Code',
		icon: 'QrCode',
		description: 'Generate QR code',
		category: 'special'
	},
	{
		type: 'barcode',
		label: 'Barcode',
		icon: 'Barcode',
		description: 'Generate barcode',
		category: 'special'
	},
	{
		type: 'button',
		label: 'Button',
		icon: 'MousePointer',
		description: 'Action button',
		category: 'special'
	},
	{ type: 'json', label: 'JSON', icon: 'Braces', description: 'JSON data', category: 'special' }
];

// ============================================================================
// View Types
// ============================================================================

export type ViewType = 'grid' | 'gallery' | 'kanban' | 'calendar' | 'form';

export interface ViewTypeMeta {
	type: ViewType;
	label: string;
	icon: string;
	description: string;
}

export const VIEW_TYPES: ViewTypeMeta[] = [
	{
		type: 'grid',
		label: 'Grid',
		icon: 'Table2',
		description: 'Spreadsheet-like table view'
	},
	{
		type: 'kanban',
		label: 'Kanban',
		icon: 'Columns3',
		description: 'Board with draggable cards'
	},
	{
		type: 'gallery',
		label: 'Gallery',
		icon: 'LayoutGrid',
		description: 'Card grid with images'
	},
	{
		type: 'calendar',
		label: 'Calendar',
		icon: 'Calendar',
		description: 'Events on a calendar'
	},
	{
		type: 'form',
		label: 'Form',
		icon: 'FileInput',
		description: 'Data entry form'
	}
];

// ============================================================================
// Table & Column Interfaces
// ============================================================================

/**
 * Table source - where the data originated
 */
export type TableSource = 'custom' | 'import' | 'integration';

/**
 * Main Table interface
 */
export interface Table {
	id: string;
	name: string;
	description?: string;
	icon?: string;
	source: TableSource;
	source_integration?: string; // e.g., 'google_sheets', 'airtable', 'notion'
	source_external_id?: string; // ID in the external system
	columns: Column[];
	views: TableView[];
	row_count: number;
	is_favorite: boolean;
	created_at: string;
	updated_at: string;
}

/**
 * Table list item (lighter for list views)
 */
export interface TableListItem {
	id: string;
	name: string;
	description?: string;
	icon?: string;
	source: TableSource;
	source_integration?: string;
	row_count: number;
	column_count: number;
	is_favorite: boolean;
	updated_at: string;
	// Optional expanded data for card views
	columns?: Column[];
	views?: TableView[];
}

/**
 * Column definition
 */
export interface Column {
	id: string;
	table_id: string;
	name: string;
	type: ColumnType;
	order: number;
	width?: number;
	is_primary: boolean;
	is_required: boolean;
	is_unique: boolean;
	is_hidden: boolean;
	default_value?: unknown;
	options?: ColumnOptions;
	created_at: string;
	updated_at: string;
}

/**
 * Type-specific column options
 */
export interface ColumnOptions {
	// For single_select / multi_select
	choices?: SelectChoice[];

	// For number / currency / percent
	precision?: number;
	min_value?: number;
	max_value?: number;

	// For currency
	currency_code?: string;
	currency_locale?: string;

	// For rating
	rating_max?: number;
	rating_icon?: 'star' | 'heart' | 'thumb';

	// For duration
	duration_format?: 'h:mm' | 'h:mm:ss' | 'days';

	// For formula
	formula?: string;
	formula_result_type?: ColumnType;

	// For link_to_record
	linked_table_id?: string;
	is_symmetric?: boolean; // Creates reverse link automatically

	// For lookup
	lookup_linked_column_id?: string;
	lookup_target_column_id?: string;

	// For rollup
	rollup_linked_column_id?: string;
	rollup_target_column_id?: string;
	rollup_function?: RollupFunction;

	// For button
	button_label?: string;
	button_action?: 'url' | 'webhook' | 'script';
	button_config?: Record<string, unknown>;

	// For date / datetime
	date_format?: string;
	time_format?: '12h' | '24h';
	include_time?: boolean;
}

/**
 * Select field choice
 */
export interface SelectChoice {
	id: string;
	label: string;
	color: string;
	order: number;
}

/**
 * Rollup aggregate functions
 */
export type RollupFunction =
	| 'count'
	| 'count_values'
	| 'count_unique'
	| 'sum'
	| 'avg'
	| 'min'
	| 'max'
	| 'and'
	| 'or'
	| 'concat';

// ============================================================================
// View Interfaces
// ============================================================================

/**
 * Table View configuration
 */
export interface TableView {
	id: string;
	table_id: string;
	name: string;
	type: ViewType;
	is_default: boolean;
	is_locked: boolean;
	order: number;
	filters: Filter[];
	sorts: Sort[];
	hidden_columns: string[];
	column_order: string[];
	column_widths: Record<string, number>;
	row_height: 'short' | 'medium' | 'tall' | 'extra_tall';

	// View-specific settings
	kanban_column_id?: string;
	kanban_stack_by?: string;
	calendar_date_column_id?: string;
	calendar_end_date_column_id?: string;
	gallery_cover_column_id?: string;
	gallery_cover_fit?: 'cover' | 'contain';
	form_title?: string;
	form_description?: string;
	form_submit_label?: string;
	form_show_branding?: boolean;

	created_at: string;
	updated_at: string;
}

// ============================================================================
// Filter & Sort Interfaces
// ============================================================================

/**
 * Filter operators by category
 */
export type FilterOperator =
	// Equality
	| 'eq'
	| 'neq'
	// Comparison
	| 'gt'
	| 'gte'
	| 'lt'
	| 'lte'
	// Text
	| 'contains'
	| 'not_contains'
	| 'starts_with'
	| 'ends_with'
	// Null checks
	| 'is_empty'
	| 'is_not_empty'
	| 'is_null'
	| 'is_not_null'
	// Array
	| 'in'
	| 'not_in'
	// Date
	| 'is_within'
	| 'is_before'
	| 'is_after'
	| 'is_on_or_before'
	| 'is_on_or_after';

/**
 * Filter definition
 */
export interface Filter {
	id: string;
	column_id: string;
	operator: FilterOperator;
	value: unknown;
	logical_op: 'and' | 'or';
	is_group?: boolean;
	children?: Filter[];
}

/**
 * Sort definition
 */
export interface Sort {
	id: string;
	column_id: string;
	direction: 'asc' | 'desc';
}

// ============================================================================
// Row Interfaces
// ============================================================================

/**
 * Table row with data
 */
export interface Row {
	id: string;
	table_id: string;
	order: number;
	data: Record<string, unknown>;
	created_at: string;
	updated_at: string;
	created_by?: string;
	updated_by?: string;
}

/**
 * Paginated rows response
 */
export interface RowsResponse {
	rows: Row[];
	total: number;
	page: number;
	page_size: number;
	has_more: boolean;
}

// ============================================================================
// API Request/Response Types
// ============================================================================

/**
 * Create table request
 */
export interface CreateTableData {
	name: string;
	description?: string;
	icon?: string;
	source?: TableSource;
	columns?: CreateColumnData[];
}

/**
 * Update table request
 */
export interface UpdateTableData {
	name?: string;
	description?: string;
	icon?: string;
	is_favorite?: boolean;
}

/**
 * Create column request
 */
export interface CreateColumnData {
	name: string;
	type: ColumnType;
	is_primary?: boolean;
	is_required?: boolean;
	is_unique?: boolean;
	default_value?: unknown;
	options?: ColumnOptions;
}

/**
 * Update column request
 */
export interface UpdateColumnData {
	name?: string;
	type?: ColumnType;
	width?: number;
	is_required?: boolean;
	is_unique?: boolean;
	is_hidden?: boolean;
	default_value?: unknown;
	options?: ColumnOptions;
}

/**
 * Create view request
 */
export interface CreateViewData {
	name: string;
	type: ViewType;
	is_default?: boolean;
	filters?: Filter[];
	sorts?: Sort[];
	hidden_columns?: string[];
	// View-specific
	kanban_column_id?: string;
	calendar_date_column_id?: string;
	gallery_cover_column_id?: string;
}

/**
 * Update view request
 */
export interface UpdateViewData {
	name?: string;
	is_default?: boolean;
	is_locked?: boolean;
	filters?: Filter[];
	sorts?: Sort[];
	hidden_columns?: string[];
	column_order?: string[];
	column_widths?: Record<string, number>;
	row_height?: 'short' | 'medium' | 'tall' | 'extra_tall';
	// View-specific
	kanban_column_id?: string;
	calendar_date_column_id?: string;
	gallery_cover_column_id?: string;
}

/**
 * Create row request
 */
export interface CreateRowData {
	data: Record<string, unknown>;
}

/**
 * Update row request
 */
export interface UpdateRowData {
	data: Record<string, unknown>;
}

/**
 * Bulk operations
 */
export interface BulkDeleteRowsData {
	row_ids: string[];
}

export interface BulkUpdateRowsData {
	row_ids: string[];
	data: Record<string, unknown>;
}

/**
 * Get rows params
 */
export interface GetRowsParams {
	view_id?: string;
	page?: number;
	page_size?: number;
	filters?: Filter[];
	sorts?: Sort[];
	search?: string;
}

/**
 * Get tables params
 */
export interface GetTablesParams {
	source?: TableSource;
	source_integration?: string;
	search?: string;
	is_favorite?: boolean;
}

// ============================================================================
// Utility Types
// ============================================================================

/**
 * Cell value type based on column type
 */
export type CellValue =
	| string
	| number
	| boolean
	| Date
	| string[]
	| SelectChoice[]
	| AttachmentValue[]
	| UserValue
	| LinkValue[]
	| null;

/**
 * Attachment cell value
 */
export interface AttachmentValue {
	id: string;
	name: string;
	url: string;
	size: number;
	mime_type: string;
	thumbnail_url?: string;
}

/**
 * User cell value
 */
export interface UserValue {
	id: string;
	name: string;
	email: string;
	avatar_url?: string;
}

/**
 * Link to record cell value
 */
export interface LinkValue {
	id: string;
	display_value: string;
}

// ============================================================================
// Import/Export Types
// ============================================================================

export type ImportFormat = 'csv' | 'xlsx' | 'json';
export type ExportFormat = 'csv' | 'xlsx' | 'json' | 'pdf';

export interface ImportOptions {
	format: ImportFormat;
	has_header_row: boolean;
	column_mapping?: Record<string, string>;
	create_table?: boolean;
	table_name?: string;
}

export interface ExportOptions {
	format: ExportFormat;
	view_id?: string;
	include_hidden_columns?: boolean;
	include_row_ids?: boolean;
}
