<script lang="ts">
	import { FileCode, FileText, Database, Settings, Rocket } from 'lucide-svelte';

	interface WorkflowFile {
		id: string;
		name: string;
		type: string;
		size: number;
		updated_at: string;
		language?: string;
	}

	interface Props {
		files: WorkflowFile[];
		filteredFiles: WorkflowFile[];
		selectedFile: WorkflowFile | null;
		activeTab: string;
		onSelectFile: (file: WorkflowFile) => void;
		onSetActiveTab: (tab: string) => void;
	}

	let {
		files,
		filteredFiles,
		selectedFile,
		activeTab,
		onSelectFile,
		onSetActiveTab
	}: Props = $props();

	const fileCategories: Record<string, { label: string; icon: typeof FileCode }> = {
		all: { label: 'All Files', icon: FileCode },
		code: { label: 'Code', icon: FileCode },
		schema: { label: 'Schema', icon: Database },
		config: { label: 'Config', icon: Settings },
		documentation: { label: 'Docs', icon: FileText },
		deployment: { label: 'Deploy', icon: Rocket }
	};

	function getFilesByCategory(category: string): WorkflowFile[] {
		if (category === 'all') return files;
		return files.filter((f) => f.type === category);
	}

	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return Math.round((bytes / Math.pow(k, i)) * 10) / 10 + ' ' + sizes[i];
	}
</script>

<div class="file-sidebar">
	<!-- File Type Tabs -->
	<div class="file-tabs">
		{#each Object.entries(fileCategories) as [key, category]}
			{@const count = getFilesByCategory(key).length}
			{#if count > 0 || key === 'all'}
				<button
					class="file-tab"
					class:active={activeTab === key}
					onclick={() => onSetActiveTab(key)}
				>
					<svelte:component this={category.icon} size={16} />
					<span>{category.label}</span>
					<span class="file-count">{count}</span>
				</button>
			{/if}
		{/each}
	</div>

	<!-- File List -->
	<div class="file-list">
		{#each filteredFiles as file}
			<button
				class="file-item"
				class:selected={selectedFile?.id === file.id}
				onclick={() => onSelectFile(file)}
			>
				<svelte:component
					this={fileCategories[file.type as keyof typeof fileCategories]?.icon || FileCode}
					size={16}
					class="file-icon"
				/>
				<div class="file-item-info">
					<span class="file-name">{file.name}</span>
					<span class="file-size">{formatFileSize(file.size)}</span>
				</div>
			</button>
		{/each}

		{#if filteredFiles.length === 0}
			<div class="empty-files">
				<FileText size={32} class="text-gray-500" />
				<p>No files in this category</p>
			</div>
		{/if}
	</div>
</div>

<style>
	.file-sidebar {
		width: 320px;
		display: flex;
		flex-direction: column;
		background: #1e293b;
		border-right: 1px solid #334155;
	}

	@media (max-width: 768px) {
		.file-sidebar {
			width: 100%;
			border-right: 0;
			border-bottom: 1px solid #334155;
		}
		.file-tabs {
			flex-direction: row;
			flex-wrap: wrap;
			gap: 6px;
		}
		.file-tab {
			flex: 1 1 auto;
			justify-content: center;
		}
		.file-list {
			max-height: 240px;
		}
	}

	.file-tabs {
		display: flex;
		flex-direction: column;
		gap: 4px;
		padding: 16px;
		border-bottom: 1px solid #334155;
	}

	.file-tab {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: transparent;
		border: 1px solid transparent;
		border-radius: 6px;
		color: #94a3b8;
		font-size: 14px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: left;
	}

	.file-tab:hover {
		background: #0f172a;
		color: #e2e8f0;
	}

	.file-tab.active {
		background: #0f172a;
		border-color: #3b82f6;
		color: #60a5fa;
	}

	.file-count {
		margin-left: auto;
		padding: 2px 8px;
		background: #334155;
		border-radius: 10px;
		font-size: 12px;
	}

	.file-tab.active .file-count {
		background: #1e3a8a;
		color: #93c5fd;
	}

	.file-list {
		flex: 1;
		overflow-y: auto;
		padding: 8px;
	}

	.file-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 12px;
		background: transparent;
		border: 1px solid transparent;
		border-radius: 6px;
		color: #e2e8f0;
		font-size: 14px;
		cursor: pointer;
		transition: all 0.15s ease;
		text-align: left;
		margin-bottom: 4px;
	}

	.file-item:hover {
		background: #0f172a;
		border-color: #334155;
	}

	.file-item.selected {
		background: #1e3a8a;
		border-color: #3b82f6;
	}

	:global(.file-icon) {
		flex-shrink: 0;
		color: #60a5fa;
	}

	.file-item-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.file-name {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		font-weight: 500;
	}

	.file-size {
		font-size: 12px;
		color: #64748b;
	}

	.empty-files {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 40px 20px;
		gap: 12px;
		color: #64748b;
	}
</style>
