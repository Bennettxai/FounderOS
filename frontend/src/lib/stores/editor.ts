import { writable, derived } from 'svelte/store';
import type { Block } from '$lib/api';

export type BlockType =
	| 'paragraph'
	| 'heading1'
	| 'heading2'
	| 'heading3'
	| 'bulletList'
	| 'numberedList'
	| 'todo'
	| 'toggle'
	| 'quote'
	| 'code'
	| 'divider'
	| 'callout'
	| 'image'
	| 'table'
	| 'embed'
	| 'artifact'
	| 'page'
	| 'tableOfContents'
	| 'columns'
	| 'bookmark';

export interface EditorBlock extends Block {
	id: string;
	type: BlockType;
	content: string;
	properties?: {
		checked?: boolean;
		expanded?: boolean;
		language?: string;
		artifactId?: string;
		url?: string;
		caption?: string;
		calloutType?: 'info' | 'warning' | 'success' | 'error';
		pageId?: string;
		[key: string]: unknown;
	};
	children?: EditorBlock[];
}

export interface EditorState {
	blocks: EditorBlock[];
	focusedBlockId: string | null;
	focusedBlockIndex: number;
	selectionStart: number;
	selectionEnd: number;
	isDirty: boolean;
	isSaving: boolean;
	lastSavedAt: Date | null;
	showSlashMenu: boolean;
	slashMenuPosition: { x: number; y: number } | null;
	slashMenuQuery: string;
	showAIPanel: boolean;
	// Pending block type selection from BlockMenu - Block.svelte handles this
	pendingBlockTypeSelection: { type: BlockType; blockId: string; properties?: Record<string, unknown> } | null;
}

function generateBlockId(): string {
	return Math.random().toString(36).substring(2, 11);
}

export function createEmptyBlock(type: BlockType = 'paragraph'): EditorBlock {
	return {
		id: generateBlockId(),
		type,
		content: '',
		properties: type === 'todo' ? { checked: false } : undefined
	};
}

function createEditorStore() {
	const { subscribe, update, set } = writable<EditorState>({
		blocks: [createEmptyBlock()],
		focusedBlockId: null,
		focusedBlockIndex: 0,
		selectionStart: 0,
		selectionEnd: 0,
		isDirty: false,
		isSaving: false,
		lastSavedAt: null,
		showSlashMenu: false,
		slashMenuPosition: null,
		slashMenuQuery: '',
		showAIPanel: false,
		pendingBlockTypeSelection: null
	});

	return {
		subscribe,
		update,

		initialize(blocks: Block[] | null) {
			const editorBlocks: EditorBlock[] =
				blocks && blocks.length > 0
					? blocks.map((b) => ({
							id: b.id || generateBlockId(),
							type: (b.type as BlockType) || 'paragraph',
							content: b.content || '',
							properties: b.properties as EditorBlock['properties'],
							children: b.children as EditorBlock[]
						}))
					: [createEmptyBlock()];

			update((s) => ({
				...s,
				blocks: editorBlocks,
				focusedBlockId: editorBlocks[0]?.id || null,
				focusedBlockIndex: 0,
				isDirty: false
			}));
		},

		setBlocks(blocks: EditorBlock[]) {
			update((s) => ({ ...s, blocks, isDirty: true }));
		},

		updateBlock(id: string, content: string, properties?: EditorBlock['properties']) {
			update((s) => ({
				...s,
				blocks: s.blocks.map((b) =>
					b.id === id ? { ...b, content, properties: properties ?? b.properties } : b
				),
				isDirty: true
			}));
		},

		addBlockAfter(afterId: string, type: BlockType = 'paragraph'): string {
			const newBlock = createEmptyBlock(type);
			update((s) => {
				const index = s.blocks.findIndex((b) => b.id === afterId);
				const newBlocks = [...s.blocks];
				newBlocks.splice(index + 1, 0, newBlock);
				return {
					...s,
					blocks: newBlocks,
					focusedBlockId: newBlock.id,
					focusedBlockIndex: index + 1,
					isDirty: true
				};
			});
			return newBlock.id;
		},

		addBlockBefore(beforeId: string, type: BlockType = 'paragraph'): string {
			const newBlock = createEmptyBlock(type);
			update((s) => {
				const index = s.blocks.findIndex((b) => b.id === beforeId);
				const newBlocks = [...s.blocks];
				newBlocks.splice(index, 0, newBlock);
				return {
					...s,
					blocks: newBlocks,
					focusedBlockId: newBlock.id,
					focusedBlockIndex: index,
					isDirty: true
				};
			});
			return newBlock.id;
		},

		deleteBlock(id: string) {
			update((s) => {
				if (s.blocks.length <= 1) {
					// Don't delete the last block, just clear it
					return {
						...s,
						blocks: [createEmptyBlock()],
						focusedBlockIndex: 0,
						isDirty: true
					};
				}
				const index = s.blocks.findIndex((b) => b.id === id);
				const newBlocks = s.blocks.filter((b) => b.id !== id);
				const newFocusIndex = Math.min(index, newBlocks.length - 1);
				return {
					...s,
					blocks: newBlocks,
					focusedBlockId: newBlocks[newFocusIndex]?.id || null,
					focusedBlockIndex: newFocusIndex,
					isDirty: true
				};
			});
		},

		changeBlockType(id: string, newType: BlockType) {
			update((s) => ({
				...s,
				blocks: s.blocks.map((b) =>
					b.id === id
						? {
								...b,
								type: newType,
								properties:
									newType === 'todo' ? { ...b.properties, checked: false } : b.properties
							}
						: b
				),
				isDirty: true,
				showSlashMenu: false,
				slashMenuQuery: ''
			}));
		},

		moveBlockUp(id: string) {
			update((s) => {
				const index = s.blocks.findIndex((b) => b.id === id);
				if (index <= 0) return s;
				const newBlocks = [...s.blocks];
				[newBlocks[index - 1], newBlocks[index]] = [newBlocks[index], newBlocks[index - 1]];
				return { ...s, blocks: newBlocks, focusedBlockIndex: index - 1, isDirty: true };
			});
		},

		moveBlockDown(id: string) {
			update((s) => {
				const index = s.blocks.findIndex((b) => b.id === id);
				if (index >= s.blocks.length - 1) return s;
				const newBlocks = [...s.blocks];
				[newBlocks[index], newBlocks[index + 1]] = [newBlocks[index + 1], newBlocks[index]];
				return { ...s, blocks: newBlocks, focusedBlockIndex: index + 1, isDirty: true };
			});
		},

		setFocusedBlock(id: string | null) {
			update((s) => {
				const index = id ? s.blocks.findIndex((b) => b.id === id) : 0;
				return { ...s, focusedBlockId: id, focusedBlockIndex: index >= 0 ? index : 0 };
			});
		},

		focusNextBlock() {
			update((s) => {
				const newIndex = Math.min(s.focusedBlockIndex + 1, s.blocks.length - 1);
				return {
					...s,
					focusedBlockIndex: newIndex,
					focusedBlockId: s.blocks[newIndex]?.id || null
				};
			});
		},

		focusPreviousBlock() {
			update((s) => {
				const newIndex = Math.max(s.focusedBlockIndex - 1, 0);
				return {
					...s,
					focusedBlockIndex: newIndex,
					focusedBlockId: s.blocks[newIndex]?.id || null
				};
			});
		},

		showSlashMenu(position: { x: number; y: number }) {
			update((s) => ({
				...s,
				showSlashMenu: true,
				slashMenuPosition: position,
				slashMenuQuery: ''
			}));
		},

		hideSlashMenu() {
			update((s) => ({
				...s,
				showSlashMenu: false,
				slashMenuPosition: null,
				slashMenuQuery: ''
			}));
		},

		setSlashMenuQuery(query: string) {
			update((s) => ({ ...s, slashMenuQuery: query }));
		},

		// Set pending block type selection (BlockMenu calls this, Block.svelte handles it)
		selectBlockType(type: BlockType, properties?: Record<string, unknown>) {
			update((s) => ({
				...s,
				pendingBlockTypeSelection: s.focusedBlockId ? { type, blockId: s.focusedBlockId, properties } : null,
				showSlashMenu: false,
				slashMenuPosition: null,
				slashMenuQuery: ''
			}));
		},

		// Clear pending selection after Block.svelte handles it
		clearPendingBlockTypeSelection() {
			update((s) => ({ ...s, pendingBlockTypeSelection: null }));
		},

		toggleAIPanel() {
			update((s) => ({ ...s, showAIPanel: !s.showAIPanel }));
		},

		showAIPanel() {
			update((s) => ({ ...s, showAIPanel: true }));
		},

		hideAIPanel() {
			update((s) => ({ ...s, showAIPanel: false }));
		},

		setSaving(isSaving: boolean) {
			update((s) => ({ ...s, isSaving }));
		},

		markSaved() {
			update((s) => ({ ...s, isDirty: false, isSaving: false, lastSavedAt: new Date() }));
		},

		toggleTodo(id: string) {
			update((s) => ({
				...s,
				blocks: s.blocks.map((b) =>
					b.id === id && b.type === 'todo'
						? { ...b, properties: { ...b.properties, checked: !b.properties?.checked } }
						: b
				),
				isDirty: true
			}));
		},

		toggleToggleBlock(id: string) {
			update((s) => ({
				...s,
				blocks: s.blocks.map((b) =>
					b.id === id && b.type === 'toggle'
						? { ...b, properties: { ...b.properties, expanded: !b.properties?.expanded } }
						: b
				),
				isDirty: true
			}));
		},

		getBlocks(): EditorBlock[] {
			let currentBlocks: EditorBlock[] = [];
			subscribe((s) => {
				currentBlocks = s.blocks;
			})();
			return currentBlocks;
		},

		reset() {
			set({
				blocks: [createEmptyBlock()],
				focusedBlockId: null,
				focusedBlockIndex: 0,
				selectionStart: 0,
				selectionEnd: 0,
				isDirty: false,
				isSaving: false,
				lastSavedAt: null,
				showSlashMenu: false,
				slashMenuPosition: null,
				slashMenuQuery: '',
				showAIPanel: false,
				pendingBlockTypeSelection: null
			});
		}
	};
}

export const editor = createEditorStore();

// Derived store for word count
export const wordCount = derived(editor, ($editor) => {
	return $editor.blocks.reduce((count, block) => {
		if (block.content) {
			return count + block.content.trim().split(/\s+/).filter(Boolean).length;
		}
		return count;
	}, 0);
});

// Block type definitions for slash menu with sections
export interface BlockTypeDefinition {
	type: BlockType;
	label: string;
	description: string;
	icon: string;
	keyboardShortcut?: string;
	searchAliases?: string[];
	section: 'suggested' | 'basic';
}

export const blockTypes: BlockTypeDefinition[] = [
	// SUGGESTED SECTION
	{
		type: 'page',
		label: 'Page',
		description: 'Embed a sub-page',
		icon: 'file-text',
		section: 'suggested',
		searchAliases: ['subpage', 'nested', 'link']
	},
	{
		type: 'divider',
		label: 'Divider',
		description: 'Visual separator',
		icon: 'minus',
		keyboardShortcut: '---',
		section: 'suggested',
		searchAliases: ['line', 'hr', 'separator']
	},
	{
		type: 'callout',
		label: 'Callout',
		description: 'Highlighted info box',
		icon: 'alert-circle',
		section: 'suggested',
		searchAliases: ['highlight', 'box', 'notice', 'info']
	},

	// BASIC BLOCKS SECTION
	{
		type: 'paragraph',
		label: 'Text',
		description: 'Plain text block',
		icon: 'type',
		section: 'basic',
		searchAliases: ['paragraph', 'p']
	},
	{
		type: 'heading1',
		label: 'Heading 1',
		description: 'Large section heading',
		icon: 'heading-1',
		keyboardShortcut: '#',
		section: 'basic',
		searchAliases: ['h1', 'title']
	},
	{
		type: 'heading2',
		label: 'Heading 2',
		description: 'Medium section heading',
		icon: 'heading-2',
		keyboardShortcut: '##',
		section: 'basic',
		searchAliases: ['h2', 'subtitle']
	},
	{
		type: 'heading3',
		label: 'Heading 3',
		description: 'Small section heading',
		icon: 'heading-3',
		keyboardShortcut: '###',
		section: 'basic',
		searchAliases: ['h3']
	},
	{
		type: 'bulletList',
		label: 'Bulleted list',
		description: 'Simple bullet points',
		icon: 'list',
		keyboardShortcut: '-',
		section: 'basic',
		searchAliases: ['ul', 'bullet', 'unordered']
	},
	{
		type: 'numberedList',
		label: 'Numbered list',
		description: 'Ordered list with numbers',
		icon: 'list-ordered',
		keyboardShortcut: '1.',
		section: 'basic',
		searchAliases: ['ol', 'ordered', 'number']
	},
	{
		type: 'todo',
		label: 'To-do list',
		description: 'Checklist with checkboxes',
		icon: 'check-square',
		keyboardShortcut: '[]',
		section: 'basic',
		searchAliases: ['checkbox', 'task', 'checklist']
	},
	{
		type: 'toggle',
		label: 'Toggle list',
		description: 'Collapsible content',
		icon: 'chevron-right',
		keyboardShortcut: '>',
		section: 'basic',
		searchAliases: ['collapse', 'expand', 'accordion', 'dropdown']
	},
	{
		type: 'quote',
		label: 'Quote',
		description: 'Block quote for citations',
		icon: 'quote',
		keyboardShortcut: '"',
		section: 'basic',
		searchAliases: ['blockquote', 'citation']
	},
	{
		type: 'code',
		label: 'Code',
		description: 'Code snippet with syntax',
		icon: 'code',
		keyboardShortcut: '```',
		section: 'basic',
		searchAliases: ['snippet', 'programming', 'pre']
	},
	// ADVANCED BLOCKS (in suggested section)
	{
		type: 'tableOfContents',
		label: 'Table of contents',
		description: 'Auto-generated from headings',
		icon: 'list',
		section: 'suggested',
		searchAliases: ['toc', 'outline', 'navigation', 'index']
	},
	{
		type: 'columns',
		label: 'Columns',
		description: 'Side-by-side layout',
		icon: 'columns',
		section: 'suggested',
		searchAliases: ['layout', 'side', 'grid', 'split']
	},
	{
		type: 'bookmark',
		label: 'Bookmark',
		description: 'Embed a link preview',
		icon: 'link',
		section: 'suggested',
		searchAliases: ['link', 'url', 'embed', 'preview']
	}
];

// Priority-based filtering for slash menu
// Prioritizes: exact match > starts with > contains
export function filterBlockTypes(
	query: string,
	types: BlockTypeDefinition[] = blockTypes
): BlockTypeDefinition[] {
	if (!query || query.trim() === '') return types;
	const q = query.toLowerCase().trim();

	const scored = types
		.map((bt) => {
			const label = bt.label.toLowerCase();
			const type = bt.type.toLowerCase();
			const aliases = bt.searchAliases?.map((a) => a.toLowerCase()) || [];

			let score = 0;

			// Priority 1: Exact match (highest)
			if (label === q || type === q) {
				score = 100;
			}
			// Priority 2: Label starts with query
			else if (label.startsWith(q)) {
				score = 80;
			}
			// Priority 3: Type starts with query
			else if (type.startsWith(q)) {
				score = 70;
			}
			// Priority 4: Any alias starts with query
			else if (aliases.some((a) => a.startsWith(q))) {
				score = 60;
			}
			// Priority 5: Label contains query
			else if (label.includes(q)) {
				score = 40;
			}
			// Priority 6: Any alias contains query
			else if (aliases.some((a) => a.includes(q))) {
				score = 30;
			}
			// No match
			else {
				score = 0;
			}

			return { bt, score };
		})
		.filter((r) => r.score > 0)
		.sort((a, b) => b.score - a.score);

	return scored.map((r) => r.bt);
}

// Helper to get block types by section
export function getBlockTypesBySection(types: BlockTypeDefinition[] = blockTypes) {
	return {
		suggested: types.filter((bt) => bt.section === 'suggested'),
		basic: types.filter((bt) => bt.section === 'basic')
	};
}
