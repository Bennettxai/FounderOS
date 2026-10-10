/**
 * Knowledge Base Service
 *
 * Central service for the Knowledge Base module.
 * Manages Pages (documents) and their integration with the OS Node graph.
 *
 * TAXONOMY:
 * - Page = Document in Knowledge Base (with Blocks)
 * - Node = Building block of the Operating System graph
 * - Pages can LINK to Nodes (via node_id)
 * - Context Node = A type of Node for pure reference material
 */

import { writable, derived, get, type Readable, type Writable } from 'svelte/store';
import type { BlockSnapshot, Block, BlockFlavour } from '../entities/block';
import type {
	Context as PageData,
	ContextListItem as PageListItem,
	CreateContextData as CreatePageData,
	UpdateContextData as UpdatePageData
} from '$lib/api/contexts/types';
import type { Node, NodeTree, NodeDetail } from '$lib/api/nodes/types';
import * as pagesApi from '$lib/api/contexts'; // API still uses 'contexts' endpoint
import * as nodesApi from '$lib/api/nodes';
import {
	pageToBlocks,
	blocksToPage,
	nodeTreeToDocumentTree,
	getPageIcon
} from './page-adapter';
import { createYjsDocStore, createBlockStore, type BlockStore } from '../stores/yjs-block-store';

// ============================================================================
// Types
// ============================================================================

/**
 * Represents a document in the Knowledge Base
 */
export interface KBDocument {
	id: string;
	title: string;
	icon: string | null;
	type: 'page' | 'node-folder';
	pageType?: string; // profile, reference, template, note, document
	parentId: string | null;
	nodeId: string | null; // Link to OS Node
	isTemplate?: boolean;
	isFavorite: boolean;
	isArchived: boolean;
	createdAt: string;
	updatedAt: string;
	children: KBDocument[];
}

export interface KnowledgeBaseState {
	documents: KBDocument[];
	activeDocumentId: string | null;
	loading: boolean;
	error: string | null;
}

export interface LoadedPage {
	id: string;
	blocks: BlockSnapshot[];
	blockStore: BlockStore;
	pageData: PageData | null;
	dirty: boolean;
}

// ============================================================================
// Knowledge Base Store
// ============================================================================

function createKnowledgeBaseStore() {
	const state = writable<KnowledgeBaseState>({
		documents: [],
		activeDocumentId: null,
		loading: false,
		error: null
	});

	// Cache for loaded pages
	const loadedPages = new Map<string, LoadedPage>();

	// Active page store
	const activePage = writable<LoadedPage | null>(null);

	/**
	 * Load all pages and nodes to build the document tree
	 */
	async function loadDocumentTree(): Promise<void> {
		state.update((s) => ({ ...s, loading: true, error: null }));

		try {
			// Load pages and nodes in parallel
			const [pages, nodesResult] = await Promise.all([
				pagesApi.getContexts(), // Returns ContextListItem[]
				nodesApi.getNodeTree()
			]);

			// Build document tree from pages
			const pageDocs = buildPageDocuments(pages);

			// Build node hierarchy (nodes can contain pages)
			const nodeDocs = buildNodeDocuments(nodesResult);

			// Merge: pages go under their linked nodes, orphans go to root
			const documents = mergeDocumentTrees(pageDocs, nodeDocs);

			state.update((s) => ({
				...s,
				documents,
				loading: false
			}));
		} catch (error) {
			state.update((s) => ({
				...s,
				loading: false,
				error: error instanceof Error ? error.message : 'Failed to load documents'
			}));
		}
	}

	/**
	 * Build document tree from pages
	 */
	function buildPageDocuments(pages: PageListItem[]): KBDocument[] {
		const pageMap = new Map<string, KBDocument>();
		const rootPages: KBDocument[] = [];

		// First pass: create all documents
		// Note: ContextListItem doesn't have node_id or created_at - those are only on full Context
		for (const page of pages) {
			const doc: KBDocument = {
				id: page.id,
				title: page.name,
				icon: page.icon || getPageIcon(page.type),
				type: 'page',
				pageType: page.type,
				parentId: page.parent_id ?? null,
				nodeId: null, // Not available in list item
				isTemplate: page.is_template,
				isFavorite: false,
				isArchived: page.is_archived,
				createdAt: page.updated_at, // Use updated_at as fallback
				updatedAt: page.updated_at,
				children: []
			};
			pageMap.set(page.id, doc);
		}

		// Second pass: build tree structure
		for (const doc of pageMap.values()) {
			if (doc.parentId && pageMap.has(doc.parentId)) {
				pageMap.get(doc.parentId)!.children.push(doc);
			} else {
				// Add to root (no node linking for now)
				rootPages.push(doc);
			}
		}

		return rootPages;
	}

	/**
	 * Build document tree from nodes (OS layer)
	 */
	function buildNodeDocuments(nodes: NodeTree[]): KBDocument[] {
		return nodes.map((node) => ({
			id: `node:${node.id}`,
			title: node.name,
			icon: getNodeIcon(node.type),
			type: 'node-folder' as const,
			parentId: null,
			nodeId: node.id,
			isFavorite: false,
			isArchived: false,
			createdAt: node.created_at,
			updatedAt: node.updated_at,
			children: buildNodeDocuments(node.children)
		}));
	}

	/**
	 * Get icon for node type
	 */
	function getNodeIcon(type: string): string {
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
			context: '📋'
		};
		return icons[type] ?? '📄';
	}

	/**
	 * Merge page and node trees
	 * Pages linked to nodes appear under their nodes
	 */
	function mergeDocumentTrees(
		pageDocs: KBDocument[],
		nodeDocs: KBDocument[]
	): KBDocument[] {
		// For now, return nodes at top level with their pages
		// Pages without nodes go at root level
		return [...nodeDocs, ...pageDocs];
	}

	/**
	 * Open a page for editing
	 */
	async function openPage(pageId: string): Promise<LoadedPage> {
		// Check cache first
		if (loadedPages.has(pageId)) {
			const page = loadedPages.get(pageId)!;
			activePage.set(page);
			state.update((s) => ({ ...s, activeDocumentId: pageId }));
			return page;
		}

		state.update((s) => ({ ...s, loading: true }));

		try {
			// Fetch page from API
			const pageData = await pagesApi.getContext(pageId);

			// Convert to blocks
			const blocks = pageToBlocks(pageData);

			// Create Yjs-backed block store
			const yjsDoc = createYjsDocStore(pageId);
			const blockStore = createBlockStore(yjsDoc);

			// Initialize blocks in store
			initializeBlocks(blockStore, blocks);

			const loadedPage: LoadedPage = {
				id: pageId,
				blocks,
				blockStore,
				pageData,
				dirty: false
			};

			loadedPages.set(pageId, loadedPage);
			activePage.set(loadedPage);
			state.update((s) => ({ ...s, activeDocumentId: pageId, loading: false }));

			return loadedPage;
		} catch (error) {
			state.update((s) => ({
				...s,
				loading: false,
				error: error instanceof Error ? error.message : 'Failed to load page'
			}));
			throw error;
		}
	}

	/**
	 * Initialize blocks in the block store
	 */
	function initializeBlocks(blockStore: BlockStore, blocks: BlockSnapshot[]): void {
		// Find the page block
		const pageBlock = blocks.find((b) => b.flavour === 'bos:page');
		if (!pageBlock) return;

		// Add page block first
		// Note: props may need conversion between serialized and live formats
		blockStore.addBlock('bos:page', pageBlock.props as any, undefined);

		// Add children recursively
		function addChildren(parentId: string, childIds: string[]) {
			for (const childId of childIds) {
				const block = blocks.find((b) => b.id === childId);
				if (!block) continue;

				blockStore.addBlock(block.flavour, block.props as any, parentId);

				if (block.children.length > 0) {
					addChildren(block.id, block.children);
				}
			}
		}

		addChildren(pageBlock.id, pageBlock.children);
	}

	/**
	 * Save the active page
	 */
	async function savePage(pageId?: string): Promise<void> {
		const id = pageId ?? get(state).activeDocumentId;
		if (!id) return;

		const loadedPage = loadedPages.get(id);
		if (!loadedPage) return;

		state.update((s) => ({ ...s, loading: true }));

		try {
			// Get all blocks from store
			const allBlocks = get(loadedPage.blockStore);

			// Convert to page format
			const updateData = blocksToPage(Object.values(allBlocks) as BlockSnapshot[], id);

			// Save via API
			await pagesApi.updateContext(id, updateData);

			// Mark as clean
			loadedPage.dirty = false;

			state.update((s) => ({ ...s, loading: false }));
		} catch (error) {
			state.update((s) => ({
				...s,
				loading: false,
				error: error instanceof Error ? error.message : 'Failed to save page'
			}));
			throw error;
		}
	}

	/**
	 * Create a new page
	 * Note: nodeId is reserved for future use when the API supports it
	 */
	async function createPage(
		data: Partial<CreatePageData>,
		parentId?: string,
		_nodeId?: string // Reserved for future API support
	): Promise<string> {
		state.update((s) => ({ ...s, loading: true }));

		try {
			const createData: CreatePageData = {
				name: data.name ?? 'Untitled',
				type: data.type ?? 'document',
				icon: data.icon,
				parent_id: parentId,
				blocks: data.blocks ?? []
			};

			const pageData = await pagesApi.createContext(createData);

			// Reload tree to include new page
			await loadDocumentTree();

			state.update((s) => ({ ...s, loading: false }));

			return pageData.id;
		} catch (error) {
			state.update((s) => ({
				...s,
				loading: false,
				error: error instanceof Error ? error.message : 'Failed to create page'
			}));
			throw error;
		}
	}

	/**
	 * Delete a page
	 */
	async function deletePage(pageId: string): Promise<void> {
		state.update((s) => ({ ...s, loading: true }));

		try {
			await pagesApi.deleteContext(pageId);

			// Remove from cache
			loadedPages.delete(pageId);

			// Clear active if deleted
			const currentState = get(state);
			if (currentState.activeDocumentId === pageId) {
				activePage.set(null);
			}

			// Reload tree
			await loadDocumentTree();

			state.update((s) => ({ ...s, loading: false }));
		} catch (error) {
			state.update((s) => ({
				...s,
				loading: false,
				error: error instanceof Error ? error.message : 'Failed to delete page'
			}));
			throw error;
		}
	}

	/**
	 * Close a page (remove from cache)
	 */
	function closePage(pageId: string): void {
		loadedPages.delete(pageId);

		const currentState = get(state);
		if (currentState.activeDocumentId === pageId) {
			activePage.set(null);
			state.update((s) => ({ ...s, activeDocumentId: null }));
		}
	}

	/**
	 * Move a page to a new parent
	 * Note: nodeId is reserved for future use when the API supports it
	 */
	async function movePage(
		pageId: string,
		newParentId: string | null,
		_newNodeId?: string // Reserved for future API support
	): Promise<void> {
		state.update((s) => ({ ...s, loading: true }));

		try {
			await pagesApi.updateContext(pageId, {
				parent_id: newParentId
			});

			// Reload tree
			await loadDocumentTree();

			state.update((s) => ({ ...s, loading: false }));
		} catch (error) {
			state.update((s) => ({
				...s,
				loading: false,
				error: error instanceof Error ? error.message : 'Failed to move page'
			}));
			throw error;
		}
	}

	/**
	 * Search pages
	 * Uses the API's search filter parameter
	 */
	async function searchPages(query: string): Promise<KBDocument[]> {
		try {
			// Use the search parameter in the API
			const pages = await pagesApi.getContexts({ search: query });

			return pages.map((page) => ({
				id: page.id,
				title: page.name,
				icon: page.icon || getPageIcon(page.type),
				type: 'page' as const,
				pageType: page.type,
				parentId: page.parent_id ?? null,
				nodeId: null,
				isTemplate: page.is_template,
				isFavorite: false,
				isArchived: page.is_archived,
				createdAt: page.updated_at,
				updatedAt: page.updated_at,
				children: []
			}));
		} catch (error) {
			console.error('Search failed:', error);
			return [];
		}
	}

	/**
	 * Toggle favorite status
	 */
	async function toggleFavorite(pageId: string): Promise<void> {
		// Get current state
		const doc = getDocumentById(get(state).documents, pageId);
		if (!doc) return;

		// For now, we'll track favorites client-side
		// Later this should persist to the backend
		state.update((s) => {
			const updateDoc = (docs: KBDocument[]): KBDocument[] => {
				return docs.map((d) => {
					if (d.id === pageId) {
						return { ...d, isFavorite: !d.isFavorite };
					}
					if (d.children.length > 0) {
						return { ...d, children: updateDoc(d.children) };
					}
					return d;
				});
			};
			return { ...s, documents: updateDoc(s.documents) };
		});
	}

	// Derived stores
	const documents = derived(state, ($state) => $state.documents);
	const loading = derived(state, ($state) => $state.loading);
	const error = derived(state, ($state) => $state.error);
	const activeDocumentId = derived(state, ($state) => $state.activeDocumentId);

	// Favorites derived
	const favorites = derived(state, ($state) =>
		flattenDocuments($state.documents).filter((d) => d.isFavorite)
	);

	// Recent pages (by updated date)
	const recent = derived(state, ($state) =>
		flattenDocuments($state.documents)
			.filter((d) => d.type === 'page')
			.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
			.slice(0, 10)
	);

	// Templates
	const templates = derived(state, ($state) =>
		flattenDocuments($state.documents).filter((d) => d.isTemplate)
	);

	return {
		subscribe: state.subscribe,
		documents,
		loading,
		error,
		activeDocumentId,
		activePage: { subscribe: activePage.subscribe },
		favorites,
		recent,
		templates,
		loadDocumentTree,
		openPage,
		savePage,
		createPage,
		deletePage,
		closePage,
		movePage,
		searchPages,
		toggleFavorite
	};
}

/**
 * Flatten document tree into array
 */
function flattenDocuments(docs: KBDocument[]): KBDocument[] {
	const result: KBDocument[] = [];

	function traverse(documents: KBDocument[]) {
		for (const doc of documents) {
			result.push(doc);
			if (doc.children.length > 0) {
				traverse(doc.children);
			}
		}
	}

	traverse(docs);
	return result;
}

// ============================================================================
// Singleton Export
// ============================================================================

export const knowledgeBase = createKnowledgeBaseStore();

// ============================================================================
// Convenience Functions
// ============================================================================

/**
 * Initialize knowledge base on app start
 */
export async function initializeKnowledgeBase(): Promise<void> {
	await knowledgeBase.loadDocumentTree();
}

/**
 * Get document by ID from tree
 */
export function getDocumentById(
	documents: KBDocument[],
	id: string
): KBDocument | null {
	for (const doc of documents) {
		if (doc.id === id) return doc;
		if (doc.children.length > 0) {
			const found = getDocumentById(doc.children, id);
			if (found) return found;
		}
	}
	return null;
}

/**
 * Get document path (breadcrumb)
 */
export function getDocumentPath(
	documents: KBDocument[],
	id: string
): KBDocument[] {
	const path: KBDocument[] = [];

	function findPath(docs: KBDocument[], target: string): boolean {
		for (const doc of docs) {
			if (doc.id === target) {
				path.push(doc);
				return true;
			}
			if (doc.children.length > 0 && findPath(doc.children, target)) {
				path.unshift(doc);
				return true;
			}
		}
		return false;
	}

	findPath(documents, id);
	return path;
}

// ============================================================================
// Re-exports for backwards compatibility
// ============================================================================

/** @deprecated Use knowledgeBase instead */
export const knowledgeGraph = knowledgeBase;

/** @deprecated Use initializeKnowledgeBase instead */
export const initializeKnowledgeGraph = initializeKnowledgeBase;

/** @deprecated Use KBDocument instead */
export type KnowledgeGraphDocument = KBDocument;

/** @deprecated Use KnowledgeBaseState instead */
export type KnowledgeGraphState = KnowledgeBaseState;

/** @deprecated Use LoadedPage instead */
export type LoadedDocument = LoadedPage;
