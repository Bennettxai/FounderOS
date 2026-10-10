<script lang="ts">
	import { scale } from 'svelte/transition';
	import { getFileIcon, formatFileSize } from './spotlightActions.ts';
	import type { AttachedFile } from './spotlightActions.ts';

	interface Props {
		attachedFiles: AttachedFile[];
		onRemoveFile: (fileId: string) => void;
	}

	let { attachedFiles, onRemoveFile }: Props = $props();
</script>

{#if attachedFiles.length > 0}
	<div class="attachments-preview">
		{#each attachedFiles as file (file.id)}
			<div class="attachment-item" transition:scale={{ duration: 150 }}>
				{#if file.preview}
					<img src={file.preview} alt={file.name} class="attachment-thumb" />
				{:else}
					<div class="attachment-icon">{getFileIcon(file.type)}</div>
				{/if}
				<div class="attachment-info">
					<span class="attachment-name">{file.name}</span>
					<span class="attachment-size">{formatFileSize(file.size)}</span>
				</div>
				<button
					class="attachment-remove"
					onclick={() => onRemoveFile(file.id)}
					aria-label="Remove {file.name}"
				>
					<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
						<line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
					</svg>
				</button>
			</div>
		{/each}
	</div>
{/if}

<style>
	.attachments-preview {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-bottom: 12px;
		padding-bottom: 12px;
		border-bottom: 1px solid #f0f0f0;
	}

	.attachment-item {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 6px 8px;
		background: #f5f5f5;
		border-radius: 8px;
		max-width: 200px;
	}

	.attachment-thumb {
		width: 36px;
		height: 36px;
		border-radius: 6px;
		object-fit: cover;
	}

	.attachment-icon {
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 20px;
		background: white;
		border-radius: 6px;
	}

	.attachment-info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
	}

	.attachment-name {
		font-size: 12px;
		font-weight: 500;
		color: #333;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.attachment-size {
		font-size: 10px;
		color: #888;
	}

	.attachment-remove {
		width: 20px;
		height: 20px;
		border: none;
		background: transparent;
		border-radius: 50%;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #999;
		transition: all 0.15s;
		flex-shrink: 0;
	}

	.attachment-remove:hover {
		background: #e5e5e5;
		color: #666;
	}

	.attachment-remove svg {
		width: 12px;
		height: 12px;
	}

	/* Dark mode */
	:global(.dark) .attachments-preview {
		border-bottom-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .attachment-item {
		background: #2c2c2e;
	}

	:global(.dark) .attachment-icon {
		background: #3a3a3c;
	}

	:global(.dark) .attachment-name {
		color: #f5f5f7;
	}

	:global(.dark) .attachment-size {
		color: #6e6e73;
	}

	:global(.dark) .attachment-remove {
		color: #6e6e73;
	}

	:global(.dark) .attachment-remove:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}
</style>
