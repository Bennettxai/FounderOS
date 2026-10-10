import { writable } from 'svelte/store';
import {
	getNodes,
	getNodeTree,
	getActiveNode,
	getNode,
	createNode,
	updateNode,
	activateNode,
	deactivateNode,
	deleteNode,
	reorderNode,
	getNodeLinks,
	getNodeLinkCounts,
	linkNodeProject,
	unlinkNodeProject,
	linkNodeContext,
	unlinkNodeContext,
	linkNodeConversation,
	unlinkNodeConversation
} from '$lib/api/nodes';
import type {
	Node,
	NodeTree,
	NodeDetail,
	CreateNodeData,
	UpdateNodeData,
	NodeLinks,
	NodeLinkCounts
} from '$lib/api/nodes/types';

interface NodesState {
	nodes: Node[];
	nodeTree: NodeTree[];
	activeNode: Node | null;
	currentNode: NodeDetail | null;
	currentNodeLinks: NodeLinks | null;
	loading: boolean;
	linksLoading: boolean;
}

function createNodesStore() {
	const { subscribe, update } = writable<NodesState>({
		nodes: [],
		nodeTree: [],
		activeNode: null,
		currentNode: null,
		currentNodeLinks: null,
		loading: false,
		linksLoading: false
	});

	return {
		subscribe,

		// Load flat list of nodes
		async load(includeArchived = false) {
			update((s) => ({ ...s, loading: true }));
			try {
				const nodes = await getNodes(includeArchived);
				update((s) => ({ ...s, nodes, loading: false }));
			} catch (error) {
				console.error('Failed to load nodes:', error);
				update((s) => ({ ...s, loading: false }));
			}
		},

		// Load hierarchical tree
		async loadTree(includeArchived = false) {
			update((s) => ({ ...s, loading: true }));
			try {
				const nodeTree = await getNodeTree(includeArchived);
				update((s) => ({ ...s, nodeTree, loading: false }));
			} catch (error) {
				console.error('Failed to load node tree:', error);
				update((s) => ({ ...s, loading: false }));
			}
		},

		// Load single node detail
		async loadById(id: string) {
			update((s) => ({ ...s, loading: true }));
			try {
				const node = await getNode(id);
				update((s) => ({ ...s, currentNode: node, loading: false }));
				return node;
			} catch (error) {
				console.error('Failed to load node:', error);
				update((s) => ({ ...s, loading: false }));
				throw error;
			}
		},

		// Load active node
		async loadActive() {
			try {
				const activeNode = await getActiveNode();
				update((s) => ({ ...s, activeNode }));
				return activeNode;
			} catch (error) {
				console.error('Failed to load active node:', error);
			}
		},

		// Create new node
		async create(data: CreateNodeData) {
			try {
				const node = await createNode(data);
				update((s) => ({ ...s, nodes: [node, ...s.nodes] }));
				return node;
			} catch (error) {
				console.error('Failed to create node:', error);
				throw error;
			}
		},

		// Update node
		async update(id: string, data: UpdateNodeData) {
			try {
				const node = await updateNode(id, data);
				update((s) => ({
					...s,
					nodes: s.nodes.map((n) => (n.id === id ? node : n)),
					currentNode: s.currentNode?.id === id
						? { ...s.currentNode, ...node }
						: s.currentNode,
					activeNode: s.activeNode?.id === id ? node : s.activeNode
				}));
				return node;
			} catch (error) {
				console.error('Failed to update node:', error);
				throw error;
			}
		},

		// Activate node
		async activate(id: string) {
			try {
				const response = await activateNode(id);
				update((s) => ({
					...s,
					activeNode: response.node,
					nodes: s.nodes.map((n) => ({
						...n,
						is_active: n.id === id
					})),
					currentNode: s.currentNode?.id === id
						? { ...s.currentNode, is_active: true }
						: s.currentNode?.is_active
						? { ...s.currentNode, is_active: false }
						: s.currentNode
				}));
				return response;
			} catch (error) {
				console.error('Failed to activate node:', error);
				throw error;
			}
		},

		// Deactivate node
		async deactivate(id: string) {
			try {
				const node = await deactivateNode(id);
				update((s) => ({
					...s,
					activeNode: null,
					nodes: s.nodes.map((n) => (n.id === id ? { ...n, is_active: false } : n)),
					currentNode: s.currentNode?.id === id
						? { ...s.currentNode, is_active: false }
						: s.currentNode
				}));
				return node;
			} catch (error) {
				console.error('Failed to deactivate node:', error);
				throw error;
			}
		},

		// Archive node
		async archive(id: string) {
			try {
				const node = await updateNode(id, { is_archived: true });
				update((s) => ({
					...s,
					nodes: s.nodes.filter((n) => n.id !== id),
					currentNode: s.currentNode?.id === id ? null : s.currentNode,
					activeNode: s.activeNode?.id === id ? null : s.activeNode
				}));
				return node;
			} catch (error) {
				console.error('Failed to archive node:', error);
				throw error;
			}
		},

		// Unarchive node
		async unarchive(id: string) {
			try {
				const node = await updateNode(id, { is_archived: false });
				update((s) => ({
					...s,
					nodes: [...s.nodes, node]
				}));
				return node;
			} catch (error) {
				console.error('Failed to unarchive node:', error);
				throw error;
			}
		},

		// Delete node
		async delete(id: string) {
			try {
				await deleteNode(id);
				update((s) => ({
					...s,
					nodes: s.nodes.filter((n) => n.id !== id),
					currentNode: s.currentNode?.id === id ? null : s.currentNode,
					activeNode: s.activeNode?.id === id ? null : s.activeNode
				}));
			} catch (error) {
				console.error('Failed to delete node:', error);
				throw error;
			}
		},

		// Reorder node
		async reorder(id: string, newOrder: number) {
			try {
				await reorderNode(id, newOrder);
				// Reload to get updated order
				await this.load();
			} catch (error) {
				console.error('Failed to reorder node:', error);
				throw error;
			}
		},

		// Clear current node
		clearCurrent() {
			update((s) => ({ ...s, currentNode: null }));
		},

		// Refresh all data
		async refresh(includeArchived = false) {
			await Promise.all([
				this.load(includeArchived),
				this.loadTree(includeArchived),
				this.loadActive()
			]);
		},

		// ===== LINKING METHODS =====

		// Load links for a node (gracefully handles missing backend endpoints)
		async loadLinks(nodeId: string) {
			update((s) => ({ ...s, linksLoading: true }));
			try {
				const links = await getNodeLinks(nodeId);
				update((s) => ({ ...s, currentNodeLinks: links, linksLoading: false }));
				return links;
			} catch (error) {
				// Don't throw - just log and set empty links
				// This gracefully handles when the backend endpoints don't exist yet
				console.debug('Links not available for node:', nodeId);
				const emptyLinks = { projects: [], contexts: [], conversations: [] };
				update((s) => ({ ...s, currentNodeLinks: emptyLinks, linksLoading: false }));
				return emptyLinks;
			}
		},

		// Get link counts for a node (gracefully handles missing backend endpoints)
		async getLinkCounts(nodeId: string) {
			try {
				return await getNodeLinkCounts(nodeId);
			} catch (error) {
				// Return zeros if endpoint doesn't exist
				console.debug('Link counts not available for node:', nodeId);
				return { linked_projects_count: 0, linked_contexts_count: 0, linked_conversations_count: 0 };
			}
		},

		// Link a project to a node
		async linkProject(nodeId: string, projectId: string) {
			try {
				await linkNodeProject(nodeId, projectId);
				// Reload links to update the list
				await this.loadLinks(nodeId);
			} catch (error) {
				console.error('Failed to link project:', error);
				throw error;
			}
		},

		// Unlink a project from a node
		async unlinkProject(nodeId: string, projectId: string) {
			try {
				await unlinkNodeProject(nodeId, projectId);
				update((s) => ({
					...s,
					currentNodeLinks: s.currentNodeLinks
						? {
								...s.currentNodeLinks,
								projects: s.currentNodeLinks.projects.filter((p) => p.id !== projectId)
							}
						: null
				}));
			} catch (error) {
				console.error('Failed to unlink project:', error);
				throw error;
			}
		},

		// Link a context to a node
		async linkContext(nodeId: string, contextId: string) {
			try {
				await linkNodeContext(nodeId, contextId);
				// Reload links to update the list
				await this.loadLinks(nodeId);
			} catch (error) {
				console.error('Failed to link context:', error);
				throw error;
			}
		},

		// Unlink a context from a node
		async unlinkContext(nodeId: string, contextId: string) {
			try {
				await unlinkNodeContext(nodeId, contextId);
				update((s) => ({
					...s,
					currentNodeLinks: s.currentNodeLinks
						? {
								...s.currentNodeLinks,
								contexts: s.currentNodeLinks.contexts.filter((c) => c.id !== contextId)
							}
						: null
				}));
			} catch (error) {
				console.error('Failed to unlink context:', error);
				throw error;
			}
		},

		// Link a conversation to a node
		async linkConversation(nodeId: string, conversationId: string) {
			try {
				await linkNodeConversation(nodeId, conversationId);
				// Reload links to update the list
				await this.loadLinks(nodeId);
			} catch (error) {
				console.error('Failed to link conversation:', error);
				throw error;
			}
		},

		// Unlink a conversation from a node
		async unlinkConversation(nodeId: string, conversationId: string) {
			try {
				await unlinkNodeConversation(nodeId, conversationId);
				update((s) => ({
					...s,
					currentNodeLinks: s.currentNodeLinks
						? {
								...s.currentNodeLinks,
								conversations: s.currentNodeLinks.conversations.filter(
									(c) => c.id !== conversationId
								)
							}
						: null
				}));
			} catch (error) {
				console.error('Failed to unlink conversation:', error);
				throw error;
			}
		},

		// Clear current node links
		clearLinks() {
			update((s) => ({ ...s, currentNodeLinks: null }));
		}
	};
}

export const nodes = createNodesStore();
