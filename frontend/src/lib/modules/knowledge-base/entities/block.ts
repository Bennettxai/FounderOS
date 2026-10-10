/**
 * Block Model - Core Data Structure
 *
 * BusinessOS Block System inspired by AFFiNE's architecture.
 * Everything is a block: documents, paragraphs, headings, database rows, etc.
 *
 * Key concepts:
 * - Flavour: Block type identifier (e.g., 'bos:paragraph', 'bos:database')
 * - Props: Block-specific properties defined by schema
 * - Children: Nested child blocks (Y.Array backed)
 * - Text: Rich text content (Y.Text backed for CRDT)
 */

import * as Y from 'yjs';

// ============================================================================
// Block Flavours (Type Identifiers)
// ============================================================================

/**
 * Block flavours define the type of block.
 * Format: 'namespace:type' (e.g., 'bos:paragraph')
 */
export type BlockFlavour =
	// Document-level blocks
	| 'bos:page'
	| 'bos:database'
	// Content blocks
	| 'bos:paragraph'
	| 'bos:heading'
	| 'bos:list'
	| 'bos:code'
	| 'bos:quote'
	| 'bos:callout'
	| 'bos:divider'
	| 'bos:image'
	| 'bos:bookmark'
	| 'bos:embed'
	// Database blocks
	| 'bos:database-row'
	// Layout blocks
	| 'bos:column-layout'
	| 'bos:column'
	// Extension blocks
	| 'bos:synced-block'
	| 'bos:link-to-page';

// ============================================================================
// Block System Properties (Common to all blocks)
// ============================================================================

/**
 * System properties every block must have.
 * These are managed by the block system, not user-editable.
 */
export interface BlockSysProps {
	/** Unique block identifier (UUID) */
	id: string;
	/** Block type (flavour) */
	flavour: BlockFlavour;
	/** Parent block ID (null for root/page blocks) */
	parent: string | null;
	/** Child block IDs (ordering matters) */
	children: string[];
	/** Version for optimistic updates */
	version: number;
}

// ============================================================================
// Rich Text (Y.Text backed)
// ============================================================================

/**
 * Text annotation formats supported in rich text.
 */
export interface TextDelta {
	insert: string;
	attributes?: TextAttributes;
}

export interface TextAttributes {
	bold?: boolean;
	italic?: boolean;
	underline?: boolean;
	strikethrough?: boolean;
	code?: boolean;
	link?: string;
	color?: string;
	background?: string;
	reference?: {
		type: 'page' | 'block' | 'user' | 'date';
		id: string;
	};
}

/**
 * Reference to a Y.Text instance for CRDT-backed text.
 * The actual Y.Text lives in the Yjs document, this is a typed wrapper.
 */
export interface YTextRef {
	/** Y.Text toJSON() serialization */
	delta: TextDelta[];
	/** Plain text content (computed) */
	toString(): string;
}

// ============================================================================
// Block Props by Flavour
// ============================================================================

/**
 * Page block - top-level document container
 */
export interface PageBlockProps {
	title: YTextRef;
	icon?: BlockIcon;
	cover?: {
		type: 'color' | 'gradient' | 'image';
		value: string;
	};
	isTemplate?: boolean;
	isFavorite?: boolean;
	isArchived?: boolean;
}

/**
 * Paragraph block - basic text content
 */
export interface ParagraphBlockProps {
	text: YTextRef;
}

/**
 * Heading block - heading levels 1-3
 */
export interface HeadingBlockProps {
	text: YTextRef;
	level: 1 | 2 | 3;
	collapsible?: boolean;
	collapsed?: boolean;
}

/**
 * List block - bulleted, numbered, or todo
 */
export interface ListBlockProps {
	text: YTextRef;
	type: 'bulleted' | 'numbered' | 'todo' | 'toggle';
	checked?: boolean; // for todo
	collapsed?: boolean; // for toggle
}

/**
 * Code block - syntax highlighted code
 */
export interface CodeBlockProps {
	text: YTextRef;
	language: string;
	caption?: YTextRef;
	wrap?: boolean;
}

/**
 * Quote block - blockquote
 */
export interface QuoteBlockProps {
	text: YTextRef;
}

/**
 * Callout block - highlighted info box
 */
export interface CalloutBlockProps {
	text: YTextRef;
	icon?: string;
	color?: string;
}

/**
 * Divider block - horizontal rule
 */
export interface DividerBlockProps {
	// No additional props needed
}

/**
 * Image block - embedded image
 */
export interface ImageBlockProps {
	url: string;
	caption?: YTextRef;
	width?: number;
	align?: 'left' | 'center' | 'right';
}

/**
 * Bookmark block - link preview
 */
export interface BookmarkBlockProps {
	url: string;
	title?: string;
	description?: string;
	icon?: string;
	image?: string;
	caption?: YTextRef;
}

/**
 * Embed block - iframe embed
 */
export interface EmbedBlockProps {
	url: string;
	provider?: string;
	caption?: YTextRef;
}

// ============================================================================
// Database Block Types
// ============================================================================

/**
 * Database block - table/board/calendar container
 */
export interface DatabaseBlockProps {
	/** Title of the database */
	title: YTextRef;
	/** Column definitions */
	columns: ColumnSchema[];
	/** View configurations */
	views: DatabaseView[];
	/** Cell data indexed by rowId:columnId */
	cells: Record<string, Record<string, CellValue>>;
}

/**
 * Database row block - a row in a database (child of database block)
 */
export interface DatabaseRowBlockProps {
	/** The database this row belongs to */
	databaseId: string;
}

/**
 * Column types supported in database
 */
export type ColumnType =
	| 'title'
	| 'text'
	| 'number'
	| 'select'
	| 'multi-select'
	| 'date'
	| 'checkbox'
	| 'url'
	| 'email'
	| 'phone'
	| 'person'
	| 'file'
	| 'relation'
	| 'rollup'
	| 'formula'
	| 'created-time'
	| 'created-by'
	| 'updated-time'
	| 'updated-by';

/**
 * Column schema definition
 */
export interface ColumnSchema {
	id: string;
	name: string;
	type: ColumnType;
	width?: number;
	/** Type-specific configuration */
	data?: ColumnTypeData;
}

/**
 * Type-specific column configuration
 */
export type ColumnTypeData =
	| { type: 'select'; options: SelectOption[] }
	| { type: 'multi-select'; options: SelectOption[] }
	| { type: 'number'; format?: 'number' | 'percent' | 'currency' }
	| { type: 'date'; includeTime?: boolean; dateFormat?: string }
	| { type: 'relation'; targetDatabaseId: string }
	| { type: 'rollup'; relationColumnId: string; targetColumnId: string; aggregation: RollupAggregation }
	| { type: 'formula'; formula: string };

export interface SelectOption {
	id: string;
	value: string;
	color: string;
}

export type RollupAggregation =
	| 'count'
	| 'count-values'
	| 'count-unique'
	| 'count-empty'
	| 'count-not-empty'
	| 'percent-empty'
	| 'percent-not-empty'
	| 'sum'
	| 'average'
	| 'median'
	| 'min'
	| 'max'
	| 'range'
	| 'show-original';

/**
 * Cell value types
 */
export type CellValue =
	| { type: 'title'; value: TextDelta[] }
	| { type: 'text'; value: string }
	| { type: 'number'; value: number | null }
	| { type: 'select'; value: string | null } // option id
	| { type: 'multi-select'; value: string[] } // option ids
	| { type: 'date'; value: DateCellValue | null }
	| { type: 'checkbox'; value: boolean }
	| { type: 'url'; value: string }
	| { type: 'email'; value: string }
	| { type: 'phone'; value: string }
	| { type: 'person'; value: string[] } // user ids
	| { type: 'file'; value: FileValue[] }
	| { type: 'relation'; value: string[] } // row ids
	| { type: 'rollup'; value: unknown } // computed
	| { type: 'formula'; value: unknown } // computed
	| { type: 'created-time'; value: string }
	| { type: 'created-by'; value: string }
	| { type: 'updated-time'; value: string }
	| { type: 'updated-by'; value: string };

export interface DateCellValue {
	start: string;
	end?: string;
	includeTime?: boolean;
}

export interface FileValue {
	name: string;
	url: string;
	type: string;
	size: number;
}

/**
 * Database view types
 */
export type DatabaseViewType = 'table' | 'kanban' | 'calendar' | 'gallery' | 'list';

export interface DatabaseView {
	id: string;
	name: string;
	type: DatabaseViewType;
	/** Visible column IDs in order */
	columns: string[];
	/** Filter configuration */
	filter?: FilterGroup;
	/** Sort configuration */
	sorts?: SortConfig[];
	/** Group by column (for kanban) */
	groupBy?: string;
}

export interface FilterGroup {
	type: 'and' | 'or';
	conditions: (FilterCondition | FilterGroup)[];
}

export interface FilterCondition {
	columnId: string;
	operator: FilterOperator;
	value: unknown;
}

export type FilterOperator =
	| 'equals'
	| 'not-equals'
	| 'contains'
	| 'not-contains'
	| 'starts-with'
	| 'ends-with'
	| 'is-empty'
	| 'is-not-empty'
	| 'greater-than'
	| 'less-than'
	| 'greater-equal'
	| 'less-equal'
	| 'is-before'
	| 'is-after';

export interface SortConfig {
	columnId: string;
	direction: 'asc' | 'desc';
}

// ============================================================================
// Layout Block Types
// ============================================================================

/**
 * Column layout block - multi-column container
 */
export interface ColumnLayoutBlockProps {
	/** Column widths as ratios (e.g., [1, 2] = 1:2 ratio) */
	ratios: number[];
}

/**
 * Column block - single column in layout
 */
export interface ColumnBlockProps {
	/** Width ratio */
	ratio: number;
}

// ============================================================================
// Special Block Types
// ============================================================================

/**
 * Synced block - block that syncs across multiple locations
 */
export interface SyncedBlockBlockProps {
	/** Source block ID */
	sourceId: string;
}

/**
 * Link to page block - reference to another page
 */
export interface LinkToPageBlockProps {
	/** Target page ID */
	pageId: string;
}

// ============================================================================
// Block Icon
// ============================================================================

export type BlockIcon =
	| { type: 'emoji'; value: string }
	| { type: 'icon'; name: string; color?: string }
	| { type: 'image'; url: string };

// ============================================================================
// Block Props Union Type
// ============================================================================

/**
 * Map of flavour to props type
 */
export interface BlockPropsMap {
	'bos:page': PageBlockProps;
	'bos:paragraph': ParagraphBlockProps;
	'bos:heading': HeadingBlockProps;
	'bos:list': ListBlockProps;
	'bos:code': CodeBlockProps;
	'bos:quote': QuoteBlockProps;
	'bos:callout': CalloutBlockProps;
	'bos:divider': DividerBlockProps;
	'bos:image': ImageBlockProps;
	'bos:bookmark': BookmarkBlockProps;
	'bos:embed': EmbedBlockProps;
	'bos:database': DatabaseBlockProps;
	'bos:database-row': DatabaseRowBlockProps;
	'bos:column-layout': ColumnLayoutBlockProps;
	'bos:column': ColumnBlockProps;
	'bos:synced-block': SyncedBlockBlockProps;
	'bos:link-to-page': LinkToPageBlockProps;
}

/**
 * Get props type for a specific flavour
 */
export type BlockProps<F extends BlockFlavour> = F extends keyof BlockPropsMap
	? BlockPropsMap[F]
	: never;

// ============================================================================
// Full Block Type (System + Props)
// ============================================================================

/**
 * Full block type combining system props and flavour-specific props.
 */
export interface Block<F extends BlockFlavour = BlockFlavour> extends BlockSysProps {
	flavour: F;
	props: BlockProps<F>;
	/** Timestamps */
	createdAt: string;
	updatedAt: string;
	createdBy?: string;
	updatedBy?: string;
}

// ============================================================================
// Block Snapshot (Serialized Form)
// ============================================================================

/**
 * Serialized block snapshot for persistence/transfer.
 * Y.Text is serialized to delta format.
 */
export interface BlockSnapshot<F extends BlockFlavour = BlockFlavour> {
	id: string;
	flavour: F;
	parent: string | null;
	children: string[];
	props: SerializedBlockProps<F>;
	version: number;
	createdAt: string;
	updatedAt: string;
	createdBy?: string;
	updatedBy?: string;
}

/**
 * Serialized props (Y.Text converted to delta arrays)
 */
export type SerializedBlockProps<F extends BlockFlavour> = F extends keyof BlockPropsMap
	? SerializeYText<BlockPropsMap[F]>
	: never;

/**
 * Helper type to convert YTextRef to TextDelta[] in props
 */
type SerializeYText<T> = {
	[K in keyof T]: T[K] extends YTextRef
		? TextDelta[]
		: T[K] extends YTextRef | undefined
			? TextDelta[] | undefined
			: T[K];
};

// ============================================================================
// Block Collection (Document-level container)
// ============================================================================

/**
 * A BlockCollection represents a document/workspace containing blocks.
 * It wraps a Yjs document and provides block operations.
 */
export interface BlockCollection {
	/** Unique collection ID (document/workspace ID) */
	id: string;
	/** Root block ID */
	rootId: string;
	/** Underlying Yjs document */
	ydoc: Y.Doc;
	/** Get block by ID */
	getBlock(id: string): Block | null;
	/** Get all blocks */
	getBlocks(): Map<string, Block>;
	/** Add a block */
	addBlock(flavour: BlockFlavour, props: Partial<BlockProps<BlockFlavour>>, parentId?: string): Block;
	/** Update block props */
	updateBlock(id: string, props: Partial<BlockProps<BlockFlavour>>): void;
	/** Delete a block */
	deleteBlock(id: string): void;
	/** Move block to new parent/position */
	moveBlock(id: string, newParentId: string, index?: number): void;
	/** Export to snapshot */
	toSnapshot(): BlockSnapshot[];
	/** Import from snapshot */
	fromSnapshot(snapshots: BlockSnapshot[]): void;
}

// ============================================================================
// Block Events
// ============================================================================

export type BlockEventType =
	| 'block:added'
	| 'block:updated'
	| 'block:deleted'
	| 'block:moved'
	| 'block:children-changed';

export interface BlockEvent {
	type: BlockEventType;
	blockId: string;
	data?: unknown;
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Generate a unique block ID
 */
export function generateBlockId(): string {
	return crypto.randomUUID();
}

/**
 * Check if a flavour is a container (can have children)
 */
export function isContainerFlavour(flavour: BlockFlavour): boolean {
	return [
		'bos:page',
		'bos:database',
		'bos:list',
		'bos:heading',
		'bos:callout',
		'bos:quote',
		'bos:column-layout',
		'bos:column',
		'bos:synced-block'
	].includes(flavour);
}

/**
 * Check if a flavour supports text content
 */
export function hasTextContent(flavour: BlockFlavour): boolean {
	return [
		'bos:paragraph',
		'bos:heading',
		'bos:list',
		'bos:code',
		'bos:quote',
		'bos:callout'
	].includes(flavour);
}

/**
 * Get default props for a flavour
 */
export function getDefaultProps<F extends BlockFlavour>(flavour: F): Partial<BlockProps<F>> {
	const defaults: Record<string, unknown> = {
		'bos:page': { title: { delta: [] } },
		'bos:paragraph': { text: { delta: [] } },
		'bos:heading': { text: { delta: [] }, level: 1 },
		'bos:list': { text: { delta: [] }, type: 'bulleted' },
		'bos:code': { text: { delta: [] }, language: 'plain' },
		'bos:quote': { text: { delta: [] } },
		'bos:callout': { text: { delta: [] }, icon: '💡' },
		'bos:divider': {},
		'bos:image': { url: '' },
		'bos:bookmark': { url: '' },
		'bos:embed': { url: '' },
		'bos:database': { title: { delta: [] }, columns: [], views: [], cells: {} },
		'bos:database-row': { databaseId: '' },
		'bos:column-layout': { ratios: [1, 1] },
		'bos:column': { ratio: 1 },
		'bos:synced-block': { sourceId: '' },
		'bos:link-to-page': { pageId: '' }
	};

	return (defaults[flavour] ?? {}) as Partial<BlockProps<F>>;
}
