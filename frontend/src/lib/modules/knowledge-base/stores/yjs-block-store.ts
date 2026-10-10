/**
 * Yjs Block Store - CRDT-backed Block Management
 *
 * Provides reactive Svelte stores backed by Yjs for real-time collaboration.
 * Blocks are stored in a Y.Map, children in Y.Array, text in Y.Text.
 *
 * Key features:
 * - Automatic Yjs <-> Svelte store sync
 * - Local-first with offline support
 * - Real-time collaboration via Yjs providers
 * - Undo/redo support
 */

import * as Y from 'yjs';
import { IndexeddbPersistence } from 'y-indexeddb';
import { writable, derived, get, type Readable, type Writable } from 'svelte/store';
import {
	type Block,
	type BlockFlavour,
	type BlockProps,
	type BlockSnapshot,
	type TextDelta,
	type BlockSysProps,
	generateBlockId,
	getDefaultProps,
	type BlockCollection,
	type YTextRef
} from '../entities/block';

// ============================================================================
// Yjs Document Store
// ============================================================================

interface YjsDocState {
	/** The Yjs document */
	doc: Y.Doc;
	/** IndexedDB persistence */
	persistence: IndexeddbPersistence | null;
	/** Whether the doc is loaded from persistence */
	loaded: boolean;
	/** Connection status (for remote providers) */
	connected: boolean;
	/** Undo manager */
	undoManager: Y.UndoManager | null;
}

/**
 * Create a Yjs-backed store for a document.
 */
export function createYjsDocStore(docId: string) {
	const doc = new Y.Doc({ guid: docId });

	const { subscribe, set, update } = writable<YjsDocState>({
		doc,
		persistence: null,
		loaded: false,
		connected: false,
		undoManager: null
	});

	// Y.Map for blocks: blockId -> block data
	const blocksMap = doc.getMap<Y.Map<unknown>>('blocks');

	// Initialize persistence
	let persistence: IndexeddbPersistence | null = null;

	return {
		subscribe,

		/**
		 * Initialize the document with IndexedDB persistence
		 */
		async init() {
			persistence = new IndexeddbPersistence(`bos-doc-${docId}`, doc);

			persistence.on('synced', () => {
				update((s) => ({ ...s, loaded: true }));
			});

			update((s) => ({ ...s, persistence }));

			// Wait for initial sync
			await new Promise<void>((resolve) => {
				if (persistence!.synced) {
					resolve();
				} else {
					persistence!.once('synced', () => resolve());
				}
			});
		},

		/**
		 * Setup undo manager for the blocks map
		 */
		setupUndoManager() {
			const undoManager = new Y.UndoManager(blocksMap, {
				captureTimeout: 500
			});
			update((s) => ({ ...s, undoManager }));
			return undoManager;
		},

		/**
		 * Get the underlying Y.Doc
		 */
		getDoc() {
			return doc;
		},

		/**
		 * Get the blocks Y.Map
		 */
		getBlocksMap() {
			return blocksMap;
		},

		/**
		 * Destroy the store and clean up
		 */
		destroy() {
			const state = get({ subscribe });
			state.undoManager?.destroy();
			state.persistence?.destroy();
			doc.destroy();
		}
	};
}

// ============================================================================
// Block Store (Reactive Svelte Store backed by Yjs)
// ============================================================================

interface BlockStoreState {
	blocks: Map<string, Block>;
	rootId: string | null;
	loading: boolean;
	error: string | null;
}

/**
 * Create a reactive block store backed by Yjs.
 * Automatically syncs changes from Yjs to Svelte stores.
 */
export function createBlockStore(yjsDocStore: ReturnType<typeof createYjsDocStore>) {
	const doc = yjsDocStore.getDoc();
	const blocksMap = yjsDocStore.getBlocksMap();

	const { subscribe, set, update } = writable<BlockStoreState>({
		blocks: new Map(),
		rootId: null,
		loading: true,
		error: null
	});

	// Sync Yjs -> Svelte store
	function syncFromYjs() {
		const blocks = new Map<string, Block>();
		let rootId: string | null = null;

		blocksMap.forEach((yBlock, id) => {
			if (yBlock instanceof Y.Map) {
				const block = yMapToBlock(yBlock, id);
				blocks.set(id, block);
				if (block.parent === null && block.flavour === 'bos:page') {
					rootId = id;
				}
			}
		});

		update((s) => ({ ...s, blocks, rootId, loading: false }));
	}

	// Listen for Yjs changes
	blocksMap.observeDeep(() => {
		syncFromYjs();
	});

	// Initial sync
	syncFromYjs();

	return {
		subscribe,

		/**
		 * Get a block by ID
		 */
		getBlock(id: string): Block | null {
			return get({ subscribe }).blocks.get(id) ?? null;
		},

		/**
		 * Add a new block
		 */
		addBlock<F extends BlockFlavour>(
			flavour: F,
			props?: Partial<BlockProps<F>>,
			parentId?: string,
			index?: number
		): Block<F> {
			const id = generateBlockId();
			const now = new Date().toISOString();

			const defaultProps = getDefaultProps(flavour);
			const mergedProps = { ...defaultProps, ...props };

			const block: Block<F> = {
				id,
				flavour,
				parent: parentId ?? null,
				children: [],
				version: 1,
				props: mergedProps as BlockProps<F>,
				createdAt: now,
				updatedAt: now
			};

			doc.transact(() => {
				// Create Y.Map for the block
				const yBlock = blockToYMap(block);
				blocksMap.set(id, yBlock);

				// Add to parent's children
				if (parentId) {
					const parentYBlock = blocksMap.get(parentId);
					if (parentYBlock instanceof Y.Map) {
						const children = parentYBlock.get('children') as Y.Array<string>;
						if (children) {
							if (index !== undefined && index >= 0 && index < children.length) {
								children.insert(index, [id]);
							} else {
								children.push([id]);
							}
						}
					}
				}
			});

			return block;
		},

		/**
		 * Update block props
		 */
		updateBlock<F extends BlockFlavour>(id: string, props: Partial<BlockProps<F>>) {
			const yBlock = blocksMap.get(id);
			if (!(yBlock instanceof Y.Map)) return;

			doc.transact(() => {
				const yProps = yBlock.get('props');
				if (yProps instanceof Y.Map) {
					for (const [key, value] of Object.entries(props)) {
						if (value !== undefined) {
							yProps.set(key, serializeValue(value));
						}
					}
				}
				yBlock.set('updatedAt', new Date().toISOString());
				yBlock.set('version', (yBlock.get('version') as number ?? 0) + 1);
			});
		},

		/**
		 * Update block text (Y.Text)
		 */
		updateBlockText(id: string, propKey: string, delta: TextDelta[]) {
			const yBlock = blocksMap.get(id);
			if (!(yBlock instanceof Y.Map)) return;

			doc.transact(() => {
				const yProps = yBlock.get('props');
				if (yProps instanceof Y.Map) {
					let yText = yProps.get(propKey);
					if (!(yText instanceof Y.Text)) {
						yText = new Y.Text();
						yProps.set(propKey, yText);
					}
					// Clear and apply delta
					(yText as Y.Text).delete(0, (yText as Y.Text).length);
					(yText as Y.Text).applyDelta(delta);
				}
				yBlock.set('updatedAt', new Date().toISOString());
			});
		},

		/**
		 * Delete a block
		 */
		deleteBlock(id: string) {
			const block = this.getBlock(id);
			if (!block) return;

			doc.transact(() => {
				// Remove from parent's children
				if (block.parent) {
					const parentYBlock = blocksMap.get(block.parent);
					if (parentYBlock instanceof Y.Map) {
						const children = parentYBlock.get('children') as Y.Array<string>;
						if (children) {
							const idx = children.toArray().indexOf(id);
							if (idx >= 0) {
								children.delete(idx, 1);
							}
						}
					}
				}

				// Recursively delete children
				for (const childId of block.children) {
					this.deleteBlock(childId);
				}

				// Delete the block itself
				blocksMap.delete(id);
			});
		},

		/**
		 * Move block to new parent/position
		 */
		moveBlock(id: string, newParentId: string, index?: number) {
			const block = this.getBlock(id);
			if (!block) return;

			doc.transact(() => {
				// Remove from current parent
				if (block.parent) {
					const oldParentYBlock = blocksMap.get(block.parent);
					if (oldParentYBlock instanceof Y.Map) {
						const children = oldParentYBlock.get('children') as Y.Array<string>;
						if (children) {
							const idx = children.toArray().indexOf(id);
							if (idx >= 0) {
								children.delete(idx, 1);
							}
						}
					}
				}

				// Add to new parent
				const newParentYBlock = blocksMap.get(newParentId);
				if (newParentYBlock instanceof Y.Map) {
					const children = newParentYBlock.get('children') as Y.Array<string>;
					if (children) {
						if (index !== undefined && index >= 0 && index <= children.length) {
							children.insert(index, [id]);
						} else {
							children.push([id]);
						}
					}
				}

				// Update block's parent
				const yBlock = blocksMap.get(id);
				if (yBlock instanceof Y.Map) {
					yBlock.set('parent', newParentId);
					yBlock.set('updatedAt', new Date().toISOString());
				}
			});
		},

		/**
		 * Get children blocks of a parent
		 */
		getChildren(parentId: string): Block[] {
			const parent = this.getBlock(parentId);
			if (!parent) return [];

			const state = get({ subscribe });
			return parent.children
				.map((childId) => state.blocks.get(childId))
				.filter((b): b is Block => b !== undefined);
		},

		/**
		 * Export all blocks to snapshots
		 */
		toSnapshots(): BlockSnapshot[] {
			const state = get({ subscribe });
			const snapshots: BlockSnapshot[] = [];

			for (const [id, block] of state.blocks) {
				snapshots.push(blockToSnapshot(block));
			}

			return snapshots;
		},

		/**
		 * Import blocks from snapshots
		 */
		fromSnapshots(snapshots: BlockSnapshot[]) {
			doc.transact(() => {
				// Clear existing
				blocksMap.forEach((_, key) => blocksMap.delete(key));

				// Import new
				for (const snapshot of snapshots) {
					const block = snapshotToBlock(snapshot);
					const yBlock = blockToYMap(block);
					blocksMap.set(snapshot.id, yBlock);
				}
			});
		},

		/**
		 * Undo last change
		 */
		undo() {
			const state = get(yjsDocStore);
			state.undoManager?.undo();
		},

		/**
		 * Redo last undone change
		 */
		redo() {
			const state = get(yjsDocStore);
			state.undoManager?.redo();
		},

		/**
		 * Check if can undo
		 */
		canUndo(): boolean {
			const state = get(yjsDocStore);
			return (state.undoManager?.undoStack.length ?? 0) > 0;
		},

		/**
		 * Check if can redo
		 */
		canRedo(): boolean {
			const state = get(yjsDocStore);
			return (state.undoManager?.redoStack.length ?? 0) > 0;
		}
	};
}

// ============================================================================
// Derived Stores
// ============================================================================

/**
 * Create a derived store for a specific block
 */
export function createBlockDerived(
	blockStore: ReturnType<typeof createBlockStore>,
	blockId: string
): Readable<Block | null> {
	return derived(blockStore, ($store) => $store.blocks.get(blockId) ?? null);
}

/**
 * Create a derived store for block children
 */
export function createChildrenDerived(
	blockStore: ReturnType<typeof createBlockStore>,
	parentId: string
): Readable<Block[]> {
	return derived(blockStore, ($store) => {
		const parent = $store.blocks.get(parentId);
		if (!parent) return [];

		return parent.children
			.map((childId) => $store.blocks.get(childId))
			.filter((b): b is Block => b !== undefined);
	});
}

/**
 * Create a derived store for the root/page block
 */
export function createRootBlockDerived(
	blockStore: ReturnType<typeof createBlockStore>
): Readable<Block | null> {
	return derived(blockStore, ($store) => {
		if (!$store.rootId) return null;
		return $store.blocks.get($store.rootId) ?? null;
	});
}

// ============================================================================
// Conversion Utilities
// ============================================================================

/**
 * Convert a Block to a Y.Map for storage
 */
function blockToYMap(block: Block): Y.Map<unknown> {
	const yBlock = new Y.Map<unknown>();

	yBlock.set('id', block.id);
	yBlock.set('flavour', block.flavour);
	yBlock.set('parent', block.parent);
	yBlock.set('version', block.version);
	yBlock.set('createdAt', block.createdAt);
	yBlock.set('updatedAt', block.updatedAt);
	if (block.createdBy) yBlock.set('createdBy', block.createdBy);
	if (block.updatedBy) yBlock.set('updatedBy', block.updatedBy);

	// Children as Y.Array
	const yChildren = new Y.Array<string>();
	yChildren.push(block.children);
	yBlock.set('children', yChildren);

	// Props as Y.Map with Y.Text for text fields
	const yProps = new Y.Map<unknown>();
	for (const [key, value] of Object.entries(block.props)) {
		if (isYTextRef(value)) {
			const yText = new Y.Text();
			yText.applyDelta(value.delta);
			yProps.set(key, yText);
		} else {
			yProps.set(key, serializeValue(value));
		}
	}
	yBlock.set('props', yProps);

	return yBlock;
}

/**
 * Convert a Y.Map back to a Block
 */
function yMapToBlock(yBlock: Y.Map<unknown>, id: string): Block {
	const flavour = yBlock.get('flavour') as BlockFlavour;
	const yChildren = yBlock.get('children') as Y.Array<string>;
	const yProps = yBlock.get('props') as Y.Map<unknown>;

	// Convert props, handling Y.Text specially
	const props: Record<string, unknown> = {};
	if (yProps) {
		yProps.forEach((value, key) => {
			if (value instanceof Y.Text) {
				props[key] = {
					delta: value.toDelta(),
					toString: () => value.toString()
				} as YTextRef;
			} else {
				props[key] = value;
			}
		});
	}

	return {
		id,
		flavour,
		parent: yBlock.get('parent') as string | null,
		children: yChildren?.toArray() ?? [],
		version: yBlock.get('version') as number ?? 1,
		props: props as BlockProps<BlockFlavour>,
		createdAt: yBlock.get('createdAt') as string ?? new Date().toISOString(),
		updatedAt: yBlock.get('updatedAt') as string ?? new Date().toISOString(),
		createdBy: yBlock.get('createdBy') as string | undefined,
		updatedBy: yBlock.get('updatedBy') as string | undefined
	};
}

/**
 * Convert a Block to a serializable snapshot
 */
function blockToSnapshot(block: Block): BlockSnapshot {
	const props: Record<string, unknown> = {};

	for (const [key, value] of Object.entries(block.props)) {
		if (isYTextRef(value)) {
			props[key] = value.delta;
		} else {
			props[key] = value;
		}
	}

	return {
		id: block.id,
		flavour: block.flavour,
		parent: block.parent,
		children: [...block.children],
		props: props as BlockSnapshot['props'],
		version: block.version,
		createdAt: block.createdAt,
		updatedAt: block.updatedAt,
		createdBy: block.createdBy,
		updatedBy: block.updatedBy
	};
}

/**
 * Convert a snapshot back to a Block
 */
function snapshotToBlock(snapshot: BlockSnapshot): Block {
	const props: Record<string, unknown> = {};

	for (const [key, value] of Object.entries(snapshot.props)) {
		if (Array.isArray(value) && value.length > 0 && value[0]?.insert !== undefined) {
			// This is a delta array, convert to YTextRef
			props[key] = {
				delta: value as TextDelta[],
				toString: () => (value as TextDelta[]).map((d) => d.insert).join('')
			} as YTextRef;
		} else {
			props[key] = value;
		}
	}

	return {
		id: snapshot.id,
		flavour: snapshot.flavour,
		parent: snapshot.parent,
		children: [...snapshot.children],
		version: snapshot.version,
		props: props as BlockProps<BlockFlavour>,
		createdAt: snapshot.createdAt,
		updatedAt: snapshot.updatedAt,
		createdBy: snapshot.createdBy,
		updatedBy: snapshot.updatedBy
	};
}

/**
 * Check if a value is a YTextRef
 */
function isYTextRef(value: unknown): value is YTextRef {
	return (
		typeof value === 'object' &&
		value !== null &&
		'delta' in value &&
		Array.isArray((value as YTextRef).delta)
	);
}

/**
 * Serialize a value for Y.Map storage
 */
function serializeValue(value: unknown): unknown {
	if (value === null || value === undefined) return value;
	if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
		return value;
	}
	if (Array.isArray(value)) {
		return value.map(serializeValue);
	}
	if (typeof value === 'object') {
		const result: Record<string, unknown> = {};
		for (const [k, v] of Object.entries(value)) {
			result[k] = serializeValue(v);
		}
		return result;
	}
	return value;
}

// ============================================================================
// Active Document Store (Global singleton for current document)
// ============================================================================

interface ActiveDocStoreState {
	docId: string | null;
	yjsStore: ReturnType<typeof createYjsDocStore> | null;
	blockStore: ReturnType<typeof createBlockStore> | null;
}

function createActiveDocStore() {
	const { subscribe, set, update } = writable<ActiveDocStoreState>({
		docId: null,
		yjsStore: null,
		blockStore: null
	});

	return {
		subscribe,

		/**
		 * Open a document
		 */
		async openDocument(docId: string) {
			const state = get({ subscribe });

			// Close existing if different
			if (state.docId && state.docId !== docId) {
				this.closeDocument();
			}

			// Create new stores
			const yjsStore = createYjsDocStore(docId);
			await yjsStore.init();
			yjsStore.setupUndoManager();

			const blockStore = createBlockStore(yjsStore);

			set({
				docId,
				yjsStore,
				blockStore
			});

			return blockStore;
		},

		/**
		 * Close current document
		 */
		closeDocument() {
			const state = get({ subscribe });
			state.yjsStore?.destroy();

			set({
				docId: null,
				yjsStore: null,
				blockStore: null
			});
		},

		/**
		 * Get current block store
		 */
		getBlockStore() {
			return get({ subscribe }).blockStore;
		}
	};
}

export const activeDocStore = createActiveDocStore();

// ============================================================================
// Convenience exports
// ============================================================================

export type YjsDocStore = ReturnType<typeof createYjsDocStore>;
export type BlockStore = ReturnType<typeof createBlockStore>;
