<script lang="ts">
	import { ChevronRight, ChevronDown, File, Folder, FolderOpen } from 'lucide-svelte';
	import type { FileTreeNode, OSAFile } from './types';

	interface Props {
		nodes: FileTreeNode[];
		selectedFile?: OSAFile | null;
		onFileSelect?: (file: OSAFile) => void;
		depth?: number;
	}

	let { nodes = [], selectedFile = null, onFileSelect, depth = 0 }: Props = $props();

	let expandedNodes = $state<Set<string>>(new Set());

	function toggleNode(nodeId: string) {
		const newExpanded = new Set(expandedNodes);
		if (newExpanded.has(nodeId)) {
			newExpanded.delete(nodeId);
		} else {
			newExpanded.add(nodeId);
		}
		expandedNodes = newExpanded;
	}

	function handleFileClick(file: OSAFile) {
		if (onFileSelect) {
			onFileSelect(file);
		}
	}

	function getFileIcon(type: string): typeof File {
		return File;
	}

	function getFileColor(type: string): string {
		const colors: Record<string, string> = {
			code: 'text-blue-500',
			schema: 'text-purple-500',
			config: 'text-yellow-500',
			documentation: 'text-green-500',
			deployment: 'text-orange-500',
			markdown: 'text-gray-500',
			yaml: 'text-pink-500',
			json: 'text-indigo-500',
			text: 'text-gray-400'
		};
		return colors[type] || 'text-gray-400';
	}

	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return Math.round(bytes / Math.pow(k, i) * 10) / 10 + ' ' + sizes[i];
	}
</script>

<div class="file-tree" style="padding-left: {depth * 16}px">
	{#each nodes as node}
		{@const isExpanded = expandedNodes.has(node.id)}
		{@const isSelected = selectedFile?.id === node.file?.id}

		{#if node.type === 'folder'}
			<button
				class="btn-pill btn-pill-ghost tree-item folder"
				class:expanded={isExpanded}
				onclick={() => toggleNode(node.id)}
			>
				<span class="icon-wrapper">
					{#if isExpanded}
						<ChevronDown size={16} class="chevron" />
						<FolderOpen size={16} class="text-blue-400" />
					{:else}
						<ChevronRight size={16} class="chevron" />
						<Folder size={16} class="text-blue-400" />
					{/if}
				</span>
				<span class="name">{node.name}</span>
			</button>

			{#if isExpanded && node.children}
				<svelte:self
					nodes={node.children}
					{selectedFile}
					{onFileSelect}
					depth={depth + 1}
				/>
			{/if}
		{:else if node.file}
			<button
				class="btn-pill btn-pill-ghost tree-item file"
				class:selected={isSelected}
				onclick={() => handleFileClick(node.file!)}
			>
				<span class="icon-wrapper">
					<span class="file-icon {getFileColor(node.file.type)}">
						<svelte:component this={getFileIcon(node.file.type)} size={16} />
					</span>
				</span>
				<span class="name">{node.name}</span>
				<span class="meta">
					{formatFileSize(node.file.size)}
				</span>
			</button>
		{/if}
	{/each}
</div>

<style>
	.file-tree {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.tree-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		border-radius: 4px;
		border: none;
		background: transparent;
		cursor: pointer;
		transition: background-color 0.15s ease;
		text-align: left;
		width: 100%;
		font-size: 14px;
		color: #e5e7eb;
	}

	.tree-item:hover {
		background-color: rgba(59, 130, 246, 0.1);
	}

	.tree-item.selected {
		background-color: rgba(59, 130, 246, 0.2);
		color: #60a5fa;
	}

	.tree-item.folder {
		font-weight: 500;
	}

	.icon-wrapper {
		display: flex;
		align-items: center;
		gap: 4px;
		flex-shrink: 0;
	}

	.chevron {
		color: #9ca3af;
		transition: transform 0.2s ease;
	}

	.file-icon {
		display: flex;
		align-items: center;
	}

	.name {
		flex: 1;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.meta {
		font-size: 12px;
		color: #6b7280;
		margin-left: auto;
		flex-shrink: 0;
	}
</style>
