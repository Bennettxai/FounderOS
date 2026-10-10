<script lang="ts">
	function formatFileSize(bytes: number): string {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
	}

	interface AttachedFile {
		id: string;
		name: string;
		type: string;
		size: number;
		content?: string;
		documentId?: string;
		uploading?: boolean;
		uploadError?: string;
	}

	interface Props {
		attachedFiles: AttachedFile[];
		selectedMemoryIds?: string[];
		onRemoveFile: (fileId: string) => void;
		onClearMemories?: () => void;
	}

	let {
		attachedFiles,
		selectedMemoryIds = [],
		onRemoveFile,
		onClearMemories
	}: Props = $props();
</script>

<!-- Attached files display -->
{#if attachedFiles.length > 0}
	<div class="flex flex-wrap gap-2 mb-3">
		{#each attachedFiles as file (file.id)}
			<div class="flex items-center gap-2 px-3 py-1.5 bg-gray-100 rounded-lg text-sm {file.uploadError ? 'bg-red-50 border border-red-200' : ''}">
				{#if file.uploading}
					<svg class="w-4 h-4 text-gray-400 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
					</svg>
				{:else if file.type.startsWith('image/') && file.content}
					<img src={file.content} alt={file.name} class="w-6 h-6 rounded object-cover" />
				{:else}
					<svg class="w-4 h-4 {file.uploadError ? 'text-red-500' : 'text-gray-500'}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
					</svg>
				{/if}
				<span class="text-gray-700 truncate max-w-[150px]">{file.name}</span>
				{#if file.uploadError}
					<span class="text-red-500 text-xs">Failed</span>
				{:else}
					<span class="text-gray-400 text-xs">{formatFileSize(file.size)}</span>
				{/if}
				<button
					onclick={(e) => { e.stopPropagation(); onRemoveFile(file.id); }}
					class="p-0.5 text-gray-400 hover:text-gray-600"
					aria-label="Remove file {file.name}"
				>
					<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		{/each}
	</div>
{/if}

<!-- Active memories display -->
{#if selectedMemoryIds.length > 0}
	<div class="flex flex-wrap gap-2 mb-3">
		<div class="flex items-center gap-2 px-3 py-1.5 bg-green-50 border border-green-200 rounded-lg text-sm">
			<svg class="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125" />
			</svg>
			<span class="text-green-700 font-medium">{selectedMemoryIds.length} {selectedMemoryIds.length === 1 ? 'memory' : 'memories'} active</span>
			{#if onClearMemories}
				<button
					onclick={onClearMemories}
					class="btn-pill btn-pill-ghost btn-pill-icon"
					aria-label="Clear memories"
				>
					<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			{/if}
		</div>
	</div>
{/if}
