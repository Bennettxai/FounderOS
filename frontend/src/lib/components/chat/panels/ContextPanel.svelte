<script lang="ts">
	export interface ActiveResource {
		id: string;
		type: 'document' | 'artifact' | 'project' | 'context' | 'search_result';
		name?: string;
		source?: string;
		title?: string;  // Alternative to name
		contextId?: string;
		tokenCount?: number;
	}

	export interface AvailableContext {
		id: string;
		name: string;
		description?: string;
		document_count?: number;
	}

	interface Props {
		resources: ActiveResource[];
		availableContexts?: AvailableContext[];
		selectedContextIds?: string[];
		onResourceClick?: (resource: ActiveResource) => void;
		onSearch?: () => void;
		onContextToggle?: (contextId: string, selected: boolean) => void;
		onMemoriesSelected?: (memoryIds: string[]) => void;
	}

	let { resources = [], availableContexts = [], selectedContextIds = [], onResourceClick, onSearch, onContextToggle, onMemoriesSelected }: Props = $props();

	// Track selected memory IDs
	let selectedMemoryIds = $state<string[]>([]);

	function getResourceIcon(type: string): string {
		switch (type) {
			case 'document':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
			case 'artifact':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9" />`;
			case 'project':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M2.25 12.75V12A2.25 2.25 0 0 1 4.5 9.75h15A2.25 2.25 0 0 1 21.75 12v.75m-8.69-6.44-2.12-2.12a1.5 1.5 0 0 0-1.061-.44H4.5A2.25 2.25 0 0 0 2.25 6v12a2.25 2.25 0 0 0 2.25 2.25h15A2.25 2.25 0 0 0 21.75 18V9a2.25 2.25 0 0 0-2.25-2.25h-5.379a1.5 1.5 0 0 1-1.06-.44Z" />`;
			case 'context':
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />`;
			default:
				return `<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />`;
		}
	}

	function getResourceColor(type: string): string {
		switch (type) {
			case 'document': return '#3b82f6';
			case 'artifact': return '#8b5cf6';
			case 'project': return '#f59e0b';
			case 'context': return '#22c55e';
			default: return '#6b7280';
		}
	}

	function handleContextToggle(contextId: string) {
		const isSelected = selectedContextIds.includes(contextId);
		onContextToggle?.(contextId, !isSelected);
	}

	// Group resources by type
	let groupedResources = $derived.by(() => {
		const groups: Record<string, ActiveResource[]> = {};
		for (const resource of resources) {
			if (!groups[resource.type]) {
				groups[resource.type] = [];
			}
			groups[resource.type].push(resource);
		}
		return groups;
	});

	// View mode: contexts, resources, or memories
	let viewMode = $state<'contexts' | 'resources' | 'memories'>('contexts');

	// Handle memory selection from MemoryPanel
	function handleMemorySelect(memory: any) {
		const memoryId = memory.id;
		const isSelected = selectedMemoryIds.includes(memoryId);

		if (isSelected) {
			selectedMemoryIds = selectedMemoryIds.filter(id => id !== memoryId);
		} else {
			selectedMemoryIds = [...selectedMemoryIds, memoryId];
		}

		// Notify parent component
		onMemoriesSelected?.(selectedMemoryIds);
	}

	import MemoryPanel from './MemoryPanel.svelte';
	import type { MemoryListItem } from '$lib/api';
</script>

<div class="context-panel">
	<div class="panel-header">
		<h3 class="panel-title">Context</h3>
		<div class="header-actions">
			{#if onSearch}
				<button class="btn-pill btn-pill-ghost search-btn" onclick={onSearch} aria-label="Search contexts">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
						<path stroke-linecap="round" stroke-linejoin="round" d="m21 21-5.197-5.197m0 0A7.5 7.5 0 1 0 5.196 5.196a7.5 7.5 0 0 0 10.607 10.607Z" />
					</svg>
				</button>
			{/if}
		</div>
	</div>

	<!-- View Mode Tabs -->
	<div class="view-tabs">
		<button
			class="view-tab {viewMode === 'contexts' ? 'active' : ''}"
			onclick={() => viewMode = 'contexts'}
		>
			Contexts
			{#if selectedContextIds.length > 0}
				<span class="tab-badge">{selectedContextIds.length}</span>
			{/if}
		</button>
		<button
			class="view-tab {viewMode === 'resources' ? 'active' : ''}"
			onclick={() => viewMode = 'resources'}
		>
			Active
			{#if resources.length > 0}
				<span class="tab-badge">{resources.length}</span>
			{/if}
		</button>
		<button
			class="view-tab {viewMode === 'memories' ? 'active' : ''}"
			onclick={() => viewMode = 'memories'}
		>
			Memories
			{#if selectedMemoryIds.length > 0}
				<span class="tab-badge">{selectedMemoryIds.length}</span>
			{/if}
		</button>
	</div>

	<div class="panel-content">
		{#if viewMode === 'contexts'}
			<!-- Available Contexts for Selection -->
			{#if availableContexts.length === 0}
				<div class="empty-state">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="empty-icon">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />
					</svg>
					<p class="empty-text">No contexts available</p>
					<p class="empty-hint">Create contexts in your project to add them here</p>
				</div>
			{:else}
				<div class="context-list">
					{#each availableContexts as context (context.id)}
						{@const isSelected = selectedContextIds.includes(context.id)}
						<button
							class="context-item {isSelected ? 'selected' : ''}"
							onclick={() => handleContextToggle(context.id)}
						>
							<div class="context-checkbox {isSelected ? 'checked' : ''}">
								{#if isSelected}
									<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2.5" stroke="currentColor" width="12" height="12">
										<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 12.75 6 6 9-13.5" />
									</svg>
								{/if}
							</div>
							<div class="context-info">
								<span class="context-name">{context.name}</span>
								{#if context.description}
									<span class="context-desc">{context.description}</span>
								{/if}
								{#if context.document_count !== undefined}
									<span class="context-meta">{context.document_count} document{context.document_count !== 1 ? 's' : ''}</span>
								{/if}
							</div>
						</button>
					{/each}
				</div>
			{/if}
		{:else if viewMode === 'resources'}
			<!-- Active Resources View -->
			{#if resources.length === 0}
				<div class="empty-state">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="empty-icon">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6.429 9.75 2.25 12l4.179 2.25m0-4.5 5.571 3 5.571-3m-11.142 0L2.25 7.5 12 2.25l9.75 5.25-4.179 2.25m0 0L21.75 12l-4.179 2.25m0 0 4.179 2.25L12 21.75 2.25 16.5l4.179-2.25m11.142 0-5.571 3-5.571-3" />
					</svg>
					<p class="empty-text">Resources Claude is working with will appear here</p>
				</div>
			{:else}
				{#each Object.entries(groupedResources) as [type, items] (type)}
					<div class="resource-section">
						<div class="section-label">{type.charAt(0).toUpperCase() + type.slice(1)}s</div>
						{#each items as resource (resource.id)}
							<button
								class="btn-pill btn-pill-ghost resource-item"
								onclick={() => onResourceClick?.(resource)}
							>
								<div class="resource-icon" style="color: {getResourceColor(resource.type)}">
									<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
										{@html getResourceIcon(resource.type)}
									</svg>
								</div>
								<div class="resource-info">
									<span class="resource-name">{resource.name}</span>
									<span class="resource-source">{resource.source}</span>
								</div>
							</button>
						{/each}
					</div>
				{/each}
			{/if}
		{:else if viewMode === 'memories'}
			<MemoryPanel
				onMemoryClick={handleMemorySelect}
				{selectedMemoryIds}
			/>
		{/if}
	</div>
</div>

<style>
	.context-panel {
		display: flex;
		flex-direction: column;
		height: 100%;
	}

	/* ── Mobile: ContextPanel ─────────────────────────────────────── */
	@media (max-width: 767px) {
		/* When used as a right-side panel inside a chat layout,
		   the context panel stacks below the message area on mobile. */
		.context-panel {
			width: 100% !important;
			max-width: 100% !important;
			height: auto;
			min-height: 260px;
			border-top: 1px solid var(--color-border, #e5e7eb);
			border-left: none;
		}

		.panel-content {
			max-height: 40vh;
			overflow-y: auto;
		}
	}

	.panel-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px;
		border-bottom: 1px solid var(--color-border);
	}

	:global(.dark) .panel-header {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	.panel-title {
		font-size: 15px;
		font-weight: 600;
		color: var(--color-text);
		margin: 0;
	}

	:global(.dark) .panel-title {
		color: #f5f5f7;
	}

	.header-actions {
		display: flex;
		align-items: center;
		gap: 4px;
	}

	.search-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: transparent;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
		transition: all 0.15s ease;
	}

	.search-btn:hover {
		background: var(--color-bg-secondary);
		color: var(--color-text);
	}

	:global(.dark) .search-btn {
		color: #6e6e73;
	}

	:global(.dark) .search-btn:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	/* View Mode Tabs */
	.view-tabs {
		display: flex;
		border-bottom: 1px solid var(--color-border);
		padding: 0 8px;
	}

	:global(.dark) .view-tabs {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	.view-tab {
		display: flex;
		align-items: center;
		gap: 6px;
		padding: 10px 12px;
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text-muted);
		background: transparent;
		border: none;
		border-bottom: 2px solid transparent;
		cursor: pointer;
		transition: all 0.15s ease;
		margin-bottom: -1px;
	}

	.view-tab:hover {
		color: var(--color-text);
	}

	.view-tab.active {
		color: var(--color-text);
		border-bottom-color: var(--color-text);
	}

	:global(.dark) .view-tab {
		color: #6e6e73;
	}

	:global(.dark) .view-tab:hover {
		color: #f5f5f7;
	}

	:global(.dark) .view-tab.active {
		color: #f5f5f7;
		border-bottom-color: #f5f5f7;
	}

	.tab-badge {
		font-size: 10px;
		font-weight: 600;
		padding: 2px 6px;
		background: rgba(0, 0, 0, 0.08);
		border-radius: 10px;
	}

	:global(.dark) .tab-badge {
		background: rgba(255, 255, 255, 0.12);
	}

	.panel-content {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 32px 16px;
		text-align: center;
	}

	.empty-icon {
		width: 40px;
		height: 40px;
		color: var(--color-text-muted);
		margin-bottom: 12px;
	}

	:global(.dark) .empty-icon {
		color: #6e6e73;
	}

	.empty-text {
		font-size: 13px;
		color: var(--color-text-muted);
		line-height: 1.5;
		margin: 0;
	}

	:global(.dark) .empty-text {
		color: #6e6e73;
	}

	.empty-hint {
		font-size: 12px;
		color: var(--color-text-muted);
		margin-top: 4px;
		opacity: 0.7;
	}

	/* Context List */
	.context-list {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.context-item {
		display: flex;
		align-items: flex-start;
		gap: 10px;
		width: 100%;
		padding: 10px;
		background: transparent;
		border: 1px solid transparent;
		border-radius: 10px;
		cursor: pointer;
		text-align: left;
		transition: all 0.15s ease;
	}

	.context-item:hover {
		background: var(--color-bg-secondary);
	}

	.context-item.selected {
		background: rgba(34, 197, 94, 0.08);
		border-color: rgba(34, 197, 94, 0.3);
	}

	:global(.dark) .context-item:hover {
		background: #3a3a3c;
	}

	:global(.dark) .context-item.selected {
		background: rgba(34, 197, 94, 0.15);
		border-color: rgba(34, 197, 94, 0.4);
	}

	.context-checkbox {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 18px;
		height: 18px;
		border: 2px solid var(--color-border);
		border-radius: 4px;
		flex-shrink: 0;
		margin-top: 1px;
		transition: all 0.15s ease;
	}

	.context-checkbox.checked {
		background: #22c55e;
		border-color: #22c55e;
		color: white;
	}

	:global(.dark) .context-checkbox {
		border-color: rgba(255, 255, 255, 0.3);
	}

	.context-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.context-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text);
	}

	:global(.dark) .context-name {
		color: #f5f5f7;
	}

	.context-desc {
		font-size: 12px;
		color: var(--color-text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .context-desc {
		color: #a1a1a6;
	}

	.context-meta {
		font-size: 11px;
		color: var(--color-text-muted);
		opacity: 0.7;
	}

	:global(.dark) .context-meta {
		color: #6e6e73;
	}

	/* Resource Sections */
	.resource-section {
		margin-bottom: 16px;
	}

	.section-label {
		font-size: 11px;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.5px;
		padding: 8px 8px 4px;
	}

	:global(.dark) .section-label {
		color: #6e6e73;
	}

	.resource-item {
		display: flex;
		align-items: center;
		gap: 10px;
		width: 100%;
		padding: 10px 8px;
		background: transparent;
		border: none;
		border-radius: 8px;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s ease;
	}

	.resource-item:hover {
		background: var(--color-bg-secondary);
	}

	:global(.dark) .resource-item:hover {
		background: #3a3a3c;
	}

	.resource-icon {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		background: var(--color-bg-tertiary);
		border-radius: 6px;
		flex-shrink: 0;
	}

	:global(.dark) .resource-icon {
		background: #3a3a3c;
	}

	.resource-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.resource-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--color-text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .resource-name {
		color: #f5f5f7;
	}

	.resource-source {
		font-size: 11px;
		color: var(--color-text-muted);
	}

	:global(.dark) .resource-source {
		color: #6e6e73;
	}
</style>
