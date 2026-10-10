<script lang="ts">
	interface AttachedFile {
		id: string;
		name: string;
		type: string;
		size: number;
		content?: string;
	}

	interface Props {
		attachedFiles: AttachedFile[];
		onRemove: (fileId: string) => void;
	}

	let { attachedFiles, onRemove }: Props = $props();

	function formatFileSize(bytes: number): string {
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
	}
</script>

{#if attachedFiles.length > 0}
	<div class="attached-files">
		{#each attachedFiles as file (file.id)}
			<div class="attached-file">
				{#if file.type.startsWith('image/') && file.content}
					<img src={file.content} alt={file.name} class="file-preview-img" />
				{:else}
					<div class="file-icon">
						<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" width="16" height="16">
							<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
						</svg>
					</div>
				{/if}
				<div class="file-info">
					<span class="file-name">{file.name}</span>
					<span class="file-size">{formatFileSize(file.size)}</span>
				</div>
				<button class="btn-pill btn-pill-ghost file-remove" onclick={() => onRemove(file.id)} aria-label="Remove file">
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" width="14" height="14">
						<path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
		{/each}
	</div>
{/if}

<style>
	.attached-files {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-bottom: 4px;
	}

	.attached-file {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		background: var(--color-bg-secondary, #f3f4f6);
		border-radius: 8px;
		max-width: 200px;
	}

	:global(.dark) .attached-file {
		background: #3a3a3c;
	}

	.file-preview-img {
		width: 32px;
		height: 32px;
		border-radius: 4px;
		object-fit: cover;
	}

	.file-icon {
		width: 32px;
		height: 32px;
		border-radius: 4px;
		background: var(--color-bg, white);
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--color-text-muted, #6b7280);
	}

	:global(.dark) .file-icon {
		background: #2c2c2e;
		color: #a1a1a6;
	}

	.file-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}

	.file-name {
		font-size: 12px;
		font-weight: 500;
		color: var(--color-text, #1f2937);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	:global(.dark) .file-name {
		color: #f5f5f7;
	}

	.file-size {
		font-size: 10px;
		color: var(--color-text-muted, #6b7280);
	}

	:global(.dark) .file-size {
		color: #a1a1a6;
	}

	.file-remove {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 20px;
		height: 20px;
		border: none;
		background: transparent;
		color: var(--color-text-muted, #6b7280);
		cursor: pointer;
		border-radius: 4px;
		transition: all 0.15s ease;
	}

	.file-remove:hover {
		background: rgba(0, 0, 0, 0.1);
		color: #ef4444;
	}

	:global(.dark) .file-remove:hover {
		background: rgba(255, 255, 255, 0.1);
		color: #f87171;
	}
</style>
