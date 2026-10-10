import { writable, derived, get } from 'svelte/store';
import type { Document, DocumentMeta, TreeNode, SidebarView } from '../entities/types';

// ============================================================================
// Documents Store
// ============================================================================

interface DocumentsState {
	documents: Map<string, Document>;
	documentMetas: Map<string, DocumentMeta>;
	loading: boolean;
	error: string | null;
}

function createDocumentsStore() {
	const { subscribe, set, update } = writable<DocumentsState>({
		documents: new Map(),
		documentMetas: new Map(),
		loading: false,
		error: null
	});

	return {
		subscribe,

		setLoading(loading: boolean) {
			update((state) => ({ ...state, loading }));
		},

		setError(error: string | null) {
			update((state) => ({ ...state, error }));
		},

		setDocument(doc: Document) {
			update((state) => {
				const documents = new Map(state.documents);
				documents.set(doc.id, doc);

				// Also update meta
				const documentMetas = new Map(state.documentMetas);
				documentMetas.set(doc.id, {
					id: doc.id,
					parent_id: doc.parent_id,
					type: doc.type,
					title: doc.title,
					icon: doc.icon,
					is_favorite: doc.is_favorite,
					is_archived: doc.is_archived,
					updated_at: doc.updated_at,
					children_count: 0 // Will be updated separately
				});

				return { ...state, documents, documentMetas };
			});
		},

		setDocuments(docs: Document[]) {
			update((state) => {
				const documents = new Map(state.documents);
				const documentMetas = new Map(state.documentMetas);

				for (const doc of docs) {
					documents.set(doc.id, doc);
					documentMetas.set(doc.id, {
						id: doc.id,
						parent_id: doc.parent_id,
						type: doc.type,
						title: doc.title,
						icon: doc.icon,
						is_favorite: doc.is_favorite,
						is_archived: doc.is_archived,
						updated_at: doc.updated_at,
						children_count: 0
					});
				}

				return { ...state, documents, documentMetas };
			});
		},

		setDocumentMetas(metas: DocumentMeta[]) {
			update((state) => {
				const documentMetas = new Map(state.documentMetas);
				for (const meta of metas) {
					documentMetas.set(meta.id, meta);
				}
				return { ...state, documentMetas };
			});
		},

		removeDocument(id: string) {
			update((state) => {
				const documents = new Map(state.documents);
				const documentMetas = new Map(state.documentMetas);
				documents.delete(id);
				documentMetas.delete(id);
				return { ...state, documents, documentMetas };
			});
		},

		updateDocument(id: string, updates: Partial<Document>) {
			update((state) => {
				const documents = new Map(state.documents);
				const documentMetas = new Map(state.documentMetas);
				const now = new Date().toISOString();

				// Update full document if it exists
				const doc = state.documents.get(id);
				if (doc) {
					const updatedDoc = { ...doc, ...updates, updated_at: now };
					documents.set(id, updatedDoc);
				}

				// ALWAYS update documentMeta (for sidebar sync)
				const meta = state.documentMetas.get(id);
				if (meta) {
					documentMetas.set(id, {
						...meta,
						...(updates.title !== undefined && { title: updates.title }),
						...(updates.icon !== undefined && { icon: updates.icon }),
						...(updates.is_favorite !== undefined && { is_favorite: updates.is_favorite }),
						...(updates.is_archived !== undefined && { is_archived: updates.is_archived }),
						...(updates.parent_id !== undefined && { parent_id: updates.parent_id }),
						updated_at: now
					});
				}

				return { ...state, documents, documentMetas };
			});
		},

		getDocument(id: string): Document | undefined {
			return get({ subscribe }).documents.get(id);
		},

		clear() {
			set({
				documents: new Map(),
				documentMetas: new Map(),
				loading: false,
				error: null
			});
		}
	};
}

export const documentsStore = createDocumentsStore();

// ============================================================================
// Active Document Store
// ============================================================================

interface ActiveDocumentState {
	id: string | null;
	loading: boolean;
	saving: boolean;
	error: string | null;
	lastSaved: string | null;
}

function createActiveDocumentStore() {
	const { subscribe, set, update } = writable<ActiveDocumentState>({
		id: null,
		loading: false,
		saving: false,
		error: null,
		lastSaved: null
	});

	return {
		subscribe,

		setActiveDocument(id: string | null) {
			update((state) => ({ ...state, id, error: null }));
		},

		setLoading(loading: boolean) {
			update((state) => ({ ...state, loading }));
		},

		setSaving(saving: boolean) {
			update((state) => ({ ...state, saving }));
		},

		setLastSaved(lastSaved: string | null) {
			update((state) => ({ ...state, lastSaved }));
		},

		setError(error: string | null) {
			update((state) => ({ ...state, error }));
		}
	};
}

export const activeDocumentStore = createActiveDocumentStore();

// ============================================================================
// Document Tree Store (for Sidebar)
// ============================================================================

interface TreeState {
	expandedIds: Set<string>;
	loadingIds: Set<string>;
}

function createTreeStore() {
	const { subscribe, set, update } = writable<TreeState>({
		expandedIds: new Set(),
		loadingIds: new Set()
	});

	return {
		subscribe,

		toggleExpanded(id: string) {
			update((state) => {
				const expandedIds = new Set(state.expandedIds);
				if (expandedIds.has(id)) {
					expandedIds.delete(id);
				} else {
					expandedIds.add(id);
				}
				return { ...state, expandedIds };
			});
		},

		setExpanded(id: string, expanded: boolean) {
			update((state) => {
				const expandedIds = new Set(state.expandedIds);
				if (expanded) {
					expandedIds.add(id);
				} else {
					expandedIds.delete(id);
				}
				return { ...state, expandedIds };
			});
		},

		setLoading(id: string, loading: boolean) {
			update((state) => {
				const loadingIds = new Set(state.loadingIds);
				if (loading) {
					loadingIds.add(id);
				} else {
					loadingIds.delete(id);
				}
				return { ...state, loadingIds };
			});
		},

		isExpanded(id: string): boolean {
			return get({ subscribe }).expandedIds.has(id);
		},

		isLoading(id: string): boolean {
			return get({ subscribe }).loadingIds.has(id);
		},

		expandAll(ids: string[]) {
			update((state) => {
				const expandedIds = new Set(state.expandedIds);
				ids.forEach((id) => expandedIds.add(id));
				return { ...state, expandedIds };
			});
		},

		collapseAll() {
			update((state) => ({ ...state, expandedIds: new Set() }));
		}
	};
}

export const treeStore = createTreeStore();

// ============================================================================
// Sidebar View Store
// ============================================================================

interface SidebarState {
	view: SidebarView;
	searchQuery: string;
	width: number;
	collapsed: boolean;
}

function createSidebarStore() {
	const { subscribe, set, update } = writable<SidebarState>({
		view: 'all',
		searchQuery: '',
		width: 280,
		collapsed: false
	});

	return {
		subscribe,

		setView(view: SidebarView) {
			update((state) => ({ ...state, view }));
		},

		setSearchQuery(query: string) {
			update((state) => ({ ...state, searchQuery: query }));
		},

		setWidth(width: number) {
			update((state) => ({ ...state, width: Math.max(200, Math.min(500, width)) }));
		},

		toggleCollapsed() {
			update((state) => ({ ...state, collapsed: !state.collapsed }));
		},

		setCollapsed(collapsed: boolean) {
			update((state) => ({ ...state, collapsed }));
		}
	};
}

export const sidebarStore = createSidebarStore();

// ============================================================================
// Derived Stores
// ============================================================================

// Active document derived from documents store
export const activeDocument = derived(
	[documentsStore, activeDocumentStore],
	([$documents, $active]) => {
		if (!$active.id) return null;
		return $documents.documents.get($active.id) ?? null;
	}
);

// Root documents (no parent)
export const rootDocuments = derived(documentsStore, ($store) => {
	return Array.from($store.documentMetas.values())
		.filter((meta) => meta.parent_id === null && !meta.is_archived)
		.sort((a, b) => a.title.localeCompare(b.title));
});

// Favorite documents
export const favoriteDocuments = derived(documentsStore, ($store) => {
	return Array.from($store.documentMetas.values())
		.filter((meta) => meta.is_favorite && !meta.is_archived)
		.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());
});

// Recent documents
export const recentDocuments = derived(documentsStore, ($store) => {
	return Array.from($store.documentMetas.values())
		.filter((meta) => !meta.is_archived)
		.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
		.slice(0, 10);
});

// All document metas (for graph view)
export const documentMetas = derived(documentsStore, ($store) => {
	return Array.from($store.documentMetas.values())
		.filter((meta) => !meta.is_archived)
		.sort((a, b) => a.title.localeCompare(b.title));
});

// Children of a document
export function getChildren(parentId: string) {
	return derived(documentsStore, ($store) => {
		return Array.from($store.documentMetas.values())
			.filter((meta) => meta.parent_id === parentId && !meta.is_archived)
			.sort((a, b) => a.title.localeCompare(b.title));
	});
}

// Build tree structure
export const documentTree = derived(
	[documentsStore, treeStore],
	([$documents, $tree]) => {
		function buildNode(meta: DocumentMeta, depth: number): TreeNode {
			const children = Array.from($documents.documentMetas.values())
				.filter((m) => m.parent_id === meta.id && !m.is_archived)
				.sort((a, b) => a.title.localeCompare(b.title));

			return {
				id: meta.id,
				document: meta,
				children: children.map((child) => buildNode(child, depth + 1)),
				isExpanded: $tree.expandedIds.has(meta.id),
				isLoading: $tree.loadingIds.has(meta.id),
				depth
			};
		}

		const rootMetas = Array.from($documents.documentMetas.values())
			.filter((meta) => meta.parent_id === null && !meta.is_archived)
			.sort((a, b) => a.title.localeCompare(b.title));

		return rootMetas.map((meta) => buildNode(meta, 0));
	}
);
