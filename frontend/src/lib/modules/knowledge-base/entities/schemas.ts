/**
 * Block Schemas - Zod Validation
 *
 * Defines the shape and validation rules for each block type.
 * Used for:
 * - Runtime validation of block data
 * - Type inference for TypeScript
 * - API request/response validation
 * - Form validation in the editor
 */

import { z } from 'zod';

// ============================================================================
// Base Schemas
// ============================================================================

/**
 * Text delta schema (for Y.Text serialization)
 */
export const textAttributesSchema = z.object({
	bold: z.boolean().optional(),
	italic: z.boolean().optional(),
	underline: z.boolean().optional(),
	strikethrough: z.boolean().optional(),
	code: z.boolean().optional(),
	link: z.string().optional(),
	color: z.string().optional(),
	background: z.string().optional(),
	reference: z
		.object({
			type: z.enum(['page', 'block', 'user', 'date']),
			id: z.string()
		})
		.optional()
});

export const textDeltaSchema = z.object({
	insert: z.string(),
	attributes: textAttributesSchema.optional()
});

export const yTextRefSchema = z.object({
	delta: z.array(textDeltaSchema)
});

/**
 * Block icon schema
 */
export const blockIconSchema = z.discriminatedUnion('type', [
	z.object({ type: z.literal('emoji'), value: z.string() }),
	z.object({ type: z.literal('icon'), name: z.string(), color: z.string().optional() }),
	z.object({ type: z.literal('image'), url: z.string().url() })
]);

/**
 * Block flavours
 */
export const blockFlavourSchema = z.enum([
	'bos:page',
	'bos:database',
	'bos:paragraph',
	'bos:heading',
	'bos:list',
	'bos:code',
	'bos:quote',
	'bos:callout',
	'bos:divider',
	'bos:image',
	'bos:bookmark',
	'bos:embed',
	'bos:database-row',
	'bos:column-layout',
	'bos:column',
	'bos:synced-block',
	'bos:link-to-page'
]);

// ============================================================================
// Content Block Schemas
// ============================================================================

/**
 * Page block props
 */
export const pageBlockPropsSchema = z.object({
	title: yTextRefSchema,
	icon: blockIconSchema.optional(),
	cover: z
		.object({
			type: z.enum(['color', 'gradient', 'image']),
			value: z.string()
		})
		.optional(),
	isTemplate: z.boolean().optional(),
	isFavorite: z.boolean().optional(),
	isArchived: z.boolean().optional()
});

/**
 * Paragraph block props
 */
export const paragraphBlockPropsSchema = z.object({
	text: yTextRefSchema
});

/**
 * Heading block props
 */
export const headingBlockPropsSchema = z.object({
	text: yTextRefSchema,
	level: z.union([z.literal(1), z.literal(2), z.literal(3)]),
	collapsible: z.boolean().optional(),
	collapsed: z.boolean().optional()
});

/**
 * List block props
 */
export const listBlockPropsSchema = z.object({
	text: yTextRefSchema,
	type: z.enum(['bulleted', 'numbered', 'todo', 'toggle']),
	checked: z.boolean().optional(),
	collapsed: z.boolean().optional()
});

/**
 * Code block props
 */
export const codeBlockPropsSchema = z.object({
	text: yTextRefSchema,
	language: z.string(),
	caption: yTextRefSchema.optional(),
	wrap: z.boolean().optional()
});

/**
 * Quote block props
 */
export const quoteBlockPropsSchema = z.object({
	text: yTextRefSchema
});

/**
 * Callout block props
 */
export const calloutBlockPropsSchema = z.object({
	text: yTextRefSchema,
	icon: z.string().optional(),
	color: z.string().optional()
});

/**
 * Divider block props (empty)
 */
export const dividerBlockPropsSchema = z.object({});

/**
 * Image block props
 */
export const imageBlockPropsSchema = z.object({
	url: z.string(),
	caption: yTextRefSchema.optional(),
	width: z.number().positive().optional(),
	align: z.enum(['left', 'center', 'right']).optional()
});

/**
 * Bookmark block props
 */
export const bookmarkBlockPropsSchema = z.object({
	url: z.string().url(),
	title: z.string().optional(),
	description: z.string().optional(),
	icon: z.string().optional(),
	image: z.string().optional(),
	caption: yTextRefSchema.optional()
});

/**
 * Embed block props
 */
export const embedBlockPropsSchema = z.object({
	url: z.string().url(),
	provider: z.string().optional(),
	caption: yTextRefSchema.optional()
});

// ============================================================================
// Database Block Schemas
// ============================================================================

/**
 * Column types
 */
export const columnTypeSchema = z.enum([
	'title',
	'text',
	'number',
	'select',
	'multi-select',
	'date',
	'checkbox',
	'url',
	'email',
	'phone',
	'person',
	'file',
	'relation',
	'rollup',
	'formula',
	'created-time',
	'created-by',
	'updated-time',
	'updated-by'
]);

/**
 * Select option
 */
export const selectOptionSchema = z.object({
	id: z.string(),
	value: z.string(),
	color: z.string()
});

/**
 * Rollup aggregation
 */
export const rollupAggregationSchema = z.enum([
	'count',
	'count-values',
	'count-unique',
	'count-empty',
	'count-not-empty',
	'percent-empty',
	'percent-not-empty',
	'sum',
	'average',
	'median',
	'min',
	'max',
	'range',
	'show-original'
]);

/**
 * Column type-specific data
 */
export const columnTypeDataSchema = z.discriminatedUnion('type', [
	z.object({ type: z.literal('select'), options: z.array(selectOptionSchema) }),
	z.object({ type: z.literal('multi-select'), options: z.array(selectOptionSchema) }),
	z.object({
		type: z.literal('number'),
		format: z.enum(['number', 'percent', 'currency']).optional()
	}),
	z.object({
		type: z.literal('date'),
		includeTime: z.boolean().optional(),
		dateFormat: z.string().optional()
	}),
	z.object({ type: z.literal('relation'), targetDatabaseId: z.string() }),
	z.object({
		type: z.literal('rollup'),
		relationColumnId: z.string(),
		targetColumnId: z.string(),
		aggregation: rollupAggregationSchema
	}),
	z.object({ type: z.literal('formula'), formula: z.string() })
]);

/**
 * Column schema
 */
export const columnSchemaSchema = z.object({
	id: z.string(),
	name: z.string(),
	type: columnTypeSchema,
	width: z.number().positive().optional(),
	data: columnTypeDataSchema.optional()
});

/**
 * Filter operators
 */
export const filterOperatorSchema = z.enum([
	'equals',
	'not-equals',
	'contains',
	'not-contains',
	'starts-with',
	'ends-with',
	'is-empty',
	'is-not-empty',
	'greater-than',
	'less-than',
	'greater-equal',
	'less-equal',
	'is-before',
	'is-after'
]);

/**
 * Filter condition
 */
export const filterConditionSchema = z.object({
	columnId: z.string(),
	operator: filterOperatorSchema,
	value: z.unknown()
});

/**
 * Filter group (recursive)
 */
export const filterGroupSchema: z.ZodType<{
	type: 'and' | 'or';
	conditions: (
		| { columnId: string; operator: string; value: unknown }
		| { type: 'and' | 'or'; conditions: unknown[] }
	)[];
}> = z.lazy(() =>
	z.object({
		type: z.enum(['and', 'or']),
		conditions: z.array(z.union([filterConditionSchema, filterGroupSchema]))
	})
);

/**
 * Sort config
 */
export const sortConfigSchema = z.object({
	columnId: z.string(),
	direction: z.enum(['asc', 'desc'])
});

/**
 * Database view type
 */
export const databaseViewTypeSchema = z.enum(['table', 'kanban', 'calendar', 'gallery', 'list']);

/**
 * Database view
 */
export const databaseViewSchema = z.object({
	id: z.string(),
	name: z.string(),
	type: databaseViewTypeSchema,
	columns: z.array(z.string()),
	filter: filterGroupSchema.optional(),
	sorts: z.array(sortConfigSchema).optional(),
	groupBy: z.string().optional()
});

/**
 * Date cell value
 */
export const dateCellValueSchema = z.object({
	start: z.string(),
	end: z.string().optional(),
	includeTime: z.boolean().optional()
});

/**
 * File value
 */
export const fileValueSchema = z.object({
	name: z.string(),
	url: z.string(),
	type: z.string(),
	size: z.number()
});

/**
 * Cell value (discriminated union)
 */
export const cellValueSchema = z.discriminatedUnion('type', [
	z.object({ type: z.literal('title'), value: z.array(textDeltaSchema) }),
	z.object({ type: z.literal('text'), value: z.string() }),
	z.object({ type: z.literal('number'), value: z.number().nullable() }),
	z.object({ type: z.literal('select'), value: z.string().nullable() }),
	z.object({ type: z.literal('multi-select'), value: z.array(z.string()) }),
	z.object({ type: z.literal('date'), value: dateCellValueSchema.nullable() }),
	z.object({ type: z.literal('checkbox'), value: z.boolean() }),
	z.object({ type: z.literal('url'), value: z.string() }),
	z.object({ type: z.literal('email'), value: z.string() }),
	z.object({ type: z.literal('phone'), value: z.string() }),
	z.object({ type: z.literal('person'), value: z.array(z.string()) }),
	z.object({ type: z.literal('file'), value: z.array(fileValueSchema) }),
	z.object({ type: z.literal('relation'), value: z.array(z.string()) }),
	z.object({ type: z.literal('rollup'), value: z.unknown() }),
	z.object({ type: z.literal('formula'), value: z.unknown() }),
	z.object({ type: z.literal('created-time'), value: z.string() }),
	z.object({ type: z.literal('created-by'), value: z.string() }),
	z.object({ type: z.literal('updated-time'), value: z.string() }),
	z.object({ type: z.literal('updated-by'), value: z.string() })
]);

/**
 * Database block props
 */
export const databaseBlockPropsSchema = z.object({
	title: yTextRefSchema,
	columns: z.array(columnSchemaSchema),
	views: z.array(databaseViewSchema),
	cells: z.record(z.string(), z.record(z.string(), cellValueSchema))
});

/**
 * Database row block props
 */
export const databaseRowBlockPropsSchema = z.object({
	databaseId: z.string()
});

// ============================================================================
// Layout Block Schemas
// ============================================================================

/**
 * Column layout block props
 */
export const columnLayoutBlockPropsSchema = z.object({
	ratios: z.array(z.number().positive())
});

/**
 * Column block props
 */
export const columnBlockPropsSchema = z.object({
	ratio: z.number().positive()
});

// ============================================================================
// Special Block Schemas
// ============================================================================

/**
 * Synced block props
 */
export const syncedBlockPropsSchema = z.object({
	sourceId: z.string()
});

/**
 * Link to page block props
 */
export const linkToPageBlockPropsSchema = z.object({
	pageId: z.string()
});

// ============================================================================
// Block Props Union Schema
// ============================================================================

/**
 * All block props by flavour
 */
export const blockPropsSchemaMap = {
	'bos:page': pageBlockPropsSchema,
	'bos:paragraph': paragraphBlockPropsSchema,
	'bos:heading': headingBlockPropsSchema,
	'bos:list': listBlockPropsSchema,
	'bos:code': codeBlockPropsSchema,
	'bos:quote': quoteBlockPropsSchema,
	'bos:callout': calloutBlockPropsSchema,
	'bos:divider': dividerBlockPropsSchema,
	'bos:image': imageBlockPropsSchema,
	'bos:bookmark': bookmarkBlockPropsSchema,
	'bos:embed': embedBlockPropsSchema,
	'bos:database': databaseBlockPropsSchema,
	'bos:database-row': databaseRowBlockPropsSchema,
	'bos:column-layout': columnLayoutBlockPropsSchema,
	'bos:column': columnBlockPropsSchema,
	'bos:synced-block': syncedBlockPropsSchema,
	'bos:link-to-page': linkToPageBlockPropsSchema
} as const;

// ============================================================================
// Full Block Schema
// ============================================================================

/**
 * Block system props schema
 */
export const blockSysPropsSchema = z.object({
	id: z.string().uuid(),
	flavour: blockFlavourSchema,
	parent: z.string().uuid().nullable(),
	children: z.array(z.string().uuid()),
	version: z.number().int().positive()
});

/**
 * Full block schema (generic)
 */
export const blockSchema = blockSysPropsSchema.extend({
	props: z.unknown(), // Will be validated per flavour
	createdAt: z.string().datetime(),
	updatedAt: z.string().datetime(),
	createdBy: z.string().optional(),
	updatedBy: z.string().optional()
});

/**
 * Validate block props for a specific flavour
 */
export function validateBlockProps(flavour: keyof typeof blockPropsSchemaMap, props: unknown) {
	const schema = blockPropsSchemaMap[flavour];
	return schema.safeParse(props);
}

/**
 * Validate a full block
 */
export function validateBlock(block: unknown) {
	const baseResult = blockSchema.safeParse(block);
	if (!baseResult.success) {
		return baseResult;
	}

	const flavour = baseResult.data.flavour as keyof typeof blockPropsSchemaMap;
	const propsSchema = blockPropsSchemaMap[flavour];
	if (propsSchema) {
		const propsResult = propsSchema.safeParse(baseResult.data.props);
		if (!propsResult.success) {
			return { success: false as const, error: propsResult.error };
		}
	}

	return baseResult;
}

// ============================================================================
// Type Exports (inferred from schemas)
// ============================================================================

export type TextAttributes = z.infer<typeof textAttributesSchema>;
export type TextDelta = z.infer<typeof textDeltaSchema>;
export type BlockIcon = z.infer<typeof blockIconSchema>;
export type BlockFlavour = z.infer<typeof blockFlavourSchema>;
export type ColumnType = z.infer<typeof columnTypeSchema>;
export type SelectOption = z.infer<typeof selectOptionSchema>;
export type ColumnSchema = z.infer<typeof columnSchemaSchema>;
export type FilterOperator = z.infer<typeof filterOperatorSchema>;
export type FilterCondition = z.infer<typeof filterConditionSchema>;
export type SortConfig = z.infer<typeof sortConfigSchema>;
export type DatabaseViewType = z.infer<typeof databaseViewTypeSchema>;
export type DatabaseView = z.infer<typeof databaseViewSchema>;
export type CellValue = z.infer<typeof cellValueSchema>;

// Props types
export type PageBlockProps = z.infer<typeof pageBlockPropsSchema>;
export type ParagraphBlockProps = z.infer<typeof paragraphBlockPropsSchema>;
export type HeadingBlockProps = z.infer<typeof headingBlockPropsSchema>;
export type ListBlockProps = z.infer<typeof listBlockPropsSchema>;
export type CodeBlockProps = z.infer<typeof codeBlockPropsSchema>;
export type QuoteBlockProps = z.infer<typeof quoteBlockPropsSchema>;
export type CalloutBlockProps = z.infer<typeof calloutBlockPropsSchema>;
export type DividerBlockProps = z.infer<typeof dividerBlockPropsSchema>;
export type ImageBlockProps = z.infer<typeof imageBlockPropsSchema>;
export type BookmarkBlockProps = z.infer<typeof bookmarkBlockPropsSchema>;
export type EmbedBlockProps = z.infer<typeof embedBlockPropsSchema>;
export type DatabaseBlockProps = z.infer<typeof databaseBlockPropsSchema>;
export type DatabaseRowBlockProps = z.infer<typeof databaseRowBlockPropsSchema>;
export type ColumnLayoutBlockProps = z.infer<typeof columnLayoutBlockPropsSchema>;
export type ColumnBlockProps = z.infer<typeof columnBlockPropsSchema>;
export type SyncedBlockProps = z.infer<typeof syncedBlockPropsSchema>;
export type LinkToPageBlockProps = z.infer<typeof linkToPageBlockPropsSchema>;
