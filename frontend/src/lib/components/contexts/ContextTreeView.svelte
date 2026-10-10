<script lang="ts">
	import { api } from '$lib/api';
	import type { ContextTreeNode, ContextTree, EntityType } from '$lib/api/context-tree/types';

	interface Props {
		entityType: EntityType;
		entityId: string;
		onNodeSelect?: (node: ContextTreeNode) => void;
		maxHeight?: string;
	}

	let { entityType, entityId, onNodeSelect, maxHeight = '600px' }: Props = $props();

	let tree = $state<ContextTree | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let expandedNodes = $state<Set<string>>(new Set());

	// Load tree on mount
	$effect(() => {
		loadTree();
	});

	async function loadTree() {
		try {
			loading = true;
			error = null;
			tree = await api.getContextTree(entityType, entityId);

			// Auto-expand root node
			if (tree?.root_node) {
				expandedNodes.add(tree.root_node.id);
			}
		} catch (err) {
			console.error('Failed to load context tree:', err);
			error = err instanceof Error ? err.message : 'Failed to load context tree';
		} finally {
			loading = false;
		}
	}

	function toggleNode(nodeId: string) {
		if (expandedNodes.has(nodeId)) {
			expandedNodes.delete(nodeId);
		} else {
			expandedNodes.add(nodeId);
		}
		expandedNodes = new Set(expandedNodes); // Trigger reactivity
	}

	function handleNodeClick(node: ContextTreeNode) {
		toggleNode(node.id);
		onNodeSelect?.(node);
	}

	function getNodeIcon(type: string): string {
		switch (type) {
			case 'project':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />`;
			case 'context':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />`;
			case 'memory':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />`;
			case 'document':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
			case 'conversation':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25Z" />`;
			case 'task':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />`;
			case 'note':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
			default:
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M9.568 3H5.25A2.25 2.25 0 0 0 3 5.25v4.318c0 .597.237 1.17.659 1.591l9.581 9.581c.699.699 1.78.872 2.607.33a18.095 18.095 0 0 0 5.223-5.223c.542-.827.369-1.908-.33-2.607L11.16 3.66A2.25 2.25 0 0 0 9.568 3Z" />`;
		}
	}

	function getNodeColor(type: string): string {
		switch (type) {
			case 'project': return '#8b5cf6';
			case 'context': return '#22c55e';
			case 'memory': return '#3b82f6';
			case 'document': return '#f59e0b';
			case 'conversation': return '#ec4899';
			case 'task': return '#10b981';
			case 'note': return '#6366f1';
			default: return '#6b7280';
		}
	}

	function formatTokens(tokens: number): string {
		if (tokens >= 1000) {
			return `${(tokens / 1000).toFixed(1)}K`;
		}
		return tokens.toString();
	}

	// Calculate total tokens for a node and its children
	function getTotalTokens(node: ContextTreeNode): number {
		let total = node.token_estimate;
		for (const child of node.children) {
			total += getTotalTokens(child);
		}
		return total;
	}

	// Get depth level for styling (0 = root, 1 = tier 1, etc.)
	function getNodeDepth(nodeId: string, node: ContextTreeNode, currentDepth: number = 0): number {
		if (node.id === nodeId) return currentDepth;
		for (const child of node.children) {
			const depth = getNodeDepth(nodeId, child, currentDepth + 1);
			if (depth !== -1) return depth;
		}
		return -1;
	}

	function renderNode(node: ContextTreeNode, depth: number = 0) {
		const isExpanded = expandedNodes.has(node.id);
		const hasChildren = node.children.length > 0;
		const totalTokens = getTotalTokens(node);
		const nodeColor = getNodeColor(node.type);
		const tierLabel = depth === 0 ? 'ROOT' : `TIER ${depth}`;
		const tierColor = depth === 0 ? '#ef4444' : depth === 1 ? '#f59e0b' : depth === 2 ? '#3b82f6' : '#6b7280';

		return { node, isExpanded, hasChildren, totalTokens, nodeColor, tierLabel, tierColor, depth };
	}
</script>

<div class="tree-view" style="max-height: {maxHeight}">
	<div class="tree-header">
		<h3 class="tree-title">Context Tree</h3>
		<div class="tree-stats">
			{#if tree}
				<span class="stat-badge">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />
					</svg>
					{tree.total_items} items
				</span>
				<span class="stat-badge">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="14" height="14">
						<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
					</svg>
					{formatTokens(tree.root_node ? getTotalTokens(tree.root_node) : 0)} tokens
				</span>
			{/if}
		</div>
	</div>

	<div class="tree-content">
		{#if loading}
			<div class="loading-state">
				<div class="spinner"></div>
				<p>Loading context tree...</p>
			</div>
		{:else if error}
			<div class="error-state">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="error-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="M12 9v3.75m9-.75a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 3.75h.008v.008H12v-.008Z" />
				</svg>
				<p class="error-text">{error}</p>
				<button class="btn-pill btn-pill-ghost retry-btn" onclick={loadTree}>Try Again</button>
			</div>
		{:else if tree && tree.root_node}
			<div class="tree-nodes">
				{#snippet treeNodeSnippet(node: ContextTreeNode, depth: number)}
					{@const renderData = renderNode(node, depth)}
					<div class="tree-node" class:root-node={depth === 0} class:child-node={depth > 0} style="--tier-color: {renderData.tierColor}">
						<button
							class="btn-pill btn-pill-ghost node-button"
							onclick={() => handleNodeClick(renderData.node)}
						>
							<div class="node-expand">
								{#if renderData.hasChildren}
									<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="16" height="16" class="expand-icon" class:expanded={renderData.isExpanded}>
										<path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
									</svg>
								{:else}
									<div class="expand-placeholder"></div>
								{/if}
							</div>

							<div class="node-icon" style="color: {renderData.nodeColor}">
								<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="20" height="20">
									{@html getNodeIcon(renderData.node.type)}
								</svg>
							</div>

							<div class="node-content">
								<div class="node-header">
									<span class="node-title">{renderData.node.title}</span>
									<span class="tier-badge" style="background: {renderData.tierColor}15; color: {renderData.tierColor}">
										{renderData.tierLabel}
									</span>
								</div>
								{#if renderData.node.summary}
									<p class="node-summary">{renderData.node.summary}</p>
								{/if}
								<div class="node-meta">
									<span class="meta-badge type-badge" style="background: {renderData.nodeColor}15; color: {renderData.nodeColor}">
										{renderData.node.type}
									</span>
									<span class="meta-badge">
										<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
											<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
										</svg>
										{formatTokens(renderData.totalTokens)} total
									</span>
									{#if renderData.hasChildren}
										<span class="meta-badge">
											<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="12" height="12">
												<path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
											</svg>
											{renderData.node.children.length} items
										</span>
									{/if}
									{#if !renderData.node.is_loaded}
										<span class="meta-badge unloaded">
											Not Loaded
										</span>
									{/if}
								</div>
							</div>
						</button>

						{#if renderData.isExpanded && renderData.hasChildren}
							<div class="node-children">
								{#each renderData.node.children as child}
									{@render treeNodeSnippet(child, depth + 1)}
								{/each}
							</div>
						{/if}
					</div>
				{/snippet}

				{@render treeNodeSnippet(tree.root_node, 0)}
			</div>
		{:else}
			<div class="empty-state">
				<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="empty-icon">
					<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />
				</svg>
				<p class="empty-text">No context tree available</p>
				<p class="empty-hint">Start a conversation or create contexts to build your tree</p>
			</div>
		{/if}
	</div>
</div>

<style>
	.tree-view {
		display: flex;
		flex-direction: column;
		height: 100%;
		background: var(--color-bg);
		border-radius: 12px;
		overflow: hidden;
	}

	.tree-header {
		padding: 20px 24px;
		border-bottom: 1px solid var(--color-border);
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.tree-title {
		font-size: 18px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0;
	}

	.tree-stats {
		display: flex;
		gap: 12px;
	}

	.stat-badge {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 6px 12px;
		background: var(--color-bg-secondary);
		border: 1px solid var(--color-border);
		border-radius: 6px;
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text-muted);
	}

	.stat-badge svg {
		flex-shrink: 0;
	}

	.tree-content {
		flex: 1;
		overflow-y: auto;
		padding: 16px;
	}

	.tree-nodes {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.tree-node {
		display: flex;
		flex-direction: column;
	}

	.root-node {
		border: 2px solid #ef4444;
		border-radius: 12px;
		background: linear-gradient(135deg, rgba(239, 68, 68, 0.05) 0%, rgba(239, 68, 68, 0.02) 100%);
	}

	.child-node {
		margin-left: 32px;
		padding-left: 16px;
		border-left: 2px solid var(--tier-color, var(--color-border));
		position: relative;
	}

	.child-node::before {
		content: '';
		position: absolute;
		left: -2px;
		top: 24px;
		width: 16px;
		height: 2px;
		background: var(--tier-color, var(--color-border));
	}

	.node-button {
		display: flex;
		align-items: flex-start;
		gap: 12px;
		padding: 16px;
		background: var(--color-bg);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: left;
		width: 100%;
	}

	.node-button:hover {
		background: var(--color-bg-secondary);
		border-color: #3b82f6;
		transform: translateY(-1px);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
	}

	.node-expand {
		width: 16px;
		height: 16px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
		margin-top: 2px;
	}

	.expand-icon {
		color: var(--color-text-muted);
		transition: transform 0.2s ease;
	}

	.expand-icon.expanded {
		transform: rotate(0deg);
	}

	.expand-icon:not(.expanded) {
		transform: rotate(-90deg);
	}

	.expand-placeholder {
		width: 16px;
		height: 16px;
	}

	.node-icon {
		flex-shrink: 0;
		width: 40px;
		height: 40px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: currentColor;
		background: color-mix(in srgb, currentColor 10%, transparent);
		border-radius: 8px;
	}

	.node-content {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.node-header {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.node-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--color-text);
		flex: 1;
	}

	.tier-badge {
		padding: 3px 8px;
		font-size: 10px;
		font-weight: 700;
		border-radius: 4px;
		text-transform: uppercase;
		letter-spacing: 0.5px;
		flex-shrink: 0;
	}

	.node-summary {
		font-size: 13px;
		color: var(--color-text-muted);
		line-height: 1.5;
		margin: 0;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.node-meta {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.meta-badge {
		display: flex;
		align-items: center;
		gap: 4px;
		padding: 3px 8px;
		font-size: 11px;
		font-weight: 500;
		color: var(--color-text-muted);
		background: var(--color-bg-secondary);
		border-radius: 4px;
	}

	.meta-badge svg {
		flex-shrink: 0;
	}

	.type-badge {
		text-transform: uppercase;
		letter-spacing: 0.3px;
	}

	.meta-badge.unloaded {
		background: rgba(239, 68, 68, 0.1);
		color: #ef4444;
	}

	.node-children {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin-top: 8px;
	}

	.loading-state,
	.empty-state,
	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 60px 24px;
		text-align: center;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--color-border);
		border-top-color: #3b82f6;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
		margin-bottom: 16px;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	.loading-state p,
	.empty-state p,
	.error-state p {
		margin: 0;
	}

	.empty-icon,
	.error-icon {
		width: 48px;
		height: 48px;
		color: var(--color-text-muted);
		margin-bottom: 16px;
	}

	.error-icon {
		color: #ef4444;
	}

	.empty-text,
	.error-text {
		font-size: 14px;
		font-weight: 500;
		color: var(--color-text);
		margin-bottom: 4px;
	}

	.empty-hint {
		font-size: 13px;
		color: var(--color-text-muted);
	}

	.retry-btn {
		margin-top: 16px;
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		color: white;
		background: #3b82f6;
		border: none;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.retry-btn:hover {
		background: #2563eb;
	}
</style>
