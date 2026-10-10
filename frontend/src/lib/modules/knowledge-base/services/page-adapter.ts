/**
 * Page Adapter - Bridge between Block System and Page API
 *
 * Converts between the new Block system and the Page (formerly Context) API.
 * Pages are documents in the Knowledge Base. They link to Nodes in the OS layer.
 *
 * TAXONOMY:
 * - Page = Document in Knowledge Base (with Blocks)
 * - Node = Building block of the Operating System graph
 * - Creation = AI-generated output (in Chat module)
 */

import type {
	Block,
	BlockFlavour,
	BlockSnapshot,
	TextDelta,
	PageBlockProps,
	ParagraphBlockProps,
	HeadingBlockProps,
	ListBlockProps,
	CodeBlockProps,
	DatabaseBlockProps,
	ColumnSchema,
	CellValue,
	DatabaseView
} from '../entities/block';
import { generateBlockId, getDefaultProps } from '../entities/block';
import type {
	Context as PageData,
	ContextListItem as PageListItem,
	Block as PageBlock,
	PropertySchema,
	CreateContextData as CreatePageData,
	UpdateContextData as UpdatePageData
} from '$lib/api/contexts/types';
import type { Node, NodeTree } from '$lib/api/nodes/types';

// ============================================================================
// Type Mappings
// ============================================================================

/**
 * Map page block type to block flavour
 */
const BLOCK_TYPE_MAP: Record<string, BlockFlavour> = {
	'paragraph': 'bos:paragraph',
	'text': 'bos:paragraph',
	'heading': 'bos:heading',
	'heading_1': 'bos:heading',
	'heading_2': 'bos:heading',
	'heading_3': 'bos:heading',
	'bulleted_list': 'bos:list',
	'numbered_list': 'bos:list',
	'to_do': 'bos:list',
	'toggle': 'bos:list',
	'code': 'bos:code',
	'quote': 'bos:quote',
	'callout': 'bos:callout',
	'divider': 'bos:divider',
	'image': 'bos:image',
	'bookmark': 'bos:bookmark',
	'embed': 'bos:embed',
	'database': 'bos:database',
	'table': 'bos:database'
};

/**
 * Map property schema type to column type
 */
const PROPERTY_TYPE_MAP: Record<string, string> = {
	'text': 'text',
	'select': 'select',
	'multi_select': 'multi-select',
	'date': 'date',
	'person': 'person',
	'relation': 'relation',
	'number': 'number',
	'checkbox': 'checkbox',
	'url': 'url',
	'email': 'email'
};

// ============================================================================
// Page to Block Conversion
// ============================================================================

/**
 * Convert a Page to a page block with children
 */
export function pageToBlocks(page: PageData): BlockSnapshot[] {
	const blocks: BlockSnapshot[] = [];
	const now = new Date().toISOString();

	// Create page block
	// Note: In BlockSnapshot, YTextRef is serialized to TextDelta[] directly
	const pageBlock: BlockSnapshot<'bos:page'> = {
		id: page.id,
		flavour: 'bos:page',
		parent: null,
		children: [],
		props: {
			title: [{ insert: page.name }],
			icon: page.icon ? { type: 'emoji', value: page.icon } : undefined,
			cover: page.cover_image
				? { type: 'image', value: page.cover_image }
				: undefined,
			isTemplate: page.is_template,
			isFavorite: false,
			isArchived: page.is_archived
		},
		version: 1,
		createdAt: page.created_at,
		updatedAt: page.updated_at
	};

	// Convert page blocks to block snapshots
	if (page.blocks && page.blocks.length > 0) {
		const childIds: string[] = [];

		for (const pageBlock of page.blocks) {
			const converted = convertPageBlock(pageBlock, page.id);
			blocks.push(...converted);
			if (converted.length > 0) {
				childIds.push(converted[0].id);
			}
		}

		pageBlock.children = childIds;
	}

	// If page has property_schema, create a database block as first child
	if (page.property_schema && page.property_schema.length > 0) {
		const dbBlock = createDatabaseFromPropertySchema(
			page.id,
			page.name,
			page.property_schema,
			page.properties
		);
		blocks.push(dbBlock);
		pageBlock.children = [dbBlock.id, ...pageBlock.children];
	}

	blocks.unshift(pageBlock);
	return blocks;
}

/**
 * Convert a single page block to block snapshot(s)
 */
function convertPageBlock(
	pageBlock: PageBlock,
	parentId: string
): BlockSnapshot[] {
	const blocks: BlockSnapshot[] = [];
	const now = new Date().toISOString();
	const flavour = BLOCK_TYPE_MAP[pageBlock.type] ?? 'bos:paragraph';

	const block: BlockSnapshot = {
		id: pageBlock.id || generateBlockId(),
		flavour,
		parent: parentId,
		children: [],
		props: convertBlockProps(pageBlock, flavour),
		version: 1,
		createdAt: now,
		updatedAt: now
	};

	// Convert children recursively
	if (pageBlock.children && pageBlock.children.length > 0) {
		const childIds: string[] = [];
		for (const child of pageBlock.children) {
			const converted = convertPageBlock(child, block.id);
			blocks.push(...converted);
			if (converted.length > 0) {
				childIds.push(converted[0].id);
			}
		}
		block.children = childIds;
	}

	blocks.unshift(block);
	return blocks;
}

/**
 * Convert page block properties to block props
 */
function convertBlockProps(
	pageBlock: PageBlock,
	flavour: BlockFlavour
): Record<string, unknown> {
	const content = pageBlock.content ?? '';
	const props = pageBlock.properties ?? {};

	// Note: In BlockSnapshot, YTextRef is serialized to TextDelta[] directly
	switch (flavour) {
		case 'bos:paragraph':
			return {
				text: [{ insert: content }]
			};

		case 'bos:heading':
			const level = pageBlock.type.includes('_')
				? parseInt(pageBlock.type.split('_')[1]) || 1
				: (props.level as number) || 1;
			return {
				text: [{ insert: content }],
				level: Math.min(3, Math.max(1, level)) as 1 | 2 | 3
			};

		case 'bos:list':
			return {
				text: [{ insert: content }],
				type: getListType(pageBlock.type),
				checked: props.checked as boolean | undefined
			};

		case 'bos:code':
			return {
				text: [{ insert: content }],
				language: (props.language as string) || 'plain'
			};

		case 'bos:quote':
			return {
				text: [{ insert: content }]
			};

		case 'bos:callout':
			return {
				text: [{ insert: content }],
				icon: props.icon as string | undefined,
				color: props.color as string | undefined
			};

		case 'bos:divider':
			return {};

		case 'bos:image':
			return {
				url: (props.url as string) || content,
				caption: props.caption
					? [{ insert: props.caption as string }]
					: undefined
			};

		case 'bos:bookmark':
			return {
				url: (props.url as string) || content,
				title: props.title as string | undefined,
				description: props.description as string | undefined
			};

		case 'bos:embed':
			return {
				url: (props.url as string) || content
			};

		default:
			return {
				text: [{ insert: content }]
			};
	}
}

/**
 * Get list type from page block type
 */
function getListType(type: string): 'bulleted' | 'numbered' | 'todo' | 'toggle' {
	switch (type) {
		case 'numbered_list':
			return 'numbered';
		case 'to_do':
			return 'todo';
		case 'toggle':
			return 'toggle';
		default:
			return 'bulleted';
	}
}

/**
 * Create a database block from property schema
 */
function createDatabaseFromPropertySchema(
	parentId: string,
	name: string,
	schema: PropertySchema[],
	properties: Record<string, unknown> | null
): BlockSnapshot<'bos:database'> {
	const now = new Date().toISOString();
	const dbId = generateBlockId();

	// Convert property schema to columns
	const columns: ColumnSchema[] = schema.map((prop) => ({
		id: generateBlockId(),
		name: prop.name,
		type: (PROPERTY_TYPE_MAP[prop.type] || 'text') as ColumnSchema['type'],
		data: prop.options
			? {
					type: prop.type === 'multi_select' ? 'multi-select' : 'select',
					options: prop.options.map((opt) => ({
						id: generateBlockId(),
						value: opt,
						color: getRandomColor()
					}))
				}
			: undefined
	}));

	// Create default table view
	const views: DatabaseView[] = [
		{
			id: generateBlockId(),
			name: 'Table',
			type: 'table',
			columns: columns.map((c) => c.id)
		}
	];

	return {
		id: dbId,
		flavour: 'bos:database',
		parent: parentId,
		children: [],
		props: {
			title: [{ insert: `${name} Properties` }],
			columns,
			views,
			cells: {}
		},
		version: 1,
		createdAt: now,
		updatedAt: now
	};
}

// ============================================================================
// Block to Page Conversion
// ============================================================================

/**
 * Convert block snapshots back to page format for API
 */
export function blocksToPage(
	blocks: BlockSnapshot[],
	pageId?: string
): UpdatePageData {
	// Find the page block
	const rootPageBlock = blocks.find((b) => b.flavour === 'bos:page');
	if (!rootPageBlock) {
		throw new Error('No page block found');
	}

	const pageProps = rootPageBlock.props as PageBlockProps;

	// Convert child blocks to page blocks
	const pageBlocks = rootPageBlock.children
		.map((childId) => {
			const block = blocks.find((b) => b.id === childId);
			if (!block) return null;
			return convertToPageBlock(block, blocks);
		})
		.filter((b): b is PageBlock => b !== null);

	// Extract title text (title is TextDelta[] in serialized form)
	// Use unknown first for safe conversion from YTextRef type
	const titleDelta = pageProps.title as unknown as TextDelta[] | undefined;
	const titleText = titleDelta?.map((d) => d.insert).join('') ?? 'Untitled';

	return {
		name: titleText,
		blocks: pageBlocks,
		icon: pageProps.icon?.type === 'emoji' ? pageProps.icon.value : undefined,
		cover_image: pageProps.cover?.type === 'image' ? pageProps.cover.value : undefined,
		is_template: pageProps.isTemplate,
		is_archived: pageProps.isArchived
	};
}

/**
 * Convert a block snapshot to page block
 */
function convertToPageBlock(
	block: BlockSnapshot,
	allBlocks: BlockSnapshot[]
): PageBlock {
	const type = flavourToPageType(block.flavour);
	const content = extractTextContent(block.props);
	const properties = extractProperties(block);

	// Convert children
	const children = block.children
		.map((childId) => {
			const childBlock = allBlocks.find((b) => b.id === childId);
			if (!childBlock) return null;
			return convertToPageBlock(childBlock, allBlocks);
		})
		.filter((b): b is PageBlock => b !== null);

	return {
		id: block.id,
		type,
		content,
		properties: Object.keys(properties).length > 0 ? properties : undefined,
		children: children.length > 0 ? children : undefined
	};
}

/**
 * Convert block flavour to page block type
 */
function flavourToPageType(flavour: BlockFlavour): string {
	const reverseMap: Record<BlockFlavour, string> = {
		'bos:page': 'page',
		'bos:paragraph': 'paragraph',
		'bos:heading': 'heading',
		'bos:list': 'bulleted_list',
		'bos:code': 'code',
		'bos:quote': 'quote',
		'bos:callout': 'callout',
		'bos:divider': 'divider',
		'bos:image': 'image',
		'bos:bookmark': 'bookmark',
		'bos:embed': 'embed',
		'bos:database': 'database',
		'bos:database-row': 'database_row',
		'bos:column-layout': 'column_list',
		'bos:column': 'column',
		'bos:synced-block': 'synced_block',
		'bos:link-to-page': 'link_to_page'
	};
	return reverseMap[flavour] ?? 'paragraph';
}

/**
 * Extract text content from block props
 * Note: In serialized form, text is TextDelta[] directly
 */
function extractTextContent(props: Record<string, unknown>): string {
	if (props.text && Array.isArray(props.text)) {
		return (props.text as TextDelta[]).map((d) => d.insert).join('');
	}
	return '';
}

/**
 * Extract properties from block props
 */
function extractProperties(block: BlockSnapshot): Record<string, unknown> {
	const props: Record<string, unknown> = {};
	const blockProps = block.props as Record<string, unknown>;

	if (block.flavour === 'bos:heading') {
		props.level = (blockProps as { level?: number }).level ?? 1;
	}

	if (block.flavour === 'bos:list') {
		const listProps = blockProps as { type?: string; checked?: boolean };
		if (listProps.checked !== undefined) {
			props.checked = listProps.checked;
		}
	}

	if (block.flavour === 'bos:code') {
		props.language = (blockProps as { language?: string }).language ?? 'plain';
	}

	if (block.flavour === 'bos:image' || block.flavour === 'bos:bookmark') {
		props.url = (blockProps as { url?: string }).url ?? '';
	}

	return props;
}

// ============================================================================
// Node Integration
// ============================================================================

/**
 * Convert node tree to document tree for sidebar
 */
export function nodeTreeToDocumentTree(nodes: NodeTree[]): {
	id: string;
	title: string;
	type: 'folder';
	icon: string | null;
	children: ReturnType<typeof nodeTreeToDocumentTree>;
	nodeData: NodeTree;
}[] {
	return nodes.map((node) => ({
		id: `node:${node.id}`,
		title: node.name,
		type: 'folder' as const,
		icon: getNodeIcon(node.type),
		children: nodeTreeToDocumentTree(node.children),
		nodeData: node
	}));
}

/**
 * Get icon for node type
 */
export function getNodeIcon(type: string): string {
	const icons: Record<string, string> = {
		entity: '🏢',
		department: '🏛️',
		team: '👥',
		project: '📁',
		operational: '🔧',
		learning: '📚',
		person: '👤',
		product: '🛠️',
		partnership: '🤝',
		context: '📋' // Context as a Node type (reference material)
	};
	return icons[type] ?? '📄';
}

/**
 * Get page icon for page type
 */
export function getPageIcon(type: string): string {
	const icons: Record<string, string> = {
		profile: '👤',
		reference: '📖',
		template: '📝',
		note: '📝',
		document: '📄'
	};
	return icons[type] ?? '📄';
}

// ============================================================================
// Utilities
// ============================================================================

/**
 * Get a random color for select options
 */
function getRandomColor(): string {
	const colors = [
		'#e53935', // red
		'#d81b60', // pink
		'#8e24aa', // purple
		'#5e35b1', // deep purple
		'#3949ab', // indigo
		'#1e88e5', // blue
		'#039be5', // light blue
		'#00acc1', // cyan
		'#00897b', // teal
		'#43a047', // green
		'#7cb342', // light green
		'#c0ca33', // lime
		'#fdd835', // yellow
		'#ffb300', // amber
		'#fb8c00', // orange
		'#f4511e'  // deep orange
	];
	return colors[Math.floor(Math.random() * colors.length)];
}

/**
 * Merge page list item with block metadata
 */
export function mergePageMeta(
	pageItem: PageListItem,
	additionalMeta?: Partial<PageListItem>
): PageListItem {
	return {
		...pageItem,
		...additionalMeta
	};
}

// ============================================================================
// Re-exports with correct names (for backwards compatibility)
// ============================================================================

/** @deprecated Use pageToBlocks instead */
export const contextToBlocks = pageToBlocks;

/** @deprecated Use blocksToPage instead */
export const blocksToContext = blocksToPage;

/** @deprecated Use getPageIcon instead */
export const getContextIcon = getPageIcon;

/** @deprecated Use mergePageMeta instead */
export const mergeContextMeta = mergePageMeta;
