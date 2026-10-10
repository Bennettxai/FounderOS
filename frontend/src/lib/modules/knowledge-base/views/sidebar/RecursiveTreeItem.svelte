<script lang="ts">
	/**
	 * Recursive Tree Item - Renders tree nodes recursively
	 * This component handles the recursive rendering of tree children
	 * and passes down all necessary handlers for full functionality
	 */
	import SidebarTreeItem from './SidebarTreeItem.svelte';
	import type { TreeNode } from '../../entities/types';

	interface Props {
		node: TreeNode;
		activeId?: string | null;
		onSelect?: (node: TreeNode) => void;
		onAddChild?: (node: TreeNode) => void;
		onDelete?: (node: TreeNode) => void;
		onDuplicate?: (node: TreeNode) => void;
		onToggleFavorite?: (node: TreeNode) => void;
	}

	let {
		node,
		activeId = null,
		onSelect,
		onAddChild,
		onDelete,
		onDuplicate,
		onToggleFavorite
	}: Props = $props();

	function handleSelect() {
		onSelect?.(node);
	}

	function handleAddChild() {
		onAddChild?.(node);
	}

	function handleDelete() {
		onDelete?.(node);
	}

	function handleDuplicate() {
		onDuplicate?.(node);
	}

	function handleToggleFavorite() {
		onToggleFavorite?.(node);
	}

	function handleChildSelect(child: TreeNode) {
		onSelect?.(child);
	}

	function handleChildAddChild(child: TreeNode) {
		onAddChild?.(child);
	}

	function handleChildDelete(child: TreeNode) {
		onDelete?.(child);
	}

	function handleChildDuplicate(child: TreeNode) {
		onDuplicate?.(child);
	}

	function handleChildToggleFavorite(child: TreeNode) {
		onToggleFavorite?.(child);
	}
</script>

<SidebarTreeItem
	document={node.document}
	depth={node.depth}
	hasChildren={node.children.length > 0}
	isExpanded={node.isExpanded}
	isLoading={node.isLoading}
	isActive={activeId === node.id}
	onSelect={handleSelect}
	onAddChild={handleAddChild}
	onDelete={handleDelete}
	onDuplicate={handleDuplicate}
	onToggleFavorite={handleToggleFavorite}
>
	{#snippet children()}
		{#if node.isExpanded && node.children.length > 0}
			{#each node.children as child (child.id)}
				<svelte:self
					node={child}
					{activeId}
					onSelect={handleChildSelect}
					onAddChild={handleChildAddChild}
					onDelete={handleChildDelete}
					onDuplicate={handleChildDuplicate}
					onToggleFavorite={handleChildToggleFavorite}
				/>
			{/each}
		{/if}
	{/snippet}
</SidebarTreeItem>
